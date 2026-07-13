package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/pyjeebz/ports/internal/inspect"
)

func main() {
	jsonOut := false
	var args []string
	for _, a := range os.Args[1:] {
		switch a {
		case "--json":
			jsonOut = true
		default:
			args = append(args, a)
		}
	}
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "ports: unknown command %q\nusage: ports [--json]\n", args[0])
		os.Exit(2)
	}

	services, err := inspect.Scan()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ports:", err)
		os.Exit(1)
	}
	if jsonOut {
		writeJSON(services)
	} else {
		writeTable(services)
	}
}

func writeJSON(services []inspect.Service) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(services); err != nil {
		fmt.Fprintln(os.Stderr, "ports:", err)
		os.Exit(1)
	}
}

func writeTable(services []inspect.Service) {
	if len(services) == 0 {
		fmt.Println("no listening TCP ports")
		return
	}
	// group by project: project rows first, same project adjacent, so a
	// frontend, api, and db that belong together read as one application
	rows := slices.Clone(services)
	slices.SortStableFunc(rows, func(a, b inspect.Service) int {
		if (a.Project == "") != (b.Project == "") {
			if a.Project != "" {
				return -1
			}
			return 1
		}
		if c := strings.Compare(a.Project, b.Project); c != 0 {
			return c
		}
		return cmp.Compare(a.Port, b.Port)
	})

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "PORT\tPROCESS\tPROJECT\tUPTIME")
	unknown := false
	for _, s := range rows {
		if !s.Known() {
			unknown = true
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", s.Port, process(s), project(s), uptime(s))
	}
	w.Flush()
	if unknown {
		fmt.Fprintln(os.Stderr, "\nsome ports belong to other users' processes (shown as ?) — run with sudo to see them")
	}
}

func process(s inspect.Service) string {
	if s.Framework != "" {
		return s.Framework
	}
	return s.Process
}

func project(s inspect.Service) string {
	if s.Project == "" {
		return "—"
	}
	if home, err := os.UserHomeDir(); err == nil {
		if rel := strings.TrimPrefix(s.Project, home); rel != s.Project {
			return "~" + rel
		}
	}
	return s.Project
}

func uptime(s inspect.Service) string {
	if s.Started.IsZero() {
		return "?"
	}
	d := time.Since(s.Started)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
