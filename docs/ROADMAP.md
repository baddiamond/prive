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
- ⬜ Switch the store to `ncruces/go-sqlite3` and add the FTS5 table, before the first release
  ([ADR 0006](adr/0006-sqlite-driver-ncruces.md))

**Test:** `go test ./...` (includes multi-account scenarios with the fake engine).

### ⬜ 2. WhatsApp login

- WhatsApp engine on whatsmeow
- First-run onboarding ([`docs/ONBOARDING.md`](ONBOARDING.md)): welcome, responsible-use agreement, connect, link, ready
- Add-account flow with QR code in the app
- Session persisted per account; automatic `Connect` on startup; reconnection with backoff
- Logout / remove account
- Unlimited accounts

**Test on macOS:** link two WhatsApp numbers, quit the app, reopen it — both reconnect without scanning again.

### ⬜ 3. Chat list

- Initial history sync into SQLite
- Unified inbox and per-account filter, sorted by last message
- Unread counters per chat, per account and in total

**Test:** your real chats appear in the right order with correct unread counts.

### ⬜ 4. Message history

- Paginated history (infinite scroll upwards), cached locally

**Test:** open a chat and scroll back through older messages; reopening is instant.

### ⬜ 5. Real-time messaging

- Send and receive text messages live
- Message status (sent / delivered / read), mark chat as read

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

- ⬜ **Encryption at rest, on by default** — random data key in the OS keychain, 12-word recovery phrase
  with Privé's own word lists, optional app password ([ADR 0007](adr/0007-encryption-at-rest.md)).
  Depends on the SQLite driver of [ADR 0006](adr/0006-sqlite-driver-ncruces.md), adopted in phase 1.
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
