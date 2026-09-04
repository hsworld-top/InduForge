package ops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// requireDeploymentIdle 必须在持有部署行锁后调用；创建、升级及运维动作共用该检查。
func requireDeploymentIdle(ctx context.Context, tx pgx.Tx, deploymentID string) error {
	var pending string
	err := tx.QueryRow(ctx, `SELECT id FROM deployment_runs WHERE project_deployment_id=$1 AND observed_status='pending'`, deploymentID).Scan(&pending)
	if err == nil {
		return ErrDeploymentBusy
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	return err
}

// storedDeploymentRequirements 只解析当前持久化的制品，不读取工程新配置或构建新包。
// 开发态使用原 descriptor；生产态按已固定版本读取。返回制品实际需要的引擎集合。
func storedDeploymentRequirements(ctx context.Context, reader releaseMetadataReader, tenant string, in CreateDeploymentInput, descriptor []byte) ([]string, error) {
	if in.Mode == "development" {
		var artifact DevelopmentArtifact
		if err := json.Unmarshal(descriptor, &artifact); err != nil {
			return nil, fmt.Errorf("%w: 已存开发制品描述无效", ErrReleaseNotDeployable)
		}
		in.DevelopmentArtifact = &artifact
	}
	metadata, err := deploymentMetadata(ctx, reader, tenant, in)
	if err != nil {
		return nil, err
	}
	if in.Mode == "development" {
		return deploymentRequirementsForDevelopmentArtifact(metadata, in.ProjectID)
	}
	return deploymentRequirementsForRelease(metadata, in.ProjectID)
}

const operateDeploymentServicesSQL = `UPDATE deployment_services SET desired_status=$1,observed_status='pending',desired_generation=desired_generation+1,last_operation=$2,updated_at=now() WHERE project_deployment_id=$3 AND tenant_id=$4 AND ($1='stopped' OR service_type=ANY($5::text[]))`
const operateDeploymentStatusSQL = `UPDATE project_deployments SET desired_status=$1,observed_status='pending',updated_at=now() WHERE id=$2 AND tenant_id=$3`

// queueDeploymentOperation 保持版本、模式、descriptor、绑定、端口及节点不变，只推进代次。
// redeploy 沿用内部 deploy 命令（K3s/Agent 已支持），审计明确区分“当前制品重新部署”。
func queueDeploymentOperation(ctx context.Context, tx pgx.Tx, tenant, did, op, user string) (DeploymentRun, error) {
	if op != "start" && op != "stop" && op != "restart" && op != "redeploy" {
		return DeploymentRun{}, fmt.Errorf("工程部署操作不支持")
	}
	var in CreateDeploymentInput
	var descriptor []byte
	var deleting bool
	err := tx.QueryRow(ctx, `SELECT project_id::text,mode,COALESCE(application_version_id::text,''),artifact_descriptor,deletion_requested_at IS NOT NULL FROM project_deployments WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL FOR UPDATE`, did, tenant).Scan(&in.ProjectID, &in.Mode, &in.ApplicationVersionID, &descriptor, &deleting)
	if err != nil {
		return DeploymentRun{}, mapNotFound(err)
	}
	if deleting {
		return DeploymentRun{}, ErrDeploymentBusy
	}
	if err := requireDeploymentIdle(ctx, tx, did); err != nil {
		return DeploymentRun{}, err
	}
	desired := "running"
	required := []string{}
	if op == "stop" {
		desired = "stopped"
	} else {
		required, err = storedDeploymentRequirements(ctx, tx, tenant, in, descriptor)
		if err != nil {
			return DeploymentRun{}, err
		}
		// 缺少当前制品所需服务时拒绝操作；不能默默调度新节点或以部分运行冒充成功。
		var available int
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM deployment_services WHERE project_deployment_id=$1 AND tenant_id=$2 AND service_type=ANY($3::text[])`, did, tenant, required).Scan(&available); err != nil {
			return DeploymentRun{}, err
		}
		if available != len(required) {
			return DeploymentRun{}, fmt.Errorf("%w: 当前制品所需引擎部署不完整", ErrReleaseNotDeployable)
		}
	}
	message := op + " deployment queued"
	if op == "redeploy" {
		op = "deploy"
		message = "重新部署任务已创建：重下发当前制品，保留模式、版本、节点、端口及数据"
	}
	// 运行相关动作只推进当前引擎；历史已停引擎保留状态及代次，不能复活或凭空变 pending。
	tag, err := tx.Exec(ctx, operateDeploymentServicesSQL, desired, op, did, tenant, required)
	if err != nil {
		return DeploymentRun{}, err
	}
	if tag.RowsAffected() == 0 {
		return DeploymentRun{}, ErrNotFound
	}
	if _, err = tx.Exec(ctx, operateDeploymentStatusSQL, desired, did, tenant); err != nil {
		return DeploymentRun{}, err
	}
	var run DeploymentRun
	err = tx.QueryRow(ctx, `INSERT INTO deployment_runs(tenant_id,project_deployment_id,operation,desired_status,message,created_by) VALUES($1,$2,$3,$4,$5,$6) RETURNING id,tenant_id,project_deployment_id,operation,desired_status,observed_status,progress,COALESCE(message,''),started_at,completed_at`, tenant, did, op, desired, message, user).Scan(runScanArgs(&run)...)
	if err != nil {
		return DeploymentRun{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO deployment_run_events(deployment_run_id,stage,message) VALUES($1,'queued',$2)`, run.ID, message)
	return run, err
}
