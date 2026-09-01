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
	BuildProjectFrontend(context.Context, Project) (string, error)
}

type ProjectReleaseSourceBuilderConfig struct {
	DataServiceURL       string
	BuilderID            string
	MaxResponseBytes     int64
	MaxArchiveInputBytes int64
}

// ProjectReleaseSourceBuilder 汇聚数据域的正式运行工件、受控前端构建结果与运行资源。
// 它不保存 Authorization，且所有归档都按固定顺序和固定 tar 元数据生成。
type ProjectReleaseSourceBuilder struct {
	dataServiceURL string
	builderID      string
	responseLimit  int64
	archiveLimit   int64
	httpClient     *http.Client
	frontend       ProjectFrontendBuildRunner
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
	}, nil
}

func (b *ProjectReleaseSourceBuilder) SetHTTPClient(client *http.Client) {
	if client != nil {
		b.httpClient = client
	}
}

func (b *ProjectReleaseSourceBuilder) BuildReleaseSource(ctx context.Context, project Project, authorization string) (ReleaseSource, error) {
	if b == nil || b.frontend == nil || b.httpClient == nil {
		return ReleaseSource{}, fmt.Errorf("正式 ReleaseSource 构建器未初始化")
	}
	artifact, runtimeJSON, err := b.fetchRuntimeArtifact(ctx, project.ID, authorization)
	if err != nil {
		return ReleaseSource{}, err
	}
	if containsEngine(releasebuilder.DeriveEngineRequirements(artifact), "collector") {
		return ReleaseSource{}, fmt.Errorf("工程启用了采集引擎，但当前未提供受信 Collector Release 工件")
	}
	dist, err := b.frontend.BuildProjectFrontend(ctx, project)
	if err != nil {
		return ReleaseSource{}, fmt.Errorf("构建工程前端失败: %w", err)
	}
	client, err := b.packFrontend(dist)
	if err != nil {
		return ReleaseSource{}, err
	}
	runtime, err := b.packRuntime(project.WorkspacePath, runtimeJSON)
	if err != nil {
		return ReleaseSource{}, err
	}
	sourceRevision := sourceRevision(client, runtime)
	sbom, err := json.Marshal(map[string]any{
		"bomFormat": "CycloneDX", "specVersion": "1.5", "version": 1,
		"metadata": map[string]any{"component": map[string]any{"type": "application", "name": project.Code, "version": sourceRevision}},
	})
	if err != nil {
		return ReleaseSource{}, fmt.Errorf("生成 Release SBOM 失败: %w", err)
	}
	return ReleaseSource{
		Client: client, Runtime: runtime, SBOM: sbom,
		ResourceRecommendation: []byte(`{"cpu":"250m","memory":"256Mi"}`),
		HealthContract:         []byte(`{"schemaVersion":"release-health.v1","readiness":"http"}`),
		SchemaPlan:             []byte(`{"schemaVersion":"release-schema-plan.v1","changes":[]}`),
		ProjectDocument:        artifact, SourceRevision: sourceRevision, BuilderID: b.builderID,
	}, nil
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
		return nil, fmt.Errorf("前端构建目录缺少 index.html")
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

func sourceRevision(client, runtime []byte) string {
	hash := sha256.New()
	_, _ = hash.Write([]byte("client\x00"))
	_, _ = hash.Write(client)
	_, _ = hash.Write([]byte("runtime\x00"))
	_, _ = hash.Write(runtime)
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
