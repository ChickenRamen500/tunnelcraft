package protocols

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/ChickenRamen500/tunnelcraft/core/internal/engine"
	protos "github.com/ChickenRamen500/tunnelcraft/core/internal/proto"
)

// SingBoxHandler manages the sing-box.exe subprocess.
// Supports VLESS, Trojan, VMess, Hysteria, Hysteria2, Tuic, Shadowsocks with TUN inbound.
type SingBoxHandler struct {
	*BaseHandler
}

// NewSingBoxHandler creates a new sing-box wrapper.
func NewSingBoxHandler(binPath string) *SingBoxHandler {
	return &SingBoxHandler{
		BaseHandler: NewBaseHandler("sing-box", binPath),
	}
}

// Start generates a sing-box JSON config from the server config and launches sing-box.exe.
func (s *SingBoxHandler) Start(ctx context.Context, server *engine.ServerConfig, tunMode bool, localProxyPort uint32) error {
	return s.StartWithSettings(ctx, server, tunMode, localProxyPort, nil)
}

// StartWithSettings generates a sing-box JSON config with user settings and launches sing-box.exe.
func (s *SingBoxHandler) StartWithSettings(ctx context.Context, server *engine.ServerConfig, tunMode bool, localProxyPort uint32, settings *protos.Settings) error {
	if s.IsRunning() {
		return fmt.Errorf("sing-box: already running")
	}

	cfgPath, err := s.generateConfig(server, tunMode, localProxyPort, settings)
	if err != nil {
		return fmt.Errorf("sing-box: failed to generate config: %w", err)
	}

	s.appendLog("[sing-box] starting sing-box with config: %s", cfgPath)

	args := []string{"run", "-c", cfgPath}
	if err := s.launchProcess(ctx, args, cfgPath); err != nil {
		return err
	}

	s.appendLog("[sing-box] process started (PID: %d)", s.cmd.Process.Pid)
	return nil
}

// generateConfig creates a sing-box JSON config file.
func (s *SingBoxHandler) generateConfig(server *engine.ServerConfig, tunMode bool, localProxyPort uint32, settings *protos.Settings) (string, error) {
	cfg := map[string]interface{}{
		"log": map[string]interface{}{
			"level":      "info",
			"timestamp":  true,
			"output":     "",
		},
		"dns":       s.buildDNS(settings),
		"inbounds":  s.buildInbounds(tunMode, localProxyPort),
		"outbounds": s.buildOutbounds(server),
		"route":     s.buildRoute(settings),
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal sing-box config: %w", err)
	}

	configDir := os.TempDir()
	configPath := filepath.Join(configDir, fmt.Sprintf("tunnelcraft-singbox-%s.json", server.ID))

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write sing-box config: %w", err)
	}

	return configPath, nil
}

// buildInbounds creates TUN or mixed inbound.
func (s *SingBoxHandler) buildInbounds(tunMode bool, localProxyPort uint32) []map[string]interface{} {
	if tunMode {
		routeAddress := []string{"0.0.0.0/0"}
		exclude := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.0/8"}
		return []map[string]interface{}{
			{
				"type":                       "tun",
				"tag":                        "tun-in",
				"interface_name":             "tunnelcraft",
				"stack":                      "mixed",
				"mtu":                        1400,
				"auto_route":                 true,
				"strict_route":               true,
				"endpoint_independent_nat":   true,
				"route_address":              routeAddress,
				"route_exclude_address":      exclude,
			},
		}
	}
	return []map[string]interface{}{
		{
			"type":       "mixed",
			"tag":        "mixed-in",
			"listen":     "127.0.0.1",
			"listen_port": localProxyPort,
		},
	}
}

// buildOutbounds creates the proxy outbound based on protocol.
func (s *SingBoxHandler) buildOutbounds(server *engine.ServerConfig) []map[string]interface{} {
	outbounds := []map[string]interface{}{}

	proxyOutbound := s.buildProxyOutbound(server)
	if proxyOutbound != nil {
		outbounds = append(outbounds, proxyOutbound)
	}

	// Add direct and block
	outbounds = append(outbounds,
		map[string]interface{}{"type": "direct", "tag": "direct"},
		map[string]interface{}{"type": "block", "tag": "block"},
	)

	return outbounds
}

// buildProxyOutbound constructs the proxy outbound configuration.
func (s *SingBoxHandler) buildProxyOutbound(server *engine.ServerConfig) map[string]interface{} {
	var outbound map[string]interface{}

	switch server.Protocol {
	case engine.ProtocolVLESS:
		outbound = s.buildVLESS(server)
	case "trojan":
		outbound = s.buildTrojan(server)
	case engine.ProtocolVMESS:
		outbound = s.buildVMess(server)
	case "hysteria2":
		outbound = s.buildHysteria2(server)
	case engine.ProtocolHysteria:
		outbound = s.buildHysteria(server)
	case "shadowsocks":
		outbound = s.buildShadowsocks(server)
	default:
		s.appendLog("[sing-box] unsupported protocol: %s", server.Protocol)
		return nil
	}

	return outbound
}

