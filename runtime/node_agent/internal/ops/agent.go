package ops

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/indu-forge/node_agent/internal/hostd"
	"github.com/indu-forge/node_agent/internal/pkg/utils"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

// Config 是 ops Agent 的最小持久配置。enrollmentCode 只在未领取身份时使用，
// 成功后只保存 node id/token，不会把 enrollment code 长期写入磁盘。
type Config struct {
	InstallRoot         string              `yaml:"-"`
	SecurityMode        string              `yaml:"securityMode"`
	Enabled             bool                `yaml:"enabled"`
	ServerURL           string              `yaml:"serverUrl"`
	EnrollmentCode      string              `yaml:"enrollmentCode"`
	AgentVersion        string              `yaml:"agentVersion"`
	HeartbeatEvery      time.Duration       `yaml:"heartbeatEvery"`
	DataDir             string              `yaml:"dataDir"`
	HostdSocket         string              `yaml:"hostdSocket"`
	HostDataDir         string              `yaml:"hostDataDir"`
	NodeIP              string              `yaml:"nodeIp"`
	TrustKeys           []TrustedSigningKey `yaml:"trustKeys"`
	RuntimeVersion      string              `yaml:"runtimeVersion"`
	ClearEnrollmentCode func() error        `yaml:"-"`
	ReconcileError      func(error)         `yaml:"-"`
}

// TrustedSigningKey 是节点本地受信 Release 签名公钥。中心只能在 Binding 中引用
// keyId，不能下发或替换这个公钥。
type TrustedSigningKey struct {
	KeyID     string `yaml:"keyId"`
	PublicKey string `yaml:"publicKey"`
}

const maxCenterResponseBytes = 1 << 20

// Identity 是领取成功后的设备凭据，文件权限只允许服务账户读取。
type Identity struct {
	NodeID          string `json:"nodeId"`
	AgentToken      string `json:"agentToken"`
	PendingApproval bool   `json:"pendingApproval"`
}

type HostInfo struct {
	DisplayName        string `json:"displayName,omitempty"`
	Hostname           string `json:"hostname"`
	OS                 string `json:"os"`
	Architecture       string `json:"architecture"`
	MachineFingerprint string `json:"machineFingerprint"`
	IPAddress          string `json:"ipAddress,omitempty"`
}

type AgentCommand struct {
	NodeID          string `json:"nodeId"`
	RunID           string `json:"runId"`
	DeploymentID    string `json:"deploymentId"`
	ReleaseID       string `json:"releaseId"`
	ServiceID       string `json:"serviceId"`
	ServiceType     string `json:"serviceType"`
	Operation       string `json:"operation"`
	DesiredStatus   string `json:"desiredStatus"`
	Generation      int64  `json:"generation"`
	Version         string `json:"version"`
	ArchiveSHA256   string `json:"archiveSha256"`
	ManifestSHA256  string `json:"manifestSha256"`
	ChecksumsSHA256 string `json:"checksumsSha256"`
	SigningKeyID    string `json:"signingKeyId"`
	BindingRevision int    `json:"bindingRevision"`
	ReplicasDesired int    `json:"replicasDesired"`
}

// Agent 将中心 desired 状态与本机 observed 状态分开处理。中心只能选择本机
// 声明的服务组，不能传递命令、路径、参数或环境变量。
type Agent struct {
	cfg                Config
	identity           Identity
	supervisor         *Supervisor
	client             *http.Client
	trustKeys          map[string]ed25519.PublicKey
	mu                 sync.RWMutex
	applied            map[string]int64
	hostd              *hostd.Client
	clusterFailure     *hostd.ClusterState
	foundationFailure  *hostd.FoundationState
	lastReconcileError string
}

