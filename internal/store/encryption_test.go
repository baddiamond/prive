package store

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/baddiamond/prive/internal/core"
)

func testKey(b byte) []byte { return bytes.Repeat([]byte{b}, KeySize) }

func TestDatabaseIsEncryptedAtRest(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), FileName)
	s, err := Open(ctx, path, Options{Key: testKey(1)})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetSetting(ctx, "secret-marker", "visible-if-plaintext"); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	for _, f := range []string{path, path + "-wal"} {
		raw, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if bytes.HasPrefix(raw, plaintextHeader) || bytes.Contains(raw, []byte("visible-if-plaintext")) {
			t.Errorf("%s contains plaintext", filepath.Base(f))
		}
	}

	if s, err := Open(ctx, path, Options{Key: testKey(2)}); err == nil {
		_ = s.Close()
		t.Error("opened with the wrong key")
	}

	s, err = Open(ctx, path, Options{Key: testKey(1)})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if v, ok, err := s.Setting(ctx, "secret-marker"); err != nil || !ok || v != "visible-if-plaintext" {
		t.Errorf("reopen: got %q ok=%v err=%v", v, ok, err)
	}
}

func TestLegacyPlaintextDatabaseIsEncryptedInPlace(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), FileName)

	legacy, err := Open(ctx, path, Options{}) // a development build without encryption
	if err != nil {
		t.Fatal(err)
	}
	if err := legacy.SetSetting(ctx, SettingTheme, "dark"); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}

	s, err := Open(ctx, path, Options{Key: testKey(7)})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if v, _, _ := s.Setting(ctx, SettingTheme); v != "dark" {
		t.Errorf("setting lost during encryption: %q", v)
	}
	if clear, _ := isPlaintext(path); clear {
		t.Error("database still in clear")
	}
}

func TestFullTextSearchIndex(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	addAccount(t, s, "a", 0)
	addAccount(t, s, "b", 1)

	msg := func(acc core.AccountID, chat core.ChatID, id core.MessageID, text string) {
		t.Helper()
		if err := s.UpsertMessage(ctx, core.Message{AccountID: acc, ChatID: chat, ID: id, Text: text, Timestamp: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}
	search := func(q string) []string {
		t.Helper()
		rows, err := s.db.QueryContext(ctx, `
			SELECT m.id FROM messages_fts f JOIN messages m ON m.rowid = f.rowid
			WHERE messages_fts MATCH ? ORDER BY m.id`, q)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var ids []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				t.Fatal(err)
			}
			ids = append(ids, id)
		}
		return ids
	}

	if err := s.UpsertChat(ctx, core.Chat{AccountID: "a", ID: "secret"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE chats SET search_excluded = 1 WHERE id = 'secret'`); err != nil {
		t.Fatal(err)
	}

	msg("a", "c1", "m1", "Me encanta esta canción")
	msg("b", "c2", "m2", "Otra CANCION distinta")
	msg("a", "secret", "m3", "canción privada")

	if got := search("cancion"); len(got) != 2 || got[0] != "m1" || got[1] != "m2" {
		t.Errorf("search(cancion) = %v, want [m1 m2] (diacritics ignored, excluded chat skipped)", got)
	}

	msg("a", "c1", "m1", "texto cambiado") // edits are re-indexed
	if got := search("cancion"); len(got) != 1 || got[0] != "m2" {
		t.Errorf("after edit: %v, want [m2]", got)
	}
	if got := search("cambiado"); len(got) != 1 {
		t.Errorf("edited text not indexed: %v", got)
	}

	if err := s.DeleteAccount(ctx, "b"); err != nil { // cascades to messages and their index rows
		t.Fatal(err)
	}
	if got := search("cancion"); len(got) != 0 {
		t.Errorf("deleted account still searchable: %v", got)
	}
}
