package deployment

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/indu-forge/dev_core/internal/releasebuilder"
	"github.com/klauspost/compress/zstd"
)

const (
	defaultReleaseSourceResponseLimit int64 = 8 << 20
	defaultReleaseSourceArchiveLimit  int64 = 64 << 20
)

// ProjectFrontendBuildRunner 只返回受控构建目录。它的实现将由后续受信构建器提供，
// 聚合器不执行工作空间中的任意脚本。
type ProjectFrontendBuildRunner interface {
	BuildProjectFrontend(context.Context, Project, Version) (FrontendBuildOutput, error)
}

type FrontendBuildOutput struct {
	DistDir string
	Cleanup func() error
}

type ProjectReleaseSourceBuilderConfig struct {
	DataServiceURL       string
	BuilderID            string
	MaxResponseBytes     int64
	MaxArchiveInputBytes int64
	TenantBindingEnsurer interface {
		EnsureProjectTenantBinding(context.Context, string, string) error
	}
}

// ProjectReleaseSourceBuilder 汇聚数据域的正式运行工件、受控前端构建结果与运行资源。
// 它不保存 Authorization，且所有归档都按固定顺序和固定 tar 元数据生成。
type ProjectReleaseSourceBuilder struct {
	dataServiceURL       string
	builderID            string
	responseLimit        int64
	archiveLimit         int64
	httpClient           *http.Client
	frontend             ProjectFrontendBuildRunner
	tenantBindingEnsurer interface {
		EnsureProjectTenantBinding(context.Context, string, string) error
	}
}

func NewProjectReleaseSourceBuilder(config ProjectReleaseSourceBuilderConfig, frontend ProjectFrontendBuildRunner) (*ProjectReleaseSourceBuilder, error) {
	endpoint := strings.TrimRight(strings.TrimSpace(config.DataServiceURL), "/")
	parsed, err := url.ParseRequestURI(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("DATA_SERVICE_URL 无效")
	}
	if strings.TrimSpace(config.BuilderID) == "" || frontend == nil {
		return nil, fmt.Errorf("正式 ReleaseSource 构建器配置不完整")
	}
	if config.MaxResponseBytes <= 0 {
		config.MaxResponseBytes = defaultReleaseSourceResponseLimit
	}
	if config.MaxArchiveInputBytes <= 0 {
		config.MaxArchiveInputBytes = defaultReleaseSourceArchiveLimit
	}
	return &ProjectReleaseSourceBuilder{
		dataServiceURL: endpoint, builderID: config.BuilderID, responseLimit: config.MaxResponseBytes,
		archiveLimit: config.MaxArchiveInputBytes, httpClient: &http.Client{Timeout: 20 * time.Second}, frontend: frontend,
		tenantBindingEnsurer: config.TenantBindingEnsurer,
	}, nil
}

func (b *ProjectReleaseSourceBuilder) SetHTTPClient(client *http.Client) {
	if client != nil {
		b.httpClient = client
	}
}

// DevelopmentEngineRequirements 从当前数据域 runtime artifact 派生开发部署需求。
// 该预检刻意跳过前端构建、Release 签名和对象存储，保持开发更新前的快速只读边界。
func (b *ProjectReleaseSourceBuilder) DevelopmentEngineRequirements(ctx context.Context, project Project, authorization string) ([]string, error) {
	if b == nil || b.httpClient == nil || b.tenantBindingEnsurer == nil {
		return nil, fmt.Errorf("开发引擎需求预检未初始化")
	}
	if err := b.tenantBindingEnsurer.EnsureProjectTenantBinding(ctx, project.ID, project.TenantID); err != nil {
		return nil, fmt.Errorf("同步数据服务项目租户绑定失败: %w", err)
	}
	artifact, _, err := b.fetchRuntimeArtifact(ctx, project.ID, authorization)
	if err != nil {
		return nil, err
	}
	return releasebuilder.DeriveEngineRequirements(artifact), nil
}

