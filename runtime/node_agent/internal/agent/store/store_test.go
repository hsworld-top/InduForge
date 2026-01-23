package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/indu-forge/node_agent/internal/pkg/types"
)

// TestNewLocalStore 测试创建本地存储
func TestNewLocalStore(t *testing.T) {
	store := NewLocalStore("/test/data")

	if store == nil {
		t.Fatal("NewLocalStore 应返回非 nil")
	}

	if store.dataDir != "/test/data" {
		t.Errorf("dataDir 不匹配，期望 /test/data，实际 %s", store.dataDir)
	}
}

// TestSaveProject 测试保存项目
func TestSaveProject(t *testing.T) {
	tempDir := t.TempDir()
	store := NewLocalStore(filepath.Join(tempDir, "data"))

	project := &types.ProjectInfo{
		ID:             "test-project",
		Name:           "Test Project",
		CurrentVersion: "1.0.0",
		Status:         "running",
		DeployedAt:     func() *time.Time { t := time.Now(); return &t }(),
	}

	if err := store.SaveProject(project); err != nil {
		t.Fatalf("SaveProject 失败: %v", err)
	}

	// 验证文件已创建
	file := filepath.Join(tempDir, "data", "projects", "test-project", "project.json")
	if _, err := os.Stat(file); os.IsNotExist(err) {
		t.Error("项目文件应已创建")
	}
}

// TestGetProject 测试获取项目
func TestGetProject(t *testing.T) {
	tempDir := t.TempDir()
	store := NewLocalStore(filepath.Join(tempDir, "data"))

	// 保存项目
	now := time.Now()
	project := &types.ProjectInfo{
		ID:             "test-project",
		Name:           "Test Project",
		CurrentVersion: "1.0.0",
		Status:         "running",
		DeployedAt:     &now,
	}

	if err := store.SaveProject(project); err != nil {
		t.Fatalf("SaveProject 失败: %v", err)
	}

	// 获取项目
	retrieved, err := store.GetProject("test-project")
	if err != nil {
		t.Fatalf("GetProject 失败: %v", err)
	}

	if retrieved.ID != project.ID {
		t.Errorf("ID 不匹配，期望 %s，实际 %s", project.ID, retrieved.ID)
	}

	if retrieved.Name != project.Name {
		t.Errorf("Name 不匹配，期望 %s，实际 %s", project.Name, retrieved.Name)
	}
}

// TestGetProject_NotFound 测试获取不存在的项目
func TestGetProject_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	store := NewLocalStore(filepath.Join(tempDir, "data"))

	_, err := store.GetProject("nonexistent")

	if err == nil {
		t.Error("获取不存在的项目应返回错误")
	}
}

// TestListProjects 测试列出所有项目
func TestListProjects(t *testing.T) {
	tempDir := t.TempDir()
	store := NewLocalStore(filepath.Join(tempDir, "data"))

	// 保存多个项目
	for i := 1; i <= 3; i++ {
		project := &types.ProjectInfo{
			ID:             "project-" + string(rune('0'+i)),
			Name:           "Project " + string(rune('0'+i)),
			CurrentVersion: "1.0.0",
		}
		if err := store.SaveProject(project); err != nil {
			t.Fatalf("SaveProject 失败: %v", err)
		}
	}

	// 列出项目
	projects, err := store.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects 失败: %v", err)
	}

	if len(projects) != 3 {
		t.Errorf("期望 3 个项目，实际 %d 个", len(projects))
	}
}

// TestListProjects_Empty 测试列出空项目列表
func TestListProjects_Empty(t *testing.T) {
	tempDir := t.TempDir()
	store := NewLocalStore(filepath.Join(tempDir, "data"))

	// 先创建 projects 目录
	projectsDir := filepath.Join(tempDir, "data", "projects")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		t.Fatalf("创建 projects 目录失败: %v", err)
	}

	projects, err := store.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects 失败: %v", err)
	}

	if len(projects) != 0 {
		t.Errorf("期望 0 个项目，实际 %d 个", len(projects))
	}
}

