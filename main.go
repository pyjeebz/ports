package main

import (
	"encoding/json"
	"fmt"
	"os"
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
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "PORT\tPROCESS\tPROJECT\tUPTIME")
	unknown := false
	for _, s := range services {
		if !s.Known() {
			unknown = true
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", s.Port, s.Process, project(s), uptime(s))
	}
	w.Flush()
	if unknown {
		fmt.Fprintln(os.Stderr, "\nsome ports belong to other users' processes (shown as ?) — run with sudo to see them")
	}
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
