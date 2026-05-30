package repository

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

// RealtimeKeyRecord 是统一实时库工作台保存的 key 元数据。
// Redis 的真实值仍在外部 Redis 中，IF 实时库的真实值在开发态 Redis 命名空间中。
type RealtimeKeyRecord struct {
	ID                string
	ProjectID         string
	ConnectionID      string
	Provider          string
	KeyPath           string
	RedisType         string
	ValueType         string
	DefaultTtlSeconds int
	Description       string
	DataPointID       *string
	DataPointPath     *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type UpsertRealtimeKeyParams struct {
	ProjectID         string
	ConnectionID      string
	Provider          string
	KeyPath           string
	RedisType         string
	ValueType         string
	DefaultTtlSeconds int
	Description       string
}

type CreateRealtimeKeyDataPointParams struct {
	ProjectID         string
	ConnectionID      string
	KeyID             string
	KeyPath           string
	Provider          string
	RedisType         string
	DataType          string
	SourceConfig      map[string]any
	UserID            *string
}

type RealtimeStoreRepository struct {
	pool *pgxpool.Pool
}

func NewRealtimeStoreRepository(pool *pgxpool.Pool) *RealtimeStoreRepository {
	return &RealtimeStoreRepository{pool: pool}
}

func (r *RealtimeStoreRepository) List(ctx context.Context, projectID, connectionID, provider string) ([]RealtimeKeyRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT k.id::text, k.project_id::text, k.connection_id::text, k.provider, k.key_path,
		       k.redis_type, k.value_type, k.default_ttl_seconds, k.description,
		       dp.id::text, dp.path, k.created_at, k.updated_at
		FROM data_realtime_keys k
		LEFT JOIN data_points dp
		  ON dp.project_id = k.project_id
		 AND dp.source_type = 'realtime.key'
		 AND dp.source_config->>'keyId' = k.id::text
		 AND dp.status <> 'invalid'
		WHERE k.project_id = $1 AND k.connection_id = $2 AND k.provider = $3
		ORDER BY k.key_path ASC
	`, projectID, connectionID, provider)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询实时库 key 元数据失败", err)
	}
	defer rows.Close()

	items := make([]RealtimeKeyRecord, 0)
	for rows.Next() {
		record := RealtimeKeyRecord{}
		if err := rows.Scan(
			&record.ID,
			&record.ProjectID,
			&record.ConnectionID,
			&record.Provider,
			&record.KeyPath,
			&record.RedisType,
			&record.ValueType,
			&record.DefaultTtlSeconds,
			&record.Description,
			&record.DataPointID,
			&record.DataPointPath,
			&record.CreatedAt,
			&record.UpdatedAt,
		); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取实时库 key 元数据失败", err)
		}
		items = append(items, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历实时库 key 元数据失败", err)
	}
	return items, nil
}

func (r *RealtimeStoreRepository) GetByKey(ctx context.Context, projectID, connectionID, provider, keyPath string) (*RealtimeKeyRecord, error) {
	record := RealtimeKeyRecord{}
	err := r.pool.QueryRow(ctx, `
		SELECT k.id::text, k.project_id::text, k.connection_id::text, k.provider, k.key_path,
		       k.redis_type, k.value_type, k.default_ttl_seconds, k.description,
		       dp.id::text, dp.path, k.created_at, k.updated_at
		FROM data_realtime_keys k
		LEFT JOIN data_points dp
		  ON dp.project_id = k.project_id
		 AND dp.source_type = 'realtime.key'
		 AND dp.source_config->>'keyId' = k.id::text
		 AND dp.status <> 'invalid'
		WHERE k.project_id = $1 AND k.connection_id = $2 AND k.provider = $3 AND k.key_path = $4
	`, projectID, connectionID, provider, keyPath).Scan(
		&record.ID,
		&record.ProjectID,
		&record.ConnectionID,
		&record.Provider,
		&record.KeyPath,
		&record.RedisType,
		&record.ValueType,
		&record.DefaultTtlSeconds,
		&record.Description,
		&record.DataPointID,
		&record.DataPointPath,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "实时库 key 不存在")
		}
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取实时库 key 元数据失败", err)
	}
	return &record, nil
}

func (r *RealtimeStoreRepository) Upsert(ctx context.Context, params UpsertRealtimeKeyParams) (*RealtimeKeyRecord, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO data_realtime_keys (
			project_id, connection_id, provider, key_path, redis_type,
			value_type, default_ttl_seconds, description
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (project_id, connection_id, key_path)
		DO UPDATE SET provider = EXCLUDED.provider,
		              redis_type = EXCLUDED.redis_type,
		              value_type = EXCLUDED.value_type,
		              default_ttl_seconds = EXCLUDED.default_ttl_seconds,
		              description = EXCLUDED.description,
		              updated_at = now()
		RETURNING id::text, project_id::text, connection_id::text, provider, key_path,
		          redis_type, value_type, default_ttl_seconds, description, created_at, updated_at
	`, params.ProjectID, params.ConnectionID, params.Provider, params.KeyPath, params.RedisType, params.ValueType, params.DefaultTtlSeconds, params.Description)

	record := RealtimeKeyRecord{}
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.ConnectionID,
		&record.Provider,
		&record.KeyPath,
		&record.RedisType,
		&record.ValueType,
		&record.DefaultTtlSeconds,
		&record.Description,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return nil, translateRealtimeStoreWriteError(err, "保存实时库 key 元数据失败")
	}
	return &record, nil
}

