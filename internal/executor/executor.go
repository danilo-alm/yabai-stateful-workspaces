// Package executor wraps external command invocation (yabai).
package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
)

// Space represents a space returned by `yabai -m query --spaces`.
type Space struct {
	Index int    `json:"index"`
	Label string `json:"label"`
}

// YabaiExecutor executes yabai subcommands.
type YabaiExecutor struct {
	binary string // path to the yabai binary, e.g. "/usr/local/bin/yabai"
}

// New creates an YabaiExecutor. If binary is empty it falls back to "yabai"
// resolved via PATH.
func New(binary string) *YabaiExecutor {
	if binary == "" {
		binary = "yabai"
	}
	return &YabaiExecutor{binary: binary}
}

// FocusSpace runs `yabai -m space --focus <name>`.
func (y *YabaiExecutor) FocusSpace(name string) error {
	args := []string{"-m", "space", "--focus", name}
	cmd := exec.Command(y.binary, args...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("yabai %v: %w (output: %s)", args, err, out)
	}

	slog.Info("yabai space focused", "space", name)
	return nil
}

// MoveWindowToSpace runs `yabai -m window --space <name>`.
func (y *YabaiExecutor) MoveWindowToSpace(name string) error {
	args := []string{"-m", "window", "--space", name}
	cmd := exec.Command(y.binary, args...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("yabai %v: %w (output: %s)", args, err, out)
	}

	slog.Info("yabai window moved to space", "space", name)
	return nil
}

// QuerySpaces runs `yabai -m query --spaces` and returns parsed spaces.
func (y *YabaiExecutor) QuerySpaces(ctx context.Context) ([]Space, error) {
	args := []string{"-m", "query", "--spaces"}
	cmd := exec.CommandContext(ctx, y.binary, args...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("yabai %v: %w (output: %s)", args, err, out)
	}

	var spaces []Space
	if err := json.Unmarshal(out, &spaces); err != nil {
		return nil, fmt.Errorf("unmarshal spaces query output: %w", err)
	}

	return spaces, nil
}

// LabelSpace runs `yabai -m space <index> --label <label>`.
func (y *YabaiExecutor) LabelSpace(ctx context.Context, index int, label string) error {
	args := []string{"-m", "space", fmt.Sprintf("%d", index), "--label", label}
	cmd := exec.CommandContext(ctx, y.binary, args...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("yabai %v: %w (output: %s)", args, err, out)
	}

	slog.Info("yabai space labeled", "index", index, "label", label)
	return nil
}

// CreateSpace runs `yabai -m space --create`.
func (y *YabaiExecutor) CreateSpace(ctx context.Context) error {
	args := []string{"-m", "space", "--create"}
	cmd := exec.CommandContext(ctx, y.binary, args...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("yabai %v: %w (output: %s)", args, err, out)
	}

	slog.Info("yabai space created")
	return nil
}

// LabelLastSpace runs `yabai -m space last --label <label>`.
func (y *YabaiExecutor) LabelLastSpace(ctx context.Context, label string) error {
	args := []string{"-m", "space", "last", "--label", label}
	cmd := exec.CommandContext(ctx, y.binary, args...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("yabai %v: %w (output: %s)", args, err, out)
	}

	slog.Info("yabai last space labeled", "label", label)
	return nil
}
