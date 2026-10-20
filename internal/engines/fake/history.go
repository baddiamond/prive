package fake

import (
	"context"
	"fmt"
	"time"

	"github.com/baddiamond/prive/internal/core"
)

// olderPages is how many pages of older history each fake chat has "on the phone".
const olderPages = 3

// historyDelay is how long the fake phone takes to answer a history request.
var historyDelay = 800 * time.Millisecond

// RequestOlderHistory implements core.HistoryRequester. Shortly after, it pushes up to count
// fictional messages older than oldest; the olderPages-th answer for a chat is Complete.
func (s *Session) RequestOlderHistory(_ context.Context, chatID core.ChatID, oldest core.Message, count int) error {
	if err := s.requireConnected(); err != nil {
		return err
	}
	s.mu.Lock()
	s.historyPages[chatID]++
	page := s.historyPages[chatID]
	group := s.groups[chatID]
	s.mu.Unlock()

	time.AfterFunc(historyDelay, func() {
		msgs := fictionalMessages(chatID, group, oldest.SenderName, oldest.Timestamp, count, fmt.Sprintf("p%d-%d", page, oldest.Timestamp.Unix()))
		s.events.Push(core.HistoryBatchEvent{Messages: msgs})
		s.events.Push(core.HistoryResultEvent{ChatID: chatID, Messages: len(msgs), Complete: page >= olderPages})
	})
	return nil
}

// fictionalLines are sample messages, as fictional as the names in design/prototypes.
var fictionalLines = []string{
	"Good morning!", "Are we still on for later?", "Sure, sounds good", "I'll be there in 10 minutes",
	"Did you see the photos?", "Haha, yes", "Can you send me the address?", "Thanks a lot",
	"Let me check and get back to you", "That works for me", "On my way", "Running a bit late, sorry",
	"Perfect", "What time works for you?", "How about Thursday?", "See you soon",
	"I found the notes from last week", "Nice!", "Call me when you can", "All done here",
}

// fictionalSenders write in fake group chats.
var fictionalSenders = []string{"Marta Gil", "Leo Ruiz", "Inés Vidal"}

// fictionalMessages builds n messages ending a few minutes before end, oldest first. In
// groups they come from fictionalSenders; in direct chats from name.
func fictionalMessages(chatID core.ChatID, group bool, name string, end time.Time, n int, tag string) []core.Message {
	out := make([]core.Message, n)
	for i := range n {
		age := n - i // the first message is the oldest
		m := core.Message{
			ChatID:    chatID,
			ID:        core.MessageID(fmt.Sprintf("%s-%s-%d", chatID, tag, i)),
			Timestamp: end.Add(-time.Duration(age) * 47 * time.Minute).Truncate(time.Second),
			Kind:      core.MessageText,
			Text:      fictionalLines[(age*7+len(tag))%len(fictionalLines)],
			FromMe:    age%3 == 0,
		}
		if m.FromMe {
			m.Status = core.StatusRead
		} else if group {
			m.SenderName = fictionalSenders[age%len(fictionalSenders)]
			m.SenderID = "contact:" + m.SenderName
		} else {
			m.SenderName = name
			m.SenderID = "contact:" + string(chatID)
		}
		out[i] = m
	}
	return out
}
