package inspect

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

var sgr = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func colourRows() []Service {
	return []Service{
		{Port: 3000, Pid: 1, Process: "node", Framework: "Next.js", Started: time.Now().Add(-2 * time.Hour), URL: "http://localhost:3000"},
		{Port: 2024, Process: "?", User: "root"},
		{Port: 5432, Pid: 2, Process: "postgres", Framework: "PostgreSQL", Started: time.Now().Add(-time.Hour)},
		// not valid UTF-8, as a /proc name can be: colour must pass it through
		{Port: 6000, Pid: 3, Process: "bad\xffname", Started: time.Now().Add(-time.Minute)},
	}
}

func withColour(t *testing.T, on bool) {
	t.Helper()
	saved, savedErr, savedLinks := Color, ColorErr, Hyperlinks
	Color, ColorErr, Hyperlinks = on, on, on
	t.Cleanup(func() { Color, ColorErr, Hyperlinks = saved, savedErr, savedLinks })
}

// What a pipe gets, pinned: the table as it read before colour existed.
func TestPlainTableUnchanged(t *testing.T) {
	withColour(t, false)
	rows := []Service{
		{Port: 3000, Pid: 1, Process: "node", Framework: "Next.js", Started: time.Now().Add(-2 * time.Hour), URL: "http://localhost:3000"},
		{Port: 5432, Pid: 2, Process: "postgres", Framework: "PostgreSQL", Started: time.Now().Add(-time.Hour)},
	}
	want := "PORT   PROCESS      PROJECT   UPTIME   URL\n" +
		"3000   Next.js      —         2h       http://localhost:3000\n" +
		"5432   PostgreSQL   —         1h\n"
	if got := captureStdout(t, func() { WriteTable(rows, SortDefault) }); got != want {
		t.Errorf("the plain table changed:\n%q\nwant\n%q", got, want)
	}
}

// Colour only adds escape codes: the same table with them stripped is
// byte-for-byte the plain one, and the plain one has none.
func TestColourOnlyAddsEscapes(t *testing.T) {
	withColour(t, false)
	plain := captureStdout(t, func() { WriteTable(colourRows(), SortDefault) })
	if strings.Contains(plain, "\x1b") {
		t.Fatalf("colour off, yet escapes:\n%q", plain)
	}
	withColour(t, true)
	coloured := captureStdout(t, func() { WriteTable(colourRows(), SortDefault) })
	if coloured == plain {
		t.Fatal("colour on, yet no colour")
	}
	links := regexp.MustCompile("\x1b\\]8;;[^\x1b]*\x1b\\\\")
	withColour(t, false)
	if got := links.ReplaceAllString(sgr.ReplaceAllString(coloured, ""), ""); got != plain {
		t.Errorf("stripped of colour, the table changed:\n%s\nwant\n%s", got, plain)
	}
}

// Uptime is dimmed; another user's row is dimmed whole; the framework
// and the rest stay in the terminal's ink.
func TestColourTable(t *testing.T) {
	withColour(t, true)
	out := captureStdout(t, func() { WriteTable(colourRows(), SortDefault) })
	lines := strings.Split(out, "\n")
	find := func(prefix string) string {
		for _, l := range lines {
			if strings.HasPrefix(sgr.ReplaceAllString(l, ""), prefix) {
				return l
			}
		}
		t.Fatalf("no %s row:\n%s", prefix, out)
		return ""
	}
	if l := find("3000"); !strings.Contains(l, sgrDim+"2h"+sgrReset) || strings.Contains(l, sgrDim+"Next.js") {
		t.Errorf("the 3000 row should dim only its uptime: %q", l)
	}
	if l := find("2024"); !strings.HasPrefix(l, sgrDim) {
		t.Errorf("another user's row should be dimmed whole: %q", l)
	}
	if strings.Contains(lines[0], "\x1b") {
		t.Errorf("the header stays plain: %q", lines[0])
	}
}

func TestMark(t *testing.T) {
	withColour(t, false)
	if got := Mark("✓ stopped", false); got != "✓ stopped" {
		t.Errorf("colour off: %q", got)
	}
	withColour(t, true)
	for line, want := range map[string]string{
		"✓ stopped":  sgrGreen + "✓" + sgrReset + " stopped",
		"⚠ zombie":   sgrYellow + "⚠" + sgrReset + " zombie",
		"✗ failed":   sgrRed + "✗" + sgrReset + " failed",
		"· info":     "· info",
		"no glyph ✓": "no glyph ✓",
	} {
		if got := Mark(line, false); got != want {
			t.Errorf("Mark(%q) = %q, want %q", line, got, want)
		}
	}
	// each stream on its own: stderr may be a terminal while stdout pipes
	Color, ColorErr = false, true
	if Mark("✗ x", false) != "✗ x" || Mark("✗ x", true) == "✗ x" {
		t.Error("Mark should follow the stream it's for")
	}
}
