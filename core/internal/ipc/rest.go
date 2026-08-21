package ipc

import (
        "context"
        "encoding/json"
        "fmt"
        "log"
        "net/http"
        "os"
        "strings"
        "time"

        "github.com/ChickenRamen500/tunnelcraft/core/internal/config"
        "github.com/ChickenRamen500/tunnelcraft/core/internal/engine"
        "github.com/ChickenRamen500/tunnelcraft/core/internal/subscription"
)

// RESTServer provides a simple HTTP/JSON API alongside the gRPC server.
// This allows the Tauri frontend to communicate without gRPC dependencies.
type RESTServer struct {
        cfg     *config.Manager
        mgr     *engine.Manager
        logger  *log.Logger
        httpSrv *http.Server
}

// NewRESTServer creates a new REST API server.
func NewRESTServer(cfg *config.Manager, mgr *engine.Manager) *RESTServer {
        return &RESTServer{
                cfg:    cfg,
                mgr:    mgr,
                logger: log.New(os.Stderr, "[rest] ", log.LstdFlags),
        }
}

// Start begins serving the REST API on the given port.
func (r *RESTServer) Start(port int) error {
        mux := http.NewServeMux()

        mux.HandleFunc("/api/health", r.handleHealth)
        mux.HandleFunc("/api/status", r.handleStatus)
        mux.HandleFunc("/api/connect", r.handleConnect)
        mux.HandleFunc("/api/disconnect", r.handleDisconnect)
        mux.HandleFunc("/api/servers", r.handleServers)
        mux.HandleFunc("/api/servers/import", r.handleImport)
        mux.HandleFunc("/api/subscriptions", r.handleSubscriptions)
        mux.HandleFunc("/api/subscriptions/refresh", r.handleRefreshSubscription)
        mux.HandleFunc("/api/settings", r.handleSettings)
        mux.HandleFunc("/api/routing", r.handleRouting)
        mux.HandleFunc("/api/logs", r.handleLogs)

        r.httpSrv = &http.Server{
                Addr:              fmt.Sprintf("127.0.0.1:%d", port),
                Handler:           corsMiddleware(mux),
                ReadHeaderTimeout: 5 * time.Second,
        }

        go func() {
                if err := r.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
                        r.logger.Printf("REST server error: %v", err)
                }
        }()

        r.logger.Printf("REST API listening on 127.0.0.1:%d", port)
        return nil
}

// Stop shuts down the REST server.
func (r *RESTServer) Stop() {
        if r.httpSrv != nil {
                r.httpSrv.Close()
        }
}

func corsMiddleware(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
                w.Header().Set("Content-Type", "application/json")
                w.Header().Set("Access-Control-Allow-Origin", "*")
                w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
                w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
                if req.Method == http.MethodOptions {
                        w.WriteHeader(http.StatusOK)
                        return
                }
                next.ServeHTTP(w, req)
        })
}

func (r *RESTServer) json(w http.ResponseWriter, status int, v interface{}) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(status)
        json.NewEncoder(w).Encode(v)
}

func (r *RESTServer) jsonErr(w http.ResponseWriter, status int, msg string) {
        r.json(w, status, map[string]string{"error": msg})
}

// --- Handlers ---

func (r *RESTServer) handleHealth(w http.ResponseWriter, req *http.Request) {
        r.json(w, http.StatusOK, map[string]interface{}{
                "healthy": true,
                "version": "0.1.0",
        })
}

func (r *RESTServer) handleStatus(w http.ResponseWriter, req *http.Request) {
        state := r.mgr.State()
        up, down, dur := r.mgr.Stats()
        active := r.mgr.ActiveServer()
        serverID := ""
        if active != nil {
                serverID = active.ID
        }
        cfg := r.cfg.Get()
        r.json(w, http.StatusOK, map[string]interface{}{
                "state":       state.String(),
                "server_id":   serverID,
                "mode":        "SYSTEM",
                "socks_port":  cfg.Tunnel.SOCKSPort,
                "http_port":   cfg.Tunnel.HTTPPort,
                "stats": map[string]interface{}{
                        "bytes_uploaded":   up,
                        "bytes_downloaded": down,
                        "duration":          int(dur.Seconds()),
                },
        })
}

func (r *RESTServer) handleConnect(w http.ResponseWriter, req *http.Request) {
        if req.Method != http.MethodPost {
                r.jsonErr(w, http.StatusMethodNotAllowed, "POST required")
                return
        }
        var body struct {
                ServerID string `json:"server_id"`
        }
        if err := json.NewDecoder(req.Body).Decode(&body); err != nil || body.ServerID == "" {
                r.jsonErr(w, http.StatusBadRequest, "server_id required")
                return
        }
        ctx := context.Background()
        if err := r.mgr.Connect(ctx, body.ServerID); err != nil {
                r.jsonErr(w, http.StatusInternalServerError, err.Error())
                return
        }
        r.json(w, http.StatusOK, map[string]string{"state": "CONNECTING", "server_id": body.ServerID})
}

