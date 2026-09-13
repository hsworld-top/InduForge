package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

// allocateGeneratedDataPointPath 为工作台自动生成的数据点分配不冲突的 path。
// 输入是项目、基础 path、来源类型和来源对象 ID；输出是可插入 data_points 的最终 path。
// 如果基础 path 已经属于同一个来源对象，直接复用；如果被其他对象占用，则追加对象 ID 前缀避免覆盖。
func allocateGeneratedDataPointPath(ctx context.Context, tx pgx.Tx, projectID, basePath, sourceType, sourceObjectID string) (string, error) {
	basePath = normalizeGeneratedDataPointPath(basePath)
	if basePath == "" {
		basePath = "generated.unnamed"
	}
	idSegment := sourceObjectID
	if len(idSegment) > 8 {
		idSegment = idSegment[:8]
	}
	if strings.TrimSpace(idSegment) == "" {
		idSegment = "dup"
	}

	candidates := []string{basePath, basePath + "_" + idSegment}
	for index := 2; index < 100; index++ {
		candidates = append(candidates, fmt.Sprintf("%s_%s_%d", basePath, idSegment, index))
	}

	sourceKey := generatedDataPointSourceConfigKey(sourceType)
	for _, candidate := range candidates {
		recordSourceType, recordSourceConfig, err := readDataPointPathOwner(ctx, tx, projectID, candidate)
		if err != nil {
			return "", err
		}
		if recordSourceType == "" {
			return candidate, nil
		}
		if recordSourceType == sourceType && sourceKey != "" && strings.TrimSpace(toJSONText(recordSourceConfig[sourceKey])) == sourceObjectID {
			return candidate, nil
		}
	}
	return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "自动生成数据点路径冲突过多")
}

func readDataPointPathOwner(ctx context.Context, tx pgx.Tx, projectID, path string) (string, map[string]any, error) {
	var sourceType string
	var sourceConfigPayload []byte
	err := tx.QueryRow(ctx, `
		SELECT source_type, source_config
		FROM data_points
		WHERE project_id = $1 AND path = $2
	`, projectID, path).Scan(&sourceType, &sourceConfigPayload)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil, nil
		}
		return "", nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "检查数据点路径占用失败", err)
	}
	sourceConfig := map[string]any{}
	if len(sourceConfigPayload) > 0 {
		if err := json.Unmarshal(sourceConfigPayload, &sourceConfig); err != nil {
			return "", nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析数据点路径占用信息失败", err)
		}
	}
	return sourceType, sourceConfig, nil
}

func generatedDataPointSourceConfigKey(sourceType string) string {
	switch sourceType {
	case "http.request":
		return "requestId"
	case "websocket.session":
		return "sessionId"
	case "kafka.field":
		return "fieldId"
	case "kafka.raw":
		return "topicMappingId"
	case "realtime.key":
		return "keyId"
	case "mqtt.tag":
		return "tagId"
	default:
		return ""
	}
}

// normalizeGeneratedDataPointPath 统一工作台生成路径的分隔符与空白，避免同一逻辑路径因输入格式不同产生重复点。
func normalizeGeneratedDataPointPath(path string) string {
	parts := strings.FieldsFunc(strings.TrimSpace(path), func(r rune) bool { return r == '/' || r == '\\' })
	return strings.Join(parts, ".")
}

func toJSONText(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	default:
		return fmt.Sprintf("%v", typed)
	}
}
