package cmd

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/pyjeebz/ports/internal/inspect"
)

// captureStdout runs f with os.Stdout and os.Stderr redirected into one
// pipe and returns what both wrote — stopAll reports success to stdout
// and failure to stderr, and the tests read the whole report. cmd's twin
// of inspect's helper, since the two packages can't share test files.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	real, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	savedOut, savedErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = w, w
	done := make(chan string, 1)
	go func() {
		buf, _ := io.ReadAll(real) // read to EOF: one Read can race the writer
		done <- string(buf)
	}()
	f()
	os.Stdout, os.Stderr = savedOut, savedErr
	_ = w.Close()
	return <-done
}

// swapStop replaces the stop seam with fn and records every service it
// was asked to stop. Tests get the recorder back.
func swapStop(t *testing.T, fn func(inspect.Service, bool) error) func() []inspect.Service {
	t.Helper()
	var stopped []inspect.Service
	saved := stopOne
	stopOne = func(s inspect.Service, force bool) error {
		stopped = append(stopped, s)
		return fn(s, force)
	}
	t.Cleanup(func() { stopOne = saved })
	return func() []inspect.Service { return stopped }
}

func okLine(s inspect.Service, ports string) string {
	return "stopped " + inspect.ProcessName(s) + " (" + ports + ")"
}

// One pid holding two ports is one process — stopAll must signal it once
// and name both ports in the line it prints.
func TestStopAllGroupsPortsByPid(t *testing.T) {
	rows := []inspect.Service{
		{Pid: 101, Port: 3000, Process: "node", Framework: "Next.js"},
		{Pid: 101, Port: 3001, Process: "node", Framework: "Next.js"},
	}
	stops := swapStop(t, func(inspect.Service, bool) error { return nil })
	var code int
	out := captureStdout(t, func() {
		code = stopAll(rows, stopOpts{}, func(s inspect.Service, ports string) string {
			return okLine(s, ports)
		})
	})

	if code != 0 {
		t.Errorf("code = %d, want 0", code)
	}
	if len(stops()) != 1 {
		t.Fatalf("stop called %d times, want 1 (one pid, two ports)", len(stops()))
	}
	if !strings.Contains(out, ":3000, :3001") {
		t.Errorf("line should name both ports: %q", out)
	}
}

// docker-proxy runs once per published port, but the container is what
// the user can stop — rows sharing a container id are one stop.
func TestStopAllGroupsPortsByContainer(t *testing.T) {
	rows := []inspect.Service{
		{Pid: 1, Port: 5432, Process: "docker-proxy", ContainerID: "abc123", Container: "shop-db"},
		{Pid: 2, Port: 5433, Process: "docker-proxy", ContainerID: "abc123", Container: "shop-db"},
	}
	stops := swapStop(t, func(inspect.Service, bool) error { return nil })
	out := captureStdout(t, func() {
		_ = stopAll(rows, stopOpts{}, okLine)
	})

	if len(stops()) != 1 {
		t.Fatalf("stop called %d times, want 1 (one container, two ports)", len(stops()))
	}
	if stops()[0].ContainerID != "abc123" {
		t.Errorf("stopped %q, want the container row", stops()[0].ContainerID)
	}
	if !strings.Contains(out, ":5432, :5433") {
		t.Errorf("line should name both ports: %q", out)
	}
}

// Another user's service has no pid to group by, but one unit is one
// stop — mysqld on :3306 and :33060 is a single line, not two.
func TestStopAllGroupsUnownedPortsByUnit(t *testing.T) {
	rows := []inspect.Service{
		{Port: 3306, Process: "mysqld", Framework: "MySQL", Unit: "mysql", User: "mysql"},
		{Port: 33060, Process: "mysqld", Framework: "MySQL", Unit: "mysql", User: "mysql"},
	}
	stops := swapStop(t, func(inspect.Service, bool) error { return nil })
	out := captureStdout(t, func() {
		_ = stopAll(rows, stopOpts{}, okLine)
	})
	if len(stops()) != 1 {
		t.Fatalf("stop called %d times, want 1 (one unit, two ports)", len(stops()))
	}
	if !strings.Contains(out, ":3306, :33060") {
		t.Errorf("line should name both ports: %q", out)
	}
}

