package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/viper"
	"golang.org/x/sys/windows"
	"github.com/indu-forge/node_agent/internal/agent/executor"
	"github.com/indu-forge/node_agent/internal/agent/health"
	"github.com/indu-forge/node_agent/internal/agent/network"
	"github.com/indu-forge/node_agent/internal/agent/orchestrator"
	"github.com/indu-forge/node_agent/internal/agent/store"
	"github.com/indu-forge/node_agent/internal/pkg/autostart"
	"github.com/indu-forge/node_agent/internal/pkg/logger"
	"github.com/indu-forge/node_agent/internal/web/handler"
	pkgConfig "github.com/indu-forge/node_agent/internal/pkg/config"
)

type Config struct {
	Agent   AgentConfig   `mapstructure:"agent"`
	Logging LoggingConfig `mapstructure:"logging"`
}

type AgentConfig struct {
	ID       string           `mapstructure:"id"`
	Listen   ListenConfig     `mapstructure:"listen"`
	Executor ExecutorConfig   `mapstructure:"executor"`
}

type ListenConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type ExecutorConfig struct {
	Type    string           `mapstructure:"type"`
	Process ProcessConfig    `mapstructure:"process"`
	Docker  DockerConfig     `mapstructure:"docker"`
}

type ProcessConfig struct {
	WorkDir string `mapstructure:"workDir"`
	LogDir  string `mapstructure:"logDir"`
	Binary  string `mapstructure:"binary"`
}

type DockerConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	Socket     string `mapstructure:"socket"`
	Network    string `mapstructure:"network"`
	ImagePrefix string `mapstructure:"imagePrefix"`
}

type LoggingConfig struct {
	Level string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
	File   string `mapstructure:"file"`
}

// 全局日志器
var fileLogger *logger.FileLogger
var consoleLogger *logger.SimpleLogger

func main() {
	// 检查是否后台运行模式
	daemonMode := false
	for _, arg := range os.Args {
		if arg == "-d" || arg == "--daemon" || arg == "--hidden" {
			daemonMode = true
			break
		}
	}

	// 后台运行模式（无控制台）
	if daemonMode {
		runDaemon()
		return
	}

	// 单例检查 - 只能运行一个实例
	if !checkSingleInstance(false) {
		fmt.Println("错误: NodeAgent 已在运行中！")
		fmt.Println("请勿重复启动。")
		// 直接退出，不等待用户输入
		os.Exit(1)
	}

	// 获取程序所在目录
	exePath, _ := os.Executable()
	workDir := filepath.Dir(exePath)
	os.Chdir(workDir)

	// 初始化控制台日志器（用于交互界面）
	consoleLogger = logger.NewSimpleLogger(logger.LevelInfo)

	// 加载配置
	config, err := loadConfig()
	if err != nil {
		consoleLogger.Error(fmt.Sprintf("加载配置失败: %v", err))
		fmt.Println("按 Enter 退出...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
		os.Exit(1)
	}

	// 初始化文件日志器
	if err := initFileLogger(*config); err != nil {
		consoleLogger.Error(fmt.Sprintf("初始化文件日志失败: %v", err))
	}

	// 打印欢迎界面
	printWelcome(*config)

	// 获取节点模式
	configPath := pkgConfig.GetConfigPath()
	nodeMode, _ := pkgConfig.GetMode(configPath)

	// 初始化组件
	storeInstance := store.NewLocalStore("./data")
	execInstance, err := createExecutor(config.Agent.Executor)
	if err != nil {
		consoleLogger.Error(fmt.Sprintf("创建执行器失败: %v", err))
		pauseExit()
		return
	}

	orchInstance := orchestrator.NewOrchestrator(execInstance, storeInstance)

	healthChecker := health.NewHealthChecker(health.HealthConfig{
		Enabled:  true,
		Interval: 30 * time.Second,
		Timeout:  5 * time.Second,
		Retries:  3,
	})

	// 显示配置信息
	printConfigInfo(*config, nodeMode, workDir)

	// 询问是否设置开机自启动
	askAutoStart(*config)

	// 询问是否立即启动
	if !askStart() {
		fmt.Println("\n已取消启动。再见！")
		cleanupLockFile()
		return
	}

	// 启动服务 - 交互模式下启动后台进程
	startServiceBackground(*config, storeInstance, orchInstance, healthChecker, nodeMode, configPath)
}

// checkSingleInstance 检查是否已运行实例
// daemonMode: true 表示后台模式，不删除已有的 lock 文件
func checkSingleInstance(daemonMode bool) bool {
	// 锁定文件路径
	lockFile := filepath.Join(os.TempDir(), "node_agent.lock")

	// 检查锁定文件是否存在
	if _, err := os.Stat(lockFile); err == nil {
		// 文件存在，检查进程是否还在运行
		content, readErr := os.ReadFile(lockFile)
		if readErr == nil && len(content) > 0 {
			// 尝试解析 PID
			var pid int
			fmt.Sscanf(string(content), "%d", &pid)

			if pid > 0 {
				// 检查进程是否存在
				process, err := os.FindProcess(pid)
				if err == nil {
					// Windows 上 FindProcess 总是成功，需要用 Signal 检查
					// 发送信号 0 来检查进程是否存在
					err := process.Signal(syscall.Signal(0))
					if err == nil {
						// 进程还在运行
						return false
					}
				}

				// 进程已不存在，清理残留的锁定文件
				if !daemonMode {
					// 只有交互模式才清理残留的 lock 文件
					fmt.Println("检测到残留的锁定文件，正在清理...")
					os.Remove(lockFile)
				}
			}
		}
	}

	// 创建新的锁定文件
	file, err := os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0644)
	if err != nil {
		// 文件已存在，说明有其他实例
		return false
	}
	defer file.Close()

	// 写入当前 PID
	file.WriteString(fmt.Sprintf("%d", os.Getpid()))

	return true
}

