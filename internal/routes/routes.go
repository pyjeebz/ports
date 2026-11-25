// Package routes is the router's table: which name goes to which port
// right now. Routes are discovered, not declared — every web dev server
// the scan finds and can put a project to gets the name naming gives
// it, and keeps it across restarts because the name never came from the
// port. The table is rebuilt from a fresh scan every two seconds, and on
// a miss, so a server that moved is found where it went.
package routes

import (
	"cmp"
	"net"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/pyjeebz/ports/internal/naming"
)

// Route is one name and the server holding it: the port, the address it
// listens on, and the process — pid and start time together, so a pid
// reused by some other program never inherits the route.
type Route struct {
	Name      string    `json:"name"`
	Port      uint32    `json:"port"`
	Addr      string    `json:"addr"`
	Pid       int32     `json:"pid"`
	Started   time.Time `json:"started,omitzero"`
	Framework string    `json:"framework,omitempty"`
	Project   string    `json:"project,omitempty"`
}

// Targets are the addresses to reach the route's server at, in order. A
// loopback or wildcard bind is tried on both loopbacks: a dev server
// bound to "localhost" may hold only ::1 (Node 17+ resolves it that way)
// or only 127.0.0.1.
func (r Route) Targets() []string {
	port := strconv.Itoa(int(r.Port))
	switch r.Addr {
	case "::", "::1":
		return []string{net.JoinHostPort("::1", port), net.JoinHostPort("127.0.0.1", port)}
	case "", "0.0.0.0", "127.0.0.1", "*":
		return []string{net.JoinHostPort("127.0.0.1", port), net.JoinHostPort("::1", port)}
	default:
		return []string{net.JoinHostPort(r.Addr, port)}
	}
}

// Nameable reports whether a row can be given a name: a web dev server
// on the port its app serves, run by a process of this machine's that
// the scan can see. Containers and Windows-side listeners wait for their
// own way in; another user's server is never routed to.
func Nameable(s inspect.Service) bool {
	return s.URL != "" && s.Known() && !s.Windows && s.Container == "" && s.Project != ""
}

// Name gives each nameable row its Name, all at once — names depend on
// what else runs in the same project. claims may be nil (no memory of
// earlier clashes, as in tests).
func Name(services []inspect.Service, claims *naming.Claims) {
	var apps []naming.App
	var at []int
	for i, s := range services {
		if !Nameable(s) {
			continue
		}
		apps = append(apps, naming.App{Project: s.Project, Cwd: s.Cwd, Framework: s.Framework, Port: s.Port})
		at = append(at, i)
	}
	for k, name := range naming.Names(apps, claims) {
		services[at[k]].Name = name
	}
	nameContainers(services, claims)
}

// nameContainers names each published compose container after its
// service, inside the project it was started from: web in ~/shop is
// web.shop, db is db.shop — the same base the host apps get. A container
// publishing several ports names the lowest; the rest count up
// (minio-2.shop). A name a host app already has goes to the host app.
func nameContainers(services []inspect.Service, claims *naming.Claims) {
	taken := map[string]bool{}
	for _, s := range services {
		if s.Name != "" {
			taken[s.Name] = true
		}
	}
	var rows []int
	for i, s := range services {
		if s.ComposeService != "" && !s.Windows {
			rows = append(rows, i)
		}
	}
	slices.SortStableFunc(rows, func(a, b int) int { return cmp.Compare(services[a].Port, services[b].Port) })
	scopes := map[string]string{}
	for _, i := range rows {
		s := &services[i]
		key := s.ComposeDir + "\x00" + s.ComposeProject
		scope, ok := scopes[key]
		if !ok {
			scope = composeScope(*s, claims)
			scopes[key] = scope
		}
		service := naming.Label(s.ComposeService)
		if scope == "" || service == "" {
			continue
		}
		name := service + "." + scope
		for n := 2; taken[name]; n++ {
			name = service + "-" + strconv.Itoa(n) + "." + scope
		}
		taken[name] = true
		s.Name = name
	}
}