func (b *ProjectReleaseSourceBuilder) BuildReleaseSource(ctx context.Context, project Project, version Version, authorization string) (ReleaseSource, error) {
	if b == nil || b.frontend == nil || b.httpClient == nil {
		return ReleaseSource{}, fmt.Errorf("正式 ReleaseSource 构建器未初始化")
	}
	if b.tenantBindingEnsurer == nil {
		return ReleaseSource{}, fmt.Errorf("数据服务项目租户绑定客户端未配置")
	}
	if err := b.tenantBindingEnsurer.EnsureProjectTenantBinding(ctx, project.ID, project.TenantID); err != nil {
		return ReleaseSource{}, fmt.Errorf("同步数据服务项目租户绑定失败: %w", err)
	}
	artifact, runtimeJSON, err := b.fetchRuntimeArtifact(ctx, project.ID, authorization)
	if err != nil {
		return ReleaseSource{}, err
	}
	var collector, collectorSourceSnapshot []byte
	if containsEngine(releasebuilder.DeriveEngineRequirements(artifact), "collector") {
		collector, collectorSourceSnapshot, err = b.fetchCollectorArtifact(ctx, project, version, artifact, authorization)
		if err != nil {
			return ReleaseSource{}, err
		}
	}
	frontend, err := b.frontend.BuildProjectFrontend(ctx, project, version)
	if err != nil {
		return ReleaseSource{}, fmt.Errorf("构建工程前端失败: %w", err)
	}
	if frontend.Cleanup != nil {
		defer frontend.Cleanup()
	}
	client, err := b.packFrontend(frontend.DistDir)
	if err != nil {
		return ReleaseSource{}, err
	}
	runtime, err := b.packRuntime(project.WorkspacePath, runtimeJSON)
	if err != nil {
		return ReleaseSource{}, err
	}
	sourceRevision := sourceRevision(client, runtime, collector)
	sbom, err := json.Marshal(map[string]any{
		"bomFormat": "CycloneDX", "specVersion": "1.5", "version": 1,
		"metadata": map[string]any{"component": map[string]any{"type": "application", "name": project.Code, "version": sourceRevision}},
	})
	if err != nil {
		return ReleaseSource{}, fmt.Errorf("生成 Release SBOM 失败: %w", err)
	}
	return ReleaseSource{
		Client: client, Runtime: runtime, Collector: collector, CollectorSourceSnapshot: collectorSourceSnapshot, SBOM: sbom,
		ResourceRecommendation: []byte(`{"cpu":"250m","memory":"256Mi"}`),
		HealthContract:         []byte(`{"schemaVersion":"release-health.v1","readiness":"http"}`),
		SchemaPlan:             []byte(`{"schemaVersion":"release-schema-plan.v1","changes":[]}`),
		ProjectDocument:        artifact, SourceRevision: sourceRevision, BuilderID: b.builderID,
	}, nil
}

