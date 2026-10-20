# CLAUDE.md

Rules for AI assistants (and humans) working on **Privé** (`prive`).
These rules apply to every session. Read them before changing anything.

> Other AI tools: `AGENTS.md` points here. This file is the single source of truth.

## 1. Language (strict)

- The maintainer may talk to you in Spanish. **Everything you write into the repository must be in English**:
  code, identifiers, comments, docstrings, commit messages, PR titles/bodies, docs, log messages and error codes.
- The only Spanish text allowed in the repo lives in `frontend/src/locales/es.json`
  (and in future `locales/<lang>.json` files for other languages).

## 2. Internationalization (strict)

- The app ships in **English (default)** and **Spanish**. More languages can be added by the community.
- Frontend uses **react-i18next**. Translations live in:
  - `frontend/src/locales/en.json` (source of truth, every key must exist here)
  - `frontend/src/locales/es.json`
- **No user-facing string is hardcoded** in components. Always use `t("some.key")`, including
  `aria-label`, `title`, placeholders, tooltips, empty states and toasts.
- When you add a key, add it to **all** locale files in the same change.
- Key naming: `area.subarea.name` in lowerCamelCase segments, e.g. `chatList.empty`, `settings.language.title`.
  Engine-specific strings go under `engines.<engineId>.*`.
- The **Go backend never returns user-facing text.** It returns stable codes
  (e.g. `errors.whatsapp.sendFailed`, `connection.state.reconnecting`) plus optional params;
  the frontend translates them.
- Initial language: the OS locale (read by the backend), falling back to English.
  The user's explicit choice in Settings is stored locally (SQLite `settings` table) and wins.
- How to add a language: see `docs/I18N.md`.

## 3. Product principles

- **Privacy first**: everything is stored locally. No intermediary servers, no telemetry, no analytics,
  no crash reporting to third parties, no accounts. Network traffic goes only to the messaging networks themselves.
- **Performance**: fast startup, low memory. Avoid heavy dependencies; justify every new one in the PR.
- **Multi-account, unlimited**: any number of accounts per network (e.g. 3 WhatsApp numbers) running at the same time.
  Never assume "one account per network".
- **Clean, documented code**: small functions, every exported identifier has a doc comment.

## 4. Architecture (summary — full version in `docs/ARCHITECTURE.md`)

- Desktop app: **Wails v2** (Go backend + web frontend). Targets macOS, Windows and Linux.
- Frontend: React + TypeScript + Tailwind, state in Zustand.
- Storage: SQLite via `modernc.org/sqlite` (pure Go, no CGO).
- Every messaging network is an **engine** in `internal/engines/<id>/`:
  - `core.Engine` = factory for one network; declares a `Manifest` (id, name, login method, capabilities).
  - `core.Session` = one running account: `Connect`, `Login`, `Logout`, `ListChats`, `GetMessages`,
    `SendMessage`, `SubscribeEvents`, `Close`.
  - Engines self-register in `internal/engines/registry.go`; bundled engines are enabled in `internal/engines/all`.
  - Every engine must pass the shared conformance suite in `internal/engines/enginetest`.
  - `internal/core` never imports a specific engine. Engines never import each other or the frontend.
- WhatsApp engine uses **whatsmeow** (`go.mau.fi/whatsmeow`) directly. No HTTP/gRPC layer in between.
  WAHA/GOWS are references only (sessions, events, reconnection), never dependencies.
- Backend → frontend real-time updates go through **Wails events** (`runtime.EventsEmit`).
  Every event payload includes `accountId`.
- The frontend never branches on engine id for behavior; it reads the engine `Manifest.Capabilities`.

## 5. Repository layout

```
main.go, app.go          Wails entry point and the thin API bound to the frontend
internal/core            Engine/Session interfaces, Manager, events, error codes
internal/engines         Registry, conformance tests, template, one folder per engine
internal/store           SQLite: accounts, chats, messages, settings, migrations
internal/platform        OS paths and system locale
frontend/                React app (src/locales holds translations)
docs/                    Architecture, guides and ADRs
```

## 6. Workflow

- Work one roadmap phase at a time (`docs/ROADMAP.md`). Do not start the next phase without the maintainer's approval.
- At the end of each phase, explain how to test it on macOS.
- Before committing: `go vet ./...`, `go test ./...`, and in `frontend/`: `npm run lint`, `npm run typecheck`.
- Commit messages: Conventional Commits in English (`feat(whatsapp): add QR login`, `docs: ...`, `fix(store): ...`).
- Significant design decisions get an ADR in `docs/adr/`.
- Never commit databases, session files, QR codes, phone numbers or message content (including in tests and fixtures).
