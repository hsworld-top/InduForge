package repository

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const historyStorageConfigColumns = `id, project_id, access_source_id, collector_connection_id, datapoint_id,
       is_enabled, write_mode, interval_ms, deadband, max_silence_ms, offline_behavior,
       created_by, updated_by, created_at, updated_at`

// historyStorageDataPointOriginsCTE 统一解析数据点所属来源，供来源列表、单点详情和例外统计复用。
// source_config 优先于结构化关联；Collector 点只归属工业采集连接。
const historyStorageDataPointOriginsCTE = `datapoint_origins AS (
    SELECT dp.id AS datapoint_id,
           dp.project_id,
           dp.name AS datapoint_name,
           dp.path AS datapoint_path,
           dp.data_type,
           CASE WHEN dp.source_type = 'collector.point' THEN NULL
                ELSE COALESCE(
                    NULLIF(BTRIM(dp.source_config->>'connectionId'), ''),
                    NULLIF(BTRIM(dp.source_config->>'sourceConnectionId'), ''),
                    source_query.connection_id::text,
                    direct_subscription.connection_id::text,
                    tag_subscription.connection_id::text,
                    dp.source_id::text
                ) END AS access_source_id,
           CASE WHEN dp.source_type = 'collector.point' THEN collector_point.connection_id::text END AS collector_connection_id
    FROM data_points dp
    LEFT JOIN data_queries source_query
      ON dp.source_type = 'db.query' AND source_query.id = dp.source_id AND source_query.project_id = dp.project_id
    LEFT JOIN data_mqtt_subscriptions direct_subscription
      ON dp.source_type = 'mqtt.subscription' AND direct_subscription.id = dp.source_id AND direct_subscription.project_id = dp.project_id
    LEFT JOIN data_mqtt_tags tag
      ON dp.source_type = 'mqtt.tag' AND tag.id = dp.source_id AND tag.project_id = dp.project_id
    LEFT JOIN data_mqtt_subscriptions tag_subscription
      ON tag_subscription.id = tag.subscription_id AND tag_subscription.project_id = dp.project_id
    LEFT JOIN data_collector_points collector_point
      ON dp.source_type = 'collector.point' AND collector_point.id = dp.source_id AND collector_point.project_id = dp.project_id
    WHERE dp.project_id = $1
)`

type HistoryStorageRepository struct {
	pool *pgxpool.Pool
}

type HistoryStorageScope struct {
	Type string
	ID   string
}

type HistoryStorageConfigRecord struct {
	ID                    string                       `json:"id"`
	ProjectID             string                       `json:"projectId"`
	AccessSourceID        *string                      `json:"accessSourceId,omitempty"`
	CollectorConnectionID *string                      `json:"collectorConnectionId,omitempty"`
	DatapointID           *string                      `json:"datapointId,omitempty"`
	IsEnabled             bool                         `json:"isEnabled"`
	WriteMode             string                       `json:"writeMode"`
	IntervalMS            *int64                       `json:"intervalMs,omitempty"`
	Deadband              *float64                     `json:"deadband,omitempty"`
	MaxSilenceMS          *int64                       `json:"maxSilenceMs,omitempty"`
	OfflineBehavior       string                       `json:"offlineBehavior"`
	CreatedBy             string                       `json:"createdBy"`
	UpdatedBy             *string                      `json:"updatedBy,omitempty"`
	CreatedAt             time.Time                    `json:"createdAt"`
	UpdatedAt             time.Time                    `json:"updatedAt"`
	Targets               []HistoryStorageTargetRecord `json:"targets"`
}

