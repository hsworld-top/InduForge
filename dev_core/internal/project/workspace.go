package project

import (
	"encoding/base64"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type FileWorkspace struct {
	root string
}

const (
	maxWorkspaceFileSize int64 = 32 << 20
	// base64 与 scene/data JSON 仍需落在 128MiB 总快照上限内。
	maxWorkspaceTotalSize int64 = 64 << 20
)

func NewFileWorkspace(root string) (*FileWorkspace, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("工程工作空间根目录不能为空")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("解析工作空间根目录失败: %w", err)
	}
	if err := os.MkdirAll(absolute, 0o755); err != nil {
		return nil, fmt.Errorf("创建工作空间根目录失败: %w", err)
	}
	return &FileWorkspace{root: absolute}, nil
}

func (w *FileWorkspace) Initialize(projectID string) (string, error) {
	projectDirectory, workspacePath, err := w.paths(projectID)
	if err != nil {
		return "", err
	}
	projectExisted := false
	if _, statErr := os.Stat(projectDirectory); statErr == nil {
		projectExisted = true
	} else if !os.IsNotExist(statErr) {
		return "", fmt.Errorf("检查工程目录失败: %w", statErr)
	}
	cleanupNewProject := func() {
		if !projectExisted {
			_ = os.RemoveAll(projectDirectory)
		}
	}
	directories := []string{
		filepath.Join(projectDirectory, "code-server-data"),
		filepath.Join(projectDirectory, "code-server-config"),
		filepath.Join(projectDirectory, "cache"),
		filepath.Join(projectDirectory, "context-state", "current"),
		filepath.Join(projectDirectory, "context-state", "staging"),
	}
	for _, directory := range directories {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			cleanupNewProject()
			return "", fmt.Errorf("创建工程工作空间失败: %w", err)
		}
	}
	if info, statErr := os.Stat(workspacePath); statErr == nil {
		if !info.IsDir() {
			return "", fmt.Errorf("工程源码路径不是目录: %s", workspacePath)
		}
		return workspacePath, nil
	} else if !os.IsNotExist(statErr) {
		return "", fmt.Errorf("检查工程工作空间失败: %w", statErr)
	}
	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		cleanupNewProject()
		return "", fmt.Errorf("创建工程源码目录失败: %w", err)
	}
	return workspacePath, nil
}

func (w *FileWorkspace) Remove(path string) error {
	resolved, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if !isWithin(w.root, resolved) {
		return fmt.Errorf("拒绝删除工作空间根目录之外的路径")
	}
	projectDirectory := resolved
	if filepath.Base(resolved) == "workspace" {
		projectDirectory = filepath.Dir(resolved)
	}
	if projectDirectory == w.root || !isWithin(w.root, projectDirectory) {
		return fmt.Errorf("拒绝删除工程根目录之外的路径")
	}
	return os.RemoveAll(projectDirectory)
}

func (w *FileWorkspace) Export(path string) (map[string]string, error) {
	resolved, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if !isWithin(w.root, resolved) {
		return nil, fmt.Errorf("工作空间路径越界")
	}
	return exportWorkspace(resolved, maxWorkspaceFileSize, maxWorkspaceTotalSize)
}

func (w *FileWorkspace) ExportAuthoring(path string) (map[string]string, map[string]uint32, error) {
	files, err := w.Export(path)
	if err != nil {
		return nil, nil, err
	}
	resolved, _ := filepath.Abs(path)
	modes := make(map[string]uint32, len(files))
	for name := range files {
		info, statErr := os.Stat(filepath.Join(resolved, filepath.FromSlash(name)))
		if statErr != nil || !info.Mode().IsRegular() {
			return nil, nil, fmt.Errorf("读取工作空间文件模式失败: %s", name)
		}
		modes[name] = uint32(info.Mode().Perm())
	}
	return files, modes, nil
}

