package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Every `ports …` line the README and the docs show in a code block must
// be a command this binary accepts: a real command, flags it takes, the
// right number of arguments. Docs drift one rename at a time; this is
// the check that catches it, without running a scan.
func TestDocumentedCommandsParse(t *testing.T) {
	files, _ := filepath.Glob("../web/content/docs/*.mdx")
	files = append(files, "../README.md")
	checked := 0
	for _, file := range files {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range documentedCommands(string(b)) {
			checked++
			if err := parseDocumented(line); err != nil {
				t.Errorf("%s: `%s`: %v", filepath.Base(file), line, err)
			}
		}
	}
	if checked < 20 {
		t.Fatalf("found only %d documented commands — the extraction broke", checked)
	}
}

var (
	fence  = regexp.MustCompile("^\\s*```")
	cmdRe  = regexp.MustCompile(`^\s*(?:\$ )?(ports(?: .*)?)$`)
	breaks = regexp.MustCompile(`\s(?:&&|\|\||\||>|;|#)\s?`)
)

// documentedCommands is each `ports …` invocation inside a fenced code
// block, cut at the first shell operator so `ports a && ports b` yields
// both. Lines with a <placeholder> are usage, not an invocation.
func documentedCommands(doc string) []string {
	var out []string
	in := false
	for _, line := range strings.Split(doc, "\n") {
		if fence.MatchString(line) {
			in = !in
			continue
		}
		if !in {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "$ "))
		for _, part := range breaks.Split(line, -1) {
			part = strings.Trim(strings.TrimSpace(part), "()&")
			if m := cmdRe.FindStringSubmatch(part); m != nil && !strings.Contains(m[1], "<") {
				out = append(out, strings.TrimSpace(m[1]))
			}
		}
	}
	return out
}

// parseDocumented resolves a line against the command tree the way the
// binary would — command, flags, argument count — without setting a flag.
func parseDocumented(line string) error {
	words := strings.Fields(line)[1:] // drop "ports"
	c, rest, err := rootCmd.Find(words)
	if err != nil {
		return err
	}
	var positional []string
	for i := 0; i < len(rest); i++ {
		w := rest[i]
		if !strings.HasPrefix(w, "-") || w == "-" {
			positional = append(positional, w)
			continue
		}
		name, _, hasValue := strings.Cut(strings.TrimLeft(w, "-"), "=")
		f := c.Flags().Lookup(name)
		if !strings.HasPrefix(w, "--") && len(name) == 1 {
			f = c.Flags().ShorthandLookup(name)
		}
		if f == nil {
			f = c.InheritedFlags().Lookup(name)
		}
		if f == nil {
			return errString("`ports " + c.Name() + "` has no flag " + w)
		}
		if f.Value.Type() != "bool" && !hasValue {
			if i+1 >= len(rest) {
				return errString(w + " needs a value")
			}
			i++ // the flag's value
		}
	}
	if c.Args != nil {
		if err := c.Args(c, positional); err != nil {
			return err
		}
	} else if c == rootCmd && len(positional) > 0 {
		return errString("unknown command " + positional[0])
	}
	return nil
}

// The check catches each kind of drift a rename leaves behind — and
// still passes the forms the docs really use.
func TestParseDocumented(t *testing.T) {
	for _, bad := range []string{
		"ports frees",             // a command that doesn't exist
		"ports kill --watch shop", // a flag that belongs to another command
		"ports wait 3000 --htttp", // a misspelled flag
		"ports free",              // a missing argument
		"ports free 3000 8000",    // one too many
		"ports --format",          // a flag missing its value
	} {
		if parseDocumented(bad) == nil {
			t.Errorf("`%s` should fail the docs check", bad)
		}
	}
	for _, good := range []string{
		"ports", "ports --json", "ports -w", "ports find postgres --sort uptime",
		"ports wait 3000 --http --timeout 60s", "ports free 3000 --expect nextjs --json",
		"ports --format snapshot", "ports diff before.json", "ports ps",
	} {
		if err := parseDocumented(good); err != nil {
			t.Errorf("`%s` should pass the docs check: %v", good, err)
		}
	}
	got := documentedCommands("text\n```sh\n$ ports free 3000 && ports wait 3000 --gone\nports here --port 3000 --json || (npm run dev &)\nports <version>\n```\nports outside a fence\n")
	want := []string{"ports free 3000", "ports wait 3000 --gone", "ports here --port 3000 --json"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("documentedCommands = %q, want %q", got, want)
	}
}
