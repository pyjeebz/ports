package inspect

import "testing"

const netstatOut = "\r\nActive Connections\r\n\r\n  Proto  Local Address          Foreign Address        State           PID\r\n" +
	"  TCP    0.0.0.0:135            0.0.0.0:0              LISTENING       1180\r\n" +
	"  TCP    0.0.0.0:5432           0.0.0.0:0              LISTENING       4321\r\n" +
	"  TCP    [::]:135               [::]:0                 LISTENING       1180\r\n" +
	"  TCP    127.0.0.1:3000         0.0.0.0:0              LISTENING       8765\r\n" +
	"  TCP    192.168.1.5:54321      13.107.42.14:443       ESTABLISHED     9999\r\n"

const tasklistOut = `"System","4","Services","0","20 K"` + "\r\n" +
	`"svchost.exe","1180","Services","0","25,116 K"` + "\r\n" +
	`"postgres.exe","4321","Services","0","41,000 K"` + "\r\n" +
	`"node.exe","8765","Console","1","98,304 K"` + "\r\n"

func TestParseNetstat(t *testing.T) {
	socks := parseNetstat(netstatOut)
	if len(socks) != 3 {
		t.Fatalf("got %d sockets, want 3 (v6 dup collapsed, ESTABLISHED skipped): %+v", len(socks), socks)
	}
	if socks[0].port != 135 || socks[0].pid != 1180 {
		t.Errorf("first sock: %+v", socks[0])
	}
	if socks[2].port != 3000 || socks[2].addr != "127.0.0.1" || socks[2].pid != 8765 {
		t.Errorf("localhost sock: %+v", socks[2])
	}
}

func TestParseTasklist(t *testing.T) {
	names := parseTasklist(tasklistOut)
	if names[4321] != "postgres.exe" || names[8765] != "node.exe" {
		t.Errorf("names: %v", names)
	}
}

// The state column localizes with the Windows display language; listeners
// must still be found positionally when it does.
const netstatGermanOut = "\r\nAktive Verbindungen\r\n\r\n  Protokoll  Lokale Adresse         Remoteadresse          Status           PID\r\n" +
	"  TCP    0.0.0.0:135            0.0.0.0:0              ABHÖREN          1180\r\n" +
	"  TCP    127.0.0.1:3000         127.0.0.1:54321        HERGESTELLT      8765\r\n" +
	"  TCP    0.0.0.0:8080           0.0.0.0:0              ABHÖREN          2468\r\n"

func TestParseNetstatGermanLocale(t *testing.T) {
	socks := parseNetstat(netstatGermanOut)
	if len(socks) != 2 {
		t.Fatalf("got %d sockets, want 2 (ABHÖREN rows, HERGESTELLT skipped): %+v", len(socks), socks)
	}
	if socks[1].port != 8080 || socks[1].pid != 2468 {
		t.Errorf("second sock: %+v", socks[1])
	}
}

// A Windows row reads through the same framework table as its Linux
// twin: the image name stands in for both exe and cmdline, so a bin rule
// (next, not just node.exe) fires for Windows rows too.
func TestWindowsListenersMatchFrameworkRules(t *testing.T) {
	out := "\r\nActive Connections\r\n\r\n  Proto  Local Address          Foreign Address        State           PID\r\n" +
		"  TCP    0.0.0.0:3000           0.0.0.0:0              LISTENING       8765\r\n"
	tl := `"next.exe","8765","Console","1","120,000 K"` + "\r\n"
	old := runWin
	runWin = func(exe string, _ ...string) ([]byte, error) {
		if exe == "netstat.exe" {
			return []byte(out), nil
		}
		return []byte(tl), nil
	}
	t.Cleanup(func() { runWin = old })

	rows := windowsListeners()
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1: %+v", len(rows), rows)
	}
	if rows[0].Framework != "Next.js" {
		t.Errorf("next.exe framework = %q, want Next.js", rows[0].Framework)
	}
}

func TestWindowsListeners(t *testing.T) {
	old := runWin
	runWin = func(exe string, _ ...string) ([]byte, error) {
		if exe == "netstat.exe" {
			return []byte(netstatOut), nil
		}
		return []byte(tasklistOut), nil
	}
	t.Cleanup(func() { runWin = old })

	rows := windowsListeners()
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2 (svchost filtered): %+v", len(rows), rows)
	}
	pg := rows[0]
	if pg.Port != 5432 || !pg.Windows || pg.WindowsPid != 4321 || pg.Pid != 0 {
		t.Errorf("postgres row: %+v", pg)
	}
	if pg.Process != "postgres" || pg.Framework != "PostgreSQL" {
		t.Errorf("postgres naming: process %q framework %q", pg.Process, pg.Framework)
	}
	if node := rows[1]; node.Port != 3000 || node.Framework != "Node.js" {
		t.Errorf("node row: %+v", node)
	}
}

// pid 0 is Windows declining to name an owner; tasklist labels it
// "System Idle Process". Those rows can be neither named nor stopped.
func TestWindowsListenersSkipPidZero(t *testing.T) {
	out := "\r\nActive Connections\r\n\r\n  Proto  Local Address          Foreign Address        State           PID\r\n" +
		"  TCP    127.0.0.1:50473        127.0.0.1:0            LISTENING       0\r\n" +
		"  TCP    0.0.0.0:3000           0.0.0.0:0              LISTENING       8765\r\n"
	tl := `"System Idle Process","0","Services","0","8 K"` + "\r\n" +
		`"node.exe","8765","Console","1","98,304 K"` + "\r\n"
	old := runWin
	runWin = func(exe string, _ ...string) ([]byte, error) {
		if exe == "netstat.exe" {
			return []byte(out), nil
		}
		return []byte(tl), nil
	}
	t.Cleanup(func() { runWin = old })

	rows := windowsListeners()
	if len(rows) != 1 || rows[0].Port != 3000 {
		t.Fatalf("got %+v, want only the node row", rows)
	}
}
