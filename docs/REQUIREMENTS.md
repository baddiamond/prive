# System requirements and defaults

Privé should run on almost any computer from the last ten years. The core app sets the minimum;
heavier local features are optional and recommended per computer.

> The numbers below are **estimates** to be replaced with measurements from a benchmark spike
> (one Apple Silicon Mac and one older Windows PC) before the features ship.

## Device tiers

| | **Basic** (minimum) | **Standard** (recommended) | **High** |
|---|---|---|---|
| Example | Intel laptop from 2015–2017, Celeron / i3 | Most PCs from 2019 on | Apple Silicon Mac, PC with a GPU |
| CPU | 2 cores, 64-bit (x64 or arm64) | 4 cores | 8 cores or Apple Silicon |
| RAM | 4 GB | 8 GB | 16 GB |
| Free disk | 500 MB + your chats (SSD recommended) | 2 GB | 3 GB |
| OS | macOS 11+, Windows 10+ (WebView2), Linux with WebKitGTK | same | same |
| Works | App, encryption, full-text and semantic search | Everything; transcription with `small` | Everything; `large-v3-turbo` and automatic transcription |

The core app (Wails + the system web view) should use about 150–250 MB of RAM; that sets the minimum.

The backend detects RAM, cores and architecture with OS calls (no heavy dependency) and returns a code
(`device.tier.basic`, `device.tier.standard`, `device.tier.high`). The UI uses it to mark the
"Recommended for this computer" option. Nothing is blocked: the user can always choose.

## Defaults per feature

| Feature | Default | Estimated cost | ADR |
|---|---|---|---|
| Encryption at rest | **On** | Near zero (Adiantum is fast without AES hardware) | [0007](adr/0007-encryption-at-rest.md) |
| Full-text search (FTS5) | **On, always** | Index ≈ 30–50 % of text (1 M messages ≈ 25–50 MB); queries in milliseconds | [0009](adr/0009-hybrid-local-search.md) |
| Semantic search | **Off**, offered once in ⌘K | ~130 MB download; int8 vectors ≈ 25 MB per 100 k conversation windows; microseconds per embedding; memory-mapped model | [0009](adr/0009-hybrid-local-search.md) |
| Voice transcription | **Off** | 140–550 MB model; heavy CPU/GPU while running | [0008](adr/0008-local-voice-transcription.md) |
| Automatic transcription | **Off** (separate switch) | Runs only on AC power or when idle | [0008](adr/0008-local-voice-transcription.md) |

## Transcription estimates

Time to transcribe a 1-minute voice note with whisper.cpp:

| Model | RAM | Older PC, CPU only | Modern PC, 4–8 cores | Apple Silicon (Metal) |
|---|---|---|---|---|
| `base` (~140 MB) | ~400 MB | ~15–30 s | ~5–10 s | ~2 s |
| `small` (~460 MB) | ~850 MB | ~1 min or more | ~15–30 s | ~4–6 s |
| `large-v3-turbo` q5 (~550 MB) | ~1–1.5 GB | Not recommended | ~40–60 s | ~6–10 s |

## Resource rules for background work

- One heavy job at a time (transcription, initial indexing), low priority, half the cores at most.
- Automatic jobs pause on battery and resume on AC power or when the computer is idle.
- Models are loaded on first use and unloaded after about two idle minutes.
- Initial indexing can be paused and resumes after a restart.
