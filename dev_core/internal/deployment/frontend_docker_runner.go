package deployment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const frontendBuildLogLimit = 64 << 10

type DockerFrontendBuildRunnerConfig struct {
	DockerHost, Image, WorkspaceVolume, WorkspaceRoot string
	Timeout                                           time.Duration
	MemoryBytes, NanoCPUs                             int64
}

type dockerFrontendSpec struct {
	Name, Image, Volume, WorkspaceSubpath, CacheSubpath, OutputSubpath string
	Command                                                            []string
	MemoryBytes, NanoCPUs                                              int64
}
type dockerFrontendEngine interface {
	Run(context.Context, dockerFrontendSpec) error
}

// DockerFrontendBuildRunner 只使用共享工作空间卷的受限 subpath，绝不把 control
// 容器内路径作为 Docker bind source，也不复用 code-server。
type DockerFrontendBuildRunner struct {
	engine                       dockerFrontendEngine
	image, volume, workspaceRoot string
	timeout                      time.Duration
	memoryBytes, nanoCPUs        int64
}

func NewDockerFrontendBuildRunner(config DockerFrontendBuildRunnerConfig) (*DockerFrontendBuildRunner, error) {
	engine, err := newDockerFrontendEngine(config.DockerHost)
	if err != nil {
		return nil, err
	}
	return newDockerFrontendBuildRunner(engine, config)
}
func newDockerFrontendBuildRunner(engine dockerFrontendEngine, config DockerFrontendBuildRunnerConfig) (*DockerFrontendBuildRunner, error) {
	config.Image, config.WorkspaceVolume, config.WorkspaceRoot = strings.TrimSpace(config.Image), strings.TrimSpace(config.WorkspaceVolume), strings.TrimSpace(config.WorkspaceRoot)
	if engine == nil || config.Image == "" || config.WorkspaceVolume == "" || config.WorkspaceRoot == "" {
		return nil, fmt.Errorf("受控前端构建器依赖不完整")
	}
	root, err := filepath.Abs(config.WorkspaceRoot)
	if err != nil {
		return nil, err
	}
	if config.Timeout <= 0 {
		config.Timeout = 5 * time.Minute
	}
	if config.MemoryBytes <= 0 {
		config.MemoryBytes = 1 << 30
	}
	if config.NanoCPUs <= 0 {
		config.NanoCPUs = 1_000_000_000
	}
	return &DockerFrontendBuildRunner{engine: engine, image: config.Image, volume: config.WorkspaceVolume, workspaceRoot: root, timeout: config.Timeout, memoryBytes: config.MemoryBytes, nanoCPUs: config.NanoCPUs}, nil
}
func (r *DockerFrontendBuildRunner) BuildProjectFrontend(ctx context.Context, project Project, version Version) (FrontendBuildOutput, error) {
	if _, err := uuid.Parse(project.ID); err != nil {
		return FrontendBuildOutput{}, fmt.Errorf("工程 ID 无效")
	}
	if _, err := uuid.Parse(version.ID); err != nil {
		return FrontendBuildOutput{}, fmt.Errorf("Release ID 无效")
	}
	projectRoot := filepath.Join(r.workspaceRoot, project.ID)
	outputRoot := filepath.Join(projectRoot, "release-builds", version.ID)
	if !withinDirectory(r.workspaceRoot, projectRoot) || !withinDirectory(projectRoot, outputRoot) {
		return FrontendBuildOutput{}, fmt.Errorf("前端构建输出路径越界")
	}
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		return FrontendBuildOutput{}, err
	}
	if err := os.RemoveAll(outputRoot); err != nil {
		return FrontendBuildOutput{}, fmt.Errorf("清理上次前端构建结果失败: %w", err)
	}
	if err := os.MkdirAll(outputRoot, 0o755); err != nil {
		return FrontendBuildOutput{}, err
	}
	buildCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	buildID := uuid.NewString()
	spec := dockerFrontendSpec{Name: "induforge-release-build-" + project.ID + "-" + buildID, Image: r.image, Volume: r.volume, WorkspaceSubpath: path.Join(project.ID, "workspace"), CacheSubpath: path.Join(project.ID, "cache"), OutputSubpath: path.Join(project.ID, "release-builds", version.ID), MemoryBytes: r.memoryBytes, NanoCPUs: r.nanoCPUs, Command: []string{"mkdir -p /tmp/home /tmp/corepack /build/src /build/dist; cp -a /source/. /build/src/; cd /build/src; corepack pnpm install --frozen-lockfile --offline; corepack pnpm run build -- --outDir /build/dist"}}
	if err := r.engine.Run(buildCtx, spec); err != nil {
		_ = os.RemoveAll(outputRoot)
		return FrontendBuildOutput{}, fmt.Errorf("受控前端构建失败: %w", err)
	}
	dist := filepath.Join(outputRoot, "dist")
	info, err := os.Lstat(dist)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		_ = os.RemoveAll(outputRoot)
		return FrontendBuildOutput{}, fmt.Errorf("受控前端构建未生成 dist 目录")
	}
	return FrontendBuildOutput{DistDir: dist, Cleanup: func() error { return os.RemoveAll(outputRoot) }}, nil
}

type dockerFrontendClient struct {
	baseURL    string
	httpClient *http.Client
	mu         sync.Mutex
	apiVersion string
}

