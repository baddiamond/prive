//go:build devfake

package all

import (
	"github.com/baddiamond/prive/internal/engines"
	"github.com/baddiamond/prive/internal/engines/fake"
)

// With the devfake build tag (wails dev -tags devfake), the in-memory test engine is offered
// as a network: it links instantly, so the account UI can be developed without a phone.
// Release builds never include it.
func init() {
	engines.Register(fake.New())
}
