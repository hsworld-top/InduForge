//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

// processMatchesCurrentExecutable 防止崩溃后的锁 PID 被其他进程复用。Linux
// 可以从 procfs 核对可执行文件；其他 Unix 平台保守沿用 PID 存活判断。
func processMatchesCurrentExecutable(pid int) bool {
	if runtime.GOOS != "linux" {
		return true
	}
	owner, err := os.Readlink(filepath.Join("/proc", fmt.Sprintf("%d", pid), "exe"))
	if err != nil {
		return false
	}
	current, err := os.Executable()
	if err != nil {
		return false
	}
	owner, ownerErr := filepath.EvalSymlinks(owner)
	current, currentErr := filepath.EvalSymlinks(current)
	return ownerErr == nil && currentErr == nil && owner == current
}
