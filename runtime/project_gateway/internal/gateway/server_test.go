package gateway

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/klauspost/compress/zstd"
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

	request := httptest.NewRequest(http.MethodGet, "/api/v1/runtime/points/line.temperature", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "upstream") {
		t.Fatalf("unexpected proxy response: %d %s", response.Code, response.Body.String())
	}
	if response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("security headers must be kept on proxied responses")
	}
}

func TestGatewayViewerProxyNeverForwardsClientCredentialsOrMutations(t *testing.T) {
	var receivedAuthorization string
	var receivedPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		receivedAuthorization = request.Header.Get("Authorization")
		receivedPath = request.URL.Path
		writer.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("release-index"), 0o600); err != nil {
		t.Fatal(err)
	}
	server, err := New(testConfig(root, upstream.URL))
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{"/api/v1/runtime/points/line.temperature/history", "/ws/v1/points", "/ws/v1/alarms"} {
		receivedPath, receivedAuthorization = "", ""
		read := httptest.NewRequest(http.MethodGet, path, nil)
		read.Header.Set("Authorization", "Bearer client-token")
		read.Header.Set("Origin", "http://example.test")
		read.Host = "example.test"
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, read)
		if response.Code != http.StatusOK || receivedPath != path || receivedAuthorization != "Bearer viewer-token" {
			t.Fatalf("viewer proxy must use only the mounted token: status=%d path=%q authorization=%q", response.Code, receivedPath, receivedAuthorization)
		}
	}

	for _, mutation := range []struct{ method, path string }{
		{http.MethodPost, "/api/v1/runtime/points/line.temperature/write"},
		{http.MethodPost, "/api/v1/runtime/computes/compute-1/run"},
		{http.MethodPost, "/api/v1/runtime/session"},
		{http.MethodPost, "/ws/v1/points"},
	} {
		receivedPath, receivedAuthorization = "", ""
		write := httptest.NewRequest(mutation.method, mutation.path, nil)
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, write)
		if response.Code != http.StatusForbidden || receivedPath != "" || strings.Contains(response.Body.String(), "viewer-token") {
			t.Fatalf("mutation %s %s must remain local forbidden: status=%d upstream=%q body=%q", mutation.method, mutation.path, response.Code, receivedPath, response.Body.String())
		}
	}
}

func TestGatewayInjectsTokenFreeRuntimeBootstrapOnlyIntoHTMLEntry(t *testing.T) {
	root := t.TempDir()
	index := `<!doctype html><html><head><script type="module" src="/assets/app.js"></script></head><body><div id="app"></div></body></html>`
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte(index), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "assets"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "assets", "app.js"), []byte("console.log('app')"), 0o600); err != nil {
		t.Fatal(err)
	}
	previousAssets := browserRuntimeAssets
	browserRuntimeAssets = fstest.MapFS{"runtime-sdk.js": {Data: []byte("export function createHttpRuntime() { return { adapter: {} } }")}}
	defer func() { browserRuntimeAssets = previousAssets }()

	server, upstream := newTestServerWithRoot(t, root)
	defer upstream.Close()
	entry := httptest.NewRecorder()
	server.Handler().ServeHTTP(entry, httptest.NewRequest(http.MethodGet, "/", nil))
	body := entry.Body.String()
	if entry.Code != http.StatusOK || strings.Count(body, "data-induforge-runtime-bootstrap") != 1 || strings.Contains(body, "viewer-token") {
		t.Fatalf("entry bootstrap must be token-free and injected once: status=%d body=%s", entry.Code, body)
	}
	if strings.Index(body, "runtime-bootstrap.js") > strings.Index(body, "/assets/app.js") {
		t.Fatalf("bootstrap must precede user module scripts: %s", body)
	}

	asset := httptest.NewRecorder()
	server.Handler().ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))
	if asset.Code != http.StatusOK || strings.Contains(asset.Body.String(), "runtime-bootstrap") {
		t.Fatalf("non-html asset must not be injected: status=%d body=%s", asset.Code, asset.Body.String())
	}

	bootstrap := httptest.NewRecorder()
	server.Handler().ServeHTTP(bootstrap, httptest.NewRequest(http.MethodGet, browserRuntimeBootstrapPath, nil))
	if bootstrap.Code != http.StatusOK || !strings.Contains(bootstrap.Header().Get("Content-Type"), "javascript") || !strings.Contains(bootstrap.Body.String(), "createHttpRuntime") || strings.Contains(bootstrap.Body.String(), "viewer-token") {
		t.Fatalf("bootstrap must be a token-free external module: status=%d headers=%v body=%s", bootstrap.Code, bootstrap.Header(), bootstrap.Body.String())
	}

	sdk := httptest.NewRecorder()
	server.Handler().ServeHTTP(sdk, httptest.NewRequest(http.MethodGet, browserRuntimeSDKPath, nil))
	if sdk.Code != http.StatusOK || !strings.Contains(sdk.Header().Get("Content-Type"), "javascript") || strings.Contains(strings.ToLower(sdk.Body.String()), "<html") {
		t.Fatalf("runtime sdk asset must remain JavaScript: status=%d headers=%v body=%s", sdk.Code, sdk.Header(), sdk.Body.String())
	}
}

