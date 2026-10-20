// Package wordlists holds Privé's recovery phrase word lists (ADR 0007).
//
// Each list has exactly 2,048 words, so one word carries 11 bits. The lists are Privé's own
// work, released under the MIT license with the rest of the repository; they are not the
// BIP39 lists. Rules every list follows (checked by the tests):
//
//   - 3 to 8 letters, lowercase; Spanish may use accented vowels, never ñ;
//   - unique by their first four letters once accents are removed, so four letters are
//     always enough to tell a word apart;
//   - no word appears in two lists, so a phrase reveals its own language.
//
// A list is frozen once released: changing it would make existing phrases unreadable.
// A new language adds a new file and never changes an existing one.
package wordlists

import (
	_ "embed"
	"fmt"
	"strings"
)

// Size is the number of words in every list.
const Size = 2048

// PrefixLength is how many letters identify a word.
const PrefixLength = 4

//go:embed en.txt
var english string

//go:embed es.txt
var spanish string

// List is one language's word list.
type List struct {
	// Language is a BCP 47 base language tag, e.g. "en".
	Language string
	words    []string
	// index maps a normalized word, and the normalized first four letters of every word
	// longer than four letters, to the word's position.
	index map[string]int
}

var lists = []*List{
	mustParse("en", english),
	mustParse("es", spanish),
}

// All returns every shipped list, English first.
func All() []*List {
	return append([]*List(nil), lists...)
}

// ForLanguage returns the list for a language tag such as "es" or "es-419", falling back
// to English.
func ForLanguage(tag string) *List {
	base := strings.ToLower(strings.SplitN(tag, "-", 2)[0])
	for _, l := range lists {
		if l.Language == base {
			return l
		}
	}
	return lists[0]
}

// Word returns the word at index i (0 ≤ i < Size).
func (l *List) Word(i int) string {
	return l.words[i]
}

// Words returns a copy of the list.
func (l *List) Words() []string {
	return append([]string(nil), l.words...)
}

// Index finds a word typed by the user. Case and accents are ignored, and the first four
// letters are enough. It reports false for anything that is not in the list.
func (l *List) Index(typed string) (int, bool) {
	n := Normalize(typed)
	if i, ok := l.index[n]; ok {
		return i, true
	}
	if len(n) > PrefixLength {
		if i, ok := l.index[n[:PrefixLength]]; ok {
			return i, true
		}
	}
	return 0, false
}

// Normalize lowercases a word and removes Spanish accents, so "Árbol" and "arbol" match.
// The result is plain ASCII for every word of every list.
func Normalize(word string) string {
	return accents.Replace(strings.ToLower(strings.TrimSpace(word)))
}

var accents = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u")

func mustParse(lang, text string) *List {
	l, err := parse(lang, text)
	if err != nil {
		panic(err) // the lists are embedded and tested; this only fires on a broken build
	}
	return l
}

func parse(lang, text string) (*List, error) {
	words := strings.Fields(text)
	if len(words) != Size {
		return nil, fmt.Errorf("wordlists: %s has %d words, want %d", lang, len(words), Size)
	}
	l := &List{Language: lang, words: words, index: make(map[string]int, 2*Size)}
	for i, w := range words {
		n := Normalize(w)
		keys := []string{n}
		if len(n) > PrefixLength {
			keys = append(keys, n[:PrefixLength])
		}
		for _, k := range keys {
			if j, dup := l.index[k]; dup && j != i {
				return nil, fmt.Errorf("wordlists: %s: %q and %q share %q", lang, words[j], w, k)
			}
			l.index[k] = i
		}
	}
	return l, nil
}
