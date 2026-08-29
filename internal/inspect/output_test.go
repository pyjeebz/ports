package inspect

import (
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout runs f with os.Stdout redirected and returns what f wrote.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	real, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	saved := os.Stdout
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		buf, _ := io.ReadAll(real) // read to EOF: one Read can race the writer
		done <- string(buf)
	}()
	f()
	os.Stdout = saved
	_ = w.Close()
	return <-done
}

// captureTree renders services and returns the table body (header off).
func captureTree(t *testing.T, services []Service) []string {
	t.Helper()
	out := captureStdout(t, func() { WriteTree(services) })
	lines := strings.Split(strings.TrimSpace(out), "\n")
	return lines[1:] // drop the header
}

func TestWriteTreeNestsChildren(t *testing.T) {
	services := []Service{
		{Port: 3000, Pid: 100, ParentPid: 90, Process: "next-server"},
		{Port: 5000, Pid: 101, ParentPid: 100, Process: "node"},
		{Port: 5173, Pid: 200, ParentPid: 180, Process: "vite"},
		{Port: 8080, Pid: 0, ParentPid: 1, Process: "?"}, // unknown owner: root even with a ppid
	}
	got := captureTree(t, services)
	// scan-ordered roots first (3000, then its child, then 5173, 8080)
	wantPorts := []string{"3000", "5000", "5173", "8080"}
	if len(got) != len(wantPorts) {
		t.Fatalf("got %d rows, want %d:\n%q", len(got), len(wantPorts), got)
	}
	for i, port := range wantPorts {
		if !strings.HasPrefix(got[i], port) {
			t.Errorf("row %d = %q, want port %s first", i, got[i], port)
		}
	}
	if !strings.Contains(got[1], "└─ node") {
		t.Errorf("child row not drawn as node: %q", got[1])
	}
	// the child's connector sits right of where the parent's content starts
	if child, parent := strings.Index(got[1], "└─"), strings.Index(got[0], "next-server"); child < parent {
		t.Errorf("child not indented under parent: %q under %q", got[1], got[0])
	}
}

func TestWriteTreeLastChildAndContinuation(t *testing.T) {
	services := []Service{
		{Port: 3000, Pid: 100, Process: "npm"},
		{Port: 3001, Pid: 101, ParentPid: 100, Process: "next"},
		{Port: 3002, Pid: 102, ParentPid: 101, Process: "worker"},
	}
	got := captureTree(t, services)
	if len(got) != len(services) {
		t.Fatalf("got %d rows, want %d:\n%q", len(got), len(services), got)
	}
	// next is npm's only child (└─); worker nests under next's └ with a
	// space continuation — the subtree skips │ past the last branch
	if !strings.Contains(got[1], "└─ next") || !strings.Contains(got[2], "   └─ worker") {
		t.Errorf("unexpected tree:\n%q", got)
	}
	if strings.Contains(got[2], "│") {
		t.Errorf("last-child subtree must not draw a │ continuation: %q", got[2])
	}
}

func TestWriteTreeContainersAndWindowsAreRoots(t *testing.T) {
	services := []Service{
		{Port: 3306, Pid: 0, ParentPid: 42, Process: "?", Container: "mysql", ContainerID: "abc"},
		{Port: 9999, Pid: 7, ParentPid: 5, Process: "app", Windows: true, WindowsPid: 7},
	}
	got := captureTree(t, services)
	if len(got) != 2 || strings.Contains(got[0], "└─") || strings.Contains(got[1], "└─") {
		t.Errorf("container and windows rows must be roots, got %q", got)
	}
}
