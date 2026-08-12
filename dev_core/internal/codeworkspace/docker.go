package codeworkspace

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	ErrContainerNotFound = errors.New("代码工作区容器不存在")
	ErrContainerConflict = errors.New("代码工作区容器名称冲突")
)

type Mount struct {
	Source   string
	Target   string
	Subpath  string
	ReadOnly bool
}

type ContainerSpec struct {
	Name          string
	Image         string
	Command       []string
	User          string
	WorkingDir    string
	Environment   []string
	Mounts        []Mount
	ContainerPort string
	BindHost      string
	Labels        map[string]string
}

type ContainerState struct {
	Name     string
	Status   string
	Running  bool
	Health   string
	HostPort string
	Labels   map[string]string
}

type Engine interface {
	Inspect(context.Context, string) (ContainerState, error)
	Create(context.Context, ContainerSpec) error
	Start(context.Context, string) error
	Stop(context.Context, string) error
	Remove(context.Context, string) error
}

type DockerClient struct {
	baseURL    string
	httpClient *http.Client
	mu         sync.Mutex
	apiVersion string
}

func NewDockerClient(rawHost string) (*DockerClient, error) {
	host := strings.TrimSpace(rawHost)
	if host == "" {
		host = "unix:///var/run/docker.sock"
	}
	parsed, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("解析 Docker Host 失败: %w", err)
	}

	transport := &http.Transport{}
	baseURL := host
	switch parsed.Scheme {
	case "unix":
		socketPath := parsed.Path
		if socketPath == "" {
			return nil, fmt.Errorf("Docker Unix Socket 路径为空")
		}
		transport.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		}
		baseURL = "http://docker"
	case "tcp":
		parsed.Scheme = "http"
		baseURL = strings.TrimRight(parsed.String(), "/")
	case "http", "https":
		baseURL = strings.TrimRight(parsed.String(), "/")
	default:
		return nil, fmt.Errorf("不支持的 Docker Host 协议: %s", parsed.Scheme)
	}

	return &DockerClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
	}, nil
}

func (c *DockerClient) Inspect(ctx context.Context, name string) (ContainerState, error) {
	var response struct {
		Name   string `json:"Name"`
		Config struct {
			Labels map[string]string `json:"Labels"`
		} `json:"Config"`
		State struct {
			Status  string `json:"Status"`
			Running bool   `json:"Running"`
			Health  *struct {
				Status string `json:"Status"`
			} `json:"Health"`
		} `json:"State"`
		NetworkSettings struct {
			Ports map[string][]struct {
				HostPort string `json:"HostPort"`
			} `json:"Ports"`
		} `json:"NetworkSettings"`
	}
	if err := c.request(ctx, http.MethodGet, "/containers/"+url.PathEscape(name)+"/json", nil, &response); err != nil {
		return ContainerState{}, err
	}
	state := ContainerState{
		Name:    strings.TrimPrefix(response.Name, "/"),
		Status:  response.State.Status,
		Running: response.State.Running,
		Labels:  response.Config.Labels,
	}
	if response.State.Health != nil {
		state.Health = response.State.Health.Status
	}
	if bindings := response.NetworkSettings.Ports["3000/tcp"]; len(bindings) > 0 {
		state.HostPort = bindings[0].HostPort
	}
	return state, nil
}

