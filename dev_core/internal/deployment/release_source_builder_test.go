package deployment

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/indu-forge/dev_core/internal/releasebuilder"
	"github.com/klauspost/compress/zstd"
)

const releaseSourceProjectID = "11111111-1111-4111-8111-111111111111"

type fakeFrontendRunner struct {
	directory string
	err       error
}

func (r fakeFrontendRunner) BuildProjectFrontend(context.Context, Project, Version) (FrontendBuildOutput, error) {
	return FrontendBuildOutput{DistDir: r.directory}, r.err
}

func TestProjectReleaseSourceBuilderBuildsDeterministicAuthorizedSource(t *testing.T) {
	workspace, dist := releaseSourceDirectories(t)
	artifact := runtimeArtifact(releaseSourceProjectID, "calc.output", true)
	var gotAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "ok", "data": artifact})
	}))
	defer server.Close()
	builder := releaseSourceBuilder(t, server.URL, fakeFrontendRunner{directory: dist})
	project := Project{ID: releaseSourceProjectID, Code: "demo", WorkspacePath: workspace}
	version := Version{ID: "22222222-2222-4222-8222-222222222222", Version: "1.0.0"}
	first, err := builder.BuildReleaseSource(context.Background(), project, version, "Bearer forwarded-token")
	if err != nil {
		t.Fatal(err)
	}
	if gotAuthorization != "Bearer forwarded-token" {
		t.Fatalf("数据域调用未保留受限身份: auth=%q", gotAuthorization)
	}
	second, err := builder.BuildReleaseSource(context.Background(), project, version, "Bearer forwarded-token")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Client, second.Client) || !bytes.Equal(first.Runtime, second.Runtime) || first.SourceRevision != second.SourceRevision {
		t.Fatal("受控输入必须生成确定性 ReleaseSource")
	}
	if strings.Contains(string(first.Client), "forwarded-token") || strings.Contains(string(first.Runtime), "forwarded-token") || !strings.HasPrefix(first.SourceRevision, "git:") {
		t.Fatal("转发身份不得泄漏到制品，且 sourceRevision 必须是内容摘要")
	}
	engines := releasebuilder.DeriveEngineRequirements(first.ProjectDocument)
	for _, engine := range []string{"base", "compute", "alarm"} {
		if !containsEngine(engines, engine) {
			t.Fatalf("正式运行工件未保留引擎能力 %s: %v", engine, engines)
		}
	}
	clientFiles := unpackReleaseSource(t, first.Client)
	if string(clientFiles["index.html"]) != "<main>demo</main>" {
		t.Fatalf("前端 dist 未进入 client 归档: %v", mapKeys(clientFiles))
	}
	if _, ok := clientFiles["dist/index.html"]; ok {
		t.Fatal("client 归档不得额外包含 dist 根目录")
	}
	runtimeFiles := unpackReleaseSource(t, first.Runtime)
	for _, file := range []string{"runtime-project-artifact.json", "displays/panel.json", "symbols/pump.svg", ".induforge/scenes/main.json"} {
		if _, ok := runtimeFiles[file]; !ok {
			t.Fatalf("运行工件缺少正式资源 %s: %v", file, mapKeys(runtimeFiles))
		}
	}
	if _, ok := runtimeFiles[".induforge/context/secret.md"]; ok {
		t.Fatal("工作空间上下文不得进入运行制品")
	}
	if !json.Valid(first.SBOM) || !json.Valid(first.HealthContract) || !json.Valid(first.ResourceRecommendation) || !json.Valid(first.SchemaPlan) {
		t.Fatal("ReleaseSource 附属正式 JSON 无效")
	}
}

func TestProjectReleaseSourceBuilderRejectsInvalidDataServiceArtifacts(t *testing.T) {
	workspace, dist := releaseSourceDirectories(t)
	project := Project{ID: releaseSourceProjectID, Code: "demo", WorkspacePath: workspace}
	for _, test := range []struct {
		name   string
		status int
		body   any
	}{
		{name: "envelope error", status: http.StatusOK, body: map[string]any{"code": 1001, "msg": "denied", "data": map[string]any{}}},
		{name: "wrong project", status: http.StatusOK, body: map[string]any{"code": 0, "data": runtimeArtifact("22222222-2222-4222-8222-222222222222", "manual.input", false)}},
		{name: "wrong schema", status: http.StatusOK, body: map[string]any{"code": 0, "data": map[string]any{"schemaVersion": "unknown", "projectId": releaseSourceProjectID}}},
		{name: "http error", status: http.StatusBadGateway, body: map[string]any{"code": 0, "data": runtimeArtifact(releaseSourceProjectID, "manual.input", false)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
				_ = json.NewEncoder(w).Encode(test.body)
			}))
			defer server.Close()
			if _, err := releaseSourceBuilder(t, server.URL, fakeFrontendRunner{directory: dist}).BuildReleaseSource(context.Background(), project, Version{ID: "22222222-2222-4222-8222-222222222222", Version: "1.0.0"}, ""); err == nil {
				t.Fatal("非法数据域响应不得进入正式制品")
			}
		})
	}
}

