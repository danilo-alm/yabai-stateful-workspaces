// Package main is the entry point for the yabai workspace daemon.
//
// The daemon listens on a Unix named pipe (FIFO) for single-character commands
// written by skhd and translates them into yabai focus-space calls according
// to the configured key mapping.
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/danilo-alm/yabai-stateful-workspaces/internal/executor"
	"github.com/danilo-alm/yabai-stateful-workspaces/internal/fifo"
	"github.com/danilo-alm/yabai-stateful-workspaces/internal/state"
)

// Config holds all tuneable parameters for the daemon.
type Config struct {
	FIFOPath       string
	YabaiBinary    string
	GlobalKeys     string
	DynamicLetters string
	Sets           int
}

var defaultConfig = Config{
	FIFOPath:       "/tmp/yabai-stateful-workspaces.fifo",
	YabaiBinary:    "yabai",
	GlobalKeys:     "zx",
	DynamicLetters: "qweasd",
	Sets:           3,
}

type Daemon struct {
	cfg    Config
	state  *state.Manager
	exec   *executor.YabaiExecutor
	reader *fifo.Reader
	global map[byte]struct{}
}

func NewDaemon(cfg Config) *Daemon {
	global := make(map[byte]struct{}, len(cfg.GlobalKeys))
	for i := 0; i < len(cfg.GlobalKeys); i++ {
		global[cfg.GlobalKeys[i]] = struct{}{}
	}
	return &Daemon{
		cfg:    cfg,
		state:  state.New(),
		exec:   executor.New(cfg.YabaiBinary),
		reader: fifo.New(cfg.FIFOPath),
		global: global,
	}
}

func (d *Daemon) Run(ctx context.Context) error {
	if err := d.setupWorkspaces(ctx); err != nil {
		return fmt.Errorf("failed to setup workspaces: %w", err)
	}
	readerErrCh := make(chan error, 1)
	go func() {
		readerErrCh <- d.reader.Run(ctx)
	}()
	slog.Info("daemon started", "fifo", d.cfg.FIFOPath, "global_keys", d.cfg.GlobalKeys)
	for {
		select {
		case <-ctx.Done():
			if err := <-readerErrCh; err != nil {
				slog.Warn("reader exited with error", "err", err)
			}
			slog.Info("daemon stopped")
			return nil
		case token, ok := <-d.reader.Lines:
			if !ok {
				return <-readerErrCh
			}
			d.dispatch(token)
		}
	}
}

func (d *Daemon) dispatch(token string) {
	if len(token) != 1 {
		slog.Warn("ignored multi-character token", "token", token)
		return
	}
	key := token[0]
	switch {
	case key >= '1' && key <= '9':
		n := int(key - '0')
		if err := d.state.UpdateSetNumber(n); err != nil {
			slog.Error("state update failed", "err", err)
			return
		}
		slog.Info("set number updated", "set", n)
	case key >= 'a' && key <= 'z' && d.isGlobal(key):
		spaceName := string([]byte{key})
		if err := d.exec.FocusSpace(spaceName); err != nil {
			slog.Error("focus global space failed", "space", spaceName, "err", err)
		}
	case key >= 'A' && key <= 'Z' && d.isGlobal(key+32):
		spaceName := string([]byte{key + 32})
		if err := d.exec.MoveWindowToSpace(spaceName); err != nil {
			slog.Error("move to global space failed", "space", spaceName, "err", err)
		}
	case key >= 'a' && key <= 'z':
		spaceName := d.state.DynamicSpaceName(key)
		if err := d.exec.FocusSpace(spaceName); err != nil {
			slog.Error("focus dynamic space failed", "space", spaceName, "err", err)
		}
	case key >= 'A' && key <= 'Z':
		spaceName := d.state.DynamicSpaceName(key + 32)
		if err := d.exec.MoveWindowToSpace(spaceName); err != nil {
			slog.Error("move to dynamic space failed", "space", spaceName, "err", err)
		}
	default:
		slog.Warn("unrecognised key, ignored", "key", string(key))
	}
}

func (d *Daemon) isGlobal(key byte) bool {
	_, ok := d.global[key]
	return ok
}

func (d *Daemon) DesiredLabels() []string {
	var labels []string
	for set := 1; set <= d.cfg.Sets; set++ {
		for i := 0; i < len(d.cfg.DynamicLetters); i++ {
			labels = append(labels, fmt.Sprintf("%c%d", d.cfg.DynamicLetters[i], set))
		}
	}
	for i := 0; i < len(d.cfg.GlobalKeys); i++ {
		labels = append(labels, string([]byte{d.cfg.GlobalKeys[i]}))
	}
	return labels
}

func (d *Daemon) setupWorkspaces(ctx context.Context) error {
	slog.Info("syncing yabai workspaces", "dynamic_letters", d.cfg.DynamicLetters, "sets", d.cfg.Sets)
	spaces, err := d.exec.QuerySpaces(ctx)
	if err != nil {
		return fmt.Errorf("query spaces: %w", err)
	}
	existingLabels := make(map[int]string)
	for _, s := range spaces {
		existingLabels[s.Index] = s.Label
	}
	desired := d.DesiredLabels()
	for i, name := range desired {
		idx := i + 1
		currentLabel, exists := existingLabels[idx]
		if exists {
			if currentLabel != name {
				slog.Info("relabeling workspace", "index", idx, "old", currentLabel, "new", name)
				if err := d.exec.LabelSpace(ctx, idx, name); err != nil {
					return fmt.Errorf("label space %d: %w", idx, err)
				}
			}
		} else {
			slog.Info("creating and labeling workspace", "label", name)
			if err := d.exec.CreateSpace(ctx); err != nil {
				return fmt.Errorf("create space: %w", err)
			}
			if err := d.exec.LabelLastSpace(ctx, name); err != nil {
				return fmt.Errorf("label last space: %w", err)
			}
		}
	}
	return nil
}
