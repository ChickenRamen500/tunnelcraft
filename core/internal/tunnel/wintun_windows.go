//go:build windows

package tunnel

import (
	"sync"
	"syscall"
)

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
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.loaded {
		return nil
	}

	// For now, just mark as loaded - actual loading happens via wireguard.exe
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
// Note: WireGuard manages its own TUN adapter via wireguard.exe.
func (w *WintunDLL) CreateAdapter(name string, mtu uint32) (*WintunAdapter, error) {
	if !w.loaded {
		if err := w.Load(); err != nil {
			return nil, err
		}
	}

	adapter := &WintunAdapter{name: name, mtu: mtu, running: true}
	w.adapter = adapter
	return adapter, nil
}

// CloseAdapter closes and destroys the TUN adapter.
func (w *WintunDLL) CloseAdapter() error {
	if w.adapter == nil {
		return nil
	}

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
