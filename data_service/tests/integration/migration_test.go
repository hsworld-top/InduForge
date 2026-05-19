package integration_test

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/indu-forge/data_service/internal/db/migrate"
	"github.com/indu-forge/data_service/internal/db/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	testDatabaseURLEnv = "DATA_SERVICE_TEST_DATABASE_URL"
	postgresTestImage  = "postgres:16.4-alpine"
)

func TestMigrateUp_CreatesCoreTables(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupMigrator(t, fixture.pool)

	start := make(chan struct{})
	errCh := make(chan error, 2)
	var wg sync.WaitGroup

	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errCh <- migrator.Up(ctx)
		}()
	}
	close(start)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("骞跺彂鎵ц Up 澶辫触: %v", err)
		}
	}

	for _, tableName := range []string{
		"data_connections",
		"data_relational_configs",
		"data_queries",
		"data_points",
		"data_mqtt_configs",
		"data_mqtt_subscriptions",
		"data_mqtt_messages",
		"data_kafka_configs",
		"data_http_configs",
		"data_websocket_configs",
		"data_redis_configs",
		"data_opcua_configs",
		"data_s7_configs",
		"data_modbus_configs",
		"data_tdengine_configs",
		"data_preview_sessions",
		"data_compute_folders",
		"data_compute_units",
		"data_compute_runs",
	} {
		if !tableExists(ctx, t, fixture.pool, fixture.schemaName, tableName) {
			t.Fatalf("expected table %s to exist", tableName)
		}
	}

	var appliedCount int
	if err := fixture.pool.QueryRow(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&appliedCount); err != nil {
		t.Fatalf("鏌ヨ schema_migrations 澶辫触: %v", err)
	}
	if appliedCount != 16 {
		t.Fatalf("expected 16 migration records, got %d", appliedCount)
	}

	if err := migrator.DownAll(ctx); err != nil {
		t.Fatalf("鎵ц DownAll 澶辫触: %v", err)
	}

	for _, tableName := range []string{
		"data_connections",
		"data_relational_configs",
		"data_queries",
		"data_points",
		"data_mqtt_configs",
		"data_mqtt_subscriptions",
		"data_mqtt_messages",
		"data_kafka_configs",
		"data_http_configs",
		"data_websocket_configs",
		"data_redis_configs",
		"data_opcua_configs",
		"data_s7_configs",
		"data_modbus_configs",
		"data_tdengine_configs",
		"data_preview_sessions",
		"data_compute_folders",
		"data_compute_units",
		"data_compute_runs",
	} {
		if tableExists(ctx, t, fixture.pool, fixture.schemaName, tableName) {
			t.Fatalf("鏈熸湜琛?%s 宸茶鍒犻櫎", tableName)
		}
	}
}

func TestMigrationIndexes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupMigrator(t, fixture.pool)

	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("鎵ц Up 澶辫触: %v", err)
	}

	indexes := loadIndexNames(ctx, t, fixture.pool, fixture.schemaName)
	for _, indexName := range []string{
		"data_connections_project_type_idx",
		"data_connections_project_status_idx",
		"data_relational_configs_connection_id_key",
		"data_relational_configs_db_type_idx",
		"data_relational_configs_ssl_config_gin_idx",
		"data_queries_project_name_key",
		"data_queries_project_connection_idx",
		"data_queries_connection_project_idx",
		"data_queries_type_enabled_idx",
		"data_queries_config_gin_idx",
		"data_points_project_path_key",
		"data_points_project_status_idx",
		"data_points_source_idx",
		"data_points_source_config_gin_idx",
		"data_points_tags_gin_idx",
		"data_mqtt_configs_protocol_idx",
		"data_mqtt_subscriptions_project_name_key",
		"data_mqtt_subscriptions_project_connection_idx",
		"data_mqtt_subscriptions_connection_enabled_idx",
		"data_mqtt_messages_project_subscription_received_idx",
		"data_mqtt_messages_subscription_received_idx",
		"data_kafka_configs_topic_idx",
		"data_http_configs_method_idx",
		"data_websocket_configs_url_idx",
		"data_redis_configs_mode_idx",
		"data_opcua_configs_endpoint_idx",
		"data_s7_configs_host_idx",
		"data_modbus_configs_mode_idx",
		"data_tdengine_configs_database_idx",
		"data_preview_sessions_project_user_status_idx",
		"data_preview_sessions_last_active_at_idx",
		"data_compute_units_project_name_key",
		"data_compute_units_project_enabled_idx",
		"data_compute_units_project_language_idx",
		"data_compute_units_project_folder_idx",
		"data_compute_folders_project_parent_idx",
		"data_compute_folders_project_root_name_key",
		"data_compute_folders_project_parent_name_key",
		"data_compute_runs_unit_created_idx",
		"data_compute_runs_project_created_idx",
	} {
		if _, ok := indexes[indexName]; !ok {
			t.Fatalf("鏈熸湜绱㈠紩/绾︽潫绱㈠紩 %s 瀛樺湪锛屽綋鍓嶇储寮曢泦鍚堜负 %v", indexName, mapsKeys(indexes))
		}
	}
}

