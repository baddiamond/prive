# Adding an engine

An **engine** connects Privé to one messaging network (WhatsApp, Telegram, Slack…).
This guide walks you through adding one. It is written so that both people and AI assistants can follow it
step by step.

> Status: the template and the conformance suite land in Phase 1. Until then this guide describes the target workflow.
> Read [`ARCHITECTURE.md`](ARCHITECTURE.md) first for the `Engine` and `Session` interfaces.

## Before you start

1. Open an issue with the **Engine proposal** template. Describe the library you will use, its license,
   how login works, and any terms-of-service concerns.
2. Check that the library is compatible with the MIT license and does not require a third-party server
   (other than the network itself). Privé never proxies traffic through anyone else's servers.

## 1. Copy the template

```bash
cp -r internal/engines/_template internal/engines/<id>
```

`<id>` is lowercase, no spaces, stable forever (it is stored in the database): `telegram`, `signal`, `slack`.
Rename the package and the type names inside.

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
| `SubscribeEvents` | Return a channel; never block the network library's goroutines when the reader is slow |
| `Close` | Disconnect and release resources without logging out |

Rules:

- **Many sessions of your engine will run at once** (multi-account). No package-level mutable state.
- Store private data **only** under `deps.DataDir`.
- Log with `deps.Logger`. Never log message text, phone numbers, tokens or keys.
- Errors that a user may see are `core.CodedError` with a key under `errors.<id>.*`.
- Reconnect on your own with backoff and emit `AccountStateEvent` on every change.

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
    enginetest.Run(t, New(), enginetest.Options{ /* fakes or recorded fixtures */ })
}
```

The suite checks the contract: manifest validity, event ordering, concurrent sessions, `Close` without leaks,
coded errors, and that nothing is written outside `DataDir`. Tests must not hit the real network by default.

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
