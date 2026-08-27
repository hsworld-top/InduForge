package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const sandboxResponseLimit = 4 << 20

// SandboxClient 只通过内部令牌调用独立计算沙箱，不接收或转发用户 JWT。
type SandboxClient struct {
	baseURL string
	token   string
	client  *http.Client
}

// SandboxRunner 将语言固定到一个沙箱客户端，兼容 ComputeService 的 Runner/SyntaxChecker 抽象。
type SandboxRunner struct {
	language string
	client   *SandboxClient
}

func NewSandboxClient(rawURL, token string) *SandboxClient {
	return &SandboxClient{
		baseURL: strings.TrimRight(strings.TrimSpace(rawURL), "/"),
		token:   strings.TrimSpace(token),
		client:  &http.Client{Timeout: 125 * time.Second},
	}
}

func (c *SandboxClient) Runner(language string) *SandboxRunner {
	return &SandboxRunner{language: strings.ToLower(strings.TrimSpace(language)), client: c}
}

func (r *SandboxRunner) Run(ctx context.Context, request ExecuteRequest) (ExecuteResult, error) {
	if r == nil || r.client == nil {
		return ExecuteResult{}, fmt.Errorf("计算沙箱未配置")
	}
	var response struct {
		Output      any    `json:"output"`
		SideEffects []any  `json:"sideEffects"`
		Stdout      string `json:"stdout"`
		Stderr      string `json:"stderr"`
		DurationMS  int64  `json:"durationMs"`
	}
	err := r.client.call(ctx, http.MethodPost, "/v1/execute", map[string]any{
		"projectId":    request.ProjectID,
		"language":     r.language,
		"script":       request.Script,
		"input":        request.Input,
		"sdkContext":   request.SDKContext,
		"dependencies": request.Dependencies,
		"timeoutMs":    request.Timeout.Milliseconds(),
	}, &response)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return ExecuteResult{}, ErrTimeout
		}
		return ExecuteResult{}, err
	}
	return ExecuteResult{
		Output: response.Output, SideEffects: response.SideEffects, Stdout: response.Stdout,
		Stderr: response.Stderr, Duration: time.Duration(response.DurationMS) * time.Millisecond,
	}, nil
}

func (r *SandboxRunner) InstallDependency(ctx context.Context, request DependencyInstallRequest) (RuntimeDependency, error) {
	if r == nil || r.client == nil {
		return RuntimeDependency{}, fmt.Errorf("计算沙箱未配置")
	}
	var result RuntimeDependency
	err := r.client.call(ctx, http.MethodPost, "/v1/dependencies/install", request, &result)
	return result, err
}

func (r *SandboxRunner) ImportDependency(ctx context.Context, request DependencyImportRequest) (RuntimeDependency, error) {
	if r == nil || r.client == nil {
		return RuntimeDependency{}, fmt.Errorf("计算沙箱未配置")
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("projectId", request.ProjectID)
	_ = writer.WriteField("language", request.Language)
	part, err := writer.CreateFormFile("file", request.Filename)
	if err != nil {
		return RuntimeDependency{}, fmt.Errorf("创建离线依赖上传请求失败: %w", err)
	}
	if _, err := part.Write(request.Content); err != nil {
		return RuntimeDependency{}, fmt.Errorf("写入离线依赖上传请求失败: %w", err)
	}
	if err := writer.Close(); err != nil {
		return RuntimeDependency{}, fmt.Errorf("完成离线依赖上传请求失败: %w", err)
	}
	var result RuntimeDependency
	err = r.client.callBody(ctx, http.MethodPost, "/v1/dependencies/import", &body, writer.FormDataContentType(), &result)
	return result, err
}

func (r *SandboxRunner) UninstallDependency(ctx context.Context, request DependencyInstallRequest) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("计算沙箱未配置")
	}
	return r.client.call(ctx, http.MethodPost, "/v1/dependencies/uninstall", request, nil)
}

func (r *SandboxRunner) CheckSyntax(ctx context.Context, request SyntaxCheckRequest) (SyntaxCheckResult, error) {
	if r == nil || r.client == nil {
		return SyntaxCheckResult{}, fmt.Errorf("计算沙箱未配置")
	}
	var result SyntaxCheckResult
	err := r.client.call(ctx, http.MethodPost, "/v1/syntax-check", map[string]any{
		"language":  r.language,
		"script":    request.Script,
		"timeoutMs": request.Timeout.Milliseconds(),
	}, &result)
	return result, err
}

func (r *SandboxRunner) Capabilities(ctx context.Context) (SandboxCapabilities, error) {
	if r == nil || r.client == nil {
		return SandboxCapabilities{Available: false}, fmt.Errorf("计算沙箱未配置")
	}
	return r.client.Capabilities(ctx)
}

func (c *SandboxClient) Capabilities(ctx context.Context) (SandboxCapabilities, error) {
	var result SandboxCapabilities
	if err := c.call(ctx, http.MethodGet, "/v1/capabilities", nil, &result); err != nil {
		return SandboxCapabilities{Available: false, Languages: []LanguageCapability{}, SDK: []string{}, Dependencies: []DependencyCapability{}, Triggers: []string{}}, err
	}
	return result, nil
}

func (c *SandboxClient) call(ctx context.Context, method, path string, payload any, output any) error {
	if c == nil || c.baseURL == "" || c.token == "" {
		return fmt.Errorf("计算沙箱未配置或内部令牌为空")
	}
	if _, err := url.ParseRequestURI(c.baseURL); err != nil {
		return fmt.Errorf("计算沙箱地址无效: %w", err)
	}
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("编码计算沙箱请求失败: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	contentType := ""
	if payload != nil {
		contentType = "application/json"
	}
	return c.callBody(ctx, method, path, body, contentType, output)
}

func (c *SandboxClient) callBody(ctx context.Context, method, path string, body io.Reader, contentType string, output any) error {
	if c == nil || c.baseURL == "" || c.token == "" {
		return fmt.Errorf("计算沙箱未配置或内部令牌为空")
	}
	if _, err := url.ParseRequestURI(c.baseURL); err != nil {
		return fmt.Errorf("计算沙箱地址无效: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("创建计算沙箱请求失败: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	response, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("计算沙箱不可用: %w", err)
	}
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, sandboxResponseLimit+1)
	encoded, err := io.ReadAll(limited)
	if err != nil {
		return fmt.Errorf("读取计算沙箱响应失败: %w", err)
	}
	if len(encoded) > sandboxResponseLimit {
		return fmt.Errorf("计算沙箱响应超过限制")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var failure struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(encoded, &failure)
		if strings.TrimSpace(failure.Error) == "" {
			failure.Error = http.StatusText(response.StatusCode)
		}
		if response.StatusCode == http.StatusRequestTimeout {
			return ErrTimeout
		}
		return fmt.Errorf("计算沙箱拒绝请求: %s", failure.Error)
	}
	if output == nil || len(encoded) == 0 {
		return nil
	}
	if err := json.Unmarshal(encoded, output); err != nil {
		return fmt.Errorf("解析计算沙箱响应失败: %w", err)
	}
	return nil
}
