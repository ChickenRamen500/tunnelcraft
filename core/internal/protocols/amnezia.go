package protocols

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ChickenRamen500/tunnelcraft/core/internal/engine"
)

// AmneziaHandler manages the amnezia-wg.exe subprocess.
// AmneziaWG (AWG) is a WireGuard fork with anti-DPI obfuscation.
type AmneziaHandler struct {
	*BaseHandler
}

// NewAmneziaHandler creates a new amnezia-wg wrapper.
func NewAmneziaHandler(binPath string) *AmneziaHandler {
	return &AmneziaHandler{
		BaseHandler: NewBaseHandler("amneziawg", binPath),
	}
}

// Start generates an AmneziaWG config and launches amnezia-wg.exe.
func (a *AmneziaHandler) Start(ctx context.Context, server *engine.ServerConfig, socksPort, httpPort uint32) error {
	if a.IsRunning() {
		return fmt.Errorf("amneziawg: already running")
	}

	// Generate config
	cfgPath, err := a.generateConfig(server, socksPort)
	if err != nil {
		return fmt.Errorf("amneziawg: failed to generate config: %w", err)
	}

	a.appendLog("[amneziawg] starting amnezia-wg with config: %s", cfgPath)

	// amnezia-wg uses similar CLI to wireguard-go
	tunName := "TunnelCraft-AWG"
	args := []string{tunName, "-f", cfgPath}

	if err := a.launchProcess(ctx, args, cfgPath); err != nil {
		return err
	}

	a.appendLog("[amneziawg] process started (PID: %d), TUN: %s", a.cmd.Process.Pid, tunName)
	return nil
}

// generateConfig creates an AmneziaWG config file from the server config.
// AmneziaWG uses the same format as WireGuard but adds junk packet parameters.
func (a *AmneziaHandler) generateConfig(server *engine.ServerConfig, socksPort uint32) (string, error) {
	dnsServers := server.AmneziaDNS
	if dnsServers == "" {
		dnsServers = "1.1.1.1,8.8.8.8"
	}

	var sb strings.Builder
	sb.WriteString("[Interface]\n")
	sb.WriteString("# TunnelCraft AmneziaWG Configuration\n")
	sb.WriteString(fmt.Sprintf("PrivateKey = %s\n", server.AmneziaPrivateKey))
	sb.WriteString(fmt.Sprintf("Address = %s\n", server.AmneziaLocalAddr))
	sb.WriteString(fmt.Sprintf("DNS = %s\n", dnsServers))

	// AmneziaWG-specific junk packet parameters
	// Jc = number of junk packets
	// Jmin/Jmax = min/max size of junk packets
	// S1/S2 = magic header sizes
	// H1-H4 = handshake packet type markers
	sb.WriteString(fmt.Sprintf("Jc = %d\n", server.AmneziaJc))
	sb.WriteString(fmt.Sprintf("Jmin = %d\n", server.AmneziaJmin))
	sb.WriteString(fmt.Sprintf("Jmax = %d\n", server.AmneziaJmax))
	sb.WriteString(fmt.Sprintf("S1 = %d\n", server.AmneziaS1))
	sb.WriteString(fmt.Sprintf("S2 = %d\n", server.AmneziaS2))
	sb.WriteString(fmt.Sprintf("H1 = %d\n", server.AmneziaH1))
	sb.WriteString(fmt.Sprintf("H2 = %d\n", server.AmneziaH2))
	sb.WriteString(fmt.Sprintf("H3 = %d\n", server.AmneziaH3))
	sb.WriteString(fmt.Sprintf("H4 = %d\n", server.AmneziaH4))

	sb.WriteString("\n[Peer]\n")
	sb.WriteString(fmt.Sprintf("# %s\n", server.Name))
	sb.WriteString(fmt.Sprintf("PublicKey = %s\n", server.AmneziaPublicKey))
	if server.AmneziaPresharedKey != "" {
		sb.WriteString(fmt.Sprintf("PresharedKey = %s\n", server.AmneziaPresharedKey))
	}
	sb.WriteString(fmt.Sprintf("Endpoint = %s:%d\n", server.Host, server.Port))
	sb.WriteString("AllowedIPs = 0.0.0.0/0, ::/0\n")
	sb.WriteString("PersistentKeepalive = 25\n")

	// Write to temp file
	configDir := os.TempDir()
	configPath := filepath.Join(configDir, fmt.Sprintf("tunnelcraft-amnezia-%s.conf", server.ID))

	if err := os.WriteFile(configPath, []byte(sb.String()), 0600); err != nil {
		return "", fmt.Errorf("failed to write amnezia config: %w", err)
	}

	return configPath, nil
}

// GetTUNName returns the TUN interface name used by amnezia-wg.
func (a *AmneziaHandler) GetTUNName() string {
	return "TunnelCraft-AWG"
}
