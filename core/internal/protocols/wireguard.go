package protocols

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ChickenRamen500/tunnelcraft/core/internal/engine"
)

// WireGuardHandler manages WireGuard via wireguard.exe and Windows services.
type WireGuardHandler struct {
	mu          sync.Mutex
	name        string
	binPath     string  // путь к wireguard.exe
	confPath    string  // путь к текущему .conf
	tunName     string  // "TunnelCraft-WG"
	logBuffer   []string
	maxLogs     int
	serviceName string  // "WireGuardTunnel$TunnelCraft-WG"
	startedAt   time.Time
}

// NewWireGuardHandler creates a new WireGuard handler using wireguard.exe.
func NewWireGuardHandler(binPath string) *WireGuardHandler {
	return &WireGuardHandler{
		name:      "wireguard",
		binPath:   binPath,
		tunName:   "TunnelCraft-WG",
		maxLogs:   1000,
		serviceName: "WireGuardTunnel$TunnelCraft-WG",
	}
}

// Name returns the protocol name.
func (w *WireGuardHandler) Name() string {
	return w.name
}

// appendLogLocked adds a log line to the buffer. MUST be called with w.mu already locked.
func (w *WireGuardHandler) appendLogLocked(format string, args ...interface{}) {
	line := fmt.Sprintf(format, args...)
	if len(w.logBuffer) >= w.maxLogs {
		w.logBuffer = w.logBuffer[len(w.logBuffer)/2:]
	}
	w.logBuffer = append(w.logBuffer, line)
}

// appendLog adds a log line to the buffer.
func (w *WireGuardHandler) appendLog(format string, args ...interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.appendLogLocked(format, args...)
}

