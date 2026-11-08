package cmd

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/pyjeebz/ports/internal/inspect"
)

// keys feeds the picker one keypress per read, the way a person types —
// an arrow's escape sequence arrives whole, as a terminal sends it. A
// whole string in one read is a paste, and a paste never confirms.
func keys(s string) io.Reader {
	var presses []string
	for s != "" {
		n := 1
		if strings.HasPrefix(s, "\x1b[") && len(s) >= 3 {
			n = 3
		}
		presses = append(presses, s[:n])
		s = s[n:]
	}
	return &pressReader{presses: presses}
}

// pressReader hands over one keypress per Read.
type pressReader struct{ presses []string }

func (p *pressReader) Read(b []byte) (int, error) {
	if len(p.presses) == 0 {
		return 0, io.EOF
	}
	n := copy(b, p.presses[0])
	p.presses = p.presses[1:]
	return n, nil
}

func pickRows() []inspect.Service {
	return []inspect.Service{
		{Port: 2024, Process: "?", User: "root"}, // another user's: can't be stopped
		{Port: 3000, Pid: 1, Process: "node", Framework: "Next.js"},
		{Port: 5432, Pid: 2, Process: "postgres", Framework: "PostgreSQL"},
		{Port: 8000, Pid: 3, Process: "python", Framework: "FastAPI"},
	}
}

// The picker is driven by keys alone: arrows or j/k move, Enter asks, y
// confirms. The cursor starts on the first row it can stop and skips the
// ones it can't.
func TestPickMovesAndConfirms(t *testing.T) {
	for _, c := range []struct {
		name, keys string
		want       uint32
	}{
		{"enter on the first stoppable row", "\ry", 3000},
		{"down arrow", "\x1b[B\ry", 5432},
		{"j twice", "jj\ry", 8000},
		{"past the end stays put", "jjjjjj\ry", 8000},
		{"up never lands on a row it can't stop", "\x1b[A\x1b[A\ry", 3000},
		{"k after j", "jjk\rY", 5432},
	} {
		s, err := pick(pickRows(), keys(c.keys), io.Discard, 0, 0)
		if err != nil || s.Port != c.want {
			t.Errorf("%s: got :%d, %v — want :%d", c.name, s.Port, err, c.want)
		}
	}
}

// Leaving stops nothing: Esc, q, Ctrl-C, a no at the confirm, or input
// that ends.
func TestPickCancels(t *testing.T) {
	for name, k := range map[string]string{
		"esc": "\x1b", "q": "q", "ctrl-c": "\x03", "no": "\rn", "enter then enter": "\r\r", "eof": "",
	} {
		if _, err := pick(pickRows(), keys(k), io.Discard, 0, 0); !errors.Is(err, errPickCancelled) {
			t.Errorf("%s: got %v, want cancelled", name, err)
		}
	}
}

// What the picker draws: the table's own rows, the cursor, the rows it
// can't stop marked, and the confirm naming what will stop.
func TestPickDraws(t *testing.T) {
	var out strings.Builder
	_, _ = pick(pickRows(), keys("j\rn"), &out, 0, 0)
	for _, want := range []string{"PORT", "PROCESS", "› 5432", "Next.js", "(needs sudo)", "stop PostgreSQL (:5432)? [y/N]"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("picker output missing %q:\n%q", want, out.String())
		}
	}
}

func TestPickNothingStoppable(t *testing.T) {
	if _, err := pick(pickRows()[:1], keys("\ry"), io.Discard, 0, 0); err == nil || errors.Is(err, errPickCancelled) {
		t.Errorf("a list of only other users' rows should say so, got %v", err)
	}
}

// A paste that runs on past the Enter — "\ry" in one read — is not a
// yes: the confirm takes only a key pressed after it asks.
func TestPickPasteNeverConfirms(t *testing.T) {
	if _, err := pick(pickRows(), strings.NewReader("\ry"), io.Discard, 0, 0); !errors.Is(err, errPickCancelled) {
		t.Errorf("a pasted \\ry confirmed: got %v, want cancelled", err)
	}
}

// The confirm names every port the target holds — stopping it frees all
// of them.
func TestPickConfirmNamesEveryPort(t *testing.T) {
	rows := append(pickRows(), inspect.Service{Port: 9229, Pid: 1, Process: "node", Framework: "Next.js"})
	var out strings.Builder
	_, _ = pick(rows, keys("\rn"), &out, 0, 0)
	if !strings.Contains(out.String(), "(:3000, :9229)? [y/N]") {
		t.Errorf("confirm should list both ports pid 1 holds:\n%q", out.String())
	}
}

