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
	t.Parallel()

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
			t.Fatalf("并发执行 Up 失败: %v", err)
		}
	}

	for _, tableName := range []string{
		"data_connections",
		"data_relational_configs",
		"data_queries",
		"data_points",
	} {
		if !tableExists(ctx, t, fixture.pool, fixture.schemaName, tableName) {
			t.Fatalf("期望表 %s 已创建", tableName)
		}
	}

	var appliedCount int
	if err := fixture.pool.QueryRow(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&appliedCount); err != nil {
		t.Fatalf("查询 schema_migrations 失败: %v", err)
	}
	if appliedCount != 1 {
		t.Fatalf("期望已有 1 条 migration 记录，实际为 %d", appliedCount)
	}

	if err := migrator.DownAll(ctx); err != nil {
		t.Fatalf("执行 DownAll 失败: %v", err)
	}

	for _, tableName := range []string{
		"data_connections",
		"data_relational_configs",
		"data_queries",
		"data_points",
	} {
		if tableExists(ctx, t, fixture.pool, fixture.schemaName, tableName) {
			t.Fatalf("期望表 %s 已被删除", tableName)
		}
	}
}

func TestMigrationIndexes(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupMigrator(t, fixture.pool)

	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("执行 Up 失败: %v", err)
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
	} {
		if _, ok := indexes[indexName]; !ok {
			t.Fatalf("期望索引/约束索引 %s 存在，当前索引集合为 %v", indexName, mapsKeys(indexes))
		}
	}
}

func setupMigrator(t *testing.T, pool *pgxpool.Pool) *migrate.Migrator {
	t.Helper()

	migrator, err := migrate.NewMigrator(pool)
	if err != nil {
		t.Fatalf("创建迁移器失败: %v", err)
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
		t.Fatalf("创建测试管理连接池失败: %v", err)
	}
	t.Cleanup(adminPool.Close)

	schemaName := uniqueSchemaName(t.Name())
	if _, err := adminPool.Exec(ctx, fmt.Sprintf(`CREATE SCHEMA %s`, pgx.Identifier{schemaName}.Sanitize())); err != nil {
		t.Fatalf("创建测试 schema 失败: %v", err)
	}
	t.Cleanup(func() {
		dropCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, err := adminPool.Exec(dropCtx, fmt.Sprintf(`DROP SCHEMA IF EXISTS %s CASCADE`, pgx.Identifier{schemaName}.Sanitize())); err != nil {
			t.Fatalf("删除测试 schema 失败: %v", err)
		}
	})

	pool, err := postgres.NewPool(ctx, postgres.PoolConfig{
		DatabaseURL: databaseURL,
		SearchPath:  schemaName,
	})
	if err != nil {
		t.Fatalf("创建测试连接池失败: %v", err)
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
			"未设置 %s，且无法启动 PostgreSQL testcontainer: %v。可先启动 Docker Desktop，或显式设置 %s 指向可写测试库后重试。",
			testDatabaseURLEnv,
			err,
			testDatabaseURLEnv,
		)
	}

	databaseURL, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		t.Fatalf("获取 testcontainer 连接串失败: %v", err)
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
		t.Fatalf("查询表 %s 是否存在失败: %v", tableName, err)
	}

	return regclass != nil && *regclass != ""
}

func loadIndexNames(ctx context.Context, t *testing.T, pool *pgxpool.Pool, schemaName string) map[string]struct{} {
	t.Helper()

	rows, err := pool.Query(ctx, `
        SELECT indexname
        FROM pg_indexes
        WHERE schemaname = $1
          AND tablename IN ('data_connections', 'data_relational_configs', 'data_queries', 'data_points')
    `, schemaName)
	if err != nil {
		t.Fatalf("查询索引列表失败: %v", err)
	}
	defer rows.Close()

	indexes := make(map[string]struct{})
	for rows.Next() {
		var indexName string
		if err := rows.Scan(&indexName); err != nil {
			t.Fatalf("读取索引名称失败: %v", err)
		}
		indexes[indexName] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("遍历索引列表失败: %v", err)
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
