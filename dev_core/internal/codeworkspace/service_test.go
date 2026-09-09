package codeworkspace

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/project"
)

const testProjectID = "11111111-1111-4111-8111-111111111111"

type fakeProjects struct{ item project.Project }

func (f fakeProjects) Get(_ context.Context, tenantID, projectID string) (project.Project, error) {
	if tenantID != f.item.TenantID || projectID != f.item.ID {
		return project.Project{}, project.ErrNotFound
	}
	return f.item, nil
}

type fakeEngine struct {
	state      ContainerState
	inspectErr error
	created    []ContainerSpec
	started    []string
	stopped    []string
	removed    []string
}

func (f *fakeEngine) Inspect(context.Context, string) (ContainerState, error) {
	if f.inspectErr != nil {
		return ContainerState{}, f.inspectErr
	}
	return f.state, nil
}
func (f *fakeEngine) Create(_ context.Context, spec ContainerSpec) error {
	f.created = append(f.created, spec)
	f.inspectErr = nil
	f.state = ContainerState{Name: spec.Name, Status: "created", Labels: spec.Labels}
	return nil
}
func (f *fakeEngine) Start(_ context.Context, name string) error {
	f.started = append(f.started, name)
	f.state.Running = true
	f.state.Status = "running"
	f.state.HostPort = "49152"
	return nil
}
func (f *fakeEngine) Stop(_ context.Context, name string) error {
	f.stopped = append(f.stopped, name)
	f.state.Running = false
	f.state.Status = "exited"
	return nil
}
func (f *fakeEngine) Remove(_ context.Context, name string) error {
	f.removed = append(f.removed, name)
	f.inspectErr = ErrContainerNotFound
	return nil
}

func TestStartCreatesDeterministicSharedContainerWithControlledMounts(t *testing.T) {
	localRoot := t.TempDir()
	item := project.Project{
		ID: testProjectID, TenantID: "tenant", CreatedBy: "owner", Visibility: "private",
		WorkspacePath: filepath.Join(localRoot, testProjectID, "workspace"),
	}
	engine := &fakeEngine{inspectErr: ErrContainerNotFound}
	service, err := NewService(fakeProjects{item: item}, engine, Config{
		Image: "induforge/code-workspace:test", BindHost: "127.0.0.1", VolumeName: "induforge-control-workspaces",
	})
	if err != nil {
		t.Fatal(err)
	}

	status, err := service.Start(context.Background(), auth.User{ID: "owner", TenantID: "tenant", Role: "PROJECT_ADMIN"}, testProjectID)
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "running" || status.HostPort != "49152" {
		t.Fatalf("unexpected status: %#v", status)
	}
	if len(engine.created) != 1 || len(engine.started) != 1 {
		t.Fatalf("unexpected lifecycle: created=%d started=%d", len(engine.created), len(engine.started))
	}
	spec := engine.created[0]
	if spec.Name != "induforge-code-"+testProjectID {
		t.Fatalf("unexpected container name: %s", spec.Name)
	}
	if len(spec.Mounts) != 6 {
		t.Fatalf("unexpected mounts: %#v", spec.Mounts)
	}
	wanted := map[string]bool{
		"/workspace":                           false,
		"/workspace/.induforge/context":        true,
		"/home/coder/.local/share/code-server": false,
		"/home/coder/.config/code-server":      false,
		"/cache":                               false,
		"/home/coder/.pi/agent":                false,
	}
	for _, mount := range spec.Mounts {
		if mount.Target == "/workspace/.induforge/context" && mount.Subpath != testProjectID+"/context-state" {
			t.Fatal("必须挂载稳定的上下文父目录，不能挂载会被替换的 current")
		}
		readOnly, ok := wanted[mount.Target]
		if !ok || readOnly != mount.ReadOnly {
			t.Fatalf("unexpected mount: %#v", mount)
		}
		if mount.Target == "/var/run/docker.sock" {
			t.Fatal("code-server 子容器不得挂载 Docker Socket")
		}
	}
	if len(spec.Environment) != 3 {
		t.Fatalf("子容器不应继承控制面环境: %#v", spec.Environment)
	}
	if !slices.Contains(spec.Environment, "npm_config_store_dir=/cache/pnpm-store") {
		t.Fatalf("pnpm store 未指向共享缓存: %#v", spec.Environment)
	}
}

