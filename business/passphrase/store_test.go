package passphrase

import (
	"fmt"
	"sync"
	"testing"

	"golang.org/x/crypto/argon2"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{"valid", "validpass", nil},
		{"minimum length", "12345678", nil},
		{"maximum length", string(make([]byte, 128)), nil},
		{"too short", "short", ErrTooShort},
		{"too long", string(make([]byte, 129)), ErrTooLong},
		{"leading space", " leading", ErrLeadingSpace},
		{"leading tab", "\tleading", ErrLeadingSpace},
		{"trailing space", "trailing ", ErrTrailingSpace},
		{"trailing tab", "trailing\t", ErrTrailingSpace},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.input)
			if err != tt.wantErr {
				t.Errorf("Validate(%q) = %v, want %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestStore_GetSet(t *testing.T) {
	s := &Store{}

	if got := s.Get(); got != "" {
		t.Errorf("new store Get() = %q, want empty string", got)
	}

	s.Set("mypassphrase")
	if got := s.Get(); got != "mypassphrase" {
		t.Errorf("Get() = %q, want %q", got, "mypassphrase")
	}

	s.Set("newpassphrase")
	if got := s.Get(); got != "newpassphrase" {
		t.Errorf("Get() after update = %q, want %q", got, "newpassphrase")
	}
}

func TestStore_Hash(t *testing.T) {
	s := &Store{}
	s.Set("mypassphrase")

	key := argon2.IDKey([]byte("mypassphrase"), reversePassphrase("mypassphrase"), 1, 64*1024, 4, 32)
	want := fmt.Sprintf("%x", key)

	if got := s.Hash(); got != want {
		t.Errorf("Hash() = %q, want %q", got, want)
	}
}

func TestStore_HashChangesWithPassphrase(t *testing.T) {
	s := &Store{}
	s.Set("first")
	h1 := s.Hash()

	s.Set("second")
	h2 := s.Hash()

	if h1 == h2 {
		t.Error("Hash() returned same value for different passphrases")
	}
}

func TestStore_HashEmptyBeforeSet(t *testing.T) {
	s := &Store{}
	if got := s.Hash(); got != "" {
		t.Errorf("Hash() before Set() = %q, want empty string", got)
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	s := &Store{}
	s.Set("initial")

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(3)
		go func() { defer wg.Done(); s.Get() }()
		go func() { defer wg.Done(); s.Set("concurrent") }()
		go func() { defer wg.Done(); s.Hash() }()
	}
	wg.Wait()
}
