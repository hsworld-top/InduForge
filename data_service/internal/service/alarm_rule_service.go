package service

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/indu-forge/data_service/internal/auth"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

var allowedAlarmRuleTypes = map[string]struct{}{
	"threshold":  {},
	"range":      {},
	"expression": {},
}

var allowedAlarmSeverities = map[string]struct{}{
	"info":     {},
	"warning":  {},
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
	RuleType          string         `json:"ruleType"`
	Condition         map[string]any `json:"condition"`
	Severity          string         `json:"severity"`
	Hysteresis        *float64       `json:"hysteresis,omitempty"`
	SampleWindowMS    *int           `json:"sampleWindowMs,omitempty"`
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
	Name           string
	Description    *string
	TargetPath     string
	RuleType       string
	Condition      map[string]any
	Severity       string
	Hysteresis     *float64
	SampleWindowMS *int
	Contract       map[string]any
	IsEnabled      *bool
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
}

type AlarmRuleTestResult struct {
	Triggered   bool           `json:"triggered"`
	Severity    string         `json:"severity"`
	RuleType    string         `json:"ruleType"`
	TargetPath  string         `json:"targetPath"`
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
	contract := buildAlarmRuleContract(normalized, target, "")
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
		contract = buildAlarmRuleContract(normalized, target, current.ID)
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
	if target.DataType != "number" && target.DataType != "integer" && rule.RuleType != "expression" {
		result.Valid = false
		result.Reason = "阈值/区间规则需要数值型数据点"
	}
	return result, nil
}

func (s *AlarmRuleService) Test(ctx context.Context, claims *auth.Claims, projectID, ruleID string, input AlarmRuleTestInput) (*AlarmRuleTestResult, error) {
	rule, err := s.Get(ctx, claims, projectID, ruleID)
	if err != nil {
		return nil, err
	}
	triggered, diagnostics, err := evaluateAlarmRule(rule.RuleType, rule.Condition, input.Value)
	if err != nil {
		return nil, err
	}
	if input.Timestamp != nil {
		diagnostics["timestamp"] = input.Timestamp.Format(time.RFC3339)
	}
	return &AlarmRuleTestResult{
		Triggered:   triggered,
		Severity:    rule.Severity,
		RuleType:    rule.RuleType,
		TargetPath:  rule.TargetPath,
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
	contract["targetPath"] = rule.TargetPath
	contract["ruleType"] = rule.RuleType
	contract["severity"] = rule.Severity
	contract["enabled"] = rule.IsEnabled
	contract["version"] = rule.UpdatedAt.Format(time.RFC3339)
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
	Name           string
	Description    *string
	TargetPath     string
	RuleType       string
	Condition      map[string]any
	Severity       string
	Hysteresis     *float64
	SampleWindowMS *int
	Contract       map[string]any
	IsEnabled      bool
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
		ruleType = "threshold"
	}
	if severity == "" {
		severity = "warning"
	}
	if err := validateAlarmCondition(ruleType, input.Condition); err != nil {
		return normalizedAlarmRuleInput{}, nil, err
	}
	enabled := true
	if input.IsEnabled != nil {
		enabled = *input.IsEnabled
	}
	return normalizedAlarmRuleInput{
		Name:           name,
		Description:    trimOptionalString(input.Description),
		TargetPath:     target.Path,
		RuleType:       ruleType,
		Condition:      cloneMap(input.Condition),
		Severity:       severity,
		Hysteresis:     input.Hysteresis,
		SampleWindowMS: input.SampleWindowMS,
		Contract:       cloneMap(input.Contract),
		IsEnabled:      enabled,
	}, target, nil
}

func (s *AlarmRuleService) mergeUpdateInput(ctx context.Context, projectID string, current repository.AlarmRuleRecord, input UpdateAlarmRuleInput) (normalizedAlarmRuleInput, *repository.DataPointRecord, error) {
	result := normalizedAlarmRuleInput{
		Name:           current.Name,
		Description:    cloneOptionalString(current.Description),
		TargetPath:     current.TargetPath,
		RuleType:       current.RuleType,
		Condition:      cloneMap(current.Condition),
		Severity:       current.Severity,
		Hysteresis:     cloneOptionalFloat64(current.Hysteresis),
		SampleWindowMS: cloneOptionalInt(current.SampleWindowMS),
		Contract:       cloneMap(current.Contract),
		IsEnabled:      current.IsEnabled,
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
		RuleType:          record.RuleType,
		Condition:         cloneMap(record.Condition),
		Severity:          record.Severity,
		Hysteresis:        cloneOptionalFloat64(record.Hysteresis),
		SampleWindowMS:    cloneOptionalInt(record.SampleWindowMS),
		Contract:          cloneMap(record.Contract),
		IsEnabled:         record.IsEnabled,
		CreatedAt:         record.CreatedAt,
		UpdatedAt:         record.UpdatedAt,
	}
}

