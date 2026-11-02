package inspect

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"text/tabwriter"
	"time"
)

// snapshotVersion pins the snapshot envelope. A file stamped with a
// version this ports doesn't know refuses to load rather than guess at
// its shape.
const snapshotVersion = 1

// Snapshot is the versioned envelope `--format snapshot` writes and
// `ports diff` reads: the scan rows, plus the moment they were true.
type Snapshot struct {
	Version     int       `json:"version"`
	GeneratedAt time.Time `json:"generated_at"`
	Services    []Service `json:"services"`
}

// Diff is one change between a snapshot and the machine now.
type Diff struct {
	Port    uint32 `json:"port"`
	Change  string `json:"change"` // new, gone, changed
	Process string `json:"process"`
	Project string `json:"project"`
	Message string `json:"message"`
}

// WriteSnapshot renders services as `--format snapshot` — the versioned
// envelope diff reads back, with the rows exactly as json writes them.
func WriteSnapshot(services []Service) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(Snapshot{
		Version:     snapshotVersion,
		GeneratedAt: time.Now().UTC(),
		Services:    services,
	})
}

// LoadSnapshot reads a snapshot from disk. What `--format snapshot`
// writes is the expected shape; a bare array — a plain `--format json`
// capture — loads too, with no moment to report.
func LoadSnapshot(path string) (Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, fmt.Errorf("cannot read snapshot %s: %w", path, err)
	}
	snap, err := ParseSnapshot(data)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%s: %w", path, err)
	}
	return snap, nil
}

// ParseSnapshot decodes snapshot bytes: the versioned envelope, or a
// bare array of rows as `--format json` writes them. Pure over its
// input, so the shapes can be tested without a filesystem.
func ParseSnapshot(data []byte) (Snapshot, error) {
	// a plain json capture is just the rows — same shapes, no envelope
	if trimmed := bytes.TrimLeft(data, " \t\r\n"); len(trimmed) > 0 && trimmed[0] == '[' {
		var services []Service
		if err := json.Unmarshal(data, &services); err != nil {
			return Snapshot{}, fmt.Errorf("does not parse as ports output: %w", err)
		}
		return Snapshot{Services: services}, nil
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return Snapshot{}, fmt.Errorf("does not parse as ports output: %w", err)
	}
	if snap.Version != snapshotVersion {
		return Snapshot{}, fmt.Errorf("snapshot version %d — this ports reads version %d", snap.Version, snapshotVersion)
	}
	return snap, nil
}

// DiffServices compares a snapshot's rows against the machine now,
// keyed by port — the port is what the user can name, and a restart is
// the story diff exists to tell. A row's identity is its pid, the
// container id for container rows, or the Windows pid across the WSL
// boundary: the same identity on both sides is the same listener, and
// everything else — uptime included — is expected to drift and ignored.
func DiffServices(was, now []Service) []Diff {
	then := groupByPort(was)
	current := groupByPort(now)

	ports := make([]uint32, 0, len(then)+len(current))
	for p := range then {
		ports = append(ports, p)
	}
	for p := range current {
		if _, ok := then[p]; !ok {
			ports = append(ports, p)
		}
	}
	slices.Sort(ports)

	var diffs []Diff
	for _, p := range ports {
		before, hadBefore := then[p]
		after, hasAfter := current[p]
		switch {
		case !hadBefore:
			diffs = append(diffs, Diff{Port: p, Change: "new",
				Process: ProcessName(after[0]), Project: project(after[0]),
				Message: "listening now — not in the snapshot"})
		case !hasAfter:
			diffs = append(diffs, Diff{Port: p, Change: "gone",
				Process: ProcessName(before[0]), Project: project(before[0]),
				Message: "in the snapshot, no longer listening"})
		case !sameIdentities(before, after):
			diffs = append(diffs, Diff{Port: p, Change: "changed",
				Process: ProcessName(after[0]), Project: project(after[0]),
				Message: changeMessage(before, after)})
		}
	}
	return diffs
}

func groupByPort(services []Service) map[uint32][]Service {
	by := make(map[uint32][]Service)
	for _, s := range services {
		by[s.Port] = append(by[s.Port], s)
	}
	return by
}

// rowIdentity names the listener behind a row: the container id for
// container rows (the proxy pid is incidental), the Windows pid across
// the WSL boundary, else the pid. An unknowable row — another user's
// process, shown as ? — has no identity; only its presence can be
// compared.
func rowIdentity(s Service) string {
	switch {
	case s.Windows:
		return fmt.Sprintf("w%d", s.WindowsPid)
	case s.ContainerID != "":
		return "c" + s.ContainerID
	case s.Known():
		return fmt.Sprintf("p%d", s.Pid)
	default:
		return ""
	}
}

func identities(rows []Service) map[string]bool {
	set := make(map[string]bool)
	for _, s := range rows {
		set[rowIdentity(s)] = true
	}
	return set
}

func sameIdentities(was, now []Service) bool {
	a, b := identities(was), identities(now)
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

// changeMessage says what happened on a port whose listeners changed:
// another process took the port, a same-named one restarted, or the
// crowd around a shared port shifted.
func changeMessage(was, now []Service) string {
	wasName, nowName := ProcessName(was[0]), ProcessName(now[0])
	if wasName != nowName {
		return fmt.Sprintf("was %s, now %s", wasName, nowName)
	}
	if len(was) == 1 && len(now) == 1 {
		switch {
		case was[0].ContainerID != "":
			return fmt.Sprintf("restarted — container %s → %s", was[0].ContainerID, now[0].ContainerID)
		case rowIdentity(was[0]) != "" && rowIdentity(now[0]) != "":
			return fmt.Sprintf("restarted — pid %d → %d", was[0].Pid, now[0].Pid)
		default:
			return "restarted"
		}
	}
	msg := fmt.Sprintf("now %d processes (was %d)", len(now), len(was))
	for id := range identities(now) {
		if strings.HasPrefix(id, "p") && identities(was)[id] {
			return msg + fmt.Sprintf(" — pid %s survives", strings.TrimPrefix(id, "p"))
		}
	}
	return msg
}

// WriteDiffs renders the diff table — one line per changed port, in the
// idiom of the scan table — or, when nothing changed, the sentence that
// says so, with the moment the snapshot was taken when the file has one.
// was is the snapshot's side of the comparison, already filtered to the
// rows diff actually weighed.
func WriteDiffs(diffs []Diff, was []Service, at time.Time) {
	if len(diffs) == 0 {
		when := ""
		if age := time.Since(at); !at.IsZero() && age > 0 {
			when = fmt.Sprintf(" taken %s ago", durationString(age))
		}
		fmt.Printf("no changes — the machine matches the snapshot%s (%d listening)\n", when, len(was))
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(w, "PORT\tCHANGE\tPROCESS\tPROJECT\tNOTE")
	for _, d := range diffs {
		_, _ = fmt.Fprintf(w, ":%d\t%s\t%s\t%s\t%s\n", d.Port, d.Change, d.Process, d.Project, d.Message)
	}
	_ = w.Flush()
}

// WriteDiffsJSON renders diffs as an array — a clean diff is an empty
// list, not null — for the scripts that branch on the changes.
func WriteDiffsJSON(diffs []Diff) error {
	if diffs == nil {
		diffs = []Diff{}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(diffs)
}
