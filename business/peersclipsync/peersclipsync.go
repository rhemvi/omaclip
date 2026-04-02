// Package peersclipsync fetches clipboard history from discovered remote peers.
package peersclipsync

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"clipmaster/business/clipboard"
	fmdns "clipmaster/foundation/mdns"
)

// PeerClipboard holds a remote peer's name and its clipboard entries.
type PeerClipboard struct {
	PeerName string                     `json:"peerName"`
	Entries  []clipboard.ClipboardEntry `json:"entries"`
}

// Fetcher periodically fetches clipboard history from all discovered peers.
type Fetcher struct {
	log          *slog.Logger
	discoverer   *fmdns.Discoverer
	syncInterval time.Duration
	client       *http.Client

	mu    sync.RWMutex
	cache map[string]PeerClipboard

	OnUpdate func()
}

// New creates a Fetcher. Call Start to begin periodic fetching.
func New(log *slog.Logger, discoverer *fmdns.Discoverer, syncInterval time.Duration) *Fetcher {
	return &Fetcher{
		log:          log,
		discoverer:   discoverer,
		syncInterval: syncInterval,
		client:       &http.Client{Timeout: 5 * time.Second},
		cache:        make(map[string]PeerClipboard),
	}
}

// Start begins the 30s fetch loop until ctx is cancelled.
func (f *Fetcher) Start(ctx context.Context) {
	go f.loop(ctx)
}

// GetAll returns a snapshot of all remote peer clipboards.
func (f *Fetcher) GetAll() []PeerClipboard {
	f.mu.RLock()
	defer f.mu.RUnlock()
	out := make([]PeerClipboard, 0, len(f.cache))
	for _, pc := range f.cache {
		out = append(out, pc)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].PeerName < out[j].PeerName
	})
	return out
}

func (f *Fetcher) loop(ctx context.Context) {
	f.fetchAll()
	ticker := time.NewTicker(f.syncInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			f.fetchAll()
		}
	}
}

func (f *Fetcher) fetchAll() {
	peers := f.discoverer.Peers()
	changed := false

	activePeers := make(map[string]struct{}, len(peers))
	for _, p := range peers {
		activePeers[p.Name] = struct{}{}
	}

	f.mu.Lock()
	for name := range f.cache {
		if _, ok := activePeers[name]; !ok {
			delete(f.cache, name)
			changed = true
		}
	}
	f.mu.Unlock()

	for _, p := range peers {
		entries, err := f.fetchPeer(p)
		if err != nil {
			f.log.Debug("failed to fetch peer clipboard", "peer", p.Name, "error", err)
			continue
		}
		displayName := strings.SplitN(p.Name, ".", 2)[0]
		f.mu.Lock()
		f.cache[p.Name] = PeerClipboard{PeerName: displayName, Entries: entries}
		f.mu.Unlock()
		changed = true
	}
	if changed && f.OnUpdate != nil {
		f.OnUpdate()
	}
}

func (f *Fetcher) fetchPeer(p fmdns.Peer) ([]clipboard.ClipboardEntry, error) {
	url := fmt.Sprintf("http://%s:%d/api/clipboard", p.Addr, p.Port)
	resp, err := f.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck

	var entries []clipboard.ClipboardEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return entries, nil
}
