package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/indu-forge/node_agent/internal/agent/executor"
	"github.com/indu-forge/node_agent/internal/agent/store"
	"github.com/indu-forge/node_agent/internal/pkg/logger"
	"github.com/indu-forge/node_agent/internal/pkg/types"
)

// State 状态枚举
type State string

const (
	StateDeploying   State = "deploying"
	StateRunning     State = "running"
	StateStopping    State = "stopping"
	StateStopped     State = "stopped"
	StateError       State = "error"
	StateRollingBack State = "rolling_back"
)

// Event 事件枚举
type Event string

const (
	EventDeploy     Event = "deploy"
	EventStart      Event = "start"
	EventStop       Event = "stop"
	EventRestart    Event = "restart"
	EventRollback   Event = "rollback"
	EventHealthOK   Event = "health_ok"
	EventHealthFail Event = "health_fail"
	EventError      Event = "error"
)

// StateMachine 状态机
type StateMachine struct {
	current     State
	transitions map[State]map[Event]State
}

func NewStateMachine() *StateMachine {
	return &StateMachine{
		current: StateStopped,
		transitions: map[State]map[Event]State{
			StateStopped: {
				EventDeploy:   StateDeploying,
				EventStart:    StateRunning,
				EventRollback: StateStopped,
			},
			StateDeploying: {
				EventHealthOK: StateRunning,
				EventError:    StateError,
			},
			StateRunning: {
				EventStop:     StateStopping,
				EventRestart:  StateRunning,
				EventRollback: StateRollingBack,
				EventError:    StateError,
			},
			StateStopping: {
				EventStop: StateStopped,
			},
			StateRollingBack: {
				EventHealthOK: StateStopped,
				EventError:    StateError,
			},
			StateError: {
				EventRollback: StateStopped,
			},
		},
	}
}

func (sm *StateMachine) Current() State {
	return sm.current
}

func (sm *StateMachine) Next(event Event) (State, error) {
	transitions, ok := sm.transitions[sm.current]
	if !ok {
		return sm.current, fmt.Errorf("无效的状态: %s", sm.current)
	}

	next, ok := transitions[event]
	if !ok {
		return sm.current, fmt.Errorf("无效的事件: %s", event)
	}

	sm.current = next
	return sm.current, nil
}

// Orchestrator 编排器
type Orchestrator struct {
	executor      executor.Executor
	store         *store.LocalStore
	logger        *logger.SimpleLogger
	stateMachines map[string]*StateMachine
}

// NewOrchestrator 创建编排器
func NewOrchestrator(exec executor.Executor, store *store.LocalStore) *Orchestrator {
	return &Orchestrator{
		executor:      exec,
		store:         store,
		logger:        logger.GlobalLogger,
		stateMachines: make(map[string]*StateMachine),
	}
}

// Deploy 部署项目
func (o *Orchestrator) Deploy(ctx context.Context, req types.DeployRequest) error {
	projectID := req.ProjectID
	o.logger.Info("开始部署", "project", projectID, "version", req.Version)

	// 创建状态机
	sm := NewStateMachine()
	o.stateMachines[projectID] = sm

	// 状态转换: stopped -> deploying
	if _, err := sm.Next(EventDeploy); err != nil {
		return fmt.Errorf("状态转换失败: %w", err)
	}

	// 1. 部署到执行器
	if err := o.executor.Deploy(ctx, req); err != nil {
		sm.Next(EventError)
		return fmt.Errorf("部署失败: %w", err)
	}

	// 2. 保存项目信息
	source := req.Source
	if source == "" {
		source = types.ProjectSourceLocal
	}
	project := &types.ProjectInfo{
		ID:                projectID,
		Name:              req.ProjectID,
		Source:            source,
		CurrentVersion:    req.Version,
		Status:            string(StateDeploying),
		ConnectionProfile: req.ConnectionProfile,
		DeployedAt:        &time.Time{},
	}

	*project.DeployedAt = time.Now()

	if err := o.store.SaveProject(project); err != nil {
		sm.Next(EventError)
		return fmt.Errorf("保存项目失败: %w", err)
	}

	// 3. 切换版本
	if err := o.store.SwitchVersion(projectID, req.Version); err != nil {
		sm.Next(EventError)
		return fmt.Errorf("切换版本失败: %w", err)
	}

	// 4. 保存连接配置
	if err := o.store.SaveConnectionProfile(projectID, req.ConnectionProfile); err != nil {
		sm.Next(EventError)
		return fmt.Errorf("保存连接配置失败: %w", err)
	}

	// 5. 状态转换: deploying -> running
	if _, err := sm.Next(EventHealthOK); err != nil {
		return fmt.Errorf("状态转换失败: %w", err)
	}

	// 6. 自动启动
	if req.AutoStart {
		if err := o.Start(ctx, projectID); err != nil {
			return fmt.Errorf("自动启动失败: %w", err)
		}
	}

	o.logger.Info("部署完成", "project", projectID)
	return nil
}

