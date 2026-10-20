//go:build !darwin && !windows

package platform

import "os"

// systemLocale reads the POSIX locale environment variables (Linux and other Unix systems).
func systemLocale() string {
	return localeFromEnv(os.Getenv)
}
