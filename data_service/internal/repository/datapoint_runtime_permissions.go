package repository

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

// RuntimePermissionGrant 表示单个动作的轻量角色授权规则。
// 当前仅支持 write，保持 allow/deny/inherit 三个字段，避免提前引入复杂权限引擎。
type RuntimePermissionGrant struct {
	AllowRoles []string `json:"allowRoles"`
	DenyRoles  []string `json:"denyRoles"`
	Inherit    bool     `json:"inherit"`
}

// DataPointRuntimePermissions 表示数据点运行态权限集合。
type DataPointRuntimePermissions struct {
	Write RuntimePermissionGrant `json:"write"`
}

type runtimePermissionGrantPayload struct {
	AllowRoles json.RawMessage `json:"allowRoles"`
	DenyRoles  json.RawMessage `json:"denyRoles"`
	Inherit    json.RawMessage `json:"inherit"`
}

func normalizeDataPointRuntimePermissions(value DataPointRuntimePermissions) DataPointRuntimePermissions {
	return DataPointRuntimePermissions{
		Write: sanitizeRuntimePermissionGrant(value.Write),
	}
}

// DefaultDataPointRuntimePermissions 返回运行态权限的默认值。
func DefaultDataPointRuntimePermissions() DataPointRuntimePermissions {
	return DataPointRuntimePermissions{
		Write: RuntimePermissionGrant{
			AllowRoles: []string{},
			DenyRoles:  []string{},
			Inherit:    true,
		},
	}
}

func (value *DataPointRuntimePermissions) UnmarshalJSON(payload []byte) error {
	if value == nil {
		return nil
	}
	trimmedPayload := bytes.TrimSpace(payload)
	if len(trimmedPayload) == 0 || bytes.Equal(trimmedPayload, []byte("null")) {
		return fmt.Errorf("runtimePermissions 不能为空")
	}

	var raw struct {
		Write json.RawMessage `json:"write"`
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmedPayload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return err
	}
	if decoder.More() {
		return fmt.Errorf("runtimePermissions 只能包含一个 JSON 对象")
	}
	if len(bytes.TrimSpace(raw.Write)) == 0 {
		return fmt.Errorf("runtimePermissions.write 不能为空")
	}
	if bytes.Equal(bytes.TrimSpace(raw.Write), []byte("null")) {
		return fmt.Errorf("runtimePermissions.write 不能为空")
	}

	var grant runtimePermissionGrantPayload
	grantDecoder := json.NewDecoder(bytes.NewReader(raw.Write))
	grantDecoder.DisallowUnknownFields()
	if err := grantDecoder.Decode(&grant); err != nil {
		return err
	}
	if grantDecoder.More() {
		return fmt.Errorf("runtimePermissions.write 只能包含一个 JSON 对象")
	}

	parsedGrant, err := parseRuntimePermissionGrantPayload(grant, "runtimePermissions.write")
	if err != nil {
		return err
	}

	*value = DataPointRuntimePermissions{
		Write: parsedGrant,
	}
	return nil
}

func (record *DataPointRecord) UnmarshalJSON(payload []byte) error {
	if record == nil {
		return nil
	}

	type alias DataPointRecord
	aux := struct {
		alias
		RuntimePermissions json.RawMessage `json:"runtimePermissions"`
	}{}
	if err := json.Unmarshal(payload, &aux); err != nil {
		return err
	}

	*record = DataPointRecord(aux.alias)
	if len(aux.RuntimePermissions) > 0 {
		if err := json.Unmarshal(aux.RuntimePermissions, &record.RuntimePermissions); err != nil {
			return err
		}
		record.RuntimePermissions = normalizeDataPointRuntimePermissions(record.RuntimePermissions)
		record.RuntimePermissionsDefined = true
	} else {
		record.RuntimePermissions = DefaultDataPointRuntimePermissions()
		record.RuntimePermissionsDefined = false
	}
	return nil
}

func sanitizeRuntimePermissionGrant(value RuntimePermissionGrant) RuntimePermissionGrant {
	return RuntimePermissionGrant{
		AllowRoles: normalizeRoleList(value.AllowRoles),
		DenyRoles:  normalizeRoleList(value.DenyRoles),
		Inherit:    value.Inherit,
	}
}

func marshalDataPointRuntimePermissions(value DataPointRuntimePermissions) ([]byte, error) {
	payload, err := json.Marshal(DataPointRuntimePermissions{
		Write: sanitizeRuntimePermissionGrant(value.Write),
	})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "运行态权限 JSON 格式无效", err)
	}
	return payload, nil
}

func unmarshalDataPointRuntimePermissions(payload []byte) (DataPointRuntimePermissions, error) {
	trimmedPayload := bytes.TrimSpace(payload)
	if len(trimmedPayload) == 0 || bytes.Equal(trimmedPayload, []byte("{}")) {
		return DefaultDataPointRuntimePermissions(), nil
	}

	var result DataPointRuntimePermissions
	if err := json.Unmarshal(trimmedPayload, &result); err != nil {
		return DataPointRuntimePermissions{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析数据点运行态权限失败", err)
	}
	return result, nil
}

func parseRuntimePermissionGrantPayload(value runtimePermissionGrantPayload, fieldPath string) (RuntimePermissionGrant, error) {
	result := RuntimePermissionGrant{
		AllowRoles: []string{},
		DenyRoles:  []string{},
		Inherit:    true,
	}

	if len(bytes.TrimSpace(value.AllowRoles)) > 0 {
		if bytes.Equal(bytes.TrimSpace(value.AllowRoles), []byte("null")) {
			return RuntimePermissionGrant{}, fmt.Errorf("%s.allowRoles 不能为空", fieldPath)
		}
		var allowRoles []string
		if err := json.Unmarshal(value.AllowRoles, &allowRoles); err != nil {
			return RuntimePermissionGrant{}, err
		}
		result.AllowRoles = normalizeRoleList(allowRoles)
	}

	if len(bytes.TrimSpace(value.DenyRoles)) > 0 {
		if bytes.Equal(bytes.TrimSpace(value.DenyRoles), []byte("null")) {
			return RuntimePermissionGrant{}, fmt.Errorf("%s.denyRoles 不能为空", fieldPath)
		}
		var denyRoles []string
		if err := json.Unmarshal(value.DenyRoles, &denyRoles); err != nil {
			return RuntimePermissionGrant{}, err
		}
		result.DenyRoles = normalizeRoleList(denyRoles)
	}

	if len(bytes.TrimSpace(value.Inherit)) > 0 {
		if bytes.Equal(bytes.TrimSpace(value.Inherit), []byte("null")) {
			return RuntimePermissionGrant{}, fmt.Errorf("%s.inherit 不能为空", fieldPath)
		}
		if err := json.Unmarshal(value.Inherit, &result.Inherit); err != nil {
			return RuntimePermissionGrant{}, err
		}
	}

	return result, nil
}

func normalizeRoleList(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return []string{}
	}
	return result
}
