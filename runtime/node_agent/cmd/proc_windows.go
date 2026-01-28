//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

// setupDetachedProcess 设置进程为完全脱离模式 (Windows)
func setupDetachedProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// CREATE_NEW_PROCESS_GROUP (0x200) - 创建新进程组
		// DETACHED_PROCESS (0x8) - 与控制台脱离
		// CREATE_NO_WINDOW (0x08000000) - 不创建窗口
		// CREATE_BREAKAWAY_FROM_JOB (0x01000000) - 脱离父进程 Job，避免关闭窗口时被连带结束
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | 0x00000008 | 0x08000000 | 0x01000000,
	}
}

// isProcessRunning 判断进程是否仍在运行 (Windows)
func isProcessRunning(pid int) bool {
	const processQueryInfo = 0x0400
	const stillActive = 259

	handle, err := syscall.OpenProcess(processQueryInfo, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(handle)

	var exitCode uint32
	if err := syscall.GetExitCodeProcess(handle, &exitCode); err != nil {
		return false
	}

	return exitCode == stillActive
}
