package tunnel

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/ChickenRamen500/tunnelcraft/core/internal/config"
	"github.com/ChickenRamen500/tunnelcraft/core/internal/engine"
)

// RoutingManager manages Windows system routes for split tunneling.
// On non-Windows, all operations are stubs.
type RoutingManager struct {
	mu           sync.Mutex
	wintun       *WintunDLL
	appliedRoutes []string
	tunIP        string
	killSwitch   bool
}

// NewRoutingManager creates a new routing manager.
func NewRoutingManager(wintun *WintunDLL) *RoutingManager {
	return &RoutingManager{wintun: wintun}
}

// Setup creates the TUN adapter and configures system routing.
func (r *RoutingManager) Setup(socksPort, httpPort uint32, server *engine.ServerConfig) error {
	if runtime.GOOS != "windows" {
		log.Println("[routing] non-Windows platform, skipping TUN/routing setup")
		return nil
	}

	// For WireGuard/AmneziaWG, they manage their own TUN interface
	// We don't need to create a TUN adapter for them
	switch server.Protocol {
	case engine.ProtocolWireGuard, engine.ProtocolAmneziaWG:
		log.Println("[routing] WireGuard/AmneziaWG: skipping TUN adapter creation (they manage their own)")
		return nil
	}

	// For other protocols (xray, hysteria), create TUN adapter
	adapter, err := r.wintun.CreateAdapter(TUNAdapterName, DefaultMTU)
	if err != nil {
		return fmt.Errorf("failed to create TUN adapter: %w", err)
	}
	log.Printf("[routing] TUN adapter '%s' created (MTU: %d)", adapter.Name(), adapter.MTU())

	// For SOCKS/HTTP proxy mode (xray, hysteria), we configure the system proxy
	return r.setupProxyRouting(socksPort, httpPort)
}

// Teardown removes all applied routes and destroys the TUN adapter.
func (r *RoutingManager) Teardown() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Remove all applied routes in reverse order
	for i := len(r.appliedRoutes) - 1; i >= 0; i-- {
		route := r.appliedRoutes[i]
		if err := r.deleteRoute(route); err != nil {
			log.Printf("[routing] failed to remove route %s: %v", route, err)
		}
	}
	r.appliedRoutes = nil

	// Close TUN adapter
	if err := r.wintun.CloseAdapter(); err != nil {
		log.Printf("[routing] failed to close TUN adapter: %v", err)
	}

	// Restore DNS
	_ = exec.Command("netsh", "interface", "ipv4", "set", "dns", "Ethernet", "dhcp").Run()

	log.Println("[routing] teardown complete")
	return nil
}

// ApplyRoutingRules applies split tunneling rules.
func (r *RoutingManager) ApplyRoutingRules(rules []config.RoutingRule) error {
	if runtime.GOOS != "windows" {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		switch rule.Action {
		case "proxy":
			// Route through TUN/proxy
			for _, cidr := range rule.IPCidrs {
				r.addRoute(cidr)
			}
			for _, domain := range rule.Domains {
				ips := resolveDomain(domain)
				for _, ip := range ips {
					r.addRoute(ip + "/32")
				}
			}

		case "direct":
			// Route directly (bypass VPN)
			for _, cidr := range rule.IPCidrs {
				r.addDirectRoute(cidr)
			}

		case "block":
			// Block by adding a blackhole route
			for _, cidr := range rule.IPCidrs {
				r.addBlockRoute(cidr)
			}
		}
	}

	return nil
}

// setupProxyRouting configures routing for proxy-based protocols.
func (r *RoutingManager) setupProxyRouting(socksPort, httpPort uint32) error {
	// On Windows, we can use:
	// 1. System proxy settings (IE/WinHTTP)
	// 2. TUN-based routing where all traffic goes through TUN
	//    and is forwarded to the SOCKS proxy

	// For now, configure the TUN adapter IP and default route
	tunIP := "10.200.0.1"
	r.tunIP = tunIP

	// Set TUN adapter IP
	if err := r.setInterfaceIP(TUNAdapterName, tunIP+"/24"); err != nil {
		log.Printf("[routing] failed to set TUN IP: %v", err)
	}

	// Add default route through TUN (with high metric to coexist with LAN)
	if err := r.addRoute("0.0.0.0/0"); err != nil {
		log.Printf("[routing] failed to add default route: %v", err)
	}

	log.Printf("[routing] proxy routing configured (SOCKS: %d, HTTP: %d)", socksPort, httpPort)
	return nil
}

