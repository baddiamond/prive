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

	"github.com/baddiamond/prive/internal/platform"
	"github.com/baddiamond/prive/internal/store"
)

// assets holds the production frontend build (frontend/dist).
//
//go:embed all:frontend/dist
var assets embed.FS

// appID identifies Privé to the OS, e.g. for the single-instance lock.
const appID = "io.github.baddiamond.prive"

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("privé stopped with an error", "err", err)
		os.Exit(1)
	}
}

// run opens local storage and blocks until the window is closed.
func run(logger *slog.Logger) error {
	dataDir, err := platform.EnsureDataDir()
	if err != nil {
		return err
	}
	st, err := store.Open(context.Background(), filepath.Join(dataDir, store.FileName))
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	app := NewApp(st, logger)

	return wails.Run(&options.App{
		Title:     "Privé",
		Width:     1100,
		Height:    720,
		MinWidth:  720,
		MinHeight: 480,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind:       []any{app},
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
