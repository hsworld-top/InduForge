package deployment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
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
	BootstrapProjectID, BootstrapTemplateID           string
	Timeout                                           time.Duration
	MemoryBytes, NanoCPUs                             int64
}

type dockerFrontendSpec struct {
	Name, Image, Volume, WorkspaceSubpath, CacheSubpath, OutputSubpath string
	BootstrapTemplateID                                                string
	WorkspaceReadOnly                                                  bool
	Command                                                            []string
	MemoryBytes, NanoCPUs                                              int64
}
type dockerFrontendEngine interface {
	Run(context.Context, dockerFrontendSpec) error
}

// DockerFrontendBuildRunner 只使用共享工作空间卷的受限 subpath，绝不把 control
// 容器内路径作为 Docker bind source，也不复用 code-server。
type DockerFrontendBuildRunner struct {
	engine                                  dockerFrontendEngine
	image, volume, workspaceRoot            string
	bootstrapProjectID, bootstrapTemplateID string
	timeout                                 time.Duration
	memoryBytes, nanoCPUs                   int64
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
	config.BootstrapProjectID, config.BootstrapTemplateID = strings.TrimSpace(config.BootstrapProjectID), strings.TrimSpace(config.BootstrapTemplateID)
	if engine == nil || config.Image == "" || config.WorkspaceVolume == "" || config.WorkspaceRoot == "" {
		return nil, fmt.Errorf("受控前端构建器依赖不完整")
	}
	if (config.BootstrapProjectID == "") != (config.BootstrapTemplateID == "") {
		return nil, fmt.Errorf("内置工程源码初始化配置不完整")
	}
	if config.BootstrapTemplateID != "" && (strings.Contains(config.BootstrapTemplateID, "/") || config.BootstrapTemplateID == "." || config.BootstrapTemplateID == "..") {
		return nil, fmt.Errorf("内置工程模板标识无效")
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
	return &DockerFrontendBuildRunner{engine: engine, image: config.Image, volume: config.WorkspaceVolume, workspaceRoot: root, bootstrapProjectID: config.BootstrapProjectID, bootstrapTemplateID: config.BootstrapTemplateID, timeout: config.Timeout, memoryBytes: config.MemoryBytes, nanoCPUs: config.NanoCPUs}, nil
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
	if err := r.bootstrapBuiltinWorkspace(ctx, project); err != nil {
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
	// 大型内容寻址 files 始终留在镜像只读层；每次构建只在 tmpfs 复制 27MB 级索引和
	// 离线策略元数据，依赖安装结果与本次输出一起清理，绝不占用工程共享卷。
	spec := dockerFrontendSpec{Name: "induforge-release-build-" + project.ID + "-" + buildID, Image: r.image, Volume: r.volume, WorkspaceSubpath: path.Join(project.ID, "workspace"), OutputSubpath: path.Join(project.ID, "release-builds", version.ID), WorkspaceReadOnly: true, MemoryBytes: r.memoryBytes, NanoCPUs: r.nanoCPUs, Command: []string{"mkdir -p /tmp/home /tmp/corepack /tmp/pnpm-cache /tmp/pnpm-store/v11/files /output/src /output/dist; cp -a /opt/induforge/corepack/. /tmp/corepack/; cp /opt/induforge/pnpm-store/v11/index.db /tmp/pnpm-store/v11/; chmod u+rw /tmp/pnpm-store/v11/index.db; cp -a /opt/induforge/pnpm-store/v11/projects /tmp/pnpm-store/v11/projects; chmod -R u+rwX /tmp/pnpm-store/v11/projects; cd /opt/induforge/pnpm-store/v11; find files -type f -exec sh -c 'for rel do mkdir -p \"/tmp/pnpm-store/v11/$(dirname \"$rel\")\"; ln -s \"/opt/induforge/pnpm-store/v11/$rel\" \"/tmp/pnpm-store/v11/$rel\"; done' sh {} +; if test -d /opt/induforge/pnpm-store/v11/file+; then cp -a /opt/induforge/pnpm-store/v11/file+ /tmp/pnpm-store/v11/file+; chmod -R u+rwX /tmp/pnpm-store/v11/file+; fi; cp -a /source/. /output/src; cd /output/src; corepack pnpm install --config.trust-lockfile=true --frozen-lockfile --offline --package-import-method=copy; corepack pnpm exec vite build --outDir /output/dist --emptyOutDir"}}
	if err := r.engine.Run(buildCtx, spec); err != nil {
		_ = os.RemoveAll(outputRoot)
		return FrontendBuildOutput{}, fmt.Errorf("受控前端构建失败: %w", err)
	}
	dist := filepath.Join(outputRoot, "dist")
	info, err := os.Lstat(dist)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		_ = os.RemoveAll(outputRoot)
		return FrontendBuildOutput{}, fmt.Errorf("受控前端构建未生成 dist 目录: %s", buildOutputSummary(outputRoot))
	}
	log.Printf("Release 前端构建输出 projectId=%s versionId=%s items=%s", project.ID, version.ID, buildOutputSummary(outputRoot))
	return FrontendBuildOutput{DistDir: dist, Cleanup: func() error { return os.RemoveAll(outputRoot) }}, nil
}

// buildOutputSummary 仅返回受控输出目录两层内的名称与大小，便于诊断挂载路径，绝不读取源码内容。
func buildOutputSummary(root string) string {
	const limit = 32
	items := make([]string, 0, limit)
	_ = filepath.WalkDir(root, func(name string, entry fs.DirEntry, err error) error {
		if err != nil || name == root || len(items) >= limit {
			return nil
		}
		rel, relErr := filepath.Rel(root, name)
		if relErr != nil || strings.Count(rel, string(filepath.Separator)) > 1 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			items = append(items, rel+"/")
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr == nil {
			items = append(items, fmt.Sprintf("%s:%d", rel, info.Size()))
		}
		return nil
	})
	if len(items) == 0 {
		return "空"
	}
	return strings.Join(items, ",")
}

// bootstrapBuiltinWorkspace 仅为内置教程工程在共享卷为空时落入审核过的官方模板。
// 它运行于无网络、受限的一次性 Release Builder 中，源码随后仍由同一个共享卷供 IDE
// 和正式构建读取，既不复制用户工程，也不会给普通工程选择默认模板。
func (r *DockerFrontendBuildRunner) bootstrapBuiltinWorkspace(ctx context.Context, project Project) error {
	if project.ID != r.bootstrapProjectID || r.bootstrapTemplateID == "" {
		return nil
	}
	workspace := filepath.Join(r.workspaceRoot, project.ID, "workspace")
	staging := filepath.Join(workspace, ".induforge-initialize")
	entries, err := os.ReadDir(workspace)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("检查内置工程源码失败: %w", err)
		}
		entries = nil
	}
	if len(entries) > 0 {
		if _, err := os.Stat(filepath.Join(workspace, "package.json")); err == nil {
			return nil
		}
		if !isPlatformContextOnly(entries, workspace) {
			return fmt.Errorf("内置工程工作区包含未识别文件，拒绝覆盖")
		}
	}
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		return fmt.Errorf("创建内置工程工作区失败: %w", err)
	}
	if err := os.RemoveAll(staging); err != nil {
		return fmt.Errorf("清理内置工程初始化暂存目录失败: %w", err)
	}
	bootstrapCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	buildID := uuid.NewString()
	spec := dockerFrontendSpec{
		Name: "induforge-release-bootstrap-" + project.ID + "-" + buildID, Image: r.image, Volume: r.volume,
		WorkspaceSubpath: path.Join(project.ID, "workspace"), BootstrapTemplateID: r.bootstrapTemplateID,
		WorkspaceReadOnly: false, MemoryBytes: r.memoryBytes, NanoCPUs: r.nanoCPUs,
		Command: []string{bootstrapCommand(r.bootstrapTemplateID)},
	}
	if err := r.engine.Run(bootstrapCtx, spec); err != nil {
		return fmt.Errorf("初始化内置工程源码失败: %w", err)
	}
	if _, err := os.Stat(filepath.Join(workspace, "package.json")); err != nil {
		return fmt.Errorf("内置工程模板初始化后缺少 package.json")
	}
	return nil
}

