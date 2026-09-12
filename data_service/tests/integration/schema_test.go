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

	"github.com/indu-forge/data_service/internal/db/postgres"
	"github.com/indu-forge/data_service/internal/db/schema"
	"github.com/indu-forge/data_service/internal/repository"
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

func TestCollectorPointGroupDeleteMovesSubtreePointsToParent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	fixture := setupTestDatabase(t, ctx)
	initializer := setupSchemaInitializer(t, fixture.pool)
	if err := initializer.Ensure(ctx); err != nil {
		t.Fatalf("初始化数据库结构失败: %v", err)
	}

	projectID := "550e8400-e29b-41d4-a716-446655440000"
	connectionID := "550e8400-e29b-41d4-a716-446655440010"
	parentID := "550e8400-e29b-41d4-a716-446655440020"
	rootID := "550e8400-e29b-41d4-a716-446655440021"
	childID := "550e8400-e29b-41d4-a716-446655440022"
	pointID := "550e8400-e29b-41d4-a716-446655440030"
	userID := "550e8400-e29b-41d4-a716-446655440001"

	_, err := fixture.pool.Exec(ctx, `
		INSERT INTO data_collector_connections (
			id, project_id, name, code, is_enabled, display_order, protocol_family, driver_id,
			driver_version, schema_version, config, metadata, created_by
		) VALUES ($1,$2,'测试连接','test_connection',true,0,'opcua','opcua.standard','1.0.0',2,'{}','{}',$3);
		INSERT INTO data_collector_point_groups (id,project_id,connection_id,parent_id,name) VALUES
			($4,$2,$1,NULL,'上级分组'),
			($5,$2,$1,$4,'待删除分组'),
			($6,$2,$1,$5,'子分组');
		INSERT INTO data_collector_points (
			id,project_id,connection_id,group_id,code,name,address,address_text,address_schema_version,
			data_type,element_count,read_options,acquisition_mode,acquisition_overrides,enabled,sort_order,metadata
		) VALUES ($7,$2,$1,$6,'temperature','温度','{"nodeId":"ns=2;s=Temperature"}',
			'ns=2;s=Temperature',2,'float32',1,'{}','inherit','{}',true,0,'{}')
	`, connectionID, projectID, userID, parentID, rootID, childID, pointID)
	if err != nil {
		t.Fatalf("准备分组删除测试数据失败: %v", err)
	}

	repo := repository.NewCollectorRepository(fixture.pool)
	if err := repo.DeletePointGroup(ctx, projectID, connectionID, rootID); err != nil {
		t.Fatalf("删除分组失败: %v", err)
	}

	var movedGroupID string
	if err := fixture.pool.QueryRow(ctx, `SELECT group_id FROM data_collector_points WHERE id=$1`, pointID).Scan(&movedGroupID); err != nil {
		t.Fatalf("查询移动后的变量失败: %v", err)
	}
	if movedGroupID != parentID {
		t.Fatalf("变量分组 = %s, want %s", movedGroupID, parentID)
	}
	var remaining int
	if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM data_collector_point_groups WHERE id=ANY($1::uuid[])`, []string{rootID, childID}).Scan(&remaining); err != nil {
		t.Fatalf("查询剩余分组失败: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("待删除分组子树仍有 %d 条记录", remaining)
	}
}

func TestCollectorPointBatchCreateRollsBackConflictsAndExportsSpecifiedPages(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	fixture := setupTestDatabase(t, ctx)
	initializer := setupSchemaInitializer(t, fixture.pool)
	if err := initializer.Ensure(ctx); err != nil {
		t.Fatalf("初始化数据库结构失败: %v", err)
	}
	projectID := "550e8400-e29b-41d4-a716-446655440100"
	connectionID := "550e8400-e29b-41d4-a716-446655440110"
	userID := "550e8400-e29b-41d4-a716-446655440101"
	_, err := fixture.pool.Exec(ctx, `INSERT INTO data_collector_connections (id,project_id,name,code,is_enabled,display_order,protocol_family,driver_id,driver_version,schema_version,config,metadata,created_by) VALUES ($1,$2,'批量测试连接','batch_test',true,0,'opcua','opcua.standard','1.0.0',1,'{}','{}',$3); INSERT INTO data_collector_points (id,project_id,connection_id,code,name,address,address_text,address_schema_version,data_type,element_count,read_options,acquisition_mode,acquisition_overrides,enabled,sort_order,metadata) VALUES ('550e8400-e29b-41d4-a716-446655440120',$2,$1,'existing','Existing','{"nodeId":"ns=2;s=Existing"}','ns=2;s=Existing',1,'float32',1,'{}','inherit','{}',true,0,'{}')`, connectionID, projectID, userID)
	if err != nil {
		t.Fatalf("准备批量创建测试数据失败: %v", err)
	}
	repo := repository.NewCollectorRepository(fixture.pool)
	params := []repository.CreateCollectorPointParams{
		{ID: "550e8400-e29b-41d4-a716-446655440121", ProjectID: projectID, ConnectionID: connectionID, ConnectionCode: "batch_test", UserID: userID, Code: "existing_2", Name: "existing", Address: map[string]any{"nodeId": "ns=2;s=Existing2"}, AddressText: "ns=2;s=Existing2", AddressSchemaVersion: 1, DataType: "float32", ElementCount: 1, ReadOptions: map[string]any{}, Acquisition: map[string]any{"intervalMs": 1000}, Enabled: true, Metadata: map[string]any{}},
		{ID: "550e8400-e29b-41d4-a716-446655440122", ProjectID: projectID, ConnectionID: connectionID, ConnectionCode: "batch_test", UserID: userID, Code: "pressure", Name: "Pressure", Address: map[string]any{"nodeId": "ns=2;s=Pressure"}, AddressText: "ns=2;s=Pressure", AddressSchemaVersion: 1, DataType: "float32", ElementCount: 1, ReadOptions: map[string]any{}, Acquisition: map[string]any{"intervalMs": 1000}, Enabled: true, SortOrder: 1, Metadata: map[string]any{}},
		{ID: "550e8400-e29b-41d4-a716-446655440123", ProjectID: projectID, ConnectionID: connectionID, ConnectionCode: "batch_test", UserID: userID, Code: "temperature", Name: "Temperature", Address: map[string]any{"nodeId": "ns=2;s=Temperature"}, AddressText: "ns=2;s=Temperature", AddressSchemaVersion: 1, DataType: "float32", ElementCount: 1, ReadOptions: map[string]any{}, Acquisition: map[string]any{"intervalMs": 1000}, Enabled: true, SortOrder: 2, Metadata: map[string]any{}},
	}
	created, err := repo.CreatePointsBatch(ctx, params)
	if err == nil {
		t.Fatalf("批量创建包含名称冲突时应整体失败，实际创建 %#v", created)
	}
	var countAfterRollback int
	if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM data_collector_points WHERE connection_id=$1`, connectionID).Scan(&countAfterRollback); err != nil {
		t.Fatalf("读取回滚后的点位数量失败: %v", err)
	}
	if countAfterRollback != 1 {
		t.Fatalf("冲突批次应完整回滚，当前点位数量 = %d, want 1", countAfterRollback)
	}
	created, err = repo.CreatePointsBatch(ctx, params[1:])
	if err != nil {
		t.Fatalf("创建无冲突批次失败: %v", err)
	}
	if len(created) != 2 {
		t.Fatalf("无冲突批次创建数量 = %d, want 2", len(created))
	}
	exported := make([]repository.CollectorPointExportRecord, 0)
	err = repo.StreamPointsForExport(ctx, projectID, connectionID, repository.CollectorPointExportFilter{Scope: "pages", PageSize: 1, Pages: []int64{2}, SortBy: "sortOrder", SortOrder: "asc"}, func(record repository.CollectorPointExportRecord) error {
		exported = append(exported, record)
		return nil
	})
	if err != nil {
		t.Fatalf("导出指定页面失败: %v", err)
	}
	if len(exported) != 1 {
		t.Fatalf("导出数量 = %d, want 1", len(exported))
	}
}