func (r *RealtimeStoreRepository) Rename(ctx context.Context, projectID, connectionID, provider, oldKey, newKey string) (*RealtimeKeyRecord, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启实时库 key 重命名事务失败", err)
	}
	defer rollbackProtocolTxQuietly(ctx, tx)

	row := tx.QueryRow(ctx, `
		UPDATE data_realtime_keys
		SET key_path = $5,
		    updated_at = now()
		WHERE project_id = $1 AND connection_id = $2 AND provider = $3 AND key_path = $4
		RETURNING id::text, project_id::text, connection_id::text, provider, key_path,
		          redis_type, value_type, default_ttl_seconds, description, created_at, updated_at
	`, projectID, connectionID, provider, oldKey, newKey)
	record := RealtimeKeyRecord{}
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.ConnectionID,
		&record.Provider,
		&record.KeyPath,
		&record.RedisType,
		&record.ValueType,
		&record.DefaultTtlSeconds,
		&record.Description,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "实时库 key 元数据不存在")
		}
		return nil, translateRealtimeStoreWriteError(err, "重命名实时库 key 元数据失败")
	}
	if _, err := tx.Exec(ctx, `
		UPDATE data_points
		SET name = $3,
		    source_config = jsonb_set(source_config, '{key}', to_jsonb($3::text), true),
		    updated_at = now()
		WHERE project_id = $1
		  AND source_type = 'realtime.key'
		  AND source_config->>'keyId' = $2
	`, projectID, record.ID, newKey); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "同步实时库 key 数据点配置失败", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交实时库 key 重命名事务失败", err)
	}
	return &record, nil
}