func setupMigrator(t *testing.T, pool *pgxpool.Pool) *migrate.Migrator {
	t.Helper()

	migrator, err := migrate.NewMigrator(pool)
	if err != nil {
		t.Fatalf("鍒涘缓杩佺Щ鍣ㄥけ璐? %v", err)
	}

	return migrator
}

type testDatabase struct {
	pool        *pgxpool.Pool
	adminPool   *pgxpool.Pool
	schemaName  string
	databaseURL string
}

func setupTestDatabase(t *testing.T, ctx context.Context) *testDatabase {
	t.Helper()

	databaseURL, cleanup := resolveTestDatabaseURL(t, ctx)
	t.Cleanup(cleanup)

	adminPool, err := postgres.NewPoolFromURL(ctx, databaseURL)
	if err != nil {
		t.Fatalf("鍒涘缓娴嬭瘯绠＄悊杩炴帴姹犲け璐? %v", err)
	}
	t.Cleanup(adminPool.Close)

	schemaName := uniqueSchemaName(t.Name())
	if _, err := adminPool.Exec(ctx, fmt.Sprintf(`CREATE SCHEMA %s`, pgx.Identifier{schemaName}.Sanitize())); err != nil {
		t.Fatalf("鍒涘缓娴嬭瘯 schema 澶辫触: %v", err)
	}
	t.Cleanup(func() {
		dropCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, err := adminPool.Exec(dropCtx, fmt.Sprintf(`DROP SCHEMA IF EXISTS %s CASCADE`, pgx.Identifier{schemaName}.Sanitize())); err != nil {
			t.Fatalf("鍒犻櫎娴嬭瘯 schema 澶辫触: %v", err)
		}
	})

	pool, err := postgres.NewPool(ctx, postgres.PoolConfig{
		DatabaseURL: databaseURL,
		SearchPath:  schemaName,
	})
	if err != nil {
		t.Fatalf("鍒涘缓娴嬭瘯杩炴帴姹犲け璐? %v", err)
	}
	t.Cleanup(pool.Close)

	return &testDatabase{
		pool:        pool,
		adminPool:   adminPool,
		schemaName:  schemaName,
		databaseURL: databaseURL,
	}
}

func resolveTestDatabaseURL(t *testing.T, ctx context.Context) (string, func()) {
	t.Helper()

	if databaseURL := strings.TrimSpace(os.Getenv(testDatabaseURLEnv)); databaseURL != "" {
		return databaseURL, func() {}
	}

	container, err := tcpostgres.Run(
		ctx,
		postgresTestImage,
		tcpostgres.WithDatabase("data_service_test"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp").WithStartupTimeout(90*time.Second),
		),
	)
	if err != nil {
		t.Skipf(
			"%s is not set and PostgreSQL testcontainer cannot be started: %v. Start Docker Desktop or set %s to a writable test database.",
			testDatabaseURLEnv,
			err,
			testDatabaseURLEnv,
		)
	}

	databaseURL, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		t.Fatalf("鑾峰彇 testcontainer 杩炴帴涓插け璐? %v", err)
	}

	cleanup := func() {
		_ = testcontainers.TerminateContainer(container)
	}

	return databaseURL, cleanup
}

func tableExists(ctx context.Context, t *testing.T, pool *pgxpool.Pool, schemaName string, tableName string) bool {
	t.Helper()

	var regclass *string
	if err := pool.QueryRow(ctx, `SELECT to_regclass($1)`, schemaName+"."+tableName).Scan(&regclass); err != nil {
		t.Fatalf("鏌ヨ琛?%s 鏄惁瀛樺湪澶辫触: %v", tableName, err)
	}

	return regclass != nil && *regclass != ""
}

func loadIndexNames(ctx context.Context, t *testing.T, pool *pgxpool.Pool, schemaName string) map[string]struct{} {
	t.Helper()

	rows, err := pool.Query(ctx, `
        SELECT indexname
        FROM pg_indexes
        WHERE schemaname = $1
          AND tablename IN (
              'data_connections',
              'data_relational_configs',
              'data_queries',
              'data_points',
              'data_mqtt_configs',
              'data_mqtt_subscriptions',
              'data_mqtt_messages',
              'data_kafka_configs',
              'data_http_configs',
              'data_websocket_configs',
              'data_redis_configs',
              'data_opcua_configs',
              'data_s7_configs',
              'data_modbus_configs',
              'data_tdengine_configs',
              'data_preview_sessions',
              'data_compute_units',
              'data_compute_runs'
          )
    `, schemaName)
	if err != nil {
		t.Fatalf("鏌ヨ绱㈠紩鍒楄〃澶辫触: %v", err)
	}
	defer rows.Close()

	indexes := make(map[string]struct{})
	for rows.Next() {
		var indexName string
		if err := rows.Scan(&indexName); err != nil {
			t.Fatalf("璇诲彇绱㈠紩鍚嶇О澶辫触: %v", err)
		}
		indexes[indexName] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("閬嶅巻绱㈠紩鍒楄〃澶辫触: %v", err)
	}

	return indexes
}

func mapsKeys(items map[string]struct{}) []string {
	result := make([]string, 0, len(items))
	for key := range items {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func uniqueSchemaName(testName string) string {
	replacer := strings.NewReplacer("/", "_", "-", "_", " ", "_")
	return fmt.Sprintf("it_%s_%d", replacer.Replace(strings.ToLower(testName)), time.Now().UnixNano())
}
