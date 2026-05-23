package service

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/indu-forge/data_service/internal/auth"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

var allowedAlarmRuleTypes = map[string]struct{}{
	"H":                {},
	"L":                {},
	"HH":               {},
	"LL":               {},
	"deviation_high":   {},
	"deviation_low":    {},
	"rate_of_change":   {},
	"bool_equal":       {},
	"bool_transition":  {},
	"string_equal":     {},
	"string_not_equal": {},
	"string_contains":  {},
	"string_regex":     {},
	"cel":              {},
}

var allowedAlarmSeverities = map[string]struct{}{
	"info":     {},
	"warning":  {},
	"major":    {},
	"critical": {},
}

// AlarmRule 是面向 HTTP 层返回的报警规则定义。
type AlarmRule struct {
	ID                string         `json:"id"`
	ProjectID         string         `json:"projectId"`
	Name              string         `json:"name"`
	Description       *string        `json:"description,omitempty"`
	TargetDataPointID *string        `json:"targetDatapointId,omitempty"`
	TargetPath        string         `json:"targetPath"`
	TargetName        *string        `json:"targetName,omitempty"`
	TargetDataType    string         `json:"targetDataType,omitempty"`
	RuleType          string         `json:"ruleType"`
	Condition         map[string]any `json:"condition"`
	Severity          string         `json:"severity"`
	Hysteresis        *float64       `json:"hysteresis,omitempty"`
	SampleWindowMS    *int           `json:"sampleWindowMs,omitempty"`
	Suppression       map[string]any `json:"suppression"`
	MessageTemplate   string         `json:"messageTemplate"`
	Contract          map[string]any `json:"contract"`
	IsEnabled         bool           `json:"isEnabled"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
}

type AlarmRulePagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

type AlarmRuleListResult struct {
	Rules      []AlarmRule         `json:"list"`
	Pagination AlarmRulePagination `json:"pagination"`
}

type AlarmRuleListFilter struct {
	TargetPath string
	RuleType   string
	Severity   string
	Enabled    *bool
	Search     string
	Page       int
	PageSize   int
}

type CreateAlarmRuleInput struct {
	Name            string
	Description     *string
	TargetPath      string
	RuleType        string
	Condition       map[string]any
	Severity        string
	Hysteresis      *float64
	SampleWindowMS  *int
	Suppression     map[string]any
	MessageTemplate string
	Contract        map[string]any
	IsEnabled       *bool
}

type UpdateAlarmRuleInput struct {
	Name                *string
	Description         *string
	TargetPath          *string
	RuleType            *string
	Condition           map[string]any
	HasCondition        bool
	Severity            *string
	Hysteresis          *float64
	ClearHysteresis     bool
	SampleWindowMS      *int
	ClearSampleWindowMS bool
	Suppression         map[string]any
	HasSuppression      bool
	MessageTemplate     *string
	Contract            map[string]any
	HasContract         bool
	IsEnabled           *bool
}

type AlarmRuleTargetValidation struct {
	Valid      bool                `json:"valid"`
	Reason     string              `json:"reason,omitempty"`
	Datapoint  *AlarmRuleDatapoint `json:"datapoint,omitempty"`
	Suggestion map[string]any      `json:"suggestion,omitempty"`
}

type AlarmRuleDatapoint struct {
	ID       string `json:"id"`
	Path     string `json:"path"`
	Name     string `json:"name"`
	DataType string `json:"dataType"`
	Status   string `json:"status"`
}

type AlarmRuleTestInput struct {
	Value     any
	Timestamp *time.Time
	Context   map[string]any
}

type AlarmRuleTestResult struct {
	Triggered   bool           `json:"triggered"`
	State       string         `json:"state"`
	Severity    string         `json:"severity"`
	RuleType    string         `json:"ruleType"`
	TargetPath  string         `json:"targetPath"`
	Message     string         `json:"message"`
	Diagnostics map[string]any `json:"diagnostics"`
}

type AlarmRuleEvaluationResult struct {
	Triggered   bool           `json:"triggered"`
	State       string         `json:"state"`
	Diagnostics map[string]any `json:"diagnostics"`
}

// AlarmRuleService 承载报警规则的校验、试算和契约预览逻辑。
type AlarmRuleService struct {
	repository *repository.AlarmRuleRepository
	datapoints *repository.DataPointRepository
}

func NewAlarmRuleService(repo *repository.AlarmRuleRepository, datapoints *repository.DataPointRepository) *AlarmRuleService {
	return &AlarmRuleService{repository: repo, datapoints: datapoints}
}

func (s *AlarmRuleService) List(ctx context.Context, claims *auth.Claims, projectID string, filter AlarmRuleListFilter) (*AlarmRuleListResult, error) {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return nil, err
	}
	ruleType, severity, err := normalizeAlarmFilters(filter.RuleType, filter.Severity)
	if err != nil {
		return nil, err
	}
	records, total, err := s.repository.ListByProject(ctx, projectID, repository.AlarmRuleListFilter{
		TargetPath: strings.TrimSpace(filter.TargetPath),
		RuleType:   ruleType,
		Severity:   severity,
		Enabled:    filter.Enabled,
		Search:     strings.TrimSpace(filter.Search),
		Page:       filter.Page,
		PageSize:   filter.PageSize,
	})
	if err != nil {
		return nil, err
	}
	rules := make([]AlarmRule, 0, len(records))
	for _, record := range records {
		rules = append(rules, toAlarmRule(record))
	}
	page, pageSize := normalizePageAndSize(filter.Page, filter.PageSize, 20, 100)
	return &AlarmRuleListResult{
		Rules: rules,
		Pagination: AlarmRulePagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages(total, pageSize),
		},
	}, nil
}

func (s *AlarmRuleService) Get(ctx context.Context, claims *auth.Claims, projectID, ruleID string) (*AlarmRule, error) {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return nil, err
	}
	if err := validateAlarmRuleID(ruleID); err != nil {
		return nil, err
	}
	record, err := s.repository.GetByProjectAndID(ctx, projectID, ruleID)
	if err != nil {
		return nil, err
	}
	rule := toAlarmRule(*record)
	return &rule, nil
}

func (s *AlarmRuleService) Create(ctx context.Context, claims *auth.Claims, projectID string, input CreateAlarmRuleInput) (*AlarmRule, error) {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return nil, err
	}
	normalized, target, err := s.normalizeCreateInput(ctx, projectID, input)
	if err != nil {
		return nil, err
	}
	contract := buildAlarmRuleContract(normalized, target, "", projectID)
	record, err := s.repository.Create(ctx, repository.CreateAlarmRuleParams{
		ProjectID:         projectID,
		UserID:            claims.UserID,
		Name:              normalized.Name,
		Description:       normalized.Description,
		TargetDataPointID: &target.ID,
		TargetPath:        normalized.TargetPath,
		RuleType:          normalized.RuleType,
		Condition:         normalized.Condition,
		Severity:          normalized.Severity,
		Hysteresis:        normalized.Hysteresis,
		SampleWindowMS:    normalized.SampleWindowMS,
		Suppression:       normalized.Suppression,
		MessageTemplate:   normalized.MessageTemplate,
		Contract:          contract,
		IsEnabled:         normalized.IsEnabled,
	})
	if err != nil {
		return nil, err
	}
	rule := toAlarmRule(*record)
	return &rule, nil
}

func (s *AlarmRuleService) Update(ctx context.Context, claims *auth.Claims, projectID, ruleID string, input UpdateAlarmRuleInput) (*AlarmRule, error) {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return nil, err
	}
	if err := validateAlarmRuleID(ruleID); err != nil {
		return nil, err
	}
	current, err := s.repository.GetByProjectAndID(ctx, projectID, ruleID)
	if err != nil {
		return nil, err
	}
	normalized, target, err := s.mergeUpdateInput(ctx, projectID, *current, input)
	if err != nil {
		return nil, err
	}
	contract := normalized.Contract
	if !input.HasContract {
		contract = buildAlarmRuleContract(normalized, target, current.ID, projectID)
	}
	record, err := s.repository.Update(ctx, repository.UpdateAlarmRuleParams{
		ID:                current.ID,
		ProjectID:         projectID,
		UserID:            claims.UserID,
		Name:              normalized.Name,
		Description:       normalized.Description,
		TargetDataPointID: &target.ID,
		TargetPath:        normalized.TargetPath,
		RuleType:          normalized.RuleType,
		Condition:         normalized.Condition,
		Severity:          normalized.Severity,
		Hysteresis:        normalized.Hysteresis,
		SampleWindowMS:    normalized.SampleWindowMS,
		Suppression:       normalized.Suppression,
		MessageTemplate:   normalized.MessageTemplate,
		Contract:          contract,
		IsEnabled:         normalized.IsEnabled,
	})
	if err != nil {
		return nil, err
	}
	rule := toAlarmRule(*record)
	return &rule, nil
}

func (s *AlarmRuleService) Delete(ctx context.Context, claims *auth.Claims, projectID, ruleID string) error {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return err
	}
	if err := validateAlarmRuleID(ruleID); err != nil {
		return err
	}
	return s.repository.Delete(ctx, projectID, ruleID)
}

func (s *AlarmRuleService) ToggleEnabled(ctx context.Context, claims *auth.Claims, projectID, ruleID string, enabled bool) (*AlarmRule, error) {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return nil, err
	}
	if err := validateAlarmRuleID(ruleID); err != nil {
		return nil, err
	}
	current, err := s.repository.GetByProjectAndID(ctx, projectID, ruleID)
	if err != nil {
		return nil, err
	}
	input := UpdateAlarmRuleInput{IsEnabled: &enabled}
	normalized, target, err := s.mergeUpdateInput(ctx, projectID, *current, input)
	if err != nil {
		return nil, err
	}
	contract := buildAlarmRuleContract(normalized, target, current.ID, projectID)
	record, err := s.repository.Update(ctx, repository.UpdateAlarmRuleParams{
		ID:                current.ID,
		ProjectID:         projectID,
		UserID:            claims.UserID,
		Name:              normalized.Name,
		Description:       normalized.Description,
		TargetDataPointID: &target.ID,
		TargetPath:        normalized.TargetPath,
		RuleType:          normalized.RuleType,
		Condition:         normalized.Condition,
		Severity:          normalized.Severity,
		Hysteresis:        normalized.Hysteresis,
		SampleWindowMS:    normalized.SampleWindowMS,
		Suppression:       normalized.Suppression,
		MessageTemplate:   normalized.MessageTemplate,
		Contract:          contract,
		IsEnabled:         normalized.IsEnabled,
	})
	if err != nil {
		return nil, err
	}
	rule := toAlarmRule(*record)
	return &rule, nil
}

func (s *AlarmRuleService) ValidateDraft(ctx context.Context, claims *auth.Claims, projectID string, input CreateAlarmRuleInput) (map[string]any, error) {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return nil, err
	}
	normalized, target, err := s.normalizeCreateInput(ctx, projectID, input)
	if err != nil {
		return nil, err
	}
	contract := buildAlarmRuleContract(normalized, target, "", projectID)
	return map[string]any{
		"valid":    true,
		"errors":   []any{},
		"contract": contract,
		"target": map[string]any{
			"datapointId": target.ID,
			"path":        target.Path,
			"name":        target.Name,
			"dataType":    target.DataType,
		},
	}, nil
}

func (s *AlarmRuleService) ValidateTarget(ctx context.Context, claims *auth.Claims, projectID, ruleID string) (*AlarmRuleTargetValidation, error) {
	rule, err := s.Get(ctx, claims, projectID, ruleID)
	if err != nil {
		return nil, err
	}
	target, err := s.datapoints.GetByProjectAndPath(ctx, projectID, rule.TargetPath)
	if err != nil {
		return &AlarmRuleTargetValidation{Valid: false, Reason: "目标数据点不存在"}, nil
	}
	result := &AlarmRuleTargetValidation{
		Valid: target.Status == "active",
		Datapoint: &AlarmRuleDatapoint{
			ID:       target.ID,
			Path:     target.Path,
			Name:     target.Name,
			DataType: target.DataType,
			Status:   target.Status,
		},
	}
	if !result.Valid {
		result.Reason = "目标数据点不是 active 状态"
	}
	if target.DataType != "number" && target.DataType != "integer" && rule.RuleType != "cel" {
		result.Valid = false
		result.Reason = "非 CEL 规则需要数值型数据点"
	}
	return result, nil
}

func (s *AlarmRuleService) Test(ctx context.Context, claims *auth.Claims, projectID, ruleID string, input AlarmRuleTestInput) (*AlarmRuleTestResult, error) {
	rule, err := s.Get(ctx, claims, projectID, ruleID)
	if err != nil {
		return nil, err
	}
	result, err := evaluateAlarmRule(rule.RuleType, rule.Condition, input.Value, input.Context)
	if err != nil {
		return nil, err
	}
	diagnostics := cloneMap(result.Diagnostics)
	if input.Timestamp != nil {
		diagnostics["timestamp"] = input.Timestamp.Format(time.RFC3339)
	}
	return &AlarmRuleTestResult{
		Triggered:   result.Triggered,
		State:       result.State,
		Severity:    rule.Severity,
		RuleType:    rule.RuleType,
		TargetPath:  rule.TargetPath,
		Message:     renderAlarmMessage(rule.MessageTemplate, rule, input.Value),
		Diagnostics: diagnostics,
	}, nil
}

func (s *AlarmRuleService) Contract(ctx context.Context, claims *auth.Claims, projectID, ruleID string) (map[string]any, error) {
	rule, err := s.Get(ctx, claims, projectID, ruleID)
	if err != nil {
		return nil, err
	}
	contract := cloneMap(rule.Contract)
	contract["ruleId"] = rule.ID
	contract["projectId"] = projectID
	contract["schemaVersion"] = "alarm.rule.v1"
	contract["ruleType"] = rule.RuleType
	contract["severity"] = rule.Severity
	contract["suppression"] = cloneMap(rule.Suppression)
	contract["messageTemplate"] = rule.MessageTemplate
	contract["enabled"] = rule.IsEnabled
	contract["updatedAt"] = rule.UpdatedAt.Format(time.RFC3339)
	return contract, nil
}

func (s *AlarmRuleService) validateReadAccess(claims *auth.Claims, projectID string) error {
	if s == nil || s.repository == nil || s.datapoints == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "alarm rule 服务未初始化")
	}
	if claims == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeAuthTokenRequired, http.StatusUnauthorized, "请先完成认证")
	}
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	if !claims.HasProjectAccess(projectID) {
		return apperrors.NewAppError(apperrors.ErrorCodePermissionProjectMismatch, http.StatusForbidden, "项目范围不足")
	}
	return nil
}

func (s *AlarmRuleService) validateWriteAccess(claims *auth.Claims, projectID string) error {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return err
	}
	return validateUserID(claims.UserID)
}

type normalizedAlarmRuleInput struct {
	Name            string
	Description     *string
	TargetPath      string
	RuleType        string
	Condition       map[string]any
	Severity        string
	Hysteresis      *float64
	SampleWindowMS  *int
	Suppression     map[string]any
	MessageTemplate string
	Contract        map[string]any
	IsEnabled       bool
}

func (s *AlarmRuleService) normalizeCreateInput(ctx context.Context, projectID string, input CreateAlarmRuleInput) (normalizedAlarmRuleInput, *repository.DataPointRecord, error) {
	name, err := normalizeNamedField(input.Name, "name", 100)
	if err != nil {
		return normalizedAlarmRuleInput{}, nil, err
	}
	target, err := s.resolveAlarmTarget(ctx, projectID, input.TargetPath)
	if err != nil {
		return normalizedAlarmRuleInput{}, nil, err
	}
	ruleType, severity, err := normalizeAlarmFilters(input.RuleType, input.Severity)
	if err != nil {
		return normalizedAlarmRuleInput{}, nil, err
	}
	if ruleType == "" {
		ruleType = "H"
	}
	if severity == "" {
		severity = "warning"
	}
	if err := validateAlarmTargetDataType(ruleType, target.DataType); err != nil {
		return normalizedAlarmRuleInput{}, nil, err
	}
	if err := validateAlarmCondition(ruleType, input.Condition); err != nil {
		return normalizedAlarmRuleInput{}, nil, err
	}
	suppression, err := normalizeAlarmSuppression(input.Suppression)
	if err != nil {
		return normalizedAlarmRuleInput{}, nil, err
	}
	messageTemplate := normalizeAlarmMessageTemplate(input.MessageTemplate, target.Path, ruleType)
	enabled := true
	if input.IsEnabled != nil {
		enabled = *input.IsEnabled
	}
	return normalizedAlarmRuleInput{
		Name:            name,
		Description:     trimOptionalString(input.Description),
		TargetPath:      target.Path,
		RuleType:        ruleType,
		Condition:       cloneMap(input.Condition),
		Severity:        severity,
		Hysteresis:      input.Hysteresis,
		SampleWindowMS:  input.SampleWindowMS,
		Suppression:     suppression,
		MessageTemplate: messageTemplate,
		Contract:        cloneMap(input.Contract),
		IsEnabled:       enabled,
	}, target, nil
}

func (s *AlarmRuleService) mergeUpdateInput(ctx context.Context, projectID string, current repository.AlarmRuleRecord, input UpdateAlarmRuleInput) (normalizedAlarmRuleInput, *repository.DataPointRecord, error) {
	result := normalizedAlarmRuleInput{
		Name:            current.Name,
		Description:     cloneOptionalString(current.Description),
		TargetPath:      current.TargetPath,
		RuleType:        current.RuleType,
		Condition:       cloneMap(current.Condition),
		Severity:        current.Severity,
		Hysteresis:      cloneOptionalFloat64(current.Hysteresis),
		SampleWindowMS:  cloneOptionalInt(current.SampleWindowMS),
		Suppression:     cloneMap(current.Suppression),
		MessageTemplate: current.MessageTemplate,
		Contract:        cloneMap(current.Contract),
		IsEnabled:       current.IsEnabled,
	}
	var err error
	if input.Name != nil {
		result.Name, err = normalizeNamedField(*input.Name, "name", 100)
		if err != nil {
			return normalizedAlarmRuleInput{}, nil, err
		}
	}
	if input.Description != nil {
		result.Description = trimOptionalString(input.Description)
	}
	if input.TargetPath != nil {
		result.TargetPath = strings.TrimSpace(*input.TargetPath)
	}
	if input.RuleType != nil {
		result.RuleType, _, err = normalizeAlarmFilters(*input.RuleType, "")
		if err != nil {
			return normalizedAlarmRuleInput{}, nil, err
		}
	}
	if input.Severity != nil {
		_, result.Severity, err = normalizeAlarmFilters("", *input.Severity)
		if err != nil {
			return normalizedAlarmRuleInput{}, nil, err
		}
	}
	if input.HasCondition {
		result.Condition = cloneMap(input.Condition)
	}
	if input.HasSuppression {
		result.Suppression, err = normalizeAlarmSuppression(input.Suppression)
		if err != nil {
			return normalizedAlarmRuleInput{}, nil, err
		}
	}
	if input.MessageTemplate != nil {
		result.MessageTemplate = strings.TrimSpace(*input.MessageTemplate)
	}
	if input.Hysteresis != nil || input.ClearHysteresis {
		result.Hysteresis = input.Hysteresis
	}
	if input.SampleWindowMS != nil || input.ClearSampleWindowMS {
		result.SampleWindowMS = input.SampleWindowMS
	}
	if input.HasContract {
		result.Contract = cloneMap(input.Contract)
	}
	if input.IsEnabled != nil {
		result.IsEnabled = *input.IsEnabled
	}
	if err := validateAlarmCondition(result.RuleType, result.Condition); err != nil {
		return normalizedAlarmRuleInput{}, nil, err
	}
	target, err := s.resolveAlarmTarget(ctx, projectID, result.TargetPath)
	if err != nil {
		return normalizedAlarmRuleInput{}, nil, err
	}
	if err := validateAlarmTargetDataType(result.RuleType, target.DataType); err != nil {
		return normalizedAlarmRuleInput{}, nil, err
	}
	if result.MessageTemplate == "" {
		result.MessageTemplate = normalizeAlarmMessageTemplate("", target.Path, result.RuleType)
	}
	return result, target, nil
}

func (s *AlarmRuleService) resolveAlarmTarget(ctx context.Context, projectID, targetPath string) (*repository.DataPointRecord, error) {
	targetPath = strings.TrimSpace(targetPath)
	if targetPath == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "targetPath 不能为空")
	}
	target, err := s.datapoints.GetByProjectAndPath(ctx, projectID, targetPath)
	if err != nil {
		return nil, err
	}
	if target.Status != "active" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "目标数据点不是 active 状态")
	}
	return target, nil
}

func toAlarmRule(record repository.AlarmRuleRecord) AlarmRule {
	return AlarmRule{
		ID:                record.ID,
		ProjectID:         record.ProjectID,
		Name:              record.Name,
		Description:       cloneOptionalString(record.Description),
		TargetDataPointID: cloneOptionalString(record.TargetDataPointID),
		TargetPath:        record.TargetPath,
		TargetName:        cloneOptionalString(record.TargetName),
		TargetDataType:    record.TargetDataType,
		RuleType:          record.RuleType,
		Condition:         cloneMap(record.Condition),
		Severity:          record.Severity,
		Hysteresis:        cloneOptionalFloat64(record.Hysteresis),
		SampleWindowMS:    cloneOptionalInt(record.SampleWindowMS),
		Suppression:       cloneMap(record.Suppression),
		MessageTemplate:   record.MessageTemplate,
		Contract:          cloneMap(record.Contract),
		IsEnabled:         record.IsEnabled,
		CreatedAt:         record.CreatedAt,
		UpdatedAt:         record.UpdatedAt,
	}
}

func normalizeAlarmFilters(ruleType, severity string) (string, string, error) {
	ruleType = strings.TrimSpace(ruleType)
	if ruleType != "" {
		if _, ok := allowedAlarmRuleTypes[ruleType]; !ok {
			return "", "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "ruleType 不受支持")
		}
	}
	severity = strings.TrimSpace(strings.ToLower(severity))
	if severity != "" {
		if _, ok := allowedAlarmSeverities[severity]; !ok {
			return "", "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "severity 不受支持")
		}
	}
	return ruleType, severity, nil
}

func normalizeNamedField(value, field string, maxLen int) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, field+" 不能为空")
	}
	if len([]rune(value)) > maxLen {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, field+" 长度超出限制")
	}
	return value, nil
}

func validateAlarmRuleID(ruleID string) error {
	if _, err := uuid.Parse(strings.TrimSpace(ruleID)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "id 格式无效", err)
	}
	return nil
}

func validateAlarmCondition(ruleType string, condition map[string]any) error {
	switch ruleType {
	case "H", "HH", "L", "LL", "deviation_high", "deviation_low", "rate_of_change":
		if _, err := anyToFloat64(condition["limit"]); err != nil {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "condition.limit 必须是数值")
		}
		if isThresholdConditionType(ruleType) {
			if hysteresis, ok, err := optionalAlarmFloat(condition, "hysteresis"); err != nil {
				return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "condition.hysteresis 必须是数值")
			} else if ok && hysteresis < 0 {
				return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "condition.hysteresis 不能小于 0")
			}
		}
		if ruleType == "rate_of_change" {
			if _, err := anyToFloat64(condition["windowMs"]); err != nil {
				return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "rate_of_change 规则需要数值 condition.windowMs")
			}
		}
	case "bool_equal":
		if _, ok := condition["expected"].(bool); !ok {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "bool_equal 规则需要布尔 condition.expected")
		}
	case "bool_transition":
		if _, ok := condition["from"].(bool); !ok {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "bool_transition 规则需要布尔 condition.from")
		}
		if _, ok := condition["to"].(bool); !ok {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "bool_transition 规则需要布尔 condition.to")
		}
	case "string_equal", "string_not_equal", "string_contains":
		if strings.TrimSpace(fmt.Sprintf("%v", condition["expected"])) == "" {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "文本匹配规则需要 condition.expected")
		}
	case "string_regex":
		pattern := strings.TrimSpace(fmt.Sprintf("%v", condition["pattern"]))
		if pattern == "" {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "正则匹配规则需要 condition.pattern")
		}
		if _, err := regexp.Compile(pattern); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "condition.pattern 正则格式无效", err)
		}
	case "cel":
		if strings.TrimSpace(fmt.Sprintf("%v", condition["expression"])) == "" {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "cel 规则需要 condition.expression")
		}
	default:
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "ruleType 不受支持")
	}
	return nil
}

func evaluateAlarmRule(ruleType string, condition map[string]any, value any, context map[string]any) (*AlarmRuleEvaluationResult, error) {
	switch ruleType {
	case "H", "HH":
		return evaluateUpperLimit(ruleType, condition, value, context)
	case "L", "LL":
		return evaluateLowerLimit(ruleType, condition, value, context)
	case "deviation_high", "deviation_low":
		return evaluateDeviation(ruleType, condition, value, context)
	case "rate_of_change":
		return evaluateRateOfChange(condition, value, context)
	case "bool_equal":
		return evaluateBoolEqual(condition, value)
	case "bool_transition":
		return evaluateBoolTransition(condition, value, context)
	case "string_equal", "string_not_equal", "string_contains", "string_regex":
		return evaluateStringMatch(ruleType, condition, value)
	case "cel":
		return evaluateCEL(condition, value, context)
	default:
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "ruleType 不受支持")
	}
}

func evaluateUpperLimit(ruleType string, condition map[string]any, value any, context map[string]any) (*AlarmRuleEvaluationResult, error) {
	number, err := anyToFloat64(value)
	if err != nil {
		return nil, err
	}
	limit, err := anyToFloat64(condition["limit"])
	if err != nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "condition.limit 必须是数值")
	}
	hysteresis, err := thresholdHysteresis(condition)
	if err != nil {
		return nil, err
	}
	wasTriggered := contextAlarmActive(context)
	triggerLimit := limit + hysteresis
	recoverLimit := limit - hysteresis
	triggered := number >= triggerLimit
	stateReason := "value >= limit + hysteresis"
	if wasTriggered {
		triggered = number > recoverLimit
		stateReason = "active until value <= limit - hysteresis"
	}
	return alarmEvaluation(triggered, map[string]any{
		"sampleValue":  number,
		"limit":        limit,
		"hysteresis":   hysteresis,
		"triggerLimit": triggerLimit,
		"recoverLimit": recoverLimit,
		"wasTriggered": wasTriggered,
		"deadband":     []float64{recoverLimit, triggerLimit},
		"ruleType":     ruleType,
		"reason":       stateReason,
	}), nil
}

func evaluateLowerLimit(ruleType string, condition map[string]any, value any, context map[string]any) (*AlarmRuleEvaluationResult, error) {
	number, err := anyToFloat64(value)
	if err != nil {
		return nil, err
	}
	limit, err := anyToFloat64(condition["limit"])
	if err != nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "condition.limit 必须是数值")
	}
	hysteresis, err := thresholdHysteresis(condition)
	if err != nil {
		return nil, err
	}
	wasTriggered := contextAlarmActive(context)
	triggerLimit := limit - hysteresis
	recoverLimit := limit + hysteresis
	triggered := number <= triggerLimit
	stateReason := "value <= limit - hysteresis"
	if wasTriggered {
		triggered = number < recoverLimit
		stateReason = "active until value >= limit + hysteresis"
	}
	return alarmEvaluation(triggered, map[string]any{
		"sampleValue":  number,
		"limit":        limit,
		"hysteresis":   hysteresis,
		"triggerLimit": triggerLimit,
		"recoverLimit": recoverLimit,
		"wasTriggered": wasTriggered,
		"deadband":     []float64{triggerLimit, recoverLimit},
		"ruleType":     ruleType,
		"reason":       stateReason,
	}), nil
}

func evaluateDeviation(ruleType string, condition map[string]any, value any, context map[string]any) (*AlarmRuleEvaluationResult, error) {
	number, err := anyToFloat64(value)
	if err != nil {
		return nil, err
	}
	baseline, ok := condition["baselineValue"]
	if !ok {
		baseline, ok = contextValue(context, "baselineValue")
	}
	if !ok {
		return insufficientAlarmInput(map[string]any{"missing": "baselineValue"}), nil
	}
	baselineValue, err := anyToFloat64(baseline)
	if err != nil {
		return nil, err
	}
	limit, err := anyToFloat64(condition["limit"])
	if err != nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "condition.limit 必须是数值")
	}
	deviation := number - baselineValue
	triggered := deviation >= limit
	reason := "value - baselineValue >= limit"
	if ruleType == "deviation_low" {
		triggered = baselineValue-number >= limit
		reason = "baselineValue - value >= limit"
	}
	return alarmEvaluation(triggered, map[string]any{
		"sampleValue":   number,
		"baselineValue": baselineValue,
		"deviation":     deviation,
		"limit":         limit,
		"reason":        reason,
	}), nil
}

func evaluateRateOfChange(condition map[string]any, value any, context map[string]any) (*AlarmRuleEvaluationResult, error) {
	number, err := anyToFloat64(value)
	if err != nil {
		return nil, err
	}
	previous, ok := contextValue(context, "previousValue")
	if !ok {
		return insufficientAlarmInput(map[string]any{"missing": "previousValue"}), nil
	}
	previousValue, err := anyToFloat64(previous)
	if err != nil {
		return nil, err
	}
	limit, err := anyToFloat64(condition["limit"])
	if err != nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "condition.limit 必须是数值")
	}
	windowMs, err := anyToFloat64(condition["windowMs"])
	if err != nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "rate_of_change 规则需要数值 condition.windowMs")
	}
	delta := number - previousValue
	direction := strings.TrimSpace(fmt.Sprintf("%v", condition["direction"]))
	if direction == "" {
		direction = "up"
	}
	triggered := delta >= limit
	reason := "value - previousValue >= limit"
	if direction == "down" {
		triggered = previousValue-number >= limit
		reason = "previousValue - value >= limit"
	}
	return alarmEvaluation(triggered, map[string]any{
		"sampleValue":   number,
		"previousValue": previousValue,
		"delta":         delta,
		"limit":         limit,
		"windowMs":      windowMs,
		"direction":     direction,
		"reason":        reason,
	}), nil
}

func evaluateBoolEqual(condition map[string]any, value any) (*AlarmRuleEvaluationResult, error) {
	expected, ok := condition["expected"].(bool)
	if !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "bool_equal 规则需要布尔 condition.expected")
	}
	actual, ok := value.(bool)
	if !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "采样值不是布尔值")
	}
	return alarmEvaluation(actual == expected, map[string]any{
		"sampleValue": actual,
		"expected":    expected,
		"reason":      "value == expected",
	}), nil
}

func evaluateBoolTransition(condition map[string]any, value any, context map[string]any) (*AlarmRuleEvaluationResult, error) {
	from, ok := condition["from"].(bool)
	if !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "bool_transition 规则需要布尔 condition.from")
	}
	to, ok := condition["to"].(bool)
	if !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "bool_transition 规则需要布尔 condition.to")
	}
	actual, ok := value.(bool)
	if !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "采样值不是布尔值")
	}
	previous, ok := contextValue(context, "previousValue")
	if !ok {
		return insufficientAlarmInput(map[string]any{"missing": "previousValue"}), nil
	}
	previousBool, ok := previous.(bool)
	if !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "previousValue 不是布尔值")
	}
	return alarmEvaluation(previousBool == from && actual == to, map[string]any{
		"sampleValue":   actual,
		"previousValue": previousBool,
		"from":          from,
		"to":            to,
		"reason":        "previousValue == from && value == to",
	}), nil
}

func evaluateStringMatch(ruleType string, condition map[string]any, value any) (*AlarmRuleEvaluationResult, error) {
	actual := fmt.Sprintf("%v", value)
	expected := strings.TrimSpace(fmt.Sprintf("%v", condition["expected"]))
	triggered := false
	reason := ""
	switch ruleType {
	case "string_equal":
		triggered = actual == expected
		reason = "value == expected"
	case "string_not_equal":
		triggered = actual != expected
		reason = "value != expected"
	case "string_contains":
		triggered = strings.Contains(actual, expected)
		reason = "value contains expected"
	case "string_regex":
		pattern := strings.TrimSpace(fmt.Sprintf("%v", condition["pattern"]))
		matched, err := regexp.MatchString(pattern, actual)
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "condition.pattern 正则格式无效", err)
		}
		triggered = matched
		expected = pattern
		reason = "value matches pattern"
	default:
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "ruleType 不受支持")
	}
	return alarmEvaluation(triggered, map[string]any{
		"sampleValue": actual,
		"expected":    expected,
		"reason":      reason,
	}), nil
}

func evaluateCEL(condition map[string]any, value any, context map[string]any) (*AlarmRuleEvaluationResult, error) {
	expression := strings.TrimSpace(fmt.Sprintf("%v", condition["expression"]))
	triggered, err := evaluateAlarmExpression(expression, value, context)
	if err != nil {
		return nil, err
	}
	return alarmEvaluation(triggered, map[string]any{
		"sampleValue": value,
		"expression":  expression,
	}), nil
}

func alarmEvaluation(triggered bool, diagnostics map[string]any) *AlarmRuleEvaluationResult {
	state := "not_triggered"
	if triggered {
		state = "triggered"
	}
	return &AlarmRuleEvaluationResult{Triggered: triggered, State: state, Diagnostics: diagnostics}
}

func insufficientAlarmInput(diagnostics map[string]any) *AlarmRuleEvaluationResult {
	return &AlarmRuleEvaluationResult{Triggered: false, State: "insufficient_input", Diagnostics: diagnostics}
}

func evaluateAlarmExpression(expression string, value any, context map[string]any) (bool, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return false, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "expression 不能为空")
	}
	if strings.Contains(expression, "||") {
		parts := strings.Split(expression, "||")
		for _, part := range parts {
			matched, err := evaluateAlarmExpression(strings.TrimSpace(part), value, context)
			if err != nil {
				return false, err
			}
			if matched {
				return true, nil
			}
		}
		return false, nil
	}
	if strings.Contains(expression, "&&") {
		parts := strings.Split(expression, "&&")
		for _, part := range parts {
			matched, err := evaluateAlarmExpression(strings.TrimSpace(part), value, context)
			if err != nil {
				return false, err
			}
			if !matched {
				return false, nil
			}
		}
		return true, nil
	}
	return evaluateAlarmExpressionClause(expression, value, context)
}

func evaluateAlarmExpressionClause(clause string, value any, context map[string]any) (bool, error) {
	clause = strings.TrimSpace(strings.Trim(clause, "() "))
	for _, operator := range []string{">=", "<=", "==", "!=", ">", "<"} {
		index := strings.Index(clause, operator)
		if index < 0 {
			continue
		}
		left := strings.TrimSpace(clause[:index])
		right := strings.TrimSpace(clause[index+len(operator):])
		actual, ok := alarmExpressionVariable(left, value, context)
		if !ok {
			return false, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "expression 包含不支持的变量")
		}
		return compareExpressionValue(actual, right, operator)
	}
	return false, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "expression 子句格式无效")
}

func compareExpressionValue(sample any, rawExpected, operator string) (bool, error) {
	rawExpected = strings.TrimSpace(rawExpected)
	if strings.HasPrefix(rawExpected, `"`) && strings.HasSuffix(rawExpected, `"`) {
		expected := strings.Trim(rawExpected, `"`)
		actual := strings.TrimSpace(fmt.Sprintf("%v", sample))
		switch operator {
		case "==":
			return actual == expected, nil
		case "!=":
			return actual != expected, nil
		default:
			return false, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "字符串 expression 仅支持 ==/!=")
		}
	}
	expected, err := anyToFloat64(rawExpected)
	if err != nil {
		return false, err
	}
	actual, err := anyToFloat64(sample)
	if err != nil {
		return false, err
	}
	return compareAlarmValue(actual, expected, operator)
}