func (s *SingBoxHandler) buildVLESS(server *engine.ServerConfig) map[string]interface{} {
	outbound := map[string]interface{}{
		"type":         "vless",
		"tag":          "proxy-main",
		"server":       server.Host,
		"server_port":  server.Port,
		"uuid":         server.UUID,
		"flow":         server.Flow,
	}

	if server.Security == "tls" || server.Security == "reality" {
		outbound["tls"] = s.buildTLS(server)
	}

	if transport := s.buildTransport(server); transport != nil {
		outbound["transport"] = transport
	}

	return outbound
}

func (s *SingBoxHandler) buildTrojan(server *engine.ServerConfig) map[string]interface{} {
	outbound := map[string]interface{}{
		"type":         "trojan",
		"tag":          "proxy-main",
		"server":       server.Host,
		"server_port":  server.Port,
		"password":     server.TrojanPassword,
	}

	if server.Security == "tls" || server.Security == "reality" {
		outbound["tls"] = s.buildTLS(server)
	}

	if transport := s.buildTransport(server); transport != nil {
		outbound["transport"] = transport
	}

	return outbound
}

func (s *SingBoxHandler) buildVMess(server *engine.ServerConfig) map[string]interface{} {
	outbound := map[string]interface{}{
		"type":         "vmess",
		"tag":          "proxy-main",
		"server":       server.Host,
		"server_port":  server.Port,
		"uuid":         server.UUID,
		"security":     "auto",
	}

	if server.Security == "tls" {
		outbound["tls"] = s.buildTLS(server)
	}

	if transport := s.buildTransport(server); transport != nil {
		outbound["transport"] = transport
	}

	return outbound
}

func (s *SingBoxHandler) buildHysteria2(server *engine.ServerConfig) map[string]interface{} {
	outbound := map[string]interface{}{
		"type":        "hysteria2",
		"tag":         "proxy-main",
		"server":      server.Host,
		"server_port": server.Port,
		"password":    server.HysteriaAuth,
	}

	tlsConfig := map[string]interface{}{
		"enabled":     true,
		"server_name": server.HysteriaSNI,
		"insecure":    server.HysteriaInsecure,
	}
	if server.HysteriaALPN != "" {
		tlsConfig["alpn"] = []string{server.HysteriaALPN}
	}
	outbound["tls"] = tlsConfig

	if server.HysteriaObfs != "" {
		outbound["obfs"] = map[string]interface{}{
			"type":     "salamander",
			"password": server.HysteriaObfs,
		}
	}

	return outbound
}

func (s *SingBoxHandler) buildHysteria(server *engine.ServerConfig) map[string]interface{} {
	outbound := map[string]interface{}{
		"type":        "hysteria",
		"tag":         "proxy-main",
		"server":      server.Host,
		"server_port": server.Port,
		"auth_str":    server.HysteriaAuth,
	}

	if server.HysteriaBwUp > 0 {
		outbound["up_mbps"] = server.HysteriaBwUp / 1_000_000
	}
	if server.HysteriaBwDown > 0 {
		outbound["down_mbps"] = server.HysteriaBwDown / 1_000_000
	}

	tlsConfig := map[string]interface{}{
		"enabled":     true,
		"server_name": server.HysteriaSNI,
		"insecure":    server.HysteriaInsecure,
	}
	if server.HysteriaALPN != "" {
		tlsConfig["alpn"] = []string{server.HysteriaALPN}
	}
	outbound["tls"] = tlsConfig

	return outbound
}

func (s *SingBoxHandler) buildShadowsocks(server *engine.ServerConfig) map[string]interface{} {
	return map[string]interface{}{
		"type":         "shadowsocks",
		"tag":          "proxy-main",
		"server":       server.Host,
		"server_port":  server.Port,
		"method":       "aes-256-gcm",
		"password":     server.HysteriaAuth,
	}
}

// buildTLS constructs TLS or REALITY settings.
func (s *SingBoxHandler) buildTLS(server *engine.ServerConfig) map[string]interface{} {
	tls := map[string]interface{}{
		"enabled":     true,
		"server_name": server.SNI,
		"insecure":    server.AllowInsecure,
	}

	if server.Fingerprint != "" {
		tls["utls"] = map[string]interface{}{
			"enabled":     true,
			"fingerprint": server.Fingerprint,
		}
	}

	if server.ALPN != "" {
		tls["alpn"] = stringsSplit(server.ALPN, ",")
	}

	if server.Security == "reality" {
		tls["reality"] = map[string]interface{}{
			"enabled":    true,
			"public_key": server.PublicKey,
			"short_id":   server.ShortID,
		}
	}

	return tls
}