// Start generates a WireGuard config, copies it to a permanent location,
// and installs it as a Windows service via wireguard.exe.
func (w *WireGuardHandler) Start(ctx context.Context, server *engine.ServerConfig, socksPort, httpPort uint32) error {
	log.Printf("[wireguard] >>> Start() entered, server=%s, protocol=%v", server.ID, server.Protocol)

	w.mu.Lock()
	if w.confPath != "" && w.serviceName != "" {
		// Already running
		w.mu.Unlock()
		return fmt.Errorf("wireguard: already running")
	}
	w.mu.Unlock()

	// Check if wintun.dll exists next to wireguard.exe BEFORE attempting to start
	binDir := filepath.Dir(w.binPath)
	wintunPath := filepath.Join(binDir, "wintun.dll")
	if _, err := os.Stat(wintunPath); os.IsNotExist(err) {
		return fmt.Errorf("wireguard: wintun.dll not found at %s — download it or run scripts\\download-binaries.ps1", wintunPath)
	}
	log.Printf("[wireguard] wintun.dll found at %s", wintunPath)

	// Generate config content
	log.Printf("[wireguard] Generating config...")
	configContent, err := w.generateConfigContent(server, socksPort)
	if err != nil {
		log.Printf("[wireguard] generateConfig FAILED: %v", err)
		return fmt.Errorf("wireguard: failed to generate config: %w", err)
	}

	// Determine config path - prefer WireGuard's standard Data directory
	var confPath string
	programFiles := os.Getenv("ProgramFiles")
	if programFiles == "" {
		programFiles = `C:\Program Files`
	}
	wgDataDir := filepath.Join(programFiles, "WireGuard", "Data")
	
	// Try to use WireGuard's standard Data directory first
	if err := os.MkdirAll(wgDataDir, 0755); err == nil {
		confPath = filepath.Join(wgDataDir, "TunnelCraft-WG.conf")
		if err := os.WriteFile(confPath, []byte(configContent), 0600); err == nil {
			log.Printf("[wireguard] Config saved to: %s", confPath)
		} else {
			// Fallback to local tunnels directory
			confPath = ""
		}
	}
	
	// Fallback to local tunnels directory if WireGuard Data dir is not accessible
	if confPath == "" {
		baseDir := filepath.Dir(w.binPath)
		if baseDir == "" {
			exe, _ := os.Executable()
			baseDir = filepath.Dir(exe)
		}
		tunnelsDir := filepath.Join(baseDir, "data", "tunnels")
		if err := os.MkdirAll(tunnelsDir, 0755); err != nil {
			return fmt.Errorf("wireguard: failed to create tunnels dir: %w", err)
		}
		confPath = filepath.Join(tunnelsDir, "TunnelCraft-WG.conf")
		if err := os.WriteFile(confPath, []byte(configContent), 0600); err != nil {
			return fmt.Errorf("wireguard: failed to write config file: %w", err)
		}
		log.Printf("[wireguard] Config saved to: %s (fallback)", confPath)
	}

	// Create independent context for shell commands (not tied to gRPC client context)
	cmdCtx, cmdCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cmdCancel()

	w.mu.Lock()
	w.confPath = confPath
	w.appendLogLocked("[wireguard] installing tunnel service with config: %s", confPath)
	w.mu.Unlock()

	// Install tunnel service using wireguard.exe
	// Command: wireguard.exe /installtunnelservice <path-to-conf>
	args := []string{"/installtunnelservice", confPath}
	log.Printf("[wireguard] Running: %s %v", w.binPath, args)

	cmd := exec.CommandContext(cmdCtx, w.binPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		w.appendLog("[wireguard] installtunnelservice FAILED: %v, output: %s", err, string(output))
		log.Printf("[wireguard] wireguard.exe output: %s", string(output))
		return fmt.Errorf("wireguard: failed to install tunnel service: %w, output: %s", err, string(output))
	}

	w.appendLog("[wireguard] tunnel service installed successfully")
	log.Printf("[wireguard] wireguard.exe output: %s", string(output))

	// After install, wait and check if service actually started
	time.Sleep(3 * time.Second)
	queryCmd := exec.Command("sc", "query", w.serviceName)
	queryOutput, _ := queryCmd.CombinedOutput()
	log.Printf("[wireguard] service status after 3s: %s", string(queryOutput))

	if !strings.Contains(string(queryOutput), "RUNNING") {
		// Read Windows Event Log for the failure reason
		psCmd := exec.Command("powershell", "-Command", 
			"Get-WinEvent -LogName Application -MaxEvents 5 | Where-Object { $_.Message -like '*WireGuard*' -or $_.ProviderName -like '*WireGuard*' } | Select-Object -ExpandProperty Message")
		eventOutput, _ := psCmd.CombinedOutput()
		log.Printf("[wireguard] Windows Event Log (WireGuard errors):\n%s", string(eventOutput))
		
		// Also check if wintun.dll exists next to wireguard.exe
		binDir := filepath.Dir(w.binPath)
		wintunPath := filepath.Join(binDir, "wintun.dll")
		if _, err := os.Stat(wintunPath); os.IsNotExist(err) {
			log.Printf("[wireguard] CRITICAL: wintun.dll NOT FOUND at %s", wintunPath)
		} else {
			log.Printf("[wireguard] wintun.dll found at %s", wintunPath)
		}
	}

	// Wait for service to be in RUNNING state
	w.appendLog("[wireguard] waiting for service to enter RUNNING state...")
	serviceReady := w.waitForServiceRunning(cmdCtx, 10*time.Second)
	if !serviceReady {
		w.appendLog("[wireguard] WARNING: service did not enter RUNNING state within timeout")
		// Continue anyway - service might still be starting
	}

	w.mu.Lock()
	w.startedAt = time.Now()
	w.appendLogLocked("[wireguard] service started, TUN: %s", w.tunName)
	w.mu.Unlock()

	log.Printf("[wireguard] >>> Process launched successfully")
	return nil
}

// generateConfig creates a WireGuard .conf file from the server config.
// Deprecated: use generateConfigContent instead.
func (w *WireGuardHandler) generateConfig(server *engine.ServerConfig, socksPort uint32) (string, error) {
	content, err := w.generateConfigContent(server, socksPort)
	if err != nil {
		return "", err
	}
	configDir := os.TempDir()
	configPath := filepath.Join(configDir, fmt.Sprintf("tunnelcraft-wg-%s.conf", server.ID))
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		return "", fmt.Errorf("failed to write wg config: %w", err)
	}
	return configPath, nil
}

