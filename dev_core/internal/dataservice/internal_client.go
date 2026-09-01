package dataservice

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	maxResponseBytes       int64 = 1 << 20
	maxBundleResponseBytes int64 = 48 << 20
	maxBundleSecretBytes         = 32 << 20
)

// CollectorBindingBundleRequest 是控制面提交给 data_service 的最小可信部署上下文。
type CollectorBindingBundleRequest struct {
	TenantID, ProjectID, DeploymentID, EnvironmentID, NodeID, ReleaseID, AccountID string
	Revision                                                                       int64
	NATSEndpoint, NATSResourceRef, NATSCredentialSecretRef                         string
	SourceSnapshot                                                                 json.RawMessage
}

// CollectorBindingBundle 是经过完整性检查后才允许进入 Kubernetes Secret/ConfigMap 的结果。
type CollectorBindingBundle struct {
	Binding, Index             json.RawMessage
	BindingSHA256, IndexSHA256 string
	BindingSize, IndexSize     int
	SecretFiles                map[string][]byte
}

// InternalClient 仅用于控制面到数据域的受控内部调用，不复用用户 Bearer 凭据。
type InternalClient struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewInternalClient(rawURL, token string) (*InternalClient, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(rawURL), "/")
	parsed, err := url.ParseRequestURI(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("DATA_SERVICE_URL 无效")
	}
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("DATA_SERVICE_INTERNAL_TOKEN 不能为空")
	}
	return &InternalClient{baseURL: baseURL, token: strings.TrimSpace(token), client: &http.Client{Timeout: 10 * time.Second}}, nil
}

