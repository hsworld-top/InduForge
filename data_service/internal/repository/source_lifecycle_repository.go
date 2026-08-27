package repository

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

// SourceDeleteImpactRecord 是接入源删除前的精确影响摘要。
type SourceDeleteImpactRecord struct {
	ScopeType           string                      `json:"scopeType"`
	ScopeID             string                      `json:"scopeId"`
	Name                string                      `json:"name"`
	GeneratedPointCount int                         `json:"generatedPointCount"`
	OwnedResources      []SourceOwnedResourceRecord `json:"ownedResources"`
	BlockingUsages      []SourceBlockingUsageRecord `json:"blockingUsages"`
}

type SourceOwnedResourceRecord struct {
	Type  string `json:"type"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

type SourceBlockingUsageRecord struct {
	Type     string                     `json:"type"`
	Label    string                     `json:"label"`
	Count    int                        `json:"count"`
	Examples []SourceBlockingUsageEntry `json:"examples"`
}

type SourceBlockingUsageEntry struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	DatapointPath string `json:"datapointPath,omitempty"`
}

func (impact SourceDeleteImpactRecord) CanDelete() bool {
	for _, usage := range impact.BlockingUsages {
		if usage.Count > 0 {
			return false
		}
	}
	return true
}

type sourceLifecycleQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (r *ConnectionRepository) GetDeleteImpact(ctx context.Context, projectID, connectionID string) (*SourceDeleteImpactRecord, error) {
	return loadSourceDeleteImpact(ctx, r.pool, projectID, "connection", connectionID)
}

func (r *CollectorRepository) GetDeleteImpact(ctx context.Context, projectID, connectionID string) (*SourceDeleteImpactRecord, error) {
	return loadSourceDeleteImpact(ctx, r.pool, projectID, "collector_connection", connectionID)
}

func (r *ConnectionRepository) DeleteWithImpact(ctx context.Context, projectID, connectionID, userID string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启接入源删除事务失败", err)
	}
	defer tx.Rollback(ctx)
	if err := lockSourceProject(ctx, tx, projectID); err != nil {
		return err
	}
	impact, err := loadSourceDeleteImpact(ctx, tx, projectID, "connection", connectionID)
	if err != nil {
		return err
	}
	if !impact.CanDelete() {
		return sourceOccupiedError(impact)
	}
	if err := invalidateSourceDatapoints(ctx, tx, projectID, "connection", connectionID, userID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM data_connections WHERE project_id=$1 AND id=$2`, projectID, connectionID)
	if err != nil {
		return translateConnectionWriteError("删除连接失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "连接不存在")
	}
	if err := tx.Commit(ctx); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交接入源删除事务失败", err)
	}
	return nil
}

func (r *CollectorRepository) DeleteWithImpact(ctx context.Context, projectID, connectionID, userID string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启工业连接删除事务失败", err)
	}
	defer tx.Rollback(ctx)
	if err := lockSourceProject(ctx, tx, projectID); err != nil {
		return err
	}
	impact, err := loadSourceDeleteImpact(ctx, tx, projectID, "collector_connection", connectionID)
	if err != nil {
		return err
	}
	if !impact.CanDelete() {
		return sourceOccupiedError(impact)
	}
	if err := invalidateSourceDatapoints(ctx, tx, projectID, "collector_connection", connectionID, userID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM data_collector_connections WHERE project_id=$1 AND id=$2`, projectID, connectionID)
	if err != nil {
		return translateCollectorWriteError("删除工业采集连接失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "工业采集连接不存在")
	}
	if err := tx.Commit(ctx); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交工业连接删除事务失败", err)
	}
	return nil
}

func lockSourceProject(ctx context.Context, tx pgx.Tx, projectID string) error {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 20260825))`, projectID); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "锁定工程配置失败", err)
	}
	return nil
}

