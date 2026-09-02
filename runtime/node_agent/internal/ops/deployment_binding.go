package ops

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

const (
	deploymentBindingSchema = "deployment-binding.v1"
	engineBindingSchema     = "deployment-binding.v2"
	// 默认值只用于示例和测试；正式 Binding 以客户在部署阶段选择的访问端口为准。
	gatewayPublicPort      = 17800
	runtimeAPILoopbackPort = 17801
	engineLoopbackPort     = 17802
	collectorHealthPort    = 17803
)

// deploymentBinding 是唯一可以将节点、部署、Release、端口和服务类型关联起来的
// 中心 JSON 快照。它没有命令行、路径、环境变量或下载 URL 字段。
type deploymentBinding struct {
	SchemaVersion   string   `json:"schemaVersion"`
	BindingID       string   `json:"bindingId"`
	Revision        int      `json:"revision"`
	NodeID          string   `json:"nodeId"`
	DeploymentID    string   `json:"deploymentId"`
	ProjectID       string   `json:"projectId"`
	EnvironmentID   string   `json:"environmentId,omitempty"`
	ServiceID       string   `json:"serviceId,omitempty"`
	Mode            string   `json:"mode,omitempty"`
	Engine          string   `json:"engine,omitempty"`
	EnabledServices []string `json:"enabledServices"`
	Ports           struct {
		HostPort                *int `json:"hostPort,omitempty"`
		GatewayPublic           int  `json:"gatewayPublic"`
		RuntimeAPILoopback      int  `json:"runtimeApiLoopback"`
		EngineLoopback          int  `json:"engineLoopback"`
		CollectorHealthLoopback *int `json:"collectorHealthLoopback,omitempty"`
	} `json:"ports"`
	Release struct {
		ID              string `json:"id"`
		ArchiveSHA256   string `json:"archiveSha256"`
		ManifestSHA256  string `json:"manifestSha256"`
		ChecksumsSHA256 string `json:"checksumsSha256"`
		SigningKeyID    string `json:"signingKeyId"`
	} `json:"release"`
	Secrets   []bindingSecret `json:"secrets"`
	IssuedAt  string          `json:"issuedAt"`
	ExpiresAt string          `json:"expiresAt,omitempty"`
}

type bindingSecret struct {
	Name          string `json:"name"`
	SHA256        string `json:"sha256"`
	SchemaVersion string `json:"schemaVersion"`
}

// commandUsesFormalRelease prevents a partially populated new command from
// silently falling back to the legacy static supervisor path.
func commandUsesFormalRelease(command AgentCommand) (bool, error) {
	values := []string{
		command.DeploymentID, command.ReleaseID, command.ArchiveSHA256,
		command.ManifestSHA256, command.ChecksumsSHA256, command.SigningKeyID,
	}
	formal := command.BindingRevision != 0
	for _, value := range values {
		formal = formal || strings.TrimSpace(value) != ""
	}
	if !formal {
		return false, nil
	}
	if !validRuntimeUUID(command.NodeID) || !validRuntimeUUID(command.DeploymentID) || !validRuntimeUUID(command.ReleaseID) || !validRuntimeUUID(command.ServiceID) ||
		!validSHA256(command.ArchiveSHA256) || !validSHA256(command.ManifestSHA256) ||
		!validSHA256(command.ChecksumsSHA256) || !validStableID(command.SigningKeyID) ||
		command.BindingRevision < 1 || command.Generation < 1 || command.ReplicasDesired != 1 || !validVersionValue(command.Version) {
		return true, fmt.Errorf("中心正式 Release 命令字段不完整或非法")
	}
	return true, nil
}

func commandRequiresRunningRelease(command AgentCommand) bool {
	operation := strings.ToLower(strings.TrimSpace(command.Operation))
	return operation == "restart" || operation == "deploy" || strings.EqualFold(strings.TrimSpace(command.DesiredStatus), "running")
}

// fetchDeploymentBinding 只能用当前 Agent 身份请求自身 nodeId 下的部署 Binding。
func (a *Agent) fetchDeploymentBinding(ctx context.Context, command AgentCommand) (deploymentBinding, error) {
	identity := a.currentIdentity()
	if identity.NodeID == "" || identity.AgentToken == "" {
		return deploymentBinding{}, fmt.Errorf("节点尚未领取身份")
	}
	var raw json.RawMessage
	path := "/api/v1/ops/agent/nodes/" + identity.NodeID + "/deployments/" + command.DeploymentID + "/binding?serviceId=" + command.ServiceID
	if err := a.request(ctx, http.MethodGet, path, identity.AgentToken, nil, &raw); err != nil {
		return deploymentBinding{}, err
	}
	var binding deploymentBinding
	if err := strictDecodeJSON(raw, &binding); err != nil {
		return deploymentBinding{}, fmt.Errorf("DeploymentBinding JSON 无效: %w", err)
	}
	if err := validateDeploymentBinding(binding, command, identity.NodeID, a.supervisor); err != nil {
		return deploymentBinding{}, err
	}
	if binding.SchemaVersion == engineBindingSchema {
		binding.EnabledServices = engineBindingCapabilities(binding.Engine)
	}
	return binding, nil
}

