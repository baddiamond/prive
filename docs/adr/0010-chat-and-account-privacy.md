# 0010 — Per-chat and per-account privacy: exclude, lock, hide

- **Status:** accepted (planned — see [roadmap](../ROADMAP.md#future-features))
- **Date:** 2026-10-20

## Context

People share computers, present their screen, or simply want some conversations out of sight.
WhatsApp offers "Chat lock" with a hidden folder and a secret code. Privé brings many accounts into one place,
so the need applies to whole accounts too (e.g. keep a personal number out of "All inboxes").

## Decision

### Levels for a chat

| Level | Effect |
|---|---|
| **Exclude from search** | Not shown in ⌘K and never indexed: no full-text, vectors or automatic transcription. |
| **Locked** | Moves to a "Locked chats" row at the top of the chat list. Opening it requires OS authentication (Touch ID, Windows Hello) or the app password. No message previews; notifications say only "New message". Not in search while locked. |
| **Hidden** | Appears nowhere: chat list, search, unread counts, badge and notifications. Revealed by typing the **secret code** in ⌘K. |

### Options for an account

- Exclude from "All inboxes".
- Exclude from search.
- Lock or hide the whole account, with the same rules as chats (rail entry included).

### Hidden data lives in a vault

A flag in the main database would only hide the chat from the UI. Hidden chats and everything they produce
(messages, transcripts, full-text and vector index) are stored instead in **`vault.db`**, encrypted with its own
**vault key**:

- The vault key is wrapped by the secret code (Argon2id) and by the recovery phrase
  ([ADR 0007](0007-encryption-at-rest.md)), which stays the master key.
- **Messages that arrive while the vault is closed:** the vault has an X25519 key pair. Incoming messages for
  hidden chats are sealed to the vault's public key into a queue in the main database, and moved into
  `vault.db` the next time it is opened.
- **Routing without names:** the main database keeps only a salted hash of each hidden chat's id, so it can
  route incoming messages without storing who the chat is with.

### UI

- Details panel: **Lock** joins Mute · Pin · Archive; "Exclude from search" and "Hide" sit in the same menu.
- Settings → Privacy: secret code, "Unlock with Touch ID / Windows Hello", per-account options.
- Copy is honest and calm (see BRAND.md voice rules).

### Honest scope

- This hides the chat **on this computer only**. The network's own apps (e.g. WhatsApp on the phone) still show it.
- Someone with access to the main database could test whether a specific, known chat id is hidden by hashing it.
  We accept this trade-off and document it.
- Locked (not hidden) chats are protected by OS authentication at the UI level; their data stays in the main
  encrypted database.

### Phases

1. **Exclude from search** and **Locked** (flags and UI; simple).
2. **Hidden** with the vault, after encryption at rest (ADR 0007).

## Consequences

- A privacy feature users already know from WhatsApp, extended to every network and to whole accounts.
- The vault adds a second database, a sealed queue and routing logic, all covered by tests.
- Unread counters, badge and notification code must respect these levels everywhere.
- New translation namespaces: `privacy.lock.*`, `privacy.hide.*`, `privacy.exclude.*`.

## Alternatives considered

- **Hidden as a UI flag only** — anyone opening the database sees the chats.
- **Separate profile per hidden set** (a second data folder) — heavy, duplicates sessions and accounts.
- **Keep incoming messages for hidden chats unencrypted until unlock** — leaks exactly what we want to hide.