func TestProjectReleaseSourceBuilderLimitsDataServiceResponse(t *testing.T) {
	workspace, dist := releaseSourceDirectories(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("x"), 1024))
	}))
	defer server.Close()
	builder := releaseSourceBuilder(t, server.URL, fakeFrontendRunner{directory: dist})
	builder.responseLimit = 32
	if _, err := builder.BuildReleaseSource(context.Background(), Project{ID: releaseSourceProjectID, Code: "demo", WorkspacePath: workspace}, Version{ID: "22222222-2222-4222-8222-222222222222", Version: "1.0.0"}, ""); err == nil {
		t.Fatal("超限数据域响应不得进入正式制品")
	}
}

func TestProjectReleaseSourceBuilderRejectsCollectorWithoutTrustedArtifact(t *testing.T) {
	workspace, dist := releaseSourceDirectories(t)
	server := artifactServer(t, runtimeArtifact(releaseSourceProjectID, "collector.point", false))
	defer server.Close()
	_, err := releaseSourceBuilder(t, server.URL, fakeFrontendRunner{directory: dist}).BuildReleaseSource(context.Background(), Project{ID: releaseSourceProjectID, Code: "demo", WorkspacePath: workspace}, Version{ID: "22222222-2222-4222-8222-222222222222", Version: "1.0.0"}, "")
	if err == nil || !strings.Contains(err.Error(), "采集") {
		t.Fatalf("采集引擎没有受信制品必须失败关闭: %v", err)
	}
}

func TestProjectReleaseSourceBuilderIncludesTrustedCollectorArtifact(t *testing.T) {
	workspace, dist := releaseSourceDirectories(t)
	runtime := runtimeArtifact(releaseSourceProjectID, "collector.point", false)
	collector, _ := json.Marshal(map[string]any{"schemaVersion": "collector-runtime-artifact.v1", "artifactId": "collector-a", "artifactRevision": 1, "projectId": releaseSourceProjectID, "collectorVersion": "1.0.0", "connections": []any{}, "pointMappings": []any{}, "wal": map[string]any{}})
	sum := sha256.Sum256(collector)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"schemaVersion": "collector-runtime-artifact.v1", "projectId": releaseSourceProjectID, "artifactRevision": 1, "sha256": "sha256:" + hex.EncodeToString(sum[:]), "size": len(collector), "artifact": json.RawMessage(collector)}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": runtime})
	}))
	defer server.Close()
	source, err := releaseSourceBuilder(t, server.URL, fakeFrontendRunner{directory: dist}).BuildReleaseSource(context.Background(), Project{ID: releaseSourceProjectID, TenantID: "tenant-a", Code: "demo", WorkspacePath: workspace}, Version{ID: "22222222-2222-4222-8222-222222222222", Version: "1.0.0"}, "")
	if err != nil || len(source.Collector) == 0 {
		t.Fatalf("collector source: %v", err)
	}
	if _, ok := unpackReleaseSource(t, source.Collector)["collector-runtime-artifact.json"]; !ok {
		t.Fatal("collector artifact not packed")
	}
	if !containsEngine(releasebuilder.DeriveEngineRequirements(source.ProjectDocument), "collector") {
		t.Fatal("collector capability lost")
	}
	// 模拟 application_versions.manifest 的 JSONB 往返：嵌套对象重编码后，
	// sourceSnapshot 及其内部 artifact 的摘要必须继续成立。
	var snapshot map[string]any
	if err = json.Unmarshal(source.CollectorSourceSnapshot, &snapshot); err != nil {
		t.Fatal(err)
	}
	roundTripped, err := json.Marshal(snapshot)
	if err != nil || !bytes.Equal(roundTripped, source.CollectorSourceSnapshot) {
		t.Fatalf("collector sourceSnapshot 不是 JSONB 稳定表示: %v", err)
	}
	var checked struct {
		SHA256   string          `json:"sha256"`
		Size     int             `json:"size"`
		Artifact json.RawMessage `json:"artifact"`
	}
	if err = json.Unmarshal(roundTripped, &checked); err != nil || checked.Size != len(checked.Artifact) {
		t.Fatalf("collector sourceSnapshot 元数据失效: %v", err)
	}
	artifactSum := sha256.Sum256(checked.Artifact)
	if checked.SHA256 != "sha256:"+hex.EncodeToString(artifactSum[:]) {
		t.Fatal("collector sourceSnapshot 内部 artifact 摘要失效")
	}
}

