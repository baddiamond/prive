package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"

	"github.com/baddiamond/prive/internal/core"
)

// storeName is the whatsmeow database inside the account's DataDir (store.db).
const storeName = "store"

// Session is one WhatsApp account. It implements core.Session.
type Session struct {
	account core.Account
	deps    core.SessionDeps
	log     *slog.Logger
	waLog   waLog.Logger
	events  *core.EventQueue

	mu        sync.Mutex
	container *sqlstore.Container // nil until first use
	client    *whatsmeow.Client   // nil while disconnected
	closed    bool
}

func newSession(account core.Account, deps core.SessionDeps) *Session {
	log := deps.Logger
	if log == nil {
		log = slog.Default()
	}
	return &Session{
		account: account,
		deps:    deps,
		log:     log,
		waLog:   newLogger(log),
		events:  core.NewEventQueue(),
	}
}

// Connect implements core.Session: it resumes the stored link and returns once connecting
// has started. If the network is down, whatsmeow keeps retrying in the background.
func (s *Session) Connect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return core.ErrShuttingDown
	}
	if s.client != nil && s.client.IsConnected() {
		return nil
	}
	device, err := s.device(ctx)
	if err != nil {
		return err
	}
	if device.ID == nil {
		return core.ErrNotLoggedIn
	}
	s.replaceClient(s.newClient(device))
	if err := s.client.ConnectContext(ctx); err != nil {
		s.replaceClient(nil)
		return core.ErrConnectFailed.Wrap(err)
	}
	return nil
}

// Login implements core.Session: it starts QR pairing and returns. QR codes, success and
// failure arrive as events. Canceling ctx (or removing the account) stops pairing.
func (s *Session) Login(ctx context.Context, method core.LoginMethod) error {
	if method != core.LoginQR {
		return core.ErrLoginMethodUnsupported
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return core.ErrShuttingDown
	}
	device, err := s.device(ctx)
	if err != nil {
		return err
	}
	if device.ID != nil {
		return ErrAlreadyLinked
	}

	client := s.newClient(device)
	qr, err := client.GetQRChannel(ctx)
	if err != nil {
		return ErrPairingFailed.Wrap(err)
	}
	s.replaceClient(client)
	if err := client.ConnectContext(ctx); err != nil {
		s.replaceClient(nil)
		return core.ErrConnectFailed.Wrap(err)
	}
	go s.forwardQR(client, qr)
	return nil
}

// forwardQR turns whatsmeow's QR channel into Privé events until pairing ends.
func (s *Session) forwardQR(client *whatsmeow.Client, qr <-chan whatsmeow.QRChannelItem) {
	linked := func() (string, string) {
		if client.Store.ID == nil {
			return "", ""
		}
		return client.Store.ID.ToNonAD().String(), client.Store.PushName
	}
	for item := range qr {
		res := qrEvent(item, time.Now(), linked)
		if res.event != nil {
			s.events.Push(res.event)
		}
		if !res.done {
			continue
		}
		if _, failed := res.event.(core.LoginFailedEvent); failed {
			s.mu.Lock()
			if s.client == client {
				s.replaceClient(nil)
			}
			s.mu.Unlock()
		}
		return
	}
}

// Logout implements core.Session: it unlinks the device on WhatsApp's side when online and
// always deletes the local session. Offline, it returns ErrUnlinkOnPhone after wiping, so
// the UI can tell the user to remove the device from the phone.
func (s *Session) Logout(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	device, err := s.device(ctx)
	if err != nil {
		return err
	}
	if device.ID == nil {
		return core.ErrNotLoggedIn
	}

	var remoteErr error
	if s.client != nil && s.client.IsConnected() && s.client.IsLoggedIn() {
		remoteErr = s.client.Logout(ctx)
	} else {
		remoteErr = errors.New("not connected")
	}
	s.replaceClient(nil)
	if err := s.wipe(); err != nil {
		return core.ErrUnknown.Wrap(err)
	}
	s.events.Push(core.AccountStateEvent{State: core.StateLoggedOut})
	if remoteErr != nil {
		s.log.Warn("could not unlink on WhatsApp's side; local data deleted", "err", redact(remoteErr.Error()))
		return ErrUnlinkOnPhone.Wrap(remoteErr)
	}
	return nil
}

// ListChats implements core.Session. Chat sync arrives in phase 3.
func (s *Session) ListChats(context.Context, core.ListChatsOptions) ([]core.Chat, error) {
	return nil, core.ErrNotImplemented
}

// GetMessages implements core.Session. History arrives in phase 4.
func (s *Session) GetMessages(context.Context, core.ChatID, core.GetMessagesOptions) ([]core.Message, error) {
	return nil, core.ErrNotImplemented
}

// SendMessage implements core.Session. Sending arrives in phase 5; the manifest does not
// declare SendText yet.
func (s *Session) SendMessage(context.Context, core.ChatID, core.OutgoingMessage) (core.Message, error) {
	return core.Message{}, core.ErrCapabilityUnsupported
}

// SubscribeEvents implements core.Session.
func (s *Session) SubscribeEvents() <-chan core.Event { return s.events.Events() }

// Close implements core.Session: it disconnects without unlinking and closes the store.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	s.replaceClient(nil)
	var err error
	if s.container != nil {
		err = s.container.Close()
		s.container = nil
	}
	s.events.Close()
	return err
}

// device returns the account's whatsmeow device, opening the store on first use.
// The device has a nil ID until the account is linked. Callers hold s.mu.
func (s *Session) device(ctx context.Context) (*store.Device, error) {
	if s.container == nil {
		if s.deps.OpenDB == nil {
			return nil, core.ErrNotImplemented
		}
		db, err := s.deps.OpenDB(ctx, storeName)
		if err != nil {
			return nil, core.ErrUnknown.Wrap(fmt.Errorf("open whatsapp store: %w", err))
		}
		c := sqlstore.NewWithDB(db, "sqlite3", s.waLog.Sub("store"))
		if err := c.Upgrade(ctx); err != nil {
			_ = c.Close()
			return nil, core.ErrUnknown.Wrap(fmt.Errorf("upgrade whatsapp store: %w", err))
		}
		s.container = c
	}
	device, err := s.container.GetFirstDevice(ctx)
	if err != nil {
		return nil, core.ErrUnknown.Wrap(fmt.Errorf("load whatsapp device: %w", err))
	}
	return device, nil
}

// newClient creates a whatsmeow client whose connection events become account states.
func (s *Session) newClient(device *store.Device) *whatsmeow.Client {
	client := whatsmeow.NewClient(device, s.waLog.Sub("client"))
	client.EnableAutoReconnect = true
	client.InitialAutoReconnect = true // a network outage at startup retries instead of failing
	client.AddEventHandler(func(evt any) {
		if e, ok := stateEvent(evt); ok {
			s.events.Push(e)
		}
	})
	return client
}

// replaceClient disconnects the current client, if any, and installs c. Callers hold s.mu.
func (s *Session) replaceClient(c *whatsmeow.Client) {
	if s.client != nil && s.client != c {
		s.client.Disconnect()
	}
	s.client = c
}

// wipe closes the store and deletes everything in DataDir. Callers hold s.mu.
func (s *Session) wipe() error {
	if s.container != nil {
		if err := s.container.Close(); err != nil {
			s.log.Warn("close whatsapp store", "err", err)
		}
		s.container = nil
	}
	entries, err := os.ReadDir(s.deps.DataDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(s.deps.DataDir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}
