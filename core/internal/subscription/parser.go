// Package subscription parses VPN subscription payloads in multiple formats:
// base64-encoded share-link lists, SIP008 JSON, sing-box JSON, and Clash YAML.
package subscription

import (
        "encoding/base64"
        "encoding/json"
        "fmt"
        "net"
        "net/url"
        "strconv"
        "strings"

        "github.com/ChickenRamen500/tunnelcraft/core/internal/engine"
        "github.com/google/uuid"
        "gopkg.in/yaml.v3"
)

// ParseError captures a per-line or per-entry parsing failure so callers can
// display actionable diagnostics without aborting the entire parse.
type ParseError struct {
        Line    int
        Message string
}

func (e ParseError) Error() string {
        if e.Line > 0 {
                return fmt.Sprintf("line %d: %s", e.Line, e.Message)
        }
        return e.Message
}

// ---------------------------------------------------------------------------
// DetectFormat
// ---------------------------------------------------------------------------

// DetectFormat inspects raw bytes and returns a format tag without parsing.
// Possible return values: "base64-links", "sip008", "singbox", "clash", "unknown".
func DetectFormat(rawContent []byte) string {
        trimmed := strings.TrimSpace(string(rawContent))
        if trimmed == "" {
                return "unknown"
        }

        // 1. Try base64 decode first.
        decoded, err := tryBase64Decode(rawContent)
        if err == nil {
                decodedStr := strings.TrimSpace(string(decoded))

                // Decoded content is JSON – check SIP008 first (more specific), then sing-box.
                if isJSON(decodedStr) {
                        if isSIP008JSON(decoded) {
                                return "sip008"
                        }
                        if isSingBoxJSON(decoded) {
                                return "singbox"
                        }
                        // Base64-encoded JSON that isn't SIP008 or sing-box.
                        return "base64-links"
                }

                // Decoded content is YAML (Clash).
                if isClashYAML(decoded) {
                        return "clash"
                }

                // Decoded plain text – check if it looks like share links.
                if hasShareLinkPrefix(decodedStr) {
                        return "base64-links"
                }

                // If none of the above, still call it base64-links (the caller will
                // attempt line-by-line parsing and report errors for unrecognised lines).
                return "base64-links"
        }

        // 2. Not valid base64 – inspect raw content directly.
        // JSON takes priority over YAML (JSON is a valid YAML subset).
        if isJSON(trimmed) {
                if isSIP008JSON(rawContent) {
                        return "sip008"
                }
                if isSingBoxJSON(rawContent) {
                        return "singbox"
                }
        }
        if isClashYAML(rawContent) {
                return "clash"
        }
        if hasShareLinkPrefix(trimmed) {
                return "base64-links"
        }

        return "unknown"
}

// ---------------------------------------------------------------------------
// Parse – main entry point
// ---------------------------------------------------------------------------

// Parse accepts raw subscription bytes and returns every server it can extract
// along with any per-entry parse errors.  It is tolerant of malformed lines:
// bad entries are reported but do not prevent sibling entries from being parsed.
func Parse(rawContent []byte) ([]engine.ServerConfig, []ParseError) {
        var servers []engine.ServerConfig
        var errs []ParseError

        trimmed := strings.TrimSpace(string(rawContent))
        if trimmed == "" {
                return servers, errs
        }

        // Attempt base64 decode.
        decoded, b64err := tryBase64Decode(rawContent)

        // Determine working payload: prefer decoded bytes, fall back to raw.
        var payload []byte
        if b64err == nil {
                payload = decoded
        } else {
                payload = rawContent
        }

        payloadStr := strings.TrimSpace(string(payload))

        // ---- Route to the correct sub-parser. ----

        // JSON takes priority over YAML since JSON is a subset of YAML and
        // yaml.Unmarshal would happily consume JSON.
        if isJSON(payloadStr) {
                if isSIP008JSON(payload) {
                        return parseSIP008(payload)
                }
                if isSingBoxJSON(payload) {
                        return parseSingBox(payload)
                }
                // Unknown JSON format.
                return servers, append(errs, ParseError{Message: "unrecognised JSON format"})
        }

        // Check for Clash YAML (non-JSON YAML with proxies key).
        if isClashYAML(payload) {
                return parseClash(payload)
        }

        // Treat as newline-separated share links.
        lines := strings.Split(payloadStr, "\n")
        for i, line := range lines {
                line = strings.TrimSpace(line)
                if line == "" {
                        continue
                }

                s, err := parseShareLink(line)
                if err != nil {
                        errs = append(errs, ParseError{Line: i + 1, Message: err.Error()})
                        continue
                }
                servers = append(servers, s)
        }

        return servers, errs
}

// parseShareLink routes a single line to the appropriate protocol parser.
func parseShareLink(line string) (engine.ServerConfig, error) {
        switch {
        case strings.HasPrefix(line, "vless://"):
                return parseVLESS(line)
        case strings.HasPrefix(line, "vmess://"):
                return parseVMESS(line)
        case strings.HasPrefix(line, "hysteria2://") || strings.HasPrefix(line, "hy2://"):
                return parseHysteria2(line)
        case strings.HasPrefix(line, "wg://"):
                return parseWireGuard(line)
        case strings.HasPrefix(line, "awg://"):
                return parseAmneziaWG(line)
        default:
                return engine.ServerConfig{}, fmt.Errorf("unrecognised link prefix: %s", truncate(line, 30))
        }
}

// ---------------------------------------------------------------------------
// VLESS parser
// ---------------------------------------------------------------------------

