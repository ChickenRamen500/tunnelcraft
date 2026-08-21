package ipc

import (
        "context"
        "encoding/json"
        "fmt"
        "log"
        "net/http"
        "os"
        "os/signal"
        "strconv"
        "strings"
        "syscall"
        "time"

        "github.com/ChickenRamen500/tunnelcraft/core/internal/config"
        "github.com/ChickenRamen500/tunnelcraft/core/internal/engine"
        "github.com/ChickenRamen500/tunnelcraft/core/internal/subscription"
)

// HTTPServer provides a simple HTTP/JSON REST API alongside the gRPC server.
// This allows the Tauri frontend to communicate without gRPC dependencies.
type HTTPServer struct {
        cfg     *config.Manager
        mgr     *engine.Manager
        logger  *log.Logger
        httpSrv *http.Server
}

// NewHTTPServer creates a new HTTP REST API server.
func NewHTTPServer(cfg *config.Manager, mgr *engine.Manager) *HTTPServer {
        return &HTTPServer{
                cfg:    cfg,
                mgr:    mgr,
                logger: log.New(os.Stderr, "[http] ", log.LstdFlags),
        }
}

// Start begins serving the REST API on the configured address.
// The address defaults to "127.0.0.1:50052" but can be overridden by
// the DaemonConfig.GRPCAddr host combined with port 50052, or by passing
// an explicit addr string.
func (h *HTTPServer) Start(addr string) error {
        if addr == "" {
                cfg := h.cfg.Get()
                // Derive from gRPC address host, default to localhost:50052
                host := "127.0.0.1"
                if cfg.Daemon.GRPCAddr != "" {
                        if idx := strings.LastIndex(cfg.Daemon.GRPCAddr, ":"); idx >= 0 {
                                host = cfg.Daemon.GRPCAddr[:idx]
                        }
                }
                addr = host + ":50052"
        }

        mux := http.NewServeMux()

        mux.HandleFunc("/api/health", h.handleHealth)
        mux.HandleFunc("/api/status", h.handleStatus)
        mux.HandleFunc("/api/connect", h.handleConnect)
        mux.HandleFunc("/api/disconnect", h.handleDisconnect)
        mux.HandleFunc("/api/servers", h.handleServers)
        mux.HandleFunc("/api/servers/import", h.handleImport)
        mux.HandleFunc("/api/subscriptions", h.handleSubscriptions)
        mux.HandleFunc("/api/subscriptions/refresh/", h.handleRefreshSubscription)
        mux.HandleFunc("/api/settings", h.handleSettings)
        mux.HandleFunc("/api/logs", h.handleLogs)

        h.httpSrv = &http.Server{
                Addr:              addr,
                Handler:           httpCORSMiddleware(mux),
                ReadHeaderTimeout: 5 * time.Second,
        }

        go func() {
                if err := h.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
                        h.logger.Printf("HTTP server error: %v", err)
                }
        }()

        h.logger.Printf("HTTP REST API listening on %s", addr)
        return nil
}

// Stop gracefully shuts down the HTTP server.
func (h *HTTPServer) Stop() {
        if h.httpSrv != nil {
                h.logger.Println("shutting down HTTP server...")
                ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
                defer cancel()
                h.httpSrv.Shutdown(ctx)
        }
}

// Wait blocks until a shutdown signal is received.
func (h *HTTPServer) Wait() {
        sigCh := make(chan os.Signal, 1)
        signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
        sig := <-sigCh
        h.logger.Printf("received signal: %v, shutting down...", sig)
        h.Stop()
}

// --- Middleware ---

func httpCORSMiddleware(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
                w.Header().Set("Access-Control-Allow-Origin", "http://localhost:*")
                w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
                w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
                w.Header().Set("Access-Control-Max-Age", "86400")
                if req.Method == http.MethodOptions {
                        w.WriteHeader(http.StatusOK)
                        return
                }
                next.ServeHTTP(w, req)
        })
}

// --- JSON helpers ---

func (h *HTTPServer) json(w http.ResponseWriter, status int, v interface{}) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(status)
        _ = json.NewEncoder(w).Encode(v)
}

