# Adding an engine

An **engine** connects Privé to one messaging network (WhatsApp, Telegram, Slack…).
This guide walks you through adding one. It is written so that both people and AI assistants can follow it
step by step.

> Read [`ARCHITECTURE.md`](ARCHITECTURE.md) first for the `Engine` and `Session` interfaces.
> A complete, working example is the test engine in `internal/engines/fake`.

## Before you start

1. Open an issue with the **Engine proposal** template. Describe the library you will use, its license,
   how login works, and any terms-of-service concerns.
2. Check that the library is compatible with the MIT license and does not require a third-party server
   (other than the network itself). Privé never proxies traffic through anyone else's servers.

## 1. Copy the template

```bash
cp -r internal/engines/_template internal/engines/<id>
```

`<id>` is lowercase letters and digits, stable forever (it is stored in the database): `telegram`, `signal`, `slack`.
Rename the package (`example` in the template) and replace every `TODO`. The template already passes the
conformance suite, so you can run `go test ./internal/engines/<id>` from the first minute.

## 2. Write the manifest

```go
func (e *Engine) Manifest() core.Manifest {
    return core.Manifest{
        ID:           "telegram",
        Name:         "Telegram",
        LoginMethods: []core.LoginMethod{core.LoginQR, core.LoginPhoneCode},
        Capabilities: core.Capabilities{SendText: true, ReadReceipts: true, Typing: true},
    }
}
```

Only declare capabilities you actually implement — the UI trusts them.

## 3. Implement the session

| Method | Must |
|---|---|
| `Connect` | Resume a stored login from `deps.DataDir`; return `core.ErrNotLoggedIn` if none |
| `Login` | Start the interactive flow and report progress via events (`LoginQREvent`, …) |
| `Logout` | Unlink on the network side and delete everything in `deps.DataDir` |
| `ListChats` / `GetMessages` | Map network objects to `core.Chat` / `core.Message`; support pagination options |
| `SendMessage` | Return the created `core.Message` with a provisional status; confirm later via `MessageStatusEvent` |
| `SubscribeEvents` | Return the same channel every time; never block the network library's goroutines when the reader is slow (use `core.EventQueue`) |
| `Close` | Disconnect, release resources and close the event channel, without logging out; safe to call twice |

Rules:

- **Many sessions of your engine will run at once** (multi-account). No package-level mutable state.
- Store private data **only** under `deps.DataDir`.
- Log with `deps.Logger`. Never log message text, phone numbers, tokens or keys.
- Errors that a user may see are `core.CodedError` with a key under `errors.<id>.*`.
- Reconnect on your own with backoff and emit `AccountStateEvent` on every change.
- Leave `AccountID` empty in events, chats and messages: the Manager stamps it (an engine can never write
  into another account).
- Don't persist chats or messages yourself: emit `ChatUpdatedEvent` / `MessageEvent` and the Manager stores them.
  Your `DataDir` is for the network library's own state (keys, session).

## 4. Register it

In your package:

```go
func init() { engines.Register(New()) }
```

And enable it in `internal/engines/all/all.go`:

```go
import _ "github.com/baddiamond/prive/internal/engines/<id>"
```

## 5. Pass the conformance suite

```go
func TestConformance(t *testing.T) {
    enginetest.Run(t, New(), enginetest.Options{
        Login:  func(t *testing.T, s core.Session) { /* log in against a fake or recorded server */ },
        ChatID: "some-chat", // needed when you declare SendText
    })
}
```

The suite checks the contract: manifest validity, coded errors (`Connect` without a login returns
`core.ErrNotLoggedIn`, unsupported login methods return `core.ErrLoginMethodUnsupported`), the event channel
lifecycle, many concurrent sessions, and that nothing is written outside `DataDir`. With `Options.Login` it also
checks sending, resuming the login after a restart and that `Logout` wipes `DataDir`.
Tests must not hit the real network by default.

## 6. Add translations and an icon

- Add `engines.<id>.*` keys (login instructions, engine-specific errors) to **every** file in `frontend/src/locales/`.
- Add an SVG icon at `frontend/src/assets/engines/<id>.svg` (check the brand's usage guidelines).

## 7. Document it

- Add a short `internal/engines/<id>/README.md`: library used, supported capabilities, known limitations.
- Add the engine to the table in the main README.

## Checklist for the PR

- [ ] `go vet ./...` and `go test ./...` pass, including `enginetest`
- [ ] No package-level mutable state; works with several accounts at once
- [ ] Only declared capabilities are implemented, and all declared ones work
- [ ] All user-facing strings are translation keys in every locale file
- [ ] No message content or personal data in logs, tests or fixtures
- [ ] Engine README and main README updated
