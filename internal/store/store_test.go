package store

import (
	"context"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func openTestStore(t *testing.T, path string) *Store {
	t.Helper()
	s, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestSettingsRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t, filepath.Join(t.TempDir(), FileName))

	if _, ok, err := s.Setting(ctx, SettingLanguage); err != nil || ok {
		t.Fatalf("missing setting: ok=%v err=%v", ok, err)
	}
	if err := s.SetSetting(ctx, SettingLanguage, "es"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetSetting(ctx, SettingLanguage, "en"); err != nil {
		t.Fatal(err)
	}
	if v, ok, err := s.Setting(ctx, SettingLanguage); err != nil || !ok || v != "en" {
		t.Fatalf("got %q ok=%v err=%v, want en", v, ok, err)
	}
	if err := s.DeleteSetting(ctx, SettingLanguage); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteSetting(ctx, SettingLanguage); err != nil {
		t.Fatalf("deleting a missing key: %v", err)
	}
	if _, ok, _ := s.Setting(ctx, SettingLanguage); ok {
		t.Fatal("setting still present after delete")
	}
}

func TestSettingsSurviveReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), FileName)

	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetSetting(ctx, SettingTheme, "dark"); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	// Reopening must not re-run migrations or lose data.
	s = openTestStore(t, path)
	if v, ok, err := s.Setting(ctx, SettingTheme); err != nil || !ok || v != "dark" {
		t.Fatalf("got %q ok=%v err=%v, want dark", v, ok, err)
	}
}

func TestLoadMigrations(t *testing.T) {
	ok := fstest.MapFS{
		"migrations/0002_b.sql": {Data: []byte("B")},
		"migrations/0001_a.sql": {Data: []byte("A")},
	}
	got, err := loadMigrations(ok)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].version != 1 || got[1].version != 2 {
		t.Fatalf("unexpected order: %+v", got)
	}

	bad := map[string]fstest.MapFS{
		"no number": {"migrations/init.sql": {Data: []byte("X")}},
		"duplicate": {
			"migrations/0001_a.sql": {Data: []byte("A")},
			"migrations/1_b.sql":    {Data: []byte("B")},
		},
	}
	for name, fsys := range bad {
		if _, err := loadMigrations(fsys); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}
