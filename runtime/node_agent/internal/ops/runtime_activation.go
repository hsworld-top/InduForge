package ops

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

const bindingDigestRawCenterJSON = "raw-center-json.v1"

// RuntimeActivationDerivationInput 只接收已经由 Agent 严格校验过的 Binding 原始
// JSON、已安装 Release 和节点本地 Foundation/Secret 描述。没有 Activation 的网络
// 输入，避免中心把命令、环境变量或任意路径间接带入本机启动配置。
type RuntimeActivationDerivationInput struct {
	BindingJSON      []byte
	InstalledRelease InstalledRelease
	SiteID           string
	AccountID        string
	Foundation       RuntimeFoundationPlan
	Secrets          []RuntimeSecretReference
	ExpiresAt        string
	Now              time.Time
}

// DerivedRuntimeActivation 记录本地派生结果。BindingDigestMode 明确当前没有引入
// RFC8785/JCS 库：摘要的规范化边界是 Center 返回且 Agent 已严格解析的原始 JSON
// 字节。不得把它误称为 JCS；Center 改变空白或字段顺序会得到不同摘要并触发新意图。
type DerivedRuntimeActivation struct {
	Activation        RuntimeActivationFoundationInput
	BindingDigestMode string
	ReleaseVersion    string
}

// DeriveRuntimeActivation 重新解析 Binding 原始字节并和已封存 Release 交叉校验，
// 然后由节点本地事实构造 runtime-activation.v1。它从不接受 Center Activation。
func DeriveRuntimeActivation(input RuntimeActivationDerivationInput) (DerivedRuntimeActivation, error) {
	if len(input.BindingJSON) == 0 {
		return DerivedRuntimeActivation{}, errors.New("DeploymentBinding 原始 JSON 不能为空")
	}
	var binding deploymentBinding
	if err := strictDecodeJSON(input.BindingJSON, &binding); err != nil {
		return DerivedRuntimeActivation{}, fmt.Errorf("DeploymentBinding 原始 JSON 无效: %w", err)
	}
	now := input.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if err := validateActivationBinding(binding, now); err != nil {
		return DerivedRuntimeActivation{}, err
	}
	manifest, checksums, err := verifyInstalledActivationRelease(input.InstalledRelease, binding)
	if err != nil {
		return DerivedRuntimeActivation{}, err
	}
	if !validRuntimeUUID(input.SiteID) || !validRuntimeUUID(input.AccountID) {
		return DerivedRuntimeActivation{}, errors.New("本地 siteId 或 accountId 无效")
	}
	if _, err := parseRuntimeActivationTime(input.ExpiresAt, true, now); err != nil {
		return DerivedRuntimeActivation{}, fmt.Errorf("本地 activation expiresAt 无效: %w", err)
	}

	services, _ := validatedBindingServices(binding.EnabledServices, binding.Ports.CollectorHealthLoopback)
	activation := RuntimeActivationFoundationInput{
		SchemaVersion: runtimeActivationSchema,
		ActivationID:  derivedActivationID(input.BindingJSON, input.InstalledRelease.ReleaseDigest),
		Revision:      binding.Revision,
		NodeID:        binding.NodeID,
		DeploymentID:  binding.DeploymentID,
		ProjectID:     binding.ProjectID,
		ReleaseID:     binding.Release.ID,
		SiteID:        input.SiteID,
		AccountID:     input.AccountID,
		Binding: RuntimeActivationBinding{
			BindingID: binding.BindingID, Revision: binding.Revision, BindingSHA256: digestBytes(input.BindingJSON),
		},
		Ports: RuntimeActivationPorts{
			GatewayPublic: binding.Ports.GatewayPublic, RuntimeAPILoopback: binding.Ports.RuntimeAPILoopback,
			EngineLoopback: binding.Ports.EngineLoopback, CollectorHealthLoopback: binding.Ports.CollectorHealthLoopback,
		},
		Foundation: input.Foundation,
		Secrets:    append([]RuntimeSecretReference(nil), input.Secrets...),
		Artifacts: RuntimeActivationArtifacts{
			Client:  activationArtifact(manifest.Artifacts.Client, "client"),
			Runtime: activationArtifact(manifest.Artifacts.Runtime, "runtime"),
		},
		Components: activationComponents(services),
		IssuedAt:   now.Format(time.RFC3339),
		ExpiresAt:  input.ExpiresAt,
	}
	if _, collector := services[ServiceCollector]; collector {
		if manifest.Artifacts.Collector == nil || !releaseChecksumsContain(checksums, optionalCollectorPayload, manifest.Artifacts.Collector.Checksum) {
			return DerivedRuntimeActivation{}, errors.New("DeploymentBinding 启用 collector，但已安装 Release 缺少受校验 collector artifact")
		}
		artifact := activationArtifact(*manifest.Artifacts.Collector, "collector")
		activation.Artifacts.Collector = &artifact
	}
	if err := validateRuntimeActivationFoundationInput(activation, now); err != nil {
		return DerivedRuntimeActivation{}, fmt.Errorf("本地派生 runtime activation 无效: %w", err)
	}
	return DerivedRuntimeActivation{Activation: activation, BindingDigestMode: bindingDigestRawCenterJSON, ReleaseVersion: manifest.Version}, nil
}