type HistoryStorageTargetRecord struct {
	ID               string    `json:"id"`
	ProjectID        string    `json:"projectId"`
	ConfigID         string    `json:"configId"`
	ConnectionID     string    `json:"connectionId"`
	ConnectionName   string    `json:"connectionName"`
	ConnectionType   string    `json:"connectionType"`
	ConnectionStatus string    `json:"connectionStatus"`
	IsPrimary        bool      `json:"isPrimary"`
	SortOrder        int       `json:"sortOrder"`
	RetentionDays    *int64    `json:"retentionDays"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type HistoryStorageSourceRecord struct {
	ScopeType          string
	ScopeID            string
	Name               string
	SourceType         string
	DatapointCount     int
	PointOverrideCount int
	ConfigID           *string
	IsEnabled          bool
	WriteMode          *string
	TargetCount        int
	PrimaryTargetName  *string
	PrimaryTargetType  *string
	RetentionDays      *int64
	Total              int
}

type HistoryStorageSourceListFilter struct {
	Search       string
	ScopeType    string
	HistoryState string
	Page         int
	PageSize     int
}

type HistoryStorageTargetOptionRecord struct {
	ID        string
	Name      string
	Type      string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type HistoryStorageDataPointOriginRecord struct {
	DatapointID     string
	DatapointName   string
	DatapointPath   string
	DataType        string
	ScopeType       string
	ScopeID         string
	ScopeName       string
	ScopeSourceType string
}

type SaveHistoryStorageConfigParams struct {
	ProjectID       string
	UserID          string
	Scope           HistoryStorageScope
	IsEnabled       bool
	WriteMode       string
	IntervalMS      *int64
	Deadband        *float64
	MaxSilenceMS    *int64
	OfflineBehavior string
	Targets         []SaveHistoryStorageTargetParams
	ReplaceTargets  bool
}

type SaveHistoryStorageTargetParams struct {
	ConnectionID  string
	IsPrimary     bool
	SortOrder     int
	RetentionDays *int64
}

func NewHistoryStorageRepository(pool *pgxpool.Pool) *HistoryStorageRepository {
	return &HistoryStorageRepository{pool: pool}
}

func (r *HistoryStorageRepository) ListSources(ctx context.Context, projectID string, filter HistoryStorageSourceListFilter) ([]HistoryStorageSourceRecord, int, error) {
	page, pageSize := filter.Page, filter.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	args := []any{projectID}
	conditions := []string{"1=1"}
	if search := strings.TrimSpace(filter.Search); search != "" {
		args = append(args, "%"+search+"%")
		conditions = append(conditions, fmt.Sprintf("source_name ILIKE $%d", len(args)))
	}
	if scopeType := strings.TrimSpace(filter.ScopeType); scopeType != "" {
		args = append(args, scopeType)
		conditions = append(conditions, fmt.Sprintf("scope_type = $%d", len(args)))
	}
	if state := strings.TrimSpace(filter.HistoryState); state != "" {
		enabled := state == "enabled"
		args = append(args, enabled)
		conditions = append(conditions, fmt.Sprintf("is_enabled = $%d", len(args)))
	}
	args = append(args, pageSize, (page-1)*pageSize)
	query := fmt.Sprintf(`WITH `+historyStorageDataPointOriginsCTE+`, source_rows AS (
    SELECT 'access_source'::text AS scope_type, conn.id::text AS scope_id,
           conn.name AS source_name, conn.type AS source_type,
           COUNT(DISTINCT origin.datapoint_id)::int AS datapoint_count,
           COUNT(DISTINCT point_config.id)::int AS point_override_count,
           config.id::text AS config_id, COALESCE(config.is_enabled, false) AS is_enabled,
           config.write_mode,
           COUNT(DISTINCT target.id)::int AS target_count,
           MAX(target_conn.name) FILTER (WHERE target.is_primary) AS primary_target_name,
           MAX(target_conn.type) FILTER (WHERE target.is_primary) AS primary_target_type,
           MAX(target.retention_days) FILTER (WHERE target.is_primary) AS retention_days
    FROM data_connections conn
    LEFT JOIN datapoint_origins origin ON origin.access_source_id = conn.id::text
    LEFT JOIN data_history_storage_configs point_config ON point_config.project_id = conn.project_id AND point_config.datapoint_id = origin.datapoint_id
    LEFT JOIN data_history_storage_configs config ON config.project_id = conn.project_id AND config.access_source_id = conn.id
    LEFT JOIN data_history_storage_targets target ON target.config_id = config.id
    LEFT JOIN data_connections target_conn ON target_conn.id = target.connection_id
    WHERE conn.project_id = $1
    GROUP BY conn.id, conn.name, conn.type, conn.display_order, conn.created_at, config.id, config.is_enabled, config.write_mode
    UNION ALL
    SELECT 'collector_connection'::text AS scope_type, collector.id::text AS scope_id,
           collector.name AS source_name, collector.protocol_family AS source_type,
           COUNT(DISTINCT origin.datapoint_id)::int AS datapoint_count,
           COUNT(DISTINCT point_config.id)::int AS point_override_count,
           config.id::text AS config_id, COALESCE(config.is_enabled, false) AS is_enabled,
           config.write_mode,
           COUNT(DISTINCT target.id)::int AS target_count,
           MAX(target_conn.name) FILTER (WHERE target.is_primary) AS primary_target_name,
           MAX(target_conn.type) FILTER (WHERE target.is_primary) AS primary_target_type,
           MAX(target.retention_days) FILTER (WHERE target.is_primary) AS retention_days
    FROM data_collector_connections collector
    LEFT JOIN datapoint_origins origin ON origin.collector_connection_id = collector.id::text
    LEFT JOIN data_history_storage_configs point_config ON point_config.project_id = collector.project_id AND point_config.datapoint_id = origin.datapoint_id
    LEFT JOIN data_history_storage_configs config ON config.project_id = collector.project_id AND config.collector_connection_id = collector.id
    LEFT JOIN data_history_storage_targets target ON target.config_id = config.id
    LEFT JOIN data_connections target_conn ON target_conn.id = target.connection_id
    WHERE collector.project_id = $1
    GROUP BY collector.id, collector.name, collector.protocol_family, collector.created_at, config.id, config.is_enabled, config.write_mode
)
SELECT scope_type, scope_id, source_name, source_type, datapoint_count, point_override_count,
       config_id, is_enabled, write_mode, target_count, primary_target_name, primary_target_type,
       retention_days, COUNT(*) OVER()::int
FROM source_rows
WHERE %s
ORDER BY source_name ASC, scope_type ASC, scope_id ASC
LIMIT $%d OFFSET $%d`, strings.Join(conditions, " AND "), len(args)-1, len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, wrapHistoryStorageError("查询历史存储来源失败", err)
	}
	defer rows.Close()
	result := make([]HistoryStorageSourceRecord, 0)
	total := 0
	for rows.Next() {
		var item HistoryStorageSourceRecord
		if err := rows.Scan(&item.ScopeType, &item.ScopeID, &item.Name, &item.SourceType, &item.DatapointCount,
			&item.PointOverrideCount, &item.ConfigID, &item.IsEnabled, &item.WriteMode, &item.TargetCount,
			&item.PrimaryTargetName, &item.PrimaryTargetType, &item.RetentionDays, &item.Total); err != nil {
			return nil, 0, wrapHistoryStorageError("读取历史存储来源失败", err)
		}
		total = item.Total
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, wrapHistoryStorageError("遍历历史存储来源失败", err)
	}
	return result, total, nil
}

func (r *HistoryStorageRepository) ListTargetOptions(ctx context.Context, projectID string) ([]HistoryStorageTargetOptionRecord, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, type, status, created_at, updated_at
        FROM data_connections WHERE project_id=$1 AND type IN ('builtin.timeseries','tdengine')
        ORDER BY CASE WHEN type='builtin.timeseries' THEN 0 ELSE 1 END, display_order, created_at, id`, projectID)
	if err != nil {
		return nil, wrapHistoryStorageError("查询历史存储目标失败", err)
	}
	defer rows.Close()
	result := make([]HistoryStorageTargetOptionRecord, 0)
	for rows.Next() {
		var item HistoryStorageTargetOptionRecord
		if err := rows.Scan(&item.ID, &item.Name, &item.Type, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, wrapHistoryStorageError("读取历史存储目标失败", err)
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *HistoryStorageRepository) GetTargetOptionsByIDs(ctx context.Context, projectID string, ids []string) ([]HistoryStorageTargetOptionRecord, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, type, status, created_at, updated_at
        FROM data_connections WHERE project_id=$1 AND id=ANY($2::uuid[]) AND type IN ('builtin.timeseries','tdengine')`, projectID, ids)
	if err != nil {
		return nil, wrapHistoryStorageError("校验历史存储目标失败", err)
	}
	defer rows.Close()
	result := make([]HistoryStorageTargetOptionRecord, 0, len(ids))
	for rows.Next() {
		var item HistoryStorageTargetOptionRecord
		if err := rows.Scan(&item.ID, &item.Name, &item.Type, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *HistoryStorageRepository) ScopeExists(ctx context.Context, projectID string, scope HistoryStorageScope) (bool, error) {
	table, err := historyStorageScopeTable(scope.Type)
	if err != nil {
		return false, err
	}
	var exists bool
	err = r.pool.QueryRow(ctx, fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s WHERE project_id=$1 AND id=$2)`, table), projectID, scope.ID).Scan(&exists)
	if err != nil {
		return false, wrapHistoryStorageError("校验历史存储作用域失败", err)
	}
	return exists, nil
}

