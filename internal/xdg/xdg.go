package xdg

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

const appName = "tmux-yule-log"

// ConfigDir returns the configuration directory for tmux-yule-log.
// On Linux: $XDG_CONFIG_HOME/tmux-yule-log or ~/.config/tmux-yule-log
// On macOS: ~/Library/Application Support/tmux-yule-log (fallback to XDG if set)
//
// Note: This function creates the directory (with 0700 permissions) if it doesn't exist.
func ConfigDir() (string, error) {
	var base string

	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		base = configHome
	} else if runtime.GOOS == "darwin" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, "Library", "Application Support")
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}

	dir := filepath.Join(base, appName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

// RuntimeDir returns the runtime directory for tmux-yule-log.
// On Linux: $XDG_RUNTIME_DIR/tmux-yule-log or /tmp/tmux-yule-log-$UID
// On macOS: $TMPDIR/tmux-yule-log-$UID
// Runtime dir is for ephemeral state that should be cleared on logout/reboot.
//
// Note: This function creates the directory (with 0700 permissions) if it doesn't exist.
func RuntimeDir() (string, error) {
	var base string

	if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
		base = runtimeDir
	} else {
		tmpdir := os.TempDir()
		base = filepath.Join(tmpdir, appName+"-"+uidString())
	}

	dir := filepath.Join(base, appName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

// PasswordFile returns the path to the password hash file.
func PasswordFile() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "passwd"), nil
}

// LockStateFile returns the path to the lock state file.
func LockStateFile() (string, error) {
	dir, err := RuntimeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "lock.state"), nil
}

// uidString returns the current user's UID as a string.
func uidString() string {
	return strconv.Itoa(os.Getuid())
}