func validateActivationBinding(binding deploymentBinding, now time.Time) error {
	if binding.SchemaVersion != deploymentBindingSchema || !validRuntimeUUID(binding.BindingID) || binding.Revision < 1 ||
		!validRuntimeUUID(binding.NodeID) || !validRuntimeUUID(binding.DeploymentID) || !validRuntimeUUID(binding.ProjectID) || !validRuntimeUUID(binding.Release.ID) ||
		!validSHA256(binding.Release.ArchiveSHA256) || !validSHA256(binding.Release.ManifestSHA256) || !validSHA256(binding.Release.ChecksumsSHA256) || !validStableID(binding.Release.SigningKeyID) {
		return errors.New("DeploymentBinding 身份或 Release 字段无效")
	}
	if !validDeploymentPorts(binding) || len(binding.Secrets) != 0 {
		return errors.New("DeploymentBinding 违反端口或 Secret 安全基线")
	}
	if _, err := parseBindingTime(binding.IssuedAt); err != nil {
		return fmt.Errorf("DeploymentBinding issuedAt 无效: %w", err)
	}
	if binding.ExpiresAt != "" {
		expiresAt, err := parseBindingTime(binding.ExpiresAt)
		if err != nil || !expiresAt.After(now) {
			return errors.New("DeploymentBinding expiresAt 无效或已过期")
		}
	}
	if _, err := validatedBindingServices(binding.EnabledServices, binding.Ports.CollectorHealthLoopback); err != nil {
		return err
	}
	return nil
}

func activationArtifact(artifact releaseArtifactForVerification, layout string) RuntimeActivationArtifact {
	// true 表示未来 process manager 必须把该已封存产物作为只读输入交给组件；本函数
	// 不负责 mount，ValidateFixedLaunchGate 也不会把声明当作已完成的运行条件。
	return RuntimeActivationArtifact{ReleaseFile: artifact.File, Digest: artifact.Checksum, MaterializedLayout: layout, ReadOnlyMount: true}
}

func activationComponents(services map[ServiceGroup]struct{}) []RuntimeActivationComponent {
	components := []RuntimeActivationComponent{
		{Name: "project-gateway", ServiceGroup: string(ServiceProjectEntry), Enabled: true, DependsOn: []string{"runtime-api"}},
		{Name: "runtime-api", ServiceGroup: string(ServiceProjectEntry), Enabled: true, DependsOn: []string{}},
		{Name: "runtime-engine", ServiceGroup: string(ServiceDataRuntime), Enabled: true, DependsOn: []string{}},
	}
	if _, enabled := services[ServiceCollector]; enabled {
		components = append(components, RuntimeActivationComponent{Name: "collector", ServiceGroup: string(ServiceCollector), Enabled: true, DependsOn: []string{}})
	}
	return components
}

func derivedActivationID(bindingRaw []byte, releaseDigest string) string {
	sum := sha256.Sum256(append(append([]byte(nil), bindingRaw...), []byte("\x00"+releaseDigest)...))
	bytes := sum[:16]
	bytes[6] = (bytes[6] & 0x0f) | 0x50 // UUID v5 形状，仅作可审计的确定性本地标识。
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(bytes)
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32]
}

