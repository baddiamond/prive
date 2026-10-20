# 0008 — On-device voice note transcription

- **Status:** accepted (planned — see [roadmap](../ROADMAP.md#future-features))
- **Date:** 2026-10-20

## Context

Voice notes are a large part of messaging, and they cannot be skimmed or searched. Cloud transcription
would break the first promise ("Local by design"). Users must not have to install anything else
(no Python, no separate server, no ffmpeg).

## Decision

### Engine

- **[whisper.cpp](https://github.com/ggml-org/whisper.cpp)** (MIT) through its official Go bindings
  (`bindings/go`), compiled into the Privé binary. CGO is already required by Wails on macOS and Linux,
  so this adds no new build constraint for those targets.
- Acceleration: **Metal** on macOS, CPU (with SIMD) on Windows and Linux; GPU backends are evaluated later.
- Audio decoding: **libopus** (BSD-3-Clause), statically linked, to turn OGG/Opus voice notes into 16 kHz mono PCM.
  No ffmpeg.
- **Parakeet TDT 0.6B v3** (CC-BY-4.0) through sherpa-onnx is a candidate fast engine for CPU-only
  computers (25 European languages). It is only adopted if a benchmark shows a clear win, and then with
  visible attribution.

### Where it lives

Transcription is not a messaging engine (`core.Engine` is one network). It lives in
`internal/local/transcribe` behind a small interface:

```go
// Transcriber turns a voice note into text on this computer.
type Transcriber interface {
    Transcribe(ctx context.Context, audio io.Reader, lang string) (Transcript, error)
}
```

- A single background worker with low priority; one job at a time, using half the cores.
- The model is loaded on first use and unloaded after about two idle minutes.
- Results go to a `message_transcripts` table and into the search index ([ADR 0009](0009-hybrid-local-search.md)).
- Chats excluded from search or hidden ([ADR 0010](0010-chat-and-account-privacy.md)) are never auto-transcribed.
- The UI shows a **Transcribe** action on any voice message, whatever the network; it never checks the engine id.

### Models and defaults

| Model | Download | Use |
|---|---|---|
| `base` | ~140 MB | Basic computers |
| `small` | ~460 MB | Recommended default for standard computers |
| `large-v3-turbo` (quantized) | ~550 MB | High-end computers and Apple Silicon |

- **Off by default.** Turning it on downloads one model, chosen by the user; the app marks the one
  recommended for this computer ([`docs/REQUIREMENTS.md`](../REQUIREMENTS.md)).
- Once a model exists, the manual **Transcribe** action is available on every voice note.
- **Automatic transcription** of new voice notes is a separate switch, also off. When on, it only runs
  on AC power or while the computer is idle.

### Network exception

Downloading a model is the only traffic that does not go to a messaging network. Rules:

- Only after an explicit user action, never in the background.
- From a fixed URL we control (the project's GitHub Releases), with a pinned SHA-256 verified before use.
- Models can also be imported from a file, for offline computers.
- The download screen shows name, size, license and attribution.
- `PRIVACY.md` documents this exception when the feature ships.

## Consequences

- A clear differentiator: searchable, readable voice notes, fully offline after one download.
- Binary grows by a few MB (whisper.cpp + libopus); models stay out of the installer.
- Windows builds need a C toolchain for this package (CI already builds Wails there).
- New translation namespace: `transcription.*`; new Settings section "On this computer".

## Alternatives considered

- **Cloud speech APIs** — break the privacy promise.
- **OS speech APIs** (e.g. macOS on-device speech) — not cross-platform; may be added later as an optional
  zero-download path on macOS.
- **Bundling a model in the installer** — +140–550 MB for everyone, most of whom may never use it.
- **Third-party Go wrappers under GPL** (e.g. `autowhisper`) — incompatible with MIT ([ADR 0011](0011-licensing-policy.md)).