// parseVLESS handles the URI form:
//
//      vless://uuid@host:port?type=tcp&security=tls&path=/&...#fragment-name
//
// All query parameters are extracted into the corresponding ServerConfig fields.
func parseVLESS(raw string) (engine.ServerConfig, error) {
        var cfg engine.ServerConfig
        cfg.ID = uuid.New().String()
        cfg.Protocol = engine.ProtocolVLESS

        // Strip the scheme.
        body, err := trimScheme(raw, "vless://")
        if err != nil {
                return cfg, err
        }

        // Split fragment (name).
        body, frag := splitFragment(body)
        cfg.Name = frag

        // Split query string.
        body, queryStr := splitQuery(body)
        query, err := url.ParseQuery(queryStr)
        if err != nil {
                return cfg, fmt.Errorf("invalid query string: %w", err)
        }

        // Parse uuid@host:port.
        host, port, uuidPart, err := parseUserHostPort(body)
        if err != nil {
                return cfg, err
        }
        cfg.UUID = uuidPart
        cfg.Host = host
        cfg.Port = port

        // --- Transport ---
        // "type" in VLESS URI = transport type.
        cfg.Transport = query.Get("type")
        cfg.WSPath = query.Get("path")
        cfg.GRPCService = query.Get("serviceName")
        cfg.KCPSeed = query.Get("seed")
        // KCP header type (not stored directly, but seed is the main KCP param).

        // xhttp-specific overrides.
        if v := query.Get("xhttpPath"); v != "" {
                cfg.XHTTPPath = v
        }
        if v := query.Get("xhttpMode"); v != "" {
                cfg.XHTTPMode = v
        }
        // Some implementations use mode=xhttp instead of type=xhttp.
        if query.Get("mode") == "xhttp" {
                cfg.Transport = "xhttp"
        }

        // --- Security ---
        cfg.Security = query.Get("security")
        cfg.SNI = query.Get("sni")
        cfg.Fingerprint = query.Get("fp")
        cfg.ALPN = query.Get("alpn")
        cfg.PublicKey = query.Get("pbk")
        cfg.ShortID = query.Get("sid")
        cfg.AllowInsecure = query.Get("allowInsecure") == "1"

        // --- Flow ---
        cfg.Flow = query.Get("flow")

        return cfg, nil
}

// ---------------------------------------------------------------------------
// VMESS parser
// ---------------------------------------------------------------------------

// vmessJSON mirrors the V2RayN vmess:// base64-encoded JSON fields.
type vmessJSON struct {
        V    string `json:"v"`    // version
        PS   string `json:"ps"`   // remarks / name
        Add  string `json:"add"`  // server address
        Port string `json:"port"` // server port (string)
        ID   string `json:"id"`   // UUID
        Aid  string `json:"aid"`  // alter ID
        Scy  string `json:"scy"`  // cipher / security method
        Net  string `json:"net"`  // transport network (tcp, ws, grpc, h2, kcp)
        Type string `json:"type"` // transport header type
        Host string `json:"host"` // host header for ws/h2
        Path string `json:"path"` // path for ws/h2
        TLS  string `json:"tls"`  // "tls" or ""
        SNI  string `json:"sni"`  // server name indication
        ALPN string `json:"alpn"` // ALPN protocols (comma-separated)
        FP   string `json:"fp"`   // uTLS fingerprint
}

// parseVMESS decodes a vmess:// URI.  The body after "vmess://" is expected to
// be standard base64 (with URL-safe or standard alphabet) that decodes to JSON.
func parseVMESS(raw string) (engine.ServerConfig, error) {
        var cfg engine.ServerConfig
        cfg.ID = uuid.New().String()
        cfg.Protocol = engine.ProtocolVMESS

        body, err := trimScheme(raw, "vmess://")
        if err != nil {
                return cfg, err
        }

        // The payload may contain padding or not; try both standard and URL-safe.
        decoded, err := base64DecodeAny(body)
        if err != nil {
                return cfg, fmt.Errorf("vmess base64 decode: %w", err)
        }

        var v vmessJSON
        if err := json.Unmarshal(decoded, &v); err != nil {
                return cfg, fmt.Errorf("vmess JSON unmarshal: %w", err)
        }

        cfg.Name = v.PS
        cfg.Host = v.Add
        cfg.UUID = v.ID

        port, err := strconv.ParseUint(strings.TrimSpace(v.Port), 10, 32)
        if err != nil {
                return cfg, fmt.Errorf("vmess invalid port %q: %w", v.Port, err)
        }
        cfg.Port = uint32(port)

        // Transport.
        cfg.Transport = v.Net
        cfg.WSPath = v.Path
        cfg.GRPCService = v.Host // In V2RayN, for grpc, "host" field holds service name.
        // For ws/h2, "host" is the Host header.
        if v.Net == "ws" || v.Net == "h2" {
                cfg.WSPath = v.Path
        }
        if v.Net == "grpc" {
                cfg.GRPCService = v.Path
        }

        // Security / TLS.
        switch v.TLS {
        case "tls":
                cfg.Security = "tls"
        case "reality":
                cfg.Security = "reality"
        default:
                cfg.Security = "none"
        }

        cfg.SNI = v.SNI
        cfg.Fingerprint = v.FP
        cfg.ALPN = v.ALPN

        // Cipher is informational only (mapped to a field we don't have, skip).
        _ = v.Scy
        _ = v.Aid

        return cfg, nil
}

// ---------------------------------------------------------------------------
// Hysteria2 parser
// ---------------------------------------------------------------------------

// parseHysteria2 handles both hysteria2:// and hy2:// URI forms:
//
//      hysteria2://auth@host:port/?insecure=1&obfs=salamander&obfs-password=x&sni=y&alpn=z&upmbps=100&downmbps=200#name
func parseHysteria2(raw string) (engine.ServerConfig, error) {
        var cfg engine.ServerConfig
        cfg.ID = uuid.New().String()
        cfg.Protocol = engine.ProtocolHysteria

        // Normalise scheme.
        var body string
        switch {
        case strings.HasPrefix(raw, "hysteria2://"):
                body = raw[len("hysteria2://"):]
        case strings.HasPrefix(raw, "hy2://"):
                body = raw[len("hy2://"):]
        default:
                return cfg, fmt.Errorf("not a hysteria2 URI")
        }

        // Split fragment.
        body, frag := splitFragment(body)
        cfg.Name = frag

        // Split query.
        body, queryStr := splitQuery(body)
        query, err := url.ParseQuery(queryStr)
        if err != nil {
                return cfg, fmt.Errorf("invalid query string: %w", err)
        }

        // Parse auth@host:port.
        host, port, auth, err := parseUserHostPort(body)
        if err != nil {
                return cfg, err
        }
        cfg.Host = host
        cfg.Port = port
        cfg.HysteriaAuth = auth

        // Query params.
        cfg.HysteriaSNI = query.Get("sni")
        cfg.HysteriaInsecure = query.Get("insecure") == "1"
        cfg.HysteriaObfs = query.Get("obfs")
        cfg.HysteriaALPN = query.Get("alpn")

        // Bandwidth in Mbps.
        if v := query.Get("upmbps"); v != "" {
                if mbps, e := strconv.ParseUint(v, 10, 64); e == nil {
                        cfg.HysteriaBwUp = mbps * 1_000_000 // convert to bits/s approx.
                }
        }
        if v := query.Get("downmbps"); v != "" {
                if mbps, e := strconv.ParseUint(v, 10, 64); e == nil {
                        cfg.HysteriaBwDown = mbps * 1_000_000
                }
        }

        // obfs-password may be appended to the obfs param in some implementations.
        if v := query.Get("obfs-password"); v != "" {
                cfg.HysteriaObfs = v
        }

        return cfg, nil
}