func alarmExpressionVariable(name string, value any, context map[string]any) (any, bool) {
	switch name {
	case "value":
		return value, true
	case "quality", "timestamp", "baselineValue", "previousValue":
		return contextValue(context, name)
	default:
		return nil, false
	}
}

func contextValue(context map[string]any, name string) (any, bool) {
	if context == nil {
		return nil, false
	}
	value, ok := context[name]
	return value, ok
}

func compareAlarmValue(sample, threshold float64, operator string) (bool, error) {
	switch operator {
	case ">":
		return sample > threshold, nil
	case ">=":
		return sample >= threshold, nil
	case "<":
		return sample < threshold, nil
	case "<=":
		return sample <= threshold, nil
	case "==":
		return sample == threshold, nil
	case "!=":
		return sample != threshold, nil
	default:
		return false, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "condition.operator 不受支持")
	}
}

func anyToFloat64(value any) (float64, error) {
	switch typed := value.(type) {
	case float64:
		return typed, nil
	case float32:
		return float64(typed), nil
	case int:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err == nil {
			return parsed, nil
		}
	}
	return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "样本值必须是数值")
}

func validateAlarmTargetDataType(ruleType, dataType string) error {
	if ruleType == "cel" {
		return nil
	}
	if isNumericDataType(dataType) && isNumericConditionType(ruleType) {
		return nil
	}
	if isBooleanDataType(dataType) && ruleType == "bool_equal" {
		return nil
	}
	if isStringDataType(dataType) && isStringConditionType(ruleType) {
		return nil
	}
	return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "数据点类型不支持该报警条件")
}

