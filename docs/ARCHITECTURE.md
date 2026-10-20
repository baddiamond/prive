# Architecture

> Status: phases 0–1 implemented: Wails shell, `platform`, `store`, `core` (contract, events, errors, Manager),
> engine registry, template, conformance suite and a fake engine. No real network engine yet (phase 2).
> Changes to the engine contract must update this document. The Go doc comments in `internal/core` are the
> precise reference; this page is the overview.

## Goals

1. **Private & local-first** — all data on the user's machine; the only network traffic is to the messaging networks.
2. **Fast & light** — one native process, quick startup, low memory.
3. **Extensible** — new networks ("engines") can be added without touching the core or the UI.
4. **Unlimited multi-account** — any number of accounts per network, all running concurrently.
5. **Approachable** — a newcomer (human or AI) can understand the system from this page.

## Big picture

```
┌──────────────────────────── Privé process ─────────────────────────────┐
│                                                                        │
│  frontend/ (React)                                                     │
│    ├── calls bound Go methods (App.ListChats, App.SendMessage, …)      │
│    └── listens to Wails events (message:new, chat:updated, …)          │
│                 ▲                        │                             │
│                 │ events                 │ calls                       │
│                 │                        ▼                             │
│  app.go  ── thin API layer: validates input, maps errors to codes      │
│                 │                                                      │
│                 ▼                                                      │
│  internal/core.Manager                                                 │
│    ├── owns one Session per Account (map[AccountID]Session)            │
│    ├── persists incoming data through internal/store                   │
│    └── fans out session events to the UI (adds accountId)              │
│                 │                                                      │
│      ┌──────────┼──────────────┬──────────────┐                        │
│      ▼          ▼              ▼              ▼                        │
│  whatsapp    whatsapp      telegram        …more engines               │
│  session A   session B     session C                                   │
│  (whatsmeow) (whatsmeow)                                               │
│                                                                        │
│  internal/store (SQLite, modernc.org/sqlite)                           │
└────────────────────────────────────────────────────────────────────────┘
```

## Repository layout

```
prive/
├── main.go                     # Wails bootstrap: window options, lifecycle hooks, embedded assets
├── app.go                      # Methods bound to the frontend (thin; delegates to core.Manager)
├── wails.json
├── internal/
│   ├── core/                   # Network-agnostic domain
│   │   ├── engine.go           # Engine, Session, Manifest, Capabilities
│   │   ├── types.go            # Account, Chat, Message, IDs
│   │   ├── events.go           # Event types emitted by sessions and by the manager
│   │   ├── errors.go           # CodedError + well-known codes
│   │   ├── queue.go            # EventQueue: unbounded, non-blocking event channel for engines
│   │   ├── repository.go       # Repository: the persistence the Manager needs (store implements it)
│   │   └── manager.go          # Account lifecycle, routing, persistence of events, event fan-out
│   ├── engines/
│   │   ├── registry.go         # Register / Get / List engines
│   │   ├── all/all.go          # Blank-imports every bundled engine
│   │   ├── enginetest/         # Conformance suite every engine must pass
│   │   ├── _template/          # Skeleton for new engines (ignored by the Go toolchain; CI tests a copy)
│   │   ├── fake/               # In-memory engine for tests (never registered in the app)
│   │   └── whatsapp/           # whatsmeow-based engine (phase 2)
│   ├── store/                  # SQLite access + embedded migrations
│   └── platform/               # Data directory per OS, system locale detection
├── frontend/
│   └── src/
│       ├── i18n.ts             # react-i18next setup
│       ├── locales/{en,es}.json
│       ├── lib/                # Typed wrappers for Wails bindings and events
│       ├── state/              # Zustand stores
│       └── components/         # AccountRail, ChatList, ChatView, Composer, AddAccount, Settings…
└── docs/
```

Dependency rules (enforced by review, later by a lint check):

- `core` imports nothing from `engines`, `store` or Wails. (Its external tests, package `core_test`,
  may use `store` and `engines/fake` to exercise the Manager end to end.)
- An engine imports `core` (and its own third-party library) — never another engine, `store` or Wails.
- `store` imports `core` types only.
- Only `main.go` / `app.go` know about Wails.

## Core concepts

### Account

