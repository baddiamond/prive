package main

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"rsc.io/qr"
)

func TestMaskRemote(t *testing.T) {
	if got := maskRemote("34600112233@s.whatsapp.net"); got != "•••• ••33" || strings.Contains(got, "600") {
		t.Errorf("maskRemote = %q", got)
	}
	if got := maskRemote(""); got != "—" {
		t.Errorf("empty = %q", got)
	}
}

func TestPrintQR(t *testing.T) {
	var b bytes.Buffer
	printQR(&b, "2@example,payload")
	lines := strings.Count(b.String(), "\n")
	if lines < 10 || !strings.Contains(b.String(), "▀") {
		t.Errorf("unexpected QR output (%d lines)", lines)
	}
}

// TestPrintQRMatchesCode rebuilds the module grid from the printed colors and compares it
// with the encoded QR code, so the terminal drawing is exactly scannable.
func TestPrintQRMatchesCode(t *testing.T) {
	const text = "2@example,payload,with,commas"
	var b bytes.Buffer
	printQR(&b, text)

	code, err := qr.Encode(text, qr.L)
	if err != nil {
		t.Fatal(err)
	}
	cell := regexp.MustCompile(`\x1b\[(\d+);(\d+)m▀`)
	rows := strings.Split(strings.TrimSuffix(b.String(), "\n"), "\n")
	const quiet = 2
	for ry, row := range rows {
		cells := cell.FindAllStringSubmatch(row, -1)
		for x, c := range cells {
			for half, darkCode := range map[int]string{0: "30", 1: "40"} {
				y := ry*2 + half - quiet
				mx := x - quiet
				want := mx >= 0 && y >= 0 && mx < code.Size && y < code.Size && code.Black(mx, y)
				got := c[1+half] == darkCode
				if got != want {
					t.Fatalf("module (%d,%d): printed dark=%v, want %v", mx, y, got, want)
				}
			}
		}
	}
}
