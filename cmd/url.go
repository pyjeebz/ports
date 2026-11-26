package cmd

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/spf13/cobra"
)

var urlLocal bool

var urlCmd = &cobra.Command{
	Use:   "url [name]",
	Short: "Print a dev server's URL by name, for scripts",
	Long: `ports url prints the URL a dev server answers at, by its name — the one
to hand a script, a test runner or an agent instead of a port number.

  $ ports url api.shop
  http://api.shop.localhost
  $ curl "$(ports url api.shop)/health"

With no name it's the app in the directory you're in, else the main app
of the project. While the router runs (ports proxy) the URL is the
name's; without one it's the server's own address, http://localhost:8000,
so a script works either way.

Browsers and curl resolve every *.localhost name on their own; some
systems' resolvers don't (Windows, and Linux without systemd-resolved's
stub), so Node's fetch, Python or Go may not find shop.localhost there.
--local prints the server's own address, which every client reaches;
ports hosts sync makes the names resolve for them too.

Exit codes: 0 with the URL on stdout, 1 when nothing running has that
name (the names that are running go to stderr), 2 for a bad invocation.
--json prints {"name", "url", "port", "local_url"}.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// "text" is what this prints; "table", the shared flag's default,
		// means the same here
		if format != "table" && format != "text" && format != "json" {
			return exitError{code: 2, msg: fmt.Sprintf("unknown format %q (want text or json)", format)}
		}
		cfg := buildInspectConfig()
		services, err := scanNamed(cfg)
		if err != nil {
			return err
		}
		var s inspect.Service
		var ok bool
		if len(args) == 1 {
			name := nameArg(args[0])
			if name == "" || strings.Trim(name, "abcdefghijklmnopqrstuvwxyz0123456789-.") != "" {
				return exitError{code: 2, msg: fmt.Sprintf("%q is not a name", args[0])}
			}
			s, ok = byName(services, name)
			if !ok {
				return exitError{code: 1, msg: fmt.Sprintf("nothing running is called %s%s", name, runningNames(services))}
			}
		} else {
			dir, err := os.Getwd()
			if err != nil {
				return err
			}
			dir = inspect.RealPath(dir)
			root := inspect.ProjectRoot(cfg, dir)
			// the app you're in (apps/api is api.shop), else the project's
			// main app
			s, ok = mainApp(inspect.Here(services, dir))
			if !ok {
				s, ok = mainApp(inspect.Here(services, root))
			}
			if !ok {
				return exitError{code: 1, msg: fmt.Sprintf("no web dev server is running for %s%s", inspect.ShortPath(root), runningNames(services))}
			}
		}
		url := cmp.Or(s.NameURL, s.URL)
		if urlLocal {
			url = s.URL
		}
		if wantJSON() {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(struct {
				Name     string `json:"name"`
				URL      string `json:"url"`
				Port     uint32 `json:"port"`
				LocalURL string `json:"local_url"`
			}{s.Name, url, s.Port, s.URL})
		}
		fmt.Println(url)
		if s.NameURL == "" && !urlLocal && inspect.ColorErr {
			// only where someone reads it: a script capturing stdout
			// gets a URL that works either way
			fmt.Fprintf(os.Stderr, "the router isn't running — `ports proxy` serves this at http://%s.localhost\n", s.Name)
		}
		return nil
	},
}

func init() {
	formatFlags(urlCmd, "text, json")
	urlCmd.Flags().BoolVar(&urlLocal, "local", false, "Print the server's own address (http://localhost:4123), not the name's")
	rootCmd.AddCommand(urlCmd)
}

// nameArg is the name in what was typed: shop, shop.localhost, or a URL
// copied from the table — http://shop.localhost:8480/ is shop.
func nameArg(arg string) string {
	s := strings.ToLower(strings.TrimSpace(arg))
	s = strings.TrimPrefix(strings.TrimPrefix(s, "http://"), "https://")
	if i := strings.IndexAny(s, "/:"); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSuffix(strings.TrimSuffix(s, "."), ".localhost")
	return s
}

func byName(services []inspect.Service, name string) (inspect.Service, bool) {
	for _, s := range services {
		if s.Name != "" && s.Name == name {
			return s, true
		}
	}
	return inspect.Service{}, false
}

// mainApp is a project's main web app: the one with the shortest name —
// shop over api.shop — then the lowest port.
func mainApp(rows []inspect.Service) (inspect.Service, bool) {
	rows = slices.DeleteFunc(slices.Clone(rows), func(s inspect.Service) bool { return s.Name == "" })
	if len(rows) == 0 {
		return inspect.Service{}, false
	}
	return slices.MinFunc(rows, func(a, b inspect.Service) int {
		if c := cmp.Compare(strings.Count(a.Name, "."), strings.Count(b.Name, ".")); c != 0 {
			return c
		}
		return cmp.Compare(a.Port, b.Port)
	}), true
}

// runningNames is the tail of a not-found message: the names there are.
func runningNames(services []inspect.Service) string {
	var names []string
	for _, s := range services {
		if s.Name != "" {
			names = append(names, s.Name)
		}
	}
	if len(names) == 0 {
		return " — no named dev servers are running"
	}
	slices.Sort(names)
	return " — running: " + strings.Join(names, ", ")
}
