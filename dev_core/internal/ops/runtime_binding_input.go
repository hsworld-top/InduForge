package ops

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

const runtimeBindingInputVersion = "runtime-binding.input.v1"
const runtimeStateSchema = "runtime_engine"

// BuildRuntimeBindingInput 从受信 deployment context 生成 prepare 的唯一输入。它不
// 接收外部 producer/ownership/Artifact digest，运行制品身份只能来自 Release Manifest。
func BuildRuntimeBindingInput(workload ProjectWorkload, context ProjectRuntimeContext) ([]byte, error) {
	if workload.Engine != ServiceBase && workload.Engine != ServiceCompute && workload.Engine != ServiceAlarm {
		return nil, fmt.Errorf("仅 base/compute/alarm 需要运行绑定输入")
	}
	if workload.DeploymentID != context.DeploymentID || workload.EnvironmentID != context.EnvironmentID || workload.ReleaseID != context.Release.ID || workload.NodeID == "" || workload.Generation < 1 {
		return nil, fmt.Errorf("工作负载与运行上下文身份不一致")
	}
	var manifest deployableReleaseManifest
	if err := json.Unmarshal(context.Release.Manifest, &manifest); err != nil || manifest.Artifacts.Runtime.File != runtimeArtifactFile || !validSHA256Checksum(manifest.Artifacts.Runtime.Checksum) {
		return nil, fmt.Errorf("运行制品描述缺失或无效")
	}
	if context.Support.NATSEndpoint == "" || context.Support.NATSResourceRef == "" || context.Support.NATSCredentialSecretRef == "" || context.Support.StateStoreResourceRef == "" || context.Support.StateStoreDSNSecretRef == "" {
		return nil, fmt.Errorf("运行支撑引用缺失")
	}
	role := workload.Engine
	manualEpoch := workload.BindingRevision
	if manualEpoch < 1 {
		manualEpoch = 1
	}
	key := stableRuntimeKey(context.ProjectID, context.DeploymentID)
	input := map[string]any{
		"schemaVersion": runtimeBindingInputVersion,
		"releaseId":     workload.ReleaseID,
		// 工作负载挂载的是摘要固定的 Release 目录，容器内没有节点 Agent 的 current 链接。
		"runtimeArtifactPath":   "/opt/induforge/release/" + runtimeArtifactFile,
		"runtimeArtifactSha256": manifest.Artifacts.Runtime.Checksum,
		"artifactDir":           "/work/artifact", "bundleDir": "/work/bundle",
		"binding": map[string]any{
			"tenantId": context.TenantID, "siteId": context.EnvironmentID, "nodeId": workload.NodeID,
			"instanceId": "if-" + role + "-" + stableRuntimeKey(context.DeploymentID, fmt.Sprint(workload.Generation)),
			// generation 是本角色受控滚动的单调代次。它只作为 fencing epoch；
			// RuntimeEngine 会自行从 deployment+role 推导稳定 owner，不能把 Pod
			// instanceId（会随 generation 改变）混入 owner。
			"fencingEpoch": workload.Generation,
			"projectId":    context.ProjectID, "deploymentId": context.DeploymentID, "accountId": runtimeAccountID(context.ProjectID, context.DeploymentID),
			"role": role, "manualOwner": "runtime-api", "manualEpoch": manualEpoch, "artifactMountPath": "/opt/induforge/release/runtime-artifact", "artifactFile": "runtime-project-artifact.json",
			"jetStream":  runtimeJetStreamInput(role, context.RuntimeEngines, key, context.Support),
			"stateStore": map[string]any{"resourceRef": context.Support.StateStoreResourceRef, "credentialSecretRef": context.Support.StateStoreDSNSecretRef, "credentialSecretFile": "secrets/postgres.json", "schema": runtimeStateSchema},
		},
	}
	if role == ServiceCompute {
		input["binding"].(map[string]any)["computeSandbox"] = map[string]any{"serverResourceRef": "site-resource://" + context.EnvironmentID + "/compute-sandbox", "credentialSecretRef": "secret://deployment/" + context.DeploymentID + "/compute-sandbox"}
		// 计算沙箱是 compute 工作负载的同 Pod sidecar，必须使用 loopback，避免
		// 将不存在的 Kubernetes Service 名称写入受信运行时索引。
		input["binding"].(map[string]any)["computeSandboxEndpoint"] = "http://127.0.0.1:18103"
		input["binding"].(map[string]any)["computeSandboxSecretFile"] = "secrets/sandbox.json"
	}
	return json.Marshal(input)
}

