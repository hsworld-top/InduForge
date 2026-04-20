package repository

import (
	"context"
	"net/http"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

// CreateDataPointParams 表示新建数据点时的仓储参数。
type CreateDataPointParams struct {
	ProjectID         string
	UserID            *string
	Path              string
	Name              string
	Description       *string
	SourceType        string
	SourceID          *string
	SourceConfig      map[string]any
	DataType          string
	Unit              *string
	PrecisionNum      *int
	DefaultValue      *string
	MinValue          *float64
	MaxValue          *float64
	AlarmLow          *float64
	AlarmHigh         *float64
	Tags              []any
	RefreshMode       string
	RefreshIntervalMS *int
	Status            string
}

// GetByProjectAndSource 按来源读取单个数据点。
func (r *DataPointRepository) GetByProjectAndSource(ctx context.Context, projectID, sourceType, sourceID string) (*DataPointRecord, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, project_id, path, name, description, source_type, source_id, source_config, data_type,
               unit, precision_num, default_value, min_value, max_value, alarm_low, alarm_high, tags,
               refresh_mode, refresh_interval_ms, status, created_by, updated_by, created_at, updated_at
        FROM data_points
        WHERE project_id = $1 AND source_type = $2 AND source_id = $3
        ORDER BY created_at ASC
        LIMIT 1
    `, projectID, sourceType, sourceID)

	record, err := scanDataPointRecord(row)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// ListByProjectAndPaths 按项目与路径批量读取数据点。
func (r *DataPointRepository) ListByProjectAndPaths(ctx context.Context, projectID string, paths []string) ([]DataPointRecord, error) {
	if len(paths) == 0 {
		return []DataPointRecord{}, nil
	}

	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, path, name, description, source_type, source_id, source_config, data_type,
               unit, precision_num, default_value, min_value, max_value, alarm_low, alarm_high, tags,
               refresh_mode, refresh_interval_ms, status, created_by, updated_by, created_at, updated_at
        FROM data_points
        WHERE project_id = $1 AND path = ANY($2::text[])
        ORDER BY path ASC
    `, projectID, paths)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "按路径批量读取数据点失败", err)
	}
	defer rows.Close()

	records := make([]DataPointRecord, 0, len(paths))
	for rows.Next() {
		record, scanErr := scanDataPointRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历路径数据点结果失败", err)
	}

	return records, nil
}

// Create 新建单个数据点。
func (r *DataPointRepository) Create(ctx context.Context, params CreateDataPointParams) (*DataPointRecord, error) {
	sourceConfigBytes, err := marshalJSONObject(params.SourceConfig)
	if err != nil {
		return nil, err
	}
	tagsBytes, err := marshalJSONArray(params.Tags)
	if err != nil {
		return nil, err
	}

	row := r.pool.QueryRow(ctx, `
        INSERT INTO data_points (
            project_id, path, name, description, source_type, source_id, source_config, data_type,
            unit, precision_num, default_value, min_value, max_value, alarm_low, alarm_high, tags,
            refresh_mode, refresh_interval_ms, status, created_by, updated_by
        )
        VALUES (
            $1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10, $11, $12, $13, $14, $15,
            $16::jsonb, $17, $18, $19, $20, $20
        )
        RETURNING id, project_id, path, name, description, source_type, source_id, source_config, data_type,
                  unit, precision_num, default_value, min_value, max_value, alarm_low, alarm_high, tags,
                  refresh_mode, refresh_interval_ms, status, created_by, updated_by, created_at, updated_at
    `, params.ProjectID, params.Path, params.Name, params.Description, params.SourceType, params.SourceID, string(sourceConfigBytes), params.DataType, params.Unit, params.PrecisionNum, params.DefaultValue, params.MinValue, params.MaxValue, params.AlarmLow, params.AlarmHigh, string(tagsBytes), params.RefreshMode, params.RefreshIntervalMS, params.Status, params.UserID)

	record, err := scanDataPointRecord(row)
	if err != nil {
		return nil, translateDataPointWriteError(err)
	}

	return &record, nil
}

// MarkInvalidBySource 按来源标记数据点失效。
func (r *DataPointRepository) MarkInvalidBySource(ctx context.Context, projectID, sourceType, sourceID string, userID *string) (int64, error) {
	commandTag, err := r.pool.Exec(ctx, `
        UPDATE data_points
        SET status = 'invalid',
            updated_by = COALESCE($4, updated_by),
            updated_at = now()
        WHERE project_id = $1
          AND source_type = $2
          AND source_id = $3
    `, projectID, sourceType, sourceID, userID)
	if err != nil {
		return 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "标记数据点失效失败", err)
	}
	return commandTag.RowsAffected(), nil
}