func (c *DockerClient) Create(ctx context.Context, spec ContainerSpec) error {
	type portBinding struct {
		HostIP   string `json:"HostIp,omitempty"`
		HostPort string `json:"HostPort,omitempty"`
	}
	type restartPolicy struct {
		Name string `json:"Name"`
	}
	type volumeOptions struct {
		Subpath string `json:"Subpath,omitempty"`
	}
	type dockerMount struct {
		Type          string        `json:"Type"`
		Source        string        `json:"Source"`
		Target        string        `json:"Target"`
		ReadOnly      bool          `json:"ReadOnly"`
		VolumeOptions volumeOptions `json:"VolumeOptions"`
	}
	type hostConfig struct {
		PortBindings map[string][]portBinding `json:"PortBindings"`
		Restart      restartPolicy            `json:"RestartPolicy"`
		Mounts       []dockerMount            `json:"Mounts"`
	}
	body := struct {
		Image        string              `json:"Image"`
		Cmd          []string            `json:"Cmd"`
		User         string              `json:"User,omitempty"`
		WorkingDir   string              `json:"WorkingDir"`
		Env          []string            `json:"Env,omitempty"`
		Labels       map[string]string   `json:"Labels"`
		ExposedPorts map[string]struct{} `json:"ExposedPorts"`
		HostConfig   hostConfig          `json:"HostConfig"`
	}{
		Image: spec.Image, Cmd: spec.Command, User: spec.User, WorkingDir: spec.WorkingDir,
		Env: spec.Environment, Labels: spec.Labels,
		ExposedPorts: map[string]struct{}{spec.ContainerPort: {}},
		HostConfig: hostConfig{
			PortBindings: map[string][]portBinding{spec.ContainerPort: {{HostIP: spec.BindHost}}},
			Restart:      restartPolicy{Name: "no"},
		},
	}
	for _, mount := range spec.Mounts {
		body.HostConfig.Mounts = append(body.HostConfig.Mounts, dockerMount{
			Type: "volume", Source: mount.Source, Target: mount.Target, ReadOnly: mount.ReadOnly,
			VolumeOptions: volumeOptions{Subpath: mount.Subpath},
		})
	}
	path := "/containers/create?name=" + url.QueryEscape(spec.Name)
	return c.request(ctx, http.MethodPost, path, body, nil)
}

func (c *DockerClient) Start(ctx context.Context, name string) error {
	return c.request(ctx, http.MethodPost, "/containers/"+url.PathEscape(name)+"/start", nil, nil)
}

func (c *DockerClient) Stop(ctx context.Context, name string) error {
	err := c.request(ctx, http.MethodPost, "/containers/"+url.PathEscape(name)+"/stop?t=10", nil, nil)
	if errors.Is(err, ErrContainerNotFound) {
		return nil
	}
	return err
}

func (c *DockerClient) Remove(ctx context.Context, name string) error {
	err := c.request(ctx, http.MethodDelete, "/containers/"+url.PathEscape(name)+"?force=1", nil, nil)
	if errors.Is(err, ErrContainerNotFound) {
		return nil
	}
	return err
}

func (c *DockerClient) request(ctx context.Context, method, path string, body, output any) error {
	version, err := c.version(ctx)
	if err != nil {
		return err
	}
	var reader io.Reader
	if body != nil {
		encoded, encodeErr := json.Marshal(body)
		if encodeErr != nil {
			return encodeErr
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+"/"+version+path, reader)
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
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var payload struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload)
		switch response.StatusCode {
		case http.StatusNotFound:
			return ErrContainerNotFound
		case http.StatusConflict:
			return fmt.Errorf("%w: %s", ErrContainerConflict, payload.Message)
		default:
			return fmt.Errorf("Docker Engine 返回 %d: %s", response.StatusCode, payload.Message)
		}
	}
	if output != nil && response.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(response.Body).Decode(output); err != nil {
			return fmt.Errorf("解析 Docker Engine 响应失败: %w", err)
		}
	}
	return nil
}

func (c *DockerClient) version(ctx context.Context) (string, error) {
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
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("读取 Docker Engine 版本失败: HTTP %d", response.StatusCode)
	}
	var payload struct {
		APIVersion string `json:"ApiVersion"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("解析 Docker Engine 版本失败: %w", err)
	}
	if strings.TrimSpace(payload.APIVersion) == "" {
		return "", fmt.Errorf("Docker Engine 未返回 API 版本")
	}
	c.apiVersion = "v" + strings.TrimPrefix(payload.APIVersion, "v")
	return c.apiVersion, nil
}
