// Package binding 将部署侧已经解析的非敏感运行信息绑定为 RuntimeEngine 配置。
package binding

import (
	"fmt"
	"strings"

	"github.com/indu-forge/runtime-engine/internal/model"
)

const (
	roleCompute = "compute"
	roleAlarm   = "alarm"
)

// Input 是构建 runtime-engine.config.v2 所需的非敏感部署快照。
//
// Endpoint、StateStore.ResourceRef 和 StateStore.Schema 属于部署预检上下文，
// 当前冻结的 EngineConfig 只允许写入受控引用，不能把这些字段或任何 secret 值
// 序列化进运行配置。保留它们可阻止调用方以不完整的运行支撑信息构造配置。
type Input struct {
	TenantID     string `json:"tenantId"`
	SiteID       string `json:"siteId"`
	NodeID       string `json:"nodeId"`
	InstanceID   string `json:"instanceId"`
	ProjectID    string `json:"projectId"`
	DeploymentID string `json:"deploymentId"`
	AccountID    string `json:"accountId"`
	Role         string `json:"role"`
	// ManualOwner/ManualEpoch 由控制面随 deployment binding 原子下发；不得由 Runtime API 自行推导。
	ManualOwner       string                `json:"manualOwner,omitempty"`
	ManualEpoch       int64                 `json:"manualEpoch,omitempty"`
	ArtifactMountPath string                `json:"artifactMountPath"`
	ArtifactFile      string                `json:"artifactFile"`
	JetStream         JetStreamInput        `json:"jetStream"`
	StateStore        StateStoreInput       `json:"stateStore"`
	ComputeSandbox    *model.ComputeSandbox `json:"computeSandbox,omitempty"`
	// ComputeSandboxEndpoint 与 ComputeSandboxSecretFile 仅供 source index 使用；
	// EngineConfig 本身只可保存 ComputeSandbox 中的两个受控引用。
	ComputeSandboxEndpoint   string `json:"computeSandboxEndpoint,omitempty"`
	ComputeSandboxSecretFile string `json:"computeSandboxSecretFile,omitempty"`
}

// BuildInput 是安全解包并验证 runtime-project-artifact 后才可获得的内部快照。
// 这些字段不得出现在 runtime-binding.input.v1，避免 Ops 输入伪造 producer fence。
type BuildInput struct {
	Input
	ProjectArtifact  model.ArtifactRef
	RoleOwnership    model.Ownership
	ComputeProducers []ComputeProducer
	AlarmOwnership   model.Ownership
	ManualOwnership  model.Ownership
}

// ComputeProducer 是 compute role 对一个计算单元的唯一 producer fencing 绑定。
type ComputeProducer struct {
	ComputeID string          `json:"computeId"`
	Ownership model.Ownership `json:"ownership"`
}

// JetStreamInput 只保存 endpoint 与受控引用，不包含认证值。
// Consumers 来自已准备完成的 JetStream 拓扑；BuildEngineConfig 会复制它们，
// 避免调用方随后修改输入切片影响已构建配置。
type JetStreamInput struct {
	Endpoint            string `json:"endpoint"`
	ServerResourceRef   string `json:"serverResourceRef"`
	CredentialSecretRef string `json:"credentialSecretRef"`
	DataRawStream       string `json:"dataRawStream"`
	DataDerivedStream   string `json:"dataDerivedStream"`
	EventStream         string `json:"eventStream"`
	CommandStream       string `json:"commandStream"`
	DeadLetterStream    string `json:"deadLetterStream"`
	// CredentialSecretFile 是由受控挂载提供的相对文件路径；它只供
	// resolver index 引用，绝不携带认证值。
	CredentialSecretFile string `json:"credentialSecretFile"`
	// Consumers 是本角色执行的消费项；TopologyConsumers 是同 deployment 已启用
	// compute/alarm 角色的完整受控集合，供共享 JetStream 的严格调和和预检使用。
	Consumers         []model.Consumer `json:"consumers"`
	TopologyConsumers []model.Consumer `json:"topologyConsumers"`
}

// StateStoreInput 描述状态库的部署支撑。CredentialSecretRef 写入 v2 的
// dsnSecretRef；resourceRef 与 schema 用于部署预检，v2 正式类型不承载它们。
type StateStoreInput struct {
	ResourceRef         string `json:"resourceRef"`
	CredentialSecretRef string `json:"credentialSecretRef"`
	Schema              string `json:"schema"`
	// CredentialSecretFile 是由受控挂载提供的 PostgreSQL DSN secret 文件路径。
	CredentialSecretFile string `json:"credentialSecretFile"`
}