// ---------------------------------------------------------------------------
// WireGuard parser  (wg://)
// ---------------------------------------------------------------------------

// wgConfigJSON is the expected JSON shape embedded (base64) inside a wg:// link.
type wgConfigJSON struct {
        PrivateKey   string `json:"private_key"`
        PublicKey    string `json:"public_key"`
        PresharedKey string `json:"preshared_key"`
        Address      string `json:"address"`
        DNS          string `json:"dns"`
        Endpoint     string `json:"endpoint"`
        AllowedIPs   string `json:"allowed_ips"`
        Name         string `json:"name"`
}

// parseWireGuard handles the wg:// URI.  The body after "wg://" is treated as
// base64-encoded JSON containing WireGuard connection parameters.  If base64
// decoding fails, the parser falls back to a URL-style format:
//
//      wg://privatekey@host:port?publickey=x&address=y&dns=z#name
func parseWireGuard(raw string) (engine.ServerConfig, error) {
        var cfg engine.ServerConfig
        cfg.ID = uuid.New().String()
        cfg.Protocol = engine.ProtocolWireGuard

        body, err := trimScheme(raw, "wg://")
        if err != nil {
                return cfg, err
        }

        // Split fragment (name).
        body, frag := splitFragment(body)
        cfg.Name = frag

        // Attempt base64-decoded JSON payload.
        decoded, b64Err := base64DecodeAny(body)
        if b64Err == nil {
                var w wgConfigJSON
                if jsonErr := json.Unmarshal(decoded, &w); jsonErr == nil {
                        cfg.WGPrivateKey = w.PrivateKey
                        cfg.WGPublicKey = w.PublicKey
                        cfg.WGPresharedKey = w.PresharedKey
                        cfg.WGLocalAddress = w.Address
                        cfg.WGDNSServers = w.DNS
                        cfg.WGAllowedIPs = w.AllowedIPs
                        if w.Name != "" {
                                cfg.Name = w.Name
                        }

                        // Endpoint may be "host:port".
                        if w.Endpoint != "" {
                                h, p, pErr := netSplitHostPort(w.Endpoint)
                                if pErr == nil {
                                        cfg.Host = h
                                        cfg.Port = p
                                }
                        }
                        return cfg, nil
                }
        }

        // Fallback: URL-style wg://privatekey@host:port?params#name
        body, queryStr := splitQuery(body)
        query, qErr := url.ParseQuery(queryStr)
        if qErr != nil {
                return cfg, fmt.Errorf("wg invalid query: %w", qErr)
        }

        host, port, privKey, uErr := parseUserHostPort(body)
        if uErr != nil {
                return cfg, uErr
        }
        cfg.WGPrivateKey = privKey
        cfg.Host = host
        cfg.Port = port

        cfg.WGPublicKey = query.Get("publickey")
        cfg.WGPresharedKey = query.Get("presharedkey")
        cfg.WGLocalAddress = query.Get("address")
        cfg.WGDNSServers = query.Get("dns")
        cfg.WGAllowedIPs = query.Get("allowedips")

        return cfg, nil
}

// ---------------------------------------------------------------------------
// AmneziaWG parser  (awg://)
// ---------------------------------------------------------------------------

// awgConfigJSON extends the wgConfigJSON with AmneziaWG junk-packet parameters.
type awgConfigJSON struct {
        wgConfigJSON
        Jc   uint32 `json:"jc"`
        Jmin uint32 `json:"jmin"`
        Jmax uint32 `json:"jmax"`
        S1   uint32 `json:"s1"`
        S2   uint32 `json:"s2"`
        H1   uint32 `json:"h1"`
        H2   uint32 `json:"h2"`
        H3   uint32 `json:"h3"`
        H4   uint32 `json:"h4"`
}

// parseAmneziaWG is identical to parseWireGuard but also extracts the
// AmneziaWG junk-packet fields (Jc, Jmin, Jmax, S1, S2, H1–H4).
func parseAmneziaWG(raw string) (engine.ServerConfig, error) {
        var cfg engine.ServerConfig
        cfg.ID = uuid.New().String()
        cfg.Protocol = engine.ProtocolAmneziaWG

        body, err := trimScheme(raw, "awg://")
        if err != nil {
                return cfg, err
        }

        // Fragment.
        body, frag := splitFragment(body)
        cfg.Name = frag

        // Try base64-decoded JSON.
        decoded, b64Err := base64DecodeAny(body)
        if b64Err == nil {
                var a awgConfigJSON
                if jsonErr := json.Unmarshal(decoded, &a); jsonErr == nil {
                        cfg.AmneziaPrivateKey = a.PrivateKey
                        cfg.AmneziaPublicKey = a.PublicKey
                        cfg.AmneziaPresharedKey = a.PresharedKey
                        cfg.AmneziaLocalAddr = a.Address
                        cfg.AmneziaDNS = a.DNS
                        cfg.WGAllowedIPs = a.AllowedIPs
                        if a.Name != "" {
                                cfg.Name = a.Name
                        }
                        cfg.AmneziaJc = a.Jc
                        cfg.AmneziaJmin = a.Jmin
                        cfg.AmneziaJmax = a.Jmax
                        cfg.AmneziaS1 = a.S1
                        cfg.AmneziaS2 = a.S2
                        cfg.AmneziaH1 = a.H1
                        cfg.AmneziaH2 = a.H2
                        cfg.AmneziaH3 = a.H3
                        cfg.AmneziaH4 = a.H4

                        if a.Endpoint != "" {
                                h, p, pErr := netSplitHostPort(a.Endpoint)
                                if pErr == nil {
                                        cfg.Host = h
                                        cfg.Port = p
                                }
                        }
                        return cfg, nil
                }
        }

        // Fallback: URL-style.
        body, queryStr := splitQuery(body)
        query, qErr := url.ParseQuery(queryStr)
        if qErr != nil {
                return cfg, fmt.Errorf("awg invalid query: %w", qErr)
        }

        host, port, privKey, uErr := parseUserHostPort(body)
        if uErr != nil {
                return cfg, uErr
        }
        cfg.AmneziaPrivateKey = privKey
        cfg.Host = host
        cfg.Port = port

        cfg.AmneziaPublicKey = query.Get("publickey")
        cfg.AmneziaPresharedKey = query.Get("presharedkey")
        cfg.AmneziaLocalAddr = query.Get("address")
        cfg.AmneziaDNS = query.Get("dns")
        cfg.WGAllowedIPs = query.Get("allowedips")

        // Junk-packet params from query.
        cfg.AmneziaJc = parseUint32Q(query, "jc")
        cfg.AmneziaJmin = parseUint32Q(query, "jmin")
        cfg.AmneziaJmax = parseUint32Q(query, "jmax")
        cfg.AmneziaS1 = parseUint32Q(query, "s1")
        cfg.AmneziaS2 = parseUint32Q(query, "s2")
        cfg.AmneziaH1 = parseUint32Q(query, "h1")
        cfg.AmneziaH2 = parseUint32Q(query, "h2")
        cfg.AmneziaH3 = parseUint32Q(query, "h3")
        cfg.AmneziaH4 = parseUint32Q(query, "h4")

        return cfg, nil
}


