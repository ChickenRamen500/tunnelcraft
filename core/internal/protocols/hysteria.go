package protocols

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ChickenRamen500/tunnelcraft/core/internal/engine"
)

// HysteriaHandler manages the hysteria.exe subprocess.
// Supports Hysteria2 (QUIC-based) protocol.
type HysteriaHandler struct {
	*BaseHandler
}

// NewHysteriaHandler creates a new hysteria wrapper.
func NewHysteriaHandler(binPath string) *HysteriaHandler {
	return &HysteriaHandler{
		BaseHandler: NewBaseHandler("hysteria", binPath),
	}
}

// Start generates a Hysteria YAML config and launches hysteria.exe.
func (h *HysteriaHandler) Start(ctx context.Context, server *engine.ServerConfig, socksPort, httpPort uint32) error {
	if h.IsRunning() {
		return fmt.Errorf("hysteria: already running")
	}

	// Generate config
	cfgPath, err := h.generateConfig(server, socksPort, httpPort)
	if err != nil {
		return fmt.Errorf("hysteria: failed to generate config: %w", err)
	}

	h.appendLog("[hysteria] starting hysteria with config: %s", cfgPath)

	// Launch hysteria client
	args := []string{"client", "-c", cfgPath}

	if err := h.launchProcess(ctx, args, cfgPath); err != nil {
		return err
	}

	h.appendLog("[hysteria] process started (PID: %d)", h.cmd.Process.Pid)
	return nil
}

// generateConfig creates a Hysteria2 YAML config file from the server config.
func (h *HysteriaHandler) generateConfig(server *engine.ServerConfig, socksPort, httpPort uint32) (string, error) {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("server: %s:%d\n", server.Host, server.Port))
	sb.WriteString(fmt.Sprintf("auth_str: %s\n", server.HysteriaAuth))

	// TLS settings
	sb.WriteString("tls:\n")
	if server.HysteriaSNI != "" {
		sb.WriteString(fmt.Sprintf("  sni: %s\n", server.HysteriaSNI))
	}
	sb.WriteString(fmt.Sprintf("  insecure: %v\n", server.HysteriaInsecure))
	if server.HysteriaALPN != "" {
		sb.WriteString(fmt.Sprintf("  alpn: [%s]\n", strings.Join(strings.Split(server.HysteriaALPN, ","), ", ")))
	}

	// Obfuscation
	if server.HysteriaObfs != "" {
		sb.WriteString("obfs:\n")
		sb.WriteString(fmt.Sprintf("  type: salamander\n"))
		sb.WriteString(fmt.Sprintf("  password: %s\n", server.HysteriaObfs))
	}

	// SOCKS5 inbound
	sb.WriteString(fmt.Sprintf("socks5:\n"))
	sb.WriteString(fmt.Sprintf("  listen: 127.0.0.1:%d\n", socksPort))

	// HTTP proxy inbound
	sb.WriteString(fmt.Sprintf("http:\n"))
	sb.WriteString(fmt.Sprintf("  listen: 127.0.0.1:%d\n", httpPort))

	// Bandwidth (optional, for congestion control)
	if server.HysteriaBwUp > 0 || server.HysteriaBwDown > 0 {
		sb.WriteString("bandwidth:\n")
		if server.HysteriaBwUp > 0 {
			sb.WriteString(fmt.Sprintf("  up: %d\n", server.HysteriaBwUp))
		}
		if server.HysteriaBwDown > 0 {
			sb.WriteString(fmt.Sprintf("  down: %d\n", server.HysteriaBwDown))
		}
	}

	sb.WriteString(fmt.Sprintf("fast_open: %v\n", server.HysteriaFastOpen))
	sb.WriteString("lazy: false\n")

	// Write to temp file
	configDir := os.TempDir()
	configPath := filepath.Join(configDir, fmt.Sprintf("tunnelcraft-hysteria-%s.yaml", server.ID))

	if err := os.WriteFile(configPath, []byte(sb.String()), 0600); err != nil {
		return "", fmt.Errorf("failed to write hysteria config: %w", err)
	}

	return configPath, nil
}