func (r *HistoryStorageRepository) GetConfigByScope(ctx context.Context, projectID string, scope HistoryStorageScope) (*HistoryStorageConfigRecord, error) {
	column, err := historyStorageScopeColumn(scope.Type)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx, `SELECT `+historyStorageConfigColumns+` FROM data_history_storage_configs WHERE project_id=$1 AND `+column+`=$2`, projectID, scope.ID)
	config, err := scanHistoryStorageConfig(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapHistoryStorageError("读取历史存储配置失败", err)
	}
	targets, err := r.listTargetsByConfig(ctx, projectID, config.ID)
	if err != nil {
		return nil, err
	}
	config.Targets = targets
	return &config, nil
}

func (r *HistoryStorageRepository) CountPointOverridesByScope(ctx context.Context, projectID string, scope HistoryStorageScope) (int, error) {
	var query string
	switch scope.Type {
	case "access_source":
		query = `WITH ` + historyStorageDataPointOriginsCTE + `
          SELECT COUNT(*) FROM data_history_storage_configs config
          JOIN datapoint_origins origin ON origin.datapoint_id=config.datapoint_id AND origin.project_id=config.project_id
          WHERE config.project_id=$1 AND origin.access_source_id=$2`
	case "collector_connection":
		query = `WITH ` + historyStorageDataPointOriginsCTE + `
          SELECT COUNT(*) FROM data_history_storage_configs config
          JOIN datapoint_origins origin ON origin.datapoint_id=config.datapoint_id AND origin.project_id=config.project_id
          WHERE config.project_id=$1 AND origin.collector_connection_id=$2`
	default:
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "只有来源作用域可以统计单点例外")
	}
	var count int
	if err := r.pool.QueryRow(ctx, query, projectID, scope.ID).Scan(&count); err != nil {
		return 0, wrapHistoryStorageError("统计历史存储单点例外失败", err)
	}
	return count, nil
}