// ---------------------------------------------------------------------------
// WireGuard .conf file parser
// ---------------------------------------------------------------------------

// ParseWireGuardConf parses a standard WireGuard INI-style config file.
// It extracts all relevant fields from [Interface] and [Peer] sections.
func ParseWireGuardConf(text string) (engine.ServerConfig, error) {
        var cfg engine.ServerConfig
        cfg.ID = engine.GenerateID()
        cfg.Protocol = engine.ProtocolWireGuard

        lines := strings.Split(text, "\n")
        inInterface := false
        inPeer := false

        for _, line := range lines {
                line = strings.TrimSpace(line)
                if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
                        continue
                }

                // Section headers
                if line == "[Interface]" {
                        inInterface = true
                        inPeer = false
                        continue
                }
                if line == "[Peer]" {
                        inPeer = true
                        inInterface = false
                        continue
                }

                // Key = Value parsing
                eqIdx := strings.Index(line, "=")
                if eqIdx == -1 {
                        continue
                }
                key := strings.TrimSpace(line[:eqIdx])
                value := strings.TrimSpace(line[eqIdx+1:])

                if inInterface {
                        switch key {
                        case "PrivateKey":
                                cfg.WGPrivateKey = value
                        case "Address":
                                cfg.WGLocalAddress = value
                        case "DNS":
                                cfg.WGDNSServers = value
                        case "MTU":
                                if mtu, err := strconv.ParseUint(value, 10, 32); err == nil {
                                        // MTU stored but not used directly (wireguard-go handles it)
                                        _ = mtu
                                }
                        case "Jc", "Jmin", "Jmax", "S1", "S2", "H1", "H2", "H3", "H4", "I1", "I2", "I3":
                                // AmneziaWG parameters detected - switch protocol
                                cfg.Protocol = engine.ProtocolAmneziaWG
                                cfg.AmneziaPrivateKey = cfg.WGPrivateKey
                                cfg.WGPrivateKey = ""
                        }
                }

                if inPeer {
                        switch key {
                        case "PublicKey":
                                cfg.WGPublicKey = value
                        case "PresharedKey":
                                cfg.WGPresharedKey = value
                        case "AllowedIPs":
                                cfg.WGAllowedIPs = value
                        case "PersistentKeepalive":
                                // Stored but wireguard-go handles it in the config file
                                _ = value
                        case "Endpoint":
                                h, p, err := netSplitHostPort(value)
                                if err == nil {
                                        cfg.Host = h
                                        cfg.Port = p
                                }
                        }
                }
        }

        // If AmneziaWG was detected, copy WG fields to Amnezia fields
        if cfg.Protocol == engine.ProtocolAmneziaWG {
                if cfg.AmneziaPrivateKey == "" {
                        cfg.AmneziaPrivateKey = cfg.WGPrivateKey
                }
        }

        return cfg, nil
}

// ---------------------------------------------------------------------------
// sing-box JSON parser
// ---------------------------------------------------------------------------

// singBoxConfig represents the top-level sing-box configuration.
type singBoxConfig struct {
        Outbounds []json.RawMessage `json:"outbounds"`
}

// singBoxOutbound is a minimal outbound descriptor used for type routing.
type singBoxOutbound struct {
        Type string `json:"type"`
        Tag  string `json:"tag"`
}

// parseSingBox iterates over sing-box outbound entries and converts supported
// types (vless, vmess, hysteria, wireguard) into ServerConfig.
func parseSingBox(data []byte) ([]engine.ServerConfig, []ParseError) {
        var sb singBoxConfig
        if err := json.Unmarshal(data, &sb); err != nil {
                return nil, []ParseError{{Message: fmt.Sprintf("sing-box JSON parse: %v", err)}}
        }

        var servers []engine.ServerConfig
        var errs []ParseError

        for i, raw := range sb.Outbounds {
                var meta singBoxOutbound
                if err := json.Unmarshal(raw, &meta); err != nil {
                        errs = append(errs, ParseError{Line: i + 1, Message: fmt.Sprintf("outbound metadata: %v", err)})
                        continue
                }

                var cfg engine.ServerConfig
                var err error

                switch meta.Type {
                case "vless":
                        cfg, err = parseSingBoxVLESS(raw)
                case "vmess":
                        cfg, err = parseSingBoxVMESS(raw)
                case "hysteria2":
                        cfg, err = parseSingBoxHysteria2(raw)
                case "wireguard":
                        cfg, err = parseSingBoxWireGuard(raw)
                default:
                        // Unsupported outbound type – skip silently.
                        continue
                }

                if err != nil {
                        errs = append(errs, ParseError{Line: i + 1, Message: fmt.Sprintf("%s: %v", meta.Type, err)})
                        continue
                }

                if cfg.Name == "" {
                        cfg.Name = meta.Tag
                }
                if cfg.ID == "" {
                        cfg.ID = uuid.New().String()
                }

                servers = append(servers, cfg)
        }

        return servers, errs
}

// ---------- sing-box VLESS ----------

