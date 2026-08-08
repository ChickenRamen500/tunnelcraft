//go:build !windows

package tunnel

// WintunDLL is a stub for non-Windows platforms.
type WintunDLL struct{}

// NewWintunDLL creates a new wintun DLL wrapper (stub for non-Windows).
func NewWintunDLL(configDir string) *WintunDLL {
	return &WintunDLL{}
}

// Load loads wintun.dll (stub - always succeeds on non-Windows).
func (w *WintunDLL) Load() error {
	return nil
}

// Unload releases the DLL (stub).
func (w *WintunDLL) Unload() error {
	return nil
}

// CreateAdapter creates a new TUN adapter (stub).
func (w *WintunDLL) CreateAdapter(name string, mtu uint32) (*WintunAdapter, error) {
	return &WintunAdapter{name: name, mtu: mtu}, nil
}

// CloseAdapter closes the TUN adapter (stub).
func (w *WintunDLL) CloseAdapter() error {
	return nil
}

// WintunAdapter represents a TUN adapter (stub for non-Windows).
type WintunAdapter struct {
	name string
	mtu  uint32
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

const (
	// TUNAdapterName is the name of the virtual network adapter.
	TUNAdapterName = "TunnelCraft"

	// DefaultMTU is the default MTU for the TUN adapter.
	DefaultMTU = 1420
)
