package deployment

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

type fakeDockerFrontendEngine struct {
	spec dockerFrontendSpec
	err  error
}

func (e *fakeDockerFrontendEngine) Run(_ context.Context, spec dockerFrontendSpec) error {
	e.spec = spec
	if e.err != nil {
		return e.err
	}
	if err := os.MkdirAll(filepath.Join(spec.Output, "dist"), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(spec.Output, "dist", "index.html"), []byte("ok"), 0o644)
}

func TestDockerFrontendBuildRunnerUsesFixedIsolatedSpec(t *testing.T) {
	workspace, outputRoot := t.TempDir(), t.TempDir()
	engine := &fakeDockerFrontendEngine{}
	runner, err := newDockerFrontendBuildRunner(engine, "induforge/frontend-builder:fixed", outputRoot, 0)
	if err != nil {
		t.Fatal(err)
	}
	dist, err := runner.BuildProjectFrontend(context.Background(), Project{ID: releaseSourceProjectID, WorkspacePath: workspace})
	if err != nil {
		t.Fatal(err)
	}
	if dist != filepath.Join(outputRoot, releaseSourceProjectID, "dist") || engine.spec.Image != "induforge/frontend-builder:fixed" || engine.spec.Workspace != workspace || engine.spec.Output != filepath.Join(outputRoot, releaseSourceProjectID) {
		t.Fatalf("受控构建目录或镜像不正确: %#v dist=%s", engine.spec, dist)
	}
	if got, want := engine.spec.Command, []string{"pnpm", "run", "build", "--", "--outDir", "/output/dist"}; len(got) != len(want) {
		t.Fatalf("固定构建命令不正确: %v", got)
	}
}

func TestDockerFrontendBuildRunnerCleansFailedOutput(t *testing.T) {
	workspace, outputRoot := t.TempDir(), t.TempDir()
	runner, err := newDockerFrontendBuildRunner(&fakeDockerFrontendEngine{err: errors.New("failed")}, "image", outputRoot, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.BuildProjectFrontend(context.Background(), Project{ID: releaseSourceProjectID, WorkspacePath: workspace}); err == nil {
		t.Fatal("构建失败不应返回 dist")
	}
	if _, err := os.Stat(filepath.Join(outputRoot, releaseSourceProjectID)); !os.IsNotExist(err) {
		t.Fatalf("失败构建残留输出目录: %v", err)
	}
}

func TestDockerFrontendClientCreatesLockedDownBuildContainer(t *testing.T) {
	var create map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/version":
			_ = json.NewEncoder(w).Encode(map[string]string{"ApiVersion": "1.47"})
		case "/v1.47/containers/create":
			_ = json.NewDecoder(r.Body).Decode(&create)
			_ = json.NewEncoder(w).Encode(map[string]string{"Id": "build-id"})
		case "/v1.47/containers/build-id/start", "/v1.47/containers/build-id":
			w.WriteHeader(http.StatusNoContent)
		case "/v1.47/containers/build-id/wait":
			_ = json.NewEncoder(w).Encode(map[string]any{"StatusCode": 0})
		default:
			t.Fatalf("unexpected Docker path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client, err := newDockerFrontendEngine(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Run(context.Background(), dockerFrontendSpec{Name: "build", Image: "fixed-image", Workspace: "/trusted/workspace", Output: "/trusted/output", Command: []string{"pnpm", "run", "build"}}); err != nil {
		t.Fatal(err)
	}
	host := create["HostConfig"].(map[string]any)
	if host["NetworkMode"] != "none" || host["ReadonlyRootfs"] != true || host["PidsLimit"] != float64(256) {
		t.Fatalf("构建容器隔离配置不正确: %#v", host)
	}
	if caps := host["CapDrop"].([]any); len(caps) != 1 || caps[0] != "ALL" {
		t.Fatalf("构建容器未删除全部 capabilities: %#v", caps)
	}
	mounts := host["Mounts"].([]any)
	if len(mounts) != 2 || mounts[0].(map[string]any)["ReadOnly"] != true || mounts[0].(map[string]any)["Target"] != "/workspace" || mounts[1].(map[string]any)["Target"] != "/output" {
		t.Fatalf("构建容器挂载不正确: %#v", mounts)
	}
}
