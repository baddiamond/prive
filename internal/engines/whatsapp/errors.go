package whatsapp

import "github.com/baddiamond/prive/internal/core"

// Error codes specific to WhatsApp. They are translation keys (ADR 0004).
var (
	// ErrQRTimeout means no QR code was scanned before WhatsApp stopped offering new ones.
	ErrQRTimeout = core.NewError("errors.whatsapp.qrTimeout")
	// ErrPairingFailed means the phone scanned the code but linking did not complete.
	ErrPairingFailed = core.NewError("errors.whatsapp.pairingFailed")
	// ErrClientOutdated means WhatsApp rejected this version of the protocol; Privé needs an update.
	ErrClientOutdated = core.NewError("errors.whatsapp.clientOutdated")
	// ErrMultideviceRequired means the phone's WhatsApp is too old to link devices.
	ErrMultideviceRequired = core.NewError("errors.whatsapp.multideviceRequired")
	// ErrAlreadyLinked means Login was called on an account that is already linked.
	ErrAlreadyLinked = core.NewError("errors.whatsapp.alreadyLinked")
	// ErrLoggedOutRemotely means the device was removed from the phone (Linked devices).
	ErrLoggedOutRemotely = core.NewError("errors.whatsapp.loggedOutRemotely")
	// ErrStreamReplaced means another client connected with the same session.
	ErrStreamReplaced = core.NewError("errors.whatsapp.streamReplaced")
	// ErrTemporaryBan means WhatsApp temporarily blocked this account.
	ErrTemporaryBan = core.NewError("errors.whatsapp.temporaryBan")
	// ErrUnlinkOnPhone means local data was removed while offline, so the device may still
	// appear on the phone until the user removes it there.
	ErrUnlinkOnPhone = core.NewError("errors.whatsapp.unlinkOnPhone")
)

// Codes lists every code this engine may send to the UI, for the translation check in tests.
func Codes() []*core.CodedError {
	return []*core.CodedError{
		ErrQRTimeout, ErrPairingFailed, ErrClientOutdated, ErrMultideviceRequired, ErrAlreadyLinked,
		ErrLoggedOutRemotely, ErrStreamReplaced, ErrTemporaryBan, ErrUnlinkOnPhone,
	}
}
