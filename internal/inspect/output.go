package inspect

import (
	"cmp"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"text/tabwriter"
)

// WriteJSON writes services as JSON to stdout. A nil slice renders as [],
// the shape every consumer — the diff included — expects, not null.
func WriteJSON(services []Service) error {
	if services == nil {
		services = []Service{}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(services)
}

// WriteCSV writes services as CSV to stdout.
func WriteCSV(services []Service, sort SortMode) error {
	return writeDelimited(services, ',', sort)
}

// WriteTSV writes services as TSV to stdout.
func WriteTSV(services []Service, sort SortMode) error {
	return writeDelimited(services, '\t', sort)
}

// SortMode names a display order for the table and its delimited forms.
// The zero value is the default order, not a third state — sorting is
// never off, it just has a preferred arrangement unless --sort says
// otherwise.
type SortMode string

const (
	SortDefault SortMode = "" // project-grouped, the load-bearing default
	SortPort    SortMode = "port"
	SortUptime  SortMode = "uptime"
	SortProcess SortMode = "process"
	SortProject SortMode = "project"
)

// ParseSortMode validates a --sort value. "" passes through as the
// default order; anything unknown is a bad argument.
func ParseSortMode(s string) (SortMode, error) {
	switch mode := SortMode(s); mode {
	case SortDefault, SortPort, SortUptime, SortProcess, SortProject:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid sort %q (want port, uptime, process, or project)", s)
	}
}

// sortRows is the shared display order. The default groups by project —
// rows with a project first, then by project name and path, then by port —
// so a project's services appear together regardless of which sockets the
// scan returned in. An explicit --sort replaces the arrangement; --tree
// always uses the default, since nesting carries the structure instead.
func sortRows(services []Service, sort SortMode) []Service {
	rows := slices.Clone(services)
	slices.SortStableFunc(rows, func(a, b Service) int {
		switch sort {
		case SortPort:
			return cmp.Compare(a.Port, b.Port)
		case SortUptime:
			// longest-lived first — ascending by start time; unknown start
			// times sort last
			if (a.Started.IsZero()) != (b.Started.IsZero()) {
				if a.Started.IsZero() {
					return 1
				}
				return -1
			}
			if c := a.Started.Compare(b.Started); c != 0 {
				return c
			}
			return cmp.Compare(a.Port, b.Port)
		case SortProcess:
			if c := strings.Compare(strings.ToLower(ProcessName(a)), strings.ToLower(ProcessName(b))); c != 0 {
				return c
			}
			return cmp.Compare(a.Port, b.Port)
		case SortProject:
			if c := strings.Compare(strings.ToLower(a.ProjectName), strings.ToLower(b.ProjectName)); c != 0 {
				return c
			}
			return cmp.Compare(a.Port, b.Port)
		default:
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
		}
	})
	return rows
}

func writeDelimited(services []Service, delim rune, sort SortMode) error {
	if len(services) == 0 {
		return nil
	}
	rows := sortRows(services, sort)

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
func WriteTable(services []Service, sort SortMode) {
	if len(services) == 0 {
		fmt.Println("no listening TCP ports")
		return
	}
	rows := sortRows(services, sort)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(w, "PORT\tPROCESS\tPROJECT\tUPTIME")
	unknown := false
	for _, s := range rows {
		if s.Anonymous() {
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

// ProjectLabel is the PROJECT column's text for a row: a path, a
// container, a systemd unit, windows, or —.
func ProjectLabel(s Service) string { return project(s) }

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
	if s.Unit != "" {
		return "systemd: " + s.Unit
	}
	if s.User != "" {
		return "user: " + s.User
	}
	if s.Project == "" {
		return "—"
	}
	return ShortPath(s.Project)
}

// ShortPath renders a path the way the PROJECT column does: under the
// home directory it reads as ~/…, anywhere else as itself.
func ShortPath(path string) string {
	if home, err := os.UserHomeDir(); err == nil {
		// the home itself and siblings of it both cut cleanly — only a
		// path *under* home gets the ~, so ~/ports and ports-tools stay
		// distinct projects. Windows builds the tree with \, so the guard
		// and the render both go through the separator, not a literal /.
		sep := string(filepath.Separator)
		if rel, ok := strings.CutPrefix(path, home); ok && (rel == "" || strings.HasPrefix(rel, sep)) {
			return "~" + filepath.ToSlash(rel)
		}
	}
	return path
}

func uptime(s Service) string {
	if s.Started.IsZero() {
		return "?"
	}
	return durationString(time.Since(s.Started))
}

// durationString rounds a duration the way the table does — one unit,
// the coarsest that hasn't passed — shared by uptime and the snapshot
// age so they can't drift apart.
func durationString(d time.Duration) string {
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
// other side of a boundary is not another scanned row's child. The tree
// always renders in the default order — --sort would scramble the
// parent/child walk for no gain.
func WriteTree(services []Service) {
	if len(services) == 0 {
		fmt.Println("no listening TCP ports")
		return
	}
	rows := sortRows(services, SortDefault)

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