func loadSourceDeleteImpact(ctx context.Context, q sourceLifecycleQuerier, projectID, scopeType, scopeID string) (*SourceDeleteImpactRecord, error) {
	if scopeType != "connection" && scopeType != "collector_connection" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "接入源作用域无效")
	}
	table := "data_connections"
	if scopeType == "collector_connection" {
		table = "data_collector_connections"
	}
	var name string
	if err := q.QueryRow(ctx, `SELECT name FROM `+table+` WHERE project_id=$1 AND id=$2`, projectID, scopeID).Scan(&name); err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "接入源不存在")
		}
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取接入源失败", err)
	}

	impact := &SourceDeleteImpactRecord{ScopeType: scopeType, ScopeID: scopeID, Name: name, OwnedResources: []SourceOwnedResourceRecord{}, BlockingUsages: []SourceBlockingUsageRecord{}}
	pointWhere := sourceDatapointWhere(scopeType, 2)
	if err := q.QueryRow(ctx, `WITH `+historyStorageDataPointOriginsCTE+` SELECT COUNT(*) FROM datapoint_origins origin WHERE origin.project_id=$1 AND `+pointWhere, projectID, scopeID).Scan(&impact.GeneratedPointCount); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计来源数据点失败", err)
	}

	owned, err := loadSourceOwnedResources(ctx, q, projectID, scopeType, scopeID)
	if err != nil {
		return nil, err
	}
	impact.OwnedResources = owned
	blocking, err := loadSourceBlockingUsages(ctx, q, projectID, scopeType, scopeID)
	if err != nil {
		return nil, err
	}
	impact.BlockingUsages = blocking
	return impact, nil
}