func TestStartReusesExistingProjectContainer(t *testing.T) {
	root := t.TempDir()
	item := project.Project{ID: testProjectID, TenantID: "tenant", CreatedBy: "owner", Visibility: "private", WorkspacePath: filepath.Join(root, testProjectID, "workspace")}
	engine := &fakeEngine{state: ContainerState{
		Name: containerName(testProjectID), Status: "running", Running: true, HostPort: "49153",
		Labels: map[string]string{"com.induforge.managed": "true", "com.induforge.project-id": testProjectID, "com.induforge.role": "code-workspace"},
	}}
	service, err := NewService(fakeProjects{item: item}, engine, Config{Image: "image", VolumeName: "induforge-control-workspaces"})
	if err != nil {
		t.Fatal(err)
	}
	status, err := service.Start(context.Background(), auth.User{ID: "owner", TenantID: "tenant", Role: "PROJECT_ADMIN"}, testProjectID)
	if err != nil {
		t.Fatal(err)
	}
	if status.HostPort != "49153" || len(engine.created) != 0 || len(engine.started) != 0 {
		t.Fatalf("existing shared container was not reused: %#v", status)
	}
}

func TestStatusReportsPendingClusterWorkspaceAsStarting(t *testing.T) {
	root := t.TempDir()
	item := project.Project{ID: testProjectID, TenantID: "tenant", CreatedBy: "owner", Visibility: "private", WorkspacePath: filepath.Join(root, testProjectID, "workspace")}
	service, err := NewService(fakeProjects{item: item}, &fakeEngine{state: ContainerState{
		Name: containerName(testProjectID), Health: "starting",
		Labels: map[string]string{"com.induforge.managed": "true", "com.induforge.project-id": testProjectID, "com.induforge.role": "code-workspace"},
	}}, Config{Image: "image", VolumeName: "workspaces"})
	if err != nil {
		t.Fatal(err)
	}
	status, err := service.Status(context.Background(), auth.User{ID: "owner", TenantID: "tenant", Role: "PROJECT_ADMIN"}, testProjectID)
	if err != nil || status.Status != "starting" {
		t.Fatalf("K3s Pending 工作区状态错误: %#v, %v", status, err)
	}
}

func TestStartReportsUnhealthyWorkspaceAsErrorWithoutRestart(t *testing.T) {
	root := t.TempDir()
	item := project.Project{ID: testProjectID, TenantID: "tenant", CreatedBy: "owner", Visibility: "private", WorkspacePath: filepath.Join(root, testProjectID, "workspace")}
	engine := &fakeEngine{state: ContainerState{
		Name: containerName(testProjectID), Health: "unhealthy",
		Labels: map[string]string{"com.induforge.managed": "true", "com.induforge.project-id": testProjectID, "com.induforge.role": "code-workspace"},
	}}
	service, err := NewService(fakeProjects{item: item}, engine, Config{Image: "image", VolumeName: "workspaces"})
	if err != nil {
		t.Fatal(err)
	}
	status, err := service.Start(context.Background(), auth.User{ID: "owner", TenantID: "tenant", Role: "PROJECT_ADMIN"}, testProjectID)
	if err != nil || status.Status != "error" {
		t.Fatalf("失败工作区必须明确返回 error: %#v, %v", status, err)
	}
	if len(engine.started) != 0 {
		t.Fatalf("失败工作区不得被当作可启动容器: %#v", engine.started)
	}
}

