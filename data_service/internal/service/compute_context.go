package service

import (
	"context"
	"net/http"
	"strings"
	"time"

	enginecompute "github.com/indu-forge/data_service/internal/engine/compute"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

// prepareComputeSDKContext 将计算单元声明的 inputBindings 转换成脚本内可访问的 ctx。
// 当前阶段采用“声明式预取”：脚本只能读取已经声明的数据点或查询结果，避免调试态脚本绕过项目边界。
func (s *ComputeService) prepareComputeSDKContext(ctx context.Context, unit repository.ComputeUnitRecord, runtimeInput map[string]any) (enginecompute.SDKContext, error) {
	sdk := enginecompute.SDKContext{
		Datapoints:    map[string]enginecompute.SDKDataPointValue{},
		PointBindings: map[string]string{},
		Variables:     map[string]any{},
		SQL:           map[string]any{},
		Metadata: map[string]any{
			"projectId":     unit.ProjectID,
			"computeUnitId": unit.ID,
			"computeName":   unit.Name,
			"runtimeInput":  cloneMap(runtimeInput),
		},
	}
	if s == nil {
		return sdk, nil
	}

	for _, binding := range extractComputeDatapointVariableBindings(unit.InputBindings) {
		value, err := s.readComputeSDKDatapoint(ctx, unit.ProjectID, binding.Path)
		if err != nil {
			return sdk, err
		}
		sdk.Datapoints[binding.Path] = value
		sdk.PointBindings[binding.Alias] = binding.Path
		sdk.Variables[binding.Alias] = value.Value
	}

	for key, binding := range extractComputeSQLBindings(unit.InputBindings) {
		result, err := s.executeComputeSDKQuery(ctx, unit.ProjectID, binding)
		if err != nil {
			return sdk, err
		}
		sdk.SQL[key] = result
	}
	return sdk, nil
}

// applyComputeDebugDatapointValues 只在开发态试运行时用用户填写的模拟值覆盖预取快照。
// 正式运行仍读取节点当前值，避免调试数据进入真实计算链路。
func applyComputeDebugDatapointValues(sdk *enginecompute.SDKContext, runtimeInput map[string]any) error {
	if sdk == nil || runtimeInput == nil {
		return nil
	}
	rawValues, ok := runtimeInput["datapoints"].(map[string]any)
	if !ok {
		return nil
	}
	for alias, rawSnapshot := range rawValues {
		path, declared := sdk.PointBindings[alias]
		if !declared {
			continue
		}
		snapshot, prefetched := sdk.Datapoints[path]
		if !prefetched {
			continue
		}
		debugSnapshot, ok := rawSnapshot.(map[string]any)
		if !ok {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "数据点模拟快照 "+alias+" 必须是对象")
		}
		if _, exists := debugSnapshot["value"]; !exists {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "数据点模拟快照 "+alias+" 缺少 value")
		}
		if value, exists := debugSnapshot["value"]; exists {
			snapshot.Value = value
		}
		if quality, ok := debugSnapshot["quality"].(string); ok && strings.TrimSpace(quality) != "" {
			snapshot.Quality = strings.TrimSpace(quality)
		}
		if timestamp, exists := debugSnapshot["timestamp"]; exists {
			snapshot.Timestamp = computeDebugTimestamp(timestamp)
		}
		if observedAt, exists := debugSnapshot["observedAt"]; exists {
			snapshot.ObservedAt = computeDebugOptionalTimestamp(observedAt)
		}
		if sourceTimestamp, exists := debugSnapshot["sourceTimestamp"]; exists {
			snapshot.SourceTimestamp = computeDebugOptionalTimestamp(sourceTimestamp)
		}
		sdk.Datapoints[path] = snapshot
		sdk.Variables[alias] = snapshot.Value
	}
	return nil
}

func computeDebugTimestamp(value any) string {
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func computeDebugOptionalTimestamp(value any) *string {
	text := computeDebugTimestamp(value)
	if text == "" {
		return nil
	}
	return &text
}

type computeSQLBinding struct {
	Key        string
	QueryID    string
	Parameters map[string]any
}

type computeDatapointVariableBinding struct {
	Alias string
	Path  string
}

func (s *ComputeService) readComputeSDKDatapoint(ctx context.Context, projectID, path string) (enginecompute.SDKDataPointValue, error) {
	if s.currentValues == nil || s.datapoints == nil {
		return enginecompute.SDKDataPointValue{}, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "compute SDK 当前值解析器未初始化")
	}
	record, err := s.datapoints.GetByProjectAndPath(ctx, projectID, path)
	if err != nil {
		return enginecompute.SDKDataPointValue{}, err
	}
	if record.Status != "active" {
		return enginecompute.SDKDataPointValue{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "compute SDK 数据点不是 active 状态: "+path)
	}
	resolved, err := s.currentValues.GetDataPointValue(ctx, projectID, path)
	if err != nil {
		return enginecompute.SDKDataPointValue{}, err
	}
	timestamp := resolved.Timestamp
	if resolved.ObservedAt != nil {
		timestamp = *resolved.ObservedAt
	}
	formattedObservedAt := optionalComputeSDKTimestamp(resolved.ObservedAt)
	formattedSourceTimestamp := optionalComputeSDKTimestamp(resolved.SourceTimestamp)
	capabilities := dataPointCapabilities(*record)
	return enginecompute.SDKDataPointValue{
		ID:              record.ID,
		Path:            record.Path,
		Name:            record.Name,
		DisplayName:     record.Name,
		DataType:        record.DataType,
		SourceType:      record.SourceType,
		SourceID:        cloneOptionalString(record.SourceID),
		Value:           resolved.Value,
		DefaultValue:    cloneOptionalString(record.DefaultValue),
		Quality:         resolved.Quality,
		Timestamp:       timestamp.UTC().Format(time.RFC3339Nano),
		ObservedAt:      formattedObservedAt,
		SourceTimestamp: formattedSourceTimestamp,
		Status:          record.Status,
		Unit:            cloneOptionalString(record.Unit),
		Precision:       cloneOptionalInt(record.PrecisionNum),
		Min:             cloneOptionalFloat64(record.MinValue),
		Max:             cloneOptionalFloat64(record.MaxValue),
		Tags:            cloneJSONArray(record.Tags),
		Attributes:      cloneStringMap(record.AttributeDefaults),
		Capabilities: enginecompute.SDKDataPointCapabilities{
			Get:  capabilities.Get.Enabled,
			Read: capabilities.Get.Enabled,
			Peek: capabilities.Get.Enabled,
			Set:  capabilities.Set.Enabled,
			// 计算沙箱是一次性执行上下文，不持有长期订阅；节点侧事件触发由触发配置负责。
			Subscribe: false,
			Refresh:   record.RefreshMode != "manual",
			Run:       record.SourceType == "calc.output",
			Execute:   record.SourceType == "db.query" || record.SourceType == "http.request",
			Publish:   record.SourceType == "mqtt.tag" || record.SourceType == "mqtt.subscription",
		},
	}, nil
}

