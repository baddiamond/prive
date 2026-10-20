package whatsapp

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/proto/waWeb"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"github.com/baddiamond/prive/internal/core"
	"github.com/baddiamond/prive/internal/store"
)

// Test identities are fictional (CLAUDE.md: no real phone numbers in tests).
const (
	alice = "100000000001@s.whatsapp.net"
	team  = "120000000000000001@g.us"
)

func TestContent(t *testing.T) {
	cases := []struct {
		msg  *waE2E.Message
		kind core.MessageKind
		text string
		ok   bool
	}{
		{&waE2E.Message{Conversation: proto.String("hi")}, core.MessageText, "hi", true},
		{&waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{Text: proto.String("link")}}, core.MessageText, "link", true},
		{&waE2E.Message{ImageMessage: &waE2E.ImageMessage{Caption: proto.String("beach")}}, core.MessageImage, "beach", true},
		{&waE2E.Message{AudioMessage: &waE2E.AudioMessage{PTT: proto.Bool(true)}}, core.MessageVoice, "", true},
		{&waE2E.Message{AudioMessage: &waE2E.AudioMessage{}}, core.MessageAudio, "", true},
		{&waE2E.Message{DocumentMessage: &waE2E.DocumentMessage{FileName: proto.String("plan.pdf")}}, core.MessageDocument, "plan.pdf", true},
		{&waE2E.Message{StickerMessage: &waE2E.StickerMessage{}}, core.MessageSticker, "", true},
		{&waE2E.Message{ReactionMessage: &waE2E.ReactionMessage{}}, "", "", false},
		{&waE2E.Message{ProtocolMessage: &waE2E.ProtocolMessage{}}, "", "", false},
		{&waE2E.Message{SenderKeyDistributionMessage: &waE2E.SenderKeyDistributionMessage{}}, "", "", false},
		{&waE2E.Message{}, core.MessageUnsupported, "", true},
		{nil, "", "", false},
	}
	for i, c := range cases {
		kind, text, ok := content(c.msg)
		if kind != c.kind || text != c.text || ok != c.ok {
			t.Errorf("case %d: got (%q, %q, %v), want (%q, %q, %v)", i, kind, text, ok, c.kind, c.text, c.ok)
		}
	}
}

func TestIsChatAndMute(t *testing.T) {
	for jid, want := range map[string]bool{
		alice: true, team: true, "status@broadcast": false, "123@newsletter": false, "1@lid": true,
	} {
		j, _ := types.ParseJID(jid)
		if isChat(j) != want {
			t.Errorf("isChat(%s) = %v", jid, !want)
		}
	}
	future := uint64(time.Now().Add(time.Hour).Unix())
	if !mutedUntil(future) || !mutedUntil(future*1000) || !mutedUntil(^uint64(0)) || mutedUntil(0) || mutedUntil(1000) {
		t.Error("mutedUntil")
	}
}

// recorder collects what the syncer pushes.
type recorder struct {
	mu     sync.Mutex
	events []core.Event
}

func (r *recorder) push(e core.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
}

