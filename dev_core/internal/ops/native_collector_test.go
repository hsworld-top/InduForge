package ops

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/indu-forge/dev_core/internal/dataservice"
)

func TestNativeCollectorBundleUsesHostNATSAndFrozenBinding(t *testing.T) {
	state := validRuntimeContext()
	state.BindingRevision = 7
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/nodes" {
			_, _ = w.Write([]byte(`{"items":[{"status":{"addresses":[{"type":"InternalIP","address":"172.16.125.129"}],"conditions":[{"type":"Ready","status":"True"}]}}]}`))
			return
		}
		if r.Method != "PATCH" || !strings.HasSuffix(r.URL.Path, "/services/nats-native") {
			t.Fatalf("意外的集群写入: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"spec":{"ports":[{"nodePort":31222}]}}`))
	}))
	defer server.Close()
	r := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL}
	r.SetRuntimeContextLoader(runtimeContextLoaderFunc(func(context.Context, string) (ProjectRuntimeContext, error) { return state, nil }))
	r.SetDeploymentSecretManager(NewDeploymentSecretManager(&memorySecretClient{values: sourceSecrets()}))
	r.SetCollectorBindingBundleClient(collectorBundleClientFunc(func(_ context.Context, input dataservice.CollectorBindingBundleRequest) (*dataservice.CollectorBindingBundle, error) {
		if input.NATSEndpoint != "nats://172.16.125.129:31222" || input.Revision != 7 || input.ReleaseID != state.Release.ID {
			t.Fatalf("原生绑定错误: %+v", input)
		}
		return &dataservice.CollectorBindingBundle{Binding: json.RawMessage(`{}`), Index: json.RawMessage(`{}`)}, nil
	}))
	content, _ := json.Marshal(map[string]any{"schemaVersion": "deployment-binding.v2", "engine": "collector", "serviceId": "service", "environmentId": state.EnvironmentID, "release": map[string]string{"id": state.Release.ID}})
	authorized := DeploymentBinding{NodeID: testNodeID, TenantID: state.TenantID, ProjectID: state.ProjectID, ProjectDeploymentID: state.DeploymentID, Revision: 7, Content: content}
	bundle, err := r.BuildNativeCollectorBundle(context.Background(), authorized)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.NodeID != testNodeID || bundle.Revision != 7 || !strings.Contains(string(bundle.SecretFiles["nats.json"]), "test-token") {
		t.Fatal("配置未绑定节点或缺少最小凭据")
	}
	authorized.Revision = 8
	if _, err := r.BuildNativeCollectorBundle(context.Background(), authorized); err == nil {
		t.Fatal("接受了旧绑定")
	}
}

func TestKubernetesReconcilerRejectsNativeCollectorBeforeAnyClusterRequest(t *testing.T) {
	r := &KubernetesProjectReconciler{}
	err := r.Reconcile(context.Background(), ProjectWorkload{Engine: ServiceCollector})
	if err == nil || !strings.Contains(err.Error(), "NodeAgent") {
		t.Fatalf("Kubernetes 仍拥有采集生命周期: %v", err)
	}
}

func TestNativeCollectorUnauthorizedRequestNeverBuildsSecrets(t *testing.T) {
	service := NewService(&agentReleaseRepository{bindingErr: ErrAgentUnauthorized}, nil)
	if _, err := service.AgentNativeCollector(context.Background(), testNodeID, "untrusted", "deployment", "service"); err != ErrAgentUnauthorized {
		t.Fatalf("越权请求未在读取凭据前拒绝: %v", err)
	}
}

func TestDeploymentPortsReserveNativeNATSNodePortRange(t *testing.T) {
	for _, port := range []int{30000, 31222, 32767} {
		if !isReservedDeploymentPort(port) {
			t.Fatalf("工程入口可抢占 NodePort %d", port)
		}
	}
	if isReservedDeploymentPort(20000) {
		t.Fatal("普通工程端口被误拦截")
	}
}