func (h *HTTPServer) jsonErr(w http.ResponseWriter, status int, msg string) {
        h.json(w, status, map[string]string{"error": msg})
}

// --- Handlers ---

// GET /api/health
func (h *HTTPServer) handleHealth(w http.ResponseWriter, req *http.Request) {
        if req.Method != http.MethodGet {
                h.jsonErr(w, http.StatusMethodNotAllowed, "GET required")
                return
        }
        h.json(w, http.StatusOK, map[string]interface{}{
                "healthy": true,
                "version": "0.1.0",
        })
}

// GET /api/status
func (h *HTTPServer) handleStatus(w http.ResponseWriter, req *http.Request) {
        if req.Method != http.MethodGet {
                h.jsonErr(w, http.StatusMethodNotAllowed, "GET required")
                return
        }

        state := h.mgr.State()
        bytesUp, bytesDown, duration := h.mgr.Stats()
        active := h.mgr.ActiveServer()

        resp := map[string]interface{}{
                "state":            state.String(),
                "server_id":         "",
                "server_name":       "",
                "protocol":          "",
                "socks_port":        0,
                "http_port":         0,
                "proxy_mode":        "",
                "stats": map[string]interface{}{
                        "bytes_uploaded":   bytesUp,
                        "bytes_downloaded": bytesDown,
                        "duration_seconds":  int(duration.Seconds()),
                },
        }

        if active != nil {
                resp["server_id"] = active.ID
                resp["server_name"] = active.Name
                resp["protocol"] = string(active.Protocol)
        }

        cfg := h.cfg.Get()
        resp["socks_port"] = cfg.Tunnel.SOCKSPort
        resp["http_port"] = cfg.Tunnel.HTTPPort
        resp["proxy_mode"] = cfg.Tunnel.ProxyMode

        h.json(w, http.StatusOK, resp)
}

// POST /api/connect
func (h *HTTPServer) handleConnect(w http.ResponseWriter, req *http.Request) {
        if req.Method != http.MethodPost {
                h.jsonErr(w, http.StatusMethodNotAllowed, "POST required")
                return
        }

        var body struct {
                ServerID string `json:"server_id"`
        }
        if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
                h.jsonErr(w, http.StatusBadRequest, "invalid JSON body")
                return
        }
        if body.ServerID == "" {
                h.jsonErr(w, http.StatusBadRequest, "server_id is required")
                return
        }

        ctx := context.Background()
        if err := h.mgr.Connect(ctx, body.ServerID); err != nil {
                h.jsonErr(w, http.StatusInternalServerError, err.Error())
                return
        }

        cfg := h.cfg.Get()
        h.json(w, http.StatusOK, map[string]interface{}{
                "state":     "CONNECTING",
                "server_id": body.ServerID,
                "socks_port": cfg.Tunnel.SOCKSPort,
                "http_port":  cfg.Tunnel.HTTPPort,
        })
}

// POST /api/disconnect
func (h *HTTPServer) handleDisconnect(w http.ResponseWriter, req *http.Request) {
        if req.Method != http.MethodPost {
                h.jsonErr(w, http.StatusMethodNotAllowed, "POST required")
                return
        }

        var body struct {
                Force bool `json:"force"`
        }
        // Body is optional; default force=false
        _ = json.NewDecoder(req.Body).Decode(&body)

        if err := h.mgr.Disconnect(body.Force); err != nil {
                h.jsonErr(w, http.StatusInternalServerError, err.Error())
                return
        }

        h.json(w, http.StatusOK, map[string]string{"state": "DISCONNECTED"})
}

// GET /api/servers
func (h *HTTPServer) handleServers(w http.ResponseWriter, req *http.Request) {
        if req.Method != http.MethodGet {
                h.jsonErr(w, http.StatusMethodNotAllowed, "GET required")
                return
        }

        entries := h.cfg.GetServers()
        var servers []map[string]interface{}
        for _, e := range entries {
                servers = append(servers, map[string]interface{}{
                        "id":              e.ID,
                        "name":            e.Name,
                        "host":            e.Host,
                        "port":            e.Port,
                        "protocol":        e.Protocol,
                        "favorite":        e.Favorite,
                        "tags":            e.Tags,
                        "subscription_id": e.SubscriptionID,
                        "sort_order":      e.SortOrder,
                })
        }
        if servers == nil {
                servers = []map[string]interface{}{}
        }

        h.json(w, http.StatusOK, map[string]interface{}{
                "servers": servers,
                "total":   len(servers),
        })
}

