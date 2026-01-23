package network

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/indu-forge/node_agent/internal/pkg/types"
	"github.com/indu-forge/node_agent/internal/pkg/logger"
)

// HeartbeatConfig 心跳配置
type HeartbeatConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Interval time.Duration `yaml:"interval"`
	Endpoint string        `yaml:"endpoint"`
}

// HeartbeatSender 心跳发送器
type HeartbeatSender struct {
	config            HeartbeatConfig
	nodeID            string
	registrationToken string
	logger            *logger.SimpleLogger
	metrics           map[string]interface{}
	projects          []string
}

// NewHeartbeatSender 创建心跳发送器
func NewHeartbeatSender(config HeartbeatConfig, nodeID string, registrationToken string) *HeartbeatSender {
	return &HeartbeatSender{
		config:            config,
		nodeID:            nodeID,
		registrationToken: registrationToken,
		logger:            logger.GlobalLogger,
		metrics:           make(map[string]interface{}),
		projects:          make([]string, 0),
	}
}

// Start 开始发送心跳
func (h *HeartbeatSender) Start(ctx context.Context) {
	if !h.config.Enabled {
		h.logger.Info("心跳发送已禁用")
		return
	}

	ticker := time.NewTicker(h.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.sendHeartbeat(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// sendHeartbeat 发送心跳
func (h *HeartbeatSender) sendHeartbeat(ctx context.Context) {
	// 收集指标
	metrics := h.collectMetrics()

	// 创建心跳请求
	req := types.HeartbeatRequest{
		NodeID:            h.nodeID,
		RegistrationToken: h.registrationToken,
		Timestamp:         time.Now(),
		Status:            "healthy",
		Metrics:           metrics,
		Projects:          h.projects,
		AgentVersion:      "1.0.0",
	}

	// 发送 HTTP 请求
	data, err := json.Marshal(req)
	if err != nil {
		h.logger.Error("序列化心跳请求失败", "error", err)
		return
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// 构建完整的心跳端点
	endpoint := fmt.Sprintf("%s/%s/heartbeat", h.config.Endpoint, h.nodeID)

	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(data))
	if err != nil {
		h.logger.Error("创建心跳请求失败", "error", err)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Registration-Token", h.registrationToken)

	resp, err := client.Do(httpReq)
	if err != nil {
		h.logger.Warn("发送心跳失败", "endpoint", endpoint, "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		h.logger.Warn("心跳响应异常", "status_code", resp.StatusCode)
	} else {
		h.logger.Debug("心跳发送成功")
		
		// 解析响应，获取待执行指令
		var response types.HeartbeatResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err == nil {
			if response.Success && len(response.Data.Commands) > 0 {
				h.logger.Info("收到待执行指令", "count", len(response.Data.Commands))
				// TODO: 处理待执行指令
			}
		}
	}
}

// collectMetrics 收集系统指标
func (h *HeartbeatSender) collectMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	// CPU 使用率（简化示例）
	metrics["cpu_usage"] = 0.0

	// 内存使用（简化示例）
	metrics["memory_usage"] = 0.0

	// 磁盘使用（简化示例）
	metrics["disk_usage"] = 0.0

	// 运行时数量
	metrics["running_projects"] = len(h.projects)

	// 主机名
	if hostname, err := os.Hostname(); err == nil {
		metrics["hostname"] = hostname
	}

	// 时间戳
	metrics["timestamp"] = time.Now().Unix()

	return metrics
}

// UpdateProjects 更新项目列表
func (h *HeartbeatSender) UpdateProjects(projects []string) {
	h.projects = projects
}
