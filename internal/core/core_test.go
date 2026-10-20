package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestMergeStatus(t *testing.T) {
	cases := []struct{ current, next, want MessageStatus }{
		{"", StatusSent, StatusSent},
		{StatusPending, StatusSent, StatusSent},
		{StatusSent, StatusRead, StatusRead},
		{StatusRead, StatusDelivered, StatusRead},
		{StatusDelivered, StatusDelivered, StatusDelivered},
		{"", StatusFailed, StatusFailed},
		{StatusPending, StatusFailed, StatusFailed},
		{StatusSent, StatusFailed, StatusSent},
		{StatusFailed, StatusSent, StatusSent}, // a retry that succeeded
	}
	for _, c := range cases {
		if got := MergeStatus(c.current, c.next); got != c.want {
			t.Errorf("MergeStatus(%q, %q) = %q, want %q", c.current, c.next, got, c.want)
		}
	}
}

func TestCodedError(t *testing.T) {
	cause := errors.New("timeout")
	err := fmt.Errorf("send: %w", ErrSendFailed.Wrap(cause).With("reason", "timeout"))

	if !errors.Is(err, ErrSendFailed) {
		t.Error("wrapped coded error does not match its sentinel")
	}
	if errors.Is(err, ErrNotLoggedIn) {
		t.Error("coded error matches a different code")
	}
	if !errors.Is(err, cause) {
		t.Error("cause lost")
	}
	ce := AsCoded(err)
	if ce.Code != CodeSendFailed || ce.Params["reason"] != "timeout" {
		t.Errorf("AsCoded = %+v", ce)
	}
	if ErrSendFailed.Params != nil {
		t.Error("With mutated the sentinel")
	}
	if got := AsCoded(errors.New("plain")); got.Code != CodeUnknown {
		t.Errorf("plain error code = %q, want %q", got.Code, CodeUnknown)
	}
	if AsCoded(nil) != nil {
		t.Error("AsCoded(nil) != nil")
	}
}

func TestManifestValidate(t *testing.T) {
	ok := Manifest{ID: "whatsapp", Name: "WhatsApp", ShortName: "WA", LoginMethods: []LoginMethod{LoginQR}}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	bad := []Manifest{
		{ID: "WhatsApp", Name: "x", ShortName: "XX", LoginMethods: []LoginMethod{LoginQR}},
		{ID: "w", Name: "x", ShortName: "XX", LoginMethods: []LoginMethod{LoginQR}},
		{ID: "what-sapp", Name: "x", ShortName: "XX", LoginMethods: []LoginMethod{LoginQR}},
		{ID: "whatsapp", LoginMethods: []LoginMethod{LoginQR}},
		{ID: "whatsapp", Name: "x"},
		{ID: "whatsapp", Name: "x", ShortName: "XX", LoginMethods: []LoginMethod{"carrier-pigeon"}},
		{ID: "whatsapp", Name: "x", ShortName: "wa", LoginMethods: []LoginMethod{LoginQR}},
	}
	for _, m := range bad {
		if m.Validate() == nil {
			t.Errorf("invalid manifest accepted: %+v", m)
		}
	}
}

func TestNewAccountID(t *testing.T) {
	uuid := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	seen := map[AccountID]bool{}
	for range 100 {
		id := NewAccountID()
		if !uuid.MatchString(string(id)) {
			t.Fatalf("%q is not a v4 UUID", id)
		}
		if seen[id] {
			t.Fatalf("duplicate id %q", id)
		}
		seen[id] = true
	}
}

func TestEventStamping(t *testing.T) {
	ev := MessageEvent{Message: Message{ID: "m"}}.stamped("acc")
	got := ev.(MessageEvent)
	if got.AccountID != "acc" || got.Message.AccountID != "acc" {
		t.Errorf("not stamped: %+v", got)
	}
	// Engines cannot attribute events to another account.
	spoofed := ChatUpdatedEvent{AccountID: "other", Chat: Chat{AccountID: "other"}}.stamped("acc").(ChatUpdatedEvent)
	if spoofed.AccountID != "acc" || spoofed.Chat.AccountID != "acc" {
		t.Errorf("spoofed account kept: %+v", spoofed)
	}
}

func TestEventQueue(t *testing.T) {
	q := NewEventQueue()
	// Push never blocks, even with nobody reading.
	for i := range 1000 {
		q.Push(SyncProgressEvent{Done: i})
	}
	for i := range 1000 {
		select {
		case ev := <-q.Events():
			if got := ev.(SyncProgressEvent).Done; got != i {
				t.Fatalf("event %d arrived as %d", i, got)
			}
		case <-time.After(time.Second):
			t.Fatal("event not delivered")
		}
	}
	q.Close()
	q.Close()
	q.Push(SyncProgressEvent{}) // dropped, must not panic
	select {
	case _, open := <-q.Events():
		if open {
			t.Error("event delivered after Close")
		}
	case <-time.After(time.Second):
		t.Fatal("channel not closed")
	}
}

// TestCodesHaveTranslations keeps backend codes and frontend locale keys in sync (ADR 0004).
func TestCodesHaveTranslations(t *testing.T) {
	raw, err := os.ReadFile("../../frontend/src/locales/en.json")
	if err != nil {
		t.Fatal(err)
	}
	var locale map[string]any
	if err := json.Unmarshal(raw, &locale); err != nil {
		t.Fatal(err)
	}
	for _, code := range KnownCodes() {
		if !hasKey(locale, code) {
			t.Errorf("code %q has no translation in en.json", code)
		}
	}
}

func hasKey(tree map[string]any, key string) bool {
	head, rest, nested := strings.Cut(key, ".")
	v, ok := tree[head]
	if !ok {
		return false
	}
	if !nested {
		_, isString := v.(string)
		return isString
	}
	sub, ok := v.(map[string]any)
	return ok && hasKey(sub, rest)
}
