# 0002 — Pure-Go SQLite

- **Status:** accepted
- **Date:** 2026-10-20

## Context

Privé stores accounts, chats, messages and settings locally. SQLite is the natural fit.
The most common Go driver (`mattn/go-sqlite3`) requires CGO, which complicates cross-compilation
for macOS universal, Windows and Linux builds.

## Decision

Use **`modernc.org/sqlite`**, a CGO-free translation of SQLite to Go, with WAL mode enabled.
Schema migrations are plain `.sql` files embedded in the binary and applied at startup.

## Consequences

- Simple builds and cross-compilation; contributors need no C toolchain for the backend.
- Slightly slower than the C version in heavy benchmarks — irrelevant for a messaging client's workload.
- whatsmeow's `sqlstore` works with this driver.

## Alternatives considered

- **mattn/go-sqlite3** — fastest, but CGO.
- **Embedded key-value stores (bbolt, Badger)** — no SQL queries for inbox sorting, pagination and search.
