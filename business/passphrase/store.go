// Package passphrase provides a concurrency-safe store for the shared passphrase.
package passphrase

import "sync"

// Store holds the current passphrase and allows safe concurrent access.
type Store struct {
	mu    sync.RWMutex
	value string
}

// Get returns the current passphrase.
func (s *Store) Get() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.value
}

// Set updates the passphrase.
func (s *Store) Set(passphrase string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value = passphrase
}
