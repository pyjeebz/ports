package inspect

import (
	"strings"
	"unicode/utf8"
)

// Color and ColorErr turn on colour for stdout and stderr. The command
// layer sets each only for a terminal that renders it, without NO_COLOR;
// a pipe, a file and every machine format get the plain bytes.
var Color, ColorErr bool

const (
	sgrDim    = "\x1b[2m"
	sgrRed    = "\x1b[31m"
	sgrGreen  = "\x1b[32m"
	sgrYellow = "\x1b[33m"
	sgrReset  = "\x1b[0m"
)

// marks are the status glyphs and their colours: done, worth a look,
// failed. Information (·) stays in the terminal's ink.
var marks = []struct{ glyph, sgr string }{
	{"✓", sgrGreen}, {"⚠", sgrYellow}, {"✗", sgrRed},
}

// Mark colours a line's leading status glyph, for the stream it's bound
// for — the rest of the line stays in the terminal's own ink.
func Mark(line string, stderr bool) string {
	on := Color
	if stderr {
		on = ColorErr
	}
	if !on {
		return line
	}
	for _, m := range marks {
		if rest, ok := strings.CutPrefix(line, m.glyph); ok {
			return m.sgr + m.glyph + sgrReset + rest
		}
	}
	return line
}

// colourTable colours the aligned table after tabwriter has measured it
// — escape bytes in a cell would throw the columns out. Uptime is dimmed;
// so is the whole row of a listener that belongs to another user, the
// rows there's least to do with. lines[0] is the header; lines[i+1] is
// rows[i]. Columns are found the way tabwriter counts them, one per
// UTF-8 rune (an invalid byte counts one), and cut by byte offset so the
// line's bytes pass through untouched.
func colourTable(lines []string, rows []Service) {
	if !Color || len(lines) == 0 {
		return
	}
	start := column(lines[0], "UPTIME")
	end := column(lines[0], "URL")
	for i, s := range rows {
		line := lines[i+1]
		if s.Unowned() {
			lines[i+1] = sgrDim + line + sgrReset
			continue
		}
		from := byteAt(line, start)
		if start < 0 || from >= len(line) {
			continue
		}
		to := len(line)
		if end > start {
			to = byteAt(line, end)
		}
		cell := strings.TrimRight(line[from:to], " ")
		lines[i+1] = line[:from] + sgrDim + cell + sgrReset + line[from+len(cell):]
	}
}

// column is the column (in runes) where word starts in the header, or -1.
func column(header, word string) int {
	i := strings.Index(header, word)
	if i < 0 {
		return -1
	}
	return utf8.RuneCountInString(header[:i])
}

// byteAt is the byte offset of the n-th rune of s, or len(s) past its end.
func byteAt(s string, n int) int {
	for i := range s {
		if n == 0 {
			return i
		}
		n--
	}
	return len(s)
}
