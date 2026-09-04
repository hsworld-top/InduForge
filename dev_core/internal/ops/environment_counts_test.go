package ops

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestEnvironmentCountsProjectionAndPayload(t *testing.T) {
	now := time.Now()
	var recent *time.Time
	item, err := scanRuntimeEnvironment(lifecycleRow{values: []any{"env", "tenant", "name", "code", "active", false, "available", 2, 2, 8, 8, 6, 2, "change", "operator", recent, now, now}})
	if err != nil {
		t.Fatal(err)
	}
	payload := runtimeEnvironmentPayload(item)
	if payload["projectCount"] != 6 || payload["runningDeploymentCount"] != 2 {
		t.Fatalf("错位计数: %+v", payload)
	}
	for _, fragment := range []string{"d.tenant_id=$1 AND d.deleted_at IS NULL", "s.tenant_id=d.tenant_id", "s.observed_generation=s.desired_generation", "deployment.has_running_service", "deployment.services_converged"} {
		if !strings.Contains(runtimeEnvironmentDeploymentCountsSQL, fragment) {
			t.Fatalf("缺少范围/收敛条件 %s", fragment)
		}
	}
	if strings.Count(runtimeEnvironmentProjectionSQL, runtimeEnvironmentDeploymentCountsSQL) != 1 {
		t.Fatal("计数必须单次批量聚合")
	}
}

// 真实 PostgreSQL 只读 CTE 覆盖租户、删除、停用历史服务和换代，绝不创建或更改表。
func TestEnvironmentCountsPostgreSQL(t *testing.T) {
	dsn := os.Getenv("INDUFORGE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("未配置 PostgreSQL 集成测试 DSN")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	const fixture = `WITH project_deployments(id,tenant_id,environment_id,desired_status,observed_status,deleted_at) AS (VALUES
 ('ok','tenant','a','running','running',NULL::timestamp),
 ('stopped','tenant','a','stopped','stopped',NULL::timestamp),
 ('failed','tenant','a','running','failed',NULL::timestamp),
 ('old','tenant','a','running','running',NULL::timestamp),
 ('empty','tenant','a','running','running',NULL::timestamp),
 ('deleted','tenant','a','running','running',now()),
 ('other','other','a','running','running',NULL::timestamp),
 ('envb','tenant','b','running','running',NULL::timestamp)),
 deployment_services(project_deployment_id,tenant_id,desired_status,observed_status,desired_generation,observed_generation) AS (VALUES
 ('ok','tenant','running','running',3,3),('ok','tenant','stopped','stopped',2,2),
 ('stopped','tenant','stopped','stopped',3,3),('failed','tenant','running','failed',3,2),
 ('old','tenant','running','running',3,2),('deleted','tenant','running','running',3,3),
 ('other','other','running','running',3,3),('envb','tenant','running','running',3,3),
 ('ok','other','running','failed',3,2)) `
	rows, err := conn.Query(ctx, fixture+runtimeEnvironmentDeploymentCountsSQL+` ORDER BY deployment.environment_id`, "tenant")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	want := map[string][2]int{"a": {5, 1}, "b": {1, 1}}
	for rows.Next() {
		var tenant, env string
		var total, running int
		if err = rows.Scan(&tenant, &env, &total, &running); err != nil {
			t.Fatal(err)
		}
		expected, ok := want[env]
		if !ok || tenant != "tenant" || expected != [2]int{total, running} {
			t.Fatalf("tenant=%s env=%s counts=%d/%d", tenant, env, total, running)
		}
		delete(want, env)
	}
	if rows.Err() != nil || len(want) > 0 {
		t.Fatalf("missing=%v err=%v", want, rows.Err())
	}
}
