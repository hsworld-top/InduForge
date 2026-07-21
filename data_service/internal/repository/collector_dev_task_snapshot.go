package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/jackc/pgx/v5"
)

type collectorPointReadResultPayload struct {
	Values []collectorPointReadResultValue `json:"values"`
}

type collectorPointReadResultValue struct {
	PointID         string          `json:"pointId"`
	Succeeded       bool            `json:"succeeded"`
	Value           json.RawMessage `json:"value"`
	DataType        *string         `json:"dataType"`
	Quality         *string         `json:"quality"`
	SourceTimestamp *time.Time      `json:"sourceTimestamp"`
	ServerTimestamp *time.Time      `json:"serverTimestamp"`
	ErrorCode       *string         `json:"errorCode"`
	ErrorMessage    *string         `json:"errorMessage"`
}

type collectorPointSnapshotAttempt struct {
	PointID         string          `json:"pointId"`
	Succeeded       bool            `json:"succeeded"`
	Value           json.RawMessage `json:"value"`
	ValueText       *string         `json:"valueText"`
	DataType        *string         `json:"dataType"`
	Quality         *string         `json:"quality"`
	SourceTimestamp *time.Time      `json:"sourceTimestamp"`
	ServerTimestamp *time.Time      `json:"serverTimestamp"`
	ErrorCode       *string         `json:"errorCode"`
	ErrorMessage    *string         `json:"errorMessage"`
}

func buildCollectorPointSnapshotAttempts(
	task CollectorDevTaskRecord,
	status string,
	resultPayload []byte,
	errorCode string,
	errorMessage string,
) ([]collectorPointSnapshotAttempt, error) {
	var request struct {
		PointIDs []string `json:"pointIds"`
	}
	if err := json.Unmarshal(task.RequestPayload, &request); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "解析变量读取任务请求失败", err)
	}
	requested := make(map[string]struct{}, len(request.PointIDs))
	for index, pointID := range request.PointIDs {
		pointID = strings.TrimSpace(pointID)
		request.PointIDs[index] = pointID
		if _, err := uuid.Parse(pointID); err != nil {
			return nil, badCollectorTaskCompletion("变量读取任务包含无效 pointId")
		}
		if _, exists := requested[pointID]; exists {
			return nil, badCollectorTaskCompletion("变量读取任务包含重复 pointId")
		}
		requested[pointID] = struct{}{}
	}
	if len(requested) == 0 {
		return nil, badCollectorTaskCompletion("变量读取任务缺少 pointIds")
	}

	if status == "failed" {
		code := optionalCollectorText(errorCode, "COLLECTOR_TASK_FAILED")
		message := optionalCollectorText(errorMessage, "调试任务执行失败")
		attempts := make([]collectorPointSnapshotAttempt, 0, len(request.PointIDs))
		for _, pointID := range request.PointIDs {
			attempts = append(attempts, collectorPointSnapshotAttempt{
				PointID: pointID, Succeeded: false, ErrorCode: &code, ErrorMessage: &message,
			})
		}
		return attempts, nil
	}

	var result collectorPointReadResultPayload
	if err := json.Unmarshal(resultPayload, &result); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "解析变量读取任务结果失败", err)
	}
	byPointID := make(map[string]collectorPointReadResultValue, len(result.Values))
	for _, value := range result.Values {
		value.PointID = strings.TrimSpace(value.PointID)
		if _, exists := requested[value.PointID]; !exists {
			return nil, badCollectorTaskCompletion("变量读取结果包含未请求的 pointId")
		}
		if _, exists := byPointID[value.PointID]; exists {
			return nil, badCollectorTaskCompletion("变量读取结果包含重复 pointId")
		}
		byPointID[value.PointID] = value
	}

	attempts := make([]collectorPointSnapshotAttempt, 0, len(request.PointIDs))
	for _, pointID := range request.PointIDs {
		value, exists := byPointID[pointID]
		if !exists {
			code := "COLLECTOR_POINT_RESULT_MISSING"
			message := "Agent 未返回该变量的读取结果"
			attempts = append(attempts, collectorPointSnapshotAttempt{
				PointID: pointID, Succeeded: false, ErrorCode: &code, ErrorMessage: &message,
			})
			continue
		}
		if !value.Succeeded {
			code := optionalCollectorTextPointer(value.ErrorCode, "COLLECTOR_POINT_READ_FAILED")
			message := optionalCollectorTextPointer(value.ErrorMessage, "变量读取失败")
			attempts = append(attempts, collectorPointSnapshotAttempt{
				PointID: pointID, Succeeded: false, ErrorCode: &code, ErrorMessage: &message,
			})
			continue
		}
		valueText := formatCollectorDebugValue(value.Value)
		attempts = append(attempts, collectorPointSnapshotAttempt{
			PointID:         pointID,
			Succeeded:       true,
			Value:           value.Value,
			ValueText:       &valueText,
			DataType:        trimCollectorTextPointer(value.DataType),
			Quality:         trimCollectorTextPointer(value.Quality),
			SourceTimestamp: normalizeCollectorDebugTimestamp(value.SourceTimestamp),
			ServerTimestamp: normalizeCollectorDebugTimestamp(value.ServerTimestamp),
		})
	}
	return attempts, nil
}