func TestSchemaInitializer_CreatesCoreTables(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	initializer := setupSchemaInitializer(t, fixture.pool)

	start := make(chan struct{})
	errCh := make(chan error, 2)
	var wg sync.WaitGroup

	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errCh <- initializer.Ensure(ctx)
		}()
	}
	close(start)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("骞跺彂执行数据库结构初始化失败: %v", err)
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
		"data_collector_connections",
		"data_collector_point_groups",
		"data_collector_points",
		"data_collector_point_debug_snapshots",
		"data_collector_import_sessions",
		"data_tdengine_configs",
		"data_preview_sessions",
		"data_compute_folders",
		"data_authoring_fences",
		"data_compute_units",
		"data_compute_runs",
		"data_alarm_groups",
		"data_alarm_items",
		"data_alarm_item_inputs",
		"data_alarm_item_conditions",
		"data_realtime_keys",
		"data_workbench_object_groups",
		"collector_dev_agents",
		"collector_dev_registration_codes",
		"collector_dev_tasks",
		"data_table_group_members",
		"data_history_storage_configs",
		"data_history_storage_targets",
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

}

func TestSchemaInitializer_CreatesIndexes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	initializer := setupSchemaInitializer(t, fixture.pool)

	if err := initializer.Ensure(ctx); err != nil {
		t.Fatalf("执行数据库结构初始化失败: %v", err)
	}

	indexes := loadIndexNames(ctx, t, fixture.pool, fixture.schemaName)
	for _, indexName := range []string{
		"data_connections_project_type_idx",
		"data_connections_project_type_idx",
		"data_relational_configs_connection_id_key",
		"data_relational_configs_db_type_idx",
		"data_relational_configs_ssl_config_gin_idx",
		"data_queries_connection_name_key",
		"data_queries_project_connection_idx",
		"data_queries_connection_project_idx",
		"data_queries_type_enabled_idx",
		"data_queries_config_gin_idx",
		"data_points_project_path_key",
		"data_points_project_status_idx",
		"data_points_source_idx",
		"data_points_source_config_gin_idx",
		"data_points_tags_gin_idx",
		"data_history_storage_configs_access_source_key",
		"data_history_storage_configs_collector_key",
		"data_history_storage_configs_datapoint_key",
		"data_history_storage_targets_primary_key",
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
		"data_collector_connections_project_driver_idx",
		"data_collector_point_groups_parent_order_idx",
		"data_collector_points_list_idx",
		"data_collector_points_connection_name_key",
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
		"data_authoring_fences_expiry_idx",
		"data_compute_runs_unit_created_idx",
		"data_compute_runs_project_created_idx",
		"data_alarm_groups_project_sort_idx",
		"data_alarm_groups_project_parent_idx",
		"data_alarm_groups_project_root_name_key",
		"data_alarm_groups_project_parent_name_key",
		"data_alarm_items_project_group_idx",
		"data_alarm_items_project_updated_idx",
		"data_alarm_items_project_enabled_idx",
		"data_alarm_items_datapoint_idx",
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

func TestSchemaInitializer_CreatesBuiltinRuntimeStores(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	initializer := setupSchemaInitializer(t, fixture.pool)
	if err := initializer.Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
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
			INSERT INTO data_connections (project_id, name, type, category, is_enabled, metadata, created_by, updated_by)
			VALUES ($1, $2, $3, 'builtin', true, jsonb_build_object('runtimeKey', $3 || ':' || $1::text), $4, $4)
		`, projectID, storeType, storeType, userID)
		if err != nil {
			t.Fatalf("insert builtin type %s failed: %v", storeType, err)
		}
	}

	_, err := fixture.pool.Exec(ctx, `
		INSERT INTO data_connections (project_id, name, type, category, is_enabled, metadata, created_by, updated_by)
		VALUES ($1, 'repeat relation', 'builtin.relation', 'builtin', true, jsonb_build_object('runtimeKey', 'builtin.relation:repeat'), $2, $2)
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
			t.Fatalf("expected builtin connection %s", storeType)
		}
	}
}

