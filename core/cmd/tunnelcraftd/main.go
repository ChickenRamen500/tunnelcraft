// Package main is the entry point for the TunnelCraft daemon (tunnelcraftd).
// This daemon runs as a background service and manages all VPN operations.
package main

import (
        "context"
        "flag"
        "log"
        "os"
        "path/filepath"
        "strings"
        "time"

        "github.com/ChickenRamen500/tunnelcraft/core/internal/config"
        "github.com/ChickenRamen500/tunnelcraft/core/internal/dns"
        "github.com/ChickenRamen500/tunnelcraft/core/internal/engine"
        "github.com/ChickenRamen500/tunnelcraft/core/internal/ipc"
        "github.com/ChickenRamen500/tunnelcraft/core/internal/protocols"
        "github.com/ChickenRamen500/tunnelcraft/core/internal/subscription"
        "github.com/ChickenRamen500/tunnelcraft/core/internal/tunnel"
)

// Version is set at build time via -ldflags.
var Version = "0.1.0-dev"

func main() {
        // Parse command-line flags
        configPath := flag.String("config", "", "path to config.yaml")
        dataDir := flag.String("data", "", "path to data directory")
        binDir := flag.String("bin", "", "path to binaries directory")
        showVersion := flag.Bool("version", false, "print version and exit")
        flag.Parse()

        if *showVersion {
                log.Printf("tunnelcraftd v%s\n", Version)
                os.Exit(0)
        }

        // Determine paths
        exe, _ := os.Executable()
        baseDir := filepath.Dir(exe)

        if *configPath == "" {
                *configPath = filepath.Join(baseDir, "data", "config.yaml")
        }
        if *dataDir == "" {
                *dataDir = filepath.Join(baseDir, "data")
        }
        if *binDir == "" {
                *binDir = filepath.Join(baseDir, "bin")
        }

        // Ensure directories exist
        os.MkdirAll(*dataDir, 0755)
        os.MkdirAll(filepath.Dir(*configPath), 0755)

        // Load configuration
        cfgMgr := config.NewManager(*configPath)
        if err := cfgMgr.Load(); err != nil {
                log.Printf("[warn] failed to load config: %v, using defaults", err)
        }

        // Override paths from flags
        cfg := cfgMgr.Get()
        _ = cfgMgr.Update(func(c *config.Config) {
                c.Daemon.DataDir = *dataDir
                c.Daemon.BinDir = *binDir
                c.Daemon.ConfigDir = filepath.Join(baseDir, "configs")
        })

        log.Printf("[info] tunnelcraftd v%s starting...", Version)
        log.Printf("[info] config:  %s", *configPath)
        log.Printf("[info] data:   %s", *dataDir)
        log.Printf("[info] bin:    %s", *binDir)

        // --- Initialize subsystems ---
        ctx := context.Background()

        // Determine path to wireguard.exe
        // Default: C:\Program Files\WireGuard\wireguard.exe
        wireguardExePath := `C:\Program Files\WireGuard\wireguard.exe`
        if _, err := os.Stat(wireguardExePath); os.IsNotExist(err) {
                // Try to find in PATH
                var found bool
                wireguardExePath, found = findInPath("wireguard.exe")
                if !found {
                        log.Printf("[warn] wireguard.exe not found at default location or in PATH")
                        log.Printf("[warn] WireGuard protocol will not be available")
                } else {
                        log.Printf("[info] wireguard.exe found in PATH: %s", wireguardExePath)
                }
        } else {
                log.Printf("[info] wireguard.exe found at: %s", wireguardExePath)
        }

        // 1. Protocol handlers
        protoHandlers := map[engine.Protocol]engine.ProtocolHandler{
                engine.ProtocolVLESS:     protocols.NewXrayHandler(filepath.Join(*binDir, "xray-core.exe")),
                engine.ProtocolVMESS:     protocols.NewXrayHandler(filepath.Join(*binDir, "xray-core.exe")),
                engine.ProtocolWireGuard: protocols.NewWireGuardHandler(wireguardExePath),
                engine.ProtocolHysteria:  protocols.NewHysteriaHandler(filepath.Join(*binDir, "hysteria.exe")),
                engine.ProtocolAmneziaWG: protocols.NewAmneziaHandler(filepath.Join(*binDir, "amnezia-wg.exe")),
        }

        // 2. Connection manager
        mgr := engine.NewManager(cfgMgr)
        mgr.SetProtocolHandlers(protoHandlers)

        // 3. TUN + Routing
        wintun := tunnel.NewWintunDLL(*binDir)
        routingMgr := tunnel.NewRoutingManager(wintun)
        mgr.SetTunnelController(routingMgr)

        // 4. DNS
        dohResolver := dns.NewDoHResolver(cfg.DNS.DoHURL)
        dnsProxy := dns.NewDNSProxy(dohResolver, cfg.DNS.DNSServers, cfg.DNS.DNSProxyPort)

        // 5. Subscription provider + updater
        subProvider := subscription.NewProvider(cfgMgr)
        subUpdater := subscription.NewUpdater(cfgMgr, subProvider)

        // 6. Fallback manager
        fallbackMgr := subscription.NewFallbackManager(cfgMgr)
        fallbackMgr.OnActivate(func(serverID string) {
                log.Printf("[info] FALLBACK activated, switching to %s", serverID)
                _ = mgr.Disconnect(false)
                time.Sleep(1 * time.Second)
                if err := mgr.Connect(ctx, serverID); err != nil {
                        log.Printf("[error] fallback connect failed: %v", err)
                }
        })
        fallbackMgr.OnDeactivate(func() {
                log.Println("[info] subscription servers recovered")
                // The UI should show the recovery
        })

        // 7. Health checker
        healthChecker := engine.NewHealthChecker(cfgMgr, mgr)
        healthChecker.OnFallback(func(serverID string) {
                fallbackMgr.ActivateFallback(serverID)
        })
        healthChecker.OnRecovery(func() {
                fallbackMgr.DeactivateFallback()
        })
        mgr.SetHealthChecker(healthChecker)

        // 8. Start gRPC server
        server := ipc.NewServer(cfgMgr, mgr)
        if err := server.Start(); err != nil {
                log.Fatalf("[fatal] failed to start gRPC server: %v", err)
        }

        // 9. Start DNS proxy
        if err := dnsProxy.Start(ctx); err != nil {
                log.Printf("[warn] DNS proxy failed to start: %v", err)
        }

        // 10. Start subscription updater
        subUpdater.Start(ctx)

        // 11. Start health checker
        healthChecker.Start(ctx)

        // 12. Start fallback manager
        fallbackMgr.Start(ctx)

        // 13. Auto-connect on startup
        cfg = cfgMgr.Get()
        if cfg.Daemon.ConnectOnStartup && cfg.Tunnel.ActiveServerID != "" {
                log.Printf("[info] auto-connecting to server %s...", cfg.Tunnel.ActiveServerID)
                if err := mgr.Connect(ctx, cfg.Tunnel.ActiveServerID); err != nil {
                        log.Printf("[warn] auto-connect failed: %v", err)
                }
        }

        // Block until shutdown signal
        server.Wait()

        // Cleanup on exit
        log.Println("[info] shutting down...")
        _ = mgr.Disconnect(true)
        dnsProxy.Stop()
        healthChecker.Stop()
        fallbackMgr.Stop()
        subUpdater.Stop()
        wintun.Unload()
        log.Println("[info] shutdown complete")
}

// findInPath searches for an executable in the system PATH.
func findInPath(name string) (string, bool) {
        path := os.Getenv("PATH")
        for _, dir := range strings.Split(path, ";") {
                fullPath := filepath.Join(dir, name)
                if _, err := os.Stat(fullPath); err == nil {
                        return fullPath, true
                }
        }
        return "", false
}
