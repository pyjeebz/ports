package inspect

import (
	"strings"
	"testing"
)

// A known web dev server gets the address it answers on from here; any
// other row — a database, an unknown process, a Windows-side listener —
// gets none, because the listing never sends a request to find out.
func TestLocalURL(t *testing.T) {
	for _, c := range []struct {
		s    Service
		want string
	}{
		{Service{Port: 3000, Framework: "Next.js", Addr: "0.0.0.0"}, "http://localhost:3000"},
		{Service{Port: 5173, Framework: "Vite", Addr: "::1"}, "http://localhost:5173"},
		{Service{Port: 8000, Framework: "FastAPI", Addr: "127.0.0.1"}, "http://localhost:8000"},
		{Service{Port: 8000, Framework: "Django", Addr: "192.168.1.5"}, "http://192.168.1.5:8000"},
		{Service{Port: 4000, Framework: "Phoenix", Addr: "fe80::1"}, "http://[fe80::1]:4000"},
		{Service{Port: 5432, Framework: "PostgreSQL", Addr: "0.0.0.0"}, ""},
		{Service{Port: 443, Framework: "nginx", Addr: "0.0.0.0"}, ""}, // may be HTTPS: not guessed
		{Service{Port: 8791, Process: "python3.13", Addr: "0.0.0.0"}, ""},
		{Service{Port: 3000, Framework: "Next.js", Windows: true}, ""},
	} {
		if got := localURL(c.s); got != c.want {
			t.Errorf("localURL(%s on %s) = %q, want %q", c.s.Framework, c.s.Addr, got, c.want)
		}
	}
}

// The URL column shows only when a row has a URL, goes last, leaves no
// trailing spaces on rows without one, and is a link only when enabled.
func TestTableURLColumn(t *testing.T) {
	rows := []Service{
		{Port: 3000, Pid: 1, Process: "node", Framework: "Next.js", URL: "http://localhost:3000"},
		{Port: 5432, Pid: 2, Process: "postgres", Framework: "PostgreSQL"},
	}
	out := captureStdout(t, func() { WriteTable(rows, SortPort) })
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if !strings.HasSuffix(lines[0], "UPTIME   URL") || !strings.HasSuffix(lines[1], "http://localhost:3000") {
		t.Errorf("URL column missing or out of place:\n%s", out)
	}
	for _, l := range lines {
		if strings.HasSuffix(l, " ") {
			t.Errorf("trailing space on %q", l)
		}
	}
	if strings.Contains(out, "\x1b") {
		t.Errorf("no links unless enabled:\n%q", out)
	}

	Hyperlinks = true
	defer func() { Hyperlinks = false }()
	linked := captureStdout(t, func() { WriteTable(rows, SortPort) })
	if !strings.Contains(linked, "\x1b]8;;http://localhost:3000\x1b\\http://localhost:3000\x1b]8;;\x1b\\") {
		t.Errorf("enabled links should wrap the URL in OSC 8:\n%q", linked)
	}

	plain := captureStdout(t, func() { WriteTable(rows[1:], SortPort) })
	if strings.Contains(plain, "URL") {
		t.Errorf("no URL column when no row has one:\n%s", plain)
	}
}