func TestProjectReleaseSourceBuilderRejectsUnsafeFrontendDirectory(t *testing.T) {
	root := t.TempDir()
	builder := releaseSourceBuilder(t, "http://data.invalid", fakeFrontendRunner{})
	if _, err := builder.packFrontend(root); err == nil {
		t.Fatal("缺少 index.html 的 dist 不得打包")
	}
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "index.html"), filepath.Join(root, "linked.html")); err != nil {
		t.Fatal(err)
	}
	if _, err := builder.packFrontend(root); err == nil {
		t.Fatal("dist 符号链接不得进入 Release")
	}
	if err := os.Remove(filepath.Join(root, "linked.html")); err != nil {
		t.Fatal(err)
	}
	builder.archiveLimit = 1
	if _, err := builder.packFrontend(root); err == nil {
		t.Fatal("超限 dist 不得进入 Release")
	}
}

func releaseSourceBuilder(t *testing.T, dataURL string, runner ProjectFrontendBuildRunner) *ProjectReleaseSourceBuilder {
	t.Helper()
	builder, err := NewProjectReleaseSourceBuilder(ProjectReleaseSourceBuilderConfig{DataServiceURL: dataURL, BuilderID: "release-source-builder-v1", TenantBindingEnsurer: noopTenantBindingEnsurer{}}, runner)
	if err != nil {
		t.Fatal(err)
	}
	return builder
}

type noopTenantBindingEnsurer struct{}

func (noopTenantBindingEnsurer) EnsureProjectTenantBinding(context.Context, string, string) error {
	return nil
}

func releaseSourceDirectories(t *testing.T) (string, string) {
	t.Helper()
	workspace, dist := filepath.Join(t.TempDir(), "workspace"), filepath.Join(t.TempDir(), "dist")
	for path, content := range map[string]string{
		filepath.Join(dist, "index.html"):                              "<main>demo</main>",
		filepath.Join(workspace, "displays", "panel.json"):             "{}",
		filepath.Join(workspace, "symbols", "pump.svg"):                "<svg/>",
		filepath.Join(workspace, ".induforge", "scenes", "main.json"):  "{}",
		filepath.Join(workspace, ".induforge", "context", "secret.md"): "must-not-package",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return workspace, dist
}

func runtimeArtifact(projectID, sourceType string, compute bool) map[string]any {
	points := []any{}
	if sourceType != "" {
		points = append(points, map[string]any{"sourceType": sourceType})
	}
	units := []any{}
	if compute {
		units = append(units, map[string]any{"id": "compute"})
	}
	return map[string]any{"schemaVersion": "runtime-project-artifact.v1", "projectArtifactVersion": "1.0", "projectId": projectID, "generatedAt": time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC).Format(time.RFC3339), "dataPoints": points, "computeUnits": units, "alarmItems": []any{map[string]any{"id": "alarm"}}}
}

func artifactServer(t *testing.T, artifact map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "ok", "data": artifact})
	}))
}

func unpackReleaseSource(t *testing.T, archive []byte) map[string][]byte {
	t.Helper()
	decoder, err := zstd.NewReader(bytes.NewReader(archive))
	if err != nil {
		t.Fatal(err)
	}
	defer decoder.Close()
	reader := tar.NewReader(decoder)
	files := map[string][]byte{}
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return files
		}
		if err != nil {
			t.Fatal(err)
		}
		files[header.Name], err = io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func mapKeys(values map[string][]byte) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func TestCapturedCollectorIsPackedWithVerifiableFrozenSnapshot(t *testing.T) {
	workspace, dist := releaseSourceDirectories(t)
	artifact := runtimeArtifact(releaseSourceProjectID, "collector.point", false)
	runtimeJSON, _ := json.Marshal(artifact)
	collector, _ := json.Marshal(map[string]any{"schemaVersion": "collector-runtime-artifact.v1", "artifactId": "collector-captured", "artifactRevision": 1, "projectId": releaseSourceProjectID, "collectorVersion": "1.0.0", "connections": []any{}, "pointMappings": []any{}, "wal": map[string]any{}})
	builder := releaseSourceBuilder(t, "http://unused.invalid", fakeFrontendRunner{directory: dist})
	source, err := builder.buildReleaseSourceFromCaptured(context.Background(), Project{ID: releaseSourceProjectID, Code: "demo", WorkspacePath: workspace}, Version{ID: "22222222-2222-4222-8222-222222222222", Version: "1.0.0"}, artifact, runtimeJSON, collector, collector)
	if err != nil {
		t.Fatal(err)
	}
	payload := unpackReleaseSource(t, source.Collector)["collector-runtime-artifact.json"]
	var frozen struct {
		Artifact json.RawMessage `json:"artifact"`
		SHA256   string          `json:"sha256"`
		Size     int             `json:"size"`
	}
	if err = json.Unmarshal(source.CollectorSourceSnapshot, &frozen); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	if !bytes.Equal(payload, frozen.Artifact) || frozen.Size != len(payload) || frozen.SHA256 != "sha256:"+hex.EncodeToString(sum[:]) {
		t.Fatal("冻结采集快照与归档不一致")
	}
}
