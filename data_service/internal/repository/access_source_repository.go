package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

// AccessSourceDataPointCount 表示接入源关联的数据点数量。
type AccessSourceDataPointCount struct {
	ConnectionID string
	Count        int
}

// AccessSourceRecord 表示接入源最近开发态记录摘要。
type AccessSourceRecord struct {
	ConnectionID string
	RecordType   string
	Title        string
	Status       string
	Detail       map[string]any
	CreatedAt    time.Time
}

// AccessSourceRepository 聚合连接、查询、MQTT 与数据点之间的开发态读模型。
// 说明：这里不拥有领域写入逻辑，只服务接入源工作区的跨表只读视图。
type AccessSourceRepository struct {
	pool *pgxpool.Pool
}

// NewAccessSourceRepository 创建接入源聚合仓储。
func NewAccessSourceRepository(pool *pgxpool.Pool) *AccessSourceRepository {
	return &AccessSourceRepository{pool: pool}
}

// CountDataPointsByConnections 按连接聚合其直接或间接映射的数据点数量。
func (r *AccessSourceRepository) CountDataPointsByConnections(ctx context.Context, projectID string) (map[string]int, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT
            conn.id::text AS connection_id,
            COUNT(DISTINCT dp.id)::int AS datapoint_count
        FROM data_connections conn
        LEFT JOIN data_queries q
          ON q.project_id = conn.project_id
         AND q.connection_id = conn.id
        LEFT JOIN data_mqtt_subscriptions sub
          ON sub.project_id = conn.project_id
         AND sub.connection_id = conn.id
        LEFT JOIN data_mqtt_tags tag
          ON tag.project_id = conn.project_id
         AND tag.subscription_id = sub.id
        LEFT JOIN data_points dp
          ON dp.project_id = conn.project_id
         AND (
              (dp.source_type = 'db.query' AND dp.source_id = q.id)
           OR (dp.source_type = 'mqtt.subscription' AND dp.source_id = sub.id)
           OR (dp.source_type = 'mqtt.tag' AND dp.source_id = tag.id)
           OR (dp.source_id = conn.id)
         )
        WHERE conn.project_id = $1
        GROUP BY conn.id
    `, projectID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计接入源数据点数量失败", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var item AccessSourceDataPointCount
		if err := rows.Scan(&item.ConnectionID, &item.Count); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取接入源数据点数量失败", err)
		}
		result[item.ConnectionID] = item.Count
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历接入源数据点数量失败", err)
	}
	return result, nil
}

// ListRecentRecords 返回项目内每个接入源最近一条开发态记录。
func (r *AccessSourceRepository) ListRecentRecords(ctx context.Context, projectID string) (map[string]AccessSourceRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT DISTINCT ON (records.connection_id)
            records.connection_id,
            records.record_type,
            records.title,
            records.status,
            records.detail,
            records.created_at
        FROM (
            SELECT
                sub.connection_id::text AS connection_id,
                'mqtt.message' AS record_type,
                '收到 MQTT 消息' AS title,
                'ok' AS status,
                jsonb_build_object('topic', msg.topic) AS detail,
                msg.received_at AS created_at
            FROM data_mqtt_messages msg
            JOIN data_mqtt_subscriptions sub
              ON sub.id = msg.subscription_id
             AND sub.project_id = msg.project_id
            WHERE msg.project_id = $1
            UNION ALL
            SELECT
                connection_id::text,
                record_type,
                title,
                status,
                detail,
                created_at
            FROM data_access_source_records
            WHERE project_id = $1
        ) records
        ORDER BY records.connection_id, records.created_at DESC
    `, projectID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询接入源最近记录失败", err)
	}
	defer rows.Close()

	result := make(map[string]AccessSourceRecord)
	for rows.Next() {
		var (
			item        AccessSourceRecord
			detailBytes []byte
		)
		if err := rows.Scan(&item.ConnectionID, &item.RecordType, &item.Title, &item.Status, &detailBytes, &item.CreatedAt); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取接入源最近记录失败", err)
		}
		item.Detail = unmarshalAccessSourceRecordDetail(detailBytes)
		result[item.ConnectionID] = item
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历接入源最近记录失败", err)
	}
	return result, nil
}

// ListRecordsByConnection 返回单个接入源的最近开发态记录。
func (r *AccessSourceRepository) ListRecordsByConnection(ctx context.Context, projectID, connectionID string, limit int) ([]AccessSourceRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	rows, err := r.pool.Query(ctx, `
        SELECT
            records.connection_id,
            records.record_type,
            records.title,
            records.status,
            records.detail,
            records.created_at
        FROM (
            SELECT
                sub.connection_id::text AS connection_id,
                'mqtt.message' AS record_type,
                '收到 MQTT 消息' AS title,
                'ok' AS status,
                jsonb_build_object('topic', msg.topic) AS detail,
                msg.received_at AS created_at
            FROM data_mqtt_messages msg
            JOIN data_mqtt_subscriptions sub
              ON sub.id = msg.subscription_id
             AND sub.project_id = msg.project_id
            WHERE msg.project_id = $1
              AND sub.connection_id = $2
            UNION ALL
            SELECT
                connection_id::text,
                record_type,
                title,
                status,
                detail,
                created_at
            FROM data_access_source_records
            WHERE project_id = $1
              AND connection_id = $2
        ) records
        ORDER BY records.created_at DESC
        LIMIT $3
    `, projectID, connectionID, limit)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询接入源记录失败", err)
	}
	defer rows.Close()

	records := make([]AccessSourceRecord, 0)
	for rows.Next() {
		var (
			item        AccessSourceRecord
			detailBytes []byte
		)
		if err := rows.Scan(&item.ConnectionID, &item.RecordType, &item.Title, &item.Status, &detailBytes, &item.CreatedAt); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取接入源记录失败", err)
		}
		item.Detail = unmarshalAccessSourceRecordDetail(detailBytes)
		records = append(records, item)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历接入源记录失败", err)
	}
	return records, nil
}

func unmarshalAccessSourceRecordDetail(payload []byte) map[string]any {
	if len(payload) == 0 {
		return map[string]any{}
	}
	var detail map[string]any
	if err := json.Unmarshal(payload, &detail); err != nil || detail == nil {
		return map[string]any{}
	}
	return detail
}
