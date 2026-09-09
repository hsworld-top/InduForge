package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/indu-forge/dev_core/internal/dataservice"
)

type hostNodeAddressLoaderFunc func(context.Context, []string) (map[string]string, error)

func (f hostNodeAddressLoaderFunc) LoadHostNodeAddresses(ctx context.Context, ids []string) (map[string]string, error) {
	return f(ctx, ids)
}

type builtInHostNodeLoader struct{}

func (builtInHostNodeLoader) LoadHostNodeAddresses(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (builtInHostNodeLoader) LoadHostNodeSchedulingTargets(context.Context, []string) (map[string]HostNodeSchedulingTarget, error) {
	return map[string]HostNodeSchedulingTarget{testNodeID: {NodeSource: "built_in", Hostname: "center-01", IPAddress: "10.0.0.129"}}, nil
}

func TestProjectReleaseDigestUsesCanonicalSHA256(t *testing.T) {
	hex := strings.Repeat("a", 64)
	for _, input := range []string{hex, "sha256:" + hex, " SHA256:" + strings.ToUpper(hex) + " "} {
		if got := projectReleaseDigest(input); got != "sha256:"+hex {
			t.Fatalf("projectReleaseDigest(%q)=%q", input, got)
		}
	}
}

func TestKubernetesApplyPathUsesCoreGroupForService(t *testing.T) {
	if got := kubernetesApplyPath("Service", "project", "services", "entry"); got != "/api/v1/namespaces/project/services/entry" {
		t.Fatalf("Service apply path=%q", got)
	}
	if got := kubernetesApplyPath("Deployment", "project", "deployments", "entry"); got != "/apis/apps/v1/namespaces/project/deployments/entry" {
		t.Fatalf("Deployment apply path=%q", got)
	}
}

func TestKubernetesProjectReconcilerStopsOnlyStableDeploymentResources(t *testing.T) {
	paths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		paths = append(paths, request.Method+" "+request.URL.Path)
		if request.Method == http.MethodGet && strings.Contains(request.URL.Path, "/deployments/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if request.Method != http.MethodDelete {
			t.Fatalf("unexpected request %s %s", request.Method, request.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	reconciler := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL, token: "test"}
	if err := reconciler.StopDeployment(context.Background(), testEnvironmentID, "99999999-9999-4999-8999-999999999999"); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(paths, "\n")
	for _, expected := range []string{
		"DELETE /apis/apps/v1/namespaces/if-env-666666666666/deployments/if-project-999999999999-base",
		"DELETE /api/v1/namespaces/if-env-666666666666/services/if-project-999999999999-compute",
		"DELETE /api/v1/namespaces/if-env-666666666666/secrets/if-project-999999999999-compute-sandbox",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("missing exact cleanup resource %q:\n%s", expected, joined)
		}
	}
	if strings.Contains(joined, "collector") {
		t.Fatalf("Kubernetes 不应清理原生采集资源: %s", joined)
	}
	if strings.Contains(joined, "other-project") {
		t.Fatalf("cleanup must not use a broad selector: %s", joined)
	}
}

func TestKubernetesProjectReconcilerLabelsHistoricalK3sNodeByInternalIP(t *testing.T) {
	patches := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "GET /api/v1/nodes":
			_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"if-5fb-history","labels":{}},"status":{"addresses":[{"type":"InternalIP","address":"10.0.0.129"}],"conditions":[{"type":"Ready","status":"True"}]}}]}`))
		case "PATCH /api/v1/nodes/if-5fb-history":
			patches++
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	r := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL, token: "test"}
	r.SetHostNodeAddressLoader(hostNodeAddressLoaderFunc(func(context.Context, []string) (map[string]string, error) {
		return map[string]string{testNodeID: "10.0.0.129"}, nil
	}))
	if err := r.EnsureHostNodeLabels(context.Background(), []string{testNodeID}); err != nil || patches != 1 {
		t.Fatalf("historical node name must be labelled by IP: err=%v patches=%d", err, patches)
	}
}

func TestKubernetesProjectReconcilerAcceptsBuiltInNodeWithoutTenantSpecificLabel(t *testing.T) {
	patches := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			patches++
		}
		_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"center-01","labels":{"induforge.io/center-node":"true"}},"status":{"addresses":[{"type":"InternalIP","address":"10.0.0.129"}],"conditions":[{"type":"Ready","status":"True"}]}}]}`))
	}))
	defer server.Close()
	reconciler := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL, token: "test"}
	reconciler.SetHostNodeAddressLoader(builtInHostNodeLoader{})
	if err := reconciler.EnsureHostNodeLabels(context.Background(), []string{testNodeID}); err != nil {
		t.Fatal(err)
	}
	if patches != 0 {
		t.Fatal("中心内置节点不得写入租户专属节点标签")
	}
}

