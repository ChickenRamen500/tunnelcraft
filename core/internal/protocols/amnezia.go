package protocols

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ChickenRamen500/tunnelcraft/core/internal/engine"
)

// AmneziaHandler manages the amneziawg-go.exe subprocess.
// On Windows, amneziawg-go.exe accepts ONLY the interface name as argument.
// Configuration must be sent via UAPI named pipe after the process starts.
type AmneziaHandler struct {
	*BaseHandler
	tunName string
}

// NewAmneziaHandler creates a new amnezia-wg wrapper.
func NewAmneziaHandler(binPath string) *AmneziaHandler {
	return &AmneziaHandler{
		BaseHandler: NewBaseHandler("amneziawg", binPath),
		tunName:   "TunnelCraft-AWG",
	}
}

// uapiPipePath returns the Windows named pipe path for AmneziaWG UAPI.
func (a *AmneziaHandler) uapiPipePath() string {
	// Go raw string: backslashes are literal.
	// File has \\ which Go reads as two backslashes = correct Windows path prefix.
	return "\\\\." + "\\pipe\\ProtectedPrefix\\Administrators\\AmneziaWG\\" + a.tunName
}

// Start launches amneziawg-go.exe and sends config via UAPI named pipe.
func (a *AmneziaHandler) Start(ctx context.Context, server *engine.ServerConfig, socksPort, httpPort uint32) error {
	if a.IsRunning() {
		return fmt.Errorf("amneziawg: already running")
	}

	// Check wintun.dll exists next to the binary
	binDir := filepath.Dir(a.binPath)
	wintunPath := filepath.Join(binDir, "wintun.dll")
	if _, err := os.Stat(wintunPath); os.IsNotExist(err) {
		return fmt.Errorf("amneziawg: wintun.dll not found at %s", wintunPath)
	}
	a.appendLog("[amneziawg] wintun.dll found at %s", wintunPath)

	// Build IPC config string (WireGuard UAPI key=value format)
	ipcConfig, err := a.buildIPCConfig(server)
	if err != nil {
		return fmt.Errorf("amneziawg: failed to build IPC config: %w", err)
	}

	a.appendLog("[amneziawg] starting amneziawg-go with TUN: %s", a.tunName)

	// CRITICAL: amneziawg-go.exe on Windows accepts ONLY the interface name.
	// Do NOT pass -f or config path - that causes immediate silent exit.
	args := []string{a.tunName}

	if err := a.launchProcess(ctx, args, ""); err != nil {
		return err
	}

	a.appendLog("[amneziawg] process started (PID: %d), TUN: %s", a.cmd.Process.Pid, a.tunName)

	// Send config via UAPI named pipe
	pipePath := a.uapiPipePath()
	a.appendLog("[amneziawg] connecting to UAPI pipe: %s", pipePath)

	f, err := a.connectUAPI(pipePath, 6*time.Second)
	if err != nil {
		a.appendLog("[amneziawg] %v", err)
		a.Stop()
		return err
	}
	defer f.Close()

	a.appendLog("[amneziawg] UAPI connected, sending config (%d bytes)...", len(ipcConfig))

	if _, err := f.WriteString(ipcConfig); err != nil {
		a.appendLog("[amneziawg] failed to write config: %v", err)
		a.Stop()
		return fmt.Errorf("amneziawg: failed to send config via UAPI: %w", err)
	}

	// Read response (should be errno=0)
	f.SetReadDeadline(time.Now().Add(5 * time.Second))
	resp := make([]byte, 256)
	n, err := f.Read(resp)
	if err != nil {
		a.appendLog("[amneziawg] failed to read UAPI response: %v", err)
		a.Stop()
		return fmt.Errorf("amneziawg: UAPI read error: %w", err)
	}

	responseStr := strings.TrimSpace(string(resp[:n]))
	a.appendLog("[amneziawg] UAPI response: %s", responseStr)

	if !strings.Contains(responseStr, "errno=0") {
		a.appendLog("[amneziawg] UAPI returned error: %s", responseStr)
		a.Stop()
		return fmt.Errorf("amneziawg: UAPI error: %s", responseStr)
	}

	a.appendLog("[amneziawg] config applied successfully via UAPI")
	return nil
}

// connectUAPI connects to the UAPI named pipe with retries.
func (a *AmneziaHandler) connectUAPI(pipePath string, maxWait time.Duration) (*os.File, error) {
	deadline := time.Now().Add(maxWait)
	var lastErr error

	for time.Now().Before(deadline) {
		if !a.IsRunning() {
			return nil, fmt.Errorf("process exited while waiting for UAPI pipe")
		}

		f, err := os.OpenFile(pipePath, os.O_RDWR, 0)
		if err == nil {
			return f, nil
		}
		lastErr = err
		time.Sleep(200 * time.Millisecond)
	}

	return nil, fmt.Errorf("UAPI pipe not available after %v: %w", maxWait, lastErr)
}

// buildIPCConfig builds a WireGuard UAPI-format config string.
func (a *AmneziaHandler) buildIPCConfig(server *engine.ServerConfig) (string, error) {
	if server.AmneziaPrivateKey == "" {
		return "", fmt.Errorf("private key is empty")
	}
	if server.AmneziaPublicKey == "" {
		return "", fmt.Errorf("public key is empty")
	}

	var sb strings.Builder

	// Clear existing peers
	sb.WriteString("replace_peers=true\n")

	// Interface section
	sb.WriteString(fmt.Sprintf("private_key=%s\n", server.AmneziaPrivateKey))

	// AmneziaWG junk packet parameters (interface-level)
	sb.WriteString(fmt.Sprintf("Jc=%d\n", server.AmneziaJc))
	sb.WriteString(fmt.Sprintf("Jmin=%d\n", server.AmneziaJmin))
	sb.WriteString(fmt.Sprintf("Jmax=%d\n", server.AmneziaJmax))
	sb.WriteString(fmt.Sprintf("S1=%d\n", server.AmneziaS1))
	sb.WriteString(fmt.Sprintf("S2=%d\n", server.AmneziaS2))
	if server.AmneziaS3 > 0 {
		sb.WriteString(fmt.Sprintf("S3=%d\n", server.AmneziaS3))
	}
	if server.AmneziaH1 != "" {
		sb.WriteString(fmt.Sprintf("H1=%s\n", server.AmneziaH1))
	}
	if server.AmneziaH2 != "" {
		sb.WriteString(fmt.Sprintf("H2=%s\n", server.AmneziaH2))
	}
	if server.AmneziaH3 != "" {
		sb.WriteString(fmt.Sprintf("H3=%s\n", server.AmneziaH3))
	}
	if server.AmneziaH4 != "" {
		sb.WriteString(fmt.Sprintf("H4=%s\n", server.AmneziaH4))
	}

	// Peer section
	sb.WriteString(fmt.Sprintf("public_key=%s\n", server.AmneziaPublicKey))
	if server.AmneziaPresharedKey != "" {
		sb.WriteString(fmt.Sprintf("preshared_key=%s\n", server.AmneziaPresharedKey))
	}
	sb.WriteString(fmt.Sprintf("endpoint=%s:%d\n", server.Host, server.Port))
	sb.WriteString("allowed_ip=0.0.0.0/0\n")
	sb.WriteString("allowed_ip=::/0\n")
	sb.WriteString("persistent_keepalive_interval=25\n")

	return sb.String(), nil
}

// GetTUNName returns the TUN interface name.
func (a *AmneziaHandler) GetTUNName() string {
	return a.tunName
}
