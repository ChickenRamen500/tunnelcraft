package ipc

import (
        "context"
        "fmt"
        "log"
        "net"
        "os"
        "os/signal"
        "runtime/debug"
        "syscall"
        "time"

        "google.golang.org/grpc"
        "google.golang.org/grpc/codes"
        "google.golang.org/grpc/credentials/insecure"
        "google.golang.org/grpc/reflection"
        "google.golang.org/grpc/status"

        "github.com/ChickenRamen500/tunnelcraft/core/internal/config"
        "github.com/ChickenRamen500/tunnelcraft/core/internal/engine"
        protos "github.com/ChickenRamen500/tunnelcraft/core/internal/proto"
		"google.golang.org/protobuf/types/known/emptypb"

	"github.com/ChickenRamen500/tunnelcraft/core/internal/subscription"
)

// Server wraps the gRPC server and all service implementations.
type Server struct {
        grpcServer *grpc.Server
        cfg        *config.Manager
        mgr        *engine.Manager
        logger     *log.Logger
}

// NewServer creates a new gRPC IPC server.
func NewServer(cfg *config.Manager, mgr *engine.Manager) *Server {
        s := &Server{
                cfg:    cfg,
                mgr:    mgr,
                logger: log.New(os.Stderr, "[ipc] ", log.LstdFlags),
        }

        s.grpcServer = grpc.NewServer(
                grpc.MaxRecvMsgSize(16*1024*1024),
                grpc.MaxSendMsgSize(16*1024*1024),
                grpc.UnaryInterceptor(s.unaryInterceptor()),
                grpc.StreamInterceptor(s.streamInterceptor()),
        )

        // Register all services
        protos.RegisterTunnelServiceServer(s.grpcServer, newTunnelService(s))
        protos.RegisterServerServiceServer(s.grpcServer, newServerService(s))
        protos.RegisterSubscriptionServiceServer(s.grpcServer, newSubscriptionService(s))
        protos.RegisterSettingsServiceServer(s.grpcServer, newSettingsService(s))
        protos.RegisterRoutingServiceServer(s.grpcServer, newRoutingService(s))
        protos.RegisterDiagnosticsServiceServer(s.grpcServer, newDiagnosticsService(s))
        protos.RegisterBackupServiceServer(s.grpcServer, newBackupService(s))

        // Enable reflection for debugging
        reflection.Register(s.grpcServer)

        return s
}

// unaryInterceptor recovers panics and logs errors for unary RPCs.
func (s *Server) unaryInterceptor() grpc.UnaryServerInterceptor {
        return func(
                ctx context.Context,
                req interface{},
                info *grpc.UnaryServerInfo,
                handler grpc.UnaryHandler,
        ) (resp interface{}, err error) {
                defer func() {
                        if r := recover(); r != nil {
                                s.logger.Printf("[PANIC] %s: %v\n%s", info.FullMethod, r, debug.Stack())
                                err = status.Errorf(codes.Internal, "internal error in %s", info.FullMethod)
                }
                }()
                resp, err = handler(ctx, req)
                if err != nil {
                        s.logger.Printf("[ERROR] %s: %v", info.FullMethod, err)
                }
                return
        }
}

// streamInterceptor recovers panics and logs errors for streaming RPCs.
func (s *Server) streamInterceptor() grpc.StreamServerInterceptor {
        return func(
                srv interface{},
                ss grpc.ServerStream,
                info *grpc.StreamServerInfo,
                handler grpc.StreamHandler,
        ) (err error) {
                defer func() {
                        if r := recover(); r != nil {
                                s.logger.Printf("[PANIC STREAM] %s: %v\n%s", info.FullMethod, r, debug.Stack())
                                err = status.Errorf(codes.Internal, "internal error in %s", info.FullMethod)
                }
                }()
                err = handler(srv, ss)
                if err != nil {
                        s.logger.Printf("[ERROR STREAM] %s: %v", info.FullMethod, err)
                }
                return
        }
}

// Start begins listening on the configured gRPC address.
func (s *Server) Start() error {
        cfg := s.cfg.Get()
        addr := cfg.Daemon.GRPCAddr
        if addr == "" {
                addr = "127.0.0.1:50051"
        }

        lis, err := net.Listen("tcp", addr)
        if err != nil {
                return fmt.Errorf("failed to listen on %s: %w", addr, err)
        }

        s.logger.Printf("gRPC server listening on %s", addr)

        go func() {
                if err := s.grpcServer.Serve(lis); err != nil {
                        s.logger.Printf("gRPC server error: %v", err)
                }
        }()

        return nil
}

// Stop gracefully shuts down the gRPC server.
func (s *Server) Stop() {
        s.logger.Println("shutting down gRPC server...")
        s.grpcServer.GracefulStop()
}