func verifyInstalledActivationRelease(installed InstalledRelease, binding deploymentBinding) (releaseManifestForVerification, releaseChecksums, error) {
	if !validSHA256(installed.ReleaseDigest) || strings.TrimSpace(installed.ReleaseDir) == "" {
		return releaseManifestForVerification{}, releaseChecksums{}, errors.New("已安装 Release 结果无效")
	}
	releaseDir, err := filepath.Abs(installed.ReleaseDir)
	if err != nil {
		return releaseManifestForVerification{}, releaseChecksums{}, err
	}
	if filepath.Base(releaseDir) != "sha256-"+strings.TrimPrefix(installed.ReleaseDigest, "sha256:") {
		return releaseManifestForVerification{}, releaseChecksums{}, errors.New("已安装 Release 内容寻址目录与摘要不匹配")
	}
	if err := requireSealedReleaseTree(releaseDir); err != nil {
		return releaseManifestForVerification{}, releaseChecksums{}, err
	}
	checksumsRaw, err := readReleaseRegularFile(releaseDir, "checksums.json", DefaultReleaseInstallLimits().MaxFileBytes)
	if err != nil || digestBytes(checksumsRaw) != binding.Release.ChecksumsSHA256 {
		return releaseManifestForVerification{}, releaseChecksums{}, errors.New("已安装 Release checksums.json 与 DeploymentBinding 不匹配")
	}
	checksums, err := parseReleaseChecksums(checksumsRaw)
	if err != nil || verifyReleaseChecksums(releaseDir, checksums, DefaultReleaseInstallLimits().MaxFileBytes) != nil {
		return releaseManifestForVerification{}, releaseChecksums{}, errors.New("已安装 Release checksum 校验失败")
	}
	manifestRaw, err := readReleaseRegularFile(releaseDir, "release-manifest.json", DefaultReleaseInstallLimits().MaxFileBytes)
	if err != nil || digestBytes(manifestRaw) != binding.Release.ManifestSHA256 {
		return releaseManifestForVerification{}, releaseChecksums{}, errors.New("已安装 Release manifest 与 DeploymentBinding 不匹配")
	}
	var manifest releaseManifestForVerification
	if err := strictDecodeJSON(manifestRaw, &manifest); err != nil {
		return releaseManifestForVerification{}, releaseChecksums{}, fmt.Errorf("已安装 Release manifest 无效: %w", err)
	}
	if manifest.SchemaVersion != releaseManifestSchema || manifest.ProjectID != binding.ProjectID || manifest.ReleaseID != binding.Release.ID || !validProjectCode(manifest.ProjectCode) || !validVersionValue(manifest.Version) || manifest.SupplyChain.SigningKeyID != binding.Release.SigningKeyID {
		return releaseManifestForVerification{}, releaseChecksums{}, errors.New("已安装 Release manifest 身份或签名 key 与 DeploymentBinding 不匹配")
	}
	if err := verifyManifestArtifact(manifest.Artifacts.Client, "client-assets.tar.zst", checksumMap(checksums)); err != nil {
		return releaseManifestForVerification{}, releaseChecksums{}, err
	}
	if err := verifyManifestArtifact(manifest.Artifacts.Runtime, "runtime-artifact.tar.zst", checksumMap(checksums)); err != nil {
		return releaseManifestForVerification{}, releaseChecksums{}, err
	}
	return manifest, checksums, nil
}

func requireSealedReleaseTree(root string) error {
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0o222 != 0 {
		return errors.New("已安装 Release 根目录不是受封存的只读目录")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		entryInfo, statErr := os.Lstat(filepath.Join(root, entry.Name()))
		// 安装器会物化唯一的 runtime-artifact 子目录；只接受固定布局，仍拒绝可写文件和链接。
		if statErr == nil && entry.Name() == "runtime-artifact" && entryInfo.IsDir() && entryInfo.Mode().Perm()&0o222 == 0 {
			children, err := os.ReadDir(filepath.Join(root, entry.Name()))
			if err != nil || len(children) != 1 || children[0].Name() != "runtime-project-artifact.json" {
				return errors.New("已安装 runtime artifact 目录结构无效")
			}
			child, err := os.Lstat(filepath.Join(root, entry.Name(), children[0].Name()))
			if err != nil || !child.Mode().IsRegular() || child.Mode().Perm()&0o222 != 0 {
				return errors.New("已安装 runtime artifact 不是只读普通文件")
			}
			continue
		}
		if statErr != nil || !entryInfo.Mode().IsRegular() || entryInfo.Mode()&os.ModeSymlink != 0 || entryInfo.Mode().Perm()&0o222 != 0 {
			return fmt.Errorf("已安装 Release 包含非只读普通文件: %s", entry.Name())
		}
	}
	return nil
}

