package ops

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFoundationCredentialsRemainStable(t *testing.T) {
	stored := map[string]string{}
	writes := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			if len(stored) == 0 {
				w.WriteHeader(404)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": stored})
			return
		}
		writes++
		var obj struct{ StringData map[string]string }
		if err := json.NewDecoder(r.Body).Decode(&obj); err != nil {
			t.Error(err)
		}
		for key, value := range obj.StringData {
			stored[key] = base64.StdEncoding.EncodeToString([]byte(value))
		}
		w.WriteHeader(200)
	}))
	defer server.Close()
	r := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL}
	for i := 0; i < 2; i++ {
		if err := r.ensureFoundationCredentials(context.Background(), "if-env-test"); err != nil {
			t.Fatal(err)
		}
	}
	if writes != 1 {
		t.Fatalf("credential writes = %d", writes)
	}
	raw, _ := base64.StdEncoding.DecodeString(stored["nats.conf"])
	if !strings.Contains(string(raw), "2097152") {
		t.Fatal("NATS max payload contract lost")
	}
}

func TestFoundationWorkloadsUseEnvironmentStorageAndCenterSelector(t *testing.T) {
	var workload map[string]any
	requests := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.Path] = true
		if strings.Contains(r.URL.Path, "statefulsets") {
			if err := json.NewDecoder(r.Body).Decode(&workload); err != nil {
				t.Error(err)
			}
		}
		w.WriteHeader(200)
	}))
	defer server.Close()
	r := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL}
	if err := r.applyFoundationWorkload(context.Background(), "if-env-test", "postgres", "data-postgres-0", map[string]string{"induforge.io/center-node": "true"}); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(workload)
	text := string(raw)
	for _, want := range []string{"induforge.io/center-node", "data-postgres-0", "foundation-credentials", "Never"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %s", want)
		}
	}
	if !requests["/api/v1/namespaces/if-env-test/persistentvolumeclaims/data-postgres-0"] {
		t.Fatal("PVC not created")
	}
}

func TestFoundationStatusDoesNotReportStaleReadyOrFailedImageAsRunning(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "pods") {
			_, _ = w.Write([]byte(`{"items":[{"status":{"containerStatuses":[{"state":{"waiting":{"reason":"ImagePullBackOff","message":"image unavailable"}}}]}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"metadata":{"generation":2},"status":{"observedGeneration":1,"readyReplicas":1}}`))
	}))
	defer server.Close()
	r := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL}
	status, message, err := r.foundationWorkloadStatus(context.Background(), "if-env-test", "postgres")
	if err != nil || status != "failed" || !strings.Contains(message, "ImagePullBackOff") {
		t.Fatalf("%s %s %v", status, message, err)
	}
}

func TestFoundationTraefikIsObservedInSystemNamespace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/apis/apps/v1/namespaces/kube-system/deployments/traefik" {
			t.Error(r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"metadata":{"generation":1},"status":{"observedGeneration":1,"readyReplicas":1}}`))
	}))
	defer server.Close()
	r := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL}
	status, _, err := r.foundationWorkloadStatus(context.Background(), "if-env-test", "traefik")
	if err != nil || status != "running" {
		t.Fatalf("%s %v", status, err)
	}
	for _, s := range foundationServiceTypes {
		if foundationWorkloadForService(s) == "" {
			t.Fatalf("unmapped service %s", s)
		}
	}
}

func TestFoundationDeleteDoesNotTreatForbiddenAsAbsent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) }))
	defer server.Close()
	r := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL}
	_, err := r.foundationRequest(context.Background(), http.MethodDelete, "/api/v1/namespaces/if-env-test", nil, nil)
	if err == nil {
		t.Fatal("forbidden deletion accepted")
	}
}

// 旧副本健康不能掩盖新节点缺镜像；旧节点的错误也不能污染切回后的状态。
func TestFoundationDeploymentSwitchStatus(t *testing.T) {
	for _, tc := range []struct{ name, podTarget, want string }{
		{"new target missing image", "new", "failed"},
		{"old target error ignored", "old", "pending"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if strings.Contains(req.URL.Path, "/deployments/") {
					_, _ = w.Write([]byte(`{"metadata":{"generation":2},"spec":{"replicas":1,"template":{"spec":{"nodeSelector":{"host":"new"}}}},"status":{"observedGeneration":2,"readyReplicas":1,"updatedReplicas":1,"replicas":2}}`))
					return
				}
				_, _ = w.Write([]byte(`{"items":[{"spec":{"nodeSelector":{"host":"` + tc.podTarget + `"}},"status":{"containerStatuses":[{"state":{"waiting":{"reason":"ErrImageNeverPull","message":"missing local image"}}}]}}]}`))
			}))
			defer server.Close()
			r := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL}
			status, _, err := r.foundationWorkloadStatus(context.Background(), "test", "nginx")
			if err != nil || status != tc.want {
				t.Fatalf("status=%s err=%v, want %s", status, err, tc.want)
			}
		})
	}
}