func loadSourceOwnedResources(ctx context.Context, q sourceLifecycleQuerier, projectID, scopeType, scopeID string) ([]SourceOwnedResourceRecord, error) {
	query := `SELECT resource_type,label,count_value FROM (`
	if scopeType == "collector_connection" {
		query += `SELECT 'collector_point'::text resource_type,'采集变量'::text label,COUNT(*)::int count_value FROM data_collector_points WHERE project_id=$1 AND connection_id=$2
		UNION ALL SELECT 'collector_group','变量目录',COUNT(*)::int FROM data_collector_point_groups WHERE project_id=$1 AND connection_id=$2`
	} else {
		query += `SELECT 'query'::text resource_type,'保存查询'::text label,COUNT(*)::int count_value FROM data_queries WHERE project_id=$1 AND connection_id=$2
		UNION ALL SELECT 'mqtt_subscription','MQTT 订阅',COUNT(*)::int FROM data_mqtt_subscriptions WHERE project_id=$1 AND connection_id=$2
		UNION ALL SELECT 'kafka_mapping','Kafka 映射',COUNT(*)::int FROM data_kafka_topic_mappings WHERE project_id=$1 AND connection_id=$2
		UNION ALL SELECT 'http_request','HTTP 请求',COUNT(*)::int FROM data_http_requests WHERE project_id=$1 AND connection_id=$2
		UNION ALL SELECT 'websocket_session','WebSocket 会话',COUNT(*)::int FROM data_websocket_sessions WHERE project_id=$1 AND connection_id=$2
		UNION ALL SELECT 'realtime_key','实时 Key',COUNT(*)::int FROM data_realtime_keys WHERE project_id=$1 AND connection_id=$2`
	}
	query += `) resources WHERE count_value > 0 ORDER BY resource_type`
	rows, err := q.Query(ctx, query, projectID, scopeID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计来源子对象失败", err)
	}
	defer rows.Close()
	result := []SourceOwnedResourceRecord{}
	for rows.Next() {
		var item SourceOwnedResourceRecord
		if err := rows.Scan(&item.Type, &item.Label, &item.Count); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func loadSourceBlockingUsages(ctx context.Context, q sourceLifecycleQuerier, projectID, scopeType, scopeID string) ([]SourceBlockingUsageRecord, error) {
	pointWhere := sourceDatapointWhere(scopeType, 2)
	query := `WITH ` + historyStorageDataPointOriginsCTE + `, impacted AS (
		SELECT origin.datapoint_id FROM datapoint_origins origin WHERE origin.project_id=$1 AND ` + pointWhere + `
	), usages AS (
		SELECT 'alarm'::text usage_type,'报警项'::text label,item.id::text object_id,item.display_name AS name,point.path
		FROM data_alarm_items item JOIN data_points point ON point.id=item.datapoint_id
		WHERE item.project_id=$1 AND item.datapoint_id IN (SELECT datapoint_id FROM impacted)
		UNION ALL
		SELECT 'alarm','组合报警',item.id::text,item.display_name AS name,point.path
		FROM data_alarm_item_inputs input JOIN data_alarm_items item ON item.id=input.alarm_item_id JOIN data_points point ON point.id=input.datapoint_id
		WHERE item.project_id=$1 AND input.datapoint_id IN (SELECT datapoint_id FROM impacted)
		UNION ALL
		SELECT 'compute','计算单元',unit.id::text,unit.name,point.path
		FROM data_compute_unit_datapoint_refs ref JOIN data_compute_units unit ON unit.id=ref.compute_unit_id JOIN data_points point ON point.id=ref.datapoint_id
		WHERE ref.project_id=$1 AND ref.datapoint_id IN (SELECT datapoint_id FROM impacted)
	), ranked AS (
		SELECT *,ROW_NUMBER() OVER(PARTITION BY usage_type ORDER BY name,object_id) row_num,COUNT(*) OVER(PARTITION BY usage_type)::int usage_count FROM usages
	)
	SELECT usage_type,label,object_id,name,path,usage_count FROM ranked WHERE row_num <= 5 ORDER BY usage_type,row_num`
	rows, err := q.Query(ctx, query, projectID, scopeID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计来源阻断引用失败", err)
	}
	defer rows.Close()
	byType := map[string]*SourceBlockingUsageRecord{}
	order := []string{}
	for rows.Next() {
		var usageType, label, id, name, path string
		var count int
		if err := rows.Scan(&usageType, &label, &id, &name, &path, &count); err != nil {
			return nil, err
		}
		item := byType[usageType]
		if item == nil {
			item = &SourceBlockingUsageRecord{Type: usageType, Label: label, Count: count, Examples: []SourceBlockingUsageEntry{}}
			byType[usageType] = item
			order = append(order, usageType)
		}
		item.Examples = append(item.Examples, SourceBlockingUsageEntry{ID: id, Name: name, DatapointPath: path})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if scopeType == "connection" {
		var count int
		if err := q.QueryRow(ctx, `SELECT COUNT(*) FROM data_history_storage_targets WHERE project_id=$1 AND connection_id=$2`, projectID, scopeID).Scan(&count); err != nil {
			return nil, err
		}
		if count > 0 {
			byType["history_target"] = &SourceBlockingUsageRecord{Type: "history_target", Label: "历史存储目标", Count: count, Examples: []SourceBlockingUsageEntry{}}
			order = append(order, "history_target")
		}
	}
	result := make([]SourceBlockingUsageRecord, 0, len(order))
	for _, key := range order {
		result = append(result, *byType[key])
	}
	return result, nil
}

func invalidateSourceDatapoints(ctx context.Context, tx pgx.Tx, projectID, scopeType, scopeID, userID string) error {
	pointWhere := sourceDatapointWhere(scopeType, 2)
	_, err := tx.Exec(ctx, `WITH `+historyStorageDataPointOriginsCTE+`, impacted AS (
		SELECT origin.datapoint_id FROM datapoint_origins origin WHERE origin.project_id=$1 AND `+pointWhere+`
	) UPDATE data_points SET status='invalid',updated_by=NULLIF($3,'')::uuid,updated_at=now()
	WHERE project_id=$1 AND id IN (SELECT datapoint_id FROM impacted)`, projectID, scopeID, userID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "标记来源数据点失效失败", err)
	}
	return nil
}

func sourceDatapointWhere(scopeType string, argument int) string {
	if scopeType == "collector_connection" {
		return fmt.Sprintf("origin.collector_connection_id=$%d", argument)
	}
	return fmt.Sprintf("origin.access_source_id=$%d", argument)
}

func sourceOccupiedError(impact *SourceDeleteImpactRecord) error {
	parts := make([]string, 0, len(impact.BlockingUsages))
	for _, usage := range impact.BlockingUsages {
		if usage.Count > 0 {
			parts = append(parts, fmt.Sprintf("%s %d 项", usage.Label, usage.Count))
		}
	}
	return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "接入源仍被引用，请先解除："+strings.Join(parts, "、"))
}
