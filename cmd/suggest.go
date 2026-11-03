package cmd

import (
	"fmt"
	"slices"
	"strings"

	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/spf13/cobra"
)

// suggestName is the name on this machine a mistyped query most likely
// meant — a project, framework, process, container, or unit — or "" when
// nothing is close enough to be worth saying. Close means a few edits
// away from the whole name or from its start, so "pyhton" finds
// "python3.13" and "shp" finds "shop"; a tie between two names says
// nothing rather than guess.
func suggestName(q string, services []inspect.Service, accept func(inspect.Service, string) bool) string {
	q = strings.ToLower(q)
	// a third of the query in edits: one for "shp", two for "pyhton",
	// none for a letter or two — those are a few edits from anything
	limit := len([]rune(q)) / 3
	// score ranks by edits, and at equal edits a whole name over the start
	// of a longer one — "shp" means shop before shop-db
	best, bestScore, tied := "", 2*(limit+1), false
	seen := map[string]bool{}
	for _, s := range services {
		for _, name := range []string{s.ProjectName, s.Framework, s.Process, s.Container, s.ComposeService, s.Unit} {
			if name == "" || name == "?" || seen[name] || (accept != nil && !accept(s, name)) {
				continue
			}
			seen[name] = true
			d, viaStart := nameDistance(q, strings.ToLower(name))
			if d > limit {
				continue
			}
			score := 2 * d
			if viaStart {
				score++
			}
			switch {
			case score < bestScore:
				best, bestScore, tied = name, score, false
			case score == bestScore:
				tied = true
			}
		}
	}
	if best == "" || tied {
		return ""
	}
	return best
}

// nameDistance is the edit distance from q to name, or to name's start
// when name runs on past q (python3.13 for "pyhton"); viaStart says the
// start was the closer of the two.
func nameDistance(q, name string) (d int, viaStart bool) {
	d = editDistance(q, name)
	n := []rune(name)
	for _, k := range []int{len([]rune(q)) - 1, len([]rune(q)), len([]rune(q)) + 1} {
		if k > 0 && k < len(n) {
			if e := editDistance(q, string(n[:k])); e < d {
				d, viaStart = e, true
			}
		}
	}
	return d, viaStart
}

// editDistance counts the insertions, deletions, substitutions and
// adjacent swaps that turn a into b — a swap is one typo, not two.
func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	d := make([][]int, len(ra)+1)
	for i := range d {
		d[i] = make([]int, len(rb)+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}
	for i := 1; i <= len(ra); i++ {
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			d[i][j] = min(d[i-1][j]+1, d[i][j-1]+1, d[i-1][j-1]+cost)
			if i > 1 && j > 1 && ra[i-1] == rb[j-2] && ra[i-2] == rb[j-1] {
				d[i][j] = min(d[i][j], d[i-2][j-2]+1)
			}
		}
	}
	return d[len(ra)][len(rb)]
}

// nothingMatching is the "nothing matching" line for find and kill, with the
// nearest name on the machine when there is one. accept, when set, keeps
// only names the command would act on — kill matches names exactly and
// never a docker row's docker-proxy, so it must never suggest one.
func nothingMatching(q string, services []inspect.Service, accept func(inspect.Service, string) bool, otherwise string) string {
	if s := suggestName(q, services, accept); s != "" {
		return fmt.Sprintf("nothing matching %q — did you mean %q?", q, s)
	}
	return fmt.Sprintf("nothing matching %q%s", q, otherwise)
}

// argNouns says what each command's argument is, for the messages a
// missing or extra one gets: what it needs, and an example to copy.
var argNouns = map[string]struct{ noun, bare, example string }{
	"find":       {"a name", "name", "ports find postgres"},
	"kill":       {"a name", "name", "ports kill shop"},
	"free":       {"a port", "port", "ports free 3000"},
	"wait":       {"a port or a name", "port or name", "ports wait 3000"},
	"doctor":     {"a port", "port", "ports doctor 3000"},
	"here":       {"a directory", "directory", "ports here ~/projects/shop"},
	"diff":       {"a snapshot file", "snapshot file", "ports diff before.json"},
	"completion": {"a shell", "shell", "ports completion zsh"},
}

// argsError rewrites cobra's argument-count errors ("accepts 1 arg(s),
// received 0") in ports' voice: what the command needs, with an example.
// required says whether the command's own validator refuses zero
// arguments — asked of the validator, not guessed from the Use line.
func argsError(c *cobra.Command, args []string, required bool, err error) error {
	n, ok := argNouns[c.Name()]
	if !ok {
		return err
	}
	optional := !required
	switch {
	case len(args) == 0 && !optional:
		return fmt.Errorf("`ports %s` needs %s — e.g. `%s`", c.Name(), n.noun, n.example)
	case len(args) > 1:
		takes := "takes one"
		if optional {
			takes = "takes at most one"
		}
		return fmt.Errorf("`ports %s` %s %s — got %d: %s", c.Name(), takes, n.bare, len(args), strings.Join(args, " "))
	case len(c.ValidArgs) > 0 && len(args) == 1 && !slices.Contains(c.ValidArgs, args[0]):
		return fmt.Errorf("unknown %s %q — %s", n.bare, args[0], orList(c.ValidArgs))
	}
	return err
}

func orList(items []string) string {
	if len(items) < 2 {
		return strings.Join(items, "")
	}
	return strings.Join(items[:len(items)-1], ", ") + ", or " + items[len(items)-1]
}

// suggestCommand is the command a mistyped one most likely meant — a
// typo or a swap within two edits ("frees", "wiat", "dctor"), fewer
// for a short word — or a prefix of two letters or more ("kil") — or "".
// cobra's own suggestions only catch prefixes here.
func suggestCommand(root *cobra.Command, typed string) string {
	typed = strings.ToLower(typed)
	// a short word is a few edits from anything: allow at most half its
	// length, and never more than two
	best, bestDist, tied := "", min(2, len([]rune(typed))/2)+1, false
	for _, c := range root.Commands() {
		if !c.IsAvailableCommand() {
			continue
		}
		for _, name := range append([]string{c.Name()}, c.Aliases...) {
			d := editDistance(typed, name)
			if len(typed) >= 2 && strings.HasPrefix(name, typed) {
				d = 0
			}
			switch {
			case d < bestDist:
				best, bestDist, tied = name, d, false
			case d == bestDist && name != best:
				tied = true
			}
		}
	}
	if tied {
		return "" // "lill" is as near kill as list: say nothing rather than guess
	}
	return best
}
