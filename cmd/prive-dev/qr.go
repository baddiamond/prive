package main

import (
	"fmt"
	"io"
	"strings"

	"rsc.io/qr"
)

// printQR draws a QR code with half-block characters, two modules per text row, dark on
// light regardless of the terminal theme, with the quiet zone scanners need.
func printQR(w io.Writer, text string) {
	code, err := qr.Encode(text, qr.L)
	if err != nil {
		fmt.Fprintln(w, "cannot draw QR code:", err)
		return
	}
	const quiet = 2
	size := code.Size
	dark := func(x, y int) bool {
		x, y = x-quiet, y-quiet
		return x >= 0 && y >= 0 && x < size && y < size && code.Black(x, y)
	}
	var b strings.Builder
	total := size + 2*quiet
	for y := 0; y < total; y += 2 {
		for x := 0; x < total; x++ {
			top, bottom := dark(x, y), dark(x, y+1)
			fg, bg := 97, 107 // bright white
			if top {
				fg = 30
			}
			if bottom {
				bg = 40
			}
			fmt.Fprintf(&b, "\033[%d;%dm▀", fg, bg)
		}
		b.WriteString("\033[0m\n")
	}
	_, _ = io.WriteString(w, b.String())
}
