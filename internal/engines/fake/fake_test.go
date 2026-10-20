package fake

import (
	"context"
	"testing"

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
