package inspect

import (
	"os/exec"
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
	go cmd.Wait() // reap on exit
	t.Cleanup(func() { cmd.Process.Kill() })
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
	s := startVictim(t, "sh", "-c", `trap "" TERM; while :; do sleep 0.2; done`)
	time.Sleep(100 * time.Millisecond) // let the trap install
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

func TestStopRefusesUnknownOwner(t *testing.T) {
	if err := Stop(Service{Pid: 0, Process: "?"}, false); err == nil {
		t.Fatal("Stop accepted pid 0; it must refuse")
	}
}
