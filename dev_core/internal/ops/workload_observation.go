package ops

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type statementExecutor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

// captured generation是CAS边界；重复readiness/message不写库、不发事件，旧代次不能确认新代次。
const recordWorkloadObservationSQL = `UPDATE deployment_services SET observed_status=$1,last_message=$2,observed_generation=CASE WHEN $1='running' THEN $3 ELSE observed_generation END,replicas_observed=$4,observed_at=now(),updated_at=now() WHERE id=$5 AND desired_status='running' AND desired_generation=$3 AND (observed_status,COALESCE(last_message,''),replicas_observed,observed_generation) IS DISTINCT FROM ($1,$2,$4,CASE WHEN $1='running' THEN $3 ELSE observed_generation END)`
const insertRunEventOnceSQL = `INSERT INTO deployment_run_events(deployment_run_id,stage,message) SELECT r.id,$1,$2 FROM deployment_runs r WHERE r.project_deployment_id=$3 AND r.observed_status='pending' AND NOT EXISTS (SELECT 1 FROM deployment_run_events e WHERE e.deployment_run_id=r.id AND e.stage=$1 AND e.message=$2)`

func recordProjectWorkloadObservation(ctx context.Context, executor statementExecutor, workload ProjectWorkload, status ProjectWorkloadStatus) (bool, error) {
	observed, stage, message := "pending", "dispatched", "Kubernetes 工作负载已提交"
	if status.Ready {
		observed, stage, message = "running", "observed", "工程引擎已就绪"
	}
	if status.Failed {
		observed, stage, message = "failed", workloadFailureStage(status.Message), "工程引擎启动失败"
	}
	if len(status.Message) > 1024 {
		status.Message = status.Message[:1024]
	}
	tag, err := executor.Exec(ctx, recordWorkloadObservationSQL, observed, status.Message, workload.Generation, status.ReplicasObserved, workload.ServiceID)
	if err != nil || tag.RowsAffected() == 0 {
		return false, err
	}
	// 固定中文阶段消息不复制基础设施报错，业务详情仍通过受鉴权HTTP查询。
	_, err = executor.Exec(ctx, insertRunEventOnceSQL, stage, workload.Engine+"："+message, workload.DeploymentID)
	return true, err
}

type deploymentReconciliation struct {
	tenant, observed string
	changed          bool
}

