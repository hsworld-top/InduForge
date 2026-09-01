package ops

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	runtimeActivationSchema      = "runtime-activation.v1"
	runtimeFoundationStateSchema = "runtime-foundation-state.v1"
	runtimeFoundationDeclared    = "declared"
)

// RuntimeActivationFoundationInput 是 runtime-activation.v1 在 NodeAgent 内用于
// 本地基础设施声明的完整、无明文 Secret 投影。它只描述“需要什么”，不能携带 DSN、
// URL、命令、环境变量或密码。
type RuntimeActivationFoundationInput struct {
	SchemaVersion string                       `json:"schemaVersion"`
	ActivationID  string                       `json:"activationId"`
	Revision      int                          `json:"revision"`
	NodeID        string                       `json:"nodeId"`
	DeploymentID  string                       `json:"deploymentId"`
	ProjectID     string                       `json:"projectId"`
	ReleaseID     string                       `json:"releaseId"`
	SiteID        string                       `json:"siteId"`
	AccountID     string                       `json:"accountId"`
	Binding       RuntimeActivationBinding     `json:"binding"`
	Ports         RuntimeActivationPorts       `json:"ports"`
	Foundation    RuntimeFoundationPlan        `json:"foundation"`
	Secrets       []RuntimeSecretReference     `json:"secrets"`
	Artifacts     RuntimeActivationArtifacts   `json:"artifacts"`
	Components    []RuntimeActivationComponent `json:"components"`
	IssuedAt      string                       `json:"issuedAt"`
	ExpiresAt     string                       `json:"expiresAt"`
}

type RuntimeActivationBinding struct {
	BindingID     string `json:"bindingId"`
	Revision      int    `json:"revision"`
	BindingSHA256 string `json:"bindingSha256"`
}

type RuntimeActivationPorts struct {
	GatewayPublic           int  `json:"gatewayPublic"`
	RuntimeAPILoopback      int  `json:"runtimeApiLoopback"`
	EngineLoopback          int  `json:"engineLoopback"`
	CollectorHealthLoopback *int `json:"collectorHealthLoopback,omitempty"`
}

type RuntimeFoundationPlan struct {
	Postgres      RuntimeFoundationAllocation `json:"postgres"`
	NATSJetStream RuntimeFoundationAllocation `json:"natsJetStream"`
}

// RuntimeFoundationAllocation 不含连接信息；allocationDigest 所代表的真实资源配置
// 必须由后续受信 launcher/基础设施管理器解析和供给。
type RuntimeFoundationAllocation struct {
	AllocationID     string `json:"allocationId"`
	Revision         int    `json:"revision"`
	AllocationDigest string `json:"allocationDigest"`
	SchemaPlanDigest string `json:"schemaPlanDigest,omitempty"`
	StreamPlanDigest string `json:"streamPlanDigest,omitempty"`
	State            string `json:"state"`
}

type RuntimeSecretReference struct {
	Name          string `json:"name"`
	Ref           string `json:"ref"`
	SchemaVersion string `json:"schemaVersion"`
	SHA256        string `json:"sha256"`
	Revision      int    `json:"revision"`
	ExpiresAt     string `json:"expiresAt"`
}

type RuntimeActivationArtifacts struct {
	Client    RuntimeActivationArtifact  `json:"client"`
	Runtime   RuntimeActivationArtifact  `json:"runtime"`
	Collector *RuntimeActivationArtifact `json:"collector,omitempty"`
}

type RuntimeActivationArtifact struct {
	ReleaseFile        string `json:"releaseFile"`
	Digest             string `json:"digest"`
	MaterializedLayout string `json:"materializedLayout"`
	ReadOnlyMount      bool   `json:"readOnlyMount"`
}

type RuntimeActivationComponent struct {
	Name         string   `json:"name"`
	ServiceGroup string   `json:"serviceGroup"`
	Enabled      bool     `json:"enabled"`
	DependsOn    []string `json:"dependsOn"`
}

// RuntimeFoundationState 仅保存可审计的资源意图和 Secret 引用。Status=declared
// 特意不表示 PostgreSQL/NATS 已运行，防止调用方把声明误当成基础设施健康状态。
type RuntimeFoundationState struct {
	SchemaVersion string                  `json:"schemaVersion"`
	DeploymentID  string                  `json:"deploymentId"`
	Status        string                  `json:"status"`
	Intent        RuntimeFoundationIntent `json:"intent"`
	CreatedAt     string                  `json:"createdAt"`
	UpdatedAt     string                  `json:"updatedAt"`
}

