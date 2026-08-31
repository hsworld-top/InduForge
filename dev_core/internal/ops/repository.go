package ops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgreSQLRepository struct{ pool *pgxpool.Pool }

// pendingServicesSQL 只以物理 node_id 选取期望状态。不得添加按角色或资源池扩散的条件。
const pendingServicesSQL = `SELECT s.id,s.tenant_id,s.project_deployment_id,s.node_id,s.service_type,s.desired_status,s.observed_status,COALESCE(s.last_message,''),COALESCE(s.endpoint,''),s.replicas_desired,s.replicas_observed,s.desired_generation,s.observed_generation,s.last_operation,s.observed_at,s.created_at,s.updated_at FROM deployment_services s JOIN project_deployments d ON d.id=s.project_deployment_id WHERE d.node_id=$1 AND s.node_id=$1 AND (s.desired_generation<>s.observed_generation OR s.desired_status<>s.observed_status) ORDER BY s.updated_at`

const (
	lockDeploymentProjectSQL = `SELECT id FROM projects WHERE tenant_id=$1 AND id=$2 FOR UPDATE`
	lockDeploymentNodeSQL    = `SELECT approved_at IS NOT NULL AND desired_status='active' AND observed_status='online' AND last_heartbeat_at>now()-interval '45 seconds' AND capabilities @> $3::jsonb FROM host_nodes WHERE tenant_id=$1 AND id=$2 FOR UPDATE`
)

func NewPostgreSQLRepository(pool *pgxpool.Pool) *PostgreSQLRepository {
	return &PostgreSQLRepository{pool: pool}
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
	rows, e := r.pool.Query(ctx, `SELECT n.id,n.tenant_id,n.enrollment_id,n.display_name,n.hostname,n.platform,n.architecture,COALESCE(n.agent_version,''),COALESCE(n.machine_fingerprint,''),COALESCE(n.ip_address,''),n.desired_status,n.observed_status,n.capabilities,n.resource_summary,n.last_heartbeat_at,n.approved_at,n.created_at,n.updated_at,COALESCE(d.id::text,''),COALESCE(d.project_id::text,''),COALESCE(p.name,'') FROM host_nodes n LEFT JOIN LATERAL (SELECT d.id,d.project_id FROM project_deployments d WHERE d.tenant_id=n.tenant_id AND d.node_id=n.id ORDER BY d.created_at DESC,d.id DESC LIMIT 1) d ON true LEFT JOIN projects p ON p.id=d.project_id AND p.tenant_id=n.tenant_id WHERE n.tenant_id=$1 AND ($2='' OR n.display_name ILIKE '%'||$2||'%' OR n.hostname ILIKE '%'||$2||'%') ORDER BY n.updated_at DESC,n.id DESC LIMIT $3 OFFSET $4`, tenant, f.Search, f.PageSize, (f.Page-1)*f.PageSize)
	if e != nil {
		return nil, 0, e
	}
	defer rows.Close()
	items := []Node{}
	for rows.Next() {
		x, e := scanNodeWithAssignment(rows)
		if e != nil {
			return nil, 0, e
		}
		items = append(items, x)
	}
	var total int64
	e = r.pool.QueryRow(ctx, `SELECT count(*) FROM host_nodes WHERE tenant_id=$1 AND ($2='' OR display_name ILIKE '%'||$2||'%' OR hostname ILIKE '%'||$2||'%')`, tenant, f.Search).Scan(&total)
	return items, total, e
}
func (r *PostgreSQLRepository) GetNode(ctx context.Context, tenant, id string) (Node, error) {
	x, e := scanNode(r.pool.QueryRow(ctx, `SELECT id,tenant_id,enrollment_id,display_name,hostname,platform,architecture,COALESCE(agent_version,''),COALESCE(machine_fingerprint,''),COALESCE(ip_address,''),desired_status,observed_status,capabilities,resource_summary,last_heartbeat_at,approved_at,created_at,updated_at FROM host_nodes WHERE tenant_id=$1 AND id=$2`, tenant, id))
	return x, mapNotFound(e)
}
func (r *PostgreSQLRepository) Heartbeat(ctx context.Context, id, hash string, in HeartbeatInput) (Node, []DeploymentService, error) {
	summary, _ := json.Marshal(in.ResourceSummary)
	n, e := scanNode(r.pool.QueryRow(ctx, `UPDATE host_nodes SET observed_status=CASE WHEN desired_status='revoked' THEN 'revoked' WHEN observed_status='pending_approval' THEN 'pending_approval' ELSE 'online' END,resource_summary=$1,agent_version=COALESCE(NULLIF($2,''),agent_version),last_heartbeat_at=now(),updated_at=now() WHERE id=$3 AND agent_token_hash=$4 RETURNING id,tenant_id,enrollment_id,display_name,hostname,platform,architecture,COALESCE(agent_version,''),COALESCE(machine_fingerprint,''),COALESCE(ip_address,''),desired_status,observed_status,capabilities,resource_summary,last_heartbeat_at,approved_at,created_at,updated_at`, summary, in.AgentVersion, id, hash))
	if e != nil {
		return Node{}, nil, ErrAgentUnauthorized
	}
	if n.ObservedStatus != "online" || n.DesiredStatus != "active" {
		return n, []DeploymentService{}, nil
	}
	affected := map[string]struct{}{}
	for _, o := range in.Services {
		var did string
		e = r.pool.QueryRow(ctx, `UPDATE deployment_services s SET observed_status=$1,replicas_observed=$2,observed_generation=$3,last_message=$4,endpoint=CASE WHEN s.service_type='project_entry' THEN $5 ELSE s.endpoint END,observed_at=now(),updated_at=now() FROM project_deployments d WHERE s.project_deployment_id=d.id AND s.id=$6 AND s.desired_generation=$3 AND d.node_id=$7 AND (s.service_type='project_entry' OR $5='') RETURNING s.project_deployment_id`, o.ObservedStatus, o.ReplicasObserved, o.ObservedGeneration, o.Message, o.Endpoint, o.ServiceID, n.ID).Scan(&did)
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
		var run, ver string
		_ = r.pool.QueryRow(ctx, `SELECT COALESCE((SELECT id::text FROM deployment_runs WHERE project_deployment_id=$1 AND observed_status='pending' ORDER BY started_at DESC LIMIT 1),''),v.version FROM project_deployments d JOIN application_versions v ON v.id=d.application_version_id WHERE d.id=$1`, s.ProjectDeploymentID).Scan(&run, &ver)
		out = append(out, AgentCommand{NodeID: n.ID, RunID: run, DeploymentID: s.ProjectDeploymentID, ServiceID: s.ID, ServiceType: s.ServiceType, DesiredStatus: s.DesiredStatus, Operation: s.LastOperation, Generation: s.DesiredGeneration, Version: ver, ReplicasDesired: s.ReplicasDesired})
	}
	return out, nil
}
func (r *PostgreSQLRepository) GetNodeByToken(ctx context.Context, id, hash string) (Node, error) {
	x, e := scanNode(r.pool.QueryRow(ctx, `SELECT id,tenant_id,enrollment_id,display_name,hostname,platform,architecture,COALESCE(agent_version,''),COALESCE(machine_fingerprint,''),COALESCE(ip_address,''),desired_status,observed_status,capabilities,resource_summary,last_heartbeat_at,approved_at,created_at,updated_at FROM host_nodes WHERE id=$1 AND agent_token_hash=$2`, id, hash))
	return x, e
}

