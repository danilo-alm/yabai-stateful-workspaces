// cmd_skhd.go implements the generate-skhd-bindings subcommand, which prints
// a ready-to-use skhd configuration that wires keyboard shortcuts to the daemon.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
)

func runGenerateSkhd(args []string) {
	cfg := defaultConfig
	configPath := resolveConfigPath()
	if err := loadConfigFile(configPath, &cfg); err != nil {
		slog.Error("failed to load configuration file", "path", configPath, "err", err)
		os.Exit(1)
	}
	fs := flag.NewFlagSet("generate-skhd-bindings", flag.ExitOnError)
	mod := fs.String("mod", "alt", "modifier key prefix to use (e.g. alt, hyper, cmd + alt)")
	dynamicLetters := fs.String("dynamic-letters", cfg.DynamicLetters, "lowercase letters to use as dynamic workspace keys")
	sets := fs.Int("sets", cfg.Sets, "number of workspace sets (1–9)")
	fifoPath := fs.String("fifo", cfg.FIFOPath, "path to the FIFO named pipe")
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
	if explicitFlags["global-keys"] {
		cfg.GlobalKeys = *globalKeys
	}
	if err := validateConfig(cfg); err != nil {
		slog.Error("invalid configuration", "err", err)
		os.Exit(1)
	}
	fmt.Println("# ─────────────────────────────────────────────────────────────")
	fmt.Println("# yabai-stateful-workspaces  ·  skhd configuration")
	fmt.Println("#")
	fmt.Println("# Each binding writes a single character into the daemon's FIFO.")
	fmt.Println("# The daemon translates it into the appropriate yabai command.")
	fmt.Println("# ─────────────────────────────────────────────────────────────")
	fmt.Println()
	fmt.Println()
	fmt.Printf("# ── Set number keys (%s) ──────────────────────────────────────────\n", *mod)
	for i := 1; i <= cfg.Sets; i++ {
		fmt.Printf("%s - %d : echo -n \"%d\" > %s\n", *mod, i, i, cfg.FIFOPath)
	}
	fmt.Println()
	fmt.Printf("# ── Dynamic workspace keys ────────────────────────────────────\n")
	for i := 0; i < len(cfg.DynamicLetters); i++ {
		fmt.Printf("%s - %c : echo -n \"%c\" > %s\n", *mod, cfg.DynamicLetters[i], cfg.DynamicLetters[i], cfg.FIFOPath)
		fmt.Printf("%s + shift - %c : echo -n \"%c\" > %s\n", *mod, cfg.DynamicLetters[i], cfg.DynamicLetters[i]-32, cfg.FIFOPath)
	}
	fmt.Println()
	fmt.Printf("# ── Global workspace keys ─────────────────────────────────────\n")
	for i := 0; i < len(cfg.GlobalKeys); i++ {
		fmt.Printf("%s - %c : echo -n \"%c\" > %s\n", *mod, cfg.GlobalKeys[i], cfg.GlobalKeys[i], cfg.FIFOPath)
		fmt.Printf("%s + shift - %c : echo -n \"%c\" > %s\n", *mod, cfg.GlobalKeys[i], cfg.GlobalKeys[i]-32, cfg.FIFOPath)
	}
}
