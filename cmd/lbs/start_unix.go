//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

// setProcAttributes sets Unix-specific process attributes for daemon detachment
func setProcAttributes(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Create new session
	}
}