// composeScope is the project part of a compose service's name: the
// project the compose file sits in, named as ports names any project,
// else the compose project's own name.
func composeScope(s inspect.Service, claims *naming.Claims) string {
	if s.ComposeDir != "" {
		project := inspect.DetectProject(s.ComposeDir)
		if project == "" {
			project = s.ComposeDir
		}
		if scope := naming.Scope(project, s.ComposeDir, claims); scope != "" {
			return scope
		}
	}
	return naming.Label(s.ComposeProject)
}

// urlHost is the host a URL reaches a listener by: localhost for a
// wildcard or loopback bind, the bound address otherwise.
func urlHost(addr string) string {
	switch addr {
	case "", "0.0.0.0", "::", "127.0.0.1", "::1", "*":
		return "localhost"
	}
	return strings.Replace(addr, "%", "%25", 1)
}

// notHTTP are the engines a container commonly runs that speak their own
// protocol, never HTTP: a compose db service gets a name to stop and
// wait on, but no route — db.shop.localhost would only ever be an error.
var notHTTP = map[string]bool{
	"PostgreSQL": true, "MySQL": true, "MariaDB": true, "Redis": true,
	"Valkey": true, "Dragonfly": true, "MongoDB": true, "Memcached": true,
	"NATS": true,
}

// Routable reports whether a named row should be reachable through the
// router: every host app ports names is a web server already; a
// container is unless it runs an engine that never speaks HTTP.
func Routable(s inspect.Service) bool {
	return s.Name != "" && !notHTTP[s.Framework]
}

// Resolve names a scan's rows and returns the route table: discovered
// names first, then the declared ones laid over them by port — a ports
// run registration, then an alias, each naming whatever listens on its
// port, framework or not. A discovered name that a declared one took for
// another port is dropped. A declared name whose port nobody holds still
// routes, to a page saying it isn't running.
func Resolve(services []inspect.Service, claims *naming.Claims, aliases Aliases, runs []Run) map[string]Route {
	Name(services, claims)

	declared := map[string]uint32{}
	for _, r := range runs {
		declared[r.Name] = r.Port
	}
	for name, port := range aliases {
		declared[name] = port
	}
	byPort := map[uint32]string{}
	for _, r := range runs {
		byPort[r.Port] = r.Name
	}
	for name, port := range aliases {
		byPort[port] = name
	}
	for i := range services {
		s := &services[i]
		if s.Windows {
			continue
		}
		if name, ok := byPort[s.Port]; ok {
			s.Name = name
			if s.URL == "" && !notHTTP[s.Framework] {
				// named on purpose, framework or not: it's a web server
				// to whoever named it, so it gets its own address too
				s.URL = "http://" + net.JoinHostPort(urlHost(s.Addr), strconv.Itoa(int(s.Port)))
			}
			continue
		}
		if port, ok := declared[s.Name]; ok && port != s.Port {
			s.Name = ""
		}
	}

	out := map[string]Route{}
	for _, s := range services {
		if s.Name == "" || (byPort[s.Port] == "" && !Routable(s)) {
			continue
		}
		if _, ok := out[s.Name]; ok && byPort[s.Port] == "" {
			continue // the declared row keeps the name
		}
		out[s.Name] = Route{
			Name: s.Name, Port: s.Port, Addr: s.Addr, Pid: s.Pid, Started: s.Started,
			Framework: inspect.ProcessName(s), Project: s.Project,
		}
	}
	for name, port := range declared {
		if _, ok := out[name]; !ok {
			out[name] = Route{Name: name, Port: port, Addr: "127.0.0.1"}
		}
	}
	for _, r := range runs {
		if rt, ok := out[r.Name]; ok && rt.Project == "" {
			rt.Project = r.Dir
			out[r.Name] = rt
		}
	}
	return out
}

// Change is a route that appeared, moved or went away in a refresh.
type Change struct {
	Route Route
	Kind  string // "added", "moved" or "removed"
}

