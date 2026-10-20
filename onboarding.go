package main

import (
	"strconv"
	"time"

	"github.com/baddiamond/prive/internal/store"
)

// legalVersion is the version of DISCLAIMER.md and ACCEPTABLE_USE.md. Raising it asks every
// user to agree again before the inbox (docs/ONBOARDING.md).
const legalVersion = 1

// Onboarding steps (docs/ONBOARDING.md).
const (
	stepWelcome = 1
	stepReady   = 5
)

// Onboarding is the first-run state the frontend needs to decide what to show.
type Onboarding struct {
	// Completed is true once the user reached the inbox at least once.
	Completed bool `json:"completed"`
	// Step is the step to resume at (1–5).
	Step int `json:"step"`
	// LegalAccepted is true when the user agreed to the current legal version.
	LegalAccepted bool `json:"legalAccepted"`
	// LegalVersion is the current version of the legal documents.
	LegalVersion int `json:"legalVersion"`
}

// GetOnboarding returns the first-run state.
func (a *App) GetOnboarding() (Onboarding, error) {
	if a.failure != nil {
		return Onboarding{}, a.failure
	}
	ctx := a.context()
	completedAt, completed, err := a.store.Setting(ctx, store.SettingOnboardingCompletedAt)
	if err != nil {
		return Onboarding{}, a.loadFailed(err)
	}
	stepText, _, err := a.store.Setting(ctx, store.SettingOnboardingStep)
	if err != nil {
		return Onboarding{}, a.loadFailed(err)
	}
	accepted, _, err := a.store.Setting(ctx, store.SettingLegalAcceptedVersion)
	if err != nil {
		return Onboarding{}, a.loadFailed(err)
	}

	step, convErr := strconv.Atoi(stepText)
	if convErr != nil || step < stepWelcome || step > stepReady {
		step = stepWelcome
	}
	acceptedVersion, _ := strconv.Atoi(accepted)
	return Onboarding{
		Completed:     completed && completedAt != "",
		Step:          step,
		LegalAccepted: acceptedVersion >= legalVersion,
		LegalVersion:  legalVersion,
	}, nil
}

// SetOnboardingStep remembers the step the user reached.
func (a *App) SetOnboardingStep(step int) error {
	if a.failure != nil {
		return a.failure
	}
	if step < stepWelcome || step > stepReady {
		return errInvalidSetting
	}
	return a.saveSetting(store.SettingOnboardingStep, strconv.Itoa(step))
}

// AcceptLegal records that the user agreed to the current Disclaimer and Acceptable Use Policy.
func (a *App) AcceptLegal() error {
	if a.failure != nil {
		return a.failure
	}
	if err := a.saveSetting(store.SettingLegalAcceptedVersion, strconv.Itoa(legalVersion)); err != nil {
		return err
	}
	return a.saveSetting(store.SettingLegalAcceptedAt, time.Now().UTC().Format(time.RFC3339))
}

// CompleteOnboarding marks the first run as done; the app opens on the inbox from now on.
func (a *App) CompleteOnboarding() error {
	if a.failure != nil {
		return a.failure
	}
	return a.saveSetting(store.SettingOnboardingCompletedAt, time.Now().UTC().Format(time.RFC3339))
}

func (a *App) saveSetting(key, value string) error {
	if err := a.store.SetSetting(a.context(), key, value); err != nil {
		a.logger.Error("save setting", "key", key, "err", err)
		return errSettingsSave
	}
	return nil
}

func (a *App) loadFailed(err error) error {
	a.logger.Error("load onboarding state", "err", err)
	return errSettingsLoad
}
