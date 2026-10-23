package inspect

import (
	"strings"
	"testing"
	"time"

	gnet "github.com/shirou/gopsutil/v4/net"
)

// summarize compresses findings to "check:status" pairs — the sequence
// is the contract; messages are spot-checked where they carry the news.
func summarize(findings []Finding) []string {
	var out []string
	for _, f := range findings {
		out = append(out, f.Check+":"+f.Status)
	}
	return out
}

func equal(a, b []string) bool {
	return strings.Join(a, "|") == strings.Join(b, "|")
}

func listenConn(port uint32) gnet.ConnectionStat {
	return gnet.ConnectionStat{Status: "LISTEN", Laddr: gnet.Addr{Port: port}}
}

func connOn(port uint32, status string) gnet.ConnectionStat {
	return gnet.ConnectionStat{Status: status, Laddr: gnet.Addr{Port: port}}
}

func devRow(port uint32, pid int32) Service {
	return Service{Port: port, Pid: pid, Process: "node", Addr: "127.0.0.1", Cwd: "/home/dev/shop", Project: "/home/dev/shop", ProjectName: "shop"}
}

func healthyFacts() rowFacts {
	return rowFacts{states: []string{"R"}, fresh: true, exists: true}
}

func TestDiagnosePort(t *testing.T) {
	probeOK := RuntimeProbe{Runtime: "docker", Socket: "/var/run/docker.sock", Status: RuntimeOK}
	probeDenied := RuntimeProbe{Runtime: "docker", Socket: "/var/run/docker.sock", Status: RuntimePermission}

	cases := []struct {
		name       string
		rows       []Service
		facts      []rowFacts
		conns      []gnet.ConnectionStat
		dialOpen   bool
		winMirror  bool
		servesHTTP bool
		probe      RuntimeProbe
		want       []string
	}{
		{
			name:  "healthy dev server",
			rows:  []Service{devRow(3000, 48391)},
			facts: []rowFacts{healthyFacts()},
			conns: []gnet.ConnectionStat{listenConn(3000)},
			probe: probeOK,
			want:  []string{"listener:ok", "state:ok", "pid-freshness:ok", "sockets:ok"},
		},
		{
			name:       "healthy and speaking HTTP",
			rows:       []Service{devRow(3000, 48391)},
			facts:      []rowFacts{healthyFacts()},
			conns:      []gnet.ConnectionStat{listenConn(3000)},
			servesHTTP: true,
			probe:      probeOK,
			want:       []string{"listener:ok", "http:ok", "state:ok", "pid-freshness:ok", "sockets:ok"},
		},
		{
			// a non-HTTP listener (postgres, redis) stays silent — every
			// existing case doubles as this contract with servesHTTP false
			name:  "healthy but not HTTP",
			rows:  []Service{devRow(3000, 48391)},
			facts: []rowFacts{healthyFacts()},
			conns: []gnet.ConnectionStat{listenConn(3000)},
			probe: probeOK,
			want:  []string{"listener:ok", "state:ok", "pid-freshness:ok", "sockets:ok"},
		},
		{
			name:     "healthy with a live client",
			rows:     []Service{devRow(3000, 48391)},
			facts:    []rowFacts{healthyFacts()},
			conns:    []gnet.ConnectionStat{listenConn(3000), connOn(3000, "ESTABLISHED")},
			probe:    probeOK,
			want:     []string{"listener:ok", "state:ok", "pid-freshness:ok", "sockets:ok"},
		},
		{
			name: "no listener, nothing holding it",
			probe: probeOK,
			want: []string{"listener:fail"},
		},
		{
			name: "no listener but the dial answers",
			probe: probeOK, dialOpen: true,
			want: []string{"listener:fail", "listener:warn"},
		},
		{
			name:  "no listener, TIME_WAIT explains it",
			conns: []gnet.ConnectionStat{connOn(3000, "TIME_WAIT"), connOn(3000, "TIME_WAIT")},
			probe: probeOK,
			want:  []string{"listener:fail", "sockets:warn"},
		},
		{
			name:  "no listener, CLOSE_WAIT leaking",
			conns: []gnet.ConnectionStat{connOn(3000, "CLOSE_WAIT")},
			probe: probeOK,
			want:  []string{"listener:fail", "sockets:warn"},
		},
		{
			name: "SO_REUSEPORT: two owners",
			rows: []Service{devRow(3000, 100), devRow(3000, 200)},
			facts: []rowFacts{
				healthyFacts(),
				{states: []string{"S"}, fresh: true, exists: true},
			},
			conns: []gnet.ConnectionStat{listenConn(3000), listenConn(3000)},
			probe: probeOK,
			want:  []string{"listener:ok", "multiplicity:warn", "state:ok", "pid-freshness:ok", "sockets:ok"},
		},
		{
			name:  "state D survived the SIGKILL",
			rows:  []Service{devRow(3000, 48391)},
			facts: []rowFacts{{states: []string{"D"}, fresh: true, exists: true}},
			conns: []gnet.ConnectionStat{listenConn(3000)},
			probe: probeOK,
			want:  []string{"listener:ok", "state:fail", "pid-freshness:ok", "sockets:ok"},
		},
		{
			name:  "zombie: exited but unreaped",
			rows:  []Service{devRow(3000, 48391)},
			facts: []rowFacts{{states: []string{"Z"}, fresh: true, exists: true}},
			conns: []gnet.ConnectionStat{listenConn(3000)},
			probe: probeOK,
			want:  []string{"listener:ok", "state:warn", "pid-freshness:ok", "sockets:ok"},
		},
		{
			name:  "suspended process",
			rows:  []Service{devRow(3000, 48391)},
			facts: []rowFacts{{states: []string{"T"}, fresh: true, exists: true}},
			conns: []gnet.ConnectionStat{listenConn(3000)},
			probe: probeOK,
			want:  []string{"listener:ok", "state:warn", "pid-freshness:ok", "sockets:ok"},
		},
		{
			name:  "pid reuse: the row is stale",
			rows:  []Service{devRow(3000, 48391)},
			facts: []rowFacts{{states: []string{"R"}, fresh: false, exists: true}},
			conns: []gnet.ConnectionStat{listenConn(3000)},
			probe: probeOK,
			want:  []string{"listener:ok", "state:ok", "pid-freshness:fail", "sockets:ok"},
		},
		{
			name:  "process gone before doctor looked",
			rows:  []Service{devRow(3000, 48391)},
			facts: []rowFacts{{fresh: true, exists: false}},
			conns: []gnet.ConnectionStat{listenConn(3000)},
			probe: probeOK,
			want:  []string{"listener:ok", "state:warn", "sockets:ok"},
		},
		{
			name:  "another user's process",
			rows:  []Service{{Port: 3000, Pid: 0, Process: "?", Addr: "*"}},
			facts: []rowFacts{{fresh: true}},
			conns: []gnet.ConnectionStat{listenConn(3000)},
			probe: probeOK,
			want:  []string{"listener:ok", "owner:warn", "sockets:ok"},
		},
		{
			name:  "docker-proxy with no container behind it",
			rows:  []Service{{Port: 3000, Pid: 1, Process: "docker-proxy", Addr: "0.0.0.0"}},
			facts: []rowFacts{{states: []string{"S"}, fresh: true, exists: true}},
			conns: []gnet.ConnectionStat{listenConn(3000)},
			probe: probeOK,
			want:  []string{"listener:ok", "container:fail", "sockets:ok"},
		},
		{
			name:  "container row with the engine unreachable",
			rows:  []Service{{Port: 5432, Pid: 1, Process: "docker-proxy", Container: "shop-db", ContainerID: "abc123", Addr: "0.0.0.0", Cwd: "/"}},
			facts: []rowFacts{{states: []string{"S"}, fresh: true, exists: true}},
			conns: []gnet.ConnectionStat{listenConn(5432)},
			probe: probeDenied,
			want:  []string{"listener:ok", "state:ok", "pid-freshness:ok", "sockets:ok", "runtime:warn"},
		},
		{
			name:  "plain dev row never probes the engine",
			rows:  []Service{devRow(3000, 48391)},
			facts: []rowFacts{healthyFacts()},
			conns: []gnet.ConnectionStat{listenConn(3000)},
			probe: probeDenied, // what Diagnose would have produced; must not be reported
			want:  []string{"listener:ok", "state:ok", "pid-freshness:ok", "sockets:ok"},
		},
		{
			name:  "windows-side listener",
			rows:  []Service{{Port: 3000, Process: "node", Windows: true, WindowsPid: 4242}},
			probe: probeOK,
			want:  []string{"listener:ok", "windows:fail", "sockets:ok"},
		},
		{
			name:      "mirrored on both sides of WSL",
			rows:      []Service{devRow(3000, 48391)},
			facts:     []rowFacts{healthyFacts()},
			conns:     []gnet.ConnectionStat{listenConn(3000)},
			winMirror: true,
			probe:     probeOK,
			want:      []string{"listener:ok", "state:ok", "pid-freshness:ok", "sockets:ok", "mirror:warn"},
		},
		{
			name:  "system daemon: no cwd, no state to read",
			rows:  []Service{{Port: 3000, Pid: 999, Process: "rpcbind", Addr: "0.0.0.0"}},
			facts: []rowFacts{{fresh: true, exists: true}},
			conns: []gnet.ConnectionStat{listenConn(3000)},
			probe: probeOK,
			want:  []string{"listener:ok", "pid-freshness:ok", "cwd:warn", "sockets:ok"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := diagnosePort(3000, c.rows, c.facts, c.conns, c.dialOpen, c.winMirror, c.servesHTTP, c.probe)
			if !equal(summarize(got), c.want) {
				t.Errorf("diagnosePort:\n got  %v\n want %v", summarize(got), c.want)
			}
		})
	}
}