// buildTransport constructs transport settings.
func (s *SingBoxHandler) buildTransport(server *engine.ServerConfig) map[string]interface{} {
	switch server.Transport {
	case "ws":
		transport := map[string]interface{}{
			"type": "ws",
			"path": server.WSPath,
		}
		if server.GRPCService != "" {
			transport["headers"] = map[string]interface{}{
				"Host": server.GRPCService,
			}
		}
		return transport
	case "grpc":
		return map[string]interface{}{
			"type":         "grpc",
			"service_name": server.GRPCService,
		}
	case "http", "h2":
		return map[string]interface{}{
			"type": "http",
			"path": server.WSPath,
			"host": []string{server.GRPCService},
		}
	case "httpupgrade":
		return map[string]interface{}{
			"type": "httpupgrade",
			"path": server.WSPath,
			"host": server.GRPCService,
		}
	default:
		return nil
	}
}

// buildDNS creates DNS configuration.
func (s *SingBoxHandler) buildDNS(settings *protos.Settings) map[string]interface{} {
	dnsServers := []map[string]interface{}{
		{
			"tag":      "dns-remote",
			"address":  "https://1.1.1.1/dns-query",
			"address_resolver": "dns-local",
		},
		{
			"tag":     "dns-local",
			"address": "local",
			"detour":  "direct",
		},
		{
			"tag":     "dns-block",
			"address": "rcode://success",
		},
	}

	// If dns_chain is enabled, add configured providers
	if settings != nil && settings.DnsChain.Enabled {
		for _, p := range settings.DnsChain.Doh {
			dnsServers = append(dnsServers, map[string]interface{}{
				"tag":     p.Name,
				"address": p.Addr,
				"detour":  "proxy-main",
			})
		}
		for _, p := range settings.DnsChain.Dot {
			dnsServers = append(dnsServers, map[string]interface{}{
				"tag":     p.Name,
				"address": p.Addr,
				"detour":  "proxy-main",
			})
		}
		for _, p := range settings.DnsChain.Plain {
			dnsServers = append(dnsServers, map[string]interface{}{
				"tag":     p.Name,
				"address": p.Addr,
				"detour":  "direct",
			})
		}
	}

	return map[string]interface{}{
		"servers": dnsServers,
		"rules": []map[string]interface{}{
			{
				"outbound": "any",
				"server":   "dns-remote",
			},
		},
	}
}

// buildRoute creates routing rules.
func (s *SingBoxHandler) buildRoute(settings *protos.Settings) map[string]interface{} {
	rules := []map[string]interface{}{
		{
			"ip_is_private": true,
			"outbound":      "direct",
		},
	}

	// Add geoip-ru rule if bypass_ru is enabled
	if settings != nil && settings.RoutingSettings != nil && settings.RoutingSettings.BypassRu {
		rules = append(rules, map[string]interface{}{
			"rule_set": "geoip-ru",
			"outbound": "direct",
		})
		rules = append(rules, map[string]interface{}{
			"domain_suffix": []string{".ru", ".рф"},
			"outbound":      "direct",
		})
	}

	// Final rule: everything else goes through proxy
	rules = append(rules, map[string]interface{}{
		"outbound": "proxy-main",
	})

	routeCfg := map[string]interface{}{
		"rules":                 rules,
		"auto_detect_interface": true,
	}

	// Add rule_set for geoip-ru if bypass_ru is enabled
	if settings != nil && settings.RoutingSettings != nil && settings.RoutingSettings.BypassRu {
		routeCfg["rule_set"] = []map[string]interface{}{
			{
				"type":              "remote",
				"tag":               "geoip-ru",
				"format":            "binary",
				"url":               "https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-ru.srs",
				"download_detour":   "direct",
			},
		}
	}

	return routeCfg
}

func stringsSplit(s, sep string) []string {
	var result []string
	for _, v := range splitString(s, sep) {
		if v != "" {
			result = append(result, v)
		}
	}
	return result
}

func splitString(s, sep string) []string {
	if s == "" {
		return nil
	}
	var result []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i = start - 1
		}
	}
	result = append(result, s[start:])
	return result
}

// IsIPv6 checks if an address is IPv6.
func IsIPv6(addr string) bool {
	ip := net.ParseIP(addr)
	return ip != nil && ip.To4() == nil
}