type sbVLESS struct {
        Server     string          `json:"server"`
        ServerPort uint32          `json:"server_port"`
        UUID       string          `json:"uuid"`
        Flow       string          `json:"flow"`
        TLS        *sbTLS          `json:"tls"`
        Transport  *sbTransport    `json:"transport"`
}

type sbTLS struct {
        Enabled  bool       `json:"enabled"`
        SNI      string     `json:"server_name"`
        ALPN     []string   `json:"alpn"`
        Fingerprint *sbUTLS `json:"utls"`
        Reality   *sbReality `json:"reality"`
        Insecure bool      `json:"insecure"`
}

type sbUTLS struct {
        Fingerprint string `json:"fingerprint"`
}

type sbReality struct {
        Enabled   bool   `json:"enabled"`
        PublicKey string `json:"public_key"`
        ShortID   string `json:"short_id"`
}

type sbTransport struct {
        Type      string            `json:"type"`
        Path      string            `json:"path"`
        Service   string            `json:"service"`
        Seed      string            `json:"seed"`
        Headers   map[string]string `json:"headers"`
}

func parseSingBoxVLESS(raw json.RawMessage) (engine.ServerConfig, error) {
        var v sbVLESS
        if err := json.Unmarshal(raw, &v); err != nil {
                return engine.ServerConfig{}, err
        }

        cfg := engine.ServerConfig{
                ID:       uuid.New().String(),
                Protocol: engine.ProtocolVLESS,
                Host:     v.Server,
                Port:     v.ServerPort,
                UUID:     v.UUID,
                Flow:     v.Flow,
        }

        if v.TLS != nil && v.TLS.Enabled {
                if v.TLS.Reality != nil && v.TLS.Reality.Enabled {
                        cfg.Security = "reality"
                        cfg.PublicKey = v.TLS.Reality.PublicKey
                        cfg.ShortID = v.TLS.Reality.ShortID
                } else {
                        cfg.Security = "tls"
                }
                cfg.SNI = v.TLS.SNI
                cfg.ALPN = strings.Join(v.TLS.ALPN, ",")
                if v.TLS.Fingerprint != nil {
                        cfg.Fingerprint = v.TLS.Fingerprint.Fingerprint
                }
                cfg.AllowInsecure = v.TLS.Insecure
        } else {
                cfg.Security = "none"
        }

        if v.Transport != nil {
                cfg.Transport = v.Transport.Type
                cfg.WSPath = v.Transport.Path
                cfg.GRPCService = v.Transport.Service
                cfg.KCPSeed = v.Transport.Seed
        }

        return cfg, nil
}

// ---------- sing-box VMESS ----------

type sbVMESS struct {
        Server     string       `json:"server"`
        ServerPort uint32       `json:"server_port"`
        UUID       string       `json:"uuid"`
        Security   string       `json:"security"`
        AlterID   uint32       `json:"alter_id"`
        TLS        *sbTLS       `json:"tls"`
        Transport  *sbTransport `json:"transport"`
}

func parseSingBoxVMESS(raw json.RawMessage) (engine.ServerConfig, error) {
        var v sbVMESS
        if err := json.Unmarshal(raw, &v); err != nil {
                return engine.ServerConfig{}, err
        }

        cfg := engine.ServerConfig{
                ID:       uuid.New().String(),
                Protocol: engine.ProtocolVMESS,
                Host:     v.Server,
                Port:     v.ServerPort,
                UUID:     v.UUID,
        }

        if v.TLS != nil && v.TLS.Enabled {
                cfg.Security = "tls"
                cfg.SNI = v.TLS.SNI
                cfg.ALPN = strings.Join(v.TLS.ALPN, ",")
                if v.TLS.Fingerprint != nil {
                        cfg.Fingerprint = v.TLS.Fingerprint.Fingerprint
                }
                cfg.AllowInsecure = v.TLS.Insecure
        } else {
                cfg.Security = "none"
        }

        if v.Transport != nil {
                cfg.Transport = v.Transport.Type
                cfg.WSPath = v.Transport.Path
                cfg.GRPCService = v.Transport.Service
                cfg.KCPSeed = v.Transport.Seed
        }

        return cfg, nil
}

// ---------- sing-box Hysteria2 ----------

type sbHysteria2 struct {
        Server     string     `json:"server"`
        ServerPort uint32     `json:"server_port"`
        Password   string     `json:"password"`
        TLS        *sbTLS     `json:"tls"`
        Obfs       *sbH2Obfs  `json:"obfs"`
        UpMbps     uint64     `json:"up_mbps"`
        DownMbps   uint64     `json:"down_mbps"`
}

type sbH2Obfs struct {
        Type     string `json:"type"`
        Password string `json:"password"`
}

func parseSingBoxHysteria2(raw json.RawMessage) (engine.ServerConfig, error) {
        var h sbHysteria2
        if err := json.Unmarshal(raw, &h); err != nil {
                return engine.ServerConfig{}, err
        }

        cfg := engine.ServerConfig{
                ID:             uuid.New().String(),
                Protocol:       engine.ProtocolHysteria,
                Host:           h.Server,
                Port:           h.ServerPort,
                HysteriaAuth:   h.Password,
                HysteriaBwUp:   h.UpMbps * 1_000_000,
                HysteriaBwDown: h.DownMbps * 1_000_000,
        }

        if h.TLS != nil {
                cfg.HysteriaSNI = h.TLS.SNI
                cfg.HysteriaALPN = strings.Join(h.TLS.ALPN, ",")
                cfg.HysteriaInsecure = h.TLS.Insecure
        }

        if h.Obfs != nil {
                cfg.HysteriaObfs = h.Obfs.Type
                if h.Obfs.Password != "" {
                        cfg.HysteriaObfs = h.Obfs.Password
                }
        }

        return cfg, nil
}

// ---------- sing-box WireGuard ----------

type sbWireGuard struct {
        Server       string `json:"server"`
        ServerPort   uint32 `json:"server_port"`
        LocalAddress []string `json:"local_address"`
        PrivateKey   string `json:"private_key"`
        PeerPublicKey  string `json:"peer_public_key"`
        PreSharedKey   string `json:"pre_shared_key"`
        Reserved     []byte `json:"reserved"`
        MTU          uint32 `json:"mtu"`
}

