package deployment

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/indu-forge/dev_core/internal/auth"
)

type dataAuthoringFence interface {
	AcquireAuthoringFence(context.Context, string, string, string, string, int) (string, time.Time, error)
	AcquireRestoreAuthoringFence(context.Context, string, string, string, string, int) (string, time.Time, string, error)
	RenewAuthoringFence(context.Context, string, string, string, string, int) (string, time.Time, error)
	ReleaseAuthoringFence(context.Context, string, string, string, string) error
}

type authoringWorkspaceFreezer interface {
	FreezeAuthoring(context.Context, auth.User, string) (func(context.Context) error, error)
	FreezeAuthoringRestore(context.Context, auth.User, string, *bool, func(bool) error) (func(context.Context) error, bool, error)
}

type PostgreSQLCaptureCoordinator struct {
	repository Repository
	data       dataAuthoringFence
	workspace  authoringWorkspaceFreezer
	ttl        time.Duration
}
type RestoreFenceTokens struct {
	Core, Data, DataEpoch string
	WorkspaceWasRunning   bool
	ResumeWorkspace       func(context.Context) error
}
type restoreIncompleteError struct{ err error }

func (e *restoreIncompleteError) Error() string { return e.err.Error() }
func (e *restoreIncompleteError) Unwrap() error { return e.err }

type restoreTerminalError struct{ err error }

func (e *restoreTerminalError) Error() string { return e.err.Error() }
func (e *restoreTerminalError) Unwrap() error { return e.err }

func shouldRetainRestoreFences(err error) bool {
	if err == nil {
		return false
	}
	_, terminal := err.(*restoreTerminalError)
	return !terminal
}

// WithRestoreFences 在整个跨域 Saga 期间保持双域租约与代码工作区冻结。
func (c *PostgreSQLCaptureCoordinator) WithRestoreFences(ctx context.Context, actor auth.User, project Project, ownerID string, knownWorkspaceState *bool, persistWorkspaceState func(bool) error, fn func(context.Context, RestoreFenceTokens) error) error {
	if c == nil || fn == nil {
		return fmt.Errorf("恢复协调器未配置")
	}
	retainFences := false
	core, err := c.repository.AcquireAuthoringFence(ctx, project.TenantID, project.ID, "restore", ownerID, c.ttl)
	if err != nil {
		return err
	}
	defer func() {
		if retainFences {
			return
		}
		releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = c.repository.ReleaseAuthoringFence(releaseCtx, project.TenantID, project.ID, ownerID, core)
	}()
	data, _, dataEpoch, err := c.data.AcquireRestoreAuthoringFence(ctx, project.ID, project.TenantID, ownerID, projectAuthoringEpoch(project), int(c.ttl.Seconds()))
	if err != nil {
		return err
	}
	defer func() {
		if retainFences {
			return
		}
		releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = c.data.ReleaseAuthoringFence(releaseCtx, project.ID, project.TenantID, ownerID, data)
	}()
	resume, wasRunning, err := c.workspace.FreezeAuthoringRestore(ctx, actor, project.ID, knownWorkspaceState, persistWorkspaceState)
	if err != nil {
		return err
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	renewErr := make(chan error, 1)
	done := make(chan struct{})
	defer close(done)
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-runCtx.Done():
				return
			case <-ticker.C:
				if e := c.repository.RenewAuthoringFence(runCtx, project.TenantID, project.ID, ownerID, core, c.ttl); e != nil {
					renewErr <- e
					cancel()
					return
				}
				if _, _, e := c.data.RenewAuthoringFence(runCtx, project.ID, project.TenantID, ownerID, data, int(c.ttl.Seconds())); e != nil {
					renewErr <- e
					cancel()
					return
				}
			}
		}
	}()
	err = fn(runCtx, RestoreFenceTokens{Core: core, Data: data, DataEpoch: dataEpoch, WorkspaceWasRunning: wasRunning, ResumeWorkspace: resume})
	// 只有任务已持久成功（nil）或已完整补偿失败（terminal）才能解除 marker。
	// 其他错误包括租约丢失、进程取消和补偿失败，均保留 marker 供同 owner 恢复。
	retainFences = shouldRetainRestoreFences(err)
	select {
	case leaseErr := <-renewErr:
		retainFences = true
		return fmt.Errorf("恢复写栅栏失效: %w", leaseErr)
	default:
		return err
	}
}

func NewAuthoringCaptureCoordinator(repository Repository, data dataAuthoringFence, workspace authoringWorkspaceFreezer) *PostgreSQLCaptureCoordinator {
	return &PostgreSQLCaptureCoordinator{repository: repository, data: data, workspace: workspace, ttl: 60 * time.Second}
}

