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
	maxResponseBytes          int64 = 1 << 20
	maxBundleResponseBytes    int64 = 48 << 20
	maxBundleSecretBytes            = 32 << 20
	maxAuthoringSnapshotBytes int64 = 512 << 20
)

// CollectorBindingBundleRequest 是控制面提交给 data_service 的最小可信部署上下文。
type CollectorBindingBundleRequest struct {
	TenantID, ProjectID, DeploymentID, EnvironmentID, NodeID, ReleaseID, AccountID string
	Revision                                                                       int64
	NATSEndpoint, NATSResourceRef, NATSCredentialSecretRef                         string
	SourceSnapshot                                                                 json.RawMessage
}

func (c *InternalClient) GetAuthoringSnapshot(ctx context.Context, projectID, tenantID string) (json.RawMessage, error) {
	if _, err := uuid.Parse(projectID); err != nil {
		return nil, fmt.Errorf("projectId 格式无效")
	}
	if _, err := uuid.Parse(tenantID); err != nil {
		return nil, fmt.Errorf("tenantId 格式无效")
	}
	endpoint := c.baseURL + "/api/v1/internal/data/projects/" + projectID + "/snapshot?tenantId=" + url.QueryEscape(tenantID)
	return c.authoringRequest(ctx, http.MethodGet, endpoint, nil)
}

func (c *InternalClient) ReplaceAuthoringSnapshotFenced(ctx context.Context, projectID, tenantID, actorID, ownerID, fenceToken, expectedEpoch, targetEpoch, direction string, snapshot json.RawMessage) error {
	body, err := json.Marshal(map[string]any{"tenantId": tenantID, "actorId": actorID, "ownerId": ownerID, "fenceToken": fenceToken, "expectedAuthoringEpoch": expectedEpoch, "targetAuthoringEpoch": targetEpoch, "direction": direction, "snapshot": snapshot})
	if err != nil {
		return err
	}
	_, err = c.authoringRequest(ctx, http.MethodPut, c.baseURL+"/api/v1/internal/data/projects/"+projectID+"/snapshot", body)
	return err
}

func (c *InternalClient) GetAuthoringEpoch(ctx context.Context, projectID, tenantID string) (string, error) {
	raw, err := c.authoringRequest(ctx, http.MethodGet, c.baseURL+"/api/v1/internal/data/project-bindings/"+projectID+"?tenantId="+url.QueryEscape(tenantID), nil)
	if err != nil {
		return "", err
	}
	var result struct {
		AuthoringEpoch string `json:"authoringEpoch"`
	}
	if strictDecode(raw, &result) != nil || result.AuthoringEpoch == "" {
		return "", fmt.Errorf("数据服务工程编辑代次响应无效")
	}
	return result.AuthoringEpoch, nil
}

func (c *InternalClient) BuildArtifactFromSnapshot(ctx context.Context, projectID, tenantID string, snapshot json.RawMessage) (json.RawMessage, error) {
	body, err := json.Marshal(map[string]any{"tenantId": tenantID, "snapshot": snapshot})
	if err != nil {
		return nil, err
	}
	return c.authoringRequest(ctx, http.MethodPost, c.baseURL+"/api/v1/internal/data/projects/"+projectID+"/artifact", body)
}

// BuildArtifactsFromSnapshot 只把已捕获快照送入数据域纯投影器；返回值不得重新读取活动项目。
func (c *InternalClient) BuildArtifactsFromSnapshot(ctx context.Context, projectID, tenantID, versionID string, capturedAt time.Time, snapshot json.RawMessage) (json.RawMessage, json.RawMessage, json.RawMessage, error) {
	payload := map[string]any{"tenantId": tenantID, "capturedAt": capturedAt.UTC().Format(time.RFC3339Nano), "snapshot": snapshot}
	if versionID != "" {
		payload["collector"] = map[string]any{"artifactId": "collector-" + versionID, "revision": 1, "collectorVersion": "1.0.0"}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, nil, err
	}
	raw, err := c.authoringRequest(ctx, http.MethodPost, c.baseURL+"/api/v1/internal/data/projects/"+projectID+"/snapshot/artifacts", body)
	if err != nil {
		return nil, nil, nil, err
	}
	var result struct {
		RuntimeArtifact         json.RawMessage `json:"runtimeArtifact"`
		CollectorArtifact       json.RawMessage `json:"collectorArtifact"`
		CollectorSourceSnapshot json.RawMessage `json:"collectorSourceSnapshot"`
	}
	if strictDecode(raw, &result) != nil || len(result.RuntimeArtifact) == 0 || !json.Valid(result.RuntimeArtifact) {
		return nil, nil, nil, fmt.Errorf("数据服务冻结产物响应无效")
	}
	if len(result.CollectorArtifact) == 0 {
		return result.RuntimeArtifact, nil, nil, nil
	}
	var coll struct {
		SchemaVersion    string `json:"schemaVersion"`
		ProjectID        string `json:"projectId"`
		ArtifactRevision int64  `json:"artifactRevision"`
	}
	if strictDecode(result.CollectorArtifact, &coll) != nil || coll.SchemaVersion != "collector-runtime-artifact.v1" || coll.ProjectID != projectID || coll.ArtifactRevision != 1 || len(result.CollectorSourceSnapshot) == 0 || !json.Valid(result.CollectorSourceSnapshot) {
		return nil, nil, nil, fmt.Errorf("数据服务冻结采集产物响应无效")
	}
	return result.RuntimeArtifact, result.CollectorArtifact, result.CollectorSourceSnapshot, nil
}