func parseSingBoxWireGuard(raw json.RawMessage) (engine.ServerConfig, error) {
        var w sbWireGuard
        if err := json.Unmarshal(raw, &w); err != nil {
                return engine.ServerConfig{}, err
        }

        cfg := engine.ServerConfig{
                ID:              uuid.New().String(),
                Protocol:        engine.ProtocolWireGuard,
                Host:            w.Server,
                Port:            w.ServerPort,
                WGPrivateKey:    w.PrivateKey,
                WGPublicKey:     w.PeerPublicKey,
                WGPresharedKey:  w.PreSharedKey,
                WGLocalAddress:  strings.Join(w.LocalAddress, ","),
        }

        return cfg, nil
}

// ---------------------------------------------------------------------------
// SIP008 parser
// ---------------------------------------------------------------------------

// sip008Config represents the SIP008 subscription JSON structure.
// SIP008 is a standardised JSON list format originally for Shadowsocks.
// We parse what we can and map to supported protocol types.
type sip008Config struct {
        Version string        `json:"version"`
        Servers []sip008Server `json:"servers"`
}

type sip008Server struct {
        Server     string `json:"server"`
        ServerPort int    `json:"server_port"`
        Password   string `json:"password"`
        Method     string `json:"method"`
        Plugin     string `json:"plugin"`
        PluginOpts string `json:"plugin_opts"`
        Remarks    string `json:"remarks"`
        // Additional fields some providers include.
        ID    string `json:"id"`
        ALPN  string `json:"alpn"`
        SNI   string `json:"sni"`
        TLS   string `json:"tls"`
        Network string `json:"network"`
}

// parseSIP008 decodes a SIP008 JSON payload.  Since SIP008 is Shadowsocks-
// centric and we do not have a native SS protocol, we produce best-effort
// ServerConfig entries using the VLESS protocol as a fallback container.
func parseSIP008(data []byte) ([]engine.ServerConfig, []ParseError) {
        var s8 sip008Config
        if err := json.Unmarshal(data, &s8); err != nil {
                return nil, []ParseError{{Message: fmt.Sprintf("SIP008 JSON parse: %v", err)}}
        }

        var servers []engine.ServerConfig
        var errs []ParseError

        for i, s := range s8.Servers {
                if s.Server == "" {
                        errs = append(errs, ParseError{Line: i + 1, Message: "missing server address"})
                        continue
                }

                cfg := engine.ServerConfig{
                        ID:       uuid.New().String(),
                        Name:     s.Remarks,
                        Host:     s.Server,
                        Port:     uint32(s.ServerPort),
                        Protocol: engine.ProtocolVLESS, // best-effort fallback
                        UUID:     s.Password,
                        SNI:      s.SNI,
                        ALPN:     s.ALPN,
                        Transport: s.Network,
                }
                if s.TLS == "tls" {
                        cfg.Security = "tls"
                }

                servers = append(servers, cfg)
        }

        return servers, errs
}

// ---------------------------------------------------------------------------
// Clash YAML parser
// ---------------------------------------------------------------------------

// clashConfig captures the top-level Clash proxy-provider config.
type clashConfig struct {
        Proxies []json.RawMessage `yaml:"proxies"`
}

// clashProxy mirrors the common fields across Clash proxy types.
type clashProxy struct {
        Name     string `yaml:"name"`
        Type     string `yaml:"type"`
        Server   string `yaml:"server"`
        Port     int    `yaml:"port"`
        UUID     string `yaml:"uuid"`
        Password string `yaml:"password"`
        AlterID  int    `yaml:"alterId"`
        Cipher   string `yaml:"cipher"`

        // TLS.
        TLS            bool   `yaml:"tls"`
        SkipCertVerify bool   `yaml:"skip-cert-verify"`
        ServerName     string `yaml:"servername"`
        Fingerprint    string `yaml:"client-fingerprint"`
        ALPN           string `yaml:"alpn"`
        RealityOpts    *clashReality `yaml:"reality-opts"`
        // ShortIds for reality.
        ShortID string `yaml:"short-id"`
        Publickey string `yaml:"public-key"`

        // Transport.
        Network   string `yaml:"network"`
        WSOpts    *clashWSOpts    `yaml:"ws-opts"`
        GRPCOpts  *clashGRPCOpts  `yaml:"grpc-opts"`
        XHTTPOpts *clashXHTTPOpts `yaml:"xhttp-opts"`

        // Hysteria.
        AuthStr  string `yaml:"auth-str"`
        UpMbps   int    `yaml:"up"`
        DownMbps int    `yaml:"down"`
        Obfs     string `yaml:"obfs"`

        // WireGuard.
        PrivateKey   string `yaml:"private-key"`
        PublicKey    string `yaml:"public-key"`
        PresharedKey string `yaml:"preshared-key"`
        IP          string `yaml:"ip"`
        DNS         string `yaml:"dns"`
        AllowedIPs  string `yaml:"allowed-ips"`

        // AmneziaWG extras.
        Jc   uint32 `yaml:"jc"`
        Jmin uint32 `yaml:"jmin"`
        Jmax uint32 `yaml:"jmax"`
        S1   uint32 `yaml:"s1"`
        S2   uint32 `yaml:"s2"`
        H1   uint32 `yaml:"h1"`
        H2   uint32 `yaml:"h2"`
        H3   uint32 `yaml:"h3"`
        H4   uint32 `yaml:"h4"`

        // VLESS flow.
        Flow string `yaml:"flow"`
}

type clashReality struct {
        PublicKey string `yaml:"public-key"`
        ShortID   string `yaml:"short-id"`
}

type clashWSOpts struct {
        Path   string            `yaml:"path"`
        Headers map[string]string `yaml:"headers"`
}

type clashGRPCOpts struct {
        ServiceName string `yaml:"grpc-service-name"`
}

type clashXHTTPOpts struct {
        Path string `yaml:"path"`
        Mode string `yaml:"mode"`
}

// parseClash parses a Clash-format YAML config and extracts supported proxies.
func parseClash(data []byte) ([]engine.ServerConfig, []ParseError) {
        var cc clashConfig
        if err := yaml.Unmarshal(data, &cc); err != nil {
                return nil, []ParseError{{Message: fmt.Sprintf("clash YAML parse: %v", err)}}
        }

        var servers []engine.ServerConfig
        var errs []ParseError

        for i, raw := range cc.Proxies {
                var p clashProxy
                if err := yaml.Unmarshal(raw, &p); err != nil {
                        errs = append(errs, ParseError{Line: i + 1, Message: fmt.Sprintf("proxy parse: %v", err)})
                        continue
                }

                cfg, err := clashProxyToConfig(&p)
                if err != nil {
                        errs = append(errs, ParseError{Line: i + 1, Message: err.Error()})
                        continue
                }

                servers = append(servers, cfg)
        }

        return servers, errs
}