A user's login on one network. Privé creates a local UUID for it; the network's own identity
(e.g. a WhatsApp JID) is stored as `RemoteID` once login succeeds.

```go
type AccountID string // local UUID

type Account struct {
    ID          AccountID
    EngineID    string // "whatsapp", "telegram", …
    RemoteID    string // network-side identity, empty until logged in
    DisplayName string // user-editable label, e.g. "Work"
    Color       string // UI accent to tell accounts apart
    CreatedAt   time.Time
}
```

### Engine

A factory for one network, registered once at startup.

```go
type Engine interface {
    // Manifest describes the engine to the core and the UI.
    Manifest() Manifest
    // NewSession creates (but does not connect) a session for one account.
    NewSession(account Account, deps SessionDeps) (Session, error)
}

type Manifest struct {
    ID           string       // stable, lowercase: "whatsapp"
    Name         string       // brand name, not translated: "WhatsApp"
    LoginMethods []LoginMethod // e.g. LoginQR, LoginPhoneCode, LoginToken
    Capabilities Capabilities
}

type Capabilities struct {
    SendText     bool
    ReadReceipts bool
    Typing       bool
    Media        bool
    Reactions    bool
    Edit         bool
    Delete       bool
    Replies      bool
}

// SessionDeps is what the core gives each session.
type SessionDeps struct {
    DataDir string       // private folder for this account's engine data
    Logger  *slog.Logger // already tagged with engine and account
}
```

The UI shows or hides features from `Capabilities`; it never checks the engine ID to decide behavior.

### Session

One running account. All methods are safe to call concurrently.

```go
type Session interface {
    // Connect resumes a stored login. Returns ErrNotLoggedIn if there is none.
    Connect(ctx context.Context) error
    // Login starts an interactive login (QR, code…). Progress arrives as events
    // (LoginQREvent, LoginSuccessEvent, LoginFailedEvent).
    Login(ctx context.Context, method LoginMethod) error
    // Logout unlinks the account on the network side and wipes its engine data.
    Logout(ctx context.Context) error

    ListChats(ctx context.Context, opts ListChatsOptions) ([]Chat, error)
    GetMessages(ctx context.Context, chatID ChatID, opts GetMessagesOptions) ([]Message, error)
    SendMessage(ctx context.Context, chatID ChatID, msg OutgoingMessage) (Message, error)

    // SubscribeEvents returns the same channel every time; Close closes it.
    // Sending must never block the network library (engines use core.EventQueue).
    SubscribeEvents() <-chan Event
    // Close disconnects without logging out. Safe to call more than once.
    Close() error
}
```

### Events

Sessions emit typed events; the manager persists what is needed, adds the `accountId` and forwards
to the UI as Wails events.

| Wails event | Payload (JSON) | When |
|---|---|---|
| `account:state` | `{accountId, state, code?}` | disconnected / connecting / connected / reconnecting / loggedOut / error |
| `account:removed` | `{accountId}` | account deleted (emitted by the Manager only) |
| `login:qr` | `{accountId, code, expiresAt}` | new QR to display |
| `login:success` | `{accountId, remoteId, displayName}` | pairing completed |
| `login:failed` | `{accountId, code}` | interactive login ended without success |
| `chat:updated` | `{accountId, chat}` | new chat, name change, unread count, last message |
| `message:new` | `{accountId, chatId, message}` | incoming or outgoing message |
| `message:status` | `{accountId, chatId, messageId, status}` | sent / delivered / read / failed |
| `sync:progress` | `{accountId, done, total}` | initial history sync |

`code` values are **translation keys**, not text (see below); a `state` is shown with the key
`account.state.<state>`. Engines leave `accountId` empty: the Manager stamps every event (and the chats and
messages inside) with the session's own account, so an engine can never write into another account.

### Errors and user-facing codes

The backend never returns human text. Errors that can reach the UI are `CodedError`:

```go
type CodedError struct {
    Code   string            // translation key, e.g. "errors.whatsapp.sendFailed"
    Params map[string]string // interpolation values, optional
    Err    error             // wrapped cause, for logs only
}
```

Bound methods reject with `{ "code": "...", "params": {...} }` (Wails `ErrorFormatter`, see `formatError` in
`app.go`). Errors without a code are logged and reach the UI as `errors.unknown`, so internal details never leak.
The frontend renders `t(code, params)`. Unknown codes fall back to `errors.unknown`.
`core.KnownCodes()` and the app's own codes are checked against `en.json` by `go test`.

