package migrate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const schemaMigrationsTable = "schema_migrations"

const createSchemaMigrationsSQL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version text PRIMARY KEY,
	name text NOT NULL,
	applied_at timestamptz NOT NULL DEFAULT now()
);
`

// Migration 表示一组 up/down SQL 文件。
type Migration struct {
	Version string
	Name    string
	UpSQL   string
	DownSQL string
}

// Migrator 负责按照版本顺序执行 SQL 迁移。
type Migrator struct {
	pool       *pgxpool.Pool
	migrations []Migration
}

// NewMigrator 使用默认 migrations 目录创建迁移器。
func NewMigrator(pool *pgxpool.Pool) (*Migrator, error) {
	return NewMigratorFromDir(pool, defaultMigrationsDir())
}

// NewMigratorFromDir 允许测试或调用方指定 migrations 目录。
func NewMigratorFromDir(pool *pgxpool.Pool, dir string) (*Migrator, error) {
	if pool == nil {
		return nil, fmt.Errorf("pool 不能为空")
	}

	migrations, err := loadMigrations(dir)
	if err != nil {
		return nil, err
	}

	return &Migrator{
		pool:       pool,
		migrations: migrations,
	}, nil
}

// Up 执行所有未应用的 migration。
func (m *Migrator) Up(ctx context.Context) error {
	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("开启迁移事务失败: %w", err)
	}
	defer rollbackQuietly(ctx, tx)

	if err := ensureSchemaMigrationsTable(ctx, tx); err != nil {
		return err
	}

	applied, err := appliedVersions(ctx, tx)
	if err != nil {
		return err
	}

	for _, migration := range m.migrations {
		if _, exists := applied[migration.Version]; exists {
			continue
		}

		if err := execSQL(ctx, tx, migration.UpSQL); err != nil {
			return fmt.Errorf("执行 migration %s up 失败: %w", migration.Version, err)
		}

		if _, err := tx.Exec(
			ctx,
			`INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`,
			migration.Version,
			migration.Name,
		); err != nil {
			return fmt.Errorf("记录 migration %s 失败: %w", migration.Version, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交迁移事务失败: %w", err)
	}

	return nil
}

// Down 回滚最近一次已应用的 migration。
func (m *Migrator) Down(ctx context.Context) error {
	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("开启回滚事务失败: %w", err)
	}
	defer rollbackQuietly(ctx, tx)

	if err := ensureSchemaMigrationsTable(ctx, tx); err != nil {
		return err
	}

	version, err := latestAppliedVersion(ctx, tx)
	if err != nil {
		return err
	}
	if version == "" {
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("提交空回滚事务失败: %w", err)
		}
		return nil
	}

	migration, ok := m.findMigration(version)
	if !ok {
		return fmt.Errorf("未找到版本 %s 对应的 migration 文件", version)
	}

	if err := execSQL(ctx, tx, migration.DownSQL); err != nil {
		return fmt.Errorf("执行 migration %s down 失败: %w", migration.Version, err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM schema_migrations WHERE version = $1`, migration.Version); err != nil {
		return fmt.Errorf("删除 migration %s 记录失败: %w", migration.Version, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交回滚事务失败: %w", err)
	}

	return nil
}

// DownAll 以倒序回滚所有已应用的 migration。
func (m *Migrator) DownAll(ctx context.Context) error {
	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("开启全量回滚事务失败: %w", err)
	}
	defer rollbackQuietly(ctx, tx)

	if err := ensureSchemaMigrationsTable(ctx, tx); err != nil {
		return err
	}

	versions, err := allAppliedVersionsDesc(ctx, tx)
	if err != nil {
		return err
	}

	for _, version := range versions {
		migration, ok := m.findMigration(version)
		if !ok {
			return fmt.Errorf("未找到版本 %s 对应的 migration 文件", version)
		}

		if err := execSQL(ctx, tx, migration.DownSQL); err != nil {
			return fmt.Errorf("执行 migration %s down 失败: %w", migration.Version, err)
		}

		if _, err := tx.Exec(ctx, `DELETE FROM schema_migrations WHERE version = $1`, migration.Version); err != nil {
			return fmt.Errorf("删除 migration %s 记录失败: %w", migration.Version, err)
		}
	}

	if _, err := tx.Exec(ctx, `DROP TABLE IF EXISTS schema_migrations`); err != nil {
		return fmt.Errorf("删除 schema_migrations 失败: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交全量回滚事务失败: %w", err)
	}

	return nil
}

func (m *Migrator) findMigration(version string) (Migration, bool) {
	for _, migration := range m.migrations {
		if migration.Version == version {
			return migration, true
		}
	}
	return Migration{}, false
}

func defaultMigrationsDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "migrations"))
}

func loadMigrations(dir string) ([]Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("读取 migrations 目录失败: %w", err)
	}

	migrations := make([]Migration, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") || strings.HasSuffix(name, "_down.sql") {
			continue
		}

		version := migrationVersion(name)
		if version == "" {
			return nil, fmt.Errorf("migration 文件名缺少版本前缀: %s", name)
		}

		upPath := filepath.Join(dir, name)
		downPath := filepath.Join(dir, strings.TrimSuffix(name, ".sql")+"_down.sql")

		upSQL, err := os.ReadFile(upPath)
		if err != nil {
			return nil, fmt.Errorf("读取 %s 失败: %w", upPath, err)
		}
		downSQL, err := os.ReadFile(downPath)
		if err != nil {
			return nil, fmt.Errorf("读取 %s 失败: %w", downPath, err)
		}

		migrations = append(migrations, Migration{
			Version: version,
			Name:    strings.TrimSuffix(name, ".sql"),
			UpSQL:   string(upSQL),
			DownSQL: string(downSQL),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

func migrationVersion(name string) string {
	parts := strings.SplitN(name, "_", 2)
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func ensureSchemaMigrationsTable(ctx context.Context, tx pgx.Tx) error {
	if err := execSQL(ctx, tx, createSchemaMigrationsSQL); err != nil {
		return fmt.Errorf("初始化 schema_migrations 失败: %w", err)
	}
	return nil
}

func appliedVersions(ctx context.Context, tx pgx.Tx) (map[string]struct{}, error) {
	rows, err := tx.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("查询已应用 migration 失败: %w", err)
	}
	defer rows.Close()

	result := make(map[string]struct{})
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("读取 migration 版本失败: %w", err)
		}
		result[version] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历 migration 版本失败: %w", err)
	}

	return result, nil
}

func latestAppliedVersion(ctx context.Context, tx pgx.Tx) (string, error) {
	var version string
	err := tx.QueryRow(ctx, `SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1`).Scan(&version)
	if err == nil {
		return version, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return "", fmt.Errorf("查询最新 migration 失败: %w", err)
}

func allAppliedVersionsDesc(ctx context.Context, tx pgx.Tx) ([]string, error) {
	rows, err := tx.Query(ctx, `SELECT version FROM schema_migrations ORDER BY version DESC`)
	if err != nil {
		return nil, fmt.Errorf("查询已应用 migration 列表失败: %w", err)
	}
	defer rows.Close()

	versions := make([]string, 0)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("读取 migration 版本失败: %w", err)
		}
		versions = append(versions, version)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历 migration 列表失败: %w", err)
	}

	return versions, nil
}

func execSQL(ctx context.Context, tx pgx.Tx, sql string) error {
	if strings.TrimSpace(sql) == "" {
		return nil
	}

	if _, err := tx.Exec(ctx, sql); err != nil {
		return err
	}
	return nil
}

func rollbackQuietly(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}
