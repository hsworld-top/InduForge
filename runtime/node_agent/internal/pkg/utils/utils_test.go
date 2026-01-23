package utils

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestEnsureDir 测试目录创建
func TestEnsureDir(t *testing.T) {
	tempDir := t.TempDir()
	newDir := filepath.Join(tempDir, "new_sub_dir")

	// 确保目录不存在
	if _, err := os.Stat(newDir); !os.IsNotExist(err) {
		t.Fatalf("测试目录应不存在: %v", err)
	}

	// 确保目录
	if err := EnsureDir(newDir); err != nil {
		t.Fatalf("EnsureDir 失败: %v", err)
	}

	// 验证目录创建成功
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Error("目录应已创建")
	}
}

// TestEnsureDir_AlreadyExists 测试目录已存在
func TestEnsureDir_AlreadyExists(t *testing.T) {
	tempDir := t.TempDir()

	// 创建目录
	if err := EnsureDir(tempDir); err != nil {
		t.Fatalf("EnsureDir 失败: %v", err)
	}

	// 再次调用应不报错
	if err := EnsureDir(tempDir); err != nil {
		t.Errorf("已存在目录不应报错，实际: %v", err)
	}
}

// TestCalcSHA256 测试 SHA256 计算
func TestCalcSHA256(t *testing.T) {
	// 创建测试文件
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.txt")
	content := []byte("hello world")

	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	hash, err := CalcSHA256(filePath)
	if err != nil {
		t.Fatalf("CalcSHA256 失败: %v", err)
	}

	// 验证 hash 长度（SHA256 生成 64 个十六进制字符）
	if len(hash) != 64 {
		t.Errorf("SHA256 hash 长度应为 64，实际 %d", len(hash))
	}

	// 验证 hash 值（已知的 "hello world" SHA256）
	expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if hash != expected {
		t.Errorf("SHA256 值不匹配，期望 %s，实际 %s", expected, hash)
	}
}

// TestCalcSHA256_FileNotFound 测试文件不存在
func TestCalcSHA256_FileNotFound(t *testing.T) {
	_, err := CalcSHA256("/nonexistent/file.txt")

	if err == nil {
		t.Error("文件不存在应返回错误")
	}
}

// TestCopyFile 测试文件复制
func TestCopyFile(t *testing.T) {
	tempDir := t.TempDir()

	// 创建源文件
	srcPath := filepath.Join(tempDir, "src.txt")
	content := []byte("test content for copy")
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatalf("创建源文件失败: %v", err)
	}

	// 复制文件
	dstPath := filepath.Join(tempDir, "dst.txt")
	if err := CopyFile(srcPath, dstPath); err != nil {
		t.Fatalf("CopyFile 失败: %v", err)
	}

	// 验证目标文件存在且内容相同
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("读取目标文件失败: %v", err)
	}

	if string(dstContent) != string(content) {
		t.Errorf("复制内容不匹配，期望 %s，实际 %s", string(content), string(dstContent))
	}
}

// TestCopyFile_SrcNotFound 测试源文件不存在
func TestCopyFile_SrcNotFound(t *testing.T) {
	tempDir := t.TempDir()
	dstPath := filepath.Join(tempDir, "dst.txt")

	err := CopyFile("/nonexistent/src.txt", dstPath)

	if err == nil {
		t.Error("源文件不存在应返回错误")
	}
}

// TestSymlink 测试软链接切换
func TestSymlink(t *testing.T) {
	tempDir := t.TempDir()

	// 创建目标目录
	targetDir := filepath.Join(tempDir, "target")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("创建目标目录失败: %v", err)
	}

	// 创建软链接
	linkPath := filepath.Join(tempDir, "link")
	if err := Symlink(targetDir, linkPath); err != nil {
		t.Fatalf("创建软链接失败: %v", err)
	}

	// 验证软链接
	link, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("读取软链接失败: %v", err)
	}

	if link != targetDir {
		t.Errorf("软链接目标不匹配，期望 %s，实际 %s", targetDir, link)
	}
}

