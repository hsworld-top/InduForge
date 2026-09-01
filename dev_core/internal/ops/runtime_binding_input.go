package ops

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

const runtimeBindingInputVersion = "runtime-binding.input.v1"

// BuildRuntimeBindingInput 从受信 deployment context 生成 prepare 的唯一输入。它不
// 接收外部 producer/ownership/Artifact digest，运行制品身份只能来自 Release Manifest。
func BuildRuntimeBindingInput(workload ProjectWorkload, context ProjectRuntimeContext) ([]byte, error) {
	if workload.Engine != ServiceCompute && workload.Engine != ServiceAlarm {
		return nil, fmt.Errorf("仅 compute/alarm 需要运行绑定输入")
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
	key := stableRuntimeKey(context.ProjectID, context.DeploymentID)
	input := map[string]any{
		"schemaVersion":         runtimeBindingInputVersion,
		"releaseId":             workload.ReleaseID,
		"runtimeArtifactPath":   "/opt/induforge/release/current/" + runtimeArtifactFile,
		"runtimeArtifactSha256": manifest.Artifacts.Runtime.Checksum,
		"artifactDir":           "/work/artifact", "bundleDir": "/work/bundle",
		"binding": map[string]any{
			"tenantId": context.TenantID, "siteId": context.EnvironmentID, "nodeId": workload.NodeID,
			"instanceId": "if-" + role + "-" + stableRuntimeKey(context.DeploymentID, fmt.Sprint(workload.Generation)),
			"projectId":  context.ProjectID, "deploymentId": context.DeploymentID, "accountId": "if-" + key,
			"role": role, "artifactMountPath": "/work/artifact", "artifactFile": "runtime-project-artifact.json",
			"jetStream":  runtimeJetStreamInput(role, key, context.Support),
			"stateStore": map[string]any{"resourceRef": context.Support.StateStoreResourceRef, "credentialSecretRef": context.Support.StateStoreDSNSecretRef, "credentialSecretFile": "secrets/postgres.json", "schema": "runtime_" + key},
		},
	}
	if role == ServiceCompute {
		input["binding"].(map[string]any)["computeSandbox"] = map[string]any{"serverResourceRef": "site-resource://" + context.EnvironmentID + "/compute-sandbox", "credentialSecretRef": "secret://deployment/" + context.DeploymentID + "/compute-sandbox"}
		input["binding"].(map[string]any)["computeSandboxEndpoint"] = "http://compute-sandbox:18103"
		input["binding"].(map[string]any)["computeSandboxSecretFile"] = "secrets/sandbox.json"
	}
	return json.Marshal(input)
}

func runtimeJetStreamInput(role, key string, support RuntimeSupportResources) map[string]any {
	streams := map[string]string{"raw": "IF_" + strings.ToUpper(key) + "_RAW", "derived": "IF_" + strings.ToUpper(key) + "_DERIVED", "event": "IF_" + strings.ToUpper(key) + "_EVENT", "dlq": "IF_" + strings.ToUpper(key) + "_DLQ"}
	consumer := func(kind, stream, subject string) map[string]any {
		return map[string]any{"role": role, "consumerKey": role + "-" + kind + "-v1", "stream": stream, "durableName": role + "-" + kind + "-v1", "filterSubject": subject, "ackPolicy": "explicit", "ackWaitMs": 30000, "maxDeliver": 5, "backoffMs": []int{1000, 5000, 30000}, "maxAckPending": 32, "maxWaiting": 32, "maxRequestBatch": 32, "maxRequestExpiresMs": 5000, "maxRequestMaxBytes": 1 << 20, "deadLetterSubject": "dlq." + role + "-" + kind}
	}
	return map[string]any{"endpoint": support.NATSEndpoint, "serverResourceRef": support.NATSResourceRef, "credentialSecretRef": support.NATSCredentialSecretRef, "credentialSecretFile": "secrets/nats.json", "dataRawStream": streams["raw"], "dataDerivedStream": streams["derived"], "eventStream": streams["event"], "deadLetterStream": streams["dlq"], "consumers": []any{consumer("raw", streams["raw"], "data.raw.>"), consumer("derived", streams["derived"], "data.computed.>")}}
}

func stableRuntimeKey(values ...string) string {
	h := sha256.Sum256([]byte(strings.Join(values, "\x00")))
	return hex.EncodeToString(h[:])[:16]
}