func checksumMap(checksums releaseChecksums) map[string]releaseChecksumFile {
	result := make(map[string]releaseChecksumFile, len(checksums.Files))
	for _, entry := range checksums.Files {
		result[entry.Path] = entry
	}
	return result
}

func releaseChecksumsContain(checksums releaseChecksums, file, digest string) bool {
	entry, exists := checksumMap(checksums)[file]
	return exists && entry.SHA256 == digest
}

// FixedLaunchPlan 只是 Foundation process manager 将来可消费的本地计划，不能直接
// 执行。Executable、Args 与工作目录均由节点本地安装根和已校验身份派生，计划中不
// 存在来自 Binding 的 command、env、URL 或路径字段。
type FixedLaunchPlan struct {
	DeploymentID string                 `json:"deploymentId"`
	ReleaseDir   string                 `json:"releaseDir"`
	Components   []FixedLaunchComponent `json:"components"`
}

type FixedLaunchComponent struct {
	Name       string   `json:"name"`
	Executable string   `json:"executable"`
	Args       []string `json:"args"`
	WorkingDir string   `json:"workingDir"`
	HealthURL  string   `json:"healthUrl"`
}

// FixedLaunchPlanner 的目录和目标平台来自节点本地安装配置，而非 Center。当前仅
// Linux 的完整工程服务组有发布物；Windows 不得据此伪造 project_entry/runtime。
type FixedLaunchPlanner struct {
	installRoot string
	dataRoot    string
	platform    string
	arch        string
}

func NewFixedLaunchPlanner(installRoot, dataRoot, platform, arch string) (*FixedLaunchPlanner, error) {
	if strings.ToLower(strings.TrimSpace(platform)) != "linux" {
		return nil, errors.New("固定工程 Launcher 当前仅支持 Linux")
	}
	if strings.TrimSpace(arch) == "" {
		arch = runtime.GOARCH
	}
	if !validStableID(arch) {
		return nil, errors.New("本机架构标识无效")
	}
	install, err := filepath.Abs(installRoot)
	if err != nil {
		return nil, err
	}
	data, err := filepath.Abs(dataRoot)
	if err != nil {
		return nil, err
	}
	return &FixedLaunchPlanner{installRoot: filepath.Clean(install), dataRoot: filepath.Clean(data), platform: "linux", arch: arch}, nil
}

