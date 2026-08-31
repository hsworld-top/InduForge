package ops

import (
	"bytes"
	"context"
	"encoding/json"
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
	AgentVersion        string        `yaml:"agentVersion"`
	HeartbeatEvery      time.Duration `yaml:"heartbeatEvery"`
	DataDir             string        `yaml:"dataDir"`
	ClearEnrollmentCode func() error  `yaml:"-"`
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
}

type AgentCommand struct {
	NodeID          string `json:"nodeId"`
	RunID           string `json:"runId"`
	DeploymentID    string `json:"deploymentId"`
	ServiceID       string `json:"serviceId"`
	ServiceType     string `json:"serviceType"`
	Operation       string `json:"operation"`
	DesiredStatus   string `json:"desiredStatus"`
	Generation      int64  `json:"generation"`
	Version         string `json:"version"`
	ReplicasDesired int    `json:"replicasDesired"`
}

// Agent 将中心 desired 状态与本机 observed 状态分开处理。中心只能选择本机
// 声明的服务组，不能传递命令、路径、参数或环境变量。
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
	return &Agent{cfg: cfg, supervisor: supervisor, client: &http.Client{Timeout: 8 * time.Second}, applied: make(map[string]int64)}, nil
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
	return HostInfo{Hostname: hostname, OS: runtime.GOOS, Architecture: runtime.GOARCH, MachineFingerprint: utils.GetMachineID()}
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
	for _, command := range commands {
		_ = a.Reconcile(command)
	}
}

func (a *Agent) Reconcile(command AgentCommand) error {
	if command.ServiceID == "" {
		return fmt.Errorf("serviceId 不能为空")
	}
	identity := a.currentIdentity()
	if command.NodeID == "" || command.NodeID != identity.NodeID {
		return fmt.Errorf("服务未显式分配给本节点")
	}
	group := ServiceGroup(command.ServiceType)
	if !validServiceGroup(group) {
		return fmt.Errorf("不支持的服务类型: %s", command.ServiceType)
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
	var err error
	switch strings.ToLower(strings.TrimSpace(command.Operation)) {
	case "restart":
		_, err = a.supervisor.Restart(command.ServiceID, group, command.Generation)
	case "", "deploy", "start", "stop":
		switch strings.ToLower(strings.TrimSpace(command.DesiredStatus)) {
		case "running":
			_, err = a.supervisor.Start(command.ServiceID, group, command.Generation)
		case "stopped":
			_, err = a.supervisor.Stop(command.ServiceID, command.Generation)
		default:
			err = fmt.Errorf("不支持的 desiredStatus: %s", command.DesiredStatus)
		}
	default:
		err = fmt.Errorf("不支持的 operation: %s", command.Operation)
	}
	if err != nil {
		a.supervisor.RecordFailure(command.ServiceID, group, command.Generation, err)
		return err
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
	payload := map[string]any{"agentVersion": a.cfg.AgentVersion, "resourceSummary": resourceSummary(), "services": services}
	return a.request(ctx, http.MethodPost, "/api/v1/ops/agent/nodes/"+identity.NodeID+"/heartbeat", identity.AgentToken, payload, nil)
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
	var response struct {
		Commands []AgentCommand `json:"commands"`
	}
	if err := a.request(ctx, http.MethodGet, "/api/v1/ops/agent/nodes/"+identity.NodeID+"/commands", identity.AgentToken, nil, &response); err != nil {
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
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxCenterResponseBytes+1))
	if err != nil {
		return fmt.Errorf("读取中心响应失败: %w", err)
	}
	if len(raw) > maxCenterResponseBytes {
		return fmt.Errorf("中心响应超过 %d 字节上限", maxCenterResponseBytes)
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
