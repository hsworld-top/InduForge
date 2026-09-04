package deployment

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/indu-forge/dev_core/internal/auth"
)

var ErrAuthoringRestoreUnavailable = errors.New("开发态恢复尚未启用：服务端编辑代次保护未就绪")

type RestoreScheduler interface{ Enqueue(RestoreTask) }

type RestoreTask struct {
	ID, TenantID, ProjectID, VersionID, RequestedBy                                           string
	CurrentProjectRevision, State, Stage, ErrorMessage                                        string
	BackupBucket, BackupKey, BackupHash, BackupCipherHash, BackupKeyID, BackupProjectRevision string
	BackupSize                                                                                int64
	ExpectedEpoch, TargetEpoch                                                                int64
	RolledBack                                                                                bool
	WorkspaceWasRunning                                                                       *bool
	StartedAt, CompletedAt                                                                    *time.Time
	CreatedAt, UpdatedAt                                                                      time.Time
}

// RestoreDevelopment 故意失败关闭。只有三域编辑代次均由服务端强制校验后，
// 才能注入 durable Saga executor 并真正创建任务。
func (s *Service) RestoreDevelopment(ctx context.Context, actor auth.User, versionID, confirmation string) (RestoreTask, error) {
	if confirmation != "RESTORE" {
		return RestoreTask{}, errors.New("恢复确认无效")
	}
	if err := auth.RequireCapability(actor, auth.CapabilityProjectWrite); err != nil {
		return RestoreTask{}, err
	}
	if err := auth.RequireCapability(actor, auth.CapabilityReleasePublish); err != nil {
		return RestoreTask{}, err
	}
	if _, err := uuid.Parse(versionID); err != nil {
		return RestoreTask{}, errors.New("versionId 无效")
	}
	if s.restoreScheduler == nil {
		return RestoreTask{}, ErrAuthoringRestoreUnavailable
	}
	version, err := s.repository.GetVersion(ctx, actor.TenantID, versionID)
	if err != nil {
		return RestoreTask{}, err
	}
	project, err := s.repository.GetProject(ctx, actor.TenantID, version.ProjectID)
	if err != nil {
		return RestoreTask{}, err
	}
	if err = requireProject(actor, project, auth.CapabilityProjectWrite); err != nil {
		return RestoreTask{}, err
	}
	if !version.Restorable || version.Status != "ready" {
		return RestoreTask{}, errors.New("该版本没有可恢复的完整开发态快照")
	}
	task, err := s.repository.CreateRestoreTask(ctx, actor.TenantID, versionID, actor.ID)
	if err != nil {
		return RestoreTask{}, err
	}
	s.restoreScheduler.Enqueue(task)
	return task, nil
}

func (s *Service) GetRestoreTask(ctx context.Context, actor auth.User, taskID string) (RestoreTask, error) {
	if _, err := uuid.Parse(taskID); err != nil {
		return RestoreTask{}, errors.New("taskId 无效")
	}
	if err := auth.RequireCapability(actor, auth.CapabilityProjectRead); err != nil {
		return RestoreTask{}, err
	}
	task, err := s.repository.GetRestoreTask(ctx, actor.TenantID, taskID)
	if err != nil {
		return RestoreTask{}, err
	}
	project, err := s.repository.GetProject(ctx, actor.TenantID, task.ProjectID)
	if err != nil {
		return RestoreTask{}, err
	}
	if err = requireProject(actor, project, auth.CapabilityProjectRead); err != nil {
		return RestoreTask{}, err
	}
	return task, nil
}
