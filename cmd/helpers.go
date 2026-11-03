package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/pyjeebz/ports/internal/inspect"
)

// exitError carries a bare exit code through cobra when the useful
// output has already been printed (a per-row kill report, for one).
type exitError struct {
	code int
	msg  string
}

func (e exitError) Error() string { return e.msg }

// portArg reads a port argument the way --port reads one: 3000 or :3000.
// ok is false when arg isn't a port at all — a name, for the commands
// that take one. A number outside 1–65535 is never a name: it's a bad
// port, so `ports wait 70000 --gone` can't report a service "free" —
// and neither are a sign (+3000) or non-ASCII digits. Surrounding space
// is trimmed, so a port read from a CRLF file still reads as one.
func portArg(arg string) (port uint32, ok bool, err error) {
	trimmed := strings.TrimSpace(arg)
	digits := strings.TrimPrefix(trimmed, ":")
	unsigned := strings.TrimLeft(digits, "+-")
	if unsigned == "" || strings.IndexFunc(unsigned, func(r rune) bool { return !unicode.IsDigit(r) }) >= 0 {
		if digits != trimmed { // the colon promised a port
			return 0, false, exitError{code: 2, msg: fmt.Sprintf("invalid port %q", arg)}
		}
		return 0, false, nil
	}
	if unsigned != digits || strings.Trim(digits, "0123456789") != "" {
		return 0, false, exitError{code: 2, msg: fmt.Sprintf("invalid port %q — ports run from 1 to 65535", arg)}
	}
	n, err := strconv.ParseUint(digits, 10, 32)
	if err != nil || n < 1 || n > 65535 {
		return 0, false, exitError{code: 2, msg: fmt.Sprintf("invalid port %q — ports run from 1 to 65535", arg)}
	}
	return uint32(n), true, nil
}

// requirePort is portArg for the commands that take only a port.
func requirePort(arg string) (uint32, error) {
	port, ok, err := portArg(arg)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, exitError{code: 2, msg: fmt.Sprintf("invalid port %q", arg)}
	}
	return port, nil
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

// wantJSON reports whether the run renders JSON.
func wantJSON() bool { return jsonOut || format == "json" }

// noneFound is a query that matched nothing: exit 1, grep-style, with
// the reason on stderr. JSON still prints its empty shape — a script or
// agent parsing stdout gets [], never nothing.
func noneFound(msg string) error {
	if wantJSON() {
		if err := inspect.WriteJSON(nil); err != nil {
			return err
		}
	}
	return exitError{code: 1, msg: msg}
}

// noMatchMsg is the empty table under an active filter. The machine may
// well have listeners — just none the filter keeps — so "no listening
// TCP ports" would be false.
const noMatchMsg = "nothing listening matches the filter"

// noMatch reports whether a table render would be empty only because of
// the filter. Machine formats keep their empty shapes ([] and headers).
func noMatch(services []inspect.Service, filter *inspect.Filter) bool {
	return len(services) == 0 && filter.Active() && !jsonOut && format == "table"
}

