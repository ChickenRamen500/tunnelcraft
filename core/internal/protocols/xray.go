package protocols

import (
        "context"
        "encoding/json"
        "fmt"
        "os"
        "path/filepath"
        "strings"

        "github.com/ChickenRamen500/tunnelcraft/core/internal/engine"
)

// XrayHandler manages the xray-core.exe subprocess.
// Supports VLESS, VLESS+KCP, VLESS+XHTTP+REALITY, and VMESS.
type XrayHandler struct {
        *BaseHandler
}

// NewXrayHandler creates a new xray-core wrapper.
func NewXrayHandler(binPath string) *XrayHandler {
        return &XrayHandler{
                BaseHandler: NewBaseHandler("xray", binPath),
        }
}

// Start generates an xray JSON config from the server config and launches xray-core.exe.
func (x *XrayHandler) Start(ctx context.Context, server *engine.ServerConfig, socksPort, httpPort uint32) error {
        if x.IsRunning() {
                return fmt.Errorf("xray: already running")
        }

        // Generate config
        cfgPath, err := x.generateConfig(server, socksPort, httpPort)
        if err != nil {
                return fmt.Errorf("xray: failed to generate config: %w", err)
        }

        x.appendLog("[xray] starting xray-core with config: %s", cfgPath)

        // Launch xray-core
        args := []string{"run", "-c", cfgPath}
        if err := x.launchProcess(ctx, args, cfgPath); err != nil {
                return err
        }

        x.appendLog("[xray] process started (PID: %d)", x.cmd.Process.Pid)
        return nil
}

// generateConfig creates an xray-core JSON config file from the unified server config.
func (x *XrayHandler) generateConfig(server *engine.ServerConfig, socksPort, httpPort uint32) (string, error) {
        // Determine the xray protocol name
        var xrayProtocol string
        switch server.Protocol {
        case engine.ProtocolVLESS:
                xrayProtocol = "vless"
        case engine.ProtocolVMESS:
                xrayProtocol = "vmess"
        default:
                return "", fmt.Errorf("unsupported xray protocol: %s", server.Protocol)
        }

        // Build the config
        cfg := map[string]interface{}{
                "log": map[string]interface{}{
                        "loglevel": "warning",
                },
                "inbounds": []interface{}{
                        map[string]interface{}{
                                "tag":     "tunnelcraft-socks",
                                "port":    socksPort,
                                "listen":  "127.0.0.1",
                                "protocol": "socks",
                                "settings": map[string]interface{}{
                                        "auth": "noauth",
                                        "udp":  true,
                                },
                        },
                        map[string]interface{}{
                                "tag":     "tunnelcraft-http",
                                "port":    httpPort,
                                "listen":  "127.0.0.1",
                                "protocol": "http",
                                "settings": map[string]interface{}{},
                        },
                },
                "outbounds": []interface{}{
                        x.buildOutbound(xrayProtocol, server),
                        map[string]interface{}{
                                "tag":      "direct",
                                "protocol": "freedom",
                                "settings": map[string]interface{}{},
                        },
                },
                "routing": map[string]interface{}{
                        "domainStrategy": "IPIfNonMatch",
                        "rules":          []interface{}{},
                },
        }

        // Marshal to JSON
        data, err := json.MarshalIndent(cfg, "", "  ")
        if err != nil {
                return "", fmt.Errorf("failed to marshal xray config: %w", err)
        }

        // Write to temp file
        configDir := os.TempDir()
        configPath := filepath.Join(configDir, fmt.Sprintf("tunnelcraft-xray-%s.json", server.ID))

        if err := os.WriteFile(configPath, data, 0644); err != nil {
                return "", fmt.Errorf("failed to write xray config: %w", err)
        }

        return configPath, nil
}

// buildOutbound constructs the xray outbound configuration.
func (x *XrayHandler) buildOutbound(protocol string, server *engine.ServerConfig) map[string]interface{} {
        outbound := map[string]interface{}{
                "tag":      "proxy",
                "protocol": protocol,
                "settings": map[string]interface{}{
                        "vnext": []interface{}{
                                map[string]interface{}{
                                        "address": server.Host,
                                        "port":    server.Port,
                                        "users": []interface{}{
                                                map[string]interface{}{
                                                        "id":         server.UUID,
                                                        "encryption": "none",
                                                        "flow":       server.Flow,
                                                },
                                        },
                                },
                        },
                },
                "streamSettings": x.buildStreamSettings(server),
        }

        return outbound
}

// buildStreamSettings constructs the transport and security settings.
func (x *XrayHandler) buildStreamSettings(server *engine.ServerConfig) map[string]interface{} {
        stream := map[string]interface{}{
                "network":  x.transportName(server.Transport),
                "security": x.securityName(server.Security),
        }

        // TLS / Reality settings
        switch server.Security {
        case "tls":
                tlsSettings := map[string]interface{}{}
                if server.SNI != "" {
                        tlsSettings["serverName"] = server.SNI
                }
                if server.Fingerprint != "" {
                        tlsSettings["fingerprint"] = server.Fingerprint
                }
                if server.ALPN != "" {
                        tlsSettings["alpn"] = strings.Split(server.ALPN, ",")
                }
                if server.AllowInsecure {
                        tlsSettings["allowInsecure"] = true
                }
                stream["tlsSettings"] = tlsSettings

        case "reality":
                if server.PublicKey == "" {
                        // REALITY requires a public key; if missing, fall back to plain TLS or none.
                        x.appendLog("[xray] WARNING: security=reality but publicKey is empty, falling back to tls")
                        tlsSettings := map[string]interface{}{}
                        if server.SNI != "" {
                                tlsSettings["serverName"] = server.SNI
                        }
                        if server.Fingerprint != "" {
                                tlsSettings["fingerprint"] = server.Fingerprint
                        }
                        if server.ALPN != "" {
                                tlsSettings["alpn"] = strings.Split(server.ALPN, ",")
                        }
                        stream["tlsSettings"] = tlsSettings
                        stream["security"] = "tls"
                } else {
                        // xray-core expects realitySettings at the streamSettings level,
                        // NOT nested inside tlsSettings.
                        stream["realitySettings"] = map[string]interface{}{
                                "serverName": server.SNI,
                                "fingerprint": server.Fingerprint,
                                "publicKey":  server.PublicKey,
                                "shortId":    server.ShortID,
                        }
                }
        }

        // Transport-specific settings
        switch server.Transport {
        case "ws":
                stream["wsSettings"] = map[string]interface{}{
                        "path": server.WSPath,
                }
        case "grpc":
                stream["grpcSettings"] = map[string]interface{}{
                        "serviceName": server.GRPCService,
                }
        case "kcp":
                // Xray 26.x removed header & seed from kcpSettings.
                // Only pass congestion control if explicitly set.
                stream["kcpSettings"] = map[string]interface{}{}
        case "xhttp":
                stream["xhttpSettings"] = map[string]interface{}{
                        "path": server.XHTTPPath,
                        "mode": server.XHTTPMode,
                }
        }

        return stream
}

func (x *XrayHandler) transportName(t string) string {
        switch t {
        case "ws":
                return "ws"
        case "grpc":
                return "grpc"
        case "kcp":
                return "kcp"
        case "xhttp":
                return "xhttp"
        case "httpupgrade":
                return "httpupgrade"
        case "quic":
                return "quic"
        default:
                return "tcp"
        }
}

func (x *XrayHandler) securityName(s string) string {
        switch s {
        case "tls":
                return "tls"
        case "reality":
                return "reality"
        default:
                return "none"
        }
}