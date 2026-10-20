package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"github.com/baddiamond/prive/internal/core"
)

// defaultMessageLimit caps ListMessages when the options set no limit.
const defaultMessageLimit = 50

// UpsertMessage implements core.Repository. An existing message keeps the most advanced
// status, so a late event cannot move it backwards.
func (s *Store) UpsertMessage(ctx context.Context, m core.Message) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		current, err := messageStatus(ctx, tx, m.AccountID, m.ChatID, m.ID)
		if err != nil {
			return err
		}
		return upsertMessage(ctx, tx, m, current)
	})
}

// upsertMessage writes m; current is the stored status ("" when the message is new).
func upsertMessage(ctx context.Context, tx *sql.Tx, m core.Message, current core.MessageStatus) error {
	status := core.MergeStatus(current, m.Status)
	_, err := tx.ExecContext(ctx, `
		INSERT INTO messages (account_id, chat_id, id, sender_id, sender_name, from_me,
		                      timestamp, kind, text, status, reply_to)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (account_id, chat_id, id) DO UPDATE SET
			sender_id = excluded.sender_id,
			sender_name = excluded.sender_name,
			from_me = excluded.from_me,
			timestamp = excluded.timestamp,
			kind = excluded.kind,
			text = excluded.text,
			status = excluded.status,
			reply_to = excluded.reply_to`,
		m.AccountID, m.ChatID, m.ID, m.SenderID, m.SenderName, boolToInt(m.FromMe),
		toMillis(m.Timestamp), string(m.Kind), m.Text, string(status), string(m.ReplyTo))
	if err != nil {
		return fmt.Errorf("upsert message: %w", err)
	}
	return nil
}

// UpdateMessageStatus implements core.Repository.
func (s *Store) UpdateMessageStatus(ctx context.Context, account core.AccountID, chat core.ChatID,
	id core.MessageID, status core.MessageStatus) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		current, err := messageStatus(ctx, tx, account, chat, id)
		if err != nil || current == "" {
			return err // a status for a message we never stored is ignored
		}
		next := core.MergeStatus(current, status)
		if next == current {
			return nil
		}
		_, err = tx.ExecContext(ctx,
			`UPDATE messages SET status = ? WHERE account_id = ? AND chat_id = ? AND id = ?`,
			string(next), account, chat, id)
		if err != nil {
			return fmt.Errorf("update message status: %w", err)
		}
		return nil
	})
}

// ListMessages implements core.Repository: the newest page before opts.Before,
// returned oldest first.
func (s *Store) ListMessages(ctx context.Context, account core.AccountID, chat core.ChatID,
	opts core.GetMessagesOptions) ([]core.Message, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultMessageLimit
	}
	before := int64(1<<63 - 1)
	if !opts.Before.IsZero() {
		before = toMillis(opts.Before)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT account_id, chat_id, id, sender_id, sender_name, from_me, timestamp,
		       kind, text, status, reply_to
		FROM messages
		WHERE account_id = ? AND chat_id = ? AND timestamp < ?
		ORDER BY timestamp DESC, id DESC
		LIMIT ?`, account, chat, before, limit)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	var out []core.Message
	for rows.Next() {
		var (
			m                     core.Message
			fromMe                int
			ts                    int64
			kind, status, replyTo string
		)
		if err := rows.Scan(&m.AccountID, &m.ChatID, &m.ID, &m.SenderID, &m.SenderName, &fromMe,
			&ts, &kind, &m.Text, &status, &replyTo); err != nil {
			return nil, fmt.Errorf("list messages: %w", err)
		}
		m.FromMe = fromMe != 0
		m.Timestamp = fromMillis(ts)
		m.Kind = core.MessageKind(kind)
		m.Status = core.MessageStatus(status)
		m.ReplyTo = core.MessageID(replyTo)
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	slices.Reverse(out)
	return out, nil
}

// messageStatus returns the stored status of a message, or "" if it does not exist.
func messageStatus(ctx context.Context, tx *sql.Tx, account core.AccountID, chat core.ChatID,
	id core.MessageID) (core.MessageStatus, error) {
	var status string
	err := tx.QueryRowContext(ctx,
		`SELECT status FROM messages WHERE account_id = ? AND chat_id = ? AND id = ?`,
		account, chat, id).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read message status: %w", err)
	}
	return core.MessageStatus(status), nil
}