// GRPCServer returns the underlying grpc.Server for registration.
func (s *Server) GRPCServer() *grpc.Server {
        return s.grpcServer
}

// Wait blocks until a shutdown signal is received.
func (s *Server) Wait() {
        sigCh := make(chan os.Signal, 1)
        signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
        sig := <-sigCh
        s.logger.Printf("received signal: %v, shutting down...", sig)
        s.Stop()
}

// --- Tunnel Service Implementation ---

type tunnelService struct {
        protos.UnimplementedTunnelServiceServer
        s *Server
}

func newTunnelService(s *Server) *tunnelService {
        return &tunnelService{s: s}
}

func (t *tunnelService) Connect(ctx context.Context, req *protos.ConnectRequest) (*protos.ConnectResponse, error) {
        err := t.s.mgr.Connect(ctx, req.ServerId)
        if err != nil {
                return &protos.ConnectResponse{
                        State: protos.ConnectionState_CONNECTION_STATE_ERROR,
                        Error: err.Error(),
                }, nil
        }

        cfg := t.s.cfg.Get()
        return &protos.ConnectResponse{
                State:     protos.ConnectionState_CONNECTION_STATE_CONNECTED,
                ServerId:  req.ServerId,
                SocksPort: cfg.Tunnel.SOCKSPort,
                HttpPort:  cfg.Tunnel.HTTPPort,
        }, nil
}

func (t *tunnelService) Disconnect(ctx context.Context, req *protos.DisconnectRequest) (*protos.DisconnectResponse, error) {
        _ = ctx
        _ = req
        err := t.s.mgr.Disconnect(req.Force)
        if err != nil {
                return &protos.DisconnectResponse{
                        State: protos.ConnectionState_CONNECTION_STATE_ERROR,
                        Error: err.Error(),
                }, nil
        }
        return &protos.DisconnectResponse{
                State: protos.ConnectionState_CONNECTION_STATE_DISCONNECTED,
        }, nil
}

func (t *tunnelService) WatchConnection(req *protos.WatchConnectionRequest, stream protos.TunnelService_WatchConnectionServer) error {
        _ = req
        ch := t.s.mgr.Events()
        for {
                select {
                case <-stream.Context().Done():
                        return nil
                case event := <-ch:
                        state := connectionStateToProto(event.State)
                        err := stream.Send(&protos.ConnectionStateEvent{
                                State:     state,
                                Message:   event.Message,
                                Error:     event.Error,
                                ServerId:  event.ServerID,
                                Timestamp: timeToProto(event.Time),
                        })
                        if err != nil {
                                return err
                        }
                }
        }
}

func (t *tunnelService) GetConnectionStatus(ctx context.Context, req *protos.GetConnectionStatusRequest) (*protos.GetConnectionStatusResponse, error) {
        _ = ctx
        _ = req
        state := t.s.mgr.State()
        server := t.s.mgr.ActiveServer()
        bytesUp, bytesDown, duration := t.s.mgr.Stats()
        cfg := t.s.cfg.Get()

        resp := &protos.GetConnectionStatusResponse{
                State:     connectionStateToProto(state),
                Mode:      proxyModeToProto(cfg.Tunnel.ProxyMode),
                SocksPort: cfg.Tunnel.SOCKSPort,
                HttpPort:  cfg.Tunnel.HTTPPort,
                Stats: &protos.BandwidthStats{
                        BytesUploaded:   bytesUp,
                        BytesDownloaded: bytesDown,
                        Duration:        durationToProto(duration),
                },
        }

        if server != nil {
                resp.ServerId = server.ID
        }

        return resp, nil
}

func (t *tunnelService) Reconnect(ctx context.Context, req *protos.ReconnectRequest) (*protos.ReconnectResponse, error) {
        err := t.s.mgr.Reconnect(ctx, req.ServerId)
        if err != nil {
                return &protos.ReconnectResponse{
                        State: protos.ConnectionState_CONNECTION_STATE_ERROR,
                        Error: err.Error(),
                }, nil
        }
        return &protos.ReconnectResponse{
                State: protos.ConnectionState_CONNECTION_STATE_CONNECTED,
        }, nil
}

// --- Server Service Implementation ---

type serverService struct {
        protos.UnimplementedServerServiceServer
        s *Server
}

func newServerService(s *Server) *serverService {
        return &serverService{s: s}
}

func (sv *serverService) ListServers(ctx context.Context, req *protos.ListServersRequest) (*protos.ListServersResponse, error) {
        _ = ctx
        _ = req
        servers := sv.s.cfg.GetServers()
        var result []*protos.Server
        for _, s := range servers {
                result = append(result, serverEntryToProto(s))
        }
        return &protos.ListServersResponse{
                Servers: result,
                Total:   int32(len(result)),
        }, nil
}

