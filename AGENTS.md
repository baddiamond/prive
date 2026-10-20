# AGENTS.md

Instructions for AI coding agents working on this repository.

**Read [`CLAUDE.md`](./CLAUDE.md) first — it is the single source of truth for project rules.**

The non-negotiable rules, in short:

1. Everything written to the repository is in **English** (code, comments, docs, commits, logs).
2. **No hardcoded UI strings**: use react-i18next keys and update every file in `frontend/src/locales/`.
3. The Go backend returns **codes, not text**, for anything a user may see.
4. **Local-first and private**: no telemetry, no external servers, no accounts.
5. Every network is an **engine** implementing `core.Engine` / `core.Session` and passing `internal/engines/enginetest`.
6. Unlimited **multi-account**: never assume one account per network; every event carries `accountId`.

Useful docs:

- [`docs/ARCHITECTURE.md`](./docs/ARCHITECTURE.md) — how the pieces fit together
- [`docs/ADDING_AN_ENGINE.md`](./docs/ADDING_AN_ENGINE.md) — step-by-step guide for a new network
- [`docs/I18N.md`](./docs/I18N.md) — translations and adding a language
- [`docs/ROADMAP.md`](./docs/ROADMAP.md) — phases and current status
