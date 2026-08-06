package dns

import (
    "bytes"
    "context"
    "encoding/base64"
    "fmt"
    "io"
    "net/http"
    "strings"
    "time"
)

// DoHResolver performs DNS-over-HTTPS resolution.
type DoHResolver struct {
    client    *http.Client
    serverURL string
}

// NewDoHResolver creates a new DoH resolver.
// serverURL is the DoH endpoint, e.g. "https://dns.google/dns-query".
func NewDoHResolver(serverURL string) *DoHResolver {
    return &DoHResolver{
        client: &http.Client{
            Timeout: 10 * time.Second,
        },
        serverURL: strings.TrimSuffix(serverURL, "/"),
    }
}

// Resolve performs a DNS-over-HTTPS query.
// Uses RFC 8484 GET method (base64url-encoded DNS wireformat).
func (d *DoHResolver) Resolve(ctx context.Context, domain string, qtype uint16) ([]DNSRecord, error) {
    // Build DNS query wireformat
    query := buildDNSQuery(domain, qtype)

    // Base64URL encode
    encoded := base64.RawURLEncoding.EncodeToString(query)

    // GET request with DNS wireformat in query parameter
    url := fmt.Sprintf("%s?dns=%s&type=%s", d.serverURL, encoded, qtype)

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    req.Header.Set("Accept", "application/dns-message")

    resp, err := d.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("DoH request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("DoH server returned HTTP %d", resp.StatusCode)
    }

    // Read DNS wireformat response
    data, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read DoH response: %w", err)
    }

    // Parse DNS response
    return parseDNSResponse(data), nil
}

// DNSRecord represents a single DNS record.
type DNSRecord struct {
    Name  string
    Type  uint16
    TTL   uint32
    Value string
}

// QueryType constants.
const (
    QueryTypeA     uint16 = 1
    QueryTypeAAAA  uint16 = 28
    QueryTypeCNAME uint16 = 5
)

// buildDNSQuery creates a minimal DNS query wireformat message.
// TODO: use a proper DNS library like miekg/dns in production.
func buildDNSQuery(domain string, qtype uint16) []byte {
    // DNS header (12 bytes)
    header := make([]byte, 12)
    // Transaction ID: 0x1234
    header[0] = 0x12
    header[1] = 0x34
    // Flags: standard query, recursion desired
    header[2] = 0x01
    header[3] = 0x00
    // Questions: 1
    header[5] = 0x01

    // DNS question
    question := encodeDNSName(domain)
    question = append(question, byte(qtype>>8), byte(qtype))
    question = append(question, 0x00, 0x01) // Class: IN

    return append(header, question...)
}

// encodeDNSName encodes a domain name into DNS wire format.
func encodeDNSName(domain string) []byte {
    var buf bytes.Buffer
    for _, label := range strings.Split(domain, ".") {
        buf.WriteByte(byte(len(label)))
        buf.WriteString(label)
    }
    buf.WriteByte(0) // root label
    return buf.Bytes()
}

// parseDNSResponse parses a DNS wireformat response into records.
func parseDNSResponse(data []byte) []DNSRecord {
    if len(data) < 12 {
        return nil
    }

    // Parse answer count
    anCount := uint16(data[6])<<8 | uint16(data[7])
    if anCount == 0 {
        return nil
    }

    // Skip header (12 bytes) and question section
    offset := 12
    offset = skipDNSName(data, offset)
    offset += 4 // qtype + qclass

    var records []DNSRecord
    for i := 0; i < int(anCount) && offset < len(data); i++ {
        offset = skipDNSName(data, offset)
        if offset+10 > len(data) {
            break
        }

        rtype := uint16(data[offset])<<8 | uint16(data[offset+1])
        rclass := uint16(data[offset+2])<<8 | uint16(data[offset+3])
        ttl := uint32(data[offset+4])<<24 | uint32(data[offset+5])<<16 |
            uint32(data[offset+6])<<8 | uint32(data[offset+7])
        rdlength := uint16(data[offset+8])<<8 | uint16(data[offset+9])
        offset += 10

        if offset+int(rdlength) > len(data) {
            break
        }

        rdata := data[offset : offset+int(rdlength)]
        offset += int(rdlength)

        // Only parse A and AAAA records for now
        record := DNSRecord{Type: rtype, TTL: ttl}
        switch rtype {
        case 1: // A record
            if len(rdata) == 4 {
                record.Value = fmt.Sprintf("%d.%d.%d.%d", rdata[0], rdata[1], rdata[2], rdata[3])
                records = append(records, record)
            }
        case 28: // AAAA record
            if len(rdata) == 16 {
                record.Value = formatIPv6(rdata)
                records = append(records, record)
            }
        case 5: // CNAME record
            if len(rdata) > 0 {
                record.Value = decodeDNSName(data, offset-int(rdlength))
                records = append(records, record)
            }
        }

        _ = rclass
    }

    return records
}

// skipDNSName skips over a DNS name (handling compression pointers).
func skipDNSName(data []byte, offset int) int {
    for offset < len(data) {
        b := data[offset]
        if b == 0 {
            return offset + 1
        }
        if b&0xC0 == 0xC0 {
            return offset + 2 // pointer
        }
        offset += int(b) + 1
    }
    return offset
}

// decodeDNSName decodes a DNS name at the given offset.
func decodeDNSName(data []byte, offset int) string {
    var labels []string
    for offset < len(data) {
        b := data[offset]
        if b == 0 {
            break
        }
        if b&0xC0 == 0xC0 {
            // Follow pointer
            if offset+1 >= len(data) {
                break
            }
            ptr := int(data[offset]&0x3F)<<8 | int(data[offset+1])
            offset = ptr
            continue
        }
        offset++
        if offset+int(b) > len(data) {
            break
        }
        labels = append(labels, string(data[offset:offset+int(b)]))
        offset += int(b)
    }
    return strings.Join(labels, ".")
}

// formatIPv6 formats 16 bytes as an IPv6 address string.
func formatIPv6(data []byte) string {
    var parts [8]string
    for i := 0; i < 8; i++ {
        v := uint16(data[i*2])<<8 | uint16(data[i*2+1])
        parts[i] = fmt.Sprintf("%x", v)
    }
    return strings.Join(parts[:], ":")
}