package keys

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFirstRunThenReopen(t *testing.T) {
	dir := t.TempDir()
	kc := NewMemoryKeychain()

	k1, created, err := Unlock(dir, kc)
	if err != nil || !created {
		t.Fatalf("first Unlock: created=%v err=%v", created, err)
	}
	k2, created, err := Unlock(dir, kc)
	if err != nil || created {
		t.Fatalf("second Unlock: created=%v err=%v", created, err)
	}
	if !bytes.Equal(k1.Derive("prive.db"), k2.Derive("prive.db")) {
		t.Fatal("the same data dir must yield the same keys")
	}

	info, err := os.Stat(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 && os.PathSeparator == '/' {
		t.Errorf("keys.json permissions = %v, want owner-only", perm)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, FileName))
	if bytes.Contains(raw, k1.dek) || strings.Contains(string(raw), "kek") {
		t.Error("keys.json must not contain the key in clear")
	}
}

func TestDeriveSeparatesPurposes(t *testing.T) {
	k, _, err := Unlock(t.TempDir(), NewMemoryKeychain())
	if err != nil {
		t.Fatal(err)
	}
	a, b := k.Derive("prive.db"), k.Derive("engine/whatsapp/acc-1/store")
	if len(a) != KeySize || bytes.Equal(a, b) || bytes.Equal(a, k.dek) {
		t.Error("derived keys must be 32 bytes, distinct per label and different from the DEK")
	}
	other, _, _ := Unlock(t.TempDir(), NewMemoryKeychain())
	if bytes.Equal(a, other.Derive("prive.db")) {
		t.Error("two installations must not share keys")
	}
}

func TestLockedWhenKeychainEntryIsGone(t *testing.T) {
	dir := t.TempDir()
	kc := NewMemoryKeychain()
	if _, _, err := Unlock(dir, kc); err != nil {
		t.Fatal(err)
	}
	// e.g. the OS was reinstalled and the data folder restored from a backup
	_, _, err := Unlock(dir, NewMemoryKeychain())
	if !errors.Is(err, ErrLocked) {
		t.Fatalf("Unlock with an empty keychain = %v, want ErrLocked", err)
	}
}

func TestTamperedSlotIsRejected(t *testing.T) {
	dir := t.TempDir()
	kc := NewMemoryKeychain()
	if _, _, err := Unlock(dir, kc); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, FileName)
	var f slotFile
	raw, _ := os.ReadFile(path)
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	f.Slots[0].Ciphertext[0] ^= 0xff
	if err := writeFileAtomic(path, f); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Unlock(dir, kc); !errors.Is(err, ErrLocked) {
		t.Fatalf("tampered slot = %v, want ErrLocked", err)
	}
}

func TestKeychainUnavailable(t *testing.T) {
	kc := NewMemoryKeychain()
	kc.Err = errors.New("no secret service")
	dir := t.TempDir()
	if _, _, err := Unlock(dir, kc); !errors.Is(err, ErrKeychainUnavailable) {
		t.Fatalf("Unlock = %v, want ErrKeychainUnavailable", err)
	}
	if _, err := os.Stat(filepath.Join(dir, FileName)); !errors.Is(err, os.ErrNotExist) {
		t.Error("keys.json must not be written when the keychain fails")
	}
}

func TestCorruptFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Unlock(dir, NewMemoryKeychain()); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("Unlock = %v, want ErrCorrupt", err)
	}
}

func TestForget(t *testing.T) {
	dir := t.TempDir()
	kc := NewMemoryKeychain()
	if _, _, err := Unlock(dir, kc); err != nil {
		t.Fatal(err)
	}
	if err := Forget(dir, kc); err != nil {
		t.Fatal(err)
	}
	if kc.Len() != 0 {
		t.Error("keychain entry not deleted")
	}
	if _, _, err := Unlock(dir, kc); !errors.Is(err, ErrLocked) {
		t.Errorf("Unlock after Forget = %v, want ErrLocked", err)
	}
}