func (r *RESTServer) handleDisconnect(w http.ResponseWriter, req *http.Request) {
        if req.Method != http.MethodPost {
                r.jsonErr(w, http.StatusMethodNotAllowed, "POST required")
                return
        }
        _ = r.mgr.Disconnect(false)
        r.json(w, http.StatusOK, map[string]string{"state": "DISCONNECTED"})
}

func (r *RESTServer) handleServers(w http.ResponseWriter, req *http.Request) {
        if req.Method == http.MethodGet {
                entries := r.cfg.GetServers()
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
                        })
                }
                if servers == nil {
                        servers = []map[string]interface{}{}
                }
                r.json(w, http.StatusOK, map[string]interface{}{
                        "servers": servers,
                        "total":   len(servers),
                })
                return
        }
        r.jsonErr(w, http.StatusMethodNotAllowed, "GET required")
}

func (r *RESTServer) handleImport(w http.ResponseWriter, req *http.Request) {
        if req.Method != http.MethodPost {
                r.jsonErr(w, http.StatusMethodNotAllowed, "POST required")
                return
        }
        var body struct {
                Content        string `json:"content"`
                SubscriptionID string `json:"subscription_id"`
                GroupName      string `json:"group_name"`
        }
        if err := json.NewDecoder(req.Body).Decode(&body); err != nil || body.Content == "" {
                r.jsonErr(w, http.StatusBadRequest, "content required")
                return
        }

        parsed, parseErrors := subscription.Parse([]byte(body.Content))
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
                if err := r.cfg.Update(func(c *config.Config) {
                        c.Servers = append(c.Servers, entries...)
                }); err != nil {
                        r.jsonErr(w, http.StatusInternalServerError, err.Error())
                        return
                }
        }

        r.json(w, http.StatusOK, map[string]interface{}{
                "imported_server_ids": importedIDs,
                "total_parsed":        len(parsed) + len(parseErrors),
                "total_imported":      len(importedIDs),
                "errors":               errStrings,
        })
}

func (r *RESTServer) handleSubscriptions(w http.ResponseWriter, req *http.Request) {
        if req.Method == http.MethodGet {
                entries := r.cfg.GetSubscriptions()
                var subs []map[string]interface{}
                for _, e := range entries {
                        subs = append(subs, map[string]interface{}{
                                "id":              e.ID,
                                "name":            e.Name,
                                "url":             e.URL,
                                "enabled":         e.Enabled,
                                "refresh_interval": e.RefreshInterval,
                                "server_count":    0,
                        })
                }
                if subs == nil {
                        subs = []map[string]interface{}{}
                }
                r.json(w, http.StatusOK, map[string]interface{}{
                        "subscriptions": subs,
                })
                return
        }

        if req.Method == http.MethodPost {
                var body struct {
                        Name string `json:"name"`
                        URL  string `json:"url"`
                }
                if err := json.NewDecoder(req.Body).Decode(&body); err != nil || body.URL == "" {
                        r.jsonErr(w, http.StatusBadRequest, "name and url required")
                        return
                }
                entry := config.SubscriptionEntry{
                        ID:      engine.GenerateID(),
                        Name:    body.Name,
                        URL:     body.URL,
                        Enabled: true,
                }
                if err := r.cfg.Update(func(c *config.Config) {
                        c.Subscriptions = append(c.Subscriptions, entry)
                }); err != nil {
                        r.jsonErr(w, http.StatusInternalServerError, err.Error())
                        return
                }
                r.json(w, http.StatusOK, map[string]interface{}{
                        "id":            entry.ID,
                        "name":          entry.Name,
                        "url":           entry.URL,
                        "enabled":       entry.Enabled,
                        "refresh_interval": entry.RefreshInterval,
                        "server_count":  0,
                })
                return
        }

        r.jsonErr(w, http.StatusMethodNotAllowed, "GET or POST required")
}

