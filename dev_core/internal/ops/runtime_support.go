package ops

import (
	"context"
	"fmt"
	"strings"
)

// RuntimeSupportResources 是环境基础设施向工程运行时公开的非敏感资源索引。
// Secret 字段均为 Kubernetes Secret 名称或键引用，真实值只由 Pod 挂载读取。
type RuntimeSupportResources struct {
	NATSEndpoint, NATSResourceRef, NATSCredentialSecretRef string
	StateStoreDSNSecretRef, StateStoreSchema               string
	// NATS 使用环境级 credential；项目隔离依赖 project subject，而非伪造独立 account。
	NATSCredentialSource, StateStoreCredentialSource KubernetesSecretSource
}

// KubernetesSecretSource 只描述受控读取位置与键映射，绝不承载 Secret 值。
type KubernetesSecretSource struct {
	Namespace, Name string
	Keys            map[string]string
}

// DeploymentRuntimeProvisioner 定义后续幂等准备边界。实现必须创建/校验
// stream、consumer 与 state schema；它不得返回或持久化任何明文凭据。
type DeploymentRuntimeProvisioner interface {
	Prepare(context.Context, RuntimeSupportResources, ProjectDeployment, []DeploymentService) error
}

// ResolveRuntimeSupportResources 从已运行的环境基础服务引用中恢复运行支撑；
// 缺失即预检失败，调用方不能回退到猜测地址或空认证。
func ResolveRuntimeSupportResources(refs map[string]map[string]string) (RuntimeSupportResources, error) {
	lookup := func(service, key string) string { return strings.TrimSpace(refs[service][key]) }
	resources := RuntimeSupportResources{
		NATSEndpoint: lookup("nats_jetstream", "service"), NATSResourceRef: lookup("nats_jetstream", "resourceRef"), NATSCredentialSecretRef: lookup("nats_jetstream", "credentialSecretRef"),
		StateStoreDSNSecretRef: lookup("if_history", "dsnSecretRef"), StateStoreSchema: lookup("if_history", "schema"),
	}
	resources.NATSCredentialSource = KubernetesSecretSource{Namespace: lookup("nats_jetstream", "secretNamespace"), Name: lookup("nats_jetstream", "secretName"), Keys: map[string]string{"credential": lookup("nats_jetstream", "secretKey")}}
	resources.StateStoreCredentialSource = KubernetesSecretSource{Namespace: lookup("if_history", "secretNamespace"), Name: lookup("if_history", "secretName"), Keys: map[string]string{"dsn": lookup("if_history", "secretKey")}}
	for label, value := range map[string]string{"NATS Service": resources.NATSEndpoint, "NATS resourceRef": resources.NATSResourceRef, "NATS credentialSecretRef": resources.NATSCredentialSecretRef, "state-store dsnSecretRef": resources.StateStoreDSNSecretRef, "state-store schema": resources.StateStoreSchema} {
		if value == "" {
			return RuntimeSupportResources{}, fmt.Errorf("运行环境缺少 %s", label)
		}
	}
	for label, source := range map[string]KubernetesSecretSource{"NATS credential source": resources.NATSCredentialSource, "state-store credential source": resources.StateStoreCredentialSource} {
		if strings.TrimSpace(source.Namespace) == "" || strings.TrimSpace(source.Name) == "" || strings.TrimSpace(source.Keys["credential"]+source.Keys["dsn"]) == "" {
			return RuntimeSupportResources{}, fmt.Errorf("运行环境缺少 %s", label)
		}
	}
	return resources, nil
}
