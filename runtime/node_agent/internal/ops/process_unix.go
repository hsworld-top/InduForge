//go:build !windows

package ops

import (
	"os"
	"os/exec"
	"syscall"
)

func prepareChild(cmd *exec.Cmd)            { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }
func terminateChild(proc *os.Process) error { return syscall.Kill(-proc.Pid, syscall.SIGTERM) }
func processAlive(proc *os.Process) bool    { return proc.Signal(syscall.Signal(0)) == nil }
