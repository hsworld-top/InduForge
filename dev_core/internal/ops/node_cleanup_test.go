package ops

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNodeCleanupOnlyDeletesTargetWorkloads(t *testing.T) {
	var deleted []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deleted = append(deleted, r.URL.Path)
			w.WriteHeader(404)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/deployments") {
			w.Write([]byte(`{"items":[{"metadata":{"name":"mine"},"spec":{"template":{"spec":{"nodeSelector":{"induforge.io/host-node-id":"target"}}}}},{"metadata":{"name":"other"},"spec":{"template":{"spec":{"nodeSelector":{"induforge.io/host-node-id":"other"}}}}}]}`))
			return
		}
		w.Write([]byte(`{"items":[]}`))
	}))
	defer server.Close()
	k := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL}
	if err := k.cleanupNodeWorkloads(context.Background(), "env", "target"); err != nil {
		t.Fatal(err)
	}
	if len(deleted) != 2 {
		t.Fatalf("deletes=%v", deleted)
	}
	for _, path := range deleted {
		if !strings.HasSuffix(path, "/mine") {
			t.Fatalf("wrong target: %s", path)
		}
	}
}

func TestNodeCleanupDeletesOrphanPodsWithUID(t *testing.T) {
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			body, _ := io.ReadAll(r.Body)
			if r.URL.Path != "/api/v1/namespaces/env/pods/old" || !strings.Contains(string(body), `"uid":"old-uid"`) || !strings.Contains(string(body), `"gracePeriodSeconds":0`) {
				t.Errorf("unsafe delete: %s %s", r.URL.Path, body)
			}
			deleted = true
			w.WriteHeader(200)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/pods") {
			w.Write([]byte(`{"items":[{"metadata":{"name":"old","uid":"old-uid"},"spec":{"nodeSelector":{"induforge.io/host-node-id":"target"}}},{"metadata":{"name":"new","uid":"new-uid"},"spec":{"nodeSelector":{"induforge.io/host-node-id":"replacement"}}}]}`))
			return
		}
		w.Write([]byte(`{"items":[]}`))
	}))
	defer server.Close()
	k := &KubernetesProjectReconciler{client: server.Client(), endpoint: server.URL}
	if err := k.cleanupNodeWorkloads(context.Background(), "env", "target"); err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("orphan pod was not removed")
	}
}
