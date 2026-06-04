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
		"data_kafka_topic_groups",
		"data_kafka_topic_mappings",
		"data_kafka_field_groups",
		"data_kafka_fields",
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
		"data_alarm_policy_groups",
		"data_alarm_policies",
		"data_realtime_keys",
		"data_workbench_object_groups",
		"data_table_group_members",
	} {
		if !tableExists(ctx, t, fixture.pool, fixture.schemaName, tableName) {
			t.Fatalf("expected table %s to exist", tableName)
		}
	}
	for _, tableName := range []string{
		"data_builtin_message_topics",
		"data_builtin_message_variables",
	} {
		if tableExists(ctx, t, fixture.pool, fixture.schemaName, tableName) {
			t.Fatalf("expected legacy table %s to be removed", tableName)
		}
	}

	var appliedCount int
	if err := fixture.pool.QueryRow(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&appliedCount); err != nil {
		t.Fatalf("鏌ヨ schema_migrations 澶辫触: %v", err)
	}
	if appliedCount != 44 {
		t.Fatalf("expected 44 migration records, got %d", appliedCount)
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
		"data_kafka_topic_groups",
		"data_kafka_topic_mappings",
		"data_kafka_field_groups",
		"data_kafka_fields",
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
		"data_alarm_policy_groups",
		"data_alarm_policies",
		"data_workbench_object_groups",
		"data_table_group_members",
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
		"data_mqtt_subscriptions_project_connection_name_key",
		"data_mqtt_subscriptions_project_connection_idx",
		"data_mqtt_subscriptions_connection_order_idx",
		"data_mqtt_messages_project_subscription_received_idx",
		"data_mqtt_messages_subscription_received_idx",
		"data_kafka_configs_topic_idx",
		"data_kafka_topic_groups_tree_idx",
		"data_kafka_topic_mappings_group_idx",
		"data_kafka_field_groups_tree_idx",
		"data_kafka_fields_group_idx",
		"data_kafka_fields_connection_idx",
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
		"data_alarm_policy_groups_project_sort_idx",
		"data_alarm_policy_groups_project_parent_idx",
		"data_alarm_policy_groups_project_root_name_key",
		"data_alarm_policy_groups_project_parent_name_key",
		"data_alarm_policies_project_group_idx",
		"data_alarm_policies_project_updated_idx",
		"data_alarm_policies_project_enabled_idx",
		"data_workbench_object_groups_name_key",
		"data_workbench_object_groups_connection_scope_idx",
		"data_queries_project_connection_group_idx",
		"data_table_group_members_group_idx",
	} {
		if _, ok := indexes[indexName]; !ok {
			t.Fatalf("鏈熸湜绱㈠紩/绾︽潫绱㈠紩 %s 瀛樺湪锛屽綋鍓嶇储寮曢泦鍚堜负 %v", indexName, mapsKeys(indexes))
		}
	}
}

func TestBuiltinRuntimeStoreMigration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupMigrator(t, fixture.pool)
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("migrate up failed: %v", err)
	}

	projectID := "11111111-1111-1111-1111-111111111111"
	userID := "22222222-2222-2222-2222-222222222222"
	storeTypes := []string{
		"builtin.relation",
		"builtin.timeseries",
		"builtin.realtime",
		"builtin.message",
	}
	for _, storeType := range storeTypes {
		_, err := fixture.pool.Exec(ctx, `
			INSERT INTO data_connections (project_id, name, type, category, status, metadata, created_by, updated_by)
			VALUES ($1, $2, $3, 'builtin', 'connected', '{}'::jsonb, $4, $4)
		`, projectID, storeType, storeType, userID)
		if err != nil {
			t.Fatalf("insert builtin type %s failed: %v", storeType, err)
		}
	}

	_, err := fixture.pool.Exec(ctx, `
		INSERT INTO data_connections (project_id, name, type, category, status, metadata, created_by, updated_by)
		VALUES ($1, 'repeat relation', 'builtin.relation', 'builtin', 'connected', '{}'::jsonb, $2, $2)
	`, projectID, userID)
	if err != nil {
		t.Fatalf("expected duplicate builtin relation type to be allowed: %v", err)
	}

	indexes := loadIndexNames(ctx, t, fixture.pool, fixture.schemaName)
	if _, ok := indexes["data_connections_project_builtin_type_unique"]; ok {
		t.Fatalf("expected legacy builtin type unique index to be removed")
	}

	rows, err := fixture.pool.Query(ctx, `
		SELECT type, metadata
		FROM data_connections
		WHERE project_id = $1
		  AND type IN ('builtin.relation', 'builtin.timeseries', 'builtin.realtime', 'builtin.message')
	`, projectID)
	if err != nil {
		t.Fatalf("query builtin connections failed: %v", err)
	}
	defer rows.Close()

	seen := map[string]bool{}
	for rows.Next() {
		var storeType string
		var metadata map[string]any
		if err := rows.Scan(&storeType, &metadata); err != nil {
			t.Fatalf("scan builtin connection failed: %v", err)
		}
		runtimeKey, _ := metadata["runtimeKey"].(string)
		if runtimeKey == "" {
			t.Fatalf("expected runtimeKey for %s, got %#v", storeType, metadata)
		}
		seen[storeType] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate builtin connections failed: %v", err)
	}
	for _, storeType := range storeTypes {
		if !seen[storeType] {
			t.Fatalf("expected migrated builtin connection %s", storeType)
		}
	}
}

