package inspect

import (
	"strings"
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
// failed, and plain information.
var marks = []struct{ glyph, sgr string }{
	{"✓", sgrGreen}, {"⚠", sgrYellow}, {"✗", sgrRed}, {"·", sgrDim},
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
// rows[i]. Columns are found in runes, as tabwriter counts them.
func colourTable(lines []string, rows []Service) {
	if !Color || len(lines) == 0 {
		return
	}
	header := []rune(lines[0])
	start := runeIndex(header, "UPTIME")
	end := runeIndex(header, "URL")
	for i, s := range rows {
		line := lines[i+1]
		if s.Unowned() {
			lines[i+1] = sgrDim + line + sgrReset
			continue
		}
		r := []rune(line)
		if start < 0 || start >= len(r) {
			continue
		}
		stop := len(r)
		if end > start && end < len(r) {
			stop = end
		}
		cell := strings.TrimRight(string(r[start:stop]), " ")
		pad := string(r[start:stop])[len(cell):]
		lines[i+1] = string(r[:start]) + sgrDim + cell + sgrReset + pad + string(r[stop:])
	}
}

func runeIndex(r []rune, word string) int {
	i := strings.Index(string(r), word)
	if i < 0 {
		return -1
	}
	return len([]rune(string(r)[:i]))
}
