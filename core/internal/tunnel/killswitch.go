package tunnel

import (
	"context"
	"log"
	"runtime"
	"sync"
	"time"
)

// KillSwitch blocks all internet traffic when the VPN connection drops unexpectedly.
// On Windows, it works by deleting the default route and adding firewall rules.
type KillSwitch struct {
	mu        sync.Mutex
	routing   *RoutingManager
	active    bool
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewKillSwitch creates a new KillSwitch instance.
func NewKillSwitch(routing *RoutingManager) *KillSwitch {
	return &KillSwitch{routing: routing}
}

// Activate enables the kill switch — blocks all non-VPN traffic.
func (ks *KillSwitch) Activate() {
	if runtime.GOOS != "windows" {
		return
	}

	ks.mu.Lock()
	defer ks.mu.Unlock()

	if ks.active {
		return
	}

	ks.ctx, ks.cancel = context.WithCancel(context.Background())

	// Enable kill switch via routing manager
	ks.routing.SetKillSwitch(true)
	if err := ks.routing.EnableKillSwitch(); err != nil {
		log.Printf("[killswitch] failed to activate: %v", err)
		return
	}

	ks.active = true
	log.Println("[killswitch] ACTIVATED — all non-VPN traffic blocked")
}

// Deactivate restores normal internet access.
func (ks *KillSwitch) Deactivate() {
	if runtime.GOOS != "windows" {
		return
	}

	ks.mu.Lock()
	defer ks.mu.Unlock()

	if !ks.active {
		return
	}

	if ks.cancel != nil {
		ks.cancel()
		ks.cancel = nil
	}

	ks.routing.SetKillSwitch(false)
	if err := ks.routing.DisableKillSwitch(); err != nil {
		log.Printf("[killswitch] failed to deactivate: %v", err)
	}

	ks.active = false
	log.Println("[killswitch] DEACTIVATED — normal routing restored")
}

// IsActive returns whether the kill switch is currently active.
func (ks *KillSwitch) IsActive() bool {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	return ks.active
}

// Watch monitors the connection state and auto-activates/deactivates.
// Call this in a goroutine. It blocks until ctx is cancelled.
func (ks *KillSwitch) Watch(ctx context.Context, connectionState func() bool) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			connected := connectionState()
			if connected && ks.IsActive() {
				ks.Deactivate()
			} else if !connected && !ks.IsActive() {
				ks.Activate()
			}
		}
	}
}