// EnsureProjectTenantBinding 幂等地写入项目租户归属，并严格验证统一响应包络。
func (c *InternalClient) EnsureProjectTenantBinding(ctx context.Context, projectID, tenantID string) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("数据服务内部客户端未初始化")
	}
	if _, err := uuid.Parse(projectID); err != nil {
		return fmt.Errorf("projectId 格式无效")
	}
	if _, err := uuid.Parse(tenantID); err != nil {
		return fmt.Errorf("tenantId 格式无效")
	}
	body, err := json.Marshal(struct {
		TenantID string `json:"tenantId"`
	}{TenantID: tenantID})
	if err != nil {
		return fmt.Errorf("构造项目租户绑定请求失败")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"/api/v1/internal/data/project-bindings/"+projectID, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建项目租户绑定请求失败")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-InduForge-Internal-Token", c.token)
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("调用数据服务同步项目租户绑定失败")
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil || int64(len(raw)) > maxResponseBytes {
		return fmt.Errorf("读取数据服务项目租户绑定响应失败")
	}
	var envelope struct {
		Code  int             `json:"code"`
		Msg   string          `json:"msg"`
		Data  json.RawMessage `json:"data"`
		ReqID string          `json:"reqId"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil || envelope.ReqID == "" || len(envelope.Data) == 0 {
		return fmt.Errorf("数据服务项目租户绑定响应无效")
	}
	if resp.StatusCode/100 != 2 || envelope.Code != 0 {
		return fmt.Errorf("数据服务项目租户绑定失败")
	}
	var result struct {
		Created *bool `json:"created"`
	}
	if err := json.Unmarshal(envelope.Data, &result); err != nil || result.Created == nil {
		return fmt.Errorf("数据服务项目租户绑定响应无效")
	}
	return nil
}

// BuildCollectorBindingBundle 调用内部 endpoint，并在网络边界验证全部长度、摘要和
// 固定文件名。任何失败都只返回固定诊断，绝不携带 body、凭据或内部 token。
func (c *InternalClient) BuildCollectorBindingBundle(ctx context.Context, input CollectorBindingBundleRequest) (*CollectorBindingBundle, error) {
	if c == nil || c.client == nil || !validBundleRequest(input) {
		return nil, fmt.Errorf("采集器绑定请求无效")
	}
	body, err := json.Marshal(map[string]any{"tenantId": input.TenantID, "deploymentId": input.DeploymentID, "environmentId": input.EnvironmentID, "nodeId": input.NodeID, "releaseId": input.ReleaseID, "revision": input.Revision, "sourceSnapshot": input.SourceSnapshot, "accountId": input.AccountID, "natsEndpoint": input.NATSEndpoint, "natsResourceRef": input.NATSResourceRef, "natsCredentialSecretRef": input.NATSCredentialSecretRef})
	if err != nil || int64(len(body)) > maxBundleResponseBytes {
		return nil, fmt.Errorf("构造采集器绑定请求失败")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/internal/data/projects/"+input.ProjectID+"/collector-binding", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建采集器绑定请求失败")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-InduForge-Internal-Token", c.token)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用数据服务构造采集器绑定失败")
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBundleResponseBytes+1))
	if err != nil || int64(len(raw)) > maxBundleResponseBytes {
		return nil, fmt.Errorf("读取采集器绑定响应失败")
	}
	var envelope struct {
		Code  int             `json:"code"`
		Data  json.RawMessage `json:"data"`
		ReqID string          `json:"reqId"`
	}
	if json.Unmarshal(raw, &envelope) != nil || envelope.ReqID == "" || resp.StatusCode/100 != 2 || envelope.Code != 0 {
		return nil, fmt.Errorf("数据服务采集器绑定失败")
	}
	var result struct {
		SchemaVersion string            `json:"schemaVersion"`
		Binding       json.RawMessage   `json:"binding"`
		BindingSHA256 string            `json:"bindingSha256"`
		BindingSize   int               `json:"bindingSize"`
		Index         json.RawMessage   `json:"index"`
		IndexSHA256   string            `json:"indexSha256"`
		IndexSize     int               `json:"indexSize"`
		SecretFiles   map[string]string `json:"secretFiles"`
	}
	if strictDecode(envelope.Data, &result) != nil || result.SchemaVersion != "collector-binding-bundle.v1" || result.BindingSize != len(result.Binding) || result.IndexSize != len(result.Index) || !validDigest(result.Binding, result.BindingSHA256) || !validDigest(result.Index, result.IndexSHA256) || result.BindingSize < 1 || result.IndexSize < 1 || result.BindingSize > 16<<20 || result.IndexSize > 16<<20 {
		return nil, fmt.Errorf("数据服务采集器绑定响应无效")
	}
	files := map[string][]byte{}
	total := 0
	for name, value := range result.SecretFiles {
		if !safeBundleSecretName(name) {
			return nil, fmt.Errorf("数据服务采集器绑定 secret 文件名无效")
		}
		decoded, decodeErr := base64.StdEncoding.DecodeString(value)
		if decodeErr != nil || len(decoded) > 1<<20 {
			return nil, fmt.Errorf("数据服务采集器绑定 secret 文件无效")
		}
		files[name] = decoded
		total += len(decoded)
	}
	if total > maxBundleSecretBytes || !json.Valid(result.Binding) || !json.Valid(result.Index) {
		return nil, fmt.Errorf("数据服务采集器绑定响应无效")
	}
	return &CollectorBindingBundle{Binding: result.Binding, Index: result.Index, BindingSHA256: result.BindingSHA256, BindingSize: result.BindingSize, IndexSHA256: result.IndexSHA256, IndexSize: result.IndexSize, SecretFiles: files}, nil
}

func validBundleRequest(v CollectorBindingBundleRequest) bool {
	if _, err := uuid.Parse(v.ProjectID); err != nil || strings.TrimSpace(v.TenantID) == "" || v.Revision < 1 || len(v.SourceSnapshot) == 0 || len(v.SourceSnapshot) > 17<<20 {
		return false
	}
	for _, value := range []string{v.DeploymentID, v.EnvironmentID, v.NodeID, v.ReleaseID, v.AccountID, v.NATSEndpoint, v.NATSResourceRef, v.NATSCredentialSecretRef} {
		if strings.TrimSpace(value) == "" {
			return false
		}
	}
	return json.Valid(v.SourceSnapshot)
}
func validDigest(raw []byte, digest string) bool {
	sum := sha256.Sum256(raw)
	return digest == "sha256:"+hex.EncodeToString(sum[:])
}
func safeBundleSecretName(name string) bool {
	if !strings.HasPrefix(name, "connection-") || !strings.HasSuffix(name, ".json") || strings.ContainsAny(name, "/\\\x00") {
		return false
	}
	id := strings.TrimSuffix(strings.TrimPrefix(name, "connection-"), ".json")
	parsed, err := uuid.Parse(id)
	return err == nil && parsed.String() == id
}
func strictDecode(raw []byte, target any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	if d.Decode(&struct{}{}) != io.EOF {
		return fmt.Errorf("JSON 存在额外内容")
	}
	return nil
}

func (c *InternalClient) SetHTTPClient(client *http.Client) {
	if client != nil {
		c.client = client
	}
}
