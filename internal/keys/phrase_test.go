package keys

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/baddiamond/prive/internal/core"
	"github.com/baddiamond/prive/internal/crypto/wordlists"
)

func TestPhraseRoundTrip(t *testing.T) {
	for _, l := range wordlists.All() {
		for range 500 {
			entropy := randomBytes(phraseEntropy)
			words := encodePhrase(entropy, l)
			if len(words) != PhraseWords {
				t.Fatalf("%s: %d words", l.Language, len(words))
			}
			got, err := decodePhrase(strings.Join(words, " "))
			if err != nil || !bytes.Equal(got, entropy) {
				t.Fatalf("%s: decode(%v) = %x, %v; want %x", l.Language, words, got, err, entropy)
			}
			// Four letters per word, which may also be the start of a word in another list.
			short := make([]string, len(words))
			for i, w := range words {
				short[i] = wordlists.Normalize(w)
				if len(short[i]) > wordlists.PrefixLength {
					short[i] = short[i][:wordlists.PrefixLength]
				}
			}
			got, err = decodePhrase(strings.Join(short, " "))
			if err != nil || !bytes.Equal(got, entropy) {
				t.Fatalf("%s: decode(%v) = %x, %v; want %x", l.Language, short, got, err, entropy)
			}
		}
	}
}

func TestPhraseInputIsForgiving(t *testing.T) {
	entropy := bytes.Repeat([]byte{0x5a}, phraseEntropy)
	words := encodePhrase(entropy, wordlists.ForLanguage("es"))

	// Numbered, upper case, extra punctuation, first four letters only.
	var b strings.Builder
	for i, w := range words {
		typed := strings.ToUpper(w)
		if i%2 == 0 && len([]rune(typed)) > wordlists.PrefixLength {
			typed = string([]rune(typed)[:wordlists.PrefixLength])
		}
		b.WriteString(" " + string(rune('1'+i%9)) + ". " + typed + ",\n")
	}
	got, err := decodePhrase(b.String())
	if err != nil || !bytes.Equal(got, entropy) {
		t.Fatalf("decode(%q) = %x, %v", b.String(), got, err)
	}
	if _, err := decodePhrase(wordlists.Normalize(strings.Join(words, " "))); err != nil {
		t.Errorf("words without accents must be accepted: %v", err)
	}
}

func TestPhraseErrors(t *testing.T) {
	entropy := bytes.Repeat([]byte{0xa7}, phraseEntropy)
	en := encodePhrase(entropy, wordlists.ForLanguage("en"))
	es := encodePhrase(entropy, wordlists.ForLanguage("es"))

	join := func(w []string) string { return strings.Join(w, " ") }
	replace := func(w []string, i int, s string) []string {
		c := append([]string(nil), w...)
		c[i] = s
		return c
	}

	if _, err := decodePhrase(join(en[:11])); !errors.Is(err, ErrPhraseWordCount) || paramOf(err, "count") != "11" {
		t.Errorf("11 words = %v", err)
	}
	if _, err := decodePhrase(join(replace(en, 4, "zzzzzz"))); !errors.Is(err, ErrPhraseUnknownWord) ||
		paramOf(err, "position") != "5" {
		t.Errorf("unknown word = %v", err)
	}
	if _, err := decodePhrase(join(replace(en, 3, es[3]))); !errors.Is(err, ErrPhraseMixedLists) {
		t.Errorf("mixed lists = %v", err)
	}

	// Swapping two different words breaks the checksum or changes the key; either way the
	// phrase no longer decodes to the same entropy. Most swaps are caught by the checksum.
	caught := 0
	for i := 1; i < PhraseWords; i++ {
		if en[0] == en[i] {
			continue
		}
		swapped := replace(replace(en, 0, en[i]), i, en[0])
		got, err := decodePhrase(join(swapped))
		if errors.Is(err, ErrPhraseChecksum) {
			caught++
		} else if err != nil || bytes.Equal(got, entropy) {
			t.Fatalf("swap 0↔%d = %x, %v", i, got, err)
		}
	}
	if caught < 6 {
		t.Errorf("checksum caught only %d of 11 swaps", caught)
	}
}

