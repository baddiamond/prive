// Package store persists Privé's local data in a single SQLite database
// using the pure-Go modernc.org/sqlite driver (see ADR 0002).
//
// The schema is created and upgraded by the embedded migrations in migrations/.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	_ "modernc.org/sqlite" // registers the "sqlite" database/sql driver
)

// FileName is the database file name inside the data directory.
const FileName = "prive.db"

// Store is a handle to the local database. It is safe for concurrent use.
type Store struct {
	db *sql.DB
}

// Open opens (or creates) the database at path and applies pending migrations.
func Open(ctx context.Context, path string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("open database: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the database. The Store must not be used afterwards.
func (s *Store) Close() error {
	return s.db.Close()
}

// inTx runs fn in a transaction, committing if it returns nil.
func (s *Store) inTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// dsn builds the connection string. Pragmas are applied to every pooled connection:
// WAL for concurrent readers, a busy timeout instead of immediate "database is locked"
// errors, and enforced foreign keys. Transactions start with BEGIN IMMEDIATE so that
// read-then-write transactions from different accounts queue on the busy timeout
// instead of failing with SQLITE_BUSY when they upgrade to a write lock.
func dsn(path string) string {
	q := url.Values{}
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "foreign_keys(ON)")
	q.Add("_pragma", "synchronous(NORMAL)")
	q.Add("_txlock", "immediate")
	return "file:" + path + "?" + q.Encode()
}
