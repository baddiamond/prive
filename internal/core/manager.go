package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// ManagerConfig holds the Manager's dependencies.
type ManagerConfig struct {
	// Engines resolves engine IDs stored with each account.
	Engines EngineSource
	// Repo persists accounts, chats and messages.
	Repo Repository
	// DataDir is the app data directory. Each account gets a private folder
	// at <DataDir>/engines/<engineId>/<accountId>.
	DataDir string
	// Logger receives operational logs. Defaults to slog.Default().
	Logger *slog.Logger
	// Emit receives every event for the UI, already stamped with its account.
	// It is called from several goroutines and must not block for long. Optional.
	Emit func(Event)
	// Now returns the current time. Defaults to time.Now; tests may override it.
	Now func() time.Time
	// NewID creates account IDs. Defaults to NewAccountID; tests may override it.
	NewID func() AccountID
	// OpenDB opens engine databases (see SessionDeps.OpenDB). The app passes an opener that
	// encrypts every file with its own derived key. If nil, engines cannot open databases.
	OpenDB DBOpener
}

// NewAccount is a request to add an account.
type NewAccount struct {
	EngineID    string `json:"engineId"`
	DisplayName string `json:"displayName"`
	Color       string `json:"color"`
}

// AccountStatus is an account together with its live connection state.
type AccountStatus struct {
	Account Account      `json:"account"`
	State   AccountState `json:"state"`
	// Code is an optional error translation key explaining the state.
	Code string `json:"code,omitempty"`
	// Manifest describes the account's engine; it is zero if the engine is not available.
	Manifest Manifest `json:"manifest"`
}

// Manager runs every account concurrently and in isolation: it owns one Session per
// account, persists what sessions report, and forwards their events to the UI.
// A failing or slow account never blocks the others. All methods are safe for concurrent use.
type Manager struct {
	engines EngineSource
	repo    Repository
	dataDir string
	logger  *slog.Logger
	emitFn  func(Event)
	now     func() time.Time
	newID   func() AccountID
	openDB  DBOpener

	ctx    context.Context // lives until Close; used for connections and logins
	cancel context.CancelFunc
	wg     sync.WaitGroup // event pumps and connection goroutines

	// lifecycle serializes Start, AddAccount, RemoveAccount and Close.
	lifecycle sync.Mutex

	mu       sync.RWMutex
	accounts map[AccountID]*runningAccount
	started  bool
	closed   bool
}

// runningAccount is the Manager's per-account state.
type runningAccount struct {
	// ctx is canceled when the account is removed or the Manager closes.
	ctx      context.Context
	cancel   context.CancelFunc
	manifest Manifest
	session  Session       // nil when the engine is missing or the session failed to start
	pumpDone chan struct{} // closed when the session's event channel is drained

	mu      sync.Mutex
	account Account
	state   AccountState
	code    string
}

// NewManager creates a Manager. Call Start to load and connect stored accounts.
func NewManager(cfg ManagerConfig) *Manager {
	m := &Manager{
		engines:  cfg.Engines,
		repo:     cfg.Repo,
		dataDir:  cfg.DataDir,
		logger:   cfg.Logger,
		emitFn:   cfg.Emit,
		now:      cfg.Now,
		newID:    cfg.NewID,
		openDB:   cfg.OpenDB,
		accounts: map[AccountID]*runningAccount{},
	}
	if m.logger == nil {
		m.logger = slog.Default()
	}
	if m.now == nil {
		m.now = time.Now
	}
	if m.newID == nil {
		m.newID = NewAccountID
	}
	m.ctx, m.cancel = context.WithCancel(context.Background())
	return m
}

// Start loads every stored account, creates its session and connects all of them in
// parallel. It returns once the sessions exist; connections finish in the background
// and report through AccountStateEvent.
func (m *Manager) Start(ctx context.Context) error {
	m.lifecycle.Lock()
	defer m.lifecycle.Unlock()
	if m.isClosed() {
		return ErrShuttingDown
	}
	if m.started {
		return errors.New("manager: already started")
	}

	accounts, err := m.repo.ListAccounts(ctx)
	if err != nil {
		return fmt.Errorf("load accounts: %w", err)
	}
	opened := make([]*runningAccount, 0, len(accounts))
	m.mu.Lock()
	for _, acc := range accounts {
		ra := m.open(acc)
		m.accounts[acc.ID] = ra
		opened = append(opened, ra)
	}
	m.started = true
	m.mu.Unlock()

	for _, ra := range opened {
		if ra.session != nil {
			m.connect(ra)
		}
	}
	return nil
}

