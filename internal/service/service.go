// Package service keeps the router running in the background, started
// at login: a systemd user unit on Linux, a launchd agent on macOS. Where
// neither is there (Windows, WSL without systemd) the router runs as a
// detached process until logout or `ports proxy stop`. Nothing here is
// installed except by `ports proxy start`, and `ports proxy stop`
// removes all of it.
package service

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/pyjeebz/ports/internal/state"
)

// Kind names how the router is kept running.
type Kind string

const (
	Systemd Kind = "systemd" // a systemd user unit, started at login
	Launchd Kind = "launchd" // a launchd agent, started at login
	Process Kind = "process" // a detached process, until logout or stop
)

const (
	unitName = "ports-proxy.service"
	label    = "tools.ports.proxy"
	logFile  = "proxy.log"
)

// Available is the way this machine can keep the router running.
func Available() Kind {
	switch runtime.GOOS {
	case "darwin":
		if _, err := exec.LookPath("launchctl"); err == nil {
			return Launchd
		}
	case "linux":
		if exec.Command("systemctl", "--user", "show-environment").Run() == nil {
			return Systemd
		}
	}
	return Process
}

// Installed is the kind of background router set up now, if any.
func Installed() (Kind, bool) {
	if p, err := unitPath(); err == nil && exists(p) {
		return Systemd, true
	}
	if p, err := plistPath(); err == nil && exists(p) {
		return Launchd, true
	}
	return "", false
}

// LogPath is where a launchd or detached router writes what it prints;
// a systemd one writes to the journal.
func LogPath() string {
	p, _ := state.Path(logFile)
	return p
}

// Install sets up the background router running exe and starts it.
// env carries the variables the router must see that a login session
// wouldn't set — a custom state directory.
func Install(kind Kind, exe string, env []string) error {
	switch kind {
	case Systemd:
		return installSystemd(exe, env)
	case Launchd:
		return installLaunchd(exe, env)
	default:
		return startDetached(exe, env)
	}
}

// Uninstall stops the background router and removes what Install set up.
// A detached router is stopped by the caller, through its pid.
func Uninstall(kind Kind) error {
	switch kind {
	case Systemd:
		p, err := unitPath()
		if err != nil {
			return err
		}
		_ = exec.Command("systemctl", "--user", "disable", "--now", unitName).Run()
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return exec.Command("systemctl", "--user", "daemon-reload").Run()
	case Launchd:
		p, err := plistPath()
		if err != nil {
			return err
		}
		_ = exec.Command("launchctl", "bootout", "gui/"+strconv.Itoa(os.Getuid())+"/"+label).Run()
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	return nil
}

// Executable is the path a background router should run: the ports on
// PATH when it's this same binary — a package manager's stable link
// (/opt/homebrew/bin/ports), which survives an upgrade that moves the
// real file — else this binary's own path.
func Executable() (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	self, _ = filepath.EvalSymlinks(self)
	if onPath, err := exec.LookPath("ports"); err == nil {
		if abs, err := filepath.Abs(onPath); err == nil {
			if real, err := filepath.EvalSymlinks(abs); err == nil && real == self {
				return abs, nil
			}
		}
	}
	return self, nil
}

func unitPath() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "systemd", "user", unitName), nil
}

// unitText is the systemd user unit that runs exe as the router.
func unitText(exe string, env []string) string {
	var b strings.Builder
	b.WriteString("# Written by `ports proxy start`; `ports proxy stop` removes it.\n")
	b.WriteString("[Unit]\nDescription=ports router — every dev server at http://<name>.localhost\n\n")
	b.WriteString("[Service]\n")
	fmt.Fprintf(&b, "ExecStart=%s proxy\n", systemdQuote(exe))
	for _, kv := range env {
		fmt.Fprintf(&b, "Environment=%s\n", systemdQuote(kv))
	}
	// the router itself comes back from a crash; it never restarts a dev server
	b.WriteString("Restart=on-failure\nRestartSec=2\n\n")
	b.WriteString("[Install]\nWantedBy=default.target\n")
	return b.String()
}

func installSystemd(exe string, env []string) error {
	p, err := unitPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(p, []byte(unitText(exe, env)), 0o644); err != nil {
		return err
	}
	if out, err := exec.Command("systemctl", "--user", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %s", strings.TrimSpace(string(out)))
	}
	if out, err := exec.Command("systemctl", "--user", "enable", "--now", unitName).CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl enable: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func systemdQuote(s string) string {
	if !strings.ContainsAny(s, " \t\"\\") {
		return s
	}
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

func plistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", label+".plist"), nil
}

// plistText is the launchd agent that runs exe as the router, writing
// what it prints to log.
func plistText(exe string, env []string, log string) string {
	esc := func(s string) string {
		return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
	}
	var envXML strings.Builder
	if len(env) > 0 {
		envXML.WriteString("  <key>EnvironmentVariables</key>\n  <dict>\n")
		for _, kv := range env {
			k, v, _ := strings.Cut(kv, "=")
			fmt.Fprintf(&envXML, "    <key>%s</key><string>%s</string>\n", esc(k), esc(v))
		}
		envXML.WriteString("  </dict>\n")
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<!-- Written by `+"`ports proxy start`; `ports proxy stop`"+` removes it. -->
<plist version="1.0">
<dict>
  <key>Label</key><string>%s</string>
  <key>ProgramArguments</key>
  <array><string>%s</string><string>proxy</string></array>
%s  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><dict><key>SuccessfulExit</key><false/></dict>
  <key>StandardOutPath</key><string>%s</string>
  <key>StandardErrorPath</key><string>%s</string>
</dict>
</plist>
`, label, esc(exe), envXML.String(), esc(log), esc(log))
}

func installLaunchd(exe string, env []string) error {
	p, err := plistPath()
	if err != nil {
		return err
	}
	log := LogPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(log), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(p, []byte(plistText(exe, env, log)), 0o644); err != nil {
		return err
	}
	domain := "gui/" + strconv.Itoa(os.Getuid())
	_ = exec.Command("launchctl", "bootout", domain+"/"+label).Run() // an older copy
	if out, err := exec.Command("launchctl", "bootstrap", domain, p).CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl bootstrap: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// startDetached runs the router as its own process, outliving this one,
// with what it prints going to the log file.
func startDetached(exe string, env []string) error {
	log := LogPath()
	if err := os.MkdirAll(filepath.Dir(log), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(log, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	cmd := exec.Command(exe, "proxy")
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout, cmd.Stderr = f, f
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
