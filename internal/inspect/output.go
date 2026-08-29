package inspect

import (
	"cmp"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"text/tabwriter"
)

// WriteJSON writes services as JSON to stdout.
func WriteJSON(services []Service) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(services)
}

// WriteCSV writes services as CSV to stdout.
func WriteCSV(services []Service) error {
	return writeDelimited(services, ',')
}

// WriteTSV writes services as TSV to stdout.
func WriteTSV(services []Service) error {
	return writeDelimited(services, '\t')
}

// sortRows is the shared display order: rows with a project first, then
// by project name and path, then by port — so a project's services
// appear together regardless of which sockets the scan returned in.
func sortRows(services []Service) []Service {
	rows := slices.Clone(services)
	slices.SortStableFunc(rows, func(a, b Service) int {
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
	return rows
}

func writeDelimited(services []Service, delim rune) error {
	if len(services) == 0 {
		return nil
	}
	rows := sortRows(services)

	w := csv.NewWriter(os.Stdout)
	w.Comma = delim
	defer w.Flush()

	// Header
	if err := w.Write([]string{"PORT", "PROCESS", "PROJECT", "UPTIME"}); err != nil {
		return err
	}
	for _, s := range rows {
		if err := w.Write([]string{
			fmt.Sprintf("%d", s.Port),
			ProcessName(s),
			project(s),
			uptime(s),
		}); err != nil {
			return err
		}
	}
	return nil
}

// WriteTable writes services as a formatted table to stdout.
func WriteTable(services []Service) {
	if len(services) == 0 {
		fmt.Println("no listening TCP ports")
		return
	}
	rows := sortRows(services)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(w, "PORT\tPROCESS\tPROJECT\tUPTIME")
	unknown := false
	for _, s := range rows {
		if !s.Known() && s.Container == "" && !s.Windows {
			unknown = true
		}
		_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", s.Port, ProcessName(s), project(s), uptime(s))
	}
	_ = w.Flush()
	if unknown {
		fmt.Fprintln(os.Stderr, "\nsome ports belong to other users' processes (shown as ?) — run with sudo to see them")
	}
}

// ProcessName is the process as a developer knows it: the framework
// when detected, the docker image base for containers, else the exe.
func ProcessName(s Service) string {
	switch {
	case s.Framework != "":
		return s.Framework
	case s.Image != "":
		return ImageBase(s.Image)
	default:
		return s.Process
	}
}

func project(s Service) string {
	if s.Windows {
		return "windows"
	}
	if s.Container != "" {
		runtime := s.Runtime
		if runtime == "" {
			runtime = "docker"
		}
		return runtime + ": " + DockerName(s)
	}
	if s.Project == "" {
		return "—"
	}
	if home, err := os.UserHomeDir(); err == nil {
		if rel, ok := strings.CutPrefix(s.Project, home); ok {
			return "~" + rel
		}
	}
	return s.Project
}

func uptime(s Service) string {
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

// DockerName is the container as a developer knows it: the compose
// project-service pair when labeled, else the container name.
func DockerName(s Service) string {
	if s.ComposeProject != "" && s.ComposeService != "" {
		return s.ComposeProject + "-" + s.ComposeService
	}
	return s.Container
}

// WriteTree writes services as a table where child processes are nested
// under their parent, matched on pid → parent pid. Only host processes
// participate: a row whose parent is not itself in the scan is a root,
// and container / windows rows are always roots — their parent on the
// other side of a boundary is not another scanned row's child.
func WriteTree(services []Service) {
	if len(services) == 0 {
		fmt.Println("no listening TCP ports")
		return
	}
	rows := sortRows(services)

	type node struct {
		s        *Service
		children []*node
	}
	nodes := make([]*node, len(rows))
	byPid := map[int32]*node{}
	for i := range nodes {
		nodes[i] = &node{s: &rows[i]}
	}
	for _, n := range nodes {
		if _, seen := byPid[n.s.Pid]; !seen {
			byPid[n.s.Pid] = n // first row wins when a pid holds several ports
		}
	}
	var roots []*node
	for _, n := range nodes {
		s := n.s
		if parent := byPid[s.ParentPid]; s.ParentPid > 0 && parent != nil && parent.s != s &&
			s.Container == "" && !s.Windows {
			parent.children = append(parent.children, n)
			continue
		}
		roots = append(roots, n)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(w, "PORT\tPROCESS\tPROJECT\tUPTIME")
	// line prefixes the connector drawn just before this row's content;
	// base is the continuation gap its subtree hangs under.
	var draw func(n *node, line, base string)
	draw = func(n *node, line, base string) {
		writeTreeRow(w, *n.s, line)
		for i, c := range n.children {
			conn, cont := "├─ ", "│  "
			if i == len(n.children)-1 {
				conn, cont = "└─ ", "   "
			}
			draw(c, base+conn, base+cont)
		}
	}
	for _, r := range roots {
		draw(r, "", "")
	}
	_ = w.Flush()
}

func writeTreeRow(w *tabwriter.Writer, s Service, prefix string) {
	_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", s.Port, prefix+ProcessName(s), project(s), uptime(s))
}
