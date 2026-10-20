// Command prive is the Privé desktop app: a Wails v2 shell around the Go backend.
package main

import (
	"context"
	"embed"
	"log/slog"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"github.com/baddiamond/prive/internal/bootstrap"
	"github.com/baddiamond/prive/internal/core"
	"github.com/baddiamond/prive/internal/engines"
	_ "github.com/baddiamond/prive/internal/engines/all" // registers bundled engines
	"github.com/baddiamond/prive/internal/keys"
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

	local, dataDir, err := bootstrap.Open(context.Background(), keys.OSKeychain())
	var app *App
	if err != nil {
		logger.Error("open local data", "err", err)
		app = NewFailedApp(err, dataDir, logger)
	} else {
		defer func() { _ = local.Close() }()
		app = NewApp(local, engines.Default(), logger)
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
