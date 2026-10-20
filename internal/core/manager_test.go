package core_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/baddiamond/prive/internal/core"
	"github.com/baddiamond/prive/internal/engines"
	"github.com/baddiamond/prive/internal/engines/fake"
	"github.com/baddiamond/prive/internal/keys"
	"github.com/baddiamond/prive/internal/store"
)

const waitTimeout = 5 * time.Second

// env is a Manager wired to a real SQLite store and the fake engine.
type env struct {
	t        *testing.T
	dataDir  string
	store    *store.Store
	registry *engines.Registry
	fake     *fake.Engine
	keys     *keys.Keys
	events   *recorder
	manager  *core.Manager
}

func newEnv(t *testing.T, extra ...core.Engine) *env {
	t.Helper()
	dir := t.TempDir()
	k, _, err := keys.Unlock(dir, keys.NewMemoryKeychain())
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(context.Background(), filepath.Join(dir, store.FileName), store.Options{Key: k.Derive("prive.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	reg := engines.NewRegistry()
	fk := fake.New()
	for _, e := range append([]core.Engine{fk}, extra...) {
		if err := reg.Register(e); err != nil {
			t.Fatal(err)
		}
	}
	e := &env{t: t, dataDir: dir, store: st, registry: reg, fake: fk, keys: k}
	e.restart()
	return e
}

// restart closes the current Manager (if any) and starts a fresh one on the same data,
// like quitting and reopening the app.
func (e *env) restart() {
	e.t.Helper()
	if e.manager != nil {
		if err := e.manager.Close(); err != nil {
			e.t.Fatal(err)
		}
	}
	e.events = &recorder{}
	e.manager = core.NewManager(core.ManagerConfig{
		Engines: e.registry,
		Repo:    e.store,
		DataDir: e.dataDir,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		Emit:    e.events.add,
		OpenDB: func(ctx context.Context, path, label string) (*sql.DB, error) {
			return store.OpenDB(ctx, path, store.Options{Key: e.keys.Derive(label)})
		},
	})
	m := e.manager
	e.t.Cleanup(func() { _ = m.Close() })
	if err := m.Start(context.Background()); err != nil {
		e.t.Fatal(err)
	}
}

func (e *env) addLoggedIn(name string) core.Account {
	e.t.Helper()
	ctx := context.Background()
	acc, err := e.manager.AddAccount(ctx, core.NewAccount{EngineID: fake.EngineID, DisplayName: name})
	if err != nil {
		e.t.Fatal(err)
	}
	if err := e.manager.Login(ctx, acc.ID, core.LoginQR); err != nil {
		e.t.Fatal(err)
	}
	e.waitState(acc.ID, core.StateConnected)
	return acc
}

func (e *env) waitState(id core.AccountID, want core.AccountState) {
	e.t.Helper()
	waitFor(e.t, fmt.Sprintf("account %s to be %s", id, want), func() bool {
		return stateOf(e.manager, id) == want
	})
}

func stateOf(m *core.Manager, id core.AccountID) core.AccountState {
	for _, s := range m.Accounts() {
		if s.Account.ID == id {
			return s.State
		}
	}
	return ""
}

// recorder collects emitted events; Emit is called from many goroutines.
type recorder struct {
	mu     sync.Mutex
	events []core.Event
}

func (r *recorder) add(e core.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
}

func (r *recorder) snapshot() []core.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]core.Event(nil), r.events...)
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(waitTimeout)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestManyAccountsRunConcurrently(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	const n = 3 // e.g. three WhatsApp numbers at once
	accounts := make([]core.Account, n)
	for i := range accounts {
		accounts[i] = e.addLoggedIn(fmt.Sprintf("Number %d", i+1))
	}

	// Every account receives messages at the same time.
	var wg sync.WaitGroup
	for i, acc := range accounts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s := e.fake.Session(acc.ID)
			for j := range 10 {
				s.Receive(core.ChatID(fmt.Sprintf("chat-%d", j%2)), "Contact", fmt.Sprintf("msg %d from account %d", j, i))
			}
		}()
	}
	wg.Wait()

	waitFor(t, "all chats stored", func() bool {
		chats, _ := e.manager.ListChats(ctx, core.ChatFilter{})
		return len(chats) == n*2
	})
	for _, acc := range accounts {
		chats, err := e.manager.ListChats(ctx, core.ChatFilter{AccountID: acc.ID})
		if err != nil {
			t.Fatal(err)
		}
		if len(chats) != 2 {
			t.Fatalf("account %s has %d chats, want 2", acc.ID, len(chats))
		}
		for _, c := range chats {
			if c.AccountID != acc.ID {
				t.Errorf("chat %s stamped with account %s, want %s", c.ID, c.AccountID, acc.ID)
			}
		}
		waitFor(t, "messages stored", func() bool {
			msgs, _ := e.manager.GetMessages(ctx, acc.ID, "chat-0", core.GetMessagesOptions{Limit: 100})
			return len(msgs) == 5
		})
	}

	// Every forwarded event carries its account.
	for _, ev := range e.events.snapshot() {
		if msg, ok := ev.(core.MessageEvent); ok && (msg.AccountID == "" || msg.Message.AccountID != msg.AccountID) {
			t.Errorf("message event not stamped: %+v", msg)
		}
	}

	// Logins were recorded with the remote identity.
	stored, _ := e.store.ListAccounts(ctx)
	for i, a := range stored {
		if a.RemoteID != fake.RemoteID(a.ID) {
			t.Errorf("account %d RemoteID = %q", i, a.RemoteID)
		}
	}
}

