//go:build windows

package service

import (
	"os/exec"
	"syscall"
)

// detach starts the router without a console window of its own, apart
// from the console that started it.
func detach(cmd *exec.Cmd) {
	const detachedProcess, newProcessGroup = 0x00000008, 0x00000200
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: detachedProcess | newProcessGroup, HideWindow: true}
}