func (p *FixedLaunchPlanner) Plan(derived DerivedRuntimeActivation, installed InstalledRelease) (FixedLaunchPlan, error) {
	if p == nil {
		return FixedLaunchPlan{}, errors.New("固定 Launcher planner 不能为空")
	}
	activation := derived.Activation
	if err := validateRuntimeActivationFoundationInput(activation, time.Now().UTC()); err != nil {
		return FixedLaunchPlan{}, err
	}
	if !validVersionValue(derived.ReleaseVersion) {
		return FixedLaunchPlan{}, errors.New("已验证 Release version 缺失或非法")
	}
	if installed.ReleaseDir == "" || !validSHA256(installed.ReleaseDigest) {
		return FixedLaunchPlan{}, errors.New("已安装 Release 无效")
	}
	expectedPrefix := filepath.Join(p.dataRoot, "deployments", activation.DeploymentID, "release", "releases") + string(filepath.Separator)
	releaseDir, err := filepath.Abs(installed.ReleaseDir)
	if err != nil || !strings.HasPrefix(filepath.Clean(releaseDir)+string(filepath.Separator), expectedPrefix) {
		return FixedLaunchPlan{}, errors.New("已安装 Release 不在 deployment 固定目录内")
	}
	deploymentRoot := filepath.Join(p.dataRoot, "deployments", activation.DeploymentID)
	runtimeRoot := filepath.Join(deploymentRoot, "runtime")
	secretRoot := filepath.Join(deploymentRoot, "secrets")
	capabilityRoot := filepath.Join(p.installRoot, "capabilities", p.arch)
	components := []FixedLaunchComponent{
		{
			Name: "runtime-api", Executable: filepath.Join(capabilityRoot, "runtime-api"), WorkingDir: runtimeRoot, HealthURL: loopbackURL(activation.Ports.RuntimeAPILoopback, "/health"),
			Args: []string{"--listen", loopbackAddress(activation.Ports.RuntimeAPILoopback), "--artifact", filepath.Join(runtimeRoot, "runtime-project-artifact.json"), "--postgres-secret", filepath.Join(secretRoot, "runtime-api-postgres"), "--token-secret", filepath.Join(secretRoot, "runtime-api-tokens"), "--nats-url", "@foundation:nats-url", "--nats-credentials", filepath.Join(secretRoot, "runtime-api-nats"), "--deployment-id", activation.DeploymentID, "--project-id", activation.ProjectID, "--account-id", activation.AccountID, "--site-id", activation.SiteID, "--node-id", activation.NodeID, "--version", derived.ReleaseVersion, "--execution-form", "native-linux"},
		},
		{
			Name: "project-gateway", Executable: filepath.Join(capabilityRoot, "project-gateway"), WorkingDir: runtimeRoot, HealthURL: loopbackURL(activation.Ports.GatewayPublic, "/health"),
			Args: []string{"--listen", "0.0.0.0:" + fmt.Sprintf("%d", activation.Ports.GatewayPublic), "--client-root", filepath.Join(runtimeRoot, "client"), "--runtime-api", loopbackURL(activation.Ports.RuntimeAPILoopback, ""), "--deployment-id", activation.DeploymentID, "--account-id", activation.AccountID, "--project-id", activation.ProjectID, "--site-id", activation.SiteID, "--node-id", activation.NodeID, "--version", derived.ReleaseVersion, "--execution-form", "native-linux"},
		},
		{
			Name: "runtime-engine", Executable: filepath.Join(capabilityRoot, "runtime-engine"), WorkingDir: runtimeRoot, HealthURL: loopbackURL(activation.Ports.EngineLoopback, "/health"),
			Args: []string{"--config", filepath.Join(runtimeRoot, "runtime-engine.config.v2.json"), "--config-root", filepath.Join(runtimeRoot, "config"), "--index", filepath.Join(runtimeRoot, "collector-runtime-index.json"), "--listen", loopbackAddress(activation.Ports.EngineLoopback), "--production=true"},
		},
	}
	if activation.Artifacts.Collector != nil {
		components = append(components, FixedLaunchComponent{
			Name: "collector", Executable: filepath.Join(capabilityRoot, "collector", "industrial_collector"), WorkingDir: runtimeRoot, HealthURL: loopbackURL(*activation.Ports.CollectorHealthLoopback, "/health"),
			Args: []string{"--artifact", filepath.Join(runtimeRoot, "collector-artifact.json"), "--binding", filepath.Join(runtimeRoot, "collector-binding.json"), "--index", filepath.Join(runtimeRoot, "collector-index.json"), "--wal", filepath.Join(deploymentRoot, "collector-wal"), "--listen", loopbackURL(*activation.Ports.CollectorHealthLoopback, "/"), "--site-id", activation.SiteID, "--production", "true"},
		})
	}
	if err := validateDeploymentPortAvailability(activation.Ports); err != nil {
		return FixedLaunchPlan{}, err
	}
	for _, component := range components {
		if err := requireFixedExecutable(component.Executable); err != nil {
			return FixedLaunchPlan{}, fmt.Errorf("固定 Launcher 组件 %s 不可用: %w", component.Name, err)
		}
	}
	sort.Slice(components, func(i, j int) bool { return components[i].Name < components[j].Name })
	return FixedLaunchPlan{DeploymentID: activation.DeploymentID, ReleaseDir: releaseDir, Components: components}, nil
}

func loopbackAddress(port int) string          { return fmt.Sprintf("127.0.0.1:%d", port) }
func loopbackURL(port int, path string) string { return "http://" + loopbackAddress(port) + path }

