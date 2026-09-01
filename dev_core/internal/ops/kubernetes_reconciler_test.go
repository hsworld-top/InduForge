package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/indu-forge/dev_core/internal/dataservice"
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
	workload := ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", ServiceID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", NodeID: testNodeID, Engine: ServiceBase, ReleaseID: testVersionID, ReleaseDigest: "sha256:" + strings.Repeat("a", 64), Generation: 1, HostPort: &port}
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

type collectorBundleClientFunc func(context.Context, dataservice.CollectorBindingBundleRequest) (*dataservice.CollectorBindingBundle, error)

func (f collectorBundleClientFunc) BuildCollectorBindingBundle(ctx context.Context, input dataservice.CollectorBindingBundleRequest) (*dataservice.CollectorBindingBundle, error) {
	return f(ctx, input)
}

func validRuntimeContext() ProjectRuntimeContext {
	manifest, _ := json.Marshal(map[string]any{"schemaVersion": "2.0", "artifacts": map[string]any{"runtime": map[string]any{"file": "runtime-artifact.tar.zst", "checksum": "sha256:" + strings.Repeat("a", 64)}}})
	return ProjectRuntimeContext{TenantID: "tenant-a", ProjectID: testProjectID, EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", Release: releaseMetadata{ID: testVersionID, Manifest: manifest}, Support: RuntimeSupportResources{NATSEndpoint: "nats://nats:4222", NATSResourceRef: "site-resource://env/nats", NATSCredentialSecretRef: "secret://env/nats", StateStoreResourceRef: "site-resource://env/postgres", StateStoreDSNSecretRef: "secret://env/postgres", StateStoreSchema: "runtime", StateStoreEndpoint: "postgres.runtime.svc:5432", StateStoreDatabase: "induforge_runtime", StateStoreAdminUser: "postgres", NATSCredentialSource: KubernetesSecretSource{Namespace: "runtime", Name: "nats", Keys: map[string]string{"credential": "token"}}, StateStoreCredentialSource: KubernetesSecretSource{Namespace: "runtime", Name: "postgres", Keys: map[string]string{"password": "password"}}}}
}

func sourceSecrets() map[string]map[string]string {
	return map[string]map[string]string{"runtime/nats": {"token": "test-token"}, "runtime/postgres": {"password": "test-password"}}
}

func TestKubernetesProjectReconcilerAppliesCollectorBindingBeforeWorkload(t *testing.T) {
	paths, bodies := []string{}, []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(body))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	contextValue := validRuntimeContext()
	contextValue.BindingRevision = 7
	contextValue.CollectorSourceSnapshot = json.RawMessage(`{"schemaVersion":"collector-runtime-artifact.v1","projectId":"11111111-1111-4111-8111-111111111111"}`)
	contextValue.Release.Manifest, _ = json.Marshal(map[string]any{"schemaVersion": "2.0", "artifacts": map[string]any{"runtime": map[string]any{"file": "runtime-artifact.tar.zst"}, "collector": map[string]any{"file": collectorArtifactFile}}})
	reconciler := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL, token: "test"}
	reconciler.SetRuntimeContextLoader(runtimeContextLoaderFunc(func(context.Context, string) (ProjectRuntimeContext, error) { return contextValue, nil }))
	reconciler.SetDeploymentSecretManager(NewDeploymentSecretManager(&memorySecretClient{values: sourceSecrets()}))
	reconciler.SetCollectorBindingBundleClient(collectorBundleClientFunc(func(_ context.Context, input dataservice.CollectorBindingBundleRequest) (*dataservice.CollectorBindingBundle, error) {
		if input.SourceSnapshot == nil || input.Revision != 7 || input.ProjectID != testProjectID {
			t.Fatalf("collector used untrusted runtime context: %+v", input)
		}
		return &dataservice.CollectorBindingBundle{Binding: json.RawMessage(`{"schemaVersion":"collector-runtime-binding.v1","connection":"secret://collector/only"}`), Index: json.RawMessage(`{"schemaVersion":"collector-runtime-index.v1","secrets":{"secret://collector/only":"secrets/connection-aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa.json"}}`), BindingSHA256: "sha256:" + strings.Repeat("a", 64), IndexSHA256: "sha256:" + strings.Repeat("b", 64), SecretFiles: map[string][]byte{"connection-aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa.json": []byte(`{"password":"private"}`)}}, nil
	}))
	workload := ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: contextValue.DeploymentID, ServiceID: "dddddddd-dddd-4ddd-8ddd-dddddddddddd", NodeID: testNodeID, Engine: ServiceCollector, ReleaseID: testVersionID, ReleaseDigest: "sha256:" + strings.Repeat("a", 64), Generation: 1}
	if err := reconciler.Reconcile(context.Background(), workload); err != nil {
		t.Fatal(err)
	}
	if len(paths) < 4 || !strings.HasSuffix(paths[0], "-collector-binding") || !strings.Contains(strings.Join(paths[1:], "\n"), "/deployments/") {
		t.Fatalf("collector binding was not applied before workload: %v", paths)
	}
	if strings.Contains(bodies[0], "private") || strings.Contains(bodies[0], `"password"`) {
		t.Fatalf("collector ConfigMap leaked secret: %s", bodies[0])
	}
}

