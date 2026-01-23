package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// NodeMode 节点运行模式
type NodeMode string

const (
	ModeOnline  NodeMode = "online"
	ModeOffline NodeMode = "offline"
)

// OnlineConfig 在线模式配置
type OnlineConfig struct {
	CenterURL         string `yaml:"centerUrl"`
	NodeID            string `yaml:"nodeId"`
	RegistrationToken string `yaml:"registrationToken"`
}

// ModeConfig 模式配置
type ModeConfig struct {
	Mode   NodeMode     `yaml:"mode"`
	Online OnlineConfig `yaml:"online"`
}

// GetMode 获取当前模式
func GetMode(configPath string) (NodeMode, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ModeOffline, err
	}

	var config struct {
		Agent struct {
			Mode NodeMode `yaml:"mode"`
		} `yaml:"agent"`
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return ModeOffline, err
	}

	if config.Agent.Mode == "" {
		return ModeOffline, nil
	}

	return config.Agent.Mode, nil
}

// IsOnline 判断是否在线模式
func IsOnline(configPath string) bool {
	mode, _ := GetMode(configPath)
	return mode == ModeOnline
}

// GetOnlineConfig 获取在线模式配置
func GetOnlineConfig(configPath string) (*OnlineConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config struct {
		Agent struct {
			Online OnlineConfig `yaml:"online"`
		} `yaml:"agent"`
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config.Agent.Online, nil
}

// SaveOnlineConfig 保存在线模式配置
func SaveOnlineConfig(configPath string, onlineConfig OnlineConfig) error {
	// 读取现有配置
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}

	// 更新配置
	agent, ok := config["agent"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid config structure")
	}

	agent["mode"] = string(ModeOnline)
	agent["online"] = map[string]interface{}{
		"centerUrl":         onlineConfig.CenterURL,
		"nodeId":            onlineConfig.NodeID,
		"registrationToken": onlineConfig.RegistrationToken,
	}

	// 写回配置文件
	newData, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	// 创建备份
	backupPath := configPath + ".backup"
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// 写入新配置
	if err := os.WriteFile(configPath, newData, 0644); err != nil {
		return err
	}

	return nil
}

// ValidateOnlineConfig 验证在线模式配置是否完整
func ValidateOnlineConfig(config *OnlineConfig) error {
	if config.CenterURL == "" {
		return fmt.Errorf("centerUrl is required")
	}
	if config.NodeID == "" {
		return fmt.Errorf("nodeId is required")
	}
	if config.RegistrationToken == "" {
		return fmt.Errorf("registrationToken is required")
	}
	return nil
}

// GetConfigPath 获取配置文件路径
func GetConfigPath() string {
	// 优先从环境变量获取
	if path := os.Getenv("NODE_AGENT_CONFIG"); path != "" {
		return path
	}

	// 默认路径
	return filepath.Join(".", "configs", "config.yaml")
}
