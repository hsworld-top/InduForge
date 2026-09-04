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

type overviewTestReader struct {
	t     *testing.T
	calls int
	err   error
}

func (r *overviewTestReader) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	r.calls++
	if sql != environmentOverviewSQL || !reflect.DeepEqual(args, []any{"tenant", "environment"}) {
		r.t.Fatal("SQL或租户环境参数变化")
	}
	return lifecycleRow{err: r.err, values: []any{[]byte(`{"environmentId":"environment","snapshotAt":"2026-09-03T01:00:00Z","thresholds":{"freshnessSeconds":45,"capacityAttentionPercent":85,"capacityCriticalPercent":95},"deployments":{"total":0},"nodes":{"maxDiskUsagePercent":null},"riskNodes":[]}`)}}
}
func TestEnvironmentOverviewSingleQueryAndUnknownNull(t *testing.T) {
	reader := &overviewTestReader{t: t}
	out, err := loadEnvironmentOverview(context.Background(), reader, "tenant", "environment")
	if err != nil || reader.calls != 1 || out.Nodes.MaxDiskUsagePercent != nil || len(out.RiskNodes) != 0 || out.Thresholds.CapacityCriticalPercent != 95 {
		t.Fatalf("单次快照/NULL错误 %+v %v", out, err)
	}
	reader = &overviewTestReader{t: t, err: pgx.ErrNoRows}
	if _, err = loadEnvironmentOverview(context.Background(), reader, "tenant", "environment"); !errors.Is(err, ErrNotFound) {
		t.Fatal("无环境/跨租户必须相同未找到")
	}
	for _, part := range []string{"tenant_id=$1 AND id=$2 AND deleted_at IS NULL", "n.tenant_id=env.tenant_id", "en.tenant_id=env.tenant_id", "cn.tenant_id=env.tenant_id", "d.tenant_id=env.tenant_id AND d.deleted_at IS NULL", "s.tenant_id=d.tenant_id", "n.tenant_id=d.tenant_id", "jsonb_typeof", "raw_disk BETWEEN 0 AND 100", "ORDER BY priority,disk DESC NULLS LAST,id LIMIT 5", "snapshotAt',now()", "observed_generation=s.desired_generation"} {
		if !strings.Contains(environmentOverviewSQL, part) {
			t.Fatalf("缺少安全/全局排序约束 %s", part)
		}
	}
}

type overviewRepository struct {
	Repository
	called     bool
	tenant, id string
}

func (r *overviewRepository) GetEnvironmentOverview(_ context.Context, tenant, id string) (EnvironmentOverview, error) {
	r.called = true
	r.tenant = tenant
	r.id = id
	return EnvironmentOverview{EnvironmentID: id, RiskNodes: []OverviewRiskNode{}}, nil
}
func TestEnvironmentOverviewHTTPPermissionAndTenant(t *testing.T) {
	_ = httptest.NewRequest("GET", "/api/v1/ops/runtime-environments/11111111-1111-4111-8111-111111111111/overview", nil)
	const environmentID = "11111111-1111-4111-8111-111111111111"
	for _, role := range []string{"OPS_ADMIN", "USER_ADMIN"} {
		repo := &overviewRepository{}
		h := NewHandler(NewService(repo, nil), nil)
		router := chi.NewRouter()
		h.MountRoutes(router)
		req := httptest.NewRequest("GET", "/api/v1/ops/runtime-environments/"+environmentID+"/overview?tenantId=other", nil)
		req = req.WithContext(auth.WithUser(req.Context(), auth.User{TenantID: "tenant", Role: role}))
		out := httptest.NewRecorder()
		router.ServeHTTP(out, req)
		if role == "USER_ADMIN" {
			if repo.called {
				t.Fatal("无环境权限读取概览")
			}
			continue
		}
		if !repo.called || repo.tenant != "tenant" || repo.id != environmentID || out.Code != 200 {
			t.Fatal("读取scope错误")
		}
	}
	repo := &overviewRepository{}
	h := NewHandler(NewService(repo, nil), nil)
	router := chi.NewRouter()
	h.MountRoutes(router)
	req := httptest.NewRequest("GET", "/api/v1/ops/runtime-environments/not-a-uuid/overview", nil)
	req = req.WithContext(auth.WithUser(req.Context(), auth.User{TenantID: "tenant", Role: "OPS_ADMIN"}))
	out := httptest.NewRecorder()
	router.ServeHTTP(out, req)
	if repo.called || out.Code != 200 || !strings.Contains(out.Body.String(), `"code":20001`) {
		t.Fatalf("非法UUID必须业务invalid request而非PG错误 %s", out.Body.String())
	}
}

