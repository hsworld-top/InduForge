package codeworkspace

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDockerClientCreatesContainerWithoutPrivilegedHostAccess(t *testing.T) {
	var createPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/version":
			_ = json.NewEncoder(w).Encode(map[string]string{"ApiVersion": "1.47"})
		case "/v1.47/containers/create":
			if r.URL.Query().Get("name") != "induforge-code-test" {
				t.Fatalf("unexpected container name: %s", r.URL.RawQuery)
			}
			if err := json.NewDecoder(r.Body).Decode(&createPayload); err != nil {
				t.Fatal(err)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("{\"Id\":\"container\"}"))
		default:
			t.Fatalf("unexpected Docker API path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewDockerClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	err = client.Create(context.Background(), ContainerSpec{
		Name: "induforge-code-test", Image: "image", Command: []string{"code-server"},
		WorkingDir: "/workspace", ContainerPort: "3000/tcp", BindHost: "127.0.0.1",
		Mounts: []Mount{{Source: "induforge-control-workspaces", Subpath: "project/workspace", Target: "/workspace"}},
		Labels: map[string]string{"com.induforge.managed": "true"},
	})
	if err != nil {
		t.Fatal(err)
	}
	hostConfig := createPayload["HostConfig"].(map[string]any)
	if _, exists := hostConfig["Privileged"]; exists {
		t.Fatal("create payload must not enable privileged mode")
	}
	mounts := hostConfig["Mounts"].([]any)
	if len(mounts) != 1 {
		t.Fatalf("unexpected mounts: %#v", mounts)
	}
	mount := mounts[0].(map[string]any)
	if mount["Type"] != "volume" || mount["Source"] != "induforge-control-workspaces" || mount["Target"] != "/workspace" {
		t.Fatalf("unexpected mount: %#v", mount)
	}
	volumeOptions := mount["VolumeOptions"].(map[string]any)
	if volumeOptions["Subpath"] != "project/workspace" {
		t.Fatalf("unexpected volume subpath: %#v", volumeOptions)
	}
}

func TestDockerClientMapsNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/version" {
			_ = json.NewEncoder(w).Encode(map[string]string{"ApiVersion": "1.47"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "No such container"})
	}))
	defer server.Close()
	client, err := NewDockerClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Inspect(context.Background(), "missing")
	if !errors.Is(err, ErrContainerNotFound) {
		t.Fatalf("expected ErrContainerNotFound, got %v", err)
	}
}
