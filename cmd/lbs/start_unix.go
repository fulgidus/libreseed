//go:build !windows

package main

import (
	"os"
	"os/exec"
	"syscall"
)

// setProcAttributes sets Unix-specific process attributes for daemon detachment
func setProcAttributes(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Create new session
	}

	// Redirect stdin to /dev/null to prevent the daemon from reading from parent's stdin
	// This is critical for proper daemonization
	devNull, err := os.Open(os.DevNull)
	if err == nil {
		cmd.Stdin = devNull
	}
}
