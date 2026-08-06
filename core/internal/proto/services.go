// Package proto contains hand-written gRPC service interfaces matching tunnelcraft.proto.
// TODO: regenerate from .proto with protoc when available.
package proto

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ============================================================================
// Service names
// ============================================================================

const (
	TunnelService_ServiceDesc      = grpc.ServiceDesc{ServiceName: "tunnelcraft.v1.TunnelService"}
	ServerService_ServiceDesc      = grpc.ServiceDesc{ServiceName: "tunnelcraft.v1.ServerService"}
	SubscriptionService_ServiceDesc = grpc.ServiceDesc{ServiceName: "tunnelcraft.v1.SubscriptionService"}
	SettingsService_ServiceDesc     = grpc.ServiceDesc{ServiceName: "tunnelcraft.v1.SettingsService"}
	RoutingService_ServiceDesc     = grpc.ServiceDesc{ServiceName: "tunnelcraft.v1.RoutingService"}
	DiagnosticsService_ServiceDesc = grpc.ServiceDesc{ServiceName: "tunnelcraft.v1.DiagnosticsService"}
	BackupService_ServiceDesc      = grpc.ServiceDesc{ServiceName: "tunnelcraft.v1.BackupService"}
)

// ============================================================================
// TunnelService
// ============================================================================

type TunnelServiceClient interface {
	Connect(ctx context.Context, in *ConnectRequest, opts ...grpc.CallOption) (*ConnectResponse, error)
	Disconnect(ctx context.Context, in *DisconnectRequest, opts ...grpc.CallOption) (*DisconnectResponse, error)
	WatchConnection(ctx context.Context, in *WatchConnectionRequest, opts ...grpc.CallOption) (TunnelService_WatchConnectionClient, error)
	GetConnectionStatus(ctx context.Context, in *GetConnectionStatusRequest, opts ...grpc.CallOption) (*GetConnectionStatusResponse, error)
	Reconnect(ctx context.Context, in *ReconnectRequest, opts ...grpc.CallOption) (*ReconnectResponse, error)
}

type TunnelService_WatchConnectionClient interface {
	Recv() (*ConnectionStateEvent, error)
	grpc.ClientStream
}

type TunnelServiceServer interface {
	Connect(context.Context, *ConnectRequest) (*ConnectResponse, error)
	Disconnect(context.Context, *DisconnectRequest) (*DisconnectResponse, error)
	WatchConnection(*WatchConnectionRequest, TunnelService_WatchConnectionServer) error
	GetConnectionStatus(context.Context, *GetConnectionStatusRequest) (*GetConnectionStatusResponse, error)
	Reconnect(context.Context, *ReconnectRequest) (*ReconnectResponse, error)
}

type TunnelService_WatchConnectionServer interface {
	Send(*ConnectionStateEvent) error
	grpc.ServerStream
}

type UnimplementedTunnelServiceServer struct{}

func (UnimplementedTunnelServiceServer) Connect(context.Context, *ConnectRequest) (*ConnectResponse, error) {
	return nil, nil
}
func (UnimplementedTunnelServiceServer) Disconnect(context.Context, *DisconnectRequest) (*DisconnectResponse, error) {
	return nil, nil
}
func (UnimplementedTunnelServiceServer) WatchConnection(*WatchConnectionRequest, TunnelService_WatchConnectionServer) error {
	return nil
}
func (UnimplementedTunnelServiceServer) GetConnectionStatus(context.Context, *GetConnectionStatusRequest) (*GetConnectionStatusResponse, error) {
	return nil, nil
}
func (UnimplementedTunnelServiceServer) Reconnect(context.Context, *ReconnectRequest) (*ReconnectResponse, error) {
	return nil, nil
}

func RegisterTunnelServiceServer(s *grpc.Server, srv TunnelServiceServer) {
	s.RegisterService(&TunnelService_ServiceDesc, srv)
}

func NewTunnelServiceClient(cc grpc.ClientConnInterface) TunnelServiceClient {
	return nil // stub
}

// ============================================================================
// ServerService
// ============================================================================

