package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/jackc/pgx/v5"
)

// completeAuthoringSnapshot 只包含用户可编辑的工程开发配置。运行记录、预览会话、诊断结果、
// 最近值、质量状态和异步任务均不属于可恢复的开发态，因此刻意不在此投影中。
type completeAuthoringSnapshot struct {
	relationalConfigs      []SnapshotRelationalConfigRecord
	kafkaConfigs           []SnapshotKafkaConfigRecord
	kafkaTopicGroups       []SnapshotKafkaTopicGroupRecord
	kafkaTopicMappings     []SnapshotKafkaTopicMappingRecord
	kafkaFieldGroups       []SnapshotKafkaFieldGroupRecord
	kafkaFields            []SnapshotKafkaFieldRecord
	mqttSubscriptionGroups []SnapshotMqttSubscriptionGroupRecord
	computeFolders         []SnapshotComputeFolderRecord
	computeDependencies    []SnapshotComputeDependencyRecord
	workbenchObjectGroups  []SnapshotWorkbenchObjectGroupRecord
	tableGroupMembers      []SnapshotTableGroupMemberRecord
}

func (a completeAuthoringSnapshot) apply(snapshot *ProjectSnapshot) {
	snapshot.RelationalConfigs = a.relationalConfigs
	snapshot.KafkaConfigs = a.kafkaConfigs
	snapshot.KafkaTopicGroups = a.kafkaTopicGroups
	snapshot.KafkaTopicMappings = a.kafkaTopicMappings
	snapshot.KafkaFieldGroups = a.kafkaFieldGroups
	snapshot.KafkaFields = a.kafkaFields
	snapshot.MqttSubscriptionGroups = a.mqttSubscriptionGroups
	snapshot.ComputeFolders = a.computeFolders
	snapshot.ComputeDependencies = a.computeDependencies
	snapshot.WorkbenchObjectGroups = a.workbenchObjectGroups
	snapshot.TableGroupMembers = a.tableGroupMembers
}

func (r *ProjectSnapshotRepository) listCompleteAuthoringSnapshot(ctx context.Context, q projectSnapshotQuerier, projectID string) (completeAuthoringSnapshot, error) {
	var result completeAuthoringSnapshot
	var err error
	if result.relationalConfigs, err = listSnapshotRelationalConfigs(ctx, q, projectID); err != nil {
		return result, err
	}
	if result.kafkaConfigs, err = listSnapshotKafkaConfigs(ctx, q, projectID); err != nil {
		return result, err
	}
	if result.kafkaTopicGroups, err = listSnapshotKafkaTopicGroups(ctx, q, projectID); err != nil {
		return result, err
	}
	if result.kafkaTopicMappings, err = listSnapshotKafkaTopicMappings(ctx, q, projectID); err != nil {
		return result, err
	}
	if result.kafkaFieldGroups, err = listSnapshotKafkaFieldGroups(ctx, q, projectID); err != nil {
		return result, err
	}
	if result.kafkaFields, err = listSnapshotKafkaFields(ctx, q, projectID); err != nil {
		return result, err
	}
	if result.mqttSubscriptionGroups, err = listSnapshotMqttSubscriptionGroups(ctx, q, projectID); err != nil {
		return result, err
	}
	if result.computeFolders, err = listSnapshotComputeFolders(ctx, q, projectID); err != nil {
		return result, err
	}
	if result.computeDependencies, err = listSnapshotComputeDependencies(ctx, q, projectID); err != nil {
		return result, err
	}
	if result.workbenchObjectGroups, err = listSnapshotWorkbenchObjectGroups(ctx, q, projectID); err != nil {
		return result, err
	}
	if result.tableGroupMembers, err = listSnapshotTableGroupMembers(ctx, q, projectID); err != nil {
		return result, err
	}
	return result, nil
}