func (sv *serverService) GetServer(ctx context.Context, req *protos.GetServerRequest) (*protos.Server, error) {
        servers := sv.s.cfg.GetServers()
        for _, s := range servers {
                if s.ID == req.Id {
                        return serverEntryToProto(s), nil
                }
        }
        return nil, fmt.Errorf("server %s not found", req.Id)
}

func (sv *serverService) CreateServer(ctx context.Context, req *protos.CreateServerRequest) (*protos.Server, error) {
        _ = ctx
        if req.Server == nil {
                return nil, fmt.Errorf("server is required")
        }
        entry := protoToServerEntry(req.Server)
        if entry.ID == "" {
                entry.ID = engine.GenerateID()
        }
        err := sv.s.cfg.Update(func(c *config.Config) {
                c.Servers = append(c.Servers, entry)
        })
        if err != nil {
                return nil, fmt.Errorf("failed to save server: %w", err)
        }
        return serverEntryToProto(entry), nil
}

func (sv *serverService) UpdateServer(ctx context.Context, req *protos.UpdateServerRequest) (*protos.Server, error) {
        _ = ctx
        // TODO: implement server update
        return req.Server, nil
}

func (sv *serverService) DeleteServer(ctx context.Context, req *protos.DeleteServerRequest) (*emptypb.Empty, error) {
        _ = ctx
        // TODO: implement server deletion
        return &emptypb.Empty{}, nil
}

func (sv *serverService) TestServers(req *protos.TestServersRequest, stream protos.ServerService_TestServersServer) error {
        // TODO: implement server testing
        _ = req
        return nil
}

func (sv *serverService) ImportServers(ctx context.Context, req *protos.ImportServersRequest) (*protos.ImportServersResponse, error) {
        _ = ctx
        if req.Content == "" {
                return &protos.ImportServersResponse{}, nil
        }
        servers, parseErrors := subscription.Parse([]byte(req.Content))
        var errStrings []string
        for _, pe := range parseErrors {
                errStrings = append(errStrings, pe.Error())
        }
        var importedIDs []string
        var entries []config.ServerEntry
        for _, sc := range servers {
                entry := serverConfigToEntry(sc)
                if entry.ID == "" {
                        entry.ID = engine.GenerateID()
                }
                if req.SubscriptionId != "" {
                        entry.SubscriptionID = req.SubscriptionId
                }
                if req.GroupName != "" {
                        entry.Tags = append(entry.Tags, req.GroupName)
                }
                entries = append(entries, entry)
                importedIDs = append(importedIDs, entry.ID)
        }
        if len(entries) > 0 {
                err := sv.s.cfg.Update(func(c *config.Config) {
                        c.Servers = append(c.Servers, entries...)
                })
                if err != nil {
                        return nil, fmt.Errorf("failed to save imported servers: %w", err)
                }
        }
        return &protos.ImportServersResponse{
                ImportedServerIds: importedIDs,
                TotalParsed:       int32(len(servers) + len(parseErrors)),
                TotalImported:     int32(len(importedIDs)),
                Errors:            errStrings,
        }, nil
}

func (sv *serverService) ExportServer(ctx context.Context, req *protos.ExportServerRequest) (*protos.ExportServerResponse, error) {
        _ = ctx
        // TODO: implement server export
        return &protos.ExportServerResponse{}, nil
}

func (sv *serverService) PingServer(req *protos.PingServerRequest, stream protos.ServerService_PingServerServer) error {
        // TODO: implement server ping streaming
        _ = req
        return nil
}

// --- Subscription Service Implementation ---

type subscriptionService struct {
        protos.UnimplementedSubscriptionServiceServer
        s *Server
}

func newSubscriptionService(s *Server) *subscriptionService {
        return &subscriptionService{s: s}
}

func (ss *subscriptionService) ListSubscriptions(ctx context.Context, req *protos.ListSubscriptionsRequest) (*protos.ListSubscriptionsResponse, error) {
        _ = ctx
        _ = req
        subs := ss.s.cfg.GetSubscriptions()
        var result []*protos.Subscription
        for _, s := range subs {
                result = append(result, subscriptionEntryToProto(s))
        }
        return &protos.ListSubscriptionsResponse{Subscriptions: result}, nil
}

func (ss *subscriptionService) GetSubscription(ctx context.Context, req *protos.GetSubscriptionRequest) (*protos.Subscription, error) {
        _ = ctx
        subs := ss.s.cfg.GetSubscriptions()
        for _, s := range subs {
                if s.ID == req.Id {
                        return subscriptionEntryToProto(s), nil
                }
        }
        return nil, fmt.Errorf("subscription %s not found", req.Id)
}