func NewAgent(cfg Config, supervisor *Supervisor) (*Agent, error) {
	if cfg.SecurityMode != "" && cfg.SecurityMode != "development" && cfg.SecurityMode != "production" {
		return nil, fmt.Errorf("securityMode 只允许 development 或 production")
	}
	if supervisor == nil {
		return nil, fmt.Errorf("supervisor 不能为空")
	}
	if cfg.Enabled {
		if err := validateServerURL(cfg.ServerURL); err != nil {
			return nil, err
		}
	}
	if cfg.HeartbeatEvery <= 0 {
		cfg.HeartbeatEvery = 10 * time.Second
	}
	if cfg.DataDir == "" {
		cfg.DataDir = "./data"
	}
	if cfg.AgentVersion == "" {
		cfg.AgentVersion = "unknown"
	}
	trustKeys, err := parseTrustedSigningKeys(cfg.TrustKeys)
	if err != nil {
		return nil, err
	}
	var hostClient *hostd.Client
	if strings.TrimSpace(cfg.HostdSocket) != "" {
		hostClient, err = hostd.NewUnixClient(strings.TrimSpace(cfg.HostdSocket))
		if err != nil {
			return nil, err
		}
	}
	return &Agent{cfg: cfg, supervisor: supervisor, client: newCenterHTTPClient(8 * time.Second), trustKeys: trustKeys, applied: make(map[string]int64), hostd: hostClient}, nil
}

func (a *Agent) identityPath() string { return filepath.Join(a.cfg.DataDir, "ops-agent-identity.json") }
func (a *Agent) appliedPath() string {
	return filepath.Join(a.cfg.DataDir, "ops-agent-applied-generations.json")
}

func (a *Agent) loadIdentity() error {
	data, err := os.ReadFile(a.identityPath())
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var identity Identity
	if err := json.Unmarshal(data, &identity); err != nil {
		return fmt.Errorf("读取节点身份失败: %w", err)
	}
	if identity.NodeID == "" || identity.AgentToken == "" {
		return fmt.Errorf("节点身份不完整")
	}
	a.mu.Lock()
	a.identity = identity
	a.mu.Unlock()
	return nil
}

func (a *Agent) saveIdentity(identity Identity) error {
	if identity.NodeID == "" || identity.AgentToken == "" {
		return fmt.Errorf("中心返回的节点身份不完整")
	}
	if err := os.MkdirAll(a.cfg.DataDir, 0700); err != nil {
		return err
	}
	path := a.identityPath()
	temporary := path + ".tmp"
	data, err := json.Marshal(identity)
	if err != nil {
		return err
	}
	if err := os.WriteFile(temporary, data, 0600); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		return err
	}
	a.mu.Lock()
	a.identity = identity
	a.mu.Unlock()
	return nil
}

func (a *Agent) loadApplied() error {
	data, err := os.ReadFile(a.appliedPath())
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var applied map[string]int64
	if err := json.Unmarshal(data, &applied); err != nil {
		return fmt.Errorf("读取本地命令代次失败: %w", err)
	}
	if applied == nil {
		applied = make(map[string]int64)
	}
	a.mu.Lock()
	a.applied = applied
	a.mu.Unlock()
	return nil
}

func (a *Agent) appliedGeneration(serviceID string) int64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.applied[serviceID]
}
func (a *Agent) saveApplied(serviceID string, generation int64) error {
	a.mu.Lock()
	if generation <= a.applied[serviceID] {
		a.mu.Unlock()
		return nil
	}
	a.applied[serviceID] = generation
	data, err := json.Marshal(a.applied)
	a.mu.Unlock()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(a.cfg.DataDir, 0700); err != nil {
		return err
	}
	temporary := a.appliedPath() + ".tmp"
	if err := os.WriteFile(temporary, data, 0600); err != nil {
		return err
	}
	return os.Rename(temporary, a.appliedPath())
}

func (a *Agent) currentIdentity() Identity { a.mu.RLock(); defer a.mu.RUnlock(); return a.identity }