// Accounts returns every account with its current state, ordered by position.
func (m *Manager) Accounts() []AccountStatus {
	m.mu.RLock()
	out := make([]AccountStatus, 0, len(m.accounts))
	for _, ra := range m.accounts {
		ra.mu.Lock()
		out = append(out, AccountStatus{Account: ra.account, State: ra.state, Code: ra.code, Manifest: ra.manifest})
		ra.mu.Unlock()
	}
	m.mu.RUnlock()

	sort.Slice(out, func(i, j int) bool {
		a, b := out[i].Account, out[j].Account
		if a.Position != b.Position {
			return a.Position < b.Position
		}
		return a.CreatedAt.Before(b.CreatedAt)
	})
	return out
}

// AddAccount creates an account for an engine and its session. The account starts
// logged out; call Login to link it.
func (m *Manager) AddAccount(ctx context.Context, req NewAccount) (Account, error) {
	m.lifecycle.Lock()
	defer m.lifecycle.Unlock()
	if m.isClosed() {
		return Account{}, ErrShuttingDown
	}
	if _, ok := m.engines.Engine(req.EngineID); !ok {
		return Account{}, ErrEngineNotFound
	}

	acc := Account{
		ID:          m.newID(),
		EngineID:    req.EngineID,
		DisplayName: req.DisplayName,
		Color:       req.Color,
		Position:    m.nextPosition(),
		CreatedAt:   m.now().UTC().Truncate(time.Millisecond),
	}
	if err := m.repo.CreateAccount(ctx, acc); err != nil {
		return Account{}, ErrUnknown.Wrap(fmt.Errorf("create account: %w", err))
	}

	ra := m.open(acc)
	m.mu.Lock()
	m.accounts[acc.ID] = ra
	m.mu.Unlock()
	if ra.session != nil {
		m.setState(ra, StateLoggedOut, "")
	}
	return acc, nil
}

// Reconnect retries the connection of an account in the error state ("Retry" in the UI).
func (m *Manager) Reconnect(id AccountID) error {
	ra, err := m.running(id)
	if err != nil {
		return err
	}
	m.connect(ra)
	return nil
}

// UpdateAccount changes the label and color the user chose for an account.
func (m *Manager) UpdateAccount(ctx context.Context, id AccountID, displayName, color string) (Account, error) {
	ra, err := m.lookup(id)
	if err != nil {
		return Account{}, err
	}
	ra.mu.Lock()
	acc := ra.account
	acc.DisplayName, acc.Color = displayName, color
	ra.mu.Unlock()

	if err := m.repo.UpdateAccount(ctx, acc); err != nil {
		return Account{}, ErrUnknown.Wrap(fmt.Errorf("update account: %w", err))
	}
	ra.mu.Lock()
	ra.account.DisplayName, ra.account.Color = displayName, color
	ra.mu.Unlock()
	m.emit(AccountUpdatedEvent{AccountID: id, Account: acc})
	return acc, nil
}

// Login starts an interactive login on an account. Progress arrives as events.
func (m *Manager) Login(_ context.Context, id AccountID, method LoginMethod) error {
	ra, err := m.running(id)
	if err != nil {
		return err
	}
	if !ra.manifest.SupportsLogin(method) {
		return ErrLoginMethodUnsupported
	}
	// The login outlives the caller's request: it ends with an event, a removal or Close.
	if err := ra.session.Login(ra.ctx, method); err != nil {
		return AsCoded(err)
	}
	return nil
}

// RemoveAccount logs the account out on the network (best effort), closes its session
// and deletes its chats, messages and engine data. Removal works offline too: if only the
// network logout failed with a coded error (e.g. "remove the device on your phone"), the
// account is still removed and that error is returned so the UI can tell the user.
func (m *Manager) RemoveAccount(ctx context.Context, id AccountID) error {
	m.lifecycle.Lock()
	defer m.lifecycle.Unlock()
	if m.isClosed() {
		return ErrShuttingDown
	}

	m.mu.Lock()
	ra, ok := m.accounts[id]
	delete(m.accounts, id)
	m.mu.Unlock()
	if !ok {
		return ErrAccountNotFound
	}
	ra.cancel() // stops a pending connection or login

	var logoutNotice error
	if ra.session != nil {
		// Removing must work offline too, so a failed network logout does not stop it.
		if err := ra.session.Logout(ctx); err != nil && !errors.Is(err, ErrNotLoggedIn) {
			m.logger.Warn("logout failed while removing account", "account", id, "err", err)
			var ce *CodedError
			if errors.As(err, &ce) {
				logoutNotice = ce
			}
		}
		if err := ra.session.Close(); err != nil {
			m.logger.Warn("close session", "account", id, "err", err)
		}
		<-ra.pumpDone // no more writes for this account after this point
	}

	if err := m.repo.DeleteAccount(ctx, id); err != nil {
		return ErrUnknown.Wrap(fmt.Errorf("delete account: %w", err))
	}
	if err := os.RemoveAll(m.accountDir(ra.account)); err != nil {
		m.logger.Warn("remove account data", "account", id, "err", err)
	}
	m.emit(AccountRemovedEvent{AccountID: id})
	return logoutNotice
}

