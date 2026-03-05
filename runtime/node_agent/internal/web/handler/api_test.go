package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/indu-forge/node_agent/internal/agent/store"
	"github.com/indu-forge/node_agent/internal/pkg/logger"
	"github.com/indu-forge/node_agent/internal/pkg/types"
)

// createTestHandler 创建用于测试的 API 处理器
func createTestHandler(t *testing.T) *APIHandler {
	tmpDir := t.TempDir()
	st := store.NewLocalStore(tmpDir)
	t.Cleanup(func() { _ = st.Close() })
	handler := NewAPIHandler(nil, st)
	return handler
}

// TestGetNodeInfo 测试获取节点信息
func TestGetNodeInfo(t *testing.T) {
	handler := createTestHandler(t)
	req := httptest.NewRequest("GET", "/api/v1/node/info", nil)
	rec := httptest.NewRecorder()

	handler.GetNodeInfo(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("期望状态码 %d，实际 %d", http.StatusOK, rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	id, idOK := response["id"].(string)
	if !idOK || id == "" {
		t.Errorf("期望节点 ID 为非空字符串，实际 %v", response["id"])
	}
	machineID, machineIDOK := response["machineId"].(string)
	if !machineIDOK || machineID == "" {
		t.Errorf("期望 machineId 为非空字符串，实际 %v", response["machineId"])
	}
	if id != machineID {
		t.Errorf("期望 id 与 machineId 一致，实际 id=%s machineId=%s", id, machineID)
	}
	if response["name"] == nil || response["name"] != "Node Agent" {
		t.Errorf("期望节点名称 Node Agent，实际 %v", response["name"])
	}
}

// TestRollbackProject_MissingVersion 测试缺少版本参数的回滚请求
func TestRollbackProject_MissingVersion(t *testing.T) {
	handler := createTestHandler(t)

	// 创建不带 version 参数的请求
	req := httptest.NewRequest("POST", "/api/v1/projects/test/rollback", nil)
	rec := httptest.NewRecorder()

	handler.RollbackProject(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 %d，实际 %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if response["error"] != "version 参数不能为空" {
		t.Errorf("期望错误消息 'version 参数不能为空'，实际 %s", response["error"])
	}
}

// TestGetProfile_MissingProjectId 测试缺少 projectId 的获取配置请求
func TestGetProfile_MissingProjectId(t *testing.T) {
	handler := createTestHandler(t)

	req := httptest.NewRequest("GET", "/api/v1/profile", nil)
	rec := httptest.NewRecorder()

	handler.GetProfile(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 %d，实际 %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if response["error"] != "projectId 参数不能为空" {
		t.Errorf("期望错误消息 'projectId 参数不能为空'，实际 %s", response["error"])
	}
}

// TestSaveProfile_MissingProjectId 测试缺少 projectId 的保存配置请求
func TestSaveProfile_MissingProjectId(t *testing.T) {
	handler := createTestHandler(t)

	body := `{"name": "test", "endpoint": "http://localhost"}`
	req := httptest.NewRequest("POST", "/api/v1/profile", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.SaveProfile(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 %d，实际 %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if response["error"] != "projectId 参数不能为空" {
		t.Errorf("期望错误消息 'projectId 参数不能为空'，实际 %s", response["error"])
	}
}

// TestHealthCheck 测试健康检查
func TestHealthCheck(t *testing.T) {
	handler := createTestHandler(t)
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	handler.HealthCheck(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("期望状态码 %d，实际 %d", http.StatusOK, rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("期望状态 healthy，实际 %s", response["status"])
	}
	if _, ok := response["time"]; !ok || response["time"] == "" {
		t.Error("响应应该包含时间戳")
	}
}

// TestRegisterRoutes 测试路由注册
func TestRegisterRoutes(t *testing.T) {
	handler := createTestHandler(t)
	router := handler.RegisterRoutes()

	if router == nil {
		t.Fatal("路由不应为 nil")
	}

	// 测试路由是否正确注册 - 通过尝试访问一个已知路由来验证
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, httptest.NewRequest("GET", "/health", nil))

	if rec.Code != http.StatusOK {
		t.Error("健康检查路由应该已注册")
	}
}

// TestJSONResponse 测试 JSON 响应方法
func TestJSONResponse(t *testing.T) {
	handler := createTestHandler(t)

	testData := map[string]string{"key": "value"}
	rec := httptest.NewRecorder()

	handler.jsonResponse(rec, http.StatusOK, testData)

	if rec.Code != http.StatusOK {
		t.Errorf("期望状态码 %d，实际 %d", http.StatusOK, rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("期望 Content-Type application/json，实际 %s", contentType)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if response["key"] != "value" {
		t.Errorf("期望值 value，实际 %s", response["key"])
	}
}

// TestLoggerNotNil 测试日志记录器不为 nil
func TestLoggerNotNil(t *testing.T) {
	handler := createTestHandler(t)
	if handler.logger == nil {
		t.Error("logger 不应为 nil")
	}

	// 测试日志级别
	if handler.logger != logger.GlobalLogger {
		t.Log("使用全局日志实例")
	}
}

// TestStartProject_CenterManagedForbidden 测试中心托管项目禁止本地启动
func TestStartProject_CenterManagedForbidden(t *testing.T) {
	tmpDir := t.TempDir()
	st := store.NewLocalStore(tmpDir)
	t.Cleanup(func() { _ = st.Close() })
	_ = st.SaveProject(&types.ProjectInfo{
		ID:             "center-proj",
		Name:           "center-proj",
		Source:         types.ProjectSourceCenter,
		CurrentVersion: "1.0.0",
		Status:         "stopped",
	})

	// 创建 bootstrap ready，避免被初始化拦截
	bootstrapPath := filepath.Join(tmpDir, "bootstrap.json")
	_ = os.WriteFile(bootstrapPath, []byte(`{"status":"READY","updatedAt":"2026-01-01T00:00:00Z"}`), 0644)

	handler := NewAPIHandler(nil, st)
	handler.bootstrapStore = &BootstrapStore{
		filePath: bootstrapPath,
		state: BootstrapState{
			Status: BootstrapReady,
		},
	}

	req := httptest.NewRequest("POST", "/api/v1/projects/center-proj/start", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "center-proj"})
	rec := httptest.NewRecorder()
	handler.StartProject(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("期望状态码 %d，实际 %d", http.StatusForbidden, rec.Code)
	}
}

// TestEnsureProjectMutable_CenterSourceAllowed 测试中心来源可操作中心托管项目
func TestEnsureProjectMutable_CenterSourceAllowed(t *testing.T) {
	tmpDir := t.TempDir()
	st := store.NewLocalStore(tmpDir)
	t.Cleanup(func() { _ = st.Close() })
	_ = st.SaveProject(&types.ProjectInfo{
		ID:             "center-proj",
		Name:           "center-proj",
		Source:         types.ProjectSourceCenter,
		CurrentVersion: "1.0.0",
		Status:         "stopped",
	})

	handler := NewAPIHandler(nil, st)
	req := httptest.NewRequest("POST", "/api/v1/projects/center-proj/start?controlSource=center", nil)

	if err := handler.ensureProjectMutable(req, "center-proj"); err != nil {
		t.Fatalf("中心来源应允许操作中心托管项目，实际报错: %v", err)
	}
}
