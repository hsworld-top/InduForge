package service

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

// SourceOutputSelector 是 SQL、HTTP、WebSocket 与实时 Key 共用的强类型选择器。
type SourceOutputSelector struct {
	Kind     string `json:"kind"`
	Column   string `json:"column,omitempty"`
	Segments []any  `json:"segments,omitempty"`
}

// SourceOutputInput 描述用户保存的一项输出映射。
type SourceOutputInput struct {
	ID           *string              `json:"id,omitempty"`
	Key          string               `json:"key"`
	DisplayName  string               `json:"displayName"`
	Selector     SourceOutputSelector `json:"selector"`
	DataType     string               `json:"dataType"`
	Unit         *string              `json:"unit,omitempty"`
	PrecisionNum *int                 `json:"precisionNum,omitempty"`
	SortOrder    int                  `json:"sortOrder"`
}

// SourceOutput 是包含稳定映射与数据点身份的读取模型。
type SourceOutput struct {
	ID            string               `json:"id"`
	DataPointID   string               `json:"datapointId"`
	DataPointPath string               `json:"datapointPath"`
	Key           string               `json:"key"`
	DisplayName   string               `json:"displayName"`
	Selector      SourceOutputSelector `json:"selector"`
	DataType      string               `json:"dataType"`
	Unit          *string              `json:"unit,omitempty"`
	PrecisionNum  *int                 `json:"precisionNum,omitempty"`
	SortOrder     int                  `json:"sortOrder"`
}

func normalizeSourceOutputs(inputs []SourceOutputInput, defaultOutput SourceOutputInput) ([]repository.SourceOutputMappingParam, error) {
	if len(inputs) == 0 {
		inputs = []SourceOutputInput{defaultOutput}
	}
	if len(inputs) == 0 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "至少需要配置一个输出")
	}
	result := make([]repository.SourceOutputMappingParam, 0, len(inputs))
	keys := make(map[string]struct{}, len(inputs))
	ids := make(map[string]struct{}, len(inputs))
	for index, input := range inputs {
		key := strings.TrimSpace(input.Key)
		if key == "" || len([]rune(key)) > 100 {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "输出 key 长度必须为 1 到 100 个字符")
		}
		keyKey := strings.ToLower(key)
		if _, ok := keys[keyKey]; ok {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "输出 key 不能重复")
		}
		keys[keyKey] = struct{}{}
		displayName := strings.TrimSpace(input.DisplayName)
		if displayName == "" {
			displayName = key
		}
		if len([]rune(displayName)) > 100 {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "输出显示名称不能超过 100 个字符")
		}
		dataType, ok := canonicalDataPointType(input.DataType)
		if !ok {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "输出数据类型不受支持")
		}
		selector, err := normalizeSourceOutputSelector(input.Selector)
		if err != nil {
			return nil, err
		}
		if input.PrecisionNum != nil && *input.PrecisionNum < 0 {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "输出精度不能为负数")
		}
		var id *string
		if input.ID != nil && strings.TrimSpace(*input.ID) != "" {
			normalized := strings.TrimSpace(*input.ID)
			if _, err := uuid.Parse(normalized); err != nil {
				return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "输出映射 ID 格式无效")
			}
			if _, duplicate := ids[normalized]; duplicate {
				return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "输出映射 ID 不能重复")
			}
			ids[normalized] = struct{}{}
			id = &normalized
		}
		unit := cloneOptionalString(input.Unit)
		if unit != nil && strings.TrimSpace(*unit) == "" {
			unit = nil
		}
		result = append(result, repository.SourceOutputMappingParam{ID: id, Key: key, DisplayName: displayName,
			Selector: selector, DataType: dataType, Unit: unit, PrecisionNum: input.PrecisionNum, SortOrder: index})
	}
	return result, nil
}

func normalizeSourceOutputSelector(input SourceOutputSelector) (repository.SourceOutputSelectorRecord, error) {
	kind := strings.ToLower(strings.TrimSpace(input.Kind))
	result := repository.SourceOutputSelectorRecord{Kind: kind, Segments: []any{}}
	switch kind {
	case "whole":
		return result, nil
	case "column":
		column := strings.TrimSpace(input.Column)
		if column == "" {
			return result, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "字段输出必须选择列")
		}
		result.Column = &column
		return result, nil
	case "path":
		segments, err := normalizeValuePathSegments(input.Segments, false)
		if err != nil || len(segments) == 0 {
			return result, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "路径输出必须选择有效字段路径")
		}
		result.Segments = segments
		return result, nil
	default:
		return result, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "输出选择器仅支持 whole、column 或 path")
	}
}

func sourceOutputsFromRecords(records []repository.SourceOutputMappingRecord) []SourceOutput {
	result := make([]SourceOutput, 0, len(records))
	for _, record := range records {
		column := ""
		if record.Selector.Column != nil {
			column = *record.Selector.Column
		}
		result = append(result, SourceOutput{ID: record.ID, DataPointID: record.DataPointID, DataPointPath: record.DataPointPath,
			Key: record.Key, DisplayName: record.DisplayName, Selector: SourceOutputSelector{Kind: record.Selector.Kind,
				Column: column, Segments: record.Selector.Segments}, DataType: record.DataType, Unit: record.Unit,
			PrecisionNum: record.PrecisionNum, SortOrder: record.SortOrder})
	}
	return result
}
