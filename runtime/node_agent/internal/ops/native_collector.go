package ops

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type nativeCollectorBundle struct {
	SchemaVersion string            `json:"schemaVersion"`
	NodeID        string            `json:"nodeId"`
	DeploymentID  string            `json:"deploymentId"`
	ServiceID     string            `json:"serviceId"`
	ReleaseID     string            `json:"releaseId"`
	Revision      int               `json:"revision"`
	Binding       json.RawMessage   `json:"binding"`
	Index         json.RawMessage   `json:"index"`
	SecretFiles   map[string][]byte `json:"secretFiles"`
}

func (a *Agent) reconcileNativeCollector(ctx context.Context, command AgentCommand) error {
	command.Operation = strings.ToLower(strings.TrimSpace(command.Operation))
	command.DesiredStatus = strings.ToLower(strings.TrimSpace(command.DesiredStatus))
	// 在任何进程操作之前验证命令，避免未知操作被当成停止，或矛盾的停止命令启动新进程。
	if command.DesiredStatus != "running" && command.DesiredStatus != "stopped" {
		return fmt.Errorf("不支持的采集 desiredStatus: %s", command.DesiredStatus)
	}
	switch command.Operation {
	case "":
	case "start", "deploy", "restart":
		if command.DesiredStatus != "running" {
			return fmt.Errorf("采集启动命令与期望状态不一致")
		}
	case "stop", "delete":
		if command.DesiredStatus != "stopped" {
			return fmt.Errorf("采集停止命令与期望状态不一致")
		}
	default:
		return fmt.Errorf("不支持的采集 operation: %s", command.Operation)
	}
	status, _ := a.supervisor.Status(command.ServiceID)
	if command.Generation < a.appliedGeneration(command.ServiceID) || command.Generation < status.Generation {
		return nil
	}
	if !commandRequiresRunningRelease(command) {
		// 停止、WAL 清理与完成记录共用进程操作锁，避免新代次在清理前启动。
		// Supervisor 可能已持久化更新代次而 applied 仍落后，过期命令不能清理或记为完成。
		a.supervisor.opMu.Lock()
		defer a.supervisor.opMu.Unlock()
		stopped, err := a.supervisor.stop(command.ServiceID, command.Generation)
		if err != nil {
			return err
		}
		if stopped.Generation > command.Generation {
			return nil
		}
		if stopped.State != "stopped" || stopped.Generation != command.Generation {
			return fmt.Errorf("采集进程尚未确认停止，保留采集数据")
		}
		if command.Operation == "delete" {
			if err := removeCollectorWAL(a.cfg.DataDir, command.DeploymentID); err != nil {
				return err
			}
		}
		return a.saveApplied(command.ServiceID, command.Generation)
	}
	if status.Generation == command.Generation && matchesDesired(status, command) {
		return a.saveApplied(command.ServiceID, command.Generation)
	}
	if err := a.installBoundRelease(ctx, command, ServiceCollector); err != nil {
		return err
	}
	identity := a.currentIdentity()
	path := "/api/v1/ops/agent/nodes/" + identity.NodeID + "/deployments/" + command.DeploymentID + "/collector?serviceId=" + command.ServiceID
	var raw json.RawMessage
	if err := a.requestLimited(ctx, http.MethodGet, path, identity.AgentToken, nil, &raw, 16<<20); err != nil {
		return fmt.Errorf("获取原生采集配置失败")
	}
	var bundle nativeCollectorBundle
	if err := strictDecodeJSON(raw, &bundle); err != nil {
		return fmt.Errorf("原生采集配置格式无效")
	}
	if bundle.SchemaVersion != "native-collector-bundle.v1" || bundle.NodeID != identity.NodeID || bundle.DeploymentID != command.DeploymentID || bundle.ServiceID != command.ServiceID || bundle.ReleaseID != command.ReleaseID || bundle.Revision != command.BindingRevision {
		return fmt.Errorf("原生采集配置与当前命令不匹配")
	}
	service, err := a.prepareNativeCollector(command, bundle)
	if err != nil {
		return err
	}
	if _, err = a.supervisor.ActivateWorkload(command.ServiceID, ServiceCollector, command.Generation, []ServiceConfig{service}, command.Operation == "restart"); err != nil {
		return err
	}
	return a.saveApplied(command.ServiceID, command.Generation)
}