// addRoute adds a route through the TUN interface.
func (r *RoutingManager) addRoute(cidr string) error {
	if runtime.GOOS != "windows" {
		return nil
	}

	cmd := exec.Command("netsh", "interface", "ipv4", "add", "route",
		cidr, TUNAdapterName, fmt.Sprintf("metric=%d", 5))

	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("[routing] add route %s failed: %s: %v", cidr, string(output), err)
		return err
	}

	r.appliedRoutes = append(r.appliedRoutes, cidr)
	log.Printf("[routing] added route: %s via %s", cidr, TUNAdapterName)
	return nil
}

// addDirectRoute adds a route that bypasses the TUN (direct access).
func (r *RoutingManager) addDirectRoute(cidr string) error {
	if runtime.GOOS != "windows" {
		return nil
	}

	// Get the default gateway interface
	cmd := exec.Command("netsh", "interface", "ipv4", "add", "route",
		cidr, "Ethernet", fmt.Sprintf("metric=%d", 1))

	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("[routing] add direct route %s failed: %s: %v", cidr, string(output), err)
		return err
	}

	log.Printf("[routing] added direct route: %s", cidr)
	return nil
}

// addBlockRoute adds a route that drops traffic (blackhole).
func (r *RoutingManager) addBlockRoute(cidr string) error {
	if runtime.GOOS != "windows" {
		return nil
	}

	// Use a non-existent gateway to block traffic
	cmd := exec.Command("route", "add", cidr, "0.0.0.0", "metric", "1")

	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("[routing] add block route %s failed: %s: %v", cidr, string(output), err)
		return err
	}

	log.Printf("[routing] added block route: %s", cidr)
	return nil
}

// deleteRoute removes a previously added route.
func (r *RoutingManager) deleteRoute(cidr string) error {
	cmd := exec.Command("netsh", "interface", "ipv4", "delete", "route", cidr, TUNAdapterName)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Try route command fallback
		cmd2 := exec.Command("route", "delete", cidr)
		if output2, err2 := cmd2.CombinedOutput(); err2 != nil {
			log.Printf("[routing] delete route %s failed: %s / %s: %v / %v",
				cidr, strings.TrimSpace(string(output)), strings.TrimSpace(string(output2)), err, err2)
			return err2
		}
	}
	return nil
}

// setInterfaceIP configures an IP address on the TUN interface.
func (r *RoutingManager) setInterfaceIP(iface, ipCidr string) error {
	cmd := exec.Command("netsh", "interface", "ipv4", "set", "address",
		iface, "static", strings.Split(ipCidr, "/")[0], "255.255.255.0")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// SetKillSwitch enables or disables the kill switch.
// When enabled, all traffic is blocked if the VPN drops.
func (r *RoutingManager) SetKillSwitch(enabled bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.killSwitch = enabled
}

// EnableKillSwitch blocks all non-VPN traffic.
func (r *RoutingManager) EnableKillSwitch() error {
	if runtime.GOOS != "windows" {
		return nil
	}

	// Delete the default route to block all traffic
	cmd := exec.Command("route", "delete", "0.0.0.0")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("[routing] kill switch: failed to delete default route: %s: %v",
			strings.TrimSpace(string(output)), err)
	}

	log.Println("[routing] kill switch ENABLED")
	return nil
}

// DisableKillSwitch restores normal routing.
func (r *RoutingManager) DisableKillSwitch() error {
	if runtime.GOOS != "windows" {
		return nil
	}

	// Restore DHCP to get the default gateway back
	cmd := exec.Command("netsh", "interface", "ipv4", "set", "address", "Ethernet", "dhcp")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("[routing] kill switch: failed to restore DHCP: %s: %v",
			strings.TrimSpace(string(output)), err)
	}

	log.Println("[routing] kill switch DISABLED")
	return nil
}

// resolveDomain resolves a domain name to IP addresses.
func resolveDomain(domain string) []string {
	// Clean up wildcard domains
	domain = strings.TrimPrefix(domain, "*.")
	domain = strings.TrimPrefix(domain, ".")

	ips, err := net.LookupIP(domain)
	if err != nil {
		log.Printf("[routing] failed to resolve %s: %v", domain, err)
		return nil
	}

	var result []string
	for _, ip := range ips {
		// Only use IPv4 for routing
		if ipv4 := ip.To4(); ipv4 != nil {
			result = append(result, ipv4.String())
		}
	}
	return result
}
