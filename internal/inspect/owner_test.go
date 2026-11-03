package inspect

import (
	"strings"
	"testing"
	"time"
)

// Real rows from a WSL2 Ubuntu box: systemd-resolved (uid 991) on two
// loopbacks, mysqld (105), ollama (999), and root's apache on [::]:80.
// The ESTABLISHED row (01) must not read as a listener.
const procNetTCP = `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 3600007F:0035 00000000:0000 0A 00000000:00000000 00:00000000 00000000   991        0 2703 1 0000000000000000 100 0 0 10 5
   1: 0100007F:0CEA 00000000:0000 0A 00000000:00000000 00:00000000 00000000   105        0 3409 1 0000000000000000 100 0 0 10 0
   2: 3500007F:0035 00000000:0000 0A 00000000:00000000 00:00000000 00000000   991        0 2701 1 0000000000000000 100 0 0 10 5
   3: 0100007F:2CAA 00000000:0000 0A 00000000:00000000 00:00000000 00000000   999        0 6481 1 0000000000000000 100 0 0 10 0
   4: 0100007F:9C40 0100007F:0CEA 01 00000000:00000000 00:00000000 00000000  1000        0 9999 1 0000000000000000 100 0 0 10 0
`

const procNetTCP6 = `  sl  local_address                         remote_address                        st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 00000000000000000000000000000000:0050 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 7477 1 0000000000000000 100 0 0 10 0
   1: 0000000000000000FFFF00000100007F:1F90 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 7478 1 0000000000000000 100 0 0 10 0
`

func TestParseProcNetTCP(t *testing.T) {
	owners := map[sockKey]uint32{}
	parseProcNetTCP(procNetTCP, owners)
	parseProcNetTCP(procNetTCP6, owners)

	want := map[sockKey]uint32{
		{"127.0.0.54", 53}:   991,
		{"127.0.0.53", 53}:   991,
		{"127.0.0.1", 3306}:  105,
		{"127.0.0.1", 11434}: 999,
		{"::", 80}:           0,
		// v4-mapped v6 spells as the v4 address, the way the scan does
		{"127.0.0.1", 8080}: 1000,
	}
	if len(owners) != len(want) {
		t.Errorf("got %d listeners, want %d (ESTABLISHED and header skipped): %v", len(owners), len(want), owners)
	}
	for k, uid := range want {
		if got, ok := owners[k]; !ok || got != uid {
			t.Errorf("%v: uid %d (found %v), want %d", k, got, ok, uid)
		}
	}
}

// An exact miss falls back to the port, but only when the port has one
// owner — two owners on a port leave the row unowned rather than guessed.
func TestOwnerOf(t *testing.T) {
	owners := map[sockKey]uint32{
		{"127.0.0.1", 3306}:    105,
		{"127.0.0.53", 53}:     991,
		{"10.255.255.254", 53}: 0,
	}
	if uid, ok := ownerOf(owners, "127.0.0.1", 3306); !ok || uid != 105 {
		t.Errorf("exact: %d %v", uid, ok)
	}
	if uid, ok := ownerOf(owners, "::ffff:127.0.0.1", 3306); !ok || uid != 105 {
		t.Errorf("port fallback: %d %v", uid, ok)
	}
	if _, ok := ownerOf(owners, "0.0.0.0", 53); ok {
		t.Error("port 53 has two owners; the fallback must not pick one")
	}
	if _, ok := ownerOf(owners, "127.0.0.1", 9999); ok {
		t.Error("no socket on the port, no owner")
	}
}

