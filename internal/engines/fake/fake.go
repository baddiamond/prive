// Package fake is an in-memory engine for tests. It behaves like a real network
// (stored login in DataDir, login via QR, chats, messages, delivery statuses)
// without any network access, and test code can drive it through *Session.
//
// It is not registered in the default registry and never ships in the app.
package fake

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/baddiamond/prive/internal/core"
)

// EngineID is the fake engine's manifest ID.
const EngineID = "fake"

// sessionFile marks a logged-in account inside its DataDir.
const sessionFile = "session"

// Engine creates fake sessions and keeps track of them so tests can reach them.
type Engine struct {
	mu       sync.Mutex
	sessions map[core.AccountID]*Session
	// connectErr, if set, is returned by every Connect call.
	connectErr error
	// demo makes linked accounts import a fictional history (see NewDemo).
	demo bool
}

// New returns a fake engine.
func New() *Engine {
	return &Engine{sessions: map[core.AccountID]*Session{}}
}

// Manifest implements core.Engine.
func (e *Engine) Manifest() core.Manifest {
	return core.Manifest{
		ID:           EngineID,
		Name:         "Fake",
		ShortName:    "FK",
		LoginMethods: []core.LoginMethod{core.LoginQR},
		Capabilities: core.Capabilities{SendText: true, ReadReceipts: true, HistoryOnDemand: true},
	}
}

// NewSession implements core.Engine.
func (e *Engine) NewSession(account core.Account, deps core.SessionDeps) (core.Session, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	s := &Session{
		account:  account,
		deps:     deps,
		dataDir:  deps.DataDir,
		events:   core.NewEventQueue(),
		chats:    map[core.ChatID]*core.Chat{},
		messages: map[core.ChatID][]core.Message{},
		engine:   e,

		historyPages: map[core.ChatID]int{},
		groups:       map[core.ChatID]bool{},
	}
	e.sessions[account.ID] = s
	return s, nil
}

// FailConnect makes every Connect fail with err from now on (nil restores success).
func (e *Engine) FailConnect(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.connectErr = err
}

// Session returns the most recent session created for an account, or nil.
func (e *Engine) Session(id core.AccountID) *Session {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.sessions[id]
}

// Session is one fake account. It implements core.Session.
type Session struct {
	account core.Account
	deps    core.SessionDeps
	dataDir string
	events  *core.EventQueue
	engine  *Engine

	mu        sync.Mutex
	connected bool
	closed    bool
	nextID    int
	chats     map[core.ChatID]*core.Chat
	messages  map[core.ChatID][]core.Message
	// historyPages counts RequestOlderHistory calls per chat.
	historyPages map[core.ChatID]int
	// groups marks the demo's group chats, whose fictional senders vary.
	groups map[core.ChatID]bool
}

// RemoteID is the network identity a fake account gets after login.
func RemoteID(id core.AccountID) string { return "fake:" + string(id) }

// Connect implements core.Session.
func (s *Session) Connect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.engine.mu.Lock()
	connectErr := s.engine.connectErr
	s.engine.mu.Unlock()
	if connectErr != nil {
		return connectErr
	}
	if _, err := os.Stat(filepath.Join(s.dataDir, sessionFile)); errors.Is(err, os.ErrNotExist) {
		return core.ErrNotLoggedIn
	} else if err != nil {
		return core.ErrConnectFailed.Wrap(err)
	}
	s.mu.Lock()
	s.connected = true
	s.mu.Unlock()
	s.events.Push(core.AccountStateEvent{State: core.StateConnected})
	return nil
}

// Login implements core.Session: it shows one QR code and immediately "scans" it.
func (s *Session) Login(_ context.Context, method core.LoginMethod) error {
	if method != core.LoginQR {
		return core.ErrLoginMethodUnsupported
	}
	if err := os.WriteFile(filepath.Join(s.dataDir, sessionFile), []byte("ok"), 0o600); err != nil {
		return core.ErrUnknown.Wrap(err)
	}
	s.mu.Lock()
	s.connected = true
	s.mu.Unlock()
	s.events.Push(core.LoginQREvent{Code: "fake-qr", ExpiresAt: time.Now().Add(time.Minute)})
	s.events.Push(core.LoginSuccessEvent{RemoteID: RemoteID(s.account.ID), DisplayName: "Fake user"})
	s.events.Push(core.AccountStateEvent{State: core.StateConnected})
	if s.engine.demo {
		go s.importDemo()
	}
	return nil
}

// Logout implements core.Session.
func (s *Session) Logout(context.Context) error {
	if _, err := os.Stat(filepath.Join(s.dataDir, sessionFile)); errors.Is(err, os.ErrNotExist) {
		return core.ErrNotLoggedIn
	}
	if err := clearDir(s.dataDir); err != nil {
		return core.ErrUnknown.Wrap(err)
	}
	s.mu.Lock()
	s.connected = false
	s.mu.Unlock()
	s.events.Push(core.AccountStateEvent{State: core.StateLoggedOut})
	return nil
}