// cleanupLockFile 清理锁定文件
func cleanupLockFile() {
	lockFile := filepath.Join(os.TempDir(), "node_agent.lock")
	os.Remove(lockFile)
}

// initFileLogger 初始化文件日志
func initFileLogger(config Config) error {
	logDir := "./logs"
	if config.Logging.File != "" {
		logDir = filepath.Dir(config.Logging.File)
	}

	var level logger.LogLevel
	switch strings.ToLower(config.Logging.Level) {
	case "debug":
		level = logger.LevelDebug
	case "warn", "warning":
		level = logger.LevelWarn
	case "error":
		level = logger.LevelError
	default:
		level = logger.LevelInfo
	}

	// 解析日志保留天数（默认7天）
	retention := 7
	if config.Logging.Format != "" {
		fmt.Sscanf(config.Logging.Format, "%d", &retention)
	}

	var err error
	fileLogger, err = logger.NewFileLogger(level, logDir, retention)
	return err
}

// printWelcome 打印欢迎界面
func printWelcome(config Config) {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  NodeAgent 节点管理服务")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("  版本: 1.0.0\n")
	fmt.Printf("  平台: %s %s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("  工作目录: %s\n", getWorkDir())
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()
}

// printConfigInfo 打印配置信息
func printConfigInfo(config Config, mode pkgConfig.NodeMode, workDir string) {
	fmt.Println("【配置信息】")
	fmt.Printf("  节点ID: %s\n", config.Agent.ID)
	fmt.Printf("  运行模式: %s\n", mode)
	fmt.Printf("  监听地址: %s:%d\n", config.Agent.Listen.Host, config.Agent.Listen.Port)
	fmt.Printf("  执行方式: %s\n", config.Agent.Executor.Type)
	fmt.Printf("  工作目录: %s\n", config.Agent.Executor.Process.WorkDir)
	fmt.Printf("  日志目录: %s\n", config.Agent.Executor.Process.LogDir)
	fmt.Println()
}

// askAutoStart 询问是否设置开机自启动
func askAutoStart(config Config) {
	fmt.Println("【开机自启动】")
	fmt.Println("  是否设置开机自启动？")
	fmt.Println("  [Y] 是")
	fmt.Println("  [N] 否（默认）")
	fmt.Print("  请选择: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if strings.ToUpper(input) == "Y" {
		if err := setAutoStart(config); err != nil {
			consoleLogger.Error(fmt.Sprintf("设置开机自启动失败: %v", err))
			fmt.Printf("  设置失败: %v\n", err)
		} else {
			fmt.Println("  已设置开机自启动")
		}
	} else {
		fmt.Println("  已跳过开机自启动设置")
	}
	fmt.Println()
}

// setAutoStart 设置开机自启动
func setAutoStart(config Config) error {
	exePath, _ := os.Executable()
	exePath, _ = filepath.Abs(exePath)

	starter := autostart.NewAutoStarter()
	cfg := autostart.Config{
		ExePath:    exePath,
		Name:       "node_agent",
		Args:       []string{"--hidden"},
		WorkingDir: filepath.Dir(exePath),
	}

	return starter.Enable(cfg)
}

// askStart 询问是否立即启动
func askStart() bool {
	fmt.Println("【启动服务】")
	fmt.Println("  是否立即启动服务？")
	fmt.Println("  [Y] 是（按 Enter 继续）")
	fmt.Println("  [N] 否，取消启动")
	fmt.Print("  请选择: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if strings.ToUpper(input) == "N" {
		return false
	}

	fmt.Println("\n  正在启动服务...")
	return true
}

// startServiceBackground 启动后台服务进程
func startServiceBackground(config Config, storeInstance *store.LocalStore, orchInstance *orchestrator.Orchestrator, healthChecker *health.HealthChecker, nodeMode pkgConfig.NodeMode, configPath string) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  正在启动后台服务...")

	// 获取程序路径和工作目录
	exePath, _ := os.Executable()
	workDir := filepath.Dir(exePath)

	// 启动后台进程 - 使用 DETACHED_PROCESS 创建完全独立的进程
	cmd := exec.Command(exePath, "--hidden")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | 0x00000008, // DETACHED_PROCESS = 0x00000008
	}
	cmd.Dir = workDir

	// 启动后台进程
	if err := cmd.Start(); err != nil {
		fmt.Printf("  启动失败: %v\n", err)
		fmt.Println("\n按 Enter 退出...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
		cleanupLockFile()
		os.Exit(1)
	}

	// 打印启动成功信息（不等待 lock 文件）
	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  服务已启动！")
	fmt.Printf("  监听地址: http://%s:%d\n", config.Agent.Listen.Host, config.Agent.Listen.Port)
	fmt.Printf("  健康检查: http://%s:%d/health\n", config.Agent.Listen.Host, config.Agent.Listen.Port)
	if nodeMode == pkgConfig.ModeOnline {
		fmt.Println("  运行模式: 在线模式")
	} else {
		fmt.Println("  运行模式: 离线模式")
	}
	fmt.Println()
	fmt.Println("  服务正在后台运行...")
	fmt.Println(strings.Repeat("=", 60))

	// 直接退出，强制关闭控制台窗口
	if runtime.GOOS == "windows" {
		user32 := windows.NewLazySystemDLL("user32.dll")
		kernel32 := windows.NewLazySystemDLL("kernel32.dll")

		// 获取并关闭控制台窗口
		getConsoleWindow := kernel32.NewProc("GetConsoleWindow")
		hwnd, _, _ := getConsoleWindow.Call()
		if hwnd != 0 {
			postMessage := user32.NewProc("PostMessageW")
			postMessage.Call(hwnd, 0x0010, 0, 0) // WM_CLOSE
		}

		// 分离控制台
		freeConsole := kernel32.NewProc("FreeConsole")
		freeConsole.Call()

		// 直接退出进程
		exitProcess := kernel32.NewProc("ExitProcess")
		exitProcess.Call(0)
	}
	os.Exit(0)
}

