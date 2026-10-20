// Package enginetest is the conformance suite every engine must pass. It checks the
// core.Engine / core.Session contract the Manager relies on: a valid manifest, coded
// errors, the event channel lifecycle, concurrent sessions, and that an engine never
// writes outside its DataDir. It never needs network access by itself.
//
// Use it from the engine's tests:
//
//	func TestConformance(t *testing.T) {
//		enginetest.Run(t, New(), enginetest.Options{})
//	}
package enginetest

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/baddiamond/prive/internal/core"
	"github.com/baddiamond/prive/internal/store"
)

// Options tunes the suite for one engine.
type Options struct {
	// Login drives a fresh session to a logged-in, connected state without real
	// network access (fakes or recorded fixtures). If nil, the checks that need a
	// logged-in account are skipped.
	Login func(t *testing.T, s core.Session)
	// ChatID is a chat a logged-in session can send to. Required for the send check
	// when the engine declares Capabilities.SendText.
	ChatID core.ChatID
	// Sessions is how many sessions run at once in the concurrency check (default 8).
	Sessions int
	// Timeout bounds every wait (default 5s).
	Timeout time.Duration
}

func (o Options) withDefaults() Options {
	if o.Sessions <= 0 {
		o.Sessions = 8
	}
	if o.Timeout <= 0 {
		o.Timeout = 5 * time.Second
	}
	return o
}

// Run executes the whole suite against engine.
func Run(t *testing.T, engine core.Engine, opts Options) {
	t.Helper()
	opts = opts.withDefaults()
	h := &harness{engine: engine, opts: opts}

	t.Run("Manifest", h.testManifest)
	t.Run("ConnectWithoutLogin", h.testConnectWithoutLogin)
	t.Run("UnsupportedLoginMethod", h.testUnsupportedLoginMethod)
	t.Run("EventChannelLifecycle", h.testEventChannelLifecycle)
	t.Run("ConcurrentSessions", h.testConcurrentSessions)
	t.Run("LoggedIn", h.testLoggedIn)
}

type harness struct {
	engine core.Engine
	opts   Options
}

// account is an isolated test account: its DataDir sits alone inside a parent folder,
// so anything written next to it is detected.
type account struct {
	parent  string
	dataDir string
	account core.Account
}

func (h *harness) newAccount(t *testing.T) account {
	t.Helper()
	parent := t.TempDir()
	dir := filepath.Join(parent, "account")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	return account{
		parent:  parent,
		dataDir: dir,
		account: core.Account{
			ID:        core.NewAccountID(),
			EngineID:  h.engine.Manifest().ID,
			CreatedAt: time.Now().UTC(),
		},
	}
}

// open creates a session, drains its events in the background like the Manager does,
// and registers cleanup that closes it and checks the DataDir boundary.
func (h *harness) open(t *testing.T, a account) core.Session {
	t.Helper()
	s, err := h.engine.NewSession(a.account, h.deps(a))
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if s == nil {
		t.Fatal("NewSession returned a nil session")
	}
	done := drain(s.SubscribeEvents())
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
		h.waitClosed(t, done)
		assertOnlyDataDir(t, a)
	})
	return s
}

// deps mirrors what the Manager gives a session, including encrypted databases.
func (h *harness) deps(a account) core.SessionDeps {
	key := bytes.Repeat([]byte{0x5a}, store.KeySize)
	return core.SessionDeps{
		DataDir: a.dataDir,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		OpenDB: func(ctx context.Context, name string) (*sql.DB, error) {
			return store.OpenDB(ctx, filepath.Join(a.dataDir, name+".db"), store.Options{Key: key})
		},
	}
}

func (h *harness) ctx(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), h.opts.Timeout)
	t.Cleanup(cancel)
	return ctx
}

func (h *harness) testManifest(t *testing.T) {
	m := h.engine.Manifest()
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
	if again := h.engine.Manifest(); again.ID != m.ID {
		t.Errorf("Manifest ID is not stable: %q then %q", m.ID, again.ID)
	}
	if m.Capabilities.HistoryOnDemand {
		if _, ok := h.open(t, h.newAccount(t)).(core.HistoryRequester); !ok {
			t.Error("Capabilities.HistoryOnDemand is declared but the session does not implement core.HistoryRequester")
		}
	}
}

func (h *harness) testConnectWithoutLogin(t *testing.T) {
	s := h.open(t, h.newAccount(t))
	err := s.Connect(h.ctx(t))
	if !errors.Is(err, core.ErrNotLoggedIn) {
		t.Fatalf("Connect on a fresh account = %v, want core.ErrNotLoggedIn", err)
	}
	assertCoded(t, err)
}