// downloadDeploymentRelease 只接收 Center 同源端点返回的二进制流；不解释 JSON、
// objectKey 或外部下载 URL，也不跟随任何 redirect。
func (a *Agent) downloadDeploymentRelease(ctx context.Context, deploymentID, serviceID string) (io.ReadCloser, error) {
	identity := a.currentIdentity()
	if identity.NodeID == "" || identity.AgentToken == "" {
		return nil, fmt.Errorf("节点尚未领取身份")
	}
	path := "/api/v1/ops/agent/nodes/" + identity.NodeID + "/deployments/" + deploymentID + "/release?serviceId=" + serviceID
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(a.cfg.ServerURL, "/")+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/zstd")
	req.Header.Set("Authorization", "Bearer "+identity.AgentToken)
	response, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= http.StatusMultipleChoices && response.StatusCode < http.StatusBadRequest {
		response.Body.Close()
		return nil, fmt.Errorf("Release 下载不允许重定向: status=%d", response.StatusCode)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		defer response.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(response.Body, maxCenterResponseBytes+1))
		return nil, fmt.Errorf("Release 下载失败: status=%d body=%q", response.StatusCode, string(body))
	}
	contentType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || !strings.EqualFold(contentType, "application/zstd") {
		response.Body.Close()
		return nil, fmt.Errorf("Release 下载 Content-Type 必须为 application/zstd")
	}
	if response.ContentLength <= 0 || response.ContentLength > DefaultReleaseInstallLimits().MaxArchiveBytes {
		response.Body.Close()
		return nil, fmt.Errorf("Release 下载 Content-Length 非法: %d", response.ContentLength)
	}
	return response.Body, nil
}

func (a *Agent) installBoundRelease(ctx context.Context, command AgentCommand, group ServiceGroup) error {
	binding, err := a.fetchDeploymentBinding(ctx, command)
	if err != nil {
		return err
	}
	if !bindingEnablesService(binding, group) {
		return fmt.Errorf("DeploymentBinding 未显式启用服务组: %s", group)
	}
	key, exists := a.trustKeys[binding.Release.SigningKeyID]
	if !exists {
		return fmt.Errorf("本地 trust store 不信任 signingKeyId: %s", binding.Release.SigningKeyID)
	}
	bundle, err := a.downloadDeploymentRelease(ctx, command.DeploymentID, command.ServiceID)
	if err != nil {
		return err
	}
	defer bundle.Close()

	// deploymentId 已按 UUID 严格验证。目录由 Agent 自行派生，中心不能传入路径。
	store, err := NewReleaseStore(filepath.Join(a.cfg.DataDir, "deployments", command.DeploymentID, "release"))
	if err != nil {
		return err
	}
	installer, err := NewReleaseInstaller(store, DefaultReleaseInstallLimits())
	if err != nil {
		return err
	}
	boundServices := binding.EnabledServices
	if binding.SchemaVersion == engineBindingSchema {
		// v2 每条 Binding 只描述一个引擎，但同一 Release 仍依赖入口和数据运行
		// 能力；按引擎派生固定能力集合，不能把空的 v1 enabledServices 误传下去。
		boundServices = engineBindingCapabilities(binding.Engine)
	}
	_, err = installer.Install(ReleaseInstallInput{
		Archive:                 bundle,
		ExpectedOuterSHA256:     binding.Release.ArchiveSHA256,
		ExpectedManifestSHA256:  binding.Release.ManifestSHA256,
		ExpectedChecksumsSHA256: binding.Release.ChecksumsSHA256,
		ExpectedProjectID:       binding.ProjectID,
		ExpectedReleaseID:       binding.Release.ID,
		ExpectedVersion:         command.Version,
		NodeAgentVersion:        a.cfg.AgentVersion,
		RuntimeVersion:          a.cfg.RuntimeVersion,
		NodeCapabilities:        a.supervisor.Capabilities(),
		BoundServices:           boundServices,
		Engine:                  binding.Engine,
		VerificationPublicKey:   ed25519.PublicKey(key),
		KeyID:                   binding.Release.SigningKeyID,
	})
	if err != nil {
		return err
	}
	if bindingEnablesService(binding, ServiceCollector) {
		_, err = ensureCollectorWAL(a.cfg.DataDir, command.DeploymentID)
	}
	return err
}

