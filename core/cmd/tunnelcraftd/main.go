// Package main is the entry point for the TunnelCraft daemon (tunnelcraftd).
// This daemon runs as a background service and manages all VPN operations.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/ChickenRamen500/tunnelcraft/core/internal/config"
	"github.com/ChickenRamen500/tunnelcraft/core/internal/engine"
	"github.com/ChickenRamen500/tunnelcraft/core/internal/ipc"
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
	_ = cfgMgr.Update(func(c *config.Config) {
		c.Daemon.DataDir = *dataDir
		c.Daemon.BinDir = *binDir
		c.Daemon.ConfigDir = filepath.Join(baseDir, "configs")
	})

	log.Printf("[info] tunnelcraftd v%s starting...", Version)
	log.Printf("[info] config:  %s", *configPath)
	log.Printf("[info] data:   %s", *dataDir)
	log.Printf("[info] bin:    %s", *binDir)

	// Create the connection manager
	mgr := engine.NewManager(cfgMgr)

	// Create and start the gRPC server
	server := ipc.NewServer(cfgMgr, mgr)
	if err := server.Start(); err != nil {
		log.Fatalf("[fatal] failed to start gRPC server: %v", err)
	}

	// Auto-connect on startup if configured
	cfg := cfgMgr.Get()
	if cfg.Daemon.ConnectOnStartup && cfg.Tunnel.ActiveServerID != "" {
		log.Printf("[info] auto-connecting to server %s...", cfg.Tunnel.ActiveServerID)
		ctx := context.Background()
		if err := mgr.Connect(ctx, cfg.Tunnel.ActiveServerID); err != nil {
			log.Printf("[warn] auto-connect failed: %v", err)
		}
	}

	// Start subscription auto-updater
	// TODO: implement in Step 4

	// Start health checker
	// TODO: implement in Step 6

	// Block until shutdown signal
	server.Wait()

	// Cleanup on exit
	log.Println("[info] shutting down...")
	_ = mgr.Disconnect(true)
}