// fetchCollectorArtifact 只接受 data service 生成且携带自身完整性元数据的 bytes；
// 不支持 URL、路径或调用方提供的工件，避免 ReleaseBuilder 变成下载器。
func (b *ProjectReleaseSourceBuilder) fetchCollectorArtifact(ctx context.Context, project Project, version Version, snapshot map[string]any, authorization string) ([]byte, []byte, error) {
	body, err := json.Marshal(map[string]any{"tenantId": project.TenantID, "projectId": project.ID, "releaseId": version.ID, "revision": 1, "sourceSnapshot": snapshot})
	if err != nil {
		return nil, nil, fmt.Errorf("构造采集工件请求失败")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, b.dataServiceURL+"/api/v1/data/projects/"+project.ID+"/collector-artifact", bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("创建采集工件请求失败")
	}
	request.Header.Set("Content-Type", "application/json")
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	response, err := b.httpClient.Do(request)
	if err != nil {
		return nil, nil, fmt.Errorf("生成采集工件失败")
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, b.responseLimit+1))
	if err != nil || int64(len(raw)) > b.responseLimit {
		return nil, nil, fmt.Errorf("读取采集工件响应失败")
	}
	var envelope struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(raw, &envelope) != nil || response.StatusCode/100 != 2 || envelope.Code != 0 {
		return nil, nil, fmt.Errorf("采集工件响应无效")
	}
	var result struct {
		SchemaVersion    string          `json:"schemaVersion"`
		ProjectID        string          `json:"projectId"`
		ArtifactRevision int64           `json:"artifactRevision"`
		SHA256           string          `json:"sha256"`
		Size             int64           `json:"size"`
		Artifact         json.RawMessage `json:"artifact"`
	}
	if json.Unmarshal(envelope.Data, &result) != nil || result.SchemaVersion != "collector-runtime-artifact.v1" || result.ProjectID != project.ID || result.ArtifactRevision < 1 || result.Size != int64(len(result.Artifact)) || len(result.Artifact) == 0 || int64(len(result.Artifact)) > b.archiveLimit {
		return nil, nil, fmt.Errorf("采集工件元数据无效")
	}
	sum := sha256.Sum256(result.Artifact)
	if result.SHA256 != "sha256:"+hex.EncodeToString(sum[:]) {
		return nil, nil, fmt.Errorf("采集工件摘要不匹配")
	}
	var artifact struct {
		SchemaVersion    string `json:"schemaVersion"`
		ProjectID        string `json:"projectId"`
		ArtifactRevision int64  `json:"artifactRevision"`
	}
	if json.Unmarshal(result.Artifact, &artifact) != nil || artifact.SchemaVersion != result.SchemaVersion || artifact.ProjectID != project.ID || artifact.ArtifactRevision != result.ArtifactRevision {
		return nil, nil, fmt.Errorf("采集工件内容无效")
	}
	// application_versions.manifest 使用 JSONB 保存。签名前必须把嵌套 artifact 与
	// sourceSnapshot 都规范化，否则 JSONB 读回后的字段重排会让两层字节摘要失效。
	canonicalArtifact, err := canonicalJSON(result.Artifact)
	if err != nil {
		return nil, nil, fmt.Errorf("规范化采集工件失败")
	}
	result.Artifact = canonicalArtifact
	result.Size = int64(len(canonicalArtifact))
	artifactSum := sha256.Sum256(canonicalArtifact)
	result.SHA256 = "sha256:" + hex.EncodeToString(artifactSum[:])
	snapshotJSON, err := json.Marshal(result)
	if err != nil {
		return nil, nil, fmt.Errorf("规范化采集工件快照失败")
	}
	canonicalSnapshot, err := canonicalJSON(snapshotJSON)
	if err != nil {
		return nil, nil, fmt.Errorf("规范化采集工件快照失败")
	}
	packed, err := packReleaseSourceArchive(map[string][]byte{"collector-runtime-artifact.json": canonicalArtifact})
	if err != nil || int64(len(packed)) > b.archiveLimit {
		return nil, nil, fmt.Errorf("归档采集工件失败")
	}
	// 规范化只改变 JSON 表示并重算已验证内容的摘要，不接受客户端字段或改变采集语义。
	return packed, canonicalSnapshot, nil
}

func canonicalJSON(raw []byte) ([]byte, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return json.Marshal(value)
}

