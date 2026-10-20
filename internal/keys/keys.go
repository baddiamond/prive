// Package keys manages Privé's data encryption key (DEK), as decided in ADR 0007.
//
// The DEK is 256 random bits generated on first run. It never touches the disk in clear:
// it is wrapped (XChaCha20-Poly1305) into "slots" stored in <DataDir>/keys.json, and any slot
// can unwrap it. Today there is one slot, the system keychain, so the app opens without a
// prompt. The recovery phrase and app password slots arrive later.
//
// Every database gets its own key, derived from the DEK with HKDF and a label, so one leaked
// derived key does not expose the others.
package keys

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"

	"github.com/baddiamond/prive/internal/core"
)

// FileName is the slot file inside the data directory.
const FileName = "keys.json"

// KeySize is the size in bytes of the DEK and of every derived key.
const KeySize = 32

// Error codes. Like every user-facing error they are translation keys (ADR 0004).
var (
	// ErrLocked means keys.json exists but no slot could unwrap the DEK (e.g. the keychain
	// entry is gone after an OS reinstall). The recovery phrase will unlock it (ADR 0007).
	ErrLocked = core.NewError("errors.encryption.locked")
	// ErrKeychainUnavailable means the OS credential store could not be reached.
	ErrKeychainUnavailable = core.NewError("errors.encryption.keychainUnavailable")
	// ErrCorrupt means keys.json cannot be read.
	ErrCorrupt = core.NewError("errors.encryption.corrupt")
)

// Codes lists the error codes of this package, for the translation check in tests.
func Codes() []*core.CodedError {
	return []*core.CodedError{ErrLocked, ErrKeychainUnavailable, ErrCorrupt}
}

const (
	fileVersion  = 1
	kindKeychain = "keychain"
	// keychainService names Privé's entries in the OS credential store.
	keychainService = "io.github.baddiamond.prive"
	hkdfSalt        = "prive/keys/v1"
)

// Keys holds the unlocked DEK.
type Keys struct {
	dek []byte
}

// slotFile is the JSON layout of keys.json.
type slotFile struct {
	Version int    `json:"version"`
	Slots   []slot `json:"slots"`
}

// slot is one wrapped copy of the DEK.
type slot struct {
	Kind string `json:"kind"`
	// ID names the keychain entry that holds this slot's wrapping key.
	ID         string `json:"id"`
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
}

// Unlock returns the DEK for dataDir. On first run (no keys.json) it creates the DEK and
// its keychain slot; created reports that case.
func Unlock(dataDir string, kc Keychain) (k *Keys, created bool, err error) {
	path := filepath.Join(dataDir, FileName)
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		k, err := create(path, kc)
		return k, err == nil, err
	}
	if err != nil {
		return nil, false, ErrCorrupt.Wrap(err)
	}

	var f slotFile
	if err := json.Unmarshal(raw, &f); err != nil || f.Version != fileVersion {
		return nil, false, ErrCorrupt.Wrap(fmt.Errorf("parse %s: %v", FileName, err))
	}
	for _, s := range f.Slots {
		if s.Kind != kindKeychain {
			continue
		}
		dek, err := openKeychainSlot(s, kc)
		if err != nil {
			return nil, false, err
		}
		return &Keys{dek: dek}, false, nil
	}
	return nil, false, ErrLocked
}

// Derive returns a 32-byte key for one purpose, e.g. "prive.db" or "engine/whatsapp/<accountId>/store".
// The same label always yields the same key.
func (k *Keys) Derive(label string) []byte {
	out := make([]byte, KeySize)
	r := hkdf.New(sha256.New, k.dek, []byte(hkdfSalt), []byte(label))
	if _, err := io.ReadFull(r, out); err != nil {
		panic(fmt.Sprintf("keys: hkdf: %v", err)) // cannot happen for 32 bytes
	}
	return out
}

// Forget deletes every keychain entry referenced by dataDir's keys.json. Used by
// "Delete everything"; afterwards the data in dataDir can no longer be decrypted.
func Forget(dataDir string, kc Keychain) error {
	raw, err := os.ReadFile(filepath.Join(dataDir, FileName))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var f slotFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return ErrCorrupt.Wrap(err)
	}
	for _, s := range f.Slots {
		if s.Kind == kindKeychain {
			if err := kc.Delete(keychainService, s.ID); err != nil && !errors.Is(err, ErrNotFound) {
				return err
			}
		}
	}
	return nil
}

// create generates the DEK and its keychain slot. The keychain entry is written first and
// removed again if keys.json cannot be saved, so no orphan entry is left behind.
func create(path string, kc Keychain) (*Keys, error) {
	dek := randomBytes(KeySize)
	kek := randomBytes(KeySize)
	id := "data-key-" + hex.EncodeToString(randomBytes(8))

	if err := kc.Set(keychainService, id, base64.StdEncoding.EncodeToString(kek)); err != nil {
		return nil, ErrKeychainUnavailable.Wrap(err)
	}
	s, err := seal(kek, dek, kindKeychain, id)
	if err == nil {
		err = writeFileAtomic(path, slotFile{Version: fileVersion, Slots: []slot{s}})
	}
	if err != nil {
		_ = kc.Delete(keychainService, id)
		return nil, ErrCorrupt.Wrap(err)
	}
	return &Keys{dek: dek}, nil
}

func openKeychainSlot(s slot, kc Keychain) ([]byte, error) {
	secret, err := kc.Get(keychainService, s.ID)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrLocked
	}
	if err != nil {
		return nil, ErrKeychainUnavailable.Wrap(err)
	}
	kek, err := base64.StdEncoding.DecodeString(secret)
	if err != nil || len(kek) != KeySize {
		return nil, ErrLocked.Wrap(fmt.Errorf("keychain entry is malformed"))
	}
	dek, err := unseal(kek, s)
	if err != nil {
		return nil, ErrLocked.Wrap(err)
	}
	return dek, nil
}

// seal wraps dek with kek. The slot kind and id are authenticated, so a slot cannot be
// moved to another entry.
func seal(kek, dek []byte, kind, id string) (slot, error) {
	aead, err := chacha20poly1305.NewX(kek)
	if err != nil {
		return slot{}, err
	}
	nonce := randomBytes(aead.NonceSize())
	return slot{
		Kind:       kind,
		ID:         id,
		Nonce:      nonce,
		Ciphertext: aead.Seal(nil, nonce, dek, additionalData(kind, id)),
	}, nil
}

func unseal(kek []byte, s slot) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(kek)
	if err != nil {
		return nil, err
	}
	if len(s.Nonce) != aead.NonceSize() {
		return nil, errors.New("bad nonce size")
	}
	dek, err := aead.Open(nil, s.Nonce, s.Ciphertext, additionalData(s.Kind, s.ID))
	if err != nil {
		return nil, err
	}
	if len(dek) != KeySize {
		return nil, errors.New("bad key size")
	}
	return dek, nil
}

func additionalData(kind, id string) []byte {
	return []byte("prive/slot/v1/" + kind + "/" + id)
}

func writeFileAtomic(path string, f slotFile) error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("keys: crypto/rand failed: %v", err)) // never happens on supported OSes
	}
	return b
}
