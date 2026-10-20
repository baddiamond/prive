package core

import "context"

// Repository is the persistence the Manager needs. internal/store implements it with SQLite.
// Implementations must be safe for concurrent use.
type Repository interface {
	// ListAccounts returns all accounts ordered by position.
	ListAccounts(ctx context.Context) ([]Account, error)
	// CreateAccount inserts a new account.
	CreateAccount(ctx context.Context, account Account) error
	// UpdateAccount saves the remote ID, display name, color and position of an existing account.
	UpdateAccount(ctx context.Context, account Account) error
	// DeleteAccount removes an account together with its chats and messages.
	DeleteAccount(ctx context.Context, id AccountID) error

	// UpsertChat inserts or replaces a chat of an account.
	UpsertChat(ctx context.Context, chat Chat) error
	// ListChats returns chats matching filter, most recent first.
	ListChats(ctx context.Context, filter ChatFilter) ([]Chat, error)

	// GetChat returns one stored chat and whether it exists.
	GetChat(ctx context.Context, account AccountID, id ChatID) (Chat, bool, error)
	// PatchChat applies a partial change to an existing chat and returns the result.
	// It reports false, without error, when the chat does not exist.
	PatchChat(ctx context.Context, account AccountID, patch ChatPatch) (Chat, bool, error)
	// UnreadSummary counts unread messages per account, leaving out muted and archived chats.
	UnreadSummary(ctx context.Context) (UnreadSummary, error)
	// ImportHistory stores a history batch in one transaction (see HistoryBatchEvent).
	ImportHistory(ctx context.Context, account AccountID, chats []Chat, msgs []Message, authoritative bool) error
	// ApplyMessage stores a live message, creating its chat (with kind and name) if needed,
	// moves the chat's last message forward and, when countUnread is true and the message is
	// new and incoming, increments the unread count. It returns the chat afterwards.
	ApplyMessage(ctx context.Context, msg Message, kind ChatKind, name string, countUnread bool) (Chat, error)

	// UpsertMessage inserts or updates a message. An existing message keeps the
	// most advanced status (see MergeStatus).
	UpsertMessage(ctx context.Context, msg Message) error
	// UpdateMessageStatus applies a status change to a stored message, following MergeStatus.
	// A missing message is not an error.
	UpdateMessageStatus(ctx context.Context, account AccountID, chat ChatID, id MessageID, status MessageStatus) error
	// ListMessages returns a page of a chat's stored history in chronological order.
	ListMessages(ctx context.Context, account AccountID, chat ChatID, opts GetMessagesOptions) ([]Message, error)
	// OldestMessage returns the oldest stored message of a chat; found is false when the
	// chat has none.
	OldestMessage(ctx context.Context, account AccountID, chat ChatID) (msg Message, found bool, err error)
}