// validateDeploymentPortAvailability 用操作系统真实监听结果补足中心的部署记录冲突检查。
// 检查监听会立即释放；正式进程启动时仍以 bind 结果作为最终竞争门禁。
func validateDeploymentPortAvailability(ports RuntimeActivationPorts) error {
	targets := []struct {
		name    string
		address string
	}{
		{name: "工程访问端口", address: fmt.Sprintf("0.0.0.0:%d", ports.GatewayPublic)},
		{name: "Runtime API", address: loopbackAddress(ports.RuntimeAPILoopback)},
		{name: "RuntimeEngine", address: loopbackAddress(ports.EngineLoopback)},
	}
	if ports.CollectorHealthLoopback != nil {
		targets = append(targets, struct {
			name    string
			address string
		}{name: "Collector", address: loopbackAddress(*ports.CollectorHealthLoopback)})
	}
	for _, target := range targets {
		listener, err := net.Listen("tcp", target.address)
		if err != nil {
			return fmt.Errorf("%s %s 已被本机进程占用: %w", target.name, target.address, err)
		}
		if err := listener.Close(); err != nil {
			return fmt.Errorf("释放端口预检监听失败: %w", err)
		}
	}
	return nil
}

func requireFixedExecutable(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o111 == 0 {
		return errors.New("不是无链接的可执行普通文件")
	}
	return nil
}

// RuntimeSecretResolver 只能返回本机已经落盘的 Secret 描述和路径；它不接受 URL 或
// Secret 值。ValidateFixedLaunchGate 会重新 Lstat、权限和摘要校验。
type RuntimeSecretResolver interface {
	Resolve(RuntimeSecretReference) (ResolvedRuntimeSecret, error)
}

type ResolvedRuntimeSecret struct {
	Name string
	Ref  string
	Path string
}

// ValidateFixedLaunchGate 是执行前的最后一道纯校验门。当前 FoundationProvisioner
// 只会写 declared，因而必定拒绝；没有 Foundation process manager 时不能伪造启动。
func ValidateFixedLaunchGate(activation RuntimeActivationFoundationInput, state RuntimeFoundationState, resolver RuntimeSecretResolver) error {
	if resolver == nil {
		return errors.New("Secret resolver 未配置，拒绝启动")
	}
	if state.SchemaVersion != runtimeFoundationStateSchema || state.DeploymentID != activation.DeploymentID || (state.Status != "provisioned" && state.Status != "healthy") {
		return errors.New("Runtime Foundation 未明确处于 provisioned/healthy，拒绝启动")
	}
	if !runtimeFoundationIntentsEqual(state.Intent, runtimeFoundationIntentFrom(activation)) {
		return errors.New("Runtime Foundation 状态与本地 Activation 意图不一致")
	}
	for _, reference := range activation.Secrets {
		resolved, err := resolver.Resolve(reference)
		if err != nil {
			return fmt.Errorf("解析 Secret %s: %w", reference.Name, err)
		}
		if resolved.Name != reference.Name || resolved.Ref != reference.Ref || !privateRegularSecretFile(resolved.Path, reference.SHA256) {
			return fmt.Errorf("Secret %s 不是受校验的私有普通文件", reference.Name)
		}
	}
	return nil
}

func privateRegularSecretFile(secretPath, expectedDigest string) bool {
	if !filepath.IsAbs(secretPath) {
		return false
	}
	info, err := os.Lstat(secretPath)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 || info.Size() > maxCenterResponseBytes {
		return false
	}
	payload, err := os.ReadFile(secretPath)
	return err == nil && digestBytes(payload) == expectedDigest
}

// formalLaunchGateError 供正式 Release 命令在安装成功后报告真实的本地门禁状态。
// 它不尝试从 Center 补齐 Foundation，也不调用 Supervisor 启动旧静态配置。
func (a *Agent) formalLaunchGateError(deploymentID string) error {
	statePath := filepath.Join(a.cfg.DataDir, "deployments", deploymentID, "foundation.json")
	state, exists, err := readRuntimeFoundationState(statePath)
	if err != nil {
		return fmt.Errorf("读取 Runtime Foundation 状态失败，拒绝启动: %w", err)
	}
	if !exists || state.Status == runtimeFoundationDeclared {
		return fmt.Errorf("deployment %s Release 已验证安装，但 Runtime Foundation 尚未 provisioned/healthy，拒绝启动", deploymentID)
	}
	if state.Status != "provisioned" && state.Status != "healthy" {
		return fmt.Errorf("deployment %s Runtime Foundation 状态 %q 不可执行，拒绝启动", deploymentID, state.Status)
	}
	return fmt.Errorf("deployment %s Runtime Foundation 已满足基础状态，但 Foundation process manager 与 Secret resolver 尚未接入，拒绝启动", deploymentID)
}
