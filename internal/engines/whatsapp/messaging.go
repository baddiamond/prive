package whatsapp

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"github.com/baddiamond/prive/internal/core"
)

// SendMessage implements core.Session: it sends a text message and returns it as sent.
// Delivery and read receipts arrive later as MessageStatusEvent (see syncer.receipt).
func (s *Session) SendMessage(ctx context.Context, chatID core.ChatID, out core.OutgoingMessage) (core.Message, error) {
	client, err := s.loggedInClient()
	if err != nil {
		return core.Message{}, err
	}
	jid, err := types.ParseJID(string(chatID))
	if err != nil || !isChat(jid) {
		return core.Message{}, core.ErrSendFailed.Wrap(err)
	}
	resp, err := client.SendMessage(ctx, jid, &waE2E.Message{Conversation: proto.String(out.Text)})
	if err != nil {
		return core.Message{}, core.ErrSendFailed.Wrap(err)
	}
	m := core.Message{
		ChatID:    chatID,
		ID:        core.MessageID(resp.ID),
		FromMe:    true,
		Timestamp: resp.Timestamp,
		Kind:      core.MessageText,
		Text:      out.Text,
		Status:    core.StatusSent,
	}
	if own := client.Store.GetJID(); !own.IsEmpty() {
		m.SenderID = own.ToNonAD().String()
	}
	return m, nil
}

// MarkRead implements core.ReadMarker: it sends read receipts for incoming messages. If the
// user turned read receipts off in WhatsApp, whatsmeow only syncs the read state to their
// own devices. Group receipts name each message's sender, so they go out per sender.
func (s *Session) MarkRead(ctx context.Context, chatID core.ChatID, messages []core.Message) error {
	client, err := s.loggedInClient()
	if err != nil {
		return err
	}
	chat, err := types.ParseJID(string(chatID))
	if err != nil {
		return core.ErrUnknown.Wrap(err)
	}
	bySender := map[string][]types.MessageID{}
	var order []string
	for _, m := range messages {
		if m.FromMe {
			continue
		}
		sender := ""
		if chat.Server == types.GroupServer {
			sender = m.SenderID
		}
		if _, seen := bySender[sender]; !seen {
			order = append(order, sender)
		}
		bySender[sender] = append(bySender[sender], types.MessageID(m.ID))
	}
	now := time.Now()
	for _, sender := range order {
		var senderJID types.JID
		if sender != "" {
			if senderJID, err = types.ParseJID(sender); err != nil {
				continue
			}
		}
		if err := client.MarkRead(ctx, bySender[sender], now, chat, senderJID); err != nil {
			return core.ErrUnknown.Wrap(err)
		}
	}
	return nil
}

// loggedInClient returns the connected client, or ErrConnectFailed while offline.
func (s *Session) loggedInClient() (*whatsmeow.Client, error) {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil || !client.IsConnected() || !client.IsLoggedIn() {
		return nil, core.ErrConnectFailed
	}
	return client, nil
}

// receipt turns delivery and read receipts into message statuses. A "read-self" receipt
// means the user read the chat on another device (e.g. the phone): the chat is read here too.
func (y *syncer) receipt(ctx context.Context, e *events.Receipt) {
	if !isChat(e.Chat) {
		return
	}
	chat := core.ChatID(y.canonical(ctx, e.Chat).String())
	var status core.MessageStatus
	switch e.Type {
	case types.ReceiptTypeDelivered:
		status = core.StatusDelivered
	case types.ReceiptTypeRead, types.ReceiptTypePlayed:
		status = core.StatusRead
	case types.ReceiptTypeReadSelf, types.ReceiptTypePlayedSelf:
		read := true
		y.push(core.ChatPatchEvent{Patch: core.ChatPatch{ChatID: chat, Read: &read}})
		return
	default:
		return // receipts from the user's own devices, retries and errors
	}
	if e.IsFromMe {
		return // the user's own devices confirming a message, not the recipient
	}
	for _, id := range e.MessageIDs {
		y.push(core.MessageStatusEvent{ChatID: chat, MessageID: core.MessageID(id), Status: status})
	}
}