func TestAccountsReconnectAfterRestart(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	a := e.addLoggedIn("A")
	b := e.addLoggedIn("B")
	notLinked, err := e.manager.AddAccount(ctx, core.NewAccount{EngineID: fake.EngineID, DisplayName: "C"})
	if err != nil {
		t.Fatal(err)
	}

	e.restart()

	e.waitState(a.ID, core.StateConnected)
	e.waitState(b.ID, core.StateConnected)
	e.waitState(notLinked.ID, core.StateLoggedOut)

	got := e.manager.Accounts()
	if len(got) != 3 || got[0].Account.ID != a.ID || got[1].Account.ID != b.ID || got[2].Account.ID != notLinked.ID {
		t.Fatalf("accounts not restored in order: %+v", got)
	}
	if got[0].Manifest.ID != fake.EngineID {
		t.Errorf("manifest missing: %+v", got[0].Manifest)
	}
}

// blockingEngine never finishes connecting until it is canceled, like an account whose
// network is unreachable.
type blockingEngine struct{}

func (blockingEngine) Manifest() core.Manifest {
	return core.Manifest{ID: "blocking", Name: "Blocking", LoginMethods: []core.LoginMethod{core.LoginQR}}
}

func (blockingEngine) NewSession(core.Account, core.SessionDeps) (core.Session, error) {
	return &blockingSession{events: core.NewEventQueue()}, nil
}

type blockingSession struct {
	events *core.EventQueue
}

func (s *blockingSession) Connect(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}
func (s *blockingSession) Login(context.Context, core.LoginMethod) error { return nil }
func (s *blockingSession) Logout(context.Context) error                  { return core.ErrNotLoggedIn }
func (s *blockingSession) ListChats(context.Context, core.ListChatsOptions) ([]core.Chat, error) {
	return nil, nil
}
func (s *blockingSession) GetMessages(context.Context, core.ChatID, core.GetMessagesOptions) ([]core.Message, error) {
	return nil, nil
}
func (s *blockingSession) SendMessage(context.Context, core.ChatID, core.OutgoingMessage) (core.Message, error) {
	return core.Message{}, core.ErrCapabilityUnsupported
}
func (s *blockingSession) SubscribeEvents() <-chan core.Event { return s.events.Events() }
func (s *blockingSession) Close() error                       { s.events.Close(); return nil }

func TestFailingAccountsDoNotBlockOthers(t *testing.T) {
	e := newEnv(t, blockingEngine{})
	ctx := context.Background()
	healthy := e.addLoggedIn("Healthy")
	stuck, err := e.manager.AddAccount(ctx, core.NewAccount{EngineID: "blocking"})
	if err != nil {
		t.Fatal(err)
	}
	// An account whose engine is no longer bundled.
	orphan := core.Account{ID: "orphan", EngineID: "removed", Position: 9, CreatedAt: time.Now()}
	if err := e.store.CreateAccount(ctx, orphan); err != nil {
		t.Fatal(err)
	}

	e.restart()

	e.waitState(healthy.ID, core.StateConnected)
	e.waitState(stuck.ID, core.StateConnecting)
	for _, s := range e.manager.Accounts() {
		if s.Account.ID == orphan.ID && (s.State != core.StateError || s.Code != core.CodeEngineNotFound) {
			t.Errorf("orphan account state = %s/%s, want error/%s", s.State, s.Code, core.CodeEngineNotFound)
		}
	}
	if err := e.manager.Login(ctx, orphan.ID, core.LoginQR); !errors.Is(err, core.ErrEngineNotFound) {
		t.Errorf("Login on orphan = %v, want ErrEngineNotFound", err)
	}

	// The healthy account keeps working while the other one hangs.
	e.fake.Session(healthy.ID).Receive("c", "Contact", "still alive")
	waitFor(t, "message on healthy account", func() bool {
		msgs, _ := e.manager.GetMessages(ctx, healthy.ID, "c", core.GetMessagesOptions{})
		return len(msgs) == 1
	})

	// Close cancels the hanging connection instead of waiting forever.
	done := make(chan error, 1)
	go func() { done <- e.manager.Close() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(waitTimeout):
		t.Fatal("Close blocked on a hanging connection")
	}
}

func TestConnectErrorsBecomeStates(t *testing.T) {
	e := newEnv(t)
	a := e.addLoggedIn("A")

	e.fake.FailConnect(errors.New("network unreachable"))
	e.restart()
	waitFor(t, "error state", func() bool {
		for _, s := range e.manager.Accounts() {
			if s.Account.ID == a.ID {
				return s.State == core.StateError && s.Code == core.CodeConnectFailed
			}
		}
		return false
	})
}

