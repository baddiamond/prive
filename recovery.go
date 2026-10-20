package main

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/baddiamond/prive/internal/bootstrap"
	"github.com/baddiamond/prive/internal/core"
	"github.com/baddiamond/prive/internal/keys"
	"github.com/baddiamond/prive/internal/store"
)

// Recovery phrase error codes (ADR 0007). Phrase parsing codes live in internal/keys.
var (
	// errRecoveryWrongWord means a word typed to confirm the phrase is not the right one.
	// Param: position (1-based).
	errRecoveryWrongWord = core.NewError("errors.recovery.wrongWord")
	// errRecoveryTooFewAnswers means the confirmation did not include enough words.
	errRecoveryTooFewAnswers = core.NewError("errors.recovery.tooFewAnswers")
	// errNotLocked means UnlockWithPhrase was called while the data is not locked.
	errNotLocked = core.NewError("errors.recovery.notLocked")
)

// recoveryErrorCodes lists the codes defined in this file, for the translation check in tests.
var recoveryErrorCodes = []*core.CodedError{errRecoveryWrongWord, errRecoveryTooFewAnswers, errNotLocked}

// minRecoveryAnswers is how many words the user types back to show they wrote the phrase down.
const minRecoveryAnswers = 3

// RecoveryStatus says whether the user saved the recovery phrase.
type RecoveryStatus struct {
	// Confirmed is true once the user typed back some of the words.
	Confirmed bool `json:"confirmed"`
	// ConfirmedAt is when they did (RFC 3339), or "".
	ConfirmedAt string `json:"confirmedAt"`
}

// RecoveryAnswer is one word typed back by the user.
type RecoveryAnswer struct {
	// Position is the word's position in the phrase, 1–12.
	Position int `json:"position"`
	// Word is what the user typed.
	Word string `json:"word"`
}

// unlocker opens local data with the recovery phrase when it is locked.
type unlocker func(ctx context.Context, phrase string) (*bootstrap.Local, error)

// NewLockedApp creates an App for data whose key could not be unlocked from the keychain
// (keys.ErrLocked): the UI asks for the recovery phrase and UnlockWithPhrase opens it.
func NewLockedApp(dataDir string, engines engineCatalog, kc keys.Keychain, logger *slog.Logger) *App {
	a := NewFailedApp(keys.ErrLocked, dataDir, logger)
	a.engines = engines
	a.unlock = func(ctx context.Context, phrase string) (*bootstrap.Local, error) {
		return bootstrap.Recover(ctx, dataDir, phrase, kc)
	}
	return a
}

// UnlockWithPhrase opens locked data with the 12-word recovery phrase, then starts the
// accounts as a normal start would. The frontend reloads afterwards.
func (a *App) UnlockWithPhrase(phrase string) error {
	a.unlockMu.Lock()
	defer a.unlockMu.Unlock()
	if a.unlock == nil || a.failure == nil || !errors.Is(a.failure, keys.ErrLocked) {
		return errNotLocked
	}
	local, err := a.unlock(a.context(), phrase)
	if err != nil {
		return err
	}
	a.attach(local)
	a.ownedLocal = local
	a.failure = nil
	a.startup(a.context())
	return nil
}

// GetRecoveryStatus says whether the user saved the recovery phrase.
func (a *App) GetRecoveryStatus() (RecoveryStatus, error) {
	if a.failure != nil {
		return RecoveryStatus{}, a.failure
	}
	at, ok, err := a.store.Setting(a.context(), store.SettingRecoveryConfirmedAt)
	if err != nil {
		a.logger.Error("load recovery status", "err", err)
		return RecoveryStatus{}, errSettingsLoad
	}
	return RecoveryStatus{Confirmed: ok && at != "", ConfirmedAt: at}, nil
}

// GetRecoveryPhrase returns the 12 words in the list for language ("en", "es"…), English
// when there is none. Every language spells the same key, and any of them unlocks the data.
func (a *App) GetRecoveryPhrase(language string) ([]string, error) {
	if a.failure != nil {
		return nil, a.failure
	}
	return a.keys.RecoveryPhrase(language), nil
}

// ConfirmRecoveryPhrase checks words the user typed back and, when they are right,
// records that the phrase was saved.
func (a *App) ConfirmRecoveryPhrase(answers []RecoveryAnswer) error {
	if a.failure != nil {
		return a.failure
	}
	seen := map[int]bool{}
	for _, ans := range answers {
		if !a.keys.PhraseWordMatches(ans.Position-1, ans.Word) {
			return errRecoveryWrongWord.With("position", strconv.Itoa(ans.Position))
		}
		seen[ans.Position] = true
	}
	if len(seen) < minRecoveryAnswers {
		return errRecoveryTooFewAnswers
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if err := a.store.SetSetting(a.context(), store.SettingRecoveryConfirmedAt, now); err != nil {
		a.logger.Error("save recovery status", "err", err)
		return errSettingsSave
	}
	return nil
}