func (ss *subscriptionService) CreateSubscription(ctx context.Context, req *protos.CreateSubscriptionRequest) (*protos.Subscription, error) {
        _ = ctx
        // TODO: implement subscription creation
        return req.Subscription, nil
}

func (ss *subscriptionService) UpdateSubscription(ctx context.Context, req *protos.UpdateSubscriptionRequest) (*protos.Subscription, error) {
        _ = ctx
        // TODO: implement subscription update
        return req.Subscription, nil
}

func (ss *subscriptionService) DeleteSubscription(ctx context.Context, req *protos.DeleteSubscriptionRequest) (*emptypb.Empty, error) {
        _ = ctx
        // TODO: implement subscription deletion
        return &emptypb.Empty{}, nil
}

func (ss *subscriptionService) RefreshSubscription(ctx context.Context, req *protos.RefreshSubscriptionRequest) (*protos.RefreshSubscriptionResponse, error) {
        _ = ctx
        // TODO: implement subscription refresh (Step 4)
        return &protos.RefreshSubscriptionResponse{}, nil
}

func (ss *subscriptionService) StreamRefresh(req *protos.StreamRefreshRequest, stream protos.SubscriptionService_StreamRefreshServer) error {
        // TODO: implement refresh progress streaming
        _ = req
        return nil
}

// --- Settings Service Implementation ---

type settingsService struct {
        protos.UnimplementedSettingsServiceServer
        s *Server
}

func newSettingsService(s *Server) *settingsService {
        return &settingsService{s: s}
}

func (ss *settingsService) GetSettings(ctx context.Context, req *protos.GetSettingsRequest) (*protos.Settings, error) {
        _ = ctx
        _ = req
        cfg := ss.s.cfg.Get()
        return settingsToProto(&cfg), nil
}

func (ss *settingsService) UpdateSettings(ctx context.Context, req *protos.UpdateSettingsRequest) (*protos.Settings, error) {
        _ = ctx
        // TODO: implement settings update
        return req.Settings, nil
}

func (ss *settingsService) ResetSettings(ctx context.Context, req *protos.ResetSettingsRequest) (*protos.Settings, error) {
        _ = ctx
        _ = req
        return &protos.Settings{}, nil
}

// --- Routing Service Implementation ---

type routingService struct {
        protos.UnimplementedRoutingServiceServer
        s *Server
}

func newRoutingService(s *Server) *routingService {
        return &routingService{s: s}
}

func (rs *routingService) GetRouting(ctx context.Context, req *protos.GetRoutingRequest) (*protos.RoutingConfig, error) {
        _ = ctx
        _ = req
        routing := rs.s.cfg.GetRouting()
        return routingConfigToProto(&routing), nil
}

func (rs *routingService) UpdateRouting(ctx context.Context, req *protos.UpdateRoutingRequest) (*protos.RoutingConfig, error) {
        _ = ctx
        // TODO: implement routing update
        return req.Routing, nil
}

func (rs *routingService) ListRules(ctx context.Context, req *protos.ListRulesRequest) (*protos.ListRulesResponse, error) {
        _ = ctx
        _ = req
        routing := rs.s.cfg.GetRouting()
        var rules []*protos.RoutingRule
        for _, r := range routing.Rules {
                rules = append(rules, routingRuleToProto(r))
        }
        return &protos.ListRulesResponse{Rules: rules}, nil
}

func (rs *routingService) CreateRule(ctx context.Context, req *protos.CreateRuleRequest) (*protos.RoutingRule, error) {
        _ = ctx
        // TODO: implement rule creation
        return req.Rule, nil
}

func (rs *routingService) UpdateRule(ctx context.Context, req *protos.UpdateRuleRequest) (*protos.RoutingRule, error) {
        _ = ctx
        // TODO: implement rule update
        return req.Rule, nil
}

func (rs *routingService) DeleteRule(ctx context.Context, req *protos.DeleteRuleRequest) (*emptypb.Empty, error) {
        _ = ctx
        // TODO: implement rule deletion
        return &emptypb.Empty{}, nil
}

func (rs *routingService) ReorderRules(ctx context.Context, req *protos.ReorderRulesRequest) (*protos.ReorderRulesResponse, error) {
        _ = ctx
        // TODO: implement rule reordering
        return &protos.ReorderRulesResponse{}, nil
}

// --- Diagnostics Service Implementation ---

type diagnosticsService struct {
        protos.UnimplementedDiagnosticsServiceServer
        s *Server
}

func newDiagnosticsService(s *Server) *diagnosticsService {
        return &diagnosticsService{s: s}
}