func TestSchemaInitializer_RemovesLegacyAlarmRuleModel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	initializer := setupSchemaInitializer(t, fixture.pool)
	if err := initializer.Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}

	var exists bool
	if err := fixture.pool.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, fixture.schemaName+".data_alarm_rules").Scan(&exists); err != nil {
		t.Fatalf("check legacy alarm table failed: %v", err)
	}
	if exists {
		t.Fatal("data_alarm_rules should not exist in final baseline")
	}
}

func TestSchemaInitializer_CreatesAlarmItemTables(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	initializer := setupSchemaInitializer(t, fixture.pool)
	if err := initializer.Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}

	assertColumnExists(ctx, t, fixture.pool, fixture.schemaName, "data_alarm_groups", "parent_id", "uuid")
	assertColumnExists(ctx, t, fixture.pool, fixture.schemaName, "data_alarm_items", "datapoint_id", "uuid")
	assertColumnExists(ctx, t, fixture.pool, fixture.schemaName, "data_alarm_items", "alarm_type", "character varying")
	assertColumnExists(ctx, t, fixture.pool, fixture.schemaName, "data_alarm_items", "trigger_fingerprint", "character varying")
	assertColumnExists(ctx, t, fixture.pool, fixture.schemaName, "data_alarm_items", "revision", "bigint")
	assertColumnExists(ctx, t, fixture.pool, fixture.schemaName, "data_alarm_item_inputs", "input_key", "character varying")
	assertColumnExists(ctx, t, fixture.pool, fixture.schemaName, "data_alarm_item_conditions", "kind", "character varying")
	assertColumnExists(ctx, t, fixture.pool, fixture.schemaName, "data_alarm_notification_channels", "secret_status", "jsonb")
	assertColumnExists(ctx, t, fixture.pool, fixture.schemaName, "data_alarm_config_sync_requests", "idempotency_key", "character varying")

	if _, err := fixture.pool.Exec(ctx, `
		INSERT INTO data_alarm_items (
            project_id,
			display_name,
			name_key,
            mode,
			alarm_type,
			evaluation_mode,
			derived_expression,
			trigger_fingerprint
        )
		VALUES (gen_random_uuid(), '非法模式', '非法模式', 'legacy', 'threshold', 'single', '', repeat('a',64))
    `); err == nil {
		t.Fatalf("expected invalid configuration mode to violate check constraint")
	}
}

