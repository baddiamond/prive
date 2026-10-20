# Roadmap

Work proceeds **one phase at a time**. Each phase ends with a "how to test it" section
and needs the maintainer's approval before the next one starts.

Legend: ⬜ not started · 🟨 in progress · ✅ done

## Phase 1 — MVP: WhatsApp

### ✅ 0. Bootstrap

- `LICENSE`, `.gitignore`, `README.md`, `CLAUDE.md`, `AGENTS.md`, `CONTRIBUTING.md`, docs and ADRs ✅ (proposal)
- Brand identity: guide, logo, app/tray icons, banners, design tokens and generator ✅
- Legal and community: disclaimer, acceptable use, privacy, security policy, code of conduct ✅
- Onboarding design and spec ✅ ([`docs/ONBOARDING.md`](ONBOARDING.md))
- UI design and spec: inbox, states, search, details, add account, settings, notifications ✅ ([`docs/UI.md`](UI.md))
- Wails v2 project with React + TypeScript + Tailwind + Zustand ✅
- react-i18next with `en.json` / `es.json`, OS language detection, language selector persisted locally ✅
- Light/dark theme from the brand tokens (follows the OS, overridable in Settings) ✅
- GitHub issue/PR templates; CI running `go vet`, `go test`, frontend lint and typecheck ✅
- Brought forward from phase 1: `internal/store` with embedded migrations and the `settings` table only,
  so preferences live in SQLite from day one ✅
- App shell from the UI spec in its empty state: account rail, chat list header and filters, welcome screen,
  and Settings (General, Accounts, Privacy, About) ✅

**Test on macOS:** `wails dev` opens an empty window; switching language and theme works and survives a restart.

### 🟨 1. Core & store

- `core`: `Engine`, `Session`, `Manifest`, `Capabilities`, events, `CodedError` ✅
- `engines`: registry, `all` package, `_template`, `enginetest` conformance suite ✅
- `core.Manager` with concurrent, isolated sessions per account ✅
- `store`: accounts/chats/messages tables (SQLite, migrations and settings already exist since phase 0) ✅
- A fake in-memory engine used by tests ✅
- Bound methods reject with `{code, params}`; backend codes are checked against `en.json` in tests ✅
- Store on `ncruces/go-sqlite3` with the FTS5 table and search-exclusion flags
  ([ADR 0006](adr/0006-sqlite-driver-ncruces.md), [ADR 0009](adr/0009-hybrid-local-search.md)) ✅
- Encryption at rest, invisible: data key in the OS keychain, every database encrypted with its own derived key,
  engines open databases only through `SessionDeps.OpenDB` ([ADR 0007](adr/0007-encryption-at-rest.md)) ✅

**Test:** `go test ./...` (includes multi-account scenarios with the fake engine).

### ✅ 2. WhatsApp login

**2a — backend**

- WhatsApp engine on whatsmeow, store encrypted through `SessionDeps.OpenDB`; passes the conformance suite ✅
- Session persisted per account; automatic `Connect` on startup; reconnection with backoff (whatsmeow) ✅
- Logout / remove account (unlinks on the phone when online) ✅
- Unlimited accounts ✅
- API for the UI: `ListEngines`, `ListAccounts`, `AddAccount`, `UpdateAccount`, `Login`, `CancelLogin`,
  `RemoveAccount`; events `login:*`, `account:state`, `account:updated`, `account:removed` ✅
- Developer tool `cmd/prive-dev` to link and list accounts from the terminal ✅

**Test 2a on macOS:** quit Privé, run `go run ./cmd/prive-dev link Personal green` and scan the QR code, repeat
with a second number, then run `go run ./cmd/prive-dev accounts` again later: both show `connected` without scanning.

**2b — interface**

- First-run onboarding ([`docs/ONBOARDING.md`](ONBOARDING.md)): welcome, responsible-use agreement, connect, link, ready;
  resumes after a restart; asks to agree again when the legal version changes ✅
- Add-account flow with QR code in the app (also "Link again" for logged-out accounts) ✅
- Accounts in the rail (color, initial, network tag, issue badge, ⌘0–9); status banners in the chat list ✅
- Settings → Accounts (rename and recolor, log out with confirmation, link again, retry) ✅
- `devfake` build tag offers the in-memory engine to develop the account UI without a phone ✅

**Test on macOS:** link two WhatsApp numbers, quit the app, reopen it — both reconnect without scanning again.

### ✅ 3. Chat list

**3a**
- Initial history sync into SQLite (the recent history WhatsApp sends after linking; chat names from contacts,
  push names and group info; archive, pin, mute and read state from app-state sync) ✅
