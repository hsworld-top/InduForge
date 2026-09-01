package hostd

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type CommandRunner interface {
	Run(context.Context, string, ...string) ([]byte, error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

type ManagerConfig struct {
	AssetsDir        string
	BinaryPath       string
	ConfigDir        string
	SystemdDir       string
	StateDir         string
	Systemctl        string
	Nsenter          string
	Runner           CommandRunner
	RuntimeArch      string
	ChronyConfig     string
	ChronyDropIn     string
	Chronyc          string
	NetworkPreflight func(context.Context, ClusterPlan) error
}

type Manager struct {
	cfg ManagerConfig
	mu  sync.Mutex
}

func DefaultManagerConfig() ManagerConfig {
	return ManagerConfig{
		AssetsDir:        "/opt/induforge/node-agent/k3s",
		BinaryPath:       "/usr/local/bin/k3s",
		ConfigDir:        "/etc/rancher/induforge-k3s",
		SystemdDir:       "/etc/systemd/system",
		StateDir:         "/var/lib/induforge/hostd",
		Systemctl:        "systemctl",
		Nsenter:          "/usr/bin/nsenter",
		Runner:           execRunner{},
		RuntimeArch:      runtime.GOARCH,
		ChronyConfig:     "/etc/chrony/chrony.conf",
		ChronyDropIn:     "/etc/systemd/system/chrony.service.d/induforge.conf",
		Chronyc:          "/usr/bin/chronyc",
		NetworkPreflight: networkPreflight,
	}
}

func NewManager(config ManagerConfig) (*Manager, error) {
	if config.Runner == nil {
		config.Runner = execRunner{}
	}
	if config.Nsenter == "" {
		config.Nsenter = "/usr/bin/nsenter"
	}
	if config.ChronyConfig == "" {
		config.ChronyConfig = "/etc/chrony/chrony.conf"
	}
	if config.ChronyDropIn == "" {
		config.ChronyDropIn = "/etc/systemd/system/chrony.service.d/induforge.conf"
	}
	if config.Chronyc == "" {
		config.Chronyc = "/usr/bin/chronyc"
	}
	if config.RuntimeArch != "amd64" && config.RuntimeArch != "arm64" {
		return nil, fmt.Errorf("不支持的 K3s 架构: %s", config.RuntimeArch)
	}
	for name, value := range map[string]string{
		"assetsDir": config.AssetsDir, "binaryPath": config.BinaryPath, "configDir": config.ConfigDir,
		"systemdDir": config.SystemdDir, "stateDir": config.StateDir, "systemctl": config.Systemctl, "nsenter": config.Nsenter,
	} {
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("%s 不能为空", name)
		}
	}
	return &Manager{cfg: config}, nil
}

func (m *Manager) statePath() string { return filepath.Join(m.cfg.StateDir, "cluster-state.json") }

func (m *Manager) killallPath() string { return filepath.Join(m.cfg.ConfigDir, "killall.sh") }

func (m *Manager) Apply(ctx context.Context, plan ClusterPlan) (ClusterState, error) {
	if err := plan.Validate(); err != nil {
		return ClusterState{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	current, exists, err := m.loadState()
	if err != nil {
		return ClusterState{}, err
	}
	if exists && current.ObservedState != "not-installed" {
		if current.ClusterID != plan.ClusterID || current.NodeID != plan.NodeID || current.Operation != plan.Operation {
			return ClusterState{}, errors.New("节点已初始化到其他运行集群或角色；必须先显式卸载 K3s")
		}
		if plan.Generation < current.Generation {
			return ClusterState{}, errors.New("拒绝过期的 K3s 集群计划")
		}
		if plan.Generation == current.Generation {
			return m.observe(ctx, current), nil
		}
	}
	// 仅在首次安装或明确卸载后重装时检查本机监听端口；运行中的 K3s
	// 本身会占用这些端口，升级 generation 时重复检查会产生误报。
	if (!exists || current.ObservedState == "not-installed") && m.cfg.NetworkPreflight != nil {
		if err := m.cfg.NetworkPreflight(ctx, plan); err != nil {
			return ClusterState{}, err
		}
	}

	assetBinary := filepath.Join(m.cfg.AssetsDir, m.cfg.RuntimeArch, "k3s")
	assetImages := filepath.Join(m.cfg.AssetsDir, m.cfg.RuntimeArch, "k3s-airgap-images-"+m.cfg.RuntimeArch+".tar.zst")
	foundationImages := filepath.Join(m.cfg.AssetsDir, m.cfg.RuntimeArch, "induforge-foundation-images-"+m.cfg.RuntimeArch+".tar.gz")
	checksums := filepath.Join(m.cfg.AssetsDir, m.cfg.RuntimeArch, "SHA256SUMS")
	if err := verifyAssets(checksums, assetBinary, assetImages, foundationImages); err != nil {
		return ClusterState{}, err
	}
	if err := ensureRealDirectory(plan.DataDir, 0700); err != nil {
		return ClusterState{}, err
	}
	if err := ensureRealDirectory(m.cfg.ConfigDir, 0700); err != nil {
		return ClusterState{}, err
	}
	if err := ensureRealDirectory(m.cfg.StateDir, 0700); err != nil {
		return ClusterState{}, err
	}
	if err := copyAtomic(assetBinary, m.cfg.BinaryPath, 0755); err != nil {
		return ClusterState{}, fmt.Errorf("安装 K3s 二进制失败: %w", err)
	}
	imageDir := filepath.Join(plan.DataDir, "agent", "images")
	if err := ensureRealDirectory(imageDir, 0700); err != nil {
		return ClusterState{}, err
	}
	if err := copyAtomic(assetImages, filepath.Join(imageDir, filepath.Base(assetImages)), 0600); err != nil {
		return ClusterState{}, fmt.Errorf("安装 K3s 离线镜像失败: %w", err)
	}
	if err := copyAtomic(foundationImages, filepath.Join(imageDir, filepath.Base(foundationImages)), 0600); err != nil {
		return ClusterState{}, fmt.Errorf("安装基础服务离线镜像失败: %w", err)
	}
	if err := writeAtomic(filepath.Join(m.cfg.ConfigDir, "config.yaml"), []byte(plan.RenderK3sConfig()), 0600); err != nil {
		return ClusterState{}, fmt.Errorf("写入 K3s 配置失败: %w", err)
	}
	if err := writeAtomic(m.killallPath(), []byte(renderKillallScript(plan.DataDir)), 0700); err != nil {
		return ClusterState{}, fmt.Errorf("写入 K3s 清理脚本失败: %w", err)
	}
	unitPath := filepath.Join(m.cfg.SystemdDir, plan.ServiceName())
	if err := writeAtomic(unitPath, []byte(m.renderSystemdUnit(plan)), 0644); err != nil {
		return ClusterState{}, fmt.Errorf("写入 K3s systemd 单元失败: %w", err)
	}
	if output, err := m.cfg.Runner.Run(ctx, m.cfg.Systemctl, "daemon-reload"); err != nil {
		return ClusterState{}, commandError("刷新 systemd", output, err)
	}
	if output, err := m.cfg.Runner.Run(ctx, m.cfg.Systemctl, "enable", "--now", plan.ServiceName()); err != nil {
		return ClusterState{}, commandError("启动 K3s", output, err)
	}
	state := ClusterState{
		SchemaVersion: "induforge.cluster-state.v1", Generation: plan.Generation,
		ClusterID: plan.ClusterID, NodeID: plan.NodeID, Operation: plan.Operation,
		K3sVersion: plan.K3sVersion, NodeName: plan.NodeName(), NodeIP: plan.NodeIP,
		DataDir: plan.DataDir, ServiceName: plan.ServiceName(), ObservedState: "starting",
	}
	if err := m.saveState(state); err != nil {
		return ClusterState{}, err
	}
	return m.observe(ctx, state), nil
}

func (m *Manager) Status(ctx context.Context) (ClusterState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, exists, err := m.loadState()
	if err != nil {
		return ClusterState{}, err
	}
	if !exists {
		return ClusterState{SchemaVersion: "induforge.cluster-state.v1", ObservedState: "not-installed"}, nil
	}
	if state.ObservedState == "not-installed" {
		return state, nil
	}
	return m.observe(ctx, state), nil
}

func (m *Manager) Uninstall(ctx context.Context, request UninstallRequest) (ClusterState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, exists, err := m.loadState()
	if err != nil {
		return ClusterState{}, err
	}
	if !exists {
		return ClusterState{SchemaVersion: "induforge.cluster-state.v1", ObservedState: "not-installed"}, nil
	}
	if request.ClusterID != state.ClusterID || request.NodeID != state.NodeID {
		return ClusterState{}, errors.New("卸载请求与本机 K3s 身份不匹配")
	}
	if state.ObservedState == "not-installed" {
		return state, nil
	}
	if err := validateDataDir(state.DataDir); err != nil {
		return ClusterState{}, err
	}
	dataInfo, err := os.Lstat(state.DataDir)
	if err != nil && !os.IsNotExist(err) {
		return ClusterState{}, err
	}
	if err == nil && dataInfo.Mode()&os.ModeSymlink != 0 {
		return ClusterState{}, errors.New("拒绝清理符号链接 K3s 数据目录")
	}
	// 早期版本尚未落盘 killall 脚本。节点包原地升级后必须仍能完整卸载，
	// 因此依据已校验的历史状态补齐脚本，而不是要求用户先重新部署集群。
	if _, err := os.Stat(m.killallPath()); os.IsNotExist(err) {
		if err := writeAtomic(m.killallPath(), []byte(renderKillallScript(state.DataDir)), 0700); err != nil {
			return ClusterState{}, err
		}
	} else if err != nil {
		return ClusterState{}, err
	}
	if output, err := m.cfg.Runner.Run(ctx, m.cfg.Systemctl, "disable", "--now", state.ServiceName); err != nil {
		// 上一次卸载可能已删掉 unit 后在数据目录清理阶段中断。unit 已不存在时
		// 继续执行 killall 和数据清理，使卸载天然可重试。
		if _, statErr := os.Stat(filepath.Join(m.cfg.SystemdDir, state.ServiceName)); !os.IsNotExist(statErr) {
			return ClusterState{}, commandError("停止 K3s", output, err)
		}
	}
	// Hostd 的 ProtectSystem 会创建私有 mount namespace；直接执行 umount 只会
	// 影响该私有视图。进入 PID 1 的宿主 mount namespace 后再执行固定脚本，
	// 才能真正清理 kubelet、containerd 与本地卷挂载。
	if output, err := m.cfg.Runner.Run(ctx, m.cfg.Nsenter, "--mount=/proc/1/ns/mnt", "--", m.killallPath()); err != nil {
		return ClusterState{}, commandError("清理 K3s 宿主资源", output, err)
	}
	// 集群数据、容器状态和本地 PVC 不能在“已卸载”后残留，否则同 generation
	// 重装只会观察到旧状态。保留数据目录本身：systemd ReadWritePaths 会把它挂入
	// Hostd 的私有命名空间，删除根目录会让后续重装落到只读的失效挂载点。
	if err := clearDirectory(state.DataDir); err != nil {
		return ClusterState{}, fmt.Errorf("清空 K3s 数据目录: %w", err)
	}
	for _, path := range []string{
		filepath.Join(m.cfg.SystemdDir, state.ServiceName),
		filepath.Join(m.cfg.ConfigDir, "config.yaml"),
		m.killallPath(),
		m.cfg.BinaryPath,
	} {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return ClusterState{}, err
		}
	}
	if output, err := m.cfg.Runner.Run(ctx, m.cfg.Systemctl, "daemon-reload"); err != nil {
		return ClusterState{}, commandError("刷新 systemd", output, err)
	}
	// 保留不含宿主路径的卸载墓碑，使重装后 Agent 能向 Center 明确回报本地
	// generation 已清零；Center 随后会重新下发原环境计划。下一次 Apply 会覆盖它。
	tombstone := ClusterState{
		SchemaVersion: "induforge.cluster-state.v1", ClusterID: state.ClusterID,
		NodeID: state.NodeID, Operation: state.Operation, K3sVersion: state.K3sVersion,
		NodeName: state.NodeName, NodeIP: state.NodeIP, ObservedState: "not-installed",
	}
	if err := m.saveState(tombstone); err != nil {
		return ClusterState{}, err
	}
	return tombstone, nil
}

func clearDirectory(root string) error {
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return os.MkdirAll(root, 0700)
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(root, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) observe(ctx context.Context, state ClusterState) ClusterState {
	output, err := m.cfg.Runner.Run(ctx, m.cfg.Systemctl, "is-active", state.ServiceName)
	if err != nil || strings.TrimSpace(string(output)) != "active" {
		state.ObservedState = "failed"
		state.Message = sanitizedCommandMessage(output, err)
		return state
	}
	// systemd active 只表示进程尚未退出。K3s 与首节点完成 TLS 注册后才会生成
	// kubelet 客户端证书；以此区分“正在重连”和真正可承载工作负载。
	if _, err := os.Stat(filepath.Join(state.DataDir, "agent", "client-kubelet.crt")); err != nil {
		if os.IsNotExist(err) {
			state.ObservedState = "starting"
			state.Message = "正在完成运行环境节点注册"
			return state
		}
		state.ObservedState = "failed"
		state.Message = "无法检查节点注册状态"
		return state
	}
	state.ObservedState = "ready"
	state.Message = ""
	return state
}

func (m *Manager) renderSystemdUnit(plan ClusterPlan) string {
	subcommand := "agent"
	if plan.Operation == OperationInitServer {
		subcommand = "server"
	}
	return `[Unit]
Description=InduForge managed K3s
After=network-online.target
Wants=network-online.target

[Service]
Type=notify
ExecStart=` + m.cfg.BinaryPath + ` ` + subcommand + ` --config ` + filepath.Join(m.cfg.ConfigDir, "config.yaml") + `
KillMode=process
Delegate=yes
LimitNOFILE=1048576
LimitNPROC=infinity
LimitCORE=infinity
TasksMax=infinity
Restart=always
RestartSec=5s
ExecStopPost=` + m.killallPath() + `

[Install]
WantedBy=multi-user.target
`
}

// renderKillallScript 采用 K3s 官方安装器 killall 脚本的清理边界，并把 kubelet
// 挂载路径适配到节点安装时记录的数据目录。脚本只由 root Hostd 生成和执行。
func renderKillallScript(dataDir string) string {
	return `#!/bin/sh
set +e
K3S_DATA_DIR='` + dataDir + `'
for bin in "$K3S_DATA_DIR"/data/*/bin; do
  [ -d "$bin" ] && PATH="$PATH:$bin:$bin/aux"
done
pschildren() { ps -e -o ppid= -o pid= | sed -e 's/^\s*//g; s/\s\s*/\t/g' | grep -w "^$1" | cut -f2; }
pstree() { for pid in "$@"; do echo "$pid"; for child in $(pschildren "$pid"); do pstree "$child"; done; done; }
killtree() { pids="$(pstree "$@")"; [ -z "$pids" ] || kill -9 $pids 2>/dev/null; }
getshims() {
  ps -e -o pid= -o args= | sed -e 's/^ *//; s/\s\s*/\t/' |
    grep 'containerd-shim' | grep -- '-address /run/k3s/containerd/containerd.sock' | cut -f1
}
killtree $(getshims)
unmount_prefix() {
  while read -r _ path _; do case "$path" in "$1"*) echo "$path";; esac; done < /proc/self/mounts |
    sort -r | xargs -r -n 1 sh -c 'umount -f -l "$0" 2>/dev/null; rm -rf "$0"'
}
unmount_prefix /run/k3s
unmount_prefix "$K3S_DATA_DIR"
unmount_prefix /var/lib/kubelet/pods
unmount_prefix /var/lib/kubelet/plugins
unmount_prefix /run/netns/cni-
ip netns show 2>/dev/null | grep cni- | xargs -r -n 1 ip netns delete
ip link show 2>/dev/null | grep 'master cni0' | while read -r _ iface _; do iface="${iface%%@*}"; [ -z "$iface" ] || ip link delete "$iface"; done
for iface in cni0 flannel.1 flannel-v6.1 kube-ipvs0 flannel-wg flannel-wg-v6; do ip link delete "$iface" 2>/dev/null; done
rm -rf /var/lib/cni
if command -v iptables-save >/dev/null 2>&1; then iptables-save | grep -v KUBE- | grep -v CNI- | grep -iv flannel | iptables-restore; fi
if command -v ip6tables-save >/dev/null 2>&1; then ip6tables-save | grep -v KUBE- | grep -v CNI- | grep -iv flannel | ip6tables-restore; fi
exit 0
`
}

func (m *Manager) loadState() (ClusterState, bool, error) {
	data, err := os.ReadFile(m.statePath())
	if os.IsNotExist(err) {
		return ClusterState{}, false, nil
	}
	if err != nil {
		return ClusterState{}, false, err
	}
	var state ClusterState
	if err := json.Unmarshal(data, &state); err != nil {
		return ClusterState{}, false, fmt.Errorf("读取 K3s 状态失败: %w", err)
	}
	return state, true, nil
}

func (m *Manager) saveState(state ClusterState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return writeAtomic(m.statePath(), data, 0600)
}

func verifyAssets(checksumPath string, paths ...string) error {
	file, err := os.Open(checksumPath)
	if err != nil {
		return fmt.Errorf("读取 K3s SHA256SUMS 失败: %w", err)
	}
	defer file.Close()
	wanted := make(map[string]string, len(paths))
	for _, path := range paths {
		wanted[filepath.Base(path)] = ""
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) == 2 {
			if _, ok := wanted[strings.TrimPrefix(parts[1], "*")]; ok {
				wanted[strings.TrimPrefix(parts[1], "*")] = strings.ToLower(parts[0])
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	for _, path := range paths {
		expected := wanted[filepath.Base(path)]
		if expected == "" {
			return fmt.Errorf("K3s 资产缺少校验和: %s", filepath.Base(path))
		}
		actual, err := sha256File(path)
		if err != nil {
			return err
		}
		if actual != expected {
			return fmt.Errorf("K3s 资产校验失败: %s", filepath.Base(path))
		}
	}
	return nil
}

func sha256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func ensureRealDirectory(path string, mode os.FileMode) error {
	if err := os.MkdirAll(path, mode); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("拒绝使用符号链接目录: %s", path)
	}
	return os.Chmod(path, mode)
}

func copyAtomic(source, destination string, mode os.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(destination), ".induforge-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if _, err := io.Copy(temporary, input); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Chmod(mode); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, destination)
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".induforge-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Chmod(mode); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

func commandError(action string, output []byte, err error) error {
	return fmt.Errorf("%s失败: %s", action, sanitizedCommandMessage(output, err))
}

func sanitizedCommandMessage(output []byte, err error) string {
	message := strings.TrimSpace(string(output))
	if len(message) > 512 {
		message = message[:512]
	}
	if message == "" && err != nil {
		message = err.Error()
	}
	return message
}