func (ds *diagnosticsService) HealthCheck(ctx context.Context, req *protos.HealthCheckRequest) (*protos.HealthCheckResponse, error) {
        _ = ctx
        _ = req
        now := timeToProto(time.Now())
        return &protos.HealthCheckResponse{
                Healthy:   true,
                Version:   "0.1.0",
                CheckedAt: now,
                Checks:    []string{"daemon: ok", "grpc: ok"},
        }, nil
}

func (ds *diagnosticsService) GetDaemonInfo(ctx context.Context, req *protos.GetDaemonInfoRequest) (*protos.DaemonInfo, error) {
        _ = ctx
        _ = req
        return &protos.DaemonInfo{
                Version: "0.1.0",
                Os:      "windows",
                Arch:    "amd64",
        }, nil
}

func (ds *diagnosticsService) StreamLogs(req *protos.StreamLogsRequest, stream protos.DiagnosticsService_StreamLogsServer) error {
        // TODO: implement log streaming
        _ = req
        return nil
}

func (ds *diagnosticsService) SpeedTest(req *protos.SpeedTestRequest, stream protos.DiagnosticsService_SpeedTestServer) error {
        // TODO: implement speed testing
        _ = req
        return nil
}

func (ds *diagnosticsService) GenerateReport(ctx context.Context, req *protos.GenerateReportRequest) (*protos.GenerateReportResponse, error) {
        // TODO: implement report generation
        _ = ctx
        return &protos.GenerateReportResponse{}, nil
}

// --- Backup Service Implementation ---

type backupService struct {
        protos.UnimplementedBackupServiceServer
        s *Server
}

func newBackupService(s *Server) *backupService {
        return &backupService{s: s}
}

func (bs *backupService) ExportBackup(ctx context.Context, req *protos.ExportBackupRequest) (*protos.ExportBackupResponse, error) {
        _ = ctx
        // TODO: implement backup export
        return &protos.ExportBackupResponse{}, nil
}

func (bs *backupService) ImportBackup(ctx context.Context, req *protos.ImportBackupRequest) (*protos.ImportBackupResponse, error) {
        _ = ctx
        // TODO: implement backup import
        return &protos.ImportBackupResponse{}, nil
}

// --- Proto Conversion Helpers ---

func connectionStateToProto(s engine.ConnectionState) protos.ConnectionState {
        switch s {
        case engine.StateDisconnected:
                return protos.ConnectionState_CONNECTION_STATE_DISCONNECTED
        case engine.StateConnecting:
                return protos.ConnectionState_CONNECTION_STATE_CONNECTING
        case engine.StateConnected:
                return protos.ConnectionState_CONNECTION_STATE_CONNECTED
        case engine.StateReconnecting:
                return protos.ConnectionState_CONNECTION_STATE_RECONNECTING
        case engine.StateDisconnecting:
                return protos.ConnectionState_CONNECTION_STATE_DISCONNECTING
        case engine.StateError:
                return protos.ConnectionState_CONNECTION_STATE_ERROR
        case engine.StateFallback, engine.StateFallbackConnected:
                return protos.ConnectionState_CONNECTION_STATE_CONNECTED
        default:
                return protos.ConnectionState_CONNECTION_STATE_UNSPECIFIED
        }
}

func proxyModeToProto(mode string) protos.ProxyMode {
        switch mode {
        case "system":
                return protos.ProxyMode_PROXY_MODE_SYSTEM
        case "socks":
                return protos.ProxyMode_PROXY_MODE_SOCKS
        case "http":
                return protos.ProxyMode_PROXY_MODE_HTTP
        case "pac":
                return protos.ProxyMode_PROXY_MODE_PAC
        default:
                return protos.ProxyMode_PROXY_MODE_SYSTEM
        }
}

func timeToProto(t time.Time) *protos.Timestamp {
        if t.IsZero() {
                return nil
        }
        return &protos.Timestamp{
                Seconds: t.Unix(),
                Nanos:   int32(t.Nanosecond()),
        }
}

func durationToProto(d time.Duration) *protos.Duration {
        return &protos.Duration{
                Seconds: int64(d.Seconds()),
                Nanos:   int32(d.Nanoseconds() % 1e9),
        }
}