func exportWorkspace(resolved string, maxFileSize, maxTotalSize int64) (map[string]string, error) {
	result := map[string]string{}
	var totalSize int64
	err := filepath.WalkDir(resolved, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(resolved, current)
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("工作空间包含不允许的符号链接: %s", filepath.ToSlash(relative))
		}
		if entry.IsDir() {
			if relative != "." && excludedWorkspacePath(filepath.ToSlash(relative)) {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		// 发布工件只允许普通文件，避免把设备、管道或链接指向的外部内容打包进工程。
		if !info.Mode().IsRegular() {
			return fmt.Errorf("工作空间包含非普通文件: %s", filepath.ToSlash(relative))
		}
		if info.Size() > maxFileSize {
			return fmt.Errorf("工作空间单文件超过限制: %s", filepath.ToSlash(relative))
		}
		totalSize += info.Size()
		if totalSize > maxTotalSize {
			return fmt.Errorf("工作空间总大小超过限制")
		}
		content, err := os.ReadFile(current)
		if err != nil {
			return err
		}
		result[filepath.ToSlash(relative)] = base64.StdEncoding.EncodeToString(content)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("导出工程工作空间失败: %w", err)
	}
	return result, nil
}

func (w *FileWorkspace) Import(projectID string, files map[string]string) (string, error) {
	projectDirectory, path, err := w.paths(projectID)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", err
	}
	for relative, encoded := range files {
		target := filepath.Join(path, filepath.FromSlash(relative))
		target, err = filepath.Abs(target)
		if err != nil || !isWithin(path, target) {
			_ = os.RemoveAll(projectDirectory)
			return "", fmt.Errorf("导入文件路径越界: %s", relative)
		}
		content, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			_ = os.RemoveAll(projectDirectory)
			return "", fmt.Errorf("导入文件内容无效: %s", relative)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			_ = os.RemoveAll(projectDirectory)
			return "", err
		}
		if err := os.WriteFile(target, content, 0o644); err != nil {
			_ = os.RemoveAll(projectDirectory)
			return "", err
		}
	}
	if len(files) == 0 {
		_ = os.RemoveAll(projectDirectory)
		return w.Initialize(projectID)
	}
	return path, nil
}

// StageRestore 把恢复内容写入工程目录内的持久 staging。它不触碰当前 workspace，
// 因而进程在校验或写入阶段退出也不会留下半份开发源码。
func (w *FileWorkspace) StageRestore(projectID, taskID string, files map[string]string) error {
	return w.StageRestoreWithModes(projectID, taskID, files, nil)
}

func (w *FileWorkspace) StageRestoreWithModes(projectID, taskID string, files map[string]string, modes map[string]uint32) error {
	projectDirectory, _, err := w.paths(projectID)
	if err != nil {
		return err
	}
	if _, err = uuid.Parse(taskID); err != nil {
		return fmt.Errorf("恢复任务 ID 无效")
	}
	staging := filepath.Join(projectDirectory, "restore-staging", taskID, "workspace")
	if err = os.RemoveAll(filepath.Dir(staging)); err != nil {
		return err
	}
	if err = writeEncodedWorkspace(staging, files); err != nil {
		_ = os.RemoveAll(filepath.Dir(staging))
		return err
	}
	if err = applyWorkspaceModes(staging, modes); err != nil {
		_ = os.RemoveAll(filepath.Dir(staging))
		return err
	}
	return nil
}

// ActivateRestore 原子切换当前源码目录并保留按 taskID 定位的恢复前备份。
// 调用方只有在其他领域也恢复成功后才可 FinalizeRestore。
func (w *FileWorkspace) ActivateRestore(projectID, taskID string) error {
	projectDirectory, current, err := w.paths(projectID)
	if err != nil {
		return err
	}
	if _, err = uuid.Parse(taskID); err != nil {
		return fmt.Errorf("恢复任务 ID 无效")
	}
	staging := filepath.Join(projectDirectory, "restore-staging", taskID, "workspace")
	backup := filepath.Join(projectDirectory, "restore-backups", taskID, "workspace")
	if info, statErr := os.Stat(staging); statErr != nil || !info.IsDir() {
		// finalizing 阶段重启时 staging 已经完成原子切换；保留 backup 即可证明可重入。
		if backupInfo, backupErr := os.Stat(backup); backupErr == nil && backupInfo.IsDir() {
			if currentInfo, currentErr := os.Stat(current); currentErr == nil && currentInfo.IsDir() {
				return nil
			}
		}
		return fmt.Errorf("恢复 staging 不存在")
	}
	if err = os.MkdirAll(filepath.Dir(backup), 0o755); err != nil {
		return err
	}
	if _, statErr := os.Stat(current); statErr == nil {
		if err = os.Rename(current, backup); err != nil {
			return fmt.Errorf("备份当前工作空间失败: %w", err)
		}
	} else if !os.IsNotExist(statErr) {
		return statErr
	}
	if err = os.Rename(staging, current); err != nil {
		_ = os.Rename(backup, current)
		return fmt.Errorf("激活恢复工作空间失败: %w", err)
	}
	return nil
}

