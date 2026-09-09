package db

import (
	"context"
	"fmt"
	"strings"

	coreschema "github.com/indu-forge/dev_core/db/schema"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var requiredTables = []string{
	"tenants",
	"users",
	"refresh_tokens",
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
	"runtime_clusters",
	"runtime_cluster_nodes",
	"runtime_cluster_events",
	"runtime_environments",
	"runtime_environment_nodes",
	"runtime_environment_services",
	"runtime_environment_events",
	"project_deployments",
	"deployment_runs",
	"deployment_run_events",
	"deployment_services",
	"authoring_restore_tasks",
	"authoring_restore_task_events",
	"authoring_project_fences",
}

var requiredColumns = map[string][]string{
	"tenants":              {"initialized", "is_default", "admin_user_id"},
	"users":                {"must_change_password", "credential_version"},
	"refresh_tokens":       {"remember_me", "credential_version"},
	"host_nodes":           {"node_source"},
	"projects":             {"authoring_epoch"},
	"application_versions": {"authoring_snapshot_schema", "authoring_snapshot_bucket", "authoring_snapshot_key", "authoring_snapshot_hash", "authoring_snapshot_cipher_hash", "authoring_snapshot_size", "authoring_snapshot_key_id", "authoring_project_revision", "restorable"},
	"project_deployments":  {"deletion_requested_at"},
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
	return validateRequiredColumns(ctx, pool)
}

type schemaColumnReader interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// 只读取元数据；旧库缺少当前认证字段时明确拒绝，绝不启动修复。
func validateRequiredColumns(ctx context.Context, pool schemaColumnReader) error {
	for table, columns := range requiredColumns {
		for _, column := range columns {
			var exists bool
			if err := pool.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1 FROM information_schema.columns
					WHERE table_schema = current_schema() AND table_name = $1 AND column_name = $2
				)
			`, table, column).Scan(&exists); err != nil {
				return fmt.Errorf("校验控制面数据库字段 %s.%s 失败: %w", table, column, err)
			}
			if !exists {
				return fmt.Errorf("控制面数据库结构不兼容，缺少必需字段 %s.%s；请按当前最终基线重建数据库", table, column)
			}
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