// ListChats returns stored chats for the inbox, most recent first.
func (m *Manager) ListChats(ctx context.Context, filter ChatFilter) ([]Chat, error) {
	if filter.AccountID != "" {
		if _, err := m.lookup(filter.AccountID); err != nil {
			return nil, err
		}
	}
	chats, err := m.repo.ListChats(ctx, filter)
	if err != nil {
		return nil, ErrUnknown.Wrap(err)
	}
	return chats, nil
}

// UnreadSummary counts unread messages per account for the rail badges.
func (m *Manager) UnreadSummary(ctx context.Context) (UnreadSummary, error) {
	sum, err := m.repo.UnreadSummary(ctx)
	if err != nil {
		return UnreadSummary{}, ErrUnknown.Wrap(err)
	}
	return sum, nil
}

// GetMessages returns a page of a chat's stored history in chronological order.
func (m *Manager) GetMessages(ctx context.Context, id AccountID, chat ChatID, opts GetMessagesOptions) ([]Message, error) {
	if _, err := m.lookup(id); err != nil {
		return nil, err
	}
	msgs, err := m.repo.ListMessages(ctx, id, chat, opts)
	if err != nil {
		return nil, ErrUnknown.Wrap(err)
	}
	return msgs, nil
}

// historyPageSize is how many older messages RequestOlderHistory asks for at a time.
const historyPageSize = 50

// RequestOlderHistory asks the network for messages older than the oldest one stored for a
// chat. It returns once the request is sent; the answer arrives as a HistoryResultEvent.
func (m *Manager) RequestOlderHistory(ctx context.Context, id AccountID, chat ChatID) error {
	ra, err := m.running(id)
	if err != nil {
		return err
	}
	requester, ok := ra.session.(HistoryRequester)
	if !ok || !ra.manifest.Capabilities.HistoryOnDemand {
		return ErrCapabilityUnsupported
	}
	oldest, found, err := m.repo.OldestMessage(ctx, id, chat)
	if err != nil {
		return ErrUnknown.Wrap(err)
	}
	if !found {
		return ErrHistoryUnavailable
	}
	return requester.RequestOlderHistory(ctx, chat, oldest, historyPageSize)
}

// SendMessage sends a message through the account that owns the chat, stores it and
// emits it as a MessageEvent so the UI shows it immediately.
func (m *Manager) SendMessage(ctx context.Context, id AccountID, chat ChatID, out OutgoingMessage) (Message, error) {
	ra, err := m.running(id)
	if err != nil {
		return Message{}, err
	}
	if !ra.manifest.Capabilities.SendText {
		return Message{}, ErrCapabilityUnsupported
	}
	msg, err := ra.session.SendMessage(ctx, chat, out)
	if err != nil {
		var ce *CodedError
		if errors.As(err, &ce) {
			return Message{}, ce
		}
		return Message{}, ErrSendFailed.Wrap(err)
	}
	msg.AccountID = id
	msg.ChatID = chat
	msg.FromMe = true
	m.emit(MessageEvent{AccountID: id, Message: msg})
	if updated, err := m.repo.ApplyMessage(ctx, msg, ChatDirect, "", false); err != nil {
		m.logger.Error("store sent message", "account", id, "err", err)
	} else {
		m.emit(ChatUpdatedEvent{AccountID: id, Chat: updated})
	}
	return msg, nil
}

// Close disconnects every session (without logging out) and waits for their goroutines.
// The Manager cannot be used afterwards.
func (m *Manager) Close() error {
	m.lifecycle.Lock()
	defer m.lifecycle.Unlock()

	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	running := make([]*runningAccount, 0, len(m.accounts))
	for _, ra := range m.accounts {
		running = append(running, ra)
	}
	m.mu.Unlock()

	m.cancel()
	var (
		wg    sync.WaitGroup
		errMu sync.Mutex
		errs  []error
	)
	for _, ra := range running {
		if ra.session == nil {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := ra.session.Close(); err != nil {
				errMu.Lock()
				errs = append(errs, fmt.Errorf("close account %s: %w", ra.account.ID, err))
				errMu.Unlock()
			}
		}()
	}
	wg.Wait()
	m.wg.Wait()
	return errors.Join(errs...)
}

