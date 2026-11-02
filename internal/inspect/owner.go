package inspect

import (
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	gproc "github.com/shirou/gopsutil/v4/process"
)

// Without root, the kernel hides which pid holds another user's socket —
// but not who owns it. /proc/net/tcp carries every socket's uid, and the
// processes running as that uid, with the systemd unit each sits in, are
// world-readable. That is enough to name most of the rows a plain scan
// leaves as ?, without ever guessing from the port number alone.

type sockKey struct {
	ip   string
	port uint32
}

// readProcNet is a var so tests can supply the kernel's table text.
var readProcNet = func(name string) ([]byte, error) {
	return os.ReadFile("/proc/net/" + name)
}

// socketOwners maps each listening TCP socket to the uid that owns it.
// Off Linux there is no table to read, and the map comes back empty.
func socketOwners() map[sockKey]uint32 {
	owners := map[sockKey]uint32{}
	if runtime.GOOS != "linux" {
		return owners
	}
	for _, name := range []string{"tcp", "tcp6"} {
		if b, err := readProcNet(name); err == nil {
			parseProcNetTCP(string(b), owners)
		}
	}
	return owners
}

// parseProcNetTCP reads the listening rows (state 0A) of /proc/net/tcp
// or tcp6 into owners. The header row fails the state check on its own.
func parseProcNetTCP(out string, owners map[sockKey]uint32) {
	for line := range strings.Lines(out) {
		f := strings.Fields(line)
		if len(f) < 8 || f[3] != "0A" {
			continue
		}
		host, hexPort, ok := strings.Cut(f[1], ":")
		if !ok {
			continue
		}
		port, err := strconv.ParseUint(hexPort, 16, 16)
		if err != nil {
			continue
		}
		uid, err := strconv.ParseUint(f[7], 10, 32)
		if err != nil {
			continue
		}
		if ip := decodeProcIP(host); ip != "" {
			owners[sockKey{ip, uint32(port)}] = uint32(uid)
		}
	}
}

// decodeProcIP turns the kernel's hex address — host-order 32-bit words,
// one for v4, four for v6 — into the string form the scan reports.
func decodeProcIP(h string) string {
	b, err := hex.DecodeString(h)
	if err != nil || (len(b) != net.IPv4len && len(b) != net.IPv6len) {
		return ""
	}
	for i := 0; i < len(b); i += 4 {
		b[i], b[i+1], b[i+2], b[i+3] = b[i+3], b[i+2], b[i+1], b[i]
	}
	return net.IP(b).String()
}

// ownerOf finds the uid behind a socket: by exact address first, then by
// port alone when every socket on it shares one owner — address spellings
// (v4-mapped v6, zone suffixes) are the only thing that can make the
// exact lookup miss.
func ownerOf(owners map[sockKey]uint32, ip string, port uint32) (uint32, bool) {
	if uid, ok := owners[sockKey{ip, port}]; ok {
		return uid, true
	}
	var found uint32
	n := 0
	for k, uid := range owners {
		if k.port != port {
			continue
		}
		if n > 0 && uid != found {
			return 0, false
		}
		found, n = uid, n+1
	}
	return found, n > 0
}

// procInfo is what anyone may read about another user's process.
type procInfo struct {
	pid     int32
	uid     uint32 // effective
	name    string // argv[0] basename — comm truncates at 15 bytes
	cmdline string
	unit    string // systemd service, without .service; "" outside one
	started time.Time
}

// listProcs is a var so tests can supply a synthetic process table.
var listProcs = func() []procInfo {
	ps, err := gproc.Processes()
	if err != nil {
		return nil
	}
	var out []procInfo
	for _, p := range ps {
		uids, err := p.Uids()
		if err != nil || len(uids) < 2 {
			continue
		}
		pi := procInfo{pid: p.Pid, uid: uids[1]}
		if cmd, err := p.Cmdline(); err == nil {
			pi.cmdline = cmd
		}
		pi.name = argv0(pi.cmdline)
		if pi.name == "" {
			if n, err := p.Name(); err == nil {
				pi.name = n
			}
		}
		if ct, err := p.CreateTime(); err == nil && ct > 0 {
			pi.started = time.UnixMilli(ct)
		}
		if b, err := os.ReadFile(fmt.Sprintf("/proc/%d/cgroup", p.Pid)); err == nil {
			pi.unit = serviceUnit(string(b))
		}
		out = append(out, pi)
	}
	return out
}

// argv0 is the program a cmdline ran, as a basename. Daemons that retitle
// their workers ("postgres: checkpointer") keep the name before the colon.
func argv0(cmdline string) string {
	f := strings.Fields(cmdline)
	if len(f) == 0 {
		return ""
	}
	return strings.TrimSuffix(filepath.Base(f[0]), ":")
}

