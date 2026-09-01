package ops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgreSQLRepository struct {
	pool       *pgxpool.Pool
	k3sAPIPort int
}

const defaultK3sVersion = "v1.36.4+k3s1"
const defaultRuntimeEnvironmentCode = "default-runtime"

// deploymentSelect 不从 project_deployments 推断节点；节点归属是 deployment_services 的属性。
const deploymentSelect = `SELECT d.id,d.tenant_id,d.project_id,p.name,d.environment_id,e.name,COALESCE(d.application_version_id::text,''),COALESCE(v.version,CASE WHEN d.mode='development' THEN '__DEV__' ELSE '' END),COALESCE((SELECT id::text FROM deployment_runs WHERE project_deployment_id=d.id ORDER BY started_at DESC,id DESC LIMIT 1),''),d.mode,d.access_port,d.desired_status,d.observed_status,COALESCE((SELECT progress FROM deployment_runs WHERE project_deployment_id=d.id ORDER BY started_at DESC,id DESC LIMIT 1),0),COALESCE(d.last_ready_mode,''),COALESCE(d.last_ready_application_version_id::text,''),COALESCE(lv.version,CASE WHEN d.last_ready_mode='development' THEN '__DEV__' ELSE '' END),COALESCE(d.last_ready_generation,0),d.last_ready_at,d.created_at,d.updated_at FROM project_deployments d JOIN projects p ON p.id=d.project_id AND p.tenant_id=d.tenant_id JOIN runtime_environments e ON e.id=d.environment_id AND e.tenant_id=d.tenant_id LEFT JOIN application_versions v ON v.id=d.application_version_id AND v.tenant_id=d.tenant_id LEFT JOIN application_versions lv ON lv.id=d.last_ready_application_version_id AND lv.tenant_id=d.tenant_id`

type releaseMetadata struct {
	ID, Version, ArtifactKey, ArtifactHash, ManifestHash, ChecksumsHash, SigningKeyID string
	ArtifactSize                                                                      int64
	Manifest                                                                          []byte
}

// pendingServicesSQL 只由服务自身的 node_id 调度。一个工程部署可将不同引擎放在不同节点。
const pendingServicesSQL = `SELECT s.id,s.tenant_id,s.project_deployment_id,s.node_id,s.service_type,s.public_port,s.desired_status,s.observed_status,COALESCE(s.last_message,''),COALESCE(s.endpoint,''),s.replicas_desired,s.replicas_observed,s.desired_generation,s.observed_generation,s.last_operation,s.observed_at,s.created_at,s.updated_at FROM deployment_services s WHERE s.node_id=$1 AND (s.desired_generation<>s.observed_generation OR s.desired_status<>s.observed_status) ORDER BY s.updated_at`

const (
	lockDeploymentProjectSQL = `SELECT id FROM projects WHERE tenant_id=$1 AND id=$2 FOR UPDATE`
	lockDeploymentNodeSQL    = `SELECT approved_at IS NOT NULL AND desired_status='active' AND observed_status='online' AND last_heartbeat_at>now()-interval '45 seconds' AND capabilities @> $3::jsonb FROM host_nodes WHERE tenant_id=$1 AND id=$2 FOR UPDATE`
)

func NewPostgreSQLRepository(pool *pgxpool.Pool) *PostgreSQLRepository {
	return &PostgreSQLRepository{pool: pool, k3sAPIPort: 6443}
}

// SetK3sAPIPort 注入安装配置中的 K3s API 端口。端口属于整套中心集群，
// 不能由单个节点自行决定，否则工作节点会加入错误的控制面地址。
func (r *PostgreSQLRepository) SetK3sAPIPort(port int) {
	if port >= 1 && port <= 65535 {
		r.k3sAPIPort = port
	}
}

