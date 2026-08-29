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

// writeReport renders services in the --format the user asked for.
func writeReport(services []inspect.Service) error {
	if jsonOut {
		// --json is deprecated but still honored
		return inspect.WriteJSON(services)
	}
	switch format {
	case "json":
		return inspect.WriteJSON(services)
	case "csv":
		return inspect.WriteCSV(services)
	case "tsv":
		return inspect.WriteTSV(services)
	case "table":
		if showTree {
			inspect.WriteTree(services)
			return nil
		}
		inspect.WriteTable(services)
		return nil
	default:
		return fmt.Errorf("unknown format %q (want table, json, csv, or tsv)", format)
	}
}

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
		if err := inspect.Stop(g.s, force); err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s (%s): %v\n", inspect.ProcessName(g.s), ports, err)
			code = 1
			continue
		}
		fmt.Println(okMsg(g.s, ports))
	}
	return code
}
