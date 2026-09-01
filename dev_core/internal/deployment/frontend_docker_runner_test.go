package deployment

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeDockerFrontendEngine struct {
	spec dockerFrontendSpec
	err  error
}

func (e *fakeDockerFrontendEngine) Run(_ context.Context, s dockerFrontendSpec) error {
	e.spec = s
	if e.err != nil {
		return e.err
	}
	p := filepath.Join(testWorkspaceRoot(s), "dist")
	if err := os.MkdirAll(p, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(p, "index.html"), []byte("ok"), 0o644)
}
func testWorkspaceRoot(s dockerFrontendSpec) string {
	return filepath.Join(frontendTestRoot, s.OutputSubpath)
}

var frontendTestRoot string

func TestDockerFrontendBuildRunnerUsesReleaseScopedVolumeSpec(t *testing.T) {
	frontendTestRoot = t.TempDir()
	e := &fakeDockerFrontendEngine{}
	r, err := newDockerFrontendBuildRunner(e, DockerFrontendBuildRunnerConfig{Image: "builder:fixed", WorkspaceVolume: "induforge-workspaces", WorkspaceRoot: frontendTestRoot})
	if err != nil {
		t.Fatal(err)
	}
	v := Version{ID: "22222222-2222-4222-8222-222222222222"}
	out, err := r.BuildProjectFrontend(context.Background(), Project{ID: releaseSourceProjectID}, v)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(e.spec.Name, releaseSourceProjectID) || !strings.Contains(e.spec.Name, "-") || e.spec.WorkspaceSubpath != releaseSourceProjectID+"/workspace" || e.spec.CacheSubpath != releaseSourceProjectID+"/cache" || e.spec.OutputSubpath != releaseSourceProjectID+"/release-builds/"+v.ID {
		t.Fatalf("volume subpath 或一次性名称错误: %#v", e.spec)
	}
	if out.DistDir != filepath.Join(frontendTestRoot, e.spec.OutputSubpath, "dist") {
		t.Fatal("返回的 control 可见 dist 路径错误")
	}
	if len(e.spec.Command) != 1 || !strings.Contains(e.spec.Command[0], "cp -a /opt/induforge/pnpm-store/. /cache/pnpm-store/") || !strings.Contains(e.spec.Command[0], "cp -a /opt/induforge/corepack/. /tmp/corepack/") {
		t.Fatal("构建器没有把审核的只读离线 store 复制到可写 cache")
	}
	if out.Cleanup() == nil {
		if _, err := os.Stat(filepath.Dir(out.DistDir)); !os.IsNotExist(err) {
			t.Fatal("cleanup 未删除本次 release 输出")
		}
	}
}
func TestDockerFrontendBuildRunnerCleansFailedOutput(t *testing.T) {
	frontendTestRoot = t.TempDir()
	r, err := newDockerFrontendBuildRunner(&fakeDockerFrontendEngine{err: errors.New("failed")}, DockerFrontendBuildRunnerConfig{Image: "image", WorkspaceVolume: "volume", WorkspaceRoot: frontendTestRoot})
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.BuildProjectFrontend(context.Background(), Project{ID: releaseSourceProjectID}, Version{ID: "22222222-2222-4222-8222-222222222222"})
	if err == nil {
		t.Fatal("失败构建不得成功")
	}
	if _, err := os.Stat(filepath.Join(frontendTestRoot, releaseSourceProjectID, "release-builds", "22222222-2222-4222-8222-222222222222")); !os.IsNotExist(err) {
		t.Fatal("失败构建输出未清理")
	}
}

