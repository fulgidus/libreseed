//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

// setProcAttributes sets Windows-specific process attributes for daemon detachment
func setProcAttributes(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | 0x08000000, // DETACHED_PROCESS
	}
}