// EnsureRuntimeCluster 由中心安装流程调用，把指定的已审批 Linux 节点固化为唯一中心节点。
// 重复调用是幂等的；已存在其他中心节点时拒绝覆盖，避免误把工作节点提升为控制面。
func (r *PostgreSQLRepository) EnsureRuntimeCluster(ctx context.Context, centerNodeID string) error {
	if _, err := uuid.Parse(centerNodeID); err != nil {
		return fmt.Errorf("中心节点 ID 无效: %w", err)
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var tenantID, nodeName, platform, actorID string
	var approved bool
	if err := tx.QueryRow(ctx, `SELECT n.tenant_id::text,n.display_name,n.platform,n.approved_at IS NOT NULL,e.created_by::text FROM host_nodes n JOIN node_enrollments e ON e.id=n.enrollment_id WHERE n.id=$1 AND n.desired_status='active' FOR UPDATE OF n`, centerNodeID).Scan(&tenantID, &nodeName, &platform, &approved, &actorID); err != nil {
		return mapNotFound(err)
	}
	if platform != "linux" || !approved {
		return fmt.Errorf("中心节点必须是已审批的 Linux 节点")
	}

	var clusterID string
	var persistedAPIPort int
	if err := tx.QueryRow(ctx, `INSERT INTO runtime_clusters(tenant_id,k3s_version,api_port) VALUES($1,$2,$3) ON CONFLICT (tenant_id) DO UPDATE SET updated_at=runtime_clusters.updated_at RETURNING id::text,api_port`, tenantID, defaultK3sVersion, r.k3sAPIPort).Scan(&clusterID, &persistedAPIPort); err != nil {
		return err
	}
	if persistedAPIPort != r.k3sAPIPort {
		return fmt.Errorf("K3s API 端口已在首次安装时固化为 %d，不能通过重启中心改为 %d", persistedAPIPort, r.k3sAPIPort)
	}
	var existingCenterID string
	err = tx.QueryRow(ctx, `SELECT node_id::text FROM runtime_cluster_nodes WHERE cluster_id=$1 AND node_kind='center' FOR UPDATE`, clusterID).Scan(&existingCenterID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if existingCenterID != "" && existingCenterID != centerNodeID {
		return fmt.Errorf("中心运行集群已绑定其他中心节点 %s", existingCenterID)
	}
	if existingCenterID == "" {
		if _, err := tx.Exec(ctx, `DELETE FROM runtime_cluster_nodes WHERE node_id=$1 AND node_kind='worker'`, centerNodeID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO runtime_cluster_nodes(tenant_id,cluster_id,node_id,node_kind) VALUES($1,$2,$3,'center')`, tenantID, clusterID, centerNodeID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO runtime_cluster_events(tenant_id,cluster_id,node_id,event_type,name,target,result,message) VALUES($1,$2,$3,'center_initialized','中心运行集群已初始化',$4,'success','中心节点将安装 K3s Server，并作为运行环境的统一控制面')`, tenantID, clusterID, centerNodeID, nodeName); err != nil {
			return err
		}
	}
	// 默认运行范围是平台初始化的一部分。普通用户进入运维管理时直接使用它，
	// 不需要先理解或手工创建“运行环境”。
	environmentID, err := ensureDefaultRuntimeEnvironment(ctx, tx, tenantID, actorID)
	if err != nil {
		return err
	}
	if err := assignNodeToDefaultRuntimeEnvironment(ctx, tx, tenantID, environmentID, centerNodeID, nodeName, actorID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func ensureDefaultRuntimeEnvironment(ctx context.Context, tx pgx.Tx, tenantID, actorID string) (string, error) {
	var environmentID string
	err := tx.QueryRow(ctx, `INSERT INTO runtime_environments(tenant_id,name,code,is_default,created_by) VALUES($1,'默认运行范围',$2,true,$3) ON CONFLICT (tenant_id,code) DO NOTHING RETURNING id::text`, tenantID, defaultRuntimeEnvironmentCode, actorID).Scan(&environmentID)
	if err == nil {
		_, err = tx.Exec(ctx, `INSERT INTO runtime_environment_events(tenant_id,environment_id,event_type,name,target,result,message,created_by) VALUES($1,$2,'default_environment_created','默认运行范围已初始化','平台','success','平台已创建默认运行范围，后续接入的 Linux 运行节点将自动加入',$3)`, tenantID, environmentID, actorID)
		return environmentID, err
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	err = tx.QueryRow(ctx, `SELECT id::text FROM runtime_environments WHERE tenant_id=$1 AND code=$2 AND is_default AND deleted_at IS NULL FOR UPDATE`, tenantID, defaultRuntimeEnvironmentCode).Scan(&environmentID)
	return environmentID, err
}

func assignNodeToDefaultRuntimeEnvironment(ctx context.Context, tx pgx.Tx, tenantID, environmentID, nodeID, nodeName, actorID string) error {
	tag, err := tx.Exec(ctx, `INSERT INTO runtime_environment_nodes(tenant_id,environment_id,node_id,created_by) VALUES($1,$2,$3,$4) ON CONFLICT (environment_id,node_id) DO NOTHING`, tenantID, environmentID, nodeID, actorID)
	if err != nil || tag.RowsAffected() == 0 {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO runtime_environment_events(tenant_id,environment_id,event_type,name,target,result,message,created_by) VALUES($1,$2,'node_auto_assigned','物理节点已加入默认运行范围',$3,'success','节点接入后由平台自动纳入默认调度范围',$4)`, tenantID, environmentID, nodeName, actorID)
	return err
}

func (r *PostgreSQLRepository) ListEnrollments(ctx context.Context, tenant string, f PageFilter) ([]Enrollment, int64, error) {
	_, _ = r.pool.Exec(ctx, `UPDATE node_enrollments SET status='expired',updated_at=now() WHERE tenant_id=$1 AND status='created' AND expires_at<=now()`, tenant)
	rows, e := r.pool.Query(ctx, `SELECT id,tenant_id,platform,capabilities,COALESCE(display_name,''),status,expires_at,claimed_at,COALESCE(claimed_by_node_id::text,''),approved_at,rejected_at,created_at,updated_at FROM node_enrollments WHERE tenant_id=$1 AND ($2='' OR display_name ILIKE '%'||$2||'%' OR platform ILIKE '%'||$2||'%') ORDER BY created_at DESC LIMIT $3 OFFSET $4`, tenant, f.Search, f.PageSize, (f.Page-1)*f.PageSize)
	if e != nil {
		return nil, 0, e
	}
	defer rows.Close()
	items := []Enrollment{}
	for rows.Next() {
		x, e := scanEnrollment(rows)
		if e != nil {
			return nil, 0, e
		}
		r.attachEnrollmentNode(ctx, &x)
		items = append(items, x)
	}
	var total int64
	e = r.pool.QueryRow(ctx, `SELECT count(*) FROM node_enrollments WHERE tenant_id=$1 AND ($2='' OR display_name ILIKE '%'||$2||'%' OR platform ILIKE '%'||$2||'%')`, tenant, f.Search).Scan(&total)
	return items, total, e
}
func (r *PostgreSQLRepository) CreateEnrollment(ctx context.Context, tenant, user string, in CreateEnrollmentInput, hash string) (Enrollment, error) {
	caps, _ := json.Marshal(in.Capabilities)
	return scanEnrollment(r.pool.QueryRow(ctx, `INSERT INTO node_enrollments(tenant_id,platform,capabilities,display_name,code_hash,expires_at,created_by) VALUES($1,$2,$3,$4,$5,now()+$6::interval,$7) RETURNING id,tenant_id,platform,capabilities,COALESCE(display_name,''),status,expires_at,claimed_at,COALESCE(claimed_by_node_id::text,''),approved_at,rejected_at,created_at,updated_at`, tenant, in.Platform, caps, in.DisplayName, hash, in.TTL.String(), user))
}
func (r *PostgreSQLRepository) GetEnrollment(ctx context.Context, tenant, id string) (Enrollment, error) {
	_, _ = r.pool.Exec(ctx, `UPDATE node_enrollments SET status='expired',updated_at=now() WHERE tenant_id=$1 AND id=$2 AND status='created' AND expires_at<=now()`, tenant, id)
	x, e := scanEnrollment(r.pool.QueryRow(ctx, `SELECT id,tenant_id,platform,capabilities,COALESCE(display_name,''),status,expires_at,claimed_at,COALESCE(claimed_by_node_id::text,''),approved_at,rejected_at,created_at,updated_at FROM node_enrollments WHERE tenant_id=$1 AND id=$2`, tenant, id))
	if e != nil {
		return x, mapNotFound(e)
	}
	r.attachEnrollmentNode(ctx, &x)
	return x, nil
}
func (r *PostgreSQLRepository) ApproveEnrollment(ctx context.Context, tenant, id, user string, approve bool) (Enrollment, error) {
	tx, e := r.pool.Begin(ctx)
	if e != nil {
		return Enrollment{}, e
	}
	defer tx.Rollback(ctx)
	status := "approved"
	field := "approved"
	if !approve {
		status = "rejected"
		field = "rejected"
	}
	x, e := scanEnrollment(tx.QueryRow(ctx, `UPDATE node_enrollments SET status=$1,`+field+`_at=now(),`+field+`_by=$2,updated_at=now() WHERE tenant_id=$3 AND id=$4 AND status='claimed' RETURNING id,tenant_id,platform,capabilities,COALESCE(display_name,''),status,expires_at,claimed_at,COALESCE(claimed_by_node_id::text,''),approved_at,rejected_at,created_at,updated_at`, status, user, tenant, id))
	if e != nil {
		return x, mapNotFound(e)
	}
	if approve {
		_, e = tx.Exec(ctx, `UPDATE host_nodes SET observed_status='offline',approved_at=now(),approved_by=$1,updated_at=now() WHERE enrollment_id=$2`, user, id)
		if e == nil {
			_, e = tx.Exec(ctx, `INSERT INTO runtime_cluster_nodes(tenant_id,cluster_id,node_id,node_kind) SELECT n.tenant_id,c.id,n.id,'worker' FROM host_nodes n JOIN runtime_clusters c ON c.tenant_id=n.tenant_id WHERE n.enrollment_id=$1 ON CONFLICT (node_id) DO NOTHING`, id)
		}
		if e == nil {
			_, e = tx.Exec(ctx, `INSERT INTO runtime_cluster_events(tenant_id,cluster_id,node_id,event_type,name,target,result,message,created_by) SELECT n.tenant_id,c.id,n.id,'worker_join_requested','运行节点接入任务已创建',n.display_name,'success','节点审批通过，等待加入中心运行集群',$2 FROM host_nodes n JOIN runtime_clusters c ON c.tenant_id=n.tenant_id WHERE n.enrollment_id=$1`, id, user)
		}
		if e == nil {
			var environmentID, nodeID, nodeName string
			e = tx.QueryRow(ctx, `SELECT e.id::text,n.id::text,n.display_name FROM host_nodes n JOIN runtime_environments e ON e.tenant_id=n.tenant_id AND e.is_default AND e.deleted_at IS NULL WHERE n.enrollment_id=$1 AND n.platform='linux' AND n.capabilities @> '["project_entry","data_runtime"]'::jsonb`, id).Scan(&environmentID, &nodeID, &nodeName)
			if errors.Is(e, pgx.ErrNoRows) {
				e = nil
			} else if e == nil {
				e = assignNodeToDefaultRuntimeEnvironment(ctx, tx, tenant, environmentID, nodeID, nodeName, user)
			}
		}
	} else {
		_, e = tx.Exec(ctx, `UPDATE host_nodes SET desired_status='revoked',observed_status='revoked',agent_token_hash='revoked:'||id::text,updated_at=now() WHERE enrollment_id=$1`, id)
	}
	if e != nil {
		return x, e
	}
	if e = tx.Commit(ctx); e != nil {
		return x, e
	}
	r.attachEnrollmentNode(ctx, &x)
	return x, nil
}
func (r *PostgreSQLRepository) ClaimEnrollment(ctx context.Context, in ClaimEnrollmentInput, agentHash string) (Enrollment, Node, error) {
	tx, e := r.pool.Begin(ctx)
	if e != nil {
		return Enrollment{}, Node{}, e
	}
	defer tx.Rollback(ctx)
	en, e := scanEnrollment(tx.QueryRow(ctx, `UPDATE node_enrollments SET status='claimed',claimed_at=now(),updated_at=now() WHERE code_hash=$1 AND status='created' AND expires_at>now() RETURNING id,tenant_id,platform,capabilities,COALESCE(display_name,''),status,expires_at,claimed_at,COALESCE(claimed_by_node_id::text,''),approved_at,rejected_at,created_at,updated_at`, hashToken(in.Code)))
	if e != nil {
		return Enrollment{}, Node{}, ErrEnrollmentUnavailable
	}
	if en.Platform != in.Platform || !sameStrings(in.Capabilities, en.Capabilities) {
		return Enrollment{}, Node{}, fmt.Errorf("NodeAgent 平台或能力与接入任务不匹配")
	}
	name := en.DisplayName
	if name == "" {
		name = in.DisplayName
	}
	if name == "" {
		name = in.Hostname
	}
	caps, _ := json.Marshal(in.Capabilities)
	n, e := scanNode(tx.QueryRow(ctx, `INSERT INTO host_nodes(tenant_id,enrollment_id,display_name,hostname,platform,architecture,agent_version,machine_fingerprint,ip_address,agent_token_hash,capabilities) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id,tenant_id,enrollment_id,display_name,hostname,platform,architecture,COALESCE(agent_version,''),COALESCE(machine_fingerprint,''),COALESCE(ip_address,''),desired_status,observed_status,capabilities,resource_summary,last_heartbeat_at,approved_at,created_at,updated_at`, en.TenantID, en.ID, name, in.Hostname, in.Platform, in.Architecture, in.AgentVersion, in.MachineFingerprint, in.IPAddress, agentHash, caps))
	if e != nil {
		return Enrollment{}, Node{}, e
	}
	_, e = tx.Exec(ctx, `UPDATE node_enrollments SET claimed_by_node_id=$1 WHERE id=$2`, n.ID, en.ID)
	if e != nil {
		return Enrollment{}, Node{}, e
	}
	return en, n, tx.Commit(ctx)
}
func (r *PostgreSQLRepository) ListNodes(ctx context.Context, tenant string, f PageFilter) ([]Node, int64, error) {
	rows, e := r.pool.Query(ctx, `SELECT n.id,n.tenant_id,n.enrollment_id,n.display_name,n.hostname,n.platform,n.architecture,COALESCE(n.agent_version,''),COALESCE(n.machine_fingerprint,''),COALESCE(n.ip_address,''),n.desired_status,n.observed_status,n.capabilities,n.resource_summary,n.last_heartbeat_at,n.approved_at,n.created_at,n.updated_at,COALESCE(d.id::text,''),COALESCE(d.project_id::text,''),COALESCE(p.name,''),COALESCE(env.id::text,''),COALESCE(env.name,'') FROM host_nodes n LEFT JOIN LATERAL (SELECT d.id,d.project_id FROM deployment_services s JOIN project_deployments d ON d.id=s.project_deployment_id AND d.tenant_id=s.tenant_id WHERE s.tenant_id=n.tenant_id AND s.node_id=n.id ORDER BY d.created_at DESC,d.id DESC LIMIT 1) d ON true LEFT JOIN projects p ON p.id=d.project_id AND p.tenant_id=n.tenant_id LEFT JOIN LATERAL (SELECT e.id,e.name FROM runtime_environment_nodes en JOIN runtime_environments e ON e.id=en.environment_id AND e.deleted_at IS NULL WHERE en.node_id=n.id ORDER BY en.created_at DESC LIMIT 1) env ON true WHERE n.tenant_id=$1 AND ($2='' OR n.display_name ILIKE '%'||$2||'%' OR n.hostname ILIKE '%'||$2||'%') ORDER BY n.updated_at DESC,n.id DESC LIMIT $3 OFFSET $4`, tenant, f.Search, f.PageSize, (f.Page-1)*f.PageSize)
	if e != nil {
		return nil, 0, e
	}
	defer rows.Close()
	items := []Node{}
	for rows.Next() {
		x, e := scanNodeWithAssignments(rows)
		if e != nil {
			return nil, 0, e
		}
		if e = r.attachNodeCluster(ctx, &x); e != nil {
			return nil, 0, e
		}
		items = append(items, x)
	}
	var total int64
	e = r.pool.QueryRow(ctx, `SELECT count(*) FROM host_nodes WHERE tenant_id=$1 AND ($2='' OR display_name ILIKE '%'||$2||'%' OR hostname ILIKE '%'||$2||'%')`, tenant, f.Search).Scan(&total)
	return items, total, e
}
func (r *PostgreSQLRepository) GetNode(ctx context.Context, tenant, id string) (Node, error) {
	x, e := scanNodeWithAssignments(r.pool.QueryRow(ctx, `SELECT n.id,n.tenant_id,n.enrollment_id,n.display_name,n.hostname,n.platform,n.architecture,COALESCE(n.agent_version,''),COALESCE(n.machine_fingerprint,''),COALESCE(n.ip_address,''),n.desired_status,n.observed_status,n.capabilities,n.resource_summary,n.last_heartbeat_at,n.approved_at,n.created_at,n.updated_at,COALESCE(d.id::text,''),COALESCE(d.project_id::text,''),COALESCE(p.name,''),COALESCE(env.id::text,''),COALESCE(env.name,'') FROM host_nodes n LEFT JOIN LATERAL (SELECT d.id,d.project_id FROM deployment_services s JOIN project_deployments d ON d.id=s.project_deployment_id AND d.tenant_id=s.tenant_id WHERE s.tenant_id=n.tenant_id AND s.node_id=n.id ORDER BY d.created_at DESC,d.id DESC LIMIT 1) d ON true LEFT JOIN projects p ON p.id=d.project_id AND p.tenant_id=n.tenant_id LEFT JOIN LATERAL (SELECT e.id,e.name FROM runtime_environment_nodes en JOIN runtime_environments e ON e.id=en.environment_id AND e.deleted_at IS NULL WHERE en.node_id=n.id ORDER BY en.created_at DESC LIMIT 1) env ON true WHERE n.tenant_id=$1 AND n.id=$2`, tenant, id))
	if e != nil {
		return x, mapNotFound(e)
	}
	return x, r.attachNodeCluster(ctx, &x)
}
func (r *PostgreSQLRepository) RemoveNode(ctx context.Context, tenant, id, user string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var clusterID, nodeName, nodeKind, desiredAction, observedStatus string
	var lastHeartbeat *time.Time
	err = tx.QueryRow(ctx, `SELECT cn.cluster_id::text,n.display_name,cn.node_kind,cn.desired_action,n.observed_status,n.last_heartbeat_at FROM runtime_cluster_nodes cn JOIN host_nodes n ON n.id=cn.node_id WHERE cn.tenant_id=$1 AND cn.node_id=$2 FOR UPDATE OF cn,n`, tenant, id).Scan(&clusterID, &nodeName, &nodeKind, &desiredAction, &observedStatus, &lastHeartbeat)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if nodeKind == "center" {
		return ErrCenterNodeProtected
	}
	if desiredAction == "removing" {
		return tx.Commit(ctx)
	}
	var environmentCount, deploymentCount int
	if err := tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM runtime_environment_nodes WHERE node_id=$1),(SELECT count(*) FROM deployment_services WHERE node_id=$1)`, id).Scan(&environmentCount, &deploymentCount); err != nil {
		return err
	}
	if environmentCount > 0 {
		return ErrNodeEnvironmentInUse
	}
	if deploymentCount > 0 {
		return ErrNodeDeploymentInUse
	}
	if observedStatus != "online" || lastHeartbeat == nil || time.Since(*lastHeartbeat) > 45*time.Second {
		return ErrNodeOfflineForRemoval
	}
	if _, err := tx.Exec(ctx, `UPDATE runtime_cluster_nodes SET desired_action='removing',cluster_status='removing',cluster_message='等待运行节点退出中心集群' WHERE node_id=$1`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO runtime_cluster_events(tenant_id,cluster_id,node_id,event_type,name,target,result,message,created_by) VALUES($1,$2,$3,'worker_remove_requested','运行节点移除任务已创建',$4,'success','等待节点卸载 K3s Agent 并撤销接入凭据',$5)`, tenant, clusterID, id, nodeName, user); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *PostgreSQLRepository) Heartbeat(ctx context.Context, id, hash string, in HeartbeatInput) (Node, []DeploymentService, error) {
	summary, _ := json.Marshal(in.ResourceSummary)
	var previousStatus string
	var previousSummary []byte
	_ = r.pool.QueryRow(ctx, `SELECT observed_status,resource_summary FROM host_nodes WHERE id=$1 AND agent_token_hash=$2`, id, hash).Scan(&previousStatus, &previousSummary)
	n, e := scanNode(r.pool.QueryRow(ctx, `UPDATE host_nodes SET observed_status=CASE WHEN desired_status='revoked' THEN 'revoked' WHEN observed_status='pending_approval' THEN 'pending_approval' ELSE 'online' END,resource_summary=$1,agent_version=COALESCE(NULLIF($2,''),agent_version),ip_address=COALESCE(NULLIF($3,''),ip_address),last_heartbeat_at=now(),updated_at=now() WHERE id=$4 AND agent_token_hash=$5 RETURNING id,tenant_id,enrollment_id,display_name,hostname,platform,architecture,COALESCE(agent_version,''),COALESCE(machine_fingerprint,''),COALESCE(ip_address,''),desired_status,observed_status,capabilities,resource_summary,last_heartbeat_at,approved_at,created_at,updated_at`, summary, in.AgentVersion, in.IPAddress, id, hash))
	if e != nil {
		return Node{}, nil, ErrAgentUnauthorized
	}
	if n.ObservedStatus != "online" || n.DesiredStatus != "active" {
		return n, []DeploymentService{}, nil
	}
	if previousStatus == "offline" {
		_, e = r.pool.Exec(ctx, `INSERT INTO runtime_cluster_events(tenant_id,cluster_id,node_id,event_type,name,target,result,message) SELECT $1,cn.cluster_id,$3,'node_online','物理节点已恢复在线',$2,'success','节点心跳已恢复，集群期望状态保持不变' FROM runtime_cluster_nodes cn WHERE cn.node_id=$3`, n.TenantID, n.DisplayName, n.ID)
		if e != nil {
			return n, nil, e
		}
		_, e = r.pool.Exec(ctx, `INSERT INTO runtime_environment_events(tenant_id,environment_id,event_type,name,target,result,message) SELECT $1,en.environment_id,'node_online','物理节点已恢复在线',$2,'success','节点心跳已恢复，集群期望状态保持不变' FROM runtime_environment_nodes en WHERE en.node_id=$3`, n.TenantID, n.DisplayName, n.ID)
		if e != nil {
			return n, nil, e
		}
	}
	previousTimeStatus := timeSyncStatusFromSummary(previousSummary)
	currentTimeStatus, currentTimeMessage := timeSyncStatusFromMap(in.ResourceSummary)
	if currentTimeStatus != "" && timeSyncEventClass(currentTimeStatus) != timeSyncEventClass(previousTimeStatus) {
		name, result := "节点时间同步已恢复", "success"
		if currentTimeStatus == "failed" || currentTimeStatus == "unconfigured" {
			name, result = "节点时间同步异常", "failed"
		} else if currentTimeStatus == "adjusting" {
			name = "节点时间正在校准"
		}
		if _, e = r.pool.Exec(ctx, `INSERT INTO runtime_cluster_events(tenant_id,cluster_id,node_id,event_type,name,target,result,message) SELECT $1,cn.cluster_id,$3,'node_time_sync_changed',$4,$2,$5,$6 FROM runtime_cluster_nodes cn WHERE cn.node_id=$3`, n.TenantID, n.DisplayName, n.ID, name, result, currentTimeMessage); e != nil {
			return n, nil, e
		}
		if _, e = r.pool.Exec(ctx, `INSERT INTO runtime_environment_events(tenant_id,environment_id,event_type,name,target,result,message) SELECT $1,en.environment_id,'node_time_sync_changed',$4,$2,$5,$6 FROM runtime_environment_nodes en WHERE en.node_id=$3`, n.TenantID, n.DisplayName, n.ID, name, result, currentTimeMessage); e != nil {
			return n, nil, e
		}
	}
	if in.ClusterState != nil && in.ClusterState.ClusterID != "" {
		state := *in.ClusterState
		var clusterID, previousStatus, nodeKind, desiredAction string
		var previousGeneration int64
		err := r.pool.QueryRow(ctx, `SELECT cluster_id::text,cluster_status,observed_generation,node_kind,desired_action FROM runtime_cluster_nodes WHERE node_id=$1 FOR UPDATE`, n.ID).Scan(&clusterID, &previousStatus, &previousGeneration, &nodeKind, &desiredAction)
		if err == nil {
			if !acceptClusterStateIdentity(state, n.ID, clusterID) {
				return n, nil, fmt.Errorf("节点上报了不属于自身运行集群的状态")
			}
			// 工作节点重新注册后可能仍携带旧集群的卸载墓碑。墓碑不含运行资源，可归一化
			// 到当前唯一集群；任何仍在运行的跨集群状态必须拒绝。
			if state.ObservedState == "not-installed" {
				state.ClusterID = clusterID
			}
			var result pgconn.CommandTag
			var updateErr error
			if state.ObservedState == "not-installed" {
				result, updateErr = r.pool.Exec(ctx, `UPDATE runtime_cluster_nodes SET observed_generation=0,cluster_status='not-installed',cluster_message=$1,cluster_observed_at=now() WHERE node_id=$2 AND cluster_id=$3`, state.Message, n.ID, state.ClusterID)
			} else {
				result, updateErr = r.pool.Exec(ctx, `UPDATE runtime_cluster_nodes SET observed_generation=$1,cluster_status=$2,cluster_message=$3,cluster_observed_at=now() WHERE node_id=$4 AND cluster_id=$5 AND desired_generation=$1`, state.Generation, state.ObservedState, state.Message, n.ID, state.ClusterID)
			}
			if updateErr != nil {
				return n, nil, updateErr
			}
			if result.RowsAffected() == 1 && (previousStatus != state.ObservedState || previousGeneration != state.Generation) {
				resultName, eventResult := "节点集群基础设施状态已更新", "success"
				if state.ObservedState == "failed" {
					resultName, eventResult = "节点集群基础设施异常", "failed"
				}
				_, updateErr = r.pool.Exec(ctx, `INSERT INTO runtime_cluster_events(tenant_id,cluster_id,node_id,event_type,name,target,result,message) VALUES($1,$2,$3,'cluster_state_changed',$4,$5,$6,$7)`, n.TenantID, state.ClusterID, n.ID, resultName, n.DisplayName, eventResult, state.Message)
				if updateErr != nil {
					return n, nil, updateErr
				}
			}
			if state.ObservedState == "not-installed" && desiredAction == "removing" && nodeKind == "worker" {
				if err := r.finalizeClusterNodeRemoval(ctx, n.TenantID, n.ID, n.DisplayName, state.ClusterID); err != nil {
					return n, nil, err
				}
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return n, nil, err
		}
	}
	for index := range in.FoundationStates {
		state := &in.FoundationStates[index]
		if state.EnvironmentID == "" {
			continue
		}
		var environmentID, nodeKind string
		if err := r.pool.QueryRow(ctx, `SELECT e.id::text,cn.node_kind FROM runtime_environments e JOIN runtime_cluster_nodes cn ON cn.tenant_id=e.tenant_id WHERE e.id=$1 AND e.tenant_id=$2 AND e.deleted_at IS NULL AND cn.node_id=$3`, state.EnvironmentID, n.TenantID, n.ID).Scan(&environmentID, &nodeKind); errors.Is(err, pgx.ErrNoRows) {
			// 控制面重建后，节点可能在本地短暂保留已删除环境的观测文件。
			// 该状态没有可写目标，直接忽略，不能阻断当前节点心跳和新计划下发。
			continue
		} else if err != nil {
			return n, nil, err
		}
		if environmentID != state.EnvironmentID || state.NodeID != n.ID || nodeKind != "center" {
			return n, nil, fmt.Errorf("基础服务状态只能由中心节点上报")
		}
		if state.ObservedState == "not-installed" {
			if err := r.finalizeEnvironmentDeletion(ctx, n.TenantID, environmentID); err != nil {
				return n, nil, err
			}
			continue
		}
		observedStatus := "pending"
		eventResult := "success"
		name := "基础服务正在部署"
		switch state.ObservedState {
		case "ready":
			observedStatus, name = "running", "基础服务部署完成"
		case "failed":
			observedStatus, name, eventResult = "failed", "基础服务部署异常", "failed"
		}
		desiredServices := make(map[string]FoundationServiceState, len(foundationServiceTypes))
		for _, service := range state.Services {
			serviceStatus := "pending"
			if service.Status == "ready" {
				serviceStatus = "running"
			} else if service.Status == "failed" {
				serviceStatus = "failed"
			}
			for _, logicalType := range logicalTypesForFoundationWorkload(service.Workload) {
				desiredServices[logicalType] = FoundationServiceState{Status: serviceStatus, Message: service.Message}
			}
		}
		// 查询 Kubernetes 失败时 Hostd 无法给出逐项明细，此时所有基础服务都
		// 继承聚合失败；正常路径必须覆盖完整 8 项，不能先统一写再覆盖，否则
		// 单项异常会在每次心跳都制造状态变化和重复事件。
		if len(desiredServices) == 0 {
			for _, serviceType := range foundationServiceTypes {
				desiredServices[serviceType] = FoundationServiceState{Status: observedStatus, Message: state.Message}
			}
		}
		changed := false
		for _, serviceType := range foundationServiceTypes {
			desired, exists := desiredServices[serviceType]
			if !exists {
				return n, nil, fmt.Errorf("基础服务状态明细不完整")
			}
			var serviceChanged bool
			if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM runtime_environment_services WHERE environment_id=$1 AND service_type=$2 AND desired_generation=$3 AND (observed_generation<>$3 OR observed_status<>$4 OR COALESCE(last_message,'')<>$5))`, environmentID, serviceType, state.Generation, desired.Status, desired.Message).Scan(&serviceChanged); err != nil {
				return n, nil, err
			}
			if _, err := r.pool.Exec(ctx, `UPDATE runtime_environment_services SET observed_generation=$1,observed_status=$2,last_message=$3,observed_at=now(),previous_node_id=CASE WHEN $2='running' THEN NULL ELSE previous_node_id END,previous_storage_claim=CASE WHEN $2='running' THEN NULL ELSE previous_storage_claim END,operation=CASE WHEN $2='running' THEN 'apply' ELSE operation END,updated_at=now() WHERE environment_id=$4 AND service_type=$5 AND desired_generation=$1`, state.Generation, desired.Status, desired.Message, environmentID, serviceType); err != nil {
				return n, nil, err
			}
			if desired.Status == "running" {
				refs, refErr := foundationResourceRefs(environmentID, serviceType)
				if refErr != nil {
					return n, nil, refErr
				}
				if refs != nil {
					raw, marshalErr := json.Marshal(refs)
					if marshalErr != nil {
						return n, nil, marshalErr
					}
					if _, refErr = r.pool.Exec(ctx, `UPDATE runtime_environment_services SET resource_refs=$1::jsonb,updated_at=now() WHERE environment_id=$2 AND service_type=$3`, raw, environmentID, serviceType); refErr != nil {
						return n, nil, refErr
					}
				}
			}
			changed = changed || serviceChanged
		}
		if changed {
			if _, err := r.pool.Exec(ctx, `INSERT INTO runtime_environment_events(tenant_id,environment_id,event_type,name,target,result,message) VALUES($1,$2,'foundation_state_changed',$3,'全部基础服务',$4,$5)`, n.TenantID, environmentID, name, eventResult, state.Message); err != nil {
				return n, nil, err
			}
		}
	}
	affected := map[string]struct{}{}
	for _, o := range in.Services {
		var did string
		e = r.pool.QueryRow(ctx, `UPDATE deployment_services s SET observed_status=$1,replicas_observed=$2,observed_generation=$3,last_message=$4,endpoint=CASE WHEN s.service_type='base' THEN $5 ELSE s.endpoint END,observed_at=now(),updated_at=now() WHERE s.id=$6 AND s.desired_generation=$3 AND s.node_id=$7 AND (s.service_type='base' OR $5='') RETURNING s.project_deployment_id`, o.ObservedStatus, o.ReplicasObserved, o.ObservedGeneration, o.Message, o.Endpoint, o.ServiceID, n.ID).Scan(&did)
		if e == nil {
			affected[did] = struct{}{}
		} else if !errors.Is(e, pgx.ErrNoRows) {
			return n, nil, e
		} else if o.Endpoint != "" {
			return n, nil, fmt.Errorf("只有工程入口服务可以上报访问地址")
		}
	}
	for did := range affected {
		if e = r.reconcileDeployment(ctx, did); e != nil {
			return n, nil, e
		}
	}
	services, e := r.pendingServices(ctx, n.ID)
	return n, services, e
}

// acceptClusterStateIdentity 只放行重新接入节点遗留的卸载墓碑；墓碑不代表任何
// 仍在运行的集群资源。其他状态必须同时匹配新节点身份和当前集群，避免跨集群污染。
func acceptClusterStateIdentity(state ClusterState, nodeID, clusterID string) bool {
	if state.ObservedState == "not-installed" {
		return true
	}
	return state.NodeID == nodeID && state.ClusterID == clusterID
}

func timeSyncStatusFromSummary(raw []byte) string {
	var summary map[string]any
	if json.Unmarshal(raw, &summary) != nil {
		return ""
	}
	status, _ := timeSyncStatusFromMap(summary)
	return status
}

func timeSyncStatusFromMap(summary map[string]any) (string, string) {
	raw, exists := summary["timeSync"]
	if !exists {
		return "", ""
	}
	value, ok := raw.(map[string]any)
	if !ok {
		return "", ""
	}
	status, _ := value["status"].(string)
	message, _ := value["message"].(string)
	if message == "" {
		if role, _ := value["role"].(string); role == "center" {
			message = "中心服务器当前系统时间作为集群时间基准"
		} else if offset, ok := value["offsetMillis"].(float64); ok {
			message = fmt.Sprintf("与中心时间偏差 %.1f 毫秒", offset)
		} else {
			message = "节点已按中心服务器时间运行"
		}
	}
	return status, message
}

func timeSyncEventClass(status string) string {
	switch status {
	case "synchronized", "adjusting":
		return "healthy"
	case "failed", "unconfigured":
		return "failed"
	default:
		return ""
	}
}

func logicalTypesForFoundationWorkload(workload string) []string {
	switch workload {
	case "postgres":
		return []string{"if_history", "if_timeseries"}
	case "redis":
		return []string{"if_realtime"}
	case "emqx":
		return []string{"if_message"}
	case "nats":
		return []string{"nats_jetstream"}
	case "object":
		return []string{"if_object"}
	case "nginx":
		return []string{"nginx"}
	case "traefik":
		return []string{"traefik"}
	default:
		return nil
	}
}

// ReconcileNodeLiveness 把心跳超时转换为一次性的状态迁移和环境事件。仅首次从
// online 变为 offline 时写事件，定时扫描不会产生重复告警风暴。
func (r *PostgreSQLRepository) ReconcileNodeLiveness(ctx context.Context) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `WITH offline AS (
		UPDATE host_nodes SET observed_status='offline',updated_at=now()
		WHERE observed_status='online' AND desired_status<>'revoked' AND last_heartbeat_at<now()-interval '45 seconds'
		RETURNING id,tenant_id,display_name
	), cluster_events AS (
		INSERT INTO runtime_cluster_events(tenant_id,cluster_id,node_id,event_type,name,target,result,message)
		SELECT offline.tenant_id,cn.cluster_id,offline.id,'node_offline','物理节点心跳超时',offline.display_name,'failed','超过 45 秒未收到心跳；停止下发新操作，既有工作负载不被强制终止'
		FROM offline JOIN runtime_cluster_nodes cn ON cn.node_id=offline.id
		RETURNING id
	), environment_events AS (
	INSERT INTO runtime_environment_events(tenant_id,environment_id,event_type,name,target,result,message)
	SELECT offline.tenant_id,en.environment_id,'node_offline','物理节点心跳超时',offline.display_name,'failed','超过 45 秒未收到心跳；已停止向该节点下发新操作，节点上既有进程不被强制终止'
	FROM offline JOIN runtime_environment_nodes en ON en.node_id=offline.id
	RETURNING id)
	SELECT count(*) FROM offline`).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *PostgreSQLRepository) AgentClusterPlan(ctx context.Context, id, hash string) (*ClusterPlan, error) {
	var plan ClusterPlan
	var nodeKind, serverIP, serverStatus, status string
	err := r.pool.QueryRow(ctx, `SELECT cn.cluster_id::text,cn.node_id::text,cn.node_kind,cn.desired_generation,n.ip_address,cn.cluster_status,c.k3s_version,c.api_port,COALESCE(center.ip_address,''),COALESCE(center_cn.cluster_status,'') FROM host_nodes n JOIN runtime_cluster_nodes cn ON cn.node_id=n.id JOIN runtime_clusters c ON c.id=cn.cluster_id LEFT JOIN runtime_cluster_nodes center_cn ON center_cn.cluster_id=cn.cluster_id AND center_cn.node_kind='center' LEFT JOIN host_nodes center ON center.id=center_cn.node_id WHERE n.id=$1 AND n.agent_token_hash=$2 AND n.desired_status='active' AND n.observed_status='online' AND n.approved_at IS NOT NULL AND cn.desired_action='active' AND c.desired_status='active'`, id, hash).Scan(&plan.ClusterID, &plan.NodeID, &nodeKind, &plan.Generation, &plan.NodeIP, &status, &plan.K3sVersion, &plan.APIPort, &serverIP, &serverStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, tokenErr := r.GetNodeByToken(ctx, id, hash); tokenErr != nil {
			return nil, ErrAgentUnauthorized
		}
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if status == "ready" || plan.NodeIP == "" {
		return nil, nil
	}
	plan.SchemaVersion = "induforge.cluster-plan.v1"
	plan.VXLANPort = 8472
	plan.KubeletPort = 10250
	if nodeKind == "center" {
		plan.Operation = "init-server"
	} else {
		if serverIP == "" || serverStatus != "ready" {
			return nil, nil
		}
		plan.Operation = "join-agent"
		plan.ServerURL = "https://" + net.JoinHostPort(serverIP, fmt.Sprintf("%d", plan.APIPort))
	}
	return &plan, nil
}

func (r *PostgreSQLRepository) AgentClusterUninstall(ctx context.Context, id, hash string) (*ClusterUninstall, error) {
	var request ClusterUninstall
	err := r.pool.QueryRow(ctx, `SELECT cn.cluster_id::text,cn.node_id::text FROM host_nodes n JOIN runtime_cluster_nodes cn ON cn.node_id=n.id WHERE n.id=$1 AND n.agent_token_hash=$2 AND n.desired_status='active' AND n.observed_status='online' AND n.approved_at IS NOT NULL AND cn.node_kind='worker' AND cn.desired_action='removing'`, id, hash).Scan(&request.ClusterID, &request.NodeID)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, tokenErr := r.GetNodeByToken(ctx, id, hash); tokenErr != nil {
			return nil, ErrAgentUnauthorized
		}
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	request.PurgeData = true
	return &request, nil
}

func (r *PostgreSQLRepository) finalizeClusterNodeRemoval(ctx context.Context, tenant, nodeID, nodeName, clusterID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var nodeKind, desiredAction string
	err = tx.QueryRow(ctx, `SELECT node_kind,desired_action FROM runtime_cluster_nodes WHERE tenant_id=$1 AND cluster_id=$2 AND node_id=$3 FOR UPDATE`, tenant, clusterID, nodeID).Scan(&nodeKind, &desiredAction)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if nodeKind != "worker" || desiredAction != "removing" {
		return nil
	}
	if _, err := tx.Exec(ctx, `INSERT INTO runtime_cluster_events(tenant_id,cluster_id,node_id,event_type,name,target,result,message) VALUES($1,$2,$3,'worker_removed','运行节点已安全移除',$4,'success','K3s Agent 已卸载，节点凭据已撤销')`, tenant, clusterID, nodeID, nodeName); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM runtime_cluster_nodes WHERE node_id=$1`, nodeID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE host_nodes SET desired_status='revoked',observed_status='revoked',agent_token_hash='revoked:'||id::text,updated_at=now() WHERE id=$1 AND tenant_id=$2`, nodeID, tenant); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PostgreSQLRepository) AgentFoundationPlan(ctx context.Context, id, hash string) (*FoundationPlan, error) {
	var nodeKind, clusterStatus string
	err := r.pool.QueryRow(ctx, `SELECT cn.node_kind,cn.cluster_status FROM host_nodes n JOIN runtime_cluster_nodes cn ON cn.node_id=n.id WHERE n.id=$1 AND n.agent_token_hash=$2 AND n.desired_status='active' AND n.observed_status='online' AND n.approved_at IS NOT NULL AND cn.desired_action='active'`, id, hash).Scan(&nodeKind, &clusterStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, tokenErr := r.GetNodeByToken(ctx, id, hash); tokenErr != nil {
			return nil, ErrAgentUnauthorized
		}
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if nodeKind != "center" || clusterStatus != "ready" {
		return nil, nil
	}
	var environmentID string
	err = r.pool.QueryRow(ctx, `SELECT e.id::text FROM runtime_environments e WHERE e.tenant_id=(SELECT tenant_id FROM host_nodes WHERE id=$1) AND e.desired_status='active' AND e.deleted_at IS NULL AND EXISTS (SELECT 1 FROM runtime_environment_services s WHERE s.environment_id=e.id AND (s.desired_generation<>s.observed_generation OR s.observed_status<>'running')) ORDER BY e.updated_at,e.id LIMIT 1`, id).Scan(&environmentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT service_type,node_id::text,COALESCE(previous_node_id::text,''),storage_claim,COALESCE(previous_storage_claim,''),operation,desired_generation,observed_generation,observed_status FROM runtime_environment_services WHERE environment_id=$1 ORDER BY service_type`, environmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	logical := make(map[string]string)
	sourceLogical, logicalClaims, sourceLogicalClaims := make(map[string]string), make(map[string]string), make(map[string]string)
	operation := "apply"
	var generation int64
	pending := false
	count := 0
	for rows.Next() {
		var serviceType, nodeID, sourceNodeID, claim, sourceClaim, serviceOperation, observedStatus string
		var desired, observed int64
		if err := rows.Scan(&serviceType, &nodeID, &sourceNodeID, &claim, &sourceClaim, &serviceOperation, &desired, &observed, &observedStatus); err != nil {
			return nil, err
		}
		logical[serviceType] = nodeID
		logicalClaims[serviceType] = claim
		if sourceNodeID != "" {
			sourceLogical[serviceType], sourceLogicalClaims[serviceType] = sourceNodeID, sourceClaim
		}
		if serviceOperation == "migrate" {
			operation = "migrate"
		}
		if desired > generation {
			generation = desired
		}
		if desired != observed || observedStatus != "running" {
			pending = true
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if count == 0 || !pending {
		return nil, nil
	}
	if count != len(foundationServiceTypes) {
		return nil, fmt.Errorf("基础服务计划不完整")
	}
	assignments := map[string]string{
		"postgres": logical["if_history"], "redis": logical["if_realtime"],
		"emqx": logical["if_message"], "nats": logical["nats_jetstream"],
		"object": logical["if_object"], "nginx": logical["nginx"],
	}
	claims := map[string]string{"postgres": logicalClaims["if_history"], "redis": logicalClaims["if_realtime"], "emqx": logicalClaims["if_message"], "nats": logicalClaims["nats_jetstream"], "object": logicalClaims["if_object"]}
	sourceAssignments := map[string]string{}
	sourceClaims := map[string]string{}
	for workload, serviceTypes := range map[string][]string{"postgres": {"if_history"}, "redis": {"if_realtime"}, "emqx": {"if_message"}, "nats": {"nats_jetstream"}, "object": {"if_object"}, "nginx": {"nginx"}} {
		serviceType := serviceTypes[0]
		if sourceLogical[serviceType] != "" {
			sourceAssignments[workload], sourceClaims[workload] = sourceLogical[serviceType], sourceLogicalClaims[serviceType]
		}
	}
	return &FoundationPlan{SchemaVersion: "induforge.foundation-plan.v1", Generation: generation, EnvironmentID: environmentID, NodeID: id, Assignments: assignments, SourceAssignments: sourceAssignments, Claims: claims, SourceClaims: sourceClaims, Operation: operation}, nil
}

func (r *PostgreSQLRepository) AgentTimeSyncPlan(ctx context.Context, id, hash string) (*TimeSyncPlan, error) {
	var plan TimeSyncPlan
	var nodeKind string
	err := r.pool.QueryRow(ctx, `SELECT cn.node_id::text,cn.node_kind,cn.desired_generation,center.ip_address FROM host_nodes n JOIN runtime_cluster_nodes cn ON cn.node_id=n.id JOIN runtime_cluster_nodes center_cn ON center_cn.cluster_id=cn.cluster_id AND center_cn.node_kind='center' JOIN host_nodes center ON center.id=center_cn.node_id WHERE n.id=$1 AND n.agent_token_hash=$2 AND n.desired_status='active' AND n.observed_status='online' AND n.approved_at IS NOT NULL AND cn.desired_action='active' AND cn.cluster_status='ready' AND center_cn.cluster_status='ready'`, id, hash).Scan(&plan.NodeID, &nodeKind, &plan.Generation, &plan.CenterIP)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, tokenErr := r.GetNodeByToken(ctx, id, hash); tokenErr != nil {
			return nil, ErrAgentUnauthorized
		}
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	plan.SchemaVersion = "induforge.time-sync-plan.v1"
	if nodeKind == "center" {
		plan.Role = "center"
		rows, queryErr := r.pool.Query(ctx, `SELECT n.ip_address FROM runtime_cluster_nodes cn JOIN host_nodes n ON n.id=cn.node_id WHERE cn.cluster_id=(SELECT cluster_id FROM runtime_cluster_nodes WHERE node_id=$1) AND cn.node_kind='worker' AND cn.desired_action='active' AND n.ip_address<>'' ORDER BY n.ip_address`, id)
		if queryErr != nil {
			return nil, queryErr
		}
		defer rows.Close()
		for rows.Next() {
			var address string
			if scanErr := rows.Scan(&address); scanErr != nil {
				return nil, scanErr
			}
			plan.AllowedClients = append(plan.AllowedClients, address)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	} else {
		plan.Role = "client"
	}
	return &plan, nil
}

func (r *PostgreSQLRepository) AgentFoundationDelete(ctx context.Context, id, hash string) (*FoundationDelete, error) {
	var request FoundationDelete
	err := r.pool.QueryRow(ctx, `SELECT e.id::text,n.id::text FROM host_nodes n JOIN runtime_cluster_nodes cn ON cn.node_id=n.id JOIN runtime_environments e ON e.tenant_id=n.tenant_id AND e.desired_status='deleting' AND e.deleted_at IS NULL WHERE n.id=$1 AND n.agent_token_hash=$2 AND n.desired_status='active' AND n.observed_status='online' AND n.approved_at IS NOT NULL AND cn.node_kind='center' AND cn.desired_action='active' AND cn.cluster_status='ready' AND EXISTS (SELECT 1 FROM runtime_environment_services s WHERE s.environment_id=e.id) ORDER BY e.updated_at,e.id LIMIT 1`, id, hash).Scan(&request.EnvironmentID, &request.NodeID)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, tokenErr := r.GetNodeByToken(ctx, id, hash); tokenErr != nil {
			return nil, ErrAgentUnauthorized
		}
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &request, nil
}
func (r *PostgreSQLRepository) AgentCommands(ctx context.Context, id, hash string) ([]AgentCommand, error) {
	n, e := r.GetNodeByToken(ctx, id, hash)
	if e != nil {
		return nil, ErrAgentUnauthorized
	}
	if n.ObservedStatus != "online" || n.DesiredStatus != "active" || n.ApprovedAt == nil {
		return []AgentCommand{}, nil
	}
	services, e := r.pendingServices(ctx, n.ID)
	if e != nil {
		return nil, e
	}
	out := make([]AgentCommand, 0, len(services))
	for _, s := range services {
		metadata, binding, err := r.agentDeploymentAccess(ctx, n.ID, hash, s.ProjectDeploymentID, s.ID)
		if err != nil {
			return nil, fmt.Errorf("读取节点制品命令失败: %w", err)
		}
		var run string
		if err = r.pool.QueryRow(ctx, `SELECT COALESCE((SELECT id::text FROM deployment_runs WHERE project_deployment_id=$1 AND observed_status='pending' ORDER BY started_at DESC LIMIT 1),'')`, s.ProjectDeploymentID).Scan(&run); err != nil {
			return nil, err
		}
		out = append(out, AgentCommand{NodeID: n.ID, RunID: run, DeploymentID: s.ProjectDeploymentID, ReleaseID: metadata.ID, ServiceID: s.ID, ServiceType: s.ServiceType, DesiredStatus: s.DesiredStatus, Operation: s.LastOperation, Generation: s.DesiredGeneration, Version: metadata.Version, ArchiveSHA256: sha256Value(metadata.ArtifactHash), ManifestSHA256: sha256Value(metadata.ManifestHash), ChecksumsSHA256: sha256Value(metadata.ChecksumsHash), SigningKeyID: metadata.SigningKeyID, BindingRevision: binding.Revision, ReplicasDesired: s.ReplicasDesired})
	}
	return out, nil
}
func (r *PostgreSQLRepository) GetNodeByToken(ctx context.Context, id, hash string) (Node, error) {
	x, e := scanNode(r.pool.QueryRow(ctx, `SELECT id,tenant_id,enrollment_id,display_name,hostname,platform,architecture,COALESCE(agent_version,''),COALESCE(machine_fingerprint,''),COALESCE(ip_address,''),desired_status,observed_status,capabilities,resource_summary,last_heartbeat_at,approved_at,created_at,updated_at FROM host_nodes WHERE id=$1 AND agent_token_hash=$2`, id, hash))
	return x, e
}
func (r *PostgreSQLRepository) GetAgentDeploymentBinding(ctx context.Context, nodeID, tokenHash, deploymentID, serviceID string) (DeploymentBinding, error) {
	access, binding, err := r.agentDeploymentAccess(ctx, nodeID, tokenHash, deploymentID, serviceID)
	if err != nil {
		return DeploymentBinding{}, err
	}
	return DeploymentBinding{ID: binding.ID, TenantID: binding.TenantID, ProjectDeploymentID: deploymentID, ProjectID: binding.ProjectID, NodeID: nodeID, ApplicationVersionID: access.ID, Revision: binding.Revision, Content: binding.Content}, nil
}
func (r *PostgreSQLRepository) GetAgentRelease(ctx context.Context, nodeID, tokenHash, deploymentID, serviceID string) (AgentRelease, error) {
	metadata, _, err := r.agentDeploymentAccess(ctx, nodeID, tokenHash, deploymentID, serviceID)
	if err != nil {
		return AgentRelease{}, err
	}
	return AgentRelease{ReleaseID: metadata.ID, ArtifactKey: metadata.ArtifactKey, ArtifactHash: metadata.ArtifactHash, ManifestHash: metadata.ManifestHash, ChecksumsHash: metadata.ChecksumsHash, SigningKeyID: metadata.SigningKeyID, ArtifactSize: metadata.ArtifactSize}, nil
}

type bindingMetadata struct {
	ID, TenantID, ProjectID string
	Revision                int
	Content                 []byte
}

// agentDeploymentAccess 将节点令牌、节点状态、Deployment、Release 和 Binding 放进同一查询。
// 任何关联缺失都按未授权处理，不能让节点探测其他节点或工程的 Release。
func (r *PostgreSQLRepository) agentDeploymentAccess(ctx context.Context, nodeID, tokenHash, deploymentID, serviceID string) (releaseMetadata, bindingMetadata, error) {
	var metadata releaseMetadata
	var binding bindingMetadata
	var mode string
	var descriptor []byte
	err := r.pool.QueryRow(ctx, `SELECT d.mode,COALESCE(v.id::text,''),COALESCE(v.version,''),COALESCE(v.artifact_key,''),COALESCE(v.artifact_hash,''),COALESCE(v.manifest_hash,''),COALESCE(v.checksums_hash,''),COALESCE(v.signing_key_id,''),COALESCE(v.artifact_size,0),COALESCE(v.manifest,'{}'::jsonb),b.artifact_descriptor,b.id::text,b.tenant_id::text,b.project_id::text,b.revision,b.binding FROM host_nodes n JOIN deployment_services s ON s.node_id=n.id JOIN project_deployments d ON d.id=s.project_deployment_id AND d.tenant_id=n.tenant_id LEFT JOIN application_versions v ON v.id=d.application_version_id AND v.tenant_id=d.tenant_id JOIN LATERAL (SELECT id,tenant_id,project_id,revision,binding,artifact_descriptor FROM deployment_bindings WHERE deployment_service_id=s.id AND node_id=n.id AND artifact_mode=CASE WHEN d.mode='development' THEN 'development' ELSE 'release' END ORDER BY revision DESC LIMIT 1) b ON true WHERE n.id=$1 AND n.agent_token_hash=$2 AND n.desired_status='active' AND n.observed_status='online' AND n.last_heartbeat_at>now()-interval '45 seconds' AND n.approved_at IS NOT NULL AND d.id=$3 AND s.id=$4 AND d.desired_status='running' AND (d.mode='development' OR (v.status='ready' AND v.deleted_at IS NULL))`, nodeID, tokenHash, deploymentID, serviceID).Scan(&mode, &metadata.ID, &metadata.Version, &metadata.ArtifactKey, &metadata.ArtifactHash, &metadata.ManifestHash, &metadata.ChecksumsHash, &metadata.SigningKeyID, &metadata.ArtifactSize, &metadata.Manifest, &descriptor, &binding.ID, &binding.TenantID, &binding.ProjectID, &binding.Revision, &binding.Content)
	if errors.Is(err, pgx.ErrNoRows) {
		return releaseMetadata{}, bindingMetadata{}, ErrAgentUnauthorized
	}
	if err != nil {
		return releaseMetadata{}, bindingMetadata{}, err
	}
	if mode == "development" {
		var artifact DevelopmentArtifact
		if err := json.Unmarshal(descriptor, &artifact); err != nil {
			return releaseMetadata{}, bindingMetadata{}, fmt.Errorf("%w: 开发制品描述损坏", ErrAgentUnauthorized)
		}
		metadata, err = developmentReleaseMetadata(&artifact, binding.ProjectID)
	} else {
		err = validateReleaseMetadata(metadata, binding.ProjectID, false)
	}
	if err != nil {
		return releaseMetadata{}, bindingMetadata{}, err
	}
	if err := validateInitialDeploymentBinding(binding.Content, binding, metadata, nodeID, deploymentID); err != nil {
		return releaseMetadata{}, bindingMetadata{}, err
	}
	return metadata, binding, nil
}

func (r *PostgreSQLRepository) ListDeployments(ctx context.Context, tenant string, f PageFilter) ([]ProjectDeployment, int64, error) {
	rows, e := r.pool.Query(ctx, deploymentSelect+` WHERE d.tenant_id=$1 AND ($2='' OR p.name ILIKE '%'||$2||'%') AND ($5='' OR d.project_id::text=$5) ORDER BY d.updated_at DESC,d.id DESC LIMIT $3 OFFSET $4`, tenant, f.Search, f.PageSize, (f.Page-1)*f.PageSize, f.ProjectID)
	if e != nil {
		return nil, 0, e
	}
	defer rows.Close()
	out := []ProjectDeployment{}
	for rows.Next() {
		d, e := scanDeployment(rows)
		if e != nil {
			return nil, 0, e
		}
		d.Services, _ = r.listServices(ctx, d.ID)
		out = append(out, d)
	}
	var total int64
	e = r.pool.QueryRow(ctx, `SELECT count(*) FROM project_deployments d JOIN projects p ON p.id=d.project_id AND p.tenant_id=d.tenant_id WHERE d.tenant_id=$1 AND ($2='' OR p.name ILIKE '%'||$2||'%') AND ($3='' OR d.project_id::text=$3)`, tenant, f.Search, f.ProjectID).Scan(&total)
	return out, total, e
}
func (r *PostgreSQLRepository) ValidateDeploymentTargets(ctx context.Context, tenant string, in CreateDeploymentInput) error {
	metadata, err := deploymentMetadata(ctx, r.pool, tenant, in)
	if err != nil {
		return err
	}
	required, err := deploymentRequirementsForRelease(metadata, in.ProjectID)
	if err != nil {
		return err
	}
	if err = validateEnginePlacements(required, in.Placements); err != nil {
		return err
	}
	for _, engine := range required {
		var ready bool
		err = r.pool.QueryRow(ctx, `SELECT n.approved_at IS NOT NULL AND n.desired_status='active' AND n.observed_status='online' AND n.last_heartbeat_at>now()-interval '45 seconds' FROM host_nodes n JOIN runtime_environment_nodes en ON en.node_id=n.id WHERE n.tenant_id=$1 AND en.environment_id=$2 AND n.id=$3`, tenant, in.EnvironmentID, in.Placements[engine]).Scan(&ready)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if !ready {
			return fmt.Errorf("%s 引擎节点当前不能调度", engine)
		}
	}
	var conflict string
	err = r.pool.QueryRow(ctx, `SELECT s.project_deployment_id::text FROM deployment_services s JOIN project_deployments d ON d.id=s.project_deployment_id WHERE d.tenant_id=$1 AND s.service_type='base' AND s.node_id=$2 AND s.public_port=$3 AND NOT (d.project_id=$4 AND d.environment_id=$5) LIMIT 1`, tenant, in.Placements[ServiceBase], in.AccessPort, in.ProjectID, in.EnvironmentID).Scan(&conflict)
	if err == nil {
		return ErrNodePortConflict
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	return nil
}
func (r *PostgreSQLRepository) CreateDeployment(ctx context.Context, tenant, user string, in CreateDeploymentInput) (ProjectDeployment, DeploymentRun, error) {
	tx, e := r.pool.Begin(ctx)
	if e != nil {
		return ProjectDeployment{}, DeploymentRun{}, e
	}
	defer tx.Rollback(ctx)
	if e = tx.QueryRow(ctx, lockDeploymentProjectSQL, tenant, in.ProjectID).Scan(new(string)); e != nil {
		return ProjectDeployment{}, DeploymentRun{}, mapNotFound(e)
	}
	metadata, err := deploymentMetadata(ctx, tx, tenant, in)
	if err != nil {
		return ProjectDeployment{}, DeploymentRun{}, err
	}
	required, err := deploymentRequirementsForRelease(metadata, in.ProjectID)
	if err != nil {
		return ProjectDeployment{}, DeploymentRun{}, err
	}
	if err = validateEnginePlacements(required, in.Placements); err != nil {
		return ProjectDeployment{}, DeploymentRun{}, err
	}
	for _, engine := range required {
		var ready bool
		err = tx.QueryRow(ctx, `SELECT n.approved_at IS NOT NULL AND n.desired_status='active' AND n.observed_status='online' AND n.last_heartbeat_at>now()-interval '45 seconds' FROM host_nodes n JOIN runtime_environment_nodes en ON en.node_id=n.id WHERE n.tenant_id=$1 AND en.environment_id=$2 AND n.id=$3 FOR UPDATE OF n`, tenant, in.EnvironmentID, in.Placements[engine]).Scan(&ready)
		if errors.Is(err, pgx.ErrNoRows) {
			return ProjectDeployment{}, DeploymentRun{}, ErrNotFound
		}
		if err != nil {
			return ProjectDeployment{}, DeploymentRun{}, err
		}
		if !ready {
			return ProjectDeployment{}, DeploymentRun{}, fmt.Errorf("%s 引擎节点当前不能调度", engine)
		}
	}
	descriptor, err := deploymentArtifactDescriptor(in)
	if err != nil {
		return ProjectDeployment{}, DeploymentRun{}, err
	}
	var d ProjectDeployment
	d, e = scanDeployment(tx.QueryRow(ctx, `INSERT INTO project_deployments(tenant_id,project_id,environment_id,application_version_id,mode,artifact_descriptor,access_port,created_by) VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5,$6,$7,$8) ON CONFLICT (tenant_id,project_id,environment_id) DO UPDATE SET application_version_id=EXCLUDED.application_version_id,mode=EXCLUDED.mode,artifact_descriptor=EXCLUDED.artifact_descriptor,access_port=EXCLUDED.access_port,desired_status='running',observed_status='pending',updated_at=now() RETURNING id,tenant_id,project_id,'',$3,'',COALESCE(application_version_id::text,''),'',$5,$7,desired_status,observed_status,0,COALESCE(last_ready_mode,''),COALESCE(last_ready_application_version_id::text,''),'',COALESCE(last_ready_generation,0),last_ready_at,created_at,updated_at`, tenant, in.ProjectID, in.EnvironmentID, in.ApplicationVersionID, in.Mode, descriptor, in.AccessPort, user))
	if e != nil {
		return d, DeploymentRun{}, mapDeploymentCreateError(e)
	}
	// 同类型同节点更新保留 service_id，K3s 资源名稳定，从而由同一 Deployment
	// 模板变更触发滚动更新。跨节点迁移需要先由旧节点完成 stop，不能覆盖节点列。
	rows, queryErr := tx.Query(ctx, `SELECT id::text,service_type,node_id::text FROM deployment_services WHERE project_deployment_id=$1 FOR UPDATE`, d.ID)
	if queryErr != nil {
		return d, DeploymentRun{}, queryErr
	}
	existing := map[string]struct{ id, node string }{}
	for rows.Next() {
		var id, kind, node string
		if queryErr = rows.Scan(&id, &kind, &node); queryErr != nil {
			rows.Close()
			return d, DeploymentRun{}, queryErr
		}
		existing[kind] = struct{ id, node string }{id, node}
	}
	rows.Close()
	if queryErr = rows.Err(); queryErr != nil {
		return d, DeploymentRun{}, queryErr
	}
	for _, kind := range required {
		var publicPort any
		if kind == ServiceBase {
			publicPort = in.AccessPort
		}
		var serviceID string
		if old, exists := existing[kind]; exists {
			if old.node != in.Placements[kind] {
				return d, DeploymentRun{}, fmt.Errorf("%s 引擎节点迁移必须先停止旧工作负载", kind)
			}
			serviceID = old.id
			e = tx.QueryRow(ctx, `UPDATE deployment_services SET public_port=$1,desired_status='running',observed_status='pending',desired_generation=desired_generation+1,last_operation='deploy',updated_at=now() WHERE id=$2 RETURNING id::text`, publicPort, serviceID).Scan(&serviceID)
		} else {
			e = tx.QueryRow(ctx, `INSERT INTO deployment_services(tenant_id,project_deployment_id,node_id,service_type,public_port) VALUES($1,$2,$3,$4,$5) RETURNING id::text`, tenant, d.ID, in.Placements[kind], kind, publicPort).Scan(&serviceID)
		}
		if e != nil {
			return d, DeploymentRun{}, e
		}
		var revision int
		_ = tx.QueryRow(ctx, `SELECT COALESCE(max(revision),0)+1 FROM deployment_bindings WHERE deployment_service_id=$1`, serviceID).Scan(&revision)
		bindingID, bindingJSON, bindErr := newEngineDeploymentBinding(d, metadata, serviceID, in.Placements[kind], kind, revision, publicPort)
		if bindErr != nil {
			return d, DeploymentRun{}, bindErr
		}
		if _, e = tx.Exec(ctx, `INSERT INTO deployment_bindings(id,tenant_id,project_deployment_id,deployment_service_id,project_id,node_id,application_version_id,artifact_mode,artifact_descriptor,revision,binding) VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,'')::uuid,$8,$9,$10,$11)`, bindingID, tenant, d.ID, serviceID, in.ProjectID, in.Placements[kind], in.ApplicationVersionID, artifactMode(in.Mode), descriptor, revision, bindingJSON); e != nil {
			return d, DeploymentRun{}, e
		}
	}
	requiredSet := map[string]bool{}
	for _, kind := range required {
		requiredSet[kind] = true
	}
	for kind := range existing {
		if !requiredSet[kind] {
			if _, e = tx.Exec(ctx, `UPDATE deployment_services SET desired_status='stopped',observed_status='pending',desired_generation=desired_generation+1,last_operation='stop',updated_at=now() WHERE project_deployment_id=$1 AND service_type=$2`, d.ID, kind); e != nil {
				return d, DeploymentRun{}, e
			}
		}
	}
	var run DeploymentRun
	e = tx.QueryRow(ctx, `INSERT INTO deployment_runs(tenant_id,project_deployment_id,operation,desired_status,created_by) VALUES($1,$2,'deploy','running',$3) RETURNING id,tenant_id,project_deployment_id,operation,desired_status,observed_status,progress,COALESCE(message,''),started_at,completed_at`, tenant, d.ID, user).Scan(runScanArgs(&run)...)
	if e != nil {
		return d, run, e
	}
	_, e = tx.Exec(ctx, `INSERT INTO deployment_run_events(deployment_run_id,stage,message) VALUES($1,'queued','deployment queued')`, run.ID)
	if e != nil {
		return d, run, e
	}
	if e = tx.Commit(ctx); e != nil {
		return d, run, e
	}
	d, e = r.GetDeployment(ctx, tenant, d.ID)
	return d, run, e
}
func (r *PostgreSQLRepository) GetDeployment(ctx context.Context, tenant, id string) (ProjectDeployment, error) {
	d, e := scanDeployment(r.pool.QueryRow(ctx, deploymentSelect+` WHERE d.tenant_id=$1 AND d.id=$2`, tenant, id))
	if e != nil {
		return d, mapNotFound(e)
	}
	d.Services, e = r.listServices(ctx, d.ID)
	return d, e
}
func (r *PostgreSQLRepository) GetRun(ctx context.Context, tenant, id string) (DeploymentRun, error) {
	x := DeploymentRun{}
	e := r.pool.QueryRow(ctx, `SELECT id,tenant_id,project_deployment_id,operation,desired_status,observed_status,progress,COALESCE(message,''),started_at,completed_at FROM deployment_runs WHERE tenant_id=$1 AND id=$2`, tenant, id).Scan(runScanArgs(&x)...)
	return x, mapNotFound(e)
}
func (r *PostgreSQLRepository) ListRunEvents(ctx context.Context, tenant, id string) ([]DeploymentRunEvent, error) {
	rows, e := r.pool.Query(ctx, `SELECT e.id,e.deployment_run_id,e.stage,e.message,e.created_at FROM deployment_run_events e JOIN deployment_runs r ON r.id=e.deployment_run_id WHERE r.tenant_id=$1 AND e.deployment_run_id=$2 ORDER BY e.created_at`, tenant, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []DeploymentRunEvent{}
	for rows.Next() {
		var x DeploymentRunEvent
		if e = rows.Scan(&x.ID, &x.DeploymentRunID, &x.Stage, &x.Message, &x.CreatedAt); e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *PostgreSQLRepository) OperateService(ctx context.Context, tenant, did, kind, op, user string) (ProjectDeployment, DeploymentRun, error) {
	desired := "running"
	if op == "stop" {
		desired = "stopped"
	}
	tx, e := r.pool.Begin(ctx)
	if e != nil {
		return ProjectDeployment{}, DeploymentRun{}, e
	}
	defer tx.Rollback(ctx)
	var id string
	if e = tx.QueryRow(ctx, `SELECT id FROM project_deployments WHERE id=$1 AND tenant_id=$2 FOR UPDATE`, did, tenant).Scan(&id); e != nil {
		return ProjectDeployment{}, DeploymentRun{}, mapNotFound(e)
	}
	var pending string
	e = tx.QueryRow(ctx, `SELECT id FROM deployment_runs WHERE project_deployment_id=$1 AND observed_status='pending'`, did).Scan(&pending)
	if e == nil {
		return ProjectDeployment{}, DeploymentRun{}, ErrDeploymentBusy
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return ProjectDeployment{}, DeploymentRun{}, e
	}
	tag, e := tx.Exec(ctx, `UPDATE deployment_services SET desired_status=$1,observed_status='pending',desired_generation=desired_generation+1,last_operation=$2,updated_at=now() WHERE project_deployment_id=$3 AND service_type=$4 AND tenant_id=$5`, desired, op, did, kind, tenant)
	if e != nil || tag.RowsAffected() == 0 {
		return ProjectDeployment{}, DeploymentRun{}, ErrNotFound
	}
	_, e = tx.Exec(ctx, `UPDATE project_deployments SET observed_status='pending',updated_at=now() WHERE id=$1`, did)
	if e != nil {
		return ProjectDeployment{}, DeploymentRun{}, e
	}
	var run DeploymentRun
	e = tx.QueryRow(ctx, `INSERT INTO deployment_runs(tenant_id,project_deployment_id,operation,desired_status,created_by) VALUES($1,$2,$3,$4,$5) RETURNING id,tenant_id,project_deployment_id,operation,desired_status,observed_status,progress,COALESCE(message,''),started_at,completed_at`, tenant, did, op, desired, user).Scan(runScanArgs(&run)...)
	if e != nil {
		return ProjectDeployment{}, run, e
	}
	_, e = tx.Exec(ctx, `INSERT INTO deployment_run_events(deployment_run_id,stage,message) VALUES($1,'queued',$2)`, run.ID, op+" queued")
	if e != nil {
		return ProjectDeployment{}, run, e
	}
	if e = tx.Commit(ctx); e != nil {
		return ProjectDeployment{}, run, e
	}
	d, e := r.GetDeployment(ctx, tenant, did)
	return d, run, e
}

// OperateDeployment 在单一事务中变更同一 Deployment 的所有服务，并只创建一条
// DeploymentRun。这样 Agent 收到的是一次完整工程生命周期变更，而非局部服务操作。
func (r *PostgreSQLRepository) OperateDeployment(ctx context.Context, tenant, did, op, user string) (ProjectDeployment, DeploymentRun, error) {
	desired := "running"
	if op == "stop" {
		desired = "stopped"
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ProjectDeployment{}, DeploymentRun{}, err
	}
	defer tx.Rollback(ctx)
	var id string
	if err = tx.QueryRow(ctx, `SELECT id FROM project_deployments WHERE id=$1 AND tenant_id=$2 FOR UPDATE`, did, tenant).Scan(&id); err != nil {
		return ProjectDeployment{}, DeploymentRun{}, mapNotFound(err)
	}
	var pending string
	err = tx.QueryRow(ctx, `SELECT id FROM deployment_runs WHERE project_deployment_id=$1 AND observed_status='pending'`, did).Scan(&pending)
	if err == nil {
		return ProjectDeployment{}, DeploymentRun{}, ErrDeploymentBusy
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return ProjectDeployment{}, DeploymentRun{}, err
	}
	// 不按 service_type 过滤：start/stop/restart 必须同时推进工程的全部服务代次。
	tag, err := tx.Exec(ctx, `UPDATE deployment_services SET desired_status=$1,observed_status='pending',desired_generation=desired_generation+1,last_operation=$2,updated_at=now() WHERE project_deployment_id=$3 AND tenant_id=$4`, desired, op, did, tenant)
	if err != nil {
		return ProjectDeployment{}, DeploymentRun{}, err
	}
	if tag.RowsAffected() == 0 {
		return ProjectDeployment{}, DeploymentRun{}, ErrNotFound
	}
	if _, err = tx.Exec(ctx, `UPDATE project_deployments SET desired_status=$1,observed_status='pending',updated_at=now() WHERE id=$2 AND tenant_id=$3`, desired, did, tenant); err != nil {
		return ProjectDeployment{}, DeploymentRun{}, err
	}
	var run DeploymentRun
	err = tx.QueryRow(ctx, `INSERT INTO deployment_runs(tenant_id,project_deployment_id,operation,desired_status,created_by) VALUES($1,$2,$3,$4,$5) RETURNING id,tenant_id,project_deployment_id,operation,desired_status,observed_status,progress,COALESCE(message,''),started_at,completed_at`, tenant, did, op, desired, user).Scan(runScanArgs(&run)...)
	if err != nil {
		return ProjectDeployment{}, run, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO deployment_run_events(deployment_run_id,stage,message) VALUES($1,'queued',$2)`, run.ID, op+" deployment queued"); err != nil {
		return ProjectDeployment{}, run, err
	}
	if err = tx.Commit(ctx); err != nil {
		return ProjectDeployment{}, run, err
	}
	d, err := r.GetDeployment(ctx, tenant, did)
	return d, run, err
}

