# 0001 — Wails v2 for the desktop shell

- **Status:** accepted
- **Date:** 2026-10-20

## Context

Privé needs a cross-platform desktop app (macOS first, then Windows and Linux) with a Go backend,
fast startup and low memory use.

## Decision

Use **Wails v2**: a Go process that renders the UI in the operating system's native webview
(WKWebView on macOS, WebView2 on Windows, WebKitGTK on Linux). The frontend calls Go methods through
generated bindings, and Go pushes real-time updates with Wails events.

## Consequences

- One small binary; no bundled Chromium, so memory and disk usage stay low.
- The backend is plain Go and can call libraries like whatsmeow directly.
- Rendering differences between OS webviews must be tested on each platform.
- Wails v3 is still evolving; we can migrate later if it brings clear benefits.

## Alternatives considered

- **Electron** — mature, but ships Chromium and Node: much heavier, and the backend would not be Go.
- **Tauri** — light, but the backend is Rust; calling Go libraries would need a sidecar process.
- **Native UI per OS** — best feel, but triples the UI work for a small open-source team.