func (w *FileWorkspace) RollbackRestore(projectID, taskID string) error {
	projectDirectory, current, err := w.paths(projectID)
	if err != nil {
		return err
	}
	if _, err = uuid.Parse(taskID); err != nil {
		return fmt.Errorf("恢复任务 ID 无效")
	}
	backup := filepath.Join(projectDirectory, "restore-backups", taskID, "workspace")
	if _, statErr := os.Stat(backup); os.IsNotExist(statErr) {
		return nil
	}
	failed := filepath.Join(projectDirectory, "restore-staging", taskID, "failed-workspace")
	_ = os.RemoveAll(failed)
	if err = os.MkdirAll(filepath.Dir(failed), 0o755); err != nil {
		return err
	}
	if _, statErr := os.Stat(current); statErr == nil {
		if err = os.Rename(current, failed); err != nil {
			return err
		}
	}
	if err = os.Rename(backup, current); err != nil {
		_ = os.Rename(failed, current)
		return fmt.Errorf("回滚工作空间失败: %w", err)
	}
	return os.RemoveAll(filepath.Join(projectDirectory, "restore-staging", taskID))
}

func (w *FileWorkspace) FinalizeRestore(projectID, taskID string) error {
	projectDirectory, _, err := w.paths(projectID)
	if err != nil {
		return err
	}
	if _, err = uuid.Parse(taskID); err != nil {
		return fmt.Errorf("恢复任务 ID 无效")
	}
	if err = os.RemoveAll(filepath.Join(projectDirectory, "restore-backups", taskID)); err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(projectDirectory, "restore-staging", taskID))
}

// MaterializeBuildSnapshot 为一次正式构建创建只读输入目录；构建器必须显式挂载该路径，
// 不得回退到活动 workspace。
func (w *FileWorkspace) MaterializeBuildSnapshot(projectID, versionID string, files map[string]string, modes map[string]uint32, overlays map[string][]byte) (string, func() error, error) {
	projectDirectory, _, err := w.paths(projectID)
	if err != nil {
		return "", nil, err
	}
	if _, err = uuid.Parse(versionID); err != nil {
		return "", nil, fmt.Errorf("版本 ID 无效")
	}
	root := filepath.Join(projectDirectory, "authoring-builds", versionID, "workspace")
	if err = os.RemoveAll(filepath.Dir(root)); err != nil {
		return "", nil, err
	}
	if err = writeEncodedWorkspace(root, files); err != nil {
		_ = os.RemoveAll(filepath.Dir(root))
		return "", nil, err
	}
	if err = applyWorkspaceModes(root, modes); err != nil {
		_ = os.RemoveAll(filepath.Dir(root))
		return "", nil, err
	}
	for relative, content := range overlays {
		target, pathErr := filepath.Abs(filepath.Join(root, filepath.FromSlash(relative)))
		if pathErr != nil || !isWithin(root, target) || int64(len(content)) > maxWorkspaceFileSize {
			_ = os.RemoveAll(filepath.Dir(root))
			return "", nil, fmt.Errorf("构建覆盖文件无效: %s", relative)
		}
		if err = os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			_ = os.RemoveAll(filepath.Dir(root))
			return "", nil, err
		}
		if err = os.WriteFile(target, content, 0o644); err != nil {
			_ = os.RemoveAll(filepath.Dir(root))
			return "", nil, err
		}
	}
	return root, func() error { return os.RemoveAll(filepath.Dir(root)) }, nil
}

func applyWorkspaceModes(root string, modes map[string]uint32) error {
	for name, raw := range modes {
		if _, ok := map[uint32]struct{}{0o600: {}, 0o644: {}, 0o700: {}, 0o755: {}}[raw]; !ok {
			return fmt.Errorf("工作空间文件模式无效: %s", name)
		}
		target, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil || !isWithin(root, target) {
			return fmt.Errorf("工作空间文件模式路径无效: %s", name)
		}
		if err = os.Chmod(target, os.FileMode(raw)); err != nil {
			return err
		}
	}
	return nil
}

