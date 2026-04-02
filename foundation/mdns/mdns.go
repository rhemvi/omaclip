// Package mdns wraps github.com/hashicorp/mdns to advertise and discover
// Clipmaster instances on the local network.
package mdns

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/mdns"
)

const (
	serviceType = "_clipmaster._tcp"
	domain      = "local."
)

var ErrNoDiscoverableIPs = fmt.Errorf("mdns: no discoverable IPs, skipping registering to the network")

// Peer describes a discovered remote Clipmaster instance.
type Peer struct {
	Name string `json:"name"`
	Addr string `json:"addr"`
	Port int    `json:"port"`
}

// Discoverer registers this instance via mDNS and continuously browses for peers.
type Discoverer struct {
	log          *slog.Logger
	server       *mdns.Server
	myName       string
	browsePeriod time.Duration

	mu       sync.RWMutex
	peers    map[string]Peer
	hostname string
}

// New creates a Discoverer. Call Register then Start to begin advertising and browsing.
func New(log *slog.Logger, browsePeriod time.Duration, hostname string) *Discoverer {
	return &Discoverer{
		log:          log,
		browsePeriod: browsePeriod,
		peers:        make(map[string]Peer),
		hostname:     hostname,
	}
}

// Register advertises this Clipmaster instance at the given port via mDNS.
func (d *Discoverer) Register(port int) error {
	instanceName := fmt.Sprintf("%s-%d", d.hostname, port)
	d.myName = instanceName

	ips := lanIPs(d.hostname)
	if ips == nil {
		return ErrNoDiscoverableIPs
	}

	svc, err := mdns.NewMDNSService(instanceName, serviceType, domain, "", port, ips, []string{"version=1"})
	if err != nil {
		return fmt.Errorf("mdns: creating service: %w", err)
	}

	srv, err := mdns.NewServer(&mdns.Config{Zone: svc, Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		return fmt.Errorf("mdns: starting server: %w", err)
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
		d.server.Shutdown() //nolint:errcheck
	}
}

func (d *Discoverer) browseLoop(ctx context.Context) {
	d.browse()
	ticker := time.NewTicker(d.browsePeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.browse()
		}
	}
}

func (d *Discoverer) browse() {
	entries := make(chan *mdns.ServiceEntry, 16)
	go func() {
		params := mdns.DefaultParams(serviceType)
		params.Entries = entries
		params.DisableIPv6 = true
		params.Logger = log.New(io.Discard, "", 0)
		if err := mdns.Query(params); err != nil {
			d.log.Warn("mdns browse failed", "error", err)
		}
		close(entries)
	}()

	found := make(map[string]Peer)
	for entry := range entries {
		if d.myName != "" && strings.HasPrefix(entry.Name, d.myName) {
			continue
		}

		addr := ""
		if entry.AddrV4 != nil {
			addr = entry.AddrV4.String()
		} else if entry.AddrV6 != nil {
			addr = entry.AddrV6.String()
		}

		found[entry.Name] = Peer{
			Name: entry.Name,
			Addr: addr,
			Port: entry.Port,
		}
	}

	d.mu.Lock()
	d.peers = found
	d.mu.Unlock()

	for _, p := range found {
		d.log.Info("mdns peer discovered", "name", p.Name, "addr", p.Addr, "port", p.Port)
	}
}

func lanIPs(hostname string) []net.IP {
	resolved, err := net.LookupIP(hostname)
	if err != nil {
		return nil
	}

	return filterIPs(resolved)
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
