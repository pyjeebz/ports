package inspect

import (
	"errors"
	"fmt"
	"slices"
	"time"

	gproc "github.com/shirou/gopsutil/v4/process"
)

// A process's start time is its boot-relative tick count plus the boot
// time, and the kernel recomputes the boot time on every read: it jitters
// by a second on WSL and moves whenever the clock is stepped. Uncached,
// the same process can read two start times a moment apart — and the
// pid-reuse guard, IsRunning, and doctor's freshness check all compare
// start times. One boot time per run keeps those comparisons honest.
func init() { gproc.EnableBootTimeCache(true) }

const (
	stopWait = 2 * time.Second
	stopPoll = 100 * time.Millisecond
)

// Stop terminates the process behind a service: SIGTERM, up to stopWait
// for a clean exit, then SIGKILL. force skips straight to SIGKILL.
func Stop(s Service, force bool) error {
	// the Windows pid means nothing to our signals; observing across
	// the boundary is fine, killing across it is not
	if s.Windows {
		return errors.New("windows-side — stop it from Windows (Task Manager or taskkill)")
	}
	// docker-backed: stop the container through the daemon — killing
	// docker-proxy would break the mapping and leave it running
	if s.ContainerID != "" {
		return StopContainer(s.ContainerID, force)
	}
	if s.Process == "docker-proxy" {
		return fmt.Errorf("%s-backed — could not resolve the container; use `%s stop`", containerRuntime, containerRuntime)
	}
	if !s.Known() {
		if s.Unit != "" {
			return fmt.Errorf("a systemd service running as %s — stop it with `sudo systemctl stop %s`", s.User, s.Unit)
		}
		if s.User != "" {
			return fmt.Errorf("owned by user %s — run with sudo", s.User)
		}
		return errors.New("owner unknown — run with sudo")
	}
	p, err := gproc.NewProcess(s.Pid)
	if err != nil {
		return nil // already gone
	}
	// only kill the process observed at scan time, not a reused pid
	if !sameProcess(p, s.Started) {
		return fmt.Errorf("pid %d now belongs to another process", s.Pid)
	}
	if !force {
		if err := p.Terminate(); err == nil && waitGone(p, stopWait) {
			return nil
		}
		// SIGTERM undeliverable or ignored: escalate
	}
	if err := p.Kill(); err != nil {
		if isGone(p) {
			return nil // exited between the wait and the kill
		}
		return err
	}
	if !waitGone(p, time.Second) {
		return fmt.Errorf("pid %d survived SIGKILL", s.Pid)
	}
	return nil
}

func waitGone(p *gproc.Process, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if isGone(p) {
			return true
		}
		time.Sleep(stopPoll)
	}
	return false
}

// sameProcess reports whether p is still the process a scan observed at
// `started`, guarding the kill path (and doctor's findings) against pid
// reuse. When the start time can't be established either way, it errs on
// the side of "same" — no evidence of reuse is not evidence.
func sameProcess(p *gproc.Process, started time.Time) bool {
	if started.IsZero() {
		return true
	}
	ct, err := p.CreateTime()
	if err != nil || ct == 0 {
		return true
	}
	return time.UnixMilli(ct).Equal(started)
}

// isGone treats zombies as gone: the process has exited and its sockets
// are closed, the parent just hasn't reaped it yet.
func isGone(p *gproc.Process) bool {
	if st, err := p.Status(); err == nil && slices.Contains(st, gproc.Zombie) {
		return true
	}
	running, err := p.IsRunning()
	return err == nil && !running
}