// POST /api/servers/import
func (h *HTTPServer) handleImport(w http.ResponseWriter, req *http.Request) {
        if req.Method != http.MethodPost {
                h.jsonErr(w, http.StatusMethodNotAllowed, "POST required")
                return
        }

        var body struct {
                Content        string `json:"content"`
                SubscriptionID string `json:"subscription_id"`
                GroupName      string `json:"group_name"`
        }
        if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
                h.jsonErr(w, http.StatusBadRequest, "invalid JSON body")
                return
        }
        if body.Content == "" {
                h.jsonErr(w, http.StatusBadRequest, "content is required")
                return
        }

        // Detect whether the content is a WireGuard .conf file or a
        // subscription URI/list and parse accordingly.
        var parsed []engine.ServerConfig
        var parseErrors []subscription.ParseError

        if isConfFile(body.Content) {
                sc, err := subscription.ParseWireGuardConf(body.Content)
                if err != nil {
                        h.jsonErr(w, http.StatusBadRequest, fmt.Sprintf("failed to parse WireGuard config: %v", err))
                        return
                }
                parsed = append(parsed, sc)
        } else if strings.HasPrefix(strings.TrimSpace(body.Content), "amnezia://") {
                // amnezia:// URI — use the generic parser
                parsed, parseErrors = subscription.Parse([]byte(body.Content))
        } else {
                // General subscription content (base64 share links, JSON, YAML)
                parsed, parseErrors = subscription.Parse([]byte(body.Content))
        }

        var errStrings []string
        for _, pe := range parseErrors {
                errStrings = append(errStrings, pe.Error())
        }

        var importedIDs []string
        var entries []config.ServerEntry
        for _, sc := range parsed {
                entry := serverConfigToEntry(sc)
                if entry.ID == "" {
                        entry.ID = engine.GenerateID()
                }
                if body.SubscriptionID != "" {
                        entry.SubscriptionID = body.SubscriptionID
                }
                if body.GroupName != "" {
                        entry.Tags = append(entry.Tags, body.GroupName)
                }
                entries = append(entries, entry)
                importedIDs = append(importedIDs, entry.ID)
        }

        if len(entries) > 0 {
                if err := h.cfg.Update(func(c *config.Config) {
                        c.Servers = append(c.Servers, entries...)
                }); err != nil {
                        h.jsonErr(w, http.StatusInternalServerError, err.Error())
                        return
                }
        }

        h.json(w, http.StatusOK, map[string]interface{}{
                "imported_server_ids": importedIDs,
                "total_parsed":        len(parsed) + len(parseErrors),
                "total_imported":      len(importedIDs),
                "errors":               errStrings,
        })
}

// GET /api/subscriptions — list all subscriptions
// POST /api/subscriptions — add a new subscription
func (h *HTTPServer) handleSubscriptions(w http.ResponseWriter, req *http.Request) {
        switch req.Method {
        case http.MethodGet:
                entries := h.cfg.GetSubscriptions()
                var subs []map[string]interface{}
                for _, e := range entries {
                        // Count servers belonging to this subscription
                        servers := h.cfg.GetServers()
                        count := 0
                        for _, s := range servers {
                                if s.SubscriptionID == e.ID {
                                        count++
                                }
                        }
                        subs = append(subs, map[string]interface{}{
                                "id":               e.ID,
                                "name":             e.Name,
                                "url":              e.URL,
                                "enabled":          e.Enabled,
                                "refresh_interval": e.RefreshInterval,
                                "server_count":     count,
                        })
                }
                if subs == nil {
                        subs = []map[string]interface{}{}
                }
                h.json(w, http.StatusOK, map[string]interface{}{
                        "subscriptions": subs,
                })

        case http.MethodPost:
                var body struct {
                        Name string `json:"name"`
                        URL  string `json:"url"`
                }
                if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
                        h.jsonErr(w, http.StatusBadRequest, "invalid JSON body")
                        return
                }
                if body.URL == "" {
                        h.jsonErr(w, http.StatusBadRequest, "url is required")
                        return
                }

                entry := config.SubscriptionEntry{
                        ID:      engine.GenerateID(),
                        Name:    body.Name,
                        URL:     body.URL,
                        Enabled: true,
                }
                if err := h.cfg.Update(func(c *config.Config) {
                        c.Subscriptions = append(c.Subscriptions, entry)
                }); err != nil {
                        h.jsonErr(w, http.StatusInternalServerError, err.Error())
                        return
                }

                h.json(w, http.StatusOK, map[string]interface{}{
                        "id":               entry.ID,
                        "name":             entry.Name,
                        "url":              entry.URL,
                        "enabled":          entry.Enabled,
                        "refresh_interval": entry.RefreshInterval,
                        "server_count":     0,
                })

        default:
                h.jsonErr(w, http.StatusMethodNotAllowed, "GET or POST required")
        }
}

