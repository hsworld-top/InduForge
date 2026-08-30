package compute

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
