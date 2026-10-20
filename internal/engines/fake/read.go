package fake

import (
	"context"
	"time"

	"github.com/baddiamond/prive/internal/core"
)

// MarkRead implements core.ReadMarker: it records which messages were reported as read.
func (s *Session) MarkRead(_ context.Context, chatID core.ChatID, messages []core.Message) error {
	if err := s.requireConnected(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range messages {
		s.read[chatID] = append(s.read[chatID], m.ID)
	}
	return nil
}

// ReadMessages returns the IDs MarkRead reported for a chat, for tests.
func (s *Session) ReadMessages(chatID core.ChatID) []core.MessageID {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]core.MessageID(nil), s.read[chatID]...)
}

// demoReply makes the demo feel alive: the contact reads a sent message, then answers.
func (s *Session) demoReply(chatID core.ChatID, sent core.Message) {
	time.AfterFunc(1500*time.Millisecond, func() {
		s.events.Push(core.MessageStatusEvent{ChatID: chatID, MessageID: sent.ID, Status: core.StatusRead})
	})
	time.AfterFunc(3*time.Second, func() {
		if s.Closed() {
			return
		}
		s.mu.Lock()
		name := ""
		if c := s.chats[chatID]; c != nil {
			name = c.Name
		}
		s.mu.Unlock()
		s.Receive(chatID, name, fictionalLines[len(sent.Text)%len(fictionalLines)])
	})
}