func TestAlarmRuleFinalModelMigration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupMigrator(t, fixture.pool)
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("migrate up failed: %v", err)
	}

	suppressionType, suppressionDefault := loadColumnTypeAndDefault(ctx, t, fixture.pool, fixture.schemaName, "data_alarm_rules", "suppression")
	if suppressionType != "jsonb" {
		t.Fatalf("suppression data_type = %q, want jsonb", suppressionType)
	}
	if !strings.Contains(suppressionDefault, "'{}'::jsonb") {
		t.Fatalf("suppression default = %q, want empty jsonb object", suppressionDefault)
	}

	messageTemplateType, messageTemplateDefault := loadColumnTypeAndDefault(ctx, t, fixture.pool, fixture.schemaName, "data_alarm_rules", "message_template")
	if messageTemplateType != "text" {
		t.Fatalf("message_template data_type = %q, want text", messageTemplateType)
	}
	if !strings.Contains(messageTemplateDefault, "''") {
		t.Fatalf("message_template default = %q, want empty string", messageTemplateDefault)
	}

	assertInsertAlarmRuleWithTypeAndSeverity(ctx, t, fixture.pool, "H", "major")
	assertInsertAlarmRuleWithTypeAndSeverity(ctx, t, fixture.pool, "cel", "critical")

	if _, err := fixture.pool.Exec(ctx, `
        INSERT INTO data_alarm_rules (
            project_id,
            name,
            target_path,
            rule_type,
            severity,
            created_by
        )
        VALUES (gen_random_uuid(), '非法规则类型', 'metrics.bad_type', 'threshold', 'warning', gen_random_uuid())
    `); err == nil {
		t.Fatalf("expected old rule_type to violate check constraint")
	}

	indexes := loadIndexNames(ctx, t, fixture.pool, fixture.schemaName)
	if _, ok := indexes["data_alarm_rules_project_updated_idx"]; !ok {
		t.Fatalf("expected data_alarm_rules_project_updated_idx to exist, got %v", mapsKeys(indexes))
	}
}

func TestAlarmPolicyTablesMigration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupMigrator(t, fixture.pool)
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("migrate up failed: %v", err)
	}

	assertColumnExists(ctx, t, fixture.pool, fixture.schemaName, "data_alarm_policy_groups", "is_enabled", "boolean")
	assertColumnExists(ctx, t, fixture.pool, fixture.schemaName, "data_alarm_policy_groups", "parent_id", "uuid")
	assertColumnExists(ctx, t, fixture.pool, fixture.schemaName, "data_alarm_policies", "group_id", "uuid")
	assertColumnExists(ctx, t, fixture.pool, fixture.schemaName, "data_alarm_policies", "targets", "jsonb")
	assertColumnExists(ctx, t, fixture.pool, fixture.schemaName, "data_alarm_policies", "conditions", "jsonb")
	assertColumnExists(ctx, t, fixture.pool, fixture.schemaName, "data_alarm_policies", "contract", "jsonb")

	if _, err := fixture.pool.Exec(ctx, `
        INSERT INTO data_alarm_policies (
            project_id,
            name,
            mode,
            created_by
        )
        VALUES (gen_random_uuid(), '非法策略模式', 'single_rule', gen_random_uuid())
    `); err == nil {
		t.Fatalf("expected invalid policy mode to violate check constraint")
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

func loadColumnTypeAndDefault(ctx context.Context, t *testing.T, pool *pgxpool.Pool, schemaName string, tableName string, columnName string) (string, string) {
	t.Helper()

	var dataType string
	var columnDefault *string
	if err := pool.QueryRow(ctx, `
        SELECT data_type, column_default
        FROM information_schema.columns
        WHERE table_schema = $1 AND table_name = $2 AND column_name = $3
    `, schemaName, tableName, columnName).Scan(&dataType, &columnDefault); err != nil {
		t.Fatalf("expected column %s.%s: %v", tableName, columnName, err)
	}

	if columnDefault == nil {
		return dataType, ""
	}
	return dataType, *columnDefault
}

func assertColumnExists(ctx context.Context, t *testing.T, pool *pgxpool.Pool, schemaName string, tableName string, columnName string, wantType string) {
	t.Helper()

	dataType, _ := loadColumnTypeAndDefault(ctx, t, pool, schemaName, tableName, columnName)
	if dataType != wantType {
		t.Fatalf("column %s.%s data_type = %q, want %q", tableName, columnName, dataType, wantType)
	}
}

func assertInsertAlarmRuleWithTypeAndSeverity(ctx context.Context, t *testing.T, pool *pgxpool.Pool, ruleType string, severity string) {
	t.Helper()

	if _, err := pool.Exec(ctx, `
        INSERT INTO data_alarm_rules (
            project_id,
            name,
            target_path,
            rule_type,
            severity,
            created_by
        )
        VALUES (gen_random_uuid(), $1, $2, $3, $4, gen_random_uuid())
    `, "规则"+ruleType+severity, "metrics."+strings.ToLower(ruleType)+"."+severity, ruleType, severity); err != nil {
		t.Fatalf("insert alarm rule ruleType=%s severity=%s failed: %v", ruleType, severity, err)
	}
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
              'data_kafka_topic_groups',
              'data_kafka_topic_mappings',
              'data_kafka_field_groups',
              'data_kafka_fields',
              'data_http_configs',
              'data_websocket_configs',
              'data_redis_configs',
              'data_opcua_configs',
              'data_s7_configs',
              'data_modbus_configs',
              'data_tdengine_configs',
              'data_preview_sessions',
              'data_compute_units',
              'data_compute_runs',
              'data_alarm_rules',
              'data_alarm_policy_groups',
              'data_alarm_policies'
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
