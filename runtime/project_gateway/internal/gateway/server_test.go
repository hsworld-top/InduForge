package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGatewayServesReleaseAssetsAndSPAFallback(t *testing.T) {
	server, upstream := newTestServer(t)
	defer upstream.Close()

	tests := []struct {
		path, wantBody, wantCache string
		status                    int
	}{
		{path: "/", wantBody: "release-index", wantCache: "no-cache", status: http.StatusOK},
		{path: "/workshop/line/1", wantBody: "release-index", wantCache: "no-cache", status: http.StatusOK},
		{path: "/assets/app.js", wantBody: "console.log('release')", wantCache: "immutable", status: http.StatusOK},
		{path: "/assets/missing.js", status: http.StatusNotFound},
		{path: "/.secrets", status: http.StatusNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tc.path, nil)
			response := httptest.NewRecorder()
			server.Handler().ServeHTTP(response, request)
			if response.Code != tc.status {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if tc.wantBody != "" && !strings.Contains(response.Body.String(), tc.wantBody) {
				t.Fatalf("body=%q", response.Body.String())
			}
			if tc.wantCache != "" && !strings.Contains(response.Header().Get("Cache-Control"), tc.wantCache) {
				t.Fatalf("cache=%q", response.Header().Get("Cache-Control"))
			}
		})
	}
}

func TestGatewayProxiesOnlyRuntimePathsAndPreservesNodeIdentity(t *testing.T) {
	server, upstream := newTestServer(t)
	defer upstream.Close()

	request := httptest.NewRequest(http.MethodGet, "/api/v1/points/line.temperature", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "upstream") {
		t.Fatalf("unexpected proxy response: %d %s", response.Code, response.Body.String())
	}
	if response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("security headers must be kept on proxied responses")
	}
}

func TestGatewayStatusReportsRuntimeAPIDegradation(t *testing.T) {
	server, upstream := newTestServer(t)
	upstream.Close()

	request := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			SchemaVersion string `json:"schemaVersion"`
			ComponentRole string `json:"componentRole"`
			AccountID     string `json:"accountId"`
			HealthState   string `json:"healthState"`
			ReasonCode    string `json:"reasonCode"`
			ProcessID     int    `json:"processId"`
			Freshness     struct {
				State       string `json:"state"`
				EvaluatedAt string `json:"evaluatedAt"`
			} `json:"businessFreshness"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != 0 || body.Data.SchemaVersion != "runtime-health-status.v1" || body.Data.ComponentRole != "project-gateway" || body.Data.AccountID != "account-1" || body.Data.HealthState != "DEGRADED" || body.Data.ReasonCode != "runtime-api-unavailable" || body.Data.ProcessID < 1 || body.Data.Freshness.State != "UNKNOWN" || body.Data.Freshness.EvaluatedAt == "" {
		t.Fatalf("unexpected status: %#v", body)
	}
}

func TestConfigRejectsRemoteRuntimeAPIAndSymlinkedAssets(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	config := testConfig(root, "http://192.0.2.1:17801")
	if _, err := New(config); err == nil || !strings.Contains(err.Error(), "回环地址") {
		t.Fatalf("expected loopback rejection, got %v", err)
	}

	target := filepath.Join(root, "target.js")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "link.js")); err != nil {
		t.Skipf("filesystem does not support symlink: %v", err)
	}
	config = testConfig(root, "http://127.0.0.1:17801")
	if _, err := New(config); err == nil || !strings.Contains(err.Error(), "符号链接") {
		t.Fatalf("expected symlink rejection, got %v", err)
	}
}

func newTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "assets"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("release-index"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "assets", "app.js"), []byte("console.log('release')"), 0o600); err != nil {
		t.Fatal(err)
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/health" {
			writer.WriteHeader(http.StatusOK)
			return
		}
		if request.Header.Get("X-InduForge-Deployment-Id") != "deployment-1" || request.Header.Get("X-InduForge-Project-Id") != "11111111-1111-4111-8111-111111111111" {
			http.Error(writer, "missing identity", http.StatusBadRequest)
			return
		}
		_, _ = writer.Write([]byte("upstream:" + request.URL.Path))
	}))
	server, err := New(testConfig(root, upstream.URL))
	if err != nil {
		upstream.Close()
		t.Fatal(err)
	}
	return server, upstream
}

func testConfig(root, upstream string) Config {
	return Config{
		Listen:        "127.0.0.1:17800",
		ClientRoot:    root,
		RuntimeAPIURL: upstream,
		DeploymentID:  "deployment-1",
		AccountID:     "account-1",
		ProjectID:     "11111111-1111-4111-8111-111111111111",
		SiteID:        "site-1",
		NodeID:        "node-1",
		Version:       "release-1",
		ExecutionForm: "native-linux",
	}
}