func (a *Agent) claim(ctx context.Context) error {
	if strings.TrimSpace(a.cfg.EnrollmentCode) == "" {
		return fmt.Errorf("未配置 enrollmentCode 且本地没有已领取节点身份")
	}
	host := a.hostInfo()
	payload := map[string]any{
		"code":               a.cfg.EnrollmentCode,
		"hostname":           host.Hostname,
		"platform":           host.OS,
		"architecture":       host.Architecture,
		"machineFingerprint": host.MachineFingerprint,
		"ipAddress":          host.IPAddress,
		"agentVersion":       a.cfg.AgentVersion,
		"capabilities":       a.supervisor.Capabilities(),
	}
	var result struct {
		Node struct {
			ID string `json:"id"`
		} `json:"node"`
		AgentToken      string `json:"agentToken"`
		PendingApproval bool   `json:"pendingApproval"`
	}
	if err := a.request(ctx, http.MethodPost, "/api/v1/ops/agent/enrollments/claim", "", payload, &result); err != nil {
		return err
	}
	if err := a.saveIdentity(Identity{NodeID: result.Node.ID, AgentToken: result.AgentToken, PendingApproval: result.PendingApproval}); err != nil {
		return err
	}
	if a.cfg.ClearEnrollmentCode != nil {
		if err := a.cfg.ClearEnrollmentCode(); err != nil {
			return err
		}
		a.cfg.EnrollmentCode = ""
	}
	return nil
}

func (a *Agent) hostInfo() HostInfo {
	hostname, _ := os.Hostname()
	return HostInfo{Hostname: hostname, OS: runtime.GOOS, Architecture: runtime.GOARCH, MachineFingerprint: utils.GetMachineID(), IPAddress: a.nodeIP()}
}

// Run 首先加载或领取身份；之后立即心跳并按固定周期拉取 desired workload。任何单次网络
// 失败只记录为本轮失败，不影响已经运行的采集进程。
func (a *Agent) Run(ctx context.Context) error {
	if !a.cfg.Enabled {
		return nil
	}
	if err := validateServerURL(a.cfg.ServerURL); err != nil {
		return err
	}
	if err := a.loadIdentity(); err != nil {
		return err
	}
	if err := a.loadApplied(); err != nil {
		return err
	}
	a.reconcileOnce(ctx)
	ticker := time.NewTicker(a.cfg.HeartbeatEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			a.reconcileOnce(ctx)
		}
	}
}

func (a *Agent) reconcileOnce(ctx context.Context) {
	if a.currentIdentity().NodeID == "" {
		// 控制面可能比 Agent 晚就绪；领取失败留待下一周期重试，不影响本机已运行进程。
		if err := a.claim(ctx); err != nil {
			a.reportReconcileError(err)
		} else {
			a.clearReconcileError()
		}
		return
	}
	// 身份已经落盘但上一次清除引导码失败时，每轮继续重试；在明文接入码
	// 清理成功前不进入日常心跳，避免一次瞬时文件错误把敏感码永久遗留。
	if strings.TrimSpace(a.cfg.EnrollmentCode) != "" && a.cfg.ClearEnrollmentCode != nil {
		if err := a.cfg.ClearEnrollmentCode(); err != nil {
			a.reportReconcileError(fmt.Errorf("清除一次性接入码失败: %w", err))
			return
		}
		a.cfg.EnrollmentCode = ""
	}
	if err := a.Heartbeat(ctx); err != nil {
		a.reportReconcileError(fmt.Errorf("上报节点心跳失败: %w", err))
		return
	}
	commands, err := a.Commands(ctx)
	if err != nil {
		a.reportReconcileError(fmt.Errorf("同步节点期望状态失败: %w", err))
		return
	}
	for _, command := range commands {
		if err := a.reconcile(ctx, command); err != nil {
			a.reportReconcileError(fmt.Errorf("执行节点服务计划失败: %w", err))
			return
		}
	}
	a.clearReconcileError()
}