// serviceUnit names the systemd service a process runs under, from its
// /proc/<pid>/cgroup — the innermost *.service on the path, so a user
// service inside user@1000.service reads as itself. user@1000 itself is
// never the answer: it's the whole login session, so a process that sits
// straight under it belongs to no service, and naming it would point a
// stop at `systemctl stop user@1000`. Lines are id:controllers:path in
// both cgroup v2 (one line, empty controllers) and v1 (one per
// hierarchy, each carrying the same service path).
func serviceUnit(cgroup string) string {
	for line := range strings.Lines(cgroup) {
		f := strings.SplitN(strings.TrimSpace(line), ":", 3)
		if len(f) != 3 {
			continue
		}
		parts := strings.Split(f[2], "/")
		for i := len(parts) - 1; i >= 0; i-- {
			if unit, ok := strings.CutSuffix(parts[i], ".service"); ok && unit != "" {
				if strings.HasPrefix(unit, "user@") {
					return ""
				}
				return unit
			}
		}
	}
	return ""
}

// portServers breaks a tie only when an owner runs several services —
// root nearly always does. It never names a row on its own: a server
// listed here must also be running, as the socket's owner, for the row
// to read as it.
var portServers = map[uint32][]string{
	22:    {"sshd"},
	53:    {"systemd-resolved", "dnsmasq", "named", "unbound", "coredns"},
	80:    {"nginx", "apache2", "httpd", "caddy", "lighttpd", "traefik", "haproxy"},
	443:   {"nginx", "apache2", "httpd", "caddy", "lighttpd", "traefik", "haproxy"},
	631:   {"cupsd"},
	3306:  {"mysqld", "mariadbd"},
	5432:  {"postgres"},
	6379:  {"redis-server", "valkey-server"},
	8080:  {"nginx", "apache2", "httpd", "caddy", "traefik", "haproxy"},
	11211: {"memcached"},
	11434: {"ollama"},
	27017: {"mongod"},
	33060: {"mysqld"},
}

// attribute names the rows the scan could not tie to a pid. Each row
// learns its owner's user name; it is named only when that owner's
// processes leave one answer — a single service, or a single server
// known for the port. Anything less stays ? rather than read as fact.
// The pid stays hidden: the row says what holds the port, not which of
// the service's processes does.
func attribute(services []Service, procs []procInfo, rules []frameworkRule, userName func(uint32) string) {
	byUID := map[uint32][]procInfo{}
	for _, p := range procs {
		byUID[p.uid] = append(byUID[p.uid], p)
	}
	for i := range services {
		s := &services[i]
		if s.Known() || s.Container != "" || s.Windows || !s.hasOwner {
			continue
		}
		s.User = userName(s.ownerUID)
		groups := groupByService(byUID[s.ownerUID])
		if len(groups) > 1 {
			groups = slices.DeleteFunc(groups, func(g []procInfo) bool {
				return !slices.ContainsFunc(g, func(p procInfo) bool {
					return slices.Contains(portServers[s.Port], p.name)
				})
			})
		}
		if len(groups) != 1 {
			continue
		}
		main := oldest(groups[0])
		s.Process = main.name
		s.Cmdline = main.cmdline
		s.Unit = main.unit
		s.Started = main.started
		s.Framework = detectFramework(rules, main.name, main.cmdline)
	}
}

// groupByService gathers processes by systemd unit, or by program name
// for those outside one, in first-seen order.
func groupByService(procs []procInfo) [][]procInfo {
	var groups [][]procInfo
	at := map[string]int{}
	for _, p := range procs {
		key := "exe:" + p.name
		if p.unit != "" {
			key = "unit:" + p.unit
		}
		if i, ok := at[key]; ok {
			groups[i] = append(groups[i], p)
			continue
		}
		at[key] = len(groups)
		groups = append(groups, []procInfo{p})
	}
	return groups
}

// oldest is a service's main process — the one that started the rest.
func oldest(procs []procInfo) procInfo {
	return slices.MinFunc(procs, func(a, b procInfo) int {
		if c := a.started.Compare(b.started); c != 0 {
			return c
		}
		return int(a.pid - b.pid)
	})
}

// lookupUser is the uid's login name, or the number when it has none.
func lookupUser(uid uint32) string {
	id := strconv.FormatUint(uint64(uid), 10)
	if u, err := user.LookupId(id); err == nil {
		return u.Username
	}
	return id
}

// collapseUnowned keeps one row per port and owner identity among the
// rows no pid backs. A port bound on several addresses by one service
// shows once; a nameless binder beside a named one adds a ? that says
// nothing the named row doesn't.
func collapseUnowned(services []Service) []Service {
	named := map[uint32]bool{}
	for _, s := range services {
		if s.unowned() && s.Process != "?" {
			named[s.Port] = true
		}
	}
	seen := map[string]bool{}
	return slices.DeleteFunc(services, func(s Service) bool {
		if !s.unowned() {
			return false
		}
		if s.Process == "?" && named[s.Port] {
			return true
		}
		key := fmt.Sprintf("%d/%s/%s", s.Port, s.Unit, s.Process)
		if seen[key] {
			return true
		}
		seen[key] = true
		return false
	})
}
