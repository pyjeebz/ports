package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

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
// exactly as long as it's on screen — restored on return, on a panic,
// and on SIGTERM/SIGQUIT, which would otherwise exit past the defer and
// leave the shell with no echo.
func pickAndConfirm(rows []inspect.Service) (inspect.Service, error) {
	fd := int(os.Stdin.Fd())
	saved, err := term.MakeRaw(fd)
	if err != nil {
		return inspect.Service{}, err
	}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	done := make(chan struct{})
	go func() {
		select {
		case <-sig:
			_ = term.Restore(fd, saved)
			fmt.Fprintln(os.Stderr, "\r\nports: cancelled — nothing was stopped")
			os.Exit(130)
		case <-done:
		}
	}()
	defer func() {
		signal.Stop(sig)
		close(done)
		_ = term.Restore(fd, saved)
	}()
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width, height = 0, 0
	}
	return pick(rows, os.Stdin, os.Stdout, width, height)
}

// sameTarget reports whether b is the stop target a was: the same
// container, else the same process — pid and start time, since a reused
// pid is another process — else the same systemd unit.
func sameTarget(a, b inspect.Service) bool {
	switch {
	case a.ContainerID != "":
		return a.ContainerID == b.ContainerID
	case a.Known():
		return a.Pid == b.Pid && a.Started.Equal(b.Started)
	default:
		return a.Unit != "" && a.Unit == b.Unit
	}
}

// targetPorts is every port the target holds — stopping it frees all of
// them, so the confirm names all of them.
func targetPorts(chosen inspect.Service, rows []inspect.Service) string {
	var held []uint32
	for _, s := range rows {
		if sameTarget(chosen, s) {
			held = append(held, s.Port)
		}
	}
	return portList(held)
}

// pick draws the rows, moves a cursor with ↑/↓ (or k/j), and on Enter
// asks to confirm, naming the target and every port it holds. Rows
// nothing here can stop — another user's listener with no pid — are
// dimmed and skipped. A list taller than the terminal scrolls in a
// window; lines are cut to its width (0 for either means no limit). It
// reads keys from in and draws to out with raw-mode \r\n line ends, so a
// test can drive it with bytes.
func pick(rows []inspect.Service, in io.Reader, out io.Writer, width, height int) (inspect.Service, error) {
	header, lines := inspect.TableRows(rows)
	stoppable := make([]bool, len(rows))
	first := -1
	for i, s := range rows {
		// a Windows-side listener seen from WSL has no pid here to stop
		stoppable[i] = !s.Unowned() && !s.Windows
		if stoppable[i] && first < 0 {
			first = i
		}
	}
	if first < 0 {
		return inspect.Service{}, errors.New("nothing here can be stopped from this shell — other users' listeners need sudo, Windows ones Windows")
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

	// the window: room for the title, the header, the two scroll marks
	// and the confirm line. Too short a terminal for that gets no picker.
	visible := len(rows)
	if height > 0 && height < 7 {
		return inspect.Service{}, errors.New("the terminal is too short for the picker — name a port or a name, e.g. `ports stop 3000`")
	}
	if height > 0 {
		visible = min(len(rows), height-5)
	}
	top := 0
	clip := func(s string) string {
		if width <= 0 {
			return s
		}
		r := []rune(s)
		if len(r) > width-1 {
			return string(r[:max(0, width-2)]) + "…"
		}
		return s
	}

	r := bufio.NewReader(in)
	drawn := 0
	draw := func() {
		if cur < top {
			top = cur
		}
		if cur >= top+visible {
			top = cur - visible + 1
		}
		if drawn > 0 {
			_, _ = fmt.Fprintf(out, "\x1b[%dA\r\x1b[J", drawn) // back to the top, clear below
		}
		n := 0
		say := func(format string, a ...any) {
			_, _ = fmt.Fprintf(out, format+"\r\n", a...)
			n++
		}
		say("%s", clip("  stop which? ↑/↓ to move, enter to choose, esc to leave"))
		say("%s", clip("  "+header))
		if visible < len(rows) {
			say("\x1b[2m%s\x1b[0m", clip(more("↑", top)))
		}
		for i := top; i < top+visible; i++ {
			switch {
			case i == cur:
				say("\x1b[7m%s\x1b[0m", clip("› "+lines[i]))
			case !stoppable[i] && rows[i].Windows:
				say("\x1b[2m%s\x1b[0m", clip("  "+lines[i]+"  (stop it from Windows)"))
			case !stoppable[i]:
				say("\x1b[2m%s\x1b[0m", clip("  "+lines[i]+"  (needs sudo)"))
			default:
				say("%s", clip("  "+lines[i]))
			}
		}
		if visible < len(rows) {
			say("\x1b[2m%s\x1b[0m", clip(more("↓", len(rows)-top-visible)))
		}
		drawn = n
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
			// whatever arrived with the Enter — a paste that runs on past
			// the newline — is not an answer: only a key pressed after the
			// question confirms. A paste split across reads can still slip
			// a y through; the guard is for the common single-read case
			for r.Buffered() > 0 {
				_, _ = r.ReadByte()
			}
			_, _ = fmt.Fprintf(out, "  stop %s (%s)? [y/N] ", inspect.ProcessName(s), targetPorts(s, rows))
			answer, err := readKey(r)
			_, _ = fmt.Fprint(out, "\r\n")
			if err != nil || answer != "y" {
				return inspect.Service{}, errPickCancelled
			}
			return s, nil
		case "esc", "q", "ctrl-c":
			return inspect.Service{}, errPickCancelled
		}
	}
}

// more is a scroll mark — blank at the end of the list, so the frame
// keeps its height as the window moves.
func more(arrow string, n int) string {
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("  %s %d more", arrow, n)
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