// clashProxyToConfig maps a Clash proxy definition to a ServerConfig.
func clashProxyToConfig(p *clashProxy) (engine.ServerConfig, error) {
        cfg := engine.ServerConfig{
                ID:   uuid.New().String(),
                Name: p.Name,
                Host: p.Server,
                Port: uint32(p.Port),
        }

        switch p.Type {
        case "vless":
                cfg.Protocol = engine.ProtocolVLESS
                cfg.UUID = p.UUID
                cfg.Flow = p.Flow
                applyClashTLS(p, &cfg)
                applyClashTransport(p, &cfg)

        case "vmess":
                cfg.Protocol = engine.ProtocolVMESS
                cfg.UUID = p.UUID
                applyClashTLS(p, &cfg)
                applyClashTransport(p, &cfg)

        case "hysteria2":
                cfg.Protocol = engine.ProtocolHysteria
                cfg.HysteriaAuth = p.Password
                cfg.HysteriaSNI = p.ServerName
                cfg.HysteriaInsecure = p.SkipCertVerify
                cfg.HysteriaALPN = p.ALPN
                cfg.HysteriaObfs = p.Obfs
                if p.UpMbps > 0 {
                        cfg.HysteriaBwUp = uint64(p.UpMbps) * 1_000_000
                }
                if p.DownMbps > 0 {
                        cfg.HysteriaBwDown = uint64(p.DownMbps) * 1_000_000
                }

        case "wireguard":
                cfg.Protocol = engine.ProtocolWireGuard
                cfg.WGPrivateKey = p.PrivateKey
                cfg.WGPublicKey = p.PublicKey
                cfg.WGPresharedKey = p.PresharedKey
                cfg.WGLocalAddress = p.IP
                cfg.WGDNSServers = p.DNS
                cfg.WGAllowedIPs = p.AllowedIPs

        case "amnezia-wg", "amnezia-wireguard":
                cfg.Protocol = engine.ProtocolAmneziaWG
                cfg.AmneziaPrivateKey = p.PrivateKey
                cfg.AmneziaPublicKey = p.PublicKey
                cfg.AmneziaPresharedKey = p.PresharedKey
                cfg.AmneziaLocalAddr = p.IP
                cfg.AmneziaDNS = p.DNS
                cfg.WGAllowedIPs = p.AllowedIPs
                cfg.AmneziaJc = p.Jc
                cfg.AmneziaJmin = p.Jmin
                cfg.AmneziaJmax = p.Jmax
                cfg.AmneziaS1 = p.S1
                cfg.AmneziaS2 = p.S2
                cfg.AmneziaH1 = p.H1
                cfg.AmneziaH2 = p.H2
                cfg.AmneziaH3 = p.H3
                cfg.AmneziaH4 = p.H4

        default:
                return engine.ServerConfig{}, fmt.Errorf("unsupported clash proxy type: %s", p.Type)
        }

        return cfg, nil
}

// applyClashTLS fills TLS-related fields from a Clash proxy.
func applyClashTLS(p *clashProxy, cfg *engine.ServerConfig) {
        if p.TLS {
                cfg.Security = "tls"
                if p.RealityOpts != nil {
                        cfg.Security = "reality"
                        cfg.PublicKey = p.RealityOpts.PublicKey
                        cfg.ShortID = p.RealityOpts.ShortID
                } else if p.Publickey != "" {
                        cfg.Security = "reality"
                        cfg.PublicKey = p.Publickey
                        cfg.ShortID = p.ShortID
                }
        }
        cfg.SNI = p.ServerName
        cfg.Fingerprint = p.Fingerprint
        cfg.ALPN = p.ALPN
        cfg.AllowInsecure = p.SkipCertVerify
}

// applyClashTransport fills transport-related fields from a Clash proxy.
func applyClashTransport(p *clashProxy, cfg *engine.ServerConfig) {
        cfg.Transport = p.Network
        if p.WSOpts != nil {
                cfg.WSPath = p.WSOpts.Path
        }
        if p.GRPCOpts != nil {
                cfg.GRPCService = p.GRPCOpts.ServiceName
        }
        if p.XHTTPOpts != nil {
                cfg.XHTTPPath = p.XHTTPOpts.Path
                cfg.XHTTPMode = p.XHTTPOpts.Mode
                cfg.Transport = "xhttp"
        }
}

// ---------------------------------------------------------------------------
// Format detection helpers
// ---------------------------------------------------------------------------

// tryBase64Decode attempts to decode the input as standard or URL-safe base64.
// Returns an error if decoding fails after stripping whitespace.
func tryBase64Decode(data []byte) ([]byte, error) {
        s := strings.TrimSpace(string(data))
        // Add padding if missing.
        if m := len(s) % 4; m != 0 {
                s += strings.Repeat("=", 4-m)
        }

        // Try standard encoding first.
        decoded, err := base64.StdEncoding.DecodeString(s)
        if err == nil {
                return decoded, nil
        }

        // Try URL-safe encoding (common in subscriptions).
        decoded, err = base64.URLEncoding.DecodeString(strings.TrimRight(string(data), "=\n\r "))
        if err == nil {
                return decoded, nil
        }

        // Try raw (no padding) URL-safe.
        decoded, err = base64.RawURLEncoding.DecodeString(strings.TrimSpace(string(data)))
        if err == nil {
                return decoded, nil
        }

        // Try raw standard.
        decoded, err = base64.RawStdEncoding.DecodeString(strings.TrimSpace(string(data)))
        if err == nil {
                return decoded, nil
        }

        return nil, fmt.Errorf("not valid base64")
}

// base64DecodeAny tries all base64 variants on a short string (used inside
// individual link parsers for vmess://, wg://, awg:// bodies).
func base64DecodeAny(s string) ([]byte, error) {
        // Try with padding.
        if m := len(s) % 4; m != 0 {
                padded := s + strings.Repeat("=", 4-m)
                if b, err := base64.StdEncoding.DecodeString(padded); err == nil {
                        return b, nil
                }
                if b, err := base64.URLEncoding.DecodeString(padded); err == nil {
                        return b, nil
                }
        }
        // Raw variants.
        if b, err := base64.RawStdEncoding.DecodeString(s); err == nil {
                return b, nil
        }
        if b, err := base64.RawURLEncoding.DecodeString(s); err == nil {
                return b, nil
        }
        return nil, fmt.Errorf("base64 decode failed")
}