// prepareNativeCollector 的路径、二进制和参数均由本机派生，中心只提供身份与配置。
// 采集 artifact 从已验签的 Release 快照提取，不能以新的中心数据覆盖冻结制品。
func (a *Agent) prepareNativeCollector(command AgentCommand, bundle nativeCollectorBundle) (ServiceConfig, error) {
	root := filepath.Join(a.cfg.DataDir, "deployments", command.DeploymentID)
	root, err := filepath.Abs(root)
	if err != nil {
		return ServiceConfig{}, err
	}
	wal, err := prepareNativeCollectorWAL(a.cfg.DataDir, command.DeploymentID)
	if err != nil {
		return ServiceConfig{}, err
	}
	releaseRoot := filepath.Join(root, "release")
	current := filepath.Join("releases", "sha256-"+strings.TrimPrefix(command.ArchiveSHA256, "sha256:"))
	manifestRaw, err := os.ReadFile(filepath.Join(releaseRoot, current, "release-manifest.json"))
	if err != nil {
		return ServiceConfig{}, err
	}
	if digestBytes(manifestRaw) != command.ManifestSHA256 {
		return ServiceConfig{}, fmt.Errorf("采集 Release manifest 摘要不匹配")
	}
	var manifest releaseManifestForVerification
	if strictDecodeJSON(manifestRaw, &manifest) != nil || manifest.Artifacts.Collector == nil {
		return ServiceConfig{}, fmt.Errorf("已安装 Release 缺少采集快照")
	}
	if err := verifyCollectorSourceSnapshot(*manifest.Artifacts.Collector, manifest.ProjectID); err != nil {
		return ServiceConfig{}, err
	}
	var snapshot collectorSourceSnapshotForVerification
	if strictDecodeJSON(manifest.Artifacts.Collector.SourceSnapshot, &snapshot) != nil {
		return ServiceConfig{}, fmt.Errorf("采集快照无效")
	}
	artifact, err := canonicalJSONBytes(snapshot.Artifact)
	if err != nil {
		return ServiceConfig{}, err
	}
	if err = validateNativeCollectorFiles(bundle, command, snapshot.SHA256); err != nil {
		return ServiceConfig{}, err
	}
	raw, _ := json.Marshal(bundle)
	configRoot := filepath.Join(root, "collector-config", strings.TrimPrefix(digestBytes(raw), "sha256:"))
	if err := writeNativeCollectorFiles(configRoot, artifact, bundle); err != nil {
		return ServiceConfig{}, err
	}
	capabilityRoot, executable, err := nativeCollectorCapability(a.cfg.InstallRoot, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return ServiceConfig{}, err
	}
	healthURL := ""
	a.supervisor.mu.Lock()
	if previous := a.supervisor.workloads[command.ServiceID]; len(previous) == 1 {
		healthURL = previous[0].HealthURL
	}
	a.supervisor.mu.Unlock()
	if healthURL == "" {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return ServiceConfig{}, err
		}
		healthURL = "http://" + listener.Addr().String() + "/health"
		_ = listener.Close()
	}
	production := a.cfg.SecurityMode != "development"
	environment, err := nativeCollectorEnvironment(root)
	if err != nil {
		return ServiceConfig{}, err
	}
	return ServiceConfig{Group: ServiceCollector, Component: "collector", Installed: true, Enabled: true, ReleaseRoot: releaseRoot, Current: current, ReleaseDigest: command.ManifestSHA256, CapabilityRoot: capabilityRoot, Executable: executable, Environment: environment, Arguments: []string{"--artifact", filepath.Join(configRoot, "artifact.json"), "--binding", filepath.Join(configRoot, "binding.json"), "--index", filepath.Join(configRoot, "index.json"), "--wal", wal, "--listen", strings.TrimSuffix(healthURL, "health"), "--site-id", command.NodeID, "--production", strconv.FormatBool(production)}, HealthURL: healthURL, HealthTimeout: 20 * time.Second, DrainTimeout: 25 * time.Second}, nil
}

// 单文件 .NET 包需要解压原生依赖。服务账户的 HOME/安装目录通常只读，
// 因此只使用本部署的数据目录，不能依赖登录用户缓存或修改不可变的 Release。
func nativeCollectorEnvironment(root string) (map[string]string, error) {
	cache := filepath.Join(root, "state", "dotnet-bundle")
	if err := os.MkdirAll(cache, 0700); err != nil {
		return nil, fmt.Errorf("创建采集依赖缓存失败")
	}
	info, err := os.Lstat(cache)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("采集依赖缓存目录不安全")
	}
	if err := os.Chmod(cache, 0700); err != nil {
		return nil, fmt.Errorf("设置采集依赖缓存权限失败")
	}
	return map[string]string{"DOTNET_BUNDLE_EXTRACT_BASE_DIR": cache}, nil
}

