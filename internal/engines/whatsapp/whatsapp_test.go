package whatsapp

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"

	"github.com/baddiamond/prive/internal/core"
	"github.com/baddiamond/prive/internal/engines/enginetest"
)

// TestConformance runs the shared suite without network access. The logged-in checks are
// skipped: pairing needs a real phone (see README.md for the manual test).
func TestConformance(t *testing.T) {
	enginetest.Run(t, New(), enginetest.Options{})
}

func TestStateEvents(t *testing.T) {
	cases := []struct {
		evt   any
		state core.AccountState
		code  string
	}{
		{&events.Connected{}, core.StateConnected, ""},
		{&events.KeepAliveRestored{}, core.StateConnected, ""},
		{&events.Disconnected{}, core.StateReconnecting, ""},
		{&events.KeepAliveTimeout{}, core.StateReconnecting, ""},
		{&events.LoggedOut{}, core.StateLoggedOut, ErrLoggedOutRemotely.Code},
		{&events.StreamReplaced{}, core.StateError, ErrStreamReplaced.Code},
		{&events.ClientOutdated{}, core.StateError, ErrClientOutdated.Code},
		{&events.TemporaryBan{}, core.StateError, ErrTemporaryBan.Code},
		{&events.ConnectFailure{}, core.StateError, core.CodeConnectFailed},
	}
	for _, c := range cases {
		got, ok := stateEvent(c.evt)
		if !ok || got.State != c.state || got.Code != c.code {
			t.Errorf("stateEvent(%T) = %+v, %v; want %s/%q", c.evt, got, ok, c.state, c.code)
		}
	}
	if _, ok := stateEvent(&events.Message{}); ok {
		t.Error("a message must not change the connection state")
	}
}

func TestQREvents(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	linked := func() (string, string) { return "remote-id", "Push name" }

	code := qrEvent(whatsmeow.QRChannelItem{Event: whatsmeow.QRChannelEventCode, Code: "2@abc", Timeout: 60 * time.Second}, now, linked)
	qr, ok := code.event.(core.LoginQREvent)
	if !ok || code.done || qr.Code != "2@abc" || !qr.ExpiresAt.Equal(now.Add(time.Minute)) {
		t.Errorf("code item = %+v", code)
	}

	success := qrEvent(whatsmeow.QRChannelSuccess, now, linked)
	if s, ok := success.event.(core.LoginSuccessEvent); !ok || !success.done || s.RemoteID != "remote-id" || s.DisplayName != "Push name" {
		t.Errorf("success item = %+v", success)
	}

	failures := map[string]*core.CodedError{
		whatsmeow.QRChannelTimeout.Event:                   ErrQRTimeout,
		whatsmeow.QRChannelClientOutdated.Event:            ErrClientOutdated,
		whatsmeow.QRChannelScannedWithoutMultidevice.Event: ErrMultideviceRequired,
		whatsmeow.QRChannelErrUnexpectedEvent.Event:        ErrPairingFailed,
		whatsmeow.QRChannelEventError:                      ErrPairingFailed,
	}
	for event, want := range failures {
		res := qrEvent(whatsmeow.QRChannelItem{Event: event, Error: errors.New("x")}, now, linked)
		f, ok := res.event.(core.LoginFailedEvent)
		if !ok || !res.done || f.Code != want.Code {
			t.Errorf("%s = %+v, want failure %s", event, res, want.Code)
		}
	}
}

func TestRedact(t *testing.T) {
	line := redact("failed to send to 34600112233@s.whatsapp.net (device 3)")
	if strings.Contains(line, "34600112233") || !strings.Contains(line, "device 3") {
		t.Errorf("redact = %q", line)
	}
}

func TestManifest(t *testing.T) {
	m := New().Manifest()
	if m.ID != ID || !m.SupportsLogin(core.LoginQR) {
		t.Errorf("manifest = %+v", m)
	}
	// Declared capabilities must be implemented: text, receipts and on-demand history.
	if m.Capabilities != (core.Capabilities{SendText: true, ReadReceipts: true, HistoryOnDemand: true}) {
		t.Errorf("capabilities = %+v", m.Capabilities)
	}
	var _ core.HistoryRequester = (*Session)(nil)
	var _ core.ReadMarker = (*Session)(nil)
}

func TestCodesHaveTranslations(t *testing.T) {
	raw, err := os.ReadFile("../../../frontend/src/locales/en.json")
	if err != nil {
		t.Fatal(err)
	}
	var locale map[string]any
	if err := json.Unmarshal(raw, &locale); err != nil {
		t.Fatal(err)
	}
	for _, e := range Codes() {
		node := any(locale)
		for _, part := range strings.Split(e.Code, ".") {
			m, _ := node.(map[string]any)
			node = m[part]
		}
		if _, ok := node.(string); !ok {
			t.Errorf("code %q has no translation in en.json", e.Code)
		}
	}
}