func listSnapshotRelationalConfigs(ctx context.Context, q projectSnapshotQuerier, projectID string) ([]SnapshotRelationalConfigRecord, error) {
	rows, err := q.Query(ctx, `SELECT cfg.id::text,cfg.connection_id::text,cfg.db_type,cfg.host,cfg.port,cfg.database,cfg.username,cfg.schema,cfg.charset,cfg.timezone,cfg.ssl,cfg.ssl_config,cfg.pool_min,cfg.pool_max,cfg.acquire_timeout_ms,cfg.idle_timeout_ms,cfg.query_timeout_ms,cfg.options
		FROM data_relational_configs cfg JOIN data_connections c ON c.id=cfg.connection_id WHERE c.project_id=$1 ORDER BY cfg.connection_id`, projectID)
	if err != nil {
		return nil, wrapSnapshotRead("读取快照关系库配置失败", err)
	}
	defer rows.Close()
	result := make([]SnapshotRelationalConfigRecord, 0)
	for rows.Next() {
		var item SnapshotRelationalConfigRecord
		var ignoredID string
		var ssl, options []byte
		if err := rows.Scan(&ignoredID, &item.ConnectionID, &item.DBType, &item.Host, &item.Port, &item.Database, &item.Username, &item.Schema, &item.Charset, &item.Timezone, &item.SSL, &ssl, &item.PoolMin, &item.PoolMax, &item.AcquireTimeoutMS, &item.IdleTimeoutMS, &item.QueryTimeoutMS, &options); err != nil {
			return nil, err
		}
		item.SSLConfig = mustJSONObject(ssl)
		item.Options = mustJSONObject(options)
		result = append(result, item)
	}
	return result, rows.Err()
}

func listSnapshotKafkaConfigs(ctx context.Context, q projectSnapshotQuerier, projectID string) ([]SnapshotKafkaConfigRecord, error) {
	rows, err := q.Query(ctx, `SELECT cfg.connection_id::text,cfg.brokers,cfg.topic,cfg.consumer_group,cfg.start_position,cfg.options FROM data_kafka_configs cfg JOIN data_connections c ON c.id=cfg.connection_id WHERE c.project_id=$1 ORDER BY cfg.connection_id`, projectID)
	if err != nil {
		return nil, wrapSnapshotRead("读取快照 Kafka 配置失败", err)
	}
	defer rows.Close()
	result := make([]SnapshotKafkaConfigRecord, 0)
	for rows.Next() {
		var i SnapshotKafkaConfigRecord
		var raw []byte
		if err := rows.Scan(&i.ConnectionID, &i.Brokers, &i.Topic, &i.ConsumerGroup, &i.StartPosition, &raw); err != nil {
			return nil, err
		}
		i.Options = mustJSONObject(raw)
		result = append(result, i)
	}
	return result, rows.Err()
}

