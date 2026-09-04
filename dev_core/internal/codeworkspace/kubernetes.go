package codeworkspace

// KubernetesEngine 通过控制面 Pod 的 ServiceAccount 调度正式代码工作区。
// 本地开发环境仍使用 DockerEngine；K3s 中镜像仅存在于 containerd，不能误用宿主 Docker。
import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

type KubernetesConfig struct{ Namespace, WorkspaceRoot string }

type KubernetesEngine struct {
	namespace, workspaceRoot, baseURL, token string
	client                                   *http.Client
}

func NewKubernetesEngine(config KubernetesConfig) (*KubernetesEngine, error) {
	config.Namespace = strings.TrimSpace(config.Namespace)
	config.WorkspaceRoot = strings.TrimSpace(config.WorkspaceRoot)
	if config.Namespace == "" || config.WorkspaceRoot == "" {
		return nil, fmt.Errorf("K3s 代码工作区命名空间和目录不能为空")
	}
	host, port := strings.TrimSpace(os.Getenv("KUBERNETES_SERVICE_HOST")), strings.TrimSpace(os.Getenv("KUBERNETES_SERVICE_PORT_HTTPS"))
	if host == "" {
		return nil, fmt.Errorf("未检测到 Kubernetes ServiceAccount 环境")
	}
	if port == "" {
		port = "443"
	}
	token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return nil, fmt.Errorf("读取 Kubernetes ServiceAccount 令牌失败: %w", err)
	}
	ca, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		return nil, fmt.Errorf("读取 Kubernetes CA 证书失败: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(ca) {
		return nil, fmt.Errorf("Kubernetes CA 证书无效")
	}
	return &KubernetesEngine{namespace: config.Namespace, workspaceRoot: config.WorkspaceRoot, baseURL: "https://" + host + ":" + port, token: strings.TrimSpace(string(token)), client: &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}}}}, nil
}

func (k *KubernetesEngine) Inspect(ctx context.Context, name string) (ContainerState, error) {
	var pod struct {
		Metadata struct {
			Labels map[string]string `json:"labels"`
		} `json:"metadata"`
		Status struct {
			Phase             string `json:"phase"`
			ContainerStatuses []struct {
				Ready bool `json:"ready"`
				State struct {
					Waiting *struct {
						Reason string `json:"reason"`
					} `json:"waiting"`
				} `json:"state"`
			} `json:"containerStatuses"`
		} `json:"status"`
	}
	if err := k.request(ctx, http.MethodGet, "/api/v1/namespaces/"+k.namespace+"/pods/"+name, nil, &pod); err != nil {
		return ContainerState{}, err
	}
	state := ContainerState{Name: name, Status: strings.ToLower(pod.Status.Phase), Running: pod.Status.Phase == "Running", Labels: pod.Metadata.Labels}
	if pod.Status.Phase == "Pending" {
		state.Health = "starting"
	}
	// Failed（包括节点因 Evicted 驱逐的工作区）不是可重新启动的 stopped
	// Pod。必须让控制面明确返回 error，由调用方执行受控 Rebuild；否则会把
	// 已持久化的源码错误地引导到首次模板初始化流程。
	if pod.Status.Phase == "Failed" {
		state.Health = "unhealthy"
	}
	for _, c := range pod.Status.ContainerStatuses {
		if c.State.Waiting != nil && (c.State.Waiting.Reason == "ImagePullBackOff" || c.State.Waiting.Reason == "ErrImagePull") {
			state.Health = "unhealthy"
		}
		if !c.Ready && state.Running {
			state.Health = "starting"
		}
	}
	var service struct {
		Spec struct {
			Ports []struct {
				Name string `json:"name"`
				Port int    `json:"port"`
			} `json:"ports"`
		} `json:"spec"`
	}
	if err := k.request(ctx, http.MethodGet, "/api/v1/namespaces/"+k.namespace+"/services/"+name, nil, &service); err == nil {
		state.ServicePorts = make(map[string]string, len(service.Spec.Ports))
		for _, port := range service.Spec.Ports {
			if port.Port > 0 {
				state.ServicePorts[port.Name] = fmt.Sprint(port.Port)
			}
		}
	}
	return state, nil
}

