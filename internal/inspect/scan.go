package inspect

import (
	"path/filepath"
	"sort"
	"time"

	gnet "github.com/shirou/gopsutil/v4/net"
	gproc "github.com/shirou/gopsutil/v4/process"
)

// Scan returns every listening TCP socket, deduped by (pid, port) so a
// process bound on both v4 and v6 shows once, sorted by port. On WSL2
// it also includes listeners on the Windows side of the boundary.
func Scan() ([]Service, error) {
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
	type key struct {
		pid  int32
		port uint32
	}
	seen := make(map[key]bool)
	services := []Service{}
	for _, c := range conns {
		if c.Status != "LISTEN" {
			continue
		}
		k := key{c.Pid, c.Laddr.Port}
		if seen[k] {
			continue
		}
		seen[k] = true
		services = append(services, describe(c))
	}
	enrichDocker(services, listContainers())

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

func describe(c gnet.ConnectionStat) Service {
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
	if cwd, err := p.Cwd(); err == nil {
		s.Cwd = cwd
		s.Project = DetectProject(cwd)
		if s.Project != "" {
			s.ProjectName = filepath.Base(s.Project)
		}
	}
	if ct, err := p.CreateTime(); err == nil && ct > 0 {
		s.Started = time.UnixMilli(ct)
	}
	s.Framework = DetectFramework(s.Process, s.Cmdline)
	return s
}