func serverEntryToProto(s config.ServerEntry) *protos.Server {
        pb := &protos.Server{
                Id: s.ID, Name: s.Name, Host: s.Host, Port: s.Port,
                Favorite: s.Favorite, SortOrder: s.SortOrder,
                Tags: s.Tags, SubscriptionId: s.SubscriptionID,
        }
        switch s.Protocol {
        case "vless", "vmess":
                if s.Protocol == "vless" { pb.Protocol = protos.Protocol_PROTOCOL_VLESS } else { pb.Protocol = protos.Protocol_PROTOCOL_VMESS }
                if s.XrayConfig != nil {
                        pb.Xray = &protos.XrayConfig{
                                Uuid: s.XrayConfig.UUID, Flow: s.XrayConfig.Flow,
                                Sni: s.XrayConfig.SNI, Fingerprint: s.XrayConfig.Fingerprint,
                                Alpn: s.XrayConfig.ALPN, PublicKey: s.XrayConfig.PublicKey,
                                ShortId: s.XrayConfig.ShortID, KcpSeed: s.XrayConfig.KCPSeed,
                                XhttpPath: s.XrayConfig.XHTTPPath, XhttpMode: s.XrayConfig.XHTTPMode,
                                WsPath: s.XrayConfig.WSPath, GrpcService: s.XrayConfig.GRPCService,
                                AllowInsecure: s.XrayConfig.AllowInsecure,
                        }
                }
        case "wireguard":
                pb.Protocol = protos.Protocol_PROTOCOL_WIREGUARD
                if s.WGConfig != nil {
                        pb.Wireguard = &protos.WireGuardConfig{
                                PrivateKey: s.WGConfig.PrivateKey, PublicKey: s.WGConfig.PublicKey,
                                PresharedKey: s.WGConfig.PresharedKey, LocalAddress: s.WGConfig.LocalAddress,
                                DnsServers: s.WGConfig.DNSServers, Mtu: s.WGConfig.MTU,
                                PersistentKeepalive: s.WGConfig.PersistentKeepalive, AllowedIps: s.WGConfig.AllowedIPs,
                        }
                }
        case "hysteria":
                pb.Protocol = protos.Protocol_PROTOCOL_HYSTERIA
                if s.HysteriaConfig != nil {
                        pb.Hysteria = &protos.HysteriaConfig{
                                AuthPassword: s.HysteriaConfig.AuthPassword, Sni: s.HysteriaConfig.SNI,
                                Insecure: s.HysteriaConfig.Insecure, Alpn: s.HysteriaConfig.ALPN,
                                ObfsPassword: s.HysteriaConfig.ObfsPassword,
                                BandwidthUp: s.HysteriaConfig.BandwidthUp, BandwidthDown: s.HysteriaConfig.BandwidthDown,
                                FastOpen: s.HysteriaConfig.FastOpen,
                        }
                }
        case "amneziawg":
                pb.Protocol = protos.Protocol_PROTOCOL_AMNEZIAWG
                if s.AmneziaConfig != nil {
                        pb.Amneziawg = &protos.AmneziaWGConfig{
                                PrivateKey: s.AmneziaConfig.PrivateKey, PublicKey: s.AmneziaConfig.PublicKey,
                                PresharedKey: s.AmneziaConfig.PresharedKey, LocalAddress: s.AmneziaConfig.LocalAddress,
                                DnsServers: s.AmneziaConfig.DNSServers, Mtu: s.AmneziaConfig.MTU,
                        }
                }
        }
        return pb
}

func subscriptionEntryToProto(s config.SubscriptionEntry) *protos.Subscription {
        return &protos.Subscription{
                Id:              s.ID,
                Name:            s.Name,
                Url:             s.URL,
                Username:        s.Username,
                Password:        s.Password,
                RefreshInterval: s.RefreshInterval,
                Filter:          s.Filter,
                Enabled:         s.Enabled,
        }
}

func routingConfigToProto(r *config.RoutingConfig) *protos.RoutingConfig {
        var rules []*protos.RoutingRule
        for _, rule := range r.Rules {
                rules = append(rules, routingRuleToProto(rule))
        }
        return &protos.RoutingConfig{
                DomainStrategy: r.DomainStrategy,
                Rules:          rules,
        }
}

func routingRuleToProto(r config.RoutingRule) *protos.RoutingRule {
        pb := &protos.RoutingRule{
                Id:      r.ID,
                Name:    r.Name,
                Enabled: r.Enabled,
                Domains: r.Domains,
                IpCidrs: r.IPCidrs,
                GeoipCodes: r.GeoIPCodes,
                Processes: r.Processes,
                Ports:     r.Ports,
                Protocols: r.Protocols,
        }

        switch r.Action {
        case "proxy":
                pb.Action = protos.RuleAction_RULE_ACTION_PROXY
        case "direct":
                pb.Action = protos.RuleAction_RULE_ACTION_DIRECT
        case "block":
                pb.Action = protos.RuleAction_RULE_ACTION_BLOCK
        }

        return pb
}