// open creates the session of an account and starts pumping its events.
// Failures leave the account listed with an error state instead of failing the caller.
func (m *Manager) open(acc Account) *runningAccount {
	ra := &runningAccount{account: acc, state: StateDisconnected}
	ra.ctx, ra.cancel = context.WithCancel(m.ctx)

	eng, ok := m.engines.Engine(acc.EngineID)
	if !ok {
		m.logger.Warn("engine not available", "engine", acc.EngineID, "account", acc.ID)
		ra.state, ra.code = StateError, CodeEngineNotFound
		return ra
	}
	ra.manifest = eng.Manifest()

	dir := m.accountDir(acc)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		m.logger.Error("create account data dir", "account", acc.ID, "err", err)
		ra.state, ra.code = StateError, CodeUnknown
		return ra
	}
	deps := SessionDeps{
		DataDir: dir,
		Logger:  m.logger.With("engine", acc.EngineID, "account", acc.ID),
		OpenDB:  m.engineDBOpener(acc, dir),
	}
	sess, err := newSession(eng, acc, deps)
	if err != nil {
		m.logger.Error("create session", "account", acc.ID, "err", err)
		ra.state, ra.code = StateError, AsCoded(err).Code
		return ra
	}

	ra.session = sess
	ra.pumpDone = make(chan struct{})
	m.wg.Add(1)
	go m.pump(ra)
	return ra
}

// newSession calls the engine, turning a panic into an error so one broken engine
// cannot take the app down at startup.
func newSession(eng Engine, acc Account, deps SessionDeps) (sess Session, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("engine %s panicked in NewSession: %v", acc.EngineID, r)
		}
	}()
	sess, err = eng.NewSession(acc, deps)
	if err == nil && sess == nil {
		err = fmt.Errorf("engine %s returned a nil session", acc.EngineID)
	}
	return sess, err
}

// connect resumes the account's stored login in the background.
func (m *Manager) connect(ra *runningAccount) {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				m.logger.Error("engine panicked in Connect", "account", ra.accountID(), "panic", r)
				m.setState(ra, StateError, CodeConnectFailed)
			}
		}()

		m.setState(ra, StateConnecting, "")
		err := ra.session.Connect(ra.ctx)
		switch {
		case ra.ctx.Err() != nil:
			// Removed or shutting down; the state no longer matters.
		case err == nil:
			// Engines report "connected" themselves; make sure the UI hears it anyway.
			m.setStateIf(ra, StateConnecting, StateConnected)
		case errors.Is(err, ErrNotLoggedIn):
			m.setState(ra, StateLoggedOut, "")
		default:
			m.logger.Warn("connect failed", "account", ra.accountID(), "err", err)
			code := CodeConnectFailed
			var ce *CodedError
			if errors.As(err, &ce) {
				code = ce.Code
			}
			m.setState(ra, StateError, code)
		}
	}()
}

// pump forwards a session's events until its channel is closed.
func (m *Manager) pump(ra *runningAccount) {
	defer m.wg.Done()
	defer close(ra.pumpDone)
	id := ra.accountID()
	for ev := range ra.session.SubscribeEvents() {
		if ev == nil {
			continue
		}
		m.handle(ra, ev.stamped(id))
	}
}

// handle persists what an event changes and forwards the result to the UI.
func (m *Manager) handle(ra *runningAccount, ev Event) {
	for _, out := range m.persist(ra, ev) {
		m.emit(out)
	}
}