func runtimeJetStreamInput(role string, enabledRoles []string, key string, support RuntimeSupportResources) map[string]any {
	streams := map[string]string{"raw": "IF_" + strings.ToUpper(key) + "_RAW", "derived": "IF_" + strings.ToUpper(key) + "_DERIVED", "event": "IF_" + strings.ToUpper(key) + "_EVENT", "command": "IF_" + strings.ToUpper(key) + "_COMMAND", "dlq": "IF_" + strings.ToUpper(key) + "_DLQ"}
	consumer := func(consumerRole, kind, stream, subject string) map[string]any {
		return map[string]any{"role": consumerRole, "consumerKey": consumerRole + "-" + kind + "-v1", "stream": stream, "durableName": consumerRole + "-" + kind + "-v1", "filterSubject": subject, "ackPolicy": "explicit", "ackWaitMs": 30000, "maxDeliver": 5, "backoffMs": []int{1000, 5000, 30000}, "maxAckPending": 32, "maxWaiting": 32, "maxRequestBatch": 32, "maxRequestExpiresMs": 5000, "maxRequestMaxBytes": 1 << 20, "deadLetterSubject": "dlq." + consumerRole + "-" + kind}
	}
	topology := make([]any, 0, 5)
	for _, enabledRole := range enabledRoles {
		if enabledRole != ServiceCompute && enabledRole != ServiceAlarm {
			continue
		}
		topology = append(topology, consumer(enabledRole, "raw", streams["raw"], "data.raw.>"), consumer(enabledRole, "derived", streams["derived"], "data.computed.>"))
		if enabledRole == ServiceCompute {
			topology = append(topology, consumer(enabledRole, "command", streams["command"], "compute.command.>"))
		}
	}
	active := make([]any, 0, 3)
	if role == ServiceBase {
		active = append(active, topology...)
	}
	for _, item := range topology {
		if item.(map[string]any)["role"] == role {
			active = append(active, item)
		}
	}
	return map[string]any{"endpoint": support.NATSEndpoint, "serverResourceRef": support.NATSResourceRef, "credentialSecretRef": support.NATSCredentialSecretRef, "credentialSecretFile": "secrets/nats.json", "dataRawStream": streams["raw"], "dataDerivedStream": streams["derived"], "eventStream": streams["event"], "commandStream": streams["command"], "deadLetterStream": streams["dlq"], "consumers": active, "topologyConsumers": topology}
}

func stableRuntimeKey(values ...string) string {
	h := sha256.Sum256([]byte(strings.Join(values, "\x00")))
	return hex.EncodeToString(h[:])[:16]
}

// runtimeAccountID 是部署级 NATS 逻辑账户与 stream 命名的共同隔离键。它不能
// 混入 service generation、release 或模式，否则 collector、base、compute、alarm
// 会在同一部署中得到不一致的账户身份。
func runtimeAccountID(projectID, deploymentID string) string {
	return "if-" + stableRuntimeKey(projectID, deploymentID)
}

// runtimeStateDatabaseName 只以 project/environment 槽位生成数据库名；模式切换
// 与 deployment 重建均不会改变隔离边界，且从不采信 resource refs 的任意名称。
func runtimeStateDatabaseName(projectID, environmentID string) string {
	return "ifrt_" + stableRuntimeKey(projectID, environmentID)
}
