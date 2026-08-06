package subscription

import (
        "context"
        "fmt"
        "log"
        "net"
        "sync"
        "time"

        "github.com/ChickenRamen500/tunnelcraft/core/internal/config"
        "github.com/ChickenRamen500/tunnelcraft/core/internal/engine"
)

// FallbackMode indicates the current connection mode.
type FallbackMode int

const (
        // ModeSubscription means we're using a subscription server.
        ModeSubscription FallbackMode = iota
        // ModeFallback means we're using a local WG/AWG server because subscription servers are dead.
        ModeFallback
        // ModeDisconnected means no connection.
        ModeDisconnected
)

// String returns a human-readable mode name.
func (m FallbackMode) String() string {
        switch m {
        case ModeSubscription:
                return "Subscription"
        case ModeFallback:
                return "Fallback (WG)"
        case ModeDisconnected:
                return "Disconnected"
        default:
                return "Unknown"
        }
}

// FallbackManager implements the KEY FEATURE of TunnelCraft:
// When ALL subscription servers are unreachable, it automatically switches
// to a local WireGuard/AmneziaWG fallback config. When subscription servers
// come back, it silently switches back.
type FallbackManager struct {
        mu              sync.RWMutex
        cfg             *config.Manager
        mode            FallbackMode
        fallbackServerID string
        onActivate      func(fallbackServerID string)
          onDeactivate    func()
        cancel          context.CancelFunc
}

// NewFallbackManager creates a new fallback manager.
func NewFallbackManager(cfg *config.Manager) *FallbackManager {
        return &FallbackManager{
                cfg:  cfg,
                mode: ModeDisconnected,
        }
}

// OnActivate sets the callback when fallback is activated.
// The UI should use this to show "Fallback (WG)" status.
func (f *FallbackManager) OnActivate(fn func(fallbackServerID string)) {
        f.mu.Lock()
        defer f.mu.Unlock()
        f.onActivate = fn
}

// OnDeactivate sets the callback when fallback is deactivated (recovery).
// The UI should use this to show "Subscription" status.
func (f *FallbackManager) OnDeactivate(fn func()) {
        f.mu.Lock()
        defer f.mu.Unlock()
        f.onDeactivate = fn
}

// Mode returns the current fallback mode.
func (f *FallbackManager) Mode() FallbackMode {
        f.mu.RLock()
        defer f.mu.RUnlock()
        return f.mode
}

// Start begins the background health check loop.
func (f *FallbackManager) Start(ctx context.Context) {
        f.mu.Lock()
        if f.cancel != nil {
                f.cancel()
        }
        ctx, f.cancel = context.WithCancel(ctx)
        f.mu.Unlock()

        go f.runLoop(ctx)
}

// Stop halts the fallback check loop.
func (f *FallbackManager) Stop() {
        f.mu.Lock()
        defer f.mu.Unlock()
        if f.cancel != nil {
                f.cancel()
                f.cancel = nil
        }
}

// runLoop periodically checks subscription server health.
func (f *FallbackManager) runLoop(ctx context.Context) {
        cfg := f.cfg.Get()
        checkInterval := time.Duration(cfg.Fallback.CheckInterval) * time.Second
        if checkInterval == 0 {
                checkInterval = 300 * time.Second // default 5 minutes
        }

        recoveryInterval := time.Duration(cfg.Fallback.RecoveryInterval) * time.Second
        if recoveryInterval == 0 {
                recoveryInterval = 300 * time.Second
        }

        // Initial delay before first check
        time.Sleep(30 * time.Second)

        checkTicker := time.NewTicker(checkInterval)
        defer checkTicker.Stop()

        for {
                select {
                case <-ctx.Done():
                        return
                case <-checkTicker.C:
                        f.evaluate()
                        // If in fallback mode, check more frequently for recovery
                        f.mu.RLock()
                        currentMode := f.mode
                        f.mu.RUnlock()

                        if currentMode == ModeFallback {
                                // Temporarily increase check frequency during fallback
                                _ = recoveryInterval
                                // Recovery check happens in the same evaluate() call
                        }
                }
        }
}

// evaluate performs a health check and triggers fallback/recovery if needed.
func (f *FallbackManager) evaluate() {
        cfg := f.cfg.Get()

        if !cfg.Fallback.Enabled {
                return
        }

        // Check all subscription servers
        subServers := f.getSubscriptionServers()
        if len(subServers) == 0 {
                return // no subscription servers, nothing to evaluate
        }

        reachableCount := f.checkServersReachability(subServers)
        maxUnhealthy := int(cfg.Fallback.MaxUnhealthyCount)
        if maxUnhealthy == 0 {
                maxUnhealthy = 1
        }

        f.mu.RLock()
        currentMode := f.mode
        f.mu.RUnlock()

        if currentMode != ModeFallback {
                // Not in fallback — check if we need to activate
                if reachableCount < maxUnhealthy {
                        f.activateFallback(cfg.Fallback.FallbackServerID)
                }
        } else {
                // In fallback — check if subscription servers recovered
                if reachableCount >= maxUnhealthy {
                        f.deactivateFallback()
                }
        }
}

// ActivateFallback is the public method to trigger fallback.
func (f *FallbackManager) ActivateFallback(serverID string) {
    f.activateFallback(serverID)
}

// DeactivateFallback is the public method to cancel fallback.
func (f *FallbackManager) DeactivateFallback() {
    f.deactivateFallback()
}

// activateFallback switches to the local WG/AWG fallback server.
func (f *FallbackManager) activateFallback(fallbackServerID string) {
        if fallbackServerID == "" {
                log.Println("[fallback] no fallback server configured, skipping")
                return
        }

        f.mu.Lock()
        f.mode = ModeFallback
        f.fallbackServerID = fallbackServerID
        callback := f.onActivate
        f.mu.Unlock()

        log.Printf("[fallback] ACTIVATED — switching to fallback server %s", fallbackServerID)

        if callback != nil {
                callback(fallbackServerID)
        }
}

// deactivateFallback switches back to subscription servers.
func (f *FallbackManager) deactivateFallback() {
        f.mu.Lock()
        f.mode = ModeSubscription
        f.fallbackServerID = ""
        callback := f.onDeactivate
        f.mu.Unlock()

        log.Println("[fallback] DEACTIVATED — subscription servers recovered, switching back")

        if callback != nil {
                callback()
        }
}

// getSubscriptionServers returns all servers that belong to a subscription.
func (f *FallbackManager) getSubscriptionServers() []config.ServerEntry {
        cfg := f.cfg.Get()
        var subServers []config.ServerEntry
        for _, s := range cfg.Servers {
                if s.SubscriptionID != "" {
                        subServers = append(subServers, s)
                }
        }
        return subServers
}

// checkServersReachability performs TCP connect tests to all servers.
// Returns the count of reachable servers.
func (f *FallbackManager) checkServersReachability(servers []config.ServerEntry) int {
        reachable := 0

        for _, s := range servers {
                addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
                conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
                if err == nil {
                        conn.Close()
                        reachable++
                }
        }

        log.Printf("[fallback] health check: %d/%d subscription servers reachable", reachable, len(servers))
        return reachable
}