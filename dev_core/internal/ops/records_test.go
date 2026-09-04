package ops

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/auth"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
	"github.com/jackc/pgx/v5"
)

type recordReader struct {
	t                 *testing.T
	calls, size       int
	args              [][]any
	queries           []string
	rows              *historyRows
	runID, recordType string
	sourceKind        string
}

func (r *recordReader) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	r.calls++
	r.args = append(r.args, args)
	r.queries = append(r.queries, q)
	return lifecycleRow{values: []any{int64(r.size)}}
}
func (r *recordReader) Query(_ context.Context, q string, args ...any) (pgx.Rows, error) {
	r.calls++
	r.args = append(r.args, args)
	r.queries = append(r.queries, q)
	var environment *string
	var completed *time.Time
	var duration *int64
	sourceKind := r.sourceKind
	if sourceKind == "" {
		sourceKind = "deployment_run"
	}
	owner := &historyReader{t: r.t, active: true, size: r.size, values: []any{sourceKind + ":id", sourceKind, r.recordType, "deployment", "object", "工程", environment, "启动", "running", "", "actor", time.Now(), completed, duration, "message", r.runID}}
	r.rows = &historyRows{reader: owner}
	return r.rows, nil
}

func TestRestoreRecordUsesRestoreTaskReference(t *testing.T) {
	reader := &recordReader{t: t, size: 1, runID: "restore-task", recordType: "operation", sourceKind: "authoring_restore_task"}
	rows, _, err := loadRecords(context.Background(), reader, "tenant", RecordFilter{}, true, false)
	if err != nil || len(rows) != 1 || rows[0].TaskRef == nil {
		t.Fatalf("恢复记录加载失败 rows=%+v err=%v", rows, err)
	}
	if rows[0].TaskRef.RestoreTaskID != "restore-task" || rows[0].TaskRef.RunID != "" || rows[0].TaskRef.DeploymentID != "" {
		t.Fatalf("恢复记录不能伪装成部署任务 %+v", rows[0].TaskRef)
	}
}
func TestRecordsFixedDatabasePaginationPermissionsAndDeletedSummary(t *testing.T) {
	for _, size := range []int{0, 1, 20, 200} {
		for _, deleted := range []bool{false, true} {
			runID := "run"
			if deleted {
				runID = ""
			}
			reader := &recordReader{t: t, size: size, runID: runID, recordType: "operation"}
			f := RecordFilter{PageFilter: PageFilter{Page: 2, PageSize: 20, Search: `x%_\`}, Sort: "time_asc"}
			rows, total, err := loadRecords(context.Background(), reader, "tenant", f, true, false)
			if err != nil || reader.calls != 2 || len(rows) != size || total != int64(size) || reader.rows.reader.active {
				t.Fatalf("固定2query/释放rows失败 %+v %v", reader, err)
			}
			if !reflect.DeepEqual(reader.args[0], reader.args[1][:11]) || reader.args[0][0] != "tenant" || reader.args[0][2] != false || reader.args[0][3] != `x\%\_\\` || !reflect.DeepEqual(reader.args[1][11:], []any{20, 20}) {
				t.Fatalf("过滤权限/count/page条件不一致 %v", reader.args)
			}
			if !strings.Contains(reader.queries[1], "ORDER BY time ASC,id ASC LIMIT $12 OFFSET $13") {
				t.Fatal("数据库稳定排序/分页缺失")
			}
			for _, row := range rows {
				if deleted {
					if row.TaskRef != nil || row.DetailUnavailableReason == "" {
						t.Fatal("已删除审计必须留摘要并说明不可展开")
					}
				} else if row.TaskRef == nil || row.TaskRef.RunID != "run" {
					t.Fatal("真实任务引用缺失")
				}
			}
		}
	}
}

func TestRecordsSQLCanonicalSourcesAndFormalSchema(t *testing.T) {
	for _, fragment := range []string{
		"WHERE r.tenant_id=$1 AND $2::boolean", "WHERE e.tenant_id=$1 AND $3::boolean",
		"u.tenant_id=r.tenant_id", "u.tenant_id=e.tenant_id",
		"e.event_type NOT IN ('node_online','node_offline','node_time_sync_changed')",
		"right(e.event_type,10)='_requested' AND e.result='success' THEN 'accepted'",
		"WHEN e.result='failed' THEN 'warning'", "WHEN e.event_type='node_online' THEN 'recovered' ELSE 'info'",
		"r.completed_at IS NOT NULL AND r.observed_status IN ('running','stopped') THEN 'success'",
		"CASE WHEN d.deleted_at IS NULL THEN r.id::text ELSE '' END",
		"left(COALESCE(r.message,''),4)='重新部署' THEN '重新部署工程'",
		"COALESCE(NULLIF(e.target,''),c.name),NULL::text", "left(e.event_type,11)='foundation_'",
		"FROM authoring_restore_tasks t JOIN projects p ON p.id=t.project_id AND p.tenant_id=t.tenant_id",
		"LEFT JOIN LATERAL (SELECT e.message FROM authoring_restore_task_events e",
		"WHEN t.state='queued' THEN 'accepted' WHEN t.state='succeeded' THEN 'success' WHEN t.state='failed' THEN 'failed' ELSE 'running'",
		"'authoring_restore_task'", "'恢复工程开发态'",
	} {
		if !strings.Contains(opsRecordsUnionSQL, fragment) {
			t.Fatalf("状态/权限/镜像规则缺失 %s", fragment)
		}
	}
	if strings.Contains(opsRecordsUnionSQL, "AND d.deleted_at IS NULL") || strings.Contains(opsRecordsUnionSQL, "AND v.deleted_at IS NULL") {
		t.Fatal("删除对象审计被丢弃")
	}
	if strings.Count(opsRecordsUnionSQL, " UNION ALL") != 3 || strings.Contains(opsRecordsUnionSQL, "runtime_environment_nodes") {
		t.Fatal("源不精确或按当前成员猜历史归属")
	}
	schema, err := os.ReadFile("../../db/schema/core-schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"CREATE TABLE deployment_runs (", "CREATE TABLE runtime_cluster_events (", "CREATE TABLE runtime_environment_events (", "node_id uuid REFERENCES host_nodes (id) ON DELETE SET NULL", "created_by uuid REFERENCES users (id) ON DELETE SET NULL", "result text NOT NULL CHECK (result IN ('success', 'failed'))"} {
		if !strings.Contains(string(schema), fragment) {
			t.Fatalf("正式schema与投影假设不一致 %s", fragment)
		}
	}
}

type recordsRepository struct {
	Repository
	called        bool
	tenant        string
	f             RecordFilter
	tasks, events bool
}

func (r *recordsRepository) ListRecords(_ context.Context, tenant string, f RecordFilter, tasks, events bool) ([]OpsRecord, int64, error) {
	r.called = true
	r.tenant = tenant
	r.f = f
	r.tasks = tasks
	r.events = events
	return []OpsRecord{{ID: "environment_event:id", RecordType: "event", Status: "accepted"}}, 1, nil
}
func TestRecordsHTTPValidationAndTenantPermission(t *testing.T) {
	_ = httptest.NewRequest("GET", "/api/v1/ops/records", nil) // 契约扫描器的无查询参数基准请求。
	for _, test := range []struct {
		role, query string
		valid       bool
	}{
		{"OPS_ADMIN", "?page=2&limit=300&tenantId=other", true},
		{"USER_ADMIN", "", false}, {"OPERATOR", "", false},
		{"OPS_ADMIN", "?recordType=node&objectType=node", false},
		{"OPS_ADMIN", "?recordType=operation%7Cevent", false},
		{"OPS_ADMIN", "?objectType=node%7Ccluster", false},
		{"OPS_ADMIN", "?status=success%7Cfailed", false},
		{"OPS_ADMIN", "?sort=time_desc%7Ctime_asc", false},
		{"OPS_ADMIN", "?from=invalid", false},
		{"OPS_ADMIN", "?from=2026-09-04T00:00:00Z&to=2026-09-03T00:00:00Z", false},
		{"OPS_ADMIN", "?sort=drop", false}, {"OPS_ADMIN", "?environmentId=other", false},
	} {
		repo := &recordsRepository{}
		h := NewHandler(NewService(repo, nil), nil)
		router := chi.NewRouter()
		h.MountRoutes(router)
		req := httptest.NewRequest("GET", "/api/v1/ops/records"+test.query, nil)
		req = req.WithContext(auth.WithUser(req.Context(), auth.User{TenantID: "tenant", Role: test.role}))
		out := httptest.NewRecorder()
		router.ServeHTTP(out, req)
		var result struct {
			Code int `json:"code"`
			Data struct {
				List []OpsRecord `json:"list"`
			} `json:"data"`
		}
		if err := json.Unmarshal(out.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if out.Code >= 500 || repo.called != test.valid {
			t.Fatalf("请求校验失败 %s status=%d body=%s", test.query, out.Code, out.Body.String())
		}
		if test.valid {
			if result.Code != 0 || repo.tenant != "tenant" || !repo.tasks || !repo.events || repo.f.PageSize != 200 || len(result.Data.List) != 1 {
				t.Fatalf("权限或分页包络错误 %+v %+v", repo, result)
			}
		} else if test.role == "OPS_ADMIN" && result.Code != platformapi.ErrorCodeInvalidRequest {
			t.Fatalf("非法参数必须invalid request: %s", out.Body.String())
		}
	}
}