func TestGatewayRejectsCrossOriginViewerAndWebSocketRequests(t *testing.T) {
	server, upstream := newTestServer(t)
	defer upstream.Close()
	for _, path := range []string{"/api/v1/runtime/catalog", "/ws/v1/points", "/ws/v1/alarms"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.Host = "project.example.test"
		request.Header.Set("Origin", "https://other.example.test")
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, request)
		if response.Code != http.StatusForbidden {
			t.Fatalf("%s status=%d body=%s", path, response.Code, response.Body.String())
		}
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

func TestGatewayHealthRequiresRuntimeAPI(t *testing.T) {
	server, upstream := newTestServer(t)
	upstream.Close()
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), `"code":50031`) {
		t.Fatalf("health must reject unavailable upstream: %d %s", response.Code, response.Body.String())
	}
}

func TestPrepareClientAssetsVerifiesReleaseAndRejectsTraversal(t *testing.T) {
	for _, tc := range []struct {
		name      string
		files     map[string][]byte
		corrupt   bool
		wantError bool
	}{
		{name: "valid", files: map[string][]byte{"index.html": []byte("release-index"), "assets/app.js": []byte("ok")}},
		{name: "tampered", files: map[string][]byte{"index.html": []byte("release-index")}, corrupt: true, wantError: true},
		{name: "traversal", files: map[string][]byte{"../outside": []byte("bad")}, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			releaseRoot := t.TempDir()
			archive := clientArchive(t, tc.files)
			sum := sha256.Sum256(archive)
			if tc.corrupt {
				archive = append(archive, 0)
			}
			if err := os.WriteFile(filepath.Join(releaseRoot, clientArchiveName), archive, 0o444); err != nil {
				t.Fatal(err)
			}
			manifest := map[string]any{"artifacts": map[string]any{"client": map[string]string{"file": clientArchiveName, "checksum": "sha256:" + fmtHex(sum[:])}}}
			checksums := map[string]any{"schemaVersion": "release-checksums.v1", "files": []map[string]any{{"path": clientArchiveName, "sha256": "sha256:" + fmtHex(sum[:]), "size": int64(len(archive))}}}
			writeReleaseJSON(t, filepath.Join(releaseRoot, "release-manifest.json"), manifest)
			writeReleaseJSON(t, filepath.Join(releaseRoot, "checksums.json"), checksums)
			clientRoot := filepath.Join(t.TempDir(), "client")
			err := PrepareClientAssets(releaseRoot, clientRoot)
			if tc.wantError {
				if err == nil {
					t.Fatal("expected rejected release client artifact")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			body, err := os.ReadFile(filepath.Join(clientRoot, "index.html"))
			if err != nil || string(body) != "release-index" {
				t.Fatalf("client was not unpacked safely: %q %v", body, err)
			}
			if err := PrepareClientAssets(releaseRoot, clientRoot); err != nil {
				t.Fatalf("same artifact restart must be idempotent: %v", err)
			}
			if err := os.Chmod(filepath.Join(clientRoot, clientDigestMarkerName), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(clientRoot, clientDigestMarkerName), []byte("partial\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(filepath.Join(clientRoot, "index.html"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(clientRoot, "index.html"), []byte("partial"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := PrepareClientAssets(releaseRoot, clientRoot); err != nil {
				t.Fatalf("partial target must be atomically rebuilt: %v", err)
			}
			body, err = os.ReadFile(filepath.Join(clientRoot, "index.html"))
			if err != nil || string(body) != "release-index" {
				t.Fatalf("partial target was reused: %q %v", body, err)
			}
		})
	}
}

func clientArchive(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var compressed bytes.Buffer
	encoder, err := zstd.NewWriter(&compressed)
	if err != nil {
		t.Fatal(err)
	}
	writer := tar.NewWriter(encoder)
	for name, data := range files {
		if err := writer.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := encoder.Close(); err != nil {
		t.Fatal(err)
	}
	return compressed.Bytes()
}

func writeReleaseJSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o444); err != nil {
		t.Fatal(err)
	}
}

func fmtHex(value []byte) string { return fmt.Sprintf("%x", value) }

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

func TestConfigRejectsUnsafeViewerTokenFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	config := testConfig(root, "http://127.0.0.1:17801")
	if err := os.Chmod(config.ViewerTokenFile, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := New(config); err == nil || !strings.Contains(err.Error(), "其他用户") {
		t.Fatalf("expected world-readable viewer token rejection, got %v", err)
	}

	tokenRoot := t.TempDir()
	outsideRoot := t.TempDir()
	regular := filepath.Join(outsideRoot, "regular-token")
	if err := os.WriteFile(regular, []byte("viewer-token\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(tokenRoot, "viewer-token-link")
	if err := os.Symlink(regular, link); err != nil {
		t.Skipf("filesystem does not support symlink: %v", err)
	}
	config = testConfig(root, "http://127.0.0.1:17801")
	config.ViewerTokenFile = link
	if _, err := New(config); err == nil || !strings.Contains(err.Error(), "越出挂载目录") {
		t.Fatalf("expected escaping symlinked viewer token rejection, got %v", err)
	}
}

func TestConfigAcceptsKubernetesProjectedViewerToken(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	tokenRoot := t.TempDir()
	versionRoot := filepath.Join(tokenRoot, "..2026_09_03_06_22_40")
	if err := os.Mkdir(versionRoot, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(versionRoot, "token"), []byte("viewer-token\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Base(versionRoot), filepath.Join(tokenRoot, "..data")); err != nil {
		t.Skipf("filesystem does not support symlink: %v", err)
	}
	tokenPath := filepath.Join(tokenRoot, "token")
	if err := os.Symlink("..data/token", tokenPath); err != nil {
		t.Skipf("filesystem does not support symlink: %v", err)
	}
	config := testConfig(root, "http://127.0.0.1:17801")
	config.ViewerTokenFile = tokenPath
	if _, err := New(config); err != nil {
		t.Fatalf("expected Kubernetes projected viewer token to be accepted, got %v", err)
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
	return newTestServerWithRoot(t, root)
}

func newTestServerWithRoot(t *testing.T, root string) (*Server, *httptest.Server) {
	t.Helper()
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
	tokenPath := filepath.Join(root, "viewer-token")
	if err := os.WriteFile(tokenPath, []byte("viewer-token\n"), 0o640); err != nil {
		panic(err)
	}
	return Config{
		Listen:          "127.0.0.1:17800",
		ClientRoot:      root,
		RuntimeAPIURL:   upstream,
		ViewerTokenFile: tokenPath,
		DeploymentID:    "deployment-1",
		AccountID:       "account-1",
		ProjectID:       "11111111-1111-4111-8111-111111111111",
		SiteID:          "site-1",
		NodeID:          "node-1",
		Version:         "release-1",
		ExecutionForm:   "native-linux",
	}
}