func (r *HistoryStorageRepository) GetDataPointOrigin(ctx context.Context, projectID, datapointID string) (*HistoryStorageDataPointOriginRecord, error) {
	var item HistoryStorageDataPointOriginRecord
	err := r.pool.QueryRow(ctx, `WITH `+historyStorageDataPointOriginsCTE+`
	  SELECT origin.datapoint_id, origin.datapoint_name, origin.datapoint_path, origin.data_type,
		CASE WHEN origin.collector_connection_id IS NOT NULL THEN 'collector_connection'
			 WHEN connection.id IS NOT NULL THEN 'access_source' ELSE '' END,
		COALESCE(origin.collector_connection_id, connection.id::text, ''),
		CASE WHEN origin.collector_connection_id IS NOT NULL THEN COALESCE(collector.name, '') ELSE COALESCE(connection.name, '') END,
		CASE WHEN origin.collector_connection_id IS NOT NULL THEN COALESCE(collector.protocol_family, '') ELSE COALESCE(connection.type, '') END
	  FROM datapoint_origins origin
	  LEFT JOIN data_collector_connections collector
		ON collector.project_id=origin.project_id AND collector.id::text=origin.collector_connection_id
	  LEFT JOIN data_connections connection
		ON connection.project_id=origin.project_id AND connection.id::text=origin.access_source_id
	  WHERE origin.datapoint_id=$2`, projectID, datapointID).Scan(
		&item.DatapointID, &item.DatapointName, &item.DatapointPath, &item.DataType,
		&item.ScopeType, &item.ScopeID, &item.ScopeName, &item.ScopeSourceType)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "数据点不存在")
		}
		return nil, wrapHistoryStorageError("解析数据点历史来源失败", err)
	}
	if strings.TrimSpace(item.ScopeID) == "" {
		item.ScopeType = ""
	}
	return &item, nil
}

