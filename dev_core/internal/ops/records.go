package ops

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/indu-forge/dev_core/internal/auth"
)

type RecordFilter struct {
	PageFilter
	RecordType, ObjectType, ObjectID, Sort string
	From, To                               *time.Time
}
type RecordTaskRef struct {
	RunID        string `json:"runId"`
	DeploymentID string `json:"deploymentId"`
}
type OpsRecord struct {
	ID                      string         `json:"id"`
	SourceKind              string         `json:"sourceKind"`
	RecordType              string         `json:"recordType"`
	ObjectType              string         `json:"objectType"`
	ObjectID                string         `json:"objectId"`
	ObjectName              string         `json:"objectName"`
	EnvironmentID           *string        `json:"environmentId"`
	Title                   string         `json:"title"`
	Status                  string         `json:"status"`
	EventResult             string         `json:"eventResult"`
	ActorDisplayName        string         `json:"actorDisplayName"`
	Time                    time.Time      `json:"time"`
	CompletedAt             *time.Time     `json:"completedAt"`
	DurationMs              *int64         `json:"durationMs"`
	Message                 string         `json:"message"`
	TaskRef                 *RecordTaskRef `json:"taskRef"`
	DetailUnavailableReason string         `json:"detailUnavailableReason"`
}

// 三类数据在数据库内 UNION/过滤/排序，不在进程内全量合并。每个源都显式限定租户与权限。
// 节点三类扇出镜像只保留有真实 node_id 的 cluster 规范源，不通过消息、名称、时间猜关联。
const opsRecordsUnionSQL = `SELECT 'deployment_run:'||r.id::text AS id,'deployment_run'::text AS source_kind,'operation'::text AS record_type,
 'deployment'::text AS object_type,d.id::text AS object_id,p.name AS object_name,d.environment_id::text AS environment_id,
 CASE r.operation WHEN 'deploy' THEN CASE WHEN left(COALESCE(r.message,''),4)='重新部署' THEN '重新部署工程' ELSE '部署工程' END WHEN 'start' THEN '启动工程' WHEN 'stop' THEN '停止工程' WHEN 'restart' THEN '重启工程' WHEN 'delete' THEN '删除工程' ELSE '工程操作' END AS title,
 CASE WHEN r.observed_status='failed' THEN 'failed' WHEN r.completed_at IS NOT NULL AND r.observed_status IN ('running','stopped') THEN 'success' ELSE 'running' END AS status,
 ''::text AS event_result,COALESCE(NULLIF(u.full_name,''),u.username,'未知用户') AS actor_display_name,
 r.started_at AS time,r.completed_at,CASE WHEN r.completed_at IS NULL THEN NULL ELSE GREATEST(0,FLOOR(EXTRACT(EPOCH FROM (r.completed_at-r.started_at))*1000))::bigint END AS duration_ms,
 COALESCE(r.message,'') AS message,CASE WHEN d.deleted_at IS NULL THEN r.id::text ELSE '' END AS run_id
 FROM deployment_runs r JOIN project_deployments d ON d.id=r.project_deployment_id AND d.tenant_id=r.tenant_id
 JOIN projects p ON p.id=d.project_id AND p.tenant_id=r.tenant_id
 LEFT JOIN users u ON u.id=r.created_by AND u.tenant_id=r.tenant_id
 WHERE r.tenant_id=$1 AND $2::boolean
 UNION ALL
 SELECT 'cluster_event:'||e.id::text,'cluster_event',CASE WHEN e.event_type IN ('worker_join_requested','worker_remove_requested') THEN 'operation' ELSE 'event' END,CASE WHEN e.node_id IS NULL THEN 'cluster' ELSE 'node' END,
 COALESCE(e.node_id,e.cluster_id)::text,COALESCE(NULLIF(e.target,''),c.name),NULL::text,e.name,
 CASE WHEN right(e.event_type,10)='_requested' AND e.result='success' THEN 'accepted' WHEN e.event_type IN ('worker_join_requested','worker_remove_requested') THEN e.result WHEN e.result='failed' THEN 'warning' WHEN e.event_type='node_online' THEN 'recovered' ELSE 'info' END,
 e.result,CASE WHEN e.created_by IS NULL THEN '系统或已删除用户' ELSE COALESCE(NULLIF(u.full_name,''),u.username,'未知用户') END,
 e.created_at,NULL::timestamptz,NULL::bigint,COALESCE(e.message,''),''
 FROM runtime_cluster_events e JOIN runtime_clusters c ON c.id=e.cluster_id AND c.tenant_id=e.tenant_id
 LEFT JOIN users u ON u.id=e.created_by AND u.tenant_id=e.tenant_id
 WHERE e.tenant_id=$1 AND $3::boolean
 UNION ALL
 SELECT 'environment_event:'||e.id::text,'environment_event',CASE WHEN e.event_type IN ('environment_created','environment_updated','environment_delete_requested','node_added','node_unassigned','foundation_migration_requested','foundation_deploy_requested','foundation_redeploy_requested') THEN 'operation' ELSE 'event' END,CASE WHEN left(e.event_type,11)='foundation_' THEN 'foundation' ELSE 'environment' END,
 e.environment_id::text,v.name,e.environment_id::text,e.name,
 CASE WHEN right(e.event_type,10)='_requested' AND e.result='success' THEN 'accepted' WHEN e.event_type IN ('environment_created','environment_updated','environment_delete_requested','node_added','node_unassigned','foundation_migration_requested','foundation_deploy_requested','foundation_redeploy_requested') THEN e.result WHEN e.result='failed' THEN 'warning' ELSE 'info' END,
 e.result,CASE WHEN e.created_by IS NULL THEN '系统或已删除用户' ELSE COALESCE(NULLIF(u.full_name,''),u.username,'未知用户') END,
 e.created_at,NULL::timestamptz,NULL::bigint,COALESCE(e.message,''),''
 FROM runtime_environment_events e JOIN runtime_environments v ON v.id=e.environment_id AND v.tenant_id=e.tenant_id
 LEFT JOIN users u ON u.id=e.created_by AND u.tenant_id=e.tenant_id
 WHERE e.tenant_id=$1 AND $3::boolean AND e.event_type NOT IN ('node_online','node_offline','node_time_sync_changed')`