func TestDiagnosePortMessages(t *testing.T) {
	probeOK := RuntimeProbe{Runtime: "docker", Socket: "/var/run/docker.sock", Status: RuntimeOK}
	port := uint32(3000)
	listen := []gnet.ConnectionStat{listenConn(port)}

	t.Run("state D says why SIGKILL didn't land", func(t *testing.T) {
		f := diagnosePort(port, []Service{devRow(port, 7)}, []rowFacts{{states: []string{"D"}, fresh: true, exists: true}}, listen, false, false, false, probeOK)
		if !contains(f, "state", "uninterruptible") || !contains(f, "state", "SIGKILL") {
			t.Errorf("state D finding doesn't explain the unkillable process: %v", f)
		}
	})

	t.Run("stale pid names reuse", func(t *testing.T) {
		f := diagnosePort(port, []Service{devRow(port, 7)}, []rowFacts{{states: []string{"R"}, fresh: false, exists: true}}, listen, false, false, false, probeOK)
		if !contains(f, "pid-freshness", "another process") {
			t.Errorf("stale pid finding doesn't name the reuse: %v", f)
		}
	})

	t.Run("windows fix uses the windows pid", func(t *testing.T) {
		f := diagnosePort(port, []Service{{Port: port, Process: "node", Windows: true, WindowsPid: 4242}}, nil, listen, false, false, false, probeOK)
		if !contains(f, "windows", "taskkill /PID 4242") {
			t.Errorf("windows fix doesn't carry the pid: %v", f)
		}
	})

	t.Run("the http finding hands over the verified URL", func(t *testing.T) {
		f := diagnosePort(port, []Service{devRow(port, 7)}, []rowFacts{healthyFacts()}, listen, false, false, true, probeOK)
		if !contains(f, "http", "http://localhost:3000") {
			t.Errorf("http finding doesn't carry the URL: %v", f)
		}
	})

	t.Run("TIME_WAIT finding says how long", func(t *testing.T) {
		f := diagnosePort(port, nil, nil, []gnet.ConnectionStat{connOn(port, "TIME_WAIT")}, false, false, false, probeOK)
		if !contains(f, "sockets", "TIME_WAIT") || !contains(f, "sockets", "60s") {
			t.Errorf("TIME_WAIT finding doesn't say how long it lasts: %v", f)
		}
	})

	t.Run("permission denied says whose group", func(t *testing.T) {
		f := diagnosePort(port, []Service{{Port: port, Pid: 1, Process: "docker-proxy", Container: "x", ContainerID: "a"}}, []rowFacts{{states: []string{"S"}, fresh: true, exists: true}}, listen, false, false, false,
			RuntimeProbe{Runtime: "docker", Socket: "/x.sock", Status: RuntimePermission})
		if !contains(f, "runtime", "group") {
			t.Errorf("permission finding doesn't suggest the group fix: %v", f)
		}
	})

	t.Run("an engine with no well-known path is named, not guessed", func(t *testing.T) {
		f := diagnosePort(port, []Service{{Port: port, Pid: 1, Process: "docker-proxy", Container: "x", ContainerID: "a"}}, []rowFacts{{states: []string{"S"}, fresh: true, exists: true}}, listen, false, false, false,
			RuntimeProbe{Runtime: "nerdctl", Socket: "", Status: RuntimeNoSocket})
		if !contains(f, "runtime", "nerdctl") || contains(f, "runtime", "found at ") {
			t.Errorf("socket-less runtime finding must name the engine without a path: %v", f)
		}
	})
}

