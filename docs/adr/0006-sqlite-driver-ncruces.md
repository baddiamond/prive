# 0006 — SQLite driver: ncruces/go-sqlite3

- **Status:** accepted (supersedes [0002](0002-pure-go-sqlite.md))
- **Date:** 2026-10-20

## Context

[ADR 0002](0002-pure-go-sqlite.md) chose `modernc.org/sqlite` to keep the backend free of CGO.
Three planned features need more from the database layer than that driver offers:

- **Encryption at rest** ([ADR 0007](0007-encryption-at-rest.md)) — `modernc.org/sqlite` has no page-level encryption.
- **Full-text search** ([ADR 0009](0009-hybrid-local-search.md)) — needs FTS5.
- **Vector search** ([ADR 0009](0009-hybrid-local-search.md)) — needs `sqlite-vec` or an equivalent.

The store already runs on `modernc.org/sqlite` (phases 0–1), but no release has shipped. Switching drivers
before the first release costs only code changes; switching after users have data means a migration of
every installation.

## Decision

Use **[`github.com/ncruces/go-sqlite3`](https://github.com/ncruces/go-sqlite3)** through its `database/sql` driver.
It runs the real SQLite C code compiled to WebAssembly on the [wazero](https://wazero.io) runtime, so it stays
CGO-free. We use:

- its **`adiantum` VFS** for encryption at rest (details in ADR 0007);
- the **`sqlite-vec`** build from `github.com/asg017/sqlite-vec-go-bindings/ncruces` for vector search;
- **FTS5** for full-text search.

WAL mode stays on. Migrations remain plain `.sql` files embedded in the binary.
The same driver opens every SQLite file Privé owns, including each engine's private store
(e.g. whatsmeow's `sqlstore`), so all of them can be encrypted the same way.

## Consequences

- One driver gives encryption, full-text and vector search; no CGO for storage.
- WebAssembly execution is somewhat slower than native SQLite. For a messaging client's workload
  (small writes, indexed reads) this is not noticeable; we verify it in the phase 6 profiling.
- First open compiles the Wasm module (a one-time cost per process, tens of milliseconds); measured at startup.
- **Do the switch before the first release** (tracked in the roadmap, phase 1).
- **To verify during the switch:** FTS5 availability in the exact build we embed together with
  `sqlite-vec`, and that whatsmeow's `sqlstore` works unchanged on this driver (it uses `database/sql`).

## Alternatives considered

- **Keep `modernc.org/sqlite`** — no encryption; we would have to switch later with a data migration.
- **SQLCipher / SQLite3 Multiple Ciphers via `mattn/go-sqlite3`** — authenticated page encryption
  (HMAC per page), but requires CGO and a C toolchain for every target.
- **Encrypt only selected columns** — leaks metadata (who, when, how much), breaks indexes and FTS.
