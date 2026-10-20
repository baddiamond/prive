// Package whatsapp is the WhatsApp engine. It links accounts as a companion device with
// whatsmeow (go.mau.fi/whatsmeow), called directly in-process (ADR 0005).
//
// Each account has its own whatsmeow store, an encrypted SQLite database opened through
// SessionDeps.OpenDB, so any number of WhatsApp accounts run side by side.
package whatsapp

import (
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/store"

	"github.com/baddiamond/prive/internal/core"
	"github.com/baddiamond/prive/internal/engines"
)

// ID is the engine's manifest ID. It is stored with every WhatsApp account and never changes.
const ID = "whatsapp"

func init() {
	// How the account shows up on the phone, under Linked devices.
	store.SetOSInfo("Privé", [3]uint32{0, 1, 0})
	store.DeviceProps.PlatformType = waCompanionReg.DeviceProps_DESKTOP.Enum()

	engines.Register(New())
}

// Engine creates WhatsApp sessions. It holds no per-account state.
type Engine struct{}

// New returns the WhatsApp engine.
func New() *Engine { return &Engine{} }

// Manifest implements core.Engine. Capabilities grow as later phases implement them
// (chat list, history, sending).
func (e *Engine) Manifest() core.Manifest {
	return core.Manifest{
		ID:           ID,
		Name:         "WhatsApp",
		ShortName:    "WA",
		LoginMethods: []core.LoginMethod{core.LoginQR},
		Capabilities: core.Capabilities{SendText: true, ReadReceipts: true, HistoryOnDemand: true},
	}
}

// NewSession implements core.Engine. Nothing is opened until Connect or Login.
func (e *Engine) NewSession(account core.Account, deps core.SessionDeps) (core.Session, error) {
	return newSession(account, deps), nil
}
