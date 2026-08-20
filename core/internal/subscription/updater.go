package subscription

import (
        "context"
        "fmt"
        "log"
        "sync"
        "time"

        "github.com/ChickenRamen500/tunnelcraft/core/internal/config"
        "github.com/ChickenRamen500/tunnelcraft/core/internal/engine"
)

// Updater periodically refreshes subscriptions in the background.
type Updater struct {
        mu        sync.Mutex
        cfg       *config.Manager
        provider  *Provider
        onUpdate  func(subscriptionID string, servers []engine.ServerConfig)
        onError   func(subscriptionID string, err error)
        cancel    context.CancelFunc
}

// NewUpdater creates a new subscription updater.
func NewUpdater(cfg *config.Manager, provider *Provider) *Updater {
        return &Updater{
                cfg:      cfg,
                provider: provider,
        }
}

// OnUpdate sets the callback when a subscription is successfully updated.
func (u *Updater) OnUpdate(fn func(subscriptionID string, servers []engine.ServerConfig)) {
        u.mu.Lock()
        defer u.mu.Unlock()
        u.onUpdate = fn
}

// OnError sets the callback when a subscription update fails.
func (u *Updater) OnError(fn func(subscriptionID string, err error)) {
        u.mu.Lock()
        defer u.mu.Unlock()
        u.onError = fn
}

// Start begins the periodic update loop.
func (u *Updater) Start(ctx context.Context) {
        u.mu.Lock()
        if u.cancel != nil {
                u.cancel()
        }
        ctx, u.cancel = context.WithCancel(ctx)
        u.mu.Unlock()

        go u.runLoop(ctx)
}

// Stop halts the update loop.
func (u *Updater) Stop() {
        u.mu.Lock()
        defer u.mu.Unlock()
        if u.cancel != nil {
                u.cancel()
                u.cancel = nil
        }
}

// RefreshNow triggers an immediate refresh of a specific subscription.
func (u *Updater) RefreshNow(ctx context.Context, subscriptionID string) (*FetchResult, error) {
        result := u.provider.Fetch(ctx, subscriptionID)
        if result.Error != "" {
                return result, fmt.Errorf("%s", result.Error)
        }

        // Update the config with new servers
        u.replaceSubscriptionServers(subscriptionID, result.Servers)

        u.mu.Lock()
        callback := u.onUpdate
        u.mu.Unlock()

        if callback != nil {
                callback(subscriptionID, result.Servers)
        }

        return result, nil
}

// RefreshAll triggers an immediate refresh of all subscriptions.
func (u *Updater) RefreshAll(ctx context.Context) []*FetchResult {
        results := u.provider.FetchAll(ctx)
        for _, result := range results {
                if result.Error == "" {
                        u.replaceSubscriptionServers(result.SubscriptionID, result.Servers)
                        u.mu.Lock()
                        callback := u.onUpdate
                        u.mu.Unlock()
                        if callback != nil {
                                callback(result.SubscriptionID, result.Servers)
                        }
                } else {
                        u.mu.Lock()
                        errCallback := u.onError
                        u.mu.Unlock()
                        if errCallback != nil {
                                errCallback(result.SubscriptionID, fmt.Errorf("%s", result.Error))
                        }
                }
        }
        return results
}

// runLoop is the main periodic update loop.
func (u *Updater) runLoop(ctx context.Context) {
        // Wait a bit before first update (let the daemon start)
        time.Sleep(10 * time.Second)

        // Initial refresh
        u.RefreshAll(ctx)

        // Calculate the minimum refresh interval across all subscriptions
        ticker := u.calculateTicker()
        if ticker == nil {
                return // no auto-refresh needed
        }
        defer ticker.Stop()

        for {
                select {
                case <-ctx.Done():
                        return
                case <-ticker.C:
                        u.RefreshAll(ctx)
                        // Re-calculate interval (may have changed)
                        ticker.Stop()
                        ticker = u.calculateTicker()
                        if ticker == nil {
                                return
                        }
                }
        }
}

// calculateTicker creates a ticker with the minimum refresh interval.
func (u *Updater) calculateTicker() *time.Ticker {
        subs := u.cfg.GetSubscriptions()
        var minInterval uint32

        for _, s := range subs {
                if s.Enabled && s.RefreshInterval > 0 {
                        if minInterval == 0 || s.RefreshInterval < minInterval {
                                minInterval = s.RefreshInterval
                        }
                }
        }

        if minInterval == 0 {
                return nil
        }

        return time.NewTicker(time.Duration(minInterval) * time.Minute)
}

