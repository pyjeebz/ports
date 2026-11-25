package routes

import (
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/pyjeebz/ports/internal/inspect"
)

func project(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func web(dir string, port uint32, framework string) inspect.Service {
	return inspect.Service{
		Port: port, Pid: 100 + int32(port), Addr: "127.0.0.1", Process: "node",
		Framework: framework, URL: "http://localhost", Project: dir, Cwd: dir,
	}
}

func TestNameable(t *testing.T) {
	dir := project(t, "shop")
	ok := web(dir, 3000, "Next.js")
	cases := map[string]func(*inspect.Service){
		"no URL (a database, a debugger port)": func(s *inspect.Service) { s.URL = "" },
		"another user's (no pid)":              func(s *inspect.Service) { s.Pid = 0 },
		"a Windows-side listener":              func(s *inspect.Service) { s.Windows = true },
		"a container":                          func(s *inspect.Service) { s.Container = "shop-web-1" },
		"no project":                           func(s *inspect.Service) { s.Project = "" },
	}
	if !Nameable(ok) {
		t.Fatal("a web dev server in a project isn't nameable")
	}
	for what, change := range cases {
		s := ok
		change(&s)
		if Nameable(s) {
			t.Errorf("%s is nameable", what)
		}
	}
}

func TestTableFollowsAMove(t *testing.T) {
	dir := project(t, "shop")
	var mu sync.Mutex
	current := []inspect.Service{web(dir, 3000, "Next.js")}
	table := NewTable(func() ([]inspect.Service, error) {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(current), nil
	}, nil)
	changes, err := table.Refresh()
	if err != nil || len(changes) != 1 || changes[0].Kind != "added" || changes[0].Route.Name != "shop" {
		t.Fatalf("first refresh: %+v, %v", changes, err)
	}

	// restarted on another port: same name, new port
	mu.Lock()
	current = []inspect.Service{web(dir, 3001, "Next.js")}
	mu.Unlock()
	time.Sleep(minGap)
	changes, _ = table.Refresh()
	if len(changes) != 1 || changes[0].Kind != "moved" || changes[0].Route.Port != 3001 {
		t.Fatalf("after the move: %+v", changes)
	}
	if r, ok := table.Lookup("shop"); !ok || r.Port != 3001 {
		t.Errorf("shop → %+v %v, want :3001", r, ok)
	}

	// stopped: the name is gone, and remembered as gone
	mu.Lock()
	current = nil
	mu.Unlock()
	time.Sleep(minGap)
	changes, _ = table.Refresh()
	if len(changes) != 1 || changes[0].Kind != "removed" {
		t.Fatalf("after the stop: %+v", changes)
	}
	if r, ok := table.Gone("shop"); !ok || r.Port != 3001 {
		t.Errorf("gone shop → %+v %v", r, ok)
	}
}

func TestLookupMissRescans(t *testing.T) {
	dir := project(t, "shop")
	scans := 0
	var current []inspect.Service
	table := NewTable(func() ([]inspect.Service, error) {
		scans++
		return current, nil
	}, nil)
	if _, err := table.Refresh(); err != nil {
		t.Fatal(err)
	}
	current = []inspect.Service{web(dir, 5173, "Vite")}
	time.Sleep(minGap)
	if _, ok := table.Lookup("shop"); !ok {
		t.Fatal("a server that started after the last refresh isn't found on a miss")
	}
	// a burst of misses for a name nobody has costs one scan
	before := scans
	for range 20 {
		table.Lookup("nope")
	}
	if scans-before > 1 {
		t.Errorf("20 misses cost %d scans, want at most 1", scans-before)
	}
}

func TestTargets(t *testing.T) {
	for addr, want := range map[string]string{
		"0.0.0.0":     "127.0.0.1:80",
		"::":          "[::1]:80",
		"::1":         "[::1]:80",
		"192.168.1.5": "192.168.1.5:80",
	} {
		if got := (Route{Addr: addr, Port: 80}).Targets()[0]; got != want {
			t.Errorf("%s: first target %s, want %s", addr, got, want)
		}
	}
}