// Two pids on one port (SO_REUSEPORT) are two processes — both stopped,
// neither collapsed into the other.
func TestStopAllKeepsDistinctPids(t *testing.T) {
	rows := []inspect.Service{
		{Pid: 201, Port: 8080, Process: "node", Framework: "Vite"},
		{Pid: 202, Port: 8080, Process: "node", Framework: "Vite"},
	}
	stops := swapStop(t, func(inspect.Service, bool) error { return nil })
	captureStdout(t, func() {
		_ = stopAll(rows, stopOpts{}, okLine)
	})
	if len(stops()) != 2 {
		t.Fatalf("stop called %d times, want 2 (two pids share the port)", len(stops()))
	}
}

// --dry-run is the preview contract: nothing is signalled, and the lines
// say "would stop" so a rerun without the flag is the only difference.
func TestStopAllDryRunSignalsNothing(t *testing.T) {
	rows := []inspect.Service{
		{Pid: 101, Port: 3000, Process: "node", Framework: "Next.js"},
	}
	// the seam records; if stopAll ignores dryRun it will call it
	stops := swapStop(t, func(inspect.Service, bool) error { return nil })
	var code int
	out := captureStdout(t, func() {
		code = stopAll(rows, stopOpts{dryRun: true}, okLine)
	})

	if len(stops()) != 0 {
		t.Fatalf("--dry-run signalled %d processes, want 0", len(stops()))
	}
	if !strings.Contains(out, "would stop") {
		t.Errorf("dry-run should say \"would stop\": %q", out)
	}
	if code != 0 {
		t.Errorf("code = %d, want 0", code)
	}
}

// A stop that fails reports the row it failed on and the doctor pointer,
// and the exit code says the command didn't finish the job.
func TestStopAllFailureExitsOne(t *testing.T) {
	rows := []inspect.Service{
		{Pid: 101, Port: 3000, Process: "node", Framework: "Next.js"},
		{Pid: 202, Port: 8000, Process: "python", Framework: "FastAPI"},
	}
	// the first refuses, the second stops: the report must show both
	swapStop(t, func(s inspect.Service, _ bool) error {
		if s.Pid == 101 {
			return errors.New("pid 101 survived SIGKILL")
		}
		return nil
	})
	var code int
	out := captureStdout(t, func() {
		code = stopAll(rows, stopOpts{force: true}, okLine)
	})

	if code != 1 {
		t.Errorf("code = %d, want 1 (one stop failed)", code)
	}
	// the failed line and its doctor pointer go to stderr — the capture
	// reads both streams as one report
	if !strings.Contains(out, "survived SIGKILL") || !strings.Contains(out, "ports doctor 3000") {
		t.Errorf("failure report incomplete: %q", out)
	}
}

// A flag another command owns names that command; a flag nobody owns
// stays cobra's plain "unknown flag".
func TestMisplacedFlag(t *testing.T) {
	err := misplacedFlag(killCmd, errors.New("unknown flag: --watch"))
	if want := "--watch doesn't apply to `ports kill` — it's a flag of `ports`, `ports list`"; err.Error() != want {
		t.Errorf("got %q, want %q", err, want)
	}
	err = misplacedFlag(freeCmd, errors.New("unknown shorthand flag: 'w' in -w"))
	if !strings.Contains(err.Error(), "-w doesn't apply to `ports free`") {
		t.Errorf("shorthand: %q", err)
	}
	err = misplacedFlag(rootCmd, errors.New("unknown flag: --force"))
	if !strings.Contains(err.Error(), "`ports free`, `ports kill`") {
		t.Errorf("root: %q", err)
	}
	if err := misplacedFlag(killCmd, errors.New("unknown flag: --nope")); err.Error() != "unknown flag: --nope" {
		t.Errorf("unowned flag: %q", err)
	}
}

