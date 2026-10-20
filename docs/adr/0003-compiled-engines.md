# 0003 — Engines compiled into the binary

- **Status:** accepted
- **Date:** 2026-10-20

## Context

Every network is an "engine". We want the community to add engines easily, and we want the app to stay
fast, safe and simple to distribute.

## Decision

Engines are **Go packages compiled into the binary** under `internal/engines/<id>/`. They self-register
with `engines.Register` and are enabled through blank imports in `internal/engines/all`.
All engines implement the same `core.Engine` / `core.Session` contract and must pass the shared
conformance suite in `internal/engines/enginetest`.

## Consequences

- No runtime loading, no version skew, no extra processes: fast and easy to reason about.
- Adding an engine requires a pull request and a new release — acceptable, and it keeps code reviewed.
- Engines must be written in Go.

## Alternatives considered

- **Go `plugin` package** — not supported on Windows and fragile across Go versions.
- **Out-of-process engines (JSON over stdio)** — allows any language; kept on the roadmap as a future option
  that can coexist with compiled engines through an adapter engine.
