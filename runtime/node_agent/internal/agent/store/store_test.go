package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/indu-forge/node_agent/internal/pkg/types"
)

func TestLocalStore_ProjectCRUD(t *testing.T) {
	st := NewLocalStore(filepath.Join(t.TempDir(), "data"))
	t.Cleanup(func() { _ = st.Close() })

	now := time.Now().UTC().Truncate(time.Second)
	project := &types.ProjectInfo{
		ID:             "p1",
		Name:           "project-1",
		Source:         types.ProjectSourceCenter,
		CurrentVersion: "v1.0.0",
		Status:         "running",
		ConnectionProfile: types.ConnectionProfile{
			Name:     "c1",
			Endpoint: "http://127.0.0.1:9099",
		},
		DeployedAt:    &now,
		LastStartedAt: &now,
	}
	if err := st.SaveProject(project); err != nil {
		t.Fatalf("保存项目失败: %v", err)
	}

	got, err := st.GetProject("p1")
	if err != nil {
		t.Fatalf("获取项目失败: %v", err)
	}
	if got.CurrentVersion != "v1.0.0" || got.Status != "running" || got.Source != types.ProjectSourceCenter {
		t.Fatalf("项目字段不正确: %+v", got)
	}

	list, err := st.ListProjects()
	if err != nil {
		t.Fatalf("列出项目失败: %v", err)
	}
	if len(list) != 1 || list[0].ID != "p1" {
		t.Fatalf("项目列表不正确: %+v", list)
	}

	if err := st.DeleteProject("p1"); err != nil {
		t.Fatalf("删除项目失败: %v", err)
	}
	if _, err := st.GetProject("p1"); err == nil {
		t.Fatalf("删除后仍可查询到项目")
	}
}

func TestLocalStore_ConnectionProfile(t *testing.T) {
	st := NewLocalStore(filepath.Join(t.TempDir(), "data"))
	t.Cleanup(func() { _ = st.Close() })

	profile := types.ConnectionProfile{
		Name:     "profile-1",
		Endpoint: "opc.tcp://127.0.0.1:4840",
		AuthType: "token",
		Secrets: map[string]string{
			"token": "secret-value",
		},
	}
	if err := st.SaveConnectionProfile("p1", profile); err != nil {
		t.Fatalf("保存连接配置失败: %v", err)
	}
	got, err := st.GetConnectionProfile("p1")
	if err != nil {
		t.Fatalf("获取连接配置失败: %v", err)
	}
	if got.Name != "profile-1" || got.Endpoint == "" {
		t.Fatalf("连接配置不正确: %+v", got)
	}
	if got.Secrets != nil {
		t.Fatalf("Secrets 应被脱敏: %+v", got.Secrets)
	}
}

func TestLocalStore_VersionAndSwitch(t *testing.T) {
	st := NewLocalStore(filepath.Join(t.TempDir(), "data"))
	t.Cleanup(func() { _ = st.Close() })

	if err := st.SaveVersion("p1", "v1.0.0", map[string][]byte{
		"runtime_engine": []byte("binary"),
	}); err != nil {
		t.Fatalf("保存版本失败: %v", err)
	}
	if err := st.SaveVersion("p1", "v1.1.0", map[string][]byte{
		"runtime_engine": []byte("binary2"),
	}); err != nil {
		t.Fatalf("保存版本失败: %v", err)
	}

	versions, err := st.ListVersions("p1")
	if err != nil {
		t.Fatalf("列出版本失败: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("版本数量不正确: %v", versions)
	}

	if err := st.SwitchVersion("p1", "v1.1.0"); err != nil {
		t.Fatalf("切换版本失败: %v", err)
	}
	current, err := st.GetCurrentVersion("p1")
	if err != nil {
		t.Fatalf("读取当前版本失败: %v", err)
	}
	if current != "v1.1.0" {
		t.Fatalf("当前版本错误: %s", current)
	}

	versionDir, err := st.GetVersionDir("p1", "v1.1.0")
	if err != nil {
		t.Fatalf("获取版本目录失败: %v", err)
	}
	if _, err := os.Stat(filepath.Join(versionDir, "runtime_engine")); err != nil {
		t.Fatalf("版本文件不存在: %v", err)
	}
}
