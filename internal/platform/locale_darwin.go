package platform

import (
	"context"
	"os"
	"os/exec"
	"time"
)

// systemLocale asks macOS for the first entry of the user's preferred languages.
// GUI apps usually start without LANG, so the environment is only a fallback.
func systemLocale() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "defaults", "read", "-g", "AppleLanguages").Output()
	if err == nil {
		if lang := parseAppleLanguages(string(out)); lang != "" {
			return lang
		}
	}
	return localeFromEnv(os.Getenv)
}