// isJSON returns true if the trimmed string starts with '{' or '['.
func isJSON(s string) bool {
        s = strings.TrimSpace(s)
        return len(s) > 0 && (s[0] == '{' || s[0] == '[')
}

// isSingBoxJSON checks for sing-box-specific JSON keys.
// A sing-box config must contain "outbounds".
func isSingBoxJSON(data []byte) bool {
        return strings.Contains(string(data), `"outbounds"`)
}

// isSIP008JSON checks for SIP008-specific JSON keys.
func isSIP008JSON(data []byte) bool {
        s := string(data)
        return strings.Contains(s, `"version"`) && strings.Contains(s, `"servers"`)
}

// isClashYAML checks for Clash-specific YAML keys.
func isClashYAML(data []byte) bool {
        s := strings.ToLower(string(data))
        return strings.Contains(s, "proxies:") || strings.Contains(s, `"proxies"`)
}

// isWireGuardConf checks if the content looks like a WireGuard .conf file.
func isWireGuardConf(data []byte) bool {
        s := strings.TrimSpace(string(data))
        return strings.HasPrefix(s, "[Interface]")
}

// hasShareLinkPrefix returns true if any line in the text starts with a
// recognised share-link scheme.
func hasShareLinkPrefix(s string) bool {
        for _, line := range strings.Split(s, "\n") {
                line = strings.TrimSpace(line)
                if line == "" {
                        continue
                }
                switch {
                case strings.HasPrefix(line, "vless://"),
                        strings.HasPrefix(line, "vmess://"),
                        strings.HasPrefix(line, "hysteria2://"),
                        strings.HasPrefix(line, "hy2://"),
                        strings.HasPrefix(line, "wg://"),
                        strings.HasPrefix(line, "awg://"):
                        return true
                }
        }
        return false
}

// ---------------------------------------------------------------------------
// URI helpers
// ---------------------------------------------------------------------------

// trimScheme strips the given scheme prefix from raw and returns the rest.
func trimScheme(raw, scheme string) (string, error) {
        if !strings.HasPrefix(raw, scheme) {
                return "", fmt.Errorf("expected %s prefix", scheme)
        }
        return raw[len(scheme):], nil
}

// splitFragment separates the URI fragment (after '#') from the body.
// The fragment value is URL-decoded to recover the display name.
func splitFragment(body string) (mainBody, fragment string) {
        if idx := strings.Index(body, "#"); idx >= 0 {
                return body[:idx], urlFragmentDecode(body[idx+1:])
        }
        return body, ""
}

// urlFragmentDecode URL-decodes the fragment portion, falling back to the raw
// string if decoding fails.
func urlFragmentDecode(s string) string {
        decoded, err := url.PathUnescape(s)
        if err != nil {
                return s
        }
        return decoded
}

// splitQuery separates the query string (after first '?') from the body.
func splitQuery(body string) (mainBody, queryString string) {
        if idx := strings.Index(body, "?"); idx >= 0 {
                return body[:idx], body[idx+1:]
        }
        return body, ""
}

// parseUserHostPort extracts the userinfo, host, and port from a URI authority
// of the form "user@host:port" or "host:port" (empty user).
func parseUserHostPort(auth string) (host string, port uint32, user string, err error) {
        var userPart, hostPort string

        if idx := strings.LastIndex(auth, "@"); idx >= 0 {
                userPart = auth[:idx]
                hostPort = auth[idx+1:]
        } else {
                hostPort = auth
        }

        h, p, pErr := netSplitHostPort(hostPort)
        if pErr != nil {
                return "", 0, "", fmt.Errorf("invalid host:port %q: %w", hostPort, pErr)
        }

        return h, p, userPart, nil
}

// netSplitHostPort is a lenient version of net.SplitHostPort that accepts
// bracketed IPv6 addresses and bare hostnames.
func netSplitHostPort(hostPort string) (host string, port uint32, err error) {
        // Try the standard library first.
        h, pStr, err := net.SplitHostPort(hostPort)
        if err == nil {
                p, pErr := strconv.ParseUint(pStr, 10, 32)
                if pErr != nil {
                        return "", 0, fmt.Errorf("invalid port %q: %w", pStr, pErr)
                }
                return h, uint32(p), nil
        }

        // Fallback: maybe there's no port – try to split on last colon that
        // isn't part of an IPv6 address.
        if strings.Contains(hostPort, "]") {
                // IPv6: [::1]:443
                closeBracket := strings.Index(hostPort, "]")
                if closeBracket < 0 {
                        return "", 0, fmt.Errorf("invalid host:port %q", hostPort)
                }
                h = hostPort[:closeBracket+1]
                rest := hostPort[closeBracket+1:]
                if strings.HasPrefix(rest, ":") {
                        p, pErr := strconv.ParseUint(rest[1:], 10, 32)
                        if pErr != nil {
                                return "", 0, fmt.Errorf("invalid port in %q", hostPort)
                        }
                        return h, uint32(p), nil
                }
                return h, 0, nil
        }

        // Last resort: split on last colon.
        lastColon := strings.LastIndex(hostPort, ":")
        if lastColon >= 0 {
                h = hostPort[:lastColon]
                pStr = hostPort[lastColon+1:]
                p, pErr := strconv.ParseUint(pStr, 10, 32)
                if pErr != nil {
                        return "", 0, fmt.Errorf("invalid port %q", pStr)
                }
                return h, uint32(p), nil
        }

        return hostPort, 0, nil
}

// ---------------------------------------------------------------------------
// Misc helpers
// ---------------------------------------------------------------------------

// parseUint32Q reads a uint32 from a url.Values query, returning 0 on failure.
func parseUint32Q(q url.Values, key string) uint32 {
        v := q.Get(key)
        if v == "" {
                return 0
        }
        n, err := strconv.ParseUint(v, 10, 32)
        if err != nil {
                return 0
        }
        return uint32(n)
}

// truncate shortens s to at most n runes for error messages.
func truncate(s string, n int) string {
        if len(s) <= n {
                return s
        }
        return s[:n] + "..."
}