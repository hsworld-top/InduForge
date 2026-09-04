package ops

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/auth"
)

type DeploymentRunHistory struct {
	DeploymentRun
	ActorDisplayName string
	DurationMs       *int64
}

const deploymentHistoryScopeSQL = ` FROM deployment_runs r JOIN project_deployments d ON d.id=r.project_deployment_id AND d.tenant_id=r.tenant_id WHERE r.tenant_id=$1 AND d.id=$2 AND d.deleted_at IS NULL`
const runEventsPageScopeSQL = ` FROM deployment_run_events e JOIN deployment_runs r ON r.id=e.deployment_run_id JOIN project_deployments d ON d.id=r.project_deployment_id AND d.tenant_id=r.tenant_id WHERE r.tenant_id=$1 AND r.id=$2 AND d.deleted_at IS NULL`

// 历史只允许访问当前租户未删除的部署。固定三次查询，姓名在本页一次 JOIN，绝不逐任务查用户。
func loadDeploymentRuns(ctx context.Context, reader deploymentPageReader, tenant, id string, f PageFilter) ([]DeploymentRunHistory, int64, error) {
	var visible string
	if err := reader.QueryRow(ctx, `SELECT id::text FROM project_deployments WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`, tenant, id).Scan(&visible); err != nil {
		return nil, 0, mapNotFound(err)
	}
	var total int64
	if err := reader.QueryRow(ctx, `SELECT count(*)`+deploymentHistoryScopeSQL, tenant, id).Scan(&total); err != nil {
		return nil, 0, err
	}
	query := `SELECT r.id,r.tenant_id,r.project_deployment_id,r.operation,r.desired_status,r.observed_status,r.progress,COALESCE(r.message,''),r.started_at,r.completed_at,COALESCE(NULLIF(u.full_name,''),u.username,'未知用户') FROM deployment_runs r JOIN project_deployments d ON d.id=r.project_deployment_id AND d.tenant_id=r.tenant_id LEFT JOIN users u ON u.id=r.created_by AND u.tenant_id=r.tenant_id WHERE r.tenant_id=$1 AND d.id=$2 AND d.deleted_at IS NULL ORDER BY r.started_at DESC,r.id DESC LIMIT $3 OFFSET $4`
	rows, err := reader.Query(ctx, query, tenant, id, f.PageSize, (f.Page-1)*f.PageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []DeploymentRunHistory{}
	for rows.Next() {
		var item DeploymentRunHistory
		args := append(runScanArgs(&item.DeploymentRun), &item.ActorDisplayName)
		if err = rows.Scan(args...); err != nil {
			return nil, 0, err
		}
		// 未完成返回 null，避免用浏览器时钟或百分比伪造耗时；异常负时间收敛为零。
		if item.CompletedAt != nil {
			duration := item.CompletedAt.Sub(item.StartedAt).Milliseconds()
			if duration < 0 {
				duration = 0
			}
			item.DurationMs = &duration
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *PostgreSQLRepository) ListDeploymentRuns(ctx context.Context, tenant, id string, f PageFilter) ([]DeploymentRunHistory, int64, error) {
	return loadDeploymentRuns(ctx, r.pool, tenant, id, normalizePage(f))
}
func (r *PostgreSQLRepository) ListRunEventsPage(ctx context.Context, tenant, id string, f PageFilter) ([]DeploymentRunEvent, int64, error) {
	return loadRunEventsPage(ctx, r.pool, tenant, id, normalizePage(f))
}

func loadRunEventsPage(ctx context.Context, reader deploymentPageReader, tenant, id string, f PageFilter) ([]DeploymentRunEvent, int64, error) {
	var visible string
	if err := reader.QueryRow(ctx, `SELECT r.id::text FROM deployment_runs r JOIN project_deployments d ON d.id=r.project_deployment_id AND d.tenant_id=r.tenant_id WHERE r.tenant_id=$1 AND r.id=$2 AND d.deleted_at IS NULL`, tenant, id).Scan(&visible); err != nil {
		return nil, 0, mapNotFound(err)
	}
	var total int64
	if err := reader.QueryRow(ctx, `SELECT count(*)`+runEventsPageScopeSQL, tenant, id).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := reader.Query(ctx, `SELECT e.id,e.deployment_run_id,e.stage,e.message,e.created_at`+runEventsPageScopeSQL+` ORDER BY e.created_at ASC,e.id ASC LIMIT $3 OFFSET $4`, tenant, id, f.PageSize, (f.Page-1)*f.PageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []DeploymentRunEvent{}
	for rows.Next() {
		var item DeploymentRunEvent
		if err = rows.Scan(&item.ID, &item.DeploymentRunID, &item.Stage, &item.Message, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (s *Service) ListDeploymentRuns(ctx context.Context, actor auth.User, id string, f PageFilter) ([]DeploymentRunHistory, int64, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityNodeRead); err != nil {
		return nil, 0, err
	}
	return s.repository.ListDeploymentRuns(ctx, actor.TenantID, id, normalizePage(f))
}
func (s *Service) ListRunEventsPage(ctx context.Context, actor auth.User, id string, f PageFilter) ([]DeploymentRunEvent, int64, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityNodeRead); err != nil {
		return nil, 0, err
	}
	return s.repository.ListRunEventsPage(ctx, actor.TenantID, id, normalizePage(f))
}
func historyPage(r *http.Request) PageFilter {
	p, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	return normalizePage(PageFilter{Page: p, PageSize: limit})
}
func historyPagePayload(items any, total int64, f PageFilter) map[string]any {
	return map[string]any{"list": items, "pagination": map[string]any{"page": f.Page, "limit": f.PageSize, "total": total}}
}
func (h *Handler) listDeploymentRuns(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(actor auth.User) {
		f := historyPage(r)
		items, total, err := h.service.ListDeploymentRuns(r.Context(), actor, chi.URLParam(r, "id"), f)
		if err != nil {
			h.err(w, r, err)
			return
		}
		payload := make([]map[string]any, 0, len(items))
		for _, item := range items {
			x := runPayload(item.DeploymentRun)
			x["actorDisplayName"] = item.ActorDisplayName
			x["durationMs"] = item.DurationMs
			payload = append(payload, x)
		}
		h.writeSuccess(w, r, historyPagePayload(payload, total, f))
	})
}
func (h *Handler) listRunEventsPage(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(actor auth.User) {
		f := historyPage(r)
		items, total, err := h.service.ListRunEventsPage(r.Context(), actor, chi.URLParam(r, "id"), f)
		if err != nil {
			h.err(w, r, err)
			return
		}
		h.writeSuccess(w, r, historyPagePayload(items, total, f))
	})
}
