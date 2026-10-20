package core

import "time"

// Event is something that happened in a session. Sessions emit the event types in
// this file; the Manager persists what is needed, stamps the AccountID and forwards
// each event to the UI under the name returned by EventName.
type Event interface {
	// EventName is the UI event name, e.g. "message:new".
	EventName() string
	// stamped returns a copy of the event that belongs to account id.
	stamped(id AccountID) Event
}

// AccountState is the connection state of an account.
type AccountState string

// Account states. The UI translates them with the key "account.state.<state>".
const (
	StateDisconnected AccountState = "disconnected"
	StateConnecting   AccountState = "connecting"
	StateConnected    AccountState = "connected"
	StateReconnecting AccountState = "reconnecting"
	StateLoggedOut    AccountState = "loggedOut"
	StateError        AccountState = "error"
)

var accountStates = []AccountState{
	StateDisconnected, StateConnecting, StateConnected, StateReconnecting, StateLoggedOut, StateError,
}

// Key returns the translation key of the state.
func (s AccountState) Key() string { return "account.state." + string(s) }

// AccountStateEvent reports a connection state change.
type AccountStateEvent struct {
	AccountID AccountID    `json:"accountId"`
	State     AccountState `json:"state"`
	// Code is an optional error translation key explaining the state.
	Code string `json:"code,omitempty"`
}

// LoginQREvent carries a new QR code to display during pairing.
type LoginQREvent struct {
	AccountID AccountID `json:"accountId"`
	// Code is the QR payload (not a translation key).
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// LoginSuccessEvent reports that pairing completed.
type LoginSuccessEvent struct {
	AccountID AccountID `json:"accountId"`
	RemoteID  string    `json:"remoteId"`
	// DisplayName is the network-side name, used if the user has not chosen one.
	DisplayName string `json:"displayName"`
}

// LoginFailedEvent reports that an interactive login ended without success.
type LoginFailedEvent struct {
	AccountID AccountID `json:"accountId"`
	Code      string    `json:"code"`
}

// ChatUpdatedEvent carries the full, current state of a chat (new chat, rename,
// unread count, last message…).
type ChatUpdatedEvent struct {
	AccountID AccountID `json:"accountId"`
	Chat      Chat      `json:"chat"`
}

// MessageEvent carries a new incoming or outgoing message, or a new version of one.
type MessageEvent struct {
	AccountID AccountID `json:"accountId"`
	Message   Message   `json:"message"`
}

// MessageStatusEvent reports a delivery status change.
type MessageStatusEvent struct {
	AccountID AccountID     `json:"accountId"`
	ChatID    ChatID        `json:"chatId"`
	MessageID MessageID     `json:"messageId"`
	Status    MessageStatus `json:"status"`
}

// SyncProgressEvent reports progress of the initial history sync.
type SyncProgressEvent struct {
	AccountID AccountID `json:"accountId"`
	Done      int       `json:"done"`
	Total     int       `json:"total"`
}

// AccountRemovedEvent is emitted by the Manager (never by sessions) after an account is deleted.
type AccountRemovedEvent struct {
	AccountID AccountID `json:"accountId"`
}

// EventName implements Event.
func (AccountStateEvent) EventName() string { return "account:state" }

// EventName implements Event.
func (LoginQREvent) EventName() string { return "login:qr" }

// EventName implements Event.
func (LoginSuccessEvent) EventName() string { return "login:success" }

// EventName implements Event.
func (LoginFailedEvent) EventName() string { return "login:failed" }

// EventName implements Event.
func (ChatUpdatedEvent) EventName() string { return "chat:updated" }

// EventName implements Event.
func (MessageEvent) EventName() string { return "message:new" }

// EventName implements Event.
func (MessageStatusEvent) EventName() string { return "message:status" }

// EventName implements Event.
func (SyncProgressEvent) EventName() string { return "sync:progress" }

// EventName implements Event.
func (AccountRemovedEvent) EventName() string { return "account:removed" }

func (e AccountStateEvent) stamped(id AccountID) Event { e.AccountID = id; return e }
func (e LoginQREvent) stamped(id AccountID) Event      { e.AccountID = id; return e }
func (e LoginSuccessEvent) stamped(id AccountID) Event { e.AccountID = id; return e }
func (e LoginFailedEvent) stamped(id AccountID) Event  { e.AccountID = id; return e }
func (e SyncProgressEvent) stamped(id AccountID) Event { e.AccountID = id; return e }
func (e AccountRemovedEvent) stamped(id AccountID) Event {
	e.AccountID = id
	return e
}

func (e ChatUpdatedEvent) stamped(id AccountID) Event {
	e.AccountID = id
	e.Chat.AccountID = id
	return e
}

func (e MessageEvent) stamped(id AccountID) Event {
	e.AccountID = id
	e.Message.AccountID = id
	return e
}

func (e MessageStatusEvent) stamped(id AccountID) Event { e.AccountID = id; return e }
