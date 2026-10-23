package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/pyjeebz/ports/internal/inspect"
)

// exitError carries a bare exit code through cobra when the useful
// output has already been printed (a per-row kill report, for one).
type exitError struct {
	code int
	msg  string
}

func (e exitError) Error() string { return e.msg }

func match(services []inspect.Service, keep func(inspect.Service) bool) []inspect.Service {
	var out []inspect.Service
	for _, s := range services {
		if keep(s) {
			out = append(out, s)
		}
	}
	return out
}

// resolveSort validates --sort into the engine's sort mode. The empty
// default is the project-grouped order; --sort only overrides when it
// names one of port, uptime, process, or project.
func resolveSort() (inspect.SortMode, error) {
	mode, err := inspect.ParseSortMode(sortBy)
	if err != nil {
		return mode, exitError{code: 2, msg: err.Error()}
	}
	return mode, nil
}

// writeReport renders services in the --format and order the user asked for.
func writeReport(services []inspect.Service, sort inspect.SortMode) error {
	if jsonOut {
		// --json is deprecated but still honored
		return inspect.WriteJSON(services)
	}
	switch format {
	case "json":
		return inspect.WriteJSON(services)
	case "csv":
		return inspect.WriteCSV(services, sort)
	case "tsv":
		return inspect.WriteTSV(services, sort)
	case "snapshot":
		// the versioned envelope `ports diff` reads back — rows exactly
		// as json writes them, plus the moment they were true
		return inspect.WriteSnapshot(services)
	case "table":
		if showTree {
			// nesting carries the structure — --sort would scramble it
			inspect.WriteTree(services)
			return nil
		}
		inspect.WriteTable(services, sort)
		return nil
	default:
		return exitError{code: 2, msg: fmt.Sprintf("unknown format %q (want table, json, snapshot, csv, or tsv)", format)}
	}
}

// stopOne is the seam stopAll runs through — tests swap it to observe
// stops without sending signals.
var stopOne = inspect.Stop

// stopAll stops one process per pid — or one container per id, however
// many ports it publishes — and prints a line per stop. Returns the exit
// code. If dryRun is true, only shows what would be stopped without doing it.
func stopAll(matched []inspect.Service, force bool, dryRun bool, okMsg func(inspect.Service, string) string) int {
	type group struct {
		s     inspect.Service
		ports []string
	}
	var groups []*group
	byKey := map[string]*group{}
	for _, s := range matched {
		key := ""
		switch {
		case s.ContainerID != "":
			key = "c:" + s.ContainerID
		case s.Known():
			key = fmt.Sprintf("p:%d", s.Pid)
		case s.Unit != "":
			// another user's service, no pid to group by — one unit is
			// one stop, however many ports it holds
			key = "u:" + s.Unit
		}
		if g, ok := byKey[key]; ok && key != "" {
			g.ports = append(g.ports, fmt.Sprintf(":%d", s.Port))
			continue
		}
		g := &group{s: s, ports: []string{fmt.Sprintf(":%d", s.Port)}}
		groups = append(groups, g)
		if key != "" {
			byKey[key] = g
		}
	}
	code := 0
	for _, g := range groups {
		ports := strings.Join(g.ports, ", ")
		if dryRun {
			fmt.Printf("would stop %s (%s)\n", inspect.ProcessName(g.s), ports)
			continue
		}
		if err := stopOne(g.s, force); err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s (%s): %v\n", inspect.ProcessName(g.s), ports, err)
			fmt.Fprintf(os.Stderr, "  → run `ports doctor %s` to see why\n", strings.TrimPrefix(g.ports[0], ":"))
			code = 1
			continue
		}
		fmt.Println(okMsg(g.s, ports))
	}
	return code
}
