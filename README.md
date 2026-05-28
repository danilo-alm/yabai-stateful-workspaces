# yabai-stateful-workspaces

A lightweight, stateful workspace manager for [yabai](https://github.com/koekeishiya/yabai) and [skhd](https://github.com/koekeishiya/skhd).

It runs as a background service, listens to a Unix named pipe (FIFO) for commands from `skhd`, and seamlessly translates them into Yabai focus and window-movement commands based on a configurable, layered layout.

## Concepts

- **Set number (1-3 by default)**: An integer (1-9) representing the current active "layer".
- **Dynamic keys**: Keys (default: `q w e a s d`) that map to spaces based on the active set. Pressing `w` while on set `2` focuses the space named `w2`.
- **Global keys**: Keys (default: `z x`) that always map to the exact space name (e.g. `z`), regardless of the active set.
- **Modifiers**: Pressing a key focuses the space. Pressing it with `SHIFT` moves the currently focused window to that space.

## Installation

You can get the latest binary from the releases section, or just build it yourself:

```sh
git clone https://github.com/danilo-alm/yabai-stateful-workspaces
cd yabai-stateful-workspaces

# Build and install to /usr/local/bin
sudo make install
```

You can also `go install` it:

```sh
go install github.com/danilo-alm/yabai-stateful-workspaces@latest
```

Or just build the binary to `dist/yabai-stateful-workspaces`:

```sh
make build
```

## Service Management

The daemon includes built-in commands to register and manage itself as a native macOS `launchd` background service.

```sh
# Install and start the background service
yabai-stateful-workspaces --start-service

# Stop and uninstall the service
yabai-stateful-workspaces --stop-service

# Restart the service
yabai-stateful-workspaces --restart-service
```

Logs are written to `~/Library/Logs/com.local.yabai-stateful-workspaces.out.log` and `.err.log`.

## Setup and Usage

### 1. Workspace Initialization
When the service starts, it automatically queries Yabai and configures your spaces. It will assign labels to existing spaces and create new ones as needed to match your configuration.

### 2. Generate skhd Bindings
The daemon includes a generator for your `skhdrc`. By default, it uses `alt` as the modifier prefix.

```sh
yabai-stateful-workspaces generate-skhd-bindings --mod alt >> ~/.config/skhd/skhdrc
skhd --reload
```

This generates bindings for changing the active set (`alt - 1-9`), focusing spaces (`alt - key`), and moving windows (`alt + shift - key`).

## Configuration

Configuration values are resolved in the following priority:
1. **Command-Line Flags** (highest priority)
2. **Config File** at `~/.config/yabai/yabai-ds.conf` (or `$XDG_CONFIG_HOME/yabai/yabai-ds.conf`)
3. **Hardcoded Defaults**

### Config File Example

```ini
# ~/.config/yabai/yabai-ds.conf
dynamic-letters = qweasd
sets = 3
global-keys = zx
fifo = /tmp/yabai-stateful-workspaces.fifo
yabai = yabai
```

### Command-Line Flags

| Flag | Default | Description |
|---|---|---|
| `--fifo` | `/tmp/yabai-stateful-workspaces.fifo` | Path to the named pipe |
| `--yabai` | `yabai` | Path or name of the yabai executable |
| `--global-keys` | `zx` | Keys focused as-is, independent of the active set |
| `--dynamic-letters` | `qweasd` | Keys used for dynamic workspace sets |
| `--sets` | `3` | Total number of workspace sets (1-9) |
