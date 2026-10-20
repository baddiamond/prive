package wordlists

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"testing"
)

// allowed is the alphabet of each list: plain a–z, plus accented vowels in Spanish (no ñ).
var allowed = map[string]*regexp.Regexp{
	"en": regexp.MustCompile(`^[a-z]{3,8}$`),
	"es": regexp.MustCompile(`^[a-záéíóúü]{3,8}$`),
}

// frozen pins every released list. A list must never change once people have written
// phrases with it; a new language adds a new file and a new entry here.
var frozen = map[string]string{
	"en": "",
	"es": "",
}

func TestListRules(t *testing.T) {
	seen := map[string]string{} // normalized word → language
	for _, l := range All() {
		re, ok := allowed[l.Language]
		if !ok {
			t.Fatalf("no alphabet rule for %q", l.Language)
		}
		if len(l.Words()) != Size {
			t.Errorf("%s: %d words, want %d", l.Language, len(l.Words()), Size)
		}
		for i, w := range l.Words() {
			if !re.MatchString(w) || len([]rune(w)) > 8 {
				t.Errorf("%s: %q breaks the alphabet or length rule", l.Language, w)
			}
			n := Normalize(w)
			if other, dup := seen[n]; dup {
				t.Errorf("%q is in both %s and %s", w, other, l.Language)
			}
			seen[n] = l.Language
			if got, ok := l.Index(w); !ok || got != i {
				t.Errorf("%s: Index(%q) = %d, %v; want %d", l.Language, w, got, ok, i)
			}
		}
	}
}

func TestIndexAcceptsPrefixesAndAccents(t *testing.T) {
	for _, l := range All() {
		for i, w := range l.Words() {
			n := Normalize(w)
			if len(n) > PrefixLength {
				if got, ok := l.Index(n[:PrefixLength]); !ok || got != i {
					t.Fatalf("%s: prefix of %q not found", l.Language, w)
				}
			}
		}
	}
	if _, ok := ForLanguage("en").Index("x"); ok {
		t.Error("a single letter must not match")
	}
	if ForLanguage("es-419").Language != "es" || ForLanguage("fr").Language != "en" {
		t.Error("ForLanguage must match the base language and fall back to English")
	}
}

func TestReleasedListsAreFrozen(t *testing.T) {
	for _, l := range All() {
		want := frozen[l.Language]
		if want == "" {
			continue // not released yet
		}
		text := english
		if l.Language == "es" {
			text = spanish
		}
		sum := sha256.Sum256([]byte(text))
		if got := hex.EncodeToString(sum[:]); got != want {
			t.Errorf("%s.txt changed (sha256 %s); released lists must never change", l.Language, got)
		}
	}
}