// A list taller than the terminal scrolls in a window, and lines are cut
// to its width.
func TestPickWindows(t *testing.T) {
	var rows []inspect.Service
	for i := range 40 {
		rows = append(rows, inspect.Service{Port: uint32(4000 + i), Pid: int32(100 + i), Process: "python", Framework: "FastAPI", Cmdline: strings.Repeat("x", 200)})
	}
	var out strings.Builder
	s, err := pick(rows, keys(strings.Repeat("j", 30)+"\ry"), &out, 60, 12)
	if err != nil || s.Port != 4030 {
		t.Fatalf("got :%d, %v — want :4030", s.Port, err)
	}
	frames := strings.Split(out.String(), "\x1b[J")
	last := frames[len(frames)-1]
	if strings.Contains(last, "4000") || !strings.Contains(last, "› 4030") || !strings.Contains(last, "↑ ") || !strings.Contains(last, "↓ ") || strings.Contains(last, " 0 more") {
		t.Errorf("the last frame should scroll to the cursor with marks either side:\n%s", last)
	}
	if n := strings.Count(last, "\r\n"); n > 12 {
		t.Errorf("the frame is %d lines, taller than the 12-line terminal", n)
	}
	for _, line := range strings.Split(last, "\r\n") {
		plain := strings.NewReplacer("\x1b[7m", "", "\x1b[2m", "", "\x1b[0m", "").Replace(line)
		if len([]rune(plain)) > 60 {
			t.Errorf("line wider than the terminal: %q", plain)
		}
	}
}

// What was confirmed is what stops. If the machine moved on while the
// picker was open — the process restarted, or another took its port —
// nothing is stopped.
func TestStopPickedStopsOnlyTheConfirmed(t *testing.T) {
	started := time.UnixMilli(1_700_000_000_000)
	chosen := inspect.Service{Port: 8000, Pid: 3, Process: "python", Framework: "FastAPI", Started: started}
	for _, c := range []struct {
		name  string
		now   []inspect.Service
		stops int
	}{
		{"still there", []inspect.Service{chosen, {Port: 8000, Pid: 9, Process: "perl"}}, 1},
		{"another process took the port", []inspect.Service{{Port: 8000, Pid: 9, Process: "perl"}}, 0},
		{"the pid was reused", []inspect.Service{{Port: 8000, Pid: 3, Process: "python", Started: started.Add(time.Second)}}, 0},
		{"it moved to another port", []inspect.Service{{Port: 8001, Pid: 3, Process: "python", Started: started}}, 0},
	} {
		saved := rescan
		rescan = func() ([]inspect.Service, error) { return c.now, nil }
		stops := swapStop(t, func(inspect.Service, bool) error { return nil })
		var err error
		captureStdout(t, func() { err = stopPicked(chosen) })
		rescan = saved
		if len(stops()) != c.stops {
			t.Errorf("%s: %d stops, want %d", c.name, len(stops()), c.stops)
		}
		for _, s := range stops() {
			if s.Pid != 3 {
				t.Errorf("%s: stopped pid %d, not the confirmed pid 3", c.name, s.Pid)
			}
		}
		if c.stops == 0 && (err == nil || !strings.Contains(err.Error(), "changed while you were choosing")) {
			t.Errorf("%s: got %v, want the changed-while-choosing refusal", c.name, err)
		}
	}
}

// A Windows-side listener seen from WSL has no pid here: dimmed, skipped,
// never confirmed with an empty port list.
func TestPickSkipsWindowsRows(t *testing.T) {
	rows := []inspect.Service{
		{Port: 3000, Process: "node.exe", Framework: "Next.js", Windows: true},
		{Port: 8000, Pid: 3, Process: "python", Framework: "FastAPI"},
	}
	var out strings.Builder
	s, err := pick(rows, keys("k\ry"), &out, 0, 0)
	if err != nil || s.Port != 8000 {
		t.Errorf("got :%d, %v — want :8000", s.Port, err)
	}
	if !strings.Contains(out.String(), "(stop it from Windows)") {
		t.Errorf("the Windows row should say where to stop it:\n%q", out.String())
	}
}

// A terminal too short to hold a frame gets no picker, not a garbled one.
func TestPickTooShort(t *testing.T) {
	if _, err := pick(pickRows(), keys("\ry"), io.Discard, 80, 6); err == nil || errors.Is(err, errPickCancelled) {
		t.Errorf("a 6-line terminal should be refused, got %v", err)
	}
}