type ServerServiceClient interface {
	ListServers(ctx context.Context, in *ListServersRequest, opts ...grpc.CallOption) (*ListServersResponse, error)
	GetServer(ctx context.Context, in *GetServerRequest, opts ...grpc.CallOption) (*Server, error)
	CreateServer(ctx context.Context, in *CreateServerRequest, opts ...grpc.CallOption) (*Server, error)
	UpdateServer(ctx context.Context, in *UpdateServerRequest, opts ...grpc.CallOption) (*Server, error)
	DeleteServer(ctx context.Context, in *DeleteServerRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	TestServers(ctx context.Context, in *TestServersRequest, opts ...grpc.CallOption) (ServerService_TestServersClient, error)
	ImportServers(ctx context.Context, in *ImportServersRequest, opts ...grpc.CallOption) (*ImportServersResponse, error)
	ExportServer(ctx context.Context, in *ExportServerRequest, opts ...grpc.CallOption) (*ExportServerResponse, error)
	PingServer(ctx context.Context, in *PingServerRequest, opts ...grpc.CallOption) (ServerService_PingServerClient, error)
}

type ServerService_TestServersServer interface {
	Send(*ServerTestResult) error
	grpc.ServerStream
}

type ServerService_TestServersClient interface {
	Recv() (*ServerTestResult, error)
	grpc.ClientStream
}

type ServerService_PingServerServer interface {
	Send(*LatencySample) error
	grpc.ServerStream
}

type ServerService_PingServerClient interface {
	Recv() (*LatencySample, error)
	grpc.ClientStream
}

type ServerServiceServer interface {
	ListServers(context.Context, *ListServersRequest) (*ListServersResponse, error)
	GetServer(context.Context, *GetServerRequest) (*Server, error)
	CreateServer(context.Context, *CreateServerRequest) (*Server, error)
	UpdateServer(context.Context, *UpdateServerRequest) (*Server, error)
	DeleteServer(context.Context, *DeleteServerRequest) (*emptypb.Empty, error)
	TestServers(*TestServersRequest, ServerService_TestServersServer) error
	ImportServers(context.Context, *ImportServersRequest) (*ImportServersResponse, error)
	ExportServer(context.Context, *ExportServerRequest) (*ExportServerResponse, error)
	PingServer(*PingServerRequest, ServerService_PingServerServer) error
}

type UnimplementedServerServiceServer struct{}

func (UnimplementedServerServiceServer) ListServers(context.Context, *ListServersRequest) (*ListServersResponse, error) { return nil, nil }
func (UnimplementedServerServiceServer) GetServer(context.Context, *GetServerRequest) (*Server, error)          { return nil, nil }
func (UnimplementedServerServiceServer) CreateServer(context.Context, *CreateServerRequest) (*Server, error)    { return nil, nil }
func (UnimplementedServerServiceServer) UpdateServer(context.Context, *UpdateServerRequest) (*Server, error)    { return nil, nil }
func (UnimplementedServerServiceServer) DeleteServer(context.Context, *DeleteServerRequest) (*emptypb.Empty, error) { return nil, nil }
func (UnimplementedServerServiceServer) TestServers(*TestServersRequest, ServerService_TestServersServer) error   { return nil }
func (UnimplementedServerServiceServer) ImportServers(context.Context, *ImportServersRequest) (*ImportServersResponse, error) { return nil, nil }
func (UnimplementedServerServiceServer) ExportServer(context.Context, *ExportServerRequest) (*ExportServerResponse, error) { return nil, nil }
func (UnimplementedServerServiceServer) PingServer(*PingServerRequest, ServerService_PingServerServer) error     { return nil }

func RegisterServerServiceServer(s *grpc.Server, srv ServerServiceServer) {
	s.RegisterService(&ServerService_ServiceDesc, srv)
}

func NewServerServiceClient(cc grpc.ClientConnInterface) ServerServiceClient { return nil }

// ============================================================================
// SubscriptionService
// ============================================================================

type SubscriptionServiceClient interface {
	ListSubscriptions(ctx context.Context, in *ListSubscriptionsRequest, opts ...grpc.CallOption) (*ListSubscriptionsResponse, error)
	GetSubscription(ctx context.Context, in *GetSubscriptionRequest, opts ...grpc.CallOption) (*Subscription, error)
	CreateSubscription(ctx context.Context, in *CreateSubscriptionRequest, opts ...grpc.CallOption) (*Subscription, error)
	UpdateSubscription(ctx context.Context, in *UpdateSubscriptionRequest, opts ...grpc.CallOption) (*Subscription, error)
	DeleteSubscription(ctx context.Context, in *DeleteSubscriptionRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	RefreshSubscription(ctx context.Context, in *RefreshSubscriptionRequest, opts ...grpc.CallOption) (*RefreshSubscriptionResponse, error)
	StreamRefresh(ctx context.Context, in *StreamRefreshRequest, opts ...grpc.CallOption) (SubscriptionService_StreamRefreshClient, error)
}

type SubscriptionService_StreamRefreshServer interface {
	Send(*RefreshProgress) error
	grpc.ServerStream
}

type SubscriptionService_StreamRefreshClient interface {
	Recv() (*RefreshProgress, error)
	grpc.ClientStream
}

type SubscriptionServiceServer interface {
	ListSubscriptions(context.Context, *ListSubscriptionsRequest) (*ListSubscriptionsResponse, error)
	GetSubscription(context.Context, *GetSubscriptionRequest) (*Subscription, error)
	CreateSubscription(context.Context, *CreateSubscriptionRequest) (*Subscription, error)
	UpdateSubscription(context.Context, *UpdateSubscriptionRequest) (*Subscription, error)
	DeleteSubscription(context.Context, *DeleteSubscriptionRequest) (*emptypb.Empty, error)
	RefreshSubscription(context.Context, *RefreshSubscriptionRequest) (*RefreshSubscriptionResponse, error)
	StreamRefresh(*StreamRefreshRequest, SubscriptionService_StreamRefreshServer) error
}

type UnimplementedSubscriptionServiceServer struct{}

func (UnimplementedSubscriptionServiceServer) ListSubscriptions(context.Context, *ListSubscriptionsRequest) (*ListSubscriptionsResponse, error) { return nil, nil }
func (UnimplementedSubscriptionServiceServer) GetSubscription(context.Context, *GetSubscriptionRequest) (*Subscription, error)           { return nil, nil }
func (UnimplementedSubscriptionServiceServer) CreateSubscription(context.Context, *CreateSubscriptionRequest) (*Subscription, error) { return nil, nil }
func (UnimplementedSubscriptionServiceServer) UpdateSubscription(context.Context, *UpdateSubscriptionRequest) (*Subscription, error) { return nil, nil }
func (UnimplementedSubscriptionServiceServer) DeleteSubscription(context.Context, *DeleteSubscriptionRequest) (*emptypb.Empty, error) { return nil, nil }
func (UnimplementedSubscriptionServiceServer) RefreshSubscription(context.Context, *RefreshSubscriptionRequest) (*RefreshSubscriptionResponse, error) { return nil, nil }
func (UnimplementedSubscriptionServiceServer) StreamRefresh(*StreamRefreshRequest, SubscriptionService_StreamRefreshServer) error { return nil }

func RegisterSubscriptionServiceServer(s *grpc.Server, srv SubscriptionServiceServer) {
	s.RegisterService(&SubscriptionService_ServiceDesc, srv)
}

func NewSubscriptionServiceClient(cc grpc.ClientConnInterface) SubscriptionServiceClient { return nil }

// ============================================================================
// SettingsService
// ============================================================================

type SettingsServiceClient interface {
	GetSettings(context.Context, *GetSettingsRequest, ...grpc.CallOption) (*Settings, error)
	UpdateSettings(context.Context, *UpdateSettingsRequest, ...grpc.CallOption) (*Settings, error)
	ResetSettings(context.Context, *ResetSettingsRequest, ...grpc.CallOption) (*Settings, error)
}

type SettingsServiceServer interface {
	GetSettings(context.Context, *GetSettingsRequest) (*Settings, error)
	UpdateSettings(context.Context, *UpdateSettingsRequest) (*Settings, error)
	ResetSettings(context.Context, *ResetSettingsRequest) (*Settings, error)
}

type UnimplementedSettingsServiceServer struct{}

func (UnimplementedSettingsServiceServer) GetSettings(context.Context, *GetSettingsRequest) (*Settings, error)    { return nil, nil }
func (UnimplementedSettingsServiceServer) UpdateSettings(context.Context, *UpdateSettingsRequest) (*Settings, error) { return nil, nil }
func (UnimplementedSettingsServiceServer) ResetSettings(context.Context, *ResetSettingsRequest) (*Settings, error)    { return nil, nil }

func RegisterSettingsServiceServer(s *grpc.Server, srv SettingsServiceServer) {
	s.RegisterService(&SettingsService_ServiceDesc, srv)
}

func NewSettingsServiceClient(cc grpc.ClientConnInterface) SettingsServiceClient { return nil }

// ============================================================================
// RoutingService
// ============================================================================

type RoutingServiceClient interface {
	GetRouting(context.Context, *GetRoutingRequest, ...grpc.CallOption) (*RoutingConfig, error)
	UpdateRouting(context.Context, *UpdateRoutingRequest, ...grpc.CallOption) (*RoutingConfig, error)
	ListRules(context.Context, *ListRulesRequest, ...grpc.CallOption) (*ListRulesResponse, error)
	CreateRule(context.Context, *CreateRuleRequest, ...grpc.CallOption) (*RoutingRule, error)
	UpdateRule(context.Context, *UpdateRuleRequest, ...grpc.CallOption) (*RoutingRule, error)
	DeleteRule(context.Context, *DeleteRuleRequest, ...grpc.CallOption) (*emptypb.Empty, error)
	ReorderRules(context.Context, *ReorderRulesRequest, ...grpc.CallOption) (*ReorderRulesResponse, error)
}

type RoutingServiceServer interface {
	GetRouting(context.Context, *GetRoutingRequest) (*RoutingConfig, error)
	UpdateRouting(context.Context, *UpdateRoutingRequest) (*RoutingConfig, error)
	ListRules(context.Context, *ListRulesRequest) (*ListRulesResponse, error)
	CreateRule(context.Context, *CreateRuleRequest) (*RoutingRule, error)
	UpdateRule(context.Context, *UpdateRuleRequest) (*RoutingRule, error)
	DeleteRule(context.Context, *DeleteRuleRequest) (*emptypb.Empty, error)
	ReorderRules(context.Context, *ReorderRulesRequest) (*ReorderRulesResponse, error)
}

type UnimplementedRoutingServiceServer struct{}

func (UnimplementedRoutingServiceServer) GetRouting(context.Context, *GetRoutingRequest) (*RoutingConfig, error)       { return nil, nil }
func (UnimplementedRoutingServiceServer) UpdateRouting(context.Context, *UpdateRoutingRequest) (*RoutingConfig, error) { return nil, nil }
func (UnimplementedRoutingServiceServer) ListRules(context.Context, *ListRulesRequest) (*ListRulesResponse, error)     { return nil, nil }
func (UnimplementedRoutingServiceServer) CreateRule(context.Context, *CreateRuleRequest) (*RoutingRule, error)       { return nil, nil }
func (UnimplementedRoutingServiceServer) UpdateRule(context.Context, *UpdateRuleRequest) (*RoutingRule, error)       { return nil, nil }
func (UnimplementedRoutingServiceServer) DeleteRule(context.Context, *DeleteRuleRequest) (*emptypb.Empty, error)     { return nil, nil }
func (UnimplementedRoutingServiceServer) ReorderRules(context.Context, *ReorderRulesRequest) (*ReorderRulesResponse, error) { return nil, nil }

func RegisterRoutingServiceServer(s *grpc.Server, srv RoutingServiceServer) {
	s.RegisterService(&RoutingService_ServiceDesc, srv)
}

func NewRoutingServiceClient(cc grpc.ClientConnInterface) RoutingServiceClient { return nil }

// ============================================================================
// DiagnosticsService
// ============================================================================

type DiagnosticsServiceClient interface {
	StreamLogs(ctx context.Context, in *StreamLogsRequest, opts ...grpc.CallOption) (DiagnosticsService_StreamLogsClient, error)
	SpeedTest(ctx context.Context, in *SpeedTestRequest, opts ...grpc.CallOption) (DiagnosticsService_SpeedTestClient, error)
	HealthCheck(ctx context.Context, in *HealthCheckRequest, opts ...grpc.CallOption) (*HealthCheckResponse, error)
	GetDaemonInfo(ctx context.Context, in *GetDaemonInfoRequest, opts ...grpc.CallOption) (*DaemonInfo, error)
	GenerateReport(ctx context.Context, in *GenerateReportRequest, opts ...grpc.CallOption) (*GenerateReportResponse, error)
}

type DiagnosticsService_StreamLogsServer interface {
	Send(*LogEntry) error
	grpc.ServerStream
}

type DiagnosticsService_StreamLogsClient interface {
	Recv() (*LogEntry, error)
	grpc.ClientStream
}

type DiagnosticsService_SpeedTestServer interface {
	Send(*SpeedTestSample) error
	grpc.ServerStream
}

type DiagnosticsService_SpeedTestClient interface {
	Recv() (*SpeedTestSample, error)
	grpc.ClientStream
}

type DiagnosticsServiceServer interface {
	StreamLogs(*StreamLogsRequest, DiagnosticsService_StreamLogsServer) error
	SpeedTest(*SpeedTestRequest, DiagnosticsService_SpeedTestServer) error
	HealthCheck(context.Context, *HealthCheckRequest) (*HealthCheckResponse, error)
	GetDaemonInfo(context.Context, *GetDaemonInfoRequest) (*DaemonInfo, error)
	GenerateReport(context.Context, *GenerateReportRequest) (*GenerateReportResponse, error)
}

type UnimplementedDiagnosticsServiceServer struct{}

func (UnimplementedDiagnosticsServiceServer) StreamLogs(*StreamLogsRequest, DiagnosticsService_StreamLogsServer) error                        { return nil }
func (UnimplementedDiagnosticsServiceServer) SpeedTest(*SpeedTestRequest, DiagnosticsService_SpeedTestServer) error                        { return nil }
func (UnimplementedDiagnosticsServiceServer) HealthCheck(context.Context, *HealthCheckRequest) (*HealthCheckResponse, error) { return nil, nil }
func (UnimplementedDiagnosticsServiceServer) GetDaemonInfo(context.Context, *GetDaemonInfoRequest) (*DaemonInfo, error)         { return nil, nil }
func (UnimplementedDiagnosticsServiceServer) GenerateReport(context.Context, *GenerateReportRequest) (*GenerateReportResponse, error) { return nil, nil }

func RegisterDiagnosticsServiceServer(s *grpc.Server, srv DiagnosticsServiceServer) {
	s.RegisterService(&DiagnosticsService_ServiceDesc, srv)
}

func NewDiagnosticsServiceClient(cc grpc.ClientConnInterface) DiagnosticsServiceClient { return nil }

// ============================================================================
// BackupService
// ============================================================================

type BackupServiceClient interface {
	ExportBackup(ctx context.Context, in *ExportBackupRequest, opts ...grpc.CallOption) (*ExportBackupResponse, error)
	ImportBackup(ctx context.Context, in *ImportBackupRequest, opts ...grpc.CallOption) (*ImportBackupResponse, error)
}

type BackupServiceServer interface {
	ExportBackup(context.Context, *ExportBackupRequest) (*ExportBackupResponse, error)
	ImportBackup(context.Context, *ImportBackupRequest) (*ImportBackupResponse, error)
}

type UnimplementedBackupServiceServer struct{}

func (UnimplementedBackupServiceServer) ExportBackup(context.Context, *ExportBackupRequest) (*ExportBackupResponse, error) { return nil, nil }
func (UnimplementedBackupServiceServer) ImportBackup(context.Context, *ImportBackupRequest) (*ImportBackupResponse, error) { return nil, nil }

func RegisterBackupServiceServer(s *grpc.Server, srv BackupServiceServer) {
	s.RegisterService(&BackupService_ServiceDesc, srv)
}

func NewBackupServiceClient(cc grpc.ClientConnInterface) BackupServiceClient { return nil }
