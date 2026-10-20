package main

import (
	"strings"

	"rsc.io/qr"
)

// QRCode is a QR code's module grid, for the frontend to draw in Privé's style.
type QRCode struct {
	// Size is the number of modules per side (without quiet zone).
	Size int `json:"size"`
	// Modules holds Size×Size characters, row by row: '1' for a dark module, '0' for light.
	Modules string `json:"modules"`
}

// RenderQR encodes a login code (e.g. WhatsApp's pairing payload) as a QR code. It uses the
// Q error correction (25% recoverable) so the logo drawn in the middle, which covers about
// 3% of the modules, never makes it unreadable, while the code stays coarse enough to scan.
// It runs in Go so the frontend needs no QR library; the payload never leaves the computer.
func (a *App) RenderQR(text string) (QRCode, error) {
	code, err := qr.Encode(text, qr.Q)
	if err != nil {
		return QRCode{}, errInvalidSetting.Wrap(err)
	}
	var b strings.Builder
	b.Grow(code.Size * code.Size)
	for y := 0; y < code.Size; y++ {
		for x := 0; x < code.Size; x++ {
			if code.Black(x, y) {
				b.WriteByte('1')
			} else {
				b.WriteByte('0')
			}
		}
	}
	return QRCode{Size: code.Size, Modules: b.String()}, nil
}