func (r *HistoryStorageRepository) SaveConfig(ctx context.Context, params SaveHistoryStorageConfigParams) (*HistoryStorageConfigRecord, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, wrapHistoryStorageError("开启历史存储配置事务失败", err)
	}
	defer tx.Rollback(ctx)
	configID, err := upsertHistoryStorageConfig(ctx, tx, params)
	if err != nil {
		return nil, err
	}
	if params.ReplaceTargets {
		if _, err := tx.Exec(ctx, `DELETE FROM data_history_storage_targets WHERE project_id=$1 AND config_id=$2`, params.ProjectID, configID); err != nil {
			return nil, wrapHistoryStorageError("替换历史存储目标失败", err)
		}
		for _, target := range params.Targets {
			if _, err := tx.Exec(ctx, `INSERT INTO data_history_storage_targets
                (project_id,config_id,connection_id,is_primary,sort_order,retention_days)
                VALUES ($1,$2,$3,$4,$5,$6)`, params.ProjectID, configID, target.ConnectionID, target.IsPrimary, target.SortOrder, target.RetentionDays); err != nil {
				return nil, translateHistoryStorageWriteError("写入历史存储目标失败", err)
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, wrapHistoryStorageError("提交历史存储配置失败", err)
	}
	return r.GetConfigByScope(ctx, params.ProjectID, params.Scope)
}

func (r *HistoryStorageRepository) DeleteConfigByScope(ctx context.Context, projectID string, scope HistoryStorageScope) error {
	column, err := historyStorageScopeColumn(scope.Type)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `DELETE FROM data_history_storage_configs WHERE project_id=$1 AND `+column+`=$2`, projectID, scope.ID)
	if err != nil {
		return wrapHistoryStorageError("删除历史存储覆盖失败", err)
	}
	return nil
}

