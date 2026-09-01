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
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DockerFrontendBuildRunnerConfig 的镜像必须由平台运维固定提供，不能来自工程文件。
type DockerFrontendBuildRunnerConfig struct {
	DockerHost string
	Image      string
	OutputRoot string
	Timeout    time.Duration
}

type dockerFrontendSpec struct {
	Name, Image, Workspace, Output string
	Command                        []string
}

type dockerFrontendEngine interface {
	Run(context.Context, dockerFrontendSpec) error
}

// DockerFrontendBuildRunner 在隔离容器中执行固定的 pnpm 构建命令。工程源码只读挂载，
// 输出仅可写入受控 OutputRoot，且不提供网络、特权、Docker Socket 或环境密钥。
type DockerFrontendBuildRunner struct {
	engine     dockerFrontendEngine
	image      string
	outputRoot string
	timeout    time.Duration
}

func NewDockerFrontendBuildRunner(config DockerFrontendBuildRunnerConfig) (*DockerFrontendBuildRunner, error) {
	image, outputRoot := strings.TrimSpace(config.Image), strings.TrimSpace(config.OutputRoot)
	if image == "" || outputRoot == "" {
		return nil, fmt.Errorf("前端构建镜像或输出目录未配置")
	}
	absolute, err := filepath.Abs(outputRoot)
	if err != nil {
		return nil, fmt.Errorf("解析前端构建输出目录失败: %w", err)
	}
	engine, err := newDockerFrontendEngine(config.DockerHost)
	if err != nil {
		return nil, err
	}
	return newDockerFrontendBuildRunner(engine, image, absolute, config.Timeout)
}

func newDockerFrontendBuildRunner(engine dockerFrontendEngine, image, outputRoot string, timeout time.Duration) (*DockerFrontendBuildRunner, error) {
	if engine == nil || strings.TrimSpace(image) == "" || strings.TrimSpace(outputRoot) == "" {
		return nil, fmt.Errorf("受控前端构建器依赖不完整")
	}
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	return &DockerFrontendBuildRunner{engine: engine, image: image, outputRoot: outputRoot, timeout: timeout}, nil
}

func (r *DockerFrontendBuildRunner) BuildProjectFrontend(ctx context.Context, project Project) (string, error) {
	workspace, err := filepath.Abs(project.WorkspacePath)
	if err != nil {
		return "", fmt.Errorf("解析工程工作空间失败: %w", err)
	}
	info, err := os.Lstat(workspace)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("工程工作空间必须是非链接目录")
	}
	if err := os.MkdirAll(r.outputRoot, 0o755); err != nil {
		return "", fmt.Errorf("创建前端构建输出根目录失败: %w", err)
	}
	output := filepath.Join(r.outputRoot, project.ID)
	if !withinDirectory(r.outputRoot, output) {
		return "", fmt.Errorf("前端构建输出路径越界")
	}
	// 输出根由 runner 独占，清理上次的同工程临时结果不会影响工程源码或其他项目。
	if err := os.RemoveAll(output); err != nil {
		return "", fmt.Errorf("清理上次前端构建结果失败: %w", err)
	}
	if err := os.MkdirAll(output, 0o755); err != nil {
		return "", fmt.Errorf("创建前端构建输出目录失败: %w", err)
	}
	buildCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	spec := dockerFrontendSpec{
		Name: "induforge-release-build-" + project.ID, Image: r.image, Workspace: workspace, Output: output,
		Command: []string{"pnpm", "run", "build", "--", "--outDir", "/output/dist"},
	}
	if err := r.engine.Run(buildCtx, spec); err != nil {
		_ = os.RemoveAll(output)
		return "", fmt.Errorf("受控前端构建失败: %w", err)
	}
	dist := filepath.Join(output, "dist")
	if info, err := os.Lstat(dist); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		_ = os.RemoveAll(output)
		return "", fmt.Errorf("受控前端构建未生成 dist 目录")
	}
	return dist, nil
}

type dockerFrontendClient struct {
	baseURL    string
	httpClient *http.Client
	mu         sync.Mutex
	apiVersion string
}

