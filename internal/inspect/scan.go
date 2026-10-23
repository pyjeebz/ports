package inspect

import (
	"path/filepath"
	"slices"
	"sort"
	"time"

	gnet "github.com/shirou/gopsutil/v4/net"
	gproc "github.com/shirou/gopsutil/v4/process"
)

// Scan returns every listening TCP socket, deduped by (pid, port) so a
// process bound on both v4 and v6 shows once, sorted by port. On WSL2
// it also includes listeners on the Windows side of the boundary.
func Scan(cfg *Config) ([]Service, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	env := cfg.resolve()

	// ask the Windows side in parallel: the interop calls, when they
	// apply, cost far more than the whole /proc scan
	winCh := make(chan []Service, 1)
	go func() {
		if onWSL() {
			winCh <- windowsListeners()
			return
		}
		winCh <- nil
	}()

	conns, err := gnet.Connections("tcp")
	if err != nil {
		return nil, err
	}
	// another user's sockets come back without a pid; their owner's uid
	// tells them apart, read only when there are such sockets to tell
	var owners map[sockKey]uint32
	type key struct {
		pid  int32
		port uint32
		uid  uint32
	}
	seen := make(map[key]bool)
	services := []Service{}
	for _, c := range conns {
		if c.Status != "LISTEN" {
			continue
		}
		s := describe(c, env)
		if !s.Known() {
			if owners == nil {
				owners = socketOwners()
			}
			s.ownerUID, s.hasOwner = ownerOf(owners, c.Laddr.IP, c.Laddr.Port)
		}
		k := key{c.Pid, c.Laddr.Port, s.ownerUID}
		if seen[k] {
			continue
		}
		seen[k] = true
		services = append(services, s)
	}
	enrichDocker(services, listContainers())
	if slices.ContainsFunc(services, Service.Anonymous) {
		attribute(services, listProcs(), env.rules, lookupUser)
		services = collapseUnowned(services)
	}

	// merge windows rows; a port also held inside WSL keeps its WSL row
	// (mirrored mode shows one socket from both sides, and forwarders
	// like wslrelay shadow WSL ports by design)
	held := map[uint32]bool{}
	for _, s := range services {
		held[s.Port] = true
	}
	for _, w := range <-winCh {
		if !held[w.Port] {
			services = append(services, w)
		}
	}

	sort.Slice(services, func(i, j int) bool {
		if services[i].Port != services[j].Port {
			return services[i].Port < services[j].Port
		}
		return services[i].Pid < services[j].Pid
	})
	return services, nil
}

func describe(c gnet.ConnectionStat, env scanEnv) Service {
	s := Service{Port: c.Laddr.Port, Pid: c.Pid, Addr: c.Laddr.IP, Process: "?"}
	if !s.Known() {
		return s
	}
	p, err := gproc.NewProcess(c.Pid)
	if err != nil {
		return s
	}
	// Exe basename over Name(): /proc/<pid>/comm is the thread name,
	// which node reports as "MainThread".
	if exe, err := p.Exe(); err == nil && exe != "" {
		s.Process = filepath.Base(exe)
	} else if name, err := p.Name(); err == nil && name != "" {
		s.Process = name
	}
	if cmd, err := p.Cmdline(); err == nil {
		s.Cmdline = cmd
	}
	if pp, err := p.Ppid(); err == nil {
		s.ParentPid = pp
	}
	if cwd, err := p.Cwd(); err == nil {
		s.Cwd = cwd
		s.Project = detectProject(cwd, env.markers)
		if s.Project != "" {
			s.ProjectName = filepath.Base(s.Project)
		}
	}
	if ct, err := p.CreateTime(); err == nil && ct > 0 {
		s.Started = time.UnixMilli(ct)
	}
	s.Framework = detectFramework(env.rules, s.Process, s.Cmdline)
	return s
}
