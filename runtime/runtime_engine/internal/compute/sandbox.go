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
	maxSandboxRequestBytes    = 1 << 20
	maxSandboxResponseBytes   = 2 << 20
	maxPreflightResponseBytes = 128 << 10
)

var preflightTimeout = 5 * time.Second

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

// SandboxIdentity is the Engine identity that a shared compute-sandbox must
// attest before it can be trusted to execute project artifacts.  It is kept
// separate from Endpoint so neither a URL nor a bearer token crosses status
// or lifecycle APIs.
type SandboxIdentity struct {
	SiteID, DeploymentID, ProjectID string
}

// Preflight verifies the sandbox's two public read-only endpoints.  It never
// invokes /v1/execute, follows no redirect, bounds the total call time and
// rejects non-canonical JSON so a proxy cannot smuggle a conflicting status.
func (c *SandboxClient) Preflight(ctx context.Context, expected SandboxIdentity) error {
	if c == nil || c.resolver == nil || !expected.valid() {
		return errors.New("sandbox 预检配置非法")
	}
	endpoint, err := c.resolver.ResolveComputeSandbox(ctx, c.resourceRef, c.secretRef)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("sandbox resolver 不可用")
	}
	callCtx, cancel := context.WithTimeout(ctx, preflightTimeout)
	defer cancel()
	client := &http.Client{
		Timeout: preflightTimeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	if err := preflightGET(callCtx, client, endpoint, "/health", validateSandboxHealth); err != nil {
		return err
	}
	if err := preflightGET(callCtx, client, endpoint, "/api/v1/status", func(raw []byte) error {
		return validateSandboxStatus(raw, expected)
	}); err != nil {
		return err
	}
	return nil
}

func (identity SandboxIdentity) valid() bool {
	for _, value := range []string{identity.SiteID, identity.DeploymentID, identity.ProjectID} {
		if value == "" || len(value) > 256 || strings.TrimSpace(value) != value || strings.ContainsRune(value, '\x00') {
			return false
		}
	}
	return true
}

func preflightGET(ctx context.Context, client *http.Client, endpoint SandboxEndpoint, suffix string, validate func([]byte) error) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.url+suffix, nil)
	if err != nil {
		return errors.New("sandbox 预检请求创建失败")
	}
	request.Header.Set("Authorization", "Bearer "+endpoint.bearer)
	response, err := client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return errors.New("sandbox 预检超时")
		}
		return errors.New("sandbox 预检调用失败")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return errors.New("sandbox 预检响应被拒绝")
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxPreflightResponseBytes+1))
	if err != nil || len(raw) > maxPreflightResponseBytes {
		return errors.New("sandbox 预检响应过大")
	}
	if err := validate(raw); err != nil {
		return errors.New("sandbox 预检响应非法")
	}
	return nil
}

func validateSandboxHealth(raw []byte) error {
	data, err := strictEnvelope(raw)
	if err != nil {
		return err
	}
	object, err := strictJSONObject(data, "status", "observedAt")
	if err != nil || requiredString(object, "status") != "UP" || !validRFC3339(requiredString(object, "observedAt")) {
		return errors.New("invalid health")
	}
	return nil
}

