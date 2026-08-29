package ops

import (
	"errors"
	"os"
	"path/filepath"
)

// FilePackageStore 仅从配置目录中的固定文件名读取，禁止把请求路径拼接进文件系统路径。
type FilePackageStore struct{ directory string }

func NewFilePackageStore(directory string) *FilePackageStore {
	return &FilePackageStore{directory: directory}
}
func (s *FilePackageStore) definitions() []NodePackage {
	return []NodePackage{
		{ID: RoleRuntimeLinux, Name: "Linux 运行节点代理", Role: RoleRuntimeLinux, OS: "linux", Platform: "linux", Architecture: "multi", Version: "demo-v1", FileName: "induforge-node-runtime-linux.tar.gz"},
		{ID: RoleCollectorLinux, Name: "Linux 采集节点代理", Role: RoleCollectorLinux, OS: "linux", Platform: "linux", Architecture: "multi", Version: "demo-v1", FileName: "induforge-node-collector-linux.tar.gz"},
		{ID: RoleCollectorWindows, Name: "Windows 采集节点代理", Role: RoleCollectorWindows, OS: "windows", Platform: "windows", Architecture: "amd64", Version: "demo-v1", FileName: "induforge-node-collector-windows.zip"},
	}
}
func (s *FilePackageStore) List() []NodePackage {
	items := s.definitions()
	for i := range items {
		info, err := os.Stat(filepath.Join(s.directory, items[i].FileName))
		if err == nil && !info.IsDir() {
			items[i].Available = true
			items[i].Size = info.Size()
		}
	}
	return items
}
func (s *FilePackageStore) Open(id string) (NodePackage, string, error) {
	for _, item := range s.List() {
		if item.ID == id {
			if !item.Available {
				return item, "", errors.New("节点安装包尚未生成，请先构建对应安装包")
			}
			return item, filepath.Join(s.directory, item.FileName), nil
		}
	}
	return NodePackage{}, "", ErrNotFound
}