// ListChats implements core.Session.
func (s *Session) ListChats(_ context.Context, opts core.ListChatsOptions) ([]core.Chat, error) {
	if err := s.requireConnected(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]core.Chat, 0, len(s.chats))
	for _, c := range s.chats {
		if !opts.Before.IsZero() && !c.LastMessageAt.Before(opts.Before) {
			continue
		}
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastMessageAt.After(out[j].LastMessageAt) })
	if opts.Limit > 0 && len(out) > opts.Limit {
		out = out[:opts.Limit]
	}
	return out, nil
}

// GetMessages implements core.Session.
func (s *Session) GetMessages(_ context.Context, chatID core.ChatID, opts core.GetMessagesOptions) ([]core.Message, error) {
	if err := s.requireConnected(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []core.Message
	for _, m := range s.messages[chatID] {
		if opts.Before.IsZero() || m.Timestamp.Before(opts.Before) {
			out = append(out, m)
		}
	}
	if opts.Limit > 0 && len(out) > opts.Limit {
		out = out[len(out)-opts.Limit:]
	}
	return out, nil
}

// SendMessage implements core.Session. The message is "sent" at once and
// "delivered" shortly after through a MessageStatusEvent.
func (s *Session) SendMessage(_ context.Context, chatID core.ChatID, out core.OutgoingMessage) (core.Message, error) {
	if err := s.requireConnected(); err != nil {
		return core.Message{}, err
	}
	msg := s.addMessage(chatID, core.Message{
		SenderID: RemoteID(s.account.ID),
		FromMe:   true,
		Kind:     core.MessageText,
		Text:     out.Text,
		Status:   core.StatusSent,
		ReplyTo:  out.ReplyTo,
	})
	s.events.Push(core.MessageStatusEvent{ChatID: chatID, MessageID: msg.ID, Status: core.StatusDelivered})
	return msg, nil
}

// SubscribeEvents implements core.Session.
func (s *Session) SubscribeEvents() <-chan core.Event { return s.events.Events() }

// Close implements core.Session.
func (s *Session) Close() error {
	s.mu.Lock()
	s.closed = true
	s.connected = false
	s.mu.Unlock()
	s.events.Close()
	return nil
}

// Receive simulates an incoming text message from a contact and emits the
// resulting MessageEvent and ChatUpdatedEvent, like a real engine would.
func (s *Session) Receive(chatID core.ChatID, chatName, text string) core.Message {
	return s.addMessage(chatID, core.Message{
		SenderID:   "contact:" + string(chatID),
		SenderName: chatName,
		Kind:       core.MessageText,
		Text:       text,
		Status:     core.StatusDelivered,
	}, chatName)
}

// Emit pushes an arbitrary event, for tests of the Manager's event handling.
func (s *Session) Emit(e core.Event) { s.events.Push(e) }

// Deps returns the dependencies the core gave this session.
func (s *Session) Deps() core.SessionDeps { return s.deps }

// Closed reports whether Close was called.
func (s *Session) Closed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

func (s *Session) addMessage(chatID core.ChatID, m core.Message, name ...string) core.Message {
	s.mu.Lock()
	s.nextID++
	m.ID = core.MessageID("m" + strconv.Itoa(s.nextID))
	m.ChatID = chatID
	// Strictly increasing timestamps keep ordering deterministic in tests.
	m.Timestamp = time.Now().UTC().Truncate(time.Millisecond).Add(time.Duration(s.nextID) * time.Millisecond)
	s.messages[chatID] = append(s.messages[chatID], m)

	chat, ok := s.chats[chatID]
	if !ok {
		chat = &core.Chat{ID: chatID, Kind: core.ChatDirect, Name: string(chatID)}
		s.chats[chatID] = chat
	}
	if len(name) > 0 && name[0] != "" {
		chat.Name = name[0]
	}
	chat.LastMessagePreview = m.Text
	chat.LastMessageAt = m.Timestamp
	if !m.FromMe {
		chat.UnreadCount++
	}
	chatName, kind := chat.Name, chat.Kind
	s.mu.Unlock()

	// Like a real engine: only the message; the core derives the chat's last message and
	// unread count from it.
	s.events.Push(core.MessageEvent{Message: m, ChatKind: kind, ChatName: chatName})
	return m
}

func (s *Session) requireConnected() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.connected {
		return core.ErrNotLoggedIn
	}
	return nil
}

// clearDir deletes everything inside dir but keeps dir itself.
func clearDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read %s: %w", dir, err)
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}
