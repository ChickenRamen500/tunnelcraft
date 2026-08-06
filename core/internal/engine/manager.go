package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ChickenRamen500/tunnelcraft/core/internal/config"
	protos "github.com/ChickenRamen500/tunnelcraft/core/internal/proto"
	"google.golang.org/grpc"
)

// Manager orchestrates the full VPN connection lifecycle.
// It selects the protocol, starts the subprocess, manages TUN,
// configures routing, and monitors health.
type Manager struct {
	mu          sync.RWMutex
	cfg         *config.Manager
	state       ConnectionState
	activeServer *ServerConfig
	stats       *ConnectionStats
	events      chan ConnectionEvent
	cancelFunc  context.CancelFunc
	proto       ProtocolHandler // current active protocol handler
	tunnel      TunnelController
	healthChecker *HealthChecker
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
	m.mu.Lock()
	if m.state == StateConnected || m.state == StateConnecting {
		m.mu.Unlock()
		return fmt.Errorf("already connected or connecting")
	}

	m.setState(StateConnecting)
	m.mu.Unlock()

	m.emit(ConnectionEvent{
		State:  StateConnecting,
		Message: fmt.Sprintf("Connecting to %s", serverID),
	})

	// Find the server by ID
	server := m.findServer(serverID)
	if server == nil {
		m.setState(StateError)
		m.emit(ConnectionEvent{
			State:  StateError,
			Error:  fmt.Sprintf("server %s not found", serverID),
			Time:   time.Now(),
		})
		return fmt.Errorf("server %s not found", serverID)
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)
	m.cancelFunc = cancel

	// Get ports from config
	cfg := m.cfg.Get()
	socksPort := cfg.Tunnel.SOCKSPort
	httpPort := cfg.Tunnel.HTTPPort

	// Create protocol handler
	handler, err := m.createHandler(server.Protocol)
	if err != nil {
		cancel()
		m.setState(StateError)
		m.emit(ConnectionEvent{
			State:  StateError,
			Error:  err.Error(),
			Time:   time.Now(),
		})
		return fmt.Errorf("failed to create handler: %w", err)
	}
	m.proto = handler

	// Start protocol subprocess
	if err := handler.Start(ctx, server, socksPort, httpPort); err != nil {
		cancel()
		m.setState(StateError)
		m.emit(ConnectionEvent{
			State:  StateError,
			Error:  fmt.Sprintf("failed to start protocol: %v", err),
			Time:   time.Now(),
		})
		return fmt.Errorf("failed to start protocol: %w", err)
	}

	// Setup TUN and routing
	if m.tunnel != nil {
		if err := m.tunnel.Setup(socksPort, httpPort, server); err != nil {
			// TUN setup failed — stop the protocol
			handler.Stop()
			cancel()
			m.setState(StateError)
			m.emit(ConnectionEvent{
				State:  StateError,
				Error:  fmt.Sprintf("TUN setup failed: %v", err),
				Time:   time.Now(),
			})
			return fmt.Errorf("TUN setup failed: %w", err)
		}

		// Apply routing rules
		_ = m.tunnel.ApplyRoutingRules(cfg.Routing.Rules)
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
		State:  StateDisconnecting,
		Message: "Disconnecting...",
		Time:   time.Now(),
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

	// Teardown TUN
	if m.tunnel != nil {
		m.tunnel.Teardown()
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

// TODO: implement createHandler — dispatches to the correct protocol wrapper
func (m *Manager) createHandler(p Protocol) (ProtocolHandler, error) {
	// Will be implemented in Step 3 with protocol wrappers
	return nil, fmt.Errorf("protocol %s not yet implemented", p)
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
	// TODO: AmneziaWG config conversion
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
		Healthy:  true,
		Version:  "0.1.0",
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
