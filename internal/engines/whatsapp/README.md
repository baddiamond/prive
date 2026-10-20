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
| Chat list and initial history sync | 3 ✅ |
| Message history, older messages on demand from the phone | 4 ✅ |
| Sending, delivery and read receipts, mark as read | 5 ✅ |

The manifest declares `SendText`, `ReadReceipts` and `HistoryOnDemand`; every other capability is added in the
phase that implements it.

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
- **History sync:** `INITIAL_BOOTSTRAP`, `RECENT` and `FULL` batches are stored as authoritative (unread counts,
  archive, pin, mute); `ON_DEMAND` batches only add messages; `PUSH_NAME` batches update contact names.
  Chats are keyed by phone-number JID even when WhatsApp sends a LID. Names come from the contact store
  (full name, first name, business name, push name) and group info, falling back to the number.
- **Older messages on demand:** `RequestOlderHistory` sends a history sync request to the phone
  (`BuildHistorySyncRequest` + `SendPeerMessage`) anchored on the oldest stored message. The answer is an
  `ON_DEMAND` history sync: its messages are stored without touching newer chat state, and a `history:result`
  event says how many arrived and whether the phone has nothing older (`endOfHistoryTransferType`). If the phone
  does not answer within 20 seconds, the request is sent again with the chat's LID (the phone may know a chat by
  either address); after that the UI shows `errors.history.noAnswer`.
- **Sending:** text messages go out with `SendMessage` to the chat as stored (phone number or group); whatsmeow
  handles LID addressing. Offline, sending fails at once with `errors.account.connectFailed`.
- **Receipts:** delivery and read receipts from recipients become `message:status` (delivery only moves forward).
  In groups the first receipt sets the status, like the single grey ticks WhatsApp shows before everyone has read.
  A `read-self` receipt (the user read the chat on the phone) marks the chat read in Privé.
- **Mark as read:** `MarkRead` sends read receipts for the chat's unread messages, per sender in groups. If the
  user turned read receipts off in WhatsApp, whatsmeow sends `read-self`, so contacts are not told.
- **Senders:** incoming messages store their author by phone number when known, with the best name from the
  contact store, so group messages show who wrote them.
- **Chat state from other devices:** archive, pin, mute and mark-as-read app-state events become `chat:patch`.
- **Logs:** only warnings and errors from whatsmeow are kept, and long numbers (phone numbers, JIDs) are redacted.

## Known limitations

- Accounts linked before phase 3 must be linked again to import their history: WhatsApp sends it only once,
  right after linking.
- After linking, only the recent history WhatsApp sends is imported; older messages are asked from the phone as you
  scroll, so the phone must be online for that.
- Media is shown as its kind (photo, voice message, …); downloading media comes later.

## Tests

- `go test ./internal/engines/whatsapp` — conformance suite (without network), event mapping, log redaction.
- `go test -tags smoke -run TestSmokeQR -v ./internal/engines/whatsapp` — connects to WhatsApp for real and waits for
  the first QR code (needs network access; nothing is linked).
- Linking itself needs a phone: use `go run ./cmd/prive-dev link` (see the main README).
