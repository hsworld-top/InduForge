package network

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/indu-forge/node_agent/internal/pkg/logger"
	"github.com/indu-forge/node_agent/internal/pkg/types"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
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

	// 首次启动时立即上报一次，避免首次状态同步等待一个周期。
	h.sendHeartbeat(ctx)

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

	// CPU 使用率（0~1）
	if percents, err := cpu.Percent(0, false); err == nil && len(percents) > 0 {
		metrics["cpu"] = percents[0] / 100.0
	} else {
		metrics["cpu"] = 0.0
	}

	// 内存使用率（0~1）
	if vm, err := mem.VirtualMemory(); err == nil {
		metrics["memory"] = float64(vm.UsedPercent) / 100.0
	} else {
		metrics["memory"] = 0.0
	}

	// 磁盘使用率（0~1）
	diskPath, diskLabel := getDiskPath()
	if du, err := disk.Usage(diskPath); err == nil {
		metrics["disk"] = float64(du.UsedPercent) / 100.0
	} else {
		metrics["disk"] = 0.0
	}
	metrics["disk_label"] = diskLabel

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

// getDiskPath 获取当前系统默认磁盘路径与标签
func getDiskPath() (string, string) {
	exePath, err := os.Executable()
	if err != nil {
		if runtime.GOOS == "windows" {
			return "C:\\", "C"
		}
		return "/", "System"
	}

	exeDir := filepath.Dir(exePath)
	if runtime.GOOS == "windows" {
		volume := filepath.VolumeName(exeDir) // e.g. C:
		if volume == "" {
			return "C:\\", "C"
		}
		return volume + "\\", volume
	}

	// Linux/Unix 直接使用程序所在路径，disk.Usage 会解析到对应挂载点
	return exeDir, "System"
}


// UpdateProjects 更新项目列表
func (h *HeartbeatSender) UpdateProjects(projects []string) {
	h.projects = projects
}
