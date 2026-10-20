package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/baddiamond/prive/internal/core"
	"github.com/baddiamond/prive/internal/engines"
	"github.com/baddiamond/prive/internal/keys"
	"github.com/baddiamond/prive/internal/store"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	k, _, err := keys.Unlock(dir, keys.NewMemoryKeychain())
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(context.Background(), filepath.Join(dir, store.FileName), store.Options{Key: k.Derive(store.FileName)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return NewApp(st, engines.NewRegistry(), dir, k, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestFailedAppReportsTheCause(t *testing.T) {
	app := NewFailedApp(keys.ErrLocked, t.TempDir(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if _, err := app.GetSettings(); !errors.Is(err, keys.ErrLocked) {
		t.Errorf("GetSettings = %v, want ErrLocked", err)
	}
	if err := app.SetTheme(ThemeDark); !errors.Is(err, keys.ErrLocked) {
		t.Errorf("SetTheme = %v, want ErrLocked", err)
	}
	app.shutdown(context.Background()) // must not panic without a manager
}

func TestAppInfo(t *testing.T) {
	info := newTestApp(t).GetAppInfo()
	if info.Version == "" || info.DataDir == "" {
		t.Errorf("incomplete app info: %+v", info)
	}
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
	want := mustJSON(t, errorPayload{Code: core.CodeSendFailed, Params: map[string]string{"reason": "timeout"}})
	if got != want {
		t.Errorf("coded error = %v, want %s", got, want)
	}

	// Errors without a code never leak their text to the UI.
	if got := format(errors.New("private detail")); got != `{"code":"errors.unknown"}` {
		t.Errorf("plain error = %v", got)
	}
}

func TestAppCodesHaveTranslations(t *testing.T) {
	raw, err := os.ReadFile("frontend/src/locales/en.json")
	if err != nil {
		t.Fatal(err)
	}
	var locale map[string]any
	if err := json.Unmarshal(raw, &locale); err != nil {
		t.Fatal(err)
	}
	codes := append(append([]*core.CodedError{}, appErrorCodes...), keys.Codes()...)
	for _, e := range codes {
		if !hasTranslation(locale, e.Code) {
			t.Errorf("code %q has no translation in en.json", e.Code)
		}
	}
}

func hasTranslation(tree map[string]any, key string) bool {
	head, rest, nested := strings.Cut(key, ".")
	v, ok := tree[head]
	if !ok {
		return false
	}
	if !nested {
		_, isString := v.(string)
		return isString
	}
	sub, ok := v.(map[string]any)
	return ok && hasTranslation(sub, rest)
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
