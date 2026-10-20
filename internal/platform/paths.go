package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// DataDir returns the folder where Privé keeps all of its local data
// (database, engine sessions). It does not create the folder.
//
//   - macOS:   ~/Library/Application Support/Prive
//   - Windows: %AppData%\Prive
//   - Linux:   $XDG_DATA_HOME/prive (default ~/.local/share/prive)
func DataDir() (string, error) {
	return dataDir(runtime.GOOS, os.Getenv, os.UserHomeDir, os.UserConfigDir)
}

// EnsureDataDir returns DataDir after creating it (owner-only permissions) if needed.
func EnsureDataDir() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create data dir: %w", err)
	}
	return dir, nil
}

// dataDir is DataDir with its OS inputs injected, so every platform can be tested anywhere.
func dataDir(goos string, getenv func(string) string, homeDir, configDir func() (string, error)) (string, error) {
	if goos == "darwin" || goos == "windows" {
		base, err := configDir()
		if err != nil {
			return "", fmt.Errorf("resolve config dir: %w", err)
		}
		return filepath.Join(base, "Prive"), nil
	}

	if xdg := getenv("XDG_DATA_HOME"); xdg != "" && filepath.IsAbs(xdg) {
		return filepath.Join(xdg, "prive"), nil
	}
	home, err := homeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".local", "share", "prive"), nil
}
