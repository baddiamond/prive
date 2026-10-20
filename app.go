package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/baddiamond/prive/internal/bootstrap"
	"github.com/baddiamond/prive/internal/core"
	"github.com/baddiamond/prive/internal/keys"
	"github.com/baddiamond/prive/internal/platform"
	"github.com/baddiamond/prive/internal/store"
)

// Settings error codes. Like every error that reaches the frontend they are translation
// keys, never user-facing text (see ADR 0004).
var (
	errInvalidSetting = core.NewError("errors.settings.invalidValue")
	errSettingsSave   = core.NewError("errors.settings.saveFailed")
	errSettingsLoad   = core.NewError("errors.settings.loadFailed")
)

// Account input error codes.
var (
	errInvalidAccountName  = core.NewError("errors.account.invalidName")
	errInvalidAccountColor = core.NewError("errors.account.invalidColor")
)

// appErrorCodes lists the codes defined in this file, for the translation check in tests.
var appErrorCodes = []*core.CodedError{
	errInvalidSetting, errSettingsSave, errSettingsLoad, errInvalidAccountName, errInvalidAccountColor,
}

// accountColors are the account accent tokens of the brand (assets/brand/tokens, "account.*").
var accountColors = map[string]bool{
	"green": true, "amber": true, "rose": true, "sky": true, "violet": true, "coral": true,
}

// maxAccountNameLength caps account labels, counted in characters.
const maxAccountNameLength = 40

// engineCatalog is what the App needs from the engine registry.
type engineCatalog interface {
	core.EngineSource
	// Manifests lists every bundled engine.
	Manifests() []core.Manifest
}

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
	ctx     context.Context
	store   *store.Store
	keys    *keys.Keys
	engines engineCatalog
	manager *core.Manager
	dataDir string
	logger  *slog.Logger
	// failure is set when local data could not be opened; every data method returns it so
	// the UI can explain the problem instead of the app quitting silently.
	failure *core.CodedError
	// unlock opens locked data with the recovery phrase; nil unless the app started locked.
	unlock   unlocker
	unlockMu sync.Mutex
	// ownedLocal is local data opened by UnlockWithPhrase, closed on shutdown.
	ownedLocal *bootstrap.Local
}

// AppInfo is static information shown in Settings.
type AppInfo struct {
	Version string `json:"version"`
	// DataDir is the folder that holds every local file Privé writes.
	DataDir string `json:"dataDir"`
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

// NewApp creates the bound API on top of opened local data. The account manager is
// created here and started once the window exists (see startup).
func NewApp(local *bootstrap.Local, engines engineCatalog, logger *slog.Logger) *App {
	a := &App{engines: engines, logger: logger}
	a.attach(local)
	return a
}

// attach wires opened local data and creates the account manager.
func (a *App) attach(local *bootstrap.Local) {
	a.store, a.keys, a.dataDir = local.Store, local.Keys, local.DataDir
	a.manager = core.NewManager(local.ManagerConfig(a.engines, a.logger, a.emit))
}

// NewFailedApp creates an App that only reports why local data could not be opened.
func NewFailedApp(err error, dataDir string, logger *slog.Logger) *App {
	return &App{dataDir: dataDir, logger: logger, failure: core.AsCoded(err)}
}

// startup is called by Wails when the app starts. The context is needed by the runtime package.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if a.failure != nil {
		return
	}
	// Sessions connect in the background; the UI renders from local data meanwhile.
	if err := a.manager.Start(ctx); err != nil {
		a.logger.Error("start accounts", "err", err)
	}
	a.removeUnlinkedAccounts()
}

// removeUnlinkedAccounts deletes accounts that were added but never linked, e.g. when the
// window was closed on the QR step. They hold nothing, and the flow creates a new one.
func (a *App) removeUnlinkedAccounts() {
	for _, s := range a.manager.Accounts() {
		if s.Account.RemoteID == "" {
			if err := a.manager.RemoveAccount(a.context(), s.Account.ID); err != nil {
				a.logger.Warn("remove unlinked account", "err", err)
			}
		}
	}
}

