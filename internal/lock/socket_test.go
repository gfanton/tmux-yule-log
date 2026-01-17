package lock

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTmuxSocketPath(t *testing.T) {
	tests := []struct {
		name     string
		tmuxEnv  string
		want     string
		wantErr  error
		clearEnv bool
	}{
		{
			name:    "valid format with pid and session",
			tmuxEnv: "/tmp/tmux-501/default,12345,0",
			want:    "/tmp/tmux-501/default",
			wantErr: nil,
		},
		{
			name:    "path with spaces",
			tmuxEnv: "/path with spaces/tmux socket,123,0",
			want:    "/path with spaces/tmux socket",
			wantErr: nil,
		},
		{
			name:     "empty TMUX env",
			tmuxEnv:  "",
			clearEnv: true,
			want:     "",
			wantErr:  ErrNoTmuxSocket,
		},
		{
			name:    "just commas",
			tmuxEnv: ",,,",
			want:    "",
			wantErr: ErrInvalidTmuxSocket,
		},
		{
			name:    "single comma with empty path",
			tmuxEnv: ",123",
			want:    "",
			wantErr: ErrInvalidTmuxSocket,
		},
		{
			name:    "only socket path no commas",
			tmuxEnv: "/tmp/tmux-socket",
			want:    "/tmp/tmux-socket",
			wantErr: nil,
		},
		{
			name:    "socket path with trailing comma",
			tmuxEnv: "/tmp/tmux-socket,",
			want:    "/tmp/tmux-socket",
			wantErr: nil,
		},
		{
			name:    "complex path",
			tmuxEnv: "/private/var/folders/abc123/T/tmux-501/default,98765,1",
			want:    "/private/var/folders/abc123/T/tmux-501/default",
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original value
			originalEnv, hadEnv := os.LookupEnv("TMUX")
			defer func() {
				if hadEnv {
					os.Setenv("TMUX", originalEnv)
				} else {
					os.Unsetenv("TMUX")
				}
			}()

			// Set test value
			if tt.clearEnv {
				os.Unsetenv("TMUX")
			} else {
				os.Setenv("TMUX", tt.tmuxEnv)
			}

			got, err := GetTmuxSocketPath()

			assert.Equal(t, tt.wantErr, err, "error mismatch")
			assert.Equal(t, tt.want, got, "path mismatch")
		})
	}
}
