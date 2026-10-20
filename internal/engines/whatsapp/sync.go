package whatsapp

import (
	"context"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/proto/waWeb"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"github.com/baddiamond/prive/internal/core"
)

// syncer turns whatsmeow's history, message and app-state events into Privé events for
// one session. It runs on whatsmeow's event goroutine.
type syncer struct {
	client *whatsmeow.Client
	push   func(core.Event)
	log    func(msg string, args ...any)

	// sendPeer sends a message to the user's phone; replaced in tests.
	sendPeer func(context.Context, *waE2E.Message) error

	mu          sync.Mutex
	knownGroups map[types.JID]bool // groups whose name was already looked up this session
	pending     map[core.ChatID]*historyRequest
}

func newSyncer(client *whatsmeow.Client, push func(core.Event), log func(string, ...any)) *syncer {
	y := &syncer{
		client:      client,
		push:        push,
		log:         log,
		knownGroups: map[types.JID]bool{},
		pending:     map[core.ChatID]*historyRequest{},
	}
	y.sendPeer = sendPeerMessage(y)
	return y
}

// handle dispatches one whatsmeow event. It reports whether the event was handled.
func (y *syncer) handle(evt any) bool {
	ctx := context.Background()
	switch e := evt.(type) {
	case *events.HistorySync:
		y.history(ctx, e.Data)
	case *events.Message:
		y.message(ctx, e)
	case *events.Archive:
		y.patch(ctx, e.JID, func(p *core.ChatPatch) { v := e.Action.GetArchived(); p.Archived = &v })
	case *events.Pin:
		y.patch(ctx, e.JID, func(p *core.ChatPatch) { v := e.Action.GetPinned(); p.Pinned = &v })
	case *events.Mute:
		y.patch(ctx, e.JID, func(p *core.ChatPatch) { v := e.Action.GetMuted(); p.Muted = &v })
	case *events.MarkChatAsRead:
		y.patch(ctx, e.JID, func(p *core.ChatPatch) { v := e.Action.GetRead(); p.Read = &v })
	case *events.Contact:
		y.rename(ctx, e.JID)
	case *events.PushName:
		y.rename(ctx, e.JID)
	case *events.GroupInfo:
		if e.Name != nil {
			name := e.Name.Name
			y.patch(ctx, e.JID, func(p *core.ChatPatch) { p.Name = &name })
		}
	default:
		return false
	}
	return true
}

// history converts one history sync blob. The initial sync (bootstrap, recent, full) is
// authoritative for chat state; on-demand batches only add older messages.
func (y *syncer) history(ctx context.Context, data *waHistorySync.HistorySync) {
	if data == nil {
		return
	}
	authoritative := true
	switch data.GetSyncType() {
	case waHistorySync.HistorySync_INITIAL_BOOTSTRAP, waHistorySync.HistorySync_RECENT, waHistorySync.HistorySync_FULL:
	case waHistorySync.HistorySync_ON_DEMAND:
		authoritative = false
	case waHistorySync.HistorySync_PUSH_NAME:
		// whatsmeow stores the names; refresh the chats that use them.
		for _, p := range data.GetPushnames() {
			if jid, err := types.ParseJID(p.GetID()); err == nil {
				y.rename(ctx, jid)
			}
		}
		return
	default:
		return // status updates, settings and other data Privé does not show
	}

	var (
		chats   []core.Chat
		msgs    []core.Message
		results []core.HistoryResultEvent
		names   = map[types.JID]string{}
	)
	for _, conv := range data.GetConversations() {
		chat, ok := y.conversation(ctx, conv)
		if !ok {
			continue
		}
		jid, _ := types.ParseJID(conv.GetID())
		n := 0
		for _, hm := range conv.GetMessages() {
			if m, ok := y.historyMessage(ctx, jid, chat.ID, hm.GetMessage(), names); ok {
				msgs = append(msgs, m)
				n++
			}
		}
		chats = append(chats, chat)
		if !authoritative {
			results = append(results, core.HistoryResultEvent{ChatID: chat.ID, Messages: n, Complete: historyComplete(conv)})
		}
	}
	if len(chats) > 0 || len(msgs) > 0 {
		y.push(core.HistoryBatchEvent{Chats: chats, Messages: msgs, Authoritative: authoritative})
	}
	// After the batch, so the UI reloads a chat once its older messages are stored.
	for _, r := range results {
		y.answered(r.ChatID)
		y.push(r)
	}
	if authoritative && data.GetProgress() > 0 {
		y.push(core.SyncProgressEvent{Done: int(data.GetProgress()), Total: 100})
	}
}