func validateSandboxStatus(raw []byte, expected SandboxIdentity) error {
	data, err := strictEnvelope(raw)
	if err != nil {
		return err
	}
	object, err := strictJSONObject(data,
		"schemaVersion", "componentRole", "siteId", "deploymentId", "accountId", "projectId", "projectCode", "deploymentStage",
		"executionForm", "nodeId", "processId", "lifecycleState", "healthState", "version", "startedAt", "uptimeSeconds", "observedAt", "lastError", "reasonCode", "businessFreshness", "collector")
	if err != nil {
		return err
	}
	for _, field := range []string{"schemaVersion", "componentRole", "siteId", "executionForm", "lifecycleState", "healthState", "version", "startedAt", "uptimeSeconds", "observedAt", "deploymentId", "projectId"} {
		if _, ok := object[field]; !ok {
			return errors.New("missing status field")
		}
	}
	if requiredString(object, "schemaVersion") != "runtime-health-status.v1" ||
		requiredString(object, "componentRole") != "compute-sandbox" ||
		requiredString(object, "lifecycleState") != "RUNNING" ||
		(requiredString(object, "healthState") != "HEALTHY" && requiredString(object, "healthState") != "DEGRADED") ||
		requiredString(object, "siteId") != expected.SiteID ||
		requiredString(object, "deploymentId") != expected.DeploymentID ||
		requiredString(object, "projectId") != expected.ProjectID ||
		!validExecutionForm(requiredString(object, "executionForm")) ||
		requiredString(object, "version") == "" ||
		!validRFC3339(requiredString(object, "startedAt")) ||
		!validRFC3339(requiredString(object, "observedAt")) ||
		!validNonNegativeInteger(object["uptimeSeconds"]) {
		return errors.New("invalid status")
	}
	// Shared sandbox instances must not impersonate a collector or a deployment
	// business-freshness source.  These fields are not needed for execution.
	if _, present := object["collector"]; present {
		return errors.New("collector status forbidden")
	}
	if _, present := object["businessFreshness"]; present {
		return errors.New("business freshness forbidden")
	}
	return nil
}

func strictEnvelope(raw []byte) (json.RawMessage, error) {
	object, err := strictJSONObject(raw, "code", "msg", "data", "reqId")
	if err != nil {
		return nil, err
	}
	var code int
	var message, requestID string
	if json.Unmarshal(object["code"], &code) != nil || code != 0 || json.Unmarshal(object["msg"], &message) != nil || json.Unmarshal(object["reqId"], &requestID) != nil || message == "" || requestID == "" || len(object["data"]) == 0 {
		return nil, errors.New("invalid envelope")
	}
	return object["data"], nil
}

// strictJSONObject rejects duplicate members recursively and requires exactly
// one complete JSON object.  json.Unmarshal alone would silently retain the
// final duplicate key, which is unsuitable for a trust decision.
func strictJSONObject(raw []byte, allowed ...string) (map[string]json.RawMessage, error) {
	if err := rejectDuplicateJSON(raw); err != nil {
		return nil, err
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return nil, errors.New("not object")
	}
	permitted := make(map[string]struct{}, len(allowed))
	for _, key := range allowed {
		permitted[key] = struct{}{}
	}
	for key := range object {
		if _, ok := permitted[key]; !ok {
			return nil, errors.New("unknown field")
		}
	}
	return object, nil
}

func rejectDuplicateJSON(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	var walk func() error
	walk = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		switch delimiter := token.(type) {
		case json.Delim:
			switch delimiter {
			case '{':
				seen := map[string]struct{}{}
				for decoder.More() {
					key, err := decoder.Token()
					if err != nil {
						return err
					}
					name, ok := key.(string)
					if !ok {
						return errors.New("invalid object key")
					}
					if _, duplicate := seen[name]; duplicate {
						return errors.New("duplicate key")
					}
					seen[name] = struct{}{}
					if err := walk(); err != nil {
						return err
					}
				}
				_, err := decoder.Token()
				return err
			case '[':
				for decoder.More() {
					if err := walk(); err != nil {
						return err
					}
				}
				_, err := decoder.Token()
				return err
			}
		}
		return nil
	}
	if err := walk(); err != nil {
		return err
	}
	if token, err := decoder.Token(); err != io.EOF || token != nil {
		return errors.New("trailing JSON")
	}
	return nil
}

func requiredString(object map[string]json.RawMessage, key string) string {
	var value string
	if raw, ok := object[key]; !ok || json.Unmarshal(raw, &value) != nil {
		return ""
	}
	return value
}

func validRFC3339(value string) bool {
	if !strings.HasSuffix(value, "Z") {
		return false
	}
	_, err := time.Parse(time.RFC3339, value)
	return err == nil
}

func validExecutionForm(value string) bool {
	return value == "k3s-workload" || value == "native-linux" || value == "native-windows"
}

func validNonNegativeInteger(raw json.RawMessage) bool {
	var value int64
	return json.Unmarshal(raw, &value) == nil && value >= 0
}

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
