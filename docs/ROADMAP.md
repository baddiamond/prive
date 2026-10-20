# Roadmap

Work proceeds **one phase at a time**. Each phase ends with a "how to test it" section
and needs the maintainer's approval before the next one starts.

Legend: ⬜ not started · 🟨 in progress · ✅ done

## Phase 1 — MVP: WhatsApp

### 🟨 0. Bootstrap

- `LICENSE`, `.gitignore`, `README.md`, `CLAUDE.md`, `AGENTS.md`, `CONTRIBUTING.md`, docs and ADRs ✅ (proposal)
- Wails v2 project with React + TypeScript + Tailwind + Zustand
- react-i18next with `en.json` / `es.json`, OS language detection, language selector persisted locally
- Light/dark theme (follows the OS, overridable in Settings)
- GitHub issue/PR templates; CI running `go vet`, `go test`, frontend lint and typecheck

**Test on macOS:** `wails dev` opens an empty window; switching language and theme works and survives a restart.

### ⬜ 1. Core & store

- `core`: `Engine`, `Session`, `Manifest`, `Capabilities`, events, `CodedError`
- `engines`: registry, `all` package, `_template`, `enginetest` conformance suite
- `core.Manager` with concurrent, isolated sessions per account
- `store`: SQLite (`modernc.org/sqlite`), embedded migrations, accounts/chats/messages/settings
- A fake in-memory engine used by tests

**Test:** `go test ./...` (includes multi-account scenarios with the fake engine).

### ⬜ 2. WhatsApp login

- WhatsApp engine on whatsmeow
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

**Test:** `wails build -platform darwin/universal` produces `build/bin/prive.app`.

## Later (not scheduled)

- Media: images, voice notes, documents; reactions, replies, edits, deletes
- Local full-text search across all accounts
- Native notifications, tray / menu bar icon, badge count
- Engines: Telegram, Signal, Slack, Instagram, Discord, Matrix… (community-driven)
- Optional encryption at rest for the local database
- Out-of-process engines (any language, JSON over stdio)
- Signed and notarized releases, auto-update (opt-in, no tracking)
