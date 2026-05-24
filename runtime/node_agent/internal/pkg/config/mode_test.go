package config

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestConfig 创建测试配置文件
func setupTestConfig(t *testing.T, content string) string {
	t.Helper()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// 如果没有提供内容，创建一个基本的离线配置
	if content == "" {
		content = `agent:
  mode: offline
`
	}

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	return configPath
}

// TestGetMode_DefaultOffline 测试默认模式为离线
func TestGetMode_DefaultOffline(t *testing.T) {
	configPath := setupTestConfig(t, "")

	mode, err := GetMode(configPath)

	if err != nil {
		t.Fatalf("GetMode 应返回空配置时的默认值，错误: %v", err)
	}

	if mode != ModeOffline {
		t.Errorf("期望模式为 %s，实际为 %s", ModeOffline, mode)
	}
}

// TestGetMode_OnlineMode 测试在线模式
func TestGetMode_OnlineMode(t *testing.T) {
	configContent := `
agent:
  mode: online
`
	configPath := setupTestConfig(t, configContent)

	mode, err := GetMode(configPath)

	if err != nil {
		t.Fatalf("GetMode 失败: %v", err)
	}

	if mode != ModeOnline {
		t.Errorf("期望模式为 %s，实际为 %s", ModeOnline, mode)
	}
}

// TestGetMode_ExplicitOfflineMode 测试显式离线模式
func TestGetMode_ExplicitOfflineMode(t *testing.T) {
	configContent := `
agent:
  mode: offline
`
	configPath := setupTestConfig(t, configContent)

	mode, err := GetMode(configPath)

	if err != nil {
		t.Fatalf("GetMode 失败: %v", err)
	}

	if mode != ModeOffline {
		t.Errorf("期望模式为 %s，实际为 %s", ModeOffline, mode)
	}
}

// TestGetMode_FileNotFound 测试文件不存在
func TestGetMode_FileNotFound(t *testing.T) {
	configPath := "/nonexistent/path/config.yaml"

	mode, err := GetMode(configPath)

	if err == nil {
		t.Error("期望错误，但未返回错误")
	}

	if mode != ModeOffline {
		t.Errorf("期望模式为 %s，实际为 %s", ModeOffline, mode)
	}
}

// TestIsOnline 测试在线判断
func TestIsOnline(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "空配置应返回离线",
			content:  "",
			expected: false,
		},
		{
			name: "在线模式应返回 true",
			content: `
agent:
  mode: online
`,
			expected: true,
		},
		{
			name: "离线模式应返回 false",
			content: `
agent:
  mode: offline
`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := setupTestConfig(t, tt.content)
			result := IsOnline(configPath)

			if result != tt.expected {
				t.Errorf("期望 %v，实际 %v", tt.expected, result)
			}
		})
	}
}

// TestGetOnlineConfig 测试获取在线配置
func TestGetOnlineConfig(t *testing.T) {
	configContent := `
agent:
  online:
    centerUrl: "http://localhost:8080"
    nodeId: "test-node-001"
    registrationToken: "test-token-123"
`
	configPath := setupTestConfig(t, configContent)

	config, err := GetOnlineConfig(configPath)

	if err != nil {
		t.Fatalf("GetOnlineConfig 失败: %v", err)
	}

	if config.CenterURL != "http://localhost:8080" {
		t.Errorf("期望 CenterURL 为 http://localhost:8080，实际为 %s", config.CenterURL)
	}

	if config.NodeID != "test-node-001" {
		t.Errorf("期望 NodeID 为 test-node-001，实际为 %s", config.NodeID)
	}

	if config.RegistrationToken != "test-token-123" {
		t.Errorf("期望 RegistrationToken 为 test-token-123，实际为 %s", config.RegistrationToken)
	}
}

// TestGetOnlineConfig_NotFound 测试文件不存在
func TestGetOnlineConfig_NotFound(t *testing.T) {
	configPath := "/nonexistent/config.yaml"

	_, err := GetOnlineConfig(configPath)

	if err == nil {
		t.Error("期望错误，但未返回错误")
	}
}

// TestValidateOnlineConfig 测试验证在线配置
func TestValidateOnlineConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      OnlineConfig
		expectError bool
	}{
		{
			name: "完整配置应通过验证",
			config: OnlineConfig{
				CenterURL:         "http://localhost:8080",
				NodeID:            "test-node",
				RegistrationToken: "token",
			},
			expectError: false,
		},
		{
			name: "缺少 centerUrl 应失败",
			config: OnlineConfig{
				NodeID:            "test-node",
				RegistrationToken: "token",
			},
			expectError: true,
		},
		{
			name: "缺少 nodeId 应失败",
			config: OnlineConfig{
				CenterURL:         "http://localhost:8080",
				RegistrationToken: "token",
			},
			expectError: true,
		},
		{
			name: "缺少 registrationToken 应失败",
			config: OnlineConfig{
				CenterURL: "http://localhost:8080",
				NodeID:    "test-node",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOnlineConfig(&tt.config)

			if tt.expectError && err == nil {
				t.Error("期望错误，但未返回错误")
			}

			if !tt.expectError && err != nil {
				t.Errorf("不期望错误，但返回: %v", err)
			}
		})
	}
}

// TestGetConfigPath 测试配置路径获取
func TestGetConfigPath(t *testing.T) {
	// 测试环境变量
	os.Setenv("NODE_AGENT_CONFIG", "/custom/path/config.yaml")
	defer os.Unsetenv("NODE_AGENT_CONFIG")

	path := GetConfigPath()

	if path != "/custom/path/config.yaml" {
		t.Errorf("期望 /custom/path/config.yaml，实际为 %s", path)
	}
}

// TestGetConfigPath_Default 测试默认配置路径
func TestGetConfigPath_Default(t *testing.T) {
	os.Unsetenv("NODE_AGENT_CONFIG")

	path := GetConfigPath()

	expected := "./" + filepath.Join("config.yaml")
	if path != expected {
		t.Errorf("期望 %s，实际为 %s", expected, path)
	}
}
