package routes

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	gproc "github.com/shirou/gopsutil/v4/process"

	"github.com/pyjeebz/ports/internal/state"
)

// Declared names are the ones someone asked for, beside the discovered
// ones: an alias (ports alias stripe 4242) for what ports can't infer,
// and a ports run registration for a server it started. Both win over a
// discovered name — an alias over a run — and both live in the state
// directory, user-only, as files the router reads on every refresh.

// Aliases maps a fixed name to a port.
type Aliases map[string]uint32

const aliasFile = "aliases.json"

// LoadAliases reads the aliases; none yet is an empty set.
func LoadAliases() (Aliases, error) {
	var f struct {
		Aliases Aliases `json:"aliases"`
	}
	if err := state.Read(aliasFile, &f); err != nil {
		return nil, fmt.Errorf("reading aliases: %w", err)
	}
	if f.Aliases == nil {
		f.Aliases = Aliases{}
	}
	return f.Aliases, nil
}

// SaveAliases writes the aliases back.
func SaveAliases(a Aliases) error {
	return state.Write(aliasFile, struct {
		Aliases Aliases `json:"aliases"`
	}{a})
}

// Run is a ports run registration: the name a command asked for and the
// port it was handed, held for exactly as long as the ports run process
// lives. Pid and start time together identify that process, so a crash
// that skips the cleanup still leaves no route: the next reader sees the
// process is gone and drops the file.
type Run struct {
	Name    string `json:"name"`
	Port    uint32 `json:"port"`
	Pid     int32  `json:"pid"`
	Started int64  `json:"started"` // the process's create time, ms since epoch
	Dir     string `json:"dir"`
	Command string `json:"command"`
}

func runFile(pid int32) string { return "run-" + strconv.Itoa(int(pid)) + ".json" }

// Register records r until Unregister — or until its process ends.
func Register(r Run) error { return state.Write(runFile(r.Pid), r) }

// Unregister drops the registration of pid.
func Unregister(pid int32) error { return state.Remove(runFile(pid)) }

// ProcessStarted is pid's create time, ms since epoch, as a Run records it.
func ProcessStarted(pid int32) (int64, error) {
	p, err := gproc.NewProcess(pid)
	if err != nil {
		return 0, err
	}
	return p.CreateTime()
}

// LiveRuns is every registration whose process still runs, by name. A
// registration whose process is gone — or whose pid now belongs to some
// other process — is removed on the way.
func LiveRuns() []Run {
	files, err := state.Glob("run-*.json")
	if err != nil {
		return nil
	}
	var out []Run
	for _, f := range files {
		var r Run
		if err := state.Read(f, &r); err != nil || r.Pid == 0 {
			continue
		}
		if started, err := ProcessStarted(r.Pid); err != nil || started != r.Started {
			_ = state.Remove(f)
			continue
		}
		out = append(out, r)
	}
	slices.SortFunc(out, func(a, b Run) int { return strings.Compare(a.Name, b.Name) })
	return out
}

// Declared is what the route table merges over discovery: the aliases and
// the live runs. Tests and callers that want discovery alone pass nil.
type Declared func() (Aliases, []Run)

// LoadDeclared reads both from the state directory. An unreadable alias
// file counts as none — the router keeps routing what it can.
func LoadDeclared() (Aliases, []Run) {
	a, err := LoadAliases()
	if err != nil {
		a = Aliases{}
	}
	return a, LiveRuns()
}

// ValidName reports whether s can be a name: dot-separated labels of
// lowercase letters, digits and inner hyphens — api.shop, stripe,
// fix-ui.shop — so that <name>.localhost is a hostname.
func ValidName(s string) bool {
	if s == "" || len(s) > 200 {
		return false
	}
	for label := range strings.SplitSeq(s, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
				return false
			}
		}
	}
	return true
}
