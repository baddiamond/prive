package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/baddiamond/prive/internal/core"
	"github.com/baddiamond/prive/internal/engines"
	"github.com/baddiamond/prive/internal/store"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	st, err := store.Open(context.Background(), filepath.Join(t.TempDir(), store.FileName))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return NewApp(st, engines.NewRegistry(), t.TempDir(), slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestSettingsDefaults(t *testing.T) {
	app := newTestApp(t)
	s, err := app.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if s.Language != "" || s.Theme != ThemeSystem {
		t.Fatalf("unexpected defaults: %+v", s)
	}
}

func TestSetLanguage(t *testing.T) {
	app := newTestApp(t)
	for _, lang := range []string{"es", "pt-BR", "es-419", "zh-Hans-CN"} {
		if err := app.SetLanguage(lang); err != nil {
			t.Fatalf("SetLanguage(%q): %v", lang, err)
		}
		if s, _ := app.GetSettings(); s.Language != lang {
			t.Fatalf("got %q, want %q", s.Language, lang)
		}
	}
	if err := app.SetLanguage(""); err != nil {
		t.Fatal(err)
	}
	if s, _ := app.GetSettings(); s.Language != "" {
		t.Fatalf("empty language should reset to system, got %q", s.Language)
	}
	for _, bad := range []string{"Spanish", "e", "es_ES", "<script>"} {
		if err := app.SetLanguage(bad); !errors.Is(err, errInvalidSetting) {
			t.Errorf("SetLanguage(%q) = %v, want errInvalidSetting", bad, err)
		}
	}
}

func TestSetTheme(t *testing.T) {
	app := newTestApp(t)
	for _, theme := range []string{ThemeDark, ThemeLight, ThemeSystem} {
		if err := app.SetTheme(theme); err != nil {
			t.Fatal(err)
		}
		if s, _ := app.GetSettings(); s.Theme != theme {
			t.Fatalf("got %q, want %q", s.Theme, theme)
		}
	}
	if err := app.SetTheme("blue"); !errors.Is(err, errInvalidSetting) {
		t.Fatalf("got %v, want errInvalidSetting", err)
	}
}

func TestFormatError(t *testing.T) {
	format := formatError(slog.New(slog.NewTextHandler(io.Discard, nil)))

	got := format(core.ErrSendFailed.Wrap(errors.New("private detail")).With("reason", "timeout"))
	want := errorPayload{Code: core.CodeSendFailed, Params: map[string]string{"reason": "timeout"}}
	if gotJSON, wantJSON := mustJSON(t, got), mustJSON(t, want); gotJSON != wantJSON {
		t.Errorf("coded error = %s, want %s", gotJSON, wantJSON)
	}

	// Errors without a code never leak their text to the UI.
	if got := mustJSON(t, format(errors.New("private detail"))); got != `{"code":"errors.unknown"}` {
		t.Errorf("plain error = %s", got)
	}
}

func TestAppCodesHaveTranslations(t *testing.T) {
	raw, err := os.ReadFile("frontend/src/locales/en.json")
	if err != nil {
		t.Fatal(err)
	}
	var locale struct {
		Errors struct {
			Settings map[string]string `json:"settings"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(raw, &locale); err != nil {
		t.Fatal(err)
	}
	for _, e := range appErrorCodes {
		key := e.Code[len("errors.settings."):]
		if locale.Errors.Settings[key] == "" {
			t.Errorf("code %q has no translation in en.json", e.Code)
		}
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
