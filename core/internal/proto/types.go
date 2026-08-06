// Package proto contains hand-written gRPC types matching tunnelcraft.proto.
// TODO: regenerate from .proto with: protoc --go_out=. --go-grpc_out=. proto/tunnelcraft.proto
package proto

import (
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ============================================================================
// Enums
// ============================================================================

type Protocol int32

const (
	Protocol_PROTOCOL_UNSPECIFIED Protocol = 0
	Protocol_PROTOCOL_VLESS        Protocol = 1
	Protocol_PROTOCOL_VMESS        Protocol = 2
	Protocol_PROTOCOL_TROJAN       Protocol = 3
	Protocol_PROTOCOL_SHADOWSOCKS  Protocol = 4
	Protocol_PROTOCOL_WIREGUARD    Protocol = 5
	Protocol_PROTOCOL_HYSTERIA     Protocol = 6
	Protocol_PROTOCOL_AMNEZIAWG    Protocol = 7
	Protocol_PROTOCOL_SSH          Protocol = 8
)

type Transport int32

const (
	Transport_TRANSPORT_UNSPECIFIED Transport = 0
	Transport_TRANSPORT_TCP         Transport = 1
	Transport_TRANSPORT_WS          Transport = 2
	Transport_TRANSPORT_GRPC        Transport = 3
	Transport_TRANSPORT_KCP         Transport = 4
	Transport_TRANSPORT_XHTTP       Transport = 5
	Transport_TRANSPORT_HTTPUPGRADE Transport = 6
	Transport_TRANSPORT_QUIC        Transport = 7
)

type Security int32

const (
	Security_SECURITY_UNSPECIFIED Security = 0
	Security_SECURITY_NONE        Security = 1
	Security_SECURITY_TLS         Security = 2
	Security_SECURITY_REALITY     Security = 3
)

type ConnectionState int32

const (
	ConnectionState_CONNECTION_STATE_UNSPECIFIED  ConnectionState = 0
	ConnectionState_CONNECTION_STATE_DISCONNECTED ConnectionState = 1
	ConnectionState_CONNECTION_STATE_CONNECTING   ConnectionState = 2
	ConnectionState_CONNECTION_STATE_CONNECTED    ConnectionState = 3
	ConnectionState_CONNECTION_STATE_RECONNECTING ConnectionState = 4
	ConnectionState_CONNECTION_STATE_DISCONNECTING ConnectionState = 5
	ConnectionState_CONNECTION_STATE_ERROR        ConnectionState = 6
)

type LogLevel int32

const (
	LogLevel_LOG_LEVEL_UNSPECIFIED LogLevel = 0
	LogLevel_LOG_LEVEL_DEBUG       LogLevel = 1
	LogLevel_LOG_LEVEL_INFO        LogLevel = 2
	LogLevel_LOG_LEVEL_WARN        LogLevel = 3
	LogLevel_LOG_LEVEL_ERROR       LogLevel = 4
	LogLevel_LOG_LEVEL_FATAL       LogLevel = 5
)

type ProxyMode int32

const (
	ProxyMode_PROXY_MODE_UNSPECIFIED ProxyMode = 0
	ProxyMode_PROXY_MODE_SYSTEM      ProxyMode = 1
	ProxyMode_PROXY_MODE_SOCKS       ProxyMode = 2
	ProxyMode_PROXY_MODE_HTTP        ProxyMode = 3
	ProxyMode_PROXY_MODE_PAC         ProxyMode = 4
)

type ServerStatus int32

const (
	ServerStatus_SERVER_STATUS_UNSPECIFIED ServerStatus = 0
	ServerStatus_SERVER_STATUS_ONLINE      ServerStatus = 1
	ServerStatus_SERVER_STATUS_OFFLINE     ServerStatus = 2
	ServerStatus_SERVER_STATUS_UNKNOWN     ServerStatus = 3
)

type RuleAction int32

const (
	RuleAction_RULE_ACTION_UNSPECIFIED RuleAction = 0
	RuleAction_RULE_ACTION_PROXY       RuleAction = 1
	RuleAction_RULE_ACTION_DIRECT      RuleAction = 2
	RuleAction_RULE_ACTION_BLOCK       RuleAction = 3
)

// ============================================================================
// Common Messages
// ============================================================================

type Timestamp = timestamppb.Timestamp
type Duration = durationpb.Duration

type MetadataEntry struct {
	Key   string `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value string `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
}

func (*MetadataEntry) ProtoMessage()               {}
func (x *MetadataEntry) ProtoReflect() interface{} { return nil }

type GeoLocation struct {
	Country     string  `protobuf:"bytes,1,opt,name=country,proto3" json:"country,omitempty"`
	CountryCode string  `protobuf:"bytes,2,opt,name=country_code,json=country_code,omitempty"`
	City        string  `protobuf:"bytes,3,opt,name=city,proto3" json:"city,omitempty"`
	Latitude    float64 `protobuf:"fixed64,4,opt,name=latitude,proto3" json:"latitude,omitempty"`
	Longitude   float64 `protobuf:"fixed64,5,opt,name=longitude,proto3" json:"longitude,omitempty"`
	ISP         string  `protobuf:"bytes,6,opt,name=isp,proto3" json:"isp,omitempty"`
}

func (*GeoLocation) ProtoMessage()               {}
func (x *GeoLocation) ProtoReflect() interface{} { return nil }

type BandwidthStats struct {
	BytesUploaded   uint64           `protobuf:"varint,1,opt,name=bytes_uploaded,json=bytesUploaded,proto3" json:"bytes_uploaded,omitempty"`
	BytesDownloaded uint64           `protobuf:"varint,2,opt,name=bytes_downloaded,json=bytesDownloaded,proto3" json:"bytes_downloaded,omitempty"`
	SessionStart    *timestamppb.Timestamp `protobuf:"bytes,3,opt,name=session_start,json=sessionStart,proto3" json:"session_start,omitempty"`
	Duration        *durationpb.Duration  `protobuf:"bytes,4,opt,name=duration,proto3" json:"duration,omitempty"`
}

func (*BandwidthStats) ProtoMessage()               {}
func (x *BandwidthStats) ProtoReflect() interface{} { return nil }

type LatencySample struct {
	Sequence  uint32          `protobuf:"varint,1,opt,name=sequence,proto3" json:"sequence,omitempty"`
	Rtt       *durationpb.Duration `protobuf:"bytes,2,opt,name=rtt,proto3" json:"rtt,omitempty"`
	TimedOut bool            `protobuf:"varint,3,opt,name=timed_out,json=timedOut,proto3" json:"timed_out,omitempty"`
}

func (*LatencySample) ProtoMessage()               {}
func (x *LatencySample) ProtoReflect() interface{} { return nil }

// ============================================================================
// Protocol Config Messages
// ============================================================================

type XrayConfig struct {
	Uuid          string    `protobuf:"bytes,1,opt,name=uuid,proto3" json:"uuid,omitempty"`
	Flow          string    `protobuf:"bytes,2,opt,name=flow,proto3" json:"flow,omitempty"`
	Security      Security  `protobuf:"varint,3,opt,name=security,proto3,enum=tunnelcraft.v1.Security" json:"security,omitempty"`
	Transport     Transport `protobuf:"varint,4,opt,name=transport,proto3,enum=tunnelcraft.v1.Transport" json:"transport,omitempty"`
	Sni           string    `protobuf:"bytes,5,opt,name=sni,proto3" json:"sni,omitempty"`
	Fingerprint   string    `protobuf:"bytes,6,opt,name=fingerprint,proto3" json:"fingerprint,omitempty"`
	Alpn          string    `protobuf:"bytes,7,opt,name=alpn,proto3" json:"alpn,omitempty"`
	PublicKey     string    `protobuf:"bytes,8,opt,name=public_key,json=publicKey,proto3" json:"public_key,omitempty"`
	ShortId       string    `protobuf:"bytes,9,opt,name=short_id,json=shortId,proto3" json:"short_id,omitempty"`
	KcpSeed       string    `protobuf:"bytes,10,opt,name=kcp_seed,json=kcpSeed,proto3" json:"kcp_seed,omitempty"`
	XhttpPath     string    `protobuf:"bytes,11,opt,name=xhttp_path,json=xhttpPath,proto3" json:"xhttp_path,omitempty"`
	XhttpMode     string    `protobuf:"bytes,12,opt,name=xhttp_mode,json=xhttpMode,proto3" json:"xhttp_mode,omitempty"`
	WsPath        string    `protobuf:"bytes,13,opt,name=ws_path,json=wsPath,proto3" json:"ws_path,omitempty"`
	GrpcService   string    `protobuf:"bytes,14,opt,name=grpc_service,json=grpcService,proto3" json:"grpc_service,omitempty"`
	AllowInsecure bool      `protobuf:"varint,15,opt,name=allow_insecure,json=allowInsecure,proto3" json:"allow_insecure,omitempty"`
}

func (*XrayConfig) ProtoMessage()               {}
func (x *XrayConfig) ProtoReflect() interface{} { return nil }

type WireGuardConfig struct {
	PrivateKey         string `protobuf:"bytes,1,opt,name=private_key,json=privateKey,proto3" json:"private_key,omitempty"`
	PublicKey          string `protobuf:"bytes,2,opt,name=public_key,json=publicKey,proto3" json:"public_key,omitempty"`
	PresharedKey       string `protobuf:"bytes,3,opt,name=preshared_key,json=presharedKey,proto3" json:"preshared_key,omitempty"`
	LocalAddress       string `protobuf:"bytes,4,opt,name=local_address,json=localAddress,proto3" json:"local_address,omitempty"`
	DnsServers         string `protobuf:"bytes,5,opt,name=dns_servers,json=dnsServers,proto3" json:"dns_servers,omitempty"`
	Mtu                uint32 `protobuf:"varint,6,opt,name=mtu,proto3" json:"mtu,omitempty"`
	PersistentKeepalive uint32 `protobuf:"varint,7,opt,name=persistent_keepalive,json=persistentKeepalive,proto3" json:"persistent_keepalive,omitempty"`
	AllowedIps         string `protobuf:"bytes,8,opt,name=allowed_ips,json=allowedIps,proto3" json:"allowed_ips,omitempty"`
}

func (*WireGuardConfig) ProtoMessage()               {}
func (x *WireGuardConfig) ProtoReflect() interface{} { return nil }

type HysteriaConfig struct {
	AuthPassword string `protobuf:"bytes,1,opt,name=auth_password,json=authPassword,proto3" json:"auth_password,omitempty"`
	Sni          string `protobuf:"bytes,2,opt,name=sni,proto3" json:"sni,omitempty"`
	Insecure     bool   `protobuf:"varint,3,opt,name=insecure,proto3" json:"insecure,omitempty"`
	Alpn         string `protobuf:"bytes,4,opt,name=alpn,proto3" json:"alpn,omitempty"`
	ObfsPassword string `protobuf:"bytes,5,opt,name=obfs_password,json=obfsPassword,proto3" json:"obfs_password,omitempty"`
	Protocol     string `protobuf:"bytes,6,opt,name=protocol,proto3" json:"protocol,omitempty"`
	BandwidthUp  uint64 `protobuf:"varint,7,opt,name=bandwidth_up,json=bandwidthUp,proto3" json:"bandwidth_up,omitempty"`
	BandwidthDown uint64 `protobuf:"varint,8,opt,name=bandwidth_down,json=bandwidthDown,proto3" json:"bandwidth_down,omitempty"`
	FastOpen     bool   `protobuf:"varint,9,opt,name=fast_open,json=fastOpen,proto3" json:"fast_open,omitempty"`
}

func (*HysteriaConfig) ProtoMessage()               {}
func (x *HysteriaConfig) ProtoReflect() interface{} { return nil }

type ShadowsocksConfig struct {
	Method     string `protobuf:"bytes,1,opt,name=method,proto3" json:"method,omitempty"`
	Password   string `protobuf:"bytes,2,opt,name=password,proto3" json:"password,omitempty"`
	Plugin     string `protobuf:"bytes,3,opt,name=plugin,proto3" json:"plugin,omitempty"`
	PluginOpts string `protobuf:"bytes,4,opt,name=plugin_opts,json=pluginOpts,proto3" json:"plugin_opts,omitempty"`
}

func (*ShadowsocksConfig) ProtoMessage()               {}
func (x *ShadowsocksConfig) ProtoReflect() interface{} { return nil }

type AmneziaWGConfig struct {
	PrivateKey  string `protobuf:"bytes,1,opt,name=private_key,json=privateKey,proto3" json:"private_key,omitempty"`
	PublicKey   string `protobuf:"bytes,2,opt,name=public_key,json=publicKey,proto3" json:"public_key,omitempty"`
	PresharedKey string `protobuf:"bytes,3,opt,name=preshared_key,json=presharedKey,proto3" json:"preshared_key,omitempty"`
	LocalAddress string `protobuf:"bytes,4,opt,name=local_address,json=localAddress,proto3" json:"local_address,omitempty"`
	DnsServers   string `protobuf:"bytes,5,opt,name=dns_servers,json=dnsServers,proto3" json:"dns_servers,omitempty"`
	Mtu          uint32 `protobuf:"varint,6,opt,name=mtu,proto3" json:"mtu,omitempty"`
}

func (*AmneziaWGConfig) ProtoMessage()               {}
func (x *AmneziaWGConfig) ProtoReflect() interface{} { return nil }

type SSHConfig struct {
	Username      string `protobuf:"bytes,1,opt,name=username,proto3" json:"username,omitempty"`
	Password      string `protobuf:"bytes,2,opt,name=password,proto3" json:"password,omitempty"`
	PrivateKey    string `protobuf:"bytes,3,opt,name=private_key,json=privateKey,proto3" json:"private_key,omitempty"`
	KeyPassphrase string `protobuf:"bytes,4,opt,name=key_passphrase,json=keyPassphrase,proto3" json:"key_passphrase,omitempty"`
}

func (*SSHConfig) ProtoMessage()               {}
func (x *SSHConfig) ProtoReflect() interface{} { return nil }

// ============================================================================
// Server
// ============================================================================

type Server struct {
	Id             string           `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Name           string           `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Host           string           `protobuf:"bytes,3,opt,name=host,proto3" json:"host,omitempty"`
	Port           uint32           `protobuf:"varint,4,opt,name=port,proto3" json:"port,omitempty"`
	Protocol       Protocol         `protobuf:"varint,5,opt,name=protocol,proto3,enum=tunnelcraft.v1.Protocol" json:"protocol,omitempty"`
	Xray           *XrayConfig      `protobuf:"bytes,10,opt,name=xray,proto3" json:"xray,omitempty"`
	Wireguard      *WireGuardConfig `protobuf:"bytes,11,opt,name=wireguard,proto3" json:"wireguard,omitempty"`
	Hysteria       *HysteriaConfig  `protobuf:"bytes,12,opt,name=hysteria,proto3" json:"hysteria,omitempty"`
	Shadowsocks    *ShadowsocksConfig `protobuf:"bytes,13,opt,name=shadowsocks,proto3" json:"shadowsocks,omitempty"`
	Amneziawg      *AmneziaWGConfig `protobuf:"bytes,14,opt,name=amneziawg,proto3" json:"amneziawg,omitempty"`
	Ssh            *SSHConfig       `protobuf:"bytes,15,opt,name=ssh,proto3" json:"ssh,omitempty"`
	Geo            *GeoLocation     `protobuf:"bytes,20,opt,name=geo,proto3" json:"geo,omitempty"`
	Status         ServerStatus     `protobuf:"varint,21,opt,name=status,proto3,enum=tunnelcraft.v1.ServerStatus" json:"status,omitempty"`
	Favorite       bool             `protobuf:"varint,22,opt,name=favorite,proto3" json:"favorite,omitempty"`
	SortOrder      uint32           `protobuf:"varint,23,opt,name=sort_order,json=sortOrder,proto3" json:"sort_order,omitempty"`
	Tags           []string         `protobuf:"bytes,24,rep,name=tags,proto3" json:"tags,omitempty"`
	LastConnected  *timestamppb.Timestamp `protobuf:"bytes,25,opt,name=last_connected,json=lastConnected,proto3" json:"last_connected,omitempty"`
	CreatedAt      *timestamppb.Timestamp `protobuf:"bytes,26,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"`
	UpdatedAt      *timestamppb.Timestamp `protobuf:"bytes,27,opt,name=updated_at,json=updatedAt,proto3" json:"updated_at,omitempty"`
	SubscriptionId string           `protobuf:"bytes,28,opt,name=subscription_id,json=subscriptionId,proto3" json:"subscription_id,omitempty"`
}

func (*Server) ProtoMessage()               {}
func (x *Server) ProtoReflect() interface{} { return nil }

// ============================================================================
// Subscription
// ============================================================================

type Subscription struct {
	Id             string                `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Name           string                `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Url            string                `protobuf:"bytes,3,opt,name=url,proto3" json:"url,omitempty"`
	Username       string                `protobuf:"bytes,4,opt,name=username,proto3" json:"username,omitempty"`
	Password       string                `protobuf:"bytes,5,opt,name=password,proto3" json:"password,omitempty"`
	RefreshInterval uint32               `protobuf:"varint,6,opt,name=refresh_interval,json=refreshInterval,proto3" json:"refresh_interval,omitempty"`
	LastRefreshed  *timestamppb.Timestamp `protobuf:"bytes,7,opt,name=last_refreshed,json=lastRefreshed,proto3" json:"last_refreshed,omitempty"`
	ServerCount    int64                 `protobuf:"varint,8,opt,name=server_count,json=serverCount,proto3" json:"server_count,omitempty"`
	Filter         string                `protobuf:"bytes,9,opt,name=filter,proto3" json:"filter,omitempty"`
	Enabled        bool                  `protobuf:"varint,10,opt,name=enabled,proto3" json:"enabled,omitempty"`
	CreatedAt      *timestamppb.Timestamp `protobuf:"bytes,11,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"`
	UpdatedAt      *timestamppb.Timestamp `protobuf:"bytes,12,opt,name=updated_at,json=updatedAt,proto3" json:"updated_at,omitempty"`
}

func (*Subscription) ProtoMessage()               {}
func (x *Subscription) ProtoReflect() interface{} { return nil }

// ============================================================================
// Routing
// ============================================================================

type RoutingRule struct {
	Id         string      `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Name       string      `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Enabled    bool        `protobuf:"varint,3,opt,name=enabled,proto3" json:"enabled,omitempty"`
	Action     RuleAction  `protobuf:"varint,4,opt,name=action,proto3,enum=tunnelcraft.v1.RuleAction" json:"action,omitempty"`
	Domains    []string    `protobuf:"bytes,10,rep,name=domains,proto3" json:"domains,omitempty"`
	IpCidrs    []string    `protobuf:"bytes,11,rep,name=ip_cidrs,json=ipCidrs,proto3" json:"ip_cidrs,omitempty"`
	GeoipCodes []string    `protobuf:"bytes,12,rep,name=geoip_codes,json=geoipCodes,proto3" json:"geoip_codes,omitempty"`
	Processes  []string    `protobuf:"bytes,13,rep,name=processes,proto3" json:"processes,omitempty"`
	Ports      []string    `protobuf:"bytes,14,rep,name=ports,proto3" json:"ports,omitempty"`
	Protocols  []string    `protobuf:"bytes,15,rep,name=protocols,proto3" json:"protocols,omitempty"`
}

func (*RoutingRule) ProtoMessage()               {}
func (x *RoutingRule) ProtoReflect() interface{} { return nil }

type RoutingConfig struct {
	DomainStrategy string        `protobuf:"bytes,1,opt,name=domain_strategy,json=domainStrategy,proto3" json:"domain_strategy,omitempty"`
	Rules          []*RoutingRule `protobuf:"bytes,2,rep,name=rules,proto3" json:"rules,omitempty"`
}

func (*RoutingConfig) ProtoMessage()               {}
func (x *RoutingConfig) ProtoReflect() interface{} { return nil }

// ============================================================================
// Settings
// ============================================================================

type Settings struct {
	ProxyMode         ProxyMode      `protobuf:"varint,1,opt,name=proxy_mode,json=proxyMode,proto3,enum=tunnelcraft.v1.ProxyMode" json:"proxy_mode,omitempty"`
	SocksPort         uint32          `protobuf:"varint,2,opt,name=socks_port,json=socksPort,proto3" json:"socks_port,omitempty"`
	HttpPort          uint32          `protobuf:"varint,3,opt,name=http_port,json=httpPort,proto3" json:"http_port,omitempty"`
	DnsServers        string          `protobuf:"bytes,4,opt,name=dns_servers,json=dnsServers,proto3" json:"dns_servers,omitempty"`
	LogLevel          LogLevel        `protobuf:"varint,5,opt,name=log_level,json=logLevel,proto3,enum=tunnelcraft.v1.LogLevel" json:"log_level,omitempty"`
	AutoConnect       bool            `protobuf:"varint,6,opt,name=auto_connect,json=autoConnect,proto3" json:"auto_connect,omitempty"`
	ConnectOnStartup  bool            `protobuf:"varint,7,opt,name=connect_on_startup,json=connectOnStartup,proto3" json:"connect_on_startup,omitempty"`
	KillSwitch        bool            `protobuf:"varint,8,opt,name=kill_switch,json=killSwitch,proto3" json:"kill_switch,omitempty"`
	SplitTunneling    bool            `protobuf:"varint,9,opt,name=split_tunneling,json=splitTunneling,proto3" json:"split_tunneling,omitempty"`
	LanInterface      string          `protobuf:"bytes,10,opt,name=lan_interface,json=lanInterface,proto3" json:"lan_interface,omitempty"`
	BypassDomain      string          `protobuf:"bytes,11,opt,name=bypass_domain,json=bypassDomain,proto3" json:"bypass_domain,omitempty"`
	AllowLan          bool            `protobuf:"varint,12,opt,name=allow_lan,json=allowLan,proto3" json:"allow_lan,omitempty"`
	ConnectionTimeout uint32          `protobuf:"varint,13,opt,name=connection_timeout,json=connectionTimeout,proto3" json:"connection_timeout,omitempty"`
	ReconnectAttempts uint32          `protobuf:"varint,14,opt,name=reconnect_attempts,json=reconnectAttempts,proto3" json:"reconnect_attempts,omitempty"`
	ReconnectDelay    uint32          `protobuf:"varint,15,opt,name=reconnect_delay,json=reconnectDelay,proto3" json:"reconnect_delay,omitempty"`
	TelemetryEnabled  bool            `protobuf:"varint,16,opt,name=telemetry_enabled,json=telemetryEnabled,proto3" json:"telemetry_enabled,omitempty"`
	Language           string          `protobuf:"bytes,17,opt,name=language,proto3" json:"language,omitempty"`
	Theme              string          `protobuf:"bytes,18,opt,name=theme,proto3" json:"theme,omitempty"`
	ActiveServerId     string          `protobuf:"bytes,19,opt,name=active_server_id,json=activeServerId,proto3" json:"active_server_id,omitempty"`
	Routing            *RoutingConfig  `protobuf:"bytes,20,opt,name=routing,proto3" json:"routing,omitempty"`
}

func (*Settings) ProtoMessage()               {}
func (x *Settings) ProtoReflect() interface{} { return nil }

// ============================================================================
// TunnelService messages
// ============================================================================

type ConnectRequest struct {
	ServerId    string            `protobuf:"bytes,1,opt,name=server_id,json=serverId,proto3" json:"server_id,omitempty"`
	OverrideMode ProxyMode         `protobuf:"varint,2,opt,name=override_mode,json=overrideMode,proto3,enum=tunnelcraft.v1.ProxyMode" json:"override_mode,omitempty"`
	Extra        map[string]string `protobuf:"bytes,3,rep,name=extra,proto3" json:"extra,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (*ConnectRequest) ProtoMessage()               {}
func (x *ConnectRequest) ProtoReflect() interface{} { return nil }

type ConnectResponse struct {
	State       ConnectionState   `protobuf:"varint,1,opt,name=state,proto3,enum=tunnelcraft.v1.ConnectionState" json:"state,omitempty"`
	ServerId    string                `protobuf:"bytes,2,opt,name=server_id,json=serverId,proto3" json:"server_id,omitempty"`
	SocksPort   uint32                `protobuf:"varint,3,opt,name=socks_port,json=socksPort,proto3" json:"socks_port,omitempty"`
	HttpPort    uint32                `protobuf:"varint,4,opt,name=http_port,json=httpPort,proto3" json:"http_port,omitempty"`
	Error       string                `protobuf:"bytes,5,opt,name=error,proto3" json:"error,omitempty"`
	ConnectedAt *timestamppb.Timestamp `protobuf:"bytes,6,opt,name=connected_at,json=connectedAt,proto3" json:"connected_at,omitempty"`
}

func (*ConnectResponse) ProtoMessage()               {}
func (x *ConnectResponse) ProtoReflect() interface{} { return nil }

type DisconnectRequest struct {
	Force bool `protobuf:"varint,1,opt,name=force,proto3" json:"force,omitempty"`
}
func (*DisconnectRequest) ProtoMessage()               {}
func (x *DisconnectRequest) ProtoReflect() interface{} { return nil }

type DisconnectResponse struct {
	State      ConnectionState `protobuf:"varint,1,opt,name=state,proto3,enum=tunnelcraft.v1.ConnectionState" json:"state,omitempty"`
	FinalStats *BandwidthStats `protobuf:"bytes,2,opt,name=final_stats,json=finalStats,proto3" json:"final_stats,omitempty"`
	Error      string           `protobuf:"bytes,3,opt,name=error,proto3" json:"error,omitempty"`
}
func (*DisconnectResponse) ProtoMessage()               {}
func (x *DisconnectResponse) ProtoReflect() interface{} { return nil }

type WatchConnectionRequest struct{}
func (*WatchConnectionRequest) ProtoMessage()               {}
func (x *WatchConnectionRequest) ProtoReflect() interface{} { return nil }

type ConnectionStateEvent struct {
	Timestamp *timestamppb.Timestamp `protobuf:"bytes,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	State     ConnectionState         `protobuf:"varint,1,opt,name=state,proto3,enum=tunnelcraft.v1.ConnectionState" json:"state,omitempty"`
	Message   string                  `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
	Error     string                  `protobuf:"bytes,4,opt,name=error,proto3" json:"error,omitempty"`
	ServerId  string                  `protobuf:"bytes,5,opt,name=server_id,json=serverId,proto3" json:"server_id,omitempty"`
}
func (*ConnectionStateEvent) ProtoMessage()               {}
func (x *ConnectionStateEvent) ProtoReflect() interface{} { return nil }

type GetConnectionStatusRequest struct{}
func (*GetConnectionStatusRequest) ProtoMessage()               {}
func (x *GetConnectionStatusRequest) ProtoReflect() interface{} { return nil }

type GetConnectionStatusResponse struct {
	State       ConnectionState   `protobuf:"varint,1,opt,name=state,proto3,enum=tunnelcraft.v1.ConnectionState" json:"state,omitempty"`
	ServerId    string                `protobuf:"bytes,2,opt,name=server_id,json=serverId,proto3" json:"server_id,omitempty"`
	Stats       *BandwidthStats       `protobuf:"bytes,3,opt,name=stats,proto3" json:"stats,omitempty"`
	ConnectedAt *timestamppb.Timestamp `protobuf:"bytes,4,opt,name=connected_at,json=connectedAt,proto3" json:"connected_at,omitempty"`
	Mode        ProxyMode             `protobuf:"varint,5,opt,name=mode,proto3,enum=tunnelcraft.v1.ProxyMode" json:"mode,omitempty"`
	SocksPort   uint32                `protobuf:"varint,6,opt,name=socks_port,json=socksPort,proto3" json:"socks_port,omitempty"`
	HttpPort    uint32                `protobuf:"varint,7,opt,name=http_port,json=httpPort,proto3" json:"http_port,omitempty"`
	LocalIp     string                `protobuf:"bytes,8,opt,name=local_ip,json=localIp,proto3" json:"local_ip,omitempty"`
}
func (*GetConnectionStatusResponse) ProtoMessage()               {}
func (x *GetConnectionStatusResponse) ProtoReflect() interface{} { return nil }

type ReconnectRequest struct {
	ServerId string `protobuf:"bytes,1,opt,name=server_id,json=serverId,proto3" json:"server_id,omitempty"`
}
func (*ReconnectRequest) ProtoMessage()               {}
func (x *ReconnectRequest) ProtoReflect() interface{} { return nil }

type ReconnectResponse struct {
	State    ConnectionState `protobuf:"varint,1,opt,name=state,proto3,enum=tunnelcraft.v1.ConnectionState" json:"state,omitempty"`
	ServerId string           `protobuf:"bytes,2,opt,name=server_id,json=serverId,proto3" json:"server_id,omitempty"`
	Error    string           `protobuf:"bytes,3,opt,name=error,proto3" json:"error,omitempty"`
}
func (*ReconnectResponse) ProtoMessage()               {}
func (x *ReconnectResponse) ProtoReflect() interface{} { return nil }

// ============================================================================
// ServerService messages
// ============================================================================

type ListServersRequest struct {
	Protocol       Protocol `protobuf:"varint,1,opt,name=protocol,proto3,enum=tunnelcraft.v1.Protocol" json:"protocol,omitempty"`
	Tag            string   `protobuf:"bytes,2,opt,name=tag,proto3" json:"tag,omitempty"`
	SubscriptionId string   `protobuf:"bytes,3,opt,name=subscription_id,json=subscriptionId,proto3" json:"subscription_id,omitempty"`
	FavoritesOnly  bool     `protobuf:"varint,4,opt,name=favorites_only,json=favoritesOnly,proto3" json:"favorites_only,omitempty"`
	SearchQuery    string   `protobuf:"bytes,5,opt,name=search_query,json=searchQuery,proto3" json:"search_query,omitempty"`
}
func (*ListServersRequest) ProtoMessage()               {}
func (x *ListServersRequest) ProtoReflect() interface{} { return nil }

type ListServersResponse struct {
	Servers []*Server `protobuf:"bytes,1,rep,name=servers,proto3" json:"servers,omitempty"`
	Total   int32      `protobuf:"varint,2,opt,name=total,proto3" json:"total,omitempty"`
}
func (*ListServersResponse) ProtoMessage()               {}
func (x *ListServersResponse) ProtoReflect() interface{} { return nil }

type GetServerRequest struct{Id string}
func (*GetServerRequest) ProtoMessage()               {}
func (x *GetServerRequest) ProtoReflect() interface{} { return nil }

type CreateServerRequest struct{Server *Server}
func (*CreateServerRequest) ProtoMessage()               {}
func (x *CreateServerRequest) ProtoReflect() interface{} { return nil }

type UpdateServerRequest struct {
	Server     *Server                `protobuf:"bytes,1,opt,name=server,proto3" json:"server,omitempty"`
	UpdateMask *fieldmaskpb.FieldMask `protobuf:"bytes,2,opt,name=update_mask,json=updateMask,proto3" json:"update_mask,omitempty"`
}
func (*UpdateServerRequest) ProtoMessage()               {}
func (x *UpdateServerRequest) ProtoReflect() interface{} { return nil }

type DeleteServerRequest struct{Id string}
func (*DeleteServerRequest) ProtoMessage()               {}
func (x *DeleteServerRequest) ProtoReflect() interface{} { return nil }

type TestServersRequest struct {
	ServerIds     []string `protobuf:"bytes,1,rep,name=server_ids,json=serverIds,proto3" json:"server_ids,omitempty"`
	TimeoutSeconds uint32  `protobuf:"varint,2,opt,name=timeout_seconds,json=timeoutSeconds,proto3" json:"timeout_seconds,omitempty"`
}
func (*TestServersRequest) ProtoMessage()               {}
func (x *TestServersRequest) ProtoReflect() interface{} { return nil }

type ServerTestResult struct {
	ServerId string            `protobuf:"bytes,1,opt,name=server_id,json=serverId,proto3" json:"server_id,omitempty"`
	Reachable bool              `protobuf:"varint,2,opt,name=reachable,proto3" json:"reachable,omitempty"`
	Latency   *durationpb.Duration `protobuf:"bytes,3,opt,name=latency,proto3" json:"latency,omitempty"`
	Error     string            `protobuf:"bytes,4,opt,name=error,proto3" json:"error,omitempty"`
	Status    ServerStatus      `protobuf:"varint,5,opt,name=status,proto3,enum=tunnelcraft.v1.ServerStatus" json:"status,omitempty"`
}
func (*ServerTestResult) ProtoMessage()               {}
func (x *ServerTestResult) ProtoReflect() interface{} { return nil }

type ImportServersRequest struct {
	Content        string `protobuf:"bytes,1,opt,name=content,proto3" json:"content,omitempty"`
	SubscriptionId string `protobuf:"bytes,2,opt,name=subscription_id,json=subscriptionId,proto3" json:"subscription_id,omitempty"`
	GroupName      string `protobuf:"bytes,3,opt,name=group_name,json=groupName,proto3" json:"group_name,omitempty"`
}
func (*ImportServersRequest) ProtoMessage()               {}
func (x *ImportServersRequest) ProtoReflect() interface{} { return nil }

type ImportServersResponse struct {
	ImportedServerIds []string `protobuf:"bytes,1,rep,name=imported_server_ids,json=importedServerIds,proto3" json:"imported_server_ids,omitempty"`
	TotalParsed       int32    `protobuf:"varint,2,opt,name=total_parsed,json=totalParsed,proto3" json:"total_parsed,omitempty"`
	TotalImported     int32    `protobuf:"varint,3,opt,name=total_imported,json=totalImported,proto3" json:"total_imported,omitempty"`
	Errors            []string `protobuf:"bytes,4,rep,name=errors,proto3" json:"errors,omitempty"`
}
func (*ImportServersResponse) ProtoMessage()               {}
func (x *ImportServersResponse) ProtoReflect() interface{} { return nil }

type ExportServerRequest struct{ServerId string}
func (*ExportServerRequest) ProtoMessage()               {}
func (x *ExportServerRequest) ProtoReflect() interface{} { return nil }

type ExportServerResponse struct{ShareLink string}
func (*ExportServerResponse) ProtoMessage()               {}
func (x *ExportServerResponse) ProtoReflect() interface{} { return nil }

type PingServerRequest struct {
	ServerId string            `protobuf:"bytes,1,opt,name=server_id,json=serverId,proto3" json:"server_id,omitempty"`
	Count    uint32            `protobuf:"varint,2,opt,name=count,proto3" json:"count,omitempty"`
	Interval  *durationpb.Duration `protobuf:"bytes,3,opt,name=interval,proto3" json:"interval,omitempty"`
}
func (*PingServerRequest) ProtoMessage()               {}
func (x *PingServerRequest) ProtoReflect() interface{} { return nil }

// ============================================================================
// SubscriptionService messages
// ============================================================================

type ListSubscriptionsRequest struct{}
func (*ListSubscriptionsRequest) ProtoMessage()               {}
func (x *ListSubscriptionsRequest) ProtoReflect() interface{} { return nil }

type ListSubscriptionsResponse struct {
	Subscriptions []*Subscription `protobuf:"bytes,1,rep,name=subscriptions,proto3" json:"subscriptions,omitempty"`
}
func (*ListSubscriptionsResponse) ProtoMessage()               {}
func (x *ListSubscriptionsResponse) ProtoReflect() interface{} { return nil }

type GetSubscriptionRequest struct{Id string}
func (*GetSubscriptionRequest) ProtoMessage()               {}
func (x *GetSubscriptionRequest) ProtoReflect() interface{} { return nil }

type CreateSubscriptionRequest struct{Subscription *Subscription}
func (*CreateSubscriptionRequest) ProtoMessage()               {}
func (x *CreateSubscriptionRequest) ProtoReflect() interface{} { return nil }

type UpdateSubscriptionRequest struct {
	Subscription *Subscription           `protobuf:"bytes,1,opt,name=subscription,proto3" json:"subscription,omitempty"`
	UpdateMask  *fieldmaskpb.FieldMask `protobuf:"bytes,2,opt,name=update_mask,json=updateMask,proto3" json:"update_mask,omitempty"`
}
func (*UpdateSubscriptionRequest) ProtoMessage()               {}
func (x *UpdateSubscriptionRequest) ProtoReflect() interface{} { return nil }

type DeleteSubscriptionRequest struct{Id string}
func (*DeleteSubscriptionRequest) ProtoMessage()               {}
func (x *DeleteSubscriptionRequest) ProtoReflect() interface{} { return nil }

type RefreshSubscriptionRequest struct{Id string}
func (*RefreshSubscriptionRequest) ProtoMessage()               {}
func (x *RefreshSubscriptionRequest) ProtoReflect() interface{} { return nil }

type RefreshSubscriptionResponse struct {
	Subscription  *Subscription `protobuf:"bytes,1,opt,name=subscription,proto3" json:"subscription,omitempty"`
	ImportedIds []string     `protobuf:"bytes,2,rep,name=imported_ids,json=importedIds,proto3" json:"imported_ids,omitempty"`
	Added        int32        `protobuf:"varint,3,opt,name=added,proto3" json:"added,omitempty"`
	Updated      int32        `protobuf:"varint,4,opt,name=updated,proto3" json:"updated,omitempty"`
	Removed      int32        `protobuf:"varint,5,opt,name=removed,proto3" json:"removed,omitempty"`
}
func (*RefreshSubscriptionResponse) ProtoMessage()               {}
func (x *RefreshSubscriptionResponse) ProtoReflect() interface{} { return nil }

type StreamRefreshRequest struct{Id string}
func (*StreamRefreshRequest) ProtoMessage()               {}
func (x *StreamRefreshRequest) ProtoReflect() interface{} { return nil }

type RefreshProgress struct {
	Percent  float32 `protobuf:"fixed32,1,opt,name=percent,proto3" json:"percent,omitempty"`
	Stage    string  `protobuf:"bytes,2,opt,name=stage,proto3" json:"stage,omitempty"`
	Parsed   int32   `protobuf:"varint,3,opt,name=parsed,proto3" json:"parsed,omitempty"`
	Imported int32   `protobuf:"varint,4,opt,name=imported,proto3" json:"imported,omitempty"`
	Error    string  `protobuf:"bytes,5,opt,name=error,proto3" json:"error,omitempty"`
}
func (*RefreshProgress) ProtoMessage()               {}
func (x *RefreshProgress) ProtoReflect() interface{} { return nil }

// ============================================================================
// SettingsService messages
// ============================================================================

type GetSettingsRequest struct{}
func (*GetSettingsRequest) ProtoMessage()               {}
func (x *GetSettingsRequest) ProtoReflect() interface{} { return nil }

type UpdateSettingsRequest struct {
	Settings    *Settings              `protobuf:"bytes,1,opt,name=settings,proto3" json:"settings,omitempty"`
	UpdateMask  *fieldmaskpb.FieldMask `protobuf:"bytes,2,opt,name=update_mask,json=updateMask,proto3" json:"update_mask,omitempty"`
}
func (*UpdateSettingsRequest) ProtoMessage()               {}
func (x *UpdateSettingsRequest) ProtoReflect() interface{} { return nil }

type ResetSettingsRequest struct{}
func (*ResetSettingsRequest) ProtoMessage()               {}
func (x *ResetSettingsRequest) ProtoReflect() interface{} { return nil }

// ============================================================================
// RoutingService messages
// ============================================================================

type GetRoutingRequest struct{}
func (*GetRoutingRequest) ProtoMessage()               {}
func (x *GetRoutingRequest) ProtoReflect() interface{} { return nil }

type UpdateRoutingRequest struct{Routing *RoutingConfig}
func (*UpdateRoutingRequest) ProtoMessage()               {}
func (x *UpdateRoutingRequest) ProtoReflect() interface{} { return nil }

type ListRulesRequest struct{Action RuleAction}
func (*ListRulesRequest) ProtoMessage()               {}
func (x *ListRulesRequest) ProtoReflect() interface{} { return nil }

type ListRulesResponse struct{Rules []*RoutingRule}
func (*ListRulesResponse) ProtoMessage()               {}
func (x *ListRulesResponse) ProtoReflect() interface{} { return nil }

type CreateRuleRequest struct{Rule *RoutingRule}
func (*CreateRuleRequest) ProtoMessage()               {}
func (x *CreateRuleRequest) ProtoReflect() interface{} { return nil }

type UpdateRuleRequest struct {
	Rule        *RoutingRule           `protobuf:"bytes,1,opt,name=rule,proto3" json:"rule,omitempty"`
	UpdateMask  *fieldmaskpb.FieldMask `protobuf:"bytes,2,opt,name=update_mask,json=updateMask,proto3" json:"update_mask,omitempty"`
}
func (*UpdateRuleRequest) ProtoMessage()               {}
func (x *UpdateRuleRequest) ProtoReflect() interface{} { return nil }

type DeleteRuleRequest struct{Id string}
func (*DeleteRuleRequest) ProtoMessage()               {}
func (x *DeleteRuleRequest) ProtoReflect() interface{} { return nil }

type ReorderRulesRequest struct{RuleIds []string}
func (*ReorderRulesRequest) ProtoMessage()               {}
func (x *ReorderRulesRequest) ProtoReflect() interface{} { return nil }

type ReorderRulesResponse struct{Rules []*RoutingRule}
func (*ReorderRulesResponse) ProtoMessage()               {}
func (x *ReorderRulesResponse) ProtoReflect() interface{} { return nil }

// ============================================================================
// DiagnosticsService messages
// ============================================================================

type StreamLogsRequest struct {
	MinLevel LogLevel `protobuf:"varint,1,opt,name=min_level,json=minLevel,proto3,enum=tunnelcraft.v1.LogLevel" json:"min_level,omitempty"`
	Filter    string   `protobuf:"bytes,2,opt,name=filter,proto3" json:"filter,omitempty"`
	Tail      bool     `protobuf:"varint,3,opt,name=tail,proto3" json:"tail,omitempty"`
}
func (*StreamLogsRequest) ProtoMessage()               {}
func (x *StreamLogsRequest) ProtoReflect() interface{} { return nil }

type LogEntry struct {
	Timestamp *timestamppb.Timestamp `protobuf:"bytes,1,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Level     LogLevel              `protobuf:"varint,2,opt,name=level,proto3,enum=tunnelcraft.v1.LogLevel" json:"level,omitempty"`
	Component string                `protobuf:"bytes,3,opt,name=component,proto3" json:"component,omitempty"`
	Message   string                `protobuf:"bytes,4,opt,name=message,proto3" json:"message,omitempty"`
	Fields    []*MetadataEntry      `protobuf:"bytes,5,rep,name=fields,proto3" json:"fields,omitempty"`
}
func (*LogEntry) ProtoMessage()               {}
func (x *LogEntry) ProtoReflect() interface{} { return nil }

type SpeedTestRequest struct {
	DurationSeconds uint32 `protobuf:"varint,1,opt,name=duration_seconds,json=durationSeconds,proto3" json:"duration_seconds,omitempty"`
	Parallel        uint32 `protobuf:"varint,2,opt,name=parallel,proto3" json:"parallel,omitempty"`
	ServerId        string `protobuf:"bytes,3,opt,name=server_id,json=serverId,proto3" json:"server_id,omitempty"`
}
func (*SpeedTestRequest) ProtoMessage()               {}
func (x *SpeedTestRequest) ProtoReflect() interface{} { return nil }

type SpeedTestSample struct {
	Phase       string          `protobuf:"bytes,1,opt,name=phase,proto3" json:"phase,omitempty"`
	BytesPerSec float32         `protobuf:"fixed32,2,opt,name=bytes_per_sec,json=bytesPerSec,proto3" json:"bytes_per_sec,omitempty"`
	TotalBytes  uint64          `protobuf:"varint,3,opt,name=total_bytes,json=totalBytes,proto3" json:"total_bytes,omitempty"`
	ElapsedMs   uint32          `protobuf:"varint,4,opt,name=elapsed_ms,json=elapsedMs,proto3" json:"elapsed_ms,omitempty"`
	Complete    bool            `protobuf:"varint,5,opt,name=complete,proto3" json:"complete,omitempty"`
	FinalResult *SpeedTestResult `protobuf:"bytes,6,opt,name=final_result,json=finalResult,proto3" json:"final_result,omitempty"`
}
func (*SpeedTestSample) ProtoMessage()               {}
func (x *SpeedTestSample) ProtoReflect() interface{} { return nil }

type SpeedTestResult struct {
	DownloadMbps float32 `protobuf:"fixed32,1,opt,name=download_mbps,json=downloadMbps,proto3" json:"download_mbps,omitempty"`
	UploadMbps   float32 `protobuf:"fixed32,2,opt,name=upload_mbps,json=uploadMbps,proto3" json:"upload_mbps,omitempty"`
	LatencyMs    float32 `protobuf:"fixed32,3,opt,name=latency_ms,json=latencyMs,proto3" json:"latency_ms,omitempty"`
	JitterMs     float32 `protobuf:"fixed32,4,opt,name=jitter_ms,json=jitterMs,proto3" json:"jitter_ms,omitempty"`
	PacketLoss   float32 `protobuf:"fixed32,5,opt,name=packet_loss,json=packetLoss,proto3" json:"packet_loss,omitempty"`
}
func (*SpeedTestResult) ProtoMessage()               {}
func (x *SpeedTestResult) ProtoReflect() interface{} { return nil }

type HealthCheckRequest struct{}
func (*HealthCheckRequest) ProtoMessage()               {}
func (x *HealthCheckRequest) ProtoReflect() interface{} { return nil }

type HealthCheckResponse struct {
	Healthy   bool                  `protobuf:"varint,1,opt,name=healthy,proto3" json:"healthy,omitempty"`
	Version   string                `protobuf:"bytes,2,opt,name=version,proto3" json:"version,omitempty"`
	CheckedAt *timestamppb.Timestamp `protobuf:"bytes,3,opt,name=checked_at,json=checkedAt,proto3" json:"checked_at,omitempty"`
	Checks    []string              `protobuf:"bytes,4,rep,name=checks,proto3" json:"checks,omitempty"`
}
func (*HealthCheckResponse) ProtoMessage()               {}
func (x *HealthCheckResponse) ProtoReflect() interface{} { return nil }

type GetDaemonInfoRequest struct{}
func (*GetDaemonInfoRequest) ProtoMessage()               {}
func (x *GetDaemonInfoRequest) ProtoReflect() interface{} { return nil }

type DaemonInfo struct {
	Version           string                `protobuf:"bytes,1,opt,name=version,proto3" json:"version,omitempty"`
	GoVersion          string                `protobuf:"bytes,2,opt,name=go_version,json=goVersion,proto3" json:"go_version,omitempty"`
	Os                 string                `protobuf:"bytes,3,opt,name=os,proto3" json:"os,omitempty"`
	Arch               string                `protobuf:"bytes,4,opt,name=arch,proto3" json:"arch,omitempty"`
	Uptime             *durationpb.Duration  `protobuf:"bytes,5,opt,name=uptime,proto3" json:"uptime,omitempty"`
	GoroutineCount     uint32                `protobuf:"varint,6,opt,name=goroutine_count,json=goroutineCount,proto3" json:"goroutine_count,omitempty"`
	MemoryAllocBytes   uint64                `protobuf:"varint,7,opt,name=memory_alloc_bytes,json=memoryAllocBytes,proto3" json:"memory_alloc_bytes,omitempty"`
	MemorySysBytes     uint64                `protobuf:"varint,8,opt,name=memory_sys_bytes,json=memorySysBytes,proto3" json:"memory_sys_bytes,omitempty"`
	ActiveConnections  uint32                `protobuf:"varint,9,opt,name=active_connections,json=activeConnections,proto3" json:"active_connections,omitempty"`
	BinaryPath         string                `protobuf:"bytes,10,opt,name=binary_path,json=binaryPath,proto3" json:"binary_path,omitempty"`
	ConfigPath         string                `protobuf:"bytes,11,opt,name=config_path,json=configPath,proto3" json:"config_path,omitempty"`
	DataDir            string                `protobuf:"bytes,12,opt,name=data_dir,json=dataDir,proto3" json:"data_dir,omitempty"`
	Pid                string                `protobuf:"bytes,13,opt,name=pid,proto3" json:"pid,omitempty"`
	StartedAt          *timestamppb.Timestamp `protobuf:"bytes,14,opt,name=started_at,json=startedAt,proto3" json:"started_at,omitempty"`
}
func (*DaemonInfo) ProtoMessage()               {}
func (x *DaemonInfo) ProtoReflect() interface{} { return nil }

type GenerateReportRequest struct {
	IncludeLogs   bool `protobuf:"varint,1,opt,name=include_logs,json=includeLogs,proto3" json:"include_logs,omitempty"`
	IncludeConfig bool `protobuf:"varint,2,opt,name=include_config,json=includeConfig,proto3" json:"include_config,omitempty"`
	IncludeStats  bool `protobuf:"varint,3,opt,name=include_stats,json=includeStats,proto3" json:"include_stats,omitempty"`
}
func (*GenerateReportRequest) ProtoMessage()               {}
func (x *GenerateReportRequest) ProtoReflect() interface{} { return nil }

type GenerateReportResponse struct {
	ReportZip []byte `protobuf:"bytes,1,opt,name=report_zip,json=reportZip,proto3" json:"report_zip,omitempty"`
	Filename   string `protobuf:"bytes,2,opt,name=filename,proto3" json:"filename,omitempty"`
}
func (*GenerateReportResponse) ProtoMessage()               {}
func (x *GenerateReportResponse) ProtoReflect() interface{} { return nil }

// ============================================================================
// BackupService messages
// ============================================================================

type ExportBackupRequest struct {
	IncludeServers      bool   `protobuf:"varint,1,opt,name=include_servers,json=includeServers,proto3" json:"include_servers,omitempty"`
	IncludeSubscriptions bool   `protobuf:"varint,2,opt,name=include_subscriptions,json=includeSubscriptions,proto3" json:"include_subscriptions,omitempty"`
	IncludeSettings      bool   `protobuf:"varint,3,opt,name=include_settings,json=includeSettings,proto3" json:"include_settings,omitempty"`
	IncludeRouting       bool   `protobuf:"varint,4,opt,name=include_routing,json=includeRouting,proto3" json:"include_routing,omitempty"`
	Encrypt              bool   `protobuf:"varint,5,opt,name=encrypt,proto3" json:"encrypt,omitempty"`
	Passphrase           string `protobuf:"bytes,6,opt,name=passphrase,proto3" json:"passphrase,omitempty"`
}
func (*ExportBackupRequest) ProtoMessage()               {}
func (x *ExportBackupRequest) ProtoReflect() interface{} { return nil }

type ExportBackupResponse struct {
	Data     []byte `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	Filename string `protobuf:"bytes,2,opt,name=filename,proto3" json:"filename,omitempty"`
}
func (*ExportBackupResponse) ProtoMessage()               {}
func (x *ExportBackupResponse) ProtoReflect() interface{} { return nil }

type ImportBackupRequest struct {
	Data       []byte `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	Passphrase string `protobuf:"bytes,2,opt,name=passphrase,proto3" json:"passphrase,omitempty"`
	Merge      bool   `protobuf:"varint,3,opt,name=merge,proto3" json:"merge,omitempty"`
}
func (*ImportBackupRequest) ProtoMessage()               {}
func (x *ImportBackupRequest) ProtoReflect() interface{} { return nil }

type ImportBackupResponse struct {
	ServersImported      int32    `protobuf:"varint,1,opt,name=servers_imported,json=serversImported,proto3" json:"servers_imported,omitempty"`
	SubscriptionsImported int32   `protobuf:"varint,2,opt,name=subscriptions_imported,json=subscriptionsImported,proto3" json:"subscriptions_imported,omitempty"`
	SettingsApplied       bool     `protobuf:"varint,3,opt,name=settings_applied,json=settingsApplied,proto3" json:"settings_applied,omitempty"`
	RoutingApplied        bool     `protobuf:"varint,4,opt,name=routing_applied,json=routingApplied,proto3" json:"routing_applied,omitempty"`
	Warnings              []string `protobuf:"bytes,5,rep,name=warnings,proto3" json:"warnings,omitempty"`
}
func (*ImportBackupResponse) ProtoMessage()               {}
func (x *ImportBackupResponse) ProtoReflect() interface{} { return nil }
