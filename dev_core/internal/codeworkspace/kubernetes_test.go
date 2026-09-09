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
	var service map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/services"):
			if err := json.NewDecoder(r.Body).Decode(&service); err != nil {
				t.Fatalf("解析 Service 请求失败: %v", err)
			}
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
		Environment: []string{"PNPM_HOME=/cache/pnpm", "PI_WEB_ALLOWED_HOSTS=ai-project-1.workspace.induforge.test", "__VITE_ADDITIONAL_SERVER_ALLOWED_HOSTS=preview-project-1.workspace.induforge.test"}, Labels: map[string]string{"com.induforge.managed": "true"},
		Mounts: []Mount{
			{Target: "/workspace", Subpath: "project-1/workspace"},
			{Target: "/workspace/.induforge/context", Subpath: "project-1/context-state", ReadOnly: true},
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
		`"mountPath":"/project/workspace/.induforge/context","name":"workspaces","readOnly":true,"subPath":"project-1/context-state"`,
		`chown 1000:1000 /project/workspace/.induforge`,
		`/project/cache /project/pi-agent`,
		`"INDUFORGE_KUBERNETES_WORKSPACE","value":"true"`,
		`"PI_WEB_ALLOWED_HOSTS","value":"ai-project-1.workspace.induforge.test"`,
		`"__VITE_ADDITIONAL_SERVER_ALLOWED_HOSTS","value":"preview-project-1.workspace.induforge.test"`,
		`"name":"code-workspace"`,
	} {
		if !strings.Contains(payload, expected) {
			t.Fatalf("Kubernetes 工作区 Pod 缺少 %s: %s", expected, payload)
		}
	}
	if strings.Contains(payload, `"allowPrivilegeEscalation":true`) {
		t.Fatalf("工作区 Pod 不得允许权限提升: %s", payload)
	}
	servicePayload, err := json.Marshal(service)
	if err != nil {
		t.Fatalf("序列化 Service 失败: %v", err)
	}
	if !strings.Contains(string(servicePayload), `"type":"ClusterIP"`) || strings.Contains(string(servicePayload), `"type":"NodePort"`) || strings.Contains(string(servicePayload), `"nodePort"`) {
		t.Fatalf("Kubernetes 工作区必须只发布 ClusterIP: %s", servicePayload)
	}
	if strings.Contains(payload, "chown -R") {
		t.Fatalf("初始化器不得递归遍历用户缓存目录: %s", payload)
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

func TestKubernetesInspectMapsEvictedWorkspaceToUnhealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/pods/"):
			_, _ = w.Write([]byte(`{
				"metadata":{"labels":{"com.induforge.managed":"true"}},
				"status":{"phase":"Failed","reason":"Evicted","containerStatuses":[{"ready":false}]}
			}`))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/services/"):
			_, _ = w.Write([]byte(`{"spec":{"ports":[{"name":"code","port":3000},{"name":"preview-control","port":5174}]}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	engine := &KubernetesEngine{namespace: "induforge-system", baseURL: server.URL, token: "test", client: server.Client()}
	state, err := engine.Inspect(context.Background(), "induforge-code-project-1")
	if err != nil {
		t.Fatalf("读取被驱逐工作区失败: %v", err)
	}
	if state.Running || state.Health != "unhealthy" {
		t.Fatalf("Evicted 工作区必须映射为不可用状态: %#v", state)
	}
	if state.ServicePorts["preview-control"] != "5174" || state.HostPort != "" {
		t.Fatalf("工作区 Service 端口未保留: %#v", state.ServicePorts)
	}
}