// withJSON renders a test's stops as --format json.
func withJSON(t *testing.T) {
	t.Helper()
	saved := jsonOut
	jsonOut = true
	t.Cleanup(func() { jsonOut = saved })
}

// --expect that matches every target is no obstacle: the stop runs.
func TestStopAllExpectMatches(t *testing.T) {
	rows := []inspect.Service{{Pid: 101, Port: 3000, Process: "node", Framework: "Next.js", ProjectName: "shop"}}
	stops := swapStop(t, func(inspect.Service, bool) error { return nil })
	var code int
	captureStdout(t, func() { code = stopAll(rows, stopOpts{expect: "nextjs"}, okLine) })
	if code != 0 || len(stops()) != 1 {
		t.Errorf("code %d, %d stops — want 0 and 1", code, len(stops()))
	}
}

// The guard is all or nothing: one target that isn't what the caller
// expected stops none of them, and the report names what is there.
func TestStopAllExpectRefusesAll(t *testing.T) {
	rows := []inspect.Service{
		{Pid: 101, Port: 3000, Process: "node", Framework: "Next.js", ProjectName: "shop"},
		{Port: 3000, Process: "docker-proxy", Container: "shop-db", ContainerID: "abc", Framework: "PostgreSQL", Runtime: "docker"},
	}
	stops := swapStop(t, func(inspect.Service, bool) error { return nil })
	var code int
	out := captureStdout(t, func() { code = stopAll(rows, stopOpts{expect: "nextjs"}, okLine) })
	if len(stops()) != 0 {
		t.Fatalf("a refused stop signalled %d targets, want 0", len(stops()))
	}
	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	for _, want := range []string{`:3000 is PostgreSQL (docker: shop-db), not "nextjs"`, "nothing was stopped"} {
		if !strings.Contains(out, want) {
			t.Errorf("report %q missing %q", out, want)
		}
	}
	if strings.Contains(out, "Next.js") {
		t.Errorf("the matching target needs no line of its own: %q", out)
	}
}

// A row whose owner is hidden can't prove it is what was expected — the
// guard fails closed.
func TestStopAllExpectFailsClosed(t *testing.T) {
	rows := []inspect.Service{{Port: 3000, Process: "?"}}
	stops := swapStop(t, func(inspect.Service, bool) error { return nil })
	var code int
	captureStdout(t, func() { code = stopAll(rows, stopOpts{expect: "nextjs"}, okLine) })
	if code != 1 || len(stops()) != 0 {
		t.Errorf("code %d, %d stops — an anonymous row must be refused", code, len(stops()))
	}
}

func decodeResults(t *testing.T, out string) []stopResult {
	t.Helper()
	var results []stopResult
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("stop results are not JSON: %v\n%s", err, out)
	}
	return results
}

// --json reports one result per target: what happened, every port it
// held, the row itself, and for a failure the error and the next step.
func TestStopAllJSONResults(t *testing.T) {
	withJSON(t)
	rows := []inspect.Service{
		{Pid: 101, Port: 3000, Process: "node", Framework: "Next.js"},
		{Pid: 101, Port: 3001, Process: "node", Framework: "Next.js"},
		{Pid: 202, Port: 8000, Process: "python", Framework: "FastAPI"},
	}
	swapStop(t, func(s inspect.Service, _ bool) error {
		if s.Pid == 202 {
			return errors.New("pid 202 survived SIGKILL")
		}
		return nil
	})
	var code int
	out := captureStdout(t, func() { code = stopAll(rows, stopOpts{}, okLine) })
	results := decodeResults(t, out)
	if code != 1 || len(results) != 2 {
		t.Fatalf("code %d, %d results — want 1 and 2:\n%s", code, len(results), out)
	}
	if r := results[0]; r.Action != actionStopped || len(r.Ports) != 2 || r.Service.Pid != 101 {
		t.Errorf("first result: %+v", r)
	}
	if r := results[1]; r.Action != actionFailed || r.Error != "pid 202 survived SIGKILL" || r.Fix != "ports doctor 8000" {
		t.Errorf("second result: %+v", r)
	}
	if strings.Contains(out, "✓") || strings.Contains(out, "✗") {
		t.Errorf("JSON mode prints no lines: %q", out)
	}
}

