package compute

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	maxSandboxRequestBytes  = 1 << 20
	maxSandboxResponseBytes = 2 << 20
)

// SandboxResolver is the sole boundary where a trusted deployment resolver
// turns refs into an endpoint and bearer credential.  Values are intentionally
// opaque and have no String/JSON method, so config/status/log paths cannot
// accidentally serialize them.
type SandboxResolver interface {
	ResolveComputeSandbox(context.Context, string, string) (SandboxEndpoint, error)
}
type SandboxEndpoint struct{ url, bearer string }

func NewSandboxEndpoint(rawURL, bearer string) (SandboxEndpoint, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" || len(bearer) < 24 {
		return SandboxEndpoint{}, errors.New("sandbox resolver 返回非法连接")
	}
	return SandboxEndpoint{url: strings.TrimRight(parsed.String(), "/"), bearer: bearer}, nil
}

type SandboxClient struct {
	resolver               SandboxResolver
	client                 *http.Client
	resourceRef, secretRef string
}

func NewSandboxClient(resolver SandboxResolver, resourceRef, secretRef string) (*SandboxClient, error) {
	if resolver == nil || resourceRef == "" || secretRef == "" {
		return nil, errors.New("sandbox client 配置非法")
	}
	return &SandboxClient{resolver: resolver, client: &http.Client{Timeout: 125 * time.Second}, resourceRef: resourceRef, secretRef: secretRef}, nil
}

type ExecutionRequest struct {
	ExecutionID, DeploymentID, ProjectID, ComputeUnitID, ArtifactDigest string
	ComputeRevision                                                     int64
	Input                                                               map[string]json.RawMessage
	Timeout                                                             time.Duration
}
type ExecutionResult struct{ Output json.RawMessage }

func (c *SandboxClient) Execute(ctx context.Context, request ExecutionRequest) (ExecutionResult, error) {
	if c == nil || c.resolver == nil || request.ExecutionID == "" || request.DeploymentID == "" || request.ProjectID == "" || request.ComputeUnitID == "" || request.ArtifactDigest == "" || request.ComputeRevision < 1 || request.Timeout <= 0 || request.Timeout > 120*time.Second {
		return ExecutionResult{}, errors.New("sandbox 执行请求非法")
	}
	endpoint, err := c.resolver.ResolveComputeSandbox(ctx, c.resourceRef, c.secretRef)
	if err != nil {
		if outerErr := ctx.Err(); outerErr != nil {
			return ExecutionResult{}, outerErr
		}
		return ExecutionResult{}, errors.New("sandbox resolver 不可用")
	}
	input := make(map[string]any, len(request.Input))
	for alias, raw := range request.Input {
		var value any
		d := json.NewDecoder(bytes.NewReader(raw))
		d.UseNumber()
		if d.Decode(&value) != nil || !finiteConditionValue(value) {
			return ExecutionResult{}, errors.New("sandbox 输入非法")
		}
		input[alias] = value
	}
	body, err := json.Marshal(struct {
		ExecutionID    string         `json:"executionId"`
		DeploymentID   string         `json:"deploymentId"`
		ProjectID      string         `json:"projectId"`
		ComputeUnitID  string         `json:"computeUnitId"`
		ArtifactDigest string         `json:"artifactDigest"`
		Input          map[string]any `json:"input"`
		SDKContext     map[string]any `json:"sdkContext"`
	}{request.ExecutionID, request.DeploymentID, request.ProjectID, request.ComputeUnitID, request.ArtifactDigest, input, map[string]any{}})
	if err != nil || len(body) > maxSandboxRequestBytes {
		return ExecutionResult{}, errors.New("sandbox 请求过大")
	}
	callCtx, cancel := context.WithTimeout(ctx, request.Timeout)
	defer cancel()
	httpRequest, err := http.NewRequestWithContext(callCtx, http.MethodPost, endpoint.url+"/v1/execute", bytes.NewReader(body))
	if err != nil {
		return ExecutionResult{}, errors.New("sandbox 请求创建失败")
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+endpoint.bearer)
	response, err := c.client.Do(httpRequest)
	if err != nil {
		if timeoutErr := sandboxTimeoutError(ctx, callCtx); timeoutErr != nil {
			return ExecutionResult{}, timeoutErr
		}
		return ExecutionResult{}, errors.New("sandbox 调用失败")
	}
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, maxSandboxResponseBytes+1)
	raw, err := io.ReadAll(limited)
	if err != nil || len(raw) > maxSandboxResponseBytes {
		if timeoutErr := sandboxTimeoutError(ctx, callCtx); timeoutErr != nil {
			return ExecutionResult{}, timeoutErr
		}
		return ExecutionResult{}, errors.New("sandbox 响应过大")
	}
	if timeoutErr := sandboxTimeoutError(ctx, callCtx); timeoutErr != nil {
		return ExecutionResult{}, timeoutErr
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ExecutionResult{}, errors.New("sandbox 执行被拒绝")
	}
	var decoded struct {
		Output         json.RawMessage   `json:"output"`
		SideEffects    []json.RawMessage `json:"sideEffects"`
		Stdout         string            `json:"stdout"`
		Stderr         string            `json:"stderr"`
		DurationMS     int64             `json:"durationMs"`
		ExecutionID    string            `json:"executionId"`
		ArtifactDigest string            `json:"artifactDigest"`
		ComputeUnitID  string            `json:"computeUnitId"`
		Revision       int64             `json:"revision"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&decoded) != nil || decoder.Decode(&struct{}{}) != io.EOF || len(decoded.Output) == 0 || len(decoded.SideEffects) != 0 || decoded.ExecutionID != request.ExecutionID || decoded.ArtifactDigest != request.ArtifactDigest || decoded.ComputeUnitID != request.ComputeUnitID || decoded.Revision != request.ComputeRevision || decoded.DurationMS < 0 {
		return ExecutionResult{}, errors.New("sandbox 响应非法")
	}
	if !validRaw(decoded.Output) {
		return ExecutionResult{}, errors.New("sandbox 输出非法")
	}
	if timeoutErr := sandboxTimeoutError(ctx, callCtx); timeoutErr != nil {
		return ExecutionResult{}, timeoutErr
	}
	return ExecutionResult{Output: append([]byte(nil), decoded.Output...)}, nil
}

func sandboxTimeoutError(outer, callCtx context.Context) error {
	if outerErr := outer.Err(); outerErr != nil {
		return outerErr
	}
	if errors.Is(callCtx.Err(), context.DeadlineExceeded) {
		return ErrSandboxTimeout
	}
	return nil
}

// ExecutionUUID deterministically maps an event hash to an RFC4122-shaped ID;
// it is only a sandbox correlation identifier, never a data-plane identity.
func ExecutionUUID(eventID string) string {
	if len(eventID) < 32 {
		return "00000000-0000-4000-8000-000000000000"
	}
	b, err := hex.DecodeString(eventID[:32])
	if err != nil || len(b) != 16 {
		return "00000000-0000-4000-8000-000000000000"
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b)
	return h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}
func validRaw(raw json.RawMessage) bool {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	return decoder.Decode(&value) == nil && decoder.Decode(&struct{}{}) == io.EOF && finiteConditionValue(value)
}
