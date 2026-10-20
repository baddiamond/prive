//go:build smoke

package whatsapp

import (
	"bytes"
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/baddiamond/prive/internal/core"
	"github.com/baddiamond/prive/internal/store"
)

// TestSmokeQR connects to WhatsApp for real and waits for the first QR code.
// It needs network access and is excluded from normal runs. Run it manually with:
//
//	go test -tags smoke -run TestSmokeQR -v ./internal/engines/whatsapp
func TestSmokeQR(t *testing.T) {
	dir := t.TempDir()
	key := bytes.Repeat([]byte{1}, store.KeySize)
	s := newSession(core.Account{ID: "smoke", EngineID: ID}, core.SessionDeps{
		DataDir: dir,
		Logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
		OpenDB: func(ctx context.Context, name string) (*sql.DB, error) {
			return store.OpenDB(ctx, filepath.Join(dir, name+".db"), store.Options{Key: key})
		},
	})
	defer s.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	if err := s.Login(ctx, core.LoginQR); err != nil {
		t.Fatal(err)
	}
	for {
		select {
		case ev := <-s.SubscribeEvents():
			t.Logf("event %s", ev.EventName())
			if qr, ok := ev.(core.LoginQREvent); ok {
				t.Logf("got a QR code (%d chars), expires in %s", len(qr.Code), time.Until(qr.ExpiresAt).Round(time.Second))
				return
			}
			if f, ok := ev.(core.LoginFailedEvent); ok {
				t.Fatalf("login failed: %s", f.Code)
			}
		case <-ctx.Done():
			t.Fatal("no QR code before timeout")
		}
	}
}
