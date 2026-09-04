package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/authoringsnapshot"
	"github.com/indu-forge/dev_core/internal/sceneasset"
	"github.com/jackc/pgx/v5"
)

type restoreRepository interface {
	GetRestoreTask(context.Context, string, string) (RestoreTask, error)
	ListRecoverableRestoreTasks(context.Context) ([]RestoreTask, error)
	SetRestoreStage(context.Context, string, string, string) error
	SetRestoreBackup(context.Context, string, AuthoringSnapshotMetadata) error
	SetRestoreWorkspaceWasRunning(context.Context, string, bool) error
	FailRestoreTask(context.Context, string, string, bool) error
	FinalizeRestoreCore(context.Context, RestoreTask, applyRestoreCore) error
	CompensateRestoreCore(context.Context, RestoreTask, applyRestoreCore) error
	CompleteRestore(context.Context, string) error
	MarkRestoreCleanupComplete(context.Context, string) error
	GetProject(context.Context, string, string) (Project, error)
	GetVersion(context.Context, string, string) (Version, error)
}
type restoreWorkspace interface {
	StageRestoreWithModes(string, string, map[string]string, map[string]uint32) error
	ActivateRestore(string, string) error
	RollbackRestore(string, string) error
	FinalizeRestore(string, string) error
}
type restoreScenes interface {
	PrepareAuthoringRestore(context.Context, auth.User, string, sceneasset.AuthoringSnapshotOptions, sceneasset.AuthoringSnapshot) (*sceneasset.PreparedAuthoringRestore, error)
	ApplyPreparedAuthoringRestore(context.Context, pgx.Tx, auth.User, string, string, *sceneasset.PreparedAuthoringRestore) error
}
type restoreData interface {
	ReplaceAuthoringSnapshotFenced(context.Context, string, string, string, string, string, string, string, string, json.RawMessage) error
	GetAuthoringEpoch(context.Context, string, string) (string, error)
}
type restoreAuthoring interface {
	Capture(context.Context, auth.User, Project) (CapturedAuthoring, error)
	PersistBackup(context.Context, Project, string, CapturedAuthoring) (AuthoringSnapshotMetadata, error)
	OpenStored(context.Context, Project, AuthoringSnapshotMetadata) (authoringsnapshot.Snapshot, error)
	DeleteSnapshot(context.Context, string) error
}
type restoreFenceCoordinator interface {
	WithRestoreFences(context.Context, auth.User, Project, string, *bool, func(bool) error, func(context.Context, RestoreFenceTokens) error) error
	CleanupRestoreFences(context.Context, Project, string) error
}

type RestoreExecutor struct {
	repo      restoreRepository
	authoring restoreAuthoring
	workspace restoreWorkspace
	scenes    restoreScenes
	data      restoreData
	fences    restoreFenceCoordinator
	logger    *slog.Logger
	ctx       context.Context
	mu        sync.Mutex
	running   map[string]struct{}
}