// ReconnectAccount retries an account in the error state.
func (a *App) ReconnectAccount(accountID string) error {
	if a.failure != nil {
		return a.failure
	}
	return a.manager.Reconnect(core.AccountID(accountID))
}

// shutdown is called by Wails after the window has closed.
func (a *App) shutdown(context.Context) {
	if a.manager == nil {
		return
	}
	if err := a.manager.Close(); err != nil {
		a.logger.Warn("close accounts", "err", err)
	}
	if a.ownedLocal != nil {
		if err := a.ownedLocal.Close(); err != nil {
			a.logger.Warn("close local data", "err", err)
		}
	}
}

// emit forwards a Manager event to the frontend as a Wails event.
// Every payload carries its accountId.
func (a *App) emit(ev core.Event) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, ev.EventName(), ev)
}

// errorPayload is the shape of every error the frontend receives from a bound method.
type errorPayload struct {
	Code   string            `json:"code"`
	Params map[string]string `json:"params,omitempty"`
}

// formatError turns errors returned by bound methods into the JSON text of {code, params}.
// It must be a string: the Wails runtime wraps whatever it receives in a JavaScript Error,
// which would turn an object into "[object Object]". Errors without a code are logged and
// reported as errors.unknown.
func formatError(logger *slog.Logger) func(error) any {
	return func(err error) any {
		payload := errorPayload{Code: core.CodeUnknown}
		var ce *core.CodedError
		if errors.As(err, &ce) {
			payload = errorPayload{Code: ce.Code, Params: ce.Params}
			if ce.Err != nil {
				logger.Warn("error returned to the UI", "code", ce.Code, "err", ce.Err)
			}
		} else {
			logger.Error("uncoded error returned to the UI", "err", err)
		}
		text, mErr := json.Marshal(payload)
		if mErr != nil {
			return `{"code":"` + core.CodeUnknown + `"}`
		}
		return string(text)
	}
}

// onSecondInstance brings the existing window forward when the user launches Privé again.
func (a *App) onSecondInstance(options.SecondInstanceData) {
	if a.ctx == nil {
		return
	}
	runtime.WindowUnminimise(a.ctx)
	runtime.Show(a.ctx)
}

// GetAppInfo returns the app version and the local data folder.
func (a *App) GetAppInfo() AppInfo {
	return AppInfo{Version: version, DataDir: a.dataDir}
}

// GetSystemLocale returns the OS language as a BCP 47 tag. It works even when local data
// could not be opened, so the error screen can still speak the user's language.
func (a *App) GetSystemLocale() string {
	return platform.SystemLocale()
}