func TestBuiltinDemoWorkspaceReceivesOfficialDefaultTemplateOnly(t *testing.T) {
	root := t.TempDir()
	demoID := "00000000-0000-4000-8000-000000000001"
	item := project.Project{ID: demoID, TenantID: "tenant", CreatedBy: "owner", Visibility: "internal", WorkspacePath: filepath.Join(root, demoID, "workspace")}
	engine := &fakeEngine{inspectErr: ErrContainerNotFound}
	service, err := NewService(fakeProjects{item: item}, engine, Config{
		Image: "image", VolumeName: "workspaces", DefaultTemplateProjectID: demoID, DefaultTemplateID: "vite-vue-js",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Start(context.Background(), auth.User{ID: "owner", TenantID: "tenant", Role: "PROJECT_ADMIN"}, demoID); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(engine.created[0].Environment, "INDUFORGE_DEFAULT_WORKSPACE_TEMPLATE=vite-vue-js") {
		t.Fatalf("内置教程工程未接收官方默认模板: %#v", engine.created[0].Environment)
	}
	if _, err := NewService(fakeProjects{}, &fakeEngine{}, Config{Image: "image", VolumeName: "workspaces", DefaultTemplateProjectID: demoID}); err == nil {
		t.Fatal("不完整默认模板配置必须拒绝")
	}
}

func TestWorkspacePublicOriginInjectsExactAIAndViteHosts(t *testing.T) {
	root := t.TempDir()
	item := project.Project{ID: testProjectID, TenantID: "tenant", CreatedBy: "owner", Visibility: "private", WorkspacePath: filepath.Join(root, testProjectID, "workspace")}
	engine := &fakeEngine{inspectErr: ErrContainerNotFound}
	service, err := NewService(fakeProjects{item: item}, engine, Config{
		Image: "image", VolumeName: "workspaces", WorkspacePublicOriginTemplate: "https://{service}-{projectId}.workspace.induforge.test:18443",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Start(context.Background(), auth.User{ID: "owner", TenantID: "tenant", Role: "PROJECT_ADMIN"}, testProjectID); err != nil {
		t.Fatal(err)
	}
	environment := engine.created[0].Environment
	if !slices.Contains(environment, "PI_WEB_ALLOWED_HOSTS=ai-"+testProjectID+".workspace.induforge.test") ||
		!slices.Contains(environment, "__VITE_ADDITIONAL_SERVER_ALLOWED_HOSTS=preview-"+testProjectID+".workspace.induforge.test") {
		t.Fatalf("工作区未注入精确 AI/Vite Host: %#v", environment)
	}
	for _, value := range environment {
		if strings.Contains(value, "workspace.induforge.test:") || strings.Contains(value, "*") {
			t.Fatalf("服务允许 Host 不得携带端口或通配符: %#v", environment)
		}
	}
}

func TestWorkspacePublicOriginRejectsWildcardHost(t *testing.T) {
	if _, err := NewService(fakeProjects{}, &fakeEngine{}, Config{Image: "image", VolumeName: "workspaces", WorkspacePublicOriginTemplate: "https://{service}-{projectId}.*.example.test"}); err == nil {
		t.Fatal("工作区公开模板不得接受通配 Host")
	}
}

func TestCodeWorkspaceRequiresProjectWriteAccess(t *testing.T) {
	root := t.TempDir()
	item := project.Project{ID: testProjectID, TenantID: "tenant", CreatedBy: "owner", Visibility: "internal", WorkspacePath: filepath.Join(root, testProjectID, "workspace")}
	service, err := NewService(fakeProjects{item: item}, &fakeEngine{inspectErr: ErrContainerNotFound}, Config{Image: "image", VolumeName: "induforge-control-workspaces"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Status(context.Background(), auth.User{ID: "viewer", TenantID: "tenant", Role: "VIEWER"}, testProjectID)
	if !errors.Is(err, auth.ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestRebuildPreservesPersistentDirectories(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, testProjectID, "workspace")
	item := project.Project{ID: testProjectID, TenantID: "tenant", CreatedBy: "owner", Visibility: "private", WorkspacePath: workspace}
	engine := &fakeEngine{state: ContainerState{
		Name: containerName(testProjectID), Running: true,
		Labels: map[string]string{"com.induforge.managed": "true", "com.induforge.project-id": testProjectID, "com.induforge.role": "code-workspace"},
	}}
	service, err := NewService(fakeProjects{item: item}, engine, Config{Image: "image", VolumeName: "induforge-control-workspaces"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Rebuild(context.Background(), auth.User{ID: "owner", TenantID: "tenant", Role: "PROJECT_ADMIN"}, testProjectID); err != nil {
		t.Fatal(err)
	}
	if len(engine.removed) != 1 || len(engine.created) != 1 || len(engine.started) != 1 {
		t.Fatalf("unexpected rebuild lifecycle: removed=%d created=%d started=%d", len(engine.removed), len(engine.created), len(engine.started))
	}
	for _, relative := range []string{"workspace", "code-server-data", "code-server-config", "cache", filepath.Join("context-state", "current")} {
		if _, err := os.Stat(filepath.Join(root, testProjectID, relative)); err != nil {
			t.Fatalf("persistent directory missing after rebuild: %s: %v", relative, err)
		}
	}
}