func TestStopAllJSONRefusedAndDryRun(t *testing.T) {
	withJSON(t)
	swapStop(t, func(inspect.Service, bool) error { return nil })
	rows := []inspect.Service{
		{Pid: 101, Port: 3000, Process: "node", Framework: "Next.js"},
		{Pid: 202, Port: 3000, Process: "node", Framework: "Vite"},
	}
	out := captureStdout(t, func() { _ = stopAll(rows, stopOpts{expect: "vite"}, okLine) })
	results := decodeResults(t, out)
	if len(results) != 2 || results[0].Action != actionRefused || results[1].Action != actionRefused {
		t.Fatalf("refused results: %+v", results)
	}
	if !strings.Contains(results[0].Error, `not "vite"`) || results[1].Error != refusedAlong {
		t.Errorf("refusal reasons: %q / %q", results[0].Error, results[1].Error)
	}

	out = captureStdout(t, func() { _ = stopAll(rows[:1], stopOpts{dryRun: true}, okLine) })
	if r := decodeResults(t, out); len(r) != 1 || r[0].Action != actionWouldStop {
		t.Errorf("dry-run results: %+v", r)
	}
}

func TestStopFix(t *testing.T) {
	cases := []struct {
		s    inspect.Service
		want string
	}{
		{inspect.Service{Port: 3306, Process: "mysqld", Unit: "mysql", User: "mysql"}, "sudo systemctl stop mysql"},
		{inspect.Service{Port: 3000, Process: "node", Windows: true, WindowsPid: 8765}, "taskkill /PID 8765 /F"},
		{inspect.Service{Port: 3000, Pid: 101, Process: "node"}, "ports doctor 3000"},
	}
	for _, c := range cases {
		if got := stopFix(c.s, c.s.Port); got != c.want {
			t.Errorf("stopFix(%+v) = %q, want %q", c.s, got, c.want)
		}
	}
}

// A port argument reads like --port: 3000 or :3000. Anything numeric is
// a port or a usage error — never a name — so `wait 70000 --gone` can't
// report a service called "70000" free.
func TestPortArg(t *testing.T) {
	cases := []struct {
		arg    string
		port   uint32
		isPort bool
		bad    bool
	}{
		{arg: "3000", port: 3000, isPort: true},
		{arg: ":8791", port: 8791, isPort: true},
		{arg: "1", port: 1, isPort: true},
		{arg: "65535", port: 65535, isPort: true},
		{arg: "shop"},
		{arg: "nextjs"},
		{arg: "3000-api"},
		{arg: " 3000", port: 3000, isPort: true},
		{arg: "3000\r", port: 3000, isPort: true},
		{arg: "0", bad: true},
		{arg: "+3000", bad: true},
		{arg: "-3000", bad: true},
		{arg: "٣٠٠٠", bad: true},
		{arg: ":+3000", bad: true},
		{arg: "65536", bad: true},
		{arg: "70000", bad: true},
		{arg: "99999999999999999999", bad: true},
		{arg: ":", bad: true},
		{arg: ":shop", bad: true},
	}
	for _, c := range cases {
		port, isPort, err := portArg(c.arg)
		if c.bad {
			var ee exitError
			if !errors.As(err, &ee) || ee.code != 2 {
				t.Errorf("portArg(%q) = %d, %v, %v — want an exit-2 error", c.arg, port, isPort, err)
			}
			continue
		}
		if err != nil || port != c.port || isPort != c.isPort {
			t.Errorf("portArg(%q) = %d, %v, %v — want %d, %v", c.arg, port, isPort, err, c.port, c.isPort)
		}
	}

	if _, err := requirePort("shop"); err == nil {
		t.Error(`requirePort("shop") should refuse a name`)
	}
	if p, err := requirePort(":3000"); err != nil || p != 3000 {
		t.Errorf(`requirePort(":3000") = %d, %v — want 3000`, p, err)
	}
}

