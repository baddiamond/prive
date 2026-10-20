package whatsapp

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"

	"github.com/baddiamond/prive/internal/core"
)

// sentRequests records the history requests a syncer sends to the phone.
type sentRequests struct {
	mu    sync.Mutex
	chats []string
}

func (s *sentRequests) send(_ context.Context, m *waE2E.Message) error {
	req := m.GetProtocolMessage().GetPeerDataOperationRequestMessage().GetHistorySyncOnDemandRequest()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chats = append(s.chats, req.GetChatJID())
	return nil
}

func (s *sentRequests) list() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.chats...)
}

func (r *recorder) results() []core.HistoryResultEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []core.HistoryResultEvent
	for _, e := range r.events {
		if res, ok := e.(core.HistoryResultEvent); ok {
			out = append(out, res)
		}
	}
	return out
}

var oldestAlice = core.Message{ChatID: alice, ID: "m1", Timestamp: time.Unix(990, 0)}

func TestOnDemandRequestIsAnswered(t *testing.T) {
	y, r := newTestSyncer(t)
	sent := &sentRequests{}
	y.sendPeer = sent.send
	jid, _ := types.ParseJID(alice)

	if err := y.requestHistory(context.Background(), alice, []types.JID{jid}, oldestAlice, 50); err != nil {
		t.Fatal(err)
	}
	if got := sent.list(); len(got) != 1 || got[0] != alice {
		t.Fatalf("sent = %v", got)
	}

	y.history(context.Background(), &waHistorySync.HistorySync{
		SyncType: waHistorySync.HistorySync_ON_DEMAND.Enum(),
		Conversations: []*waHistorySync.Conversation{{
			ID: proto.String(alice),
			Messages: []*waHistorySync.HistorySyncMsg{
				webMsg(alice, "m0", 980, false, &waE2E.Message{Conversation: proto.String("older")}),
			},
			EndOfHistoryTransferType: waHistorySync.Conversation_COMPLETE_AND_NO_MORE_MESSAGE_REMAIN_ON_PRIMARY.Enum(),
		}},
	})

	r.mu.Lock()
	_, batchFirst := r.events[0].(core.HistoryBatchEvent)
	r.mu.Unlock()
	res := r.results()
	if !batchFirst || len(res) != 1 || res[0].ChatID != alice || res[0].Messages != 1 || !res[0].Complete || res[0].Code != "" {
		t.Fatalf("events = %+v", r.events)
	}
	y.mu.Lock()
	pending := len(y.pending)
	y.mu.Unlock()
	if pending != 0 {
		t.Error("the answered request must not stay pending")
	}
}

func TestOnDemandRequestTriesTheOtherAddressThenGivesUp(t *testing.T) {
	old := historyAnswerWait
	historyAnswerWait = 30 * time.Millisecond
	t.Cleanup(func() { historyAnswerWait = old })

	y, r := newTestSyncer(t)
	sent := &sentRequests{}
	y.sendPeer = sent.send
	pn, _ := types.ParseJID(alice)
	lid := types.NewJID("200000000000001", types.HiddenUserServer)

	if err := y.requestHistory(context.Background(), alice, []types.JID{pn, lid}, oldestAlice, 50); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for len(r.results()) == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if got := sent.list(); len(got) != 2 || got[0] != alice || got[1] != lid.String() {
		t.Errorf("sent = %v, want the phone number then the LID", got)
	}
	res := r.results()
	if len(res) != 1 || res[0].Code != core.CodeHistoryNoAnswer || res[0].Complete {
		t.Fatalf("results = %+v", res)
	}
}
