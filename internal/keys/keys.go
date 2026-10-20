// Package keys manages Privé's data encryption key (DEK), as decided in ADR 0007.
//
// The DEK is 256 random bits generated on first run. It never touches the disk in clear:
// it is wrapped (XChaCha20-Poly1305) into "slots" stored in <DataDir>/keys.json, and any slot
// can unwrap it:
//
//   - the system keychain slot opens the app without a prompt;
//   - the recovery phrase slot is unwrapped by 12 words the user writes down. The phrase is
//     derived from the DEK, so it can be shown again at any time while the app is unlocked,
//     and it unlocks the data on a new computer or when the keychain entry is gone.
//
// The app password slot arrives later.
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
	"github.com/baddiamond/prive/internal/crypto/wordlists"
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
	return append([]*core.CodedError{ErrLocked, ErrKeychainUnavailable, ErrCorrupt}, phraseCodes...)
}

const (
	fileVersion  = 1
	kindKeychain = "keychain"
	kindRecovery = "recovery"
	// recoveryID names the first recovery slot. Regenerating the phrase would add
	// "recovery-2" and remove the old slot.
	recoveryID = "recovery-1"
	// keychainService names Privé's entries in the OS credential store.
	keychainService = "io.github.baddiamond.prive"
	hkdfSalt        = "prive/keys/v1"
)

// Keys holds the unlocked DEK.
type Keys struct {
	dek []byte
	// recoveryID is the ID of the recovery slot the phrase belongs to.
	recoveryID string
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
		k, err := ensureRecoverySlot(path, f, dek)
		return k, false, err
	}
	return nil, false, ErrLocked
}

// Recover unlocks dataDir with the recovery phrase, e.g. on a new computer or after the
// keychain entry was lost, and stores a new keychain slot so the next start needs no phrase.
func Recover(dataDir, phrase string, kc Keychain) (*Keys, error) {
	path := filepath.Join(dataDir, FileName)
	f, err := readSlotFile(path)
	if err != nil {
		return nil, err
	}
	entropy, err := decodePhrase(phrase)
	if err != nil {
		return nil, err
	}

	kek := phraseKEK(entropy)
	var dek []byte
	var id string
	for _, s := range f.Slots {
		if s.Kind != kindRecovery {
			continue
		}
		if d, err := unseal(kek, s); err == nil {
			dek, id = d, s.ID
			break
		}
	}
	if dek == nil {
		return nil, ErrPhraseNoMatch
	}

	// Replace the keychain slots: their entries are gone or belong to another computer.
	var keep []slot
	var stale []string
	for _, s := range f.Slots {
		if s.Kind == kindKeychain {
			stale = append(stale, s.ID)
			continue
		}
		keep = append(keep, s)
	}
	ks, newID, err := newKeychainSlot(dek, kc)
	if err != nil {
		return nil, err
	}
	f.Slots = append([]slot{ks}, keep...)
	if err := writeFileAtomic(path, f); err != nil {
		_ = kc.Delete(keychainService, newID)
		return nil, ErrCorrupt.Wrap(err)
	}
	for _, old := range stale {
		_ = kc.Delete(keychainService, old) // best effort: usually already gone
	}
	return &Keys{dek: dek, recoveryID: id}, nil
}

// RecoveryPhrase returns the 12 words that unlock this data, in the list for language
// (English when there is no list for it). Every language spells the same key.
func (k *Keys) RecoveryPhrase(language string) []string {
	return encodePhrase(k.phraseEntropy(), wordlists.ForLanguage(language))
}

// PhraseWordMatches reports whether typed is word number position (0-based) of the
// recovery phrase, in any language. Used to check that the user wrote the phrase down.
func (k *Keys) PhraseWordMatches(position int, typed string) bool {
	if position < 0 || position >= PhraseWords {
		return false
	}
	en := k.RecoveryPhrase("en")
	want, _ := wordlists.ForLanguage("en").Index(en[position])
	// Every list spells the same bits, so position i has the same index in each list.
	for _, l := range wordlists.All() {
		if got, ok := l.Index(typed); ok && got == want {
			return true
		}
	}
	return false
}

