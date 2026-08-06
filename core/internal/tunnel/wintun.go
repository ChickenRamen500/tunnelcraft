package tunnel

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
)

// Windows TUN interface management via wintun.dll.
// On non-Windows platforms, all operations are stubs.
//
// The real implementation will use CGo to call wintun.dll functions:
//   WintunCreateAdapter, WintunCloseAdapter,
//   WintunReceivePacket, WintunSendPacket, etc.

const (
	// TUNAdapterName is the name of the virtual network adapter.
	TUNAdapterName = "TunnelCraft"

	// DefaultMTU is the default MTU for the TUN adapter.
	DefaultMTU = 1420

	// wintunDLL is the filename of the wintun library.
	wintunDLL = "wintun.dll"
)

// WintunDLL holds the loaded DLL handle and manages adapter lifecycle.
type WintunDLL struct {
	handle    syscall.Handle
	adapter   *WintunAdapter
	mu        sync.Mutex
	loaded    bool
	configDir string
}

// WintunAdapter represents a created TUN adapter.
type WintunAdapter struct {
	name    string
	luid    uint64
	handle  syscall.Handle
	mtu     uint32
	running bool
}

// NewWintunDLL creates a new wintun DLL wrapper (does not load yet).
func NewWintunDLL(configDir string) *WintunDLL {
	return &WintunDLL{configDir: configDir}
}

// Load loads wintun.dll from the configured directory.
func (w *WintunDLL) Load() error {
	if runtime.GOOS != "windows" {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.loaded {
		return nil
	}

	dllPath := filepath.Join(w.configDir, wintunDLL)
	if _, err := os.Stat(dllPath); os.IsNotExist(err) {
		dllPath = wintunDLL
	}

	handle, err := syscall.LoadDLL(dllPath)
	if err != nil {
		return fmt.Errorf("failed to load %s: %w", dllPath, err)
	}

	w.handle = handle
	w.loaded = true
	return nil
}

// Unload releases the DLL and closes any active adapter.
func (w *WintunDLL) Unload() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.loaded {
		return nil
	}

	if w.adapter != nil {
		w.CloseAdapter()
	}

	w.loaded = false
	return nil
}

// CreateAdapter creates a new TUN adapter.
func (w *WintunDLL) CreateAdapter(name string, mtu uint32) (*WintunAdapter, error) {
	if !w.loaded {
		if err := w.Load(); err != nil {
			return nil, err
		}
	}

	if runtime.GOOS != "windows" {
		return &WintunAdapter{name: name, mtu: mtu, running: true}, nil
	}

	// TODO: implement actual wintun adapter creation via CGo
	// WintunCreateAdapter(name, &GUID, mtu)
	adapter := &WintunAdapter{name: name, mtu: mtu, running: true}
	w.adapter = adapter
	return adapter, nil
}

// CloseAdapter closes and destroys the TUN adapter.
func (w *WintunDLL) CloseAdapter() error {
	if w.adapter == nil {
		return nil
	}

	// TODO: call WintunCloseAdapter via CGo
	w.adapter.running = false
	w.adapter = nil
	return nil
}

// Adapter returns the current adapter.
func (w *WintunDLL) Adapter() *WintunAdapter {
	return w.adapter
}

// Name returns the adapter name.
func (a *WintunAdapter) Name() string {
	if a == nil {
		return ""
	}
	return a.name
}

// MTU returns the adapter MTU.
func (a *WintunAdapter) MTU() uint32 {
	if a == nil {
		return 0
	}
	return a.mtu
}

// IsRunning returns whether the adapter is active.
func (a *WintunAdapter) IsRunning() bool {
	return a != nil && a.running
}

// ReadPacket reads a single packet from the TUN adapter.
// TODO: implement via CGo calling WintunReceivePacket.
func (a *WintunAdapter) ReadPacket(ctx context.Context) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return nil, errors.New("TUN read: stub mode (CGo not linked)")
}

// WritePacket writes a packet to the TUN adapter.
// TODO: implement via CGo calling WintunAllocateSendPacket + WintunSendPacket.
func (a *WintunAdapter) WritePacket(packet []byte) error {
	return errors.New("TUN write: stub mode (CGo not linked)")
}
