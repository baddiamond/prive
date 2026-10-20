package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Well-known setting keys.
const (
	// SettingLanguage holds the UI language chosen by the user (BCP 47). Absent means "follow the OS".
	SettingLanguage = "language"
	// SettingTheme holds "light" or "dark". Absent means "follow the OS".
	SettingTheme = "theme"
	// SettingLegalAcceptedVersion holds the version of the Disclaimer and Acceptable Use Policy
	// the user agreed to during onboarding.
	SettingLegalAcceptedVersion = "legal.acceptedVersion"
	// SettingLegalAcceptedAt holds when they agreed (RFC 3339).
	SettingLegalAcceptedAt = "legal.acceptedAt"
	// SettingOnboardingStep holds the last onboarding step reached, so closing the window
	// mid-flow resumes there.
	SettingOnboardingStep = "onboarding.step"
	// SettingOnboardingCompletedAt holds when onboarding finished (RFC 3339).
	SettingOnboardingCompletedAt = "onboarding.completedAt"
)

// Setting returns the value stored under key and whether it exists.
func (s *Store) Setting(ctx context.Context, key string) (string, bool, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read setting %q: %w", key, err)
	}
	return value, true, nil
}

// SetSetting stores value under key, replacing any previous value.
func (s *Store) SetSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value, time.Now().UnixMilli())
	if err != nil {
		return fmt.Errorf("write setting %q: %w", key, err)
	}
	return nil
}

// DeleteSetting removes key. Deleting a missing key is not an error.
func (s *Store) DeleteSetting(ctx context.Context, key string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM settings WHERE key = ?`, key); err != nil {
		return fmt.Errorf("delete setting %q: %w", key, err)
	}
	return nil
}