func validateDeploymentBinding(binding deploymentBinding, command AgentCommand, nodeID string, supervisor *Supervisor) error {
	if binding.SchemaVersion == engineBindingSchema {
		return validateEngineBinding(binding, command, nodeID, supervisor)
	}
	if binding.SchemaVersion != deploymentBindingSchema || !validRuntimeUUID(binding.BindingID) ||
		binding.Revision < 1 || !validRuntimeUUID(binding.NodeID) || !validRuntimeUUID(binding.DeploymentID) ||
		!validRuntimeUUID(binding.ProjectID) || !validRuntimeUUID(binding.Release.ID) {
		return fmt.Errorf("DeploymentBinding 身份字段无效")
	}
	if binding.NodeID != nodeID || binding.DeploymentID != command.DeploymentID || binding.Revision != command.BindingRevision ||
		binding.Release.ID != command.ReleaseID || binding.Release.ArchiveSHA256 != command.ArchiveSHA256 ||
		binding.Release.ManifestSHA256 != command.ManifestSHA256 || binding.Release.ChecksumsSHA256 != command.ChecksumsSHA256 ||
		binding.Release.SigningKeyID != command.SigningKeyID {
		return fmt.Errorf("DeploymentBinding 与当前节点、部署、revision 或 Release 命令不匹配")
	}
	if !validSHA256(binding.Release.ArchiveSHA256) || !validSHA256(binding.Release.ManifestSHA256) ||
		!validSHA256(binding.Release.ChecksumsSHA256) || !validStableID(binding.Release.SigningKeyID) {
		return fmt.Errorf("DeploymentBinding Release 字段无效")
	}
	if !validDeploymentPorts(binding) || len(binding.Secrets) != 0 {
		return fmt.Errorf("DeploymentBinding 违反当前端口或 Secret 安全基线")
	}
	if _, err := parseBindingTime(binding.IssuedAt); err != nil {
		return fmt.Errorf("DeploymentBinding issuedAt 无效: %w", err)
	}
	if binding.ExpiresAt != "" {
		expiresAt, err := parseBindingTime(binding.ExpiresAt)
		if err != nil || !expiresAt.After(time.Now().UTC()) {
			return fmt.Errorf("DeploymentBinding expiresAt 无效或已过期")
		}
	}
	services, err := validatedBindingServices(binding.EnabledServices, binding.Ports.CollectorHealthLoopback)
	if err != nil {
		return err
	}
	for group := range services {
		if !supervisor.HasInstalledService(group) {
			return fmt.Errorf("本机未安装 DeploymentBinding 所需服务组: %s", group)
		}
	}
	return nil
}

func validateEngineBinding(binding deploymentBinding, command AgentCommand, nodeID string, supervisor *Supervisor) error {
	if !validRuntimeUUID(binding.BindingID) || binding.Revision < 1 || !validRuntimeUUID(binding.NodeID) ||
		!validRuntimeUUID(binding.DeploymentID) || !validRuntimeUUID(binding.ProjectID) ||
		!validRuntimeUUID(binding.EnvironmentID) || !validRuntimeUUID(binding.ServiceID) || !validRuntimeUUID(binding.Release.ID) {
		return fmt.Errorf("引擎 DeploymentBinding 身份字段无效")
	}
	if binding.NodeID != nodeID || binding.DeploymentID != command.DeploymentID || binding.ServiceID != command.ServiceID ||
		binding.Engine != command.ServiceType || binding.Revision != command.BindingRevision || binding.Release.ID != command.ReleaseID ||
		binding.Release.ArchiveSHA256 != command.ArchiveSHA256 || binding.Release.ManifestSHA256 != command.ManifestSHA256 ||
		binding.Release.ChecksumsSHA256 != command.ChecksumsSHA256 || binding.Release.SigningKeyID != command.SigningKeyID {
		return fmt.Errorf("引擎 DeploymentBinding 与当前节点、部署、服务或 Release 命令不匹配")
	}
	if binding.Mode != "development" && binding.Mode != "release" {
		return fmt.Errorf("引擎 DeploymentBinding 模式无效")
	}
	group, ok := engineServiceGroup(binding.Engine)
	if !ok || !supervisor.HasInstalledService(group) {
		return fmt.Errorf("本机未安装引擎 %s 所需服务能力", binding.Engine)
	}
	if !validSHA256(binding.Release.ArchiveSHA256) || !validSHA256(binding.Release.ManifestSHA256) ||
		!validSHA256(binding.Release.ChecksumsSHA256) || !validStableID(binding.Release.SigningKeyID) || len(binding.Secrets) != 0 {
		return fmt.Errorf("引擎 DeploymentBinding Release 或 Secret 字段无效")
	}
	if binding.Engine == "base" {
		if binding.Ports.HostPort == nil || *binding.Ports.HostPort < 1024 || *binding.Ports.HostPort > 65532 {
			return fmt.Errorf("基础引擎 DeploymentBinding 端口无效")
		}
	} else if binding.Ports.HostPort != nil {
		return fmt.Errorf("非基础引擎 DeploymentBinding 不能携带主机端口")
	}
	if _, err := parseBindingTime(binding.IssuedAt); err != nil {
		return fmt.Errorf("引擎 DeploymentBinding issuedAt 无效: %w", err)
	}
	return nil
}