func TestAlarmConfigSyncIsIdempotentOrderedAndTransactional(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	initializer := setupSchemaInitializer(t, fixture.pool)
	if err := initializer.Ensure(ctx); err != nil {
		t.Fatalf("初始化数据库结构失败: %v", err)
	}

	projectID := "550e8400-e29b-41d4-a716-446655440100"
	actorID := "550e8400-e29b-41d4-a716-446655440101"
	datapointID := "550e8400-e29b-41d4-a716-446655440102"
	groupID := "550e8400-e29b-41d4-a716-446655440103"
	alarmItemID := "550e8400-e29b-41d4-a716-446655440104"
	conditionID := "550e8400-e29b-41d4-a716-446655440105"
	rollbackGroupID := "550e8400-e29b-41d4-a716-446655440106"
	invalidConfigurationID := "550e8400-e29b-41d4-a716-446655440107"
	missingDatapointID := "550e8400-e29b-41d4-a716-446655440108"

	if _, err := fixture.pool.Exec(ctx, `
		INSERT INTO data_points(id,project_id,path,name,source_type,data_type,created_by)
		VALUES($1,$2,'line.temperature','温度','manual','float64',$3)
	`, datapointID, projectID, actorID); err != nil {
		t.Fatalf("准备报警同步数据点失败: %v", err)
	}

	repo := repository.NewAlarmRepository(fixture.pool)
	first, err := repo.ApplyConfigSync(ctx, projectID, actorID, "epoch-1", 1, "request-1", []repository.AlarmConfigSyncOperationParams{{
		Resource: "group",
		Action:   "upsert",
		ID:       groupID,
		Group: &repository.SaveAlarmGroupParams{
			ID: groupID, ProjectID: projectID, UserID: actorID, Name: "生产线",
		},
	}})
	if err != nil {
		t.Fatalf("首次同步失败: %v", err)
	}
	repeated, err := repo.ApplyConfigSync(ctx, projectID, actorID, "epoch-1", 1, "request-1", nil)
	if err != nil || !repeated.Idempotent || repeated.ConfigRevision != first.ConfigRevision {
		t.Fatalf("重复同步未命中幂等记录: result=%#v err=%v", repeated, err)
	}
	if _, err = repo.ApplyConfigSync(ctx, projectID, actorID, "epoch-1", 1, "request-stale", []repository.AlarmConfigSyncOperationParams{{Resource: "group", Action: "delete", ID: groupID}}); err == nil {
		t.Fatal("旧同步序号应被拒绝")
	}

	alarmItem := repository.SaveAlarmItemParams{
		ID: alarmItemID, ProjectID: projectID, UserID: actorID, DatapointID: &datapointID, GroupID: &groupID, IsCreate: true,
		DisplayName: "温度高报警", NameKey: "温度高报警", Mode: "point", AlarmType: "threshold", EvaluationMode: "single", TriggerFingerprint: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", NotificationMode: "inherit", IsEnabled: true,
		NotificationChannelIDs: []string{}, Contract: map[string]any{"schema": "alarm.item.v1", "alarmItemId": alarmItemID},
		Conditions: []repository.AlarmItemConditionRecord{{ID: conditionID, Kind: "threshold", Operator: "gt", Label: "高温", Severity: "warning", Params: map[string]any{"threshold": 80.0}}},
	}
	if _, err = repo.ApplyConfigSync(ctx, projectID, actorID, "epoch-1", 2, "request-2", []repository.AlarmConfigSyncOperationParams{{Resource: "alarm_item", Action: "upsert", ID: alarmItemID, AlarmItem: &alarmItem}}); err != nil {
		t.Fatalf("同步报警配置失败: %v", err)
	}

	invalidAlarmItem := alarmItem
	invalidAlarmItem.ID = invalidConfigurationID
	invalidAlarmItem.DisplayName = "无效报警"
	invalidAlarmItem.NameKey = "无效报警"
	invalidAlarmItem.DatapointID = &missingDatapointID
	invalidAlarmItem.TriggerFingerprint = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if _, err = repo.ApplyConfigSync(ctx, projectID, actorID, "epoch-1", 3, "request-3", []repository.AlarmConfigSyncOperationParams{
		{Resource: "group", Action: "upsert", ID: rollbackGroupID, Group: &repository.SaveAlarmGroupParams{ID: rollbackGroupID, ProjectID: projectID, UserID: actorID, Name: "应回滚目录"}},
		{Resource: "alarm_item", Action: "upsert", ID: invalidConfigurationID, AlarmItem: &invalidAlarmItem},
	}); err == nil {
		t.Fatal("包含无效点位的同步批次应失败")
	}

	var rollbackGroupCount int
	if err = fixture.pool.QueryRow(ctx, `SELECT count(*) FROM data_alarm_groups WHERE id=$1`, rollbackGroupID).Scan(&rollbackGroupCount); err != nil {
		t.Fatalf("检查回滚目录失败: %v", err)
	}
	state, err := repo.GetSyncState(ctx, projectID)
	if err != nil {
		t.Fatalf("查询同步状态失败: %v", err)
	}
	if rollbackGroupCount != 0 || state.LastSequence != 2 {
		t.Fatalf("失败批次未完整回滚: groupCount=%d state=%#v", rollbackGroupCount, state)
	}
}

func setupSchemaInitializer(t *testing.T, pool *pgxpool.Pool) *schema.Initializer {
	t.Helper()

	initializer, err := schema.NewInitializer(pool)
	if err != nil {
		t.Fatalf("鍒涘缓杩佺Щ鍣ㄥけ璐? %v", err)
	}

	return initializer
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
              'data_collector_connections',
              'data_collector_point_groups',
              'data_collector_points',
              'data_tdengine_configs',
              'data_preview_sessions',
			  'data_compute_units',
			  'data_compute_folders',
			  'data_compute_runs',
			  'data_authoring_fences',
			  'data_history_storage_configs',
			  'data_history_storage_targets',
			  'data_workbench_object_groups',
			  'data_table_group_members',
			  'data_alarm_groups',
			  'data_alarm_items',
			  'data_alarm_item_inputs',
			  'data_alarm_item_conditions',
			  'data_alarm_notification_channels',
			  'data_alarm_config_sync_requests'
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
