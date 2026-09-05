package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/dataservice"
)

// NativeCollectorBundle 只交付采集配置和最小凭据，不接受或输出可执行路径、命令与环境变量。
// SecretFiles 仅出现在已认证节点的 no-store 响应中，不进入日志、事件或数据库。
type NativeCollectorBundle struct {
	SchemaVersion string            `json:"schemaVersion"`
	NodeID        string            `json:"nodeId"`
	DeploymentID  string            `json:"deploymentId"`
	ServiceID     string            `json:"serviceId"`
	ReleaseID     string            `json:"releaseId"`
	Revision      int               `json:"revision"`
	Binding       json.RawMessage   `json:"binding"`
	Index         json.RawMessage   `json:"index"`
	SecretFiles   map[string][]byte `json:"secretFiles"`
}

type NativeCollectorProvider interface {
	BuildNativeCollectorBundle(context.Context, DeploymentBinding) (NativeCollectorBundle, error)
}

func (s *Service) SetNativeCollectorProvider(provider NativeCollectorProvider) {
	s.nativeCollector = provider
}

func (s *Service) AgentNativeCollector(ctx context.Context, nodeID, token, deploymentID, serviceID string) (NativeCollectorBundle, error) {
	binding, err := s.AgentDeploymentBinding(ctx, nodeID, token, deploymentID, serviceID)
	if err != nil {
		return NativeCollectorBundle{}, err
	}
	if s.nativeCollector == nil {
		return NativeCollectorBundle{}, fmt.Errorf("原生采集配置提供器未就绪")
	}
	return s.nativeCollector.BuildNativeCollectorBundle(ctx, binding)
}

func (h *Handler) agentNativeCollector(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	bundle, err := h.service.AgentNativeCollector(r.Context(), chi.URLParam(r, "id"), agentToken(r), chi.URLParam(r, "deployment"), r.URL.Query().Get("serviceId"))
	if err != nil {
		h.err(w, r, err)
		return
	}
	h.writeSuccess(w, r, bundle)
}

func (r *KubernetesProjectReconciler) BuildNativeCollectorBundle(ctx context.Context, authorized DeploymentBinding) (NativeCollectorBundle, error) {
	var binding struct {
		SchemaVersion string `json:"schemaVersion"`
		Engine        string `json:"engine"`
		ServiceID     string `json:"serviceId"`
		EnvironmentID string `json:"environmentId"`
		Release       struct {
			ID string `json:"id"`
		} `json:"release"`
	}
	if json.Unmarshal(authorized.Content, &binding) != nil || binding.SchemaVersion != "deployment-binding.v2" || binding.Engine != ServiceCollector {
		return NativeCollectorBundle{}, fmt.Errorf("节点绑定不是原生采集服务")
	}
	if r.runtimeContext == nil || r.collectorBundle == nil || r.secretManager == nil {
		return NativeCollectorBundle{}, fmt.Errorf("采集运行依赖未初始化")
	}
	runtimeContext, err := r.runtimeContext.LoadProjectRuntimeContext(ctx, authorized.ProjectDeploymentID)
	if err != nil {
		return NativeCollectorBundle{}, err
	}
	if runtimeContext.TenantID != authorized.TenantID || runtimeContext.ProjectID != authorized.ProjectID || runtimeContext.EnvironmentID != binding.EnvironmentID || runtimeContext.Release.ID != binding.Release.ID || runtimeContext.BindingRevision != authorized.Revision {
		return NativeCollectorBundle{}, fmt.Errorf("采集绑定已变化，请重新获取命令")
	}
	endpoint, err := r.nativeNATSEndpoint(ctx, runtimeContext.EnvironmentID)
	if err != nil {
		return NativeCollectorBundle{}, err
	}
	bundle, err := r.collectorBundle.BuildCollectorBindingBundle(ctx, dataservice.CollectorBindingBundleRequest{TenantID: runtimeContext.TenantID, ProjectID: runtimeContext.ProjectID, DeploymentID: authorized.ProjectDeploymentID, EnvironmentID: runtimeContext.EnvironmentID, NodeID: authorized.NodeID, ReleaseID: runtimeContext.Release.ID, Revision: int64(authorized.Revision), SourceSnapshot: runtimeContext.CollectorSourceSnapshot, AccountID: runtimeAccountID(runtimeContext.ProjectID, authorized.ProjectDeploymentID), NATSEndpoint: endpoint, NATSResourceRef: runtimeContext.Support.NATSResourceRef, NATSCredentialSecretRef: runtimeContext.Support.NATSCredentialSecretRef})
	if err != nil {
		return NativeCollectorBundle{}, fmt.Errorf("构建原生采集配置失败: %w", err)
	}
	token, err := r.secretManager.sourceValue(ctx, runtimeContext.Support.NATSCredentialSource, "credential")
	if err != nil {
		return NativeCollectorBundle{}, fmt.Errorf("读取采集消息凭据失败")
	}
	files := make(map[string][]byte, len(bundle.SecretFiles)+1)
	for name, content := range bundle.SecretFiles {
		if !safeCollectorSecretFileName(name) || len(content) > 1<<20 {
			return NativeCollectorBundle{}, fmt.Errorf("采集凭据文件非法")
		}
		files[name] = content
	}
	files[resolverNATSFile], err = json.Marshal(map[string]string{"schemaVersion": "nats-credential.v1", "authType": "token", "token": token})
	if err != nil {
		return NativeCollectorBundle{}, fmt.Errorf("构造采集消息凭据失败")
	}
	return NativeCollectorBundle{SchemaVersion: "native-collector-bundle.v1", NodeID: authorized.NodeID, DeploymentID: authorized.ProjectDeploymentID, ServiceID: binding.ServiceID, ReleaseID: runtimeContext.Release.ID, Revision: authorized.Revision, Binding: bundle.Binding, Index: bundle.Index, SecretFiles: files}, nil
}