func engineServiceGroup(engine string) (ServiceGroup, bool) {
	switch engine {
	case "base":
		return ServiceProjectEntry, true
	case "compute", "alarm":
		return ServiceDataRuntime, true
	case "collector":
		return ServiceCollector, true
	default:
		return "", false
	}
}

func engineBindingCapabilities(engine string) []string {
	values := []string{string(ServiceProjectEntry), string(ServiceDataRuntime)}
	if engine == "collector" {
		values = append(values, string(ServiceCollector))
	}
	return values
}

func validDeploymentPorts(binding deploymentBinding) bool {
	base := binding.Ports.GatewayPublic
	if base < 1024 || base > 65532 || binding.Ports.RuntimeAPILoopback != base+1 || binding.Ports.EngineLoopback != base+2 {
		return false
	}
	if binding.Ports.CollectorHealthLoopback == nil {
		return true
	}
	return *binding.Ports.CollectorHealthLoopback == base+3
}

func validatedBindingServices(values []string, collectorPort *int) (map[ServiceGroup]struct{}, error) {
	services := make(map[ServiceGroup]struct{}, len(values))
	for _, value := range values {
		group := ServiceGroup(value)
		if !validServiceGroup(group) {
			return nil, fmt.Errorf("DeploymentBinding 包含未知服务组: %s", value)
		}
		if _, exists := services[group]; exists {
			return nil, fmt.Errorf("DeploymentBinding 服务组重复: %s", value)
		}
		services[group] = struct{}{}
	}
	_, entry := services[ServiceProjectEntry]
	_, runtime := services[ServiceDataRuntime]
	_, collector := services[ServiceCollector]
	if len(services) < 2 || len(services) > 3 || !entry || !runtime || (collector && collectorPort == nil) || (!collector && collectorPort != nil) {
		return nil, fmt.Errorf("DeploymentBinding 服务组或 Collector 端口无效")
	}
	return services, nil
}

func bindingEnablesService(binding deploymentBinding, group ServiceGroup) bool {
	if binding.SchemaVersion == engineBindingSchema {
		engineGroup, ok := engineServiceGroup(binding.Engine)
		return ok && engineGroup == group
	}
	for _, value := range binding.EnabledServices {
		if ServiceGroup(value) == group {
			return true
		}
	}
	return false
}

func strictDecodeJSON(raw []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("包含尾随 JSON")
		}
		return err
	}
	return nil
}

func validSHA256(value string) bool {
	_, err := parseSHA256(value)
	return err == nil
}

func validRuntimeUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for index, character := range value {
		switch index {
		case 8, 13, 18, 23:
			if character != '-' {
				return false
			}
		default:
			if !(character >= '0' && character <= '9') && !(character >= 'a' && character <= 'f') {
				return false
			}
		}
	}
	return value[14] >= '1' && value[14] <= '5' && strings.ContainsRune("89ab", rune(value[19]))
}

func validStableID(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for index, character := range value {
		letterOrDigit := (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9')
		if (index == 0 && !letterOrDigit) || (index > 0 && !letterOrDigit && !strings.ContainsRune("._:-", character)) {
			return false
		}
	}
	return true
}

func parseBindingTime(value string) (time.Time, error) {
	if !strings.HasSuffix(value, "Z") {
		return time.Time{}, fmt.Errorf("必须为 UTC RFC3339")
	}
	return time.Parse(time.RFC3339, value)
}
