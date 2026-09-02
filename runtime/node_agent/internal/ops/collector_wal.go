package ops

import (
	"fmt"
	"os"
	"path/filepath"
)

// collectorContainerUID/GID 对应 collector_engine Dockerfile 的 distroless nonroot。
// 节点 Agent 由受管宿主以特权身份运行，目录由 Agent 创建而非让 Kubelet 猜测 owner。
const (
	collectorContainerUID = 65532
	collectorContainerGID = 65532
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

// ensureCollectorWAL 在 Release 安装成功后幂等准备宿主持久目录。目录不存在或离线
// 节点没有完成物化时，Kubernetes hostPath Directory 会拒绝启动新 Pod，而不会悄悄用
// 错误 owner 创建空目录。
func ensureCollectorWAL(dataRoot, deploymentID string) (string, error) {
	return ensureCollectorWALWithChown(dataRoot, deploymentID, os.Chown)
}

func ensureCollectorWALWithChown(dataRoot, deploymentID string, chown func(string, int, int) error) (string, error) {
	path, err := collectorWALPath(dataRoot, deploymentID)
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(path, 0o770); err != nil {
		return "", fmt.Errorf("创建 collector WAL 目录失败")
	}
	for _, current := range []string{filepath.Join(filepath.Dir(filepath.Dir(path))), filepath.Dir(path), path} {
		info, statErr := os.Lstat(current)
		if statErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("collector WAL 目录不安全")
		}
	}
	if err = os.Chmod(path, 0o770); err != nil {
		return "", fmt.Errorf("设置 collector WAL 权限失败")
	}
	if chown == nil {
		return "", fmt.Errorf("设置 collector WAL owner 失败")
	}
	if err = chown(path, collectorContainerUID, collectorContainerGID); err != nil {
		// 正常 Linux 节点的 Agent 以 induforge 服务账户运行，不能自行 chown。
		// 该路径刚由自身创建，工作负载模板会把固定的服务组 1000 加入 collector
		// Pod；保留 owner 并使用 0770，比放宽到 world-writable 更安全。
		if os.Geteuid() == 0 {
			return "", fmt.Errorf("设置 collector WAL owner 失败")
		}
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