func newDockerFrontendEngine(raw string) (*dockerFrontendClient, error) {
	host := strings.TrimSpace(raw)
	if host == "" {
		host = "unix:///var/run/docker.sock"
	}
	u, err := url.Parse(host)
	if err != nil {
		return nil, err
	}
	tr := &http.Transport{}
	base := host
	switch u.Scheme {
	case "unix":
		if u.Path == "" {
			return nil, fmt.Errorf("Docker Unix Socket 路径为空")
		}
		tr.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", u.Path)
		}
		base = "http://docker"
	case "tcp":
		u.Scheme = "http"
		base = strings.TrimRight(u.String(), "/")
	case "http", "https":
		base = strings.TrimRight(u.String(), "/")
	default:
		return nil, fmt.Errorf("不支持的 Docker Host 协议: %s", u.Scheme)
	}
	return &dockerFrontendClient{baseURL: base, httpClient: &http.Client{Transport: tr, Timeout: 30 * time.Second}}, nil
}
func (c *dockerFrontendClient) Run(ctx context.Context, s dockerFrontendSpec) error {
	type volOpt struct {
		Subpath string `json:"Subpath"`
	}
	type mount struct {
		Type, Source, Target string
		ReadOnly             bool   `json:"ReadOnly"`
		VolumeOptions        volOpt `json:"VolumeOptions"`
	}
	type host struct {
		NetworkMode                 string `json:"NetworkMode"`
		ReadonlyRootfs              bool   `json:"ReadonlyRootfs"`
		CapDrop, SecurityOpt        []string
		PidsLimit, Memory, NanoCPUs int64
		Mounts                      []mount
		Tmpfs                       map[string]string
	}
	body := struct {
		Image            string
		Entrypoint, Cmd  []string
		Env              []string
		User, WorkingDir string
		Labels           map[string]string
		HostConfig       host
	}{Image: s.Image, Entrypoint: []string{"/bin/sh", "-ec"}, Cmd: s.Command, Env: []string{"PNPM_CONFIG_STORE_DIR=/cache/pnpm-store", "XDG_CACHE_HOME=/cache", "HOME=/tmp/home", "COREPACK_HOME=/tmp/corepack"}, User: "1000:1000", WorkingDir: "/build/src", Labels: map[string]string{"com.induforge.managed": "true", "com.induforge.role": "release-frontend-build"}, HostConfig: host{NetworkMode: "none", ReadonlyRootfs: true, CapDrop: []string{"ALL"}, SecurityOpt: []string{"no-new-privileges:true"}, PidsLimit: 256, Memory: s.MemoryBytes, NanoCPUs: s.NanoCPUs, Tmpfs: map[string]string{"/tmp": "rw,noexec,nosuid,size=64m"}, Mounts: []mount{{Type: "volume", Source: s.Volume, Target: "/source", ReadOnly: true, VolumeOptions: volOpt{Subpath: s.WorkspaceSubpath}}, {Type: "volume", Source: s.Volume, Target: "/cache", VolumeOptions: volOpt{Subpath: s.CacheSubpath}}, {Type: "volume", Source: s.Volume, Target: "/build", VolumeOptions: volOpt{Subpath: s.OutputSubpath}}}}}
	var created struct {
		ID string `json:"Id"`
	}
	if err := c.request(ctx, http.MethodPost, "/containers/create?name="+url.QueryEscape(s.Name), body, &created); err != nil {
		return err
	}
	if created.ID == "" {
		return fmt.Errorf("Docker 未返回构建容器 ID")
	}
	defer func() {
		_ = c.request(context.Background(), http.MethodDelete, "/containers/"+url.PathEscape(created.ID)+"?force=1", nil, nil)
	}()
	if err := c.request(ctx, http.MethodPost, "/containers/"+url.PathEscape(created.ID)+"/start", nil, nil); err != nil {
		return err
	}
	var wait struct {
		StatusCode int `json:"StatusCode"`
	}
	if err := c.request(ctx, http.MethodPost, "/containers/"+url.PathEscape(created.ID)+"/wait", nil, &wait); err != nil {
		return err
	}
	if wait.StatusCode != 0 {
		logs := c.logs(ctx, created.ID)
		return fmt.Errorf("前端构建容器退出码 %d: %s", wait.StatusCode, logs)
	}
	return nil
}
func (c *dockerFrontendClient) logs(ctx context.Context, id string) string {
	v, err := c.version(ctx)
	if err != nil {
		return ""
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/"+v+"/containers/"+url.PathEscape(id)+"/logs?stdout=1&stderr=1", nil)
	if err != nil {
		return ""
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, frontendBuildLogLimit+1))
	if len(raw) > frontendBuildLogLimit {
		raw = append(raw[:frontendBuildLogLimit], []byte("…(日志已截断)")...)
	}
	return strings.TrimSpace(string(raw))
}
func (c *dockerFrontendClient) request(ctx context.Context, m, e string, body, out any) error {
	v, err := c.version(ctx)
	if err != nil {
		return err
	}
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, m, c.baseURL+"/"+v+e, rd)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("访问 Docker Engine 失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("Docker Engine 返回 %d", resp.StatusCode)
	}
	if out != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out); err != nil {
			return err
		}
	}
	return nil
}
func (c *dockerFrontendClient) version(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.apiVersion != "" {
		return c.apiVersion, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/version", nil)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var p struct {
		APIVersion string `json:"ApiVersion"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&p) != nil || p.APIVersion == "" {
		return "", fmt.Errorf("读取 Docker Engine 版本失败")
	}
	c.apiVersion = "v" + strings.TrimPrefix(p.APIVersion, "v")
	return c.apiVersion, nil
}
func withinDirectory(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