func writeEncodedWorkspace(root string, files map[string]string) error {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	var total int64
	for relative, encoded := range files {
		target, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil || !isWithin(root, target) {
			return fmt.Errorf("恢复文件路径越界: %s", relative)
		}
		content, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil || int64(len(content)) > maxWorkspaceFileSize {
			return fmt.Errorf("恢复文件内容无效: %s", relative)
		}
		total += int64(len(content))
		if total > maxWorkspaceTotalSize {
			return fmt.Errorf("恢复工作空间总大小超过限制")
		}
		if err = os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err = os.WriteFile(target, content, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// SyncContext 将完整上下文包原子替换到工作区只读镜像目录。
// 任一写入失败都会保留旧上下文，避免代码工具读取到半成品。
func (w *FileWorkspace) SyncContext(projectID string, files map[string][]byte) error {
	projectDirectory, workspacePath, err := w.paths(projectID)
	if err != nil {
		return err
	}
	if _, err := os.Stat(workspacePath); err != nil {
		return fmt.Errorf("工程工作区不存在: %w", err)
	}

	stagingRoot := filepath.Join(projectDirectory, "context-state", "staging", newID())
	if err := os.MkdirAll(stagingRoot, 0o755); err != nil {
		return fmt.Errorf("创建上下文临时目录失败: %w", err)
	}
	defer os.RemoveAll(stagingRoot)
	for relative, content := range files {
		target := filepath.Join(stagingRoot, filepath.FromSlash(relative))
		absolute, pathErr := filepath.Abs(target)
		if pathErr != nil || !isWithin(stagingRoot, absolute) {
			return fmt.Errorf("上下文文件路径越界: %s", relative)
		}
		if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(absolute, content, 0o644); err != nil {
			return fmt.Errorf("写入上下文文件失败: %w", err)
		}
	}

	// K3s 挂载稳定的 context-state 父目录；AI 从 context/current 读取。
	// current 可替换，父目录不可替换，否则已有容器会继续读取旧 inode。
	contextRoot := filepath.Join(projectDirectory, "context-state", "current")
	if err := os.MkdirAll(filepath.Dir(contextRoot), 0o755); err != nil {
		return err
	}
	backupRoot := filepath.Join(projectDirectory, "context-state", "previous", newID())
	if err := os.MkdirAll(filepath.Dir(backupRoot), 0o755); err != nil {
		return fmt.Errorf("创建旧上下文备份目录失败: %w", err)
	}
	hasPrevious := false
	if _, err := os.Stat(contextRoot); err == nil {
		if err := os.Rename(contextRoot, backupRoot); err != nil {
			return fmt.Errorf("备份旧上下文失败: %w", err)
		}
		hasPrevious = true
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(stagingRoot, contextRoot); err != nil {
		if hasPrevious {
			_ = os.Rename(backupRoot, contextRoot)
		}
		return fmt.Errorf("替换工程上下文失败: %w", err)
	}
	if hasPrevious {
		_ = os.RemoveAll(backupRoot)
	}
	return nil
}

func (w *FileWorkspace) paths(projectID string) (string, string, error) {
	if _, err := uuid.Parse(projectID); err != nil {
		return "", "", fmt.Errorf("工程 ID 无效")
	}
	projectDirectory := filepath.Join(w.root, projectID)
	return projectDirectory, filepath.Join(projectDirectory, "workspace"), nil
}
func isWithin(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
func excludedWorkspacePath(relative string) bool {
	name := filepath.Base(filepath.FromSlash(relative))
	if name == "node_modules" || name == "dist" || name == ".git" || name == ".cache" || name == ".vite" || name == "code-server-data" || name == "code-server-config" {
		return true
	}
	return relative == ".induforge/context" || relative == ".induforge/design-imports"
}
func newID() string { return uuid.NewString() }

func projectPayload(item Project) map[string]any {
	return map[string]any{"id": item.ID, "name": item.Name, "code": item.Code, "description": item.Description, "icon": item.Icon, "status": item.Status, "visibility": item.Visibility}
}