func isNumericConditionType(ruleType string) bool {
	switch ruleType {
	case "H", "HH", "L", "LL", "deviation_high", "deviation_low", "rate_of_change":
		return true
	default:
		return false
	}
}

func isThresholdConditionType(conditionType string) bool {
	return conditionType == "HH" || conditionType == "H" || conditionType == "L" || conditionType == "LL"
}

func optionalAlarmFloat(condition map[string]any, field string) (float64, bool, error) {
	value, ok := condition[field]
	if !ok || value == nil {
		return 0, false, nil
	}
	if text, ok := value.(string); ok && strings.TrimSpace(text) == "" {
		return 0, false, nil
	}
	number, err := anyToFloat64(value)
	if err != nil {
		return 0, true, err
	}
	return number, true, nil
}

func thresholdHysteresis(condition map[string]any) (float64, error) {
	hysteresis, ok, err := optionalAlarmFloat(condition, "hysteresis")
	if err != nil {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "condition.hysteresis 必须是数值")
	}
	if !ok {
		return 0, nil
	}
	if hysteresis < 0 {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "condition.hysteresis 不能小于 0")
	}
	return hysteresis, nil
}

func contextAlarmActive(condition map[string]any) bool {
	for _, key := range []string{"active", "alarmActive", "wasTriggered", "previousTriggered"} {
		if value, ok := condition[key].(bool); ok {
			return value
		}
	}
	return false
}