- Unified inbox and per-account filter, pinned first, then by last message; "Unread" and "Groups" filters ✅
- Unread counters per chat, per account and in total ✅
- Import progress in the chat list; selecting a chat shows a placeholder until phase 4 ✅

**3b**
- Recovery phrase with Privé's own English and Spanish word lists; banner once the first chats are imported,
  dialog with a three-word check, Settings → Privacy rows ([ADR 0007](adr/0007-encryption-at-rest.md)) ✅
- Unlock screen: on a new computer or when the keychain entry is gone, the phrase unlocks the data ✅

**Test:** your real chats appear in the right order with correct unread counts. Accounts linked before phase 3
must be removed and linked again to import their history.

### ✅ 4. Message history

- Conversation view: day separators, incoming and outgoing bubbles, sender names in groups with their own
  color, media shown by kind until media support arrives ✅
- Paginated history (infinite scroll upwards, cursor by time and id), cached locally; reopening a chat is
  instant (the last 20 chats stay in memory) ✅
- Older history on demand (`Capabilities.HistoryOnDemand`): when the stored messages end, Privé asks the
  phone for 50 more (whatsmeow `BuildHistorySyncRequest` + `SendPeerMessage`). The answer arrives as an
  `ON_DEMAND` history sync and a `history:result` event; if the phone does not answer, the chat's other
  address (phone number or LID) is tried, then the user sees why and can retry ✅

**Test:** open a chat and scroll back through older messages; reopening is instant.

### ✅ 5. Real-time messaging

- Send and receive text messages live; the message shows at once as "Sending", and a failure stays as
  "Not sent" with the reason and **Retry** ✅
- Message status (sent / delivered / read) from the network's receipts ✅
- Mark chat as read when it is open and the window has focus; read receipts follow the user's own setting on
  the network (WhatsApp sends "read-self" when read receipts are off). Reading a chat on the phone clears it
  in Privé too ✅
- Drafts are kept per chat while the app runs ✅

**Test:** message yourself from your phone and see it instantly; reply from Privé and see it on the phone.

### ⬜ 6. MVP polish

- Keyboard shortcuts (switch chat, search, send)
- Startup time and memory profiling
- Builds: `darwin/universal`, `windows/amd64`, `linux/amd64`
- Platform metadata (`Info.plist`, Windows manifest) committed under `build/` next to the brand icons

**Test:** `wails build -platform darwin/universal` produces `build/bin/prive.app`.

## Future features

Not scheduled yet. Each one starts only with the maintainer's approval, like any phase.
Decisions are recorded in the linked ADRs; system requirements and defaults are in
[`REQUIREMENTS.md`](REQUIREMENTS.md).

### Privacy and security

- 🟨 **Encryption at rest, on by default** — data key in the OS keychain and encrypted databases (phase 1),
  recovery phrase and unlock screen (phase 3b) are done. Still to do: optional app password, OS authentication
  before showing the phrase, media encryption ([ADR 0007](adr/0007-encryption-at-rest.md)).
- ⬜ **Exclude from search and lock** chats and accounts, unlocked with Touch ID / Windows Hello
  ([ADR 0010](adr/0010-chat-and-account-privacy.md), step 1).
- ⬜ **Hidden chats and accounts** in an encrypted vault, revealed with a secret code
  ([ADR 0010](adr/0010-chat-and-account-privacy.md), step 2; needs encryption at rest).

### Local intelligence (on this computer only)

- ⬜ **Full-text search** across all accounts with FTS5; schema created in phase 1
  ([ADR 0009](adr/0009-hybrid-local-search.md)).
- ⬜ **Semantic search** (opt-in) with Model2Vec embeddings, conversation windows and result fusion
  ([ADR 0009](adr/0009-hybrid-local-search.md)).
- ⬜ **Voice note transcription** (opt-in) with whisper.cpp built in; transcripts are searchable
  ([ADR 0008](adr/0008-local-voice-transcription.md)). Needs media support.
- ⬜ **Benchmark spike** to replace the estimates in `REQUIREMENTS.md` with measurements.

### Messaging and platform

- ⬜ Media: images, voice notes, documents; reactions, replies, edits, deletes
- ⬜ Native notifications, tray / menu bar icon, badge count
- ⬜ Engines: Telegram, Signal, Slack, Instagram, Discord, Matrix… (community-driven)
- ⬜ Out-of-process engines (any language, JSON over stdio)
- ⬜ Signed and notarized releases, auto-update (opt-in, no tracking)

### Project hygiene

- ⬜ License checks in CI (`go-licenses`, frontend license checker) and `THIRD_PARTY_NOTICES.md`
  ([ADR 0011](adr/0011-licensing-policy.md)).
