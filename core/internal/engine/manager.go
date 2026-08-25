package engine

import (
        "context"
        "fmt"
        "log"
        "os"
        "path/filepath"
        "sync"
        "time"

        "github.com/ChickenRamen500/tunnelcraft/core/internal/config"
        protos "github.com/ChickenRamen500/tunnelcraft/core/internal/proto"
        "google.golang.org/grpc"
)

// RoutingRuleSetter is implemented by protocol handlers that accept custom routing rules.
// This avoids importing the protocols package from engine (which would create a cycle).
type RoutingRuleSetter interface {
        SetRoutingRules(rules []config.RoutingRule)
}

// Manager orchestrates the full VPN connection lifecycle.
// It selects the protocol, starts the subprocess, manages TUN,
// configures routing, and monitors health.
type Manager struct {
        mu            sync.RWMutex
        cfg           *config.Manager
        state         ConnectionState
        activeServer  *ServerConfig
        stats         *ConnectionStats
        events        chan ConnectionEvent
        cancelFunc    context.CancelFunc
        proto         ProtocolHandler // current active protocol handler
        tunnel        TunnelController
        goTUNActive   bool              // true if Go-level TUN was set up (needs teardown)
        healthChecker *HealthChecker
        protoHandlers map[Protocol]ProtocolHandler
        bridgeHandler ProtocolHandler // lazy-created bridge handler
        bridgeFactory func(xrayPath, singboxPath string) ProtocolHandler // set by daemon during init
}

// ProtocolHandler is the interface each protocol wrapper must implement.
type ProtocolHandler interface {
        // Name returns the protocol identifier (e.g. "vless", "wireguard").
        Name() string

        // Start launches the protocol binary subprocess with the given config.
        Start(ctx context.Context, server *ServerConfig, socksPort, httpPort uint32) error

        // Stop terminates the protocol subprocess.
        Stop() error

        // IsRunning returns whether the subprocess is still alive.
        IsRunning() bool

        // GetLogs returns recent log lines from the subprocess.
        GetLogs() []string
}

// TunnelController is the interface for TUN/routing management.
type TunnelController interface {
        // Setup creates the TUN adapter and configures system routes.
        Setup(socksPort, httpPort uint32, server *ServerConfig) error

        // Teardown removes routes and destroys the TUN adapter.
        Teardown() error

        ApplyRoutingRules(rules []config.RoutingRule) error
}

// NewManager creates a new connection manager.
func NewManager(cfg *config.Manager) *Manager {
        return &Manager{
                cfg:    cfg,
                state:  StateDisconnected,
                stats:  &ConnectionStats{},
                events: make(chan ConnectionEvent, 100),
        }
}

// Events returns a read-only channel of connection state events.
func (m *Manager) Events() <-chan ConnectionEvent {
        return m.events
}

// State returns the current connection state.
func (m *Manager) State() ConnectionState {
        m.mu.RLock()
        defer m.mu.RUnlock()
        return m.state
}

// ActiveServer returns the currently connected server, if any.
func (m *Manager) ActiveServer() *ServerConfig {
        m.mu.RLock()
        defer m.mu.RUnlock()
        return m.activeServer
}

// Stats returns a snapshot of connection statistics.
func (m *Manager) Stats() (bytesUp, bytesDown uint64, duration time.Duration) {
        return m.stats.Snapshot()
}