func (c *InternalClient) AcquireAuthoringFence(ctx context.Context, projectID, tenantID, ownerID, expectedEpoch string, ttlSeconds int) (string, time.Time, error) {
	token, expiry, _, err := c.writeAuthoringFence(ctx, http.MethodPost, projectID, map[string]any{"tenantId": tenantID, "ownerId": ownerID, "expectedAuthoringEpoch": expectedEpoch, "mode": "capture", "ttlSeconds": ttlSeconds})
	return token, expiry, err
}
func (c *InternalClient) AcquireRestoreAuthoringFence(ctx context.Context, projectID, tenantID, ownerID, expectedEpoch string, ttlSeconds int) (string, time.Time, string, error) {
	token, expiry, epoch, err := c.writeAuthoringFence(ctx, http.MethodPost, projectID, map[string]any{"tenantId": tenantID, "ownerId": ownerID, "expectedAuthoringEpoch": expectedEpoch, "mode": "restore", "ttlSeconds": ttlSeconds})
	if err == nil && epoch == "" {
		err = fmt.Errorf("数据服务恢复栅栏缺少编辑代次")
	}
	return token, expiry, epoch, err
}
func (c *InternalClient) RenewAuthoringFence(ctx context.Context, projectID, tenantID, ownerID, token string, ttlSeconds int) (string, time.Time, error) {
	token, expiry, _, err := c.writeAuthoringFence(ctx, http.MethodPut, projectID, map[string]any{"tenantId": tenantID, "ownerId": ownerID, "fenceToken": token, "ttlSeconds": ttlSeconds})
	return token, expiry, err
}
func (c *InternalClient) ReleaseAuthoringFence(ctx context.Context, projectID, tenantID, ownerID, token string) error {
	body, err := json.Marshal(map[string]any{"tenantId": tenantID, "ownerId": ownerID, "fenceToken": token})
	if err != nil {
		return err
	}
	_, err = c.authoringRequest(ctx, http.MethodDelete, c.baseURL+"/api/v1/internal/data/projects/"+projectID+"/authoring-fence", body)
	return err
}
func (c *InternalClient) writeAuthoringFence(ctx context.Context, method, projectID string, payload map[string]any) (string, time.Time, string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", time.Time{}, "", err
	}
	raw, err := c.authoringRequest(ctx, method, c.baseURL+"/api/v1/internal/data/projects/"+projectID+"/authoring-fence", body)
	if err != nil {
		return "", time.Time{}, "", err
	}
	var result struct {
		FenceToken     string    `json:"fenceToken"`
		ExpiresAt      time.Time `json:"expiresAt"`
		AuthoringEpoch string    `json:"authoringEpoch"`
	}
	if strictDecode(raw, &result) != nil || result.FenceToken == "" || result.ExpiresAt.IsZero() {
		return "", time.Time{}, "", fmt.Errorf("数据服务工程写栅栏响应无效")
	}
	return result.FenceToken, result.ExpiresAt, result.AuthoringEpoch, nil
}

func (c *InternalClient) authoringRequest(ctx context.Context, method, endpoint string, body []byte) (json.RawMessage, error) {
	if c == nil || c.client == nil || int64(len(body)) > maxAuthoringSnapshotBytes {
		return nil, fmt.Errorf("数据服务开发态快照请求无效")
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-InduForge-Internal-Token", c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用数据服务开发态快照接口失败")
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxAuthoringSnapshotBytes+1))
	if err != nil || int64(len(raw)) > maxAuthoringSnapshotBytes {
		return nil, fmt.Errorf("数据服务开发态快照响应过大")
	}
	var envelope struct {
		Code  int             `json:"code"`
		Data  json.RawMessage `json:"data"`
		ReqID string          `json:"reqId"`
	}
	if json.Unmarshal(raw, &envelope) != nil || envelope.ReqID == "" || resp.StatusCode/100 != 2 || envelope.Code != 0 || len(envelope.Data) == 0 {
		return nil, fmt.Errorf("数据服务开发态快照响应无效")
	}
	return envelope.Data, nil
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
	return c.EnsureProjectTenantBindingAtEpoch(ctx, projectID, tenantID, "epoch-1")
}
func (c *InternalClient) EnsureProjectTenantBindingAtEpoch(ctx context.Context, projectID, tenantID, authoringEpoch string) error {
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
		TenantID       string `json:"tenantId"`
		AuthoringEpoch string `json:"authoringEpoch"`
	}{TenantID: tenantID, AuthoringEpoch: authoringEpoch})
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
