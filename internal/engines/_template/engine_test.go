package example

import (
	"testing"

	"github.com/baddiamond/prive/internal/engines/enginetest"
)

func TestConformance(t *testing.T) {
	enginetest.Run(t, New(), enginetest.Options{
		// TODO: provide Login (driving a fake or recorded server, never the real
		// network) and ChatID so the logged-in checks run too.
	})
}
