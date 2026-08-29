//go:build windows

package ops

import (
	"os"
	"os/exec"
)

func prepareChild(cmd *exec.Cmd)            {}
func terminateChild(proc *os.Process) error { return proc.Kill() }
func processAlive(proc *os.Process) bool    { return proc.Signal(os.Signal(nil)) == nil }
