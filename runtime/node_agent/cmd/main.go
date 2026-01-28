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

	"github.com/indu-forge/node_agent/internal/agent/executor"
	"github.com/indu-forge/node_agent/internal/agent/health"
	"github.com/indu-forge/node_agent/internal/agent/network"
	"github.com/indu-forge/node_agent/internal/agent/orchestrator"
	"github.com/indu-forge/node_agent/internal/agent/store"
	"github.com/indu-forge/node_agent/internal/pkg/autostart"
	pkgConfig "github.com/indu-forge/node_agent/internal/pkg/config"
	"github.com/indu-forge/node_agent/internal/pkg/logger"
	"github.com/indu-forge/node_agent/internal/web/handler"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
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
	// 开发模式：跳过交互，直接运行服务
	if isDevMode() {
		runDaemon()
		return
	}

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
	printWelcome()

	// 获取节点模式
	configPath := pkgConfig.GetConfigPath()
	nodeMode, _ := pkgConfig.GetMode(configPath)

	// 显示配置信息
	printConfigInfo(*config, nodeMode)

	// 询问是否设置开机自启动
	askAutoStart()

	// 询问端口配置
	newPort := askPort(config.Agent.Listen.Port)
	if newPort != config.Agent.Listen.Port {
		// 端口已更改，保存到配置文件
		if err := savePortConfig(configPath, newPort); err != nil {
			fmt.Printf("  警告: 保存端口配置失败: %v\n", err)
		} else {
			config.Agent.Listen.Port = newPort
			fmt.Printf("  端口配置已更新为: %d\n", newPort)
		}
	}

	// 询问是否立即启动
	if !askStart() {
		fmt.Println("\n已取消启动。再见！")
		cleanupLockFile()
		return
	}

	// 启动服务 - 交互模式下启动后台进程
	startServiceBackground(*config, nodeMode, newPort)
}

// isDevMode 判断是否为开发模式
func isDevMode() bool {
	return strings.ToLower(os.Getenv("NODE_AGENT_ENV")) == "development"
}