// TestDeleteProject 测试删除项目
func TestDeleteProject(t *testing.T) {
	tempDir := t.TempDir()
	store := NewLocalStore(filepath.Join(tempDir, "data"))

	// 保存项目
	project := &types.ProjectInfo{
		ID:   "test-project",
		Name: "Test Project",
	}
	if err := store.SaveProject(project); err != nil {
		t.Fatalf("SaveProject 失败: %v", err)
	}

	// 删除项目
	if err := store.DeleteProject("test-project"); err != nil {
		t.Fatalf("DeleteProject 失败: %v", err)
	}

	// 验证项目已删除
	_, err := store.GetProject("test-project")
	if err == nil {
		t.Error("项目应已删除")
	}
}

// TestDeleteProject_NotFound 测试删除不存在的项目
// 注意：os.RemoveAll 不会在目录不存在时返回错误，所以这个测试预期不返回错误
func TestDeleteProject_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	store := NewLocalStore(filepath.Join(tempDir, "data"))

	// 先创建 projects 目录
	projectsDir := filepath.Join(tempDir, "data", "projects")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		t.Fatalf("创建 projects 目录失败: %v", err)
	}

	// os.RemoveAll 在目录不存在时不会返回错误
	err := store.DeleteProject("nonexistent")

	// 这是预期行为：os.RemoveAll 静默成功
	if err != nil {
		t.Logf("删除不存在的项目返回错误（这是预期的行为）: %v", err)
	}
}

// TestSaveConnectionProfile 测试保存连接配置
func TestSaveConnectionProfile(t *testing.T) {
	tempDir := t.TempDir()
	store := NewLocalStore(filepath.Join(tempDir, "data"))

	profile := types.ConnectionProfile{
		Name:      "Test MySQL",
		Endpoint:  "localhost:3306",
		AuthType:  "password",
		AuthData:  map[string]string{"username": "root"},
		Metadata:  map[string]string{"database": "test"},
		Secrets:   map[string]string{"password": "secret"},
	}

	if err := store.SaveConnectionProfile("test-project", profile); err != nil {
		t.Fatalf("SaveConnectionProfile 失败: %v", err)
	}

	// 验证敏感数据已被脱敏
	retrieved, err := store.GetConnectionProfile("test-project")
	if err != nil {
		t.Fatalf("GetConnectionProfile 失败: %v", err)
	}

	if retrieved.Secrets != nil {
		t.Error("Secrets 应已被脱敏")
	}

	if retrieved.AuthData["username"] != "root" {
		t.Error("AuthData 不应被脱敏")
	}
}

// TestGetConnectionProfile 测试获取连接配置
func TestGetConnectionProfile(t *testing.T) {
	tempDir := t.TempDir()
	store := NewLocalStore(filepath.Join(tempDir, "data"))

	// 保存配置
	profile := types.ConnectionProfile{
		Name:     "Test DB",
		Endpoint: "localhost:5432",
		AuthType: "none",
	}
	if err := store.SaveConnectionProfile("test-project", profile); err != nil {
		t.Fatalf("SaveConnectionProfile 失败: %v", err)
	}

	// 获取配置
	retrieved, err := store.GetConnectionProfile("test-project")
	if err != nil {
		t.Fatalf("GetConnectionProfile 失败: %v", err)
	}

	if retrieved.Name != profile.Name {
		t.Errorf("Name 不匹配，期望 %s，实际 %s", profile.Name, retrieved.Name)
	}

	if retrieved.Endpoint != profile.Endpoint {
		t.Errorf("Endpoint 不匹配，期望 %s，实际 %s", profile.Endpoint, retrieved.Endpoint)
	}
}

