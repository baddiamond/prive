package keys

import (
	"errors"
	"sync"

	"github.com/zalando/go-keyring"
)

// ErrNotFound is returned by a Keychain when an entry does not exist.
var ErrNotFound = errors.New("keychain entry not found")

// Keychain stores small secrets in the operating system's credential store.
type Keychain interface {
	Get(service, user string) (string, error)
	Set(service, user, secret string) error
	Delete(service, user string) error
}

// OSKeychain uses the macOS Keychain, Windows Credential Manager or the Linux Secret Service.
func OSKeychain() Keychain { return osKeychain{} }

type osKeychain struct{}

func (osKeychain) Get(service, user string) (string, error) {
	s, err := keyring.Get(service, user)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrNotFound
	}
	return s, err
}

func (osKeychain) Set(service, user, secret string) error {
	return keyring.Set(service, user, secret)
}

func (osKeychain) Delete(service, user string) error {
	err := keyring.Delete(service, user)
	if errors.Is(err, keyring.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

// MemoryKeychain is an in-process Keychain for tests and development tools.
type MemoryKeychain struct {
	mu      sync.Mutex
	entries map[string]string
	// Err, if set, is returned by every call, to simulate an unavailable keychain.
	Err error
}

// NewMemoryKeychain returns an empty in-memory keychain.
func NewMemoryKeychain() *MemoryKeychain {
	return &MemoryKeychain{entries: map[string]string{}}
}

// Get implements Keychain.
func (m *MemoryKeychain) Get(service, user string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return "", m.Err
	}
	s, ok := m.entries[service+"\x00"+user]
	if !ok {
		return "", ErrNotFound
	}
	return s, nil
}

// Set implements Keychain.
func (m *MemoryKeychain) Set(service, user, secret string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return m.Err
	}
	m.entries[service+"\x00"+user] = secret
	return nil
}

// Delete implements Keychain.
func (m *MemoryKeychain) Delete(service, user string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return m.Err
	}
	key := service + "\x00" + user
	if _, ok := m.entries[key]; !ok {
		return ErrNotFound
	}
	delete(m.entries, key)
	return nil
}

// Len returns the number of entries.
func (m *MemoryKeychain) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.entries)
}
