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

	// UpsertMessage inserts or updates a message. An existing message keeps the
	// most advanced status (see MergeStatus).
	UpsertMessage(ctx context.Context, msg Message) error
	// UpdateMessageStatus applies a status change to a stored message, following MergeStatus.
	// A missing message is not an error.
	UpdateMessageStatus(ctx context.Context, account AccountID, chat ChatID, id MessageID, status MessageStatus) error
	// ListMessages returns a page of a chat's stored history in chronological order.
	ListMessages(ctx context.Context, account AccountID, chat ChatID, opts GetMessagesOptions) ([]Message, error)
}
