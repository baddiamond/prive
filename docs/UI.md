# UI specification

> Status: **design v0.1, ready for implementation.** Developer hand-off for the desktop UI.
> Visual rules: [`docs/brand/BRAND.md`](brand/BRAND.md) · First run: [`docs/ONBOARDING.md`](ONBOARDING.md) ·
> Prototype sources: [`design/prototypes/`](../design/prototypes/)

Privé has **one skeleton for every network**. Engines translate their data into the core model
(`Account`, `Chat`, `Message`, events); the UI renders only that model and each engine's
`Capabilities`. There are no network-specific screens.

## 1. App shell

```
┌──────┬──────────────────┬──────────────────────────────────┬──────────────┐
│ Rail │ Chat list        │ Conversation                     │ Details      │
│ 76px │ 340px            │ flex (min 420px)                 │ 320px, opt.  │
│      │                  │                                  │              │
│ All  │ Title · subtitle │ Header: avatar · name · status   │ Avatar, name │
│ P  ● │ [Search ⌘K]      │─────────────────────────────────│ via account  │
│ W  ● │ (All)(Unread)(…) │ Day separator                    │ Mute Pin Arch│
│ F    │ [Status banner]  │ Bubbles…                         │ Shared media │
│ T    │ Chat rows…       │                                  │ On this comp.│
│ [+]  │                  │─────────────────────────────────│              │
│  ⚙   │                  │ Sending as Work · WhatsApp       │              │
│      │                  │ [📎] [Message………………] [➤]         │              │
└──────┴──────────────────┴──────────────────────────────────┴──────────────┘
```

| Region | Background token | Notes |
|---|---|---|
| Rail | `rail` | Right border `divider` |
| Chat list | `surface` | Right border `divider` |
| Conversation | `bg` | Header and composer separated by `divider` |
| Details | `surface` | Left border `divider`; toggled with ⌘I or the info button |
| Overlays (search, dialogs) | scrim `rgba(5,6,8,.6)` dark / `rgba(15,16,19,.35)` light | Dialog radius `dialog` (20px) |

**Window sizes.** Minimum 960×640. Below 1180 px wide, Details opens as an overlay over the
conversation instead of a column. Below 1040 px, the chat list narrows to 300 px.

## 2. Components

All components take colors, radii and sizes from the brand tokens (`tailwind.preset.js`); all text from
i18n keys. Names below are the suggested React component names.

### 2.1 AccountRail

| Element | Spec |
|---|---|
| Logo | Mark, 30 px, decorative link to "All inboxes" |
| `RailItem` "All" | 48×48, radius `card`; inbox icon; total unread badge |
| `RailItem` account | 48×48, radius `card`, background = account color, initial in `onAccountColor`, 700 weight |
| Network tag | Bottom-right pill: engine short code (`WA`, `TG`…) from `Manifest`, 9 px, 700 |
| Unread badge | Top-right, `accent` background, `onAccent` text, "99+" cap, 2 px ring in `rail` |
| Issue badge | Bottom-left "!" circle: `warning` while reconnecting, `danger` when logged out |
| Selected | Double ring: 2 px `rail` + 2 px `accent` (box-shadow), `aria-pressed="true"` |
| Add account | Dashed 1.5 px `border`, opens Add account (⌘N) |
| Settings | Bottom, 44×44 (⌘,) |

Order: "All", then accounts in user order (drag to reorder — later). ⌘1–9 select accounts, ⌘0 selects All.
Accessible name: `"{label} · {network}"` plus state ("reconnecting", "logged out") and unread count.

### 2.2 ChatList

| Element | Spec |
|---|---|
| Title | "All inboxes" or account label (`heading`, 600); subtitle = "N accounts" or network |
| Search trigger | Button styled as a field: 38 px, 1.5 px `border`, `raised` background, ⌘K hint; opens SearchPalette |
| Filter chips | All · Unread · Groups. 30 px pills; selected = inverted (`text` bg, `bg` text); `aria-pressed` |
| Status banner | Shown for the selected scope when an account is reconnecting or logged out (see §3) |
| `ChatRow` | 64 px; avatar 44 px circle with account-color dot (12 px, 2 px ring in `surface`) |
| — line 1 | Name (500, or 700 when unread) · time (`faint`, or `accent` 700 when unread) |
| — line 2 | Preview (`textSecondary`, or `text` when unread) · unread pill (`accent`) |
| — line 3 | `{account} · {network}` in `faint`, 11 px |
| Selected row | `raised` background, `aria-current="true"` |
| Empty state | Icon tile, "No chats yet", body, primary "Add account" |