// Connect starts a VPN connection to the specified server.
func (m *Manager) Connect(ctx context.Context, serverID string) error {
        log.Printf("[manager] >>> Connect() called with serverID=%s", serverID)

        m.mu.Lock()
        if m.state == StateConnected || m.state == StateConnecting {
                m.mu.Unlock()
                return fmt.Errorf("already connected or connecting")
        }

        m.state = StateConnecting
        m.mu.Unlock()

        m.emit(ConnectionEvent{
                State:   StateConnecting,
                Message: fmt.Sprintf("Connecting to %s", serverID),
        })

        // Find the server by ID
        log.Printf("[manager] Looking for server %s", serverID)
        server := m.findServer(serverID)
        if server == nil {
                log.Printf("[manager] Server %s NOT FOUND in config", serverID)
                m.setState(StateError)
                m.emit(ConnectionEvent{
                        State: StateError,
                        Error: fmt.Sprintf("server %s not found", serverID),
                        Time:  time.Now(),
                })
                return fmt.Errorf("server %s not found", serverID)
        }
        log.Printf("[manager] Server found: name=%s, protocol=%v", server.Name, server.Protocol)

        // Create cancellable context
        ctx, cancel := context.WithCancel(context.Background())
        m.cancelFunc = cancel

        // Get ports from config
        cfg := m.cfg.Get()
        socksPort := cfg.Tunnel.SOCKSPort
        httpPort := cfg.Tunnel.HTTPPort

        // Create protocol handler
        log.Printf("[manager] Creating handler for protocol %v (transport=%s)", server.Protocol, server.Transport)
        handler, err := m.createHandler(server)
        if err != nil {
                log.Printf("[manager] Failed to create handler: %v", err)
                cancel()
                m.setState(StateError)
                m.emit(ConnectionEvent{
                        State: StateError,
                        Error: err.Error(),
                        Time:  time.Now(),
                })
                return fmt.Errorf("failed to create handler: %w", err)
        }
        m.proto = handler
        log.Printf("[manager] Handler created: %v", handler.Name())

        // Inject routing rules into handlers that support split tunneling
        if rr, ok := handler.(RoutingRuleSetter); ok {
                rr.SetRoutingRules(m.cfg.Get().Routing.Rules)
        }

        // Start protocol subprocess
        log.Printf("[manager] About to call handler.Start()")
        if err := handler.Start(ctx, server, socksPort, httpPort); err != nil {
                log.Printf("[manager] handler.Start() FAILED: %v", err)
                cancel()
                m.setState(StateError)
                m.emit(ConnectionEvent{
                        State: StateError,
                        Error: fmt.Sprintf("failed to start protocol: %v", err),
                        Time:  time.Now(),
                })
                return fmt.Errorf("failed to start protocol: %w", err)
        }
        log.Printf("[manager] handler.Start() returned successfully")
        log.Printf("[manager] Process running: %v, PID: %d", handler.IsRunning(), getProcessPID(handler))

        // Verify the subprocess is still alive after a brief delay.
        // xray / sing-box often crash immediately on invalid config (e.g.
        // REALITY settings missing), so a 2-second grace period catches those
        // failures before we declare the connection successful.
        log.Printf("[manager] verifying subprocess stays alive (2s grace period)...")
        time.Sleep(2 * time.Second)
        if !handler.IsRunning() {
                log.Printf("[manager] subprocess died during grace period")
                handler.Stop()
                cancel()
                m.setState(StateError)
                m.emit(ConnectionEvent{
                        State: StateError,
                        Error: fmt.Sprintf("protocol process exited immediately (check logs for config errors)"),
                        Time:  time.Now(),
                })
                return fmt.Errorf("protocol process exited immediately (check logs for config errors)")
        }
        log.Printf("[manager] subprocess is alive after grace period")

        // Setup TUN and routing.
        // Skip for protocols that manage their own TUN adapter:
        //   - bridge mode: sing-box creates TUN internally
        //   - sing-box direct: sing-box TUN inbound creates TUN internally
        //   - wireguard/amneziawg: use their own TUN adapters
        // The Go-level RoutingManager is only needed for legacy protocols
        // that expose SOCKS/HTTP without TUN.
        handlerName := handler.Name()
        skipGoTUN := handlerName == "bridge" || handlerName == "sing-box" ||
                handlerName == "wireguard" || handlerName == "hysteria" || handlerName == "amnezia"
        if m.tunnel != nil && !skipGoTUN {
                if err := m.tunnel.Setup(socksPort, httpPort, server); err != nil {
                        // TUN setup failed — stop the protocol
                        handler.Stop()
                        cancel()
                        m.setState(StateError)
                        m.emit(ConnectionEvent{
                                State: StateError,
                                Error: fmt.Sprintf("TUN setup failed: %v", err),
                                Time:  time.Now(),
                        })
                        return fmt.Errorf("TUN setup failed: %w", err)
                }
                m.goTUNActive = true

                // Apply routing rules
                _ = m.tunnel.ApplyRoutingRules(cfg.Routing.Rules)
        } else if skipGoTUN {
                log.Printf("[manager] %s mode: skipping Go-level TUN/routing (protocol manages TUN)", handlerName)
        }

        // Connection established
        m.mu.Lock()
        m.activeServer = server
        m.state = StateConnected
        m.stats = &ConnectionStats{ConnectedAt: time.Now()}
        m.mu.Unlock()

        m.emit(ConnectionEvent{
                State:    StateConnected,
                ServerID: server.ID,
                Message:  fmt.Sprintf("Connected to %s via %s", server.Name, server.Protocol),
                Time:     time.Now(),
        })

        // Start health monitoring
        go m.monitorHealth(ctx, server)

        log.Printf("[manager] >>> Connect() completed successfully")
        return nil
}