func (k *Keys) phraseEntropy() []byte {
	return hkdfBytes(k.dek, "recovery-phrase/"+k.recoveryID, phraseEntropy)
}

// phraseKEK derives the key that wraps the DEK in the recovery slot.
func phraseKEK(entropy []byte) []byte {
	return hkdfBytes(entropy, "recovery-slot", KeySize)
}

// ensureRecoverySlot adds the recovery slot to data created before it existed.
func ensureRecoverySlot(path string, f slotFile, dek []byte) (*Keys, error) {
	for _, s := range f.Slots {
		if s.Kind == kindRecovery {
			return &Keys{dek: dek, recoveryID: s.ID}, nil
		}
	}
	k := &Keys{dek: dek, recoveryID: recoveryID}
	rs, err := seal(phraseKEK(k.phraseEntropy()), dek, kindRecovery, recoveryID)
	if err != nil {
		return nil, ErrCorrupt.Wrap(err)
	}
	f.Slots = append(f.Slots, rs)
	if err := writeFileAtomic(path, f); err != nil {
		return nil, ErrCorrupt.Wrap(err)
	}
	return k, nil
}

func readSlotFile(path string) (slotFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return slotFile{}, ErrCorrupt.Wrap(err)
	}
	var f slotFile
	if err := json.Unmarshal(raw, &f); err != nil || f.Version != fileVersion {
		return slotFile{}, ErrCorrupt.Wrap(fmt.Errorf("parse %s: %v", FileName, err))
	}
	return f, nil
}

// Derive returns a 32-byte key for one purpose, e.g. "prive.db" or "engine/whatsapp/<accountId>/store".
// The same label always yields the same key.
func (k *Keys) Derive(label string) []byte {
	return hkdfBytes(k.dek, label, KeySize)
}

func hkdfBytes(secret []byte, label string, n int) []byte {
	out := make([]byte, n)
	r := hkdf.New(sha256.New, secret, []byte(hkdfSalt), []byte(label))
	if _, err := io.ReadFull(r, out); err != nil {
		panic(fmt.Sprintf("keys: hkdf: %v", err)) // cannot happen for short outputs
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

// create generates the DEK, its keychain slot and its recovery slot. The keychain entry is
// written first and removed again if keys.json cannot be saved, so no orphan entry is left
// behind.
func create(path string, kc Keychain) (*Keys, error) {
	dek := randomBytes(KeySize)
	ks, id, err := newKeychainSlot(dek, kc)
	if err != nil {
		return nil, err
	}
	k := &Keys{dek: dek, recoveryID: recoveryID}
	rs, err := seal(phraseKEK(k.phraseEntropy()), dek, kindRecovery, recoveryID)
	if err == nil {
		err = writeFileAtomic(path, slotFile{Version: fileVersion, Slots: []slot{ks, rs}})
	}
	if err != nil {
		_ = kc.Delete(keychainService, id)
		return nil, ErrCorrupt.Wrap(err)
	}
	return k, nil
}

// newKeychainSlot stores a new random wrapping key in the keychain and wraps dek with it.
func newKeychainSlot(dek []byte, kc Keychain) (slot, string, error) {
	kek := randomBytes(KeySize)
	id := "data-key-" + hex.EncodeToString(randomBytes(8))
	if err := kc.Set(keychainService, id, base64.StdEncoding.EncodeToString(kek)); err != nil {
		return slot{}, "", ErrKeychainUnavailable.Wrap(err)
	}
	s, err := seal(kek, dek, kindKeychain, id)
	if err != nil {
		_ = kc.Delete(keychainService, id)
		return slot{}, "", ErrCorrupt.Wrap(err)
	}
	return s, id, nil
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