// isPlatformContextOnly 只认可 K3s 平台挂载的 .induforge/context；任何用户文件或
// 其他元数据均保持 fail-closed，模板初始化不会覆盖非空工程。
func isPlatformContextOnly(entries []os.DirEntry, workspace string) bool {
	for _, entry := range entries {
		if !entry.IsDir() || (entry.Name() != ".induforge" && entry.Name() != ".induforge-initialize") {
			return false
		}
	}
	if len(entries) == 0 {
		return false
	}
	metadata, err := os.ReadDir(filepath.Join(workspace, ".induforge"))
	return err == nil && len(metadata) == 1 && metadata[0].Name() == "context" && metadata[0].IsDir()
}

func bootstrapCommand(templateID string) string {
	// dockerFrontendClient 将工作区子路径固定挂载为 /source；不要使用镜像中不存在的路径，
	// 以保证初始化与正式构建读取同一受控 volume subpath。
	return "test -z \"$(find /source -mindepth 1 -maxdepth 1 ! -name .induforge -print -quit)\"; if test -e /source/.induforge; then test -d /source/.induforge/context && test -z \"$(find /source/.induforge -mindepth 1 -maxdepth 1 ! -name context -print -quit)\"; fi; staging=/source/.induforge-initialize; mkdir \"$staging\"; cp -a /opt/induforge/templates/" + templateID + "/. \"$staging\"/; chmod u+rwx \"$staging/.induforge\"; rm -rf \"$staging/node_modules\" \"$staging/dist\" \"$staging/.pnpm-store\"; mkdir -p \"$staging/.induforge\" /source/.induforge; printf '%s\\n' '{\"version\":1,\"templateId\":\"" + templateID + "\"}' > \"$staging/.induforge/project.json\"; find \"$staging\" -mindepth 1 -maxdepth 1 ! -name .induforge -exec mv {} /source/ \\;; find \"$staging/.induforge\" -mindepth 1 -maxdepth 1 -exec mv {} /source/.induforge/ \\;; rmdir \"$staging/.induforge\" \"$staging\""
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
		// 初始化容器不挂载 /build；统一从必定存在的 tmpfs 启动，命令自行切换到输出卷。
	}{Image: s.Image, Entrypoint: []string{"/bin/sh", "-ec"}, Cmd: s.Command, Env: []string{"PNPM_CONFIG_STORE_DIR=/tmp/pnpm-store", "XDG_CACHE_HOME=/tmp/pnpm-cache", "HOME=/tmp/home", "COREPACK_HOME=/tmp/corepack"}, User: "1000:1000", WorkingDir: "/tmp", Labels: map[string]string{"com.induforge.managed": "true", "com.induforge.role": "release-frontend-build"}, HostConfig: host{NetworkMode: "none", ReadonlyRootfs: true, CapDrop: []string{"ALL"}, SecurityOpt: []string{"no-new-privileges:true"}, PidsLimit: 256, Memory: s.MemoryBytes, NanoCPUs: s.NanoCPUs, Tmpfs: map[string]string{"/tmp": "rw,noexec,nosuid,size=128m"}}}
	body.HostConfig.Mounts = append(body.HostConfig.Mounts, mount{Type: "volume", Source: s.Volume, Target: "/source", ReadOnly: s.WorkspaceReadOnly, VolumeOptions: volOpt{Subpath: s.WorkspaceSubpath}})
	if s.CacheSubpath != "" {
		body.HostConfig.Mounts = append(body.HostConfig.Mounts, mount{Type: "volume", Source: s.Volume, Target: "/cache", VolumeOptions: volOpt{Subpath: s.CacheSubpath}})
	}
	if s.OutputSubpath != "" {
		body.HostConfig.Mounts = append(body.HostConfig.Mounts, mount{Type: "volume", Source: s.Volume, Target: "/output", VolumeOptions: volOpt{Subpath: s.OutputSubpath}})
	}
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
	// 成功时仅记录 stdout/stderr 的受限摘要长度；不把用户构建脚本的原文写入中心日志。
	log.Printf("受控前端构建容器完成 name=%s exitCode=0 logBytes=%d", s.Name, len(c.logs(context.Background(), created.ID)))
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
