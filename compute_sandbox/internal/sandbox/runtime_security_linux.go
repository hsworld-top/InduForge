//go:build linux

package sandbox

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	runtimeUID       = 10001
	linuxCapVersion3 = 0x20080522
	prGetNoNewPrivs  = 39
	prSetNoNewPrivs  = 38
	prCapBsetRead    = 23
	prCapBsetDrop    = 24
)

type capabilityHeader struct {
	Version uint32
	PID     int32
}

type capabilityData struct {
	Effective   uint32
	Permitted   uint32
	Inheritable uint32
}

type executionSecurity struct {
	UID, GID                          int
	Effective, Permitted, Inheritable uint64
	Bounding                          uint64
	NoNewPrivs                        bool
}

// RunVerifiedRuntime 只在 bubblewrap 完成隔离和降权后调用。任何安全属性不符都拒绝启动运行时执行器。
func RunVerifiedRuntime(command []string) error {
	if len(command) == 0 {
		return fmt.Errorf("隔离运行命令缺失")
	}
	if err := dropExecutionPrivileges(); err != nil {
		return fmt.Errorf("隔离子进程降权失败: %w", err)
	}
	security, err := readExecutionSecurity()
	if err != nil {
		return fmt.Errorf("读取隔离安全状态失败: %w", err)
	}
	if err := validateExecutionSecurity(security); err != nil {
		return err
	}
	return syscall.Exec(command[0], command, []string{"PATH=/usr/local/bin:/usr/bin:/bin", "LANG=C.UTF-8"})
}

// dropExecutionPrivileges 必须在 UID/GID 切换前清空 capability bounding set；降权后内核会清空其余能力集合。
func dropExecutionPrivileges() error {
	for capability := uintptr(0); capability < 64; capability++ {
		_, _, errno := syscall.RawSyscall(syscall.SYS_PRCTL, prCapBsetDrop, capability, 0)
		if errno != 0 && errno != syscall.EINVAL {
			return errno
		}
	}
	if err := syscall.Setgroups([]int{}); err != nil {
		return err
	}
	if err := syscall.Setgid(runtimeUID); err != nil {
		return err
	}
	if err := syscall.Setuid(runtimeUID); err != nil {
		return err
	}
	// bubblewrap 为降权 helper 保留了三项能力；setuid 后仍显式 capset 归零，不能依赖 securebits 默认值。
	header := capabilityHeader{Version: linuxCapVersion3}
	data := [2]capabilityData{}
	if _, _, errno := syscall.RawSyscall(syscall.SYS_CAPSET, uintptr(unsafe.Pointer(&header)), uintptr(unsafe.Pointer(&data[0])), 0); errno != 0 {
		return errno
	}
	if _, _, errno := syscall.RawSyscall(syscall.SYS_PRCTL, prSetNoNewPrivs, 1, 0); errno != 0 {
		return errno
	}
	return nil
}

func readExecutionSecurity() (executionSecurity, error) {
	header := capabilityHeader{Version: linuxCapVersion3}
	data := [2]capabilityData{}
	if _, _, errno := syscall.RawSyscall(syscall.SYS_CAPGET, uintptr(unsafe.Pointer(&header)), uintptr(unsafe.Pointer(&data[0])), 0); errno != 0 {
		return executionSecurity{}, errno
	}
	result := executionSecurity{
		UID: os.Geteuid(), GID: os.Getegid(),
		Effective:   uint64(data[0].Effective) | uint64(data[1].Effective)<<32,
		Permitted:   uint64(data[0].Permitted) | uint64(data[1].Permitted)<<32,
		Inheritable: uint64(data[0].Inheritable) | uint64(data[1].Inheritable)<<32,
	}
	value, _, errno := syscall.RawSyscall(syscall.SYS_PRCTL, prGetNoNewPrivs, 0, 0)
	if errno != 0 {
		return executionSecurity{}, errno
	}
	result.NoNewPrivs = value == 1
	for capability := uintptr(0); capability < 64; capability++ {
		value, _, errno = syscall.RawSyscall(syscall.SYS_PRCTL, prCapBsetRead, capability, 0)
		if errno != 0 {
			break
		}
		if value == 1 {
			result.Bounding |= uint64(1) << capability
		}
	}
	return result, nil
}

func validateExecutionSecurity(value executionSecurity) error {
	if value.UID != runtimeUID || value.GID != runtimeUID {
		return fmt.Errorf("隔离子进程身份校验失败")
	}
	if value.Effective != 0 || value.Permitted != 0 || value.Inheritable != 0 || value.Bounding != 0 {
		return fmt.Errorf("隔离子进程能力校验失败")
	}
	if !value.NoNewPrivs {
		return fmt.Errorf("隔离子进程提权保护校验失败")
	}
	return nil
}
