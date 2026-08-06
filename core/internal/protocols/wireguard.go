package protocols

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ChickenRamen500/tunnelcraft/core/internal/engine"
)

// WireGuardHandler manages the wireguard-go.exe subprocess.
type WireGuardHandler struct {
	*BaseHandler
}

// NewWireGuardHandler creates a new wireguard-go wrapper.
func NewWireGuardHandler(binPath string) *WireGuardHandler {
	return &WireGuardHandler{
		BaseHandler: NewBaseHandler("wireguard", binPath),
	}
}

// Start generates a WireGuard config and launches wireguard-go.exe.
func (w *WireGuardHandler) Start(ctx context.Context, server *engine.ServerConfig, socksPort, httpPort uint32) error {
	if w.IsRunning() {
		return fmt.Errorf("wireguard: already running")
	}

	// Generate config
	cfgPath, err := w.generateConfig(server, socksPort)
	if err != nil {
		return fmt.Errorf("wireguard: failed to generate config: %w", err)
	}

	w.appendLog("[wireguard] starting wireguard-go with config: %s", cfgPath)

	// wireguard-go uses TUN interface directly
	// It creates a tun adapter and configures it
	tunName := "TunnelCraft-WG"
	args := []string{tunName, "-f", cfgPath}

	if err := w.launchProcess(ctx, args, cfgPath); err != nil {
		return err
	}

	w.appendLog("[wireguard] process started (PID: %d), TUN: %s", w.cmd.Process.Pid, tunName)
	return nil
}

// generateConfig creates a WireGuard .conf file from the server config.
func (w *WireGuardHandler) generateConfig(server *engine.ServerConfig, socksPort uint32) (string, error) {
	dnsServers := server.WGDNSServers
	if dnsServers == "" {
		dnsServers = "1.1.1.1,8.8.8.8"
	}

	allowedIPs := server.WGAllowedIPs
	if allowedIPs == "" {
		allowedIPs = "0.0.0.0/0, ::/0"
	}

	var sb strings.Builder
	sb.WriteString("[Interface]\n")
	sb.WriteString(fmt.Sprintf("# TunnelCraft WireGuard Configuration\n"))
	sb.WriteString(fmt.Sprintf("PrivateKey = %s\n", server.WGPrivateKey))
	sb.WriteString(fmt.Sprintf("Address = %s\n", server.WGLocalAddress))
	sb.WriteString(fmt.Sprintf("DNS = %s\n", dnsServers))
	sb.WriteString("\n")
	sb.WriteString("[Peer]\n")
	sb.WriteString(fmt.Sprintf("# %s\n", server.Name))
	sb.WriteString(fmt.Sprintf("PublicKey = %s\n", server.WGPublicKey))
	if server.WGPresharedKey != "" {
		sb.WriteString(fmt.Sprintf("PresharedKey = %s\n", server.WGPresharedKey))
	}
	sb.WriteString(fmt.Sprintf("Endpoint = %s:%d\n", server.Host, server.Port))
	sb.WriteString(fmt.Sprintf("AllowedIPs = %s\n", allowedIPs))
	sb.WriteString("PersistentKeepalive = 25\n")

	// Write to temp file
	configDir := os.TempDir()
	configPath := filepath.Join(configDir, fmt.Sprintf("tunnelcraft-wg-%s.conf", server.ID))

	if err := os.WriteFile(configPath, []byte(sb.String()), 0600); err != nil {
		return "", fmt.Errorf("failed to write wg config: %w", err)
	}

	return configPath, nil
}

// GetTUNName returns the TUN interface name used by wireguard-go.
func (w *WireGuardHandler) GetTUNName() string {
	return "TunnelCraft-WG"
}