func settingsToProto(cfg *config.Config) *protos.Settings {
        return &protos.Settings{
                ProxyMode:          proxyModeToProto(cfg.Tunnel.ProxyMode),
                SocksPort:           cfg.Tunnel.SOCKSPort,
                HttpPort:            cfg.Tunnel.HTTPPort,
                DnsServers:          cfg.DNS.DNSServers,
                AutoConnect:         cfg.Daemon.AutoConnect,
                ConnectOnStartup:    cfg.Daemon.ConnectOnStartup,
                KillSwitch:          cfg.Daemon.KillSwitch,
                SplitTunneling:      len(cfg.Routing.Rules) > 0,
                AllowLan:            cfg.Daemon.AllowLAN,
                ConnectionTimeout:   cfg.Tunnel.ConnectionTimeout,
                ReconnectAttempts:   cfg.Tunnel.ReconnectAttempts,
                ReconnectDelay:      cfg.Tunnel.ReconnectDelay,
                Language:            cfg.Daemon.Language,
                Theme:               cfg.Daemon.Theme,
                ActiveServerId:      cfg.Tunnel.ActiveServerID,
                Routing:             routingConfigToProto(&cfg.Routing),
        }
}



// securityToString converts a proto Security enum to config string.
func securityToString(s protos.Security) string {
        switch s {
        case protos.Security_SECURITY_NONE: return "none"
        case protos.Security_SECURITY_TLS: return "tls"
        case protos.Security_SECURITY_REALITY: return "reality"
        default: return ""
        }
}

// transportToString converts a proto Transport enum to config string.
func transportToString(t protos.Transport) string {
        switch t {
        case protos.Transport_TRANSPORT_TCP: return "tcp"
        case protos.Transport_TRANSPORT_WS: return "ws"
        case protos.Transport_TRANSPORT_GRPC: return "grpc"
        case protos.Transport_TRANSPORT_KCP: return "kcp"
        case protos.Transport_TRANSPORT_XHTTP: return "xhttp"
        case protos.Transport_TRANSPORT_HTTPUPGRADE: return "httpupgrade"
        case protos.Transport_TRANSPORT_QUIC: return "quic"
        default: return ""
        }
}

// protoToServerEntry converts a proto Server message to a config ServerEntry.
func protoToServerEntry(pb *protos.Server) config.ServerEntry {
        entry := config.ServerEntry{
                ID: pb.Id, Name: pb.Name, Host: pb.Host, Port: pb.Port,
                Favorite: pb.Favorite, SortOrder: pb.SortOrder,
                Tags: pb.Tags, SubscriptionID: pb.SubscriptionId,
        }
        switch pb.Protocol {
        case protos.Protocol_PROTOCOL_VLESS: entry.Protocol = "vless"
        case protos.Protocol_PROTOCOL_VMESS: entry.Protocol = "vmess"
        case protos.Protocol_PROTOCOL_WIREGUARD: entry.Protocol = "wireguard"
        case protos.Protocol_PROTOCOL_HYSTERIA: entry.Protocol = "hysteria"
        case protos.Protocol_PROTOCOL_AMNEZIAWG: entry.Protocol = "amneziawg"
        }
        if pb.Xray != nil {
                entry.XrayConfig = &config.XrayConfigEntry{
                        UUID: pb.Xray.Uuid, Flow: pb.Xray.Flow,
                        Security: securityToString(pb.Xray.Security), Transport: transportToString(pb.Xray.Transport),
                        SNI: pb.Xray.Sni, Fingerprint: pb.Xray.Fingerprint,
                        ALPN: pb.Xray.Alpn, PublicKey: pb.Xray.PublicKey,
                        ShortID: pb.Xray.ShortId, KCPSeed: pb.Xray.KcpSeed,
                        XHTTPPath: pb.Xray.XhttpPath, XHTTPMode: pb.Xray.XhttpMode,
                        WSPath: pb.Xray.WsPath, GRPCService: pb.Xray.GrpcService,
                        AllowInsecure: pb.Xray.AllowInsecure,
                }
        }
        if pb.Wireguard != nil {
                entry.WGConfig = &config.WGConfigEntry{
                        PrivateKey: pb.Wireguard.PrivateKey, PublicKey: pb.Wireguard.PublicKey,
                        PresharedKey: pb.Wireguard.PresharedKey, LocalAddress: pb.Wireguard.LocalAddress,
                        DNSServers: pb.Wireguard.DnsServers, MTU: pb.Wireguard.Mtu,
                        PersistentKeepalive: pb.Wireguard.PersistentKeepalive, AllowedIPs: pb.Wireguard.AllowedIps,
                }
        }
        if pb.Hysteria != nil {
                entry.HysteriaConfig = &config.HysteriaConfigEntry{
                        AuthPassword: pb.Hysteria.AuthPassword, SNI: pb.Hysteria.Sni,
                        Insecure: pb.Hysteria.Insecure, ALPN: pb.Hysteria.Alpn,
                        ObfsPassword: pb.Hysteria.ObfsPassword,
                        BandwidthUp: pb.Hysteria.BandwidthUp, BandwidthDown: pb.Hysteria.BandwidthDown,
                        FastOpen: pb.Hysteria.FastOpen,
                }
        }
        if pb.Amneziawg != nil {
                entry.AmneziaConfig = &config.AmneziaConfigEntry{
                        PrivateKey: pb.Amneziawg.PrivateKey, PublicKey: pb.Amneziawg.PublicKey,
                        PresharedKey: pb.Amneziawg.PresharedKey, LocalAddress: pb.Amneziawg.LocalAddress,
                        DNSServers: pb.Amneziawg.DnsServers, MTU: pb.Amneziawg.Mtu,
                }
        }
        return entry
}

