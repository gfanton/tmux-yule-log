# gh-yule-log

A tiny [GitHub CLI](https://cli.github.com/) extension that turns your terminal into a festive, animated Yule log using a terminal-based fire effect.

## Requirements

- `gh` (GitHub CLI) installed and configured
- Go toolchain (Go 1.21+) installed
- A terminal that supports ANSI colors

## Installation

From the directory containing this repository (for local development):

```bash
gh extension install .
```

Or from GitHub (once this repo is pushed, replace the owner as needed):

```bash
gh extension install <your-user-or-org>/gh-yule-log
```

## Usage

Run the extension with:

```bash
gh yule-log
```

- Your terminal will fill with a flickering, colored fire effect.
- Press any key to exit.

## How it works

- The `gh-yule-log` executable is the GitHub CLI extension entrypoint.
- It runs the Go program in `main.go`, which uses the `tcell` library to:
  - Draw colored characters across the full screen.
  - Simulate heat propagation upward from the bottom row.
  - Continuously update the display until you press a key.