// reportReconcileError 只在错误内容发生变化时记录，避免网络持续中断期间每个
// 心跳周期重复刷屏；恢复成功后会清空，以便同类错误再次出现时仍可记录。
func (a *Agent) reportReconcileError(err error) {
	if err == nil || a.cfg.ReconcileError == nil {
		return
	}
	message := truncateAgentMessage(err.Error())
	a.mu.Lock()
	if message == a.lastReconcileError {
		a.mu.Unlock()
		return
	}
	a.lastReconcileError = message
	a.mu.Unlock()
	a.cfg.ReconcileError(errors.New(message))
}

func (a *Agent) clearReconcileError() {
	a.mu.Lock()
	a.lastReconcileError = ""
	a.mu.Unlock()
}

func (a *Agent) Reconcile(command AgentCommand) error {
	return a.reconcile(context.Background(), command)
}

func (a *Agent) reconcile(ctx context.Context, command AgentCommand) error {
	if command.ServiceID == "" {
		return fmt.Errorf("serviceId 不能为空")
	}
	identity := a.currentIdentity()
	if command.NodeID == "" || command.NodeID != identity.NodeID {
		return fmt.Errorf("服务未显式分配给本节点")
	}
	formal, err := commandUsesFormalRelease(command)
	if err != nil {
		return err
	}
	group := ServiceGroup(command.ServiceType)
	engineCommand := false
	if formal {
		if engineGroup, ok := engineServiceGroup(command.ServiceType); ok {
			group, engineCommand = engineGroup, true
		} else if !validServiceGroup(group) {
			return fmt.Errorf("不支持的引擎类型: %s", command.ServiceType)
		}
	} else if !validServiceGroup(group) {
		return fmt.Errorf("不支持的服务类型: %s", command.ServiceType)
	}
	if formal && group == ServiceCollector {
		err := a.reconcileNativeCollector(ctx, command)
		if err != nil {
			a.supervisor.RecordFailure(command.ServiceID, group, command.Generation, err)
		}
		return err
	}
	if formal && engineCommand && !commandRequiresRunningRelease(command) {
		// K3s 服务的停止由中心执行；不能尝试停止不存在的本机服务组，
		// 否则会阻塞同批次中真正由 NodeAgent 管理的原生采集停止命令。
		if !strings.EqualFold(strings.TrimSpace(command.DesiredStatus), "stopped") {
			return fmt.Errorf("不支持的引擎期望状态: %s", command.DesiredStatus)
		}
		return a.saveApplied(command.ServiceID, command.Generation)
	}
	if formal && commandRequiresRunningRelease(command) {
		if err := a.installBoundRelease(ctx, command, group); err != nil {
			a.supervisor.RecordFailure(command.ServiceID, group, command.Generation, err)
			return err
		}
		if engineCommand {
			// 基础、计算和报警引擎由中心 K3s 调和器唯一启动；NodeAgent 只验证并原子准备
			// 节点制品，不能回退到旧原生 Supervisor 形成第二份活动实例。
			return a.saveApplied(command.ServiceID, command.Generation)
		}
		err := a.formalLaunchGateError(command.DeploymentID)
		a.supervisor.RecordFailure(command.ServiceID, group, command.Generation, err)
		return err
	}
	if strings.EqualFold(strings.TrimSpace(command.Operation), "delete") {
		if _, err := a.supervisor.Stop(command.ServiceID, command.Generation); err != nil {
			a.supervisor.RecordFailure(command.ServiceID, group, command.Generation, err)
			return err
		}
		if group == ServiceCollector {
			if err := removeCollectorWAL(a.cfg.DataDir, command.DeploymentID); err != nil {
				a.supervisor.RecordFailure(command.ServiceID, group, command.Generation, err)
				return err
			}
		}
		return a.saveApplied(command.ServiceID, command.Generation)
	}
	if !a.supervisor.HasService(group) {
		err := fmt.Errorf("本节点的服务组 %s 仅已安装但尚未完成本地 Release 配置", group)
		a.supervisor.RecordFailure(command.ServiceID, group, command.Generation, err)
		return err
	}
	applied := a.appliedGeneration(command.ServiceID)
	if command.Generation < applied {
		return nil
	}
	// persisted generation 只说明旧 Agent 成功处理过，不能证明重启后的本机进程仍存在。
	// 相同 generation 仅在 observed status 已满足中心 desired 时才真正幂等。
	if command.Generation == applied {
		if status, err := a.supervisor.Status(command.ServiceID); err == nil && matchesDesired(status, command) {
			return nil
		}
	}
	var reconcileErr error
	switch strings.ToLower(strings.TrimSpace(command.Operation)) {
	case "restart":
		_, reconcileErr = a.supervisor.Restart(command.ServiceID, group, command.Generation)
	case "", "deploy", "start", "stop":
		switch strings.ToLower(strings.TrimSpace(command.DesiredStatus)) {
		case "running":
			_, reconcileErr = a.supervisor.Start(command.ServiceID, group, command.Generation)
		case "stopped":
			_, reconcileErr = a.supervisor.Stop(command.ServiceID, command.Generation)
		default:
			reconcileErr = fmt.Errorf("不支持的 desiredStatus: %s", command.DesiredStatus)
		}
	default:
		reconcileErr = fmt.Errorf("不支持的 operation: %s", command.Operation)
	}
	if reconcileErr != nil {
		a.supervisor.RecordFailure(command.ServiceID, group, command.Generation, reconcileErr)
		return reconcileErr
	}
	return a.saveApplied(command.ServiceID, command.Generation)
}