// Table holds the live routes. It refreshes itself only when asked —
// Run on a ticker, Lookup on a miss — and never more often than minGap,
// so a burst of requests for an unknown name costs one scan, not one
// each.
type Table struct {
	scan     func() ([]inspect.Service, error)
	claims   *naming.Claims
	declared Declared

	mu     sync.RWMutex
	routes map[string]Route
	gone   map[string]Route // names that had a server this run, and lost it
	at     time.Time

	refresh sync.Mutex // one scan at a time
}

// minGap is the shortest time between two scans.
const minGap = 250 * time.Millisecond

// NewTable is a table filled by scan, with declared names laid over it
// (nil for discovery alone). claims, when not nil, are kept up to date as
// new projects appear.
func NewTable(scan func() ([]inspect.Service, error), claims *naming.Claims, declared Declared) *Table {
	return &Table{scan: scan, claims: claims, declared: declared, routes: map[string]Route{}, gone: map[string]Route{}}
}

// Refresh rescans and replaces the routes, reporting what changed. A
// refresh asked for within minGap of the last returns no changes.
func (t *Table) Refresh() ([]Change, error) {
	t.refresh.Lock()
	defer t.refresh.Unlock()
	t.mu.RLock()
	fresh := time.Since(t.at) < minGap
	t.mu.RUnlock()
	if fresh {
		return nil, nil
	}

	services, err := t.scan()
	if err != nil {
		return nil, err
	}
	var aliases Aliases
	var runs []Run
	if t.declared != nil {
		aliases, runs = t.declared()
	}
	next := Resolve(services, t.claims, aliases, runs)
	if t.claims != nil {
		_ = t.claims.Save() // a lost save only means the claim is made again
	}

	t.mu.Lock()
	prev := t.routes
	t.routes, t.at = next, time.Now()
	for name, r := range prev {
		if _, ok := next[name]; !ok {
			t.gone[name] = r
		}
	}
	for name := range next {
		delete(t.gone, name)
	}
	t.mu.Unlock()
	return diff(prev, next), nil
}

func diff(prev, next map[string]Route) []Change {
	var out []Change
	for name, r := range next {
		old, ok := prev[name]
		switch {
		case !ok:
			out = append(out, Change{Route: r, Kind: "added"})
		case old.Port != r.Port || old.Pid != r.Pid:
			out = append(out, Change{Route: r, Kind: "moved"})
		}
	}
	for name, r := range prev {
		if _, ok := next[name]; !ok {
			out = append(out, Change{Route: r, Kind: "removed"})
		}
	}
	slices.SortFunc(out, func(a, b Change) int { return strings.Compare(a.Route.Name, b.Route.Name) })
	return out
}

// Lookup is the route for name. A miss rescans first — the server may
// have started a moment ago — and answers from the fresh table.
func (t *Table) Lookup(name string) (Route, bool) {
	t.mu.RLock()
	r, ok := t.routes[name]
	t.mu.RUnlock()
	if ok {
		return r, true
	}
	if _, err := t.Refresh(); err != nil {
		return Route{}, false
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	r, ok = t.routes[name]
	return r, ok
}

// Gone is the route a name had before its server stopped, so the router
// can say "shop isn't running" instead of "nothing is called shop".
func (t *Table) Gone(name string) (Route, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	r, ok := t.gone[name]
	return r, ok
}

// All is every route, sorted by name.
func (t *Table) All() []Route {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]Route, 0, len(t.routes))
	for _, r := range t.routes {
		out = append(out, r)
	}
	slices.SortFunc(out, func(a, b Route) int { return strings.Compare(a.Name, b.Name) })
	return out
}

// Run refreshes the table every interval until stop closes, handing each
// refresh's changes to onChange.
func (t *Table) Run(interval time.Duration, stop <-chan struct{}, onChange func([]Change)) {
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		select {
		case <-stop:
			return
		case <-tick.C:
			if changes, err := t.Refresh(); err == nil && len(changes) > 0 && onChange != nil {
				onChange(changes)
			}
		}
	}
}
