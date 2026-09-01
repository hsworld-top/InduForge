package ops

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const packageVersionFile = "node-agent-version.txt"

const (
	PackageLinuxAMD64 = "linux-amd64"
	PackageLinuxARM64 = "linux-arm64"
)

// FilePackageStore 仅从配置目录中的固定文件名读取，禁止把请求路径拼接进文件系统路径。
type FilePackageStore struct{ directory string }

func NewFilePackageStore(directory string) *FilePackageStore {
	return &FilePackageStore{directory: directory}
}
func (s *FilePackageStore) definitions() []NodePackage {
	return []NodePackage{
		{ID: PackageLinuxAMD64, Name: "Linux NodeAgent", Platform: PlatformLinux, Architecture: "amd64", FileName: "induforge-node-agent-linux-amd64.tar.gz"},
		{ID: PackageLinuxARM64, Name: "Linux NodeAgent", Platform: PlatformLinux, Architecture: "arm64", FileName: "induforge-node-agent-linux-arm64.tar.gz"},
		{ID: PlatformWindows, Name: "Windows NodeAgent", Platform: PlatformWindows, Architecture: "amd64", FileName: "induforge-node-agent-windows.zip"},
	}
}

func (s *FilePackageStore) List() []NodePackage {
	items := s.definitions()
	version, err := s.readVersion()
	if err != nil {
		// 没有经过同一构建批次验证的版本元数据时，不能把旧包伪装为可下载的正式包。
		return items
	}
	for i := range items {
		items[i].Version = version
		info, err := os.Lstat(filepath.Join(s.directory, items[i].FileName))
		if err == nil && info.Mode().IsRegular() {
			items[i].Available = true
			items[i].Size = info.Size()
		}
	}
	return items
}

func (s *FilePackageStore) Open(id string) (NodePackage, string, error) {
	version, err := s.readVersion()
	if err != nil {
		for _, item := range s.definitions() {
			if item.ID == id {
				return item, "", errors.New("节点安装包版本元数据缺失或无效，拒绝下载")
			}
		}
		return NodePackage{}, "", ErrNotFound
	}
	for _, item := range s.definitions() {
		if item.ID == id {
			item.Version = version
			info, statErr := os.Lstat(filepath.Join(s.directory, item.FileName))
			if statErr != nil || !info.Mode().IsRegular() {
				return item, "", errors.New("节点安装包缺失或不是普通文件，拒绝下载")
			}
			item.Available = true
			item.Size = info.Size()
			return item, filepath.Join(s.directory, item.FileName), nil
		}
	}
	return NodePackage{}, "", ErrNotFound
}

// readVersion 只接受构建脚本在两个包均成功后写入的单行版本文件。版本不合法时
// 两个平台一律不可下载，避免不同构建批次的安装包被混用。
func (s *FilePackageStore) readVersion() (string, error) {
	path := filepath.Join(s.directory, packageVersionFile)
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() < 2 || info.Size() > 129 {
		return "", errors.New("节点安装包版本文件无效")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	text := string(content)
	if strings.Count(text, "\n") != 1 || !strings.HasSuffix(text, "\n") || strings.ContainsRune(text, '\r') {
		return "", errors.New("节点安装包版本必须为单行")
	}
	version := strings.TrimSuffix(text, "\n")
	if !validPackageVersion(version) {
		return "", errors.New("节点安装包版本格式无效")
	}
	return version, nil
}

func validPackageVersion(version string) bool {
	if len(version) == 0 || len(version) > 128 {
		return false
	}
	if !isPackageVersionLetterOrDigit(version[0]) {
		return false
	}
	for index := 1; index < len(version); index++ {
		value := version[index]
		if isPackageVersionLetterOrDigit(value) || value == '.' || value == '_' || value == '+' || value == '-' {
			continue
		}
		return false
	}
	return true
}

func isPackageVersionLetterOrDigit(value byte) bool {
	return (value >= 'a' && value <= 'z') || (value >= 'A' && value <= 'Z') || (value >= '0' && value <= '9')
}
