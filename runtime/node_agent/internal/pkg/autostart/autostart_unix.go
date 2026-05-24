//go:build !windows

package autostart

import "os/exec"

// hideCommandWindow 在非 Windows 平台不需要处理窗口隐藏。
func hideCommandWindow(cmd *exec.Cmd) {}