func (r *PostgreSQLRepository) ListDeployments(ctx context.Context, tenant string, f PageFilter) ([]ProjectDeployment, int64, error) {
	rows, e := r.pool.Query(ctx, `SELECT d.id,d.tenant_id,d.project_id,p.name,d.node_id,n.display_name,d.application_version_id,v.version,COALESCE((SELECT id::text FROM deployment_runs WHERE project_deployment_id=d.id ORDER BY started_at DESC,id DESC LIMIT 1),''),d.desired_status,d.observed_status,COALESCE((SELECT progress FROM deployment_runs WHERE project_deployment_id=d.id ORDER BY started_at DESC,id DESC LIMIT 1),0),d.created_at,d.updated_at FROM project_deployments d JOIN projects p ON p.id=d.project_id AND p.tenant_id=d.tenant_id JOIN host_nodes n ON n.id=d.node_id AND n.tenant_id=d.tenant_id JOIN application_versions v ON v.id=d.application_version_id AND v.tenant_id=d.tenant_id WHERE d.tenant_id=$1 AND ($2='' OR p.name ILIKE '%'||$2||'%') AND ($5='' OR d.project_id::text=$5) ORDER BY d.updated_at DESC,d.id DESC LIMIT $3 OFFSET $4`, tenant, f.Search, f.PageSize, (f.Page-1)*f.PageSize, f.ProjectID)
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
	e = r.pool.QueryRow(ctx, `SELECT count(*) FROM project_deployments d JOIN projects p ON p.id=d.project_id AND p.tenant_id=d.tenant_id JOIN host_nodes n ON n.id=d.node_id AND n.tenant_id=d.tenant_id JOIN application_versions v ON v.id=d.application_version_id AND v.tenant_id=d.tenant_id WHERE d.tenant_id=$1 AND ($2='' OR p.name ILIKE '%'||$2||'%') AND ($3='' OR d.project_id::text=$3)`, tenant, f.Search, f.ProjectID).Scan(&total)
	return out, total, e
}
func (r *PostgreSQLRepository) ValidateDeploymentTargets(ctx context.Context, tenant string, in CreateDeploymentInput) error {
	var count int
	if e := r.pool.QueryRow(ctx, `SELECT count(*) FROM projects WHERE id=$1 AND tenant_id=$2`, in.ProjectID, tenant).Scan(&count); e != nil || count != 1 {
		return ErrNotFound
	}
	var status, artifactKey, artifactHash string
	var manifestJSON []byte
	if e := r.pool.QueryRow(ctx, `SELECT status,COALESCE(artifact_key,''),COALESCE(artifact_hash,''),manifest FROM application_versions WHERE id=$1 AND project_id=$2 AND tenant_id=$3 AND deleted_at IS NULL`, in.ApplicationVersionID, in.ProjectID, tenant).Scan(&status, &artifactKey, &artifactHash, &manifestJSON); errors.Is(e, pgx.ErrNoRows) {
		return ErrNotFound
	} else if e != nil {
		return e
	}
	if status != "ready" {
		return fmt.Errorf("%w: 版本尚未构建成功", ErrReleaseNotDeployable)
	}
	if e := validateDeployableReleaseForDeployment(in.ProjectID, artifactKey, artifactHash, in.EnableCollector, manifestJSON); e != nil {
		return e
	}
	required := []string{CapabilityProjectEntry, CapabilityDataRuntime}
	if in.EnableCollector {
		required = append(required, CapabilityCollector)
	}
	caps, _ := json.Marshal(required)
	var ok bool
	e := r.pool.QueryRow(ctx, `SELECT approved_at IS NOT NULL AND desired_status='active' AND observed_status='online' AND last_heartbeat_at>now()-interval '45 seconds' AND capabilities @> $3::jsonb FROM host_nodes WHERE id=$1 AND tenant_id=$2`, in.NodeID, tenant, caps).Scan(&ok)
	if e != nil {
		return ErrNotFound
	}
	if !ok {
		return fmt.Errorf("节点当前不能调度：需已审批、在线且具备工程服务能力")
	}
	var deployedProjectID string
	e = r.pool.QueryRow(ctx, `SELECT project_id::text FROM project_deployments WHERE tenant_id=$1 AND node_id=$2 LIMIT 1`, tenant, in.NodeID).Scan(&deployedProjectID)
	if e == nil {
		if deployedProjectID == in.ProjectID {
			return ErrDeploymentExists
		}
		return ErrNodeProjectConflict
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return e
	}
	return nil
}
func (r *PostgreSQLRepository) CreateDeployment(ctx context.Context, tenant, user string, in CreateDeploymentInput) (ProjectDeployment, DeploymentRun, error) {
	tx, e := r.pool.Begin(ctx)
	if e != nil {
		return ProjectDeployment{}, DeploymentRun{}, e
	}
	defer tx.Rollback(ctx)
	var lockedID string
	if e = tx.QueryRow(ctx, lockDeploymentProjectSQL, tenant, in.ProjectID).Scan(&lockedID); errors.Is(e, pgx.ErrNoRows) {
		return ProjectDeployment{}, DeploymentRun{}, ErrNotFound
	} else if e != nil {
		return ProjectDeployment{}, DeploymentRun{}, e
	}
	required := []string{CapabilityProjectEntry, CapabilityDataRuntime}
	if in.EnableCollector {
		required = append(required, CapabilityCollector)
	}
	caps, _ := json.Marshal(required)
	var nodeReady bool
	if e = tx.QueryRow(ctx, lockDeploymentNodeSQL, tenant, in.NodeID, caps).Scan(&nodeReady); errors.Is(e, pgx.ErrNoRows) {
		return ProjectDeployment{}, DeploymentRun{}, ErrNotFound
	} else if e != nil {
		return ProjectDeployment{}, DeploymentRun{}, e
	}
	if !nodeReady {
		return ProjectDeployment{}, DeploymentRun{}, fmt.Errorf("节点当前不能调度：需已审批、在线且具备工程服务能力")
	}
	var deployedProjectID string
	e = tx.QueryRow(ctx, `SELECT project_id::text FROM project_deployments WHERE tenant_id=$1 AND (project_id=$2 OR node_id=$3) ORDER BY CASE WHEN project_id=$2 THEN 0 ELSE 1 END,id LIMIT 1`, tenant, in.ProjectID, in.NodeID).Scan(&deployedProjectID)
	if e == nil {
		if deployedProjectID == in.ProjectID {
			return ProjectDeployment{}, DeploymentRun{}, ErrDeploymentExists
		}
		return ProjectDeployment{}, DeploymentRun{}, ErrNodeProjectConflict
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return ProjectDeployment{}, DeploymentRun{}, e
	}
	var d ProjectDeployment
	d, e = scanDeployment(tx.QueryRow(ctx, `INSERT INTO project_deployments(tenant_id,project_id,node_id,application_version_id,created_by) SELECT $1,$2,$3,$4,$5 RETURNING id,tenant_id,project_id,'',$3,'',$4,'','',desired_status,observed_status,0,created_at,updated_at`, tenant, in.ProjectID, in.NodeID, in.ApplicationVersionID, user))
	if e != nil {
		return d, DeploymentRun{}, mapDeploymentCreateError(e)
	}
	types := []string{ServiceProjectEntry, ServiceDataRuntime}
	if in.EnableCollector {
		types = append(types, ServiceCollector)
	}
	for _, kind := range types {
		_, e = tx.Exec(ctx, `INSERT INTO deployment_services(tenant_id,project_deployment_id,node_id,service_type) VALUES($1,$2,$3,$4)`, tenant, d.ID, in.NodeID, kind)
		if e != nil {
			return d, DeploymentRun{}, e
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
	d, e := scanDeployment(r.pool.QueryRow(ctx, `SELECT d.id,d.tenant_id,d.project_id,p.name,d.node_id,n.display_name,d.application_version_id,v.version,COALESCE((SELECT id::text FROM deployment_runs WHERE project_deployment_id=d.id ORDER BY started_at DESC,id DESC LIMIT 1),''),d.desired_status,d.observed_status,COALESCE((SELECT progress FROM deployment_runs WHERE project_deployment_id=d.id ORDER BY started_at DESC,id DESC LIMIT 1),0),d.created_at,d.updated_at FROM project_deployments d JOIN projects p ON p.id=d.project_id AND p.tenant_id=d.tenant_id JOIN host_nodes n ON n.id=d.node_id AND n.tenant_id=d.tenant_id JOIN application_versions v ON v.id=d.application_version_id AND v.tenant_id=d.tenant_id WHERE d.tenant_id=$1 AND d.id=$2`, tenant, id))
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
	rows, e := r.pool.Query(ctx, `SELECT id,tenant_id,project_deployment_id,node_id,service_type,desired_status,observed_status,COALESCE(last_message,''),COALESCE(endpoint,''),replicas_desired,replicas_observed,desired_generation,observed_generation,last_operation,observed_at,created_at,updated_at FROM deployment_services WHERE project_deployment_id=$1 ORDER BY service_type`, did)
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
func scanNodeWithAssignment(s scanner) (Node, error) {
	var x Node
	var caps, summary []byte
	args := nodeScanArgs(&x, &caps, &summary)
	args = append(args, &x.AssignedDeploymentID, &x.AssignedProjectID, &x.AssignedProjectName)
	e := s.Scan(args...)
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
	e := s.Scan(&x.ID, &x.TenantID, &x.ProjectID, &x.ProjectName, &x.NodeID, &x.NodeName, &x.ApplicationVersionID, &x.Version, &x.LatestRunID, &x.DesiredStatus, &x.ObservedStatus, &x.Progress, &x.CreatedAt, &x.UpdatedAt)
	x.Health = normalizeHealth(x.ObservedStatus)
	return x, e
}
func serviceScanArgs(x *DeploymentService) []any {
	return []any{&x.ID, &x.TenantID, &x.ProjectDeploymentID, &x.NodeID, &x.ServiceType, &x.DesiredStatus, &x.ObservedStatus, &x.LastMessage, &x.Endpoint, &x.ReplicasDesired, &x.ReplicasObserved, &x.DesiredGeneration, &x.ObservedGeneration, &x.LastOperation, &x.ObservedAt, &x.CreatedAt, &x.UpdatedAt}
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
		case "project_deployments_tenant_project_key":
			return ErrDeploymentExists
		case "project_deployments_tenant_node_key":
			return ErrNodeProjectConflict
		}
	}
	if strings.Contains(fmt.Sprint(e), "project_deployments_tenant_project_key") {
		return ErrDeploymentExists
	}
	if strings.Contains(fmt.Sprint(e), "project_deployments_tenant_node_key") {
		return ErrNodeProjectConflict
	}
	return e
}
