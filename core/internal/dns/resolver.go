package dns

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ChickenRamen500/tunnelcraft/core/internal/config"
	"github.com/miekg/dns"
)

// Resolver implements DNS-over-HTTPS, DNS-over-TLS, and plain DNS resolution.
type Resolver struct {
	Chain      [][]config.DnsProviderEntry
	Deadline   time.Duration
	PerAttempt time.Duration
}

// NewResolver creates a new DNS resolver from chain settings.
func NewResolver(chain *config.DnsChainSettingsEntry) *Resolver {
	if chain == nil || !chain.Enabled {
		return nil
	}

	r := &Resolver{
		Deadline:   time.Duration(chain.DeadlineMs) * time.Millisecond,
		PerAttempt: time.Duration(chain.PerAttemptMs) * time.Millisecond,
	}

	// Build chain: layer 0 = DoH, layer 1 = DoT, layer 2 = Plain
	var layers [][]config.DnsProviderEntry
	if len(chain.Doh) > 0 {
		layers = append(layers, chain.Doh)
	}
	if len(chain.Dot) > 0 {
		layers = append(layers, chain.Dot)
	}
	if len(chain.Plain) > 0 {
		layers = append(layers, chain.Plain)
	}
	r.Chain = layers

	return r
}

// Resolve resolves a domain name using the configured DNS chain.
// Returns the first successful IP address.
func (r *Resolver) Resolve(ctx context.Context, domain string) (string, error) {
	if len(r.Chain) == 0 {
		return "", fmt.Errorf("no DNS providers configured")
	}

	ctx, cancel := context.WithTimeout(ctx, r.Deadline)
	defer cancel()

	for _, layer := range r.Chain {
		for _, provider := range layer {
			resultCh := make(chan string, 1)
			errCh := make(chan error, 1)

			go func(p config.DnsProviderEntry) {
				ip, err := r.resolveOne(ctx, p.Kind, p.Addr, domain)
				if err != nil {
					errCh <- err
				} else {
					resultCh <- ip
				}
			}(provider)

			select {
			case ip := <-resultCh:
				return ip, nil
			case <-errCh:
				// Try next provider
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(r.PerAttempt):
				// Timeout for this attempt, try next
			}
		}
	}

	return "", fmt.Errorf("all DNS resolvers failed for %s", domain)
}

// resolveOne performs a single DNS query using the specified method.
func (r *Resolver) resolveOne(ctx context.Context, kind, addr, domain string) (string, error) {
	switch kind {
	case "doh":
		return r.resolveDoH(ctx, addr, domain)
	case "dot":
		return r.resolveDoT(ctx, addr, domain)
	case "plain":
		return r.resolvePlain(ctx, addr, domain)
	default:
		return "", fmt.Errorf("unknown DNS kind: %s", kind)
	}
}

// resolveDoH performs DNS-over-HTTPS query.
func (r *Resolver) resolveDoH(ctx context.Context, dohURL, domain string) (string, error) {
	// Build DNS message
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	msg.RecursionDesired = true

	packed, err := msg.Pack()
	if err != nil {
		return "", err
	}

	// Encode as base64url
	encoded := base64.RawURLEncoding.EncodeToString(packed)
	queryURL := fmt.Sprintf("%s?dns=%s", dohURL, encoded)

	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/dns-message")

	client := &http.Client{Timeout: r.PerAttempt}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("DoH returned status %d", resp.StatusCode)
	}

	buf := make([]byte, 512)
	n, err := resp.Body.Read(buf)
	if err != nil && n == 0 {
		return "", err
	}

	var reply dns.Msg
	if err := reply.Unpack(buf[:n]); err != nil {
		return "", err
	}

	for _, ans := range reply.Answer {
		if a, ok := ans.(*dns.A); ok {
			return a.A.String(), nil
		}
	}

	return "", fmt.Errorf("no A record in response")
}

// resolveDoT performs DNS-over-TLS query.
func (r *Resolver) resolveDoT(ctx context.Context, addr, domain string) (string, error) {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	msg.RecursionDesired = true

	// Ensure port is present
	if !strings.Contains(addr, ":") {
		addr = addr + ":853"
	}

	c := &dns.Client{
		Net:          "tcp-tls",
		ReadTimeout:  r.PerAttempt,
		WriteTimeout: r.PerAttempt,
	}

	// Create connection with context
	conn, err := net.DialTimeout("tcp", addr, r.PerAttempt)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// Wrap in TLS (miekg/dns handles this with tcp-tls)
	reply, _, err := c.ExchangeWithConnContext(ctx, msg, &dns.Conn{Conn: conn})
	if err != nil {
		return "", err
	}

	for _, ans := range reply.Answer {
		if a, ok := ans.(*dns.A); ok {
			return a.A.String(), nil
		}
	}

	return "", fmt.Errorf("no A record in response")
}

// resolvePlain performs plain DNS query over UDP.
func (r *Resolver) resolvePlain(ctx context.Context, addr, domain string) (string, error) {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	msg.RecursionDesired = true

	// Ensure port is present
	if !strings.Contains(addr, ":") {
		addr = addr + ":53"
	}

	c := &dns.Client{
		Net:          "udp",
		ReadTimeout:  r.PerAttempt,
		WriteTimeout: r.PerAttempt,
	}

	reply, _, err := c.ExchangeContext(ctx, msg, addr)
	if err != nil {
		return "", err
	}

	for _, ans := range reply.Answer {
		if a, ok := ans.(*dns.A); ok {
			return a.A.String(), nil
		}
	}

	return "", fmt.Errorf("no A record in response")
}