// Disconnect tears down the active VPN connection.
func (m *Manager) Disconnect(force bool) error {
        m.mu.Lock()
        if m.state == StateDisconnected {
                m.mu.Unlock()
                return nil
        }

        oldState := m.state
        m.state = StateDisconnecting
        m.mu.Unlock()

        m.emit(ConnectionEvent{
                State:   StateDisconnecting,
                Message: "Disconnecting...",
                Time:    time.Now(),
        })

        // Cancel health monitoring
        if m.cancelFunc != nil {
                m.cancelFunc()
                m.cancelFunc = nil
        }

        // Stop protocol
        if m.proto != nil {
                m.proto.Stop()
                m.proto = nil
        }

        // Teardown Go-level TUN (only if we set it up)
        if m.tunnel != nil && m.goTUNActive {
                m.tunnel.Teardown()
                m.goTUNActive = false
        }

        m.mu.Lock()
        m.activeServer = nil
        m.state = StateDisconnected
        m.mu.Unlock()

        m.emit(ConnectionEvent{
                State:   StateDisconnected,
                Message: "Disconnected",
                Time:    time.Now(),
        })

        _ = oldState
        _ = force
        return nil
}

// Reconnect disconnects and reconnects to the same or a new server.
func (m *Manager) Reconnect(ctx context.Context, serverID string) error {
        currentServer := m.ActiveServer()
        if serverID == "" && currentServer != nil {
                serverID = currentServer.ID
        }

        if err := m.Disconnect(false); err != nil {
                return err
        }

        // Brief pause before reconnect
        time.Sleep(1 * time.Second)
        return m.Connect(ctx, serverID)
}

// SetTunnelController sets the TUN/routing controller.
func (m *Manager) SetTunnelController(tc TunnelController) {
        m.mu.Lock()
        defer m.mu.Unlock()
        m.tunnel = tc
}

// SetHealthChecker sets the health checker.
func (m *Manager) SetHealthChecker(hc *HealthChecker) {
        m.mu.Lock()
        defer m.mu.Unlock()
        m.healthChecker = hc
}

// SetProtocolHandlers registers protocol handlers by type.
func (m *Manager) SetProtocolHandlers(handlers map[Protocol]ProtocolHandler) {
        m.mu.Lock()
        defer m.mu.Unlock()
        m.protoHandlers = handlers
}

// getProcessPID returns the PID of a running process, or -1 if not available.
func getProcessPID(handler ProtocolHandler) int {
        // Try to extract PID from handler using reflection or type assertion
        // For now, return -1 as placeholder since ProtocolHandler interface doesn't expose PID
        return -1
}

// --- internal ---

func (m *Manager) setState(s ConnectionState) {
        m.mu.Lock()
        m.state = s
        m.mu.Unlock()
}

func (m *Manager) emit(e ConnectionEvent) {
        e.Time = time.Now()
        select {
        case m.events <- e:
        default:
                // channel full, drop event
        }
}

func (m *Manager) findServer(id string) *ServerConfig {
        cfg := m.cfg.Get()
        for _, s := range cfg.Servers {
                if s.ID == id {
                        return configEntryToServer(s)
                }
        }
        return nil
}

// createHandler dispatches to the correct protocol wrapper.
// For VLESS/VMESS with XHTTP or KCP transport, it returns a BridgeHandler
// that chains xray-core (for the transport) + sing-box (for TUN).
func (m *Manager) createHandler(server *ServerConfig) (ProtocolHandler, error) {
        // Check if bridge mode is needed (XHTTP or KCP transport)
        if server.Transport == "xhttp" || server.Transport == "kcp" {
                log.Printf("[manager] transport=%s requires bridge mode (xray + sing-box)", server.Transport)
                return m.getBridgeHandler(), nil
        }

        // Standard dispatch to registered protocol handler
        if m.protoHandlers != nil {
                if h, ok := m.protoHandlers[server.Protocol]; ok {
                        return h, nil
                }
        }
        return nil, fmt.Errorf("protocol %s: no handler registered", server.Protocol)
}

