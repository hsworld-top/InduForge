package designworkspace

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/project"
)

const testProjectID = "11111111-1111-4111-8111-111111111111"

type fakeProjects struct {
	item project.Project
}

func (f fakeProjects) Get(context.Context, string, string) (project.Project, error) {
	if f.item.ID == "" {
		return project.Project{}, project.ErrNotFound
	}
	return f.item, nil
}

func TestExecuteSupportsHTFileLifecycle(t *testing.T) {
	workspace := t.TempDir()
	for _, directory := range []string{"displays", "scenes", "models", "assets", "components", "materials", "symbols"} {
		if err := os.MkdirAll(filepath.Join(workspace, directory), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	service := NewService(fakeProjects{item: project.Project{ID: testProjectID, TenantID: "tenant", CreatedBy: "owner", Visibility: "private", WorkspacePath: workspace}})
	actor := auth.User{ID: "owner", TenantID: "tenant", Role: "DEVELOPER"}

	upload := mustAction(t, "upload", map[string]any{"path": "displays/main.json", "content": `{"name":"main"}`})
	if result, err := service.Execute(context.Background(), actor, testProjectID, upload); err != nil || result != true {
		t.Fatalf("保存失败: result=%v err=%v", result, err)
	}
	source := mustAction(t, "source", map[string]any{"url": "displays/main.json"})
	if result, err := service.Execute(context.Background(), actor, testProjectID, source); err != nil || result != `{"name":"main"}` {
		t.Fatalf("读取失败: result=%v err=%v", result, err)
	}
	explore := mustAction(t, "explore", "/displays")
	result, err := service.Execute(context.Background(), actor, testProjectID, explore)
	if err != nil || result.(map[string]any)["main.json"] != true {
		t.Fatalf("目录浏览失败: result=%v err=%v", result, err)
	}
	rename := mustAction(t, "rename", map[string]any{"old": "displays/main.json", "new": "displays/renamed.json"})
	if result, err := service.Execute(context.Background(), actor, testProjectID, rename); err != nil || result != true {
		t.Fatalf("重命名失败: result=%v err=%v", result, err)
	}
	paste := mustAction(t, "paste", map[string]any{"destDir": "displays", "fileList": []string{"displays/renamed.json"}})
	if result, err := service.Execute(context.Background(), actor, testProjectID, paste); err != nil || result != true {
		t.Fatalf("复制失败: result=%v err=%v", result, err)
	}
	if _, err := os.Stat(filepath.Join(workspace, "displays", "renamed-1.json")); err != nil {
		t.Fatalf("复制文件不存在: %v", err)
	}
}

func TestExecuteDecodesBase64Upload(t *testing.T) {
	workspace := t.TempDir()
	if err := os.Mkdir(filepath.Join(workspace, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	service := NewService(fakeProjects{item: project.Project{ID: testProjectID, TenantID: "tenant", CreatedBy: "owner", WorkspacePath: workspace}})
	actor := auth.User{ID: "owner", TenantID: "tenant", Role: "DEVELOPER"}
	content := []byte{1, 2, 3, 4}
	action := mustAction(t, "upload", map[string]any{"path": "assets/test.png", "content": "data:image/png;base64," + base64.StdEncoding.EncodeToString(content)})
	if _, err := service.Execute(context.Background(), actor, testProjectID, action); err != nil {
		t.Fatal(err)
	}
	stored, err := os.ReadFile(filepath.Join(workspace, "assets", "test.png"))
	if err != nil || string(stored) != string(content) {
		t.Fatalf("二进制内容不正确: %v %v", stored, err)
	}
}

func TestExecuteRejectsPlatformManagedContextPath(t *testing.T) {
	workspace := t.TempDir()
	service := NewService(fakeProjects{item: project.Project{ID: testProjectID, TenantID: "tenant", CreatedBy: "owner", Visibility: "private", WorkspacePath: workspace}})
	actor := auth.User{ID: "owner", TenantID: "tenant", Role: "DEVELOPER"}
	_, err := service.Execute(context.Background(), actor, testProjectID, mustAction(t, "upload", map[string]any{"path": ".induforge/context/README.md", "content": "tampered"}))
	if !errors.Is(err, ErrProtectedPath) {
		t.Fatalf("expected ErrProtectedPath, got %v", err)
	}
}

func TestOpenContentReturnsWorkspaceResource(t *testing.T) {
	workspace := t.TempDir()
	assetPath := filepath.Join(workspace, "assets", "pump.png")
	if err := os.MkdirAll(filepath.Dir(assetPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(assetPath, []byte{1, 2, 3}, 0o644); err != nil {
		t.Fatal(err)
	}
	service := NewService(fakeProjects{item: project.Project{ID: testProjectID, TenantID: "tenant", CreatedBy: "owner", WorkspacePath: workspace}})
	actor := auth.User{ID: "owner", TenantID: "tenant", Role: "DEVELOPER"}

	content, err := service.OpenContent(context.Background(), actor, testProjectID, "assets/pump.png")
	if err != nil {
		t.Fatal(err)
	}
	defer content.File.Close()
	data, err := os.ReadFile(content.File.Name())
	if err != nil || string(data) != string([]byte{1, 2, 3}) {
		t.Fatalf("资源内容不正确: %v %v", data, err)
	}
}

func TestExecuteRejectsTraversalAndProtectedRoot(t *testing.T) {
	workspace := t.TempDir()
	if err := os.Mkdir(filepath.Join(workspace, "displays"), 0o755); err != nil {
		t.Fatal(err)
	}
	service := NewService(fakeProjects{item: project.Project{ID: testProjectID, TenantID: "tenant", CreatedBy: "owner", WorkspacePath: workspace}})
	actor := auth.User{ID: "owner", TenantID: "tenant", Role: "DEVELOPER"}
	if _, err := service.Execute(context.Background(), actor, testProjectID, mustAction(t, "source", map[string]any{"url": "../secret.txt"})); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("路径越界应被拒绝: %v", err)
	}
	if _, err := service.Execute(context.Background(), actor, testProjectID, mustAction(t, "remove", "displays")); !errors.Is(err, ErrProtectedPath) {
		t.Fatalf("系统目录删除应被拒绝: %v", err)
	}
	if _, err := service.Execute(context.Background(), actor, testProjectID, mustAction(t, "remove", "displays/..")); !errors.Is(err, ErrProtectedPath) {
		t.Fatalf("归一化到工程根的删除应被拒绝: %v", err)
	}
}

func TestExecuteRequiresProjectWriteAccess(t *testing.T) {
	workspace := t.TempDir()
	service := NewService(fakeProjects{item: project.Project{ID: testProjectID, TenantID: "tenant", CreatedBy: "owner", Visibility: "private", WorkspacePath: workspace}})
	actor := auth.User{ID: "other", TenantID: "tenant", Role: "VIEWER"}
	_, err := service.Execute(context.Background(), actor, testProjectID, mustAction(t, "locate", "scenes"))
	if !errors.Is(err, auth.ErrPermissionDenied) {
		t.Fatalf("无写权限用户应被拒绝: %v", err)
	}
}

func mustAction(t *testing.T, command string, data any) Action {
	t.Helper()
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	return Action{Command: command, Data: raw}
}