// kill by name sweeps up other users' listeners the scan sees with no
// pid. Nothing here can stop them, so they're skipped — reported with
// the command that would stop them — and never turn a stop that did its
// job into exit 1.
func TestStopAllSkipsUnowned(t *testing.T) {
	rows := []inspect.Service{
		{Pid: 101, Port: 3000, Process: "node", Framework: "Next.js", ProjectName: "shop"},
		{Port: 5432, Process: "postgres", Framework: "PostgreSQL", User: "postgres", Unit: "postgresql"},
		{Port: 6379, Process: "?", User: "root"},
	}
	stops := swapStop(t, func(inspect.Service, bool) error { return nil })
	var code int
	out := captureStdout(t, func() { code = stopAll(rows, stopOpts{skipUnowned: true}, okLine) })
	if code != 0 {
		t.Errorf("code = %d, want 0 — the stoppable target stopped", code)
	}
	if len(stops()) != 1 || stops()[0].Pid != 101 {
		t.Fatalf("stopped %v, want only pid 101", stops())
	}
	for _, want := range []string{"skipped PostgreSQL (:5432)", "sudo systemctl stop postgresql", "skipped ? (:6379)", "sudo ports free 6379"} {
		if !strings.Contains(out, want) {
			t.Errorf("report %q missing %q", out, want)
		}
	}
}

// When skipped rows are all there was, nothing was stopped: exit 1.
func TestStopAllOnlyUnownedExitsOne(t *testing.T) {
	rows := []inspect.Service{{Port: 5432, Process: "postgres", User: "postgres", Unit: "postgresql"}}
	stops := swapStop(t, func(inspect.Service, bool) error { return nil })
	var code int
	captureStdout(t, func() { code = stopAll(rows, stopOpts{skipUnowned: true}, okLine) })
	if code != 1 || len(stops()) != 0 {
		t.Errorf("code %d, %d stops — want 1 and 0", code, len(stops()))
	}
}

// --expect judges what would be stopped; a skipped row is never
// stopped, so it can't sink a guarded stop — and is still never stopped.
func TestStopAllExpectIgnoresSkipped(t *testing.T) {
	rows := []inspect.Service{
		{Pid: 101, Port: 3000, Process: "node", Framework: "Next.js", ProjectName: "shop"},
		{Port: 6379, Process: "?", User: "root"},
	}
	stops := swapStop(t, func(inspect.Service, bool) error { return nil })
	var code int
	captureStdout(t, func() { code = stopAll(rows, stopOpts{skipUnowned: true, expect: "shop"}, okLine) })
	if code != 0 || len(stops()) != 1 || stops()[0].Pid != 101 {
		t.Errorf("code %d, stops %v — want 0 and only pid 101", code, stops())
	}
}

// In JSON a skipped row is a result like any other: action "skipped",
// why, and the command that would stop it.
func TestStopAllSkippedJSON(t *testing.T) {
	withJSON(t)
	rows := []inspect.Service{
		{Pid: 101, Port: 3000, Process: "node"},
		{Port: 5432, Process: "postgres", User: "postgres", Unit: "postgresql"},
	}
	swapStop(t, func(inspect.Service, bool) error { return nil })
	var code int
	out := captureStdout(t, func() { code = stopAll(rows, stopOpts{skipUnowned: true}, okLine) })
	results := decodeResults(t, out)
	if code != 0 || len(results) != 2 {
		t.Fatalf("code %d, %d results:\n%s", code, len(results), out)
	}
	r := results[1]
	if r.Action != actionSkipped || r.Fix != "sudo systemctl stop postgresql" || r.Error == "" || r.Ports[0] != 5432 {
		t.Errorf("skipped result: %+v", r)
	}
}
