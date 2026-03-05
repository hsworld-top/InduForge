package orchestrator

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/indu-forge/node_agent/internal/agent/store"
	"github.com/indu-forge/node_agent/internal/pkg/types"
)

// TestStateMachine_New 测试状态机创建
func TestStateMachine_New(t *testing.T) {
	sm := NewStateMachine()

	if sm == nil {
		t.Fatal("NewStateMachine 应返回非 nil")
	}

	if sm.Current() != StateStopped {
		t.Errorf("初始状态应为 %s，实际为 %s", StateStopped, sm.Current())
	}
}

// TestStateMachine_Current 测试获取当前状态
func TestStateMachine_Current(t *testing.T) {
	sm := NewStateMachine()

	state := sm.Current()

	if state != StateStopped {
		t.Errorf("期望 %s，实际 %s", StateStopped, state)
	}
}

// TestStateMachine_Transitions 测试有效状态转换
func TestStateMachine_Transitions(t *testing.T) {
	tests := []struct {
		name        string
		events      []Event
		expectState State
		expectError bool
	}{
		{
			name:        "部署事件: stopped -> deploying",
			events:      []Event{EventDeploy},
			expectState: StateDeploying,
			expectError: false,
		},
		{
			name:        "启动事件: stopped -> running",
			events:      []Event{EventStart},
			expectState: StateRunning,
			expectError: false,
		},
		{
			name:        "回滚事件: stopped -> stopped",
			events:      []Event{EventRollback},
			expectState: StateStopped,
			expectError: false,
		},
		{
			name:        "健康检查通过: deploying -> running",
			events:      []Event{EventDeploy, EventHealthOK},
			expectState: StateRunning,
			expectError: false,
		},
		{
			name:        "停止事件: running -> stopping",
			events:      []Event{EventStart, EventStop},
			expectState: StateStopping,
			expectError: false,
		},
		{
			name:        "停止完成: stopping -> stopped",
			events:      []Event{EventStart, EventStop, EventStop},
			expectState: StateStopped,
			expectError: false,
		},
		{
			name:        "重启事件: running -> running",
			events:      []Event{EventStart, EventRestart},
			expectState: StateRunning,
			expectError: false,
		},
		{
			name:        "回滚事件: running -> rolling_back",
			events:      []Event{EventStart, EventRollback},
			expectState: StateRollingBack,
			expectError: false,
		},
		{
			name:        "回滚完成: rolling_back -> stopped",
			events:      []Event{EventStart, EventRollback, EventHealthOK},
			expectState: StateStopped,
			expectError: false,
		},
		{
			name:        "错误处理: deploying -> error",
			events:      []Event{EventDeploy, EventError},
			expectState: StateError,
			expectError: false,
		},
		{
			name:        "从错误状态回滚: error -> stopped",
			events:      []Event{EventDeploy, EventError, EventRollback},
			expectState: StateStopped,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewStateMachine()

			var err error
			for _, event := range tt.events {
				_, err = sm.Next(event)
				if tt.expectError && err == nil {
					t.Errorf("事件 %s 应返回错误", event)
					return
				}
				if !tt.expectError && err != nil {
					t.Errorf("事件 %s 不应返回错误，实际: %v", event, err)
					return
				}
			}

			if sm.Current() != tt.expectState {
				t.Errorf("期望状态 %s，实际状态 %s", tt.expectState, sm.Current())
			}
		})
	}
}

// TestStateMachine_InvalidTransitions 测试无效状态转换
func TestStateMachine_InvalidTransitions(t *testing.T) {
	tests := []struct {
		name         string
		initState    State
		initEvents   []Event
		invalidEvent Event
	}{
		{
			name:         "从 stopped 状态无法健康检查",
			initState:    StateStopped,
			initEvents:   []Event{},
			invalidEvent: EventHealthOK,
		},
		{
			name:         "从 stopped 状态无法重启",
			initState:    StateStopped,
			initEvents:   []Event{},
			invalidEvent: EventRestart,
		},
		{
			name:         "从 error 状态无法启动",
			initState:    StateStopped,
			initEvents:   []Event{EventDeploy, EventError},
			invalidEvent: EventStart,
		},
		{
			name:         "从 error 状态无法部署",
			initState:    StateStopped,
			initEvents:   []Event{EventDeploy, EventError},
			invalidEvent: EventDeploy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewStateMachine()

			// 设置初始状态
			sm.current = tt.initState

			// 执行初始化事件
			for _, event := range tt.initEvents {
				sm.Next(event)
			}

			// 测试无效事件
			_, err := sm.Next(tt.invalidEvent)

			if err == nil {
				t.Errorf("事件 %s 应返回错误", tt.invalidEvent)
			}
		})
	}
}

