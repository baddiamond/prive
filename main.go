// Command prive is the Privé desktop app: a Wails v2 shell around the Go backend.
package main

import (
	"context"
	"embed"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"github.com/baddiamond/prive/internal/core"
	"github.com/baddiamond/prive/internal/engines"
	_ "github.com/baddiamond/prive/internal/engines/all" // registers bundled engines
	"github.com/baddiamond/prive/internal/keys"
	"github.com/baddiamond/prive/internal/platform"
	"github.com/baddiamond/prive/internal/store"
)

// assets holds the production frontend build (frontend/dist).
//
//go:embed all:frontend/dist
var assets embed.FS

// version is the app version, set at build time with
// -ldflags "-X main.version=1.2.3".
var version = "0.1.0-dev"

// appID identifies Privé to the OS, e.g. for the single-instance lock.
const appID = "io.github.baddiamond.prive"

// openStorage resolves the data folder, unlocks the data key from the system keychain
// (creating it on first run) and opens the encrypted database (ADR 0007).
func openStorage(ctx context.Context) (string, *store.Store, *keys.Keys, error) {
	dataDir, err := platform.EnsureDataDir()
	if err != nil {
		return "", nil, nil, err
	}
	k, _, err := keys.Unlock(dataDir, keys.OSKeychain())
	if err != nil {
		return dataDir, nil, nil, err
	}
	st, err := store.Open(ctx, filepath.Join(dataDir, store.FileName), store.Options{Key: k.Derive(store.FileName)})
	if err != nil {
		return dataDir, nil, nil, err
	}
	return dataDir, st, k, nil
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("privé stopped with an error", "err", err)
		os.Exit(1)
	}
}

// run opens local storage and blocks until the window is closed. If storage cannot be
// opened, the window still opens and explains why (see App.failure).
func run(logger *slog.Logger) error {
	if generatingBindings {
		// Only the method signatures matter; never open the keychain or the database.
		return wails.Run(&options.App{Bind: []any{NewFailedApp(core.ErrShuttingDown, "", logger)}})
	}

	dataDir, st, k, err := openStorage(context.Background())
	var app *App
	if err != nil {
		logger.Error("open local data", "err", err)
		app = NewFailedApp(err, dataDir, logger)
	} else {
		defer func() { _ = st.Close() }()
		app = NewApp(st, engines.Default(), dataDir, k, logger)
	}

	return wails.Run(&options.App{
		Title:     "Privé",
		Width:     1180,
		Height:    760,
		MinWidth:  960, // docs/UI.md §1
		MinHeight: 640,
		// The brand's default (dark) background, so the window does not flash white
		// before the UI paints. Keep in sync with --prive-bg in assets/brand/tokens.
		BackgroundColour: &options.RGBA{R: 0x0F, G: 0x10, B: 0x13, A: 255},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind:       []any{app},
		// Bound methods reject with {code, params}, never with text.
		ErrorFormatter: formatError(logger),
		// Two instances would fight over the same database and network sessions.
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               appID,
			OnSecondInstanceLaunch: app.onSecondInstance,
		},
		Mac: &mac.Options{
			// Appearance is left unset so the window follows the system light/dark mode.
			About: &mac.AboutInfo{Title: "Privé"},
		},
		Windows: &windows.Options{
			Theme: windows.SystemDefault,
		},
		Linux: &linux.Options{
			ProgramName: "prive",
		},
	})
}