Sort: pinned first, then by `lastMessageAt` desc. Use list virtualization once a list passes ~200 rows.
Previews of sent messages start with "You: ".

### 2.3 Conversation

| Element | Spec |
|---|---|
| Header | 64 px: avatar 38 px · name (15/600) · status line `{account dot} {account} · {network} · {state}` |
| Header actions | Search (⌘K), Details (⌘I, `aria-pressed`) |
| Message log | `role="log"`, newest at bottom, day separators ("Today", dates) as pills in `raised` |
| Incoming bubble | `raised`, radius 16 16 16 4, max 62% width; group chats show sender name (700, sender color) on the first bubble of a run |
| Outgoing bubble | `accentTint`, radius 16 16 4 16 |
| Meta | Time + status for outgoing: Sending · Sent · Delivered · Read · Not sent (11 px, `textSecondary`) |
| Failed message | Bubble border 1.5 px `danger`; meta "Not sent" in `danger`; under it a `role="alert"` row: icon + "Couldn't send. Check your connection and try again." + **Retry** |
| Composer note | `Sending as {account} · {network}` with account dot (always visible — prevents replying from the wrong account) |
| Composer | Attach (disabled until the engine has `Media`), text field 42 px with 1.5 px `border`, Send (accent, 42×42). Enter sends, Shift+Enter new line |

Capabilities drive the UI: no Attach without `Media`, no reactions without `Reactions`,
no read ticks without `ReadReceipts`. Never check the engine id.

### 2.4 DetailsPanel

Header "Details" + close. Avatar 80 px, name, subtitle (masked phone or "Group · N members"),
"via {account} · {network}" chip. Actions: **Mute · Pin · Archive** (toggle buttons, `aria-pressed`).
Sections: *Shared media* ("Coming soon" until `Media`), *On this computer* (local message count,
"Clear local history" as a `danger` outlined button with confirmation).

### 2.5 SearchPalette (⌘K)

Modal dialog, 640 px wide, 88 px from the top. Search field 56 px, `esc` hint.
Results grouped **Chats** then **Messages**; message results show `chat · account · time` and a snippet
with the match in a `mark` (`accentTint`). Keyboard: ↑↓ move, ↵ open, Esc close.
Footer: "Searches only this computer". Empty: `No results for "{query}"…`.
Search runs locally against SQLite (full-text index — see roadmap).

### 2.6 Add account dialog

Step 1 network grid from `ListEngines()` (available first, others "Coming soon" disabled,
plus "Your engine" link). Step 2 login (QR for WhatsApp) + account name + color + live preview of the
rail item. Same pieces as onboarding steps 3–4; build them once and reuse.

### 2.7 Settings

Left nav: General · Accounts · Notifications · Privacy · About. Sections:

- **General:** language (System default / English / Español — switches instantly), theme (System / Light / Dark).
- **Accounts:** rows with color tile, label, `network · masked id`, state dot + text, Rename, Log out (`danger` outline).
- **Notifications:** see §2.8.
- **Privacy:** the three local-first statements and the data folder path; links to PRIVACY.md and DISCLAIMER.md.
  Below them, an "Encryption" row (on, where the key lives, what it protects against) and a "Recovery phrase" row
  with its status dot ("Saved" / "Not saved yet") and a button that opens the recovery dialog (§2.9).
- **About:** version, license, links to source, Acceptable Use Policy and Security policy.

### 2.8 Notifications

Shown by the OS; Privé only supplies title, body and icon. Settings:

| Setting | Default |
|---|---|
| Show notifications | On |
| Play a sound | On |
| Unread badge (Dock / taskbar) | On |
| What notifications show | **Sender only** (options: Sender and message · Sender only · Nothing private) |
| Per-account switch | On for each account |

Muted chats and muted accounts never notify and are not counted in the badge.

### 2.9 Recovery phrase and unlock

[ADR 0007](adr/0007-encryption-at-rest.md). Never part of onboarding.

- **Banner** (ChatList, below the account banners, `accent` border, lock icon): "Save your recovery phrase",
  one line on why, "Save now" (primary) and "Later". Shown once chats exist and no account is importing.
  "Later" hides it until the next start; confirming the phrase removes it for good.
- **Dialog** (520–560 px, steps 1–3): 1) why the phrase matters, three short rules (paper, no screenshots,
  Privé never asks for it); 2) the 12 words in a numbered 3 × 4 grid in the UI language's list (English when the
  language has none), monospace; 3) three random positions to type back (first four letters are enough, case
  and accents ignored); then "Recovery phrase saved". From Settings after confirming, only step 2 is shown.
