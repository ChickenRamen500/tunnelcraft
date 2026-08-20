package engine

import (
        "sync"
        "time"

        "github.com/google/uuid"
)

// ConnectionState represents the current VPN connection state.
type ConnectionState int

const (
        StateDisconnected  ConnectionState = iota
        StateConnecting
        StateConnected
        StateReconnecting
        StateDisconnecting
        StateError
        StateFallback      // using local WG/AWG fallback
        StateFallbackConnected
)

// String returns a human-readable state name.
func (s ConnectionState) String() string {
        switch s {
        case StateDisconnected:
                return "Disconnected"
        case StateConnecting:
                return "Connecting"
        case StateConnected:
                return "Connected"
        case StateReconnecting:
                return "Reconnecting"
        case StateDisconnecting:
                return "Disconnecting"
        case StateError:
                return "Error"
        case StateFallback:
                return "Fallback"
        case StateFallbackConnected:
                return "Fallback Connected"
        default:
                return "Unknown"
        }
}

// Protocol identifies the VPN protocol type.
type Protocol string

const (
        ProtocolVLESS     Protocol = "vless"
        ProtocolVMESS     Protocol = "vmess"
        ProtocolWireGuard Protocol = "wireguard"
        ProtocolHysteria  Protocol = "hysteria"
        ProtocolAmneziaWG Protocol = "amneziawg"
)

// ServerConfig is the unified server configuration used internally.
// This is the single source of truth for all protocol-specific settings.
type ServerConfig struct {
        ID             string
        Name           string
        Host           string
        Port           uint32
        Protocol       Protocol
        Tags           []string
        Favorite       bool
        SortOrder      uint32
        SubscriptionID string
        // Xray-specific
        UUID        string
        Flow        string
        Security    string    // "none", "tls", "reality"
        Transport   string    // "tcp", "ws", "grpc", "kcp", "xhttp"
        SNI         string
        Fingerprint string
        ALPN        string
        PublicKey   string    // Reality public key
        ShortID     string    // Reality short ID
        KCPSeed     string
        XHTTPPath   string
        XHTTPMode   string
        WSPath      string
        GRPCService string
        AllowInsecure bool
        // WireGuard-specific
        WGPrivateKey        string
        WGPublicKey         string
        WGPresharedKey      string
        WGLocalAddress      string
        WGDNSServers        string
        WGAllowedIPs        string
        // Hysteria-specific
        HysteriaAuth       string
        HysteriaSNI        string
        HysteriaInsecure   bool
        HysteriaALPN       string
        HysteriaObfs       string
        HysteriaBwUp       uint64
        HysteriaBwDown     uint64
        HysteriaFastOpen   bool
        // AmneziaWG-specific (supports AWG2 and AWG3).
        AmneziaPrivateKey   string
        AmneziaPublicKey    string
        AmneziaPresharedKey string
        AmneziaLocalAddr    string
        AmneziaDNS          string
        AmneziaJc           uint32
        AmneziaJmin         uint32
        AmneziaJmax         uint32
        AmneziaS1           uint32
        AmneziaS2           uint32
        AmneziaS3           uint32 // Cookie response packet padding (AWG2+).
        // H1-H4 are strings to support both AWG2 ("123") and AWG3 range syntax
        // (e.g. "0xff-0xffff", "[deadbeef]", "<12>", "{8}", "(6)", "t").
        AmneziaH1           string
        AmneziaH2           string
        AmneziaH3           string
        AmneziaH4           string
        // AWG3-specific fields.
        AmneziaHeaderProtectionKey    string // ChaCha20 key for header encryption.
        AmneziaContentPaddingAddition string // Range string, e.g. "0-100".
        // AWG3 timing fields (range strings, e.g. "120", "120-180"). 0-length = not set / AWG2 mode.
        AmneziaRekeyAfterTime      string // seconds
        AmneziaRekeyTimeout        string // seconds
        AmneziaRejectAfterTime     string // seconds
        AmneziaKeepaliveTimeout    string // seconds
        AmneziaMaxHandshakeAttempts string // amount
}

// GenerateID creates a new UUID for a server.
func GenerateID() string {
        return uuid.New().String()
}

// ConnectionStats tracks bandwidth and timing for the active connection.
type ConnectionStats struct {
        mu              sync.RWMutex
        BytesUp         uint64
        BytesDown       uint64
        ConnectedAt     time.Time
        LastActivityAt  time.Time
        CurrentSpeedUp  float64 // bytes/sec
        CurrentSpeedDown float64
}

// AddUpload increments the upload byte counter.
func (s *ConnectionStats) AddUpload(n uint64) {
        s.mu.Lock()
        defer s.mu.Unlock()
        s.BytesUp += n
        s.LastActivityAt = time.Now()
}

// AddDownload increments the download byte counter.
func (s *ConnectionStats) AddDownload(n uint64) {
        s.mu.Lock()
        defer s.mu.Unlock()
        s.BytesDown += n
        s.LastActivityAt = time.Now()
}

// Snapshot returns a copy of the current stats.
func (s *ConnectionStats) Snapshot() (bytesUp, bytesDown uint64, duration time.Duration) {
        s.mu.RLock()
        defer s.mu.RUnlock()
        duration = time.Since(s.ConnectedAt)
        return s.BytesUp, s.BytesDown, duration
}

// SetSpeeds updates the current speed readings.
func (s *ConnectionStats) SetSpeeds(up, down float64) {
        s.mu.Lock()
        defer s.mu.Unlock()
        s.CurrentSpeedUp = up
        s.CurrentSpeedDown = down
}

// GetSpeeds returns current speed readings.
func (s *ConnectionStats) GetSpeeds() (up, down float64) {
        s.mu.RLock()
        defer s.mu.RUnlock()
        return s.CurrentSpeedUp, s.CurrentSpeedDown
}

// Reset clears all counters.
func (s *ConnectionStats) Reset() {
        s.mu.Lock()
        defer s.mu.Unlock()
        s.BytesUp = 0
        s.BytesDown = 0
        s.ConnectedAt = time.Time{}
        s.LastActivityAt = time.Time{}
        s.CurrentSpeedUp = 0
        s.CurrentSpeedDown = 0
}

// ServerHealth tracks the health status of a server.
type ServerHealth struct {
        ServerID  string
        Reachable bool
        Latency   time.Duration
        LastCheck time.Time
        Error     string
}

// ConnectionEvent is emitted when connection state changes.
type ConnectionEvent struct {
        State    ConnectionState
        Message   string
        ServerID string
        Error    string
        Time     time.Time
}

// LogEntry represents a single log line.
type LogEntry struct {
        Timestamp time.Time
        Level     string // "debug", "info", "warn", "error"
        Component string
        Message   string
        Fields    map[string]string
}
