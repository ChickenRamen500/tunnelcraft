package engine

import (
        "context"
        "fmt"
        "net"
        "sync"
        "time"

        "github.com/ChickenRamen500/tunnelcraft/core/internal/config"
)

// HealthChecker monitors server reachability and triggers fallback.
type HealthChecker struct {
        mu            sync.RWMutex
        cfg           *config.Manager
        mgr           *Manager
        healthResults map[string]*ServerHealth
        fallbackActive bool
        fallbackServerID string
        onFallback   func(serverID string)
onRecovery   func()
        cancel       context.CancelFunc
}

// NewHealthChecker creates a new health checker instance.
func NewHealthChecker(cfg *config.Manager, mgr *Manager) *HealthChecker {
        return &HealthChecker{
                cfg:           cfg,
                mgr:           mgr,
                healthResults: make(map[string]*ServerHealth),
        }
}

// OnFallback sets the callback triggered when fallback is activated.
func (h *HealthChecker) OnFallback(fn func(serverID string)) {
        h.mu.Lock()
        defer h.mu.Unlock()
        h.onFallback = fn
}

// OnRecovery sets the callback triggered when subscription servers recover.
func (h *HealthChecker) OnRecovery(fn func()) {
        h.mu.Lock()
        defer h.mu.Unlock()
        h.onRecovery = fn
}

// Start begins periodic health checks in a background goroutine.
func (h *HealthChecker) Start(ctx context.Context) {
        h.mu.Lock()
        if h.cancel != nil {
                h.cancel()
        }
        ctx, h.cancel = context.WithCancel(ctx)
        h.mu.Unlock()

        go h.runLoop(ctx)
}

// Stop halts the health check loop.
func (h *HealthChecker) Stop() {
        h.mu.Lock()
        defer h.mu.Unlock()
        if h.cancel != nil {
                h.cancel()
                h.cancel = nil
        }
}

// CheckServer performs a one-shot health check on a specific server.
func (h *HealthChecker) CheckServer(server *ServerConfig) ServerHealth {
        result := h.pingServer(server)

        h.mu.Lock()
        h.healthResults[server.ID] = &result
        h.mu.Unlock()

        return result
}

// CheckAll checks all subscription servers and returns results.
func (h *HealthChecker) CheckAll() map[string]*ServerHealth {
        servers := h.cfg.GetServers()
        results := make(map[string]*ServerHealth)

        var mu sync.Mutex
        var wg sync.WaitGroup

        for i := range servers {
                // Skip fallback servers
                if servers[i].SubscriptionID == "" {
                        continue
                }

                s := configEntryToServer(servers[i])
                wg.Add(1)
                go func(srv *ServerConfig) {
                        defer wg.Done()
                        result := h.pingServer(srv)
                        mu.Lock()
                        results[srv.ID] = &result
                        mu.Unlock()
                }(s)
        }

        wg.Wait()

        h.mu.Lock()
        for id, r := range results {
                h.healthResults[id] = r
        }
        h.mu.Unlock()

        return results
}

// IsFallbackActive returns whether fallback mode is currently active.
func (h *HealthChecker) IsFallbackActive() bool {
        h.mu.RLock()
        defer h.mu.RUnlock()
        return h.fallbackActive
}

// GetResults returns a copy of all health check results.
func (h *HealthChecker) GetResults() map[string]ServerHealth {
        h.mu.RLock()
        defer h.mu.RUnlock()
        result := make(map[string]ServerHealth, len(h.healthResults))
        for id, r := range h.healthResults {
                result[id] = *r
        }
        return result
}

// runLoop is the main health check loop.
func (h *HealthChecker) runLoop(ctx context.Context) {
        cfg := h.cfg.Get()
        interval := time.Duration(cfg.Fallback.CheckInterval) * time.Second
        if interval == 0 {
                interval = 300 * time.Second
        }

        ticker := time.NewTicker(interval)
        defer ticker.Stop()

        // Initial check
        h.evaluateFallback()

        for {
                select {
                case <-ctx.Done():
                        return
                case <-ticker.C:
                        h.evaluateFallback()
                }
        }
}

// evaluateFallback checks if fallback should be activated or deactivated.
func (h *HealthChecker) evaluateFallback() {
        results := h.CheckAll()
        cfg := h.cfg.Get()

        if !cfg.Fallback.Enabled {
                return
        }

        h.mu.RLock()
        isFallback := h.fallbackActive
        h.mu.RUnlock()

        // Count reachable subscription servers
        reachableCount := 0
        totalSubServers := 0
        for _, r := range results {
                totalSubServers++
                if r.Reachable {
                        reachableCount++
                }
        }

        maxUnhealthy := int(cfg.Fallback.MaxUnhealthyCount)
        if maxUnhealthy == 0 {
                maxUnhealthy = 1
        }

        if !isFallback {
                // Check if we need to activate fallback
                // All subscription servers must be unreachable (or very few)
                if totalSubServers > 0 && reachableCount < maxUnhealthy {
                        h.mu.Lock()
                        h.fallbackActive = true
                        h.fallbackServerID = cfg.Fallback.FallbackServerID
                        callback := h.onFallback
                        h.mu.Unlock()

                        if callback != nil && h.fallbackServerID != "" {
                                callback(h.fallbackServerID)
                        }
                }
        } else {
                // In fallback mode — check if subscription servers recovered
                if reachableCount >= maxUnhealthy {
                        h.mu.Lock()
                        h.fallbackActive = false
                        callback := h.onRecovery
                        h.mu.Unlock()

                        if callback != nil {
                                callback()
                        }
                }
        }
}

// pingServer tests if a server is reachable by connecting to its host:port.
func (h *HealthChecker) pingServer(server *ServerConfig) ServerHealth {
        result := ServerHealth{
                ServerID:  server.ID,
                LastCheck: time.Now(),
        }

        address := net.JoinHostPort(server.Host, fmt.Sprintf("%d", server.Port))

        // TCP connect test with timeout
        dialer := net.Dialer{Timeout: 5 * time.Second}
        start := time.Now()
        conn, err := dialer.Dial("tcp", address)
        if err != nil {
                result.Reachable = false
                result.Error = err.Error()
                return result
        }
        conn.Close()

        result.Latency = time.Since(start)
        result.Reachable = true
        return result
}
