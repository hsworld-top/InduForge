package ops

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/jackc/pgx/v5"
)

type historyReader struct {
	t           *testing.T
	size, calls int
	missing     bool
	active      bool
	query       string
	args        []any
	values      []any
	rowErr      error
}

func (r *historyReader) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	r.calls++
	if r.active {
		r.t.Fatal("持rows嵌套查询")
	}
	if !reflect.DeepEqual(args, []any{"tenant", "id"}) {
		r.t.Fatalf("参数越权 %v", args)
	}
	if !strings.Contains(q, "tenant_id=$1") || !strings.Contains(q, "deleted_at IS NULL") {
		r.t.Fatal("未约束可见部署")
	}
	if r.calls == 1 {
		if r.missing {
			return lifecycleRow{err: pgx.ErrNoRows}
		}
		return lifecycleRow{values: []any{"id"}}
	}
	return lifecycleRow{values: []any{int64(r.size)}}
}
func (r *historyReader) Query(_ context.Context, q string, args ...any) (pgx.Rows, error) {
	r.calls++
	r.query = q
	r.args = args
	r.active = true
	return &historyRows{reader: r}, nil
}

type historyRows struct {
	pgx.Rows
	reader *historyReader
	index  int
}

func (r *historyRows) Next() bool { r.index++; return r.index <= r.reader.size }
func (r *historyRows) Close()     { r.reader.active = false }
func (r *historyRows) Err() error { return r.reader.rowErr }
func (r *historyRows) Scan(dest ...any) error {
	return (lifecycleRow{values: r.reader.values}).Scan(dest...)
}

func TestDeploymentHistoryFixedQueriesDurationAndTenant(t *testing.T) {
	start := time.Date(2026, 9, 3, 1, 0, 0, 0, time.UTC)
	for _, size := range []int{0, 1, 20, 200} {
		for _, delta := range []int{-1, 0, 2500} {
			var end *time.Time
			if delta != 0 {
				x := start.Add(time.Duration(delta) * time.Millisecond)
				end = &x
			}
			reader := &historyReader{t: t, size: size, values: []any{"run", "tenant", "id", "restart", "running", "running", 100, "done", start, end, "姓名"}}
			items, total, err := loadDeploymentRuns(context.Background(), reader, "tenant", "id", PageFilter{Page: 2, PageSize: 20})
			if err != nil || total != int64(size) || len(items) != size || reader.calls != 3 || reader.active {
				t.Fatalf("固定查询失败 %+v %v", reader, err)
			}
			if !reflect.DeepEqual(reader.args, []any{"tenant", "id", 20, 20}) || !strings.Contains(reader.query, "ORDER BY r.started_at DESC,r.id DESC") || !strings.Contains(reader.query, "u.tenant_id=r.tenant_id") {
				t.Fatal("排序/分页/用户租户过滤不正确")
			}
			for _, item := range items {
				if delta == 0 {
					if item.DurationMs != nil {
						t.Fatal("未完成耗时必须null")
					}
				} else {
					want := int64(delta)
					if want < 0 {
						want = 0
					}
					if item.DurationMs == nil || *item.DurationMs != want {
						t.Fatal("完成耗时必须使用真实时间差")
					}
				}
			}
		}
	}
	reader := &historyReader{t: t, missing: true}
	if _, _, err := loadDeploymentRuns(context.Background(), reader, "tenant", "id", PageFilter{}); !errors.Is(err, ErrNotFound) || reader.calls != 1 {
		t.Fatal("不存在/越租户必须在读历史前拒绝")
	}
	schema, err := os.ReadFile("../../db/schema/core-schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"full_name text", "created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT", "operation text NOT NULL CHECK (operation IN ('deploy', 'start', 'stop', 'restart', 'delete'))"} {
		if !strings.Contains(string(schema), field) {
			t.Fatalf("正式字段假设变化：%s", field)
		}
	}
}
func TestRunEventsPageQueriesScopeAndRowsErrors(t *testing.T) {
	for _, missing := range []bool{false, true} {
		reader := &historyReader{t: t, size: 2, missing: missing, values: []any{"event", "id", "observed", "ready", time.Now()}}
		items, total, err := loadRunEventsPage(context.Background(), reader, "tenant", "id", PageFilter{Page: 3, PageSize: 20})
		if missing {
			if !errors.Is(err, ErrNotFound) || reader.calls != 1 {
				t.Fatal("越租户run可读")
			}
			continue
		}
		if err != nil || len(items) != 2 || total != 2 || reader.calls != 3 || reader.active || !strings.Contains(reader.query, "ORDER BY e.created_at ASC,e.id ASC") || !reflect.DeepEqual(reader.args, []any{"tenant", "id", 20, 40}) {
			t.Fatalf("事件分页错误 %+v %v", reader, err)
		}
	}
	reader := &historyReader{t: t, rowErr: errors.New("read failed")}
	if _, _, err := loadRunEventsPage(context.Background(), reader, "tenant", "id", PageFilter{}); err == nil || reader.active {
		t.Fatal("rows错误被吞或未释放")
	}
}