func TestKubernetesProjectReconcilerCollectorBindingFailureDoesNotApply(t *testing.T) {
	applies := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { applies++; w.WriteHeader(http.StatusOK) }))
	defer server.Close()
	contextValue := validRuntimeContext()
	contextValue.CollectorSourceSnapshot = json.RawMessage(`{"schemaVersion":"collector-runtime-artifact.v1"}`)
	contextValue.Release.Manifest, _ = json.Marshal(map[string]any{"artifacts": map[string]any{"collector": map[string]any{"file": collectorArtifactFile}}})
	reconciler := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL, token: "test"}
	reconciler.SetRuntimeContextLoader(runtimeContextLoaderFunc(func(context.Context, string) (ProjectRuntimeContext, error) { return contextValue, nil }))
	reconciler.SetDeploymentSecretManager(NewDeploymentSecretManager(&memorySecretClient{values: sourceSecrets()}))
	reconciler.SetCollectorBindingBundleClient(collectorBundleClientFunc(func(context.Context, dataservice.CollectorBindingBundleRequest) (*dataservice.CollectorBindingBundle, error) {
		return nil, fmt.Errorf("bad bundle")
	}))
	err := reconciler.Reconcile(context.Background(), ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: contextValue.DeploymentID, ServiceID: "dddddddd-dddd-4ddd-8ddd-dddddddddddd", NodeID: testNodeID, Engine: ServiceCollector, ReleaseID: testVersionID, ReleaseDigest: "sha256:" + strings.Repeat("a", 64), Generation: 1})
	if err == nil || !strings.Contains(err.Error(), "collector-binding") || workloadFailureStage(err.Error()) != "collector-binding" || applies != 0 {
		t.Fatalf("collector binding failure must preserve previous workload: err=%v applies=%d", err, applies)
	}
}

func TestKubernetesProjectReconcilerReportsApplyFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) { w.WriteHeader(http.StatusForbidden) }))
	defer server.Close()
	reconciler := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL, token: "test"}
	reconciler.SetDeploymentSecretManager(NewDeploymentSecretManager(&memorySecretClient{values: sourceSecrets()}))
	reconciler.SetRuntimeContextLoader(runtimeContextLoaderFunc(func(context.Context, string) (ProjectRuntimeContext, error) { return validRuntimeContext(), nil }))
	if err := reconciler.Reconcile(context.Background(), ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", ServiceID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", NodeID: testNodeID, Engine: ServiceCompute, ReleaseID: testVersionID, ReleaseDigest: "sha256:" + strings.Repeat("a", 64), Generation: 1}); err == nil || !strings.Contains(err.Error(), "HTTP 403") {
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
