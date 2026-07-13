package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/pyjeebz/ports/internal/inspect"
)

func main() {
	jsonOut, force := false, false
	var args []string
	for _, a := range os.Args[1:] {
		switch a {
		case "--json":
			jsonOut = true
		case "--force":
			force = true
		default:
			args = append(args, a)
		}
	}
	cmd, rest := "list", args
	if len(args) > 0 {
		cmd, rest = args[0], args[1:]
	}

	services, err := inspect.Scan()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ports:", err)
		os.Exit(1)
	}

	switch cmd {
	case "list":
		render(services, jsonOut)

	case "find":
		q := needArg(rest, "find <name>")
		matched := match(services, func(s inspect.Service) bool { return s.MatchesQuery(q) })
		if len(matched) == 0 {
			fmt.Fprintf(os.Stderr, "ports: nothing matching %q\n", q)
			os.Exit(1)
		}
		render(matched, jsonOut)

	case "free":
		arg := needArg(rest, "free <port>")
		port, err := strconv.ParseUint(arg, 10, 16)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ports: invalid port %q\n", arg)
			os.Exit(2)
		}
		matched := match(services, func(s inspect.Service) bool { return s.Port == uint32(port) })
		if len(matched) == 0 {
			fmt.Fprintf(os.Stderr, "ports: nothing listening on %d\n", port)
			os.Exit(1)
		}
		os.Exit(stopAll(matched, force, func(s inspect.Service, _ string) string {
			if s.Container != "" {
				return fmt.Sprintf("✓ stopped %s (container %s) — port %d is free", process(s), dockerName(s), port)
			}
			return fmt.Sprintf("✓ killed %s (pid %d) — port %d is free", process(s), s.Pid, port)
		}))

	case "kill":
		q := needArg(rest, "kill <name>")
		matched := match(services, func(s inspect.Service) bool { return s.MatchesName(q) })
		if len(matched) == 0 {
			fmt.Fprintf(os.Stderr, "ports: nothing matching %q — try `ports find %s`\n", q, q)
			os.Exit(1)
		}
		os.Exit(stopAll(matched, force, func(s inspect.Service, ports string) string {
			return fmt.Sprintf("✓ stopped %s (%s)", process(s), ports)
		}))

	default:
		fmt.Fprintf(os.Stderr, "ports: unknown command %q\n", cmd)
		usage(os.Stderr)
		os.Exit(2)
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `usage:
  ports              list listening ports
  ports find <name>  filter by framework, process, or project
  ports free <port>  kill whatever holds a port
  ports kill <name>  stop a project or service by name

flags:
  --json   JSON output (list, find)
  --force  SIGKILL immediately instead of SIGTERM first
`)
}

func needArg(rest []string, form string) string {
	if len(rest) != 1 || rest[0] == "" {
		fmt.Fprintf(os.Stderr, "ports: usage: ports %s\n", form)
		os.Exit(2)
	}
	return rest[0]
}

func match(services []inspect.Service, keep func(inspect.Service) bool) []inspect.Service {
	var out []inspect.Service
	for _, s := range services {
		if keep(s) {
			out = append(out, s)
		}
	}
	return out
}

// stopAll stops one process per pid — or one container per id, however
// many ports it publishes — and prints a line per stop. Returns the exit
// code.
func stopAll(matched []inspect.Service, force bool, okMsg func(inspect.Service, string) string) int {
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
		if err := inspect.Stop(g.s, force); err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s (%s): %v\n", process(g.s), ports, err)
			code = 1
			continue
		}
		fmt.Println(okMsg(g.s, ports))
	}
	return code
}

func render(services []inspect.Service, jsonOut bool) {
	if jsonOut {
		writeJSON(services)
		return
	}
	writeTable(services)
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
	// group by project name, so a frontend, api, and compose db that
	// belong together read as one application even across the docker line
	rows := slices.Clone(services)
	slices.SortStableFunc(rows, func(a, b inspect.Service) int {
		if (a.ProjectName == "") != (b.ProjectName == "") {
			if a.ProjectName != "" {
				return -1
			}
			return 1
		}
		if c := strings.Compare(a.ProjectName, b.ProjectName); c != 0 {
			return c
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
		if !s.Known() && s.Container == "" {
			unknown = true // docker rows are identified via the daemon; sudo adds nothing
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", s.Port, process(s), project(s), uptime(s))
	}
	w.Flush()
	if unknown {
		fmt.Fprintln(os.Stderr, "\nsome ports belong to other users' processes (shown as ?) — run with sudo to see them")
	}
}

func process(s inspect.Service) string {
	switch {
	case s.Framework != "":
		return s.Framework
	case s.Image != "":
		return inspect.ImageBase(s.Image)
	default:
		return s.Process
	}
}

// dockerName is the container as a developer knows it: the compose
// project-service pair when labeled, else the container name.
func dockerName(s inspect.Service) string {
	if s.ComposeProject != "" && s.ComposeService != "" {
		return s.ComposeProject + "-" + s.ComposeService
	}
	return s.Container
}

func project(s inspect.Service) string {
	if s.Container != "" {
		return "docker: " + dockerName(s)
	}
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