// generateConfigContent generates WireGuard config content as a string.
func (w *WireGuardHandler) generateConfigContent(server *engine.ServerConfig, socksPort uint32) (string, error) {
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

	return sb.String(), nil
}

// Stop stops the WireGuard tunnel service and uninstalls it.
func (w *WireGuardHandler) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.serviceName == "" {
		return nil // Not running
	}

	w.appendLogLocked("[wireguard] stopping tunnel service...")
	log.Printf("[wireguard] Stopping service: %s", w.serviceName)

	// Stop the service first
	stopCmd := exec.Command("sc", "stop", w.serviceName)
	output, err := stopCmd.CombinedOutput()
	if err != nil {
		// Service might already be stopped
		log.Printf("[wireguard] sc stop output: %s", string(output))
	}

	// Wait for service to stop
	time.Sleep(2 * time.Second)

	// Uninstall the tunnel service
	w.appendLogLocked("[wireguard] uninstalling tunnel service...")
	log.Printf("[wireguard] Uninstalling service: %s", w.serviceName)

	uninstallCmd := exec.Command(w.binPath, "/uninstalltunnelservice", "TunnelCraft-WG")
	output, err = uninstallCmd.CombinedOutput()
	if err != nil {
		w.appendLogLocked("[wireguard] uninstalltunnelservice FAILED: %v, output: %s", err, string(output))
		log.Printf("[wireguard] wireguard.exe uninstall output: %s", string(output))
	} else {
		w.appendLogLocked("[wireguard] tunnel service uninstalled successfully")
		log.Printf("[wireguard] wireguard.exe uninstall output: %s", string(output))
	}

	// Remove config file
	if w.confPath != "" {
		if err := os.Remove(w.confPath); err != nil {
			log.Printf("[wireguard] failed to remove config file: %v", err)
		}
		w.confPath = ""
	}

	w.serviceName = ""
	w.startedAt = time.Time{}

	return nil
}

// IsRunning checks if the WireGuard tunnel service is running.
func (w *WireGuardHandler) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.serviceName == "" {
		return false
	}

	// Check service status via sc query
	cmd := exec.Command("sc", "query", w.serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	// Parse output to find STATE
	outputStr := string(output)
	// Look for "RUNNING" in the output
	return strings.Contains(outputStr, "RUNNING")
}

// GetLogs returns recent log lines from the handler.
func (w *WireGuardHandler) GetLogs() []string {
	w.mu.Lock()
	defer w.mu.Unlock()

	result := make([]string, len(w.logBuffer))
	copy(result, w.logBuffer)
	return result
}

// GetTUNName returns the TUN interface name used by WireGuard.
func (w *WireGuardHandler) GetTUNName() string {
	return w.tunName
}

// waitForServiceRunning waits for the WireGuard service to enter RUNNING state.
func (w *WireGuardHandler) waitForServiceRunning(ctx context.Context, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return false
		default:
		}
		cmd := exec.Command("sc", "query", w.serviceName)
		output, err := cmd.CombinedOutput()
		if err == nil && strings.Contains(string(output), "RUNNING") {
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

// waitForTUNInterface waits for the TUN interface to appear.
func (w *WireGuardHandler) waitForTUNInterface(ctx context.Context, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return false
		default:
		}

		// Check if interface exists using netsh
		cmd := exec.Command("netsh", "interface", "show", "interface", w.tunName)
		output, err := cmd.CombinedOutput()
		if err == nil && strings.Contains(string(output), w.tunName) {
			return true
		}

		// Alternative check via ipconfig
		ipconfigCmd := exec.Command("ipconfig", "/all")
		ipconfigOutput, _ := ipconfigCmd.CombinedOutput()
		if strings.Contains(string(ipconfigOutput), w.tunName) {
			return true
		}

		// Alternative check via PowerShell Get-NetAdapter
		psCmd := exec.Command("powershell", "-Command", fmt.Sprintf("Get-NetAdapter -Name \"%s\" -ErrorAction SilentlyContinue", w.tunName))
		psOutput, _ := psCmd.CombinedOutput()
		if len(psOutput) > 0 && !strings.Contains(string(psOutput), "MSFT_NetAdapter") {
			return true
		}

		time.Sleep(500 * time.Millisecond)
	}
	return false
}
