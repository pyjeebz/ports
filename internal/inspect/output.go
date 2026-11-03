package inspect

import (
	"bytes"
	"cmp"
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
// the shape every consumer expects, not null.
func WriteJSON(services []Service) error {
	if services == nil {
		services = []Service{}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(services)
}

// SortMode names a display order for the table.
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
// scan returned in. An explicit --sort replaces the arrangement.
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

// WriteTable writes services as a formatted table to stdout.
func WriteTable(services []Service, sort SortMode) {
	if len(services) == 0 {
		fmt.Println("no listening TCP ports")
		return
	}
	rows := sortRows(services, sort)

	labels := projectLabels(rows)
	// the URL column appears only when a row has one, and goes last: a
	// link's escape bytes never sit in a cell tabwriter measures
	urls := slices.ContainsFunc(rows, func(s Service) bool { return s.URL != "" })
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', 0)
	header := "PORT\tPROCESS\tPROJECT\tUPTIME"
	if urls {
		header += "\tURL"
	}
	_, _ = fmt.Fprintln(w, header)
	unknown := false
	for _, s := range rows {
		if s.Anonymous() {
			unknown = true
		}
		line := fmt.Sprintf("%d\t%s\t%s\t%s", s.Port, ProcessName(s), labels(s), uptime(s))
		if urls {
			// every row keeps the column so the block stays aligned;
			// the padding a row without a URL picks up is trimmed below
			line += "\t" + hyperlink(s.URL)
		}
		_, _ = fmt.Fprintln(w, line)
	}
	_ = w.Flush()
	for _, line := range strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n") {
		fmt.Println(strings.TrimRight(line, " "))
	}
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

// projectLabels is the PROJECT column for a table of rows: a project
// directory reads as its name (shop), not its path — a path outside the
// home directory can run to dozens of characters and push the table off
// the screen. Two directories that share a name get just enough of their
// parents to tell apart (work/shop, oss/shop). Everything else reads as
// project does. Machine output — JSON, doctor's header — keeps the path.
func projectLabels(rows []Service) func(Service) string {
	byName := map[string][]string{}
	for _, s := range rows {
		if isProjectDir(s) {
			dir := filepath.Clean(s.Project)
			name := filepath.Base(dir)
			if !slices.Contains(byName[name], dir) {
				byName[name] = append(byName[name], dir)
			}
		}
	}
	label := map[string]string{}
	for _, paths := range byName {
		for p, l := range distinctSuffixes(paths) {
			// a directory named windows or — would read as a Windows-side
			// or unknown row: it keeps its path instead
			if l == "windows" || l == "—" {
				l = ShortPath(p)
			}
			label[p] = l
		}
	}
	return func(s Service) string {
		if isProjectDir(s) {
			return label[filepath.Clean(s.Project)]
		}
		return project(s)
	}
}

// isProjectDir reports a row whose PROJECT is a host directory — the
// rows project renders as a path.
func isProjectDir(s Service) bool {
	return !s.Windows && s.Container == "" && s.Unit == "" && s.User == "" && s.Project != ""
}

// distinctSuffixes labels each path by its shortest trailing run of
// segments no other path in the set ends with; a path that's a suffix
// of another all the way up reads in full, as project does.
func distinctSuffixes(paths []string) map[string]string {
	split := make([][]string, len(paths))
	for i, p := range paths {
		split[i] = strings.Split(filepath.ToSlash(filepath.Clean(p)), "/")
	}
	out := map[string]string{}
	for i, p := range paths {
		segs := split[i]
		for k := 1; ; k++ {
			if k >= len(segs) {
				out[p] = ShortPath(p)
				break
			}
			suffix := strings.Join(segs[len(segs)-k:], "/")
			clash := false
			for j, other := range split {
				if j != i && len(other) >= k && strings.Join(other[len(other)-k:], "/") == suffix {
					clash = true
					break
				}
			}
			if !clash && suffix != "" {
				out[p] = suffix
				break
			}
		}
	}
	return out
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
// the coarsest that hasn't passed.
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
