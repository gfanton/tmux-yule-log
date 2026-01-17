# Tmux Yule Log

> Forked from [leereilly/gh-yule-log](https://github.com/leereilly/gh-yule-log)

![Yule Log GIF](screencap.gif)

A tmux screensaver plugin that turns your terminal into a festive, animated Yule log. Displays scrolling git commits from your current repository over the fire animation.

## Requirements

- tmux 3.2+ (for popup and command-alias support)
- Go 1.24+ (for building from source, or use nix/pre-built binary)
- A modern terminal that supports ANSI colors

## Installation

### Using TPM (recommended)

Add to your `~/.tmux.conf`:

```bash
set -g @plugin 'gfanton/tmux-yule-log'

# Optional: auto-start screensaver after 5 minutes of inactivity
set -g @yule-log-idle-time "300"
```

Then press `prefix + I` to install. The binary will be built automatically if Go is available.

### Using Nix

```bash
# Install full tmux plugin (includes binary)
nix profile install github:gfanton/tmux-yule-log

# Or just the binary
nix profile install github:gfanton/tmux-yule-log#yule-log
```

### Manual Installation

```bash
git clone https://github.com/gfanton/tmux-yule-log ~/.tmux/plugins/tmux-yule-log
```

Then source the plugin in your `~/.tmux.conf`:

```bash
run-shell ~/.tmux/plugins/tmux-yule-log/yule-log.tmux
```

## Usage

### Key Bindings

| Key | Action |
|-----|--------|
| `prefix + Y` | Trigger screensaver |
| `prefix + Alt+Y` | Toggle idle watcher on/off |
| `prefix + L` | Lock session (if lock enabled) |

### tmux Commands

Press `prefix + :` then type any of these commands (tab-completion works):

| Command | Action |
|---------|--------|
| `:yule-log` | Trigger screensaver |
| `:yule-start` | Start idle watcher |
| `:yule-stop` | Stop idle watcher |
| `:yule-toggle` | Toggle idle watcher on/off |
| `:yule-status` | Check if idle watcher is running |
| `:yule-lock` | Lock the session |
| `:yule-set-password` | Set lock password |

### Screensaver Controls

| Key | Action |
|-----|--------|
| <kbd>↑</kbd> | Increase flame intensity |
| <kbd>↓</kbd> | Decrease flame intensity |
| Any other key | Exit screensaver |

The screensaver displays full-screen, covering all panes and windows. Press any key to exit and return to your previous view.

## Configuration

Add to your `~/.tmux.conf`:

```bash
# Idle timeout in seconds before screensaver activates (0 = disabled)
set -g @yule-log-idle-time "300"

# Show git commit ticker: "on" or "off"
set -g @yule-log-show-ticker "on"

# Lock mode
set -g @yule-log-lock-enabled "off"        # Enable lock feature
set -g @yule-log-lock-socket-protect "on"  # Restrict socket during lock
```

## Session Locking

Password-protected session locking.

### Setup

1. **Set a password** (from tmux):
   ```
   prefix + :yule-set-password
   ```
   Supports regular characters and arrow keys for extra complexity.

2. **Enable lock mode** in `~/.tmux.conf`:
   ```bash
   set -g @yule-log-lock-enabled "on"
   ```

3. **Lock your session** with `prefix + L` or `:yule-lock`

### Features

- **Argon2id hashing** with OWASP-recommended parameters
- **Socket protection** prevents `tmux attach` bypass during lock
- **Secure memory** - password input uses memguard (mlocked, wiped)

### Limitations

This is a convenience lock for casual access protection. It does **not** protect against root users, SIGKILL, or physical attacks. Combine with OS screen lock for real security.

## Screenshots

![](images/gh-yule-log-vanilla.gif)

## Credits

- Original project by [@leereilly](https://github.com/leereilly): [gh-yule-log](https://github.com/leereilly/gh-yule-log)
- Flame intensity controls by [@shplok](https://github.com/shplok) via [#7](https://github.com/leereilly/gh-yule-log/pull/7)
- Fire algorithm inspired by [@msimpson's curses-based ASCII art fire](https://gist.github.com/msimpson/1096950)

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details.