func newDockerFrontendEngine(rawHost string) (*dockerFrontendClient, error) {
	host := strings.TrimSpace(rawHost)
	if host == "" {
		host = "unix:///var/run/docker.sock"
	}
	parsed, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("解析 Docker Host 失败: %w", err)
	}
	transport, baseURL := &http.Transport{}, host
	switch parsed.Scheme {
	case "unix":
		if parsed.Path == "" {
			return nil, fmt.Errorf("Docker Unix Socket 路径为空")
		}
		transport.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", parsed.Path)
		}
		baseURL = "http://docker"
	case "tcp":
		parsed.Scheme, baseURL = "http", strings.TrimRight(parsed.String(), "/")
	case "http", "https":
		baseURL = strings.TrimRight(parsed.String(), "/")
	default:
		return nil, fmt.Errorf("不支持的 Docker Host 协议: %s", parsed.Scheme)
	}
	return &dockerFrontendClient{baseURL: baseURL, httpClient: &http.Client{Transport: transport, Timeout: 30 * time.Second}}, nil
}

func (c *dockerFrontendClient) Run(ctx context.Context, spec dockerFrontendSpec) error {
	type mount struct {
		Type, Source, Target string
		ReadOnly             bool `json:"ReadOnly"`
	}
	type hostConfig struct {
		NetworkMode    string            `json:"NetworkMode"`
		ReadonlyRootfs bool              `json:"ReadonlyRootfs"`
		CapDrop        []string          `json:"CapDrop"`
		SecurityOpt    []string          `json:"SecurityOpt"`
		PidsLimit      int64             `json:"PidsLimit"`
		Mounts         []mount           `json:"Mounts"`
		Tmpfs          map[string]string `json:"Tmpfs"`
	}
	body := struct {
		Image      string            `json:"Image"`
		Cmd        []string          `json:"Cmd"`
		User       string            `json:"User"`
		WorkingDir string            `json:"WorkingDir"`
		Labels     map[string]string `json:"Labels"`
		HostConfig hostConfig        `json:"HostConfig"`
	}{
		Image: spec.Image, Cmd: spec.Command, User: "1000:1000", WorkingDir: "/workspace",
		Labels: map[string]string{"com.induforge.managed": "true", "com.induforge.role": "release-frontend-build"},
		HostConfig: hostConfig{NetworkMode: "none", ReadonlyRootfs: true, CapDrop: []string{"ALL"}, SecurityOpt: []string{"no-new-privileges:true"}, PidsLimit: 256,
			Mounts: []mount{{Type: "bind", Source: spec.Workspace, Target: "/workspace", ReadOnly: true}, {Type: "bind", Source: spec.Output, Target: "/output"}}, Tmpfs: map[string]string{"/tmp": "rw,noexec,nosuid,size=64m"}},
	}
	var created struct {
		ID string `json:"Id"`
	}
	if err := c.request(ctx, http.MethodPost, "/containers/create?name="+url.QueryEscape(spec.Name), body, &created); err != nil {
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
	var waited struct {
		StatusCode int `json:"StatusCode"`
		Error      *struct {
			Message string `json:"Message"`
		} `json:"Error"`
	}
	if err := c.request(ctx, http.MethodPost, "/containers/"+url.PathEscape(created.ID)+"/wait", nil, &waited); err != nil {
		return err
	}
	if waited.StatusCode != 0 {
		message := ""
		if waited.Error != nil {
			message = strings.TrimSpace(waited.Error.Message)
		}
		return fmt.Errorf("前端构建容器退出码 %d: %s", waited.StatusCode, message)
	}
	return nil
}

func (c *dockerFrontendClient) request(ctx context.Context, method, endpoint string, body, output any) error {
	version, err := c.version(ctx)
	if err != nil {
		return err
	}
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+"/"+version+endpoint, reader)
	if err != nil {
		return err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("访问 Docker Engine 失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode/100 != 2 {
		var payload struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&payload)
		return fmt.Errorf("Docker Engine 返回 %d: %s", response.StatusCode, payload.Message)
	}
	if output != nil && response.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(output); err != nil {
			return fmt.Errorf("解析 Docker Engine 响应失败: %w", err)
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
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/version", nil)
	if err != nil {
		return "", err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("读取 Docker Engine 版本失败: %w", err)
	}
	defer response.Body.Close()
	var payload struct {
		APIVersion string `json:"ApiVersion"`
	}
	if response.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload) != nil || strings.TrimSpace(payload.APIVersion) == "" {
		return "", fmt.Errorf("读取 Docker Engine 版本失败")
	}
	c.apiVersion = "v" + strings.TrimPrefix(payload.APIVersion, "v")
	return c.apiVersion, nil
}

func withinDirectory(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