// writeReport renders services in the --format and order the user asked for.
func writeReport(services []inspect.Service, sort inspect.SortMode) error {
	if jsonOut {
		// --json is --format json, kept short for the scripts and agents
		// that reach for it first
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

// stopOpts are the choices a free or kill runs with.
type stopOpts struct {
	force  bool
	dryRun bool
	expect string // stop only if every target matches this name
	// skipUnowned reports other users' pid-less rows as skipped rather
	// than failed: kill sweeps them up by name, where free targets them
	skipUnowned bool
}

// What happened to one stop target, in --format json. A target is one
// process, container, or service, however many ports it holds; service
// is its row exactly as `ports --json` prints it.
type stopResult struct {
	Action  string          `json:"action"` // stopped, would_stop, failed, refused, skipped
	Ports   []uint32        `json:"ports"`
	Service inspect.Service `json:"service"`
	Error   string          `json:"error,omitempty"`
	Fix     string          `json:"fix,omitempty"`
}

const (
	actionStopped   = "stopped"
	actionWouldStop = "would_stop"
	actionFailed    = "failed"
	actionRefused   = "refused"
	actionSkipped   = "skipped"
)

// stopTargets groups rows into what one stop ends: a pid, a container
// (docker-proxy runs once per published port, the container is what
// stops), or another user's systemd unit. A row nothing identifies
// stands alone.
func stopTargets(matched []inspect.Service) []stopResult {
	var targets []stopResult
	byKey := map[string]int{}
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
		if i, ok := byKey[key]; ok && key != "" {
			targets[i].Ports = append(targets[i].Ports, s.Port)
			continue
		}
		if key != "" {
			byKey[key] = len(targets)
		}
		targets = append(targets, stopResult{Ports: []uint32{s.Port}, Service: s})
	}
	return targets
}

func portList(ports []uint32) string {
	parts := make([]string, len(ports))
	for i, p := range ports {
		parts[i] = fmt.Sprintf(":%d", p)
	}
	return strings.Join(parts, ", ")
}

// stopFormat validates --format for free and kill: a stop reports as
// lines or as JSON results, nothing else.
func stopFormat() error {
	if !jsonOut && format != "table" && format != "json" {
		return exitError{code: 2, msg: fmt.Sprintf("--format %q — stops report as table or json", format)}
	}
	return nil
}

// stopAll stops every target and reports each — a line as it happens,
// or the whole set as JSON once it's done. With --expect, every target
// must match the expected name or nothing is stopped at all: the guard
// a script or agent puts on "free whatever holds :3000". Returns the
// exit code — 1 when anything failed or the guard refused.
func stopAll(matched []inspect.Service, opts stopOpts, okMsg func(inspect.Service, string) string) int {
	targets, skipped := stopTargets(matched), []stopResult(nil)
	if opts.skipUnowned {
		targets, skipped = splitUnowned(targets)
	}
	asJSON := wantJSON()
	code := 0

	if opts.expect != "" && refuseUnexpected(targets, opts.expect) {
		code = 1
		if !asJSON {
			fix := ""
			for _, t := range targets {
				if t.Error != refusedAlong {
					fmt.Fprintf(os.Stderr, "✗ %s %s\n", portList(t.Ports), t.Error)
					if fix == "" {
						fix = t.Fix
					}
				}
			}
			fmt.Fprintf(os.Stderr, "  → nothing was stopped — %s\n", fix)
		}
	} else {
		for i := range targets {
			t := &targets[i]
			name, ports := inspect.ProcessName(t.Service), portList(t.Ports)
			switch {
			case opts.dryRun:
				t.Action = actionWouldStop
				if !asJSON {
					fmt.Printf("would stop %s (%s)\n", name, ports)
				}
			default:
				if err := stopOne(t.Service, opts.force); err != nil {
					t.Action, t.Error = actionFailed, err.Error()
					t.Fix = stopFix(t.Service, t.Ports[0])
					code = 1
					if !asJSON {
						fmt.Fprintf(os.Stderr, "✗ %s (%s): %v\n", name, ports, err)
						fmt.Fprintf(os.Stderr, "  → run `ports doctor %d` to see why\n", t.Ports[0])
					}
					continue
				}
				t.Action = actionStopped
				if !asJSON {
					fmt.Println(okMsg(t.Service, ports))
				}
			}
		}
	}

	for _, t := range skipped {
		if !asJSON {
			fmt.Fprintf(os.Stderr, "· skipped %s (%s): %s\n  → %s\n", inspect.ProcessName(t.Service), portList(t.Ports), t.Error, t.Fix)
		}
	}
	// skipped rows never sink a stop that did its job — but when they
	// were all there was, nothing was stopped
	if len(targets) == 0 && len(skipped) > 0 {
		code = 1
	}

	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(append(targets, skipped...)); err != nil {
			return 1
		}
	}
	return code
}

// splitUnowned sets aside the targets nothing here can stop — another
// user's listeners, seen with no pid — as skipped, each with the
// command that would stop it. The rest go on to be stopped, and to be
// judged by --expect: a skipped row is never stopped, so it can't sink
// the guard.
func splitUnowned(targets []stopResult) (stop, skipped []stopResult) {
	for _, t := range targets {
		if !t.Service.Unowned() {
			stop = append(stop, t)
			continue
		}
		t.Action = actionSkipped
		switch {
		case t.Service.Process == "docker-proxy":
			// sudo can't stop a proxy either — the container behind it
			// is what stops, and doctor says why it didn't resolve
			t.Error = "a docker-proxy whose container didn't resolve"
			t.Fix = fmt.Sprintf("ports doctor %d", t.Ports[0])
		case t.Service.Unit != "":
			t.Error = fmt.Sprintf("a systemd service running as %s", t.Service.User)
			t.Fix = "sudo systemctl stop " + t.Service.Unit
		default:
			t.Error = "another user's process — its pid is hidden from this scan"
			if t.Service.User != "" {
				t.Error = fmt.Sprintf("owned by %s — its pid is hidden from this scan", t.Service.User)
			}
			t.Fix = fmt.Sprintf("sudo ports free %d", t.Ports[0])
		}
		skipped = append(skipped, t)
	}
	return stop, skipped
}

// stopFix is the command that moves a failed stop along: the one that
// stops it where ports can't, or doctor when the reason needs a look.
func stopFix(s inspect.Service, port uint32) string {
	switch {
	case s.Windows:
		return fmt.Sprintf("taskkill /PID %d /F", s.WindowsPid)
	case !s.Known() && s.Unit != "":
		return "sudo systemctl stop " + s.Unit
	default:
		return fmt.Sprintf("ports doctor %d", port)
	}
}

// refusedAlong marks a target that matched --expect but was held back
// because another one didn't: the stop is all or nothing.
const refusedAlong = "matches, but was not stopped — another target did not"

// refuseUnexpected checks every target against the expected name, the
// way `ports kill` matches one. If any target misses, all are marked
// refused — the misses say what they are instead — and it reports true.
// A row whose owner is hidden can't match anything: the guard fails
// closed.
func refuseUnexpected(targets []stopResult, expect string) bool {
	refused := false
	for _, t := range targets {
		if !t.Service.MatchesName(expect) {
			refused = true
		}
	}
	if !refused {
		return false
	}
	for i := range targets {
		t := &targets[i]
		t.Action = actionRefused
		t.Fix = "drop --expect to stop regardless, or `ports doctor " + fmt.Sprint(t.Ports[0]) + "` to look closer"
		if t.Service.MatchesName(expect) {
			t.Error = refusedAlong
			continue
		}
		t.Error = fmt.Sprintf("is %s (%s), not %q", inspect.ProcessName(t.Service), inspect.ProjectLabel(t.Service), expect)
	}
	return true
}
