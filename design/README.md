# Design sources

Reference material for the UI and the brand. **Not app code.**

| Folder | Contents |
|---|---|
| `prototypes/ui/` | Interactive screen prototypes: unified inbox (and its states), add account, settings, notifications, first run, the engine → model → UI diagram |
| `prototypes/brand/` | Brand guide boards: essence, logo concepts, logo system, color, type, voice, accessibility |

Each `*.dc.html` file is one artboard from a design canvas; `canvas.json` lists them and their layout.
The files are readable HTML: copy, layout values, colors, sample data and state logic (in the
`class Component` script) are all visible. They need the canvas runtime (`support.js`) to render, so
open them as text, or ask the maintainer for the live canvas link.

Specs written from these prototypes:

- [`docs/UI.md`](../docs/UI.md) — screens, components, states, keyboard, accessibility
- [`docs/ONBOARDING.md`](../docs/ONBOARDING.md) — first run
- [`docs/brand/BRAND.md`](../docs/brand/BRAND.md) — brand guide

All sample names, numbers and messages in the prototypes are fictional.
