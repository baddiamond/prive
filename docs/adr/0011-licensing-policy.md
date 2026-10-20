# 0011 — Dependency and model licensing policy

- **Status:** accepted
- **Date:** 2026-10-20

## Context

Privé is MIT-licensed and promises to be "Open to everyone". It will ship third-party code and offer
downloadable AI models (ADRs [0008](0008-local-voice-transcription.md), [0009](0009-hybrid-local-search.md)).
Each one must allow redistribution inside an MIT app, and their notices must reach users.

## Decision

### Allow-list

Code linked into Privé and models it offers must use one of:

| License | Notes |
|---|---|
| MIT, ISC, BSD-2-Clause, BSD-3-Clause, Zlib | Keep the copyright notice |
| Apache-2.0 | Keep the `NOTICE` file contents |
| MPL-2.0 | Only as an unmodified library (e.g. whatsmeow, ADR 0005) |
| Public domain / Unlicense / CC0 | — |
| CC-BY-4.0 | **Models and data only**; attribution shown in the app |

Anything else (GPL, AGPL, LGPL for static linking, SSPL, "non-commercial", custom model terms with use
restrictions such as the Gemma terms) needs a new ADR before it is added.

### Current review

| Component | License | Status |
|---|---|---|
| whisper.cpp + Go bindings | MIT | Allowed |
| Whisper model weights (ggml) | MIT | Allowed |
| libopus | BSD-3-Clause | Allowed |
| ncruces/go-sqlite3 | MIT | Allowed |
| wazero | Apache-2.0 | Allowed |
| SQLite | Public domain | Allowed |
| sqlite-vec | MIT / Apache-2.0 | Allowed |
| Model2Vec (reference for our loader) | MIT | Allowed |
| potion-multilingual-128M | MIT | Allowed |
| multilingual-e5-small | MIT | Allowed |
| golang.org/x/crypto | BSD-3-Clause | Allowed |
| zalando/go-keyring | MIT | Allowed |
| sherpa-onnx / ONNX Runtime | Apache-2.0 / MIT | Allowed (only if Parakeet is adopted) |
| Parakeet TDT 0.6B v3 | CC-BY-4.0 | Allowed with visible attribution |
| whatsmeow | MPL-2.0 | Allowed (unmodified) |
| Wails, React, Tailwind, Zustand, react-i18next | MIT | Allowed |
| Lucide icons | ISC | Allowed |
| EmbeddingGemma | Gemma terms | **Rejected** (use restrictions) |
| `autowhisper` | GPL-3.0 | **Rejected** |
| BIP39 word lists | Unclear | **Not used** — Privé ships its own MIT word lists (ADR 0007) |

### Enforcement

- `THIRD_PARTY_NOTICES.md` at the repo root, regenerated on release, and shown in Settings → About.
- CI fails when a dependency outside the allow-list appears: `go-licenses` for Go modules and
  `license-checker` (or equivalent) for the frontend.
- Each downloadable model has a manifest entry (`name`, `url`, `sha256`, `size`, `license`, `attribution`)
  reviewed in the PR that adds it; the download screen shows license and attribution.
- New dependencies still need the justification required by `CONTRIBUTING.md`.

## Consequences

- Contributors know up front what can be added; reviews are mechanical.
- Some strong models are out of reach while their terms stay restrictive.
- One more CI job and a notices file to maintain.

## Alternatives considered

- **Case-by-case review without a list** — slow and inconsistent.
- **Allow LGPL with dynamic linking** — complicates packaging on every OS; revisit only if a key library needs it.