func listSnapshotKafkaTopicGroups(ctx context.Context, q projectSnapshotQuerier, projectID string) ([]SnapshotKafkaTopicGroupRecord, error) {
	rows, err := q.Query(ctx, `SELECT id::text,connection_id::text,parent_id::text,name,sort_order FROM data_kafka_topic_groups WHERE project_id=$1 ORDER BY connection_id,sort_order,id`, projectID)
	if err != nil {
		return nil, wrapSnapshotRead("读取快照 Kafka Topic 分组失败", err)
	}
	defer rows.Close()
	out := make([]SnapshotKafkaTopicGroupRecord, 0)
	for rows.Next() {
		var i SnapshotKafkaTopicGroupRecord
		if err := rows.Scan(&i.ID, &i.ConnectionID, &i.ParentID, &i.Name, &i.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}
func listSnapshotKafkaTopicMappings(ctx context.Context, q projectSnapshotQuerier, projectID string) ([]SnapshotKafkaTopicMappingRecord, error) {
	rows, err := q.Query(ctx, `SELECT id::text,connection_id::text,group_id::text,name,topic,description,consumer_group,output_mode,raw_output_scope,partition_mode,partition,start_position,start_offset,decode,sample_limit,timeout_ms,sort_order FROM data_kafka_topic_mappings WHERE project_id=$1 ORDER BY connection_id,sort_order,id`, projectID)
	if err != nil {
		return nil, wrapSnapshotRead("读取快照 Kafka Topic 映射失败", err)
	}
	defer rows.Close()
	out := make([]SnapshotKafkaTopicMappingRecord, 0)
	for rows.Next() {
		var i SnapshotKafkaTopicMappingRecord
		if err := rows.Scan(&i.ID, &i.ConnectionID, &i.GroupID, &i.Name, &i.Topic, &i.Description, &i.ConsumerGroup, &i.OutputMode, &i.RawOutputScope, &i.PartitionMode, &i.Partition, &i.StartPosition, &i.StartOffset, &i.Decode, &i.SampleLimit, &i.TimeoutMS, &i.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}
func listSnapshotKafkaFieldGroups(ctx context.Context, q projectSnapshotQuerier, projectID string) ([]SnapshotKafkaFieldGroupRecord, error) {
	rows, err := q.Query(ctx, `SELECT id::text,connection_id::text,topic_mapping_id::text,parent_id::text,name,description,sort_order FROM data_kafka_field_groups WHERE project_id=$1 ORDER BY topic_mapping_id,sort_order,id`, projectID)
	if err != nil {
		return nil, wrapSnapshotRead("读取快照 Kafka 字段分组失败", err)
	}
	defer rows.Close()
	out := make([]SnapshotKafkaFieldGroupRecord, 0)
	for rows.Next() {
		var i SnapshotKafkaFieldGroupRecord
		if err := rows.Scan(&i.ID, &i.ConnectionID, &i.TopicMappingID, &i.ParentID, &i.Name, &i.Description, &i.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}
func listSnapshotKafkaFields(ctx context.Context, q projectSnapshotQuerier, projectID string) ([]SnapshotKafkaFieldRecord, error) {
	rows, err := q.Query(ctx, `SELECT id::text,connection_id::text,topic_mapping_id::text,group_id::text,name,value_path,key_path,data_type,enabled,description,sort_order FROM data_kafka_fields WHERE project_id=$1 ORDER BY topic_mapping_id,sort_order,id`, projectID)
	if err != nil {
		return nil, wrapSnapshotRead("读取快照 Kafka 字段失败", err)
	}
	defer rows.Close()
	out := make([]SnapshotKafkaFieldRecord, 0)
	for rows.Next() {
		var i SnapshotKafkaFieldRecord
		var vp, kp []byte
		if err := rows.Scan(&i.ID, &i.ConnectionID, &i.TopicMappingID, &i.GroupID, &i.Name, &vp, &kp, &i.DataType, &i.Enabled, &i.Description, &i.SortOrder); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(vp, &i.ValuePath); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(kp, &i.KeyPath); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}
func listSnapshotMqttSubscriptionGroups(ctx context.Context, q projectSnapshotQuerier, projectID string) ([]SnapshotMqttSubscriptionGroupRecord, error) {
	rows, err := q.Query(ctx, `SELECT id::text,connection_id::text,name,parent_id::text,created_at,updated_at FROM data_mqtt_subscription_groups WHERE project_id=$1 ORDER BY created_at,id`, projectID)
	if err != nil {
		return nil, wrapSnapshotRead("读取快照 MQTT 订阅分组失败", err)
	}
	defer rows.Close()
	out := make([]SnapshotMqttSubscriptionGroupRecord, 0)
	for rows.Next() {
		var i SnapshotMqttSubscriptionGroupRecord
		if err := rows.Scan(&i.ID, &i.ConnectionID, &i.Name, &i.ParentID, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}
func listSnapshotComputeFolders(ctx context.Context, q projectSnapshotQuerier, projectID string) ([]SnapshotComputeFolderRecord, error) {
	rows, err := q.Query(ctx, `SELECT id::text,name,parent_id::text,created_at,updated_at FROM data_compute_folders WHERE project_id=$1 ORDER BY created_at,id`, projectID)
	if err != nil {
		return nil, wrapSnapshotRead("读取快照计算目录失败", err)
	}
	defer rows.Close()
	out := make([]SnapshotComputeFolderRecord, 0)
	for rows.Next() {
		var i SnapshotComputeFolderRecord
		if err := rows.Scan(&i.ID, &i.Name, &i.ParentID, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}
func listSnapshotComputeDependencies(ctx context.Context, q projectSnapshotQuerier, projectID string) ([]SnapshotComputeDependencyRecord, error) {
	rows, err := q.Query(ctx, `SELECT id::text,language,package_name,import_name,version,created_at,updated_at FROM data_compute_dependencies WHERE project_id=$1 ORDER BY language,package_name`, projectID)
	if err != nil {
		return nil, wrapSnapshotRead("读取快照计算依赖失败", err)
	}
	defer rows.Close()
	out := make([]SnapshotComputeDependencyRecord, 0)
	for rows.Next() {
		var i SnapshotComputeDependencyRecord
		if err := rows.Scan(&i.ID, &i.Language, &i.PackageName, &i.ImportName, &i.Version, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}
func listSnapshotWorkbenchObjectGroups(ctx context.Context, q projectSnapshotQuerier, projectID string) ([]SnapshotWorkbenchObjectGroupRecord, error) {
	rows, err := q.Query(ctx, `SELECT id::text,connection_id::text,scope,name,sort_order FROM data_workbench_object_groups WHERE project_id=$1 ORDER BY connection_id,scope,sort_order,id`, projectID)
	if err != nil {
		return nil, wrapSnapshotRead("读取快照工作台分组失败", err)
	}
	defer rows.Close()
	out := make([]SnapshotWorkbenchObjectGroupRecord, 0)
	for rows.Next() {
		var i SnapshotWorkbenchObjectGroupRecord
		if err := rows.Scan(&i.ID, &i.ConnectionID, &i.Scope, &i.Name, &i.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}
func listSnapshotTableGroupMembers(ctx context.Context, q projectSnapshotQuerier, projectID string) ([]SnapshotTableGroupMemberRecord, error) {
	rows, err := q.Query(ctx, `SELECT connection_id::text,table_name,group_id::text,updated_at FROM data_table_group_members WHERE project_id=$1 ORDER BY connection_id,table_name`, projectID)
	if err != nil {
		return nil, wrapSnapshotRead("读取快照表分组成员失败", err)
	}
	defer rows.Close()
	out := make([]SnapshotTableGroupMemberRecord, 0)
	for rows.Next() {
		var i SnapshotTableGroupMemberRecord
		if err := rows.Scan(&i.ConnectionID, &i.TableName, &i.GroupID, &i.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func wrapSnapshotRead(message string, err error) error {
	return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, message, err)
}

func (r *ProjectSnapshotRepository) insertExplicitRelationalAndKafkaConfigs(ctx context.Context, tx pgx.Tx, s ProjectSnapshot) error {
	for _, i := range s.RelationalConfigs {
		ssl, _ := json.Marshal(i.SSLConfig)
		options, _ := json.Marshal(i.Options)
		if _, err := tx.Exec(ctx, `DELETE FROM data_relational_configs WHERE connection_id=$1`, i.ConnectionID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO data_relational_configs(id,connection_id,db_type,host,port,database,username,schema,charset,timezone,ssl,ssl_config,pool_min,pool_max,acquire_timeout_ms,idle_timeout_ms,query_timeout_ms,options) VALUES(gen_random_uuid(),$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb,$12,$13,$14,$15,$16,$17::jsonb)`, i.ConnectionID, i.DBType, i.Host, i.Port, i.Database, i.Username, i.Schema, i.Charset, i.Timezone, i.SSL, nullableJSONString(ssl), i.PoolMin, i.PoolMax, i.AcquireTimeoutMS, i.IdleTimeoutMS, i.QueryTimeoutMS, string(options)); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "恢复快照关系库配置失败", err)
		}
	}
	for _, i := range s.KafkaConfigs {
		options, _ := json.Marshal(i.Options)
		if _, err := tx.Exec(ctx, `DELETE FROM data_kafka_configs WHERE connection_id=$1`, i.ConnectionID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO data_kafka_configs(connection_id,brokers,topic,consumer_group,start_position,options) VALUES($1,$2,$3,$4,$5,$6::jsonb)`, i.ConnectionID, i.Brokers, i.Topic, i.ConsumerGroup, i.StartPosition, string(options)); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "恢复快照 Kafka 配置失败", err)
		}
	}
	return nil
}

func (r *ProjectSnapshotRepository) insertCompleteAuthoringGroups(ctx context.Context, tx pgx.Tx, projectID, actorID string, s ProjectSnapshot) error {
	if err := insertComputeFolders(ctx, tx, projectID, actorID, s.ComputeFolders); err != nil {
		return err
	}
	for _, i := range s.ComputeDependencies {
		if _, err := tx.Exec(ctx, `INSERT INTO data_compute_dependencies(id,project_id,language,package_name,import_name,version,created_by,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, i.ID, projectID, i.Language, i.PackageName, i.ImportName, i.Version, actorID, coalesceTime(i.CreatedAt), coalesceTime(i.UpdatedAt)); err != nil {
			return err
		}
	}
	for _, i := range s.WorkbenchObjectGroups {
		if _, err := tx.Exec(ctx, `INSERT INTO data_workbench_object_groups(id,project_id,connection_id,scope,name,sort_order,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$7)`, i.ID, projectID, i.ConnectionID, i.Scope, i.Name, i.SortOrder, actorID); err != nil {
			return err
		}
	}
	for _, i := range s.TableGroupMembers {
		if _, err := tx.Exec(ctx, `INSERT INTO data_table_group_members(project_id,connection_id,table_name,group_id,updated_by,updated_at) VALUES($1,$2,$3,$4,$5,$6)`, projectID, i.ConnectionID, i.TableName, i.GroupID, actorID, coalesceTime(i.UpdatedAt)); err != nil {
			return err
		}
	}
	if err := insertMqttGroups(ctx, tx, projectID, actorID, s.MqttSubscriptionGroups); err != nil {
		return err
	}
	if err := insertKafkaTopicGroups(ctx, tx, projectID, actorID, s.KafkaTopicGroups); err != nil {
		return err
	}
	for _, i := range s.KafkaTopicMappings {
		if _, err := tx.Exec(ctx, `INSERT INTO data_kafka_topic_mappings(id,project_id,connection_id,group_id,name,topic,description,consumer_group,output_mode,raw_output_scope,partition_mode,partition,start_position,start_offset,decode,sample_limit,timeout_ms,sort_order,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$19)`, i.ID, projectID, i.ConnectionID, i.GroupID, i.Name, i.Topic, i.Description, i.ConsumerGroup, i.OutputMode, i.RawOutputScope, i.PartitionMode, i.Partition, i.StartPosition, i.StartOffset, i.Decode, i.SampleLimit, i.TimeoutMS, i.SortOrder, actorID); err != nil {
			return err
		}
	}
	if err := insertKafkaFieldGroups(ctx, tx, projectID, actorID, s.KafkaFieldGroups); err != nil {
		return err
	}
	for _, i := range s.KafkaFields {
		vp, _ := json.Marshal(i.ValuePath)
		kp, _ := json.Marshal(i.KeyPath)
		if _, err := tx.Exec(ctx, `INSERT INTO data_kafka_fields(id,project_id,connection_id,topic_mapping_id,group_id,name,value_path,key_path,data_type,enabled,description,quality,sort_order,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7::jsonb,$8::jsonb,$9,$10,$11,'unknown',$12,$13,$13)`, i.ID, projectID, i.ConnectionID, i.TopicMappingID, i.GroupID, i.Name, string(vp), string(kp), i.DataType, i.Enabled, i.Description, i.SortOrder, actorID); err != nil {
			return err
		}
	}
	return nil
}

func insertComputeFolders(ctx context.Context, tx pgx.Tx, projectID, actorID string, items []SnapshotComputeFolderRecord) error {
	pending := append([]SnapshotComputeFolderRecord(nil), items...)
	done := map[string]bool{}
	for len(pending) > 0 {
		next := pending[:0]
		progress := false
		for _, i := range pending {
			if i.ParentID != nil && !done[*i.ParentID] {
				next = append(next, i)
				continue
			}
			if _, err := tx.Exec(ctx, `INSERT INTO data_compute_folders(id,project_id,name,parent_id,created_by,updated_by,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$5,$6,$7)`, i.ID, projectID, i.Name, i.ParentID, actorID, coalesceTime(i.CreatedAt), coalesceTime(i.UpdatedAt)); err != nil {
				return err
			}
			done[i.ID] = true
			progress = true
		}
		if !progress {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照计算目录存在循环或缺失父目录")
		}
		pending = append([]SnapshotComputeFolderRecord(nil), next...)
	}
	return nil
}
func insertMqttGroups(ctx context.Context, tx pgx.Tx, projectID, actorID string, items []SnapshotMqttSubscriptionGroupRecord) error {
	pending := append([]SnapshotMqttSubscriptionGroupRecord(nil), items...)
	done := map[string]bool{}
	for len(pending) > 0 {
		next := pending[:0]
		progress := false
		for _, i := range pending {
			if i.ParentID != nil && !done[*i.ParentID] {
				next = append(next, i)
				continue
			}
			if _, err := tx.Exec(ctx, `INSERT INTO data_mqtt_subscription_groups(id,project_id,connection_id,name,parent_id,created_by,updated_by,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$6,$7,$8)`, i.ID, projectID, i.ConnectionID, i.Name, i.ParentID, actorID, coalesceTime(i.CreatedAt), coalesceTime(i.UpdatedAt)); err != nil {
				return err
			}
			done[i.ID] = true
			progress = true
		}
		if !progress {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照 MQTT 分组存在循环或缺失父目录")
		}
		pending = append([]SnapshotMqttSubscriptionGroupRecord(nil), next...)
	}
	return nil
}
func insertKafkaTopicGroups(ctx context.Context, tx pgx.Tx, projectID, actorID string, items []SnapshotKafkaTopicGroupRecord) error {
	pending := append([]SnapshotKafkaTopicGroupRecord(nil), items...)
	done := map[string]bool{}
	for len(pending) > 0 {
		next := pending[:0]
		progress := false
		for _, i := range pending {
			if i.ParentID != nil && !done[*i.ParentID] {
				next = append(next, i)
				continue
			}
			if _, err := tx.Exec(ctx, `INSERT INTO data_kafka_topic_groups(id,project_id,connection_id,parent_id,name,sort_order,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$7)`, i.ID, projectID, i.ConnectionID, i.ParentID, i.Name, i.SortOrder, actorID); err != nil {
				return err
			}
			done[i.ID] = true
			progress = true
		}
		if !progress {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照 Kafka Topic 分组存在循环或缺失父目录")
		}
		pending = append([]SnapshotKafkaTopicGroupRecord(nil), next...)
	}
	return nil
}
func insertKafkaFieldGroups(ctx context.Context, tx pgx.Tx, projectID, actorID string, items []SnapshotKafkaFieldGroupRecord) error {
	pending := append([]SnapshotKafkaFieldGroupRecord(nil), items...)
	done := map[string]bool{}
	for len(pending) > 0 {
		next := pending[:0]
		progress := false
		for _, i := range pending {
			if i.ParentID != nil && !done[*i.ParentID] {
				next = append(next, i)
				continue
			}
			if _, err := tx.Exec(ctx, `INSERT INTO data_kafka_field_groups(id,project_id,connection_id,topic_mapping_id,parent_id,name,description,sort_order,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$9)`, i.ID, projectID, i.ConnectionID, i.TopicMappingID, i.ParentID, i.Name, i.Description, i.SortOrder, actorID); err != nil {
				return err
			}
			done[i.ID] = true
			progress = true
		}
		if !progress {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照 Kafka 字段分组存在循环或缺失父目录")
		}
		pending = append([]SnapshotKafkaFieldGroupRecord(nil), next...)
	}
	return nil
}

var _ = time.Time{}