// POST /api/subscriptions/refresh/{id}
func (h *HTTPServer) handleRefreshSubscription(w http.ResponseWriter, req *http.Request) {
        if req.Method != http.MethodPost {
                h.jsonErr(w, http.StatusMethodNotAllowed, "POST required")
                return
        }

        // Extract subscription ID from path: /api/subscriptions/refresh/{id}
        path := strings.TrimPrefix(req.URL.Path, "/api/subscriptions/refresh/")
        subID := strings.TrimRight(path, "/")
        if subID == "" {
                h.jsonErr(w, http.StatusBadRequest, "subscription id is required in path")
                return
        }

        ctx := context.Background()
        prov := subscription.NewProvider(h.cfg)
        result := prov.Fetch(ctx, subID)

        if result.Error != "" {
                h.jsonErr(w, http.StatusBadGateway, result.Error)
                return
        }

        // Replace old servers from this subscription with new ones
        var entries []config.ServerEntry
        for _, sc := range result.Servers {
                entry := serverConfigToEntry(sc)
                entry.SubscriptionID = subID
                entries = append(entries, entry)
        }

        removed := 0
        if err := h.cfg.Update(func(c *config.Config) {
                var filtered []config.ServerEntry
                for _, s := range c.Servers {
                        if s.SubscriptionID == subID {
                                removed++
                        } else {
                                filtered = append(filtered, s)
                        }
                }
                c.Servers = append(filtered, entries...)
        }); err != nil {
                h.jsonErr(w, http.StatusInternalServerError, err.Error())
                return
        }

        h.json(w, http.StatusOK, map[string]interface{}{
                "added":   len(entries),
                "updated": 0,
                "removed": removed,
        })
}

