package store

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/baddiamond/prive/internal/core"
)

var t0 = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

func newStore(t *testing.T) *Store {
	t.Helper()
	return openTestStore(t, filepath.Join(t.TempDir(), FileName))
}

func addAccount(t *testing.T, s *Store, id core.AccountID, position int) core.Account {
	t.Helper()
	a := core.Account{ID: id, EngineID: "fake", DisplayName: "Account " + string(id), Position: position, CreatedAt: t0}
	if err := s.CreateAccount(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	return a
}

func TestAccounts(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	addAccount(t, s, "b", 1)
	a := addAccount(t, s, "a", 0)

	got, err := s.ListAccounts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "a" || got[1].ID != "b" {
		t.Fatalf("accounts not ordered by position: %+v", got)
	}
	if !got[0].CreatedAt.Equal(t0) {
		t.Errorf("CreatedAt = %v, want %v", got[0].CreatedAt, t0)
	}

	a.RemoteID, a.DisplayName = "remote-a", "Work"
	if err := s.UpdateAccount(ctx, a); err != nil {
		t.Fatal(err)
	}
	got, _ = s.ListAccounts(ctx)
	if got[0].RemoteID != "remote-a" || got[0].DisplayName != "Work" {
		t.Errorf("update not saved: %+v", got[0])
	}
	if err := s.UpdateAccount(ctx, core.Account{ID: "missing"}); !errors.Is(err, core.ErrAccountNotFound) {
		t.Errorf("updating a missing account = %v, want ErrAccountNotFound", err)
	}
}

func TestChatsInbox(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	addAccount(t, s, "a", 0)
	addAccount(t, s, "b", 1)

	chats := []core.Chat{
		{AccountID: "a", ID: "a1", Name: "A1", Kind: core.ChatDirect, LastMessageAt: t0.Add(1 * time.Minute)},
		{AccountID: "a", ID: "a2", Name: "A2", Kind: core.ChatGroup, LastMessageAt: t0.Add(3 * time.Minute), UnreadCount: 2},
		{AccountID: "b", ID: "b1", Name: "B1", Kind: core.ChatDirect, LastMessageAt: t0.Add(2 * time.Minute)},
		{AccountID: "b", ID: "b2", Name: "B2", Archived: true, LastMessageAt: t0.Add(4 * time.Minute)},
		{AccountID: "b", ID: "b3", Name: "Empty"},
	}
	for _, c := range chats {
		if err := s.UpsertChat(ctx, c); err != nil {
			t.Fatal(err)
		}
	}

	ids := func(cs []core.Chat) string {
		out := ""
		for _, c := range cs {
			out += string(c.ID) + " "
		}
		return out
	}

	unified, err := s.ListChats(ctx, core.ChatFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(unified); got != "a2 b1 a1 b3 " {
		t.Errorf("unified inbox = %q", got)
	}
	if unified[0].UnreadCount != 2 || unified[0].Kind != core.ChatGroup || !unified[0].LastMessageAt.Equal(t0.Add(3*time.Minute)) {
		t.Errorf("fields not round-tripped: %+v", unified[0])
	}

	onlyB, _ := s.ListChats(ctx, core.ChatFilter{AccountID: "b", IncludeArchived: true})
	if got := ids(onlyB); got != "b2 b1 b3 " {
		t.Errorf("account b inbox = %q", got)
	}

	page, _ := s.ListChats(ctx, core.ChatFilter{Limit: 1, Before: t0.Add(3 * time.Minute)})
	if got := ids(page); got != "b1 " {
		t.Errorf("paged inbox = %q", got)
	}

	// Upsert replaces the chat state.
	chats[0].Name, chats[0].UnreadCount = "Renamed", 5
	if err := s.UpsertChat(ctx, chats[0]); err != nil {
		t.Fatal(err)
	}
	onlyA, _ := s.ListChats(ctx, core.ChatFilter{AccountID: "a"})
	if onlyA[1].Name != "Renamed" || onlyA[1].UnreadCount != 5 {
		t.Errorf("chat not updated: %+v", onlyA[1])
	}
}

func TestMessagesHistoryAndStatus(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	addAccount(t, s, "a", 0)

	for i := range 5 {
		m := core.Message{
			AccountID: "a", ChatID: "c", ID: core.MessageID(fmt.Sprintf("m%d", i)),
			Timestamp: t0.Add(time.Duration(i) * time.Second), Kind: core.MessageText,
			Text: fmt.Sprintf("text %d", i), Status: core.StatusSent, FromMe: i%2 == 0,
		}
		if err := s.UpsertMessage(ctx, m); err != nil {
			t.Fatal(err)
		}
	}

	latest, err := s.ListMessages(ctx, "a", "c", core.GetMessagesOptions{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(latest) != 2 || latest[0].ID != "m3" || latest[1].ID != "m4" {
		t.Fatalf("latest page = %+v, want m3, m4 in order", latest)
	}
	older, _ := s.ListMessages(ctx, "a", "c", core.GetMessagesOptions{Limit: 10, Before: latest[0].Timestamp})
	if len(older) != 3 || older[0].ID != "m0" || older[2].ID != "m2" {
		t.Fatalf("older page = %+v, want m0..m2", older)
	}
	if !older[0].FromMe || older[1].FromMe || older[0].Text != "text 0" {
		t.Errorf("fields not round-tripped: %+v", older[0])
	}

	steps := []struct {
		status core.MessageStatus
		want   core.MessageStatus
	}{
		{core.StatusRead, core.StatusRead},
		{core.StatusDelivered, core.StatusRead}, // late event never downgrades
		{core.StatusFailed, core.StatusRead},
	}
	for _, step := range steps {
		if err := s.UpdateMessageStatus(ctx, "a", "c", "m4", step.status); err != nil {
			t.Fatal(err)
		}
		got, _ := s.ListMessages(ctx, "a", "c", core.GetMessagesOptions{Limit: 1})
		if got[0].Status != step.want {
			t.Errorf("after %s: status = %s, want %s", step.status, got[0].Status, step.want)
		}
	}

	// Re-delivering a message with an older status keeps the newer one.
	if err := s.UpsertMessage(ctx, core.Message{AccountID: "a", ChatID: "c", ID: "m4",
		Timestamp: t0.Add(4 * time.Second), Kind: core.MessageText, Status: core.StatusSent}); err != nil {
		t.Fatal(err)
	}
	got, _ := s.ListMessages(ctx, "a", "c", core.GetMessagesOptions{Limit: 1})
	if got[0].Status != core.StatusRead {
		t.Errorf("upsert downgraded status to %s", got[0].Status)
	}

	if err := s.UpdateMessageStatus(ctx, "a", "c", "missing", core.StatusRead); err != nil {
		t.Errorf("status for an unknown message: %v", err)
	}
}

func TestDeleteAccountCascades(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	addAccount(t, s, "a", 0)
	addAccount(t, s, "b", 1)
	for _, id := range []core.AccountID{"a", "b"} {
		if err := s.UpsertChat(ctx, core.Chat{AccountID: id, ID: "c", LastMessageAt: t0}); err != nil {
			t.Fatal(err)
		}
		if err := s.UpsertMessage(ctx, core.Message{AccountID: id, ChatID: "c", ID: "m", Timestamp: t0}); err != nil {
			t.Fatal(err)
		}
	}

	if err := s.DeleteAccount(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if chats, _ := s.ListChats(ctx, core.ChatFilter{}); len(chats) != 1 || chats[0].AccountID != "b" {
		t.Errorf("chats after delete = %+v", chats)
	}
	if msgs, _ := s.ListMessages(ctx, "a", "c", core.GetMessagesOptions{}); len(msgs) != 0 {
		t.Errorf("messages of deleted account survived: %+v", msgs)
	}
	if msgs, _ := s.ListMessages(ctx, "b", "c", core.GetMessagesOptions{}); len(msgs) != 1 {
		t.Errorf("other account's messages lost: %+v", msgs)
	}
}

func TestConcurrentWritesFromManyAccounts(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	const accounts, perAccount = 6, 40
	for i := range accounts {
		addAccount(t, s, core.AccountID(fmt.Sprintf("acc%d", i)), i)
	}

	var wg sync.WaitGroup
	errs := make(chan error, accounts*perAccount*2)
	for i := range accounts {
		id := core.AccountID(fmt.Sprintf("acc%d", i))
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range perAccount {
				mid := core.MessageID(fmt.Sprintf("m%d", j))
				errs <- s.UpsertMessage(ctx, core.Message{AccountID: id, ChatID: "c", ID: mid,
					Timestamp: t0.Add(time.Duration(j) * time.Second), Status: core.StatusSent})
				errs <- s.UpdateMessageStatus(ctx, id, "c", mid, core.StatusDelivered)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	for i := range accounts {
		msgs, _ := s.ListMessages(ctx, core.AccountID(fmt.Sprintf("acc%d", i)), "c", core.GetMessagesOptions{Limit: 100})
		if len(msgs) != perAccount {
			t.Errorf("account %d has %d messages, want %d", i, len(msgs), perAccount)
		}
	}
}