// exitProcess 直接退出进程
func exitProcess(exitCode int) {
	if runtime.GOOS == "windows" {
		kernel32 := windows.NewLazySystemDLL("kernel32.dll")
		ep := kernel32.NewProc("ExitProcess")
		ep.Call(uintptr(exitCode))
	}
}

// closeConsole 关闭控制台窗口
func closeConsole() {
	if runtime.GOOS == "windows" {
		user32 := windows.NewLazySystemDLL("user32.dll")
		kernel32 := windows.NewLazySystemDLL("kernel32.dll")

		// 获取控制台窗口句柄
		getConsoleWindow := kernel32.NewProc("GetConsoleWindow")
		hwnd, _, _ := getConsoleWindow.Call()

		if hwnd != 0 {
			// 发送 WM_CLOSE 消息关闭控制台窗口
			postQuitMessage := user32.NewProc("PostMessageW")
			postQuitMessage.Call(hwnd, 0x0010, 0, 0) // WM_CLOSE = 0x0010
		}

		// 分离控制台
		freeConsole := kernel32.NewProc("FreeConsole")
		freeConsole.Call()
	}
}

// runDaemon 后台守护进程模式（由 startServiceBackground 启动）
func runDaemon() {
	// 单例检查 - 只能运行一个实例
	if !checkSingleInstance() {
		writeDaemonLog("单例检查失败，退出")
		os.Exit(1)
	}

	// 获取程序所在目录
	exePath, _ := os.Executable()
	workDir := filepath.Dir(exePath)
	os.Chdir(workDir)

	// 加载配置
	config, err := loadConfig()
	if err != nil {
		writeDaemonLog(fmt.Sprintf("加载配置失败: %v", err))
		os.Exit(1)
	}

	// 初始化文件日志器
	if err := initFileLogger(*config); err != nil {
		writeDaemonLog(fmt.Sprintf("初始化文件日志失败: %v", err))
		os.Exit(1)
	}

	// 获取节点模式
	configPath := pkgConfig.GetConfigPath()
	nodeMode, _ := pkgConfig.GetMode(configPath)

	// 初始化组件
	storeInstance := store.NewLocalStore("./data")
	execInstance, err := createExecutor(config.Agent.Executor)
	if err != nil {
		fileLogger.Error(fmt.Sprintf("创建执行器失败: %v", err))
		cleanupLockFile()
		os.Exit(1)
	}

	orchInstance := orchestrator.NewOrchestrator(execInstance, storeInstance)

	healthChecker := health.NewHealthChecker(health.HealthConfig{
		Enabled:  true,
		Interval: 30 * time.Second,
		Timeout:  5 * time.Second,
		Retries:  3,
	})

	// 直接启动 HTTP 服务
	runHTTPServer(*config, storeInstance, orchInstance, healthChecker, nodeMode, configPath)
}

