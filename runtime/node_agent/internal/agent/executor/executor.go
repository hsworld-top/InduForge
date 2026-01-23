package executor

import (
	"context"

	"github.com/indu-forge/node_agent/internal/pkg/types"
)

// Executor 执行器接口
type Executor interface {
	// 部署
	Deploy(ctx context.Context, req types.DeployRequest) error

	// 启动
	Start(ctx context.Context, projectID string) error

	// 停止
	Stop(ctx context.Context, projectID string) error

	// 重启
	Restart(ctx context.Context, projectID string) error

	// 回滚
	Rollback(ctx context.Context, projectID string, version string) error

	// 获取状态
	GetStatus(ctx context.Context, projectID string) (*types.RuntimeStatus, error)

	// 健康检查
	HealthCheck(ctx context.Context) error
}
