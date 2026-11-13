package main

import (
	"os/exec"
	"syscall"
)

func detachSetup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000200, HideWindow: true}
}