func TestRecoverWithPhrase(t *testing.T) {
	dir := t.TempDir()
	kc := NewMemoryKeychain()
	k, _, err := Unlock(dir, kc)
	if err != nil {
		t.Fatal(err)
	}
	// The same key in every language.
	en, es := k.RecoveryPhrase("en"), k.RecoveryPhrase("es-419")
	if strings.Join(en, " ") == strings.Join(es, " ") {
		t.Fatal("languages must use their own lists")
	}

	// A new computer: the keychain is empty.
	fresh := NewMemoryKeychain()
	if _, _, err := Unlock(dir, fresh); !errors.Is(err, ErrLocked) {
		t.Fatalf("Unlock = %v, want ErrLocked", err)
	}
	r, err := Recover(dir, strings.Join(es, " "), fresh)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(r.Derive("prive.db"), k.Derive("prive.db")) {
		t.Fatal("recovered key differs")
	}
	if strings.Join(r.RecoveryPhrase("en"), " ") != strings.Join(en, " ") {
		t.Error("the phrase must stay the same after recovery")
	}
	// The next start needs no phrase.
	again, _, err := Unlock(dir, fresh)
	if err != nil || !bytes.Equal(again.Derive("prive.db"), k.Derive("prive.db")) {
		t.Fatalf("Unlock after Recover = %v", err)
	}
	if fresh.Len() != 1 {
		t.Errorf("keychain has %d entries, want 1", fresh.Len())
	}
	if slots := readSlots(t, dir); len(slots) != 2 {
		t.Errorf("keys.json has %d slots, want keychain + recovery", len(slots))
	}
}

func TestRecoverRejectsAnotherInstallationsPhrase(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := Unlock(dir, NewMemoryKeychain()); err != nil {
		t.Fatal(err)
	}
	other, _, err := Unlock(t.TempDir(), NewMemoryKeychain())
	if err != nil {
		t.Fatal(err)
	}
	_, err = Recover(dir, strings.Join(other.RecoveryPhrase("en"), " "), NewMemoryKeychain())
	if !errors.Is(err, ErrPhraseNoMatch) {
		t.Fatalf("Recover = %v, want ErrPhraseNoMatch", err)
	}
}

func TestOlderDataGetsARecoverySlot(t *testing.T) {
	dir := t.TempDir()
	kc := NewMemoryKeychain()
	k, _, err := Unlock(dir, kc)
	if err != nil {
		t.Fatal(err)
	}
	// Data created before phase 3b had only the keychain slot.
	path := filepath.Join(dir, FileName)
	slots := readSlots(t, dir)
	if err := writeFileAtomic(path, slotFile{Version: fileVersion, Slots: slots[:1]}); err != nil {
		t.Fatal(err)
	}
	k2, _, err := Unlock(dir, kc)
	if err != nil {
		t.Fatal(err)
	}
	if got := readSlots(t, dir); len(got) != 2 || got[1].Kind != kindRecovery {
		t.Fatalf("slots after Unlock = %+v", got)
	}
	if strings.Join(k.RecoveryPhrase("en"), " ") != strings.Join(k2.RecoveryPhrase("en"), " ") {
		t.Error("the phrase must not change when the slot is added again")
	}
	if _, err := Recover(dir, strings.Join(k2.RecoveryPhrase("en"), " "), NewMemoryKeychain()); err != nil {
		t.Errorf("Recover = %v", err)
	}
}

func TestPhraseWordMatches(t *testing.T) {
	k, _, err := Unlock(t.TempDir(), NewMemoryKeychain())
	if err != nil {
		t.Fatal(err)
	}
	en, es := k.RecoveryPhrase("en"), k.RecoveryPhrase("es")
	if !k.PhraseWordMatches(2, strings.ToUpper(en[2])) || !k.PhraseWordMatches(2, es[2]) {
		t.Error("the right word must match, in any case and language")
	}
	if k.PhraseWordMatches(2, en[3]) && en[2] != en[3] {
		t.Error("another word must not match")
	}
	if k.PhraseWordMatches(12, en[0]) || k.PhraseWordMatches(-1, en[0]) || k.PhraseWordMatches(0, "") {
		t.Error("out of range positions and empty words never match")
	}
}

func readSlots(t *testing.T, dir string) []slot {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	var f slotFile
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	return f.Slots
}

func paramOf(err error, key string) string {
	var ce *core.CodedError
	if errors.As(err, &ce) {
		return ce.Params[key]
	}
	return ""
}
