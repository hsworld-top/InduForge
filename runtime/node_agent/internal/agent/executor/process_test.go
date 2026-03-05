package executor

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/indu-forge/node_agent/internal/pkg/types"
)

// TestProcessExecutorDeployFromIFP_ResolveBinary 验证 IFP 解包后可正确定位可执行文件。
func TestProcessExecutorDeployFromIFP_ResolveBinary(t *testing.T) {
	tempDir := t.TempDir()
	workDir := filepath.Join(tempDir, "runtime")
	logDir := filepath.Join(tempDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("创建日志目录失败: %v", err)
	}

	ifpPath := filepath.Join(tempDir, "demo.ifp")
	entries := map[string]string{
		"bundle/runtime_engine": "#!/bin/sh\necho ok\n",
		"manifest.json":         `{"projectId":"demo","version":"v1.0.0"}`,
	}
	if err := writeZipFile(ifpPath, entries); err != nil {
		t.Fatalf("创建测试 IFP 失败: %v", err)
	}

	exec := NewProcessExecutor(workDir, logDir, "runtime_engine")
	req := types.DeployRequest{
		ProjectID:  "demo",
		Version:    "v1.0.0",
		IFPPackage: ifpPath,
	}
	if err := exec.Deploy(context.Background(), req); err != nil {
		t.Fatalf("部署失败: %v", err)
	}

	binaryPath, runDir, err := exec.resolveBinary("demo")
	if err != nil {
		t.Fatalf("解析 binary 失败: %v", err)
	}
	if filepath.Base(binaryPath) != "runtime_engine" {
		t.Fatalf("binary 名称不正确: %s", binaryPath)
	}
	normalizedRunDir := filepath.ToSlash(runDir)
	if !strings.Contains(normalizedRunDir, "/demo/") {
		t.Fatalf("运行目录不正确: %s", runDir)
	}
}

// TestProcessExecutorDeployFromIFP_MissingBinary 验证 IFP 缺少 binary 时返回错误。
func TestProcessExecutorDeployFromIFP_MissingBinary(t *testing.T) {
	tempDir := t.TempDir()
	workDir := filepath.Join(tempDir, "runtime")
	logDir := filepath.Join(tempDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("创建日志目录失败: %v", err)
	}

	ifpPath := filepath.Join(tempDir, "broken.ifp")
	if err := writeZipFile(ifpPath, map[string]string{
		"README.txt": "missing binary",
	}); err != nil {
		t.Fatalf("创建测试 IFP 失败: %v", err)
	}

	exec := NewProcessExecutor(workDir, logDir, "runtime_engine")
	err := exec.Deploy(context.Background(), types.DeployRequest{
		ProjectID:  "demo",
		Version:    "v1.0.0",
		IFPPackage: ifpPath,
	})
	if err == nil {
		t.Fatalf("预期部署失败，但返回成功")
	}
	if !strings.Contains(err.Error(), "binary 文件不存在") {
		t.Fatalf("错误信息不符合预期: %v", err)
	}
}

func writeZipFile(target string, files map[string]string) error {
	file, err := os.Create(target)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			_ = writer.Close()
			return err
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			_ = writer.Close()
			return err
		}
	}
	return writer.Close()
}