// A mirror needs both sides: a Windows row alone is not a mirror, and a
// local row alone is not either. The netstat fixture is German so the
// check rides the locale-proof parse on its way to windowsHolds.
func TestMirrorHeldNeedsBothSides(t *testing.T) {
	const winNetstat = "\r\n  TCP    0.0.0.0:3000           0.0.0.0:0              ABHÖREN          8765\r\n"
	const winTasklist = `"node.exe","8765","Console","1","98,304 K"` + "\r\n"
	old := runWin
	runWin = func(exe string, _ ...string) ([]byte, error) {
		if exe == "netstat.exe" {
			return []byte(winNetstat), nil
		}
		return []byte(winTasklist), nil
	}
	t.Cleanup(func() { runWin = old })

	windowsOnly := []Service{{Port: 3000, Process: "node", Windows: true, WindowsPid: 8765}}
	if mirrorHeld(windowsOnly, 3000) {
		t.Error("a Windows row alone is not a mirror — the Linux twin is what makes it one")
	}

	both := append([]Service{devRow(3000, 48391)}, windowsOnly...)
	if !mirrorHeld(both, 3000) {
		t.Error("both sides hold the port: a mirror")
	}
	if mirrorHeld(both, 3001) {
		t.Error("a different port is not held on the Windows side")
	}
}

