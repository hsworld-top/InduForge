package repository

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

// SourceOutputSelectorRecord 是来源输出选择器的规范化仓储表示。
// Path 始终使用字符串字段名或非负整数数组索引，避免点分字符串无法表达含点号字段。
type SourceOutputSelectorRecord struct {
	Kind     string  `json:"kind"`
	Column   *string `json:"column,omitempty"`
	Segments []any   `json:"segments"`
}

// SourceOutputMappingParam 描述一个来源对象要生成的稳定数据点映射。
type SourceOutputMappingParam struct {
	ID           *string
	Key          string
	DisplayName  string
	Selector     SourceOutputSelectorRecord
	DataType     string
	Unit         *string
	PrecisionNum *int
	SortOrder    int
	DefaultValue *string
}

// SourceOutputMappingRecord 是来源映射及其生成点的完整投影。
type SourceOutputMappingRecord struct {
	ID            string                     `json:"id"`
	DataPointID   string                     `json:"datapointId"`
	DataPointPath string                     `json:"datapointPath"`
	Key           string                     `json:"key"`
	DisplayName   string                     `json:"displayName"`
	Selector      SourceOutputSelectorRecord `json:"selector"`
	DataType      string                     `json:"dataType"`
	Unit          *string                    `json:"unit,omitempty"`
	PrecisionNum  *int                       `json:"precisionNum,omitempty"`
	SortOrder     int                        `json:"sortOrder"`
}

type sourceOutputOwner struct {
	Kind         string
	ID           string
	ProjectID    string
	SourceType   string
	SourceID     string
	PathPrefix   string
	Status       string
	Description  *string
	BaseConfig   map[string]any
	DefaultValue *string
	UserID       string
}

var sourceOutputOwnerColumns = map[string]string{
	"query":     "query_id",
	"http":      "http_request_id",
	"websocket": "websocket_session_id",
	"realtime":  "realtime_key_id",
}

func listSourceOutputMappings(ctx context.Context, queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, ownerKind, ownerID string) ([]SourceOutputMappingRecord, error) {
	column, ok := sourceOutputOwnerColumns[ownerKind]
	if !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "输出映射所属对象无效")
	}
	rows, err := queryer.Query(ctx, `
		SELECT mapping.id::text, mapping.datapoint_id::text, point.path,
		       mapping.output_key, mapping.display_name, mapping.selector_kind,
		       mapping.selector_column, mapping.selector_path, mapping.data_type,
		       mapping.unit, mapping.precision_num, mapping.sort_order
		FROM data_source_output_mappings mapping
		JOIN data_points point ON point.id = mapping.datapoint_id
		WHERE mapping.`+column+` = $1
		ORDER BY mapping.sort_order, mapping.created_at
	`, ownerID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取来源输出映射失败", err)
	}
	defer rows.Close()
	result := make([]SourceOutputMappingRecord, 0)
	for rows.Next() {
		var record SourceOutputMappingRecord
		var pathBytes []byte
		if err := rows.Scan(&record.ID, &record.DataPointID, &record.DataPointPath, &record.Key, &record.DisplayName,
			&record.Selector.Kind, &record.Selector.Column, &pathBytes, &record.DataType, &record.Unit,
			&record.PrecisionNum, &record.SortOrder); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析来源输出映射失败", err)
		}
		if err := json.Unmarshal(pathBytes, &record.Selector.Segments); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析来源输出路径失败", err)
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历来源输出映射失败", err)
	}
	return result, nil
}