// nativeNATSEndpoint 使用环境专属 NodePort，保留集群内 nats Service。端口由 Kubernetes
// 分配，节点地址从当前 Ready 节点读取，避免给宿主进程下发只能在 Pod 内解析的短名称。
func (r *KubernetesProjectReconciler) nativeNATSEndpoint(ctx context.Context, environmentID string) (string, error) {
	ns, err := projectNamespace(environmentID)
	if err != nil {
		return "", err
	}
	nodes, err := r.ListNodes(ctx)
	if err != nil {
		return "", err
	}
	address := ""
	for _, node := range nodes {
		if node.Ready && net.ParseIP(node.InternalIP) != nil {
			address = node.InternalIP
			break
		}
	}
	if address == "" {
		return "", fmt.Errorf("没有可访问消息服务的就绪节点")
	}
	doc := map[string]any{"apiVersion": "v1", "kind": "Service", "metadata": map[string]string{"name": "nats-native", "namespace": ns}, "spec": map[string]any{"type": "NodePort", "selector": map[string]string{"app": "nats"}, "ports": []map[string]any{{"name": "client", "port": 4222, "targetPort": 4222, "protocol": "TCP"}}}}
	raw, _ := json.Marshal(doc)
	request, err := http.NewRequestWithContext(ctx, http.MethodPatch, r.endpoint+"/api/v1/namespaces/"+ns+"/services/nats-native?fieldManager=induforge-native-collector", strings.NewReader(string(raw)))
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+r.token)
	request.Header.Set("Content-Type", "application/apply-patch+yaml")
	response, err := r.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("配置原生消息入口失败")
	}
	defer response.Body.Close()
	if response.StatusCode/100 != 2 {
		return "", fmt.Errorf("配置原生消息入口失败: HTTP %d", response.StatusCode)
	}
	var service struct {
		Spec struct {
			Ports []struct {
				NodePort int `json:"nodePort"`
			} `json:"ports"`
		} `json:"spec"`
	}
	if json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&service) != nil || len(service.Spec.Ports) != 1 || service.Spec.Ports[0].NodePort < 1 || service.Spec.Ports[0].NodePort > 65535 {
		return "", fmt.Errorf("原生消息入口端口未分配")
	}
	return "nats://" + net.JoinHostPort(address, strconv.Itoa(service.Spec.Ports[0].NodePort)), nil
}