// 外部K3s操作结束后才开启此短事务。先锁部署再校验服务代次，使服务观测、原run阶段、
// 汇总终态及last-ready一起提交；失败重试不会留下“服务ready但run永久pending”的半状态。
func recordAndReconcileWorkload(ctx context.Context, tx pgx.Tx, workload ProjectWorkload, status ProjectWorkloadStatus) (deploymentReconciliation, error) {
	var id string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM project_deployments WHERE id=$1 FOR UPDATE`, workload.DeploymentID).Scan(&id); err != nil {
		return deploymentReconciliation{}, err
	}
	changed, err := recordProjectWorkloadObservation(ctx, tx, workload, status)
	if err != nil {
		return deploymentReconciliation{}, err
	}
	state, err := reconcileDeploymentTransaction(ctx, tx, workload.DeploymentID)
	state.changed = state.changed || changed
	return state, err
}

func (r *PostgreSQLRepository) recordStoppedOutcome(ctx context.Context, did string, generation int64, failed bool, message string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	state, err := recordStoppedTransaction(ctx, tx, did, generation, failed, message)
	if err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	if state.changed {
		r.publish(state.tenant, []string{"deployments", "events"}, []string{did}, state.observed != "pending")
		r.publish(state.tenant, []string{"environments"}, nil, state.observed != "pending")
	}
	return nil
}

// 停止操作同样以部署锁和捕获的最大代次校验，既不确认新stop，也不把旧失败写进新run。
func recordStoppedTransaction(ctx context.Context, tx pgx.Tx, did string, generation int64, failed bool, message string) (deploymentReconciliation, error) {
	var desired string
	var currentGeneration int64
	if err := tx.QueryRow(ctx, `SELECT d.desired_status,COALESCE((SELECT max(desired_generation) FROM deployment_services WHERE project_deployment_id=d.id),0) FROM project_deployments d WHERE d.id=$1 FOR UPDATE`, did).Scan(&desired, &currentGeneration); err != nil {
		return deploymentReconciliation{}, err
	}
	if desired != "stopped" || currentGeneration != generation {
		return deploymentReconciliation{}, nil
	}
	observed, stage, eventMessage := "stopped", "observed", "Kubernetes 工作负载已停止"
	if failed {
		observed, stage, eventMessage = "failed", "failed", "Kubernetes 工作负载停止失败"
	}
	if len(message) > 1024 {
		message = message[:1024]
	}
	tag, err := tx.Exec(ctx, `UPDATE deployment_services SET observed_status=$1,last_message=$2,replicas_observed=CASE WHEN $1='stopped' THEN 0 ELSE replicas_observed END,observed_generation=CASE WHEN $1='stopped' THEN desired_generation ELSE observed_generation END,observed_at=now(),updated_at=now() WHERE project_deployment_id=$3 AND desired_status='stopped' AND (observed_status,COALESCE(last_message,''),observed_generation) IS DISTINCT FROM ($1,$2,CASE WHEN $1='stopped' THEN desired_generation ELSE observed_generation END)`, observed, message, did)
	if err != nil {
		return deploymentReconciliation{}, err
	}
	if tag.RowsAffected() > 0 {
		if _, err = tx.Exec(ctx, insertRunEventOnceSQL, stage, eventMessage, did); err != nil {
			return deploymentReconciliation{}, err
		}
	}
	state, err := reconcileDeploymentTransaction(ctx, tx, did)
	state.changed = state.changed || tag.RowsAffected() > 0
	return state, err
}

// 部署行锁与新操作共享，观测汇总、last-ready及run终态属于同一事务，不能覆盖刚入队的新操作。
func reconcileDeploymentTransaction(ctx context.Context, tx pgx.Tx, did string) (deploymentReconciliation, error) {
	var result deploymentReconciliation
	var desired, previous string
	if err := tx.QueryRow(ctx, `SELECT tenant_id::text,desired_status,observed_status FROM project_deployments WHERE id=$1 FOR UPDATE`, did).Scan(&result.tenant, &desired, &previous); err != nil {
		return result, err
	}
	var pending, failed int
	var generation int64
	if err := tx.QueryRow(ctx, `SELECT count(*) FILTER (WHERE observed_generation<>desired_generation OR observed_status<>desired_status),count(*) FILTER (WHERE observed_status='failed'),COALESCE(min(desired_generation) FILTER (WHERE desired_status='running'),0) FROM deployment_services WHERE project_deployment_id=$1`, did).Scan(&pending, &failed, &generation); err != nil {
		return result, err
	}
	result.observed = desired
	if failed > 0 {
		result.observed = "failed"
	} else if pending > 0 {
		result.observed = "pending"
	}
	if previous != result.observed {
		if _, err := tx.Exec(ctx, `UPDATE project_deployments SET observed_status=$1,updated_at=now() WHERE id=$2`, result.observed, did); err != nil {
			return result, err
		}
		result.changed = true
	}
	if result.observed == "pending" {
		return result, nil
	}
	if result.observed == "running" && generation > 0 {
		promoted, err := promoteDeploymentLastReady(ctx, tx, did, generation)
		if err != nil {
			return result, err
		}
		result.changed = result.changed || promoted
	}
	var runID string
	err := tx.QueryRow(ctx, `WITH latest AS (SELECT id FROM deployment_runs WHERE project_deployment_id=$1 AND observed_status='pending' ORDER BY started_at DESC,id DESC LIMIT 1) UPDATE deployment_runs SET observed_status=$2,progress=100,completed_at=now() WHERE id=(SELECT id FROM latest) RETURNING id`, did, result.observed).Scan(&runID)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	result.changed = true
	// 终态复用正式阶段枚举；run.completed_at 表示完成，不引入不存在的 completed 阶段。
	stage := "observed"
	if result.observed == "failed" {
		stage = "failed"
	}
	_, err = tx.Exec(ctx, `INSERT INTO deployment_run_events(deployment_run_id,stage,message) VALUES($1,$2,$3)`, runID, stage, "工程部署操作已结束："+result.observed)
	return result, err
}
