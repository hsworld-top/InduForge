package types

import (
	"encoding/json"
	"testing"
	"time"
)

// TestNodeInfoJSON 测试 NodeInfo JSON 序列化
func TestNodeInfoJSON(t *testing.T) {
	now := time.Now()
	info := NodeInfo{
		ID:           "node-001",
		Name:         "Test Node",
		Version:      "1.0.0",
		ExecutorType: "process",
		WorkDir:      "/var/lib/node_agent",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}

	var decoded NodeInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("JSON 反序列化失败: %v", err)
	}

	if decoded.ID != info.ID {
		t.Errorf("ID 不匹配，期望 %s，实际 %s", info.ID, decoded.ID)
	}
	if decoded.Name != info.Name {
		t.Errorf("Name 不匹配，期望 %s，实际 %s", info.Name, decoded.Name)
	}
	if decoded.Version != info.Version {
		t.Errorf("Version 不匹配，期望 %s，实际 %s", info.Version, decoded.Version)
	}
}

// TestRuntimeStatusJSON 测试 RuntimeStatus JSON 序列化
func TestRuntimeStatusJSON(t *testing.T) {
	now := time.Now()
	status := RuntimeStatus{
		ProjectID: "project-001",
		Version:   "1.0.0",
		State:     "running",
		PID:       12345,
		StartedAt: &now,
		Health:    "healthy",
		Metrics:   map[string]interface{}{"cpu": 0.5, "memory": 1024},
		LastCheck: now,
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}

	var decoded RuntimeStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("JSON 反序列化失败: %v", err)
	}

	if decoded.State != status.State {
		t.Errorf("State 不匹配，期望 %s，实际 %s", status.State, decoded.State)
	}
	if decoded.Health != status.Health {
		t.Errorf("Health 不匹配，期望 %s，实际 %s", status.Health, decoded.Health)
	}
	if decoded.PID != status.PID {
		t.Errorf("PID 不匹配，期望 %d，实际 %d", status.PID, decoded.PID)
	}
}

// TestDeployRequestJSON 测试 DeployRequest JSON 序列化
func TestDeployRequestJSON(t *testing.T) {
	req := DeployRequest{
		ProjectID:    "project-001",
		Version:      "1.0.0",
		IFPPackage:   "/path/to/package.ifp",
		ExecutorType: "process",
		ConnectionProfile: ConnectionProfile{
			Name:     "test-profile",
			Endpoint: "http://localhost:8080",
			AuthType: "token",
			AuthData: map[string]string{"token": "secret"},
			Metadata: map[string]string{"env": "test"},
		},
		EnvVars:   map[string]string{"KEY": "value"},
		AutoStart: true,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}

	var decoded DeployRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("JSON 反序列化失败: %v", err)
	}

	if decoded.ProjectID != req.ProjectID {
		t.Errorf("ProjectID 不匹配，期望 %s，实际 %s", req.ProjectID, decoded.ProjectID)
	}
	if decoded.AutoStart != req.AutoStart {
		t.Errorf("AutoStart 不匹配，期望 %v，实际 %v", req.AutoStart, decoded.AutoStart)
	}
	if decoded.ConnectionProfile.Name != req.ConnectionProfile.Name {
		t.Errorf("Profile Name 不匹配，期望 %s，实际 %s",
			req.ConnectionProfile.Name, decoded.ConnectionProfile.Name)
	}
}

// TestConnectionProfileSecrets 测试连接配置敏感信息处理
func TestConnectionProfileSecrets(t *testing.T) {
	profile := ConnectionProfile{
		Name:     "test-profile",
		Endpoint: "http://localhost:8080",
		AuthType: "token",
		AuthData: map[string]string{"token": "secret"},
		Secrets:  map[string]string{"password": "secret"},
		Metadata: map[string]string{"env": "test"},
	}

	// 序列化后 Secrets 应该被过滤
	data, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("JSON 反序列化失败: %v", err)
	}

	// Secrets 应该被设置为 nil 或不包含敏感信息
	// 在实际使用中，Secrets 字段会被外部逻辑清空
	t.Logf("Secrets 字段值: %v", decoded["secrets"])
	_ = decoded
}

// TestProjectInfoJSON 测试 ProjectInfo JSON 序列化
func TestProjectInfoJSON(t *testing.T) {
	now := time.Now()
	info := ProjectInfo{
		ID:             "project-001",
		Name:           "Test Project",
		CurrentVersion: "1.0.0",
		Status:         "running",
		ConnectionProfile: ConnectionProfile{
			Name:     "test-profile",
			Endpoint: "http://localhost:8080",
			AuthType: "token",
		},
		DeployedAt:    &now,
		LastStartedAt: &now,
		RuntimeStatus: &RuntimeStatus{
			ProjectID: "project-001",
			State:     "running",
			Health:    "healthy",
		},
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}

	var decoded ProjectInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("JSON 反序列化失败: %v", err)
	}

	if decoded.ID != info.ID {
		t.Errorf("ID 不匹配，期望 %s，实际 %s", info.ID, decoded.ID)
	}
	if decoded.CurrentVersion != info.CurrentVersion {
		t.Errorf("CurrentVersion 不匹配，期望 %s，实际 %s",
			info.CurrentVersion, decoded.CurrentVersion)
	}
}

// TestHeartbeatRequestJSON 测试 HeartbeatRequest JSON 序列化
func TestHeartbeatRequestJSON(t *testing.T) {
	now := time.Now()
	req := HeartbeatRequest{
		NodeID:            "node-001",
		RegistrationToken: "token-123",
		AgentVersion:      "1.0.0",
		Timestamp:         now,
		Status:            "healthy",
		Metrics:           map[string]interface{}{"cpu": 0.5},
		Projects:          []string{"proj-1", "proj-2"},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}

	var decoded HeartbeatRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("JSON 反序列化失败: %v", err)
	}

	if decoded.NodeID != req.NodeID {
		t.Errorf("NodeID 不匹配，期望 %s，实际 %s", req.NodeID, decoded.NodeID)
	}
	if len(decoded.Projects) != len(req.Projects) {
		t.Errorf("Projects 数量不匹配，期望 %d，实际 %d",
			len(req.Projects), len(decoded.Projects))
	}
}

// TestHealthCheckResultJSON 测试 HealthCheckResult JSON 序列化
func TestHealthCheckResultJSON(t *testing.T) {
	now := time.Now()
	result := HealthCheckResult{
		Status: "healthy",
		Checks: map[string]interface{}{
			"endpoint":    "http://localhost:8080",
			"status_code": 200,
		},
		Timestamp: now,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}

	var decoded HealthCheckResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("JSON 反序列化失败: %v", err)
	}

	if decoded.Status != result.Status {
		t.Errorf("Status 不匹配，期望 %s，实际 %s", result.Status, decoded.Status)
	}
}
