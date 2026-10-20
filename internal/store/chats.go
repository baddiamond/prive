package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/baddiamond/prive/internal/core"
)

// defaultChatLimit caps ListChats when the filter sets no limit.
const defaultChatLimit = 500

// UpsertChat implements core.Repository.
func (s *Store) UpsertChat(ctx context.Context, c core.Chat) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO chats (account_id, id, name, kind, unread_count, last_message_preview,
		                   last_message_at, muted, archived)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (account_id, id) DO UPDATE SET
			name = excluded.name,
			kind = excluded.kind,
			unread_count = excluded.unread_count,
			last_message_preview = excluded.last_message_preview,
			last_message_at = excluded.last_message_at,
			muted = excluded.muted,
			archived = excluded.archived`,
		c.AccountID, c.ID, c.Name, string(c.Kind), c.UnreadCount, c.LastMessagePreview,
		toMillis(c.LastMessageAt), boolToInt(c.Muted), boolToInt(c.Archived))
	if err != nil {
		return fmt.Errorf("upsert chat: %w", err)
	}
	return nil
}

// ListChats implements core.Repository.
func (s *Store) ListChats(ctx context.Context, f core.ChatFilter) ([]core.Chat, error) {
	var (
		where []string
		args  []any
	)
	if f.AccountID != "" {
		where = append(where, "account_id = ?")
		args = append(args, f.AccountID)
	}
	if !f.IncludeArchived {
		where = append(where, "archived = 0")
	}
	if !f.Before.IsZero() {
		where = append(where, "last_message_at < ?")
		args = append(args, toMillis(f.Before))
	}
	limit := f.Limit
	if limit <= 0 {
		limit = defaultChatLimit
	}

	query := `SELECT account_id, id, name, kind, unread_count, last_message_preview,
	                 last_message_at, muted, archived
	          FROM chats`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY last_message_at DESC, account_id, id LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list chats: %w", err)
	}
	defer rows.Close()

	var out []core.Chat
	for rows.Next() {
		var (
			c               core.Chat
			kind            string
			lastAt          int64
			muted, archived int
		)
		if err := rows.Scan(&c.AccountID, &c.ID, &c.Name, &kind, &c.UnreadCount,
			&c.LastMessagePreview, &lastAt, &muted, &archived); err != nil {
			return nil, fmt.Errorf("list chats: %w", err)
		}
		c.Kind = core.ChatKind(kind)
		c.LastMessageAt = fromMillis(lastAt)
		c.Muted, c.Archived = muted != 0, archived != 0
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list chats: %w", err)
	}
	return out, nil
}
