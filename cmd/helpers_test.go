package cmd

import (
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
		code = stopAll(rows, false, false, func(s inspect.Service, ports string) string {
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
		_ = stopAll(rows, false, false, okLine)
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

// Two pids on one port (SO_REUSEPORT) are two processes — both stopped,
// neither collapsed into the other.
func TestStopAllKeepsDistinctPids(t *testing.T) {
	rows := []inspect.Service{
		{Pid: 201, Port: 8080, Process: "node", Framework: "Vite"},
		{Pid: 202, Port: 8080, Process: "node", Framework: "Vite"},
	}
	stops := swapStop(t, func(inspect.Service, bool) error { return nil })
	captureStdout(t, func() {
		_ = stopAll(rows, false, false, okLine)
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
		code = stopAll(rows, false, true, okLine)
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
		code = stopAll(rows, true, false, okLine)
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