func (b *ProjectReleaseSourceBuilder) fetchRuntimeArtifact(ctx context.Context, projectID, authorization string) (map[string]any, []byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, b.dataServiceURL+"/api/v1/data/projects/"+projectID+"/artifact", nil)
	if err != nil {
		return nil, nil, fmt.Errorf("创建数据域工件请求失败: %w", err)
	}
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	response, err := b.httpClient.Do(request)
	if err != nil {
		return nil, nil, fmt.Errorf("读取数据域工件失败: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, b.responseLimit+1))
	if err != nil {
		return nil, nil, fmt.Errorf("读取数据域工件响应失败: %w", err)
	}
	if int64(len(body)) > b.responseLimit {
		return nil, nil, fmt.Errorf("数据域工件响应超过大小限制")
	}
	var envelope struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil || response.StatusCode/100 != 2 || envelope.Code != 0 || len(envelope.Data) == 0 {
		return nil, nil, fmt.Errorf("数据域工件响应无效")
	}
	var artifact map[string]any
	if err := json.Unmarshal(envelope.Data, &artifact); err != nil {
		return nil, nil, fmt.Errorf("数据域工件不是合法 JSON: %w", err)
	}
	if err := validateRuntimeArtifact(projectID, artifact); err != nil {
		return nil, nil, err
	}
	runtimeJSON, err := json.Marshal(artifact)
	if err != nil {
		return nil, nil, fmt.Errorf("规范化数据域工件失败: %w", err)
	}
	return artifact, runtimeJSON, nil
}

func validateRuntimeArtifact(projectID string, artifact map[string]any) error {
	if artifact["schemaVersion"] != "runtime-project-artifact.v1" || artifact["projectArtifactVersion"] != "1.0" || artifact["projectId"] != projectID {
		return fmt.Errorf("数据域工件 schema 或 projectId 不匹配")
	}
	if _, err := uuid.Parse(projectID); err != nil {
		return fmt.Errorf("数据域工件 projectId 无效")
	}
	if _, ok := artifact["generatedAt"].(string); !ok {
		return fmt.Errorf("数据域工件缺少 generatedAt")
	}
	for _, key := range []string{"dataPoints", "computeUnits", "alarmItems"} {
		if _, ok := artifact[key].([]any); !ok {
			return fmt.Errorf("数据域工件字段 %s 不符合 runtime-project-artifact.v1", key)
		}
	}
	return nil
}

func (b *ProjectReleaseSourceBuilder) packFrontend(dist string) ([]byte, error) {
	files, err := readTreeFiles(dist, "", b.archiveLimit)
	if err != nil {
		return nil, fmt.Errorf("读取前端构建目录失败: %w", err)
	}
	if _, ok := files["index.html"]; !ok {
		return nil, fmt.Errorf("前端构建目录缺少 index.html: %s", buildOutputSummary(dist))
	}
	return packReleaseSourceArchive(files)
}

func (b *ProjectReleaseSourceBuilder) packRuntime(workspacePath string, runtimeJSON []byte) ([]byte, error) {
	files := map[string][]byte{"runtime-project-artifact.json": runtimeJSON}
	var total int64 = int64(len(runtimeJSON))
	if total > b.archiveLimit {
		return nil, fmt.Errorf("运行工件超过大小限制")
	}
	for _, directory := range []string{"displays", "symbols", ".induforge/scenes"} {
		root := filepath.Join(workspacePath, filepath.FromSlash(directory))
		items, err := readTreeFiles(root, directory, b.archiveLimit-total)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("读取工程运行资源失败: %w", err)
		}
		for name, content := range items {
			files[name] = content
			total += int64(len(content))
		}
	}
	return packReleaseSourceArchive(files)
}

func readTreeFiles(root, prefix string, limit int64) (map[string][]byte, error) {
	info, err := os.Lstat(root)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("归档根目录必须是非链接目录")
	}
	files := map[string][]byte{}
	var total int64
	err = filepath.WalkDir(root, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, current)
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("归档目录包含符号链接: %s", filepath.ToSlash(relative))
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("归档目录包含非普通文件: %s", filepath.ToSlash(relative))
		}
		if relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return fmt.Errorf("归档文件路径越界")
		}
		total += info.Size()
		if total > limit {
			return fmt.Errorf("归档输入超过大小限制")
		}
		content, err := os.ReadFile(current)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(filepath.Join(prefix, relative))
		files[name] = content
		return nil
	})
	return files, err
}

func packReleaseSourceArchive(files map[string][]byte) ([]byte, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("归档不能为空")
	}
	names := make([]string, 0, len(files))
	for name := range files {
		clean := filepath.ToSlash(filepath.Clean(name))
		if clean == "." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
			return nil, fmt.Errorf("归档文件路径非法: %s", name)
		}
		names = append(names, clean)
	}
	sort.Strings(names)
	var output bytes.Buffer
	encoder, err := zstd.NewWriter(&output, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return nil, err
	}
	writer := tar.NewWriter(encoder)
	for _, name := range names {
		content := files[name]
		header := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(content)), ModTime: time.Unix(0, 0).UTC(), Format: tar.FormatUSTAR}
		if err := writer.WriteHeader(header); err != nil {
			_ = encoder.Close()
			return nil, err
		}
		if _, err := writer.Write(content); err != nil {
			_ = encoder.Close()
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		_ = encoder.Close()
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func sourceRevision(client, runtime, collector []byte) string {
	hash := sha256.New()
	_, _ = hash.Write([]byte("client\x00"))
	_, _ = hash.Write(client)
	_, _ = hash.Write([]byte("runtime\x00"))
	_, _ = hash.Write(runtime)
	_, _ = hash.Write([]byte("collector\x00"))
	_, _ = hash.Write(collector)
	return "git:" + hex.EncodeToString(hash.Sum(nil))
}

func containsEngine(engines []string, want string) bool {
	for _, engine := range engines {
		if engine == want {
			return true
		}
	}
	return false
}