// syncSourceOutputMappingsTx 在所属配置同一事务中整体同步映射和生成点。
// 删除或类型变化会先做引用检查，任何一步失败均回滚主记录、映射和数据点。
func syncSourceOutputMappingsTx(ctx context.Context, tx pgx.Tx, owner sourceOutputOwner, mappings []SourceOutputMappingParam) ([]SourceOutputMappingRecord, error) {
	column, ok := sourceOutputOwnerColumns[owner.Kind]
	if !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "输出映射所属对象无效")
	}
	existing, err := listSourceOutputMappings(ctx, tx, owner.Kind, owner.ID)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]SourceOutputMappingRecord, len(existing))
	for _, item := range existing {
		byID[item.ID] = item
	}
	kept := make(map[string]struct{}, len(mappings))
	result := make([]SourceOutputMappingRecord, 0, len(mappings))
	for index, input := range mappings {
		mappingID := ""
		var current *SourceOutputMappingRecord
		if input.ID != nil && strings.TrimSpace(*input.ID) != "" {
			mappingID = strings.TrimSpace(*input.ID)
			item, found := byID[mappingID]
			if !found {
				return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "输出映射不存在或不属于当前对象")
			}
			current = &item
		} else {
			mappingID = uuid.NewString()
		}
		if _, duplicate := kept[mappingID]; duplicate {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "输出映射 ID 重复")
		}
		kept[mappingID] = struct{}{}

		if current != nil && current.DataType != input.DataType {
			if err := ensureNoDatapointBlockingUsagesTx(ctx, tx, owner.ProjectID, []string{current.DataPointID}); err != nil {
				return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "输出点已被引用，不能直接修改数据类型")
			}
		}
		pathPayload, err := json.Marshal(input.Selector.Segments)
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "输出选择器路径无效", err)
		}
		config := cloneProtocolMap(owner.BaseConfig)
		config["mappingId"] = mappingID
		config["outputKey"] = input.Key
		config["selector"] = map[string]any{"kind": input.Selector.Kind, "column": input.Selector.Column, "segments": input.Selector.Segments}
		configPayload, err := json.Marshal(config)
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "输出点来源配置无效", err)
		}

		defaultValue := owner.DefaultValue
		if input.DefaultValue != nil {
			defaultValue = input.DefaultValue
		}
		pointID := ""
		pointPath := ""
		desiredPath := desiredSourceOutputPath(owner, input)
		currentPointID := ""
		if current != nil {
			currentPointID = current.DataPointID
		}
		if err := releaseStaleGeneratedPathTx(ctx, tx, owner, currentPointID, desiredPath); err != nil {
			return nil, err
		}
		if current != nil {
			pointID = current.DataPointID
			if current.DataPointPath == desiredPath {
				pointPath = current.DataPointPath
			} else {
				pointPath, err = allocateGeneratedDataPointPath(ctx, tx, owner.ProjectID, desiredPath, owner.SourceType, owner.ID)
				if err != nil {
					return nil, err
				}
			}
			if _, err := tx.Exec(ctx, `
				UPDATE data_points SET path=$3,name=$4,description=$5,source_type=$6,source_id=$7,
				       source_config=$8::jsonb,data_type=$9,unit=$10,precision_num=$11,
				       default_value=COALESCE($12,default_value),refresh_mode='manual',status=$13,
				       updated_by=$14,updated_at=now()
				WHERE project_id=$1 AND id=$2
			`, owner.ProjectID, pointID, pointPath, input.DisplayName, owner.Description, owner.SourceType,
				owner.SourceID, string(configPayload), input.DataType, input.Unit, input.PrecisionNum,
				defaultValue, owner.Status, owner.UserID); err != nil {
				return nil, translateDataPointWriteError(err)
			}
		} else {
			pointPath, err = allocateGeneratedDataPointPath(ctx, tx, owner.ProjectID, desiredPath, owner.SourceType, owner.ID)
			if err != nil {
				return nil, err
			}
			if err := tx.QueryRow(ctx, `
				INSERT INTO data_points(project_id,path,name,description,source_type,source_id,source_config,
				 data_type,unit,precision_num,default_value,tags,runtime_permissions,refresh_mode,status,display_order,created_by,updated_by)
				VALUES($1,$2,$3,$4,$5,$6,$7::jsonb,$8,$9,$10,$11,'[]'::jsonb,
				 '{"write":{"inherit":true,"denyRoles":[],"allowRoles":[]}}'::jsonb,'manual',$12,
				 COALESCE((SELECT MAX(display_order)+1 FROM data_points WHERE project_id=$1),0),$13,$13)
				RETURNING id::text
			`, owner.ProjectID, pointPath, input.DisplayName, owner.Description, owner.SourceType, owner.SourceID,
				string(configPayload), input.DataType, input.Unit, input.PrecisionNum, defaultValue, owner.Status, owner.UserID).Scan(&pointID); err != nil {
				return nil, translateDataPointWriteError(err)
			}
		}

		if current != nil {
			_, err = tx.Exec(ctx, `UPDATE data_source_output_mappings SET output_key=$3,display_name=$4,
				selector_kind=$5,selector_column=$6,selector_path=$7::jsonb,data_type=$8,unit=$9,
				precision_num=$10,sort_order=$11,updated_at=now() WHERE id=$1 AND `+column+`=$2`,
				mappingID, owner.ID, input.Key, input.DisplayName, input.Selector.Kind, input.Selector.Column,
				string(pathPayload), input.DataType, input.Unit, input.PrecisionNum, index)
		} else {
			_, err = tx.Exec(ctx, `INSERT INTO data_source_output_mappings(id,project_id,`+column+`,datapoint_id,
				output_key,display_name,selector_kind,selector_column,selector_path,data_type,unit,precision_num,sort_order)
				VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10,$11,$12,$13)`, mappingID, owner.ProjectID,
				owner.ID, pointID, input.Key, input.DisplayName, input.Selector.Kind, input.Selector.Column,
				string(pathPayload), input.DataType, input.Unit, input.PrecisionNum, index)
		}
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "保存来源输出映射失败", err)
		}
		result = append(result, SourceOutputMappingRecord{ID: mappingID, DataPointID: pointID, DataPointPath: pointPath,
			Key: input.Key, DisplayName: input.DisplayName, Selector: input.Selector, DataType: input.DataType,
			Unit: input.Unit, PrecisionNum: input.PrecisionNum, SortOrder: index})
	}

	removedPointIDs := make([]string, 0)
	removedMappingIDs := make([]string, 0)
	for _, item := range existing {
		if _, ok := kept[item.ID]; ok {
			continue
		}
		removedPointIDs = append(removedPointIDs, item.DataPointID)
		removedMappingIDs = append(removedMappingIDs, item.ID)
	}
	if len(removedPointIDs) > 0 {
		if err := ensureNoDatapointBlockingUsagesTx(ctx, tx, owner.ProjectID, removedPointIDs); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM data_source_output_mappings WHERE id=ANY($1::uuid[])`, removedMappingIDs); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除来源输出映射失败", err)
		}
		if _, err := tx.Exec(ctx, `UPDATE data_points SET status='invalid',updated_by=$2,updated_at=now() WHERE id=ANY($1::uuid[])`, removedPointIDs, owner.UserID); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "标记已删除输出点失效失败", err)
		}
	}
	return result, nil
}

// releaseStaleGeneratedPathTx 仅回收同一来源对象留下且无引用的失效数据点，
// 避免用户重新选择字段时路径被迫追加随机后缀。
func releaseStaleGeneratedPathTx(ctx context.Context, tx pgx.Tx, owner sourceOutputOwner, currentPointID, desiredPath string) error {
	if owner.Kind != "query" {
		return nil
	}
	var pointID string
	var sourceType string
	var sourceID string
	var status string
	err := tx.QueryRow(ctx, `
		SELECT id::text,source_type,COALESCE(source_id::text,''),status
		FROM data_points
		WHERE project_id=$1 AND path=$2
		FOR UPDATE
	`, owner.ProjectID, desiredPath).Scan(&pointID, &sourceType, &sourceID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "检查生成路径占用失败", err)
	}
	if pointID == currentPointID || sourceType != owner.SourceType || sourceID != owner.SourceID || status != "invalid" {
		return nil
	}
	if err := ensureNoDatapointBlockingUsagesTx(ctx, tx, owner.ProjectID, []string{pointID}); err != nil {
		return err
	}
	command, err := tx.Exec(ctx, `
		DELETE FROM data_points point
		WHERE point.project_id=$1 AND point.id=$2
		  AND NOT EXISTS (
			SELECT 1 FROM data_source_output_mappings mapping WHERE mapping.datapoint_id=point.id
		  )
	`, owner.ProjectID, pointID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "回收失效生成路径失败", err)
	}
	if command.RowsAffected() != 1 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "目标数据点路径仍被其他输出占用")
	}
	return nil
}

// 查询完整结果和实时 Key 整体值直接占用来源路径；只有字段提取才追加输出 key。
// 其他来源继续保留原有的“来源路径.输出 key”规则。
func desiredSourceOutputPath(owner sourceOutputOwner, input SourceOutputMappingParam) string {
	prefix := strings.TrimSuffix(strings.TrimSpace(owner.PathPrefix), ".")
	if (owner.Kind == "query" || owner.Kind == "realtime") && input.Selector.Kind == "whole" {
		return prefix
	}
	return prefix + "." + normalizeGeneratedPathSegment(input.Key)
}

func normalizeGeneratedPathSegment(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "output"
	}
	var builder strings.Builder
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_' || char == '-' {
			builder.WriteRune(char)
		} else {
			builder.WriteRune('_')
		}
	}
	result := strings.Trim(builder.String(), "_-")
	if result == "" {
		return "output"
	}
	return result
}

func sourceMappingNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func sourceOutputPointIDsForUpdate(ctx context.Context, tx pgx.Tx, ownerKind, ownerID string) ([]string, error) {
	column, ok := sourceOutputOwnerColumns[ownerKind]
	if !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "输出映射所属对象无效")
	}
	rows, err := tx.Query(ctx, `SELECT datapoint_id::text FROM data_source_output_mappings WHERE `+column+`=$1 FOR UPDATE`, ownerID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "锁定来源输出点失败", err)
	}
	defer rows.Close()
	result := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, rows.Err()
}
