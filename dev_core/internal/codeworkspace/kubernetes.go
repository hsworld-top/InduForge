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
				NodePort int `json:"nodePort"`
			} `json:"ports"`
		} `json:"spec"`
	}
	if err := k.request(ctx, http.MethodGet, "/api/v1/namespaces/"+k.namespace+"/services/"+name, nil, &service); err == nil && len(service.Spec.Ports) > 0 && service.Spec.Ports[0].NodePort > 0 {
		state.HostPort = fmt.Sprint(service.Spec.Ports[0].NodePort)
	}
	return state, nil
}

func (k *KubernetesEngine) Create(ctx context.Context, spec ContainerSpec) error {
	labels := spec.Labels
	labels["app.kubernetes.io/name"] = spec.Name
	service := map[string]any{"apiVersion": "v1", "kind": "Service", "metadata": map[string]any{"name": spec.Name, "labels": labels}, "spec": map[string]any{"type": "NodePort", "selector": map[string]string{"app.kubernetes.io/name": spec.Name}, "ports": []any{map[string]any{"port": 3000, "targetPort": 3000, "protocol": "TCP"}}}}
	if err := k.request(ctx, http.MethodPost, "/api/v1/namespaces/"+k.namespace+"/services", service, nil); err != nil && !errors.Is(err, ErrContainerConflict) {
		return err
	}
	mounts := make([]any, 0, len(spec.Mounts))
	for i, m := range spec.Mounts {
		mounts = append(mounts, map[string]any{"name": "workspaces", "mountPath": m.Target, "subPath": m.Subpath, "readOnly": m.ReadOnly})
		_ = i
	}
	pod := map[string]any{"apiVersion": "v1", "kind": "Pod", "metadata": map[string]any{"name": spec.Name, "labels": labels}, "spec": map[string]any{"nodeSelector": map[string]string{"induforge.io/center-node": "true"}, "securityContext": map[string]any{"fsGroup": 1000}, "containers": []any{map[string]any{"name": "code-workspace", "image": spec.Image, "imagePullPolicy": "Never", "args": spec.Command, "workingDir": spec.WorkingDir, "env": envValues(spec.Environment), "ports": []any{map[string]any{"containerPort": 3000}}, "securityContext": map[string]any{"runAsUser": 1000, "runAsGroup": 1000, "allowPrivilegeEscalation": false, "capabilities": map[string]any{"drop": []string{"ALL"}}}, "volumeMounts": mounts}}, "volumes": []any{map[string]any{"name": "workspaces", "hostPath": map[string]any{"path": k.workspaceRoot, "type": "Directory"}}}}}
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
	err := k.request(ctx, http.MethodDelete, "/api/v1/namespaces/"+k.namespace+"/pods/"+name, nil, nil)
	if err != nil && !errors.Is(err, ErrContainerNotFound) {
		return err
	}
	err = k.request(ctx, http.MethodDelete, "/api/v1/namespaces/"+k.namespace+"/services/"+name, nil, nil)
	if err != nil && !errors.Is(err, ErrContainerNotFound) {
		return err
	}
	return nil
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
