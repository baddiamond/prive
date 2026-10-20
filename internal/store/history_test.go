package store

import (
	"context"
	"testing"
	"time"

	"github.com/baddiamond/prive/internal/core"
)

func ptr[T any](v T) *T { return &v }

func TestApplyMessage(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	addAccount(t, s, "a", 0)

	incoming := core.Message{AccountID: "a", ChatID: "g1", ID: "m1", Kind: core.MessageText, Text: "hola", Timestamp: t0}
	chat, err := s.ApplyMessage(ctx, incoming, core.ChatGroup, "Climbing", true)
	if err != nil {
		t.Fatal(err)
	}
	if chat.Name != "Climbing" || chat.Kind != core.ChatGroup || chat.UnreadCount != 1 ||
		chat.LastMessagePreview != "hola" || !chat.LastMessageAt.Equal(t0) {
		t.Fatalf("new chat from message = %+v", chat)
	}

	// The same message again (e.g. redelivered) does not count twice.
	if chat, _ = s.ApplyMessage(ctx, incoming, core.ChatGroup, "", true); chat.UnreadCount != 1 {
		t.Errorf("duplicate counted: unread %d", chat.UnreadCount)
	}
	// My own message moves the preview but not the unread count.
	mine := core.Message{AccountID: "a", ChatID: "g1", ID: "m2", FromMe: true, Kind: core.MessageImage, Timestamp: t0.Add(time.Minute)}
	if chat, _ = s.ApplyMessage(ctx, mine, "", "", true); chat.UnreadCount != 1 || !chat.LastMessageFromMe ||
		chat.LastMessageKind != core.MessageImage || chat.Name != "Climbing" {
		t.Errorf("after my message = %+v", chat)
	}
	// An older message does not replace the preview.
	old := core.Message{AccountID: "a", ChatID: "g1", ID: "m0", Kind: core.MessageText, Text: "old", Timestamp: t0.Add(-time.Hour)}
	if chat, _ = s.ApplyMessage(ctx, old, "", "", true); chat.LastMessageKind != core.MessageImage || chat.UnreadCount != 1 {
		t.Errorf("older message changed the chat: %+v", chat)
	}
}

func TestImportHistory(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	addAccount(t, s, "a", 0)

	chats := []core.Chat{{ID: "c1", Name: "Ana", Kind: core.ChatDirect, UnreadCount: 3, LastMessageAt: t0, Pinned: true}}
	msgs := []core.Message{
		{ChatID: "c1", ID: "m1", Kind: core.MessageText, Text: "first", Timestamp: t0.Add(-2 * time.Minute)},
		{ChatID: "c1", ID: "m2", Kind: core.MessageVoice, FromMe: true, Timestamp: t0.Add(-time.Minute)},
	}
	if err := s.ImportHistory(ctx, "a", chats, msgs, true); err != nil {
		t.Fatal(err)
	}
	c, ok, _ := s.GetChat(ctx, "a", "c1")
	if !ok || c.UnreadCount != 3 || !c.Pinned || c.LastMessageKind != core.MessageVoice || !c.LastMessageFromMe ||
		!c.LastMessageAt.Equal(t0) || c.AccountID != "a" {
		t.Fatalf("imported chat = %+v", c)
	}

	// A live message arrives, then an older on-demand batch: the chat keeps its newer state.
	live := core.Message{AccountID: "a", ChatID: "c1", ID: "m3", Kind: core.MessageText, Text: "new", Timestamp: t0.Add(time.Hour)}
	if _, err := s.ApplyMessage(ctx, live, "", "", true); err != nil {
		t.Fatal(err)
	}
	older := []core.Message{{ChatID: "c1", ID: "m0", Kind: core.MessageText, Text: "ancient", Timestamp: t0.Add(-24 * time.Hour)}}
	stale := []core.Chat{{ID: "c1", Name: "Ana", UnreadCount: 0}}
	if err := s.ImportHistory(ctx, "a", stale, older, false); err != nil {
		t.Fatal(err)
	}
	c, _, _ = s.GetChat(ctx, "a", "c1")
	if c.UnreadCount != 4 || c.LastMessagePreview != "new" {
		t.Errorf("on-demand batch overwrote the chat: %+v", c)
	}
	page, _ := s.ListMessages(ctx, "a", "c1", core.GetMessagesOptions{Limit: 10})
	if len(page) != 4 || page[0].ID != "m0" {
		t.Errorf("history = %+v", page)
	}

	// An authoritative snapshot taken before the live message keeps the newer preview.
	if err := s.ImportHistory(ctx, "a", chats, nil, true); err != nil {
		t.Fatal(err)
	}
	if c, _, _ = s.GetChat(ctx, "a", "c1"); c.LastMessagePreview != "new" || c.UnreadCount != 3 {
		t.Errorf("snapshot after live message = %+v", c)
	}
}

func TestPatchChatAndUnreadSummary(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	addAccount(t, s, "a", 0)
	addAccount(t, s, "b", 1)
	for _, c := range []core.Chat{
		{AccountID: "a", ID: "c1", UnreadCount: 2, LastMessageAt: t0},
		{AccountID: "a", ID: "c2", UnreadCount: 5, LastMessageAt: t0.Add(time.Minute)},
		{AccountID: "b", ID: "c3", UnreadCount: 1, LastMessageAt: t0},
	} {
		if err := s.UpsertChat(ctx, c); err != nil {
			t.Fatal(err)
		}
	}

	if c, ok, err := s.PatchChat(ctx, "a", core.ChatPatch{ChatID: "c2", Muted: ptr(true), Name: ptr("Work")}); err != nil || !ok || !c.Muted || c.Name != "Work" {
		t.Fatalf("patch = %+v %v %v", c, ok, err)
	}
	if _, ok, err := s.PatchChat(ctx, "a", core.ChatPatch{ChatID: "missing", Name: ptr("x")}); err != nil || ok {
		t.Errorf("patching a missing chat must report false: ok=%v err=%v", ok, err)
	}

	sum, err := s.UnreadSummary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Total != 3 || sum.PerAccount["a"] != 2 || sum.PerAccount["b"] != 1 {
		t.Errorf("summary (muted excluded) = %+v", sum)
	}

	c, _, _ := s.PatchChat(ctx, "a", core.ChatPatch{ChatID: "c1", Read: ptr(true)})
	if c.UnreadCount != 0 {
		t.Errorf("read: unread %d", c.UnreadCount)
	}
	c, _, _ = s.PatchChat(ctx, "a", core.ChatPatch{ChatID: "c1", Read: ptr(false)})
	if c.UnreadCount != 1 {
		t.Errorf("mark unread: unread %d", c.UnreadCount)
	}

	// Pinned chats come first.
	if _, _, err := s.PatchChat(ctx, "b", core.ChatPatch{ChatID: "c3", Pinned: ptr(true)}); err != nil {
		t.Fatal(err)
	}
	list, _ := s.ListChats(ctx, core.ChatFilter{})
	if len(list) != 3 || list[0].ID != "c3" || list[1].ID != "c2" {
		t.Errorf("order = %v, %v, %v", list[0].ID, list[1].ID, list[2].ID)
	}
}
