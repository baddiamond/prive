package whatsapp

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/types"

	"github.com/baddiamond/prive/internal/core"
)

// historyAnswerWait is how long the phone has to answer one history request before the
// next chat address is tried, or the request is reported as unanswered.
var historyAnswerWait = 20 * time.Second

// historyRequest is an on-demand history request waiting for the phone's answer.
type historyRequest struct {
	// targets are the chat addresses still to try. The phone may know a person's chat by
	// phone number or by LID; Privé stores the phone number (see syncer.canonical).
	targets []types.JID
	oldest  core.Message
	count   int
	timer   *time.Timer
}

// RequestOlderHistory implements core.HistoryRequester: it asks the phone for messages older
// than oldest. The phone answers with an ON_DEMAND history sync (see syncer.history).
func (s *Session) RequestOlderHistory(ctx context.Context, chatID core.ChatID, oldest core.Message, count int) error {
	s.mu.Lock()
	client, y := s.client, s.syncer
	s.mu.Unlock()
	if client == nil || y == nil || !client.IsLoggedIn() {
		return core.ErrConnectFailed
	}
	jid, err := types.ParseJID(string(chatID))
	if err != nil || !isChat(jid) {
		return core.ErrHistoryUnavailable
	}
	return y.requestHistory(ctx, chatID, y.chatAddresses(ctx, jid), oldest, count)
}

// chatAddresses returns the ways the phone may know a chat: as stored, then the LID of a
// phone-number chat.
func (y *syncer) chatAddresses(ctx context.Context, jid types.JID) []types.JID {
	out := []types.JID{jid}
	if jid.Server == types.DefaultUserServer && y.client.Store.LIDs != nil {
		if lid, err := y.client.Store.LIDs.GetLIDForPN(ctx, jid); err == nil && !lid.IsEmpty() {
			out = append(out, lid.ToNonAD())
		}
	}
	return out
}

// requestHistory sends the request for the first address and arms the answer timer. A new
// request for the same chat replaces the previous one.
func (y *syncer) requestHistory(ctx context.Context, chatID core.ChatID, targets []types.JID,
	oldest core.Message, count int) error {
	req := &historyRequest{targets: targets, oldest: oldest, count: count}
	y.mu.Lock()
	if old := y.pending[chatID]; old != nil && old.timer != nil {
		old.timer.Stop()
	}
	y.pending[chatID] = req
	y.mu.Unlock()
	return y.sendNext(ctx, chatID, req)
}

// sendNext sends the request to the next address. When none is left it reports that the
// phone did not answer.
func (y *syncer) sendNext(ctx context.Context, chatID core.ChatID, req *historyRequest) error {
	y.mu.Lock()
	if y.pending[chatID] != req {
		y.mu.Unlock()
		return nil // answered or replaced meanwhile
	}
	if len(req.targets) == 0 {
		delete(y.pending, chatID)
		y.mu.Unlock()
		y.push(core.HistoryResultEvent{ChatID: chatID, Code: core.CodeHistoryNoAnswer})
		return nil
	}
	target := req.targets[0]
	req.targets = req.targets[1:]
	req.timer = time.AfterFunc(historyAnswerWait, func() {
		_ = y.sendNext(context.Background(), chatID, req)
	})
	y.mu.Unlock()

	info := &types.MessageInfo{
		MessageSource: types.MessageSource{Chat: target, IsFromMe: req.oldest.FromMe},
		ID:            string(req.oldest.ID),
		Timestamp:     req.oldest.Timestamp,
	}
	if err := y.sendPeer(ctx, y.client.BuildHistorySyncRequest(info, req.count)); err != nil {
		y.mu.Lock()
		if y.pending[chatID] == req {
			req.timer.Stop()
			delete(y.pending, chatID)
		}
		y.mu.Unlock()
		y.log("request older history", "err", err)
		return core.ErrConnectFailed.Wrap(err)
	}
	return nil
}

// answered ends the pending request for a chat, if any.
func (y *syncer) answered(chatID core.ChatID) {
	y.mu.Lock()
	defer y.mu.Unlock()
	if req := y.pending[chatID]; req != nil {
		if req.timer != nil {
			req.timer.Stop()
		}
		delete(y.pending, chatID)
	}
}

// historyComplete reports whether an on-demand answer says the phone has nothing older.
func historyComplete(conv *waHistorySync.Conversation) bool {
	switch conv.GetEndOfHistoryTransferType() {
	case waHistorySync.Conversation_COMPLETE_AND_NO_MORE_MESSAGE_REMAIN_ON_PRIMARY,
		waHistorySync.Conversation_COMPLETE_ON_DEMAND_SYNC_WITH_MORE_MSG_ON_PRIMARY_BUT_NO_ACCESS:
		return true
	}
	return false
}

// sendPeerMessage sends a message to the user's own phone.
func sendPeerMessage(y *syncer) func(context.Context, *waE2E.Message) error {
	return func(ctx context.Context, msg *waE2E.Message) error {
		_, err := y.client.SendPeerMessage(ctx, msg)
		return err
	}
}