// persist stores what an event changes and returns the events the UI should receive:
// usually the event itself, sometimes a summary (history batches) or the resulting chat.
func (m *Manager) persist(ra *runningAccount, ev Event) []Event {
	// Writes must land even while the app is shutting down.
	ctx := context.Background()
	id := ra.accountID()

	switch e := ev.(type) {
	case AccountStateEvent:
		ra.mu.Lock()
		ra.state, ra.code = e.State, e.Code
		ra.mu.Unlock()
	case LoginSuccessEvent:
		ra.mu.Lock()
		ra.account.RemoteID = e.RemoteID
		if ra.account.DisplayName == "" {
			ra.account.DisplayName = e.DisplayName
		}
		acc := ra.account
		ra.mu.Unlock()
		if err := m.repo.UpdateAccount(ctx, acc); err != nil {
			m.logger.Error("store login", "account", id, "err", err)
		}
		return []Event{ev, AccountUpdatedEvent{AccountID: id, Account: acc}}
	case ChatUpdatedEvent:
		if e.Chat.ID == "" {
			m.logger.Warn("dropping chat without id", "account", id)
			return nil
		}
		if err := m.repo.UpsertChat(ctx, e.Chat); err != nil {
			m.logger.Error("store chat", "account", id, "err", err)
		}
	case MessageEvent:
		if e.Message.ID == "" || e.Message.ChatID == "" {
			m.logger.Warn("dropping message without id or chat", "account", id)
			return nil
		}
		chat, err := m.repo.ApplyMessage(ctx, e.Message, e.ChatKind, e.ChatName, true)
		if err != nil {
			m.logger.Error("store message", "account", id, "err", err)
			return []Event{ev}
		}
		return []Event{ev, ChatUpdatedEvent{AccountID: id, Chat: chat}}
	case HistoryBatchEvent:
		if err := m.repo.ImportHistory(ctx, id, e.Chats, e.Messages, e.Authoritative); err != nil {
			m.logger.Error("store history", "account", id, "err", err)
			return nil
		}
		return []Event{ChatsSyncedEvent{AccountID: id, Chats: len(e.Chats), Messages: len(e.Messages)}}
	case ChatPatchEvent:
		chat, found, err := m.repo.PatchChat(ctx, id, e.Patch)
		if err != nil {
			m.logger.Error("update chat", "account", id, "err", err)
		}
		if !found || err != nil {
			return nil // e.g. a contact without a chat
		}
		return []Event{ChatUpdatedEvent{AccountID: id, Chat: chat}}
	case MessageStatusEvent:
		if err := m.repo.UpdateMessageStatus(ctx, id, e.ChatID, e.MessageID, e.Status); err != nil {
			m.logger.Error("store message status", "account", id, "err", err)
		}
	}
	return []Event{ev}
}

// setState records and announces a manager-detected state change.
func (m *Manager) setState(ra *runningAccount, state AccountState, code string) {
	ra.mu.Lock()
	if ra.state == state && ra.code == code {
		ra.mu.Unlock()
		return
	}
	ra.state, ra.code = state, code
	id := ra.account.ID
	ra.mu.Unlock()
	m.emit(AccountStateEvent{AccountID: id, State: state, Code: code})
}

// setStateIf changes the state only if it still is from.
func (m *Manager) setStateIf(ra *runningAccount, from, to AccountState) {
	ra.mu.Lock()
	if ra.state != from {
		ra.mu.Unlock()
		return
	}
	ra.state, ra.code = to, ""
	id := ra.account.ID
	ra.mu.Unlock()
	m.emit(AccountStateEvent{AccountID: id, State: to})
}

func (m *Manager) emit(ev Event) {
	if m.emitFn != nil {
		m.emitFn(ev)
	}
}

// lookup returns a known account, with or without a running session.
func (m *Manager) lookup(id AccountID) (*runningAccount, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.closed {
		return nil, ErrShuttingDown
	}
	ra, ok := m.accounts[id]
	if !ok {
		return nil, ErrAccountNotFound
	}
	return ra, nil
}

// running returns an account whose session is available.
func (m *Manager) running(id AccountID) (*runningAccount, error) {
	ra, err := m.lookup(id)
	if err != nil {
		return nil, err
	}
	if ra.session == nil {
		ra.mu.Lock()
		code := ra.code
		ra.mu.Unlock()
		if code == "" {
			code = CodeUnknown
		}
		return nil, NewError(code)
	}
	return ra, nil
}

func (m *Manager) nextPosition() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	next := 0
	for _, ra := range m.accounts {
		ra.mu.Lock()
		if ra.account.Position >= next {
			next = ra.account.Position + 1
		}
		ra.mu.Unlock()
	}
	return next
}

// engineDBOpener returns the SessionDeps.OpenDB of one account: names are validated so a
// database can only be created inside the account's own folder.
func (m *Manager) engineDBOpener(acc Account, dir string) func(context.Context, string) (*sql.DB, error) {
	return func(ctx context.Context, name string) (*sql.DB, error) {
		if !dbNamePattern.MatchString(name) {
			return nil, fmt.Errorf("engine database name %q must match %s", name, dbNamePattern)
		}
		if m.openDB == nil {
			return nil, ErrNotImplemented
		}
		label := "engine/" + acc.EngineID + "/" + string(acc.ID) + "/" + name
		return m.openDB(ctx, filepath.Join(dir, name+".db"), label)
	}
}

func (m *Manager) accountDir(acc Account) string {
	return filepath.Join(m.dataDir, "engines", acc.EngineID, string(acc.ID))
}

func (m *Manager) isClosed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closed
}

func (ra *runningAccount) accountID() AccountID {
	ra.mu.Lock()
	defer ra.mu.Unlock()
	return ra.account.ID
}
