# Privacy

Privé has no servers, no accounts and no telemetry. This page lists exactly what happens to your data.

## What Privé stores, and where

Everything is stored **locally, on your computer**, in your user's data folder:

| OS | Location |
|---|---|
| macOS | `~/Library/Application Support/Prive` |
| Windows | `%AppData%\Prive` |
| Linux | `~/.local/share/prive` |

| Data | Why |
|---|---|
| Session keys for each connected account | So you don't need to scan a QR code every time |
| Chats, contacts' display names, messages (cache) | To open instantly and work offline |
| Settings (language, theme, account names and colors) | To remember your preferences |

The local database is **not encrypted by Privé yet** (planned: optional encryption at rest).
Use your operating system's disk encryption.

## What Privé sends, and to whom

Privé connects **only** to the messaging networks you add (for example, WhatsApp's servers) to send and
receive your messages. It does not contact any Privé server, because there is none.

- No analytics, telemetry, crash reporting or usage statistics.
- No advertising or tracking identifiers.
- No automatic update checks. When updates are added they will be **opt-in**.
- Fonts and assets are bundled; nothing is loaded from third-party CDNs.

The networks you connect have their own privacy policies, which still apply to your messages.

## Deleting your data

- **One account:** Settings → Accounts → Log out. This unlinks the device and deletes that account's
  local data.
- **Everything:** quit Privé and delete the data folder listed above.

## Changes

Any change to this page is made in public, in this repository, with its history visible in git.

_Last updated: 2026-10-20 · Version 1_
