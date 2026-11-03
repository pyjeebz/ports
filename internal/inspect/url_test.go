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
		{Service{Port: 4000, Framework: "Phoenix", Addr: "fe80::1%eth0"}, "http://[fe80::1%25eth0]:4000"},
		{Service{Port: 8787, Framework: "Wrangler", Addr: "127.0.0.1"}, ""}, // several processes, random ports
		{Service{Port: 1323, Framework: "Echo", Addr: "0.0.0.0"}, ""},       // a bare word, too weak to link
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

	Hyperlinks = false
	plain := captureStdout(t, func() { WriteTable(rows[1:], SortPort) })
	if strings.Contains(plain, "URL") {
		t.Errorf("no URL column when no row has one:\n%s", plain)
	}
}

// The framework label belongs to the process, so one dev server can hold
// several ports: only the app's gets a URL. A --port/-p on the command
// line names it; else the lowest port that isn't a known side port (the
// Node inspector, Java's debug wire, debugpy, LiveReload). HTTPS chosen
// on the command line gets no http:// URL at all.
func TestFillURLs(t *testing.T) {
	urls := func(rows []Service) map[uint32]string {
		fillURLs(rows)
		got := map[uint32]string{}
		for _, r := range rows {
			got[r.Port] = r.URL
		}
		return got
	}
	next := func(port uint32, cmd string) Service {
		return Service{Port: port, Pid: 10, Framework: "Next.js", Addr: "0.0.0.0", Cmdline: cmd}
	}

	got := urls([]Service{next(9229, "node --inspect next dev"), next(3000, "node --inspect next dev")})
	if got[3000] != "http://localhost:3000" || got[9229] != "" {
		t.Errorf("--inspect: the app gets the URL, the debugger none: %v", got)
	}
	got = urls([]Service{next(3001, "next dev -p 4000"), next(4000, "next dev -p 4000")})
	if got[4000] == "" || got[3001] != "" {
		t.Errorf("-p names the app port: %v", got)
	}
	// next dev's parent names -p 3458 but holds only its inspector; the
	// server child holds the app port and an inspector of its own
	got = urls([]Service{
		{Port: 9239, Pid: 40, Framework: "Next.js", Addr: "127.0.0.1", Cmdline: "node next dev -p 3458"},
		{Port: 3458, Pid: 41, Framework: "Next.js", Addr: "::", Cmdline: "next-server (v16)"},
		{Port: 9240, Pid: 41, Framework: "Next.js", Addr: "127.0.0.1", Cmdline: "next-server (v16)"},
	})
	if got[3458] == "" || got[9239] != "" || got[9240] != "" {
		t.Errorf("only the app port gets a URL, across parent and child: %v", got)
	}
	got = urls([]Service{next(3000, "next dev --port=3000")})
	if got[3000] == "" {
		t.Errorf("--port=N names the app port: %v", got)
	}
	spring := []Service{
		{Port: 5005, Pid: 20, Framework: "Spring Boot", Addr: "0.0.0.0", Cmdline: "java -agentlib:jdwp=transport=dt_socket,address=*:5005 -jar app.jar"},
		{Port: 8080, Pid: 20, Framework: "Spring Boot", Addr: "0.0.0.0", Cmdline: "java -agentlib:jdwp=transport=dt_socket,address=*:5005 -jar app.jar"},
	}
	if got := urls(spring); got[8080] == "" || got[5005] != "" {
		t.Errorf("the debug wire port is not the app: %v", got)
	}
	for _, cmd := range []string{"uvicorn app:app --ssl-keyfile k.pem", "next dev --experimental-https", "ng serve --ssl", "flask run --cert adhoc"} {
		if got := urls([]Service{{Port: 8000, Pid: 30, Framework: "FastAPI", Addr: "0.0.0.0", Cmdline: cmd}}); got[8000] != "" {
			t.Errorf("%q chose HTTPS: no http:// URL, got %v", cmd, got)
		}
	}
	// two processes, two apps: each keeps its own URL
	two := []Service{next(3000, "next dev"), {Port: 5173, Pid: 11, Framework: "Vite", Addr: "::"}}
	if got := urls(two); got[3000] == "" || got[5173] == "" {
		t.Errorf("each process keeps its app URL: %v", got)
	}
}