// SetBridgeFactory sets the factory function for creating bridge handlers.
// This must be called by the daemon during initialization to avoid
// an import cycle between engine and protocols packages.
func (m *Manager) SetBridgeFactory(factory func(xrayPath, singboxPath string) ProtocolHandler) {
        m.mu.Lock()
        defer m.mu.Unlock()
        m.bridgeFactory = factory
}

// getBridgeHandler lazily creates a BridgeHandler using binary paths from config.
func (m *Manager) getBridgeHandler() ProtocolHandler {
        m.mu.Lock()
        defer m.mu.Unlock()
        if m.bridgeHandler != nil {
                return m.bridgeHandler
        }

        if m.bridgeFactory == nil {
                log.Println("[manager] ERROR: bridge factory not set, cannot create bridge handler")
                return nil
        }

        cfg := m.cfg.Get()
        binDir := cfg.Daemon.BinDir

        xrayPath := filepath.Join(binDir, "xray-core.exe")
        singboxPath := filepath.Join(binDir, "sing-box.exe")

        // Allow override via env vars (useful for dev/debug)
        if v := os.Getenv("TUNNELCRAFT_XRAY_PATH"); v != "" {
                xrayPath = v
        }
        if v := os.Getenv("TUNNELCRAFT_SINGBOX_PATH"); v != "" {
                singboxPath = v
        }

        m.bridgeHandler = m.bridgeFactory(xrayPath, singboxPath)
        log.Printf("[manager] bridge handler created (xray=%s, singbox=%s)", xrayPath, singboxPath)
        return m.bridgeHandler
}

func (m *Manager) monitorHealth(ctx context.Context, server *ServerConfig) {
        // TODO: implement health monitoring loop
        // Periodically check if traffic is flowing
        // Trigger fallback if all subscription servers are dead
        _ = ctx
        _ = server
}

// configEntryToServer converts a persisted config entry to an internal ServerConfig.
func configEntryToServer(e config.ServerEntry) *ServerConfig {
        s := &ServerConfig{
                ID:             e.ID,
                Name:           e.Name,
                Host:           e.Host,
                Port:           e.Port,
                Protocol:       Protocol(e.Protocol),
                Tags:           e.Tags,
                Favorite:       e.Favorite,
                SortOrder:      e.SortOrder,
                SubscriptionID: e.SubscriptionID,
        }

        if e.XrayConfig != nil {
                s.UUID = e.XrayConfig.UUID
                s.Flow = e.XrayConfig.Flow
                s.Security = e.XrayConfig.Security
                s.Transport = e.XrayConfig.Transport
                s.SNI = e.XrayConfig.SNI
                s.Fingerprint = e.XrayConfig.Fingerprint
                s.ALPN = e.XrayConfig.ALPN
                s.PublicKey = e.XrayConfig.PublicKey
                s.ShortID = e.XrayConfig.ShortID
                s.KCPSeed = e.XrayConfig.KCPSeed
                s.XHTTPPath = e.XrayConfig.XHTTPPath
                s.XHTTPMode = e.XrayConfig.XHTTPMode
                s.WSPath = e.XrayConfig.WSPath
                s.GRPCService = e.XrayConfig.GRPCService
                s.AllowInsecure = e.XrayConfig.AllowInsecure
        }

        if e.WGConfig != nil {
                s.WGPrivateKey = e.WGConfig.PrivateKey
                s.WGPublicKey = e.WGConfig.PublicKey
                s.WGPresharedKey = e.WGConfig.PresharedKey
                s.WGLocalAddress = e.WGConfig.LocalAddress
                s.WGDNSServers = e.WGConfig.DNSServers
                s.WGAllowedIPs = e.WGConfig.AllowedIPs
        }

        if e.HysteriaConfig != nil {
                s.HysteriaAuth = e.HysteriaConfig.AuthPassword
                s.HysteriaSNI = e.HysteriaConfig.SNI
                s.HysteriaInsecure = e.HysteriaConfig.Insecure
                s.HysteriaALPN = e.HysteriaConfig.ALPN
                s.HysteriaObfs = e.HysteriaConfig.ObfsPassword
                s.HysteriaBwUp = e.HysteriaConfig.BandwidthUp
                s.HysteriaBwDown = e.HysteriaConfig.BandwidthDown
                s.HysteriaFastOpen = e.HysteriaConfig.FastOpen
        }

        if e.AmneziaConfig != nil {
                s.AmneziaPrivateKey = e.AmneziaConfig.PrivateKey
                s.AmneziaPublicKey = e.AmneziaConfig.PublicKey
                s.AmneziaPresharedKey = e.AmneziaConfig.PresharedKey
                s.AmneziaLocalAddr = e.AmneziaConfig.LocalAddress
                s.AmneziaDNS = e.AmneziaConfig.DNSServers
                s.AmneziaJc = e.AmneziaConfig.Jc
                s.AmneziaJmin = e.AmneziaConfig.Jmin
                s.AmneziaJmax = e.AmneziaConfig.Jmax
                s.AmneziaS1 = e.AmneziaConfig.S1
                s.AmneziaS2 = e.AmneziaConfig.S2
                s.AmneziaS3 = e.AmneziaConfig.S3
                s.AmneziaH1 = e.AmneziaConfig.H1
                s.AmneziaH2 = e.AmneziaConfig.H2
                s.AmneziaH3 = e.AmneziaConfig.H3
                s.AmneziaH4 = e.AmneziaConfig.H4
                s.AmneziaHeaderProtectionKey = e.AmneziaConfig.HeaderProtectionKey
                s.AmneziaContentPaddingAddition = e.AmneziaConfig.ContentPaddingAddition
                s.AmneziaRekeyAfterTime = e.AmneziaConfig.RekeyAfterTime
                s.AmneziaRekeyTimeout = e.AmneziaConfig.RekeyTimeout
                s.AmneziaRejectAfterTime = e.AmneziaConfig.RejectAfterTime
                s.AmneziaKeepaliveTimeout = e.AmneziaConfig.KeepaliveTimeout
                s.AmneziaMaxHandshakeAttempts = e.AmneziaConfig.MaxHandshakeAttempts
        }

        return s
}

