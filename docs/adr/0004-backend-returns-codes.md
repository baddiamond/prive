# 0004 — Backend returns codes, not text

- **Status:** accepted
- **Date:** 2026-10-20

## Context

The app is multilingual. If the Go backend produced user-facing strings, translations would be split
between Go and the frontend, and the backend would need to know the UI language.

## Decision

Anything the backend sends that a user may read (errors, connection states, sync progress) is a **code**
that is also a translation key, plus optional parameters: `{ "code": "errors.whatsapp.sendFailed", "params": {…} }`.
All translations live in `frontend/src/locales/*.json`.

## Consequences

- One place for translations; adding a language never touches Go code.
- Backend codes and locale keys must stay in sync (checked in tests and review).
- Log messages remain plain English and are never shown to users.

## Alternatives considered

- **Translate in Go (go-i18n)** — two translation systems to maintain.
