# 0007 — Encryption at rest and recovery phrase

- **Status:** accepted — partly implemented: DEK, keychain slot, per-database derived keys and encrypted
  databases (phase 1); recovery phrase slot, banner, dialog and unlock screen (phase 3b). App password, OS
  authentication before showing the phrase, and media encryption are still planned.
- **Date:** 2026-10-20

## Context

Privé keeps everything on the user's computer: messages, media, search indexes, and the session keys
of every linked account. Today that data relies on the OS disk encryption, which many users never turn on,
and which does not cover copies (Time Machine, cloud-synced folders, a copied data folder).

Requirements:

- On by default, invisible in daily use (no password at every start unless the user asks for one).
- Light enough for old, low-end computers (see [`docs/REQUIREMENTS.md`](../REQUIREMENTS.md)).
- Recoverable on a new computer or after an OS reinstall, without any server.
- Honest about what it protects.

## Decision

### Keys

- On first run, generate a random 256-bit **data encryption key (DEK)**.
- The DEK encrypts:
  - `prive.db` and every engine's private store, opened through the `adiantum` VFS
    ([ADR 0006](0006-sqlite-driver-ncruces.md)); the WAL and temporary files are covered by the VFS,
    and `temp_store=MEMORY` keeps sorts and temp tables off disk;
  - media files on disk, with **XChaCha20-Poly1305** (authenticated), one subkey per file derived with HKDF.
- Per-database keys are derived from the DEK with HKDF and a label (`prive.db`, `engine/<engineId>/<accountId>`),
  so one leaked derived key does not expose the others.

### Key slots

The DEK is never stored in clear. It is wrapped (XChaCha20-Poly1305) into one or more **slots** stored in
`<DataDir>/keys.json`. Any slot can unwrap it:

| Slot | Unwrapped by | Default |
|---|---|---|
| **System keychain** | macOS Keychain, Windows Credential Manager (DPAPI), Linux Secret Service, via `zalando/go-keyring` | Always on — the app opens without prompting |
| **Recovery phrase** | 12 words (128 bits of entropy) from a Privé word list | Always created; shown to the user once there is data to lose |
| **App password** | User password through **Argon2id** (`golang.org/x/crypto/argon2`) | Off; turning it on removes the keychain slot so the password is required at start |

Changing the password or regenerating the phrase rewraps the DEK; the database is never re-encrypted for that.

### Recovery phrase

- **Word lists:** our own lists of 2,048 words per language, MIT-licensed, in `internal/crypto/wordlists/`
  (English and Spanish first). Rules: common, short, unambiguous words, unique by their first four letters,
  no offensive words. We do **not** use BIP39 lists: their license is unclear
  ([ADR 0011](0011-licensing-policy.md)) and a "12 words" format identical to crypto wallets invites scams.
  A word list is frozen once released; a new language adds a new list and never changes an existing one.
- **Format:** 12 words = 128 bits; the last word carries a checksum so typos are caught. The phrase records
  its list, and recovery accepts any shipped list (detected automatically).
- **When it is shown:** never during onboarding (onboarding stays at five steps, under a minute).
  After the first account finishes its initial sync, a calm, non-blocking banner offers
  "Save your recovery phrase". The user writes it down and confirms three random words.
  The banner can be postponed and comes back on later starts until confirmed.
- **Always available** in Settings → Privacy → Recovery phrase, behind OS authentication.
- **Used when:** moving to a new computer, after reinstalling the OS or restoring a backup,
  when the keychain entry is gone, or when the app password is forgotten.

### Honest scope

UI copy and docs say exactly this:

- Protects against: a lost or stolen computer, disk images and backups, other users of the same computer,
  someone copying the data folder.
- Does **not** protect against malware running as the user while Privé is unlocked.
- `adiantum` gives confidentiality, not integrity: a modified page is not detected
  (SQLCipher's per-page HMAC would, at the cost of CGO — see ADR 0006).
- If both the keychain entry and the recovery phrase are lost, local data cannot be recovered.
  Accounts can be linked again and networks may sync part of the history back.

"Delete everything" (Settings and `PRIVACY.md`) also deletes the keychain entry.

### Implementation notes (phase 3b)

- The phrase's 128 bits are derived from the DEK with HKDF and the slot ID (`recovery-1`), and the slot's
  wrapping key is derived from those bits with HKDF. So the phrase can be shown again at any time while the app is
  unlocked, without storing it anywhere; regenerating it means a new slot ID (`recovery-2`) and removing the old
  slot. Plain HKDF (no Argon2id) is enough because the input already has 128 bits of entropy.
- Checksum: the high 4 bits of `SHA-256("prive/recovery/v1/" ‖ entropy)`, stored in the last word.
- Lists never share a word, so the language is detected from the words; every language spells the same bits.
  Input ignores case, accents, numbering and punctuation, and four letters identify a word.
- Data created before phase 3b gets its recovery slot on the next start.
- Settings → Privacy shows the words after a click, without OS authentication for now; Touch ID / Windows Hello
  arrives with the lock feature of [ADR 0010](0010-chat-and-account-privacy.md) and will guard this too.

## Consequences

- Encrypted by default with no extra step for most users; strong protection for copies and backups.
- Adiantum is fast on CPUs without AES instructions, so low-end computers pay almost nothing.
- New code paths to test: first run, unlock, recovery, password change, keychain missing.
- `PRIVACY.md` changes from "not encrypted yet" to the scope above when this ships.
- New translation namespaces: `encryption.*`, `recovery.*`, `unlock.*`.

## Alternatives considered

- **Password required at every start** — strong, but most users would turn encryption off or forget it.
- **Keychain only, no recovery** — the first OS reinstall loses everything.
- **BIP39 lists and format** — unclear license, crypto-wallet look-alike, non-English lists discouraged by BIP39 itself.
- **SQLCipher** — authenticated, but requires CGO (see ADR 0006).
