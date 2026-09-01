package dataservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

const maxResponseBytes int64 = 1 << 20

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

func (c *InternalClient) SetHTTPClient(client *http.Client) {
	if client != nil {
		c.client = client
	}
}