// replaceSubscriptionServers updates the config with new servers from a subscription.
func (u *Updater) replaceSubscriptionServers(subscriptionID string, newServers []engine.ServerConfig) {
        u.cfg.Update(func(c *config.Config) {
                // Remove old servers from this subscription
                var filtered []config.ServerEntry
                for _, s := range c.Servers {
                        if s.SubscriptionID != subscriptionID {
                                filtered = append(filtered, s)
                        }
                }
                c.Servers = filtered

                // Add new servers
                for _, srv := range newServers {
                        entry := serverConfigToEntry(srv)
                        c.Servers = append(c.Servers, entry)
                }
        })

        log.Printf("[subscription] updated %d servers for subscription %s", len(newServers), subscriptionID)
}

// serverConfigToEntry converts an engine.ServerConfig to a config.ServerEntry for persistence.
func serverConfigToEntry(s engine.ServerConfig) config.ServerEntry {
        e := config.ServerEntry{
                ID:             s.ID,
                Name:           s.Name,
                Host:           s.Host,
                Port:           s.Port,
                Protocol:       string(s.Protocol),
                Tags:           s.Tags,
                Favorite:       s.Favorite,
                SortOrder:      s.SortOrder,
                SubscriptionID: s.SubscriptionID,
        }

        // Xray-specific fields
        if s.UUID != "" || s.Protocol == engine.ProtocolVLESS || s.Protocol == engine.ProtocolVMESS {
                e.XrayConfig = &config.XrayConfigEntry{
                        UUID:         s.UUID,
                        Flow:         s.Flow,
                        Security:     s.Security,
                        Transport:    s.Transport,
                        SNI:          s.SNI,
                        Fingerprint:  s.Fingerprint,
                        ALPN:         s.ALPN,
                        PublicKey:    s.PublicKey,
                        ShortID:      s.ShortID,
                        KCPSeed:      s.KCPSeed,
                        XHTTPPath:    s.XHTTPPath,
                        XHTTPMode:    s.XHTTPMode,
                        WSPath:       s.WSPath,
                        GRPCService:  s.GRPCService,
                        AllowInsecure: s.AllowInsecure,
                }
        }

        // WireGuard-specific fields
        if s.WGPrivateKey != "" || s.Protocol == engine.ProtocolWireGuard {
                e.WGConfig = &config.WGConfigEntry{
                        PrivateKey:        s.WGPrivateKey,
                        PublicKey:         s.WGPublicKey,
                        PresharedKey:      s.WGPresharedKey,
                        LocalAddress:      s.WGLocalAddress,
                        DNSServers:        s.WGDNSServers,
                        AllowedIPs:        s.WGAllowedIPs,
                }
        }

        // Hysteria-specific fields
        if s.HysteriaAuth != "" || s.Protocol == engine.ProtocolHysteria {
                e.HysteriaConfig = &config.HysteriaConfigEntry{
                        AuthPassword: s.HysteriaAuth,
                        SNI:          s.HysteriaSNI,
                        Insecure:     s.HysteriaInsecure,
                        ALPN:         s.HysteriaALPN,
                        ObfsPassword: s.HysteriaObfs,
                        BandwidthUp:  s.HysteriaBwUp,
                        BandwidthDown: s.HysteriaBwDown,
                        FastOpen:     s.HysteriaFastOpen,
                }
        }

        // AmneziaWG-specific fields
        if s.AmneziaPrivateKey != "" || s.Protocol == engine.ProtocolAmneziaWG {
                e.AmneziaConfig = &config.AmneziaConfigEntry{
                        PrivateKey:   s.AmneziaPrivateKey,
                        PublicKey:    s.AmneziaPublicKey,
                        PresharedKey: s.AmneziaPresharedKey,
                        LocalAddress: s.AmneziaLocalAddr,
                        DNSServers:   s.AmneziaDNS,
                        Jc:           s.AmneziaJc,
                        Jmin:         s.AmneziaJmin,
                        Jmax:         s.AmneziaJmax,
                        S1:           s.AmneziaS1,
                        S2:           s.AmneziaS2,
                        S3:           s.AmneziaS3,
                        H1:           s.AmneziaH1,
                        H2:           s.AmneziaH2,
                        H3:           s.AmneziaH3,
                        H4:           s.AmneziaH4,
                        HeaderProtectionKey:   s.AmneziaHeaderProtectionKey,
                        ContentPaddingAddition: s.AmneziaContentPaddingAddition,
                        RekeyAfterTime: s.AmneziaRekeyAfterTime,
                        RekeyTimeout: s.AmneziaRekeyTimeout,
                        RejectAfterTime: s.AmneziaRejectAfterTime,
                        KeepaliveTimeout: s.AmneziaKeepaliveTimeout,
                        MaxHandshakeAttempts: s.AmneziaMaxHandshakeAttempts,
                }
        }

        return e
}