## Storage

One SQLite database per installation at `<DataDir>/prive.db` (see the README for OS paths):

| Table | Purpose |
|---|---|
| `accounts` | One row per account (`id`, `engine_id`, `remote_id`, `display_name`, `color`, `position`, `created_at`) |
| `chats` | `(account_id, id)` primary key; name, kind (direct/group), unread count, last message preview and time, muted, archived |
| `messages` | `(account_id, chat_id, id)` primary key; sender, timestamp, kind, text, status, reply-to |
| `settings` | Key/value: `language`, `theme`, window size… |
| `schema_migrations` | Applied migration versions |

Engine-private data (e.g. whatsmeow's device keys and session store) lives in
`<DataDir>/engines/<engineId>/<accountId>/`, given to the session as `SessionDeps.DataDir`.
Removing an account deletes its rows and that folder.

Indexes: `chats(account_id, last_message_at DESC)` and `chats(last_message_at DESC)` for the per-account and unified
inbox, `messages(account_id, chat_id, timestamp DESC)` for history pagination. Times are stored as Unix milliseconds.

Messages reference their account (cascade delete) but not their chat, because a message can arrive before its chat.
Transactions start with `BEGIN IMMEDIATE` (`_txlock=immediate`) so concurrent writers from different accounts wait on
the busy timeout instead of failing with `SQLITE_BUSY`.

## Manager

`core.Manager` is the only component that runs sessions:

- **Start** loads accounts from the `Repository`, creates one session per account and calls `Connect` on all of
  them in parallel goroutines; it returns immediately. `ErrNotLoggedIn` becomes `loggedOut`, other failures
  become `error` with a code. An account whose engine is not bundled stays listed with `errors.engine.notFound`.
- **Event pump**: one goroutine per session reads its events in order, persists what changes (chats, messages,
  statuses, the remote ID after login), then forwards the event to the UI. Message statuses only move forward
  (`core.MergeStatus`): a late "delivered" never overwrites "read".
- **AddAccount / Login / RemoveAccount / SendMessage / ListChats / GetMessages** are what `app.go` exposes to the UI.
  Removing an account logs out (best effort, so it works offline), closes the session, deletes its rows and its
  data folder.
- **Close** cancels pending connections and logins, closes every session and waits for their goroutines.

The Manager depends on interfaces only (`EngineSource`, `Repository`), so tests run it with the fake engine and a
real SQLite store in a temp folder.

## Multi-account

- The manager starts all accounts **concurrently** at startup; a failing account never blocks the others.
- Each session reconnects on its own with exponential backoff (inspired by GOWS) and reports `account:state`.
- The UI shows an **account rail** (unified "All" inbox + one entry per account with its color and unread count)
  and a "+" button that opens the add-account flow (choose network → login method).
- A chat's replies always go through the account that owns it.

## Internationalization

- `frontend/src/locales/en.json` is the source of truth; other files mirror its keys.
- Language resolution: saved setting → OS locale (from `internal/platform`) → `en`.
- Backend codes (`errors.*`, `account.state.*`) are regular keys in the same files.
- Engine-specific text lives under `engines.<engineId>.*`.

See [`I18N.md`](I18N.md).

## Startup sequence

1. Resolve data directory, open SQLite, run migrations.
2. Load settings (language, theme) — the window can render immediately.
3. Create the manager, load accounts, create one session per account.
4. Call `Connect` on every session in parallel (non-blocking for the UI).
5. Frontend loads the cached inbox from SQLite right away; live events update it as sessions come online.

## Why these choices

Recorded as ADRs in [`docs/adr/`](adr/):

- [0001 — Wails v2 for the desktop shell](adr/0001-wails-desktop-shell.md)
- [0002 — Pure-Go SQLite](adr/0002-pure-go-sqlite.md)
- [0003 — Engines compiled into the binary](adr/0003-compiled-engines.md)
- [0004 — Backend returns codes, not text](adr/0004-backend-returns-codes.md)
- [0005 — whatsmeow called directly, no HTTP/gRPC layer](adr/0005-direct-whatsmeow.md)
