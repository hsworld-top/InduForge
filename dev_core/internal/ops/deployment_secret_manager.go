package ops

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

const (
	resolverNATSFile       = "nats.json"
	resolverPostgresFile   = "postgres.json"
	bootstrapPostgresFile  = "postgres-bootstrap.json"
	resolverSandboxFile    = "sandbox.json"
	runtimeAPINATSFile     = "runtime-api-nats.json"
	runtimeAPIPostgresFile = "runtime-api-postgres.json"
	runtimeAPITokensFile   = "runtime-api-tokens.json"
)

type DeploymentSecretManager struct{ client KubeSecretClient }

func NewDeploymentSecretManager(c KubeSecretClient) *DeploymentSecretManager {
	return &DeploymentSecretManager{client: c}
}

// Ensure 将环境级 source Secret 的最小凭据在内存中封装为 resolver 的严格 JSON，
// 写入 deployment 专属 Secret。不会返回、记录或持久化任何源 Secret 值。
func (m *DeploymentSecretManager) Ensure(ctx context.Context, ns, deploymentID, projectID, environmentID, role string, support RuntimeSupportResources) (string, error) {
	if role != ServiceBase && role != ServiceCompute && role != ServiceAlarm {
		return "", fmt.Errorf("运行角色非法")
	}
	database := runtimeStateDatabaseName(projectID, environmentID)
	if projectID == "" || environmentID == "" || database == "" {
		return "", fmt.Errorf("运行状态库槽位非法")
	}
	nats, err := m.sourceValue(ctx, support.NATSCredentialSource, "credential")
	if err != nil {
		return "", fmt.Errorf("读取 NATS 凭据失败")
	}
	postgresPassword, err := m.sourceValue(ctx, support.StateStoreCredentialSource, "password")
	if err != nil {
		return "", fmt.Errorf("读取状态库凭据失败")
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
		data["runtime-db-username"] = runtimeDatabaseUser(deploymentID)
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
	// bootstrap 必须连 maintenance database；目标 runtime database 首次尚不存在，
	// 不能把它伪装成可连接的 admin DSN。
	bootstrapDSN, err := runtimePostgresDSNForDatabase(support, support.StateStoreAdminUser, postgresPassword, "postgres")
	if err != nil {
		return "", fmt.Errorf("构造 bootstrap 状态库连接失败")
	}
	runtimeDSN, err := runtimePostgresDSNForDatabase(support, data["runtime-db-username"], data["runtime-db-password"], database)
	if err != nil {
		return "", fmt.Errorf("构造运行状态库连接失败")
	}
	files, err := BuildResolverSecretFiles(nats, runtimeDSN)
	if err != nil {
		return "", fmt.Errorf("构造 resolver 凭据失败")
	}
	bootstrap, err := postgresBootstrapSecretFile(bootstrapDSN, database, runtimeStateSchema, data["runtime-db-username"], data["runtime-db-password"])
	if err != nil {
		return "", fmt.Errorf("构造 bootstrap 凭据失败")
	}
	files[bootstrapPostgresFile] = bootstrap
	if role == ServiceBase {
		// Runtime API 只得到运行库用户 DSN 与项目 NATS token；admin bootstrap
		// 凭据不会投影到 API 容器。
		files[runtimeAPINATSFile] = files[resolverNATSFile]
		files[runtimeAPIPostgresFile] = files[resolverPostgresFile]
		if data["runtime-api-token"] == "" {
			value, e := secretValue()
			if e != nil {
				return "", e
			}
			data["runtime-api-token"] = value
			changed = true
		}
		digest := sha256.Sum256([]byte(data["runtime-api-token"]))
		tokens, e := json.Marshal(map[string]any{"schemaVersion": "runtime-api-tokens.v1", "tokens": []map[string]any{{"tokenSha256": hex.EncodeToString(digest[:]), "subjectId": "deployment-user", "roles": []string{"viewer"}}}})
		if e != nil {
			return "", fmt.Errorf("构造 Runtime API token 凭据失败")
		}
		files[runtimeAPITokensFile] = string(tokens)
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

func safeCollectorSecretFileName(name string) bool {
	if !strings.HasPrefix(name, "connection-") || !strings.HasSuffix(name, ".json") || strings.ContainsAny(name, "/\\\x00") {
		return false
	}
	id := strings.TrimSuffix(strings.TrimPrefix(name, "connection-"), ".json")
	if len(id) != 36 {
		return false
	}
	for i, c := range id {
		if (i == 8 || i == 13 || i == 18 || i == 23) && c == '-' {
			continue
		}
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func runtimePostgresDSN(support RuntimeSupportResources, username, password string) (string, error) {
	return runtimePostgresDSNForDatabase(support, username, password, support.StateStoreDatabase)
}

func runtimePostgresDSNForDatabase(support RuntimeSupportResources, username, password, database string) (string, error) {
	if support.StateStoreEndpoint == "" || database == "" || username == "" || password == "" {
		return "", fmt.Errorf("状态库连接元数据缺失")
	}
	endpoint := strings.TrimPrefix(support.StateStoreEndpoint, "postgres://")
	if strings.Contains(endpoint, "/") || strings.ContainsAny(endpoint, "?@") {
		return "", fmt.Errorf("状态库 endpoint 非法")
	}
	return "postgres://" + url.QueryEscape(username) + ":" + url.QueryEscape(password) + "@" + endpoint + "/" + url.PathEscape(database) + "?sslmode=disable", nil
}

func runtimeDatabaseUser(deploymentID string) string {
	return "runtime_" + strings.ReplaceAll(strings.ToLower(deploymentID), "-", "")[:12]
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
	pg, err := postgresSecretFile(postgresDSN)
	if err != nil {
		return nil, err
	}
	return map[string]string{resolverNATSFile: string(nats), resolverPostgresFile: pg}, nil
}
func postgresSecretFile(dsn string) (string, error) {
	raw, err := json.Marshal(map[string]string{"schemaVersion": "postgres-dsn.v1", "dsn": dsn})
	return string(raw), err
}

func postgresBootstrapSecretFile(maintenanceDSN, database, schema, username, password string) (string, error) {
	if maintenanceDSN == "" || database == "" || schema == "" || username == "" || password == "" {
		return "", fmt.Errorf("bootstrap 状态库参数缺失")
	}
	raw, err := json.Marshal(map[string]string{"schemaVersion": "postgres-bootstrap.v1", "maintenanceDsn": maintenanceDSN, "database": database, "schema": schema, "username": username, "password": password})
	return string(raw), err
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
