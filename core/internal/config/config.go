package config

import (
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config represents the full application configuration loaded from YAML.
type Config struct {
	Daemon  DaemonConfig  `yaml:"daemon"`
	Tunnel  TunnelConfig  `yaml:"tunnel"`
	DNS     DNSConfig     `yaml:"dns"`
	Log     LogConfig     `yaml:"log"`
	Fallback FallbackConfig `yaml:"fallback"`
	Servers []ServerEntry `yaml:"servers"`
	Subscriptions []SubscriptionEntry `yaml:"subscriptions"`
	Routing RoutingConfig `yaml:"routing"`
}

// DaemonConfig holds daemon-specific settings.
type DaemonConfig struct {
	GRPCAddr         string `yaml:"grpc_addr"`          // e.g. "127.0.0.1:50051"
	DataDir          string `yaml:"data_dir"`           // path to persistent data
	BinDir           string `yaml:"bin_dir"`            // path to protocol binaries
	ConfigDir        string `yaml:"config_dir"`         // path to config templates
	AutoConnect      bool   `yaml:"auto_connect"`
	ConnectOnStartup bool   `yaml:"connect_on_startup"`
	KillSwitch       bool   `yaml:"kill_switch"`
	AllowLAN         bool   `yaml:"allow_lan"`
	Language         string `yaml:"language"`           // BCP-47, e.g. "ru", "en"
	Theme            string `yaml:"theme"`              // "system", "light", "dark"
}

// TunnelConfig holds tunnel/connection settings.
type TunnelConfig struct {
	ProxyMode         string `yaml:"proxy_mode"`          // "system", "socks", "http", "pac"
	SOCKSPort         uint32 `yaml:"socks_port"`
	HTTPPort          uint32 `yaml:"http_port"`
	ConnectionTimeout uint32 `yaml:"connection_timeout"`  // seconds
	ReconnectAttempts uint32 `yaml:"reconnect_attempts"`
	ReconnectDelay    uint32 `yaml:"reconnect_delay"`     // seconds
	ActiveServerID    string `yaml:"active_server_id"`
	MTU               uint32 `yaml:"mtu"`                 // TUN MTU, default 1420
}

// DNSConfig holds DNS resolver settings.
type DNSConfig struct {
	DNSServers   string `yaml:"dns_servers"`     // comma-separated, e.g. "1.1.1.1,8.8.8.8"
	DoHEnabled   bool   `yaml:"doh_enabled"`
	DoHURL       string `yaml:"doh_url"`         // e.g. "https://dns.google/dns-query"
	DNSProxyPort uint32 `yaml:"dns_proxy_port"` // local DNS proxy port
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level  string `yaml:"level"`  // "debug", "info", "warn", "error"
	File   string `yaml:"file"`   // log file path, empty = stderr only
	MaxSize int    `yaml:"max_size"` // MB before rotation
}

// FallbackConfig holds fallback behavior settings.
type FallbackConfig struct {
	Enabled           bool   `yaml:"enabled"`
	CheckInterval     uint32 `yaml:"check_interval"`     // seconds between health checks
	MaxUnhealthyCount uint32 `yaml:"max_unhealthy_count"` // how many servers must fail before fallback
	RecoveryInterval  uint32 `yaml:"recovery_interval"`  // seconds between recovery checks
	FallbackServerID  string `yaml:"fallback_server_id"` // local WG/AWG server to use
}

// ServerEntry is a persisted server configuration.
type ServerEntry struct {
	ID             string `yaml:"id"`
	Name           string `yaml:"name"`
	Host           string `yaml:"host"`
	Port           uint32 `yaml:"port"`
	Protocol       string `yaml:"protocol"`
	Favorite       bool   `yaml:"favorite"`
	SortOrder      uint32 `yaml:"sort_order"`
	Tags           []string `yaml:"tags"`
	SubscriptionID string `yaml:"subscription_id"`
	// Protocol-specific fields stored as nested YAML
	XrayConfig    *XrayConfigEntry    `yaml:"xray_config,omitempty"`
	WGConfig      *WGConfigEntry      `yaml:"wg_config,omitempty"`
	HysteriaConfig *HysteriaConfigEntry `yaml:"hysteria_config,omitempty"`
	AmneziaConfig *AmneziaConfigEntry  `yaml:"amnezia_config,omitempty"`
}

// XrayConfigEntry holds Xray/VLESS/VMESS specific fields.
type XrayConfigEntry struct {
	UUID         string `yaml:"uuid"`
	Flow         string `yaml:"flow"`
	Security     string `yaml:"security"`
	Transport    string `yaml:"transport"`
	SNI          string `yaml:"sni"`
	Fingerprint  string `yaml:"fingerprint"`
	ALPN         string `yaml:"alpn"`
	PublicKey    string `yaml:"public_key"`
	ShortID      string `yaml:"short_id"`
	KCPSeed      string `yaml:"kcp_seed"`
	XHTTPPath    string `yaml:"xhttp_path"`
	XHTTPMode    string `yaml:"xhttp_mode"`
	WSPath       string `yaml:"ws_path"`
	GRPCService  string `yaml:"grpc_service"`
	AllowInsecure bool  `yaml:"allow_insecure"`
}

// WGConfigEntry holds WireGuard specific fields.
type WGConfigEntry struct {
	PrivateKey        string `yaml:"private_key"`
	PublicKey         string `yaml:"public_key"`
	PresharedKey      string `yaml:"preshared_key"`
	LocalAddress      string `yaml:"local_address"`
	DNSServers        string `yaml:"dns_servers"`
	MTU               uint32 `yaml:"mtu"`
	PersistentKeepalive uint32 `yaml:"persistent_keepalive"`
	AllowedIPs        string `yaml:"allowed_ips"`
}

// HysteriaConfigEntry holds Hysteria2 specific fields.
type HysteriaConfigEntry struct {
	AuthPassword string `yaml:"auth_password"`
	SNI          string `yaml:"sni"`
	Insecure     bool   `yaml:"insecure"`
	ALPN         string `yaml:"alpn"`
	ObfsPassword string `yaml:"obfs_password"`
	BandwidthUp  uint64 `yaml:"bandwidth_up"`
	BandwidthDown uint64 `yaml:"bandwidth_down"`
	FastOpen     bool   `yaml:"fast_open"`
}

// AmneziaConfigEntry holds AmneziaWG specific fields.
type AmneziaConfigEntry struct {
	PrivateKey string `yaml:"private_key"`
	PublicKey  string `yaml:"public_key"`
	PresharedKey string `yaml:"preshared_key"`
	LocalAddress string `yaml:"local_address"`
	DNSServers   string `yaml:"dns_servers"`
	MTU          uint32 `yaml:"mtu"`
	Jc           uint32 `yaml:"jc"`
	Jmin         uint32 `yaml:"jmin"`
	Jmax         uint32 `yaml:"jmax"`
	S1           uint32 `yaml:"s1"`
	S2           uint32 `yaml:"s2"`
	H1           uint32 `yaml:"h1"`
	H2           uint32 `yaml:"h2"`
	H3           uint32 `yaml:"h3"`
	H4           uint32 `yaml:"h4"`
}

// SubscriptionEntry is a persisted subscription source.
type SubscriptionEntry struct {
	ID               string `yaml:"id"`
	Name             string `yaml:"name"`
	URL              string `yaml:"url"`
	Username         string `yaml:"username"`
	Password         string `yaml:"password"`
	RefreshInterval  uint32 `yaml:"refresh_interval"`  // minutes, 0 = manual
	Filter           string `yaml:"filter"`
	Enabled          bool   `yaml:"enabled"`
}

// RoutingConfig holds split tunneling rules.
type RoutingConfig struct {
	DomainStrategy string       `yaml:"domain_strategy"`
	Rules          []RoutingRule `yaml:"rules"`
}

// RoutingRule is a single split tunneling rule.
type RoutingRule struct {
	ID        string   `yaml:"id"`
	Name      string   `yaml:"name"`
	Enabled   bool     `yaml:"enabled"`
	Action    string   `yaml:"action"`     // "proxy", "direct", "block"
	Domains   []string `yaml:"domains"`
	IPCidrs   []string `yaml:"ip_cidrs"`
	GeoIPCodes []string `yaml:"geoip_codes"`
	Processes []string `yaml:"processes"`
	Ports     []string `yaml:"ports"`
	Protocols []string `yaml:"protocols"`
}

// Manager handles loading, saving, and accessing the application config.
type Manager struct {
	mu      sync.RWMutex
	path    string
	config  *Config
}

// NewManager creates a new config manager.
func NewManager(path string) *Manager {
	return &Manager{
		path:   path,
		config: DefaultConfig(),
	}
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() *Config {
	exe, _ := os.Executable()
	baseDir := filepath.Dir(exe)
	return &Config{
		Daemon: DaemonConfig{
			GRPCAddr:  "127.0.0.1:50051",
			DataDir:   filepath.Join(baseDir, "data"),
			BinDir:    filepath.Join(baseDir, "bin"),
			ConfigDir: filepath.Join(baseDir, "configs"),
			Language:  "en",
			Theme:     "dark",
		},
		Tunnel: TunnelConfig{
			ProxyMode:         "system",
			SOCKSPort:         1080,
			HTTPPort:          8080,
			ConnectionTimeout: 30,
			ReconnectAttempts: 3,
			ReconnectDelay:    5,
			MTU:               1420,
		},
		DNS: DNSConfig{
			DNSServers:   "1.1.1.1,8.8.8.8",
			DoHEnabled:   true,
			DoHURL:       "https://dns.google/dns-query",
			DNSProxyPort: 15353,
		},
		Log: LogConfig{
			Level:   "info",
			MaxSize: 10,
		},
		Fallback: FallbackConfig{
			Enabled:           true,
			CheckInterval:     300,
			MaxUnhealthyCount: 1,
			RecoveryInterval:  300,
		},
		Routing: RoutingConfig{
			DomainStrategy: "IPIfNonMatch",
		},
	}
}

// Load reads the config from disk. If the file doesn't exist, uses defaults.
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			return m.saveLocked()
		}
		return err
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return err
	}

	m.config = cfg
	return nil
}

// Save writes the current config to disk.
func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveLocked()
}

func (m *Manager) saveLocked() error {
	// Ensure parent directory exists
	if dir := filepath.Dir(m.path); dir != "" {
		os.MkdirAll(dir, 0755)
	}

	data, err := yaml.Marshal(m.config)
	if err != nil {
		return err
	}
	return os.WriteFile(m.path, data, 0644)
}

// Get returns a copy of the current config.
func (m *Manager) Get() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return *m.config
}

// Update applies a function to modify the config, then saves.
func (m *Manager) Update(fn func(*Config)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	fn(m.config)
	return m.saveLocked()
}

// GetServers returns the list of configured servers.
func (m *Manager) GetServers() []ServerEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]ServerEntry, len(m.config.Servers))
	copy(result, m.config.Servers)
	return result
}

// GetSubscriptions returns the list of subscriptions.
func (m *Manager) GetSubscriptions() []SubscriptionEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]SubscriptionEntry, len(m.config.Subscriptions))
	copy(result, m.config.Subscriptions)
	return result
}

// GetRouting returns the routing config.
func (m *Manager) GetRouting() RoutingConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Routing
}
