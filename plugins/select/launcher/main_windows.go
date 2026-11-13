// Windows supplies .exe when the shared MCP command ends in /server/select.
// This small x64 launcher also runs under Windows 11 ARM64 emulation, then
// starts the native ARM64 Select executable. End users need no Go runtime.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"
)

func main() {
	arch := "amd64"
	kernel := syscall.NewLazyDLL("kernel32.dll")
	query := kernel.NewProc("IsWow64Process2")
	if query.Find() == nil {
		process, _, _ := kernel.NewProc("GetCurrentProcess").Call()
		var emulated, native uint16
		ok, _, err := query.Call(process, uintptr(unsafe.Pointer(&emulated)), uintptr(unsafe.Pointer(&native)))
		if ok == 0 {
			fail(fmt.Errorf("detect processor: %w", err))
		}
		switch native {
		case 0xaa64:
			arch = "arm64"
		case 0x8664:
		default:
			fail(errors.New("x64 or ARM64 Windows is required"))
		}
	}
	self, err := os.Executable()
	if err != nil {
		fail(err)
	}
	command := exec.Command(filepath.Join(filepath.Dir(self), "windows-"+arch, "select.exe"), os.Args[1:]...)
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			os.Exit(exit.ExitCode())
		}
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "Select launcher:", err)
	os.Exit(1)
}
