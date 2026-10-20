// Package bootstrap opens Privé's local data the same way for every program: the desktop
// app and the developer tool in cmd/prive-dev.
package bootstrap

import (
	"context"
	"database/sql"
	"log/slog"
	"path/filepath"

	"github.com/baddiamond/prive/internal/core"
	"github.com/baddiamond/prive/internal/keys"
	"github.com/baddiamond/prive/internal/platform"
	"github.com/baddiamond/prive/internal/store"
)

// Local is the opened local data: folder, encrypted database and data key.
type Local struct {
	DataDir string
	Store   *store.Store
	Keys    *keys.Keys
}

// Open resolves the data folder, unlocks the data key from the keychain (creating it on
// first run) and opens the encrypted database (ADR 0007). The data folder is returned even
// on error, when it is known.
func Open(ctx context.Context, kc keys.Keychain) (*Local, string, error) {
	dataDir, err := platform.EnsureDataDir()
	if err != nil {
		return nil, "", err
	}
	l, err := OpenAt(ctx, dataDir, kc)
	return l, dataDir, err
}

// OpenAt is Open for an explicit data folder.
func OpenAt(ctx context.Context, dataDir string, kc keys.Keychain) (*Local, error) {
	k, _, err := keys.Unlock(dataDir, kc)
	if err != nil {
		return nil, err
	}
	return open(ctx, dataDir, k)
}

// Recover unlocks the data folder with the recovery phrase (when Open failed with
// keys.ErrLocked) and opens the database. The next Open needs no phrase.
func Recover(ctx context.Context, dataDir, phrase string, kc keys.Keychain) (*Local, error) {
	k, err := keys.Recover(dataDir, phrase, kc)
	if err != nil {
		return nil, err
	}
	return open(ctx, dataDir, k)
}

func open(ctx context.Context, dataDir string, k *keys.Keys) (*Local, error) {
	st, err := store.Open(ctx, filepath.Join(dataDir, store.FileName), store.Options{Key: k.Derive(store.FileName)})
	if err != nil {
		return nil, err
	}
	return &Local{DataDir: dataDir, Store: st, Keys: k}, nil
}

// ManagerConfig returns the account manager's configuration for this local data:
// engine databases are encrypted with keys derived from the data key.
func (l *Local) ManagerConfig(engines core.EngineSource, logger *slog.Logger, emit func(core.Event)) core.ManagerConfig {
	return core.ManagerConfig{
		Engines: engines,
		Repo:    l.Store,
		DataDir: l.DataDir,
		Logger:  logger,
		Emit:    emit,
		OpenDB: func(ctx context.Context, path, label string) (*sql.DB, error) {
			return store.OpenDB(ctx, path, store.Options{Key: l.Keys.Derive(label)})
		},
	}
}

// Close closes the database.
func (l *Local) Close() error {
	return l.Store.Close()
}