// GET /api/settings — get current settings
// PUT /api/settings — update settings
func (h *HTTPServer) handleSettings(w http.ResponseWriter, req *http.Request) {
        switch req.Method {
        case http.MethodGet:
                cfg := h.cfg.Get()
                h.json(w, http.StatusOK, map[string]interface{}{
                        "proxy_mode":         cfg.Tunnel.ProxyMode,
                        "socks_port":         cfg.Tunnel.SOCKSPort,
                        "http_port":          cfg.Tunnel.HTTPPort,
                        "dns_servers":        cfg.DNS.DNSServers,
                        "doh_enabled":        cfg.DNS.DoHEnabled,
                        "doh_url":            cfg.DNS.DoHURL,
                        "auto_connect":       cfg.Daemon.AutoConnect,
                        "connect_on_startup": cfg.Daemon.ConnectOnStartup,
                        "kill_switch":        cfg.Daemon.KillSwitch,
                        "allow_lan":          cfg.Daemon.AllowLAN,
                        "connection_timeout": cfg.Tunnel.ConnectionTimeout,
                        "reconnect_attempts": cfg.Tunnel.ReconnectAttempts,
                        "reconnect_delay":    cfg.Tunnel.ReconnectDelay,
                        "language":           cfg.Daemon.Language,
                        "theme":              cfg.Daemon.Theme,
                        "mtu":                cfg.Tunnel.MTU,
                        "split_tunneling":    len(cfg.Routing.Rules) > 0,
                })

        case http.MethodPut:
                var settings map[string]interface{}
                if err := json.NewDecoder(req.Body).Decode(&settings); err != nil {
                        h.jsonErr(w, http.StatusBadRequest, "invalid JSON body")
                        return
                }

                if err := h.cfg.Update(func(c *config.Config) {
                        if v, ok := settings["proxy_mode"].(string); ok {
                                c.Tunnel.ProxyMode = v
                        }
                        if v, ok := settings["socks_port"].(float64); ok {
                                c.Tunnel.SOCKSPort = uint32(v)
                        }
                        if v, ok := settings["http_port"].(float64); ok {
                                c.Tunnel.HTTPPort = uint32(v)
                        }
                        if v, ok := settings["dns_servers"].(string); ok {
                                c.DNS.DNSServers = v
                        }
                        if v, ok := settings["doh_enabled"].(bool); ok {
                                c.DNS.DoHEnabled = v
                        }
                        if v, ok := settings["doh_url"].(string); ok {
                                c.DNS.DoHURL = v
                        }
                        if v, ok := settings["auto_connect"].(bool); ok {
                                c.Daemon.AutoConnect = v
                        }
                        if v, ok := settings["connect_on_startup"].(bool); ok {
                                c.Daemon.ConnectOnStartup = v
                        }
                        if v, ok := settings["kill_switch"].(bool); ok {
                                c.Daemon.KillSwitch = v
                        }
                        if v, ok := settings["allow_lan"].(bool); ok {
                                c.Daemon.AllowLAN = v
                        }
                        if v, ok := settings["connection_timeout"].(float64); ok {
                                c.Tunnel.ConnectionTimeout = uint32(v)
                        }
                        if v, ok := settings["reconnect_attempts"].(float64); ok {
                                c.Tunnel.ReconnectAttempts = uint32(v)
                        }
                        if v, ok := settings["reconnect_delay"].(float64); ok {
                                c.Tunnel.ReconnectDelay = uint32(v)
                        }
                        if v, ok := settings["language"].(string); ok {
                                c.Daemon.Language = v
                        }
                        if v, ok := settings["theme"].(string); ok {
                                c.Daemon.Theme = v
                        }
                        if v, ok := settings["mtu"].(float64); ok {
                                c.Tunnel.MTU = uint32(v)
                        }
                }); err != nil {
                        h.jsonErr(w, http.StatusInternalServerError, err.Error())
                        return
                }

                h.json(w, http.StatusOK, map[string]string{"status": "ok"})

        default:
                h.jsonErr(w, http.StatusMethodNotAllowed, "GET or PUT required")
        }
}

// GET /api/logs — return recent daemon logs
func (h *HTTPServer) handleLogs(w http.ResponseWriter, req *http.Request) {
        if req.Method != http.MethodGet {
                h.jsonErr(w, http.StatusMethodNotAllowed, "GET required")
                return
        }

        // Parse optional query parameters for log control
        limitStr := req.URL.Query().Get("limit")
        limit := 100
        if limitStr != "" {
                if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 10000 {
                        limit = n
                }
        }

        level := req.URL.Query().Get("level")

        // Attempt to read log file if configured
        cfg := h.cfg.Get()
        logFile := cfg.Log.File

        var logLines []string
        if logFile != "" {
                data, err := os.ReadFile(logFile)
                if err == nil {
                        lines := strings.Split(string(data), "\n")
                        // Return the last N lines
                        start := 0
                        if len(lines) > limit {
                                start = len(lines) - limit
                        }
                        for i := start; i < len(lines); i++ {
                                line := strings.TrimSpace(lines[i])
                                if line == "" {
                                        continue
                        }
                                // Filter by level if specified
                                if level != "" {
                                        lower := strings.ToLower(line)
                                        if !strings.Contains(lower, "["+strings.ToLower(level)+"]") {
                                                continue
                                        }
                                }
                                logLines = append(logLines, line)
                        }
                }
        }

        if logLines == nil {
                logLines = []string{}
        }

        h.json(w, http.StatusOK, map[string]interface{}{
                "logs":  logLines,
                "count": len(logLines),
        })
}
