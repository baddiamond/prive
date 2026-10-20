package main

import (
	"context"
	"errors"
	"log/slog"
	"regexp"

	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/baddiamond/prive/internal/platform"
	"github.com/baddiamond/prive/internal/store"
)

// Error codes returned to the frontend. They are translation keys, never user-facing text
// (see ADR 0004). Phase 1 replaces these with core.CodedError values.
var (
	errInvalidSetting = errors.New("errors.settings.invalidValue")
	errSettingsSave   = errors.New("errors.settings.saveFailed")
	errSettingsLoad   = errors.New("errors.settings.loadFailed")
)

// Theme values accepted by SetTheme.
const (
	ThemeSystem = "system"
	ThemeLight  = "light"
	ThemeDark   = "dark"
)

// languageTag loosely matches a BCP 47 tag such as "en", "es-419" or "zh-Hans-CN".
var languageTag = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)

// App is the API bound to the frontend. It stays thin: it validates input,
// delegates to internal packages and maps failures to error codes.
type App struct {
	ctx    context.Context
	store  *store.Store
	logger *slog.Logger
}

// Settings is the user's preferences as seen by the frontend.
type Settings struct {
	// Language is the language chosen in Settings, or "" to follow the system.
	Language string `json:"language"`
	// SystemLocale is the OS language as a BCP 47 tag, or "" if unknown.
	SystemLocale string `json:"systemLocale"`
	// Theme is "system", "light" or "dark".
	Theme string `json:"theme"`
}

// NewApp creates the bound API on top of an open store.
func NewApp(st *store.Store, logger *slog.Logger) *App {
	return &App{store: st, logger: logger}
}

// startup is called by Wails when the app starts. The context is needed by the runtime package.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// shutdown is called by Wails after the window has closed.
func (a *App) shutdown(context.Context) {}

// onSecondInstance brings the existing window forward when the user launches Privé again.
func (a *App) onSecondInstance(options.SecondInstanceData) {
	if a.ctx == nil {
		return
	}
	runtime.WindowUnminimise(a.ctx)
	runtime.Show(a.ctx)
}

// GetSettings returns the stored preferences plus the detected system language.
func (a *App) GetSettings() (Settings, error) {
	ctx := a.context()
	lang, _, err := a.store.Setting(ctx, store.SettingLanguage)
	if err != nil {
		a.logger.Error("load language setting", "err", err)
		return Settings{}, errSettingsLoad
	}
	theme, ok, err := a.store.Setting(ctx, store.SettingTheme)
	if err != nil {
		a.logger.Error("load theme setting", "err", err)
		return Settings{}, errSettingsLoad
	}
	if !ok || !isTheme(theme) {
		theme = ThemeSystem
	}
	return Settings{
		Language:     lang,
		SystemLocale: platform.SystemLocale(),
		Theme:        theme,
	}, nil
}

// SetLanguage saves the UI language (a BCP 47 tag). An empty string means "follow the system".
func (a *App) SetLanguage(lang string) error {
	ctx := a.context()
	var err error
	switch {
	case lang == "":
		err = a.store.DeleteSetting(ctx, store.SettingLanguage)
	case languageTag.MatchString(lang):
		err = a.store.SetSetting(ctx, store.SettingLanguage, lang)
	default:
		return errInvalidSetting
	}
	if err != nil {
		a.logger.Error("save language setting", "err", err)
		return errSettingsSave
	}
	return nil
}

// SetTheme saves the color theme: "system", "light" or "dark".
func (a *App) SetTheme(theme string) error {
	if !isTheme(theme) {
		return errInvalidSetting
	}
	ctx := a.context()
	var err error
	if theme == ThemeSystem {
		err = a.store.DeleteSetting(ctx, store.SettingTheme)
	} else {
		err = a.store.SetSetting(ctx, store.SettingTheme, theme)
	}
	if err != nil {
		a.logger.Error("save theme setting", "err", err)
		return errSettingsSave
	}
	return nil
}

// context returns the Wails context once started, or a background context (e.g. in tests).
func (a *App) context() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

func isTheme(s string) bool {
	return s == ThemeSystem || s == ThemeLight || s == ThemeDark
}