func (r *RESTServer) handleRefreshSubscription(w http.ResponseWriter, req *http.Request) {
        if req.Method != http.MethodPost {
                r.jsonErr(w, http.StatusMethodNotAllowed, "POST required")
                return
        }
        var body struct {
                ID string `json:"id"`
        }
        if err := json.NewDecoder(req.Body).Decode(&body); err != nil || body.ID == "" {
                r.jsonErr(w, http.StatusBadRequest, "id required")
                return
        }

        ctx := context.Background()
        prov := subscription.NewProvider(r.cfg)
        result := prov.Fetch(ctx, body.ID)

        if result.Error != "" {
                r.jsonErr(w, http.StatusBadGateway, result.Error)
                return
        }

        // Save imported servers
        var entries []config.ServerEntry
        for _, sc := range result.Servers {
                entry := serverConfigToEntry(sc)
                entry.SubscriptionID = body.ID
                entries = append(entries, entry)
        }
        if len(entries) > 0 {
                _ = r.cfg.Update(func(c *config.Config) {
                        // Remove old servers from this subscription
                        var filtered []config.ServerEntry
                        for _, s := range c.Servers {
                                if s.SubscriptionID != body.ID {
                                        filtered = append(filtered, s)
                                }
                        }
                        c.Servers = append(filtered, entries...)
                })
        }

        r.json(w, http.StatusOK, map[string]interface{}{
                "added":   len(entries),
                "updated": 0,
                "removed": 0,
        })
}

func (r *RESTServer) handleSettings(w http.ResponseWriter, req *http.Request) {
        cfg := r.cfg.Get()

        if req.Method == http.MethodGet {
                r.json(w, http.StatusOK, map[string]interface{}{
                        "proxy_mode":          "SYSTEM",
                        "socks_port":          cfg.Tunnel.SOCKSPort,
                        "http_port":           cfg.Tunnel.HTTPPort,
                        "dns_servers":         cfg.DNS.DNSServers,
                        "auto_connect":        cfg.Daemon.ConnectOnStartup,
                        "connect_on_startup":  cfg.Daemon.ConnectOnStartup,
                        "kill_switch":         cfg.Daemon.KillSwitch,
                        "split_tunneling":     false,
                        "allow_lan":           false,
                        "connection_timeout":  30,
                        "reconnect_attempts":  3,
                        "language":            "ru",
                        "theme":               "dark",
                })
                return
        }

        if req.Method == http.MethodPut {
                var settings map[string]interface{}
                if err := json.NewDecoder(req.Body).Decode(&settings); err != nil {
                        r.jsonErr(w, http.StatusBadRequest, err.Error())
                        return
                }
                if err := r.cfg.Update(func(c *config.Config) {
                        if v, ok := settings["socks_port"].(float64); ok {
                                c.Tunnel.SOCKSPort = uint32(v)
                        }
                        if v, ok := settings["http_port"].(float64); ok {
                                c.Tunnel.HTTPPort = uint32(v)
                        }
                        if v, ok := settings["dns_servers"].(string); ok {
                                c.DNS.DNSServers = v
                        }
                        if v, ok := settings["auto_connect"].(bool); ok {
                                c.Daemon.ConnectOnStartup = v
                        }
                        if v, ok := settings["kill_switch"].(bool); ok {
                                c.Daemon.KillSwitch = v
                        }
                        if v, ok := settings["language"].(string); ok {
                                c.Daemon.Language = v
                        }
                        if v, ok := settings["theme"].(string); ok {
                                c.Daemon.Theme = v
                        }
                }); err != nil {
                        r.jsonErr(w, http.StatusInternalServerError, err.Error())
                        return
                }
                r.json(w, http.StatusOK, map[string]string{"status": "ok"})
                return
        }

        r.jsonErr(w, http.StatusMethodNotAllowed, "GET or PUT required")
}

func (r *RESTServer) handleRouting(w http.ResponseWriter, req *http.Request) {
        routing := r.cfg.GetRouting()
        var rules []map[string]interface{}
        for _, rule := range routing.Rules {
                rules = append(rules, map[string]interface{}{
                        "id":      rule.ID,
                        "name":    rule.Name,
                        "enabled": rule.Enabled,
                        "action":  rule.Action,
                        "domains": rule.Domains,
                        "ip_cidrs": rule.IPCidrs,
                })
        }
        if rules == nil {
                rules = []map[string]interface{}{}
        }
        r.json(w, http.StatusOK, map[string]interface{}{
                "domain_strategy": routing.DomainStrategy,
                "rules":           rules,
        })
}

func (r *RESTServer) handleLogs(w http.ResponseWriter, req *http.Request) {
        // Return empty logs for now — the daemon logs go to stderr
        r.json(w, http.StatusOK, map[string]interface{}{
                "logs": []string{},
        })
}

// isConfFile checks if the content looks like a WireGuard/AmneziaWG .conf file.
func isConfFile(content string) bool {
        return strings.Contains(content, "[Interface]") && strings.Contains(content, "[Peer]")
}
