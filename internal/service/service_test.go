package service

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestUnitText(t *testing.T) {
	unit := unitText("/opt/my tools/ports", []string{"PORTS_STATE_DIR=/tmp/st"})
	for _, want := range []string{
		`ExecStart="/opt/my tools/ports" proxy`,
		"Environment=PORTS_STATE_DIR=/tmp/st",
		"Restart=on-failure",
		"WantedBy=default.target",
	} {
		if !strings.Contains(unit, want) {
			t.Errorf("unit lacks %q:\n%s", want, unit)
		}
	}
}

func TestPlistText(t *testing.T) {
	plist := plistText("/opt/homebrew/bin/ports", []string{"PORTS_STATE_DIR=/a&b"}, "/tmp/proxy.log")
	for _, want := range []string{
		"<string>/opt/homebrew/bin/ports</string><string>proxy</string>",
		"<key>PORTS_STATE_DIR</key><string>/a&amp;b</string>",
		"<key>RunAtLoad</key><true/>",
		"<key>StandardOutPath</key><string>/tmp/proxy.log</string>",
	} {
		if !strings.Contains(plist, want) {
			t.Errorf("plist lacks %q:\n%s", want, plist)
		}
	}
	// an XML comment may not hold "--"
	if i := strings.Index(plist, "<!--"); i >= 0 {
		body := plist[i+4 : strings.Index(plist, "-->")]
		if strings.Contains(body, "--") {
			t.Errorf("the plist's comment holds --: %q", body)
		}
	}
}

func TestStartDetached(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a shell script stands in for the router")
	}
	t.Setenv("PORTS_STATE_DIR", t.TempDir())
	dir := t.TempDir()
	exe := filepath.Join(dir, "fake-ports")
	script := "#!/bin/sh\necho \"started with $1 in $PORTS_STATE_DIR\"\n"
	if err := os.WriteFile(exe, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := startDetached(exe, []string{"PORTS_STATE_DIR=" + os.Getenv("PORTS_STATE_DIR")}); err != nil {
		t.Fatal(err)
	}
	want := "started with proxy in " + os.Getenv("PORTS_STATE_DIR")
	for range 50 {
		if data, _ := os.ReadFile(LogPath()); strings.Contains(string(data), want) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	data, _ := os.ReadFile(LogPath())
	t.Errorf("log %q, want %q", data, want)
}