// BuildEngineConfig 构造仅运行 compute 或 alarm 的 runtime-engine.config.v2。
// 它不加载文件、不解析索引、也不接触任何 secret 值；调用方必须在此之前完成
// 资源与 secret 引用的受控准备。
func BuildEngineConfig(input BuildInput) (model.EngineConfig, error) {
	if input.ManualOwnership.OwnerID == "" && input.ManualOwnership.Epoch == 0 {
		input.ManualOwnership = model.Ownership{OwnerID: "runtime-api", Epoch: 1}
	}
	if err := validateInput(input); err != nil {
		return model.EngineConfig{}, err
	}

	config := model.EngineConfig{
		SchemaVersion:   "runtime-engine.config.v2",
		SiteID:          input.SiteID,
		NodeID:          input.NodeID,
		ProjectID:       input.ProjectID,
		ExecutionForm:   "native-linux",
		DeploymentID:    input.DeploymentID,
		AccountID:       input.AccountID,
		ProjectArtifact: input.ProjectArtifact,
		ArtifactMount: model.ArtifactMount{
			Source:       "native-release",
			MountPath:    input.ArtifactMountPath,
			ArtifactFile: input.ArtifactFile,
			ReadOnly:     true,
		},
		Roles:           []string{input.Role},
		RoleAssignments: []model.RoleAssignment{{Role: input.Role, Ownership: input.RoleOwnership}},
		JetStream: model.JetStream{
			ServerResourceRef:   input.JetStream.ServerResourceRef,
			CredentialSecretRef: input.JetStream.CredentialSecretRef,
			DataRawStream:       input.JetStream.DataRawStream,
			DataDerivedStream:   input.JetStream.DataDerivedStream,
			EventStream:         input.JetStream.EventStream,
			CommandStream:       input.JetStream.CommandStream,
			DeadLetterStream:    input.JetStream.DeadLetterStream,
			Consumers:           cloneConsumers(input.JetStream.Consumers),
			TopologyConsumers:   cloneConsumers(input.JetStream.TopologyConsumers),
		},
		StateStore: model.StateStore{
			Engine:                 "postgresql",
			DSNSecretRef:           input.StateStore.CredentialSecretRef,
			CASRequired:            true,
			OutboxRequired:         true,
			ProcessedEventRequired: true,
			CheckpointRequired:     true,
		},
	}
	// 所有消费角色必须共享同一 manual producer fence，保证同一人工事件可被 writer/compute/alarm 验证。
	config.ProducerAssignments = append(config.ProducerAssignments, model.ProducerAssignment{ProducerType: "manual", ManualID: "runtime-api", Ownership: input.ManualOwnership})

	switch input.Role {
	case roleCompute:
		config.ComputeSandbox = &model.ComputeSandbox{
			ServerResourceRef:   input.ComputeSandbox.ServerResourceRef,
			CredentialSecretRef: input.ComputeSandbox.CredentialSecretRef,
		}
		// 人工命令由 Runtime API producer fence 保护；compute 输出仍使用各单元
		// 自己的 producer fence。两者必须同时下发，不能以 append 覆盖 manual。
		config.ProducerAssignments = append([]model.ProducerAssignment(nil), config.ProducerAssignments...)
		for _, producer := range input.ComputeProducers {
			config.ProducerAssignments = append(config.ProducerAssignments, model.ProducerAssignment{
				ProducerType: "compute",
				ComputeID:    producer.ComputeID,
				Role:         roleCompute,
				Ownership:    producer.Ownership,
			})
		}
	case roleAlarm:
		config.ProducerAssignments = []model.ProducerAssignment{{
			ProducerType: "alarm",
			Role:         roleAlarm,
			Ownership:    input.AlarmOwnership,
		}}
	}

	if err := model.ValidateEngineConfig(config); err != nil {
		return model.EngineConfig{}, fmt.Errorf("构造的 RuntimeEngine 配置未通过正式校验: %w", err)
	}
	return config, nil
}

