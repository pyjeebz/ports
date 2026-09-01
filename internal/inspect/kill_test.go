package inspect

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	gproc "github.com/shirou/gopsutil/v4/process"
)

func startVictim(t *testing.T, args ...string) Service {
	t.Helper()
	cmd := exec.Command(args[0], args[1:]...)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go func() { _ = cmd.Wait() }() // reap on exit
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	s := Service{Pid: int32(cmd.Process.Pid), Process: args[0]}
	if p, err := gproc.NewProcess(s.Pid); err == nil {
		if ct, err := p.CreateTime(); err == nil && ct > 0 {
			s.Started = time.UnixMilli(ct)
		}
	}
	return s
}

func assertGone(t *testing.T, pid int32) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		p, err := gproc.NewProcess(pid)
		if err != nil || isGone(p) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("pid %d still running", pid)
}

func TestStopGraceful(t *testing.T) {
	s := startVictim(t, "sleep", "30")
	start := time.Now()
	if err := Stop(s, false); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) >= stopWait {
		t.Errorf("graceful stop escalated: took %v", time.Since(start))
	}
	assertGone(t, s.Pid)
}

func TestStopEscalatesWhenTermIgnored(t *testing.T) {
	// windows Terminate is TerminateProcess — always forceful, so there is
	// no ignorable SIGTERM and nothing to escalate from
	if runtime.GOOS == "windows" {
		t.Skip("escalation needs unix signal semantics")
	}
	// the trap is installed before the marker is written, so waiting for
	// the file proves TERM is ignorable — no guessed sleep racing sh's
	// startup
	marker := filepath.Join(t.TempDir(), "trapped")
	s := startVictim(t, "sh", "-c",
		`trap "" TERM; touch `+marker+`; while :; do sleep 0.2; done`)
	waitForFile(t, marker)
	start := time.Now()
	if err := Stop(s, false); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) < stopWait {
		t.Errorf("stop returned in %v; expected a full SIGTERM wait before SIGKILL", time.Since(start))
	}
	assertGone(t, s.Pid)
}

func TestStopForce(t *testing.T) {
	s := startVictim(t, "sleep", "30")
	start := time.Now()
	if err := Stop(s, true); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) >= stopWait {
		t.Errorf("force stop waited: took %v", time.Since(start))
	}
	assertGone(t, s.Pid)
}

func TestStopRefusesDockerProxy(t *testing.T) {
	if err := Stop(Service{Pid: 1, Process: "docker-proxy"}, false); err == nil {
		t.Fatal("Stop killed docker-proxy; it must refuse")
	}
}

func TestStopRefusesWindows(t *testing.T) {
	if err := Stop(Service{Pid: 0, Windows: true, WindowsPid: 42}, false); err == nil {
		t.Fatal("Stop accepted a windows-side row; it must refuse")
	}
}

func TestStopRefusesUnknownOwner(t *testing.T) {
	if err := Stop(Service{Pid: 0, Process: "?"}, false); err == nil {
		t.Fatal("Stop accepted pid 0; it must refuse")
	}
}

// A scan an hour old cannot describe a process started seconds ago: the
// pid was reused, and Stop must refuse rather than kill the newcomer.
func TestStopRefusesReusedPid(t *testing.T) {
	s := startVictim(t, "sleep", "30")
	s.Started = s.Started.Add(-time.Hour)
	err := Stop(s, false)
	if err == nil || !strings.Contains(err.Error(), "another process") {
		t.Fatalf("Stop accepted a reused pid: %v", err)
	}
	if p, err := gproc.NewProcess(s.Pid); err != nil || isGone(p) {
		t.Fatalf("pid %d died — the refusal was not advisory", s.Pid)
	}
}

// waitForFile blocks until path exists — the readiness handshake for
// victims that signal setup completion by touching a file.
func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
}

// startZombie leaves an unreaped exited child behind: sh backgrounds a
// short sleep, echoes its pid, then execs a long one — the exec'd process
// never calls wait, so the dead child stays a zombie until the test ends.
// It returns the zombie's pid, for doctor's state checks to read.
func startZombie(t *testing.T) int32 {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("zombies need unix semantics")
	}
	cmd := exec.Command("sh", "-c", "sleep 1 & echo $!; exec sleep 60")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil {
		t.Fatalf("child echoed %q, not its pid", line)
	}
	// the child exits on its own schedule — poll for the kernel's verdict
	// (state Z) instead of sleeping past a guessed exit time
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		p, err := gproc.NewProcess(int32(pid))
		if err != nil {
			t.Fatalf("zombie %d vanished before it could zombie", pid)
		}
		if st, err := p.Status(); err == nil && slices.Contains(st, gproc.Zombie) {
			return int32(pid)
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("pid %d never became a zombie", pid)
	return 0
}

// The zombie is the answer to "free said it's stopped but the port's row
// won't go away": doctor must read it as "already exited, effectively
// free", never as a kill target.
func TestDoctorReadsZombie(t *testing.T) {
	zpid := startZombie(t)
	p, err := gproc.NewProcess(zpid)
	if err != nil {
		t.Skipf("pid %d was reaped before doctor looked", zpid)
	}
	s := Service{Pid: zpid, Process: "sleep", Port: 3999}
	if ct, err := p.CreateTime(); err == nil && ct > 0 {
		s.Started = time.UnixMilli(ct)
	}

	f := factsFor(s)
	if !f.exists || !f.fresh || len(f.states) == 0 {
		t.Errorf("factsFor on a zombie: exists=%v fresh=%v states=%v", f.exists, f.fresh, f.states)
	}
	got := stateFindings(s, f)
	if len(got) != 1 || got[0].Status != CheckWarn || !strings.Contains(got[0].Message, "zombie") {
		t.Fatalf("zombie findings: %+v", got)
	}
	if !strings.Contains(got[0].Message, "effectively free") {
		t.Errorf("zombie finding doesn't say the port is free: %q", got[0].Message)
	}
}
