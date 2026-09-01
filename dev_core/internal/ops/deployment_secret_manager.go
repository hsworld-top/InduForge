package ops

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

type DeploymentSecretManager struct{ client KubeSecretClient }

func NewDeploymentSecretManager(c KubeSecretClient) *DeploymentSecretManager {
	return &DeploymentSecretManager{client: c}
}
func (m *DeploymentSecretManager) Ensure(ctx context.Context, ns, deploymentID string, needSandbox bool) (string, error) {
	name, err := computeSandboxSecretName(deploymentID)
	if err != nil {
		return "", err
	}
	data, exists, err := m.client.GetSecret(ctx, ns, name)
	if err != nil {
		return "", err
	}
	if !exists {
		data = map[string]string{}
	}
	changed := false
	for _, key := range []string{"runtime-db-password"} {
		if data[key] == "" {
			v, e := secretValue()
			if e != nil {
				return "", e
			}
			data[key] = v
			changed = true
		}
	}
	if needSandbox && data["sandbox-token"] == "" {
		v, e := secretValue()
		if e != nil {
			return "", e
		}
		data["sandbox-token"] = v
		changed = true
	}
	if !exists || changed {
		if err = m.client.ApplySecret(ctx, ns, name, data); err != nil {
			return "", err
		}
	}
	return name, nil
}

// BuildResolverSecretFiles 在内存中把环境级来源凭据封装成 resolver 所需文件。
// NATS 保持 environment credential + project subject isolation 的事实；本函数不
// 将 data 返回给 API/数据库或日志，调用方只能立即交给受控 Secret client。
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
	return map[string]string{"nats.json": string(nats), "postgres.json": string(pg)}, nil
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