func newTestSyncer(t *testing.T) (*syncer, *recorder) {
	t.Helper()
	ctx := context.Background()
	db, err := store.OpenDB(ctx, filepath.Join(t.TempDir(), "store.db"), store.Options{Key: bytes.Repeat([]byte{3}, store.KeySize)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	c := sqlstore.NewWithDB(db, "sqlite3", newLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))
	if err := c.Upgrade(ctx); err != nil {
		t.Fatal(err)
	}
	client := whatsmeow.NewClient(c.NewDevice(), nil)
	r := &recorder{}
	return newSyncer(client, r.push, func(string, ...any) {}), r
}

func webMsg(chat, id string, ts uint64, fromMe bool, m *waE2E.Message) *waHistorySync.HistorySyncMsg {
	return &waHistorySync.HistorySyncMsg{Message: &waWeb.WebMessageInfo{
		Key:              &waCommon.MessageKey{RemoteJID: proto.String(chat), ID: proto.String(id), FromMe: proto.Bool(fromMe)},
		Message:          m,
		MessageTimestamp: proto.Uint64(ts),
		PushName:         proto.String("Alice"),
		Participant:      proto.String(alice),
	}}
}

func TestHistoryConversion(t *testing.T) {
	y, r := newTestSyncer(t)
	y.history(context.Background(), &waHistorySync.HistorySync{
		SyncType: waHistorySync.HistorySync_INITIAL_BOOTSTRAP.Enum(),
		Progress: proto.Uint32(40),
		Conversations: []*waHistorySync.Conversation{
			{
				ID: proto.String(alice), UnreadCount: proto.Uint32(2), ConversationTimestamp: proto.Uint64(1000), Pinned: proto.Uint32(1),
				Messages: []*waHistorySync.HistorySyncMsg{
					webMsg(alice, "m1", 990, false, &waE2E.Message{Conversation: proto.String("hello")}),
					webMsg(alice, "r1", 995, false, &waE2E.Message{ReactionMessage: &waE2E.ReactionMessage{}}),
				},
			},
			{ID: proto.String(team), Name: proto.String("Team"), Archived: proto.Bool(true)},
			{ID: proto.String("status@broadcast")},
		},
	})

	if len(r.events) != 2 {
		t.Fatalf("pushed %d events, want batch + progress", len(r.events))
	}
	batch := r.events[0].(core.HistoryBatchEvent)
	if !batch.Authoritative || len(batch.Chats) != 2 || len(batch.Messages) != 1 {
		t.Fatalf("batch = %+v", batch)
	}
	direct, group := batch.Chats[0], batch.Chats[1]
	if direct.Kind != core.ChatDirect || direct.UnreadCount != 2 || !direct.Pinned || direct.Name != "+100000000001" ||
		!direct.LastMessageAt.Equal(time.Unix(1000, 0)) {
		t.Errorf("direct chat = %+v", direct)
	}
	if group.Kind != core.ChatGroup || group.Name != "Team" || !group.Archived {
		t.Errorf("group chat = %+v", group)
	}
	m := batch.Messages[0]
	if m.ChatID != core.ChatID(alice) || m.Text != "hello" || m.FromMe || m.SenderName != "Alice" {
		t.Errorf("message = %+v", m)
	}
	if p := r.events[1].(core.SyncProgressEvent); p.Done != 40 || p.Total != 100 {
		t.Errorf("progress = %+v", p)
	}
}

func TestOnDemandIsNotAuthoritative(t *testing.T) {
	y, r := newTestSyncer(t)
	y.history(context.Background(), &waHistorySync.HistorySync{
		SyncType: waHistorySync.HistorySync_ON_DEMAND.Enum(),
		Conversations: []*waHistorySync.Conversation{{
			ID:       proto.String(alice),
			Messages: []*waHistorySync.HistorySyncMsg{webMsg(alice, "old", 10, false, &waE2E.Message{Conversation: proto.String("old")})},
		}},
	})
	if batch := r.events[0].(core.HistoryBatchEvent); batch.Authoritative {
		t.Error("on-demand history must not overwrite chat state")
	}
}

func TestLiveMessageAndAppState(t *testing.T) {
	y, r := newTestSyncer(t)
	chat, _ := types.ParseJID(alice)
	y.handle(&events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{Chat: chat, Sender: chat},
			ID:            "live1", PushName: "Alice", Timestamp: time.Unix(2000, 0),
		},
		Message: &waE2E.Message{Conversation: proto.String("are you there?")},
	})
	ev := r.events[0].(core.MessageEvent)
	if ev.ChatKind != core.ChatDirect || ev.ChatName != "Alice" || ev.Message.Text != "are you there?" {
		t.Errorf("live message = %+v", ev)
	}

	y.handle(&events.MarkChatAsRead{JID: chat, Action: markRead()})
	p := r.events[1].(core.ChatPatchEvent).Patch
	if p.ChatID != core.ChatID(alice) || p.Read == nil || !*p.Read {
		t.Errorf("read patch = %+v", p)
	}
}
