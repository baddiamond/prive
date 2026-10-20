// Package example is the skeleton for a new engine. Copy this folder to
// internal/engines/<id>/, rename the package and the manifest, and replace every TODO.
// See docs/ADDING_AN_ENGINE.md.
//
// The Go toolchain ignores folders starting with "_", so this copy is never built
// into the app. CI compiles a copy and runs the conformance suite on it.
package example

import (
	"context"
	"sync"

	"github.com/baddiamond/prive/internal/core"
)

// TODO: register the engine once it is ready (and add it to internal/engines/all):
//
//	func init() { engines.Register(New()) }

// Engine is the factory for this network's sessions. It must hold no per-account state:
// many accounts of the same network run at once.
type Engine struct{}

// New returns the engine.
func New() *Engine { return &Engine{} }

// Manifest implements core.Engine. Declare only the capabilities you implement:
// the UI trusts them.
func (e *Engine) Manifest() core.Manifest {
	return core.Manifest{
		ID:           "example",                        // TODO: lowercase, stable forever
		Name:         "Example",                        // TODO: brand name, not translated
		ShortName:    "EX",                             // TODO: 2-3 letter tag for the rail
		LoginMethods: []core.LoginMethod{core.LoginQR}, // TODO
		Capabilities: core.Capabilities{},              // TODO
	}
}

// NewSession implements core.Engine. It must not connect yet.
func (e *Engine) NewSession(account core.Account, deps core.SessionDeps) (core.Session, error) {
	return &Session{
		account: account,
		deps:    deps,
		events:  core.NewEventQueue(),
	}, nil
}

// Session is one account on this network.
type Session struct {
	account core.Account
	deps    core.SessionDeps
	// events never blocks the network library: push from any goroutine.
	events *core.EventQueue

	mu sync.Mutex
	// TODO: the network client, kept under mu.
}

// Connect implements core.Session: resume the login stored in s.deps.DataDir.
func (s *Session) Connect(ctx context.Context) error {
	// TODO: load stored credentials from s.deps.DataDir and connect. Push
	// core.AccountStateEvent on every change and reconnect with backoff on your own.
	return core.ErrNotLoggedIn
}

// Login implements core.Session: start the interactive flow and return.
func (s *Session) Login(ctx context.Context, method core.LoginMethod) error {
	if !New().Manifest().SupportsLogin(method) {
		return core.ErrLoginMethodUnsupported
	}
	// TODO: start pairing in a goroutine bound to ctx; push LoginQREvent,
	// then LoginSuccessEvent or LoginFailedEvent.
	return core.ErrNotImplemented
}

// Logout implements core.Session: unlink on the network and wipe s.deps.DataDir.
func (s *Session) Logout(ctx context.Context) error {
	// TODO
	return core.ErrNotLoggedIn
}

// ListChats implements core.Session.
func (s *Session) ListChats(ctx context.Context, opts core.ListChatsOptions) ([]core.Chat, error) {
	return nil, core.ErrNotImplemented // TODO
}

// GetMessages implements core.Session.
func (s *Session) GetMessages(ctx context.Context, chatID core.ChatID, opts core.GetMessagesOptions) ([]core.Message, error) {
	return nil, core.ErrNotImplemented // TODO
}

// SendMessage implements core.Session.
func (s *Session) SendMessage(ctx context.Context, chatID core.ChatID, msg core.OutgoingMessage) (core.Message, error) {
	return core.Message{}, core.ErrCapabilityUnsupported // TODO
}

// SubscribeEvents implements core.Session.
func (s *Session) SubscribeEvents() <-chan core.Event { return s.events.Events() }

// Close implements core.Session: disconnect without logging out.
func (s *Session) Close() error {
	// TODO: disconnect the network client.
	s.events.Close()
	return nil
}