// GetSettings returns the stored preferences plus the detected system language.
func (a *App) GetSettings() (Settings, error) {
	if a.failure != nil {
		return Settings{}, a.failure
	}
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
	if a.failure != nil {
		return a.failure
	}
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
	if a.failure != nil {
		return a.failure
	}
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

// ListEngines returns the manifests of every bundled network, for the "Add account" grid.
func (a *App) ListEngines() ([]core.Manifest, error) {
	if a.failure != nil {
		return nil, a.failure
	}
	return a.engines.Manifests(), nil
}

// ListAccounts returns every account with its live connection state, in rail order.
func (a *App) ListAccounts() ([]core.AccountStatus, error) {
	if a.failure != nil {
		return nil, a.failure
	}
	list := a.manager.Accounts()
	if list == nil {
		list = []core.AccountStatus{}
	}
	return list, nil
}

// AddAccount creates an account for a network. It starts logged out: call Login next.
func (a *App) AddAccount(engineID, displayName, color string) (core.Account, error) {
	if a.failure != nil {
		return core.Account{}, a.failure
	}
	name, err := validateAccountInput(displayName, color)
	if err != nil {
		return core.Account{}, err
	}
	return a.manager.AddAccount(a.context(), core.NewAccount{EngineID: engineID, DisplayName: name, Color: color})
}

// UpdateAccount changes an account's label and color.
func (a *App) UpdateAccount(accountID, displayName, color string) (core.Account, error) {
	if a.failure != nil {
		return core.Account{}, a.failure
	}
	name, err := validateAccountInput(displayName, color)
	if err != nil {
		return core.Account{}, err
	}
	return a.manager.UpdateAccount(a.context(), core.AccountID(accountID), name, color)
}

// Login starts linking an account ("qr" for WhatsApp). QR codes, success and failure
// arrive as login:* events.
func (a *App) Login(accountID, method string) error {
	if a.failure != nil {
		return a.failure
	}
	return a.manager.Login(a.context(), core.AccountID(accountID), core.LoginMethod(method))
}

// CancelLogin abandons linking a new account (Back from the link step) and removes it.
func (a *App) CancelLogin(accountID string) error {
	return a.RemoveAccount(accountID)
}

// RemoveAccount logs the account out and deletes all of its local data.
func (a *App) RemoveAccount(accountID string) error {
	if a.failure != nil {
		return a.failure
	}
	return a.manager.RemoveAccount(a.context(), core.AccountID(accountID))
}

// validateAccountInput trims the label and checks it and the color. An empty label is
// allowed: the network's own name fills it in after linking.
func validateAccountInput(displayName, color string) (string, error) {
	name := strings.TrimSpace(displayName)
	if utf8.RuneCountInString(name) > maxAccountNameLength || strings.ContainsFunc(name, unicode.IsControl) {
		return "", errInvalidAccountName
	}
	if !accountColors[color] {
		return "", errInvalidAccountColor
	}
	return name, nil
}

// ListChats returns the stored chats for the inbox, pinned first then most recent.
// An empty accountID returns the unified inbox of every account.
func (a *App) ListChats(accountID string) ([]core.Chat, error) {
	if a.failure != nil {
		return nil, a.failure
	}
	chats, err := a.manager.ListChats(a.context(), core.ChatFilter{AccountID: core.AccountID(accountID)})
	if chats == nil && err == nil {
		chats = []core.Chat{} // JSON [] rather than null
	}
	return chats, err
}

// maxMessagePage caps GetMessages pages.
const maxMessagePage = 200

// GetMessages returns a page of a chat's stored history, oldest first. With beforeMs = 0
// it returns the newest messages; to page backwards, pass the oldest message of the
// previous page as beforeMs (its Unix time in milliseconds) and beforeID.
func (a *App) GetMessages(accountID, chatID string, beforeMs int64, beforeID string, limit int) ([]core.Message, error) {
	if a.failure != nil {
		return nil, a.failure
	}
	opts := core.GetMessagesOptions{Limit: min(max(limit, 1), maxMessagePage), BeforeID: core.MessageID(beforeID)}
	if beforeMs > 0 {
		opts.Before = time.UnixMilli(beforeMs)
	}
	msgs, err := a.manager.GetMessages(a.context(), core.AccountID(accountID), core.ChatID(chatID), opts)
	if msgs == nil && err == nil {
		msgs = []core.Message{}
	}
	return msgs, err
}

// RequestOlderHistory asks the network for messages older than those stored for a chat
// (Capabilities.HistoryOnDemand). The answer arrives as a history:result event.
func (a *App) RequestOlderHistory(accountID, chatID string) error {
	if a.failure != nil {
		return a.failure
	}
	return a.manager.RequestOlderHistory(a.context(), core.AccountID(accountID), core.ChatID(chatID))
}

// UnreadSummary counts unread messages per account (muted and archived chats excluded).
func (a *App) UnreadSummary() (core.UnreadSummary, error) {
	if a.failure != nil {
		return core.UnreadSummary{}, a.failure
	}
	return a.manager.UnreadSummary(a.context())
}
