package keys

import (
	"crypto/sha256"
	"strconv"
	"strings"
	"unicode"

	"github.com/baddiamond/prive/internal/core"
	"github.com/baddiamond/prive/internal/crypto/wordlists"
)

// PhraseWords is the number of words in a recovery phrase.
const PhraseWords = 12

// phraseEntropy is the phrase's random part in bytes: 12 words × 11 bits = 128 bits of
// entropy plus a 4-bit checksum.
const phraseEntropy = 16

// Recovery phrase error codes. Params never contain the phrase itself.
var (
	// ErrPhraseWordCount means the phrase does not have 12 words. Param: count.
	ErrPhraseWordCount = core.NewError("errors.recovery.wordCount")
	// ErrPhraseUnknownWord means a word is in no list. Param: position (1-based).
	ErrPhraseUnknownWord = core.NewError("errors.recovery.unknownWord")
	// ErrPhraseMixedLists means the words come from different languages.
	ErrPhraseMixedLists = core.NewError("errors.recovery.mixedLists")
	// ErrPhraseChecksum means every word exists but the phrase is wrong: a word was
	// misread or two were swapped.
	ErrPhraseChecksum = core.NewError("errors.recovery.checksum")
	// ErrPhraseNoMatch means the phrase is valid but belongs to other data.
	ErrPhraseNoMatch = core.NewError("errors.recovery.noMatch")
)

// phraseCodes lists the phrase error codes, for Codes.
var phraseCodes = []*core.CodedError{
	ErrPhraseWordCount, ErrPhraseUnknownWord, ErrPhraseMixedLists, ErrPhraseChecksum, ErrPhraseNoMatch,
}

// encodePhrase turns 16 bytes of entropy into 12 words of list l.
func encodePhrase(entropy []byte, l *wordlists.List) []string {
	bits := append(append([]byte(nil), entropy...), checksum(entropy))
	words := make([]string, PhraseWords)
	for i := range words {
		words[i] = l.Word(readBits(bits, i*11, 11))
	}
	return words
}

// decodePhrase parses what the user typed: words separated by spaces, commas, numbers or
// new lines, in any case, with or without accents, full or by their first four letters.
// The list is detected from the words.
func decodePhrase(text string) ([]byte, error) {
	words := strings.FieldsFunc(text, func(r rune) bool { return !unicode.IsLetter(r) })
	if len(words) != PhraseWords {
		return nil, ErrPhraseWordCount.With("count", strconv.Itoa(len(words)))
	}

	lists, err := candidateLists(words)
	if err != nil {
		return nil, err
	}
	// A word typed by its first letters may exist in two lists ("fres" is "fresa" and
	// "fresh"); the checksum tells which list the phrase was written in.
	for _, list := range lists {
		bits := make([]byte, phraseEntropy+1)
		for i, w := range words {
			idx, _ := list.Index(w)
			writeBits(bits, i*11, 11, idx)
		}
		entropy := bits[:phraseEntropy]
		// The last word's low 4 bits are the checksum, in the high half of the last byte.
		if bits[phraseEntropy]>>4 == checksum(entropy)>>4 {
			return entropy, nil
		}
	}
	return nil, ErrPhraseChecksum
}

// candidateLists returns the lists that know every word. When none does, the error names
// the first word no list knows, or reports that the words mix languages.
func candidateLists(words []string) ([]*wordlists.List, error) {
	var found []*wordlists.List
	for _, l := range wordlists.All() {
		all := true
		for _, w := range words {
			if _, ok := l.Index(w); !ok {
				all = false
				break
			}
		}
		if all {
			found = append(found, l)
		}
	}
	if len(found) > 0 {
		return found, nil
	}
	for i, w := range words {
		known := false
		for _, l := range wordlists.All() {
			if _, ok := l.Index(w); ok {
				known = true
				break
			}
		}
		if !known {
			return nil, ErrPhraseUnknownWord.With("position", strconv.Itoa(i+1))
		}
	}
	return nil, ErrPhraseMixedLists
}

// checksum returns a byte whose high 4 bits are the phrase checksum.
func checksum(entropy []byte) byte {
	sum := sha256.Sum256(append([]byte("prive/recovery/v1/"), entropy...))
	return sum[0] & 0xF0
}

// readBits reads n bits starting at bit offset off (most significant bit first).
func readBits(b []byte, off, n int) int {
	v := 0
	for i := range n {
		bit := off + i
		v = v<<1 | int(b[bit/8]>>(7-bit%8)&1)
	}
	return v
}

// writeBits writes the n low bits of v starting at bit offset off.
func writeBits(b []byte, off, n, v int) {
	for i := range n {
		bit := off + i
		if v>>(n-1-i)&1 == 1 {
			b[bit/8] |= 1 << (7 - bit%8)
		}
	}
}