func matchesDesired(status ProcessStatus, command AgentCommand) bool {
	if strings.EqualFold(command.Operation, "restart") {
		return status.State == "running" && status.Generation >= command.Generation
	}
	switch strings.ToLower(strings.TrimSpace(command.DesiredStatus)) {
	case "running":
		return status.State == "running" && status.Generation >= command.Generation
	case "stopped":
		return status.State == "stopped" && status.Generation >= command.Generation
	default:
		return false
	}
}

func (a *Agent) Heartbeat(ctx context.Context) error {
	identity := a.currentIdentity()
	if identity.NodeID == "" {
		return fmt.Errorf("节点尚未领取身份")
	}
	services := make([]map[string]any, 0)
	for _, status := range a.supervisor.List() {
		if status.WorkloadID == "" {
			continue
		}
		services = append(services, map[string]any{
			"serviceId":          status.WorkloadID,
			"observedStatus":     heartbeatStatus(status.State),
			"observedGeneration": status.Generation,
			"replicasObserved":   status.ReplicasObserved,
			"message":            status.LastError,
			"endpoint":           a.supervisor.PublicURL(status.Role),
		})
	}
	payload := map[string]any{"agentVersion": a.cfg.AgentVersion, "ipAddress": a.nodeIP(), "resourceSummary": resourceSummary(), "services": services}
	if a.hostd != nil {
		if state, err := a.hostd.TimeSyncStatus(ctx); err == nil {
			payload["resourceSummary"].(map[string]any)["timeSync"] = state
		}
		if state, err := a.hostd.Status(ctx); err == nil {
			a.mu.RLock()
			if a.clusterFailure != nil && state.ObservedState == "not-installed" {
				state = *a.clusterFailure
			}
			a.mu.RUnlock()
			// Hostd 状态还含数据目录和 systemd 单元名；这些属于本机实现细节，
			// 不进入中心的严格 Agent 协议。
			payload["clusterState"] = clusterStatePayload(state)
		}
		if states, err := a.hostd.FoundationStatuses(ctx); err == nil {
			a.mu.RLock()
			if a.foundationFailure != nil {
				states = append(states, *a.foundationFailure)
			}
			a.mu.RUnlock()
			payload["foundationStates"] = states
		}
	}
	if err := a.request(ctx, http.MethodPost, "/api/v1/ops/agent/nodes/"+identity.NodeID+"/heartbeat", identity.AgentToken, payload, nil); err != nil {
		return err
	}
	a.mu.Lock()
	if a.foundationFailure != nil && a.foundationFailure.ObservedState == "not-installed" {
		a.foundationFailure = nil
	}
	a.mu.Unlock()
	return nil
}

