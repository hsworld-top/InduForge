package db

import (
	"context"
	"fmt"
	"strings"

	coreschema "github.com/indu-forge/dev_core/db/schema"
	"github.com/jackc/pgx/v5/pgxpool"
)

var requiredTables = []string{
	"tenants",
	"users",
	"projects",
	"scene_provider_state",
	"scene_documents",
	"scene_file_nodes",
	"scene_content_objects",
	"scene_assets",
	"scene_asset_generations",
	"scene_asset_generation_files",
	"scene_asset_draft_files",
	"scene_asset_bindings",
	"scene_revisions",
	"scene_revision_files",
	"scene_revision_assets",
	"project_runtime_users",
	"project_roles",
	"audit_logs",
	"application_versions",
	"nodes",
	"node_deployments",
	"node_commands",
	"node_enrollments",
	"host_nodes",
	"project_deployments",
	"deployment_runs",
	"deployment_run_events",
	"deployment_services",
}

func EnsureSchema(ctx context.Context, pool *pgxpool.Pool, allowCreate bool) error {
	existing, err := listBusinessTables(ctx, pool)
	if err != nil {
		return err
	}
	if len(existing) == 0 {
		if !allowCreate {
			return fmt.Errorf("控制面数据库为空且 DB_AUTO_SCHEMA_SYNC 未启用")
		}
		if _, err := pool.Exec(ctx, coreschema.CoreSQL); err != nil {
			return fmt.Errorf("初始化控制面数据库失败: %w", err)
		}
		return nil
	}

	for _, table := range requiredTables {
		if _, ok := existing[table]; !ok {
			return fmt.Errorf("控制面数据库已有业务表但缺少必需表 %s，服务拒绝自动修改", table)
		}
	}
	return nil
}

func listBusinessTables(ctx context.Context, pool *pgxpool.Pool) (map[string]struct{}, error) {
	rows, err := pool.Query(ctx, `
		SELECT tablename
		FROM pg_catalog.pg_tables
		WHERE schemaname = current_schema()
		  AND tablename NOT LIKE 'pg_%'
		  AND tablename NOT LIKE 'sql_%'
	`)
	if err != nil {
		return nil, fmt.Errorf("读取控制面数据库表失败: %w", err)
	}
	defer rows.Close()

	tables := make(map[string]struct{})
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, fmt.Errorf("读取控制面数据库表名失败: %w", err)
		}
		tables[strings.TrimSpace(table)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历控制面数据库表失败: %w", err)
	}
	return tables, nil
}
