package autostart

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Config 自启动配置
type Config struct {
	ExePath    string   // 程序路径
	Name       string   // 服务名称
	Args       []string // 启动参数
	WorkingDir string   // 工作目录
}

// AutoStarter 自启动接口
type AutoStarter interface {
	Enable(cfg Config) error
	Disable(cfg Config) error
	IsEnabled(cfg Config) (bool, error)
}

// platformDetect 检测当前平台
func platformDetect() string {
	return runtime.GOOS
}

// NewAutoStarter 创建对应平台的自启动器
func NewAutoStarter() AutoStarter {
	switch platformDetect() {
	case "windows":
		return &WindowsAutoStart{}
	case "linux":
		return &LinuxAutoStart{}
	default:
		return &LinuxAutoStart{}
	}
}

// WindowsAutoStart Windows 自启动实现
type WindowsAutoStart struct{}

// Enable 启用 Windows 自启动（注册表方式）
func (w *WindowsAutoStart) Enable(cfg Config) error {
	// 获取注册表路径
	key := `HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run`

	// 构建命令行参数
	var cmdLine string
	if len(cfg.Args) > 0 {
		cmdLine = fmt.Sprintf(`"%s" %s`, cfg.ExePath, joinArgs(cfg.Args))
	} else {
		cmdLine = fmt.Sprintf(`"%s"`, cfg.ExePath)
	}

	// 使用 reg add 命令
	// 注意：reg 命令需要在注册表路径前加引号，如果路径包含空格
	cmd := exec.Command("reg", "add", key, "/v", cfg.Name, "/t", "REG_SZ", "/d", cmdLine, "/f")
	hideCommandWindow(cmd)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("注册表写入失败: %v, 输出: %s", err, string(output))
	}

	return nil
}

// Disable 禁用 Windows 自启动
func (w *WindowsAutoStart) Disable(cfg Config) error {
	key := fmt.Sprintf(`HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run`)

	cmd := exec.Command("reg", "delete", key, "/v", cfg.Name, "/f")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// 如果键不存在，删除可能失败，忽略该错误
		if !contains(string(output), "系统找不到") {
			return fmt.Errorf("注册表删除失败: %v, 输出: %s", err, string(output))
		}
	}

	return nil
}

// IsEnabled 检查是否已启用自启动
func (w *WindowsAutoStart) IsEnabled(cfg Config) (bool, error) {
	key := fmt.Sprintf(`HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run`)

	cmd := exec.Command("reg", "query", key, "/v", cfg.Name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, nil // 不存在或查询失败
	}

	return contains(string(output), cfg.Name), nil
}

// LinuxAutoStart Linux 自启动实现
type LinuxAutoStart struct{}

// Enable 启用 Linux 自启动（用户级 systemd 或 rc.local）
func (w *LinuxAutoStart) Enable(cfg Config) error {
	// 优先尝试 systemd 用户服务
	if err := w.enableSystemdUser(cfg); err == nil {
		return nil
	}

	// 回退到 rc.local 方式
	return w.enableRcLocal(cfg)
}

// Disable 禁用 Linux 自启动
func (w *LinuxAutoStart) Disable(cfg Config) error {
	// 尝试 systemd 用户服务
	if err := w.disableSystemdUser(cfg); err == nil {
		return nil
	}

	// 回退到 rc.local 方式
	return w.disableRcLocal(cfg)
}

// IsEnabled 检查是否已启用自启动
func (w *LinuxAutoStart) IsEnabled(cfg Config) (bool, error) {
	// 检查 systemd 用户服务
	if enabled, _ := w.isSystemdUserEnabled(cfg); enabled {
		return true, nil
	}

	// 检查 rc.local
	return w.isRcLocalEnabled(cfg)
}

