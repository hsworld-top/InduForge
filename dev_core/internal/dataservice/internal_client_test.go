package dataservice

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestInternalClientAcquireAuthoringFenceAcceptsDataServiceContract(t *testing.T) {
	projectID, tenantID, ownerID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	expiresAt := time.Now().UTC().Add(time.Minute).Truncate(time.Microsecond)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("X-InduForge-Internal-Token") != "internal-token" {
			t.Fatal("internal authoring fence request invalid")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"msg":  "success",
			"data": map[string]any{
				"fenceToken":     "fence-token",
				"expiresAt":      expiresAt,
				"authoringEpoch": "epoch-1",
				"mode":           "capture",
			},
			"reqId": "data-request",
		})
	}))
	defer server.Close()

	client, err := NewInternalClient(server.URL, "internal-token")
	if err != nil {
		t.Fatal(err)
	}
	token, expiry, err := client.AcquireAuthoringFence(context.Background(), projectID, tenantID, ownerID, "epoch-1", 60)
	if err != nil {
		t.Fatal(err)
	}
	if token != "fence-token" || !expiry.Equal(expiresAt) {
		t.Fatalf("unexpected fence response: token=%q expiry=%s", token, expiry)
	}
}

func TestInternalClientEnsureProjectTenantBindingUsesInternalToken(t *testing.T) {
	const token = "internal-test-token"
	projectID, tenantID := uuid.NewString(), uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.Header.Get("X-InduForge-Internal-Token") != token || r.Header.Get("Authorization") != "" {
			t.Fatal("internal client did not use the expected authentication boundary")
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), tenantID) {
			t.Fatal("tenant binding request body is invalid")
		}
		_, _ = w.Write([]byte(`{"code":0,"msg":"success","data":{"created":true},"reqId":"data-request"}`))
	}))
	defer server.Close()
	client, err := NewInternalClient(server.URL, token)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.EnsureProjectTenantBinding(context.Background(), projectID, tenantID); err != nil {
		t.Fatal(err)
	}
}

func TestInternalClientBuildCollectorBindingBundleValidatesHashesAndSecretName(t *testing.T) {
	projectID, connectionID := uuid.NewString(), uuid.NewString()
	binding, index := []byte(`{"schemaVersion":"collector-runtime-binding.v1"}`), []byte(`{"schemaVersion":"collector-runtime-index.v1","resources":{},"secrets":{}}`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("X-InduForge-Internal-Token") != "internal-token" {
			t.Fatal("internal collector request invalid")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "reqId": "request-a", "data": map[string]any{"schemaVersion": "collector-binding-bundle.v1", "binding": json.RawMessage(binding), "bindingSha256": digestForTest(binding), "bindingSize": len(binding), "index": json.RawMessage(index), "indexSha256": digestForTest(index), "indexSize": len(index), "secretFiles": map[string]string{"connection-" + connectionID + ".json": base64.StdEncoding.EncodeToString([]byte(`{"password":"x"}`))}}})
	}))
	defer server.Close()
	client, _ := NewInternalClient(server.URL, "internal-token")
	bundle, err := client.BuildCollectorBindingBundle(context.Background(), bundleRequestForTest(projectID))
	if err != nil || string(bundle.SecretFiles["connection-"+connectionID+".json"]) != `{"password":"x"}` {
		t.Fatalf("bundle=%+v err=%v", bundle, err)
	}
}

func TestInternalClientBuildCollectorBindingBundleRejectsBadHashAndFileName(t *testing.T) {
	projectID := uuid.NewString()
	for _, data := range []map[string]any{{"bindingSha256": "sha256:" + strings.Repeat("0", 64), "secretFiles": map[string]string{}}, {"secretFiles": map[string]string{"../bad.json": base64.StdEncoding.EncodeToString([]byte("x"))}}} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			binding, index := []byte(`{}`), []byte(`{}`)
			result := map[string]any{"schemaVersion": "collector-binding-bundle.v1", "binding": json.RawMessage(binding), "bindingSha256": digestForTest(binding), "bindingSize": len(binding), "index": json.RawMessage(index), "indexSha256": digestForTest(index), "indexSize": len(index), "secretFiles": map[string]string{}}
			for k, v := range data {
				result[k] = v
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "reqId": "request-a", "data": result})
		}))
		client, _ := NewInternalClient(server.URL, "token")
		if _, err := client.BuildCollectorBindingBundle(context.Background(), bundleRequestForTest(projectID)); err == nil {
			t.Fatal("invalid bundle must fail")
		}
		server.Close()
	}
}

func bundleRequestForTest(projectID string) CollectorBindingBundleRequest {
	return CollectorBindingBundleRequest{TenantID: "tenant-a", ProjectID: projectID, DeploymentID: "deployment-a", EnvironmentID: "environment-a", NodeID: "node-a", ReleaseID: "release-a", AccountID: "account-a", Revision: 1, NATSEndpoint: "nats://nats:4222", NATSResourceRef: "site-resource://env/nats", NATSCredentialSecretRef: "secret://env/nats", SourceSnapshot: json.RawMessage(`{"schemaVersion":"collector-runtime-artifact.v1"}`)}
}
func digestForTest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func TestInternalClientDoesNotLeakTokenOnFailure(t *testing.T) {
	const token = "internal-secret-must-not-leak"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":10002,"msg":"内部认证失败","data":null,"reqId":"data-request"}`))
	}))
	defer server.Close()
	client, err := NewInternalClient(server.URL, token)
	if err != nil {
		t.Fatal(err)
	}
	err = client.EnsureProjectTenantBinding(context.Background(), uuid.NewString(), uuid.NewString())
	if err == nil || strings.Contains(err.Error(), token) {
		t.Fatalf("failure leaked token: %v", err)
	}
}

func TestBuildArtifactsFromSnapshotAcceptsCompleteCollectorContract(t *testing.T) {
	projectID := uuid.NewString()
	collector := map[string]any{"schemaVersion": "collector-runtime-artifact.v1", "artifactId": "collector-release", "artifactRevision": 1, "projectId": projectID, "collectorVersion": "1.0.0", "connections": []any{}, "pointMappings": []any{}, "wal": map[string]any{"maxBytes": 1048576}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "success", "reqId": "test", "data": map[string]any{"runtimeArtifact": map[string]any{}, "collectorArtifact": collector, "collectorSourceSnapshot": collector}})
	}))
	defer server.Close()
	client, err := NewInternalClient(server.URL, "internal-token")
	if err != nil {
		t.Fatal(err)
	}
	_, artifact, snapshot, err := client.BuildArtifactsFromSnapshot(context.Background(), projectID, uuid.NewString(), "release", time.Now(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(artifact) == 0 || len(snapshot) == 0 {
		t.Fatal("丢失采集冻结产物")
	}
	collector["projectId"] = uuid.NewString()
	if _, _, _, err = client.BuildArtifactsFromSnapshot(context.Background(), projectID, uuid.NewString(), "release", time.Now(), json.RawMessage(`{}`)); err == nil {
		t.Fatal("接受了其他工程的采集产物")
	}
}
