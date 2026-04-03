// Package passphrase provides a concurrency-safe store for the shared passphrase.
package passphrase

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/crypto/argon2"
)

const minLength = 8
const maxLength = 128

var (
	ErrTooShort        = errors.New("passphrase must be at least 8 characters")
	ErrTooLong         = errors.New("passphrase must be at most 128 characters")
	ErrLeadingSpace    = errors.New("passphrase must not start with whitespace")
	ErrTrailingSpace   = errors.New("passphrase must not end with whitespace")
)

// Validate returns an error if the passphrase does not meet requirements.
func Validate(p string) error {
	if len(p) < minLength {
		return ErrTooShort
	}
	if len(p) > maxLength {
		return ErrTooLong
	}
	if p != strings.TrimLeft(p, " \t") {
		return ErrLeadingSpace
	}
	if p != strings.TrimRight(p, " \t") {
		return ErrTrailingSpace
	}
	return nil
}

// Store holds the current passphrase and allows safe concurrent access.
type Store struct {
	mu    sync.RWMutex
	value string
	hash  string
}

// Get returns the current passphrase.
func (s *Store) Get() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.value
}

// reversePassphrase returns the UTF-8 reversal of p, used as the Argon2id salt.
// Deriving the salt from the input avoids a hardcoded constant visible in source code.
func reversePassphrase(p string) []byte {
	runes := []rune(p)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return []byte(string(runes))
}

// Set updates the passphrase and recomputes the cached Argon2id hash.
func (s *Store) Set(passphrase string) {
	key := argon2.IDKey([]byte(passphrase), reversePassphrase(passphrase), 1, 64*1024, 4, 32)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value = passphrase
	s.hash = fmt.Sprintf("%x", key)
}

// Hash returns the cached Argon2id hex digest computed when the passphrase was last set.
func (s *Store) Hash() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.hash
}

// ShortHash returns the first 16 hex characters (8 bytes) of the cached hash.
// Returns an empty string if no passphrase has been set.
func (s *Store) ShortHash() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.hash) < 16 {
		return ""
	}
	return s.hash[:16]
}