func NewRestoreExecutor(repo restoreRepository, authoring restoreAuthoring, workspace restoreWorkspace, scenes restoreScenes, data restoreData, fences restoreFenceCoordinator, logger *slog.Logger) *RestoreExecutor {
	if logger == nil {
		logger = slog.Default()
	}
	return &RestoreExecutor{repo: repo, authoring: authoring, workspace: workspace, scenes: scenes, data: data, fences: fences, logger: logger, ctx: context.Background(), running: map[string]struct{}{}}
}
func (e *RestoreExecutor) Start(ctx context.Context) { e.ctx = ctx; go e.recover(ctx) }
func (e *RestoreExecutor) recover(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		items, err := e.repo.ListRecoverableRestoreTasks(ctx)
		if err == nil {
			for _, item := range items {
				e.Enqueue(item)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (e *RestoreExecutor) Enqueue(task RestoreTask) {
	e.mu.Lock()
	if _, ok := e.running[task.ID]; ok {
		e.mu.Unlock()
		return
	}
	e.running[task.ID] = struct{}{}
	e.mu.Unlock()
	go func() {
		defer func() { e.mu.Lock(); delete(e.running, task.ID); e.mu.Unlock() }()
		if err := e.process(e.ctx, task); err != nil {
			e.logger.Error("恢复工程开发态失败", "taskId", task.ID, "error", releaseDiagnosticError(err))
		}
	}()
}

func (e *RestoreExecutor) process(ctx context.Context, task RestoreTask) error {
	current, err := e.repo.GetRestoreTask(ctx, task.TenantID, task.ID)
	if err != nil {
		return err
	}
	task = current
	project, err := e.repo.GetProject(ctx, task.TenantID, task.ProjectID)
	if err != nil {
		return err
	}
	if task.State == "failed" {
		return nil
	}
	if task.State == "succeeded" {
		if err = e.fences.CleanupRestoreFences(ctx, project, task.ID); err != nil {
			return err
		}
		if err = e.workspace.FinalizeRestore(project.ID, task.ID); err != nil {
			return err
		}
		return e.repo.MarkRestoreCleanupComplete(ctx, task.ID)
	}
	var target authoringsnapshot.Snapshot
	var targetErr error
	if task.State != "compensating" && task.State != "finalizing" && task.State != "succeeded" {
		version, versionErr := e.repo.GetVersion(ctx, task.TenantID, task.VersionID)
		if versionErr != nil {
			targetErr = versionErr
			if task.BackupKey == "" {
				return e.failBeforeFence(ctx, task, versionErr)
			}
		} else {
			// 目标快照在加双域 fence 前完成解密和完整性校验，缩短禁止编辑的时间。
			target, targetErr = e.authoring.OpenStored(ctx, project, AuthoringSnapshotMetadata{Bucket: version.AuthoringSnapshotBucket, Key: version.AuthoringSnapshotKey, ContentHash: version.AuthoringSnapshotHash, CipherHash: version.AuthoringSnapshotCipherHash, Size: version.AuthoringSnapshotSize, KeyID: version.AuthoringSnapshotKeyID, ProjectRevision: version.AuthoringProjectRevision})
			if targetErr != nil && task.BackupKey == "" {
				return e.failBeforeFence(ctx, task, targetErr)
			}
		}
	}
	actor := auth.User{ID: task.RequestedBy, TenantID: task.TenantID, Role: "TENANT_ADMIN"}
	return e.fences.WithRestoreFences(ctx, actor, project, task.ID, task.WorkspaceWasRunning, func(running bool) error {
		return e.repo.SetRestoreWorkspaceWasRunning(ctx, task.ID, running)
	}, func(runCtx context.Context, tokens RestoreFenceTokens) error {
		var err error
		task, err = e.repo.GetRestoreTask(runCtx, task.TenantID, task.ID)
		if err != nil {
			return err
		}
		if task.State == "compensating" {
			return e.compensate(runCtx, actor, project, task, tokens, fmt.Errorf("继续未完成的安全回滚"))
		}
		if task.State == "finalizing" {
			return e.finishWorkspace(runCtx, project, task, tokens)
		}
		if targetErr != nil {
			return e.compensate(runCtx, actor, project, task, tokens, targetErr)
		}
		if task.BackupKey == "" {
			captured, captureErr := e.authoring.Capture(runCtx, actor, project)
			if captureErr != nil {
				return captureErr
			}
			backup, persistErr := e.authoring.PersistBackup(runCtx, project, task.ID, captured)
			if persistErr != nil {
				return persistErr
			}
			if err = e.repo.SetRestoreBackup(runCtx, task.ID, backup); err != nil {
				// backup key 由 taskID+本次内容摘要独占；DB 未引用时可安全删除。
				_ = e.authoring.DeleteSnapshot(runCtx, backup.Key)
				return err
			}
			task, err = e.repo.GetRestoreTask(runCtx, task.TenantID, task.ID)
			if err != nil {
				return err
			}
		}
		if err = e.repo.SetRestoreStage(runCtx, task.ID, "restoring_workspace", "workspace_staging"); err != nil {
			return err
		}
		if err = e.workspace.StageRestoreWithModes(project.ID, task.ID, target.Workspace, target.WorkspaceModes); err != nil {
			return e.compensate(runCtx, actor, project, task, tokens, err)
		}
		var scenes sceneasset.AuthoringSnapshot
		if err = json.Unmarshal(target.Scenes, &scenes); err != nil {
			return e.compensate(runCtx, actor, project, task, tokens, err)
		}
		if err = e.repo.SetRestoreStage(runCtx, task.ID, "restoring_scenes", "scenes_prepared"); err != nil {
			return err
		}
		sceneCtx := sceneasset.WithRestoreFence(runCtx, tokens.Core)
		prepared, err := e.scenes.PrepareAuthoringRestore(sceneCtx, actor, project.ID, sceneasset.AuthoringSnapshotOptions{ExpectedProjectID: project.ID, Fence: tokens.Core}, scenes)
		if err != nil {
			return e.compensate(runCtx, actor, project, task, tokens, err)
		}
		if err = e.repo.SetRestoreStage(runCtx, task.ID, "restoring_data", "data"); err != nil {
			return err
		}
		dataEpoch := tokens.DataEpoch
		expected, targetEpoch := fmt.Sprintf("epoch-%d", task.ExpectedEpoch), fmt.Sprintf("epoch-%d", task.TargetEpoch)
		if dataEpoch == expected {
			if err = e.data.ReplaceAuthoringSnapshotFenced(runCtx, project.ID, project.TenantID, actor.ID, task.ID, tokens.Data, expected, targetEpoch, "forward", target.Data); err != nil {
				return e.compensate(runCtx, actor, project, task, tokens, err)
			}
		} else if dataEpoch != targetEpoch {
			return e.compensate(runCtx, actor, project, task, tokens, fmt.Errorf("数据编辑代次与恢复任务不一致"))
		}
		projectNow, err := e.repo.GetProject(runCtx, task.TenantID, task.ProjectID)
		if err != nil {
			return e.compensate(runCtx, actor, project, task, tokens, err)
		}
		if projectNow.AuthoringEpoch == task.ExpectedEpoch {
			if err = e.repo.FinalizeRestoreCore(runCtx, task, func(txCtx context.Context, tx pgx.Tx) error {
				return e.scenes.ApplyPreparedAuthoringRestore(sceneasset.WithRestoreFence(txCtx, tokens.Core), tx, actor, project.ID, tokens.Core, prepared)
			}); err != nil {
				return e.compensate(runCtx, actor, project, task, tokens, err)
			}
		} else if projectNow.AuthoringEpoch != task.TargetEpoch {
			return e.compensate(runCtx, actor, project, task, tokens, fmt.Errorf("工程编辑代次与恢复任务不一致"))
		}
		return e.finishWorkspace(runCtx, project, task, tokens)
	})
}

func (e *RestoreExecutor) finishWorkspace(ctx context.Context, project Project, task RestoreTask, tokens RestoreFenceTokens) error {
	if err := e.workspace.ActivateRestore(project.ID, task.ID); err != nil {
		return err
	}
	if tokens.ResumeWorkspace != nil {
		if err := tokens.ResumeWorkspace(ctx); err != nil {
			return fmt.Errorf("恢复代码工作区失败: %w", err)
		}
	}
	// succeeded 必须先于清理 backup/staging 持久化；否则清理后崩溃无法判断 workspace 是否已激活。
	if err := e.repo.CompleteRestore(ctx, task.ID); err != nil {
		return err
	}
	if err := e.workspace.FinalizeRestore(project.ID, task.ID); err != nil {
		e.logger.Warn("清理工程恢复临时目录失败", "taskId", task.ID, "error", releaseDiagnosticError(err))
		return nil
	}
	return e.repo.MarkRestoreCleanupComplete(ctx, task.ID)
}

func (e *RestoreExecutor) failBeforeFence(ctx context.Context, task RestoreTask, cause error) error {
	if task.BackupKey == "" {
		_ = e.repo.FailRestoreTask(ctx, task.ID, releaseDiagnosticError(cause), false)
	}
	return &restoreTerminalError{err: cause}
}

func (e *RestoreExecutor) compensate(ctx context.Context, actor auth.User, project Project, task RestoreTask, tokens RestoreFenceTokens, cause error) error {
	_ = e.repo.SetRestoreStage(ctx, task.ID, "compensating", "rollback")
	backup, openErr := e.authoring.OpenStored(ctx, project, AuthoringSnapshotMetadata{Bucket: task.BackupBucket, Key: task.BackupKey, ContentHash: task.BackupHash, CipherHash: task.BackupCipherHash, Size: task.BackupSize, KeyID: task.BackupKeyID, ProjectRevision: task.BackupProjectRevision})
	if openErr != nil {
		return &restoreIncompleteError{err: fmt.Errorf("%v；备份不可读: %w", cause, openErr)}
	}
	var scenes sceneasset.AuthoringSnapshot
	if err := json.Unmarshal(backup.Scenes, &scenes); err != nil {
		return &restoreIncompleteError{err: err}
	}
	sceneCtx := sceneasset.WithRestoreFence(ctx, tokens.Core)
	prepared, err := e.scenes.PrepareAuthoringRestore(sceneCtx, actor, project.ID, sceneasset.AuthoringSnapshotOptions{ExpectedProjectID: project.ID, Fence: tokens.Core}, scenes)
	if err != nil {
		return &restoreIncompleteError{err: err}
	}
	epoch, epochErr := e.data.GetAuthoringEpoch(ctx, project.ID, project.TenantID)
	if epochErr != nil {
		return &restoreIncompleteError{err: epochErr}
	}
	if epoch == fmt.Sprintf("epoch-%d", task.TargetEpoch) {
		if err := e.data.ReplaceAuthoringSnapshotFenced(ctx, project.ID, project.TenantID, actor.ID, task.ID, tokens.Data, epoch, fmt.Sprintf("epoch-%d", task.ExpectedEpoch), "compensate", backup.Data); err != nil {
			return &restoreIncompleteError{err: err}
		}
	} else if epoch != fmt.Sprintf("epoch-%d", task.ExpectedEpoch) {
		return &restoreIncompleteError{err: fmt.Errorf("数据编辑代次无法安全补偿")}
	}
	if err := e.repo.CompensateRestoreCore(ctx, task, func(txCtx context.Context, tx pgx.Tx) error {
		return e.scenes.ApplyPreparedAuthoringRestore(sceneasset.WithRestoreFence(txCtx, tokens.Core), tx, actor, project.ID, tokens.Core, prepared)
	}); err != nil {
		return &restoreIncompleteError{err: err}
	}
	if err := e.workspace.RollbackRestore(project.ID, task.ID); err != nil {
		return &restoreIncompleteError{err: err}
	}
	if tokens.ResumeWorkspace != nil {
		if err := tokens.ResumeWorkspace(ctx); err != nil {
			return &restoreIncompleteError{err: fmt.Errorf("恢复代码工作区失败: %w", err)}
		}
	}
	if err := e.repo.FailRestoreTask(ctx, task.ID, releaseDiagnosticError(cause), true); err != nil {
		return &restoreIncompleteError{err: err}
	}
	return &restoreTerminalError{err: cause}
}