func TestSendMessage(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	a := e.addLoggedIn("A")

	msg, err := e.manager.SendMessage(ctx, a.ID, "chat-1", core.OutgoingMessage{Text: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if msg.AccountID != a.ID || !msg.FromMe || msg.ChatID != "chat-1" {
		t.Errorf("unexpected sent message: %+v", msg)
	}
	// The fake engine reports "delivered" right after sending.
	waitFor(t, "delivered status stored", func() bool {
		msgs, _ := e.manager.GetMessages(ctx, a.ID, "chat-1", core.GetMessagesOptions{})
		return len(msgs) == 1 && msgs[0].Status == core.StatusDelivered
	})

	if _, err := e.manager.SendMessage(ctx, "nope", "chat-1", core.OutgoingMessage{Text: "x"}); !errors.Is(err, core.ErrAccountNotFound) {
		t.Errorf("send from unknown account = %v, want ErrAccountNotFound", err)
	}
}

func TestRemoveAccount(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	keep := e.addLoggedIn("Keep")
	drop := e.addLoggedIn("Drop")
	for _, acc := range []core.Account{keep, drop} {
		e.fake.Session(acc.ID).Receive("c", "Contact", "hi")
	}
	waitFor(t, "chats stored", func() bool {
		chats, _ := e.manager.ListChats(ctx, core.ChatFilter{})
		return len(chats) == 2
	})
	dropDir := filepath.Join(e.dataDir, "engines", fake.EngineID, string(drop.ID))
	if _, err := os.Stat(dropDir); err != nil {
		t.Fatalf("account data dir missing before removal: %v", err)
	}
	session := e.fake.Session(drop.ID)

	if err := e.manager.RemoveAccount(ctx, drop.ID); err != nil {
		t.Fatal(err)
	}

	if !session.Closed() {
		t.Error("session not closed")
	}
	if _, err := os.Stat(dropDir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("account data dir still exists: %v", err)
	}
	chats, _ := e.manager.ListChats(ctx, core.ChatFilter{})
	if len(chats) != 1 || chats[0].AccountID != keep.ID {
		t.Errorf("chats after removal = %+v", chats)
	}
	if got := e.manager.Accounts(); len(got) != 1 || got[0].Account.ID != keep.ID {
		t.Errorf("accounts after removal = %+v", got)
	}
	if err := e.manager.RemoveAccount(ctx, drop.ID); !errors.Is(err, core.ErrAccountNotFound) {
		t.Errorf("second removal = %v, want ErrAccountNotFound", err)
	}
	removed := false
	for _, ev := range e.events.snapshot() {
		if r, ok := ev.(core.AccountRemovedEvent); ok && r.AccountID == drop.ID {
			removed = true
		}
	}
	if !removed {
		t.Error("no account:removed event")
	}
}

func TestInputValidation(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	if _, err := e.manager.AddAccount(ctx, core.NewAccount{EngineID: "unknown"}); !errors.Is(err, core.ErrEngineNotFound) {
		t.Errorf("AddAccount(unknown engine) = %v, want ErrEngineNotFound", err)
	}
	acc, err := e.manager.AddAccount(ctx, core.NewAccount{EngineID: fake.EngineID})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.manager.Login(ctx, acc.ID, core.LoginPhoneCode); !errors.Is(err, core.ErrLoginMethodUnsupported) {
		t.Errorf("Login(phoneCode) = %v, want ErrLoginMethodUnsupported", err)
	}
	if _, err := e.manager.ListChats(ctx, core.ChatFilter{AccountID: "nope"}); !errors.Is(err, core.ErrAccountNotFound) {
		t.Errorf("ListChats(unknown account) = %v, want ErrAccountNotFound", err)
	}

	if err := e.manager.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := e.manager.AddAccount(ctx, core.NewAccount{EngineID: fake.EngineID}); !errors.Is(err, core.ErrShuttingDown) {
		t.Errorf("AddAccount after Close = %v, want ErrShuttingDown", err)
	}
}

func TestEngineDatabasesAreEncryptedAndConfined(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	a := e.addLoggedIn("A")
	deps := e.fake.Session(a.ID).Deps()

	db, err := deps.OpenDB(ctx, "store")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE t (v TEXT); INSERT INTO t VALUES ('engine-secret')`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(deps.DataDir, "store.db"))
	if err != nil {
		t.Fatalf("engine database not created inside DataDir: %v", err)
	}
	if bytes.HasPrefix(raw, []byte("SQLite format 3")) || bytes.Contains(raw, []byte("engine-secret")) {
		t.Error("engine database is not encrypted")
	}

	for _, bad := range []string{"../escape", "a/b", "", "UPPER", "store.db"} {
		if db, err := deps.OpenDB(ctx, bad); err == nil {
			_ = db.Close()
			t.Errorf("OpenDB(%q) accepted", bad)
		}
	}
}
