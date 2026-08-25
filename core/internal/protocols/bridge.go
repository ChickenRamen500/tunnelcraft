package protocols

import (
        "context"
        "encoding/json"
        "fmt"
        "net"
        "os"
        "path/filepath"

        "github.com/ChickenRamen500/tunnelcraft/core/internal/config"
        "github.com/ChickenRamen500/tunnelcraft/core/internal/engine"
        protos "github.com/ChickenRamen500/tunnelcraft/core/internal/proto"
)

// BridgeHandler manages the two-process bridge mode for transports that
// sing-box does not support (xhttp, kcp).
//
// Architecture:
//
//      xray-core.exe  ←  SOCKS5 inbound on 127.0.0.1:bridgePort
//             ↕
//      sing-box.exe  ←  TUN inbound → SOCKS5 outbound → 127.0.0.1:bridgePort
//
// sing-box captures all system traffic via TUN and forwards it to xray
// through a local SOCKS5 connection. xray then connects to the remote
// server using XHTTP/KCP transport.
type BridgeHandler struct {
        *BaseHandler
        xray       *XrayHandler
        singbox    *SingBoxHandler
        bridgePort uint32
}

const defaultBridgePort uint32 = 10810

// NewBridgeHandler creates a new bridge handler.
func NewBridgeHandler(xrayBinPath, singboxBinPath string) *BridgeHandler {
        return &BridgeHandler{
                BaseHandler: NewBaseHandler("bridge", ""),
                xray:       NewXrayHandler(xrayBinPath),
                singbox:    NewSingBoxHandler(singboxBinPath),
                bridgePort: defaultBridgePort,
        }
}

// Name returns the protocol name.
func (b *BridgeHandler) Name() string {
        return "bridge"
}

// Start launches xray first (SOCKS5 server), then sing-box in bridge mode.
func (b *BridgeHandler) Start(ctx context.Context, server *engine.ServerConfig, socksPort, httpPort uint32) error {
        if b.IsRunning() {
                return fmt.Errorf("bridge: already running")
        }

        // Step 1: Start xray with SOCKS+HTTP inbounds, XHTTP/KCP outbound
        b.appendLog("[bridge] starting xray-core on bridge port %d", b.bridgePort)
        if err := b.xray.Start(ctx, server, b.bridgePort, 0); err != nil {
                return fmt.Errorf("bridge: failed to start xray: %w", err)
        }
        b.appendLog("[bridge] xray-core started (PID running: %v)", b.xray.IsRunning())

        // Step 2: Start sing-box in bridge mode (TUN → SOCKS5 → xray)
        b.appendLog("[bridge] starting sing-box in bridge mode")
        settings := &protos.Settings{
                RoutingSettings: &protos.RoutingSettings{
                        BypassRu:    true,
                        BlockIpv6:   true,
                        DnsViaTunnel: true,
                        BypassLocal: true,
                },
        }
        if err := b.singbox.StartBridge(ctx, server, b.bridgePort, settings); err != nil {
                // Stop xray if sing-box fails
                b.xray.Stop()
                return fmt.Errorf("bridge: failed to start sing-box: %w", err)
        }
        b.appendLog("[bridge] sing-box started (PID running: %v)", b.singbox.IsRunning())
        b.appendLog("[bridge] bridge mode active: TUN → sing-box → SOCKS5(:%d) → xray → %s", b.bridgePort, server.Host)

        return nil
}

// Stop terminates both sing-box and xray.
func (b *BridgeHandler) Stop() error {
        b.appendLog("[bridge] stopping bridge mode")
        // Stop sing-box first (it depends on xray)
        if err := b.singbox.Stop(); err != nil {
                b.appendLog("[bridge] warning: sing-box stop error: %v", err)
        }
        // Then stop xray
        if err := b.xray.Stop(); err != nil {
                b.appendLog("[bridge] warning: xray stop error: %v", err)
        }
        b.appendLog("[bridge] bridge mode stopped")
        return nil
}

// IsRunning returns whether BOTH processes are alive.
// In bridge mode both xray and sing-box must be running for the connection to work.
// Using AND (not OR) so that if sing-box dies, the 2-second grace period in
// Connect() correctly detects the failure instead of reporting success.
func (b *BridgeHandler) IsRunning() bool {
        return b.singbox.IsRunning() && b.xray.IsRunning()
}

// GetLogs returns logs from both processes.
func (b *BridgeHandler) GetLogs() []string {
        xrayLogs := b.xray.GetLogs()
        singboxLogs := b.singbox.GetLogs()
        all := make([]string, 0, len(xrayLogs)+len(singboxLogs)+2)
        all = append(all, "=== XRAY LOGS ===")
        all = append(all, xrayLogs...)
        all = append(all, "=== SING-BOX LOGS ===")
        all = append(all, singboxLogs...)
        return all
}

// SetRoutingRules passes custom routing rules to the internal sing-box handler.
func (b *BridgeHandler) SetRoutingRules(rules []config.RoutingRule) {
        b.singbox.SetRoutingRules(rules)
}

// NeedsBridge returns true if the server requires bridge mode (xhttp or kcp transport).
func NeedsBridge(server *engine.ServerConfig) bool {
        return server.Transport == "xhttp" || server.Transport == "kcp"
}

// --- sing-box bridge mode config generation ---

