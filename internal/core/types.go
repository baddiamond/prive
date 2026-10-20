package core

import "time"

// AccountID is the local, stable identifier Privé assigns to an account (a UUID).
type AccountID string

// ChatID identifies a chat within one account. Its format belongs to the engine.
type ChatID string

// MessageID identifies a message within one chat. Its format belongs to the engine.
type MessageID string

// Account is one login on one network. A user may have any number of them, including
// several on the same network.
type Account struct {
	ID       AccountID `json:"id"`
	EngineID string    `json:"engineId"`
	// RemoteID is the network-side identity (e.g. a WhatsApp JID); empty until logged in.
	RemoteID string `json:"remoteId"`
	// DisplayName is a user-editable label such as "Work".
	DisplayName string `json:"displayName"`
	// Color is a UI accent token used to tell accounts apart.
	Color string `json:"color"`
	// Position orders accounts in the UI.
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"createdAt"`
}

// ChatKind tells one-to-one conversations from groups.
type ChatKind string

// Chat kinds.
const (
	ChatDirect ChatKind = "direct"
	ChatGroup  ChatKind = "group"
)

// Chat is a conversation as shown in the inbox.
type Chat struct {
	// AccountID is filled in by the Manager; engines may leave it empty.
	AccountID          AccountID `json:"accountId"`
	ID                 ChatID    `json:"id"`
	Name               string    `json:"name"`
	Kind               ChatKind  `json:"kind"`
	UnreadCount        int       `json:"unreadCount"`
	LastMessagePreview string    `json:"lastMessagePreview"`
	// LastMessageAt is the zero time when the chat has no messages.
	LastMessageAt time.Time `json:"lastMessageAt"`
	// LastMessageKind and LastMessageFromMe let the UI render a translated preview
	// ("Photo", "You: …") without the backend producing text.
	LastMessageKind   MessageKind `json:"lastMessageKind"`
	LastMessageFromMe bool        `json:"lastMessageFromMe"`
	Muted             bool        `json:"muted"`
	Archived          bool        `json:"archived"`
	Pinned            bool        `json:"pinned"`
}

// MessageKind is the type of content a message carries.
type MessageKind string

// Message kinds. Media content itself arrives with media support; until then the kind
// lets the UI show what a message is ("Photo", "Voice message").
const (
	MessageText     MessageKind = "text"
	MessageImage    MessageKind = "image"
	MessageVideo    MessageKind = "video"
	MessageAudio    MessageKind = "audio"
	MessageVoice    MessageKind = "voice"
	MessageDocument MessageKind = "document"
	MessageSticker  MessageKind = "sticker"
	MessageLocation MessageKind = "location"
	MessageContact  MessageKind = "contact"
	MessagePoll     MessageKind = "poll"
	// MessageUnsupported is content Privé cannot display yet.
	MessageUnsupported MessageKind = "unsupported"
)

// MessageStatus is the delivery state of a message.
type MessageStatus string

// Message statuses, in delivery order (except Failed).
const (
	StatusPending   MessageStatus = "pending"
	StatusSent      MessageStatus = "sent"
	StatusDelivered MessageStatus = "delivered"
	StatusRead      MessageStatus = "read"
	StatusFailed    MessageStatus = "failed"
)

// MergeStatus returns the status to keep when a message already in state current receives
// next. Delivery only moves forward (a late "delivered" never downgrades "read"), while
// Failed only replaces a status that has not reached the network yet.
func MergeStatus(current, next MessageStatus) MessageStatus {
	if next == StatusFailed {
		if current == "" || current == StatusPending {
			return StatusFailed
		}
		return current
	}
	if statusRank(next) > statusRank(current) {
		return next
	}
	return current
}

func statusRank(s MessageStatus) int {
	switch s {
	case StatusPending:
		return 1
	case StatusSent:
		return 2
	case StatusDelivered:
		return 3
	case StatusRead:
		return 4
	default: // "" and StatusFailed: any real status may replace them
		return 0
	}
}

// Message is one message in a chat.
type Message struct {
	// AccountID is filled in by the Manager; engines may leave it empty.
	AccountID AccountID `json:"accountId"`
	ChatID    ChatID    `json:"chatId"`
	ID        MessageID `json:"id"`
	// SenderID is the network identity of the author.
	SenderID   string        `json:"senderId"`
	SenderName string        `json:"senderName"`
	FromMe     bool          `json:"fromMe"`
	Timestamp  time.Time     `json:"timestamp"`
	Kind       MessageKind   `json:"kind"`
	Text       string        `json:"text"`
	Status     MessageStatus `json:"status"`
	// ReplyTo is the ID of the quoted message, if any.
	ReplyTo MessageID `json:"replyTo,omitempty"`
}

// OutgoingMessage is what the user asks to send.
type OutgoingMessage struct {
	Text    string    `json:"text"`
	ReplyTo MessageID `json:"replyTo,omitempty"`
}

// ListChatsOptions pages through a session's chats, most recent first.
type ListChatsOptions struct {
	// Limit caps the number of chats; 0 means the engine's default.
	Limit int
	// Before, if set, returns only chats whose last message is older than this time.
	Before time.Time
}

// GetMessagesOptions pages backwards through a chat's history.
type GetMessagesOptions struct {
	// Limit caps the number of messages; 0 means the implementation's default.
	Limit int
	// Before, if set, returns only messages older than this time.
	Before time.Time
}

// ChatPatch changes some fields of a stored chat. Nil fields are left as they are.
type ChatPatch struct {
	ChatID   ChatID  `json:"chatId"`
	Name     *string `json:"name,omitempty"`
	Muted    *bool   `json:"muted,omitempty"`
	Archived *bool   `json:"archived,omitempty"`
	Pinned   *bool   `json:"pinned,omitempty"`
	// Read true clears the unread count; false marks the chat unread (at least 1).
	Read *bool `json:"read,omitempty"`
}

// UnreadSummary counts unread messages for the rail badges. Muted and archived chats
// are not counted.
type UnreadSummary struct {
	Total      int               `json:"total"`
	PerAccount map[AccountID]int `json:"perAccount"`
}

// ChatFilter selects chats from local storage for the inbox.
type ChatFilter struct {
	// AccountID limits the result to one account; empty means the unified inbox.
	AccountID AccountID
	// IncludeArchived also returns archived chats.
	IncludeArchived bool
	// Limit caps the number of chats; 0 means the store's default.
	Limit int
	// Before, if set, returns only chats whose last message is older than this time.
	Before time.Time
}
