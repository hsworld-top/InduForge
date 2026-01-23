package health

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/indu-forge/node_agent/internal/pkg/types"
	"github.com/indu-forge/node_agent/internal/pkg/logger"
)

// HealthChecker 健康检查器
type HealthChecker struct {
	config     HealthConfig
	logger     *logger.SimpleLogger
	projects   map[string]*ProjectHealth
	mu         sync.RWMutex
}

// HealthConfig 健康检查配置
type HealthConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
	Retries  int           `yaml:"retries"`
}

// ProjectHealth 项目健康状态
type ProjectHealth struct {
	ProjectID    string                 `json:"projectId"`
	Status       string                 `json:"status"`
	LastCheck    time.Time              `json:"lastCheck"`
	Checks       map[string]interface{} `json:"checks"`
	FailureCount int                    `json:"failureCount"`
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(config HealthConfig) *HealthChecker {
	return &HealthChecker{
		config:   config,
		logger:   logger.GlobalLogger,
		projects: make(map[string]*ProjectHealth),
	}
}

// CheckRuntime 检查运行时健康
func (h *HealthChecker) CheckRuntime(ctx context.Context, projectID string, endpoint string) (*types.HealthCheckResult, error) {
	if !h.config.Enabled {
		return &types.HealthCheckResult{
			Status:    "disabled",
			Checks:    map[string]interface{}{},
			Timestamp: time.Now(),
		}, nil
	}

	client := &http.Client{
		Timeout: h.config.Timeout,
	}

	url := fmt.Sprintf("http://localhost:%s/%s", "8080", endpoint)

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("健康检查请求失败: %w", err)
	}
	defer resp.Body.Close()

	result := &types.HealthCheckResult{
		Status:    "healthy",
		Checks:    map[string]interface{}{
			"endpoint":    endpoint,
			"status_code": resp.StatusCode,
		},
		Timestamp: time.Now(),
	}

	if resp.StatusCode != http.StatusOK {
		result.Status = "unhealthy"
	}

	return result, nil
}

// MonitorProject 监控项目
func (h *HealthChecker) MonitorProject(ctx context.Context, projectID string, endpoint string) {
	if !h.config.Enabled {
		return
	}

	ticker := time.NewTicker(h.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			result, err := h.CheckRuntime(ctx, projectID, endpoint)
			if err != nil {
				h.logger.Warn("健康检查失败", "project", projectID, "error", err)
				h.recordFailure(projectID)
				continue
			}

			h.updateProjectHealth(projectID, result)

		case <-ctx.Done():
			return
		}
	}
}

// GetProjectHealth 获取项目健康状态
func (h *HealthChecker) GetProjectHealth(projectID string) *ProjectHealth {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.projects[projectID]
}

// RecordFailure 记录失败次数
func (h *HealthChecker) recordFailure(projectID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if health, ok := h.projects[projectID]; ok {
		health.FailureCount++
		if health.FailureCount >= h.config.Retries {
			health.Status = "unhealthy"
		}
	}
}

// updateProjectHealth 更新项目健康状态
func (h *HealthChecker) updateProjectHealth(projectID string, result *types.HealthCheckResult) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if health, ok := h.projects[projectID]; ok {
		health.LastCheck = result.Timestamp
		health.Status = result.Status
		health.Checks = result.Checks
		health.FailureCount = 0
	} else {
		h.projects[projectID] = &ProjectHealth{
			ProjectID:    projectID,
			Status:       result.Status,
			LastCheck:    result.Timestamp,
			Checks:       result.Checks,
			FailureCount: 0,
		}
	}
}
