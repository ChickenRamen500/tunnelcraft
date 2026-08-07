package protocols

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/ChickenRamen500/tunnelcraft/core/internal/engine"
)

// BaseHandler provides common subprocess management for all protocol wrappers.
// Each specific protocol (xray, wireguard, hysteria, amnezia) embeds this
// and implements the protocol-specific config generation.
type BaseHandler struct {
	mu        sync.Mutex
	name      string
	binPath   string
	cmd       *exec.Cmd
	cancel    context.CancelFunc
	logBuffer []string
	maxLogs   int
	startedAt time.Time
}

// NewBaseHandler creates a new base handler.
func NewBaseHandler(name, binPath string) *BaseHandler {
	return &BaseHandler{
		name:    name,
		binPath: binPath,
		maxLogs: 1000,
	}
}

// Name returns the protocol name.
func (b *BaseHandler) Name() string {
	return b.name
}

// BinPath returns the path to the protocol binary.
func (b *BaseHandler) BinPath() string {
	return b.binPath
}

// SetBinPath overrides the binary path.
func (b *BaseHandler) SetBinPath(p string) {
	b.binPath = p
}

// IsRunning returns whether the subprocess is alive.
func (b *BaseHandler) IsRunning() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cmd == nil || b.cmd.Process == nil {
		return false
	}
	return b.cmd.ProcessState == nil || !b.cmd.ProcessState.Exited()
}

// GetLogs returns recent log lines from the subprocess.
func (b *BaseHandler) GetLogs() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	result := make([]string, len(b.logBuffer))
	copy(result, b.logBuffer)
	return result
}

// Stop terminates the subprocess.
func (b *BaseHandler) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.cancel != nil {
		b.cancel()
		b.cancel = nil
	}

	if b.cmd != nil && b.cmd.Process != nil {
		if b.cmd.ProcessState == nil || !b.cmd.ProcessState.Exited() {
			// Try graceful shutdown first
			if err := b.cmd.Process.Signal(os.Interrupt); err != nil {
				// Fall back to kill
				_ = b.cmd.Process.Kill()
			}

			// Wait briefly for cleanup
			done := make(chan error, 1)
			go func() {
				done <- b.cmd.Wait()
			}()

			select {
			case <-done:
			case <-time.After(5 * time.Second):
				_ = b.cmd.Process.Kill()
			}
		}
	}

	b.cmd = nil
	b.startedAt = time.Time{}
	return nil
}

// launchProcess starts the binary with the given arguments.
func (b *BaseHandler) launchProcess(ctx context.Context, args []string, configPath string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check binary exists
	if b.binPath == "" {
		return fmt.Errorf("%s: binary path not set", b.name)
	}
	if _, err := os.Stat(b.binPath); os.IsNotExist(err) {
		return fmt.Errorf("%s: binary not found at %s", b.name, b.binPath)
	}

	ctx, cancel := context.WithCancel(ctx)
	b.cancel = cancel

		absBin, absErr := filepath.Abs(b.binPath)
	if absErr == nil {
		b.binPath = absBin
	}
	b.cmd = exec.CommandContext(ctx, b.binPath, args...)
	b.cmd.Dir = filepath.Dir(configPath)

	// Capture stdout and stderr for logging
	stdoutPipe, err := b.cmd.StdoutPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("%s: failed to create stdout pipe: %w", b.name, err)
	}
	stderrPipe, err := b.cmd.StderrPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("%s: failed to create stderr pipe: %w", b.name, err)
	}

	// Start the process
	if err := b.cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("%s: failed to start process: %w", b.name, err)
	}

	b.startedAt = time.Now()

	// Read logs in background
	go b.readLogs(stdoutPipe, "stdout")
	go b.readLogs(stderrPipe, "stderr")

	return nil
}

// readLogs continuously reads from a pipe and appends to the log buffer.
func (b *BaseHandler) readLogs(pipe interface{ Read([]byte) (int, error) }, prefix string) {
	// Use a simple line reader
	buf := make([]byte, 4096)
	for {
		n, err := pipe.Read(buf)
		if n > 0 {
			line := fmt.Sprintf("[%s] %s", prefix, string(buf[:n]))
fmt.Fprintf(os.Stderr, "[%s] [%s] %s", b.name, prefix, string(buf[:n]))
			b.mu.Lock()
			if len(b.logBuffer) >= b.maxLogs {
				b.logBuffer = b.logBuffer[len(b.logBuffer)/2:]
			}
			b.logBuffer = append(b.logBuffer, line)
			b.mu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

// StartedAt returns when the process was started.
func (b *BaseHandler) StartedAt() time.Time {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.startedAt
}

// appendLog adds a log line (used by specific handlers for their own messages).
func (b *BaseHandler) appendLog(format string, args ...interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()
	line := fmt.Sprintf(format, args...)
	if len(b.logBuffer) >= b.maxLogs {
		b.logBuffer = b.logBuffer[len(b.logBuffer)/2:]
	}
	b.logBuffer = append(b.logBuffer, line)
}

// ensure engine.ProtocolHandler is satisfied.
var _ engine.ProtocolHandler = (*XrayHandler)(nil)
var _ engine.ProtocolHandler = (*WireGuardHandler)(nil)
var _ engine.ProtocolHandler = (*HysteriaHandler)(nil)
var _ engine.ProtocolHandler = (*AmneziaHandler)(nil)