const opsRecordsFilterSQL = ` WHERE ($4='' OR title ILIKE '%'||$4||'%' OR object_name ILIKE '%'||$4||'%' OR message ILIKE '%'||$4||'%')
 AND ($5='' OR record_type=$5) AND ($6='' OR object_type=$6) AND ($7='' OR environment_id=$7)
 AND ($8='' OR status=$8) AND ($9::timestamptz IS NULL OR time >= $9) AND ($10::timestamptz IS NULL OR time <= $10)
 AND ($11='' OR object_id=$11)`

func normalizeRecordFilter(f RecordFilter) (RecordFilter, error) {
	f.PageFilter = normalizePage(f.PageFilter)
	f.EnvironmentID = strings.TrimSpace(f.EnvironmentID)
	f.ObjectID = strings.TrimSpace(f.ObjectID)
	if utf8.RuneCountInString(f.Search) > 200 {
		return f, fmt.Errorf("搜索词不能超过200个字符")
	}
	for _, field := range []struct{ value, allowed string }{{f.RecordType, "|operation|event|"}, {f.ObjectType, "|deployment|foundation|environment|node|cluster|"}, {f.Sort, "|time_desc|time_asc|"}, {f.Status, "|accepted|running|success|failed|warning|recovered|info|"}} {
		valid := field.value == ""
		for _, allowed := range strings.Split(field.allowed, "|") {
			valid = valid || field.value == allowed
		}
		if !valid {
			return f, fmt.Errorf("运维记录筛选参数无效")
		}
	}
	for _, id := range []string{f.EnvironmentID, f.ObjectID} {
		if id != "" {
			if _, err := uuid.Parse(id); err != nil {
				return f, fmt.Errorf("对象或环境标识无效")
			}
		}
	}
	if f.From != nil && f.To != nil && f.From.After(*f.To) {
		return f, fmt.Errorf("时间范围无效：开始时间不能晚于结束时间")
	}
	if f.Sort == "" {
		f.Sort = "time_desc"
	}
	return f, nil
}
func (s *Service) ListRecords(ctx context.Context, actor auth.User, f RecordFilter) ([]OpsRecord, int64, error) {
	// 完整事件源需要两个能力，与实时 events topic 一致；仅 node:read 可读取任务源。
	allowTasks := auth.HasCapability(actor.Role, auth.CapabilityNodeRead)
	allowEvents := allowTasks && auth.HasCapability(actor.Role, auth.CapabilityEnvironmentRead)
	if !allowTasks {
		return nil, 0, auth.RequireCapability(actor, auth.CapabilityNodeRead)
	}
	f, err := normalizeRecordFilter(f)
	if err != nil {
		return nil, 0, err
	}
	return s.repository.ListRecords(ctx, actor.TenantID, f, allowTasks, allowEvents)
}
func (r *PostgreSQLRepository) ListRecords(ctx context.Context, tenant string, f RecordFilter, allowTasks, allowEvents bool) ([]OpsRecord, int64, error) {
	return loadRecords(ctx, r.pool, tenant, f, allowTasks, allowEvents)
}
func loadRecords(ctx context.Context, reader deploymentPageReader, tenant string, f RecordFilter, allowTasks, allowEvents bool) ([]OpsRecord, int64, error) {
	// LIKE 的通配符转为字面字符，搜索不能借 '%' 绕过关键词语义。
	search := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(f.Search)
	args := []any{tenant, allowTasks, allowEvents, search, f.RecordType, f.ObjectType, f.EnvironmentID, f.Status, f.From, f.To, f.ObjectID}
	base := ` FROM (` + opsRecordsUnionSQL + `) records` + opsRecordsFilterSQL
	var total int64
	if err := reader.QueryRow(ctx, `SELECT count(*)`+base, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	direction := "DESC"
	if f.Sort == "time_asc" {
		direction = "ASC"
	}
	query := `SELECT *` + base + ` ORDER BY time ` + direction + `,id ` + direction + ` LIMIT $12 OFFSET $13`
	rows, err := reader.Query(ctx, query, append(args, f.PageSize, (f.Page-1)*f.PageSize)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []OpsRecord{}
	for rows.Next() {
		var item OpsRecord
		var runID string
		if err = rows.Scan(&item.ID, &item.SourceKind, &item.RecordType, &item.ObjectType, &item.ObjectID, &item.ObjectName, &item.EnvironmentID, &item.Title, &item.Status, &item.EventResult, &item.ActorDisplayName, &item.Time, &item.CompletedAt, &item.DurationMs, &item.Message, &runID); err != nil {
			return nil, 0, err
		}
		if runID != "" {
			item.TaskRef = &RecordTaskRef{RunID: runID, DeploymentID: item.ObjectID}
		}
		if item.SourceKind == "deployment_run" && runID == "" {
			item.DetailUnavailableReason = "对象已删除，保留操作摘要"
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}
func (h *Handler) listRecords(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(actor auth.User) {
		q := r.URL.Query()
		f := RecordFilter{PageFilter: historyPage(r), RecordType: q.Get("recordType"), ObjectType: q.Get("objectType"), ObjectID: q.Get("objectId"), Sort: q.Get("sort")}
		f.Search = q.Get("search")
		f.EnvironmentID = q.Get("environmentId")
		f.Status = q.Get("status")
		for key, dest := range map[string]**time.Time{"from": &f.From, "to": &f.To} {
			if raw := q.Get(key); raw != "" {
				parsed, err := time.Parse(time.RFC3339, raw)
				if err != nil {
					h.err(w, r, fmt.Errorf("时间筛选无效：必须为RFC3339格式"))
					return
				}
				*dest = &parsed
			}
		}
		items, total, err := h.service.ListRecords(r.Context(), actor, f)
		if err != nil {
			h.err(w, r, err)
			return
		}
		h.writeSuccess(w, r, historyPagePayload(items, total, f.PageFilter))
	})
}
