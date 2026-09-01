package ops

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestKubernetesProjectReconcilerAppliesStableRollout(t *testing.T) {
	paths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		paths = append(paths, request.Method+" "+request.URL.Path)
		if request.Method != http.MethodPatch || request.Header.Get("Authorization") != "Bearer test" || request.Header.Get("Content-Type") != "application/apply-patch+yaml" {
			t.Fatalf("unexpected Kubernetes request: %s %s", request.Method, request.Header)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	port := 17800
	reconciler := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL, token: "test"}
	reconciler.SetRuntimeContextLoader(runtimeContextLoaderFunc(func(context.Context, string) (ProjectRuntimeContext, error) { return validRuntimeContext(), nil }))
	reconciler.SetDeploymentSecretManager(NewDeploymentSecretManager(&memorySecretClient{values: sourceSecrets()}))
	workload := ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", ServiceID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", NodeID: testNodeID, Engine: ServiceBase, ReleaseID: testVersionID, Generation: 1, HostPort: &port}
	if err := reconciler.Reconcile(context.Background(), workload); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(paths, "\n")
	if !strings.Contains(joined, "PATCH /api/v1/namespaces/if-env-666666666666/configmaps/if-project-999999999999-base-config") || !strings.Contains(joined, "PATCH /apis/apps/v1/namespaces/if-env-666666666666/deployments/if-project-999999999999-base") {
		t.Fatalf("stable resources were not applied: %s", joined)
	}
}

type runtimeContextLoaderFunc func(context.Context, string) (ProjectRuntimeContext, error)

func (f runtimeContextLoaderFunc) LoadProjectRuntimeContext(ctx context.Context, id string) (ProjectRuntimeContext, error) {
	return f(ctx, id)
}

func validRuntimeContext() ProjectRuntimeContext {
	manifest, _ := json.Marshal(map[string]any{"schemaVersion": "2.0", "artifacts": map[string]any{"runtime": map[string]any{"file": "runtime-artifact.tar.zst", "checksum": "sha256:" + strings.Repeat("a", 64)}}})
	return ProjectRuntimeContext{TenantID: "tenant-a", ProjectID: testProjectID, EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", Release: releaseMetadata{ID: testVersionID, Manifest: manifest}, Support: RuntimeSupportResources{NATSEndpoint: "nats://nats:4222", NATSResourceRef: "site-resource://env/nats", NATSCredentialSecretRef: "secret://env/nats", StateStoreResourceRef: "site-resource://env/postgres", StateStoreDSNSecretRef: "secret://env/postgres", StateStoreSchema: "runtime", StateStoreEndpoint: "postgres.runtime.svc:5432", StateStoreDatabase: "induforge_runtime", StateStoreAdminUser: "postgres", NATSCredentialSource: KubernetesSecretSource{Namespace: "runtime", Name: "nats", Keys: map[string]string{"credential": "token"}}, StateStoreCredentialSource: KubernetesSecretSource{Namespace: "runtime", Name: "postgres", Keys: map[string]string{"password": "password"}}}}
}

func sourceSecrets() map[string]map[string]string {
	return map[string]map[string]string{"runtime/nats": {"token": "test-token"}, "runtime/postgres": {"password": "test-password"}}
}

func TestKubernetesProjectReconcilerReportsApplyFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) { w.WriteHeader(http.StatusForbidden) }))
	defer server.Close()
	reconciler := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL, token: "test"}
	reconciler.SetDeploymentSecretManager(NewDeploymentSecretManager(&memorySecretClient{values: sourceSecrets()}))
	reconciler.SetRuntimeContextLoader(runtimeContextLoaderFunc(func(context.Context, string) (ProjectRuntimeContext, error) { return validRuntimeContext(), nil }))
	if err := reconciler.Reconcile(context.Background(), ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", ServiceID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", NodeID: testNodeID, Engine: ServiceCompute, ReleaseID: testVersionID, Generation: 1}); err == nil || !strings.Contains(err.Error(), "HTTP 403") {
		t.Fatalf("RBAC failure must be reported, got %v", err)
	}
}

func TestKubernetesProjectReconcilerStatusRequiresObservedReady(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		_, _ = w.Write([]byte(`{"metadata":{"generation":3},"spec":{"replicas":1},"status":{"observedGeneration":3,"availableReplicas":1}}`))
	}))
	defer server.Close()
	reconciler := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL, token: "test"}
	status, err := reconciler.Status(context.Background(), ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", Engine: ServiceBase})
	if err != nil || !status.Ready || status.Failed {
		t.Fatalf("ready rollout status=%+v err=%v", status, err)
	}
}

func TestKubernetesProjectReconcilerStatusReportsInitPhase(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		_, _ = w.Write([]byte(`{"metadata":{"generation":3},"spec":{"replicas":1},"status":{"observedGeneration":3,"availableReplicas":0,"initContainerStatuses":[{"name":"runtime-provision-state","state":{"terminated":{"exitCode":1,"reason":"Error"}}}]}}`))
	}))
	defer server.Close()
	reconciler := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL, token: "test"}
	status, err := reconciler.Status(context.Background(), ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", Engine: ServiceCompute})
	if err != nil || !status.Failed || !strings.Contains(status.Message, "runtime-provision-state") || workloadFailureStage(status.Message) != "provision-state" {
		t.Fatalf("init failure status=%+v err=%v", status, err)
	}
}