// serverConfigToEntry converts an engine ServerConfig to a config ServerEntry.
func serverConfigToEntry(sc engine.ServerConfig) config.ServerEntry {
        entry := config.ServerEntry{
                ID: sc.ID, Name: sc.Name, Host: sc.Host, Port: sc.Port,
                Protocol: string(sc.Protocol), Tags: sc.Tags,
                Favorite: sc.Favorite, SortOrder: sc.SortOrder, SubscriptionID: sc.SubscriptionID,
        }
        switch sc.Protocol {
        case engine.ProtocolVLESS, engine.ProtocolVMESS:
                entry.XrayConfig = &config.XrayConfigEntry{
                        UUID: sc.UUID, Flow: sc.Flow, Security: sc.Security,
                        Transport: sc.Transport, SNI: sc.SNI, Fingerprint: sc.Fingerprint,
                        ALPN: sc.ALPN, PublicKey: sc.PublicKey, ShortID: sc.ShortID,
                        KCPSeed: sc.KCPSeed, XHTTPPath: sc.XHTTPPath, XHTTPMode: sc.XHTTPMode,
                        WSPath: sc.WSPath, GRPCService: sc.GRPCService, AllowInsecure: sc.AllowInsecure,
                }
        case engine.ProtocolWireGuard:
                entry.WGConfig = &config.WGConfigEntry{
                        PrivateKey: sc.WGPrivateKey, PublicKey: sc.WGPublicKey,
                        PresharedKey: sc.WGPresharedKey, LocalAddress: sc.WGLocalAddress,
                        DNSServers: sc.WGDNSServers, AllowedIPs: sc.WGAllowedIPs,
                }
        case engine.ProtocolHysteria:
                entry.HysteriaConfig = &config.HysteriaConfigEntry{
                        AuthPassword: sc.HysteriaAuth, SNI: sc.HysteriaSNI,
                        Insecure: sc.HysteriaInsecure, ALPN: sc.HysteriaALPN,
                        ObfsPassword: sc.HysteriaObfs,
                        BandwidthUp: sc.HysteriaBwUp, BandwidthDown: sc.HysteriaBwDown,
                        FastOpen: sc.HysteriaFastOpen,
                }
        case engine.ProtocolAmneziaWG:
                entry.AmneziaConfig = &config.AmneziaConfigEntry{
                        PrivateKey: sc.AmneziaPrivateKey, PublicKey: sc.AmneziaPublicKey,
                        PresharedKey: sc.AmneziaPresharedKey, LocalAddress: sc.AmneziaLocalAddr,
                        DNSServers: sc.AmneziaDNS,
                        Jc: sc.AmneziaJc, Jmin: sc.AmneziaJmin, Jmax: sc.AmneziaJmax,
                        S1: sc.AmneziaS1, S2: sc.AmneziaS2,
                        H1: sc.AmneziaH1, H2: sc.AmneziaH2, H3: sc.AmneziaH3, H4: sc.AmneziaH4,
                }
        }
        return entry
}

// NewGRPCClient creates a gRPC client connection for use by Tauri.
func NewGRPCClient(addr string) (protos.TunnelServiceClient, protos.ServerServiceClient, protos.SubscriptionServiceClient, protos.SettingsServiceClient, protos.RoutingServiceClient, protos.DiagnosticsServiceClient, error) {
        conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
        if err != nil {
                return nil, nil, nil, nil, nil, nil, err
        }
        return protos.NewTunnelServiceClient(conn), protos.NewServerServiceClient(conn), protos.NewSubscriptionServiceClient(conn), protos.NewSettingsServiceClient(conn), protos.NewRoutingServiceClient(conn), protos.NewDiagnosticsServiceClient(conn), nil
}