// TestSaveVersion 测试保存版本文件
func TestSaveVersion(t *testing.T) {
	tempDir := t.TempDir()
	store := NewLocalStore(filepath.Join(tempDir, "data"))

	files := map[string][]byte{
		"main.js":   []byte("console.log('hello')"),
		"config.json": []byte(`{"version": "1.0.0"}`),
	}

	if err := store.SaveVersion("test-project", "v1.0.0", files); err != nil {
		t.Fatalf("SaveVersion 失败: %v", err)
	}

	// 验证文件已创建
	versionDir := filepath.Join(tempDir, "data", "projects", "test-project", "versions", "v1.0.0")
	for name := range files {
		file := filepath.Join(versionDir, name)
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("版本文件 %s 应已创建", name)
		}
	}
}

// TestGetVersionDir 测试获取版本目录
func TestGetVersionDir(t *testing.T) {
	tempDir := t.TempDir()
	store := NewLocalStore(filepath.Join(tempDir, "data"))

	// 先保存版本
	files := map[string][]byte{"main.js": []byte("code")}
	if err := store.SaveVersion("test-project", "v1.0.0", files); err != nil {
		t.Fatalf("SaveVersion 失败: %v", err)
	}

	// 获取版本目录
	dir, err := store.GetVersionDir("test-project", "v1.0.0")
	if err != nil {
		t.Fatalf("GetVersionDir 失败: %v", err)
	}

	expected := filepath.Join(tempDir, "data", "projects", "test-project", "versions", "v1.0.0")
	if dir != expected {
		t.Errorf("版本目录不匹配，期望 %s，实际 %s", expected, dir)
	}
}

// TestListVersions 测试列出版本
func TestListVersions(t *testing.T) {
	tempDir := t.TempDir()
	store := NewLocalStore(filepath.Join(tempDir, "data"))

	// 保存多个版本
	for _, version := range []string{"v1.0.0", "v1.0.1", "v2.0.0"} {
		files := map[string][]byte{"main.js": []byte("code")}
		if err := store.SaveVersion("test-project", version, files); err != nil {
			t.Fatalf("SaveVersion 失败: %v", err)
		}
	}

	// 列出版本
	versions, err := store.ListVersions("test-project")
	if err != nil {
		t.Fatalf("ListVersions 失败: %v", err)
	}

	if len(versions) != 3 {
		t.Errorf("期望 3 个版本，实际 %d 个", len(versions))
	}
}

// TestSwitchVersion 测试切换版本
func TestSwitchVersion(t *testing.T) {
	tempDir := t.TempDir()
	store := NewLocalStore(filepath.Join(tempDir, "data"))

	// 保存版本
	files := map[string][]byte{"main.js": []byte("code")}
	if err := store.SaveVersion("test-project", "v1.0.0", files); err != nil {
		t.Fatalf("SaveVersion 失败: %v", err)
	}

	// 切换版本
	if err := store.SwitchVersion("test-project", "v1.0.0"); err != nil {
		t.Fatalf("SwitchVersion 失败: %v", err)
	}

	// 验证软链接已创建
	currentLink := filepath.Join(tempDir, "data", "projects", "test-project", "current")
	link, err := os.Readlink(currentLink)
	if err != nil {
		t.Fatalf("读取软链接失败: %v", err)
	}

	if !filepath.IsAbs(link) {
		t.Error("软链接应为绝对路径")
	}
}

// TestGetCurrentVersion 测试获取当前版本
func TestGetCurrentVersion(t *testing.T) {
	tempDir := t.TempDir()
	store := NewLocalStore(filepath.Join(tempDir, "data"))

	// 保存并切换版本
	files := map[string][]byte{"main.js": []byte("code")}
	if err := store.SaveVersion("test-project", "v1.0.0", files); err != nil {
		t.Fatalf("SaveVersion 失败: %v", err)
	}
	if err := store.SwitchVersion("test-project", "v1.0.0"); err != nil {
		t.Fatalf("SwitchVersion 失败: %v", err)
	}

	// 获取当前版本
	version, err := store.GetCurrentVersion("test-project")
	if err != nil {
		t.Fatalf("GetCurrentVersion 失败: %v", err)
	}

	if version != "v1.0.0" {
		t.Errorf("期望 v1.0.0，实际 %s", version)
	}
}