func clusterStatePayload(state hostd.ClusterState) map[string]any {
	return map[string]any{
		"schemaVersion": state.SchemaVersion,
		"generation":    state.Generation,
		"clusterId":     state.ClusterID,
		"nodeId":        state.NodeID,
		"operation":     state.Operation,
		"k3sVersion":    state.K3sVersion,
		"nodeName":      state.NodeName,
		"nodeIp":        state.NodeIP,
		"observedState": state.ObservedState,
		"message":       state.Message,
	}
}

// heartbeatStatus 把本机过渡态收敛到中心的三种观测状态，避免把 Supervisor
// 内部 starting/stopping 状态泄漏到控制面契约。
func heartbeatStatus(status string) string {
	switch status {
	case "running", "starting":
		return "running"
	case "failed":
		return "failed"
	default:
		return "stopped"
	}
}

func (a *Agent) Commands(ctx context.Context) ([]AgentCommand, error) {
	identity := a.currentIdentity()
	if identity.NodeID == "" {
		return nil, fmt.Errorf("节点尚未领取身份")
	}
	var raw json.RawMessage
	if err := a.request(ctx, http.MethodGet, "/api/v1/ops/agent/nodes/"+identity.NodeID+"/commands", identity.AgentToken, nil, &raw); err != nil {
		return nil, err
	}
	var response struct {
		Commands         []AgentCommand                 `json:"commands"`
		ClusterPlan      *hostd.ClusterPlan             `json:"clusterPlan"`
		ClusterUninstall *hostd.UninstallRequest        `json:"clusterUninstall"`
		FoundationPlan   *hostd.FoundationPlan          `json:"foundationPlan"`
		FoundationDelete *hostd.FoundationDeleteRequest `json:"foundationDelete"`
		TimeSyncPlan     *hostd.TimeSyncPlan            `json:"timeSyncPlan"`
	}
	if err := strictDecodeJSON(raw, &response); err != nil {
		return nil, fmt.Errorf("中心命令响应无效: %w", err)
	}
	if response.ClusterUninstall != nil {
		if a.hostd == nil {
			return nil, fmt.Errorf("中心下发了集群卸载计划，但本节点未配置 hostd")
		}
		request := *response.ClusterUninstall
		state, err := a.hostd.Uninstall(ctx, request)
		if state.SchemaVersion == "" {
			state.SchemaVersion = "induforge.cluster-state.v1"
		}
		if state.ClusterID == "" {
			state.ClusterID = request.ClusterID
		}
		if state.NodeID == "" {
			state.NodeID = request.NodeID
		}
		a.mu.Lock()
		a.clusterFailure = &state
		a.mu.Unlock()
		if err != nil {
			state.ObservedState = "failed"
			state.Message = truncateAgentMessage(err.Error())
			a.mu.Lock()
			a.clusterFailure = &state
			a.mu.Unlock()
			return nil, fmt.Errorf("卸载 K3s 运行底座失败: %w", err)
		}
	}
	if response.ClusterPlan != nil {
		if a.hostd == nil {
			return nil, fmt.Errorf("中心下发了集群计划，但本节点未配置 hostd")
		}
		plan := *response.ClusterPlan
		plan.DataDir = filepath.Clean(strings.TrimSpace(a.cfg.HostDataDir))
		if _, err := a.hostd.Apply(ctx, plan); err != nil {
			a.mu.Lock()
			a.clusterFailure = &hostd.ClusterState{SchemaVersion: "induforge.cluster-state.v1", Generation: plan.Generation, ClusterID: plan.ClusterID, NodeID: plan.NodeID, Operation: plan.Operation, K3sVersion: plan.K3sVersion, NodeIP: plan.NodeIP, ObservedState: "failed", Message: truncateAgentMessage(err.Error())}
			a.mu.Unlock()
			return nil, fmt.Errorf("应用 K3s 集群计划失败: %w", err)
		}
		a.mu.Lock()
		a.clusterFailure = nil
		a.mu.Unlock()
	}
	if response.TimeSyncPlan != nil {
		if a.hostd == nil {
			return nil, fmt.Errorf("中心下发了时间同步计划，但本节点未配置 hostd")
		}
		// 时间同步异常由心跳状态上报并在运维页告警，不能阻断 K3s、基础服务
		// 或工程命令的正常收敛，否则 Chrony 故障会放大为节点完全失管。
		_, _ = a.hostd.ApplyTimeSync(ctx, *response.TimeSyncPlan)
	}
	if response.FoundationPlan != nil {
		if a.hostd == nil {
			return nil, fmt.Errorf("中心下发了基础服务计划，但本节点未配置 hostd")
		}
		if _, err := a.hostd.ApplyFoundation(ctx, *response.FoundationPlan); err != nil {
			plan := response.FoundationPlan
			a.mu.Lock()
			a.foundationFailure = &hostd.FoundationState{SchemaVersion: "induforge.foundation-state.v1", Generation: plan.Generation, EnvironmentID: plan.EnvironmentID, NodeID: plan.NodeID, ObservedState: "failed", Message: truncateAgentMessage(err.Error())}
			a.mu.Unlock()
			return nil, fmt.Errorf("应用基础服务计划失败: %w", err)
		}
		a.mu.Lock()
		a.foundationFailure = nil
		a.mu.Unlock()
	}
	if response.FoundationDelete != nil {
		if a.hostd == nil {
			return nil, fmt.Errorf("中心下发了环境清理计划，但本节点未配置 hostd")
		}
		state, err := a.hostd.DeleteFoundation(ctx, *response.FoundationDelete)
		a.mu.Lock()
		a.foundationFailure = &state
		a.mu.Unlock()
		if err != nil {
			state.ObservedState = "failed"
			state.Message = truncateAgentMessage(err.Error())
			a.mu.Lock()
			a.foundationFailure = &state
			a.mu.Unlock()
			return nil, fmt.Errorf("删除运行环境资源失败: %w", err)
		}
	}
	return response.Commands, nil
}

