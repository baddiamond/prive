package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/baddiamond/prive/internal/core"
)

// ApplyMessage implements core.Repository.
func (s *Store) ApplyMessage(ctx context.Context, m core.Message, kind core.ChatKind, name string,
	countUnread bool) (core.Chat, error) {
	var out core.Chat
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		existing, err := messageStatus(ctx, tx, m.AccountID, m.ChatID, m.ID)
		if err != nil {
			return err
		}
		isNew := existing == ""
		if err := upsertMessage(ctx, tx, m, existing); err != nil {
			return err
		}

		chat, ok, err := getChat(ctx, tx, m.AccountID, m.ChatID)
		if err != nil {
			return err
		}
		if !ok {
			if kind == "" {
				kind = core.ChatDirect
			}
			chat = core.Chat{AccountID: m.AccountID, ID: m.ChatID, Name: name, Kind: kind}
		}
		if !m.Timestamp.Before(chat.LastMessageAt) {
			if countUnread && isNew && !m.FromMe && m.Timestamp.After(chat.LastMessageAt) {
				chat.UnreadCount++
			}
			setLastMessage(&chat, m)
		}
		out = chat
		return upsertChat(ctx, tx, chat)
	})
	return out, err
}

// ImportHistory implements core.Repository. Authoritative batches replace the chat state
// (name, unread count, flags); others only add messages and create missing chats.
func (s *Store) ImportHistory(ctx context.Context, account core.AccountID, chats []core.Chat,
	msgs []core.Message, authoritative bool) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		for _, m := range msgs {
			existing, err := messageStatus(ctx, tx, account, m.ChatID, m.ID)
			if err != nil {
				return err
			}
			m.AccountID = account
			if err := upsertMessage(ctx, tx, m, existing); err != nil {
				return err
			}
		}
		for _, c := range chats {
			c.AccountID = account
			current, ok, err := getChat(ctx, tx, account, c.ID)
			if err != nil {
				return err
			}
			if ok && !authoritative {
				continue // older messages only: keep the newer chat state
			}
			if ok && c.Name == "" {
				c.Name = current.Name
			}
			if err := s.refreshLastMessage(ctx, tx, &c); err != nil {
				return err
			}
			if ok && current.LastMessageAt.After(c.LastMessageAt) {
				// A live message arrived after this snapshot was taken.
				c.LastMessageAt, c.LastMessagePreview = current.LastMessageAt, current.LastMessagePreview
				c.LastMessageKind, c.LastMessageFromMe = current.LastMessageKind, current.LastMessageFromMe
			}
			if err := upsertChat(ctx, tx, c); err != nil {
				return err
			}
		}
		return nil
	})
}

// refreshLastMessage fills the chat's last message from its newest stored message, keeping
// the chat's own timestamp when it is newer (e.g. a chat whose messages were not synced).
func (s *Store) refreshLastMessage(ctx context.Context, tx *sql.Tx, c *core.Chat) error {
	var (
		text, kind string
		fromMe     int
		ts         int64
	)
	err := tx.QueryRowContext(ctx, `
		SELECT text, kind, from_me, timestamp FROM messages
		WHERE account_id = ? AND chat_id = ? ORDER BY timestamp DESC, id DESC LIMIT 1`,
		c.AccountID, c.ID).Scan(&text, &kind, &fromMe, &ts)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read last message: %w", err)
	}
	at := fromMillis(ts)
	c.LastMessagePreview = preview(text)
	c.LastMessageKind = core.MessageKind(kind)
	c.LastMessageFromMe = fromMe != 0
	if at.After(c.LastMessageAt) {
		c.LastMessageAt = at
	}
	return nil
}

func setLastMessage(c *core.Chat, m core.Message) {
	c.LastMessageAt = m.Timestamp
	c.LastMessagePreview = preview(m.Text)
	c.LastMessageKind = m.Kind
	c.LastMessageFromMe = m.FromMe
}
