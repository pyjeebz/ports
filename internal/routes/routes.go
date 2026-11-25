// Package routes is the router's table: which name goes to which port
// right now. Routes are discovered, not declared — every web dev server
// the scan finds and can put a project to gets the name naming gives
// it, and keeps it across restarts because the name never came from the
// port. The table is rebuilt from a fresh scan every two seconds, and on
// a miss, so a server that moved is found where it went.
package routes

import (
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
}

// FromServices is the route table for a scan whose rows are named.
func FromServices(services []inspect.Service) map[string]Route {
	out := map[string]Route{}
	for _, s := range services {
		if s.Name == "" {
			continue
		}
		out[s.Name] = Route{
			Name: s.Name, Port: s.Port, Addr: s.Addr, Pid: s.Pid, Started: s.Started,
			Framework: inspect.ProcessName(s), Project: s.Project,
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
	scan   func() ([]inspect.Service, error)
	claims *naming.Claims

	mu     sync.RWMutex
	routes map[string]Route
	gone   map[string]Route // names that had a server this run, and lost it
	at     time.Time

	refresh sync.Mutex // one scan at a time
}

// minGap is the shortest time between two scans.
const minGap = 250 * time.Millisecond

// NewTable is a table filled by scan. claims, when not nil, are kept up
// to date as new projects appear.
func NewTable(scan func() ([]inspect.Service, error), claims *naming.Claims) *Table {
	return &Table{scan: scan, claims: claims, routes: map[string]Route{}, gone: map[string]Route{}}
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
	Name(services, t.claims)
	if t.claims != nil {
		_ = t.claims.Save() // a lost save only means the claim is made again
	}
	next := FromServices(services)

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
