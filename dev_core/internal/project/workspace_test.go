package project

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWorkspaceExportRegularFiles(t *testing.T) {
	root := t.TempDir()
	projectPath := filepath.Join(root, "project")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, "app.js"), []byte("console.log('ok')"), 0o644); err != nil {
		t.Fatal(err)
	}
	workspace, err := NewFileWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	files, err := workspace.Export(projectPath)
	if err != nil {
		t.Fatal(err)
	}
	content, err := base64.StdEncoding.DecodeString(files["app.js"])
	if err != nil || string(content) != "console.log('ok')" {
		t.Fatalf("导出内容不正确: %q, err=%v", content, err)
	}
}

func TestWorkspaceInitializeCreatesFinalDirectoryLayout(t *testing.T) {
	root := t.TempDir()
	workspace, err := NewFileWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	projectID := "11111111-1111-4111-8111-111111111111"
	workspacePath, err := workspace.Initialize(projectID)
	if err != nil {
		t.Fatal(err)
	}
	if workspacePath != filepath.Join(root, projectID, "workspace") {
		t.Fatalf("工作空间路径不正确: %s", workspacePath)
	}
	for _, relative := range []string{
		"workspace",
		"code-server-data",
		"code-server-config",
		"cache",
		"context-state/current",
		"context-state/staging",
	} {
		if info, statErr := os.Stat(filepath.Join(root, projectID, filepath.FromSlash(relative))); statErr != nil || !info.IsDir() {
			t.Fatalf("缺少工程目录 %s: %v", relative, statErr)
		}
	}
	entries, err := os.ReadDir(workspacePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("首次选择模板前源码目录必须为空: %v", entries)
	}
}