func validateNativeCollectorFiles(bundle nativeCollectorBundle, command AgentCommand, artifactDigest string) error {
	var binding struct {
		DeploymentID    string `json:"deploymentId"`
		BindingRevision int    `json:"bindingRevision"`
		Artifact        struct {
			Digest string `json:"artifactDigest"`
		} `json:"artifact"`
	}
	if json.Unmarshal(bundle.Binding, &binding) != nil || binding.DeploymentID != command.DeploymentID || binding.BindingRevision != command.BindingRevision || binding.Artifact.Digest != artifactDigest {
		return fmt.Errorf("采集运行绑定与冻结制品不匹配")
	}
	var index struct {
		SchemaVersion string                     `json:"schemaVersion"`
		Resources     map[string]json.RawMessage `json:"resources"`
		Secrets       map[string]string          `json:"secrets"`
	}
	if strictDecodeJSON(bundle.Index, &index) != nil || index.SchemaVersion != "collector-runtime-index.v1" {
		return fmt.Errorf("采集资源索引无效")
	}
	for _, path := range index.Secrets {
		name := strings.TrimPrefix(path, "secrets/")
		if path != "secrets/"+name || !nativeCollectorSecretName(name) || len(bundle.SecretFiles[name]) == 0 {
			return fmt.Errorf("采集凭据引用无效")
		}
	}
	if len(bundle.SecretFiles["nats.json"]) == 0 {
		return fmt.Errorf("采集消息凭据缺失")
	}
	for name, content := range bundle.SecretFiles {
		if !nativeCollectorSecretName(name) || len(content) > 1<<20 || !json.Valid(content) {
			return fmt.Errorf("采集凭据文件无效")
		}
	}
	return nil
}

func nativeCollectorSecretName(name string) bool {
	if name == "nats.json" {
		return true
	}
	return strings.HasPrefix(name, "connection-") && strings.HasSuffix(name, ".json") && validRuntimeUUID(strings.TrimSuffix(strings.TrimPrefix(name, "connection-"), ".json"))
}

func nativeCollectorCapability(installRoot, platform, arch string) (string, string, error) {
	if installRoot == "" {
		return "", "", fmt.Errorf("采集安装根目录未配置")
	}
	if platform == "linux" && (arch == "amd64" || arch == "arm64") {
		return filepath.Join(installRoot, "capabilities", arch, "collector"), "industrial_collector", nil
	}
	if platform == "windows" && arch == "amd64" {
		return filepath.Join(installRoot, "capabilities", "collector", "win-x64"), "industrial_collector.exe", nil
	}
	return "", "", fmt.Errorf("原生采集不支持本机平台 %s/%s", platform, arch)
}

func writeNativeCollectorFiles(target string, artifact []byte, bundle nativeCollectorBundle) error {
	files := map[string][]byte{"artifact.json": artifact, "binding.json": bundle.Binding, "index.json": bundle.Index}
	for name, content := range bundle.SecretFiles {
		if !nativeCollectorSecretName(name) {
			return fmt.Errorf("采集凭据文件名无效")
		}
		files[filepath.Join("secrets", name)] = content
	}
	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0700); err != nil {
		return err
	}
	if info, err := os.Lstat(target); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("采集配置目录无效")
		}
		// 相同摘要只可复用相同文件；拒绝被本机修改的候选目录，不覆盖运行中的配置。
		for name, expected := range files {
			path := filepath.Join(target, name)
			info, err := os.Lstat(path)
			if err != nil || !info.Mode().IsRegular() || info.Size() != int64(len(expected)) {
				return fmt.Errorf("已保存采集配置无效")
			}
			actual, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(actual, expected) {
				return fmt.Errorf("已保存采集配置摘要不匹配")
			}
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	staging, err := os.MkdirTemp(parent, ".preparing-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	if err = os.Mkdir(filepath.Join(staging, "secrets"), 0700); err != nil {
		return err
	}
	for name, content := range files {
		if err = os.WriteFile(filepath.Join(staging, name), content, 0400); err != nil {
			return fmt.Errorf("写入采集配置失败")
		}
	}
	return os.Rename(staging, target)
}