// TestStateMachine_AllStates 测试所有状态定义
func TestStateMachine_AllStates(t *testing.T) {
	states := []State{
		StateDeploying,
		StateRunning,
		StateStopping,
		StateStopped,
		StateError,
		StateRollingBack,
	}

	for _, state := range states {
		if state == "" {
			t.Errorf("状态不应为空")
		}
	}
}

// TestStateMachine_AllEvents 测试所有事件定义
func TestStateMachine_AllEvents(t *testing.T) {
	events := []Event{
		EventDeploy,
		EventStart,
		EventStop,
		EventRestart,
		EventRollback,
		EventHealthOK,
		EventHealthFail,
		EventError,
	}

	for _, event := range events {
		if event == "" {
			t.Errorf("事件不应为空")
		}
	}
}

// TestOrchestrator_New 测试编排器创建
func TestOrchestrator_New(t *testing.T) {
	exec := &mockExecutorImpl{}
	tempDir := t.TempDir()
	realStore := store.NewLocalStore(filepath.Join(tempDir, "data"))
	t.Cleanup(func() { _ = realStore.Close() })

	orch := NewOrchestrator(exec, realStore)

	if orch == nil {
		t.Fatal("NewOrchestrator 应返回非 nil")
	}

	if orch.executor == nil {
		t.Error("executor 不应为 nil")
	}

	if orch.store == nil {
		t.Error("store 不应为 nil")
	}

	if orch.stateMachines == nil {
		t.Error("stateMachines 不应为 nil")
	}
}

// TestOrchestrator_GetStateMachine 测试获取状态机
func TestOrchestrator_GetStateMachine(t *testing.T) {
	exec := &mockExecutorImpl{}
	tempDir := t.TempDir()
	realStore := store.NewLocalStore(filepath.Join(tempDir, "data"))
	t.Cleanup(func() { _ = realStore.Close() })

	orch := NewOrchestrator(exec, realStore)

	// 初始状态机不存在
	sm := orch.getStateMachine("project-001")
	if sm != nil {
		t.Error("初始状态机应为 nil")
	}

	// 手动添加状态机
	orch.stateMachines["project-001"] = NewStateMachine()

	// 现在应该能获取到
	sm = orch.getStateMachine("project-001")
	if sm == nil {
		t.Error("状态机不应为 nil")
	}
}

// mockExecutorImpl 用于测试的模拟执行器
type mockExecutorImpl struct{}

func (m *mockExecutorImpl) Deploy(ctx context.Context, req types.DeployRequest) error {
	return nil
}

func (m *mockExecutorImpl) Start(ctx context.Context, projectID string) error {
	return nil
}

func (m *mockExecutorImpl) Stop(ctx context.Context, projectID string) error {
	return nil
}

func (m *mockExecutorImpl) Restart(ctx context.Context, projectID string) error {
	return nil
}

func (m *mockExecutorImpl) Rollback(ctx context.Context, projectID string, version string) error {
	return nil
}

func (m *mockExecutorImpl) GetStatus(ctx context.Context, projectID string) (*types.RuntimeStatus, error) {
	return &types.RuntimeStatus{
		ProjectID: projectID,
		State:     "running",
		Health:    "healthy",
	}, nil
}

func (m *mockExecutorImpl) HealthCheck(ctx context.Context) error {
	return nil
}

// mockStoreImpl 用于测试的模拟存储
type mockStoreImpl struct{}

func (m *mockStoreImpl) SaveProject(project *types.ProjectInfo) error {
	return nil
}

func (m *mockStoreImpl) GetProject(projectID string) (*types.ProjectInfo, error) {
	return &types.ProjectInfo{
		ID:   projectID,
		Name: projectID,
	}, nil
}

func (m *mockStoreImpl) ListProjects() ([]*types.ProjectInfo, error) {
	return nil, nil
}

func (m *mockStoreImpl) DeleteProject(projectID string) error {
	return nil
}