func TestDockerFrontendClientUsesLockedDownVolumeContainer(t *testing.T) {
	var create map[string]any
	var removed bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/version":
			_ = json.NewEncoder(w).Encode(map[string]string{"ApiVersion": "1.47"})
		case "/v1.47/containers/create":
			_ = json.NewDecoder(r.Body).Decode(&create)
			_ = json.NewEncoder(w).Encode(map[string]string{"Id": "build"})
		case "/v1.47/containers/build/start":
			w.WriteHeader(204)
		case "/v1.47/containers/build/wait":
			_ = json.NewEncoder(w).Encode(map[string]any{"StatusCode": 0})
		case "/v1.47/containers/build":
			removed = true
			w.WriteHeader(204)
		default:
			t.Fatalf("unexpected %s", r.URL.Path)
		}
	}))
	defer server.Close()
	c, err := newDockerFrontendEngine(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	err = c.Run(context.Background(), dockerFrontendSpec{Name: "build", Image: "image", Volume: "volume", WorkspaceSubpath: "p/workspace", CacheSubpath: "p/cache", OutputSubpath: "p/release-builds/r", MemoryBytes: 123, NanoCPUs: 456, Command: []string{"corepack pnpm install --frozen-lockfile --offline"}})
	if err != nil {
		t.Fatal(err)
	}
	h := create["HostConfig"].(map[string]any)
	if h["NetworkMode"] != "none" || h["ReadonlyRootfs"] != true || h["Memory"] != float64(123) || h["NanoCPUs"] != float64(456) || h["PidsLimit"] != float64(256) {
		t.Fatalf("安全限制缺失: %#v", h)
	}
	ms := h["Mounts"].([]any)
	if len(ms) != 3 || ms[0].(map[string]any)["Type"] != "volume" || ms[0].(map[string]any)["ReadOnly"] != true || ms[0].(map[string]any)["Target"] != "/source" {
		t.Fatalf("volume 挂载错误: %#v", ms)
	}
	if got := create["Entrypoint"].([]any); len(got) != 2 || got[0] != "/bin/sh" || got[1] != "-ec" {
		t.Fatalf("必须覆盖 IDE 镜像 entrypoint: %#v", got)
	}
	if command := create["Cmd"].([]any); len(command) != 1 || !strings.Contains(command[0].(string), "pnpm install --frozen-lockfile --offline") {
		t.Fatalf("构建命令不安全或不完整: %#v", command)
	}
	env := strings.Join(anyStrings(create["Env"].([]any)), " ")
	for _, expected := range []string{"PNPM_CONFIG_STORE_DIR=/cache/pnpm-store", "XDG_CACHE_HOME=/cache", "HOME=/tmp/home", "COREPACK_HOME=/tmp/corepack"} {
		if !strings.Contains(env, expected) {
			t.Fatalf("缺少固定构建环境 %s: %s", expected, env)
		}
	}
	if strings.Contains(strings.ToLower(string(mustJSON(t, create))), "secret") || strings.Contains(strings.ToLower(string(mustJSON(t, create))), "token") {
		t.Fatal("Docker 构建 payload 不得携带密钥或 token")
	}
	if !removed {
		t.Fatal("一次性容器未清理")
	}
}

func anyStrings(values []any) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if text, ok := value.(string); ok {
			result = append(result, text)
		}
	}
	return result
}
func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestDockerFrontendClientTruncatesFailureLogsAndCleansContainer(t *testing.T) {
	var removed bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/version":
			_ = json.NewEncoder(w).Encode(map[string]string{"ApiVersion": "1.47"})
		case "/v1.47/containers/create":
			_ = json.NewEncoder(w).Encode(map[string]string{"Id": "build"})
		case "/v1.47/containers/build/start":
			w.WriteHeader(204)
		case "/v1.47/containers/build/wait":
			_ = json.NewEncoder(w).Encode(map[string]any{"StatusCode": 1})
		case "/v1.47/containers/build/logs":
			_, _ = w.Write([]byte(strings.Repeat("x", frontendBuildLogLimit+10)))
		case "/v1.47/containers/build":
			removed = true
			w.WriteHeader(204)
		default:
			t.Fatalf("unexpected %s", r.URL.Path)
		}
	}))
	defer server.Close()
	c, _ := newDockerFrontendEngine(server.URL)
	err := c.Run(context.Background(), dockerFrontendSpec{Name: "build", Image: "image", Volume: "v", WorkspaceSubpath: "p/workspace", CacheSubpath: "p/cache", OutputSubpath: "p/out"})
	if err == nil || !strings.Contains(err.Error(), "日志已截断") || !removed {
		t.Fatalf("非零退出必须读取截断日志并清理: err=%v removed=%v", err, removed)
	}
}
