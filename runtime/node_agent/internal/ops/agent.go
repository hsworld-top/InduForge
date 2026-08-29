package ops

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/indu-forge/node_agent/internal/pkg/utils"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

// Config 是 ops Agent 的最小持久配置。enrollmentCode 只在未领取身份时使用，
// 成功后只保存 node id/token，不会把 enrollment code 长期写入磁盘。
type Config struct {
	Enabled             bool          `yaml:"enabled"`
	ServerURL           string        `yaml:"serverUrl"`
	EnrollmentCode      string        `yaml:"enrollmentCode"`
	Role                string        `yaml:"role"`
	HeartbeatEvery      time.Duration `yaml:"heartbeatEvery"`
	DataDir             string        `yaml:"dataDir"`
	DemoRuntime         bool          `yaml:"demoRuntime"`
	ClearEnrollmentCode func() error  `yaml:"-"`
}

// Identity 是领取成功后的设备凭据，文件权限只允许服务账户读取。
type Identity struct {
	HostNodeID      string `json:"hostNodeId"`
	AgentToken      string `json:"agentToken"`
	PendingApproval bool   `json:"pendingApproval"`
}

type HostInfo struct {
	DisplayName        string `json:"displayName,omitempty"`
	Hostname           string `json:"hostname"`
	OS                 string `json:"os"`
	Architecture       string `json:"architecture"`
	MachineFingerprint string `json:"machineFingerprint"`
}

type DesiredWorkload struct {
	CommandID     string `json:"commandId"`
	RunID         string `json:"runId"`
	DeploymentID  string `json:"deploymentId"`
	WorkloadID    string `json:"workloadId"`
	Role          string `json:"role"`
	Operation     string `json:"operation"`
	DesiredStatus string `json:"desiredStatus"`
	Generation    int64  `json:"generation"`
	Version       string `json:"version,omitempty"`
}

// Agent 将中心 desired 状态与本机 observed 状态分开处理。生产中的 compute/alert
// 由 RuntimeCluster/site-controller 调和；DemoRuntime 仅供本轮安装包验收，允许
// runtime_linux 节点真实托管 compute/alert 小型进程。
type Agent struct {
	cfg        Config
	identity   Identity
	supervisor *Supervisor
	client     *http.Client
	mu         sync.RWMutex
	applied    map[string]int64
}

func NewAgent(cfg Config, supervisor *Supervisor) (*Agent, error) {
	if supervisor == nil {
		return nil, fmt.Errorf("supervisor 不能为空")
	}
	if cfg.HeartbeatEvery <= 0 {
		cfg.HeartbeatEvery = 10 * time.Second
	}
	if cfg.DataDir == "" {
		cfg.DataDir = "./data"
	}
	if cfg.Role == "" {
		cfg.Role = "collector_linux"
	}
	if !validNodeRole(cfg.Role) {
		return nil, fmt.Errorf("不支持的节点角色: %s", cfg.Role)
	}
	return &Agent{cfg: cfg, supervisor: supervisor, client: &http.Client{Timeout: 8 * time.Second}, applied: make(map[string]int64)}, nil
}

func validNodeRole(role string) bool {
	return role == "runtime_linux" || role == "collector_linux" || role == "collector_windows"
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
	if identity.HostNodeID == "" || identity.AgentToken == "" {
		return fmt.Errorf("节点身份不完整")
	}
	a.mu.Lock()
	a.identity = identity
	a.mu.Unlock()
	return nil
}

