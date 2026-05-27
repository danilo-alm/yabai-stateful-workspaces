// config.go handles configuration file loading and validation for the daemon.
package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func resolveConfigPath() string {
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg != "" {
		return filepath.Join(xdg, "yabai", "yabai-ds.conf")
	}
	home := os.Getenv("HOME")
	return filepath.Join(home, ".config", "yabai", "yabai-ds.conf")
}

func loadConfigFile(path string, cfg *Config) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	slog.Info("loading configuration file", "path", path)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid config line: %q (must be key=value)", line)
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		switch key {
		case "dynamic-letters":
			cfg.DynamicLetters = val
		case "sets":
			var s int
			if _, err := fmt.Sscan(val, &s); err != nil {
				return fmt.Errorf("invalid sets value %q: %w", val, err)
			}
			cfg.Sets = s
		case "fifo":
			cfg.FIFOPath = val
		case "yabai":
			cfg.YabaiBinary = val
		case "global-keys":
			cfg.GlobalKeys = val
		default:
			return fmt.Errorf("unknown configuration option %q", key)
		}
	}
	return scanner.Err()
}

func validateConfig(cfg Config) error {
	if cfg.Sets < 1 || cfg.Sets > 9 {
		return fmt.Errorf("sets must be between 1 and 9, got %d", cfg.Sets)
	}
	if cfg.DynamicLetters == "" {
		return fmt.Errorf("dynamic-letters must not be empty")
	}
	for _, ch := range cfg.DynamicLetters {
		if ch < 'a' || ch > 'z' {
			return fmt.Errorf("dynamic-letters: %q is not a lowercase letter", ch)
		}
	}
	for i := 0; i < len(cfg.DynamicLetters); i++ {
		for j := 0; j < len(cfg.GlobalKeys); j++ {
			if cfg.DynamicLetters[i] == cfg.GlobalKeys[j] {
				return fmt.Errorf("letter %q appears in both dynamic-letters and global keys %q", cfg.DynamicLetters[i], cfg.GlobalKeys)
			}
		}
	}
	return nil
}
