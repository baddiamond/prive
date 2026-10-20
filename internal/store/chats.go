package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/baddiamond/prive/internal/core"
)

// defaultChatLimit caps ListChats when the filter sets no limit.
const defaultChatLimit = 2000

// maxPreviewRunes caps the last message preview stored with each chat.
const maxPreviewRunes = 200

const chatColumns = `account_id, id, name, kind, unread_count, last_message_preview, last_message_at,
	last_message_kind, last_message_from_me, muted, archived, pinned`

// rowScanner is satisfied by *sql.Row and *sql.Rows.
type rowScanner interface{ Scan(dest ...any) error }

func scanChat(r rowScanner) (core.Chat, error) {
	var (
		c                       core.Chat
		kind, lastKind          string
		lastAt                  int64
		fromMe, muted, archived int
		pinned                  int
	)
	err := r.Scan(&c.AccountID, &c.ID, &c.Name, &kind, &c.UnreadCount, &c.LastMessagePreview, &lastAt,
		&lastKind, &fromMe, &muted, &archived, &pinned)
	if err != nil {
		return core.Chat{}, err
	}
	c.Kind = core.ChatKind(kind)
	c.LastMessageAt = fromMillis(lastAt)
	c.LastMessageKind = core.MessageKind(lastKind)
	c.LastMessageFromMe = fromMe != 0
	c.Muted, c.Archived, c.Pinned = muted != 0, archived != 0, pinned != 0
	return c, nil
}

// querier is satisfied by *sql.DB and *sql.Tx.
type querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func upsertChat(ctx context.Context, q querier, c core.Chat) error {
	_, err := q.ExecContext(ctx, `
		INSERT INTO chats (`+chatColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (account_id, id) DO UPDATE SET
			name = excluded.name,
			kind = excluded.kind,
			unread_count = excluded.unread_count,
			last_message_preview = excluded.last_message_preview,
			last_message_at = excluded.last_message_at,
			last_message_kind = excluded.last_message_kind,
			last_message_from_me = excluded.last_message_from_me,
			muted = excluded.muted,
			archived = excluded.archived,
			pinned = excluded.pinned`,
		c.AccountID, c.ID, c.Name, string(c.Kind), c.UnreadCount, preview(c.LastMessagePreview),
		toMillis(c.LastMessageAt), string(c.LastMessageKind), boolToInt(c.LastMessageFromMe),
		boolToInt(c.Muted), boolToInt(c.Archived), boolToInt(c.Pinned))
	if err != nil {
		return fmt.Errorf("upsert chat: %w", err)
	}
	return nil
}

func getChat(ctx context.Context, q querier, account core.AccountID, id core.ChatID) (core.Chat, bool, error) {
	c, err := scanChat(q.QueryRowContext(ctx,
		`SELECT `+chatColumns+` FROM chats WHERE account_id = ? AND id = ?`, account, id))
	if errors.Is(err, sql.ErrNoRows) {
		return core.Chat{}, false, nil
	}
	if err != nil {
		return core.Chat{}, false, fmt.Errorf("read chat: %w", err)
	}
	return c, true, nil
}

// UpsertChat implements core.Repository.
func (s *Store) UpsertChat(ctx context.Context, c core.Chat) error {
	return upsertChat(ctx, s.db, c)
}

// GetChat implements core.Repository.
func (s *Store) GetChat(ctx context.Context, account core.AccountID, id core.ChatID) (core.Chat, bool, error) {
	return getChat(ctx, s.db, account, id)
}

// PatchChat implements core.Repository.
func (s *Store) PatchChat(ctx context.Context, account core.AccountID, p core.ChatPatch) (core.Chat, bool, error) {
	var (
		out   core.Chat
		found bool
	)
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		c, ok, err := getChat(ctx, tx, account, p.ChatID)
		if err != nil || !ok {
			return err
		}
		if p.Name != nil {
			c.Name = *p.Name
		}
		if p.Muted != nil {
			c.Muted = *p.Muted
		}
		if p.Archived != nil {
			c.Archived = *p.Archived
		}
		if p.Pinned != nil {
			c.Pinned = *p.Pinned
		}
		if p.Read != nil {
			if *p.Read {
				c.UnreadCount = 0
			} else if c.UnreadCount == 0 {
				c.UnreadCount = 1 // "mark as unread"
			}
		}
		out, found = c, true
		return upsertChat(ctx, tx, c)
	})
	return out, found, err
}

// ListChats implements core.Repository: pinned chats first, then the most recent.
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

	query := `SELECT ` + chatColumns + ` FROM chats`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY pinned DESC, last_message_at DESC, account_id, id LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list chats: %w", err)
	}
	defer rows.Close()

	var out []core.Chat
	for rows.Next() {
		c, err := scanChat(rows)
		if err != nil {
			return nil, fmt.Errorf("list chats: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list chats: %w", err)
	}
	return out, nil
}

// UnreadSummary implements core.Repository.
func (s *Store) UnreadSummary(ctx context.Context) (core.UnreadSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT account_id, SUM(unread_count) FROM chats
		WHERE muted = 0 AND archived = 0 AND unread_count > 0
		GROUP BY account_id`)
	if err != nil {
		return core.UnreadSummary{}, fmt.Errorf("unread summary: %w", err)
	}
	defer rows.Close()
	sum := core.UnreadSummary{PerAccount: map[core.AccountID]int{}}
	for rows.Next() {
		var (
			id core.AccountID
			n  int
		)
		if err := rows.Scan(&id, &n); err != nil {
			return core.UnreadSummary{}, fmt.Errorf("unread summary: %w", err)
		}
		sum.PerAccount[id] = n
		sum.Total += n
	}
	return sum, rows.Err()
}

// preview trims a message text for the chat list.
func preview(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	if r := []rune(text); len(r) > maxPreviewRunes {
		return string(r[:maxPreviewRunes])
	}
	return text
}
