//go:build !windows

package service

import (
	"os/exec"
	"syscall"
)

// detach puts the router in a session of its own, so closing the
// terminal that started it doesn't end it.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
