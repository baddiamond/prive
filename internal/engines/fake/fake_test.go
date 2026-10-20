package fake

import (
	"context"
	"testing"
	"time"

	"github.com/baddiamond/prive/internal/core"
	"github.com/baddiamond/prive/internal/engines/enginetest"
)

func TestConformance(t *testing.T) {
	enginetest.Run(t, New(), enginetest.Options{
		Login: func(t *testing.T, s core.Session) {
			if err := s.Login(context.Background(), core.LoginQR); err != nil {
				t.Fatalf("Login: %v", err)
			}
		},
		ChatID: "chat-1",
	})
}

func TestDemoImportsHistory(t *testing.T) {
	e := NewDemo()
	dir := t.TempDir()
	s, err := e.NewSession(core.Account{ID: "demo"}, core.SessionDeps{DataDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Login(context.Background(), core.LoginQR); err != nil {
		t.Fatal(err)
	}
	chats := 0
	timeout := time.After(10 * time.Second)
	for chats < len(demoChats) {
		select {
		case ev := <-s.SubscribeEvents():
			if b, ok := ev.(core.HistoryBatchEvent); ok {
				chats += len(b.Chats)
			}
		case <-timeout:
			t.Fatalf("got %d demo chats", chats)
		}
	}
}