func (m *mockStoreImpl) SaveConnectionProfile(projectID string, profile types.ConnectionProfile) error {
	return nil
}

func (m *mockStoreImpl) GetConnectionProfile(projectID string) (*types.ConnectionProfile, error) {
	return nil, nil
}

func (m *mockStoreImpl) SaveVersion(projectID string, version string, files map[string][]byte) error {
	return nil
}

func (m *mockStoreImpl) GetVersionDir(projectID string, version string) (string, error) {
	return "", nil
}

func (m *mockStoreImpl) ListVersions(projectID string) ([]string, error) {
	return nil, nil
}

func (m *mockStoreImpl) SwitchVersion(projectID string, version string) error {
	return nil
}

func (m *mockStoreImpl) GetCurrentVersion(projectID string) (string, error) {
	return "", nil
}

// TestOrchestrator_HealthCheck 测试编排器健康检查
func TestOrchestrator_HealthCheck(t *testing.T) {
	exec := &mockExecutorImpl{}
	tempDir := t.TempDir()
	realStore := store.NewLocalStore(filepath.Join(tempDir, "data"))
	t.Cleanup(func() { _ = realStore.Close() })

	orch := NewOrchestrator(exec, realStore)

	err := orch.HealthCheck(context.Background())

	if err != nil {
		t.Errorf("HealthCheck 不应返回错误，实际: %v", err)
	}
}

// mockExecutorWithError 带错误的模拟执行器
type mockExecutorWithError struct {
	mockExecutorImpl
	returnError bool
}

func (m *mockExecutorWithError) HealthCheck(ctx context.Context) error {
	if m.returnError {
		return &testError{"健康检查失败"}
	}
	return nil
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// TestOrchestrator_HealthCheckError 测试编排器健康检查错误
func TestOrchestrator_HealthCheckError(t *testing.T) {
	exec := &mockExecutorWithError{returnError: true}
	tempDir := t.TempDir()
	realStore := store.NewLocalStore(filepath.Join(tempDir, "data"))
	t.Cleanup(func() { _ = realStore.Close() })

	orch := NewOrchestrator(exec, realStore)

	err := orch.HealthCheck(context.Background())

	if err == nil {
		t.Error("HealthCheck 应返回错误")
	}
}

// TestStateMachine_MultipleTransitions 测试连续状态转换
func TestStateMachine_MultipleTransitions(t *testing.T) {
	// 部署流程
	tests := []struct {
		events     []Event
		finalState State
	}{
		{[]Event{EventDeploy}, StateDeploying},
		{[]Event{EventDeploy, EventHealthOK}, StateRunning},
		{[]Event{EventDeploy, EventHealthOK, EventStop}, StateStopping},
		{[]Event{EventDeploy, EventHealthOK, EventStop, EventStop}, StateStopped},
	}

	for _, tt := range tests {
		sm := NewStateMachine()

		for _, event := range tt.events {
			sm.Next(event)
		}

		if sm.Current() != tt.finalState {
			t.Errorf("事件序列 %v 期望状态 %s，实际 %s", tt.events, tt.finalState, sm.Current())
		}
	}
}

// TestStateMachine_ErrorRecovery 测试错误恢复
func TestStateMachine_ErrorRecovery(t *testing.T) {
	sm := NewStateMachine()

	// 部署 -> 错误
	sm.Next(EventDeploy)
	sm.Next(EventError)

	if sm.Current() != StateError {
		t.Errorf("期望错误状态，实际 %s", sm.Current())
	}

	// 从错误状态回滚
	_, err := sm.Next(EventRollback)
	if err != nil {
		t.Errorf("回滚不应返回错误，实际: %v", err)
	}

	if sm.Current() != StateStopped {
		t.Errorf("期望停止状态，实际 %s", sm.Current())
	}
}

// TestStateMachine_RollbackFlow 测试回滚流程
func TestStateMachine_RollbackFlow(t *testing.T) {
	sm := NewStateMachine()

	// 运行 -> 回滚中
	sm.Next(EventStart)
	sm.Next(EventRollback)

	if sm.Current() != StateRollingBack {
		t.Errorf("期望回滚状态，实际 %s", sm.Current())
	}

	// 回滚完成
	sm.Next(EventHealthOK)

	if sm.Current() != StateStopped {
		t.Errorf("期望停止状态，实际 %s", sm.Current())
	}
}
