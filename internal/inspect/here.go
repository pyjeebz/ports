package inspect

import (
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"runtime"
	"time"

	gnet "github.com/shirou/gopsutil/v4/net"
)

// ProjectRoot is the project a directory belongs to, found the way the
// table finds a process's project — so `ports here` and the PROJECT
// column always agree. A directory inside no project is its own root.
func ProjectRoot(cfg *Config, dir string) string {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	dir = filepath.Clean(dir)
	if root := detectProject(dir, cfg.resolve().markers); root != "" {
		return root
	}
	return dir
}

// Here keeps the rows that belong to a project root: host processes whose
// project is the root or whose working directory sits under it, and
// compose containers started from it. Windows-side rows carry no path,
// so they never belong.
func Here(services []Service, root string) []Service {
	var out []Service
	for _, s := range services {
		if belongsTo(s, root) {
			out = append(out, s)
		}
	}
	return out
}

func belongsTo(s Service, root string) bool {
	if s.Container != "" {
		return s.ComposeDir != "" && within(s.ComposeDir, root)
	}
	return (s.Project != "" && within(s.Project, root)) ||
		(s.Cwd != "" && within(s.Cwd, root))
}

// within reports whether path is root or lies under it — by path
// segment, so ~/shop never claims ~/shop-admin.
func within(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !filepath.IsAbs(rel) && !startsWithParent(rel))
}

func startsWithParent(rel string) bool {
	return len(rel) >= 3 && rel[:2] == ".." && rel[2] == filepath.Separator
}

// Listening reports whether anything on this machine listens on port —
// the cheap question `ports wait` polls. On Linux it reads the kernel's
// socket tables directly, skipping the per-process walk a full scan pays
// for; elsewhere it asks gopsutil. Windows-side listeners across the WSL
// boundary are not consulted: the interop costs more than a poll can.
func Listening(port uint32) bool {
	if runtime.GOOS == "linux" {
		for k := range socketOwners() {
			if k.port == port {
				return true
			}
		}
		return false
	}
	conns, err := gnet.Connections("tcp")
	if err != nil {
		return false
	}
	for _, c := range conns {
		if c.Status == "LISTEN" && c.Laddr.Port == port {
			return true
		}
	}
	return false
}

// loopbacks are where a local listener answers. Node resolves
// "localhost" to ::1 on many systems, so a dev server bound there never
// answers on 127.0.0.1 — both are tried.
var loopbacks = []string{"127.0.0.1", "::1"}

// ServesHTTP reports whether the listener on port answers HTTP — one HEAD
// request per loopback, and any answer counts (a 404 is as much proof as
// a 200). Non-HTTP listeners answer garbage or nothing, and the parse
// fails.
func ServesHTTP(port uint32) bool {
	client := &http.Client{
		Timeout: 500 * time.Millisecond,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse // the first answer is the proof
		},
	}
	for _, host := range loopbacks {
		resp, err := client.Head(fmt.Sprintf("http://%s/", net.JoinHostPort(host, fmt.Sprint(port))))
		if err != nil {
			continue
		}
		_ = resp.Body.Close()
		return true
	}
	return false
}

// dialable reports whether a TCP connection to the port succeeds on
// either loopback.
func dialable(port uint32) bool {
	for _, host := range loopbacks {
		c, err := net.DialTimeout("tcp", net.JoinHostPort(host, fmt.Sprint(port)), 500*time.Millisecond)
		if err != nil {
			continue
		}
		_ = c.Close()
		return true
	}
	return false
}
