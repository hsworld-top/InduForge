package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/indu-forge/node_agent/internal/pkg/types"
	"github.com/indu-forge/node_agent/internal/pkg/logger"
)

// ProcessExecutor 进程执行器
type ProcessExecutor struct {
	workDir    string
	logDir     string
	binary     string
	logger     *logger.SimpleLogger
	processes  map[string]*os.Process
}

// NewProcessExecutor 创建进程执行器
func NewProcessExecutor(workDir, logDir, binary string) *ProcessExecutor {
	return &ProcessExecutor{
		workDir:   workDir,
		logDir:    logDir,
		binary:    binary,
		logger:    logger.GlobalLogger,
		processes: make(map[string]*os.Process),
	}
}

// Deploy 部署
func (e *ProcessExecutor) Deploy(ctx context.Context, req types.DeployRequest) error {
	e.logger.Info("进程执行器部署", "project", req.ProjectID, "version", req.Version)

	// 检查 binary 文件
	binaryPath := filepath.Join(e.workDir, req.ProjectID, "current", e.binary)
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("binary 文件不存在: %s", binaryPath)
	}

	return nil
}

// Start 启动
func (e *ProcessExecutor) Start(ctx context.Context, projectID string) error {
	e.logger.Info("启动进程", "project", projectID)

	binaryPath := filepath.Join(e.workDir, projectID, "current", e.binary)
	logFile := filepath.Join(e.logDir, fmt.Sprintf("%s.log", projectID))

	cmd := exec.CommandContext(ctx, binaryPath)
	cmd.Dir = filepath.Join(e.workDir, projectID, "current")

	// 输出到日志文件
	logF, err := os.Create(logFile)
	if err != nil {
		return fmt.Errorf("创建日志文件失败: %w", err)
	}
	defer logF.Close()

	cmd.Stdout = logF
	cmd.Stderr = logF

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动进程失败: %w", err)
	}

	e.processes[projectID] = cmd.Process
	e.logger.Info("进程启动成功", "project", projectID, "pid", cmd.Process.Pid)

	return nil
}

// Stop 停止
func (e *ProcessExecutor) Stop(ctx context.Context, projectID string) error {
	e.logger.Info("停止进程", "project", projectID)

	proc, ok := e.processes[projectID]
	if !ok {
		return fmt.Errorf("进程不存在")
	}

	if err := proc.Signal(os.Interrupt); err != nil {
		e.logger.Warn("发送停止信号失败", "error", err)
		// 强制杀死
		if err := proc.Kill(); err != nil {
			return fmt.Errorf("杀死进程失败: %w", err)
		}
	}

	delete(e.processes, projectID)
	return nil
}

// Restart 重启
func (e *ProcessExecutor) Restart(ctx context.Context, projectID string) error {
	if err := e.Stop(ctx, projectID); err != nil {
		return err
	}
	time.Sleep(1 * time.Second)
	return e.Start(ctx, projectID)
}

// Rollback 回滚
func (e *ProcessExecutor) Rollback(ctx context.Context, projectID string, version string) error {
	e.logger.Info("回滚版本", "project", projectID, "version", version)

	// 停止当前版本
	if err := e.Stop(ctx, projectID); err != nil {
		return err
	}

	// 切换版本（由 orchestrator 调用）
	return nil
}

// GetStatus 获取状态
func (e *ProcessExecutor) GetStatus(ctx context.Context, projectID string) (*types.RuntimeStatus, error) {
	proc, ok := e.processes[projectID]
	if !ok {
		return &types.RuntimeStatus{
			ProjectID: projectID,
			State:     "stopped",
			Health:    "unhealthy",
		}, nil
	}

	// 检查进程是否存活
	if err := proc.Signal(os.Signal(nil)); err != nil {
		return &types.RuntimeStatus{
			ProjectID: projectID,
			State:     "stopped",
			Health:    "unhealthy",
		}, nil
	}

	now := time.Now()
	return &types.RuntimeStatus{
		ProjectID: projectID,
		State:     "running",
		PID:       proc.Pid,
		Health:    "healthy",
		LastCheck: now,
	}, nil
}

// HealthCheck 健康检查
func (e *ProcessExecutor) HealthCheck(ctx context.Context) error {
	// 检查 binary 是否存在
	if e.binary == "" {
		return fmt.Errorf("binary 未配置")
	}
	return nil
}
