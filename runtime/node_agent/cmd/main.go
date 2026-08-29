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
	"github.com/indu-forge/node_agent/internal/ops"
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
	ID       string         `mapstructure:"id"`
	Listen   ListenConfig   `mapstructure:"listen"`
	Executor ExecutorConfig `mapstructure:"executor"`
	Ops      OpsConfig      `mapstructure:"ops"`
}

// OpsConfig 是运维控制面连接配置；包安装器会写入 serverUrl/code/role，身份凭据单独持久化。
type OpsConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	ServerURL      string `mapstructure:"serverUrl"`
	EnrollmentCode string `mapstructure:"enrollmentCode"`
	Role           string `mapstructure:"role"`
	HeartbeatEvery string `mapstructure:"heartbeatEvery"`
	DataDir        string `mapstructure:"dataDir"`
	DemoRuntime    bool   `mapstructure:"demoRuntime"`
}

type ListenConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type ExecutorConfig struct {
	Type    string        `mapstructure:"type"`
	Process ProcessConfig `mapstructure:"process"`
	Docker  DockerConfig  `mapstructure:"docker"`
}

type ProcessConfig struct {
	WorkDir string `mapstructure:"workDir"`
	LogDir  string `mapstructure:"logDir"`
	Binary  string `mapstructure:"binary"`
}

type DockerConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	Socket      string `mapstructure:"socket"`
	Network     string `mapstructure:"network"`
	ImagePrefix string `mapstructure:"imagePrefix"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
	File   string `mapstructure:"file"`
}

// 全局日志器
var fileLogger *logger.FileLogger
var consoleLogger *logger.SimpleLogger
var initialWorkDir string

const defaultNodeAgentPort = 17601

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

// nodeAgentExecutable 返回可重新拉起当前节点代理的绝对路径。
// Supervisor 会将子进程的工作目录切换到 workload 目录，因此不能直接使用可能为相对路径的 os.Args[0]。
func nodeAgentExecutable() string {
	if executable, err := os.Executable(); err == nil && strings.TrimSpace(executable) != "" {
		if absolute, absErr := filepath.Abs(executable); absErr == nil {
			return absolute
		}
		return executable
	}
	if absolute, err := filepath.Abs(os.Args[0]); err == nil {
		return absolute
	}
	return os.Args[0]
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--service" {
		runWindowsService()
		return
	}
	if len(os.Args) >= 3 && os.Args[1] == "demo-workload" {
		runDemoWorkload(os.Args[2])
		return
	}
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
	lockFile := nodeAgentLockFile()

	// 检查锁定文件是否存在
	if _, err := os.Stat(lockFile); err == nil {
		// 文件存在，检查进程是否还在运行
		content, readErr := os.ReadFile(lockFile)
		if readErr == nil && len(content) > 0 {
			// 尝试解析 PID
			var pid int
			fmt.Sscanf(string(content), "%d", &pid)

			if pid > 0 {
				// 守护进程模式下允许接管交互父进程的锁。容器或服务异常退出后，
				// 操作系统还可能把旧 PID 分配给本次新进程；本进程尚未创建锁，
				// 因此 pid == Getpid 必然是残留锁，也应安全清理。
				if daemonMode && (pid == os.Getppid() || pid == os.Getpid()) {
					_ = os.Remove(lockFile)
				} else {
					// 检查进程是否存在
					if isProcessRunning(pid) {
						// 进程还在运行
						return false
					}

					// 进程已不存在时，无论前台还是守护模式都必须清理残留锁；
					// 否则一次崩溃会让服务永久无法再次启动。
					if !daemonMode {
						fmt.Println("检测到残留的锁定文件，正在清理...")
					}
					_ = os.Remove(lockFile)
				}
			} else {
				_ = os.Remove(lockFile)
			}
		} else {
			_ = os.Remove(lockFile)
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
	os.Remove(nodeAgentLockFile())
}

func nodeAgentLockFile() string {
	if path := strings.TrimSpace(os.Getenv("NODE_AGENT_LOCK_FILE")); path != "" {
		return path
	}
	return filepath.Join(os.TempDir(), "node_agent.lock")
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
	storeInstance := store.NewLocalStore(getDataDir(config))
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
	storeInstance := store.NewLocalStore(getDataDir(*config))
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
	supervisor := ops.NewSupervisor(nodeAgentExecutable(), config.Agent.Executor.Process.WorkDir, config.Agent.Executor.Process.LogDir)
	apiHandler := handler.NewAPIHandler(orchInstance, storeInstance).WithSupervisor(supervisor)
	opsContext, cancelOps := context.WithCancel(context.Background())
	if agent, err := newOpsAgent(config, supervisor, storeInstance.DataDir()); err != nil {
		fileLogger.Warn("运维控制面配置无效，跳过启动", "error", err)
	} else if agent != nil {
		go func() {
			if err := agent.Run(opsContext); err != nil {
				fileLogger.Warn("运维控制面连接已停止", "error", err)
			}
		}()
	}

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
	select {
	case <-quit:
	case <-serviceStopSignal():
	}

	// 关闭服务
	fileLogger.Info("正在关闭服务...")
	cancelOps()
	supervisor.Shutdown()
	notifyCenterOffline(nodeMode, configPath)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	fileLogger.Close()
	cleanupLockFile()
}

func getDataDir(config Config) string {
	if value := strings.TrimSpace(os.Getenv("NODE_AGENT_DATA_DIR")); value != "" {
		return value
	}
	if value := strings.TrimSpace(config.Agent.Ops.DataDir); value != "" {
		return value
	}
	return "./data"
}

func newOpsAgent(config Config, supervisor *ops.Supervisor, defaultDataDir string) (*ops.Agent, error) {
	if !config.Agent.Ops.Enabled {
		return nil, nil
	}
	interval, err := time.ParseDuration(config.Agent.Ops.HeartbeatEvery)
	if config.Agent.Ops.HeartbeatEvery == "" {
		interval, err = 10*time.Second, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ops heartbeatEvery 无效: %w", err)
	}
	return ops.NewAgent(ops.Config{Enabled: true, ServerURL: config.Agent.Ops.ServerURL, EnrollmentCode: config.Agent.Ops.EnrollmentCode, Role: config.Agent.Ops.Role, HeartbeatEvery: interval, DataDir: defaultDataDir, DemoRuntime: config.Agent.Ops.DemoRuntime, ClearEnrollmentCode: func() error { return pkgConfig.ClearOpsEnrollmentCode(pkgConfig.GetConfigPath()) }}, supervisor)
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
	// 安装包通过 NODE_AGENT_CONFIG 指定配置位置；开发环境仍默认当前目录。
	configPath := pkgConfig.GetConfigPath()

	// 检查配置文件是否存在，不存在则生成默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := generateDefaultConfig(configPath); err != nil {
			return nil, fmt.Errorf("生成默认配置失败: %w", err)
		}
		consoleLogger.Info(fmt.Sprintf("已生成默认配置文件: %s", configPath))
	}

	viper.Reset()
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

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
            endpoint: http://manager:19601/api/v1/nodes/heartbeat
            interval: 10s
    online:
        centerUrl: "http://127.0.0.1:19601"
        nodeId: ""
        registrationToken: ""
    ops:
        enabled: false
        serverUrl: ""
        enrollmentCode: ""
        role: collector_linux
        heartbeatEvery: 10s
        dataDir: ./data
        demoRuntime: true
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

// runDemoWorkload 是可被三类安装包共同托管的小型长期进程，用于验证 supervisor 的
// start/stop/restart/status/log 链路；它不模拟业务计算或工业协议。
func runDemoWorkload(role string) {
	if role != "compute" && role != "alert" && role != "collector" {
		os.Exit(2)
	}
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	fmt.Printf("demo workload started role=%s pid=%d\n", role, os.Getpid())
	for {
		select {
		case <-quit:
			fmt.Printf("demo workload stopped role=%s\n", role)
			return
		case now := <-ticker.C:
			fmt.Printf("demo workload heartbeat role=%s at=%s\n", role, now.UTC().Format(time.RFC3339))
		}
	}
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