type historyRepository struct {
	Repository
	called     bool
	tenant, id string
	filter     PageFilter
}

func (r *historyRepository) ListDeploymentRuns(_ context.Context, tenant, id string, f PageFilter) ([]DeploymentRunHistory, int64, error) {
	r.called = true
	r.tenant = tenant
	r.id = id
	r.filter = f
	return []DeploymentRunHistory{{DeploymentRun: DeploymentRun{ID: "run", Operation: "deploy"}, ActorDisplayName: "操作员"}}, 41, nil
}
func (r *historyRepository) ListRunEventsPage(_ context.Context, tenant, id string, f PageFilter) ([]DeploymentRunEvent, int64, error) {
	r.called = true
	r.tenant = tenant
	r.id = id
	r.filter = f
	return []DeploymentRunEvent{{ID: "event"}}, 41, nil
}
func TestHistoryHTTPPermissionsTenantPaginationAndPayload(t *testing.T) {
	for _, path := range []string{"/api/v1/ops/project-deployments/id/runs", "/api/v1/ops/deployment-runs/id/events/page"} {
		for _, role := range []string{"OPS_ADMIN", "USER_ADMIN"} {
			repo := &historyRepository{}
			h := NewHandler(NewService(repo, nil), nil)
			router := chi.NewRouter()
			h.MountRoutes(router)
			req := httptest.NewRequest("GET", path+"?page=2&limit=1000&tenantId=other", nil)
			req = req.WithContext(auth.WithUser(req.Context(), auth.User{TenantID: "tenant", Role: role}))
			out := httptest.NewRecorder()
			router.ServeHTTP(out, req)
			if role == "USER_ADMIN" {
				if repo.called {
					t.Fatal("权限不足仍查询历史")
				}
				continue
			}
			if !repo.called || repo.tenant != "tenant" || repo.id != "id" || repo.filter.Page != 2 || repo.filter.PageSize != 200 {
				t.Fatalf("scope/page错误 %+v", repo)
			}
			var body struct {
				Code int `json:"code"`
				Data struct {
					List       []map[string]any                 `json:"list"`
					Pagination struct{ Page, Limit, Total int } `json:"pagination"`
				} `json:"data"`
			}
			if err := json.Unmarshal(out.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Code != 0 || len(body.Data.List) != 1 || body.Data.Pagination.Limit != 200 || body.Data.Pagination.Total != 41 {
				t.Fatalf("不符合分页包络 %s", out.Body.String())
			}
			if strings.HasSuffix(path, "/runs") && (body.Data.List[0]["actorDisplayName"] != "操作员" || body.Data.List[0]["durationMs"] != nil) {
				t.Fatal("history扩展字段错误")
			}
		}
	}
	f := historyPage(httptest.NewRequest("GET", "/?page=-3&limit=bad", nil))
	if f.Page != 1 || f.PageSize != 20 {
		t.Fatal("默认分页错误")
	}
}
