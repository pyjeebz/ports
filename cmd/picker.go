package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pyjeebz/ports/internal/inspect"
	"golang.org/x/term"
)

// errPickCancelled is a picker the user left — Esc, q, Ctrl-C, or a no
// at the confirm. Nothing was stopped; the exit is 130, like an
// interrupted wait.
var errPickCancelled = errors.New("cancelled — nothing was stopped")

// interactive reports whether stop may ask: stdin and stdout are both a
// terminal. Scripts, pipes and agents never see the picker.
func interactive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

// pickAndConfirm runs the picker on the real terminal, in raw mode for
// exactly as long as it's on screen.
func pickAndConfirm(rows []inspect.Service) (inspect.Service, error) {
	fd := int(os.Stdin.Fd())
	saved, err := term.MakeRaw(fd)
	if err != nil {
		return inspect.Service{}, err
	}
	defer func() { _ = term.Restore(fd, saved) }()
	return pick(rows, os.Stdin, os.Stdout)
}

// pick draws the rows, moves a cursor with ↑/↓ (or k/j), and on Enter
// asks to confirm. Rows nothing here can stop — another user's
// listener with no pid — are shown dimmed and the cursor skips them.
// It reads keys from in and draws to out, raw-mode \r\n line ends, so a
// test can drive it with bytes.
func pick(rows []inspect.Service, in io.Reader, out io.Writer) (inspect.Service, error) {
	header, lines := inspect.TableRows(rows)
	stoppable := make([]bool, len(rows))
	first := -1
	for i, s := range rows {
		stoppable[i] = !s.Unowned()
		if stoppable[i] && first < 0 {
			first = i
		}
	}
	if first < 0 {
		return inspect.Service{}, errors.New("nothing here can be stopped without sudo")
	}
	cur := first
	move := func(step int) {
		for i := cur + step; i >= 0 && i < len(rows); i += step {
			if stoppable[i] {
				cur = i
				return
			}
		}
	}

	r := bufio.NewReader(in)
	drawn := 0
	draw := func() {
		if drawn > 0 {
			_, _ = fmt.Fprintf(out, "\x1b[%dA\r\x1b[J", drawn) // back to the top, clear below
		}
		_, _ = fmt.Fprint(out, "  stop which? ↑/↓ to move, enter to choose, esc to leave\r\n")
		_, _ = fmt.Fprintf(out, "  %s\r\n", header)
		for i, l := range lines {
			switch {
			case i == cur:
				_, _ = fmt.Fprintf(out, "\x1b[7m› %s\x1b[0m\r\n", l)
			case !stoppable[i]:
				_, _ = fmt.Fprintf(out, "\x1b[2m  %s  (needs sudo)\x1b[0m\r\n", l)
			default:
				_, _ = fmt.Fprintf(out, "  %s\r\n", l)
			}
		}
		drawn = len(lines) + 2
	}

	for {
		draw()
		key, err := readKey(r)
		if err != nil {
			return inspect.Service{}, errPickCancelled
		}
		switch key {
		case "up", "k":
			move(-1)
		case "down", "j":
			move(1)
		case "enter":
			s := rows[cur]
			_, _ = fmt.Fprintf(out, "  stop %s (:%d)? [y/N] ", inspect.ProcessName(s), s.Port)
			answer, err := readKey(r)
			_, _ = fmt.Fprint(out, "\r\n")
			if err != nil || (answer != "y" && answer != "Y") {
				return inspect.Service{}, errPickCancelled
			}
			return s, nil
		case "esc", "q", "ctrl-c":
			return inspect.Service{}, errPickCancelled
		}
	}
}

// readKey reads one keypress: the arrow escapes, Enter, Esc, Ctrl-C, or a
// single character.
func readKey(r *bufio.Reader) (string, error) {
	b, err := r.ReadByte()
	if err != nil {
		return "", err
	}
	switch b {
	case '\r', '\n':
		return "enter", nil
	case 3:
		return "ctrl-c", nil
	case 0x1b:
		if r.Buffered() == 0 {
			return "esc", nil
		}
		next, _ := r.ReadByte()
		if next != '[' && next != 'O' {
			return "esc", nil
		}
		code, _ := r.ReadByte()
		switch code {
		case 'A':
			return "up", nil
		case 'B':
			return "down", nil
		}
		return "", nil
	}
	return strings.ToLower(string(b)), nil
}