func truncateAgentMessage(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 1024 {
		return value[:1024]
	}
	return value
}

// discoverNodeIP 通过到中心地址的 UDP 路由选择获得节点对外地址；不发送数据，
// 避免把 SSH 隧道或反向代理的 RemoteAddr 错当成节点间通信地址。
func discoverNodeIP(serverURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(serverURL))
	if err != nil || parsed.Hostname() == "" {
		return ""
	}
	port := parsed.Port()
	if port == "" {
		port = "443"
	}
	connection, err := net.DialTimeout("udp", net.JoinHostPort(parsed.Hostname(), port), 2*time.Second)
	if err != nil {
		return ""
	}
	defer connection.Close()
	address, ok := connection.LocalAddr().(*net.UDPAddr)
	if !ok || address.IP == nil || address.IP.IsLoopback() || address.IP.IsUnspecified() {
		return ""
	}
	return address.IP.String()
}

func (a *Agent) nodeIP() string {
	if ip := net.ParseIP(strings.TrimSpace(a.cfg.NodeIP)); ip != nil && !ip.IsLoopback() && !ip.IsUnspecified() && !ip.IsMulticast() {
		return ip.String()
	}
	return discoverNodeIP(a.cfg.ServerURL)
}

func (a *Agent) request(ctx context.Context, method, path, token string, payload any, destination any) error {
	return a.requestLimited(ctx, method, path, token, payload, destination, maxCenterResponseBytes)
}
func (a *Agent) requestLimited(ctx context.Context, method, path, token string, payload any, destination any, responseLimit int) error {
	var body *bytes.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	} else {
		body = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(a.cfg.ServerURL, "/")+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusMultipleChoices && response.StatusCode < http.StatusBadRequest {
		return fmt.Errorf("中心请求不允许重定向: status=%d", response.StatusCode)
	}
	var envelope struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, int64(responseLimit)+1))
	if err != nil {
		return fmt.Errorf("读取中心响应失败: %w", err)
	}
	if len(raw) > responseLimit {
		return fmt.Errorf("中心响应超过 %d 字节上限", responseLimit)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&envelope); err != nil {
		return fmt.Errorf("解析中心响应失败: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("中心响应包含尾随 JSON")
		}
		return fmt.Errorf("中心响应包含尾随 JSON: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || envelope.Code != 0 {
		return fmt.Errorf("中心请求失败: status=%d code=%d msg=%s", response.StatusCode, envelope.Code, envelope.Msg)
	}
	if destination != nil && len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, destination); err != nil {
			return err
		}
	}
	return nil
}

func newCenterHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		// Agent token 只可发给配置的 Center。即使是同主机 redirect 也不自动跟随，
		// 避免以后引入相对/跨主机 Location 时泄露凭据。
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func parseTrustedSigningKeys(entries []TrustedSigningKey) (map[string]ed25519.PublicKey, error) {
	keys := make(map[string]ed25519.PublicKey, len(entries))
	for _, entry := range entries {
		keyID := strings.TrimSpace(entry.KeyID)
		if !validStableID(keyID) {
			return nil, fmt.Errorf("本地 trust keyId 非法: %q", entry.KeyID)
		}
		if _, exists := keys[keyID]; exists {
			return nil, fmt.Errorf("本地 trust keyId 重复: %s", keyID)
		}
		encoded := strings.TrimSpace(entry.PublicKey)
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil || len(decoded) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("本地 trust key %s 不是 base64 Ed25519 公钥", keyID)
		}
		keys[keyID] = ed25519.PublicKey(append([]byte(nil), decoded...))
	}
	return keys, nil
}

// validateServerURL 避免 enrollment code 和 Agent token 经远端明文 HTTP 外泄。
// 本机开发/同机部署允许 loopback HTTP；所有其他 Center 必须使用 HTTPS。
func validateServerURL(raw string) error {
	value := strings.TrimSpace(raw)
	if value == "" {
		return fmt.Errorf("ops serverUrl 不能为空")
	}
	parsed, err := url.Parse(value)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" || parsed.Opaque != "" {
		return fmt.Errorf("ops serverUrl 必须是绝对 URL")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("ops serverUrl 不允许 userinfo、query 或 fragment")
	}
	if path := parsed.EscapedPath(); path != "" && path != "/" {
		return fmt.Errorf("ops serverUrl 只能使用根路径")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "https":
		return nil
	case "http":
		if isLoopbackCenterHost(parsed.Hostname()) {
			return nil
		}
		return fmt.Errorf("ops serverUrl 仅本机 loopback 可使用 http，其余 Center 必须使用 https")
	default:
		return fmt.Errorf("ops serverUrl 只允许 https，或本机 loopback 的 http")
	}
}

func isLoopbackCenterHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func resourceSummary() map[string]any {
	result := map[string]any{"cpu": map[string]any{"count": runtime.NumCPU()}, "memory": map[string]any{}, "disk": map[string]any{}}
	if cpuPercent, err := cpu.Percent(0, false); err == nil && len(cpuPercent) > 0 {
		result["cpu"].(map[string]any)["usedPercent"] = cpuPercent[0]
	}
	if virtualMemory, err := mem.VirtualMemory(); err == nil {
		result["memory"].(map[string]any)["totalBytes"] = virtualMemory.Total
		result["memory"].(map[string]any)["usedPercent"] = virtualMemory.UsedPercent
	}
	if usage, err := disk.Usage("."); err == nil {
		result["disk"].(map[string]any)["totalBytes"] = usage.Total
		result["disk"].(map[string]any)["usedPercent"] = usage.UsedPercent
	}
	return result
}
