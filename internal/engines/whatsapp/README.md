# WhatsApp engine

Links WhatsApp accounts to Privé as **companion devices** (like WhatsApp Web), using
[whatsmeow](https://github.com/tulir/whatsmeow) directly in-process ([ADR 0005](../../../docs/adr/0005-direct-whatsmeow.md)).

| | |
|---|---|
| Library | `go.mau.fi/whatsmeow` (MPL-2.0); depends on `go.mau.fi/libsignal` (GPL-3.0), see [ADR 0012](../../../docs/adr/0012-gpl-binaries-with-whatsapp.md) |
| Login | QR code (`core.LoginQR`) |
| Session data | `store.db` in the account's DataDir, opened through `SessionDeps.OpenDB`, so it is encrypted with its own key |
| Shows on the phone as | "Privé" (Desktop) under Linked devices |

## Status

| Feature | Phase |
|---|---|
| Link with QR, resume on startup, automatic reconnection, log out | 2 ✅ |
| Chat list and initial history sync | 3 |
| Message history | 4 |
| Sending, delivery and read receipts | 5 |

The manifest declares no capabilities yet; each one is added in the phase that implements it.

## Behavior

- **Connect** resumes the stored link. With no link it returns `core.ErrNotLoggedIn` without touching the network.
  If the network is down at startup, whatsmeow keeps retrying in the background (`InitialAutoReconnect`).
- **Connection states:** `Connected` → connected; `Disconnected` / keep-alive timeout → reconnecting (whatsmeow
  reconnects with backoff); removed from the phone → logged out (`errors.whatsapp.loggedOutRemotely`, local chats
  are kept); another client with the same session, outdated client or temporary ban → error with a specific code.
- **Login** shows a new QR code about every 20–60 seconds until it is scanned or WhatsApp stops offering codes
  (`errors.whatsapp.qrTimeout`). Canceling (removing the half-created account) stops it.
- **Logout** unlinks the device on WhatsApp's side when online and always deletes the local session. Offline,
  it deletes local data and returns `errors.whatsapp.unlinkOnPhone` so the user can remove the device on the phone.
- **Logs:** only warnings and errors from whatsmeow are kept, and long numbers (phone numbers, JIDs) are redacted.

## Known limitations

- History that WhatsApp sends right after linking is not stored until phase 3. Accounts linked before phase 3
  will need to be linked again to import their history.

## Tests

- `go test ./internal/engines/whatsapp` — conformance suite (without network), event mapping, log redaction.
- `go test -tags smoke -run TestSmokeQR -v ./internal/engines/whatsapp` — connects to WhatsApp for real and waits for
  the first QR code (needs network access; nothing is linked).
- Linking itself needs a phone: use `go run ./cmd/prive-dev link` (see the main README).