// conversation maps a history conversation to a chat. ok is false for broadcasts, status
// updates and channels, which are not chats in Privé.
func (y *syncer) conversation(ctx context.Context, conv *waHistorySync.Conversation) (core.Chat, bool) {
	jid, err := types.ParseJID(conv.GetID())
	if err != nil || !isChat(jid) {
		return core.Chat{}, false
	}
	if pn, err := types.ParseJID(conv.GetPnJID()); err == nil && pn.Server == types.DefaultUserServer {
		jid = pn // prefer the phone-number form so live messages land in the same chat
	}
	jid = y.canonical(ctx, jid)
	kind := core.ChatDirect
	name := conv.GetName()
	if jid.Server == types.GroupServer {
		kind = core.ChatGroup
	} else {
		name = y.contactName(ctx, jid, name)
	}
	return core.Chat{
		ID:            core.ChatID(jid.String()),
		Name:          name,
		Kind:          kind,
		UnreadCount:   int(conv.GetUnreadCount()),
		LastMessageAt: unixSeconds(conv.GetConversationTimestamp()),
		Archived:      conv.GetArchived(),
		Pinned:        conv.GetPinned() > 0,
		Muted:         mutedUntil(conv.GetMuteEndTime()),
	}, true
}

func (y *syncer) historyMessage(ctx context.Context, chatJID types.JID, chatID core.ChatID,
	web *waWeb.WebMessageInfo, names map[types.JID]string) (core.Message, bool) {
	if web == nil {
		return core.Message{}, false
	}
	evt, err := y.client.ParseWebMessage(chatJID, web)
	if err != nil {
		return core.Message{}, false
	}
	m, ok := toMessage(evt, chatID)
	if !ok {
		return m, false
	}
	if m.FromMe {
		m.Status = webStatus(web.GetStatus())
	} else {
		y.identifySender(ctx, &m, evt, names)
	}
	return m, true
}

// identifySender stores the author of an incoming message by phone number when known, with
// their best name, so group messages show who wrote them. names caches lookups in a batch.
func (y *syncer) identifySender(ctx context.Context, m *core.Message, evt *events.Message, names map[types.JID]string) {
	sender := y.canonical(ctx, evt.Info.Sender)
	if sender.IsEmpty() {
		return
	}
	m.SenderID = sender.String()
	name, ok := names[sender]
	if !ok {
		name = y.contactName(ctx, sender, evt.Info.PushName)
		names[sender] = name
	}
	m.SenderName = name
}

// message handles a live message (incoming, or sent from another of the user's devices).
func (y *syncer) message(ctx context.Context, evt *events.Message) {
	if !isChat(evt.Info.Chat) {
		return
	}
	chat := y.canonical(ctx, evt.Info.Chat)
	m, ok := toMessage(evt, core.ChatID(chat.String()))
	if !ok {
		return
	}
	if !m.FromMe {
		y.identifySender(ctx, &m, evt, map[types.JID]string{})
	}
	kind, name := core.ChatDirect, ""
	if evt.Info.IsGroup {
		kind = core.ChatGroup
		y.lookupGroupName(chat)
	} else {
		fallback := ""
		if !evt.Info.IsFromMe {
			fallback = evt.Info.PushName
		}
		name = y.contactName(ctx, chat, fallback)
	}
	y.push(core.MessageEvent{Message: m, ChatKind: kind, ChatName: name})
}

// lookupGroupName fetches a group's name once per session, so a group first seen through a
// live message gets its name.
func (y *syncer) lookupGroupName(jid types.JID) {
	y.mu.Lock()
	known := y.knownGroups[jid]
	y.knownGroups[jid] = true
	y.mu.Unlock()
	if known {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		info, err := y.client.GetGroupInfo(ctx, jid)
		if err != nil {
			y.log("could not read group info", "err", redact(err.Error()))
			return
		}
		name := info.Name
		y.push(core.ChatPatchEvent{Patch: core.ChatPatch{ChatID: core.ChatID(jid.String()), Name: &name}})
	}()
}

