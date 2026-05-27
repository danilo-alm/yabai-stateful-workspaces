// cmd_service.go implements the --start-service and --stop-service subcommands,
// which manage the daemon as a macOS launchd LaunchAgent.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	launchdLabel     = "com.local.yabai-stateful-workspaces"
	launchdPlistName = launchdLabel + ".plist"
)

func launchdPlistPath() string {
	home := os.Getenv("HOME")
	return filepath.Join(home, "Library", "LaunchAgents", launchdPlistName)
}

func writePlist(plistPath, binaryPath string) error {
	home := os.Getenv("HOME")
	logDir := filepath.Join(home, "Library", "Logs")
	currentPath := os.Getenv("PATH")

	const tmpl = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>%s</string>
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s/%s.out.log</string>
    <key>StandardErrorPath</key>
    <string>%s/%s.err.log</string>
    <key>ProcessType</key>
    <string>Interactive</string>
</dict>
</plist>
`
	content := fmt.Sprintf(tmpl,
		launchdLabel,
		binaryPath,
		currentPath,
		logDir, launchdLabel,
		logDir, launchdLabel,
	)

	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return fmt.Errorf("create LaunchAgents dir: %w", err)
	}
	return os.WriteFile(plistPath, []byte(content), 0o644)
}

func guiDomain() string {
	return fmt.Sprintf("gui/%d", os.Getuid())
}

func launchctl(args ...string) error {
	cmd := exec.Command("launchctl", args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runStartService() {
	binaryPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: could not determine binary path: %v\n", err)
		os.Exit(1)
	}
	binaryPath, err = filepath.EvalSymlinks(binaryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: could not resolve binary path: %v\n", err)
		os.Exit(1)
	}
	plistPath := launchdPlistPath()
	if err := writePlist(plistPath, binaryPath); err != nil {
		fmt.Fprintf(os.Stderr, "error: could not write plist: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("plist written → %s\n", plistPath)
	if err := launchctl("bootstrap", guiDomain(), plistPath); err != nil {
		fmt.Fprintf(os.Stderr, "error: launchctl bootstrap failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("service started")
}

func runStopService() {
	plistPath := launchdPlistPath()
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "error: plist not found at %s — is the service installed?\n", plistPath)
		os.Exit(1)
	}
	if err := launchctl("bootout", guiDomain(), plistPath); err != nil {
		fmt.Fprintf(os.Stderr, "error: launchctl bootout failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("service stopped")
}
