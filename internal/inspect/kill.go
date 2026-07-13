package inspect

import (
	"errors"
	"fmt"
	"slices"
	"time"

	gproc "github.com/shirou/gopsutil/v4/process"
)

const (
	stopWait = 2 * time.Second
	stopPoll = 100 * time.Millisecond
)

// Stop terminates the process behind a service: SIGTERM, up to stopWait
// for a clean exit, then SIGKILL. force skips straight to SIGKILL.
func Stop(s Service, force bool) error {
	// docker-backed: stop the container through the daemon — killing
	// docker-proxy would break the mapping and leave it running
	if s.ContainerID != "" {
		return StopContainer(s.ContainerID, force)
	}
	if s.Process == "docker-proxy" {
		return errors.New("docker-backed — could not resolve the container; use `docker stop`")
	}
	if !s.Known() {
		return errors.New("owner unknown — run with sudo")
	}
	p, err := gproc.NewProcess(s.Pid)
	if err != nil {
		return nil // already gone
	}
	// only kill the process observed at scan time, not a reused pid
	if !s.Started.IsZero() {
		if ct, err := p.CreateTime(); err == nil && ct > 0 && !time.UnixMilli(ct).Equal(s.Started) {
			return fmt.Errorf("pid %d now belongs to another process", s.Pid)
		}
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

// isGone treats zombies as gone: the process has exited and its sockets
// are closed, the parent just hasn't reaped it yet.
func isGone(p *gproc.Process) bool {
	if st, err := p.Status(); err == nil && slices.Contains(st, gproc.Zombie) {
		return true
	}
	running, err := p.IsRunning()
	return err == nil && !running
}