// checkSingleInstance 检查是否已运行实例
// daemonMode: true 表示后台模式，不删除已有的 lock 文件
func checkSingleInstance(daemonMode bool) bool {
	if isDevMode() {
		return true
	}
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
				// 守护进程模式下，如果锁文件是由当前父进程（交互窗口）写入，则允许接管
				if daemonMode && pid == os.Getppid() {
					_ = os.Remove(lockFile)
				} else {
					// 检查进程是否存在
					if isProcessRunning(pid) {
						// 进程还在运行
						return false
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
func printWelcome() {
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
func printConfigInfo(config Config, mode pkgConfig.NodeMode) {
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
func askAutoStart() {
	fmt.Println("【开机自启动】")
	fmt.Println("  是否设置开机自启动？")
	fmt.Println("  [Y] 是")
	fmt.Println("  [N] 否（默认）")
	fmt.Print("  请选择: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if strings.ToUpper(input) == "Y" {
		if err := setAutoStart(); err != nil {
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
func setAutoStart() error {
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

// savePortConfig 保存端口配置到配置文件
func savePortConfig(configPath string, port int) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}

	// 更新配置
	agent, ok := config["agent"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid config structure")
	}

	listen, ok := agent["listen"].(map[string]interface{})
	if !ok {
		listen = make(map[string]interface{})
		agent["listen"] = listen
	}
	listen["port"] = port

	// 写回配置文件
	newData, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, newData, 0644)
}

// askPort 询问监听端口
func askPort(currentPort int) int {
	fmt.Println("【端口配置】")
	fmt.Printf("  当前监听端口: %d\n", currentPort)
	fmt.Println("  请输入新的监听端口（1-65535），或直接按 Enter 使用当前端口: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return currentPort
	}

	var newPort int
	if _, err := fmt.Sscanf(input, "%d", &newPort); err != nil || newPort < 1 || newPort > 65535 {
		fmt.Println("  无效的端口号，将使用当前端口")
		return currentPort
	}

	return newPort
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
func startServiceBackground(config Config, nodeMode pkgConfig.NodeMode, customPort int) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  正在启动后台服务...")
	fmt.Println(strings.Repeat("=", 60))

	// 获取程序路径和工作目录
	exePath, _ := os.Executable()
	workDir := filepath.Dir(exePath)

	// 构建启动参数
	args := []string{"--hidden"}
	if customPort != 0 && customPort != config.Agent.Listen.Port {
		args = append(args, "--port", fmt.Sprintf("%d", customPort))
	}

	// 启动完全独立的后台进程
	var processPID int

	// 构建命令
	cmd := exec.Command(exePath, args...)
	cmd.Dir = workDir

	// 使用平台特定的设置，确保进程完全脱离
	setupDetachedProcess(cmd)

	// 启动进程
	if err := cmd.Start(); err != nil {
		fmt.Printf("  启动失败: %v\n", err)
		pauseBeforeExit()
		os.Exit(1)
	}

	// 获取进程 PID
	processPID = cmd.Process.Pid

	// 释放进程句柄，使进程完全独立
	// 这样即使父进程退出，子进程也不会受影响
	if err := cmd.Process.Release(); err != nil {
		fmt.Printf("  警告: 无法释放进程句柄: %v\n", err)
	}

	// 确定实际使用的端口
	actualPort := config.Agent.Listen.Port
	if customPort != 0 && customPort != config.Agent.Listen.Port {
		actualPort = customPort
	}

	fmt.Println()
	fmt.Println("  ⏳ 启动中，请稍候...")
	fmt.Println()

	// 等待服务初始化并检查健康状态
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		time.Sleep(1 * time.Second)
		fmt.Printf("\r  正在检查服务状态... [%d/%d]", i+1, maxRetries)

		// 尝试连接健康检查端点
		healthURL := fmt.Sprintf("http://%s:%d/health", config.Agent.Listen.Host, actualPort)
		resp, err := http.Get(healthURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			fmt.Println(" ✓")
			break
		}
		if resp != nil {
			resp.Body.Close()
		}

		if i == maxRetries-1 {
			fmt.Println(" !")
			fmt.Println("\n  警告: 服务可能未完全启动，请检查日志")
		}
	}

	// 打印启动成功信息
	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  ✓ 服务启动成功！")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()
	fmt.Println("【服务信息】")
	if processPID > 0 {
		fmt.Printf("  进程 PID: %d\n", processPID)
	}
	fmt.Printf("  监听地址: http://%s:%d\n", config.Agent.Listen.Host, actualPort)
	fmt.Printf("  健康检查: http://%s:%d/health\n", config.Agent.Listen.Host, actualPort)
	fmt.Printf("  工作目录: %s\n", workDir)
	if nodeMode == pkgConfig.ModeOnline {
		fmt.Println("  运行模式: 在线模式")
	} else {
		fmt.Println("  运行模式: 离线模式")
	}
	fmt.Println()
	fmt.Println("【运行状态】")
	fmt.Println("  服务已在后台运行")
	fmt.Println("  关闭此窗口不会影响服务运行")
	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()
	fmt.Println("  提示: 按任意键关闭此窗口...")

	// 等待用户按键
	waitForKeyPress()
}

// runDaemon 后台守护进程模式（由 startServiceBackground 启动）
func runDaemon() {
	// 单例检查 - 只能运行一个实例
	if !checkSingleInstance(true) {
		writeDaemonLog("单例检查失败，退出")
		os.Exit(1)
	}

	// 获取程序所在目录
	exePath, _ := os.Executable()
	workDir := filepath.Dir(exePath)
	os.Chdir(workDir)

	// 解析命令行参数
	customPort := 0
	for i := 0; i < len(os.Args); i++ {
		if os.Args[i] == "--port" && i+1 < len(os.Args) {
			fmt.Sscanf(os.Args[i+1], "%d", &customPort)
			break
		}
	}

	// 加载配置
	config, err := loadConfig()
	if err != nil {
		writeDaemonLog(fmt.Sprintf("加载配置失败: %v", err))
		os.Exit(1)
	}

	// 如果指定了自定义端口，先保存再使用
	if customPort > 0 && customPort != config.Agent.Listen.Port {
		configPath := pkgConfig.GetConfigPath()
		if err := savePortConfig(configPath, customPort); err != nil {
			writeDaemonLog(fmt.Sprintf("保存端口配置失败: %v", err))
		}
		config.Agent.Listen.Port = customPort
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


// getWorkDir 获取工作目录
func getWorkDir() string {
	dir, _ := os.Getwd()
	return dir
}

func loadConfig() (*Config, error) {
	if consoleLogger == nil {
		consoleLogger = logger.NewSimpleLogger(logger.LevelInfo)
	}
	// 配置文件直接放在当前目录
	configPath := "./config.yaml"

	// 检查配置文件是否存在，不存在则生成默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := generateDefaultConfig(configPath); err != nil {
			return nil, fmt.Errorf("生成默认配置失败: %w", err)
		}
		consoleLogger.Info(fmt.Sprintf("已生成默认配置文件: %s", configPath))
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

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

// generateDefaultConfig 生成默认配置文件
func generateDefaultConfig(configPath string) error {
	defaultConfig := `agent:
    executor:
        docker:
            enabled: false
            imagePrefix: node_agent_runtime
            network: node_agent
            socket: /var/run/docker.sock
        process:
            binary: runtime_engine
            logDir: ./logs
            workDir: ./runtime
        systemd:
            enabled: false
            unitTemplate: /etc/systemd/system/node_agent_{project}.service
        type: process
    listen:
        host: 0.0.0.0
        port: 8081
    mode: offline
    network:
        heartbeat:
            enabled: true
            endpoint: http://manager:9099/api/v1/nodes/heartbeat
            interval: 10s
    online:
        centerUrl: ""
        nodeId: ""
        registrationToken: ""
    runtime:
        healthCheck:
            enabled: true
            endpoint: /health
            interval: 30s
            retries: 3
            timeout: 5s
        workDir: ./runtime
logging:
    file: ./logs/agent.log
    format: json
    level: info
    output: stdout
storage:
    local:
        dataDir: ./data
    type: local
`

	return os.WriteFile(configPath, []byte(defaultConfig), 0644)
}

// waitForKeyPress 等待用户按键
func waitForKeyPress() {
	reader := bufio.NewReader(os.Stdin)
	reader.ReadByte()
}

// pauseBeforeExit 暂停并等待用户按键后退出
func pauseBeforeExit() {
	fmt.Println()
	fmt.Println("  按任意键退出...")
	waitForKeyPress()
	cleanupLockFile()
}