func (r *HistoryStorageRepository) ResolveDataPointIDs(ctx context.Context, projectID string, ids []string, filter *DataPointListFilter) ([]string, error) {
	if filter == nil {
		rows, err := r.pool.Query(ctx, `SELECT id FROM data_points WHERE project_id=$1 AND id=ANY($2::uuid[]) ORDER BY id`, projectID, ids)
		if err != nil {
			return nil, wrapHistoryStorageError("读取批量数据点失败", err)
		}
		defer rows.Close()
		return scanStringRows(rows)
	}
	whereSQL, args, err := buildDataPointWhereClause(projectID, *filter)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT id FROM data_points WHERE `+whereSQL+` ORDER BY id`, args...)
	if err != nil {
		return nil, wrapHistoryStorageError("按筛选读取批量数据点失败", err)
	}
	defer rows.Close()
	return scanStringRows(rows)
}

func (r *HistoryStorageRepository) SaveDataPointConfigsBatch(ctx context.Context, projectID, userID string, datapointIDs []string, behavior string, config *SaveHistoryStorageConfigParams) (int, error) {
	if len(datapointIDs) == 0 {
		return 0, nil
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, wrapHistoryStorageError("开启批量历史存储事务失败", err)
	}
	defer tx.Rollback(ctx)
	if behavior == "inherit" {
		tag, err := tx.Exec(ctx, `DELETE FROM data_history_storage_configs WHERE project_id=$1 AND datapoint_id=ANY($2::uuid[])`, projectID, datapointIDs)
		if err != nil {
			return 0, wrapHistoryStorageError("批量恢复来源设置失败", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return 0, err
		}
		return int(tag.RowsAffected()), nil
	}
	if behavior == "off" {
		if _, err := tx.Exec(ctx, `UPDATE data_history_storage_configs
            SET is_enabled=false,updated_by=$3,updated_at=now()
            WHERE project_id=$1 AND datapoint_id=ANY($2::uuid[])`, projectID, datapointIDs, userID); err != nil {
			return 0, translateHistoryStorageWriteError("批量关闭历史存储配置失败", err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO data_history_storage_configs
            (project_id,datapoint_id,is_enabled,write_mode,deadband,max_silence_ms,offline_behavior,created_by,updated_by)
            SELECT $1,point.id,false,'on_change',0,3600000,'store_stale',$3,$3
            FROM data_points point
            WHERE point.project_id=$1 AND point.id=ANY($2::uuid[])
              AND NOT EXISTS (
                SELECT 1 FROM data_history_storage_configs config
                WHERE config.project_id=$1 AND config.datapoint_id=point.id
              )`, projectID, datapointIDs, userID); err != nil {
			return 0, translateHistoryStorageWriteError("批量创建关闭的历史存储配置失败", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return 0, wrapHistoryStorageError("提交批量关闭历史存储配置失败", err)
		}
		return len(datapointIDs), nil
	}
	for _, datapointID := range datapointIDs {
		params := SaveHistoryStorageConfigParams{ProjectID: projectID, UserID: userID, Scope: HistoryStorageScope{Type: "datapoint", ID: datapointID}, IsEnabled: behavior == "custom", WriteMode: "on_change", Deadband: float64Pointer(0), MaxSilenceMS: int64Pointer(3600000), OfflineBehavior: "store_stale"}
		if config != nil {
			params.IsEnabled = config.IsEnabled
			params.WriteMode = config.WriteMode
			params.IntervalMS = config.IntervalMS
			params.Deadband = config.Deadband
			params.MaxSilenceMS = config.MaxSilenceMS
			params.OfflineBehavior = config.OfflineBehavior
			params.Targets = config.Targets
			params.ReplaceTargets = config.ReplaceTargets
		}
		configID, err := upsertHistoryStorageConfig(ctx, tx, params)
		if err != nil {
			return 0, err
		}
		if params.ReplaceTargets {
			if _, err := tx.Exec(ctx, `DELETE FROM data_history_storage_targets WHERE project_id=$1 AND config_id=$2`, projectID, configID); err != nil {
				return 0, err
			}
			for _, target := range params.Targets {
				if _, err := tx.Exec(ctx, `INSERT INTO data_history_storage_targets (project_id,config_id,connection_id,is_primary,sort_order,retention_days) VALUES ($1,$2,$3,$4,$5,$6)`, projectID, configID, target.ConnectionID, target.IsPrimary, target.SortOrder, target.RetentionDays); err != nil {
					return 0, translateHistoryStorageWriteError("批量写入历史存储目标失败", err)
				}
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, wrapHistoryStorageError("提交批量历史存储配置失败", err)
	}
	return len(datapointIDs), nil
}

func upsertHistoryStorageConfig(ctx context.Context, tx pgx.Tx, params SaveHistoryStorageConfigParams) (string, error) {
	column, err := historyStorageScopeColumn(params.Scope.Type)
	if err != nil {
		return "", err
	}
	var id string
	err = tx.QueryRow(ctx, `SELECT id FROM data_history_storage_configs WHERE project_id=$1 AND `+column+`=$2 FOR UPDATE`, params.ProjectID, params.Scope.ID).Scan(&id)
	if err == nil {
		_, err = tx.Exec(ctx, `UPDATE data_history_storage_configs SET is_enabled=$3,write_mode=$4,interval_ms=$5,deadband=$6,max_silence_ms=$7,offline_behavior=$8,updated_by=$9,updated_at=now() WHERE project_id=$1 AND id=$2`, params.ProjectID, id, params.IsEnabled, params.WriteMode, params.IntervalMS, params.Deadband, params.MaxSilenceMS, params.OfflineBehavior, params.UserID)
		if err != nil {
			return "", translateHistoryStorageWriteError("更新历史存储配置失败", err)
		}
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", wrapHistoryStorageError("锁定历史存储配置失败", err)
	}
	var accessSourceID, collectorConnectionID, datapointID any
	switch params.Scope.Type {
	case "access_source":
		accessSourceID = params.Scope.ID
	case "collector_connection":
		collectorConnectionID = params.Scope.ID
	case "datapoint":
		datapointID = params.Scope.ID
	}
	err = tx.QueryRow(ctx, `INSERT INTO data_history_storage_configs
      (project_id,access_source_id,collector_connection_id,datapoint_id,is_enabled,write_mode,interval_ms,deadband,max_silence_ms,offline_behavior,created_by,updated_by)
      VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11) RETURNING id`, params.ProjectID, accessSourceID, collectorConnectionID, datapointID,
		params.IsEnabled, params.WriteMode, params.IntervalMS, params.Deadband, params.MaxSilenceMS, params.OfflineBehavior, params.UserID).Scan(&id)
	if err != nil {
		return "", translateHistoryStorageWriteError("创建历史存储配置失败", err)
	}
	return id, nil
}

func (r *HistoryStorageRepository) listTargetsByConfig(ctx context.Context, projectID, configID string) ([]HistoryStorageTargetRecord, error) {
	rows, err := r.pool.Query(ctx, `SELECT target.id,target.project_id,target.config_id,target.connection_id,
        conn.name,conn.type,conn.status,target.is_primary,target.sort_order,target.retention_days,target.created_at,target.updated_at
      FROM data_history_storage_targets target JOIN data_connections conn ON conn.id=target.connection_id
      WHERE target.project_id=$1 AND target.config_id=$2
      ORDER BY target.is_primary DESC,target.sort_order,target.id`, projectID, configID)
	if err != nil {
		return nil, wrapHistoryStorageError("查询历史存储目标失败", err)
	}
	defer rows.Close()
	result := make([]HistoryStorageTargetRecord, 0)
	for rows.Next() {
		var item HistoryStorageTargetRecord
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.ConfigID, &item.ConnectionID, &item.ConnectionName, &item.ConnectionType, &item.ConnectionStatus, &item.IsPrimary, &item.SortOrder, &item.RetentionDays, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func historyStorageScopeColumn(scopeType string) (string, error) {
	switch strings.TrimSpace(scopeType) {
	case "access_source":
		return "access_source_id", nil
	case "collector_connection":
		return "collector_connection_id", nil
	case "datapoint":
		return "datapoint_id", nil
	default:
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "历史存储作用域类型不受支持")
	}
}

func historyStorageScopeTable(scopeType string) (string, error) {
	switch strings.TrimSpace(scopeType) {
	case "access_source":
		return "data_connections", nil
	case "collector_connection":
		return "data_collector_connections", nil
	case "datapoint":
		return "data_points", nil
	default:
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "历史存储作用域类型不受支持")
	}
}

type historyStorageScanner interface{ Scan(...any) error }

func scanHistoryStorageConfig(row historyStorageScanner) (HistoryStorageConfigRecord, error) {
	var item HistoryStorageConfigRecord
	err := row.Scan(&item.ID, &item.ProjectID, &item.AccessSourceID, &item.CollectorConnectionID, &item.DatapointID, &item.IsEnabled, &item.WriteMode, &item.IntervalMS, &item.Deadband, &item.MaxSilenceMS, &item.OfflineBehavior, &item.CreatedBy, &item.UpdatedBy, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanStringRows(rows pgx.Rows) ([]string, error) {
	result := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, rows.Err()
}

func translateHistoryStorageWriteError(message string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "历史存储配置引用的来源、数据点或目标不存在")
		case "23505":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "历史存储配置包含重复的作用域或目标")
		case "23514":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "历史存储配置参数无效")
		}
	}
	return wrapHistoryStorageError(message, err)
}

func wrapHistoryStorageError(message string, err error) error {
	return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, message, err)
}

func int64Pointer(value int64) *int64       { return &value }
func float64Pointer(value float64) *float64 { return &value }
