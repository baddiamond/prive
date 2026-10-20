package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/baddiamond/prive/internal/bootstrap"
	"github.com/baddiamond/prive/internal/core"
	"github.com/baddiamond/prive/internal/engines"
	"github.com/baddiamond/prive/internal/engines/fake"
	"github.com/baddiamond/prive/internal/keys"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	local, err := bootstrap.OpenAt(context.Background(), t.TempDir(), keys.NewMemoryKeychain())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = local.Close() })
	reg := engines.NewRegistry()
	if err := reg.Register(fake.New()); err != nil {
		t.Fatal(err)
	}
	return NewApp(local, reg, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestAccountsAPI(t *testing.T) {
	app := newTestApp(t)
	engs, err := app.ListEngines()
	if err != nil || len(engs) != 1 || engs[0].ID != fake.EngineID {
		t.Fatalf("ListEngines = %+v, %v", engs, err)
	}

	acc, err := app.AddAccount(fake.EngineID, "  Work  ", "amber")
	if err != nil {
		t.Fatal(err)
	}
	if acc.DisplayName != "Work" {
		t.Errorf("name not trimmed: %q", acc.DisplayName)
	}
	if err := app.Login(string(acc.ID), "qr"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.UpdateAccount(string(acc.ID), "Personal", "green"); err != nil {
		t.Fatal(err)
	}
	list, _ := app.ListAccounts()
	if len(list) != 1 || list[0].Account.DisplayName != "Personal" || list[0].Manifest.ShortName != "FK" {
		t.Errorf("ListAccounts = %+v", list)
	}
	if err := app.CancelLogin(string(acc.ID)); err != nil {
		t.Fatal(err)
	}
	if list, _ := app.ListAccounts(); len(list) != 0 {
		t.Errorf("account not removed: %+v", list)
	}

	bad := []struct {
		name, color string
		want        error
	}{
		{strings.Repeat("x", 41), "green", errInvalidAccountName},
		{"tab\there", "green", errInvalidAccountName},
		{"ok", "magenta", errInvalidAccountColor},
		{"ok", "#ff0000", errInvalidAccountColor},
	}
	for _, b := range bad {
		if _, err := app.AddAccount(fake.EngineID, b.name, b.color); !errors.Is(err, b.want) {
			t.Errorf("AddAccount(%q, %q) = %v, want %v", b.name, b.color, err, b.want)
		}
	}
	if _, err := app.AddAccount("nope", "x", "green"); !errors.Is(err, core.ErrEngineNotFound) {
		t.Errorf("unknown engine = %v", err)
	}
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

func TestOnboarding(t *testing.T) {
	app := newTestApp(t)
	o, err := app.GetOnboarding()
	if err != nil || o.Completed || o.Step != 1 || o.LegalAccepted {
		t.Fatalf("fresh onboarding = %+v, %v", o, err)
	}
	if err := app.SetOnboardingStep(4); err != nil {
		t.Fatal(err)
	}
	if err := app.SetOnboardingStep(9); !errors.Is(err, errInvalidSetting) {
		t.Errorf("SetOnboardingStep(9) = %v", err)
	}
	if err := app.AcceptLegal(); err != nil {
		t.Fatal(err)
	}
	if err := app.CompleteOnboarding(); err != nil {
		t.Fatal(err)
	}
	o, _ = app.GetOnboarding()
	if !o.Completed || o.Step != 4 || !o.LegalAccepted || o.LegalVersion != legalVersion {
		t.Errorf("after onboarding = %+v", o)
	}
}

func TestRemoveUnlinkedAccounts(t *testing.T) {
	app := newTestApp(t)
	linked, _ := app.AddAccount(fake.EngineID, "Linked", "green")
	if err := app.Login(string(linked.ID), "qr"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.AddAccount(fake.EngineID, "Abandoned", "amber"); err != nil {
		t.Fatal(err)
	}
	// Wait until the login is recorded (the fake engine reports it asynchronously).
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		list, _ := app.ListAccounts()
		if list[0].Account.RemoteID != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	app.removeUnlinkedAccounts()
	list, _ := app.ListAccounts()
	if len(list) != 1 || list[0].Account.ID != linked.ID {
		t.Errorf("accounts = %+v, want only the linked one", list)
	}
}

func TestRenderQR(t *testing.T) {
	app := newTestApp(t)
	q, err := app.RenderQR("2@payload,for,a,test")
	if err != nil {
		t.Fatal(err)
	}
	if q.Size < 21 || len(q.Modules) != q.Size*q.Size || strings.Trim(q.Modules, "01") != "" {
		t.Fatalf("unexpected QR: size %d, %d modules", q.Size, len(q.Modules))
	}
	// Every QR code starts with a finder pattern: a dark 7-module row in the top-left corner.
	if q.Modules[:7] != "1111111" || q.Modules[7] != '0' {
		t.Errorf("first row starts %q, want a finder pattern", q.Modules[:8])
	}
}

func TestChatsAPI(t *testing.T) {
	app := newTestApp(t)
	acc, _ := app.AddAccount(fake.EngineID, "A", "green")
	if err := app.Login(string(acc.ID), "qr"); err != nil {
		t.Fatal(err)
	}
	chats, err := app.ListChats("")
	if err != nil || chats == nil || len(chats) != 0 {
		t.Fatalf("ListChats = %v, %v", chats, err)
	}
	if _, err := app.ListChats("unknown"); !errors.Is(err, core.ErrAccountNotFound) {
		t.Errorf("unknown account = %v", err)
	}
	sum, err := app.UnreadSummary()
	if err != nil || sum.Total != 0 {
		t.Errorf("UnreadSummary = %+v, %v", sum, err)
	}
}