type RuntimeFoundationIntent struct {
	ActivationID string                   `json:"activationId"`
	Revision     int                      `json:"revision"`
	NodeID       string                   `json:"nodeId"`
	ProjectID    string                   `json:"projectId"`
	ReleaseID    string                   `json:"releaseId"`
	Binding      RuntimeActivationBinding `json:"binding"`
	Foundation   RuntimeFoundationPlan    `json:"foundation"`
	Secrets      []RuntimeSecretReference `json:"secrets"`
}

// RuntimeFoundationProvisioner 只原子持久化每部署的基础设施声明；它不创建容器、
// 不执行 shell，也不尝试连接 PostgreSQL 或 NATS。正式 launcher 必须在接管后独立
// 探活并报告运行状态。
type RuntimeFoundationProvisioner struct {
	root     string
	platform string
	now      func() time.Time
}

var runtimeFoundationPathLocks sync.Map // map[string]*sync.Mutex，保证同一进程内按部署串行落盘。

func NewRuntimeFoundationProvisioner(root string) (*RuntimeFoundationProvisioner, error) {
	return newRuntimeFoundationProvisioner(root, runtime.GOOS, time.Now)
}

func newRuntimeFoundationProvisioner(root, platform string, now func() time.Time) (*RuntimeFoundationProvisioner, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("runtime foundation 根目录不能为空")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("解析 runtime foundation 根目录: %w", err)
	}
	if now == nil {
		now = time.Now
	}
	return &RuntimeFoundationProvisioner{root: filepath.Clean(abs), platform: strings.ToLower(strings.TrimSpace(platform)), now: now}, nil
}

// DecodeRuntimeActivationFoundationInput 严格拒绝契约以外的字段。因此含 value、
// password、dsn 等明文 Secret 字段的输入不能被误落盘。
func DecodeRuntimeActivationFoundationInput(raw []byte) (RuntimeActivationFoundationInput, error) {
	var input RuntimeActivationFoundationInput
	if err := strictDecodeJSON(raw, &input); err != nil {
		return RuntimeActivationFoundationInput{}, fmt.Errorf("runtime activation JSON 无效: %w", err)
	}
	return input, nil
}

// Ensure 把已校验的本地声明写入 deployment 专属目录。相同 revision 的重试只会
// 返回已有状态；同 revision 不同内容和 revision 回退都拒绝，避免竞态覆盖。
func (p *RuntimeFoundationProvisioner) Ensure(input RuntimeActivationFoundationInput) (RuntimeFoundationState, error) {
	if p == nil {
		return RuntimeFoundationState{}, errors.New("runtime foundation provisioner 不能为空")
	}
	if p.platform != "linux" {
		return RuntimeFoundationState{}, fmt.Errorf("正式本地 Runtime Foundation 当前仅支持 Linux，平台 %q 被拒绝", p.platform)
	}
	if err := validateRuntimeActivationFoundationInput(input, p.now().UTC()); err != nil {
		return RuntimeFoundationState{}, err
	}
	statePath := filepath.Join(p.root, "deployments", input.DeploymentID, "foundation.json")
	lock := runtimeFoundationLock(statePath)
	lock.Lock()
	defer lock.Unlock()

	if existing, exists, err := readRuntimeFoundationState(statePath); err != nil {
		return RuntimeFoundationState{}, err
	} else if exists {
		if err := validateRuntimeFoundationState(existing); err != nil {
			return RuntimeFoundationState{}, fmt.Errorf("已有 Runtime Foundation 状态无效: %w", err)
		}
		if existing.DeploymentID != input.DeploymentID {
			return RuntimeFoundationState{}, errors.New("Runtime Foundation 状态的 deploymentId 不匹配")
		}
		if existing.Intent.Revision > input.Revision {
			return RuntimeFoundationState{}, fmt.Errorf("拒绝 Runtime Foundation revision 回退: 已有=%d 输入=%d", existing.Intent.Revision, input.Revision)
		}
		intent := runtimeFoundationIntentFrom(input)
		if existing.Intent.Revision == input.Revision {
			if !runtimeFoundationIntentsEqual(existing.Intent, intent) {
				return RuntimeFoundationState{}, errors.New("相同 Runtime Foundation revision 的声明不一致")
			}
			return existing, nil
		}
		state := newRuntimeFoundationState(input, existing.CreatedAt, p.now().UTC())
		if err := writeRuntimeFoundationState(statePath, state); err != nil {
			return RuntimeFoundationState{}, err
		}
		return state, nil
	}

	state := newRuntimeFoundationState(input, "", p.now().UTC())
	if err := writeRuntimeFoundationState(statePath, state); err != nil {
		return RuntimeFoundationState{}, err
	}
	return state, nil
}

