package dns

import (
    "context"
    "fmt"
    "log"
    "net"
    "sync"
    "time"
)

// DNSProxy runs a local DNS proxy that resolves queries via DoH
// or forwards them to an upstream DNS server.
type DNSProxy struct {
    mu         sync.Mutex
    listener   *net.UDPConn
    resolver   *DoHResolver
    upstream   string
    port       uint32
    running    bool
    cancel     context.CancelFunc
    cache      map[string]cacheEntry
    cacheTTL   time.Duration
}

type cacheEntry struct {
    records []DNSRecord
    expires time.Time
}

// NewDNSProxy creates a new DNS proxy.
func NewDNSProxy(resolver *DoHResolver, upstream string, port uint32) *DNSProxy {
    return &DNSProxy{
        resolver: resolver,
        upstream: upstream,
        port:     port,
        cache:    make(map[string]cacheEntry),
        cacheTTL:  5 * time.Minute,
    }
}

// Start begins listening for DNS queries.
func (p *DNSProxy) Start(ctx context.Context) error {
    p.mu.Lock()
    defer p.mu.Unlock()

    if p.running {
        return fmt.Errorf("DNS proxy already running")
    }

    addr := fmt.Sprintf("127.0.0.1:%d", p.port)
    udpAddr, err := net.ResolveUDPAddr("udp", addr)
    if err != nil {
        return fmt.Errorf("failed to resolve DNS listen address: %w", err)
    }

    conn, err := net.ListenUDP("udp", udpAddr)
    if err != nil {
        return fmt.Errorf("failed to start DNS proxy on %s: %w", addr, err)
    }

    p.listener = conn
    p.running = true
    ctx, p.cancel = context.WithCancel(ctx)

    go p.serve(ctx)
    go p.cleanCache(ctx)

    log.Printf("[dns] DNS proxy listening on %s (upstream: %s)", addr, p.upstream)
    return nil
}

// Stop shuts down the DNS proxy.
func (p *DNSProxy) Stop() error {
    p.mu.Lock()
    defer p.mu.Unlock()

    if !p.running {
        return nil
    }

    if p.cancel != nil {
        p.cancel()
        p.cancel = nil
    }

    if p.listener != nil {
        p.listener.Close()
        p.listener = nil
    }

    p.running = false
    log.Println("[dns] DNS proxy stopped")
    return nil
}

// IsRunning returns whether the proxy is active.
func (p *DNSProxy) IsRunning() bool {
    p.mu.Lock()
    defer p.mu.Unlock()
    return p.running
}

// serve handles incoming DNS queries.
func (p *DNSProxy) serve(ctx context.Context) {
    buf := make([]byte, 512)

    for {
        select {
        case <-ctx.Done():
            return
        default:
        }

        p.listener.SetReadDeadline(time.Now().Add(1 * time.Second))
        n, clientAddr, err := p.listener.ReadFromUDP(buf)
        if err != nil {
            if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
                continue
            }
            if ctx.Err() != nil {
                return
            }
            continue
        }

        // Parse query domain (minimal parsing)
        domain := extractDomain(buf[:n])
        if domain == "" {
            continue
        }

        // Check cache
        if cached := p.getFromCache(domain, QueryTypeA); cached != nil {
            p.listener.WriteToUDP(buildDNSResponse(buf[:n], cached), clientAddr)
            continue
        }

        // Resolve via DoH
        records, err := p.resolver.Resolve(ctx, domain, QueryTypeA)
        if err != nil {
            log.Printf("[dns] failed to resolve %s: %v", domain, err)
            continue
        }

        // Cache the result
        p.putInCache(domain, records)

        // Send response
        response := buildDNSResponse(buf[:n], records)
        p.listener.WriteToUDP(response, clientAddr)
    }
}

// cleanCache periodically removes expired cache entries.
func (p *DNSProxy) cleanCache(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            p.mu.Lock()
            now := time.Now()
            for key, entry := range p.cache {
                if now.After(entry.expires) {
                    delete(p.cache, key)
                }
            }
            p.mu.Unlock()
        }
    }
}

// getFromCache returns cached records if available.
func (p *DNSProxy) getFromCache(domain string, qtype uint16) []DNSRecord {
    p.mu.Lock()
    defer p.mu.Unlock()

    key := fmt.Sprintf("%s:%d", domain, qtype)
    entry, ok := p.cache[key]
    if !ok || time.Now().After(entry.expires) {
        return nil
    }
    return entry.records
}

// putInCache stores records in the cache.
func (p *DNSProxy) putInCache(domain string, records []DNSRecord) {
    if len(records) == 0 {
        return
    }

    p.mu.Lock()
    defer p.mu.Unlock()

    // Use the minimum TTL across all records
    var minTTL time.Duration
    for _, r := range records {
        ttl := time.Duration(r.TTL) * time.Second
        if minTTL == 0 || ttl < minTTL {
            minTTL = ttl
        }
        key := fmt.Sprintf("%s:%d", domain, r.Type)
        p.cache[key] = cacheEntry{
            records: []DNSRecord{r},
            expires: time.Now().Add(minTTL),
        }
    }
}

// Port returns the port the proxy is listening on.
func (p *DNSProxy) Port() uint32 {
    return p.port
}

// --- Minimal DNS parsing helpers ---

// extractDomain extracts the queried domain from a DNS query.
func extractDomain(query []byte) string {
    if len(query) < 12 {
        return ""
    }
    offset := 12 // skip header
    return decodeDNSName(query, offset)
}

// buildDNSResponse builds a simple DNS response from a query.
func buildDNSResponse(query []byte, records []DNSRecord) []byte {
    // Copy header
    resp := make([]byte, len(query), len(query)+256)
    copy(resp, query)

    // Set response flags
    resp[2] = 0x81 // QR=1 (response), RD=1
    resp[3] = 0x80 // RA=1

    // Set answer count
    anCount := uint16(len(records))
    resp[6] = byte(anCount >> 8)
    resp[7] = byte(anCount)

    // Append answer records
    for _, r := range records {
        // Name pointer to question name
        resp = append(resp, 0xC0, 0x0C)
        // Type
        resp = append(resp, byte(r.Type>>8), byte(r.Type))
        // Class: IN
        resp = append(resp, 0x00, 0x01)
        // TTL
        resp = append(resp,
            byte(r.TTL>>24), byte(r.TTL>>16),
            byte(r.TTL>>8), byte(r.TTL))

        switch r.Type {
        case 1: // A record
            ip := net.ParseIP(r.Value).To4()
            if ip != nil {
                resp = append(resp, 0x00, 0x04)
                resp = append(resp, ip...)
            } else {
                resp = append(resp, 0x00, 0x00)
            }
        case 28: // AAAA record
            ip := net.ParseIP(r.Value).To16()
            if ip != nil {
                resp = append(resp, 0x00, 0x10)
                resp = append(resp, ip...)
            } else {
                resp = append(resp, 0x00, 0x00)
            }
        default:
            resp = append(resp, 0x00, 0x00)
        }
    }

    return resp
}
