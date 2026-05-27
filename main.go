// Package main is the entry point for the yabai workspace daemon.
//
// The daemon listens on a Unix named pipe (FIFO) for single-character commands
// written by skhd and translates them into yabai focus-space calls according
// to the configured key mapping.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	logHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slog.SetDefault(slog.New(logHandler))

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "generate-skhd-bindings", "generate-shkd-bindings":
			runGenerateSkhd(os.Args[2:])
			return
		case "--start-service":
			runStartService()
			return
		case "--stop-service":
			runStopService()
			return
		case "--restart-service":
			runStopService()
			runStartService()
			return
		}
	}

	runDaemon(os.Args[1:])
}

func runDaemon(args []string) {
	cfg := defaultConfig
	configPath := resolveConfigPath()
	if err := loadConfigFile(configPath, &cfg); err != nil {
		slog.Error("failed to load configuration file", "path", configPath, "err", err)
		os.Exit(1)
	}
	fs := flag.NewFlagSet("yabai-stateful-workspaces", flag.ExitOnError)
	dynamicLetters := fs.String("dynamic-letters", cfg.DynamicLetters, "lowercase letters to use as dynamic workspace keys")
	sets := fs.Int("sets", cfg.Sets, "number of workspace sets (1–9)")
	fifoPath := fs.String("fifo", cfg.FIFOPath, "path to the FIFO named pipe")
	yabaiBinary := fs.String("yabai", cfg.YabaiBinary, "path or name of the yabai executable")
	globalKeys := fs.String("global-keys", cfg.GlobalKeys, "letters whose spaces are just <letter>")
	_ = fs.Parse(args)
	explicitFlags := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		explicitFlags[f.Name] = true
	})
	if explicitFlags["dynamic-letters"] {
		cfg.DynamicLetters = *dynamicLetters
	}
	if explicitFlags["sets"] {
		cfg.Sets = *sets
	}
	if explicitFlags["fifo"] {
		cfg.FIFOPath = *fifoPath
	}
	if explicitFlags["yabai"] {
		cfg.YabaiBinary = *yabaiBinary
	}
	if explicitFlags["global-keys"] {
		cfg.GlobalKeys = *globalKeys
	}
	if err := validateConfig(cfg); err != nil {
		slog.Error("invalid configuration", "err", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	daemon := NewDaemon(cfg)
	if err := daemon.Run(ctx); err != nil {
		slog.Error("daemon exited with error", "err", err)
		os.Exit(1)
	}
}