func (r *RealtimeStoreRepository) Delete(ctx context.Context, projectID, connectionID, provider, keyPath string, userID *string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启实时库 key 删除事务失败", err)
	}
	defer rollbackProtocolTxQuietly(ctx, tx)

	var keyID string
	err = tx.QueryRow(ctx, `
		SELECT id::text
		FROM data_realtime_keys
		WHERE project_id = $1 AND connection_id = $2 AND provider = $3 AND key_path = $4
	`, projectID, connectionID, provider, keyPath).Scan(&keyID)
	if err != nil && err != pgx.ErrNoRows {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取实时库 key 元数据失败", err)
	}
	if keyID != "" {
		if _, err := tx.Exec(ctx, `
			UPDATE data_points
			SET status = 'invalid',
			    updated_by = COALESCE($3, updated_by),
			    updated_at = now()
			WHERE project_id = $1
			  AND source_type = 'realtime.key'
			  AND source_config->>'keyId' = $2
		`, projectID, keyID, userID); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "标记实时库 key 数据点失效失败", err)
		}
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM data_realtime_keys
		WHERE project_id = $1 AND connection_id = $2 AND provider = $3 AND key_path = $4
	`, projectID, connectionID, provider, keyPath); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除实时库 key 元数据失败", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交实时库 key 删除事务失败", err)
	}
	return nil
}

func (r *RealtimeStoreRepository) CreateDataPoint(ctx context.Context, params CreateRealtimeKeyDataPointParams) (*DataPointRecord, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启实时库 key 数据点事务失败", err)
	}
	defer rollbackProtocolTxQuietly(ctx, tx)

	configBytes, err := json.Marshal(params.SourceConfig)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "实时库 key 数据点配置无效", err)
	}
	basePath := "realtime." + safeDataPointPathSegment(params.KeyPath)
	if params.Provider == "redis" {
		basePath = "redis." + safeDataPointPathSegment(params.KeyPath)
	}

	record := DataPointRecord{}
	err = tx.QueryRow(ctx, `
		UPDATE data_points
		SET path = COALESCE(NULLIF(path, ''), $4),
		    name = $5,
		    source_id = $6,
		    source_config = $7::jsonb,
		    data_type = $8,
		    refresh_mode = 'manual',
		    status = 'active',
		    updated_by = COALESCE($9, updated_by),
		    updated_at = now()
		WHERE project_id = $1
		  AND source_type = 'realtime.key'
		  AND source_config->>'keyId' = $2
		RETURNING id, project_id, path, name, description, source_type, source_id, source_config, data_type,
		          unit, precision_num, default_value, min_value, max_value, alarm_low, alarm_high, tags, runtime_permissions,
		          refresh_mode, refresh_interval_ms, status, created_by, updated_by, created_at, updated_at
	`, params.ProjectID, params.KeyID, params.ConnectionID, basePath, params.KeyPath, params.ConnectionID, string(configBytes), params.DataType, params.UserID).Scan(
		&record.ID,
		&record.ProjectID,
		&record.Path,
		&record.Name,
		&record.Description,
		&record.SourceType,
		&record.SourceID,
		newJSONScanner(&record.SourceConfig),
		&record.DataType,
		&record.Unit,
		&record.PrecisionNum,
		&record.DefaultValue,
		&record.MinValue,
		&record.MaxValue,
		&record.AlarmLow,
		&record.AlarmHigh,
		newJSONScanner(&record.Tags),
		newJSONScanner(&record.RuntimePermissions),
		&record.RefreshMode,
		&record.RefreshIntervalMS,
		&record.Status,
		&record.CreatedBy,
		&record.UpdatedBy,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交实时库 key 数据点事务失败", err)
		}
		return &record, nil
	}
	if err != pgx.ErrNoRows {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "同步实时库 key 数据点失败", err)
	}

	allocatedPath, err := allocateGeneratedDataPointPath(ctx, tx, params.ProjectID, basePath, "realtime.key", params.KeyID)
	if err != nil {
		return nil, err
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO data_points (
			project_id, path, name, source_type, source_id, source_config,
			data_type, refresh_mode, status, created_by, updated_by
		)
		VALUES ($1, $2, $3, 'realtime.key', $4, $5::jsonb, $6, 'manual', 'active', $7, $7)
		RETURNING id, project_id, path, name, description, source_type, source_id, source_config, data_type,
		          unit, precision_num, default_value, min_value, max_value, alarm_low, alarm_high, tags, runtime_permissions,
		          refresh_mode, refresh_interval_ms, status, created_by, updated_by, created_at, updated_at
	`, params.ProjectID, allocatedPath, params.KeyPath, params.ConnectionID, string(configBytes), params.DataType, params.UserID).Scan(
		&record.ID,
		&record.ProjectID,
		&record.Path,
		&record.Name,
		&record.Description,
		&record.SourceType,
		&record.SourceID,
		newJSONScanner(&record.SourceConfig),
		&record.DataType,
		&record.Unit,
		&record.PrecisionNum,
		&record.DefaultValue,
		&record.MinValue,
		&record.MaxValue,
		&record.AlarmLow,
		&record.AlarmHigh,
		newJSONScanner(&record.Tags),
		newJSONScanner(&record.RuntimePermissions),
		&record.RefreshMode,
		&record.RefreshIntervalMS,
		&record.Status,
		&record.CreatedBy,
		&record.UpdatedBy,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "创建实时库 key 数据点失败", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交实时库 key 数据点事务失败", err)
	}
	return &record, nil
}

func safeDataPointPathSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unnamed"
	}
	replacer := strings.NewReplacer(":", ".", "/", ".", "\\", ".", " ", "_")
	value = replacer.Replace(value)
	value = strings.Trim(value, ".")
	if value == "" {
		return "unnamed"
	}
	return value
}

func translateRealtimeStoreWriteError(err error, fallback string) error {
	var pgErr *pgconn.PgError
	if ok := strings.TrimSpace(fallback); ok == "" {
		fallback = "实时库 key 写入失败"
	}
	if strings.Contains(err.Error(), "JSON 字段类型无效") {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, fallback, err)
	}
	if ok := strings.TrimSpace(fallback); ok == "" {
		fallback = "实时库 key 写入失败"
	}
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "同名实时库 key 已存在")
		case "23503":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "关联的实时库接入源不存在")
		}
	}
	return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, fallback, err)
}
