package lock

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

var (
	ErrNoTmuxSocket      = errors.New("TMUX environment variable not set")
	ErrInvalidTmuxSocket = errors.New("TMUX environment variable malformed")
	ErrSocketNotFound    = errors.New("tmux socket not found")
)

// GetTmuxSocketPath extracts the socket path from the TMUX environment variable.
// TMUX format: /path/to/socket,pid,session
func GetTmuxSocketPath() (string, error) {
	tmuxEnv := os.Getenv("TMUX")
	if tmuxEnv == "" {
		return "", ErrNoTmuxSocket
	}

	parts := strings.Split(tmuxEnv, ",")
	if len(parts) < 1 || parts[0] == "" {
		return "", ErrInvalidTmuxSocket
	}

	return parts[0], nil
}

// GetSocketPermissions returns the current permissions of the tmux socket.
func GetSocketPermissions(socketPath string) (os.FileMode, error) {
	info, err := os.Stat(socketPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, ErrSocketNotFound
		}
		return 0, fmt.Errorf("stat socket: %w", err)
	}

	return info.Mode().Perm(), nil
}

// RestrictSocket sets the tmux socket permissions to 000, preventing new connections.
// Returns the original permissions so they can be restored later.
func RestrictSocket(socketPath string) (os.FileMode, error) {
	originalPerm, err := GetSocketPermissions(socketPath)
	if err != nil {
		return 0, err
	}

	if err := os.Chmod(socketPath, 0000); err != nil {
		return 0, fmt.Errorf("restricting socket permissions: %w", err)
	}

	return originalPerm, nil
}

// RestoreSocket restores the original permissions on the tmux socket.
func RestoreSocket(socketPath string, perm os.FileMode) error {
	if err := os.Chmod(socketPath, perm); err != nil {
		return fmt.Errorf("restoring socket permissions: %w", err)
	}
	return nil
}

// RestoreSocketFromState restores socket permissions using the stored lock state.
func RestoreSocketFromState() error {
	state, err := LoadState()
	if err != nil {
		return err
	}

	if state.SocketPath == "" || state.SocketPerm == 0 {
		return nil
	}

	return RestoreSocket(state.SocketPath, state.SocketPerm)
}
