package ops

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// 仅在显式测试 DSN 下以只读 CTE 验证真实 PostgreSQL 参数绑定和分页语义，不建表、不改业务数据。
func TestDeploymentEnvironmentFilterPostgreSQLPagination(t *testing.T) {
	dsn := os.Getenv("INDUFORGE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("未配置 PostgreSQL 集成测试 DSN")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connection, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close(ctx)
	const fixture = `WITH d(id,tenant_id,project_id,environment_id,deleted_at) AS (VALUES
('a','tenant','project','env-a',NULL::timestamp),('b','tenant','project','env-a',NULL::timestamp),
('c','tenant','project','env-b',NULL::timestamp),('d','other','project','env-a',NULL::timestamp),
('e','tenant','project','env-a',now()),('f','tenant','project','deleted-env',NULL::timestamp)),
p(id,tenant_id,name) AS (VALUES ('project','tenant','demo'),('project','other','demo')),
e(id,tenant_id,deleted_at) AS (VALUES ('env-a','tenant',NULL::timestamp),('env-b','tenant',NULL::timestamp),('env-a','other',NULL::timestamp),('deleted-env','tenant',now())) `
	const from = ` FROM d JOIN p ON p.id=d.project_id AND p.tenant_id=d.tenant_id JOIN e ON e.id=d.environment_id AND e.tenant_id=d.tenant_id WHERE `
	for _, test := range []struct {
		environment string
		page        int
		wantIDs     []string
		wantTotal   int64
	}{
		{environment: "env-a", page: 1, wantIDs: []string{"a"}, wantTotal: 2},
		{environment: "env-a", page: 2, wantIDs: []string{"b"}, wantTotal: 2},
		{environment: "env-b", page: 1, wantIDs: []string{"c"}, wantTotal: 1},
		{environment: "missing", page: 1, wantIDs: []string{}, wantTotal: 0},
	} {
		rows, err := connection.Query(ctx, fixture+`SELECT d.id`+from+deploymentListFilter+` ORDER BY d.id LIMIT $3 OFFSET $4`, "tenant", "demo", 1, test.page-1, "project", test.environment)
		if err != nil {
			t.Fatal(err)
		}
		ids := []string{}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				t.Fatal(err)
			}
			ids = append(ids, id)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		var total int64
		if err := connection.QueryRow(ctx, fixture+`SELECT count(*)`+from+deploymentCountFilter, "tenant", "demo", "project", test.environment).Scan(&total); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(ids, test.wantIDs) || total != test.wantTotal {
			t.Fatalf("env=%s page=%d ids=%v total=%d", test.environment, test.page, ids, total)
		}
	}
}
