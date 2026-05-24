//go:build windows

package autostart

import (
	"os/exec"
	"syscall"
)

// hideCommandWindow 避免 Windows 注册表命令弹出控制台窗口。
func hideCommandWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
}