// rename refreshes the name of a direct chat after a contact or push name change.
func (y *syncer) rename(ctx context.Context, jid types.JID) {
	jid = y.canonical(ctx, jid)
	if jid.Server == types.GroupServer {
		return
	}
	if name := y.contactName(ctx, jid, ""); name != "" {
		y.patch(ctx, jid, func(p *core.ChatPatch) { p.Name = &name })
	}
}

func (y *syncer) patch(ctx context.Context, jid types.JID, set func(*core.ChatPatch)) {
	if !isChat(jid) {
		return
	}
	p := core.ChatPatch{ChatID: core.ChatID(y.canonical(ctx, jid).String())}
	set(&p)
	y.push(core.ChatPatchEvent{Patch: p})
}

// canonical returns the form of a chat JID Privé uses as chat id: no device part, and the
// phone-number form of a hidden (LID) user when whatsmeow knows it.
func (y *syncer) canonical(ctx context.Context, jid types.JID) types.JID {
	jid = jid.ToNonAD()
	if jid.Server == types.HiddenUserServer && y.client.Store.LIDs != nil {
		if pn, err := y.client.Store.LIDs.GetPNForLID(ctx, jid); err == nil && !pn.IsEmpty() {
			return pn.ToNonAD()
		}
	}
	return jid
}

// contactName picks the best name for a person: the address book name, then the business
// or WhatsApp profile name, then fallback, then the phone number.
func (y *syncer) contactName(ctx context.Context, jid types.JID, fallback string) string {
	// The stores exist once the device is linked.
	if y.client.Store.Contacts != nil {
		if c, err := y.client.Store.Contacts.GetContact(ctx, jid); err == nil && c.Found {
			for _, n := range []string{c.FullName, c.FirstName, c.BusinessName, c.PushName} {
				if n = strings.TrimSpace(n); n != "" {
					return n
				}
			}
		}
	}
	if fallback = strings.TrimSpace(fallback); fallback != "" {
		return fallback
	}
	if jid.Server == types.DefaultUserServer {
		return "+" + jid.User
	}
	return ""
}

// toMessage converts a parsed whatsmeow message; ok is false when it has nothing to show.
func toMessage(evt *events.Message, chatID core.ChatID) (core.Message, bool) {
	kind, text, ok := content(evt.Message)
	if !ok {
		return core.Message{}, false
	}
	m := core.Message{
		ChatID:     chatID,
		ID:         core.MessageID(evt.Info.ID),
		SenderID:   evt.Info.Sender.ToNonAD().String(),
		SenderName: evt.Info.PushName,
		FromMe:     evt.Info.IsFromMe,
		Timestamp:  evt.Info.Timestamp,
		Kind:       kind,
		Text:       text,
	}
	if m.FromMe {
		m.Status = core.StatusSent
	}
	return m, true
}

// isChat reports whether a JID is a conversation Privé shows: people and groups, not
// status updates, broadcast lists or channels.
func isChat(jid types.JID) bool {
	switch jid.Server {
	case types.DefaultUserServer, types.HiddenUserServer, types.GroupServer:
		return true
	}
	return false
}

func webStatus(s waWeb.WebMessageInfo_Status) core.MessageStatus {
	switch s {
	case waWeb.WebMessageInfo_PENDING:
		return core.StatusPending
	case waWeb.WebMessageInfo_SERVER_ACK:
		return core.StatusSent
	case waWeb.WebMessageInfo_DELIVERY_ACK:
		return core.StatusDelivered
	case waWeb.WebMessageInfo_READ, waWeb.WebMessageInfo_PLAYED:
		return core.StatusRead
	case waWeb.WebMessageInfo_ERROR:
		return core.StatusFailed
	}
	return core.StatusSent
}

func unixSeconds(s uint64) time.Time {
	if s == 0 {
		return time.Time{}
	}
	return time.Unix(int64(s), 0).UTC()
}

// mutedUntil interprets a mute end time, which WhatsApp sends in seconds or milliseconds;
// the maximum value means "always".
func mutedUntil(end uint64) bool {
	switch {
	case end == 0:
		return false
	case end == ^uint64(0): // -1: muted forever
		return true
	case end > 1e12:
		end /= 1000
	}
	return end > uint64(time.Now().Unix())
}
