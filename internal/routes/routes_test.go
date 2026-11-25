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
	}, nil, nil)
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
	}, nil, nil)
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

func TestResolveDeclaredWin(t *testing.T) {
	shop := project(t, "shop")
	plain := inspect.Service{Port: 4242, Pid: 9, Addr: "127.0.0.1", Process: "python3"} // no framework, no project
	services := []inspect.Service{web(shop, 3000, "Next.js"), web(shop, 4123, "Vite"), plain}
	got := Resolve(services, nil,
		Aliases{"stripe": 4242, "gone": 4999},
		[]Run{{Name: "docs", Port: 4123, Dir: shop}},
	)
	want := map[string]uint32{"shop": 3000, "docs": 4123, "stripe": 4242, "gone": 4999}
	if len(got) != len(want) {
		t.Fatalf("routes %v, want %v", got, want)
	}
	for name, port := range want {
		if got[name].Port != port {
			t.Errorf("%s → :%d, want :%d", name, got[name].Port, port)
		}
	}
	if services[1].Name != "docs" || services[2].Name != "stripe" {
		t.Errorf("rows named %q and %q, want the declared names on them", services[1].Name, services[2].Name)
	}
	if got["docs"].Project != shop {
		t.Errorf("a run's route keeps its project, got %q", got["docs"].Project)
	}
}

func TestResolveAliasOverRun(t *testing.T) {
	shop := project(t, "shop")
	services := []inspect.Service{web(shop, 4123, "Vite")}
	got := Resolve(services, nil, Aliases{"pinned": 4123}, []Run{{Name: "docs", Port: 4123}})
	if got["pinned"].Port != 4123 || services[0].Name != "pinned" {
		t.Errorf("alias didn't win: %v, row %q", got, services[0].Name)
	}
}

func TestResolveDeclaredNameTakesDiscovered(t *testing.T) {
	// an alias called shop for another port: the discovered shop gives
	// the name up rather than two servers answering to it
	shop := project(t, "shop")
	services := []inspect.Service{web(shop, 3000, "Next.js")}
	got := Resolve(services, nil, Aliases{"shop": 9000}, nil)
	if got["shop"].Port != 9000 || services[0].Name != "" {
		t.Errorf("shop → :%d, discovered row named %q; want the alias's :9000 and no name", got["shop"].Port, services[0].Name)
	}
}

func TestValidName(t *testing.T) {
	for _, ok := range []string{"shop", "api.shop", "fix-ui.shop", "a1"} {
		if !ValidName(ok) {
			t.Errorf("ValidName(%q) = false", ok)
		}
	}
	for _, bad := range []string{"", "Shop", "-x", "x-", "a..b", ".a", "a_b", "日本", "a.b."} {
		if ValidName(bad) {
			t.Errorf("ValidName(%q) = true", bad)
		}
	}
}

func TestLiveRunsDropsTheDead(t *testing.T) {
	t.Setenv("PORTS_STATE_DIR", t.TempDir())
	pid := int32(os.Getpid())
	started, err := ProcessStarted(pid)
	if err != nil {
		t.Skip(err)
	}
	if err := Register(Run{Name: "live", Port: 4001, Pid: pid, Started: started}); err != nil {
		t.Fatal(err)
	}
	// this pid, but a different start: the process that registered it is gone
	if err := Register(Run{Name: "stale", Port: 4002, Pid: pid + 1_000_000, Started: 1}); err != nil {
		t.Fatal(err)
	}
	runs := LiveRuns()
	if len(runs) != 1 || runs[0].Name != "live" {
		t.Fatalf("LiveRuns = %+v, want only live", runs)
	}
	if err := Unregister(pid); err != nil {
		t.Fatal(err)
	}
	if runs := LiveRuns(); len(runs) != 0 {
		t.Errorf("after Unregister: %+v", runs)
	}
}

func container(dir, project, service, framework string, port uint32) inspect.Service {
	return inspect.Service{
		Port: port, Process: "docker-proxy", Addr: "0.0.0.0", Framework: framework,
		Container: project + "-" + service + "-1", ContainerID: "c-" + service,
		ComposeProject: project, ComposeService: service, ComposeDir: dir, ProjectName: project,
	}
}

func TestComposeNames(t *testing.T) {
	shop := project(t, "shop")
	services := []inspect.Service{
		web(shop, 3000, "Next.js"),
		container(shop, "shop", "api", "", 8000),
		container(shop, "shop", "db", "PostgreSQL", 5432),
		container(shop, "shop", "minio", "MinIO", 9001),
		container(shop, "shop", "minio", "MinIO", 9000),
		container("", "other", "web", "nginx", 8081),
	}
	got := Resolve(services, nil, nil, nil)
	names := map[uint32]string{}
	for _, s := range services {
		names[s.Port] = s.Name
	}
	want := map[uint32]string{3000: "shop", 8000: "api.shop", 5432: "db.shop", 9000: "minio.shop", 9001: "minio-2.shop", 8081: "web.other"}
	for port, name := range want {
		if names[port] != name {
			t.Errorf(":%d named %q, want %q", port, names[port], name)
		}
	}
	if _, ok := got["db.shop"]; ok {
		t.Error("db.shop (PostgreSQL) got a route; it never speaks HTTP")
	}
	for _, name := range []string{"shop", "api.shop", "minio.shop", "web.other"} {
		if _, ok := got[name]; !ok {
			t.Errorf("%s has no route", name)
		}
	}
	if got["api.shop"].Targets()[0] != "127.0.0.1:8000" {
		t.Errorf("a published container is reached on loopback: %v", got["api.shop"].Targets())
	}
}

func TestComposeNameYieldsToHostApp(t *testing.T) {
	shop := project(t, "shop")
	if err := os.MkdirAll(filepath.Join(shop, "apps", "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	host := web(shop, 4000, "NestJS")
	host.Cwd = filepath.Join(shop, "apps", "api")
	services := []inspect.Service{web(shop, 3000, "Next.js"), host, container(shop, "shop", "api", "", 8000)}
	Resolve(services, nil, nil, nil)
	if services[1].Name != "api.shop" || services[2].Name != "api-2.shop" {
		t.Errorf("host api = %q, compose api = %q; want api.shop and api-2.shop", services[1].Name, services[2].Name)
	}
}