// RegisterGRPCServices registers all gRPC service implementations on the server.
func RegisterGRPCServices(s *grpc.Server, mgr *Manager, cfg *config.Manager) {
        // TODO: register all 7 services in Step 6
        // For now, register HealthCheck as a minimal working endpoint
        protos.RegisterDiagnosticsServiceServer(s, &diagnosticsServer{mgr: mgr, cfg: cfg})
}

// diagnosticsServer implements the DiagnosticsService gRPC service.
type diagnosticsServer struct {
        protos.UnimplementedDiagnosticsServiceServer
        mgr *Manager
        cfg *config.Manager
}

func (s *diagnosticsServer) HealthCheck(ctx context.Context, req *protos.HealthCheckRequest) (*protos.HealthCheckResponse, error) {
        _ = ctx
        _ = req
        now := time.Now()
        return &protos.HealthCheckResponse{
                Healthy:   true,
                Version:   "0.1.0",
                CheckedAt: &protos.Timestamp{
                        Seconds: now.Unix(),
                        Nanos:   int32(now.Nanosecond()),
                },
                Checks: []string{"daemon: ok", "grpc: ok"},
        }, nil
}

func (s *diagnosticsServer) GetDaemonInfo(ctx context.Context, req *protos.GetDaemonInfoRequest) (*protos.DaemonInfo, error) {
        _ = ctx
        _ = req
        return &protos.DaemonInfo{
                Version: "0.1.0",
                Os:      "windows",
                Arch:    "amd64",
        }, nil
}

// Stub implementations for remaining DiagnosticsService methods.

func (s *diagnosticsServer) StreamLogs(req *protos.StreamLogsRequest, stream protos.DiagnosticsService_StreamLogsServer) error {
        // TODO: implement log streaming
        return nil
}

func (s *diagnosticsServer) SpeedTest(req *protos.SpeedTestRequest, stream protos.DiagnosticsService_SpeedTestServer) error {
        // TODO: implement speed testing
        return nil
}

func (s *diagnosticsServer) GenerateReport(ctx context.Context, req *protos.GenerateReportRequest) (*protos.GenerateReportResponse, error) {
        // TODO: implement report generation
        return &protos.GenerateReportResponse{}, nil
}