// runHTTPServer 运行 HTTP 服务（后台模式）
func runHTTPServer(config Config, storeInstance *store.LocalStore, orchInstance *orchestrator.Orchestrator, healthChecker *health.HealthChecker, nodeMode pkgConfig.NodeMode, configPath string) {
	// 创建 API 处理器
	apiHandler := handler.NewAPIHandler(orchInstance, storeInstance)

	// 创建 HTTP 服务器
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.Agent.Listen.Host, config.Agent.Listen.Port),
		Handler: apiHandler.RegisterRoutes(),
	}

	// 启动服务
	go func() {
		fileLogger.Info(fmt.Sprintf("启动 NodeAgent，监听地址: %s", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fileLogger.Error(fmt.Sprintf("服务启动失败: %v", err))
		}
	}()

	// 启动心跳（仅在线模式）
	if nodeMode == pkgConfig.ModeOnline {
		onlineConfig, _ := pkgConfig.GetOnlineConfig(configPath)
		if onlineConfig.CenterURL != "" {
			heartbeatEndpoint := fmt.Sprintf("%s/api/v1/nodes", onlineConfig.CenterURL)
			heartbeat := network.NewHeartbeatSender(
				network.HeartbeatConfig{
					Enabled:  true,
					Interval: 10 * time.Second,
					Endpoint: heartbeatEndpoint,
				},
				onlineConfig.NodeID,
				onlineConfig.RegistrationToken,
			)
			go heartbeat.Start(context.Background())
		}
	}

	// 启动健康检查
	go func() {
		for {
			projects, _ := storeInstance.ListProjects()
			for _, project := range projects {
				go healthChecker.MonitorProject(context.Background(), project.ID, "/health")
			}
			time.Sleep(60 * time.Second)
		}
	}()

	fileLogger.Info("NodeAgent 服务已启动")

	// 等待服务关闭信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// 关闭服务
	fileLogger.Info("正在关闭服务...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	fileLogger.Close()
	cleanupLockFile()
}

// writeDaemonLog 写入守护进程日志（控制台不可用时）
func writeDaemonLog(message string) {
	logPath := filepath.Join(os.TempDir(), "node_agent_daemon.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		f.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, message))
	}
}

// pauseExit 暂停并退出
func pauseExit() {
	fmt.Println("\n按 Enter 退出...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
	os.Exit(1)
}

// getWorkDir 获取工作目录
func getWorkDir() string {
	dir, _ := os.Getwd()
	return dir
}

func loadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func createExecutor(config ExecutorConfig) (executor.Executor, error) {
	switch config.Type {
	case "docker":
		return executor.NewDockerExecutor(config.Docker.Socket, config.Docker.Network, config.Docker.ImagePrefix)
	case "process":
	default:
		config.Type = "process"
	}

	return executor.NewProcessExecutor(config.Process.WorkDir, config.Process.LogDir, config.Process.Binary), nil
}
