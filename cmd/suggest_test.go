package cmd

import (
	"strings"
	"testing"

	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/spf13/cobra"
)

func TestEditDistance(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
	}{
		{"", "", 0}, {"shop", "shop", 0}, {"shp", "shop", 1},
		{"pyhton", "python", 1}, // a swap is one typo
		{"frees", "free", 1}, {"kitten", "sitting", 3}, {"日本", "日", 1},
	} {
		if got := editDistance(c.a, c.b); got != c.want {
			t.Errorf("editDistance(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

// A mistyped name suggests the name on the machine it most likely meant;
// a short query, a far one, or a tie suggests nothing rather than guess.
func TestSuggestName(t *testing.T) {
	services := []inspect.Service{
		{Port: 3000, Process: "node", Framework: "Next.js", ProjectName: "shop"},
		{Port: 8791, Process: "python3.13", ProjectName: "admin"},
		{Port: 5432, Process: "docker-proxy", Container: "shop-db", Framework: "PostgreSQL"},
		{Port: 2024, Process: "?"},
	}
	for q, want := range map[string]string{
		"shp":      "shop",
		"pyhton":   "python3.13",
		"SHP":      "shop",
		"postgrse": "PostgreSQL",
		"x":        "", // a letter is a few edits from anything
		"ab":       "",
		"zzzz":     "",
		"?":        "", // the unknown process is not a name
	} {
		if got := suggestName(q, services, nil); got != want {
			t.Errorf("suggestName(%q) = %q, want %q", q, got, want)
		}
	}
	tied := []inspect.Service{{Process: "abd"}, {Process: "abe"}}

	// kill only acts on names MatchesName accepts: a docker row's
	// docker-proxy is never one, its container and compose service are
	docker := []inspect.Service{{Port: 5432, Process: "docker-proxy", Container: "shop-db", ComposeService: "postgres"}}
	if got := suggestName("docker-prox", docker, inspect.Service.MatchesName); got != "" {
		t.Errorf("kill must not suggest docker-proxy, got %q", got)
	}
	if got := suggestName("shop-bd", docker, inspect.Service.MatchesName); got != "shop-db" {
		t.Errorf("kill should suggest the container, got %q", got)
	}
	if got := suggestName("postgrse", docker, inspect.Service.MatchesName); got != "postgres" {
		t.Errorf("kill should suggest the compose service, got %q", got)
	}
	if got := suggestName("abc", tied, nil); got != "" {
		t.Errorf("a tie should suggest nothing, got %q", got)
	}
}

func TestSuggestCommand(t *testing.T) {
	for typed, want := range map[string]string{
		"frees": "free", "wiat": "wait", "dctor": "doctor", "kil": "kill", "fr": "free",
		"l": "", "h": "", "xyz": "",
		"lill": "kill", // one edit from kill, two from list
	} {
		if got := suggestCommand(rootCmd, typed); got != want {
			t.Errorf("suggestCommand(%q) = %q, want %q", typed, got, want)
		}
	}

	// a tie says nothing rather than guess: "gill" is one edit from both
	tree := &cobra.Command{Use: "ports"}
	tree.AddCommand(&cobra.Command{Use: "kill", Run: func(*cobra.Command, []string) {}},
		&cobra.Command{Use: "fill", Run: func(*cobra.Command, []string) {}})
	if got := suggestCommand(tree, "gill"); got != "" {
		t.Errorf("a tie should suggest nothing, got %q", got)
	}
}

// A missing or extra argument says what the command needs, in ports'
// voice, with an example to copy — never cobra's "accepts 1 arg(s)".
func TestArgsError(t *testing.T) {
	cases := []struct {
		cmd  string
		args []string
		want string
	}{
		{"free", nil, "`ports free` needs a port — e.g. `ports free 3000`"},
		{"kill", nil, "`ports kill` needs a name — e.g. `ports kill shop`"},
		{"wait", nil, "`ports wait` needs a port or a name"},
		{"free", []string{"3000", "8000"}, "`ports free` takes one port — got 2: 3000 8000"},
		{"doctor", []string{"1", "2"}, "`ports doctor` takes at most one port — got 2: 1 2"},
		{"completion", []string{"badshell"}, `unknown shell "badshell" — bash, zsh, fish, or powershell`},
		// completion's Use reads [bash|zsh|…] but it needs exactly one:
		// arity comes from the validator, not the Use line
		{"completion", nil, "`ports completion` needs a shell — e.g. `ports completion zsh`"},
		{"completion", []string{"bash", "zsh"}, "`ports completion` takes one shell — got 2: bash zsh"},
		{"here", []string{"a", "b"}, "`ports here` takes at most one directory — got 2: a b"},
	}
	for _, c := range cases {
		sub, _, err := rootCmd.Find([]string{c.cmd})
		if err != nil {
			t.Fatal(err)
		}
		got := argsError(sub, c.args, sub.Args(sub, nil) != nil, errUnused).Error()
		if !strings.Contains(got, c.want) {
			t.Errorf("ports %s %v: %q, want %q", c.cmd, c.args, got, c.want)
		}
		if strings.Contains(got, "arg(s)") {
			t.Errorf("cobra's wording leaked: %q", got)
		}
	}
}

var errUnused = errString("cobra's own error")

type errString string

func (e errString) Error() string { return string(e) }
