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
	maxWorkspaceFileSize  int64 = 32 << 20
	maxWorkspaceTotalSize int64 = 512 << 20
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

	contextRoot := filepath.Join(workspacePath, ".induforge", "context")
	if err := os.MkdirAll(filepath.Dir(contextRoot), 0o755); err != nil {
		return err
	}
	backupRoot := filepath.Join(projectDirectory, "context-state", "current", newID())
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
