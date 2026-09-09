package integration_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/indu-forge/data_service/internal/config"
	"github.com/jackc/pgx/v5"
)

func TestEmptyStartupDoesNotCreateProjectDemoData(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	fixture := setupTestDatabase(t, ctx)
	srv, err := newIntegrationServer(t, config.Config{
		DataServiceInternalToken: "integration-test-internal-token", Addr: ":0",
		DatabaseURL: fixture.databaseURL, DatabaseSearchPath: fixture.schemaName,
		JWTSecret: "empty-startup-secret", ConnectionSecretKey: []byte("0123456789abcdef0123456789abcdef"), ConnectionSecretKeyVersion: "v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(srv.Close)
	// 启动只应建立结构；工程所属连接、查询、点位和教程资源须由显式业务操作创建。
	for _, table := range []string{"data_connections", "data_queries", "data_points", "data_mqtt_subscriptions", "data_compute_units", "data_alarm_items", "data_realtime_keys"} {
		var count int
		if err := fixture.pool.QueryRow(ctx, fmt.Sprintf("SELECT count(*) FROM %s", pgx.Identifier{table}.Sanitize())).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Errorf("empty startup populated %s: got %d rows", table, count)
		}
	}
}
