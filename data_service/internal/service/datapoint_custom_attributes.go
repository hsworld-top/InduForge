package service

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

var dataPointAttributeKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,63}$`)

var reservedDataPointAttributeKeys = map[string]struct{}{
	"alarm": {}, "alarm_high": {}, "alarm_low": {}, "attribute_defaults": {}, "attributes": {},
	"data_type": {}, "default_value": {}, "description": {}, "history": {}, "id": {},
	"max": {}, "max_value": {}, "min": {}, "min_value": {}, "name": {}, "path": {},
	"permissions": {}, "precision": {}, "precision_num": {}, "project_id": {}, "quality": {},
	"runtime_permissions": {}, "source": {}, "source_config": {}, "source_id": {}, "source_type": {},
	"status": {}, "tags": {}, "timestamp": {}, "unit": {}, "value": {},
}

// DataPointCustomAttributes 是开发态自定义属性默认值接口的稳定响应。
type DataPointCustomAttributes struct {
	Attributes map[string]string `json:"attributes"`
}

// GetDataPointCustomAttributes 返回单个数据点的开发态属性默认值。
func (s *DataPointService) GetDataPointCustomAttributes(ctx context.Context, projectID, id string) (*DataPointCustomAttributes, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateDataPointID(id); err != nil {
		return nil, err
	}
	record, err := s.repository.GetByProjectAndID(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	return &DataPointCustomAttributes{Attributes: cloneStringMap(record.AttributeDefaults)}, nil
}

// UpdateDataPointCustomAttributes 校验并原子替换开发态属性默认值。
func (s *DataPointService) UpdateDataPointCustomAttributes(ctx context.Context, projectID, id, userID string, attributes map[string]string) (*DataPointCustomAttributes, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateDataPointID(id); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	normalized, err := normalizeDataPointAttributeDefaults(attributes)
	if err != nil {
		return nil, err
	}
	updated, err := s.repository.UpdateAttributeDefaults(ctx, repository.UpdateDataPointAttributeDefaultsParams{
		ID: id, ProjectID: projectID, UserID: userID, AttributeDefaults: normalized,
	})
	if err != nil {
		return nil, err
	}
	return &DataPointCustomAttributes{Attributes: cloneStringMap(updated.AttributeDefaults)}, nil
}

func normalizeDataPointAttributeDefaults(attributes map[string]string) (map[string]string, error) {
	if attributes == nil {
		return map[string]string{}, nil
	}
	result := make(map[string]string, len(attributes))
	for rawKey, value := range attributes {
		key := strings.TrimSpace(rawKey)
		if _, reserved := reservedDataPointAttributeKeys[strings.ToLower(key)]; reserved {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "属性 Key“"+key+"”已被内置属性占用")
		}
		if !dataPointAttributeKeyPattern.MatchString(key) {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "属性 Key“"+key+"”格式无效，仅支持小写字母开头及小写字母、数字、点、短横线和下划线")
		}
		if _, exists := result[key]; exists {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "属性 Key“"+key+"”重复")
		}
		result[key] = value
	}
	return result, nil
}

func cloneStringMap(value map[string]string) map[string]string {
	result := make(map[string]string, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}