func (a *Agent) saveIdentity(identity Identity) error {
	if identity.HostNodeID == "" || identity.AgentToken == "" {
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

func (a *Agent) appliedGeneration(workloadID string) int64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.applied[workloadID]
}
func (a *Agent) saveApplied(workloadID string, generation int64) error {
	a.mu.Lock()
	if generation <= a.applied[workloadID] {
		a.mu.Unlock()
		return nil
	}
	a.applied[workloadID] = generation
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
	payload := map[string]any{"code": a.cfg.EnrollmentCode, "host": a.hostInfo(), "agent": map[string]any{"version": "1.0.0", "role": a.cfg.Role, "capabilities": map[string]bool{"demoRuntime": a.cfg.DemoRuntime}}}
	var result struct {
		HostNode struct {
			ID string `json:"id"`
		} `json:"hostNode"`
		AgentToken      string `json:"agentToken"`
		PendingApproval bool   `json:"pendingApproval"`
	}
	if err := a.request(ctx, http.MethodPost, "/api/v1/ops/agent/enrollments/claim", "", payload, &result); err != nil {
		return err
	}
	if err := a.saveIdentity(Identity{HostNodeID: result.HostNode.ID, AgentToken: result.AgentToken, PendingApproval: result.PendingApproval}); err != nil {
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
	return HostInfo{Hostname: hostname, OS: runtime.GOOS, Architecture: runtime.GOARCH, MachineFingerprint: utils.GetMachineID()}
}

// Run 首先加载或领取身份；之后立即心跳并按固定周期拉取 desired workload。任何单次网络
// 失败只记录为本轮失败，不影响已经运行的采集进程。
func (a *Agent) Run(ctx context.Context) error {
	if !a.cfg.Enabled {
		return nil
	}
	if strings.TrimSpace(a.cfg.ServerURL) == "" {
		return fmt.Errorf("ops serverUrl 不能为空")
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
	if a.currentIdentity().HostNodeID == "" {
		// 控制面可能比 Agent 晚就绪；领取失败留待下一周期重试，不影响本机已运行进程。
		_ = a.claim(ctx)
		return
	}
	// 身份已经落盘但上一次清除引导码失败时，每轮继续重试；在明文接入码
	// 清理成功前不进入日常心跳，避免一次瞬时文件错误把敏感码永久遗留。
	if strings.TrimSpace(a.cfg.EnrollmentCode) != "" && a.cfg.ClearEnrollmentCode != nil {
		if err := a.cfg.ClearEnrollmentCode(); err != nil {
			return
		}
		a.cfg.EnrollmentCode = ""
	}
	if err := a.Heartbeat(ctx); err != nil {
		return
	}
	commands, err := a.Commands(ctx)
	if err != nil {
		return
	}
	for _, workload := range commands {
		_ = a.Reconcile(workload)
	}
}

func (a *Agent) Reconcile(workload DesiredWorkload) error {
	if workload.WorkloadID == "" {
		return nil
	}
	role := WorkloadRole(workload.Role)
	if !validRole(role) {
		return fmt.Errorf("不支持的 workload role: %s", workload.Role)
	}
	if !a.canManage(role) {
		return nil
	}
	applied := a.appliedGeneration(workload.WorkloadID)
	if workload.Generation < applied {
		return nil
	}
	// persisted generation 只说明旧 Agent 成功处理过，不能证明重启后的本机进程仍存在。
	// 相同 generation 仅在 observed status 已满足中心 desired 时才真正幂等。
	if workload.Generation == applied {
		if status, err := a.supervisor.Status(workload.WorkloadID); err == nil && matchesDesired(status, workload) {
			return nil
		}
	}
	var err error
	switch strings.ToLower(strings.TrimSpace(workload.Operation)) {
	case "restart":
		_, err = a.supervisor.Restart(workload.WorkloadID, role, workload.Generation)
	case "", "deploy", "start", "stop":
		// deploy 是首次落地命令；当前 demo 无独立制品安装步骤，故按 desiredStatus
		// 调和，通常等价于启动一个此前不存在的本机工作负载实例。
		switch strings.ToLower(strings.TrimSpace(workload.DesiredStatus)) {
		case "running":
			_, err = a.supervisor.Start(workload.WorkloadID, role, workload.Generation)
		case "stopped":
			_, err = a.supervisor.Stop(workload.WorkloadID, workload.Generation)
		default:
			err = fmt.Errorf("不支持的 desiredStatus: %s", workload.DesiredStatus)
		}
	default:
		err = fmt.Errorf("不支持的 operation: %s", workload.Operation)
	}
	if err != nil {
		a.supervisor.RecordFailure(workload.WorkloadID, role, workload.Generation, err)
		return err
	}
	return a.saveApplied(workload.WorkloadID, workload.Generation)
}

func matchesDesired(status ProcessStatus, workload DesiredWorkload) bool {
	if strings.EqualFold(workload.Operation, "restart") {
		return status.State == "running" && status.Generation >= workload.Generation
	}
	switch strings.ToLower(strings.TrimSpace(workload.DesiredStatus)) {
	case "running":
		return status.State == "running" && status.Generation >= workload.Generation
	case "stopped":
		return status.State == "stopped" && status.Generation >= workload.Generation
	default:
		return false
	}
}

func (a *Agent) canManage(role WorkloadRole) bool {
	if role == WorkloadCollector {
		return a.cfg.Role == "collector_linux" || a.cfg.Role == "collector_windows"
	}
	return a.cfg.DemoRuntime && a.cfg.Role == "runtime_linux" && (role == WorkloadCompute || role == WorkloadAlert)
}

func (a *Agent) Heartbeat(ctx context.Context) error {
	identity := a.currentIdentity()
	if identity.HostNodeID == "" {
		return fmt.Errorf("节点尚未领取身份")
	}
	payload := map[string]any{"agentVersion": "1.0.0", "resourceSummary": resourceSummary(), "observedState": map[string]any{"status": "online", "observedAt": time.Now().UTC().Format(time.RFC3339), "workloads": a.supervisor.List()}}
	return a.request(ctx, http.MethodPost, "/api/v1/ops/agent/host-nodes/"+identity.HostNodeID+"/heartbeat", identity.AgentToken, payload, nil)
}

func (a *Agent) Commands(ctx context.Context) ([]DesiredWorkload, error) {
	identity := a.currentIdentity()
	if identity.HostNodeID == "" {
		return nil, fmt.Errorf("节点尚未领取身份")
	}
	var response struct {
		Commands []DesiredWorkload `json:"commands"`
	}
	if err := a.request(ctx, http.MethodGet, "/api/v1/ops/agent/host-nodes/"+identity.HostNodeID+"/commands", identity.AgentToken, nil, &response); err != nil {
		return nil, err
	}
	return response.Commands, nil
}

func (a *Agent) request(ctx context.Context, method, path, token string, payload any, destination any) error {
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
	var envelope struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("解析中心响应失败: %w", err)
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
