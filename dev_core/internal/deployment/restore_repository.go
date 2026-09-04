package deployment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type applyRestoreCore func(context.Context, pgx.Tx) error

func (r *PostgreSQLRepository) CreateRestoreTask(ctx context.Context, tenantID, versionID, userID string) (RestoreTask, error) {
	if _, err := uuid.Parse(tenantID); err != nil {
		return RestoreTask{}, ErrNotFound
	}
	if _, err := uuid.Parse(versionID); err != nil {
		return RestoreTask{}, ErrNotFound
	}
	if _, err := uuid.Parse(userID); err != nil {
		return RestoreTask{}, ErrNotFound
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return RestoreTask{}, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended((SELECT project_id::text FROM application_versions WHERE id=$1 AND tenant_id=$2),0))`, versionID, tenantID); err != nil {
		return RestoreTask{}, err
	}
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO authoring_restore_tasks(tenant_id,project_id,application_version_id,requested_by,current_project_revision,expected_authoring_epoch,target_authoring_epoch) SELECT v.tenant_id,v.project_id,v.id,$3,'epoch-'||p.authoring_epoch,p.authoring_epoch,p.authoring_epoch+1 FROM application_versions v JOIN projects p ON p.id=v.project_id AND p.tenant_id=v.tenant_id WHERE v.id=$1 AND v.tenant_id=$2 AND v.status='ready' AND v.restorable AND NOT EXISTS(SELECT 1 FROM authoring_project_fences f WHERE f.project_id=p.id AND f.expires_at>now()) AND NOT EXISTS(SELECT 1 FROM application_versions building WHERE building.project_id=p.id AND building.status='building' AND building.created_at>=now()-$4::interval) RETURNING id`, versionID, tenantID, userID, fmt.Sprintf("%d seconds", int(staleProductionBuildTimeout.Seconds()))).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return RestoreTask{}, ErrAuthoringBusy
		}
		return RestoreTask{}, mapConstraint(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO authoring_restore_task_events(tenant_id,task_id,stage,message) VALUES($1,$2,'queued','已受理工程开发态恢复请求')`, tenantID, id); err != nil {
		return RestoreTask{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return RestoreTask{}, err
	}
	return r.GetRestoreTask(ctx, tenantID, id)
}

func (r *PostgreSQLRepository) ListRecoverableRestoreTasks(ctx context.Context) ([]RestoreTask, error) {
	rows, err := r.pool.Query(ctx, `SELECT tenant_id,id FROM authoring_restore_tasks WHERE state IN ('queued','staging','restoring_workspace','restoring_scenes','restoring_data','finalizing','compensating') OR (state='succeeded' AND cleanup_completed_at IS NULL) ORDER BY created_at,id LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []RestoreTask{}
	for rows.Next() {
		var tenant, id string
		if err = rows.Scan(&tenant, &id); err != nil {
			return nil, err
		}
		item, getErr := r.GetRestoreTask(ctx, tenant, id)
		if getErr != nil {
			return nil, getErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (r *PostgreSQLRepository) SetRestoreStage(ctx context.Context, taskID, state, stage string) error {
	command, err := r.pool.Exec(ctx, `WITH changed AS (UPDATE authoring_restore_tasks SET state=$2,stage=$3,started_at=COALESCE(started_at,now()),updated_at=now() WHERE id=$1 AND state NOT IN ('succeeded','failed') RETURNING id,tenant_id) INSERT INTO authoring_restore_task_events(tenant_id,task_id,stage,message) SELECT tenant_id,id,$3,'恢复任务阶段已推进' FROM changed`, taskID, state, stage)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}
func (r *PostgreSQLRepository) SetRestoreBackup(ctx context.Context, taskID string, m AuthoringSnapshotMetadata) error {
	_, err := r.pool.Exec(ctx, `WITH changed AS (UPDATE authoring_restore_tasks SET backup_schema='authoring-snapshot.v1',backup_bucket=$2,backup_key=$3,backup_hash=$4,backup_cipher_hash=$5,backup_size=$6,backup_key_id=$7,backup_project_revision=$8,state='staging',stage='backup_ready',updated_at=now() WHERE id=$1 AND state IN ('queued','staging') RETURNING id,tenant_id) INSERT INTO authoring_restore_task_events(tenant_id,task_id,stage,message) SELECT tenant_id,id,'backup_ready','恢复前完整备份已持久化' FROM changed`, taskID, m.Bucket, m.Key, m.ContentHash, m.CipherHash, m.Size, m.KeyID, m.ProjectRevision)
	return err
}
func (r *PostgreSQLRepository) SetRestoreWorkspaceWasRunning(ctx context.Context, taskID string, running bool) error {
	command, err := r.pool.Exec(ctx, `UPDATE authoring_restore_tasks SET workspace_was_running=$2,updated_at=now() WHERE id=$1 AND workspace_was_running IS NULL AND state NOT IN ('succeeded','failed')`, taskID, running)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("恢复任务工作区冻结状态无法持久化")
	}
	return nil
}
func (r *PostgreSQLRepository) FailRestoreTask(ctx context.Context, taskID, message string, rolledBack bool) error {
	_, err := r.pool.Exec(ctx, `WITH changed AS (UPDATE authoring_restore_tasks SET state='failed',stage=CASE WHEN $3 THEN 'rolled_back' ELSE 'failed' END,error_message=$2,rolled_back=$3,completed_at=now(),updated_at=now() WHERE id=$1 RETURNING id,tenant_id,stage) INSERT INTO authoring_restore_task_events(tenant_id,task_id,stage,message) SELECT tenant_id,id,stage,CASE WHEN $3 THEN '恢复失败，已完整回滚' ELSE '恢复失败' END FROM changed`, taskID, message, rolledBack)
	return err
}

// FinalizeRestoreCore 在同一事务中替换场景、推进工程代次并把任务置为 finalizing。
// workspace 尚未激活，因此此处不能把任务标记为 succeeded。
func (r *PostgreSQLRepository) FinalizeRestoreCore(ctx context.Context, task RestoreTask, apply applyRestoreCore) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, task.ProjectID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `SET LOCAL induforge.authoring_restore='on'`); err != nil {
		return err
	}
	if apply == nil {
		return fmt.Errorf("场景恢复事务未配置")
	}
	if err = apply(ctx, tx); err != nil {
		return err
	}
	command, err := tx.Exec(ctx, `UPDATE projects SET authoring_epoch=$1,updated_at=now() WHERE id=$2 AND tenant_id=$3 AND authoring_epoch=$4`, task.TargetEpoch, task.ProjectID, task.TenantID, task.ExpectedEpoch)
	if err != nil || command.RowsAffected() != 1 {
		if err == nil {
			err = fmt.Errorf("工程编辑代次已变化")
		}
		return err
	}
	command, err = tx.Exec(ctx, `WITH changed AS (UPDATE authoring_restore_tasks SET state='finalizing',stage='core_committed',updated_at=now() WHERE id=$1 AND state IN ('restoring_scenes','restoring_data','finalizing') RETURNING id,tenant_id) INSERT INTO authoring_restore_task_events(tenant_id,task_id,stage,message) SELECT tenant_id,id,'core_committed','场景与工程编辑代次已原子切换' FROM changed`, task.ID)
	if err != nil || command.RowsAffected() != 1 {
		if err == nil {
			err = ErrNotFound
		}
		return err
	}
	return tx.Commit(ctx)
}

