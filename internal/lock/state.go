package lock

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"yule-log/internal/xdg"
)

var ErrNotLocked = errors.New("session is not locked")

// State represents the current lock state.
type State struct {
	Locked     bool        `json:"locked"`
	LockedAt   time.Time   `json:"locked_at"`
	SocketPath string      `json:"socket_path,omitempty"`
	SocketPerm os.FileMode `json:"socket_perm,omitempty"`
}

// Lock creates a lock state file indicating the session is locked.
func Lock(socketPath string, socketPerm os.FileMode) error {
	state := State{
		Locked:     true,
		LockedAt:   time.Now(),
		SocketPath: socketPath,
		SocketPerm: socketPerm,
	}

	return saveState(&state)
}

// Unlock removes the lock state file.
func Unlock() error {
	path, err := xdg.LockStateFile()
	if err != nil {
		return fmt.Errorf("getting lock state file path: %w", err)
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing lock state file: %w", err)
	}

	return nil
}

// IsLocked checks if there is an active lock.
func IsLocked() bool {
	state, err := LoadState()
	if err != nil {
		return false
	}
	return state.Locked
}

// LoadState reads the current lock state from the state file.
func LoadState() (*State, error) {
	path, err := xdg.LockStateFile()
	if err != nil {
		return nil, fmt.Errorf("getting lock state file path: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotLocked
		}
		return nil, fmt.Errorf("reading lock state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parsing lock state: %w", err)
	}

	return &state, nil
}

// saveState writes the lock state to the state file.
func saveState(state *State) error {
	path, err := xdg.LockStateFile()
	if err != nil {
		return fmt.Errorf("getting lock state file path: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling lock state: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing lock state file: %w", err)
	}

	return nil
}

// LockDuration returns how long the session has been locked.
func LockDuration() (time.Duration, error) {
	state, err := LoadState()
	if err != nil {
		return 0, err
	}

	return time.Since(state.LockedAt), nil
}
