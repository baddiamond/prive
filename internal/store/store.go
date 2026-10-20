// Package store persists Privé's local data in a single SQLite database through the
// pure-Go ncruces/go-sqlite3 driver (ADR 0006), encrypted with the adiantum VFS when a key
// is given (ADR 0007).
//
// The schema is created and upgraded by the embedded migrations in migrations/.
package store

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	"github.com/ncruces/go-sqlite3/ext/fts5"
	_ "github.com/ncruces/go-sqlite3/vfs/adiantum" // registers the "adiantum" encrypting VFS
)

// FileName is the database file name inside the data directory.
const FileName = "prive.db"

// KeySize is the length in bytes of a database key.
const KeySize = 32

// Store is a handle to the local database. It is safe for concurrent use.
type Store struct {
	db *sql.DB
}

// Options configures how a database is opened.
type Options struct {
	// Key encrypts the database file with the adiantum VFS. Nil keeps the file in clear,
	// which only tests use.
	Key []byte
}

// Open opens (or creates) the database at path and applies pending migrations.
// A database still in clear from a development build is encrypted in place first.
func Open(ctx context.Context, path string, opts Options) (*Store, error) {
	if opts.Key != nil {
		if err := encryptLegacy(ctx, path, opts.Key); err != nil {
			return nil, err
		}
	}
	db, err := OpenDB(ctx, path, opts)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// OpenDB opens any SQLite file Privé owns with the app's driver, pragmas and encryption,
// without Privé's schema. Engines get their private databases this way
// (core.SessionDeps.OpenDB), so every file can be encrypted the same way.
func OpenDB(ctx context.Context, path string, opts Options) (*sql.DB, error) {
	dsn, err := buildDSN(path, opts.Key)
	if err != nil {
		return nil, err
	}
	db, err := driver.Open(dsn, initConn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("open database: %w", err)
	}
	return db, nil
}

// Close releases the database. The Store must not be used afterwards.
func (s *Store) Close() error {
	return s.db.Close()
}

// initConn runs on every new connection: it enables full-text search (ADR 0009).
func initConn(c *sqlite3.Conn) error {
	return fts5.Register(c)
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

// buildDSN returns the connection string. Pragmas apply to every pooled connection:
// WAL for concurrent readers, a busy timeout instead of immediate "database is locked"
// errors, enforced foreign keys, and temporary data kept in memory so it never reaches
// the disk unencrypted. Transactions start with BEGIN IMMEDIATE so read-then-write
// transactions from different accounts queue on the busy timeout instead of failing
// with SQLITE_BUSY when they upgrade to a write lock.
func buildDSN(path string, key []byte) (string, error) {
	q := url.Values{}
	if key != nil {
		if len(key) != KeySize {
			return "", fmt.Errorf("open database: key must be %d bytes", KeySize)
		}
		q.Set("vfs", "adiantum")
		q.Set("hexkey", hex.EncodeToString(key))
	}
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "foreign_keys(ON)")
	q.Add("_pragma", "synchronous(NORMAL)")
	q.Add("_pragma", "temp_store(MEMORY)")
	q.Set("_txlock", "immediate")
	return "file:" + (&url.URL{Path: path}).EscapedPath() + "?" + q.Encode(), nil
}

// plaintextHeader starts every unencrypted SQLite database file.
var plaintextHeader = []byte("SQLite format 3\x00")

// encryptLegacy encrypts, in place, a database created in clear by a development build
// before encryption existed. It does nothing if the file is missing or already encrypted.
func encryptLegacy(ctx context.Context, path string, key []byte) error {
	clear, err := isPlaintext(path)
	if err != nil || !clear {
		return err
	}
	src, err := OpenDB(ctx, path, Options{})
	if err != nil {
		return err
	}
	tmp := path + ".encrypting"
	_ = os.Remove(tmp)
	target, err := buildDSN(tmp, key)
	if err == nil {
		_, err = src.ExecContext(ctx, "VACUUM INTO ?", target)
	}
	if cerr := src.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("encrypt existing database: %w", err)
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		_ = os.Remove(path + suffix)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("encrypt existing database: %w", err)
	}
	return nil
}

func isPlaintext(path string) (bool, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer f.Close()
	header := make([]byte, len(plaintextHeader))
	if _, err := io.ReadFull(f, header); err != nil {
		return false, nil // empty or tiny file: nothing to migrate
	}
	return bytes.Equal(header, plaintextHeader), nil
}
