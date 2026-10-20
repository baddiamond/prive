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
It runs the real SQLite C code, translated to Go (`ncruces/go-sqlite3-wasm`, produced with `wasm2go`), so it stays
CGO-free. We use:

- its **`adiantum` VFS** for encryption at rest (details in ADR 0007);
- **FTS5** for full-text search, registered on every connection (`ext/fts5`);
- **`vec1`**, SQLite's own vector extension, for vector search (`ext/vec1`), registered when semantic search
  ships.

WAL mode stays on. Migrations remain plain `.sql` files embedded in the binary.
The same driver opens every SQLite file Privé owns, including each engine's private store
(e.g. whatsmeow's `sqlstore`), so all of them can be encrypted the same way.

## Consequences

- One driver gives encryption, full-text and vector search; no CGO for storage.
- Translated code is somewhat slower than native SQLite. For a messaging client's workload
  (small writes, indexed reads) this is not noticeable; we verify it in the phase 6 profiling.
- Extensions are not built in: FTS5 (and later `vec1`) must be registered on each connection. `store.OpenDB`
  does this for every database Privé opens.
- **Switch done in phase 1, before any release.**

### Verification (2026-10-20, ncruces v0.35.6)

A spike opened one database through the `adiantum` VFS in WAL mode and checked: FTS5 with
`unicode61 remove_diacritics 2` ("cancion" finds "canción"), `vec1` loaded on the same connection, whatsmeow's
`sqlstore` created and upgraded its schema on that database unchanged, the file on disk has no plaintext, and a
wrong key is rejected. The `asg017/sqlite-vec-go-bindings` package mentioned in the first draft of this ADR is not
usable: its last release (Nov 2024) targets ncruces v0.17 and the older Wasm runtime.

## Alternatives considered

- **Keep `modernc.org/sqlite`** — no encryption; we would have to switch later with a data migration.
- **SQLCipher / SQLite3 Multiple Ciphers via `mattn/go-sqlite3`** — authenticated page encryption
  (HMAC per page), but requires CGO and a C toolchain for every target.
- **Encrypt only selected columns** — leaks metadata (who, when, how much), breaks indexes and FTS.
