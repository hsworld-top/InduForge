package ops

import (
	"fmt"
	"os"
	"path/filepath"
)

// collectorWALPath 只从节点数据根和已验证 deployment UUID 派生路径，中心不能提供
// 任意宿主路径。
func collectorWALPath(dataRoot, deploymentID string) (string, error) {
	if !validRuntimeUUID(deploymentID) {
		return "", fmt.Errorf("deploymentId 无效")
	}
	root, err := filepath.Abs(dataRoot)
	if err != nil || root == "" {
		return "", fmt.Errorf("节点数据目录无效")
	}
	path := filepath.Join(root, "deployments", deploymentID, "state", "collector-wal")
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == "." || filepath.IsAbs(relative) || relative == ".." || len(relative) >= 3 && relative[:3] == ".."+string(filepath.Separator) {
		return "", fmt.Errorf("collector WAL 路径越界")
	}
	return path, nil
}

// 原生采集沿用节点服务账户，以私有目录保存 WAL。逐层验证工程状态目录，
// 避免父目录软链把待确认数据写到工程之外；不再设置容器 UID/GID。
func prepareNativeCollectorWAL(dataRoot, deploymentID string) (string, error) {
	path, err := collectorWALPath(dataRoot, deploymentID)
	if err != nil {
		return "", err
	}
	deploymentRoot := filepath.Dir(filepath.Dir(path))
	if err = os.MkdirAll(filepath.Dir(deploymentRoot), 0700); err != nil {
		return "", err
	}
	for _, current := range []string{deploymentRoot, filepath.Dir(path), path} {
		if err = os.Mkdir(current, 0700); err != nil && !os.IsExist(err) {
			return "", fmt.Errorf("创建采集 WAL 目录失败")
		}
		if info, err := os.Lstat(current); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("采集 WAL 目录无效")
		}
	}
	if err = os.Chmod(path, 0700); err != nil {
		return "", err
	}
	return path, nil
}

// removeCollectorWAL 只在显式删除 deployment 时调用；stop/restart 永不清理 WAL。
func removeCollectorWAL(dataRoot, deploymentID string) error {
	path, err := collectorWALPath(dataRoot, deploymentID)
	if err != nil {
		return err
	}
	if info, statErr := os.Lstat(path); statErr != nil {
		if os.IsNotExist(statErr) {
			return nil
		}
		return fmt.Errorf("读取 collector WAL 目录失败")
	} else if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("collector WAL 目录不安全")
	}
	if err = os.RemoveAll(path); err != nil {
		return fmt.Errorf("清理 collector WAL 目录失败")
	}
	return nil
}