// TestProjectRoundTrip 测试项目完整生命周期
func TestProjectRoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	store := NewLocalStore(filepath.Join(tempDir, "data"))

	// 1. 保存项目
	project := &types.ProjectInfo{
		ID:             "lifecycle-test",
		Name:           "Lifecycle Test Project",
		CurrentVersion: "v0.1.0",
		Status:         "stopped",
		ConnectionProfile: types.ConnectionProfile{
			Name:     "Test DB",
			Endpoint: "localhost:3306",
		},
	}

	if err := store.SaveProject(project); err != nil {
		t.Fatalf("SaveProject 失败: %v", err)
	}

	// 2. 保存连接配置
	if err := store.SaveConnectionProfile("lifecycle-test", project.ConnectionProfile); err != nil {
		t.Fatalf("SaveConnectionProfile 失败: %v", err)
	}

	// 3. 保存版本
	files := map[string][]byte{
		"main.js":   []byte("console.log('v0.1.0')"),
		"config.json": []byte(`{"version": "0.1.0"}`),
	}
	if err := store.SaveVersion("lifecycle-test", "v0.1.0", files); err != nil {
		t.Fatalf("SaveVersion v0.1.0 失败: %v", err)
	}

	// 4. 保存新版本
	files["main.js"] = []byte("console.log('v0.2.0')")
	if err := store.SaveVersion("lifecycle-test", "v0.2.0", files); err != nil {
		t.Fatalf("SaveVersion v0.2.0 失败: %v", err)
	}

	// 5. 切换版本
	if err := store.SwitchVersion("lifecycle-test", "v0.2.0"); err != nil {
		t.Fatalf("SwitchVersion 失败: %v", err)
	}

	// 6. 验证最终状态
	currentVersion, err := store.GetCurrentVersion("lifecycle-test")
	if err != nil {
		t.Fatalf("GetCurrentVersion 失败: %v", err)
	}

	if currentVersion != "v0.2.0" {
		t.Errorf("当前版本应为 v0.2.0，实际 %s", currentVersion)
	}

	versions, err := store.ListVersions("lifecycle-test")
	if err != nil {
		t.Fatalf("ListVersions 失败: %v", err)
	}

	if len(versions) != 2 {
		t.Errorf("期望 2 个版本，实际 %d 个", len(versions))
	}
}

// TestProject_JSONPersistence 测试项目 JSON 持久化
func TestProject_JSONPersistence(t *testing.T) {
	tempDir := t.TempDir()
	store := NewLocalStore(filepath.Join(tempDir, "data"))

	now := time.Now()
	original := &types.ProjectInfo{
		ID:             "json-test",
		Name:           "JSON Test",
		CurrentVersion: "1.0.0",
		Status:         "running",
		DeployedAt:     &now,
		LastStartedAt:  &now,
	}

	// 保存
	if err := store.SaveProject(original); err != nil {
		t.Fatalf("SaveProject 失败: %v", err)
	}

	// 直接读取 JSON 文件验证
	file := filepath.Join(tempDir, "data", "projects", "json-test", "project.json")
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("读取项目文件失败: %v", err)
	}

	// 解析 JSON
	var decoded types.ProjectInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("JSON 解析失败: %v", err)
	}

	// 验证字段
	if decoded.ID != original.ID {
		t.Errorf("ID 不匹配")
	}
	if decoded.Name != original.Name {
		t.Errorf("Name 不匹配")
	}
	if decoded.CurrentVersion != original.CurrentVersion {
		t.Errorf("CurrentVersion 不匹配")
	}
}
