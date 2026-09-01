//go:build windows

package main

import (
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/windows/svc"
)

var windowsServiceStop = make(chan struct{})
var windowsServiceStopOnce sync.Once

// runWindowsService 为 Windows Service Control Manager 提供原生 ServiceMain；直接用
// sc.exe 启动普通 console 进程会导致 1053，故安装器始终传入 --service。
func runWindowsService() {
	if interactive, err := svc.IsWindowsService(); err != nil || !interactive {
		runDaemon()
		return
	}
	if err := svc.Run("InduForgeNodeAgent", windowsServiceHandler{}); err != nil {
		os.Exit(1)
	}
}

type windowsServiceHandler struct{}

func (windowsServiceHandler) Execute(_ []string, changes <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending}
	daemonDone := make(chan struct{})
	go func() {
		runDaemon()
		close(daemonDone)
	}()

	running := svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	status <- running
	for change := range changes {
		switch change.Cmd {
		case svc.Stop, svc.Shutdown:
			status <- svc.Status{State: svc.StopPending}
			windowsServiceStopOnce.Do(func() { close(windowsServiceStop) })
			// SCM 收到停止请求后不能马上退出：runDaemon 会停止控制面轮询、
			// 回收 workload 子进程、关闭 HTTP 服务并清理单例锁。定期上报
			// StopPending checkpoint，避免清理时间较长时被 SCM 误判为卡死。
			return false, waitDaemonShutdown(daemonDone, status)
		case svc.Interrogate:
			status <- running
		}
	}
	// 控制通道异常关闭时同样走正常关闭路径，避免残留子进程和 lock 文件。
	windowsServiceStopOnce.Do(func() { close(windowsServiceStop) })
	return false, waitDaemonShutdown(daemonDone, status)
}

// waitDaemonShutdown 等待 runDaemon 完成既有的优雅清理。Windows SCM 的默认等待
// 时间有限，因此每两秒刷新一次 checkpoint，明确服务仍在停止中。
func waitDaemonShutdown(done <-chan struct{}, status chan<- svc.Status) uint32 {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	var checkpoint uint32 = 1
	for {
		select {
		case <-done:
			return 0
		case <-ticker.C:
			checkpoint++
			status <- svc.Status{
				State:      svc.StopPending,
				WaitHint:   uint32((10 * time.Second) / time.Millisecond),
				CheckPoint: checkpoint,
			}
		}
	}
}
func serviceStopSignal() <-chan struct{} { return windowsServiceStop }

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

// Windows 继续由进程句柄判活；安装路径身份校验后续使用 Win32 image path API 补充。
func processMatchesCurrentExecutable(_ int) bool { return true }
