package compute

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSandboxEndpointRejectsUnsafeURL(t *testing.T) {
	for _, raw := range []string{"https://user@example.test", "https://example.test/path", "https://example.test?q=x", "ftp://example.test"} {
		if _, err := NewSandboxEndpoint(raw, "012345678901234567890123"); err == nil {
			t.Fatalf("unsafe URL accepted: %s", raw)
		}
	}
	if _, err := NewSandboxEndpoint("https://sandbox.example.test", "012345678901234567890123"); err != nil {
		t.Fatal(err)
	}
}

func TestSandboxClientInternalDeadlineUsesStableTimeoutError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(100 * time.Millisecond):
		}
	}))
	defer server.Close()
	endpoint, err := NewSandboxEndpoint(server.URL, "012345678901234567890123")
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewSandboxClient(staticSandboxResolver{endpoint: endpoint}, "resource", "secret")
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Execute(context.Background(), validSandboxRequest(20*time.Millisecond))
	if !errors.Is(err, ErrSandboxTimeout) {
		t.Fatalf("Execute error=%v, want ErrSandboxTimeout", err)
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("internal deadline must not unwrap context error: %v", err)
	}
}

func TestSandboxClientSendsReadOnlyPointSDKContext(t *testing.T) {
	var received struct {
		SDKContext struct {
			PointBindings map[string]string         `json:"pointBindings"`
			Datapoints    map[string]map[string]any `json:"datapoints"`
		} `json:"sdkContext"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		writePreflightJSON(w, map[string]any{"output": map[string]any{"result": map[string]any{"celsius": 25, "fahrenheit": 77}}, "sideEffects": []any{}, "stdout": "", "stderr": "", "durationMs": 1, "executionId": "execution", "artifactDigest": "sha256:test", "computeUnitId": "unit", "revision": 4})
	}))
	defer server.Close()
	endpoint, _ := NewSandboxEndpoint(server.URL, "012345678901234567890123")
	client, _ := NewSandboxClient(staticSandboxResolver{endpoint: endpoint}, "resource", "secret")
	now := time.Date(2026, 9, 3, 8, 0, 0, 0, time.UTC)
	request := ExecutionRequest{ExecutionID: "execution", DeploymentID: "dep", ProjectID: "project", ComputeUnitID: "unit", ArtifactDigest: "sha256:test", ComputeRevision: 4, Input: map[string]json.RawMessage{"temperature": json.RawMessage(`25`)}, PointBindings: map[string]string{"temperature": "collector.demo.temperature"}, Datapoints: map[string]SandboxDatapoint{"collector.demo.temperature": {ID: "point", Path: "collector.demo.temperature", Name: "温度", DataType: "uint16", SourceType: "collector.point", Value: json.RawMessage(`25`), Quality: "good", SourceTimestamp: now, ServerTimestamp: now}}, Timeout: time.Second}
	if _, err := client.Execute(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if received.SDKContext.PointBindings["temperature"] != "collector.demo.temperature" {
		t.Fatalf("point binding missing: %+v", received.SDKContext.PointBindings)
	}
	point := received.SDKContext.Datapoints["collector.demo.temperature"]
	capabilities := point["capabilities"].(map[string]any)
	if capabilities["get"] != true || capabilities["read"] != true || capabilities["peek"] != true || capabilities["set"] != nil {
		t.Fatalf("capabilities are not read-only: %+v", capabilities)
	}
}

func TestSandboxClientRejectsUndeclaredPointBinding(t *testing.T) {
	request := validSandboxRequest(time.Second)
	request.PointBindings = map[string]string{"temperature": "collector.demo.temperature"}
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()
	endpoint, _ := NewSandboxEndpoint(server.URL, "012345678901234567890123")
	client, _ := NewSandboxClient(staticSandboxResolver{endpoint: endpoint}, "resource", "secret")
	if _, err := client.Execute(context.Background(), request); err == nil || !strings.Contains(err.Error(), "绑定非法") {
		t.Fatalf("Execute error=%v, want invalid binding", err)
	}
}

func TestSandboxClientPreflightStrictlyAttestsEndpoint(t *testing.T) {
	expected := SandboxIdentity{SiteID: "site-a", DeploymentID: "deployment-a", ProjectID: "project-a"}
	for _, test := range []struct {
		name    string
		respond func(http.ResponseWriter, *http.Request)
		wantErr bool
	}{
		{name: "success", respond: validPreflightResponder(expected)},
		{name: "identity drift", wantErr: true, respond: func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				writePreflightJSON(w, validHealthEnvelope())
				return
			}
			writePreflightJSON(w, validStatusEnvelope(SandboxIdentity{SiteID: "site-b", DeploymentID: expected.DeploymentID, ProjectID: expected.ProjectID}))
		}},
		{name: "health down", wantErr: true, respond: func(w http.ResponseWriter, _ *http.Request) {
			writePreflightJSON(w, map[string]any{"code": 0, "msg": "ok", "data": map[string]any{"status": "DOWN", "observedAt": "2026-08-30T10:00:00Z"}, "reqId": "r"})
		}},
		{name: "nonzero envelope", wantErr: true, respond: func(w http.ResponseWriter, _ *http.Request) {
			writePreflightJSON(w, map[string]any{"code": 1, "msg": "no", "data": map[string]any{"status": "UP", "observedAt": "2026-08-30T10:00:00Z"}, "reqId": "r"})
		}},
		{name: "duplicate key", wantErr: true, respond: func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				_, _ = w.Write([]byte(`{"code":0,"code":0,"msg":"ok","data":{"status":"UP","observedAt":"2026-08-30T10:00:00Z"},"reqId":"r"}`))
				return
			}
			writePreflightJSON(w, validStatusEnvelope(expected))
		}},
		{name: "extra field", wantErr: true, respond: func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				writePreflightJSON(w, validHealthEnvelope())
				return
			}
			payload := validStatusEnvelope(expected)
			payload["data"].(map[string]any)["unexpected"] = true
			writePreflightJSON(w, payload)
		}},
		{name: "trailing JSON", wantErr: true, respond: func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				_, _ = w.Write(append(preflightJSON(validHealthEnvelope()), []byte(" true")...))
				return
			}
			writePreflightJSON(w, validStatusEnvelope(expected))
		}},
		{name: "oversized response", wantErr: true, respond: func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(strings.Repeat("x", maxPreflightResponseBytes+1)))
		}},
		{name: "redirect", wantErr: true, respond: func(w http.ResponseWriter, _ *http.Request) {
			http.Redirect(w, &http.Request{}, "/other", http.StatusFound)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(test.respond))
			defer server.Close()
			endpoint, err := NewSandboxEndpoint(server.URL, "preflight-secret-must-not-leak-123456")
			if err != nil {
				t.Fatal(err)
			}
			client, err := NewSandboxClient(staticSandboxResolver{endpoint: endpoint}, "resource", "secret")
			if err != nil {
				t.Fatal(err)
			}
			err = client.Preflight(context.Background(), expected)
			if (err != nil) != test.wantErr {
				t.Fatalf("Preflight error=%v, wantErr=%v", err, test.wantErr)
			}
			if err != nil && strings.Contains(err.Error(), "preflight-secret-must-not-leak") {
				t.Fatalf("secret leaked in error: %v", err)
			}
		})
	}
}

func TestSandboxClientPreflightTimesOut(t *testing.T) {
	previous := preflightTimeout
	preflightTimeout = 20 * time.Millisecond
	t.Cleanup(func() { preflightTimeout = previous })
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { time.Sleep(100 * time.Millisecond) }))
	defer server.Close()
	endpoint, err := NewSandboxEndpoint(server.URL, "preflight-secret-must-not-leak-123456")
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewSandboxClient(staticSandboxResolver{endpoint: endpoint}, "resource", "secret")
	if err != nil {
		t.Fatal(err)
	}
	err = client.Preflight(context.Background(), SandboxIdentity{SiteID: "site-a", DeploymentID: "deployment-a", ProjectID: "project-a"})
	if err == nil || !strings.Contains(err.Error(), "超时") || strings.Contains(err.Error(), "preflight-secret-must-not-leak") {
		t.Fatalf("unexpected timeout error: %v", err)
	}
}

func validPreflightResponder(expected SandboxIdentity) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer preflight-secret-must-not-leak-123456" {
			t := fmt.Sprintf("unexpected authorization %q", r.Header.Get("Authorization"))
			http.Error(w, t, http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/health":
			writePreflightJSON(w, validHealthEnvelope())
		case "/api/v1/status":
			writePreflightJSON(w, validStatusEnvelope(expected))
		default:
			http.NotFound(w, r)
		}
	}
}

func validHealthEnvelope() map[string]any {
	return map[string]any{"code": 0, "msg": "ok", "data": map[string]any{"status": "UP", "observedAt": "2026-08-30T10:00:00Z"}, "reqId": "request-id"}
}

func validStatusEnvelope(identity SandboxIdentity) map[string]any {
	return map[string]any{"code": 0, "msg": "ok", "reqId": "request-id", "data": map[string]any{
		"schemaVersion": "runtime-health-status.v1", "componentRole": "compute-sandbox", "siteId": identity.SiteID,
		"deploymentId": identity.DeploymentID, "projectId": identity.ProjectID, "executionForm": "k3s-workload",
		"lifecycleState": "RUNNING", "healthState": "HEALTHY", "version": "1.0.0", "startedAt": "2026-08-30T09:00:00Z", "uptimeSeconds": 1, "observedAt": "2026-08-30T10:00:00Z",
	}}
}

func writePreflightJSON(w http.ResponseWriter, value any) { _, _ = w.Write(preflightJSON(value)) }
func preflightJSON(value any) []byte                      { raw, _ := json.Marshal(value); return raw }

type staticSandboxResolver struct{ endpoint SandboxEndpoint }

func (r staticSandboxResolver) ResolveComputeSandbox(context.Context, string, string) (SandboxEndpoint, error) {
	return r.endpoint, nil
}

func validSandboxRequest(timeout time.Duration) ExecutionRequest {
	return ExecutionRequest{
		ExecutionID: "00000000-0000-4000-8000-000000000001", DeploymentID: "dep", ProjectID: "project",
		ComputeUnitID: "unit", ArtifactDigest: "sha256:test", ComputeRevision: 1,
		Input: map[string]json.RawMessage{}, Timeout: timeout,
	}
}