func (r *PostgreSQLRepository) CompleteRestore(ctx context.Context, taskID string) error {
	command, err := r.pool.Exec(ctx, `WITH changed AS (UPDATE authoring_restore_tasks SET state='succeeded',stage='completed',completed_at=now(),updated_at=now() WHERE id=$1 AND state='finalizing' RETURNING id,tenant_id) INSERT INTO authoring_restore_task_events(tenant_id,task_id,stage,message) SELECT tenant_id,id,'completed','工程开发态恢复完成' FROM changed`, taskID)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgreSQLRepository) MarkRestoreCleanupComplete(ctx context.Context, taskID string) error {
	command, err := r.pool.Exec(ctx, `UPDATE authoring_restore_tasks SET cleanup_completed_at=COALESCE(cleanup_completed_at,now()),updated_at=now() WHERE id=$1 AND state='succeeded'`, taskID)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgreSQLRepository) CompensateRestoreCore(ctx context.Context, task RestoreTask, apply applyRestoreCore) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, task.ProjectID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `SET LOCAL induforge.authoring_restore='on'`); err != nil {
		return err
	}
	if apply == nil {
		return fmt.Errorf("场景补偿事务未配置")
	}
	if err = apply(ctx, tx); err != nil {
		return err
	}
	var epoch int64
	if err = tx.QueryRow(ctx, `SELECT authoring_epoch FROM projects WHERE id=$1 AND tenant_id=$2 FOR UPDATE`, task.ProjectID, task.TenantID).Scan(&epoch); err != nil {
		return err
	}
	if epoch == task.TargetEpoch {
		if _, err = tx.Exec(ctx, `UPDATE projects SET authoring_epoch=$1,updated_at=now() WHERE id=$2 AND tenant_id=$3`, task.ExpectedEpoch, task.ProjectID, task.TenantID); err != nil {
			return err
		}
	} else if epoch != task.ExpectedEpoch {
		return fmt.Errorf("工程编辑代次补偿条件不满足")
	}
	return tx.Commit(ctx)
}
