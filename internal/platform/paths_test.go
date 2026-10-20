package platform

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestDataDir(t *testing.T) {
	home := func() (string, error) { return "/home/u", nil }
	config := func() (string, error) { return "/cfg", nil }
	noEnv := func(string) string { return "" }

	cases := []struct {
		name   string
		goos   string
		getenv func(string) string
		want   string
	}{
		{"macos", "darwin", noEnv, filepath.Join("/cfg", "Prive")},
		{"windows", "windows", noEnv, filepath.Join("/cfg", "Prive")},
		{"linux default", "linux", noEnv, filepath.Join("/home/u", ".local", "share", "prive")},
		{"linux xdg", "linux", func(k string) string {
			if k == "XDG_DATA_HOME" {
				return "/xdg"
			}
			return ""
		}, filepath.Join("/xdg", "prive")},
		{"linux relative xdg ignored", "linux", func(string) string { return "rel" },
			filepath.Join("/home/u", ".local", "share", "prive")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := dataDir(tc.goos, tc.getenv, home, config)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDataDirError(t *testing.T) {
	fail := func() (string, error) { return "", errors.New("boom") }
	if _, err := dataDir("darwin", func(string) string { return "" }, fail, fail); err == nil {
		t.Error("expected error when config dir is unavailable")
	}
}
