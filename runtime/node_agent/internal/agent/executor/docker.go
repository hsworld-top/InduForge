package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/indu-forge/node_agent/internal/pkg/logger"
	"github.com/indu-forge/node_agent/internal/pkg/types"
)

// DockerExecutor Docker 执行器
type DockerExecutor struct {
	client      *client.Client
	network     string
	imagePrefix string
	logger      *logger.SimpleLogger
}

// NewDockerExecutor 创建 Docker 执行器
func NewDockerExecutor(socket, network, imagePrefix string) (*DockerExecutor, error) {
	cli, err := client.NewClientWithOpts(client.WithHost(socket))
	if err != nil {
		return nil, fmt.Errorf("创建 Docker 客户端失败: %w", err)
	}

	return &DockerExecutor{
		client:      cli,
		network:     network,
		imagePrefix: imagePrefix,
		logger:      logger.GlobalLogger,
	}, nil
}

// Deploy 部署
func (e *DockerExecutor) Deploy(ctx context.Context, req types.DeployRequest) error {
	e.logger.Info("Docker 执行器部署", "project", req.ProjectID, "version", req.Version)

	imageName := e.getImageName(req.ProjectID, req.Version)

	// 这里应该从 IFP 包构建镜像
	// 实际实现中需要解压 IFP 包，构建镜像
	// 示例中简化处理
	e.logger.Info("构建镜像", "image", imageName)

	return nil
}

// Start 启动
func (e *DockerExecutor) Start(ctx context.Context, projectID string) error {
	e.logger.Info("启动 Docker 容器", "project", projectID)

	imageName := e.getImageName(projectID, "latest")
	containerName := e.getContainerName(projectID)

	// 检查容器是否已存在
	if e.containerExists(ctx, containerName) {
		return fmt.Errorf("容器已存在: %s", containerName)
	}

	// 创建容器
	resp, err := e.client.ContainerCreate(ctx,
		&container.Config{
			Image: imageName,
			Env: []string{
				"PROJECT_ID=" + projectID,
			},
		},
		nil, // hostConfig
		&network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				e.network: {},
			},
		},
		nil, // platform
		containerName)

	if err != nil {
		return fmt.Errorf("创建容器失败: %w", err)
	}

	// 启动容器
	if err := e.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("启动容器失败: %w", err)
	}

	e.logger.Info("容器启动成功", "project", projectID, "container", containerName)
	return nil
}

// Stop 停止
func (e *DockerExecutor) Stop(ctx context.Context, projectID string) error {
	e.logger.Info("停止 Docker 容器", "project", projectID)

	containerName := e.getContainerName(projectID)

	// 停止容器
	if err := e.client.ContainerStop(ctx, containerName, container.StopOptions{}); err != nil {
		e.logger.Warn("停止容器失败", "error", err)
		return err
	}

	// 删除容器
	if err := e.client.ContainerRemove(ctx, containerName, container.RemoveOptions{}); err != nil {
		e.logger.Warn("删除容器失败", "error", err)
	}

	return nil
}

// Restart 重启
func (e *DockerExecutor) Restart(ctx context.Context, projectID string) error {
	e.logger.Info("重启 Docker 容器", "project", projectID)

	containerName := e.getContainerName(projectID)
	return e.client.ContainerRestart(ctx, containerName, container.StopOptions{})
}

// Rollback 回滚
func (e *DockerExecutor) Rollback(ctx context.Context, projectID string, version string) error {
	e.logger.Info("回滚 Docker 容器", "project", projectID, "version", version)

	// 停止当前容器
	if err := e.Stop(ctx, projectID); err != nil {
		return err
	}

	// 使用新版本镜像启动
	e.logger.Info("使用回滚版本启动", "project", projectID, "version", version)
	return e.Start(ctx, projectID)
}

// GetStatus 获取状态
func (e *DockerExecutor) GetStatus(ctx context.Context, projectID string) (*types.RuntimeStatus, error) {
	containerName := e.getContainerName(projectID)

	containerJSON, err := e.client.ContainerInspect(ctx, containerName)
	if err != nil {
		return &types.RuntimeStatus{
			ProjectID: projectID,
			State:     "stopped",
			Health:    "unhealthy",
		}, nil
	}

	state := containerJSON.State
	running := state.Running

	if running {
		return &types.RuntimeStatus{
			ProjectID: projectID,
			State:     "running",
			PID:       state.Pid,
			Health:    "healthy",
			LastCheck: time.Now(),
		}, nil
	}

	return &types.RuntimeStatus{
		ProjectID: projectID,
		State:     "stopped",
		Health:    "unhealthy",
	}, nil
}

// HealthCheck 健康检查
func (e *DockerExecutor) HealthCheck(ctx context.Context) error {
	if _, err := e.client.Ping(ctx); err != nil {
		return fmt.Errorf("Docker 健康检查失败: %w", err)
	}
	return nil
}

// 辅助函数
func (e *DockerExecutor) getImageName(projectID, version string) string {
	return fmt.Sprintf("%s_%s:%s", e.imagePrefix, projectID, version)
}

func (e *DockerExecutor) getContainerName(projectID string) string {
	return fmt.Sprintf("node_agent_%s", projectID)
}

func (e *DockerExecutor) containerExists(ctx context.Context, name string) bool {
	_, err := e.client.ContainerInspect(ctx, name)
	return err == nil
}