func (r *PostgreSQLRepository) pendingServices(ctx context.Context, nodeID string) ([]DeploymentService, error) {
	rows, e := r.pool.Query(ctx, pendingServicesSQL, nodeID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []DeploymentService{}
	for rows.Next() {
		var s DeploymentService
		if e = rows.Scan(serviceScanArgs(&s)...); e != nil {
			return nil, e
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
func (r *PostgreSQLRepository) listServices(ctx context.Context, did string) ([]DeploymentService, error) {
	rows, e := r.pool.Query(ctx, `SELECT id,tenant_id,project_deployment_id,node_id,service_type,public_port,desired_status,observed_status,COALESCE(last_message,''),COALESCE(endpoint,''),replicas_desired,replicas_observed,desired_generation,observed_generation,last_operation,observed_at,created_at,updated_at FROM deployment_services WHERE project_deployment_id=$1 ORDER BY service_type`, did)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []DeploymentService{}
	for rows.Next() {
		var s DeploymentService
		if e = rows.Scan(serviceScanArgs(&s)...); e != nil {
			return nil, e
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
func (r *PostgreSQLRepository) reconcileDeployment(ctx context.Context, did string) error {
	var desired string
	var pending, failed int
	e := r.pool.QueryRow(ctx, `SELECT d.desired_status,count(*) FILTER (WHERE s.observed_generation<>s.desired_generation OR s.observed_status<>s.desired_status),count(*) FILTER (WHERE s.observed_status='failed') FROM project_deployments d JOIN deployment_services s ON s.project_deployment_id=d.id WHERE d.id=$1 GROUP BY d.desired_status`, did).Scan(&desired, &pending, &failed)
	if e != nil {
		return e
	}
	observed := desired
	if failed > 0 {
		observed = "failed"
	} else if pending > 0 {
		observed = "pending"
	}
	_, e = r.pool.Exec(ctx, `UPDATE project_deployments SET observed_status=$1,updated_at=now() WHERE id=$2`, observed, did)
	if e != nil || observed == "pending" {
		return e
	}
	var run string
	e = r.pool.QueryRow(ctx, `WITH latest AS (SELECT id FROM deployment_runs WHERE project_deployment_id=$1 AND observed_status='pending' ORDER BY started_at DESC LIMIT 1) UPDATE deployment_runs SET observed_status=$2,progress=100,completed_at=now() WHERE id=(SELECT id FROM latest) RETURNING id`, did, observed).Scan(&run)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	return e
}
func (r *PostgreSQLRepository) attachEnrollmentNode(ctx context.Context, en *Enrollment) {
	if en.ClaimedByNodeID == "" {
		return
	}
	n, e := r.GetNode(ctx, en.TenantID, en.ClaimedByNodeID)
	if e != nil {
		return
	}
	en.ReportedHostName, en.MachineFingerprint, en.IPAddress = n.Hostname, n.MachineFingerprint, n.IPAddress
	en.Node = &n
}

type scanner interface{ Scan(...any) error }

func scanEnrollment(s scanner) (Enrollment, error) {
	var x Enrollment
	var caps []byte
	e := s.Scan(&x.ID, &x.TenantID, &x.Platform, &caps, &x.DisplayName, &x.Status, &x.ExpiresAt, &x.ClaimedAt, &x.ClaimedByNodeID, &x.ApprovedAt, &x.RejectedAt, &x.CreatedAt, &x.UpdatedAt)
	_ = json.Unmarshal(caps, &x.Capabilities)
	return x, e
}
func scanNode(s scanner) (Node, error) {
	var x Node
	var caps, summary []byte
	e := s.Scan(nodeScanArgs(&x, &caps, &summary)...)
	hydrateNode(&x, caps, summary)
	return x, e
}
func nodeScanArgs(x *Node, caps, summary *[]byte) []any {
	return []any{&x.ID, &x.TenantID, &x.EnrollmentID, &x.DisplayName, &x.Hostname, &x.Platform, &x.Architecture, &x.AgentVersion, &x.MachineFingerprint, &x.IPAddress, &x.DesiredStatus, &x.ObservedStatus, caps, summary, &x.LastHeartbeatAt, &x.ApprovedAt, &x.CreatedAt, &x.UpdatedAt}
}
func hydrateNode(x *Node, caps, summary []byte) {
	_ = json.Unmarshal(caps, &x.Capabilities)
	_ = json.Unmarshal(summary, &x.ResourceSummary)
	if x.ResourceSummary == nil {
		x.ResourceSummary = map[string]any{}
	}
}
func scanDeployment(s scanner) (ProjectDeployment, error) {
	var x ProjectDeployment
	e := s.Scan(&x.ID, &x.TenantID, &x.ProjectID, &x.ProjectName, &x.EnvironmentID, &x.EnvironmentName, &x.ApplicationVersionID, &x.Version, &x.LatestRunID, &x.Mode, &x.AccessPort, &x.DesiredStatus, &x.ObservedStatus, &x.Progress, &x.LastReadyMode, &x.LastReadyApplicationVersionID, &x.LastReadyVersion, &x.LastReadyGeneration, &x.LastReadyAt, &x.CreatedAt, &x.UpdatedAt)
	x.Health = normalizeHealth(x.ObservedStatus)
	return x, e
}
func serviceScanArgs(x *DeploymentService) []any {
	return []any{&x.ID, &x.TenantID, &x.ProjectDeploymentID, &x.NodeID, &x.ServiceType, &x.PublicPort, &x.DesiredStatus, &x.ObservedStatus, &x.LastMessage, &x.Endpoint, &x.ReplicasDesired, &x.ReplicasObserved, &x.DesiredGeneration, &x.ObservedGeneration, &x.LastOperation, &x.ObservedAt, &x.CreatedAt, &x.UpdatedAt}
}
func runScanArgs(x *DeploymentRun) []any {
	return []any{&x.ID, &x.TenantID, &x.ProjectDeploymentID, &x.Operation, &x.DesiredStatus, &x.ObservedStatus, &x.Progress, &x.Message, &x.StartedAt, &x.CompletedAt}
}
func containsAll(actual, required []string) bool {
	set := map[string]bool{}
	for _, v := range actual {
		set[strings.TrimSpace(v)] = true
	}
	for _, v := range required {
		if !set[v] {
			return false
		}
	}
	return true
}

func sameStrings(actual, expected []string) bool {
	return len(actual) == len(expected) && containsAll(actual, expected) && containsAll(expected, actual)
}

type releaseMetadataReader interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func loadReleaseMetadata(ctx context.Context, reader releaseMetadataReader, tenant, projectID, releaseID string) (releaseMetadata, error) {
	metadata := releaseMetadata{ID: releaseID}
	var status string
	err := reader.QueryRow(ctx, `SELECT status,version,COALESCE(artifact_key,''),COALESCE(artifact_hash,''),COALESCE(manifest_hash,''),COALESCE(checksums_hash,''),COALESCE(signing_key_id,''),COALESCE(artifact_size,0),manifest FROM application_versions WHERE id=$1 AND project_id=$2 AND tenant_id=$3 AND deleted_at IS NULL`, releaseID, projectID, tenant).Scan(&status, &metadata.Version, &metadata.ArtifactKey, &metadata.ArtifactHash, &metadata.ManifestHash, &metadata.ChecksumsHash, &metadata.SigningKeyID, &metadata.ArtifactSize, &metadata.Manifest)
	if errors.Is(err, pgx.ErrNoRows) {
		return releaseMetadata{}, ErrNotFound
	}
	if err != nil {
		return releaseMetadata{}, err
	}
	if status != "ready" {
		return releaseMetadata{}, fmt.Errorf("%w: 版本尚未构建成功", ErrReleaseNotDeployable)
	}
	return metadata, nil
}

func developmentReleaseMetadata(artifact *DevelopmentArtifact, projectID string) (releaseMetadata, error) {
	if artifact == nil || artifact.Version != "__DEV__" || !validUUID(artifact.ReleaseID) ||
		!strings.HasPrefix(artifact.ArtifactKey, "development/") {
		return releaseMetadata{}, fmt.Errorf("%w: 开发制品描述无效", ErrReleaseNotDeployable)
	}
	metadata := releaseMetadata{ID: artifact.ReleaseID, Version: "__DEV__", ArtifactKey: artifact.ArtifactKey, ArtifactHash: artifact.ArtifactHash,
		ManifestHash: artifact.ManifestHash, ChecksumsHash: artifact.ChecksumsHash, SigningKeyID: artifact.SigningKeyID,
		ArtifactSize: artifact.ArtifactSize, Manifest: artifact.Manifest}
	if _, err := deploymentRequirementsForRelease(metadata, projectID); err != nil {
		return releaseMetadata{}, err
	}
	return metadata, nil
}

func deploymentMetadata(ctx context.Context, reader releaseMetadataReader, tenant string, in CreateDeploymentInput) (releaseMetadata, error) {
	if in.Mode == "development" {
		return developmentReleaseMetadata(in.DevelopmentArtifact, in.ProjectID)
	}
	return loadReleaseMetadata(ctx, reader, tenant, in.ProjectID, in.ApplicationVersionID)
}

func artifactMode(mode string) string {
	if mode == "development" {
		return "development"
	}
	return "release"
}

func deploymentArtifactDescriptor(in CreateDeploymentInput) ([]byte, error) {
	if in.Mode != "development" {
		return []byte(`{}`), nil
	}
	if _, err := developmentReleaseMetadata(in.DevelopmentArtifact, in.ProjectID); err != nil {
		return nil, err
	}
	return json.Marshal(in.DevelopmentArtifact)
}

func validateReleaseMetadata(metadata releaseMetadata, projectID string, requireCollector bool) error {
	if metadata.ArtifactSize <= 0 {
		return fmt.Errorf("%w: Release 归档大小缺失或无效", ErrReleaseNotDeployable)
	}
	return validateDeployableReleaseForDeployment(projectID, metadata.ID, metadata.ArtifactKey, metadata.ArtifactHash, metadata.ManifestHash, metadata.ChecksumsHash, metadata.SigningKeyID, requireCollector, metadata.Manifest)
}

// deploymentRequirementsForRelease 只信任不可变、已签名 Release manifest 推导引擎。
// 采集引擎被工程内容要求时，必须同时存在受信采集工件；请求 placements 不能改变此结论。
func deploymentRequirementsForRelease(metadata releaseMetadata, projectID string) ([]string, error) {
	required, err := deploymentEngineRequirements(metadata.Manifest)
	if err != nil {
		return nil, err
	}
	requireCollector := false
	for _, engine := range required {
		if engine == ServiceCollector {
			requireCollector = true
			break
		}
	}
	if err = validateReleaseMetadata(metadata, projectID, requireCollector); err != nil {
		return nil, err
	}
	return required, nil
}

func newInitialDeploymentBinding(deployment ProjectDeployment, release releaseMetadata, nodeID string, services []string) (string, []byte, error) {
	bindingID := uuid.NewString()
	ports := map[string]int{"gatewayPublic": deployment.AccessPort, "runtimeApiLoopback": deployment.AccessPort + 1, "engineLoopback": deployment.AccessPort + 2}
	for _, service := range services {
		if service == ServiceCollector {
			ports["collectorHealthLoopback"] = deployment.AccessPort + 3
		}
	}
	// Secret 装配尚未交付时只能显式下发空引用，不能把临时凭据或节点参数写进 Release。
	binding := map[string]any{
		"schemaVersion": "deployment-binding.v1",
		"bindingId":     bindingID,
		"revision":      1,
		"nodeId":        nodeID,
		"deploymentId":  deployment.ID,
		"projectId":     deployment.ProjectID,
		"release": map[string]any{
			"id":              release.ID,
			"archiveSha256":   sha256Value(release.ArtifactHash),
			"manifestSha256":  sha256Value(release.ManifestHash),
			"checksumsSha256": sha256Value(release.ChecksumsHash),
			"signingKeyId":    release.SigningKeyID,
		},
		"enabledServices": services,
		"ports":           ports,
		"secrets":         []any{},
		"issuedAt":        time.Now().UTC().Format(time.RFC3339),
	}
	content, err := json.Marshal(binding)
	if err != nil {
		return "", nil, err
	}
	return bindingID, content, nil
}

// newEngineDeploymentBinding 为每个工作负载生成节点专用快照。基础引擎才携带 hostPort，
// 其他引擎只使用集群内部端口，避免把旧的连续四端口模型泄漏到调度层。
func newEngineDeploymentBinding(deployment ProjectDeployment, release releaseMetadata, serviceID, nodeID, serviceType string, revision int, publicPort any) (string, []byte, error) {
	bindingID := uuid.NewString()
	binding := map[string]any{
		"schemaVersion": "deployment-binding.v2", "bindingId": bindingID, "revision": revision,
		"nodeId": nodeID, "deploymentId": deployment.ID, "serviceId": serviceID,
		"projectId": deployment.ProjectID, "environmentId": deployment.EnvironmentID,
		"mode": deployment.Mode, "engine": serviceType,
		"release": map[string]any{"id": release.ID, "archiveSha256": sha256Value(release.ArtifactHash), "manifestSha256": sha256Value(release.ManifestHash), "checksumsSha256": sha256Value(release.ChecksumsHash), "signingKeyId": release.SigningKeyID},
		"ports":   map[string]any{"hostPort": publicPort}, "secrets": []any{}, "issuedAt": time.Now().UTC().Format(time.RFC3339),
	}
	content, err := json.Marshal(binding)
	return bindingID, content, err
}

type deploymentBindingDocument struct {
	SchemaVersion   string   `json:"schemaVersion"`
	BindingID       string   `json:"bindingId"`
	Revision        int      `json:"revision"`
	NodeID          string   `json:"nodeId"`
	DeploymentID    string   `json:"deploymentId"`
	ProjectID       string   `json:"projectId"`
	EnabledServices []string `json:"enabledServices"`
	Ports           struct {
		GatewayPublic           int  `json:"gatewayPublic"`
		RuntimeAPILoopback      int  `json:"runtimeApiLoopback"`
		EngineLoopback          int  `json:"engineLoopback"`
		CollectorHealthLoopback *int `json:"collectorHealthLoopback,omitempty"`
	} `json:"ports"`
	Release struct {
		ID              string `json:"id"`
		ArchiveSHA256   string `json:"archiveSha256"`
		ManifestSHA256  string `json:"manifestSha256"`
		ChecksumsSHA256 string `json:"checksumsSha256"`
		SigningKeyID    string `json:"signingKeyId"`
	} `json:"release"`
	Secrets  []any  `json:"secrets"`
	IssuedAt string `json:"issuedAt"`
}

func validateInitialDeploymentBinding(raw []byte, binding bindingMetadata, release releaseMetadata, nodeID, deploymentID string) error {
	var header struct {
		SchemaVersion string `json:"schemaVersion"`
	}
	if err := json.Unmarshal(raw, &header); err != nil {
		return fmt.Errorf("%w: DeploymentBinding 内容无效", ErrReleaseNotDeployable)
	}
	if header.SchemaVersion == "deployment-binding.v2" {
		return validateEngineDeploymentBinding(raw, binding, release, nodeID, deploymentID)
	}
	var document deploymentBindingDocument
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return fmt.Errorf("%w: DeploymentBinding 内容无效", ErrReleaseNotDeployable)
	}
	if document.SchemaVersion != "deployment-binding.v1" || document.BindingID != binding.ID || document.Revision != binding.Revision || document.NodeID != nodeID || document.DeploymentID != deploymentID || document.ProjectID != binding.ProjectID || document.Release.ID != release.ID || document.Release.ArchiveSHA256 != sha256Value(release.ArtifactHash) || document.Release.ManifestSHA256 != sha256Value(release.ManifestHash) || document.Release.ChecksumsSHA256 != sha256Value(release.ChecksumsHash) || document.Release.SigningKeyID != release.SigningKeyID {
		return fmt.Errorf("%w: DeploymentBinding 与节点或 Release 不匹配", ErrReleaseNotDeployable)
	}
	if document.Ports.GatewayPublic < 1024 || document.Ports.GatewayPublic > 65532 || document.Ports.RuntimeAPILoopback != document.Ports.GatewayPublic+1 || document.Ports.EngineLoopback != document.Ports.GatewayPublic+2 || len(document.Secrets) != 0 {
		return fmt.Errorf("%w: DeploymentBinding 违反当前单节点安全基线", ErrReleaseNotDeployable)
	}
	hasEntry, hasRuntime, hasCollector := false, false, false
	seenServices := map[string]bool{}
	for _, service := range document.EnabledServices {
		if seenServices[service] {
			return fmt.Errorf("%w: DeploymentBinding 服务类型重复", ErrReleaseNotDeployable)
		}
		seenServices[service] = true
		switch service {
		case ServiceBase:
			hasEntry = true
		case ServiceCompute:
			hasRuntime = true
		case ServiceCollector:
			hasCollector = true
		default:
			return fmt.Errorf("%w: DeploymentBinding 服务类型无效", ErrReleaseNotDeployable)
		}
	}
	if len(document.EnabledServices) < 2 || len(document.EnabledServices) > 3 || !hasEntry || !hasRuntime || (hasCollector && (document.Ports.CollectorHealthLoopback == nil || *document.Ports.CollectorHealthLoopback != document.Ports.GatewayPublic+3)) || (!hasCollector && document.Ports.CollectorHealthLoopback != nil) {
		return fmt.Errorf("%w: DeploymentBinding 服务端口无效", ErrReleaseNotDeployable)
	}
	if err := validateReleaseMetadata(release, binding.ProjectID, hasCollector); err != nil {
		return err
	}
	if _, err := time.Parse(time.RFC3339, document.IssuedAt); err != nil {
		return fmt.Errorf("%w: DeploymentBinding 签发时间无效", ErrReleaseNotDeployable)
	}
	return nil
}

func validateEngineDeploymentBinding(raw []byte, binding bindingMetadata, release releaseMetadata, nodeID, deploymentID string) error {
	var document struct {
		SchemaVersion, BindingID, NodeID, DeploymentID, ProjectID, ServiceID, Mode, Engine string
		Revision                                                                           int
		Release                                                                            struct {
			ID, ArchiveSHA256, ManifestSHA256, ChecksumsSHA256, SigningKeyID string
		}
		Ports struct {
			HostPort *int `json:"hostPort"`
		}
		Secrets []any `json:"secrets"`
	}
	if err := json.Unmarshal(raw, &document); err != nil || document.SchemaVersion != "deployment-binding.v2" ||
		document.BindingID != binding.ID || document.Revision != binding.Revision || document.NodeID != nodeID ||
		document.DeploymentID != deploymentID || document.ProjectID != binding.ProjectID || !validServiceType(document.Engine) ||
		(document.Mode != "release" && document.Mode != "development") || len(document.Secrets) != 0 ||
		document.Release.ID != release.ID || document.Release.ArchiveSHA256 != sha256Value(release.ArtifactHash) ||
		document.Release.ManifestSHA256 != sha256Value(release.ManifestHash) || document.Release.ChecksumsSHA256 != sha256Value(release.ChecksumsHash) || document.Release.SigningKeyID != release.SigningKeyID {
		return fmt.Errorf("%w: 引擎 DeploymentBinding 与节点或制品不匹配", ErrReleaseNotDeployable)
	}
	if document.Engine == ServiceBase {
		if document.Ports.HostPort == nil || *document.Ports.HostPort < 1024 || *document.Ports.HostPort > 65532 {
			return fmt.Errorf("%w: 基础引擎端口无效", ErrReleaseNotDeployable)
		}
	} else if document.Ports.HostPort != nil {
		return fmt.Errorf("%w: 非基础引擎不能暴露主机端口", ErrReleaseNotDeployable)
	}
	return validateReleaseMetadata(release, binding.ProjectID, document.Engine == ServiceCollector)
}

func sha256Value(value string) string { return "sha256:" + strings.ToLower(value) }

func mapNotFound(e error) error {
	if errors.Is(e, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return e
}
func mapDeploymentCreateError(e error) error {
	var databaseError *pgconn.PgError
	if errors.As(e, &databaseError) {
		switch databaseError.ConstraintName {
		case "project_deployments_tenant_project_environment_key":
			return ErrDeploymentExists
		case "deployment_services_base_access_port_key":
			return ErrNodePortConflict
		}
	}
	if strings.Contains(fmt.Sprint(e), "project_deployments_tenant_project_environment_key") {
		return ErrDeploymentExists
	}
	if strings.Contains(fmt.Sprint(e), "deployment_services_base_access_port_key") {
		return ErrNodePortConflict
	}
	return e
}