// CleanupRestoreFences 用于 succeeded 后崩溃恢复：只回收持久 marker，不再次停止已恢复的工作区。
func (c *PostgreSQLCaptureCoordinator) CleanupRestoreFences(ctx context.Context, project Project, ownerID string) error {
	core, err := c.repository.AcquireAuthoringFence(ctx, project.TenantID, project.ID, "restore", ownerID, c.ttl)
	if err != nil {
		return err
	}
	data, _, _, err := c.data.AcquireRestoreAuthoringFence(ctx, project.ID, project.TenantID, ownerID, projectAuthoringEpoch(project), int(c.ttl.Seconds()))
	if err != nil {
		return err
	}
	if err = c.data.ReleaseAuthoringFence(ctx, project.ID, project.TenantID, ownerID, data); err != nil {
		return err
	}
	return c.repository.ReleaseAuthoringFence(ctx, project.TenantID, project.ID, ownerID, core)
}

func (c *PostgreSQLCaptureCoordinator) Capture(ctx context.Context, actor auth.User, project Project, ownerID string, capture func(context.Context) (CapturedAuthoring, error)) (CapturedAuthoring, error) {
	if c == nil || c.repository == nil || c.data == nil || c.workspace == nil || capture == nil {
		return CapturedAuthoring{}, fmt.Errorf("开发态捕获协调器未配置")
	}
	coreToken, err := c.repository.AcquireAuthoringFence(ctx, project.TenantID, project.ID, "build", ownerID, c.ttl)
	if err != nil {
		return CapturedAuthoring{}, err
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = c.repository.ReleaseAuthoringFence(releaseCtx, project.TenantID, project.ID, ownerID, coreToken)
	}()
	dataToken, _, err := c.data.AcquireAuthoringFence(ctx, project.ID, project.TenantID, ownerID, projectAuthoringEpoch(project), int(c.ttl.Seconds()))
	if err != nil {
		return CapturedAuthoring{}, err
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = c.data.ReleaseAuthoringFence(releaseCtx, project.ID, project.TenantID, ownerID, dataToken)
	}()
	resume, err := c.workspace.FreezeAuthoring(ctx, actor, project.ID)
	if err != nil {
		return CapturedAuthoring{}, err
	}
	resumed := false
	defer func() {
		if resumed {
			return
		}
		resumeCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = resume(resumeCtx)
	}()

	captureCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var renewalErr error
	var mu sync.Mutex
	done := make(chan struct{})
	defer close(done)
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-captureCtx.Done():
				return
			case <-ticker.C:
				if err := c.repository.RenewAuthoringFence(captureCtx, project.TenantID, project.ID, ownerID, coreToken, c.ttl); err != nil {
					mu.Lock()
					renewalErr = err
					mu.Unlock()
					cancel()
					return
				}
				if _, _, err := c.data.RenewAuthoringFence(captureCtx, project.ID, project.TenantID, ownerID, dataToken, int(c.ttl.Seconds())); err != nil {
					mu.Lock()
					renewalErr = err
					mu.Unlock()
					cancel()
					return
				}
			}
		}
	}()
	result, err := capture(captureCtx)
	cancel()
	mu.Lock()
	fenceErr := renewalErr
	mu.Unlock()
	if fenceErr != nil {
		return CapturedAuthoring{}, fmt.Errorf("开发态捕获期间写栅栏失效: %w", fenceErr)
	}
	if err != nil {
		return CapturedAuthoring{}, err
	}
	releaseCtx, releaseCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if releaseErr := c.data.ReleaseAuthoringFence(releaseCtx, project.ID, project.TenantID, ownerID, dataToken); releaseErr != nil {
		releaseCancel()
		return CapturedAuthoring{}, releaseErr
	}
	if releaseErr := c.repository.ReleaseAuthoringFence(releaseCtx, project.TenantID, project.ID, ownerID, coreToken); releaseErr != nil {
		releaseCancel()
		return CapturedAuthoring{}, releaseErr
	}
	releaseCancel()
	resumeCtx, resumeCancel := context.WithTimeout(context.Background(), 30*time.Second)
	resumeErr := resume(resumeCtx)
	resumeCancel()
	if resumeErr != nil {
		return CapturedAuthoring{}, fmt.Errorf("恢复代码工作区失败: %w", resumeErr)
	}
	resumed = true
	return result, nil
}

func projectAuthoringEpoch(item Project) string {
	if item.AuthoringEpoch < 1 {
		return "epoch-1"
	}
	return fmt.Sprintf("epoch-%d", item.AuthoringEpoch)
}
