package ops

import (
	"context"
	"github.com/jackc/pgx/v5"
	"os"
	"testing"
	"time"
)

func TestAgentStoppedDeploymentCommandAccessPostgreSQL(t *testing.T) {
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
	for _, tc := range []struct {
		deployment, service string
		command, want       bool
	}{
		{"running", "running", false, true}, {"running", "running", true, true},
		{"stopped", "stopped", true, true}, {"stopped", "stopped", false, false},
		{"stopped", "running", true, false}, {"invalid", "stopped", true, false},
	} {
		var allowed bool
		err = conn.QueryRow(ctx, `SELECT `+agentDeploymentStatePredicate+` FROM (SELECT $1::text AS desired_status) d CROSS JOIN (SELECT $2::text AS desired_status) s WHERE $3::text='node' AND $4::text='token' AND ($5::boolean OR NOT $5::boolean)`, tc.deployment, tc.service, "node", "token", tc.command).Scan(&allowed)
		if err != nil || allowed != tc.want {
			t.Fatalf("deployment=%s service=%s command=%v allowed=%v err=%v", tc.deployment, tc.service, tc.command, allowed, err)
		}
	}
}
