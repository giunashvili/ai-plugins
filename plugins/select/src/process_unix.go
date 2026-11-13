//go:build darwin || linux

package main

import (
	"os/exec"
	"syscall"
)

func detachSetup(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }
