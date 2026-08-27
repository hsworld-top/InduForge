package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

// ProtocolPreviewConnectionRecord 是短时抓样所需的完整协议配置快照。
type ProtocolPreviewConnectionRecord struct {
	ID        string
	ProjectID string
	Name      string
	Type      string
	Status    string
	Config    map[string]any
}

// CreateAccessSourceRecordParams 表示接入源开发态记录摘要，不保存原始样本。
type CreateAccessSourceRecordParams struct {
	ProjectID    string
	ConnectionID string
	RecordType   string
	Title        string
	Status       string
	Protocol     string
	DurationMS   int64
	SampleCount  int
	Truncated    bool
	ErrorSummary string
	Detail       map[string]any
}

// GetPreviewConnection 按连接类型读取短时抓样需要的配置。
func (r *ProtocolConnectionRepository) GetPreviewConnection(ctx context.Context, projectID, connectionID string) (*ProtocolPreviewConnectionRecord, error) {
	var (
		record        ProtocolPreviewConnectionRecord
		configPayload []byte
	)
	err := r.pool.QueryRow(ctx, `
		SELECT conn.id::text, conn.project_id, conn.name, conn.type, conn.status,
		       CASE conn.type
		         WHEN 'kafka' THEN jsonb_build_object(
		           'brokers', COALESCE(NULLIF(conn.metadata->>'brokers', ''), kafka.brokers),
		           'topic', kafka.topic,
		           'consumerGroup', kafka.consumer_group,
		           'startPosition', kafka.start_position,
		           'options', COALESCE(conn.metadata->'options', kafka.options, '{}'::jsonb)
		         )
		         WHEN 'http' THEN jsonb_build_object(
		           'baseUrl', http_cfg.base_url,
		           'method', http_cfg.method,
		           'headers', http_cfg.headers,
		           'timeoutMs', http_cfg.timeout_ms,
		           'bodyTemplate', http_cfg.body_template
		         )
		         WHEN 'websocket' THEN jsonb_build_object(
		           'url', ws.url,
		           'topic', ws.topic,
		           'headers', ws.headers,
		           'heartbeatIntervalMs', ws.heartbeat_interval_ms
		         )
		         WHEN 'redis' THEN jsonb_build_object(
		           'address', redis_cfg.address,
		           'db', redis_cfg.db,
		           'username', redis_cfg.username,
		           'keyPattern', redis_cfg.key_pattern,
		           'mode', redis_cfg.mode,
		           'options', redis_cfg.options
		         )
		         ELSE '{}'::jsonb
		       END AS config
		FROM data_connections conn
		LEFT JOIN data_kafka_configs kafka ON kafka.connection_id = conn.id
		LEFT JOIN data_http_configs http_cfg ON http_cfg.connection_id = conn.id
		LEFT JOIN data_websocket_configs ws ON ws.connection_id = conn.id
		LEFT JOIN data_redis_configs redis_cfg ON redis_cfg.connection_id = conn.id
		WHERE conn.project_id = $1
		  AND conn.id = $2
		  AND conn.type IN ('kafka', 'http', 'websocket', 'redis')
	`, projectID, connectionID).Scan(&record.ID, &record.ProjectID, &record.Name, &record.Type, &record.Status, &configPayload)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "协议预览配置不存在")
		}
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取协议预览配置失败", err)
	}
	if err := json.Unmarshal(configPayload, &record.Config); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析协议预览配置失败", err)
	}
	if record.Config == nil {
		record.Config = map[string]any{}
	}
	return &record, nil
}

// CreateAccessSourceRecord 写入开发态抓样摘要，避免持久化原始样本和 payload。
func (r *ProtocolConnectionRepository) CreateAccessSourceRecord(ctx context.Context, params CreateAccessSourceRecordParams) error {
	detailPayload, err := json.Marshal(params.Detail)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "接入源记录 detail 格式无效", err)
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO data_access_source_records (
			project_id,
			connection_id,
			record_type,
			title,
			status,
			protocol,
			duration_ms,
			sample_count,
			truncated,
			error_summary,
			detail
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NULLIF($10, ''), $11::jsonb)
	`, params.ProjectID, params.ConnectionID, params.RecordType, params.Title, params.Status, params.Protocol, params.DurationMS, params.SampleCount, params.Truncated, params.ErrorSummary, string(detailPayload))
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入接入源预览记录失败", err)
	}
	return nil
}

func accessSourceRecordCreatedAtNow() time.Time {
	return time.Now().UTC()
}