func upsertCollectorPointDebugSnapshots(
	ctx context.Context,
	tx pgx.Tx,
	projectID string,
	connectionID string,
	attemptedAt time.Time,
	attempts []collectorPointSnapshotAttempt,
) error {
	if len(attempts) == 0 {
		return nil
	}
	payload, err := json.Marshal(attempts)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "序列化变量调试快照失败", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO data_collector_point_debug_snapshots (
			point_id, value, value_text, data_type, quality,
			source_timestamp, server_timestamp, read_at,
			last_attempt_status, last_attempt_at, last_error_code, last_error_message
		)
		SELECT input."pointId",
			CASE WHEN input.succeeded THEN input.value ELSE NULL END,
			CASE WHEN input.succeeded THEN input."valueText" ELSE NULL END,
			CASE WHEN input.succeeded THEN input."dataType" ELSE NULL END,
			CASE WHEN input.succeeded THEN input.quality ELSE NULL END,
			CASE WHEN input.succeeded THEN input."sourceTimestamp" ELSE NULL END,
			CASE WHEN input.succeeded THEN input."serverTimestamp" ELSE NULL END,
			CASE WHEN input.succeeded THEN $4::timestamptz ELSE NULL END,
			CASE WHEN input.succeeded THEN 'succeeded' ELSE 'failed' END,
			$4::timestamptz,
			CASE WHEN input.succeeded THEN NULL ELSE input."errorCode" END,
			CASE WHEN input.succeeded THEN NULL ELSE input."errorMessage" END
		FROM jsonb_to_recordset($1::jsonb) AS input(
			"pointId" uuid,
			succeeded boolean,
			value jsonb,
			"valueText" text,
			"dataType" text,
			quality text,
			"sourceTimestamp" timestamptz,
			"serverTimestamp" timestamptz,
			"errorCode" text,
			"errorMessage" text
		)
		JOIN data_collector_points point
			ON point.id = input."pointId" AND point.project_id = $2 AND point.connection_id = $3
		ON CONFLICT (point_id) DO UPDATE SET
			value = CASE WHEN EXCLUDED.last_attempt_status = 'succeeded' THEN EXCLUDED.value ELSE data_collector_point_debug_snapshots.value END,
			value_text = CASE WHEN EXCLUDED.last_attempt_status = 'succeeded' THEN EXCLUDED.value_text ELSE data_collector_point_debug_snapshots.value_text END,
			data_type = CASE WHEN EXCLUDED.last_attempt_status = 'succeeded' THEN EXCLUDED.data_type ELSE data_collector_point_debug_snapshots.data_type END,
			quality = CASE WHEN EXCLUDED.last_attempt_status = 'succeeded' THEN EXCLUDED.quality ELSE data_collector_point_debug_snapshots.quality END,
			source_timestamp = CASE WHEN EXCLUDED.last_attempt_status = 'succeeded' THEN EXCLUDED.source_timestamp ELSE data_collector_point_debug_snapshots.source_timestamp END,
			server_timestamp = CASE WHEN EXCLUDED.last_attempt_status = 'succeeded' THEN EXCLUDED.server_timestamp ELSE data_collector_point_debug_snapshots.server_timestamp END,
			read_at = CASE WHEN EXCLUDED.last_attempt_status = 'succeeded' THEN EXCLUDED.read_at ELSE data_collector_point_debug_snapshots.read_at END,
			last_attempt_status = EXCLUDED.last_attempt_status,
			last_attempt_at = EXCLUDED.last_attempt_at,
			last_error_code = EXCLUDED.last_error_code,
			last_error_message = EXCLUDED.last_error_message
	`, string(payload), projectID, connectionID, attemptedAt)
	if err != nil {
		return wrapCollectorRepositoryError("保存变量调试快照失败", err)
	}
	return nil
}

func normalizeCollectorDebugTimestamp(value *time.Time) *time.Time {
	if value == nil || value.IsZero() || value.Year() <= 1 {
		return nil
	}
	normalized := value.UTC()
	return &normalized
}

func formatCollectorDebugValue(value json.RawMessage) string {
	if len(value) == 0 || string(value) == "null" {
		return "null"
	}
	var text string
	if err := json.Unmarshal(value, &text); err == nil {
		return text
	}
	return string(value)
}

func optionalCollectorText(value string, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}

func optionalCollectorTextPointer(value *string, fallback string) string {
	if value != nil {
		return optionalCollectorText(*value, fallback)
	}
	return fallback
}

func trimCollectorTextPointer(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func badCollectorTaskCompletion(message string) error {
	return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, message)
}
