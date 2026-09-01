package ops

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

const (
	resolverNATSFile     = "nats.json"
	resolverPostgresFile = "postgres.json"
	resolverSandboxFile  = "sandbox.json"
)

type DeploymentSecretManager struct{ client KubeSecretClient }

func NewDeploymentSecretManager(c KubeSecretClient) *DeploymentSecretManager {
	return &DeploymentSecretManager{client: c}
}

// Ensure 将环境级 source Secret 的最小凭据在内存中封装为 resolver 的严格 JSON，
// 写入 deployment 专属 Secret。不会返回、记录或持久化任何源 Secret 值。
func (m *DeploymentSecretManager) Ensure(ctx context.Context, ns, deploymentID, role string, support RuntimeSupportResources) (string, error) {
	if role != ServiceCompute && role != ServiceAlarm {
		return "", fmt.Errorf("运行角色非法")
	}
	nats, err := m.sourceValue(ctx, support.NATSCredentialSource, "credential")
	if err != nil {
		return "", fmt.Errorf("读取 NATS 凭据失败")
	}
	postgresPassword, err := m.sourceValue(ctx, support.StateStoreCredentialSource, "password")
	if err != nil {
		return "", fmt.Errorf("读取状态库凭据失败")
	}
	postgres, err := runtimePostgresDSN(support, postgresPassword)
	if err != nil {
		return "", fmt.Errorf("构造状态库连接失败")
	}
	files, err := BuildResolverSecretFiles(nats, postgres)
	if err != nil {
		return "", fmt.Errorf("构造 resolver 凭据失败")
	}
	name, err := computeSandboxSecretName(deploymentID)
	if err != nil {
		return "", err
	}
	data, exists, err := m.client.GetSecret(ctx, ns, name)
	if err != nil {
		return "", fmt.Errorf("读取部署 Secret 失败")
	}
	if !exists {
		data = map[string]string{}
	}
	changed := false
	if data["runtime-db-username"] == "" {
		data["runtime-db-username"] = "runtime"
		changed = true
	}
	if data["runtime-db-password"] == "" {
		value, e := secretValue()
		if e != nil {
			return "", e
		}
		data["runtime-db-password"] = value
		changed = true
	}
	for name, value := range files {
		if data[name] != value {
			data[name] = value
			changed = true
		}
	}
	if role == ServiceCompute {
		if data["sandbox-token"] == "" {
			value, e := secretValue()
			if e != nil {
				return "", e
			}
			data["sandbox-token"] = value
			changed = true
		}
		sandbox, e := json.Marshal(map[string]string{"schemaVersion": "bearer-token.v1", "token": data["sandbox-token"]})
		if e != nil {
			return "", fmt.Errorf("构造 sandbox 凭据失败")
		}
		if data[resolverSandboxFile] != string(sandbox) {
			data[resolverSandboxFile] = string(sandbox)
			changed = true
		}
	}
	if !exists || changed {
		if err = m.client.ApplySecret(ctx, ns, name, data); err != nil {
			return "", fmt.Errorf("写入部署 Secret 失败")
		}
	}
	return name, nil
}

func runtimePostgresDSN(support RuntimeSupportResources, password string) (string, error) {
	if support.StateStoreEndpoint == "" || support.StateStoreDatabase == "" || support.StateStoreAdminUser == "" || password == "" {
		return "", fmt.Errorf("状态库连接元数据缺失")
	}
	endpoint := strings.TrimPrefix(support.StateStoreEndpoint, "postgres://")
	if strings.Contains(endpoint, "/") || strings.ContainsAny(endpoint, "?@") {
		return "", fmt.Errorf("状态库 endpoint 非法")
	}
	return "postgres://" + url.QueryEscape(support.StateStoreAdminUser) + ":" + url.QueryEscape(password) + "@" + endpoint + "/" + url.PathEscape(support.StateStoreDatabase) + "?sslmode=disable", nil
}

func (m *DeploymentSecretManager) sourceValue(ctx context.Context, source KubernetesSecretSource, key string) (string, error) {
	if source.Namespace == "" || source.Name == "" || source.Keys[key] == "" {
		return "", fmt.Errorf("来源引用不完整")
	}
	data, exists, err := m.client.GetSecret(ctx, source.Namespace, source.Name)
	if err != nil || !exists || data[source.Keys[key]] == "" {
		return "", fmt.Errorf("来源凭据不可用")
	}
	return data[source.Keys[key]], nil
}

// BuildResolverSecretFiles 的 JSON 字段必须与 runtime-engine resolver 的 strict
// schema 完全一致；endpoint/account/schema 由同一 binding input 的 resources 承载。
func BuildResolverSecretFiles(natsCredential, postgresDSN string) (map[string]string, error) {
	if natsCredential == "" || postgresDSN == "" {
		return nil, fmt.Errorf("运行凭据来源缺失")
	}
	nats, err := json.Marshal(map[string]string{"schemaVersion": "nats-credential.v1", "authType": "token", "token": natsCredential})
	if err != nil {
		return nil, err
	}
	pg, err := json.Marshal(map[string]string{"schemaVersion": "postgres-dsn.v1", "dsn": postgresDSN})
	if err != nil {
		return nil, err
	}
	return map[string]string{resolverNATSFile: string(nats), resolverPostgresFile: string(pg)}, nil
}
func (m *DeploymentSecretManager) Delete(ctx context.Context, ns, deploymentID string) error {
	name, e := computeSandboxSecretName(deploymentID)
	if e != nil {
		return e
	}
	return m.client.DeleteSecret(ctx, ns, name)
}
func secretValue() (string, error) {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		return "", fmt.Errorf("生成部署 Secret 失败: %w", e)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
