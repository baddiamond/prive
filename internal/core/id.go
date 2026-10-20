package core

import (
	"crypto/rand"
	"fmt"
)

// NewAccountID returns a random (version 4) UUID.
func NewAccountID() AccountID {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("core: crypto/rand failed: %v", err)) // never happens on supported OSes
	}
	b[6] = b[6]&0x0f | 0x40 // version 4
	b[8] = b[8]&0x3f | 0x80 // RFC 4122 variant
	return AccountID(fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]))
}