func contains(findings []Finding, check, substr string) bool {
	for _, f := range findings {
		if f.Check == check && strings.Contains(f.Message+" "+f.Fix, substr) {
			return true
		}
	}
	return false
}

func TestDiagnoseMachine(t *testing.T) {
	probeOK := RuntimeProbe{Runtime: "docker", Socket: "/var/run/docker.sock", Status: RuntimeOK}
	probeDenied := RuntimeProbe{Runtime: "docker", Socket: "/x.sock", Status: RuntimePermission}
	probeNone := RuntimeProbe{Runtime: "docker", Socket: "/var/run/docker.sock", Status: RuntimeNoSocket}

	cases := []struct {
		name     string
		services []Service
		probe    RuntimeProbe
		wsl      bool
		want     []string
	}{
		{
			name:     "quiet machine, no engine",
			services: []Service{devRow(3000, 1)},
			probe:    probeNone,
			want:     []string{"machine:ok"},
		},
		{
			name:     "engine reachable",
			services: []Service{devRow(3000, 1)},
			probe:    probeOK,
			want:     []string{"runtime:ok", "machine:ok"},
		},
		{
			name:     "engine unreachable",
			services: []Service{devRow(3000, 1)},
			probe:    probeDenied,
			want:     []string{"runtime:warn"},
		},
		{
			name:     "unresolved proxy row",
			services: []Service{devRow(3000, 1), {Port: 5432, Pid: 1, Process: "docker-proxy"}},
			probe:    probeOK,
			want:     []string{"runtime:ok", "container:fail"},
		},
		{
			name:     "SO_REUSEPORT collision",
			services: []Service{devRow(3000, 100), devRow(3000, 200)},
			probe:    probeOK,
			want:     []string{"runtime:ok", "multiplicity:warn"},
		},
		{
			name:     "unattributable listeners",
			services: []Service{{Port: 53, Pid: 0, Process: "?"}},
			probe:    probeOK,
			want:     []string{"runtime:ok", "owner:warn"},
		},
		{
			name:     "listeners traced to a service are attributable",
			services: []Service{{Port: 3306, Pid: 0, Process: "mysqld", Framework: "MySQL", Unit: "mysql", User: "mysql"}},
			probe:    probeOK,
			want:     []string{"runtime:ok", "machine:ok"},
		},
		{
			name:     "windows rows only count on WSL",
			services: []Service{{Port: 3000, Process: "node", Windows: true, WindowsPid: 1}},
			probe:    probeOK,
			wsl:      false,
			want:     []string{"runtime:ok", "machine:ok"},
		},
		{
			name:     "windows rows on WSL",
			services: []Service{{Port: 3000, Process: "node", Windows: true, WindowsPid: 1}},
			probe:    probeOK,
			wsl:      true,
			want:     []string{"runtime:ok", "windows:warn"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := diagnoseMachine(c.services, c.probe, c.wsl)
			if !equal(summarize(got), c.want) {
				t.Errorf("diagnoseMachine:\n got  %v\n want %v", summarize(got), c.want)
			}
		})
	}
}

func TestHasProblemsAndPortHeader(t *testing.T) {
	if HasProblems(nil) || HasProblems([]Finding{ok("a", "x")}) {
		t.Error("all-✓ findings should read as clean")
	}
	if !HasProblems([]Finding{ok("a", "x"), warn("b", "x", "")}) {
		t.Error("a warn should read as problems")
	}

	s := Service{Port: 3000, Pid: 48391, Process: "node", Framework: "Next.js", Project: "/home/dev/shop", ProjectName: "shop", Started: time.Now().Add(-time.Hour)}
	h := PortHeader(s)
	for _, want := range []string{":3000", "Next.js", "pid 48391", "shop", "up 1h"} {
		if !strings.Contains(h, want) {
			t.Errorf("PortHeader %q missing %q", h, want)
		}
	}
	c := PortHeader(Service{Port: 5432, Pid: 1, Process: "docker-proxy", Container: "shop-db", ContainerID: "a", Runtime: "docker", Image: "postgres"})
	if !strings.Contains(c, "container shop-db") {
		t.Errorf("container header %q missing the container", c)
	}
}