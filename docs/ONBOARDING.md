# First run (onboarding)

> Status: **design approved for implementation** (visual prototype, v0.1). Developer hand-off spec.
> Visual rules come from [`docs/brand/BRAND.md`](brand/BRAND.md); all colors from the brand tokens.

The first time Privé opens, the user goes from download to a working inbox in **under a minute**,
with **one decision per screen** and nothing to sign up for.

## Principles

1. **One primary action per screen.** A single accent button; everything else is secondary.
2. **Five short steps, always visible.** A stepper shows where you are and how much is left.
3. **Ask only what we need.** Language and theme are pre-filled from the OS; account name and color
   have defaults.
4. **Honest before useful.** The privacy and responsible-use step comes before any account is connected.
5. **Skippable.** The user can finish without connecting anything and add accounts later.
6. **Localized and accessible from the first pixel.** The language choice applies instantly.

## What we learned from similar apps

| App | What works | What we avoid |
|---|---|---|
| Beeper | Unified inbox, network grid, clear "add account" flow | Requires creating a Beeper account; data goes through their servers |
| Ferdium / Franz / Rambox | Large grid of services, search, multi-account | Heavy web views per service; Franz requires an account; long setup |
| WhatsApp / Signal Desktop | Immediate QR with three numbered phone steps | Single account per app |
| Telegram Desktop | Very short welcome, language switch on the first screen | — |
| Element | Explains privacy choices in plain words | Many technical decisions up front |

Privé combines the network grid (Beeper/Ferdium), the instant QR with numbered steps (WhatsApp/Signal),
and a language switch on the first screen (Telegram) — **without any account or server**.

## Flow

```
1 Welcome ──▶ 2 Before you start ──▶ 3 Connect ──▶ 4 Link ──▶ 5 Ready ──▶ Inbox
                                          │                        ▲
                                          └──── Skip for now ──────┘
5 Ready ── "Add another account" ──▶ 3 Connect
```

| # | Screen | Primary action | Enabled when | Other actions |
|---|---|---|---|---|
| 1 | **Welcome** — mark, title, one-line promise, language (EN/ES), appearance (System/Light/Dark) | Get started | always | — |
| 2 | **Before you start** — 3 facts: stays on this computer · unofficial connections · for your own conversations; agreement checkbox; links to Disclaimer, Acceptable use, Privacy | Continue | checkbox checked | Back |
| 3 | **Connect your first account** — grid of networks from the engine registry: available ones selectable, others "Coming soon" (disabled); "Suggest or build an engine" link | Continue | a network is selected | Back · Skip for now |
| 4 | **Link {network}** — live QR (or the engine's login method), 3 numbered phone steps, account name, account color, live status line | Continue | login succeeded | Back |
| 5 | **Ready** — confirmation, account card with import progress, 3 shortcut tips | Open my inbox | always | Add another account |

If the user skipped step 3, step 5 shows "Privé is ready" and explains how to add an account later.

## Persisted state

Stored in the SQLite `settings` table (never sent anywhere):

| Key | Value | Set at |
|---|---|---|
| `language` | `en`, `es`, … or `system` | Step 1 (on change) |
| `theme` | `system`, `light`, `dark` | Step 1 (on change) |
| `legal.acceptedVersion` | `1` (matches the version in DISCLAIMER.md / ACCEPTABLE_USE.md) | Step 2 |
| `legal.acceptedAt` | RFC 3339 timestamp | Step 2 |
| `onboarding.completedAt` | RFC 3339 timestamp | Step 5 |

On startup: if `onboarding.completedAt` is empty → show onboarding. If `legal.acceptedVersion` is lower
than the current version → show only step 2 again before the inbox (when the policy changes).
Closing the window mid-flow resumes at the last completed step.

## Backend contract

| Need | Call / event |
|---|---|
| Detected OS language | `App.SystemLocale() string` |
| Networks for the grid | `App.ListEngines() []Manifest` — `available` engines first; planned ones come from a static list in the frontend |
| Start linking | `App.AddAccount(engineId, displayName, color) AccountID`, then `App.Login(accountId, method)` |
| QR updates | Wails event `login:qr {accountId, code, expiresAt}` |
| Success / failure | `login:success {accountId, remoteId}` · `login:failed {accountId, code}` |
| Import progress | `sync:progress {accountId, done, total}` |
| Cancel (Back from step 4) | `App.CancelLogin(accountId)` — removes the half-created account |

All errors arrive as translation codes (`errors.whatsapp.qrExpired`, …) per [ADR 0004](adr/0004-backend-returns-codes.md).

## Translation keys

All copy lives under `onboarding.*` in `frontend/src/locales/{en,es}.json`. The English and Spanish
copy in the prototype is the reference text:

```
onboarding.progress
onboarding.steps.{welcome,privacy,connect,link,ready}
onboarding.welcome.{title,body,language,detected,appearance,getStarted,footer}
onboarding.privacy.{title,body,agree}
onboarding.privacy.facts.{local,unofficial,personal}.{title,body}
onboarding.connect.{title,body,available,comingSoon,selected,suggest,suggestLink,skip}
onboarding.link.{title,body,step1,step2,step3,qrAlt,waiting,linked,accountName,accountColor}
onboarding.ready.{title,body,skippedTitle,skippedBody,importing,openInbox,addAnother}
onboarding.ready.tips.{search,jump,add}
common.{back,continue}
legal.{disclaimer,acceptableUse,privacy}
engines.<id>.{name,loginSteps.*}
```

Network-specific steps ("Open WhatsApp → Settings → Linked devices") live under `engines.<id>.*`
so each engine brings its own instructions.

## Layout

- Full window, background `bg`; content column `max-width: 760px`, centered, 16 px side gutter minimum.
- Stepper at the top: numbered circles (✓ when done), labels hidden below 640 px window width.
- Footer navigation on steps 2–4: Back (outlined) on the left; Skip (text) and Continue (accent) on the right.
- Welcome and Ready are centered compositions; steps 2–4 are left-aligned.
- Network grid: `repeat(auto-fill, minmax(170px, 1fr))`, 10 px gap. Coming-soon cards: transparent
  background, `divider` border, secondary text, `disabled`.

## Accessibility

- Stepper is an ordered list in a `nav` with `aria-current="step"` on the active item.
- Each step has one `h1`; focus moves to it when the step changes.
- Choice buttons (language, theme, color, network) use `aria-pressed`; color swatches have names.
- The agreement is a real checkbox with a label. Continue is a real disabled button, not a styled div.
- The QR has a text alternative and the status line is a `role="status"` live region.
- Import progress uses `role="progressbar"` with `aria-valuenow`.
- Everything works with the keyboard: Enter activates the primary action when it is enabled; Esc does nothing
  destructive during onboarding.
- All brand contrast rules apply (≥ 4.5 : 1 text, ≥ 3 : 1 control borders and focus ring).

## Out of scope for v1

- Importing from Beeper or other apps.
- Phone-number login, passwords or OAuth (arrive with the engines that need them).
- "Notify me" for coming-soon networks — that would need a server. The link points to GitHub instead.