// Start 启动项目
func (o *Orchestrator) Start(ctx context.Context, projectID string) error {
	o.logger.Info("启动项目", "project", projectID)

	sm := o.getStateMachine(projectID)
	if sm == nil {
		return fmt.Errorf("项目未部署: %s", projectID)
	}

	// 状态转换: stopped -> running
	if _, err := sm.Next(EventStart); err != nil {
		return fmt.Errorf("状态转换失败: %w", err)
	}

	// 启动执行器
	if err := o.executor.Start(ctx, projectID); err != nil {
		sm.Next(EventError)
		return fmt.Errorf("启动失败: %w", err)
	}

	// 更新项目状态
	if project, err := o.store.GetProject(projectID); err == nil {
		project.Status = string(StateRunning)
		now := time.Now()
		project.LastStartedAt = &now
		o.store.SaveProject(project)
	}

	o.logger.Info("启动成功", "project", projectID)
	return nil
}

// Stop 停止项目
func (o *Orchestrator) Stop(ctx context.Context, projectID string) error {
	o.logger.Info("停止项目", "project", projectID)

	sm := o.getStateMachine(projectID)
	if sm == nil {
		return fmt.Errorf("项目未部署: %s", projectID)
	}

	// 状态转换: running -> stopping
	if _, err := sm.Next(EventStop); err != nil {
		return fmt.Errorf("状态转换失败: %w", err)
	}

	// 停止执行器
	if err := o.executor.Stop(ctx, projectID); err != nil {
		return fmt.Errorf("停止失败: %w", err)
	}

	// 状态转换: stopping -> stopped
	if _, err := sm.Next(EventStop); err != nil {
		return fmt.Errorf("状态转换失败: %w", err)
	}

	// 更新项目状态
	if project, err := o.store.GetProject(projectID); err == nil {
		project.Status = string(StateStopped)
		o.store.SaveProject(project)
	}

	o.logger.Info("停止成功", "project", projectID)
	return nil
}

// Restart 重启项目
func (o *Orchestrator) Restart(ctx context.Context, projectID string) error {
	o.logger.Info("重启项目", "project", projectID)

	if err := o.Stop(ctx, projectID); err != nil {
		return err
	}

	time.Sleep(1 * time.Second)

	return o.Start(ctx, projectID)
}

// Rollback 回滚项目
func (o *Orchestrator) Rollback(ctx context.Context, projectID string, version string) error {
	o.logger.Info("回滚项目", "project", projectID, "version", version)

	sm := o.getStateMachine(projectID)
	if sm == nil {
		return fmt.Errorf("项目未部署: %s", projectID)
	}

	// 状态转换: running -> rolling_back
	if _, err := sm.Next(EventRollback); err != nil {
		return fmt.Errorf("状态转换失败: %w", err)
	}

	// 1. 切换版本
	if err := o.store.SwitchVersion(projectID, version); err != nil {
		sm.Next(EventError)
		return fmt.Errorf("切换版本失败: %w", err)
	}

	// 2. 回滚执行器
	if err := o.executor.Rollback(ctx, projectID, version); err != nil {
		sm.Next(EventError)
		return fmt.Errorf("回滚失败: %w", err)
	}

	// 3. 更新项目信息
	if project, err := o.store.GetProject(projectID); err == nil {
		project.CurrentVersion = version
		project.Status = string(StateRollingBack)
		o.store.SaveProject(project)
	}

	// 4. 状态转换: rolling_back -> stopped
	if _, err := sm.Next(EventHealthOK); err != nil {
		return fmt.Errorf("状态转换失败: %w", err)
	}

	o.logger.Info("回滚成功", "project", projectID, "version", version)
	return nil
}

// GetProjectStatus 获取项目状态
func (o *Orchestrator) GetProjectStatus(ctx context.Context, projectID string) (*types.ProjectInfo, error) {
	project, err := o.store.GetProject(projectID)
	if err != nil {
		return nil, err
	}

	// 获取运行时状态
	runtimeStatus, err := o.executor.GetStatus(ctx, projectID)
	if err == nil {
		project.RuntimeStatus = runtimeStatus
	}

	return project, nil
}

// ImportIFP 导入 IFP 包
func (o *Orchestrator) ImportIFP(ctx context.Context, projectID string, filePath string) error {
	o.logger.Info("导入 IFP 包", "project", projectID, "path", filePath)

	// TODO: 实现 IFP 包解压、验证
	// 这里只是示例
	return fmt.Errorf("未实现")
}

// HealthCheck 健康检查
func (o *Orchestrator) HealthCheck(ctx context.Context) error {
	return o.executor.HealthCheck(ctx)
}

// 辅助函数
func (o *Orchestrator) getStateMachine(projectID string) *StateMachine {
	return o.stateMachines[projectID]
}
