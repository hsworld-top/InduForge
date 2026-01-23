package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// EnsureDir 确保目录存在
func EnsureDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// CalcSHA256 计算文件 SHA256
func CalcSHA256(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// CopyFile 复制文件
func CopyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// Symlink 原子性软链接切换
func Symlink(oldname, newname string) error {
	// 删除旧链接
	if _, err := os.Stat(newname); err == nil {
		if err := os.Remove(newname); err != nil {
			return err
		}
	}
	// 创建新链接
	return os.Symlink(oldname, newname)
}

// RunCommand 执行命令
func RunCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// GetEnvOrDefault 获取环境变量或默认值
func GetEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// ParseDuration 解析持续时间
func ParseDuration(d string) (time.Duration, error) {
	return time.ParseDuration(d)
}

// FormatTime 格式化时间
func FormatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// ProjectIDFromPath 从路径提取项目ID
func ProjectIDFromPath(path string) string {
	return filepath.Base(filepath.Dir(path))
}

// VersionFromPath 从路径提取版本号
func VersionFromPath(path string) string {
	return filepath.Base(path)
}