func isStringConditionType(ruleType string) bool {
	switch ruleType {
	case "string_equal", "string_not_equal", "string_contains", "string_regex":
		return true
	default:
		return false
	}
}

func normalizeAlarmSuppression(input map[string]any) (map[string]any, error) {
	suppression := cloneMap(input)
	if suppression == nil {
		suppression = map[string]any{}
	}
	if enabled, ok := suppression["enabled"].(bool); ok && enabled {
		duration, err := anyToFloat64(suppression["durationMs"])
		if err != nil || duration <= 0 {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "suppression.durationMs 必须大于 0")
		}
	}
	return suppression, nil
}

func normalizeAlarmMessageTemplate(input, targetPath, ruleType string) string {
	input = strings.TrimSpace(input)
	if input != "" {
		return input
	}
	return "{{targetPath}} 触发 {{ruleType}}"
}

func renderAlarmMessage(template string, rule *AlarmRule, value any) string {
	if rule == nil {
		return template
	}
	if strings.TrimSpace(template) == "" {
		template = normalizeAlarmMessageTemplate("", rule.TargetPath, rule.RuleType)
	}
	replacer := strings.NewReplacer(
		"{{targetPath}}", rule.TargetPath,
		"{{ruleType}}", rule.RuleType,
		"{{severity}}", rule.Severity,
		"{{value}}", fmt.Sprintf("%v", value),
	)
	return replacer.Replace(template)
}

func mapFromContract(contract map[string]any, key string) map[string]any {
	value, ok := contract[key].(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return cloneMap(value)
}

func stringFromContract(contract map[string]any, key string) string {
	value, ok := contract[key].(string)
	if !ok {
		return ""
	}
	return value
}

func buildAlarmRuleContract(input normalizedAlarmRuleInput, target *repository.DataPointRecord, ruleID string, projectID string) map[string]any {
	contract := cloneMap(input.Contract)
	contract["schemaVersion"] = "alarm.rule.v1"
	contract["ruleId"] = ruleID
	contract["projectId"] = projectID
	contract["target"] = map[string]any{
		"datapointId": target.ID,
		"path":        input.TargetPath,
		"dataType":    target.DataType,
	}
	contract["ruleType"] = input.RuleType
	contract["condition"] = cloneMap(input.Condition)
	contract["severity"] = input.Severity
	contract["suppression"] = cloneMap(input.Suppression)
	contract["messageTemplate"] = input.MessageTemplate
	contract["enabled"] = input.IsEnabled
	contract["updatedAt"] = ""
	contract["targetPath"] = input.TargetPath
	contract["targetDatapointId"] = target.ID
	contract["targetDataType"] = target.DataType
	return contract
}