func TestWorkspaceRestoreCanRollbackAndFinalize(t *testing.T) {
	root := t.TempDir()
	projectID := "11111111-1111-4111-8111-111111111111"
	taskID := "22222222-2222-4222-8222-222222222222"
	workspace, err := NewFileWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	current, err := workspace.Initialize(projectID)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(current, "version.txt"), []byte("current"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err = workspace.StageRestore(projectID, taskID, map[string]string{"version.txt": base64.StdEncoding.EncodeToString([]byte("restored"))}); err != nil {
		t.Fatal(err)
	}
	if content, _ := os.ReadFile(filepath.Join(current, "version.txt")); string(content) != "current" {
		t.Fatal("staging 不得改变当前工作空间")
	}
	if err = workspace.ActivateRestore(projectID, taskID); err != nil {
		t.Fatal(err)
	}
	if content, _ := os.ReadFile(filepath.Join(current, "version.txt")); string(content) != "restored" {
		t.Fatalf("未激活恢复内容: %q", content)
	}
	if err = workspace.RollbackRestore(projectID, taskID); err != nil {
		t.Fatal(err)
	}
	if content, _ := os.ReadFile(filepath.Join(current, "version.txt")); string(content) != "current" {
		t.Fatalf("未恢复备份: %q", content)
	}

	if err = workspace.StageRestore(projectID, taskID, map[string]string{"version.txt": base64.StdEncoding.EncodeToString([]byte("final"))}); err != nil {
		t.Fatal(err)
	}
	if err = workspace.ActivateRestore(projectID, taskID); err != nil {
		t.Fatal(err)
	}
	if err = workspace.FinalizeRestore(projectID, taskID); err != nil {
		t.Fatal(err)
	}
	if content, _ := os.ReadFile(filepath.Join(current, "version.txt")); string(content) != "final" {
		t.Fatalf("最终内容错误: %q", content)
	}
}

func TestWorkspaceSyncContextSwapsRealContextStateDirectory(t *testing.T) {
	root := t.TempDir()
	projectID := "11111111-1111-4111-8111-111111111111"
	workspace, err := NewFileWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := workspace.Initialize(projectID); err != nil {
		t.Fatal(err)
	}
	current := filepath.Join(root, projectID, "context-state", "current")
	if err := os.WriteFile(filepath.Join(current, "previous.json"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := workspace.SyncContext(projectID, map[string][]byte{"summary.json": []byte("new")}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(current, "summary.json"))
	if err != nil || string(content) != "new" {
		t.Fatalf("上下文未落入真实源目录: %q, err=%v", content, err)
	}
	if _, err := os.Stat(filepath.Join(current, "previous.json")); !os.IsNotExist(err) {
		t.Fatalf("旧上下文未被原子替换: %v", err)
	}
}

func TestWorkspaceInitializeDoesNotOverwriteExistingFiles(t *testing.T) {
	root := t.TempDir()
	projectID := "11111111-1111-4111-8111-111111111111"
	workspacePath := filepath.Join(root, projectID, "workspace")
	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		t.Fatal(err)
	}
	customFile := filepath.Join(workspacePath, "package.json")
	if err := os.WriteFile(customFile, []byte("{\"name\":\"customer-project\"}"), 0o644); err != nil {
		t.Fatal(err)
	}
	workspace, err := NewFileWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := workspace.Initialize(projectID); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(customFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "{\"name\":\"customer-project\"}" {
		t.Fatalf("已有客户文件被覆盖: %s", content)
	}
}

func TestWorkspaceInitializeKeepsExistingWorkspaceUntouched(t *testing.T) {
	root := t.TempDir()
	projectID := "11111111-1111-4111-8111-111111111111"
	workspacePath := filepath.Join(root, projectID, "workspace")
	if err := os.MkdirAll(filepath.Join(workspacePath, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	customFile := filepath.Join(workspacePath, "README.md")
	if err := os.WriteFile(customFile, []byte("customer content"), 0o644); err != nil {
		t.Fatal(err)
	}
	workspace, err := NewFileWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := workspace.Initialize(projectID); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(customFile)
	if err != nil || string(content) != "customer content" {
		t.Fatalf("已有文件被覆盖: content=%q err=%v", content, err)
	}
}

func TestWorkspaceExportRejectsFileSymlink(t *testing.T) {
	root := t.TempDir()
	projectPath := filepath.Join(root, "project")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "secret.txt")
	if err := os.WriteFile(target, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(projectPath, "link.txt")); err != nil {
		t.Skipf("当前环境无法创建符号链接: %v", err)
	}
	workspace, _ := NewFileWorkspace(root)
	files, err := workspace.Export(projectPath)
	if err == nil || !strings.Contains(err.Error(), "符号链接") {
		t.Fatalf("文件符号链接应被拒绝: files=%v err=%v", files, err)
	}
	if _, exists := files["link.txt"]; exists {
		t.Fatal("符号链接目标内容不应出现在导出结果中")
	}
}

func TestWorkspaceExportRejectsDirectorySymlink(t *testing.T) {
	root := t.TempDir()
	projectPath := filepath.Join(root, "project")
	target := filepath.Join(root, "outside")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(projectPath, "linked-dir")); err != nil {
		t.Skipf("当前环境无法创建目录符号链接: %v", err)
	}
	workspace, _ := NewFileWorkspace(root)
	if _, err := workspace.Export(projectPath); err == nil || !strings.Contains(err.Error(), "符号链接") {
		t.Fatalf("目录符号链接应被拒绝: %v", err)
	}
}

func TestWorkspaceExportRejectsOversizedFile(t *testing.T) {
	root := t.TempDir()
	projectPath := filepath.Join(root, "project")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(projectPath, "large.bin")
	if err := os.WriteFile(file, []byte("12345"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := exportWorkspace(projectPath, 4, 16); err == nil || !strings.Contains(err.Error(), "单文件") {
		t.Fatalf("超大文件应被拒绝: %v", err)
	}
}

func TestWorkspaceExportRejectsOversizedTotal(t *testing.T) {
	root := t.TempDir()
	projectPath := filepath.Join(root, "project")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, "a.bin"), []byte("1234"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, "b.bin"), []byte("5678"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := exportWorkspace(projectPath, 4, 7); err == nil || !strings.Contains(err.Error(), "总大小") {
		t.Fatalf("累计超限应被拒绝: %v", err)
	}
}

func TestWorkspaceExportExcludesGeneratedDirectories(t *testing.T) {
	root := t.TempDir()
	projectPath := filepath.Join(root, "project")
	excluded := []string{"node_modules", "dist", ".git", ".vite", "code-server-data", "code-server-config", filepath.Join(".induforge", "context"), filepath.Join(".induforge", "design-imports")}
	for _, directory := range excluded {
		path := filepath.Join(projectPath, directory)
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "ignored.txt"), []byte("ignored"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(projectPath, "kept.txt"), []byte("kept"), 0o644); err != nil {
		t.Fatal(err)
	}
	workspace, _ := NewFileWorkspace(root)
	files, err := workspace.Export(projectPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files["kept.txt"] == "" {
		t.Fatalf("排除目录未生效: %v", files)
	}
}

func TestWorkspaceLimitsMatchProductContract(t *testing.T) {
	if maxWorkspaceFileSize != 32<<20 || maxWorkspaceTotalSize != 64<<20 {
		t.Fatalf("工作空间限制不符合产品约定: file=%d total=%d", maxWorkspaceFileSize, maxWorkspaceTotalSize)
	}
	if runtime.GOOS == "windows" {
		t.Log("Windows 特殊文件创建能力有限，Linux CI 继续覆盖符号链接路径")
	}
}
