package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/indu-forge/node_agent/internal/agent/executor"
	"github.com/indu-forge/node_agent/internal/agent/health"
	"github.com/indu-forge/node_agent/internal/agent/orchestrator"
	"github.com/indu-forge/node_agent/internal/agent/store"
	pkgConfig "github.com/indu-forge/node_agent/internal/pkg/config"
	"github.com/indu-forge/node_agent/internal/pkg/logger"
	"github.com/indu-forge/node_agent/internal/web/handler"
	"github.com/spf13/viper"
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
var initialWorkDir string

const defaultNodeAgentPort = 8081

func init() {
	workDir, err := os.Getwd()
	if err == nil {
		initialWorkDir = workDir
	}
}

// getRuntimeWorkDir 获取运行目录。
// 优先使用 NODE_AGENT_WORKDIR，其次使用进程启动目录，最后回退到可执行文件目录。
func getRuntimeWorkDir() string {
	if envDir := strings.TrimSpace(os.Getenv("NODE_AGENT_WORKDIR")); envDir != "" {
		return envDir
	}
	if initialWorkDir != "" {
		return initialWorkDir
	}
	exePath, err := os.Executable()
	if err == nil {
		return filepath.Dir(exePath)
	}
	return "."
}

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

	// 固定运行目录，避免 go run 时落到临时目录导致配置丢失。
	workDir := getRuntimeWorkDir()
	_ = os.Chdir(workDir)

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

	// 获取节点模式
	configPath := pkgConfig.GetConfigPath()
	nodeMode, _ := pkgConfig.GetMode(configPath)

	// 启动服务 - 交互模式下启动后台进程
	startServiceForeground(*config, nodeMode, configPath)
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

// startServiceForeground 在当前进程前台启动服务。
func startServiceForeground(config Config, nodeMode pkgConfig.NodeMode, configPath string) {
	storeInstance := store.NewLocalStore("./data")
	execInstance, err := createExecutor(config.Agent.Executor)
	if err != nil {
		fmt.Printf("  创建执行器失败: %v\n", err)
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

	fmt.Printf("NodeAgent 启动成功 | 模式=%s | 方式=前台单进程\n", nodeMode)
	fmt.Printf("监听地址: http://%s:%d\n", config.Agent.Listen.Host, config.Agent.Listen.Port)
	fmt.Printf("健康检查: http://%s:%d/health\n", config.Agent.Listen.Host, config.Agent.Listen.Port)

	runHTTPServer(config, storeInstance, orchInstance, healthChecker, nodeMode, configPath)
}

// runDaemon 后台守护进程模式（由 startServiceBackground 启动）
func runDaemon() {
	// 单例检查 - 只能运行一个实例
	if !checkSingleInstance(true) {
		writeDaemonLog("单例检查失败，退出")
		os.Exit(1)
	}

	// 固定运行目录，避免 go run 时落到临时目录导致配置丢失。
	workDir := getRuntimeWorkDir()
	_ = os.Chdir(workDir)

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
	notifyCenterOffline(nodeMode, configPath)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	fileLogger.Close()
	cleanupLockFile()
}

// notifyCenterOffline 在 Agent 正常退出时主动通知运维中心离线。
func notifyCenterOffline(nodeMode pkgConfig.NodeMode, configPath string) {
	if nodeMode != pkgConfig.ModeOnline {
		return
	}

	onlineConfig, err := pkgConfig.GetOnlineConfig(configPath)
	if err != nil || onlineConfig == nil {
		return
	}
	if onlineConfig.CenterURL == "" || onlineConfig.NodeID == "" {
		return
	}

	targetURL := strings.TrimRight(onlineConfig.CenterURL, "/") + "/api/v1/nodes/" + onlineConfig.NodeID + "/offline"
	payload := map[string]interface{}{
		"reason":    "agent_shutdown",
		"timestamp": time.Now().Format(time.RFC3339),
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewBuffer(body))
	if err != nil {
		fileLogger.Warn("构建主动下线通知失败", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if token := strings.TrimSpace(onlineConfig.RegistrationToken); token != "" {
		req.Header.Set("X-Registration-Token", token)
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fileLogger.Warn("主动下线通知失败", "endpoint", targetURL, "error", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		fileLogger.Warn("主动下线通知响应异常", "endpoint", targetURL, "status", resp.StatusCode)
		return
	}

	fileLogger.Info("已通知运维中心节点离线", "nodeId", onlineConfig.NodeID)
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

	// 支持从项目根目录 .env 覆盖监听端口，便于统一环境配置。
	applyRootEnvOverrides(&config)

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

// applyRootEnvOverrides 从环境配置覆盖关键配置项。
func applyRootEnvOverrides(config *Config) {
	port, ok := getNodeAgentPortFromEnv()
	if ok {
		config.Agent.Listen.Port = port
		return
	}

	// 未配置环境变量时使用默认端口，不再依赖 config.yaml 端口。
	config.Agent.Listen.Port = defaultNodeAgentPort
}

// getNodeAgentPortFromEnv 获取 NodeAgent 监听端口。
// 优先读取进程环境变量，其次读取项目根目录 .env 文件中的 NODE_AGENT_PORT。
func getNodeAgentPortFromEnv() (int, bool) {
	if value := strings.TrimSpace(os.Getenv("NODE_AGENT_PORT")); value != "" {
		if port, err := strconv.Atoi(value); err == nil && port > 0 && port <= 65535 {
			return port, true
		}
	}

	rootEnvPath, found := findProjectEnvFile()
	if !found {
		return 0, false
	}

	value, ok := readEnvValueFromFile(rootEnvPath, "NODE_AGENT_PORT")
	if !ok {
		return 0, false
	}

	port, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || port < 1 || port > 65535 {
		return 0, false
	}

	return port, true
}

// findProjectEnvFile 从多个候选目录向上查找 .env 文件。
func findProjectEnvFile() (string, bool) {
	candidateBases := make([]string, 0, 4)
	if initialWorkDir != "" {
		candidateBases = append(candidateBases, initialWorkDir)
	}

	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		candidateBases = append(candidateBases, cwd)
	}

	if exePath, err := os.Executable(); err == nil && exePath != "" {
		candidateBases = append(candidateBases, filepath.Dir(exePath))
	}

	if configPath := pkgConfig.GetConfigPath(); configPath != "" {
		if !filepath.IsAbs(configPath) {
			if cwd, err := os.Getwd(); err == nil {
				configPath = filepath.Join(cwd, configPath)
			}
		}
		candidateBases = append(candidateBases, filepath.Dir(configPath))
	}

	visited := make(map[string]struct{})
	for _, base := range candidateBases {
		dir := filepath.Clean(base)
		for i := 0; i < 10; i++ {
			if _, ok := visited[dir]; !ok {
				visited[dir] = struct{}{}
				envPath := filepath.Join(dir, ".env")
				if stat, err := os.Stat(envPath); err == nil && !stat.IsDir() {
					return envPath, true
				}
			}

			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	return "", false
}

// readEnvValueFromFile 从 .env 文件读取指定 key 的值。
func readEnvValueFromFile(filePath string, key string) (string, bool) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", false
	}

	lines := strings.Split(string(content), "\n")
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 兼容 "export KEY=VALUE" 写法。
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		currentKey := strings.TrimSpace(parts[0])
		if currentKey != key {
			continue
		}

		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"'`)
		return value, true
	}

	return "", false
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
