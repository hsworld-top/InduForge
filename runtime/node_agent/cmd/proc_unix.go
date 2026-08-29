//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

func runWindowsService()                 { runDaemon() }
func serviceStopSignal() <-chan struct{} { return nil }

// setupDetachedProcess 设置进程为完全脱离模式 (Unix/Linux)
func setupDetachedProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // 创建新会话，完全脱离终端
	}
}

// isProcessRunning 判断进程是否仍在运行 (Unix/Linux)
func isProcessRunning(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil
}
