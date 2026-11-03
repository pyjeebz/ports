package cmd

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/pyjeebz/ports/internal/inspect"
)

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
		s, err := pick(pickRows(), strings.NewReader(c.keys), io.Discard)
		if err != nil || s.Port != c.want {
			t.Errorf("%s: got :%d, %v — want :%d", c.name, s.Port, err, c.want)
		}
	}
}

// Leaving stops nothing: Esc, q, Ctrl-C, a no at the confirm, or input
// that ends.
func TestPickCancels(t *testing.T) {
	for name, keys := range map[string]string{
		"esc": "\x1b", "q": "q", "ctrl-c": "\x03", "no": "\rn", "enter then enter": "\r\r", "eof": "",
	} {
		if _, err := pick(pickRows(), strings.NewReader(keys), io.Discard); !errors.Is(err, errPickCancelled) {
			t.Errorf("%s: got %v, want cancelled", name, err)
		}
	}
}

// What the picker draws: the table's own rows, the cursor, the rows it
// can't stop marked, and the confirm naming what will stop.
func TestPickDraws(t *testing.T) {
	var out strings.Builder
	_, _ = pick(pickRows(), strings.NewReader("j\rn"), &out)
	for _, want := range []string{"PORT", "PROCESS", "› 5432", "Next.js", "(needs sudo)", "stop PostgreSQL (:5432)? [y/N]"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("picker output missing %q:\n%q", want, out.String())
		}
	}
}

func TestPickNothingStoppable(t *testing.T) {
	if _, err := pick(pickRows()[:1], strings.NewReader("\ry"), io.Discard); err == nil || errors.Is(err, errPickCancelled) {
		t.Errorf("a list of only other users' rows should say so, got %v", err)
	}
}