func TestKubernetesProjectReconcilerCollectsBuiltInNodeResources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/apis/metrics.k8s.io/v1beta1/nodes":
			_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"center-01"},"usage":{"cpu":"500m","memory":"1Gi"}}]}`))
		case "/api/v1/nodes/center-01/proxy/stats/summary":
			_, _ = w.Write([]byte(`{"node":{"fs":{"capacityBytes":1000,"usedBytes":875}}}`))
		default:
			t.Fatalf("unexpected request %s", request.URL.Path)
		}
	}))
	defer server.Close()

	nodes := []KubernetesNode{{
		Name: "center-01", Labels: map[string]string{"induforge.io/center-node": "true"},
		CPUCapacity: "2", MemoryCapacity: "4Gi", OSImage: "Ubuntu 24.04.4 LTS", KernelVersion: "6.8.0",
	}}
	reconciler := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL, token: "test", centerDataPath: "/var/lib/induframe/center"}
	if err := reconciler.CollectNodeResourceSummaries(context.Background(), nodes); err != nil {
		t.Fatal(err)
	}
	for key, expected := range map[string]float64{"cpu": 25, "memory": 25, "disk": 87.5} {
		resource, ok := nodes[0].ResourceSummary[key].(map[string]any)
		if !ok || resource["usedPercent"] != expected {
			t.Fatalf("%s resource summary=%#v", key, nodes[0].ResourceSummary[key])
		}
	}
	if cpu := nodes[0].ResourceSummary["cpu"].(map[string]any); cpu["count"] != float64(2) {
		t.Fatalf("center cpu summary=%#v", cpu)
	}
	system := nodes[0].ResourceSummary["system"].(map[string]any)
	if system["distribution"] != "Ubuntu 24.04.4 LTS" || system["dataPath"] != "/var/lib/induframe/center" {
		t.Fatalf("center system summary=%#v", system)
	}
}

func TestKubernetesQuantityParsing(t *testing.T) {
	for input, expected := range map[string]float64{"250000000n": 0.25, "750m": 0.75, "2": 2} {
		actual, ok := parseCPUQuantity(input)
		if !ok || actual != expected {
			t.Fatalf("parseCPUQuantity(%q)=(%v,%v), want %v", input, actual, ok, expected)
		}
	}
	for input, expected := range map[string]float64{"1Ki": 1024, "2Mi": 2 << 20, "3G": 3e9} {
		actual, ok := parseByteQuantity(input)
		if !ok || actual != expected {
			t.Fatalf("parseByteQuantity(%q)=(%v,%v), want %v", input, actual, ok, expected)
		}
	}
}

func TestKubernetesProjectReconcilerRejectsNodeLabelConflictAndNotReady(t *testing.T) {
	for name, node := range map[string]string{"conflict": `{"metadata":{"name":"if-old","labels":{"induforge.io/host-node-id":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"}},"status":{"addresses":[{"type":"InternalIP","address":"10.0.0.130"}],"conditions":[{"type":"Ready","status":"True"}]}}`, "not-ready": `{"metadata":{"name":"if-old","labels":{}},"status":{"addresses":[{"type":"InternalIP","address":"10.0.0.130"}],"conditions":[{"type":"Ready","status":"False"}]}}`} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatal("invalid node must not be patched")
				}
				_, _ = w.Write([]byte(`{"items":[` + node + `]}`))
			}))
			defer server.Close()
			r := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL, token: "test"}
			r.SetHostNodeAddressLoader(hostNodeAddressLoaderFunc(func(context.Context, []string) (map[string]string, error) {
				return map[string]string{testNodeID: "10.0.0.130"}, nil
			}))
			if err := r.EnsureHostNodeLabels(context.Background(), []string{testNodeID}); err == nil {
				t.Fatal("conflict or offline node accepted")
			}
		})
	}
}