func (h *harness) testUnsupportedLoginMethod(t *testing.T) {
	m := h.engine.Manifest()
	for _, method := range []core.LoginMethod{core.LoginQR, core.LoginPhoneCode, core.LoginToken} {
		if m.SupportsLogin(method) {
			continue
		}
		s := h.open(t, h.newAccount(t))
		err := s.Login(h.ctx(t), method)
		if !errors.Is(err, core.ErrLoginMethodUnsupported) {
			t.Errorf("Login(%s) = %v, want core.ErrLoginMethodUnsupported", method, err)
		}
		return
	}
	t.Skip("engine supports every login method")
}

func (h *harness) testEventChannelLifecycle(t *testing.T) {
	a := h.newAccount(t)
	s, err := h.engine.NewSession(a.account, h.deps(a))
	if err != nil {
		t.Fatal(err)
	}
	ch := s.SubscribeEvents()
	if ch == nil {
		t.Fatal("SubscribeEvents returned nil")
	}
	if s.SubscribeEvents() != ch {
		t.Error("SubscribeEvents must return the same channel every time")
	}
	done := drain(ch)
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	h.waitClosed(t, done)
	if err := s.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
	assertOnlyDataDir(t, a)
}

func (h *harness) testConcurrentSessions(t *testing.T) {
	var wg sync.WaitGroup
	for range h.opts.Sessions {
		a := h.newAccount(t)
		s := h.open(t, a)
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), h.opts.Timeout)
			defer cancel()
			if err := s.Connect(ctx); !errors.Is(err, core.ErrNotLoggedIn) {
				t.Errorf("concurrent Connect = %v, want core.ErrNotLoggedIn", err)
			}
		}()
	}
	wg.Wait()
}

func (h *harness) testLoggedIn(t *testing.T) {
	if h.opts.Login == nil {
		t.Skip("Options.Login not provided")
	}
	a := h.newAccount(t)
	s := h.open(t, a)
	h.opts.Login(t, s)

	if _, err := s.ListChats(h.ctx(t), core.ListChatsOptions{Limit: 10}); err != nil {
		t.Errorf("ListChats: %v", err)
	}

	if h.engine.Manifest().Capabilities.SendText {
		h.checkSend(t, s)
	}

	t.Run("ResumeAfterRestart", func(t *testing.T) {
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
		again := h.open(t, a)
		if err := again.Connect(h.ctx(t)); err != nil {
			t.Fatalf("Connect with a stored login: %v", err)
		}

		if err := again.Logout(h.ctx(t)); err != nil {
			t.Fatalf("Logout: %v", err)
		}
		assertEmptyDir(t, a.dataDir)

		third := h.open(t, a)
		if err := third.Connect(h.ctx(t)); !errors.Is(err, core.ErrNotLoggedIn) {
			t.Errorf("Connect after Logout = %v, want core.ErrNotLoggedIn", err)
		}
	})
}

func (h *harness) checkSend(t *testing.T, s core.Session) {
	t.Helper()
	if h.opts.ChatID == "" {
		t.Error("engine declares SendText: set Options.ChatID")
		return
	}
	msg, err := s.SendMessage(h.ctx(t), h.opts.ChatID, core.OutgoingMessage{Text: "conformance"})
	if err != nil {
		t.Errorf("SendMessage: %v", err)
		return
	}
	switch {
	case msg.ID == "":
		t.Error("sent message has no ID")
	case msg.ChatID != h.opts.ChatID:
		t.Errorf("sent message ChatID = %q, want %q", msg.ChatID, h.opts.ChatID)
	case !msg.FromMe:
		t.Error("sent message must have FromMe = true")
	case msg.Kind != core.MessageText:
		t.Errorf("sent message Kind = %q, want text", msg.Kind)
	case msg.Status != core.StatusPending && msg.Status != core.StatusSent:
		t.Errorf("sent message Status = %q, want pending or sent", msg.Status)
	case msg.Timestamp.IsZero():
		t.Error("sent message has no timestamp")
	}
}

func (h *harness) waitClosed(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(h.opts.Timeout):
		t.Error("event channel not closed after Close")
	}
}

// drain reads ch until it is closed and reports that on the returned channel.
func drain(ch <-chan core.Event) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range ch {
		}
	}()
	return done
}

func assertCoded(t *testing.T, err error) {
	t.Helper()
	var ce *core.CodedError
	if !errors.As(err, &ce) || ce.Code == "" {
		t.Errorf("error %v is not a core.CodedError", err)
	}
}

// assertOnlyDataDir fails if the engine created anything next to its DataDir.
func assertOnlyDataDir(t *testing.T, a account) {
	t.Helper()
	entries, err := os.ReadDir(a.parent)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != filepath.Base(a.dataDir) {
			t.Errorf("engine wrote %q outside its DataDir", e.Name())
		}
	}
}

func assertEmptyDir(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		t.Errorf("Logout left %q in DataDir", e.Name())
	}
}
