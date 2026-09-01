package ops

import (
	"fmt"
	"strings"
)

// foundationResourceRefs 只生成服务端内部的引用元数据；其中没有 Secret 值，且
// 不会进入 RuntimeEnvironmentService/API DTO。环境基础服务异常时，已有引用保留，
// LoadProjectRuntimeContext 会再以 observed_status 健康门禁拒绝新部署。
func foundationResourceRefs(environmentID, serviceType string) (map[string]string, error) {
	compact := strings.ReplaceAll(strings.ToLower(environmentID), "-", "")
	if len(compact) != 32 {
		return nil, fmt.Errorf("运行环境 ID 无效")
	}
	namespace := "if-env-" + compact[:12]
	secretName := "foundation-credentials"
	switch serviceType {
	case "nats_jetstream":
		return map[string]string{"service": "nats." + namespace + ".svc.cluster.local:4222", "resourceRef": "site-resource://" + environmentID + "/nats", "credentialSecretRef": "secret://" + environmentID + "/foundation/nats", "secretNamespace": namespace, "secretName": secretName, "secretKey": "nats-token", "authMode": "environment-token+project-subject-isolation"}, nil
	case "if_history":
		return map[string]string{"service": "postgres." + namespace + ".svc.cluster.local:5432", "resourceRef": "site-resource://" + environmentID + "/postgres", "dsnSecretRef": "secret://" + environmentID + "/foundation/postgres", "secretNamespace": namespace, "secretName": secretName, "secretKey": "postgres-password", "database": "induforge_runtime", "adminUser": "postgres", "schemaBase": "runtime", "schemaPrefix": "runtime_"}, nil
	default:
		return nil, nil
	}
}