func optionalComputeSDKTimestamp(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339Nano)
	return &formatted
}

func (s *ComputeService) executeComputeSDKQuery(ctx context.Context, projectID string, binding computeSQLBinding) (any, error) {
	if s.queries == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "compute SDK 查询服务未初始化")
	}
	if strings.TrimSpace(binding.QueryID) == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "compute SDK queryId 不能为空")
	}
	result, err := s.queries.ExecuteQueryForProject(ctx, projectID, binding.QueryID, ExecuteQueryInput{Parameters: binding.Parameters})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"data":          result.Data,
		"rowCount":      result.RowCount,
		"executionTime": result.ExecutionTime,
	}, nil
}

func extractComputeDatapointVariableBindings(input map[string]any) []computeDatapointVariableBinding {
	raw, ok := input["datapointVariables"]
	if !ok {
		return []computeDatapointVariableBinding{}
	}
	items, ok := raw.([]any)
	if !ok {
		return []computeDatapointVariableBinding{}
	}
	bindings := make([]computeDatapointVariableBinding, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		mapped, ok := item.(map[string]any)
		if !ok {
			continue
		}
		alias := strings.TrimSpace(firstString(mapped, "alias", "name"))
		path := strings.TrimSpace(firstString(mapped, "path", "datapointPath"))
		if alias == "" || path == "" || !isSafeComputeVariableAlias(alias) {
			continue
		}
		key := alias + "\x00" + path
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		bindings = append(bindings, computeDatapointVariableBinding{Alias: alias, Path: path})
	}
	return bindings
}

func isSafeComputeVariableAlias(alias string) bool {
	if alias == "" || strings.HasPrefix(alias, "__") {
		return false
	}
	for index, char := range alias {
		if index == 0 {
			if !(char == '_' || char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z') {
				return false
			}
			continue
		}
		if !(char == '_' || char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' || char >= '0' && char <= '9') {
			return false
		}
	}
	switch alias {
	case "ctx", "dp", "argv", "console", "require", "arguments", "eval", "await", "break", "case", "catch", "class", "const", "continue",
		"debugger", "default", "delete", "do", "else", "enum", "export", "extends", "false",
		"finally", "for", "function", "if", "import", "in", "instanceof", "let", "new", "null",
		"return", "super", "switch", "this", "throw", "true", "try", "typeof", "var", "void",
		"while", "with", "yield", "False", "None", "True", "and", "as", "assert", "async",
		"def", "del", "elif", "except", "from", "global", "is", "lambda", "nonlocal",
		"not", "or", "pass", "raise":
		return false
	default:
		return true
	}
}

func extractComputeSQLBindings(input map[string]any) map[string]computeSQLBinding {
	bindings := map[string]computeSQLBinding{}
	for _, sectionKey := range []string{"queries", "sql"} {
		raw, ok := input[sectionKey]
		if !ok {
			continue
		}
		switch typed := raw.(type) {
		case []any:
			for _, item := range typed {
				if mapped, ok := item.(map[string]any); ok {
					binding := sqlBindingFromMap(mapped)
					if binding.Key != "" {
						bindings[binding.Key] = binding
					}
				}
			}
		case map[string]any:
			for key, item := range typed {
				switch value := item.(type) {
				case string:
					bindings[key] = computeSQLBinding{Key: key, QueryID: strings.TrimSpace(value)}
				case map[string]any:
					binding := sqlBindingFromMap(value)
					if binding.Key == "" {
						binding.Key = key
					}
					bindings[binding.Key] = binding
				}
			}
		}
	}
	return bindings
}

func sqlBindingFromMap(input map[string]any) computeSQLBinding {
	key := strings.TrimSpace(firstString(input, "key", "name", "alias"))
	queryID := strings.TrimSpace(firstString(input, "queryId", "id"))
	parameters := map[string]any{}
	if rawParams, ok := input["parameters"].(map[string]any); ok {
		parameters = cloneMap(rawParams)
	}
	return computeSQLBinding{Key: key, QueryID: queryID, Parameters: parameters}
}
