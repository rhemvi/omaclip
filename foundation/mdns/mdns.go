// Package mdns wraps github.com/grandcat/zeroconf to advertise and discover
// Omaclip instances on the local network.
package mdns

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/rhemvi/omaclip/business/passphrase"

	"github.com/grandcat/zeroconf"
)

const (
	serviceType = "_omaclip._tcp"
	domain      = "local."
)

var ErrNoDiscoverableIPs = fmt.Errorf("mdns: no discoverable IPs, skipping registering to the network")

// Peer describes a discovered remote Omaclip instance.
type Peer struct {
	Name string `json:"name"`
	Addr string `json:"addr"`
	Port int    `json:"port"`
}

const peerTTLCycles = 3

// Discoverer registers this instance via mDNS and continuously browses for peers.
type Discoverer struct {
	log             *slog.Logger
	server          *zeroconf.Server
	myName          string
	browsePeriod    time.Duration
	passphraseStore *passphrase.Store
	iface           *net.Interface

	mu       sync.RWMutex
	peers    map[string]Peer
	lastSeen map[string]int
	hostname string
}

// New creates a Discoverer. Call Register then Start to begin advertising and browsing.
// If ifaceName is non-empty, mDNS will bind to that network interface only.
func New(log *slog.Logger, browsePeriod time.Duration, hostname string, ps *passphrase.Store, ifaceName string) (*Discoverer, error) {
	d := &Discoverer{
		log:             log,
		browsePeriod:    browsePeriod,
		peers:           make(map[string]Peer),
		lastSeen:        make(map[string]int),
		hostname:        hostname,
		passphraseStore: ps,
	}

	if ifaceName != "" {
		iface, err := net.InterfaceByName(ifaceName)
		if err != nil {
			return nil, fmt.Errorf("mdns: looking up interface %q: %w", ifaceName, err)
		}
		d.iface = iface
	}

	return d, nil
}

// Register advertises this Omaclip instance at the given port via mDNS.
func (d *Discoverer) Register(port int) error {
	instanceName := fmt.Sprintf("%s-%d", d.hostname, port)
	d.myName = instanceName

	var ips []net.IP
	if d.iface != nil {
		ips = ifaceIPs(d.iface)
	} else {
		ips = lanIPs(d.hostname)
	}
	if ips == nil {
		return ErrNoDiscoverableIPs
	}

	txt := []string{"version=1", "ph=" + d.passphraseStore.ShortHash()}

	ipStrs := make([]string, len(ips))
	for i, ip := range ips {
		ipStrs[i] = ip.String()
	}

	var ifaces []net.Interface
	if d.iface != nil {
		ifaces = []net.Interface{*d.iface}
	}

	srv, err := zeroconf.RegisterProxy(instanceName, serviceType, domain, port, instanceName, ipStrs, txt, ifaces)
	if err != nil {
		return fmt.Errorf("mdns: registering service: %w", err)
	}

	d.server = srv
	d.log.Info("mdns registered", "instance", instanceName, "port", port)
	return nil
}

// Start begins the periodic browse loop until ctx is cancelled.
func (d *Discoverer) Start(ctx context.Context) {
	go d.browseLoop(ctx)
}

// Peers returns a snapshot of currently known peers.
func (d *Discoverer) Peers() []Peer {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make([]Peer, 0, len(d.peers))
	for _, p := range d.peers {
		out = append(out, p)
	}
	return out
}

// Shutdown tears down the mDNS server.
func (d *Discoverer) Shutdown() {
	if d.server != nil {
		d.server.Shutdown()
	}
}

func (d *Discoverer) browseLoop(ctx context.Context) {
	d.browse(ctx)
	ticker := time.NewTicker(d.browsePeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.browse(ctx)
		}
	}
}

func (d *Discoverer) browse(ctx context.Context) {
	var opts []zeroconf.ClientOption
	if d.iface != nil {
		opts = append(opts, zeroconf.SelectIfaces([]net.Interface{*d.iface}))
	}
	opts = append(opts, zeroconf.SelectIPTraffic(zeroconf.IPv4))

	resolver, err := zeroconf.NewResolver(opts...)
	if err != nil {
		d.log.Warn("mdns browse failed", "error", err)
		return
	}

	entries := make(chan *zeroconf.ServiceEntry, 16)

	browseCtx, cancel := context.WithTimeout(ctx, d.browsePeriod)
	defer cancel()

	go func() {
		if err := resolver.Browse(browseCtx, serviceType, domain, entries); err != nil {
			d.log.Warn("mdns browse failed", "error", err)
		}
	}()

	seen := make(map[string]Peer)
	for entry := range entries {
		name := entry.Instance
		if d.myName != "" && name == d.myName {
			continue
		}

		if !d.peerMatchesPassphrase(entry.Text) {
			d.log.Debug("mdns peer skipped: passphrase mismatch", "name", name)
			continue
		}

		addr := ""
		if len(entry.AddrIPv4) > 0 {
			addr = entry.AddrIPv4[0].String()
		} else if len(entry.AddrIPv6) > 0 {
			addr = entry.AddrIPv6[0].String()
		}

		seen[name] = Peer{
			Name: name,
			Addr: addr,
			Port: entry.Port,
		}
	}

	d.mu.Lock()
	for name, peer := range seen {
		d.peers[name] = peer
		d.lastSeen[name] = 0
	}
	for name := range d.peers {
		if _, ok := seen[name]; !ok {
			d.lastSeen[name]++
			if d.lastSeen[name] >= peerTTLCycles {
				delete(d.peers, name)
				delete(d.lastSeen, name)
			}
		}
	}
	d.mu.Unlock()

	for _, p := range seen {
		d.log.Debug("mdns peer discovered", "name", p.Name, "addr", p.Addr, "port", p.Port)
	}
}

func lanIPs(hostname string) []net.IP {
	resolved, err := net.LookupIP(hostname)
	if err != nil {
		return nil
	}

	return filterIPs(resolved)
}

// ifaceIPs returns the IPv4 addresses assigned to a specific network interface.
func ifaceIPs(iface *net.Interface) []net.IP {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil
	}

	var ips []net.IP
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		if ip4 := ipNet.IP.To4(); ip4 != nil {
			ips = append(ips, ip4)
		}
	}
	return ips
}

func filterIPs(candidates []net.IP) []net.IP {
	// 172.16.0.0/12 (RFC 1918 private range 172.16.0.0–172.31.255.255) is commonly used by Docker bridge networks and not routable across the LAN.
	_, blockedNet, _ := net.ParseCIDR("172.16.0.0/12")

	var ips []net.IP
	for _, ip := range candidates {
		if ip4 := ip.To4(); ip4 != nil && !blockedNet.Contains(ip4) {
			ips = append(ips, ip4)
		}
	}
	return ips
}

func (d *Discoverer) peerMatchesPassphrase(infoFields []string) bool {
	hash := d.passphraseStore.ShortHash()
	for _, field := range infoFields {
		if strings.HasPrefix(field, "ph=") {
			return strings.TrimPrefix(field, "ph=") == hash
		}
	}
	return false
}
