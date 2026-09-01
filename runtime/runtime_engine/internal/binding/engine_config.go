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
	TenantID, SiteID, NodeID, ProjectID, DeploymentID, AccountID string
	Role                                                         string
	ProjectArtifact                                              model.ArtifactRef
	ArtifactMountPath, ArtifactFile                              string
	RoleOwnership                                                model.Ownership
	ComputeProducers                                             []ComputeProducer
	AlarmOwnership                                               model.Ownership
	JetStream                                                    JetStreamInput
	StateStore                                                   StateStoreInput
	ComputeSandbox                                               *model.ComputeSandbox
	// ComputeSandboxEndpoint 与 ComputeSandboxSecretFile 仅供 source index 使用；
	// EngineConfig 本身只可保存 ComputeSandbox 中的两个受控引用。
	ComputeSandboxEndpoint, ComputeSandboxSecretFile string
}

// ComputeProducer 是 compute role 对一个计算单元的唯一 producer fencing 绑定。
type ComputeProducer struct {
	ComputeID string
	Ownership model.Ownership
}

// JetStreamInput 只保存 endpoint 与受控引用，不包含认证值。
// Consumers 来自已准备完成的 JetStream 拓扑；BuildEngineConfig 会复制它们，
// 避免调用方随后修改输入切片影响已构建配置。
type JetStreamInput struct {
	Endpoint, ServerResourceRef, CredentialSecretRef                string
	DataRawStream, DataDerivedStream, EventStream, DeadLetterStream string
	// CredentialSecretFile 是由受控挂载提供的相对文件路径；它只供
	// resolver index 引用，绝不携带认证值。
	CredentialSecretFile string
	Consumers            []model.Consumer
}

// StateStoreInput 描述状态库的部署支撑。CredentialSecretRef 写入 v2 的
// dsnSecretRef；resourceRef 与 schema 用于部署预检，v2 正式类型不承载它们。
type StateStoreInput struct {
	ResourceRef, CredentialSecretRef, Schema string
	// CredentialSecretFile 是由受控挂载提供的 PostgreSQL DSN secret 文件路径。
	CredentialSecretFile string
}

// BuildEngineConfig 构造仅运行 compute 或 alarm 的 runtime-engine.config.v2。
// 它不加载文件、不解析索引、也不接触任何 secret 值；调用方必须在此之前完成
// 资源与 secret 引用的受控准备。
func BuildEngineConfig(input Input) (model.EngineConfig, error) {
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
			DeadLetterStream:    input.JetStream.DeadLetterStream,
			Consumers:           cloneConsumers(input.JetStream.Consumers),
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

	switch input.Role {
	case roleCompute:
		config.ComputeSandbox = &model.ComputeSandbox{
			ServerResourceRef:   input.ComputeSandbox.ServerResourceRef,
			CredentialSecretRef: input.ComputeSandbox.CredentialSecretRef,
		}
		config.ProducerAssignments = make([]model.ProducerAssignment, 0, len(input.ComputeProducers))
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

func validateInput(input Input) error {
	for label, value := range map[string]string{
		"tenantId":                       input.TenantID,
		"siteId":                         input.SiteID,
		"nodeId":                         input.NodeID,
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
	if len(input.JetStream.Consumers) == 0 {
		return fmt.Errorf("构造 RuntimeEngine 配置缺少 jetStream.consumers")
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

func cloneConsumers(consumers []model.Consumer) []model.Consumer {
	cloned := make([]model.Consumer, len(consumers))
	for index, consumer := range consumers {
		cloned[index] = consumer
		cloned[index].BackoffMS = append([]int64(nil), consumer.BackoffMS...)
	}
	return cloned
}
