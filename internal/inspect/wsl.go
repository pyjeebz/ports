package inspect

import (
	"context"
	"encoding/csv"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const winSys32 = "/mnt/c/Windows/System32/"

// runWin is a var so tests can fake Windows command output. Interop can
// be slow or wedged, so every call gets a hard deadline.
var runWin = func(exe string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, winSys32+exe, args...).Output()
}

// onWSL reports whether this is a WSL distro with a Windows side to ask
// about.
func onWSL() bool {
	if os.Getenv("WSL_DISTRO_NAME") != "" || os.Getenv("WSL_INTEROP") != "" {
		return true
	}
	b, err := os.ReadFile("/proc/sys/kernel/osrelease")
	return err == nil && strings.Contains(strings.ToLower(string(b)), "microsoft")
}

// Windows plumbing that would drown the table in system noise; the
// developer-run things (node.exe, postgres.exe, …) are what matter.
var winSystemProcs = map[string]bool{
	"System":       true,
	"svchost.exe":  true,
	"services.exe": true,
	"lsass.exe":    true,
	"wininit.exe":  true,
	"spoolsv.exe":  true,
}

type winSock struct {
	port uint32
	addr string
	pid  int32
}

// windowsListeners returns TCP listeners on the Windows side of the WSL2
// boundary, via netstat/tasklist interop. Any failure returns nil: no
// interop just means no extra context.
func windowsListeners() []Service {
	out, err := runWin("netstat.exe", "-ano", "-p", "TCP")
	if err != nil {
		return nil
	}
	socks := parseNetstat(string(out))
	if len(socks) == 0 {
		return nil
	}
	// without names the rows would be bare numbers — worse than absent
	tl, err := runWin("tasklist.exe", "/fo", "csv", "/nh")
	if err != nil {
		return nil
	}
	names := parseTasklist(string(tl))

	var services []Service
	for _, w := range socks {
		name := names[w.pid]
		if name == "" || winSystemProcs[name] {
			continue
		}
		short := strings.TrimSuffix(name, ".exe")
		services = append(services, Service{
			Port:       w.port,
			Addr:       w.addr,
			Process:    short,
			Framework:  DetectFramework(strings.ToLower(short), strings.ToLower(name)),
			Windows:    true,
			WindowsPid: w.pid,
		})
	}
	return services
}

// parseNetstat picks the LISTENING rows out of `netstat -ano -p TCP`
// output, one per port (v4/v6 duplicates collapse like the linux scan).
func parseNetstat(out string) []winSock {
	seen := map[uint32]bool{}
	var socks []winSock
	for line := range strings.Lines(out) {
		f := strings.Fields(line)
		if len(f) < 5 || f[0] != "TCP" || f[3] != "LISTENING" {
			continue
		}
		i := strings.LastIndex(f[1], ":")
		if i < 0 {
			continue
		}
		port, err := strconv.ParseUint(f[1][i+1:], 10, 16)
		if err != nil || port == 0 || seen[uint32(port)] {
			continue
		}
		pid, err := strconv.ParseInt(f[4], 10, 32)
		if err != nil {
			continue
		}
		seen[uint32(port)] = true
		addr := strings.Trim(f[1][:i], "[]")
		socks = append(socks, winSock{port: uint32(port), addr: addr, pid: int32(pid)})
	}
	return socks
}

// parseTasklist maps Windows pids to image names from
// `tasklist /fo csv /nh` output.
func parseTasklist(out string) map[int32]string {
	names := map[int32]string{}
	r := csv.NewReader(strings.NewReader(out))
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return names
	}
	for _, rec := range records {
		if len(rec) < 2 {
			continue
		}
		if pid, err := strconv.ParseInt(rec[1], 10, 32); err == nil {
			names[int32(pid)] = rec[0]
		}
	}
	return names
}
