package executor

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/indu-forge/node_agent/internal/pkg/logger"
	"github.com/indu-forge/node_agent/internal/pkg/types"
	pkgUtils "github.com/indu-forge/node_agent/internal/pkg/utils"
)

// ProcessExecutor 进程执行器
type ProcessExecutor struct {
	workDir   string
	logDir    string
	binary    string
	logger    *logger.SimpleLogger
	processes map[string]*os.Process
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

	if strings.TrimSpace(req.IFPPackage) != "" {
		if err := e.deployFromIFP(req); err != nil {
			return err
		}
	}

	// 检查 binary 文件
	if _, _, err := e.resolveBinary(req.ProjectID); err != nil {
		return err
	}

	return nil
}

// Start 启动
func (e *ProcessExecutor) Start(ctx context.Context, projectID string) error {
	e.logger.Info("启动进程", "project", projectID)

	binaryPath, runDir, err := e.resolveBinary(projectID)
	if err != nil {
		return err
	}
	logFile := filepath.Join(e.logDir, fmt.Sprintf("%s.log", projectID))

	cmd := exec.CommandContext(ctx, binaryPath)
	cmd.Dir = runDir

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

// deployFromIFP 从 IFP 包解压并切换当前运行版本。
func (e *ProcessExecutor) deployFromIFP(req types.DeployRequest) error {
	if strings.TrimSpace(req.Version) == "" {
		return fmt.Errorf("版本号不能为空")
	}
	if _, err := os.Stat(req.IFPPackage); err != nil {
		return fmt.Errorf("IFP 包不存在: %w", err)
	}

	projectDir := filepath.Join(e.workDir, req.ProjectID)
	versionDir := filepath.Join(projectDir, "versions", req.Version)
	stageDir := versionDir + ".staging"
	currentDir := filepath.Join(projectDir, "current")

	if err := os.MkdirAll(filepath.Dir(versionDir), 0755); err != nil {
		return fmt.Errorf("创建版本目录失败: %w", err)
	}
	_ = os.RemoveAll(stageDir)
	if err := os.MkdirAll(stageDir, 0755); err != nil {
		return fmt.Errorf("创建暂存目录失败: %w", err)
	}

	if err := extractZipSecure(req.IFPPackage, stageDir); err != nil {
		_ = os.RemoveAll(stageDir)
		return fmt.Errorf("解压 IFP 包失败: %w", err)
	}

	_ = os.RemoveAll(versionDir)
	if err := os.Rename(stageDir, versionDir); err != nil {
		_ = os.RemoveAll(stageDir)
		return fmt.Errorf("落地版本目录失败: %w", err)
	}

	if err := switchCurrentDir(versionDir, currentDir); err != nil {
		return fmt.Errorf("切换 current 失败: %w", err)
	}

	return nil
}

// resolveBinary 解析可执行文件路径，兼容 IFP 内嵌目录结构。
func (e *ProcessExecutor) resolveBinary(projectID string) (string, string, error) {
	currentDir := filepath.Join(e.workDir, projectID, "current")
	searchDir := currentDir
	if resolvedDir, err := filepath.EvalSymlinks(currentDir); err == nil && strings.TrimSpace(resolvedDir) != "" {
		searchDir = resolvedDir
	}

	binaryPath := filepath.Join(searchDir, e.binary)
	if info, err := os.Stat(binaryPath); err == nil && !info.IsDir() {
		return binaryPath, searchDir, nil
	}

	var matchedPath string
	err := filepath.WalkDir(searchDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(d.Name(), e.binary) {
			matchedPath = path
			return io.EOF
		}
		return nil
	})
	if err != nil && err != io.EOF {
		return "", "", fmt.Errorf("遍历可执行文件失败: %w", err)
	}
	if matchedPath == "" {
		return "", "", fmt.Errorf("binary 文件不存在: %s", filepath.Join(currentDir, e.binary))
	}

	return matchedPath, filepath.Dir(matchedPath), nil
}

// switchCurrentDir 优先软链接切换，失败时降级为复制目录，确保 Windows 可用。
func switchCurrentDir(versionDir string, currentDir string) error {
	if err := pkgUtils.Symlink(versionDir, currentDir); err == nil {
		return nil
	}

	if err := os.RemoveAll(currentDir); err != nil {
		return err
	}
	return copyDir(versionDir, currentDir)
}

// extractZipSecure 安全解压 zip，防止 Zip Slip。
func extractZipSecure(archivePath string, targetDir string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	cleanTarget := filepath.Clean(targetDir)
	targetPrefix := cleanTarget + string(os.PathSeparator)
	for _, file := range reader.File {
		destination := filepath.Join(cleanTarget, file.Name)
		cleanDest := filepath.Clean(destination)
		if cleanDest != cleanTarget && !strings.HasPrefix(cleanDest, targetPrefix) {
			return fmt.Errorf("非法压缩路径: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(cleanDest, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(cleanDest), 0755); err != nil {
			return err
		}
		srcFile, err := file.Open()
		if err != nil {
			return err
		}

		perm := file.Mode()
		if perm == 0 {
			perm = 0644
		}
		dstFile, err := os.OpenFile(cleanDest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
		if err != nil {
			srcFile.Close()
			return err
		}
		if _, err := io.Copy(dstFile, srcFile); err != nil {
			_ = dstFile.Close()
			_ = srcFile.Close()
			return err
		}
		_ = dstFile.Close()
		_ = srcFile.Close()
	}

	return nil
}

// copyDir 递归复制目录。
func copyDir(src string, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		return err
	})
}
