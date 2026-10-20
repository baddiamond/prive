package whatsapp

import (
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"

	"github.com/baddiamond/prive/internal/core"
)

// stateEvent maps a whatsmeow connection event to an account state change.
// The second result is false for events that do not change the connection state.
func stateEvent(evt any) (core.AccountStateEvent, bool) {
	switch evt.(type) {
	case *events.Connected, *events.KeepAliveRestored:
		return core.AccountStateEvent{State: core.StateConnected}, true
	case *events.Disconnected, *events.KeepAliveTimeout:
		// whatsmeow reconnects on its own with backoff.
		return core.AccountStateEvent{State: core.StateReconnecting}, true
	case *events.LoggedOut:
		return core.AccountStateEvent{State: core.StateLoggedOut, Code: ErrLoggedOutRemotely.Code}, true
	case *events.StreamReplaced:
		return core.AccountStateEvent{State: core.StateError, Code: ErrStreamReplaced.Code}, true
	case *events.ClientOutdated:
		return core.AccountStateEvent{State: core.StateError, Code: ErrClientOutdated.Code}, true
	case *events.TemporaryBan:
		return core.AccountStateEvent{State: core.StateError, Code: ErrTemporaryBan.Code}, true
	case *events.ConnectFailure, *events.StreamError:
		return core.AccountStateEvent{State: core.StateError, Code: core.CodeConnectFailed}, true
	}
	return core.AccountStateEvent{}, false
}

// qrResult is what one item of whatsmeow's QR channel means for Privé.
type qrResult struct {
	// event is emitted to the UI; nil means nothing to report.
	event core.Event
	// done is true when pairing is over, successfully or not.
	done bool
}

// qrEvent maps an item of whatsmeow's QR channel. remoteID and pushName are read only on success.
func qrEvent(item whatsmeow.QRChannelItem, now time.Time, linked func() (remoteID, pushName string)) qrResult {
	switch item.Event {
	case whatsmeow.QRChannelEventCode:
		return qrResult{event: core.LoginQREvent{Code: item.Code, ExpiresAt: now.Add(item.Timeout)}}
	case whatsmeow.QRChannelSuccess.Event:
		id, name := linked()
		return qrResult{event: core.LoginSuccessEvent{RemoteID: id, DisplayName: name}, done: true}
	case whatsmeow.QRChannelTimeout.Event:
		return qrResult{event: core.LoginFailedEvent{Code: ErrQRTimeout.Code}, done: true}
	case whatsmeow.QRChannelClientOutdated.Event:
		return qrResult{event: core.LoginFailedEvent{Code: ErrClientOutdated.Code}, done: true}
	case whatsmeow.QRChannelScannedWithoutMultidevice.Event:
		return qrResult{event: core.LoginFailedEvent{Code: ErrMultideviceRequired.Code}, done: true}
	case whatsmeow.QRChannelEventPasskeyRequest, whatsmeow.QRChannelEventPasskeyResponse:
		return qrResult{} // handled by the phone
	default: // QRChannelEventError and unexpected states
		return qrResult{event: core.LoginFailedEvent{Code: ErrPairingFailed.Code}, done: true}
	}
}
