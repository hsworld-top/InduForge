package codeworkspace

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestKubernetesCreateUsesRestrictedPermissionInitializer(t *testing.T) {
	var pod map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/services"):
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/pods"):
			if err := json.NewDecoder(r.Body).Decode(&pod); err != nil {
				t.Fatalf("解析 Pod 请求失败: %v", err)
			}
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	engine := &KubernetesEngine{namespace: "induforge-system", workspaceRoot: "/var/lib/induforge/center/workspaces", baseURL: server.URL, token: "test", client: server.Client()}
	spec := ContainerSpec{
		Name: "induforge-code-project-1", Image: "induforge/designer-code-server:test", Command: []string{"--bind-addr", "0.0.0.0:3000"}, WorkingDir: "/workspace",
		Environment: []string{"PNPM_HOME=/cache/pnpm"}, Labels: map[string]string{"com.induforge.managed": "true"},
		Mounts: []Mount{
			{Target: "/workspace", Subpath: "project-1/workspace"},
			{Target: "/cache", Subpath: "project-1/cache"},
		},
	}
	if err := engine.Create(context.Background(), spec); err != nil {
		t.Fatalf("创建 Kubernetes 工作区失败: %v", err)
	}

	encoded, err := json.Marshal(pod)
	if err != nil {
		t.Fatalf("序列化 Pod 失败: %v", err)
	}
	payload := string(encoded)
	for _, expected := range []string{
		`"name":"workspace-permissions"`,
		`"runAsUser":0`,
		`"allowPrivilegeEscalation":false`,
		`"drop":["ALL"]`,
		`"add":["CHOWN"]`,
		`"mountPath":"/project","name":"workspaces","subPath":"project-1"`,
		`"INDUFORGE_KUBERNETES_WORKSPACE","value":"true"`,
		`"name":"code-workspace"`,
	} {
		if !strings.Contains(payload, expected) {
			t.Fatalf("Kubernetes 工作区 Pod 缺少 %s: %s", expected, payload)
		}
	}
	if strings.Contains(payload, `"allowPrivilegeEscalation":true`) {
		t.Fatalf("工作区 Pod 不得允许权限提升: %s", payload)
	}

}

func TestKubernetesRemoveWaitsForResourceAbsence(t *testing.T) {
	getCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			getCount++
			if getCount == 1 {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	engine := &KubernetesEngine{namespace: "induforge-system", baseURL: server.URL, token: "test", client: server.Client()}
	if err := engine.removeResource(context.Background(), "/api/v1/namespaces/induforge-system/services/induforge-code-project-1"); err != nil {
		t.Fatalf("等待 Kubernetes 资源删除失败: %v", err)
	}
	if getCount != 2 {
		t.Fatalf("资源未确认消失即结束，GET 次数=%d", getCount)
	}
}