func validateInput(input BuildInput) error {
	for label, value := range map[string]string{
		"tenantId":                       input.TenantID,
		"siteId":                         input.SiteID,
		"nodeId":                         input.NodeID,
		"instanceId":                     input.InstanceID,
		"projectId":                      input.ProjectID,
		"deploymentId":                   input.DeploymentID,
		"accountId":                      input.AccountID,
		"projectArtifact.artifactId":     input.ProjectArtifact.ArtifactID,
		"projectArtifact.artifactDigest": input.ProjectArtifact.ArtifactDigest,
		"artifactMount.mountPath":        input.ArtifactMountPath,
		"artifactMount.artifactFile":     input.ArtifactFile,
		"roleOwnership.ownerId":          input.RoleOwnership.OwnerID,
		"jetStream.endpoint":             input.JetStream.Endpoint,
		"jetStream.serverResourceRef":    input.JetStream.ServerResourceRef,
		"jetStream.credentialSecretRef":  input.JetStream.CredentialSecretRef,
		"jetStream.dataRawStream":        input.JetStream.DataRawStream,
		"jetStream.dataDerivedStream":    input.JetStream.DataDerivedStream,
		"jetStream.eventStream":          input.JetStream.EventStream,
		"jetStream.commandStream":        input.JetStream.CommandStream,
		"jetStream.deadLetterStream":     input.JetStream.DeadLetterStream,
		"stateStore.resourceRef":         input.StateStore.ResourceRef,
		"stateStore.credentialSecretRef": input.StateStore.CredentialSecretRef,
		"stateStore.schema":              input.StateStore.Schema,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("构造 RuntimeEngine 配置缺少 %s", label)
		}
	}
	if input.ProjectArtifact.ArtifactRevision < 1 {
		return fmt.Errorf("构造 RuntimeEngine 配置 projectArtifact.artifactRevision 必须至少为 1")
	}
	if input.RoleOwnership.Epoch < 1 {
		return fmt.Errorf("构造 RuntimeEngine 配置 roleOwnership.epoch 必须至少为 1")
	}
	if input.ManualOwnership.OwnerID != "runtime-api" || input.ManualOwnership.Epoch < 1 {
		return fmt.Errorf("构造 RuntimeEngine 配置 manual producer ownership 非法")
	}
	if len(input.JetStream.Consumers) == 0 {
		return fmt.Errorf("构造 RuntimeEngine 配置缺少 jetStream.consumers")
	}
	if len(input.JetStream.TopologyConsumers) == 0 {
		return fmt.Errorf("构造 RuntimeEngine 配置缺少 jetStream.topologyConsumers")
	}
	if err := validateTopologyConsumers(input.Role, input.JetStream.Consumers, input.JetStream.TopologyConsumers); err != nil {
		return err
	}

	switch input.Role {
	case roleCompute:
		if input.ComputeSandbox == nil || strings.TrimSpace(input.ComputeSandbox.ServerResourceRef) == "" || strings.TrimSpace(input.ComputeSandbox.CredentialSecretRef) == "" {
			return fmt.Errorf("compute 配置缺少 computeSandbox 受控引用")
		}
		if len(input.ComputeProducers) == 0 {
			return fmt.Errorf("compute 配置缺少 producer assignments")
		}
		for _, producer := range input.ComputeProducers {
			if strings.TrimSpace(producer.ComputeID) == "" || strings.TrimSpace(producer.Ownership.OwnerID) == "" || producer.Ownership.Epoch < 1 {
				return fmt.Errorf("compute producer assignment 不完整")
			}
		}
		if input.AlarmOwnership.OwnerID != "" || input.AlarmOwnership.Epoch != 0 {
			return fmt.Errorf("compute 配置不得声明 alarm producer ownership")
		}
	case roleAlarm:
		if len(input.ComputeProducers) != 0 || input.ComputeSandbox != nil {
			return fmt.Errorf("alarm 配置不得声明 compute producer 或 sandbox")
		}
		if strings.TrimSpace(input.AlarmOwnership.OwnerID) == "" || input.AlarmOwnership.Epoch < 1 {
			return fmt.Errorf("alarm 配置缺少 producer ownership")
		}
	default:
		return fmt.Errorf("RuntimeEngine role 必须为 compute 或 alarm")
	}
	return nil
}

func validateTopologyConsumers(role string, active, topology []model.Consumer) error {
	seen := map[string]struct{}{}
	activeCount := 0
	for _, consumer := range topology {
		if consumer.Role != roleCompute && consumer.Role != roleAlarm {
			return fmt.Errorf("jetStream.topologyConsumers 包含非法角色")
		}
		if consumer.DurableName == "" || consumer.ConsumerKey == "" {
			return fmt.Errorf("jetStream.topologyConsumers 包含空 consumer 标识")
		}
		if _, duplicate := seen[consumer.DurableName]; duplicate {
			return fmt.Errorf("jetStream.topologyConsumers 存在重复 durable")
		}
		seen[consumer.DurableName] = struct{}{}
		if consumer.Role == role {
			activeCount++
		}
	}
	if activeCount != len(active) {
		return fmt.Errorf("jetStream.consumers 必须精确等于当前角色 topology")
	}
	for _, consumer := range active {
		if _, exists := seen[consumer.DurableName]; !exists || consumer.Role != role {
			return fmt.Errorf("jetStream.consumers 必须精确等于当前角色 topology")
		}
	}
	return nil
}

func cloneConsumers(consumers []model.Consumer) []model.Consumer {
	cloned := make([]model.Consumer, len(consumers))
	for index, consumer := range consumers {
		cloned[index] = consumer
		cloned[index].BackoffMS = append([]int64(nil), consumer.BackoffMS...)
	}
	return cloned
}