func (k *KubernetesEngine) Create(ctx context.Context, spec ContainerSpec) error {
	labels := spec.Labels
	labels["app.kubernetes.io/name"] = spec.Name
	service := map[string]any{"apiVersion": "v1", "kind": "Service", "metadata": map[string]any{"name": spec.Name, "labels": labels}, "spec": map[string]any{"type": "ClusterIP", "selector": map[string]string{"app.kubernetes.io/name": spec.Name}, "ports": []any{
		map[string]any{"name": "code", "port": 3000, "targetPort": 3000, "protocol": "TCP"},
		map[string]any{"name": "ai", "port": 30141, "targetPort": 30141, "protocol": "TCP"},
		map[string]any{"name": "preview", "port": 5173, "targetPort": 5173, "protocol": "TCP"},
		map[string]any{"name": "preview-control", "port": 5174, "targetPort": 5174, "protocol": "TCP"},
	}}}
	if err := k.request(ctx, http.MethodPost, "/api/v1/namespaces/"+k.namespace+"/services", service, nil); err != nil && !errors.Is(err, ErrContainerConflict) {
		return err
	}
	mounts := make([]any, 0, len(spec.Mounts))
	initMounts := make([]any, 0, 2)
	for _, m := range spec.Mounts {
		mounts = append(mounts, map[string]any{"name": "workspaces", "mountPath": m.Target, "subPath": m.Subpath, "readOnly": m.ReadOnly})
		// context 子路径挂载会由 Kubernetes 创建 .induforge 父目录。让初始化容器
		// 同时看到该固定挂载点，才能在主容器启动前安全修正父目录属主。
		if m.Target == "/workspace/.induforge/context" {
			initMounts = append(initMounts, map[string]any{"name": "workspaces", "mountPath": "/project/workspace/.induforge/context", "subPath": m.Subpath, "readOnly": true})
		}
	}
	projectRoot := ""
	if len(spec.Mounts) > 0 {
		projectRoot = path.Dir(spec.Mounts[0].Subpath)
	}
	if projectRoot == "." || projectRoot == "" {
		return fmt.Errorf("代码工作区挂载路径不完整")
	}
	// 主容器作为非特权 coder 用户运行。持久化目录的创建和属主修正由仅挂载
	// 当前工程目录的 initContainer 完成，避免主容器为了 fixuid/sudo 关闭
	// no-new-privileges。initContainer 仅加回 chown 所需的最小能力。
	initContainer := map[string]any{
		"name":            "workspace-permissions",
		"image":           spec.Image,
		"imagePullPolicy": "Never",
		"command": []string{"/bin/sh", "-ec", `
mkdir -p /project/workspace /project/code-server-data /project/code-server-config /project/cache
# 历史编辑器缓存中可能存在 coder 的 0700 子目录；初始化器只需修正四个挂载根
# 目录，递归遍历既无必要，也会要求额外的目录绕过能力。
chown 1000:1000 /project/workspace /project/code-server-data /project/code-server-config /project/cache
# K3s 为只读上下文子路径自动创建 .induforge 父目录时，默认属主为 root。
# 仅修正这个父目录，context 本身仍保持只读挂载，不放宽主容器权限。
if test -d /project/workspace/.induforge; then chown 1000:1000 /project/workspace/.induforge; fi
`},
		"securityContext": map[string]any{
			"runAsUser":                0,
			"runAsGroup":               0,
			"allowPrivilegeEscalation": false,
			"capabilities":             map[string]any{"drop": []string{"ALL"}, "add": []string{"CHOWN"}},
		},
		"volumeMounts": append([]any{map[string]any{"name": "workspaces", "mountPath": "/project", "subPath": projectRoot}}, initMounts...),
	}
	mainEnvironment := append(append([]string{}, spec.Environment...), "INDUFORGE_KUBERNETES_WORKSPACE=true")
	pod := map[string]any{
		"apiVersion": "v1", "kind": "Pod", "metadata": map[string]any{"name": spec.Name, "labels": labels},
		"spec": map[string]any{
			"nodeSelector":    map[string]string{"induforge.io/center-node": "true"},
			"securityContext": map[string]any{"fsGroup": 1000},
			"initContainers":  []any{initContainer},
			"containers": []any{map[string]any{
				"name": "code-workspace", "image": spec.Image, "imagePullPolicy": "Never", "args": spec.Command, "workingDir": spec.WorkingDir,
				"env": envValues(mainEnvironment), "ports": []any{map[string]any{"containerPort": 3000}, map[string]any{"containerPort": 30141}, map[string]any{"containerPort": 5173}, map[string]any{"containerPort": 5174}},
				"securityContext": map[string]any{"runAsUser": 1000, "runAsGroup": 1000, "allowPrivilegeEscalation": false, "capabilities": map[string]any{"drop": []string{"ALL"}}},
				"volumeMounts":    mounts,
			}},
			"volumes": []any{map[string]any{"name": "workspaces", "hostPath": map[string]any{"path": k.workspaceRoot, "type": "Directory"}}},
		},
	}
	if err := k.request(ctx, http.MethodPost, "/api/v1/namespaces/"+k.namespace+"/pods", pod, nil); err != nil {
		return err
	}
	return nil
}
func envValues(values []string) []any {
	result := make([]any, 0, len(values))
	for _, value := range values {
		name, val, ok := strings.Cut(value, "=")
		if ok {
			result = append(result, map[string]string{"name": name, "value": val})
		}
	}
	return result
}
func (k *KubernetesEngine) Start(ctx context.Context, name string) error {
	_, err := k.Inspect(ctx, name)
	return err
}
func (k *KubernetesEngine) Stop(ctx context.Context, name string) error { return k.Remove(ctx, name) }
func (k *KubernetesEngine) Remove(ctx context.Context, name string) error {
	if err := k.removeResource(ctx, "/api/v1/namespaces/"+k.namespace+"/pods/"+name); err != nil {
		return err
	}
	return k.removeResource(ctx, "/api/v1/namespaces/"+k.namespace+"/services/"+name)
}

// removeResource 等待 API 已确认删除，避免 Rebuild 立即复用名称时遇到 409。
func (k *KubernetesEngine) removeResource(ctx context.Context, endpoint string) error {
	err := k.request(ctx, http.MethodDelete, endpoint, nil, nil)
	if errors.Is(err, ErrContainerNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		err = k.request(ctx, http.MethodGet, endpoint, nil, nil)
		if errors.Is(err, ErrContainerNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("等待 Kubernetes 删除代码工作区资源超时: %s", endpoint)
		case <-ticker.C:
		}
	}
}
func (k *KubernetesEngine) request(ctx context.Context, method, endpoint string, body, output any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, k.baseURL+path.Clean("/"+endpoint), reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+k.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := k.client.Do(req)
	if err != nil {
		return fmt.Errorf("访问 Kubernetes API 失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == 404 {
			return ErrContainerNotFound
		}
		if resp.StatusCode == 409 {
			return ErrContainerConflict
		}
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return fmt.Errorf("Kubernetes API 返回 %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if output != nil {
		if err := json.NewDecoder(resp.Body).Decode(output); err != nil {
			return err
		}
	}
	return nil
}