func (p *RuntimeFoundationProvisioner) StatePath(deploymentID string) (string, error) {
	if p == nil || !validRuntimeUUID(deploymentID) {
		return "", errors.New("deploymentId 非法")
	}
	return filepath.Join(p.root, "deployments", deploymentID, "foundation.json"), nil
}

func runtimeFoundationLock(path string) *sync.Mutex {
	value, _ := runtimeFoundationPathLocks.LoadOrStore(path, &sync.Mutex{})
	return value.(*sync.Mutex)
}

func runtimeFoundationIntentFrom(input RuntimeActivationFoundationInput) RuntimeFoundationIntent {
	secrets := append([]RuntimeSecretReference(nil), input.Secrets...)
	sort.Slice(secrets, func(i, j int) bool { return secrets[i].Name < secrets[j].Name })
	return RuntimeFoundationIntent{
		ActivationID: input.ActivationID,
		Revision:     input.Revision,
		NodeID:       input.NodeID,
		ProjectID:    input.ProjectID,
		ReleaseID:    input.ReleaseID,
		Binding:      input.Binding,
		Foundation:   input.Foundation,
		Secrets:      secrets,
	}
}

func newRuntimeFoundationState(input RuntimeActivationFoundationInput, createdAt string, now time.Time) RuntimeFoundationState {
	timestamp := now.Format(time.RFC3339)
	if createdAt == "" {
		createdAt = timestamp
	}
	return RuntimeFoundationState{
		SchemaVersion: runtimeFoundationStateSchema,
		DeploymentID:  input.DeploymentID,
		Status:        runtimeFoundationDeclared,
		Intent:        runtimeFoundationIntentFrom(input),
		CreatedAt:     createdAt,
		UpdatedAt:     timestamp,
	}
}

func runtimeFoundationIntentsEqual(left, right RuntimeFoundationIntent) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftJSON, rightJSON)
}

func readRuntimeFoundationState(path string) (RuntimeFoundationState, bool, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return RuntimeFoundationState{}, false, nil
	}
	if err != nil {
		return RuntimeFoundationState{}, false, fmt.Errorf("读取 Runtime Foundation 状态: %w", err)
	}
	var state RuntimeFoundationState
	if err := strictDecodeJSON(raw, &state); err != nil {
		return RuntimeFoundationState{}, false, fmt.Errorf("解析 Runtime Foundation 状态: %w", err)
	}
	return state, true, nil
}

func writeRuntimeFoundationState(path string, state RuntimeFoundationState) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("创建 Runtime Foundation 目录: %w", err)
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("编码 Runtime Foundation 状态: %w", err)
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, raw, 0600); err != nil {
		return fmt.Errorf("写入 Runtime Foundation 临时状态: %w", err)
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("原子更新 Runtime Foundation 状态: %w", err)
	}
	return nil
}

func validateRuntimeFoundationState(state RuntimeFoundationState) error {
	if state.SchemaVersion != runtimeFoundationStateSchema || state.Status != runtimeFoundationDeclared || !validRuntimeUUID(state.DeploymentID) ||
		state.CreatedAt == "" || state.UpdatedAt == "" {
		return errors.New("状态元数据无效")
	}
	if _, err := parseBindingTime(state.CreatedAt); err != nil {
		return err
	}
	if _, err := parseBindingTime(state.UpdatedAt); err != nil {
		return err
	}
	return validateRuntimeFoundationIntent(state.Intent)
}