const environmentOverviewFixtureSQL = `WITH runtime_environments(id,tenant_id,deleted_at) AS (VALUES ('environment','tenant',NULL::timestamptz)),
 host_nodes_base(id,tenant_id,display_name,ip_address,desired_status,observed_status,last_heartbeat_at,resource_summary) AS (VALUES
 ('01','tenant','healthy','ip','active','online',now(),'{"cpu":{"usedPercent":10},"memory":{"usedPercent":20},"disk":{"usedPercent":30}}'::jsonb),
 ('02','tenant','stale','ip','active','online',now()-interval '60 seconds','{"cpu":{"usedPercent":10},"memory":{"usedPercent":20},"disk":{"usedPercent":99}}'::jsonb),
 ('03','tenant','offline','ip','active','offline',now(),'{"cpu":{"usedPercent":10},"memory":{"usedPercent":20},"disk":{"usedPercent":98}}'::jsonb),
 ('04','tenant','failed','ip','active','online',now(),'{}'::jsonb),
 ('05','tenant','unknown','ip','active','online',now(),'{"cpu":{"usedPercent":null},"disk":{"usedPercent":"95"}}'::jsonb),
 ('99','tenant','critical','ip','active','online',now(),'{"cpu":{"usedPercent":10},"memory":{"usedPercent":20},"disk":{"usedPercent":97}}'::jsonb)),
 host_nodes AS (SELECT *,display_name AS hostname,'linux'::text AS platform,'["project_entry","data_runtime"]'::jsonb AS capabilities FROM host_nodes_base),
 runtime_environment_nodes(environment_id,node_id,tenant_id) AS (SELECT 'environment',id,tenant_id FROM host_nodes),
 runtime_cluster_nodes(node_id,tenant_id,node_kind,cluster_status,desired_action,desired_generation,observed_generation) AS (SELECT id,tenant_id,'worker',CASE WHEN id='04' THEN 'failed' ELSE 'ready' END,'active',1,1 FROM host_nodes),
 project_deployments(id,tenant_id,environment_id,desired_status,observed_status,deleted_at) AS (VALUES
 ('running','tenant','environment','running','running',NULL::timestamptz),('stopped','tenant','environment','stopped','stopped',NULL::timestamptz),
 ('failed','tenant','environment','running','failed',NULL::timestamptz),('pending','tenant','environment','running','pending',NULL::timestamptz),
 ('stale','tenant','environment','running','running',NULL::timestamptz),('unknown','tenant','environment','running','pending',NULL::timestamptz),
 ('deleted','tenant','environment','running','running',now())),
 deployment_services(id,tenant_id,project_deployment_id,node_id,desired_status,observed_status,desired_generation,observed_generation) AS (VALUES
 ('s1','tenant','running','01','running','running',1,1),('s2','tenant','stopped','02','stopped','stopped',1,1),
 ('s3','tenant','failed','01','running','failed',1,0),('s4','tenant','pending','01','running','pending',2,1),
 ('s5','tenant','stale','02','running','running',1,1)) `

func TestEnvironmentOverviewPostgreSQLProjection(t *testing.T) {
	dsn := os.Getenv("INDUFORGE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("未配置PostgreSQL只读集成测试DSN")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	// 合并CTE清单，不创建或修改表。
	sql := strings.TrimSpace(environmentOverviewFixtureSQL) + "," + strings.TrimPrefix(environmentOverviewSQL, "WITH")
	var raw []byte
	if err = conn.QueryRow(ctx, sql, "tenant", "environment").Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var out EnvironmentOverview
	if err = json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out.Deployments != (OverviewDeploymentCounts{Total: 6, Running: 1, Stopped: 1, Failed: 1, Pending: 1, Stale: 1, Unknown: 1}) {
		t.Fatalf("部署分类不互斥 %+v", out.Deployments)
	}
	if out.Nodes.Total != 6 || out.Nodes.Online != 4 || out.Nodes.Offline != 2 || out.Nodes.Fault != 1 || out.Nodes.UnknownMetrics != 2 || out.Nodes.MaxDiskUsagePercent == nil || *out.Nodes.MaxDiskUsagePercent != 97 {
		t.Fatalf("全局指标错误 %+v", out.Nodes)
	}
	ids := []string{}
	for _, node := range out.RiskNodes {
		ids = append(ids, node.ID)
		if node.ID == "02" && node.DiskPercent != nil {
			t.Fatal("过期磁盘值不得使用")
		}
	}
	if !reflect.DeepEqual(ids, []string{"03", "04", "02", "99", "05"}) {
		t.Fatalf("先全局风险排序再取top5 %v", ids)
	}
	for _, state := range []string{"starting", "not-installed"} {
		fixture := strings.Replace(environmentOverviewFixtureSQL, "CASE WHEN id='04' THEN 'failed' ELSE 'ready' END", "CASE WHEN id='04' THEN 'failed' WHEN id='01' THEN '"+state+"' ELSE 'ready' END", 1)
		variant := strings.TrimSpace(fixture) + "," + strings.TrimPrefix(environmentOverviewSQL, "WITH")
		if err = conn.QueryRow(ctx, variant, "tenant", "environment").Scan(&raw); err != nil {
			t.Fatal(err)
		}
		if err = json.Unmarshal(raw, &out); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, node := range out.RiskNodes {
			if node.ID == "01" {
				found = true
				want := "pending"
				if state == "not-installed" {
					want = "unknown"
				}
				if node.Health != want {
					t.Fatal("运行组件未就绪不得健康")
				}
			}
		}
		if !found {
			t.Fatal("组件风险未进入全局前五")
		}
	}
	fixture := strings.ReplaceAll(environmentOverviewFixtureSQL, `"usedPercent":30`, `"usedPercent":94`)
	fixture = strings.ReplaceAll(fixture, `"usedPercent":97`, `"usedPercent":85`)
	variant := strings.TrimSpace(fixture) + "," + strings.TrimPrefix(environmentOverviewSQL, "WITH")
	if err = conn.QueryRow(ctx, variant, "tenant", "environment").Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out.RiskNodes[3].ID != "01" || out.RiskNodes[4].ID != "99" {
		t.Fatal("同级94%必须排在85%前")
	}
	fixture = strings.Replace(environmentOverviewFixtureSQL, "('s1','tenant','running','01','running','running',1,1)", "('s1','tenant','running','01','running','running',1,NULL)", 1)
	variant = strings.TrimSpace(fixture) + "," + strings.TrimPrefix(environmentOverviewSQL, "WITH")
	if err = conn.QueryRow(ctx, variant, "tenant", "environment").Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out.Deployments.Unknown != 2 || out.Deployments.Running != 0 {
		t.Fatal("缺失代次必须unknown")
	}
	if err = conn.QueryRow(ctx, sql, "other", "environment").Scan(&raw); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("跨租户泄漏")
	}
}