// enableSystemdUser 启用 systemd 用户服务
func (w *LinuxAutoStart) enableSystemdUser(cfg Config) error {
	serviceDir := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user")
	serviceFile := filepath.Join(serviceDir, fmt.Sprintf("%s.service", cfg.Name))

	// 确保目录存在
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		return fmt.Errorf("创建 systemd 服务目录失败: %w", err)
	}

	// 生成服务文件内容
	serviceContent := fmt.Sprintf(`[Unit]
Description=NodeAgent Service
After=network.target

[Service]
Type=simple
ExecStart=%s %s
WorkingDirectory=%s
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
`, cfg.ExePath, joinArgs(cfg.Args), cfg.WorkingDir)

	// 写入服务文件
	if err := os.WriteFile(serviceFile, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("写入 systemd 服务文件失败: %w", err)
	}

	// 重新加载 systemd 并启用服务
	cmd := exec.Command("systemctl", "--user", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("重新加载 systemd 失败: %w", err)
	}

	cmd = exec.Command("systemctl", "--user", "enable", cfg.Name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("启用 systemd 服务失败: %w", err)
	}

	cmd = exec.Command("systemctl", "--user", "start", cfg.Name)
	return cmd.Run()
}

// disableSystemdUser 禁用 systemd 用户服务
func (w *LinuxAutoStart) disableSystemdUser(cfg Config) error {
	cmd := exec.Command("systemctl", "--user", "stop", cfg.Name)
	cmd.Run()

	cmd = exec.Command("systemctl", "--user", "disable", cfg.Name)
	cmd.Run()

	serviceFile := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user", fmt.Sprintf("%s.service", cfg.Name))
	os.Remove(serviceFile)

	cmd = exec.Command("systemctl", "--user", "daemon-reload")
	return cmd.Run()
}

// isSystemdUserEnabled 检查 systemd 用户服务是否启用
func (w *LinuxAutoStart) isSystemdUserEnabled(cfg Config) (bool, error) {
	cmd := exec.Command("systemctl", "--user", "is-enabled", cfg.Name)
	err := cmd.Run()
	return err == nil, nil
}

// enableRcLocal 启用 rc.local 自启动
func (w *LinuxAutoStart) enableRcLocal(cfg Config) error {
	rcLocalPath := "/etc/rc.local"
	backupPath := rcLocalPath + ".backup"

	// 备份原文件
	if _, err := os.Stat(rcLocalPath); err == nil {
		data, _ := os.ReadFile(rcLocalPath)
		os.WriteFile(backupPath, data, 0644)
	}

	// 构建启动命令
	startCmd := fmt.Sprintf(`cd %s && nohup %s %s > %s/agent.log 2>&1 &`,
		cfg.WorkingDir, cfg.ExePath, joinArgs(cfg.Args), cfg.WorkingDir)

	// 写入 rc.local
	var content string
	if _, err := os.Stat(rcLocalPath); os.IsNotExist(err) {
		content = fmt.Sprintf(`#!/bin/bash
%s
`, startCmd)
	} else {
		data, _ := os.ReadFile(rcLocalPath)
		content = string(data) + "\n" + startCmd + "\n"
	}

	return os.WriteFile(rcLocalPath, []byte(content), 0755)
}

// disableRcLocal 禁用 rc.local 自启动
func (w *LinuxAutoStart) disableRcLocal(cfg Config) error {
	rcLocalPath := "/etc/rc.local"

	if _, err := os.Stat(rcLocalPath); err != nil {
		return nil
	}

	data, _ := os.ReadFile(rcLocalPath)
	lines := strings.Split(string(data), "\n")
	var newLines []string

	for _, line := range lines {
		if !contains(line, cfg.ExePath) {
			newLines = append(newLines, line)
		}
	}

	return os.WriteFile(rcLocalPath, []byte(strings.Join(newLines, "\n")), 0755)
}

// isRcLocalEnabled 检查 rc.local 是否包含此服务
func (w *LinuxAutoStart) isRcLocalEnabled(cfg Config) (bool, error) {
	rcLocalPath := "/etc/rc.local"

	if _, err := os.Stat(rcLocalPath); err != nil {
		return false, nil
	}

	data, _ := os.ReadFile(rcLocalPath)
	return contains(string(data), cfg.ExePath), nil
}

// joinArgs 拼接命令行参数
func joinArgs(args []string) string {
	var result string
	for _, arg := range args {
		result += " " + arg
	}
	return strings.TrimSpace(result)
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