- **Unlock screen** (full window, replaces the app when `errors.encryption.locked`): logo, "Unlock your chats",
  why this happens, a monospace textarea for the 12 words (numbers and punctuation ignored), "Unlock", errors
  under the field (`errors.recovery.*`), and a footnote with the data folder path for starting over.

## 3. State matrix

### Account state (from `account:state`)

| State | Rail | Chat list | Conversation header | Composer |
|---|---|---|---|---|
| `connecting` | — | — | "Connecting…" (`textSecondary`) | Enabled; messages queue |
| `connected` | — | — | "Connected" / "N members" | "Sending as …" |
| `reconnecting` | "!" `warning` badge | `warning` banner: "Reconnecting to {account}…" + reassurance | "Reconnecting…" in `warning` | Note in `warning`: messages will send when it reconnects |
| `loggedOut` | "!" `danger` badge | `danger` banner: "{account} was logged out" + **Link again** | "Logged out" in `danger` | Note in `danger`: link again to send; send disabled |
| `error` | "!" `danger` badge | `danger` banner with the translated error code + Retry | — | — |

Banners use `role="status"`; they are scoped to the selected account or shown in "All inboxes"
for every affected account (stacked, max 2, then "+N more").

### Message status (from `message:status`)

`pending` → `sent` → `delivered` → `read`, or `failed`. Failed messages keep their text and offer Retry.

## 4. Keyboard

| Shortcut | Action |
|---|---|
| ⌘K | Search all accounts |
| ⌘0 / ⌘1–9 | All inboxes / account N |
| ⌥↑ / ⌥↓ | Previous / next chat |
| ⌘N | Add account |
| ⌘I | Toggle details |
| ⌘, | Settings |
| Enter / Shift+Enter | Send / new line |
| Esc | Close overlay, then clear search, then blur composer |

Ctrl replaces ⌘ on Windows and Linux.

## 5. Accessibility checklist (per component)

- Real elements: `button`, `a`, `input` + `label`, `fieldset` + `legend`; no clickable `div`s.
- Visible 2 px focus ring (`focus` token, 2 px offset) everywhere.
- Icon-only buttons have translated `aria-label`s.
- Lists: rail and chat list are keyboard-navigable with arrow keys (roving tabindex).
- New incoming messages in the open chat are announced politely; other chats only update counts.
- Never color alone: account = color + initial + label; status = icon + word.
- Respect reduced motion; no animation is required to understand state.

## 6. Translation keys

Namespaces: `rail.*`, `chatList.*`, `conversation.*`, `composer.*`, `details.*`, `search.*`,
`addAccount.*`, `settings.*`, `notifications.*`, `account.state.*`, `message.status.*`, `errors.*`,
`engines.<id>.*`, `onboarding.*`, `common.*`. The English text in the prototypes is the reference copy.

## 7. Data the UI needs (frontend stores)

| Store | Holds | Fed by |
|---|---|---|
| `accounts` | Account list, state per account, unread per account | `ListAccounts()`, `account:state`, `chat:updated` |
| `chats` | Chats per account, sorted; selected chat; filters | `ListChats()`, `chat:updated` |
| `messages` | Messages per chat (paged), send queue, failures | `GetMessages()`, `message:new`, `message:status` |
| `ui` | Scope (all/account), details open, search open, theme, language | Local |

## 8. Planned additions

Not designed yet; the decisions are in ADRs and the spec will be extended before each feature is built.

| Feature | Touches | ADR |
|---|---|---|
| App password (on/off, change), password prompt at start; OS authentication before showing the recovery phrase | Settings → Privacy, unlock screen | [0007](adr/0007-encryption-at-rest.md) |
| Transcribe action and transcript under voice bubbles; "On this computer" Settings section (models, download) | Conversation, Settings | [0008](adr/0008-local-voice-transcription.md) |
| "Voice notes" result group, "Similar meaning" label, search filters, one-time semantic search offer | SearchPalette | [0009](adr/0009-hybrid-local-search.md) |
| Lock / hide / exclude from search; "Locked chats" row; secret code in ⌘K | DetailsPanel, ChatList, AccountRail, Settings → Privacy | [0010](adr/0010-chat-and-account-privacy.md) |
| Third-party notices | Settings → About | [0011](adr/0011-licensing-policy.md) |

## 9. Prototypes

The interactive prototypes were built on a design canvas. Their sources are in
[`design/prototypes/`](../design/prototypes/) as reference (copy, layout values, states and sample data);
they are not app code. Each inbox state is the same `Main` screen with a `scenario` value:
`default`, `empty`, `reconnecting`, `loggedOut`, `sendError`, `search`, `details`.