// StartBridge generates a sing-box config in bridge mode and launches it.
// In bridge mode, sing-box creates a TUN adapter and routes all traffic
// through a SOCKS5 outbound to the local xray process.
func (s *SingBoxHandler) StartBridge(ctx context.Context, server *engine.ServerConfig, bridgePort uint32, settings *protos.Settings) error {
        if s.IsRunning() {
                return fmt.Errorf("sing-box: already running")
        }

        cfgPath, err := s.generateBridgeConfig(server, bridgePort, settings)
        if err != nil {
                return fmt.Errorf("sing-box bridge: failed to generate config: %w", err)
        }

        s.appendLog("[sing-box] starting in bridge mode with config: %s", cfgPath)

        args := []string{"run", "-c", cfgPath}
        if err := s.launchProcess(ctx, args, cfgPath); err != nil {
                return err
        }

        s.appendLog("[sing-box] bridge process started (PID: %d)", s.cmd.Process.Pid)
        return nil
}

// generateBridgeConfig creates a sing-box config that routes TUN traffic to a local SOCKS5 proxy.
func (s *SingBoxHandler) generateBridgeConfig(server *engine.ServerConfig, bridgePort uint32, settings *protos.Settings) (string, error) {
        // Build route exclude addresses for local/bypass traffic
        routeExclude := []string{
                "10.0.0.0/8",
                "172.16.0.0/12",
                "192.168.0.0/16",
                "127.0.0.0/8",
                "169.254.0.0/16",
                "::1/128",
                "fe80::/10",
        }

        // Add server endpoint IP to exclude (so xray can reach it directly)
        // This is critical: without this, the TUN would capture xray's outbound traffic
        // causing a routing loop.
        // NOTE: route_exclude_address only accepts valid CIDR IP prefixes.
        // If server.Host is a domain, resolve it first. Fall back to a DNS rule.
        if server.Host != "" {
                ip := net.ParseIP(server.Host)
                if ip != nil {
                        if ipv4 := ip.To4(); ipv4 != nil {
                                routeExclude = append(routeExclude, ipv4.String()+"/32")
                        } else {
                                routeExclude = append(routeExclude, ip.String()+"/128")
                        }
                }
                // If Host is a domain, we cannot add it to route_exclude_address.
                // Instead, we add a DNS rule to route the server domain via direct.
        }

        // Build routing rules
        rules := []map[string]interface{}{
                {
                        "ip_is_private": true,
                        "outbound":      "direct",
                },
        }

        // If server host is a domain (not IP), add a DNS rule to route it via direct
        // to avoid routing loop through the TUN adapter.
        if server.Host != "" && net.ParseIP(server.Host) == nil {
                rules = append(rules, map[string]interface{}{
                        "domain":  []string{server.Host},
                        "outbound": "direct",
                })
        }

        // Add geoip-ru bypass if enabled (domain-based only, no remote rule-set download
        // which would fail when TUN captures all traffic or raw.githubusercontent.com is blocked).
        if settings != nil && settings.RoutingSettings != nil && settings.RoutingSettings.BypassRu {
                rules = append(rules, map[string]interface{}{
                        "domain_suffix": []string{".ru", ".\u0440\u0444"},
                        "outbound":      "direct",
                })
        }

        // Final: everything else through the bridge (xray)
        rules = append(rules, map[string]interface{}{
                "outbound": "proxy-bridge",
        })

        cfg := map[string]interface{}{
                "log": map[string]interface{}{
                        "level":     "info",
                        "timestamp": true,
                        "output":    "",
                },
                "dns": s.buildDNS(settings),
                "inbounds": []map[string]interface{}{
                        {
                                "type":                      "tun",
                                "tag":                       "tun-in",
                                "interface_name":            "tunnelcraft",
                                "address":                   []string{"10.200.0.1/24", "fd00::1/126"},
                                "stack":                     "mixed",
                                "mtu":                       1400,
                                "auto_route":                true,
                                "strict_route":              true,
                                "endpoint_independent_nat":  true,
                                "route_address":             []string{"0.0.0.0/0", "::/0"},
                                "route_exclude_address":     routeExclude,
                        },
                },
                "outbounds": []map[string]interface{}{
                        {
                                "type":         "socks",
                                "tag":          "proxy-bridge",
                                "server":       "127.0.0.1",
                                "server_port":  bridgePort,
                                "version":      "5",
                        },
                        {
                                "type": "direct",
                                "tag":  "direct",
                        },
                        {
                                "type": "block",
                                "tag":  "block",
                        },
                },
                "route": map[string]interface{}{
                        "rules":                 rules,
                        "auto_detect_interface": true,
                },
        }

        // No remote rule-sets: they require downloading external files which fails
        // when TUN captures all traffic (routing loop) or the host is unreachable.
        // Russian IP bypass is handled by domain_suffix rules above (.ru, .рф).

        data, err := json.MarshalIndent(cfg, "", "  ")
        if err != nil {
                return "", fmt.Errorf("failed to marshal sing-box bridge config: %w", err)
        }

        configDir := os.TempDir()
        configPath := filepath.Join(configDir, fmt.Sprintf("tunnelcraft-singbox-bridge-%s.json", server.ID))

        if err := os.WriteFile(configPath, data, 0644); err != nil {
                return "", fmt.Errorf("failed to write sing-box bridge config: %w", err)
        }

        return configPath, nil
}

// Ensure BridgeHandler satisfies the ProtocolHandler interface.
var _ engine.ProtocolHandler = (*BridgeHandler)(nil)