func TestKubernetesProjectReconcilerKeepsCorrectNodeLabel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatal("existing correct label must not be patched")
		}
		_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"if-08b-history","labels":{"induforge.io/host-node-id":"` + testNodeID + `"}},"status":{"addresses":[{"type":"InternalIP","address":"10.0.0.130"}],"conditions":[{"type":"Ready","status":"True"}]}}]}`))
	}))
	defer server.Close()
	r := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL, token: "test"}
	r.SetHostNodeAddressLoader(hostNodeAddressLoaderFunc(func(context.Context, []string) (map[string]string, error) {
		return map[string]string{testNodeID: "10.0.0.130"}, nil
	}))
	if err := r.EnsureHostNodeLabels(context.Background(), []string{testNodeID}); err != nil {
		t.Fatal(err)
	}
}

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

func TestNATSControlEndpointQualifiesEnvironmentShortName(t *testing.T) {
	if got := natsControlEndpoint("nats://nats:4222", "if-env-0123456789ab"); got != "nats://nats.if-env-0123456789ab.svc:4222" {
		t.Fatalf("控制面必须用环境 namespace 连接 NATS: %s", got)
	}
	if got := natsControlEndpoint("nats://nats.example.internal:4222", "if-env-0123456789ab"); got != "nats://nats.example.internal:4222" {
		t.Fatalf("完整 NATS 地址不得被改写: %s", got)
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
	if err != nil || !status.Ready || status.Failed || status.ReplicasObserved != 1 {
		t.Fatalf("ready rollout status=%+v err=%v", status, err)
	}
}

func TestKubernetesProjectReconcilerStatusWaitsForBaseEntry(t *testing.T) {
	port := 17800
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if strings.Contains(request.URL.Path, "/services/") {
			_, _ = w.Write([]byte(`{"status":{"loadBalancer":{}}}`))
			return
		}
		_, _ = w.Write([]byte(`{"metadata":{"generation":3},"spec":{"replicas":1},"status":{"observedGeneration":3,"availableReplicas":1}}`))
	}))
	defer server.Close()
	reconciler := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL, token: "test"}
	status, err := reconciler.Status(context.Background(), ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", Engine: ServiceBase, HostPort: &port})
	if err != nil || status.Ready || status.ReplicasObserved != 1 || !strings.Contains(status.Message, "访问入口") {
		t.Fatalf("pending entry status=%+v err=%v", status, err)
	}
}

func TestKubernetesProjectReconcilerStatusRequiresReadyBaseEntry(t *testing.T) {
	port := 17800
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if strings.Contains(request.URL.Path, "/services/") {
			_, _ = w.Write([]byte(`{"status":{"loadBalancer":{"ingress":[{"ip":"172.16.125.130"}]}}}`))
			return
		}
		_, _ = w.Write([]byte(`{"metadata":{"generation":3},"spec":{"replicas":1},"status":{"observedGeneration":3,"availableReplicas":1}}`))
	}))
	defer server.Close()
	reconciler := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL, token: "test"}
	status, err := reconciler.Status(context.Background(), ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", Engine: ServiceBase, HostPort: &port})
	if err != nil || !status.Ready || status.ReplicasObserved != 1 {
		t.Fatalf("ready entry status=%+v err=%v", status, err)
	}
}

func TestKubernetesProjectReconcilerStatusReportsInitPhase(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if strings.Contains(request.URL.Path, "/pods") {
			_, _ = w.Write([]byte(`{"items":[{"status":{"initContainerStatuses":[{"name":"runtime-provision-state","state":{"terminated":{"exitCode":1,"reason":"Error"}}}]}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"metadata":{"generation":3},"spec":{"replicas":1},"status":{"observedGeneration":3,"availableReplicas":0}}`))
	}))
	defer server.Close()
	reconciler := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL, token: "test"}
	status, err := reconciler.Status(context.Background(), ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", Engine: ServiceCompute})
	if err != nil || !status.Failed || !strings.Contains(status.Message, "runtime-provision-state") || workloadFailureStage(status.Message) != "failed" {
		t.Fatalf("init failure status=%+v err=%v", status, err)
	}
}

func TestKubernetesProjectReconcilerStatusReportsCrashLoop(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if strings.Contains(request.URL.Path, "/pods") {
			_, _ = w.Write([]byte(`{"items":[{"status":{"containerStatuses":[{"name":"runtime-api","state":{"waiting":{"reason":"CrashLoopBackOff"}}}]}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"metadata":{"generation":3},"spec":{"replicas":1},"status":{"observedGeneration":3,"availableReplicas":0}}`))
	}))
	defer server.Close()
	reconciler := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL, token: "test"}
	status, err := reconciler.Status(context.Background(), ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", Engine: ServiceBase})
	if err != nil || !status.Failed || !strings.Contains(status.Message, "runtime-api") {
		t.Fatalf("crash loop status=%+v err=%v", status, err)
	}
}

func TestKubernetesProjectReconcilerStatusReportsRepeatedProbeRestart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if strings.Contains(request.URL.Path, "/pods") {
			_, _ = w.Write([]byte(`{"items":[{"status":{"containerStatuses":[{"name":"runtime-api","restartCount":3,"state":{"running":{}},"lastState":{"terminated":{"exitCode":0,"reason":"Completed"}}}]}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"metadata":{"generation":3},"spec":{"replicas":1},"status":{"observedGeneration":3,"availableReplicas":0}}`))
	}))
	defer server.Close()
	reconciler := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL, token: "test"}
	status, err := reconciler.Status(context.Background(), ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", Engine: ServiceBase})
	if err != nil || !status.Failed || !strings.Contains(status.Message, "runtime-api") {
		t.Fatalf("repeated restart status=%+v err=%v", status, err)
	}
}

func TestCurrentNodeIdentityWinsOverStaleIPRecords(t *testing.T) {
	patches := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "GET /api/v1/nodes":
			_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"old-node"},"status":{"addresses":[{"type":"InternalIP","address":"10.0.0.129"}]}},{"metadata":{"name":"if-333333333333","labels":{}},"status":{"addresses":[{"type":"InternalIP","address":"10.0.0.129"}],"conditions":[{"type":"Ready","status":"True"}]}}]}`))
		case "PATCH /api/v1/nodes/if-333333333333":
			patches++
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	r := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL, token: "test"}
	r.SetHostNodeAddressLoader(hostNodeAddressLoaderFunc(func(context.Context, []string) (map[string]string, error) {
		return map[string]string{testNodeID: "10.0.0.129"}, nil
	}))
	if err := r.EnsureHostNodeLabels(context.Background(), []string{testNodeID}); err != nil || patches != 1 {
		t.Fatalf("historical node name must be labelled by IP: err=%v patches=%d", err, patches)
	}
}

func TestProjectPodImageFailure(t *testing.T) {
	for _, field := range []string{"initContainerStatuses", "containerStatuses"} {
		t.Run(field, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, `{"items":[{"status":{"%s":[{"name":"engine","state":{"waiting":{"reason":"ImagePullBackOff"}}}]}}]}`, field)
			}))
			defer server.Close()
			reconciler := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL}
			status, err := reconciler.projectPodFailure(context.Background(), "test", "test")
			if err != nil || !status.Failed || !strings.Contains(status.Message, "ImagePullBackOff") {
				t.Fatalf("status=%+v err=%v", status, err)
			}
		})
	}
}
