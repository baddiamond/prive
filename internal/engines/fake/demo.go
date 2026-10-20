package fake

import (
	"fmt"
	"time"

	"github.com/baddiamond/prive/internal/core"
)

// NewDemo returns a fake engine whose accounts, once linked, import a fictional history
// (the sample names of the design prototypes) and receive a live message a few seconds
// later. It lets the chat list be developed without a phone (devfake build tag).
func NewDemo() *Engine {
	e := New()
	e.demo = true
	return e
}

// demoChats are fictional, like every name in design/prototypes.
var demoChats = []struct {
	id, name string
	group    bool
	kind     core.MessageKind
	text     string
	ago      time.Duration
	unread   int
	pinned   bool
	muted    bool
}{
	{"c1", "Lucía Romero", false, core.MessageText, "See you at 8 then", 3 * time.Minute, 2, true, false},
	{"c2", "Product team", true, core.MessageText, "The build is green again", 15 * time.Minute, 5, false, false},
	{"c3", "Mom", false, core.MessageVoice, "", 2 * time.Hour, 1, false, false},
	{"c4", "Diego Herrera", false, core.MessageDocument, "slides.pdf", 26 * time.Hour, 0, false, false},
	{"c5", "Ana Torres", false, core.MessageText, "Thanks! Talk on Monday.", 30 * time.Hour, 0, false, false},
	{"c6", "Climbing crew", true, core.MessageImage, "Saturday 10am at the usual wall?", 3 * 24 * time.Hour, 3, false, true},
	{"c7", "Sofía Méndez", false, core.MessageText, "Perfect, thank you", 4 * 24 * time.Hour, 0, false, false},
	{"c8", "Billing", false, core.MessageText, "Payment received", 12 * 24 * time.Hour, 0, false, false},
}

// demoLength is how many messages lead to chat i's last one: long for the first chats.
func demoLength(i int) int {
	if i < 2 {
		return 60
	}
	return 8
}

// importDemo pushes a fictional history in two batches with progress, then a live message.
func (s *Session) importDemo() {
	now := time.Now().UTC()
	var chats []core.Chat
	var msgs []core.Message
	for i, d := range demoChats {
		kind := core.ChatDirect
		if d.group {
			kind = core.ChatGroup
		}
		at := now.Add(-d.ago).Truncate(time.Second)
		chats = append(chats, core.Chat{
			ID: core.ChatID(d.id), Name: d.name, Kind: kind, UnreadCount: d.unread,
			LastMessageAt: at, Pinned: d.pinned, Muted: d.muted,
		})
		// A conversation leading to the last message, long enough to scroll.
		msgs = append(msgs, fictionalMessages(core.ChatID(d.id), d.group, d.name, at, demoLength(i), "demo")...)
		last := core.Message{
			ChatID: core.ChatID(d.id), ID: core.MessageID(fmt.Sprintf("demo-%d", i)), Kind: d.kind, Text: d.text,
			Timestamp: at, FromMe: i == 3, SenderName: d.name, SenderID: "contact:" + d.id,
		}
		if last.FromMe {
			last.Status = core.StatusRead
		} else if d.group {
			last.SenderName = fictionalSenders[0]
		}
		msgs = append(msgs, last)
		if d.group {
			s.mu.Lock()
			s.groups[core.ChatID(d.id)] = true
			s.mu.Unlock()
		}
	}
	half := len(chats) / 2
	s.events.Push(core.SyncProgressEvent{Done: 10, Total: 100})
	time.Sleep(700 * time.Millisecond)
	s.events.Push(core.HistoryBatchEvent{Chats: chats[:half], Messages: msgs[:half], Authoritative: true})
	s.events.Push(core.SyncProgressEvent{Done: 60, Total: 100})
	time.Sleep(700 * time.Millisecond)
	s.events.Push(core.HistoryBatchEvent{Chats: chats[half:], Messages: msgs[half:], Authoritative: true})
	s.events.Push(core.SyncProgressEvent{Done: 100, Total: 100})

	time.Sleep(4 * time.Second)
	if !s.Closed() {
		s.Receive("c5", "Ana Torres", "One more thing: can you send me the notes?")
	}
}
