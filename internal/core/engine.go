package core

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"
)

// Engine is the factory for one messaging network. It is registered once at startup
// and creates one Session per account.
type Engine interface {
	// Manifest describes the engine to the core and the UI.
	Manifest() Manifest
	// NewSession creates (but does not connect) a session for one account.
	NewSession(account Account, deps SessionDeps) (Session, error)
}

// Session is one running account. All methods must be safe to call concurrently,
// and many sessions of the same engine run at once.
type Session interface {
	// Connect resumes a stored login. It returns ErrNotLoggedIn if there is none.
	// Once connected, the session reconnects on its own and reports AccountStateEvent.
	Connect(ctx context.Context) error
	// Login starts an interactive login and returns once it has started. Progress
	// arrives as events (LoginQREvent, LoginSuccessEvent, LoginFailedEvent).
	Login(ctx context.Context, method LoginMethod) error
	// Logout unlinks the account on the network side and wipes its engine data.
	Logout(ctx context.Context) error

	// ListChats returns chats from the network, most recent first.
	ListChats(ctx context.Context, opts ListChatsOptions) ([]Chat, error)
	// GetMessages returns a page of a chat's history in chronological order.
	GetMessages(ctx context.Context, chatID ChatID, opts GetMessagesOptions) ([]Message, error)
	// SendMessage sends a message and returns it with a provisional status;
	// later changes arrive as MessageStatusEvent.
	SendMessage(ctx context.Context, chatID ChatID, msg OutgoingMessage) (Message, error)

	// SubscribeEvents returns the session's event channel. Every call returns the same
	// channel, which is closed by Close. Sending must never block the network library.
	SubscribeEvents() <-chan Event
	// Close disconnects without logging out and closes the event channel.
	// Calling it more than once is safe.
	Close() error
}

// LoginMethod is a way to link an account.
type LoginMethod string

// Login methods.
const (
	LoginQR        LoginMethod = "qr"
	LoginPhoneCode LoginMethod = "phoneCode"
	LoginToken     LoginMethod = "token"
)

func (m LoginMethod) valid() bool {
	return m == LoginQR || m == LoginPhoneCode || m == LoginToken
}

// Capabilities tells the UI which features an engine supports. The UI never checks
// the engine ID; it shows or hides features from these flags.
type Capabilities struct {
	SendText     bool `json:"sendText"`
	ReadReceipts bool `json:"readReceipts"`
	Typing       bool `json:"typing"`
	Media        bool `json:"media"`
	Reactions    bool `json:"reactions"`
	Edit         bool `json:"edit"`
	Delete       bool `json:"delete"`
	Replies      bool `json:"replies"`
}

// Manifest describes an engine.
type Manifest struct {
	// ID is stable forever (it is stored in the database): lowercase, e.g. "whatsapp".
	ID string `json:"id"`
	// Name is the brand name, not translated: "WhatsApp".
	Name         string        `json:"name"`
	LoginMethods []LoginMethod `json:"loginMethods"`
	Capabilities Capabilities  `json:"capabilities"`
}

var engineIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]{1,31}$`)

// Validate reports whether the manifest is well formed.
func (m Manifest) Validate() error {
	if !engineIDPattern.MatchString(m.ID) {
		return fmt.Errorf("manifest: id %q must match %s", m.ID, engineIDPattern)
	}
	if m.Name == "" {
		return fmt.Errorf("manifest %s: name is empty", m.ID)
	}
	if len(m.LoginMethods) == 0 {
		return fmt.Errorf("manifest %s: no login methods", m.ID)
	}
	for _, lm := range m.LoginMethods {
		if !lm.valid() {
			return fmt.Errorf("manifest %s: unknown login method %q", m.ID, lm)
		}
	}
	return nil
}

// SupportsLogin reports whether the engine offers the given login method.
func (m Manifest) SupportsLogin(method LoginMethod) bool {
	for _, lm := range m.LoginMethods {
		if lm == method {
			return true
		}
	}
	return false
}

// SessionDeps is what the core gives each session.
type SessionDeps struct {
	// DataDir is a private folder for this account's engine data. It exists and
	// the session must not write anywhere else.
	DataDir string
	// Logger is already tagged with the engine and account. Never log message
	// content, phone numbers, tokens or keys.
	Logger *slog.Logger
	// OpenDB opens (or creates) a private SQLite database named name ("store", "cache"…)
	// inside DataDir, with the app's driver and encryption (ADR 0007). Engines must use it
	// for every database instead of opening files themselves, and close it in Close.
	OpenDB func(ctx context.Context, name string) (*sql.DB, error)
}

var dbNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)

// DBOpener opens one engine database: path is the file inside the account's DataDir and
// label identifies it for key derivation ("engine/<engineId>/<accountId>/<name>").
type DBOpener func(ctx context.Context, path, label string) (*sql.DB, error)

// EngineSource looks up registered engines. internal/engines provides the implementation.
type EngineSource interface {
	// Engine returns the engine with the given ID.
	Engine(id string) (Engine, bool)
}