func validateRuntimeActivationFoundationInput(input RuntimeActivationFoundationInput, now time.Time) error {
	if input.SchemaVersion != runtimeActivationSchema || input.Revision < 1 ||
		!validRuntimeUUID(input.ActivationID) || !validRuntimeUUID(input.NodeID) || !validRuntimeUUID(input.DeploymentID) ||
		!validRuntimeUUID(input.ProjectID) || !validRuntimeUUID(input.ReleaseID) || !validRuntimeUUID(input.SiteID) || !validRuntimeUUID(input.AccountID) {
		return errors.New("runtime activation 身份字段无效")
	}
	if !validRuntimeUUID(input.Binding.BindingID) || input.Binding.Revision < 1 || !validSHA256(input.Binding.BindingSHA256) {
		return errors.New("runtime activation binding 无效")
	}
	base := input.Ports.GatewayPublic
	if base < 1024 || base > 65532 || input.Ports.RuntimeAPILoopback != base+1 || input.Ports.EngineLoopback != base+2 {
		return errors.New("runtime activation 端口不符合客户部署端口基线")
	}
	if _, err := parseRuntimeActivationTime(input.IssuedAt, false, now); err != nil {
		return fmt.Errorf("runtime activation issuedAt 无效: %w", err)
	}
	if _, err := parseRuntimeActivationTime(input.ExpiresAt, true, now); err != nil {
		return fmt.Errorf("runtime activation expiresAt 无效: %w", err)
	}
	if err := validateRuntimeFoundationIntent(runtimeFoundationIntentFrom(input)); err != nil {
		return err
	}
	for _, secret := range input.Secrets {
		expiresAt, err := parseBindingTime(secret.ExpiresAt)
		if err != nil || !expiresAt.After(now) {
			return fmt.Errorf("Secret 引用 %q expiresAt 无效或已过期", secret.Name)
		}
	}
	if err := validateRuntimeActivationArtifacts(input.Artifacts); err != nil {
		return err
	}
	return validateRuntimeActivationComponents(input.Components, input.Secrets, input.Artifacts, input.Ports.GatewayPublic, input.Ports.CollectorHealthLoopback)
}

func validateRuntimeFoundationIntent(intent RuntimeFoundationIntent) error {
	if intent.Revision < 1 || !validRuntimeUUID(intent.ActivationID) || !validRuntimeUUID(intent.NodeID) ||
		!validRuntimeUUID(intent.ProjectID) || !validRuntimeUUID(intent.ReleaseID) || !validRuntimeUUID(intent.Binding.BindingID) ||
		intent.Binding.Revision < 1 || !validSHA256(intent.Binding.BindingSHA256) {
		return errors.New("Runtime Foundation intent 身份字段无效")
	}
	if err := validateRuntimeFoundationAllocation(intent.Foundation.Postgres, true); err != nil {
		return fmt.Errorf("PostgreSQL allocation 无效: %w", err)
	}
	if err := validateRuntimeFoundationAllocation(intent.Foundation.NATSJetStream, false); err != nil {
		return fmt.Errorf("NATS JetStream allocation 无效: %w", err)
	}
	return validateRuntimeSecretReferences(intent.Secrets)
}

func validateRuntimeFoundationAllocation(allocation RuntimeFoundationAllocation, postgres bool) error {
	if !validRuntimeUUID(allocation.AllocationID) || allocation.Revision < 1 || !validSHA256(allocation.AllocationDigest) || allocation.State != "ready" {
		return errors.New("allocation 元数据无效")
	}
	if postgres {
		if !validSHA256(allocation.SchemaPlanDigest) || allocation.StreamPlanDigest != "" {
			return errors.New("PostgreSQL allocation plan 无效")
		}
	} else if !validSHA256(allocation.StreamPlanDigest) || allocation.SchemaPlanDigest != "" {
		return errors.New("NATS JetStream allocation plan 无效")
	}
	return nil
}

func validateRuntimeSecretReferences(secrets []RuntimeSecretReference) error {
	if len(secrets) < 5 || len(secrets) > 7 {
		return errors.New("Runtime Foundation Secret 引用数量无效")
	}
	required := map[string]bool{
		"runtime-api-postgres": false, "runtime-api-tokens": false, "runtime-api-nats": false,
		"runtime-engine-postgres": false, "runtime-engine-nats": false,
	}
	optional := map[string]bool{"collector-nats": false, "compute-sandbox": false}
	for _, secret := range secrets {
		if !validOpaqueSecretReference(secret.Ref) || !validStableID(secret.SchemaVersion) || !validSHA256(secret.SHA256) || secret.Revision < 1 {
			return fmt.Errorf("Secret 引用 %q 不是受信的非敏感引用", secret.Name)
		}
		if _, err := parseBindingTime(secret.ExpiresAt); err != nil {
			return fmt.Errorf("Secret 引用 %q expiresAt 无效: %w", secret.Name, err)
		}
		if _, exists := required[secret.Name]; exists {
			if required[secret.Name] {
				return fmt.Errorf("Secret 引用重复: %s", secret.Name)
			}
			required[secret.Name] = true
			continue
		}
		if _, exists := optional[secret.Name]; !exists || optional[secret.Name] {
			return fmt.Errorf("Secret 引用名称无效或重复: %s", secret.Name)
		}
		optional[secret.Name] = true
	}
	for name, exists := range required {
		if !exists {
			return fmt.Errorf("缺少必要 Secret 引用: %s", name)
		}
	}
	return nil
}