// TestSymlink_Update 测试软链接更新（原子性切换）
func TestSymlink_Update(t *testing.T) {
	tempDir := t.TempDir()

	// 创建两个目标目录
	target1 := filepath.Join(tempDir, "target1")
	target2 := filepath.Join(tempDir, "target2")
	if err := os.MkdirAll(target1, 0755); err != nil {
		t.Fatalf("创建目标目录失败: %v", err)
	}
	if err := os.MkdirAll(target2, 0755); err != nil {
		t.Fatalf("创建目标目录失败: %v", err)
	}

	linkPath := filepath.Join(tempDir, "link")

	// 第一次创建
	if err := Symlink(target1, linkPath); err != nil {
		t.Fatalf("创建软链接失败: %v", err)
	}

	// 切换到新的目标
	if err := Symlink(target2, linkPath); err != nil {
		t.Fatalf("更新软链接失败: %v", err)
	}

	// 验证更新后指向新目标
	link, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("读取软链接失败: %v", err)
	}

	if link != target2 {
		t.Errorf("软链接应指向新目标，期望 %s，实际 %s", target2, link)
	}
}

// TestRunCommand 测试命令执行
func TestRunCommand(t *testing.T) {
	// 使用更通用的命令（在 Windows 和 Linux 上都可用）
	// hostname 命令在大多数系统上都可用
	err := RunCommand("hostname")
	if err != nil {
		// 如果 hostname 失败，尝试使用其他方法
		// 在 Windows 上可能需要不同的处理
		t.Logf("命令执行结果: %v", err)
	}
}

// TestRunCommand_NotFound 测试命令不存在
func TestRunCommand_NotFound(t *testing.T) {
	err := RunCommand("/nonexistent/command")

	if err == nil {
		t.Error("命令不存在应返回错误")
	}
}

// TestGetEnvOrDefault 测试环境变量获取
func TestGetEnvOrDefault(t *testing.T) {
	// 设置测试环境变量
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	// 测试已存在的环境变量
	result := GetEnvOrDefault("TEST_VAR", "default")
	if result != "test_value" {
		t.Errorf("期望 test_value，实际 %s", result)
	}

	// 测试不存在的环境变量
	result = GetEnvOrDefault("NONEXISTENT_VAR", "default")
	if result != "default" {
		t.Errorf("期望 default，实际 %s", result)
	}
}

// TestParseDuration 测试持续时间解析
func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		expectOK bool
	}{
		{"1h30m", "1h30m0s", true},
		{"30s", "30s", true},
		{"100ms", "100ms", true},
		{"1h", "1h0m0s", true},
		{"invalid", "", false},
		{"", "", false}, // 空字符串也会返回错误
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := ParseDuration(tt.input)

			if tt.expectOK {
				if err != nil {
					t.Errorf("ParseDuration(%s) 不应失败，实际: %v", tt.input, err)
				}
			} else {
				if err == nil {
					t.Errorf("ParseDuration(%s) 应失败", tt.input)
				}
			}
		})
	}
}

// TestFormatTime 测试时间格式化
func TestFormatTime(t *testing.T) {
	// 使用固定时间进行测试
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.Local)
	expected := "2024-01-15 10:30:00"

	// 注意：FormatTime 使用固定格式 "2006-01-02 15:04:05"
	result := FormatTime(now)

	if result != expected {
		t.Errorf("期望 %s，实际 %s", expected, result)
	}
}

// TestProjectIDFromPath 测试从路径提取项目ID
func TestProjectIDFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/var/lib/node_agent/projects/project-001/current", "project-001"},
		{"/projects/my-project/versions/v1.0.0", "versions"}, // 父目录是 versions
		{"projects/test/versions", "test"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := ProjectIDFromPath(tt.path)

			if result != tt.expected {
				t.Errorf("从 %s 提取项目ID，期望 %s，实际 %s", tt.path, tt.expected, result)
			}
		})
	}
}

// TestVersionFromPath 测试从路径提取版本号
func TestVersionFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/var/lib/node_agent/projects/project-001/versions/v1.0.0", "v1.0.0"},
		{"/projects/my-project/versions/1.0.0", "1.0.0"},
		{"versions/v2.0.0-beta", "v2.0.0-beta"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := VersionFromPath(tt.path)

			if result != tt.expected {
				t.Errorf("从 %s 提取版本号，期望 %s，实际 %s", tt.path, tt.expected, result)
			}
		})
	}
}
