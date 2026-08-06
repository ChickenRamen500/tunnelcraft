package subscription

import (
        "context"
        "crypto/tls"
        "fmt"
        "io"
        "net/http"
        "net/url"
        "strings"
        "time"

        "github.com/ChickenRamen500/tunnelcraft/core/internal/config"
        "github.com/ChickenRamen500/tunnelcraft/core/internal/engine"
)

// Provider fetches subscription content from a URL and parses it into servers.
type Provider struct {
        httpClient *http.Client
        cfg        *config.Manager
}

// NewProvider creates a new subscription provider.
func NewProvider(cfg *config.Manager) *Provider {
        return &Provider{
                httpClient: &http.Client{
                        Timeout: 30 * time.Second,
                        Transport: &http.Transport{
                                TLSClientConfig: &tls.Config{
                                        InsecureSkipVerify: false,
                                },
                        },
                },
                cfg: cfg,
        }
}

// FetchResult contains the result of a subscription fetch operation.
type FetchResult struct {
        SubscriptionID string
        Servers        []engine.ServerConfig
        ParseErrors    []ParseError
        Format         string
        RawSize        int
        FetchedAt      time.Time
        Error          string
}

// Fetch downloads and parses a subscription by its ID.
func (p *Provider) Fetch(ctx context.Context, subscriptionID string) *FetchResult {
        result := &FetchResult{
                SubscriptionID: subscriptionID,
                FetchedAt:      time.Now(),
        }

        // Find the subscription
        subs := p.cfg.GetSubscriptions()
        var sub config.SubscriptionEntry
        found := false
        for _, s := range subs {
                if s.ID == subscriptionID {
                        sub = s
                        found = true
                        break
                }
        }
        if !found {
                result.Error = fmt.Sprintf("subscription %s not found", subscriptionID)
                return result
        }

        if !sub.Enabled {
                result.Error = "subscription is disabled"
                return result
        }

        // Fetch the content
        data, err := p.fetchContent(ctx, sub.URL, sub.Username, sub.Password)
        if err != nil {
                result.Error = fmt.Sprintf("fetch failed: %v", err)
                return result
        }

        result.RawSize = len(data)
        result.Format = DetectFormat(data)

        // Parse the content
        result.Servers, result.ParseErrors = Parse(data)

        // Tag servers with subscription ID and apply filter
        if sub.Filter != "" {
                filtered := make([]engine.ServerConfig, 0, len(result.Servers))
                for _, s := range result.Servers {
                        if containsKeyword(s.Name, sub.Filter) {
                                s.SubscriptionID = subscriptionID
                                filtered = append(filtered, s)
                        }
                }
                result.Servers = filtered
        } else {
                for i := range result.Servers {
                        result.Servers[i].SubscriptionID = subscriptionID
                }
        }

        return result
}

// FetchAll fetches all enabled subscriptions.
func (p *Provider) FetchAll(ctx context.Context) []*FetchResult {
        subs := p.cfg.GetSubscriptions()
        results := make([]*FetchResult, 0, len(subs))

        for _, sub := range subs {
                if !sub.Enabled {
                        continue
                }
                results = append(results, p.Fetch(ctx, sub.ID))
        }

        return results
}

// fetchContent downloads raw bytes from a URL with optional Basic Auth.
func (p *Provider) fetchContent(ctx context.Context, rawURL, username, password string) ([]byte, error) {
        req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
        if err != nil {
                return nil, fmt.Errorf("invalid URL: %w", err)
        }

        // Add User-Agent header (some providers block default UA)
        req.Header.Set("User-Agent", "TunnelCraft/0.1.0")

        // Add Basic Auth if credentials provided
        if username != "" {
                req.SetBasicAuth(username, password)
        }

        // Validate URL
        if _, err := url.Parse(rawURL); err != nil {
                return nil, fmt.Errorf("invalid URL: %w", err)
        }

        resp, err := p.httpClient.Do(req)
        if err != nil {
                return nil, fmt.Errorf("HTTP request failed: %w", err)
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
                return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
        }

        data, err := io.ReadAll(resp.Body)
        if err != nil {
                return nil, fmt.Errorf("failed to read response: %w", err)
        }

        return data, nil
}

// containsKeyword checks if the server name contains the filter keyword (case-insensitive).
func containsKeyword(name, keyword string) bool {
        // Simple case-insensitive substring match
        return len(keyword) == 0 || containsIgnoreCase(name, keyword)
}

func containsIgnoreCase(s, substr string) bool {
        s = strings.ToLower(s)
        substr = strings.ToLower(substr)
        return strings.Contains(s, substr)
}