func validOpaqueSecretReference(value string) bool {
	return validStableID(value) && !strings.ContainsAny(value, ":/")
}

func validateRuntimeActivationArtifacts(artifacts RuntimeActivationArtifacts) error {
	if !validRuntimeActivationArtifact(artifacts.Client, "client-assets.tar.zst", "client") ||
		!validRuntimeActivationArtifact(artifacts.Runtime, "runtime-artifact.tar.zst", "runtime") {
		return errors.New("runtime activation 必需 artifact 无效")
	}
	if artifacts.Collector != nil && !validRuntimeActivationArtifact(*artifacts.Collector, "collector-artifact.tar.zst", "collector") {
		return errors.New("runtime activation collector artifact 无效")
	}
	return nil
}

func validRuntimeActivationArtifact(artifact RuntimeActivationArtifact, file, layout string) bool {
	return artifact.ReleaseFile == file && artifact.MaterializedLayout == layout && artifact.ReadOnlyMount && validSHA256(artifact.Digest)
}

func validateRuntimeActivationComponents(components []RuntimeActivationComponent, secrets []RuntimeSecretReference, artifacts RuntimeActivationArtifacts, base int, collectorPort *int) error {
	if len(components) < 3 || len(components) > 4 {
		return errors.New("runtime activation component 数量无效")
	}
	expected := map[string]RuntimeActivationComponent{
		"project-gateway": {Name: "project-gateway", ServiceGroup: "project_entry", Enabled: true, DependsOn: []string{"runtime-api"}},
		"runtime-api":     {Name: "runtime-api", ServiceGroup: "project_entry", Enabled: true, DependsOn: []string{}},
		"runtime-engine":  {Name: "runtime-engine", ServiceGroup: "data_runtime", Enabled: true, DependsOn: []string{}},
	}
	collectorEnabled := false
	for _, component := range components {
		if wanted, exists := expected[component.Name]; exists {
			if !runtimeActivationComponentsEqual(component, wanted) {
				return fmt.Errorf("runtime activation component %s 不符合固定声明", component.Name)
			}
			delete(expected, component.Name)
			continue
		}
		if component.Name != "collector" || component.ServiceGroup != "collector" || !component.Enabled || len(component.DependsOn) != 0 || collectorEnabled {
			return errors.New("runtime activation collector 声明无效")
		}
		collectorEnabled = true
	}
	if len(expected) != 0 {
		return errors.New("runtime activation 缺少必要 component")
	}
	hasCollectorSecret := false
	for _, secret := range secrets {
		hasCollectorSecret = hasCollectorSecret || secret.Name == "collector-nats"
	}
	if collectorEnabled != hasCollectorSecret {
		return errors.New("collector component 与 Secret 引用不一致")
	}
	if collectorEnabled && (collectorPort == nil || *collectorPort != base+3 || artifacts.Collector == nil) {
		return errors.New("collector component 缺少部署端口或只读 artifact")
	}
	if !collectorEnabled && (collectorPort != nil || artifacts.Collector != nil) {
		return errors.New("未启用 collector 时不允许声明端口或 artifact")
	}
	return nil
}

func runtimeActivationComponentsEqual(left, right RuntimeActivationComponent) bool {
	if left.Name != right.Name || left.ServiceGroup != right.ServiceGroup || left.Enabled != right.Enabled || len(left.DependsOn) != len(right.DependsOn) {
		return false
	}
	for index := range left.DependsOn {
		if left.DependsOn[index] != right.DependsOn[index] {
			return false
		}
	}
	return true
}

func parseRuntimeActivationTime(value string, requireFuture bool, now time.Time) (time.Time, error) {
	parsed, err := parseBindingTime(value)
	if err != nil {
		return time.Time{}, err
	}
	if requireFuture && !parsed.After(now) {
		return time.Time{}, errors.New("必须晚于当前时间")
	}
	return parsed, nil
}
