package engine

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"time"
)

// TCPPing measures TCP connection latency to a host:port.
// Returns average RTT of successful attempts, or -1 if all failed.
func TCPPing(host string, port int, attempts int) time.Duration {
	if attempts <= 0 {
		attempts = 3
	}

	var total time.Duration
	successCount := 0

	for i := 0; i < attempts; i++ {
		start := time.Now()
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, itoa(port)), 2*time.Second)
		if err == nil {
			conn.Close()
			total += time.Since(start)
			successCount++
		}
	}

	if successCount == 0 {
		return -1
	}

	return total / time.Duration(successCount)
}

// RealDelayTest measures actual HTTP delay through a proxy.
// It starts sing-box in proxy mode, waits for the port, then makes an HTTP request.
func RealDelayTest(ctx context.Context, singBoxPath, configPath string, proxyPort int) (time.Duration, error) {
	// Start sing-box
	cmd := exec.Command(singBoxPath, "run", "-c", configPath)
	if err := cmd.Start(); err != nil {
		return -1, err
	}

	// Ensure cleanup
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	// Wait for port to be ready (up to 3 seconds)
	ready := make(chan bool, 1)
	go func() {
		for i := 0; i < 30; i++ {
			conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", itoa(proxyPort)), 100*time.Millisecond)
			if err == nil {
				conn.Close()
				ready <- true
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		ready <- false
	}()

	if !<-ready {
		return -1, ErrPortNotReady
	}

	// Make HTTP request through proxy
	proxyURL := "http://127.0.0.1:" + itoa(proxyPort)
	transport := &http.Transport{
		Proxy: http.ProxyURL(MustParseURL(proxyURL)),
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}

	start := time.Now()
	resp, err := client.Get("http://cp.cloudflare.com/generate_204")
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	return time.Since(start), nil
}

// MustParseURL parses a URL or panics (used for known-good proxy URLs).
func MustParseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

// ErrPortNotReady indicates the proxy port did not become available.
var ErrPortNotReady = &timeoutError{"proxy port not ready"}

type timeoutError struct {
	msg string
}

func (e *timeoutError) Error() string   { return e.msg }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }

// itoa converts int to string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negative := false
	if n < 0 {
		negative = true
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if negative {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