func normalizeAlarmFilters(ruleType, severity string) (string, string, error) {
	ruleType = strings.TrimSpace(strings.ToLower(ruleType))
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
	if ruleType == "expression" {
		if strings.TrimSpace(fmt.Sprintf("%v", condition["expression"])) == "" {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "expression 规则需要 condition.expression")
		}
	}
	if _, _, err := evaluateAlarmRule(ruleType, condition, 0); err != nil {
		return err
	}
	return nil
}

func evaluateAlarmRule(ruleType string, condition map[string]any, value any) (bool, map[string]any, error) {
	diagnostics := map[string]any{"sampleValue": value, "condition": cloneMap(condition)}
	switch ruleType {
	case "threshold":
		number, err := anyToFloat64(value)
		if err != nil {
			return false, nil, err
		}
		threshold, err := anyToFloat64(condition["value"])
		if err != nil {
			return false, nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "threshold 规则需要数值 condition.value")
		}
		operator := strings.TrimSpace(fmt.Sprintf("%v", condition["operator"]))
		if operator == "" {
			operator = ">="
		}
		triggered, err := compareAlarmValue(number, threshold, operator)
		if err != nil {
			return false, nil, err
		}
		diagnostics["operator"] = operator
		diagnostics["threshold"] = threshold
		return triggered, diagnostics, nil
	case "range":
		number, err := anyToFloat64(value)
		if err != nil {
			return false, nil, err
		}
		low, lowErr := anyToFloat64(condition["low"])
		high, highErr := anyToFloat64(condition["high"])
		if lowErr != nil || highErr != nil || low > high {
			return false, nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "range 规则需要有效的 condition.low/high")
		}
		diagnostics["low"] = low
		diagnostics["high"] = high
		return number < low || number > high, diagnostics, nil
	case "expression":
		expression := strings.TrimSpace(fmt.Sprintf("%v", condition["expression"]))
		triggered, err := evaluateAlarmExpression(expression, value)
		if err != nil {
			return false, nil, err
		}
		diagnostics["expression"] = expression
		return triggered, diagnostics, nil
	default:
		return false, nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "ruleType 不受支持")
	}
}

func evaluateAlarmExpression(expression string, value any) (bool, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return false, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "expression 不能为空")
	}
	if strings.Contains(expression, "||") {
		parts := strings.Split(expression, "||")
		for _, part := range parts {
			matched, err := evaluateAlarmExpression(strings.TrimSpace(part), value)
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
			matched, err := evaluateAlarmExpression(strings.TrimSpace(part), value)
			if err != nil {
				return false, err
			}
			if !matched {
				return false, nil
			}
		}
		return true, nil
	}
	return evaluateAlarmExpressionClause(expression, value)
}

func evaluateAlarmExpressionClause(clause string, value any) (bool, error) {
	clause = strings.TrimSpace(strings.Trim(clause, "() "))
	for _, operator := range []string{">=", "<=", "==", "!=", ">", "<"} {
		index := strings.Index(clause, operator)
		if index < 0 {
			continue
		}
		left := strings.TrimSpace(clause[:index])
		right := strings.TrimSpace(clause[index+len(operator):])
		if left != "value" && left != "$value" && left != "sample" && left != "sampleValue" {
			return false, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "expression 仅支持 value/sampleValue 变量")
		}
		return compareExpressionValue(value, right, operator)
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

func buildAlarmRuleContract(input normalizedAlarmRuleInput, target *repository.DataPointRecord, ruleID string) map[string]any {
	contract := cloneMap(input.Contract)
	contract["ruleId"] = ruleID
	contract["targetPath"] = input.TargetPath
	contract["targetDatapointId"] = target.ID
	contract["targetDataType"] = target.DataType
	contract["ruleType"] = input.RuleType
	contract["condition"] = cloneMap(input.Condition)
	contract["severity"] = input.Severity
	contract["hysteresis"] = input.Hysteresis
	contract["sampleWindowMs"] = input.SampleWindowMS
	contract["enabled"] = input.IsEnabled
	return contract
}