func TestServiceUnit(t *testing.T) {
	cases := map[string]string{
		"0::/system.slice/mysql.service\n":                                       "mysql",
		"0::/system.slice/systemd-resolved.service\n":                            "systemd-resolved",
		"0::/user.slice/user-1000.slice/user@1000.service/app.slice/foo.service": "foo",
		"0::/user.slice/user-1000.slice/session-3.scope\n":                       "",
		// a user's own processes outside any real service sit straight under
		// the session manager: user@1000 is the whole login, never the service
		// holding a port, and `systemctl stop user@1000` would log them out
		"0::/user.slice/user-1000.slice/user@1000.service/init.scope\n":          "",
		"0::/user.slice/user-1000.slice/user@1000.service/app.slice/app-x.scope": "",
		"12:pids:/user.slice/user-1000.slice/user@1000.service/app.slice\n":      "",
		// v1 lists the undelegated pids hierarchy first, stopping at the
		// manager; the user service is named on a later line
		"12:pids:/user.slice/user-1000.slice/user@1000.service\n1:name=systemd:/user.slice/user-1000.slice/user@1000.service/app.slice/foo.service\n": "foo",
		"0::/init.scope\n": "",
		// cgroup v1: the hierarchy lines carry controllers before the path
		"12:pids:/system.slice/nginx.service\n1:name=systemd:/system.slice/nginx.service\n": "nginx",
	}
	for in, want := range cases {
		if got := serviceUnit(in); got != want {
			t.Errorf("serviceUnit(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestArgv0(t *testing.T) {
	for in, want := range map[string]string{
		"/usr/sbin/apache2 -k start":      "apache2",
		"postgres: 16/main: checkpointer": "postgres",
		"":                                "",
	} {
		if got := argv0(in); got != want {
			t.Errorf("argv0(%q) = %q, want %q", in, got, want)
		}
	}
}

func unownedRow(port, uid uint32) Service {
	return Service{Port: port, Process: "?", ownerUID: uid, hasOwner: true}
}

func names(uid uint32) string {
	return map[uint32]string{0: "root", 105: "mysql", 991: "systemd-resolve", 999: "ollama"}[uid]
}

// The process table of the box the fixtures came from, trimmed.
func hostProcs() []procInfo {
	boot := time.Now().Add(-time.Hour)
	return []procInfo{
		{pid: 1, uid: 0, name: "systemd", started: boot},
		{pid: 280, uid: 991, name: "systemd-resolved", unit: "systemd-resolved", started: boot.Add(2 * time.Second)},
		{pid: 341, uid: 0, name: "apache2", cmdline: "/usr/sbin/apache2 -k start", unit: "apache2", started: boot.Add(5 * time.Second)},
		{pid: 342, uid: 33, name: "apache2", cmdline: "/usr/sbin/apache2 -k start", unit: "apache2", started: boot.Add(6 * time.Second)},
		{pid: 300, uid: 0, name: "cron", unit: "cron", started: boot.Add(3 * time.Second)},
		{pid: 310, uid: 0, name: "sshd", unit: "ssh", started: boot.Add(3 * time.Second)},
		{pid: 462, uid: 105, name: "mysqld", cmdline: "/usr/sbin/mysqld", unit: "mysql", started: boot.Add(9 * time.Second)},
		{pid: 318, uid: 999, name: "ollama", cmdline: "/usr/local/bin/ollama serve", unit: "ollama", started: boot.Add(4 * time.Second)},
	}
}

func TestAttribute(t *testing.T) {
	rows := []Service{
		unownedRow(3306, 105),  // mysql runs one service: sound by construction
		unownedRow(80, 0),      // root runs many; apache is the only web server
		unownedRow(9000, 0),    // root runs many and nothing is known for 9000
		unownedRow(22, 0),      // sshd breaks root's tie on 22, never apache
		unownedRow(4000, 4242), // an owner with no visible processes
		{Port: 3000, Pid: 77, Process: "node"},
		{Port: 5432, Process: "docker-proxy", Container: "db", ownerUID: 0, hasOwner: true},
	}
	procs := hostProcs()
	attribute(rows, procs, frameworkRules, names)

	mysql := rows[0]
	if mysql.Process != "mysqld" || mysql.Framework != "MySQL" || mysql.Unit != "mysql" || mysql.User != "mysql" {
		t.Errorf("mysql row: %+v", mysql)
	}
	if mysql.Pid != 0 || mysql.Started.IsZero() {
		t.Errorf("mysql row should keep pid 0 and gain a start time: %+v", mysql)
	}
	if web := rows[1]; web.Framework != "Apache" || web.Unit != "apache2" || web.User != "root" {
		t.Errorf("port 80 row: %+v", web)
	}
	if web := rows[1]; web.Cmdline != "/usr/sbin/apache2 -k start" || !web.Started.Equal(procs[2].started) {
		t.Errorf("port 80 should read from apache's root process, the oldest: %+v", web)
	}
	if odd := rows[2]; odd.Process != "?" || odd.User != "root" || odd.Unit != "" {
		t.Errorf("an ambiguous root port must stay ? with its owner: %+v", odd)
	}
	if ssh := rows[3]; ssh.Process != "sshd" || ssh.Unit != "ssh" {
		t.Errorf("port 22 row: %+v", ssh)
	}
	if ghost := rows[4]; ghost.Process != "?" || ghost.User != "" {
		t.Errorf("an owner with no processes stays ?: %+v", ghost)
	}
	if node := rows[5]; node.User != "" || node.Unit != "" {
		t.Errorf("a pid-backed row is not touched: %+v", node)
	}
	if proxy := rows[6]; proxy.User != "" || proxy.Unit != "" {
		t.Errorf("a container row is not touched: %+v", proxy)
	}
}

// One service across several sockets of a port reads as one row, and a
// nameless binder beside a named one drops out; nameless ones alone
// still show, once.
func TestCollapseUnowned(t *testing.T) {
	named := func(port uint32) Service {
		return Service{Port: port, Process: "systemd-resolved", Unit: "systemd-resolved"}
	}
	rows := []Service{
		named(53), named(53), {Port: 53, Process: "?", User: "root"},
		{Port: 9000, Process: "?", User: "root"}, {Port: 9000, Process: "?", User: "nobody"},
		{Port: 3000, Pid: 7, Process: "node"}, {Port: 3000, Pid: 8, Process: "node"},
	}
	got := collapseUnowned(rows)
	var summary []string
	for _, s := range got {
		summary = append(summary, s.Process)
	}
	if want := "systemd-resolved ? node node"; strings.Join(summary, " ") != want {
		t.Errorf("collapsed to %q, want %q", strings.Join(summary, " "), want)
	}
}

func TestOwnedRowRendering(t *testing.T) {
	svc := Service{Port: 3306, Process: "mysqld", Framework: "MySQL", Unit: "mysql", User: "mysql"}
	if got := project(svc); got != "systemd: mysql" {
		t.Errorf("project = %q", got)
	}
	if got := project(Service{Port: 9000, Process: "?", User: "root"}); got != "user: root" {
		t.Errorf("anonymous project = %q", got)
	}
	if svc.Anonymous() || !(Service{Port: 9000, Process: "?"}).Anonymous() {
		t.Error("Anonymous should mean nothing could name the row")
	}
	if !svc.MatchesName("mysql") || !svc.MatchesQuery("mys") {
		t.Error("the unit is a handle to stop and find a service by")
	}
	if h := PortHeader(svc); !strings.Contains(h, "MySQL") || !strings.Contains(h, "systemd: mysql") {
		t.Errorf("PortHeader %q", h)
	}

	err := Stop(svc, false)
	if err == nil || !strings.Contains(err.Error(), "sudo systemctl stop mysql") {
		t.Errorf("Stop on a service row should point at systemctl: %v", err)
	}
	f := ownerFinding(svc)
	if f.Fix != "stop it with `sudo systemctl stop mysql`" {
		t.Errorf("owner finding fix = %q", f.Fix)
	}
}
