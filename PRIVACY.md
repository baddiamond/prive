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

Privé **encrypts its databases** (settings, chats, messages, search index and each account's session keys)
with a random key created on your computer. That key is kept in your system's keychain (macOS Keychain,
Windows Credential Manager or the Linux Secret Service), so Privé opens without asking for a password.

What this protects against: a lost or stolen computer, backups and disk copies, other users of the same computer,
someone copying the data folder. What it does not protect against: malicious software running as you while Privé
is open. The encryption hides contents but does not detect tampering.

Once your chats have synced, Privé offers a **recovery phrase**: 12 words, created on your computer and never
sent anywhere, that unlock your data on a new computer or if the keychain entry is lost. Write it on paper; you can
see it again in Settings → Privacy. Anyone with the words and a copy of your data folder can read your chats, so
keep the paper safe. If both the keychain entry and the phrase are lost, the local data cannot be read; your
accounts can be linked again.

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
- **Everything:** quit Privé and delete the data folder listed above. Its key stays in your system keychain under
  `io.github.baddiamond.prive` and is useless without the folder; you can delete that entry too.

## Changes

Any change to this page is made in public, in this repository, with its history visible in git.

_Last updated: 2026-10-20 · Version 3_
