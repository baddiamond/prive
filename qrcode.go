package main

import (
	"strconv"
	"strings"

	"rsc.io/qr"
)

// QRCode is a QR code ready to draw as one SVG path: Path covers the dark modules in a
// Size×Size grid (without quiet zone).
type QRCode struct {
	Size int    `json:"size"`
	Path string `json:"path"`
}

// RenderQR encodes a login code (e.g. WhatsApp's pairing payload) as a QR code. It runs in
// Go so the frontend needs no QR library; the payload never leaves the computer.
func (a *App) RenderQR(text string) (QRCode, error) {
	code, err := qr.Encode(text, qr.M)
	if err != nil {
		return QRCode{}, errInvalidSetting.Wrap(err)
	}
	return QRCode{Size: code.Size, Path: qrPath(code)}, nil
}

// qrPath builds an SVG path with one rectangle per horizontal run of dark modules.
func qrPath(code *qr.Code) string {
	var b strings.Builder
	for y := 0; y < code.Size; y++ {
		for x := 0; x < code.Size; {
			if !code.Black(x, y) {
				x++
				continue
			}
			start := x
			for x < code.Size && code.Black(x, y) {
				x++
			}
			run := strconv.Itoa(x - start)
			b.WriteString("M" + strconv.Itoa(start) + " " + strconv.Itoa(y) + "h" + run + "v1h-" + run + "z")
		}
	}
	return b.String()
}
