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

var alarmPolicyModes = map[string]struct{}{
	"per_target": {},
	"derived":    {},
}

var alarmInputKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type AlarmPolicyGroup struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	Name        string    `json:"name"`
	ParentID    *string   `json:"parentId,omitempty"`
	Description *string   `json:"description,omitempty"`
	IsEnabled   bool      `json:"isEnabled"`
	SortOrder   int       `json:"sortOrder"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type AlarmTargetRef struct {
	DatapointID string `json:"datapointId"`
	Path        string `json:"path"`
	Name        string `json:"name,omitempty"`
	DataType    string `json:"dataType"`
}

type AlarmInputRef struct {
	Key         string `json:"key"`
	DatapointID string `json:"datapointId"`
	Path        string `json:"path"`
	Name        string `json:"name,omitempty"`
	DataType    string `json:"dataType"`
}

type AlarmCondition struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Name      string         `json:"name"`
	IsEnabled bool           `json:"isEnabled"`
	Severity  string         `json:"severity"`
	Params    map[string]any `json:"params"`
}

type AlarmPolicy struct {
	ID                string           `json:"id"`
	ProjectID         string           `json:"projectId"`
	GroupID           *string          `json:"groupId,omitempty"`
	GroupName         *string          `json:"groupName,omitempty"`
	GroupEnabled      *bool            `json:"groupEnabled,omitempty"`
	Name              string           `json:"name"`
	Description       *string          `json:"description,omitempty"`
	Mode              string           `json:"mode"`
	Targets           []AlarmTargetRef `json:"targets"`
	Inputs            []AlarmInputRef  `json:"inputs"`
	DerivedExpression string           `json:"derivedExpression"`
	Conditions        []AlarmCondition `json:"conditions"`
	Suppression       map[string]any   `json:"suppression"`
	MessageTemplate   string           `json:"messageTemplate"`
	IsEnabled         bool             `json:"isEnabled"`
	EffectiveEnabled  bool             `json:"effectiveEnabled"`
	Contract          map[string]any   `json:"contract"`
	CreatedAt         time.Time        `json:"createdAt"`
	UpdatedAt         time.Time        `json:"updatedAt"`
}

type AlarmPolicyPagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

type AlarmPolicyListResult struct {
	Policies   []AlarmPolicy         `json:"list"`
	Pagination AlarmPolicyPagination `json:"pagination"`
}

type AlarmPolicyTreeResult struct {
	Groups             []AlarmPolicyGroup `json:"groups"`
	RootPolicies       []AlarmPolicy      `json:"rootPolicies"`
	Policies           []AlarmPolicy      `json:"policies"`
	MatchedPolicyCount int                `json:"matchedPolicyCount"`
	TotalPolicyCount   int                `json:"totalPolicyCount"`
}

type AlarmPolicyCoverageItem struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Mode             string `json:"mode"`
	TargetCount      int    `json:"targetCount"`
	IsSingleTarget   bool   `json:"isSingleTarget"`
	IsEnabled        bool   `json:"isEnabled"`
	EffectiveEnabled bool   `json:"effectiveEnabled"`
	GroupName        string `json:"groupName,omitempty"`
}

type AlarmPolicyCoverageResult struct {
	DatapointID string                    `json:"datapointId,omitempty"`
	Path        string                    `json:"path,omitempty"`
	Policies    []AlarmPolicyCoverageItem `json:"policies"`
}

type AlarmPolicyListFilter struct {
	Search        string
	GroupID       *string
	Enabled       *bool
	Severity      string
	ConditionType string
	TargetPath    string
	Mode          string
	Page          int
	PageSize      int
}

type CreateAlarmPolicyGroupInput struct {
	Name        string
	ParentID    *string
	Description *string
	IsEnabled   *bool
	SortOrder   int
}

type UpdateAlarmPolicyGroupInput struct {
	Name        *string
	ParentID    *string
	HasParentID bool
	Description *string
	IsEnabled   *bool
	SortOrder   *int
}

type CreateAlarmPolicyInput struct {
	GroupID           *string
	Name              string
	Description       *string
	Mode              string
	Targets           []AlarmTargetRef
	Inputs            []AlarmInputRef
	DerivedExpression string
	Conditions        []AlarmCondition
	Suppression       map[string]any
	MessageTemplate   string
	IsEnabled         *bool
}

type UpdateAlarmPolicyInput struct {
	GroupID           *string
	HasGroupID        bool
	Name              *string
	Description       *string
	Mode              *string
	Targets           []AlarmTargetRef
	HasTargets        bool
	Inputs            []AlarmInputRef
	HasInputs         bool
	DerivedExpression *string
	Conditions        []AlarmCondition
	HasConditions     bool
	Suppression       map[string]any
	HasSuppression    bool
	MessageTemplate   *string
	IsEnabled         *bool
}

type AlarmPolicyTestInput struct {
	Value     any
	Values    map[string]any
	Timestamp *time.Time
	Context   map[string]any
}

type AlarmPolicyConditionResult struct {
	Condition   AlarmCondition `json:"condition"`
	Triggered   bool           `json:"triggered"`
	State       string         `json:"state"`
	Diagnostics map[string]any `json:"diagnostics"`
}

type AlarmPolicyEvaluationResult struct {
	Triggered           bool                         `json:"triggered"`
	State               string                       `json:"state"`
	TriggeredConditions []AlarmCondition             `json:"triggeredConditions"`
	Diagnostics         map[string]any               `json:"diagnostics"`
	ConditionResults    []AlarmPolicyConditionResult `json:"conditionResults"`
}

type AlarmBulkSelection struct {
	Mode             string
	PolicyIDs        []string
	Filters          AlarmPolicyListFilter
	ExcludePolicyIDs []string
}

type normalizedAlarmPolicyInput struct {
	GroupID           *string
	Name              string
	Description       *string
	Mode              string
	Targets           []AlarmTargetRef
	Inputs            []AlarmInputRef
	DerivedExpression string
	Conditions        []AlarmCondition
	Suppression       map[string]any
	MessageTemplate   string
	IsEnabled         bool
	Contract          map[string]any
}

type AlarmPolicyService struct {
	repository *repository.AlarmPolicyRepository
	datapoints *repository.DataPointRepository
}

func NewAlarmPolicyService(repo *repository.AlarmPolicyRepository, datapoints *repository.DataPointRepository) *AlarmPolicyService {
	return &AlarmPolicyService{repository: repo, datapoints: datapoints}
}

func (s *AlarmPolicyService) ListGroups(ctx context.Context, claims *auth.Claims, projectID string) ([]AlarmPolicyGroup, error) {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return nil, err
	}
	records, err := s.repository.ListGroups(ctx, projectID)
	if err != nil {
		return nil, err
	}
	groups := make([]AlarmPolicyGroup, 0, len(records))
	for _, record := range records {
		groups = append(groups, toAlarmPolicyGroup(record))
	}
	return groups, nil
}

func (s *AlarmPolicyService) CreateGroup(ctx context.Context, claims *auth.Claims, projectID string, input CreateAlarmPolicyGroupInput) (*AlarmPolicyGroup, error) {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return nil, err
	}
	name, err := normalizeNamedField(input.Name, "name", 100)
	if err != nil {
		return nil, err
	}
	enabled := true
	if input.IsEnabled != nil {
		enabled = *input.IsEnabled
	}
	parentID, err := normalizeOptionalAlarmPolicyGroupID(input.ParentID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureAlarmPolicyGroupInProject(ctx, projectID, parentID); err != nil {
		return nil, err
	}
	record, err := s.repository.CreateGroup(ctx, repository.CreateAlarmPolicyGroupParams{
		ProjectID: projectID, UserID: claims.UserID, Name: name, ParentID: parentID, Description: trimOptionalString(input.Description), IsEnabled: enabled, SortOrder: input.SortOrder,
	})
	if err != nil {
		return nil, err
	}
	group := toAlarmPolicyGroup(*record)
	return &group, nil
}

func (s *AlarmPolicyService) UpdateGroup(ctx context.Context, claims *auth.Claims, projectID, groupID string, input UpdateAlarmPolicyGroupInput) (*AlarmPolicyGroup, error) {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return nil, err
	}
	if err := validateAlarmPolicyID(groupID); err != nil {
		return nil, err
	}
	current, err := s.repository.GetGroupByProjectAndID(ctx, projectID, groupID)
	if err != nil {
		return nil, err
	}
	records, err := s.repository.ListGroups(ctx, projectID)
	if err != nil {
		return nil, err
	}
	name := current.Name
	if input.Name != nil {
		name, err = normalizeNamedField(*input.Name, "name", 100)
		if err != nil {
			return nil, err
		}
	}
	description := cloneOptionalString(current.Description)
	if input.Description != nil {
		description = trimOptionalString(input.Description)
	}
	parentID := cloneOptionalString(current.ParentID)
	if input.HasParentID {
		parentID, err = normalizeOptionalAlarmPolicyGroupID(input.ParentID)
		if err != nil {
			return nil, err
		}
		if parentID != nil {
			if *parentID == groupID {
				return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "上级报警分组不能指向自身")
			}
			if _, ok := findAlarmPolicyGroupRecord(records, *parentID); !ok {
				return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "上级报警分组不存在")
			}
			if isDescendantAlarmPolicyGroup(records, groupID, *parentID) {
				return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "上级报警分组不能指向下级分组")
			}
		}
	}
	enabled := current.IsEnabled
	if input.IsEnabled != nil {
		enabled = *input.IsEnabled
	}
	sortOrder := current.SortOrder
	if input.SortOrder != nil {
		sortOrder = *input.SortOrder
	}
	record, err := s.repository.UpdateGroup(ctx, repository.UpdateAlarmPolicyGroupParams{
		ID: groupID, ProjectID: projectID, UserID: claims.UserID, Name: name, ParentID: parentID, Description: description, IsEnabled: enabled, SortOrder: sortOrder,
	})
	if err != nil {
		return nil, err
	}
	group := toAlarmPolicyGroup(*record)
	return &group, nil
}

func (s *AlarmPolicyService) DeleteGroup(ctx context.Context, claims *auth.Claims, projectID, groupID string) error {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return err
	}
	if err := validateAlarmPolicyID(groupID); err != nil {
		return err
	}
	policyIDs, err := s.repository.ListPolicyIDsByGroupTree(ctx, projectID, groupID)
	if err != nil {
		return err
	}
	if err := s.repository.DeletePolicies(ctx, projectID, policyIDs); err != nil {
		return err
	}
	return s.repository.DeleteGroup(ctx, projectID, groupID)
}

func (s *AlarmPolicyService) ToggleGroupEnabled(ctx context.Context, claims *auth.Claims, projectID, groupID string, enabled bool) (*AlarmPolicyGroup, error) {
	return s.UpdateGroup(ctx, claims, projectID, groupID, UpdateAlarmPolicyGroupInput{IsEnabled: &enabled})
}

func (s *AlarmPolicyService) List(ctx context.Context, claims *auth.Claims, projectID string, filter AlarmPolicyListFilter) (*AlarmPolicyListResult, error) {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return nil, err
	}
	normalized, err := normalizeAlarmPolicyListFilter(filter)
	if err != nil {
		return nil, err
	}
	records, total, err := s.repository.ListPolicies(ctx, projectID, toRepositoryAlarmPolicyFilter(normalized))
	if err != nil {
		return nil, err
	}
	policies := make([]AlarmPolicy, 0, len(records))
	for _, record := range records {
		policies = append(policies, toAlarmPolicy(record))
	}
	page, pageSize := normalizePageAndSize(filter.Page, filter.PageSize, 20, 100)
	return &AlarmPolicyListResult{Policies: policies, Pagination: AlarmPolicyPagination{Page: page, PageSize: pageSize, Total: total, TotalPages: totalPages(total, pageSize)}}, nil
}

func (s *AlarmPolicyService) Tree(ctx context.Context, claims *auth.Claims, projectID string, filter AlarmPolicyListFilter) (*AlarmPolicyTreeResult, error) {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return nil, err
	}
	normalized, err := normalizeAlarmPolicyListFilter(filter)
	if err != nil {
		return nil, err
	}
	groups, err := s.ListGroups(ctx, claims, projectID)
	if err != nil {
		return nil, err
	}
	records, err := s.repository.ListPoliciesForTree(ctx, projectID, toRepositoryAlarmPolicyFilter(normalized))
	if err != nil {
		return nil, err
	}
	policies := make([]AlarmPolicy, 0, len(records))
	rootPolicies := make([]AlarmPolicy, 0)
	for _, record := range records {
		policy := toAlarmPolicy(record)
		policies = append(policies, policy)
		if policy.GroupID == nil {
			rootPolicies = append(rootPolicies, policy)
		}
	}
	return &AlarmPolicyTreeResult{Groups: groups, RootPolicies: rootPolicies, Policies: policies, MatchedPolicyCount: len(policies), TotalPolicyCount: len(policies)}, nil
}

func (s *AlarmPolicyService) Coverage(ctx context.Context, claims *auth.Claims, projectID, datapointID, path, excludePolicyID string) (*AlarmPolicyCoverageResult, error) {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return nil, err
	}
	datapointID = strings.TrimSpace(datapointID)
	path = strings.TrimSpace(path)
	excludePolicyID = strings.TrimSpace(excludePolicyID)
	if datapointID == "" && path == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "datapointId 或 path 至少需要一个")
	}
	if excludePolicyID != "" {
		if err := validateAlarmPolicyID(excludePolicyID); err != nil {
			return nil, err
		}
	}
	records, err := s.repository.ListPoliciesForTree(ctx, projectID, repository.AlarmPolicyListFilter{})
	if err != nil {
		return nil, err
	}
	items := make([]AlarmPolicyCoverageItem, 0)
	for _, record := range records {
		if record.ID == excludePolicyID {
			continue
		}
		policy := toAlarmPolicy(record)
		if !policyMatchesCoverageTarget(policy, datapointID, path) {
			continue
		}
		groupName := ""
		if policy.GroupName != nil {
			groupName = *policy.GroupName
		}
		items = append(items, AlarmPolicyCoverageItem{
			ID: policy.ID, Name: policy.Name, Mode: policy.Mode, TargetCount: len(policy.Targets),
			IsSingleTarget: len(policy.Targets) == 1, IsEnabled: policy.IsEnabled,
			EffectiveEnabled: policy.EffectiveEnabled, GroupName: groupName,
		})
	}
	return &AlarmPolicyCoverageResult{DatapointID: datapointID, Path: path, Policies: items}, nil
}

func (s *AlarmPolicyService) Get(ctx context.Context, claims *auth.Claims, projectID, policyID string) (*AlarmPolicy, error) {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return nil, err
	}
	if err := validateAlarmPolicyID(policyID); err != nil {
		return nil, err
	}
	record, err := s.repository.GetPolicyByProjectAndID(ctx, projectID, policyID)
	if err != nil {
		return nil, err
	}
	policy := toAlarmPolicy(*record)
	return &policy, nil
}

func (s *AlarmPolicyService) Create(ctx context.Context, claims *auth.Claims, projectID string, input CreateAlarmPolicyInput) (*AlarmPolicy, error) {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return nil, err
	}
	normalized, err := s.normalizeCreateInput(ctx, projectID, input)
	if err != nil {
		return nil, err
	}
	groupEnabled, err := s.groupEnabled(ctx, projectID, normalized.GroupID)
	if err != nil {
		return nil, err
	}
	contract := buildAlarmPolicyContract(normalized, "", projectID, groupEnabled)
	record, err := s.repository.CreatePolicy(ctx, toCreatePolicyParams(projectID, claims.UserID, normalized, contract))
	if err != nil {
		return nil, err
	}
	policy := toAlarmPolicy(*record)
	return &policy, nil
}

func (s *AlarmPolicyService) Update(ctx context.Context, claims *auth.Claims, projectID, policyID string, input UpdateAlarmPolicyInput) (*AlarmPolicy, error) {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return nil, err
	}
	if err := validateAlarmPolicyID(policyID); err != nil {
		return nil, err
	}
	current, err := s.repository.GetPolicyByProjectAndID(ctx, projectID, policyID)
	if err != nil {
		return nil, err
	}
	normalized, err := s.mergeUpdateInput(ctx, projectID, *current, input)
	if err != nil {
		return nil, err
	}
	groupEnabled, err := s.groupEnabled(ctx, projectID, normalized.GroupID)
	if err != nil {
		return nil, err
	}
	contract := buildAlarmPolicyContract(normalized, policyID, projectID, groupEnabled)
	record, err := s.repository.UpdatePolicy(ctx, toUpdatePolicyParams(policyID, projectID, claims.UserID, normalized, contract))
	if err != nil {
		return nil, err
	}
	policy := toAlarmPolicy(*record)
	return &policy, nil
}

func (s *AlarmPolicyService) Delete(ctx context.Context, claims *auth.Claims, projectID, policyID string) error {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return err
	}
	if err := validateAlarmPolicyID(policyID); err != nil {
		return err
	}
	return s.repository.DeletePolicy(ctx, projectID, policyID)
}

func (s *AlarmPolicyService) ToggleEnabled(ctx context.Context, claims *auth.Claims, projectID, policyID string, enabled bool) (*AlarmPolicy, error) {
	return s.Update(ctx, claims, projectID, policyID, UpdateAlarmPolicyInput{IsEnabled: &enabled})
}

func (s *AlarmPolicyService) Test(ctx context.Context, claims *auth.Claims, projectID, policyID string, input AlarmPolicyTestInput) (*AlarmPolicyEvaluationResult, error) {
	policy, err := s.Get(ctx, claims, projectID, policyID)
	if err != nil {
		return nil, err
	}
	value := input.Value
	contextMap := cloneMap(input.Context)
	if policy.Mode == "derived" {
		value, err = evaluateDerivedValue(policy.DerivedExpression, input.Values)
		if err != nil {
			return nil, err
		}
		contextMap["derivedValue"] = value
	}
	if input.Timestamp != nil {
		contextMap["timestamp"] = input.Timestamp.Format(time.RFC3339)
	}
	return evaluateAlarmPolicy(policy, value, contextMap)
}

func (s *AlarmPolicyService) Contract(ctx context.Context, claims *auth.Claims, projectID, policyID string) (map[string]any, error) {
	policy, err := s.Get(ctx, claims, projectID, policyID)
	if err != nil {
		return nil, err
	}
	contract := cloneMap(policy.Contract)
	contract["schemaVersion"] = "alarm.policy.v1"
	contract["policyId"] = policy.ID
	contract["projectId"] = projectID
	contract["enabled"] = policy.IsEnabled
	contract["effectiveEnabled"] = policy.EffectiveEnabled
	contract["updatedAt"] = policy.UpdatedAt.Format(time.RFC3339)
	return contract, nil
}

func (s *AlarmPolicyService) ValidateDraft(ctx context.Context, claims *auth.Claims, projectID string, input CreateAlarmPolicyInput) (map[string]any, error) {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return nil, err
	}
	enabled := true
	input.IsEnabled = &enabled
	normalized, err := s.normalizeCreateInput(ctx, projectID, input)
	if err != nil {
		return nil, err
	}
	contract := buildAlarmPolicyContract(normalized, "", projectID, nil)
	return map[string]any{"valid": true, "errors": []any{}, "contract": contract}, nil
}

func (s *AlarmPolicyService) BatchEnable(ctx context.Context, claims *auth.Claims, projectID string, selection AlarmBulkSelection) error {
	return s.setSelectedPoliciesEnabled(ctx, claims, projectID, selection, true)
}

func (s *AlarmPolicyService) BatchDisable(ctx context.Context, claims *auth.Claims, projectID string, selection AlarmBulkSelection) error {
	return s.setSelectedPoliciesEnabled(ctx, claims, projectID, selection, false)
}

func (s *AlarmPolicyService) BatchMove(ctx context.Context, claims *auth.Claims, projectID string, selection AlarmBulkSelection, groupID *string) error {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return err
	}
	if groupID != nil {
		if _, err := s.repository.GetGroupByProjectAndID(ctx, projectID, *groupID); err != nil {
			return err
		}
	}
	ids, err := s.resolveSelectionIDs(ctx, projectID, selection)
	if err != nil {
		return err
	}
	return s.repository.MovePolicies(ctx, projectID, claims.UserID, ids, groupID)
}

func (s *AlarmPolicyService) BatchApplyConditions(ctx context.Context, claims *auth.Claims, projectID string, selection AlarmBulkSelection, conditions []AlarmCondition) error {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return err
	}
	if err := validateAlarmConditions(conditions, mapFromRuleTypes("cel"), true); err != nil {
		return err
	}
	ids, err := s.resolveSelectionIDs(ctx, projectID, selection)
	if err != nil {
		return err
	}
	for _, id := range ids {
		current, err := s.repository.GetPolicyByProjectAndID(ctx, projectID, id)
		if err != nil {
			return err
		}
		policy := toAlarmPolicy(*current)
		nextConditions := mergeAlarmConditionsByType(policy.Conditions, conditions)
		_, err = s.Update(ctx, claims, projectID, id, UpdateAlarmPolicyInput{Conditions: nextConditions, HasConditions: true})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *AlarmPolicyService) validateReadAccess(claims *auth.Claims, projectID string) error {
	if s == nil || s.repository == nil || s.datapoints == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "alarm policy 服务未初始化")
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

func (s *AlarmPolicyService) validateWriteAccess(claims *auth.Claims, projectID string) error {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return err
	}
	return validateUserID(claims.UserID)
}

func (s *AlarmPolicyService) normalizeCreateInput(ctx context.Context, projectID string, input CreateAlarmPolicyInput) (normalizedAlarmPolicyInput, error) {
	name, err := normalizeNamedField(input.Name, "name", 100)
	if err != nil {
		return normalizedAlarmPolicyInput{}, err
	}
	enabled := true
	if input.IsEnabled != nil {
		enabled = *input.IsEnabled
	}
	mode, err := validateAlarmPolicyMode(input.Mode)
	if err != nil {
		return normalizedAlarmPolicyInput{}, err
	}
	targets, err := s.normalizeTargets(ctx, projectID, input.Targets)
	if err != nil {
		return normalizedAlarmPolicyInput{}, err
	}
	inputs, err := s.normalizeInputs(ctx, projectID, input.Inputs)
	if err != nil {
		return normalizedAlarmPolicyInput{}, err
	}
	derivedExpression := strings.TrimSpace(input.DerivedExpression)
	if enabled {
		if err := validateAlarmPolicyShape(mode, targets, inputs, derivedExpression); err != nil {
			return normalizedAlarmPolicyInput{}, err
		}
	}
	if err := validateAlarmConditions(input.Conditions, policyAllowedConditionTypes(mode, targets), enabled); err != nil {
		return normalizedAlarmPolicyInput{}, err
	}
	suppression, err := normalizeAlarmSuppression(input.Suppression)
	if err != nil {
		return normalizedAlarmPolicyInput{}, err
	}
	if input.GroupID != nil {
		groupID := strings.TrimSpace(*input.GroupID)
		if groupID == "" {
			input.GroupID = nil
		} else {
			if _, err := s.repository.GetGroupByProjectAndID(ctx, projectID, groupID); err != nil {
				return normalizedAlarmPolicyInput{}, err
			}
			input.GroupID = &groupID
		}
	}
	return normalizedAlarmPolicyInput{
		GroupID: input.GroupID, Name: name, Description: trimOptionalString(input.Description), Mode: mode,
		Targets: targets, Inputs: inputs, DerivedExpression: derivedExpression, Conditions: cloneAlarmConditions(input.Conditions),
		Suppression: suppression, MessageTemplate: strings.TrimSpace(input.MessageTemplate), IsEnabled: enabled, Contract: map[string]any{},
	}, nil
}

func (s *AlarmPolicyService) mergeUpdateInput(ctx context.Context, projectID string, current repository.AlarmPolicyRecord, input UpdateAlarmPolicyInput) (normalizedAlarmPolicyInput, error) {
	result := normalizedAlarmPolicyInput{
		GroupID: current.GroupID, Name: current.Name, Description: cloneOptionalString(current.Description), Mode: current.Mode,
		Targets: alarmTargetsFromMaps(current.Targets), Inputs: alarmInputsFromMaps(current.Inputs), DerivedExpression: current.DerivedExpression,
		Conditions: alarmConditionsFromMaps(current.Conditions), Suppression: cloneMap(current.Suppression),
		MessageTemplate: current.MessageTemplate, IsEnabled: current.IsEnabled, Contract: cloneMap(current.Contract),
	}
	var err error
	if input.HasGroupID {
		result.GroupID = input.GroupID
	}
	if input.Name != nil {
		result.Name, err = normalizeNamedField(*input.Name, "name", 100)
		if err != nil {
			return normalizedAlarmPolicyInput{}, err
		}
	}
	if input.Description != nil {
		result.Description = trimOptionalString(input.Description)
	}
	if input.Mode != nil {
		result.Mode, err = validateAlarmPolicyMode(*input.Mode)
		if err != nil {
			return normalizedAlarmPolicyInput{}, err
		}
	}
	if input.HasTargets {
		result.Targets, err = s.normalizeTargets(ctx, projectID, input.Targets)
		if err != nil {
			return normalizedAlarmPolicyInput{}, err
		}
	}
	if input.HasInputs {
		result.Inputs, err = s.normalizeInputs(ctx, projectID, input.Inputs)
		if err != nil {
			return normalizedAlarmPolicyInput{}, err
		}
	}
	if input.DerivedExpression != nil {
		result.DerivedExpression = strings.TrimSpace(*input.DerivedExpression)
	}
	if input.HasConditions {
		result.Conditions = cloneAlarmConditions(input.Conditions)
	}
	if input.HasSuppression {
		result.Suppression, err = normalizeAlarmSuppression(input.Suppression)
		if err != nil {
			return normalizedAlarmPolicyInput{}, err
		}
	}
	if input.MessageTemplate != nil {
		result.MessageTemplate = strings.TrimSpace(*input.MessageTemplate)
	}
	if input.IsEnabled != nil {
		result.IsEnabled = *input.IsEnabled
	}
	if result.GroupID != nil {
		groupID := strings.TrimSpace(*result.GroupID)
		if groupID == "" {
			result.GroupID = nil
		} else {
			if _, err := s.repository.GetGroupByProjectAndID(ctx, projectID, groupID); err != nil {
				return normalizedAlarmPolicyInput{}, err
			}
			result.GroupID = &groupID
		}
	}
	if result.IsEnabled {
		if err := validateAlarmPolicyShape(result.Mode, result.Targets, result.Inputs, result.DerivedExpression); err != nil {
			return normalizedAlarmPolicyInput{}, err
		}
	}
	if err := validateAlarmConditions(result.Conditions, policyAllowedConditionTypes(result.Mode, result.Targets), result.IsEnabled); err != nil {
		return normalizedAlarmPolicyInput{}, err
	}
	return result, nil
}

func (s *AlarmPolicyService) normalizeTargets(ctx context.Context, projectID string, targets []AlarmTargetRef) ([]AlarmTargetRef, error) {
	result := make([]AlarmTargetRef, 0, len(targets))
	seen := map[string]struct{}{}
	for _, target := range targets {
		record, err := s.resolveDatapoint(ctx, projectID, target.DatapointID, target.Path)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[record.ID]; ok {
			continue
		}
		seen[record.ID] = struct{}{}
		result = append(result, AlarmTargetRef{DatapointID: record.ID, Path: record.Path, Name: record.Name, DataType: record.DataType})
	}
	return result, nil
}

func (s *AlarmPolicyService) normalizeInputs(ctx context.Context, projectID string, inputs []AlarmInputRef) ([]AlarmInputRef, error) {
	result := make([]AlarmInputRef, 0, len(inputs))
	seenKeys := map[string]struct{}{}
	for _, input := range inputs {
		key := strings.TrimSpace(input.Key)
		if !alarmInputKeyPattern.MatchString(key) {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "输入变量名格式无效")
		}
		if _, ok := seenKeys[key]; ok {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "输入变量名重复")
		}
		seenKeys[key] = struct{}{}
		record, err := s.resolveDatapoint(ctx, projectID, input.DatapointID, input.Path)
		if err != nil {
			return nil, err
		}
		result = append(result, AlarmInputRef{Key: key, DatapointID: record.ID, Path: record.Path, Name: record.Name, DataType: record.DataType})
	}
	return result, nil
}

func (s *AlarmPolicyService) resolveDatapoint(ctx context.Context, projectID, datapointID, path string) (*repository.DataPointRecord, error) {
	var record *repository.DataPointRecord
	var err error
	if strings.TrimSpace(datapointID) != "" {
		record, err = s.datapoints.GetByProjectAndID(ctx, projectID, strings.TrimSpace(datapointID))
	} else if strings.TrimSpace(path) != "" {
		record, err = s.datapoints.GetByProjectAndPath(ctx, projectID, strings.TrimSpace(path))
	} else {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "目标数据点不能为空")
	}
	if err != nil {
		return nil, err
	}
	if record.Status != "active" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "目标数据点不是 active 状态")
	}
	return record, nil
}

func (s *AlarmPolicyService) setSelectedPoliciesEnabled(ctx context.Context, claims *auth.Claims, projectID string, selection AlarmBulkSelection, enabled bool) error {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return err
	}
	ids, err := s.resolveSelectionIDs(ctx, projectID, selection)
	if err != nil {
		return err
	}
	if enabled {
		for _, id := range ids {
			current, err := s.repository.GetPolicyByProjectAndID(ctx, projectID, id)
			if err != nil {
				return err
			}
			if err := validateAlarmPolicyReady(toAlarmPolicy(*current)); err != nil {
				return err
			}
		}
	}
	return s.repository.SetPoliciesEnabled(ctx, projectID, claims.UserID, ids, enabled)
}

func (s *AlarmPolicyService) resolveSelectionIDs(ctx context.Context, projectID string, selection AlarmBulkSelection) ([]string, error) {
	switch selection.Mode {
	case "ids", "":
		if len(selection.PolicyIDs) == 0 {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "策略选择不能为空")
		}
		return selection.PolicyIDs, nil
	case "filtered":
		normalized, err := normalizeAlarmPolicyListFilter(selection.Filters)
		if err != nil {
			return nil, err
		}
		ids, err := s.repository.ListPolicyIDsByFilter(ctx, projectID, toRepositoryAlarmPolicyFilter(normalized), selection.ExcludePolicyIDs)
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "筛选结果为空")
		}
		return ids, nil
	default:
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "selection.mode 不受支持")
	}
}

func (s *AlarmPolicyService) groupEnabled(ctx context.Context, projectID string, groupID *string) (*bool, error) {
	if groupID == nil {
		return nil, nil
	}
	group, err := s.repository.GetGroupByProjectAndID(ctx, projectID, *groupID)
	if err != nil {
		return nil, err
	}
	enabled := group.IsEnabled
	return &enabled, nil
}

func validateAlarmPolicyID(id string) error {
	if _, err := uuid.Parse(strings.TrimSpace(id)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "id 格式无效", err)
	}
	return nil
}

func normalizeOptionalAlarmPolicyGroupID(groupID *string) (*string, error) {
	if groupID == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*groupID)
	if trimmed == "" {
		return nil, nil
	}
	if _, err := uuid.Parse(trimmed); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "parentId 格式无效", err)
	}
	return &trimmed, nil
}

func (s *AlarmPolicyService) ensureAlarmPolicyGroupInProject(ctx context.Context, projectID string, groupID *string) error {
	if groupID == nil {
		return nil
	}
	if _, err := s.repository.GetGroupByProjectAndID(ctx, projectID, *groupID); err != nil {
		return err
	}
	return nil
}

func validateAlarmPolicyMode(mode string) (string, error) {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		mode = "per_target"
	}
	if _, ok := alarmPolicyModes[mode]; !ok {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "mode 不受支持")
	}
	return mode, nil
}

func validateAlarmPolicyShape(mode string, targets []AlarmTargetRef, inputs []AlarmInputRef, derivedExpression string) error {
	if mode == "per_target" && len(targets) == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "逐点判断策略需要目标点")
	}
	if mode == "derived" {
		if len(inputs) == 0 {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "计算后判断策略需要输入点")
		}
		if strings.TrimSpace(derivedExpression) == "" {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "计算后判断策略需要派生表达式")
		}
	}
	return nil
}

func validateAlarmPolicyReady(policy AlarmPolicy) error {
	if err := validateAlarmPolicyShape(policy.Mode, policy.Targets, policy.Inputs, policy.DerivedExpression); err != nil {
		return err
	}
	return validateAlarmConditions(policy.Conditions, policyAllowedConditionTypes(policy.Mode, policy.Targets), true)
}

func policyMatchesCoverageTarget(policy AlarmPolicy, datapointID, path string) bool {
	if policy.Mode != "per_target" {
		return false
	}
	for _, target := range policy.Targets {
		if datapointID != "" && target.DatapointID == datapointID {
			return true
		}
		if path != "" && target.Path == path {
			return true
		}
	}
	return false
}

func validateAlarmConditions(conditions []AlarmCondition, allowedTypes map[string]struct{}, requireEnabled bool) error {
	enabledCount := 0
	seenThresholds := map[string]struct{}{}
	for _, condition := range conditions {
		if !condition.IsEnabled {
			continue
		}
		enabledCount++
		if _, ok := allowedAlarmRuleTypes[condition.Type]; !ok {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "条件类型不受支持")
		}
		if _, ok := allowedAlarmSeverities[condition.Severity]; !ok {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "条件级别不受支持")
		}
		if _, ok := allowedTypes[condition.Type]; !ok {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "当前目标点类型不支持该条件")
		}
		if isThresholdConditionType(condition.Type) {
			if _, exists := seenThresholds[condition.Type]; exists {
				return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "同一策略中高高限、高限、低限、低低限每种最多配置一个")
			}
			seenThresholds[condition.Type] = struct{}{}
		}
		if err := validateAlarmConditionParams(condition); err != nil {
			return err
		}
	}
	if requireEnabled && enabledCount == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "条件集至少需要一个启用条件")
	}
	return nil
}

func policyAllowedConditionTypes(mode string, targets []AlarmTargetRef) map[string]struct{} {
	if mode == "derived" || allTargetsKind(targets, isNumericDataType) {
		return mapFromRuleTypes("HH", "H", "L", "LL", "deviation_high", "deviation_low", "rate_of_change", "cel")
	}
	if allTargetsKind(targets, isBooleanDataType) {
		return mapFromRuleTypes("bool_equal", "cel")
	}
	if allTargetsKind(targets, isStringDataType) {
		return mapFromRuleTypes("string_equal", "string_not_equal", "string_contains", "string_regex", "cel")
	}
	return mapFromRuleTypes("cel")
}

func mapFromRuleTypes(values ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func isThresholdConditionType(conditionType string) bool {
	return conditionType == "HH" || conditionType == "H" || conditionType == "L" || conditionType == "LL"
}

func validateAlarmConditionParams(condition AlarmCondition) error {
	return validateAlarmCondition(condition.Type, condition.Params)
}

func evaluateAlarmPolicy(policy *AlarmPolicy, value any, context map[string]any) (*AlarmPolicyEvaluationResult, error) {
	if policy == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "报警策略不能为空")
	}
	conditionResults := make([]AlarmPolicyConditionResult, 0, len(policy.Conditions))
	triggeredConditions := make([]AlarmCondition, 0)
	state := "not_triggered"
	diagnostics := map[string]any{}
	for _, condition := range policy.Conditions {
		if !condition.IsEnabled {
			continue
		}
		result, err := evaluateAlarmRule(condition.Type, condition.Params, value, context)
		if err != nil {
			return nil, err
		}
		conditionResults = append(conditionResults, AlarmPolicyConditionResult{
			Condition: condition, Triggered: result.Triggered, State: result.State, Diagnostics: cloneMap(result.Diagnostics),
		})
		if result.State == "insufficient_input" && state != "triggered" {
			state = "insufficient_input"
		}
		if result.Triggered {
			state = "triggered"
			triggeredConditions = append(triggeredConditions, condition)
		}
	}
	if len(conditionResults) == 0 {
		state = "insufficient_input"
		diagnostics["missing"] = "enabledConditions"
	}
	return &AlarmPolicyEvaluationResult{
		Triggered: len(triggeredConditions) > 0, State: state, TriggeredConditions: triggeredConditions,
		Diagnostics: diagnostics, ConditionResults: conditionResults,
	}, nil
}

func evaluateDerivedValue(expression string, values map[string]any) (float64, error) {
	parser := derivedExpressionParser{input: expression, values: values}
	value, err := parser.parseExpression()
	if err != nil {
		return 0, err
	}
	parser.skipSpaces()
	if parser.pos != len(parser.input) {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "派生表达式格式无效")
	}
	return value, nil
}

type derivedExpressionParser struct {
	input  string
	pos    int
	values map[string]any
}

func (p *derivedExpressionParser) parseExpression() (float64, error) {
	left, err := p.parseTerm()
	if err != nil {
		return 0, err
	}
	for {
		p.skipSpaces()
		if p.match('+') {
			right, err := p.parseTerm()
			if err != nil {
				return 0, err
			}
			left += right
		} else if p.match('-') {
			right, err := p.parseTerm()
			if err != nil {
				return 0, err
			}
			left -= right
		} else {
			return left, nil
		}
	}
}

func (p *derivedExpressionParser) parseTerm() (float64, error) {
	left, err := p.parseFactor()
	if err != nil {
		return 0, err
	}
	for {
		p.skipSpaces()
		if p.match('*') {
			right, err := p.parseFactor()
			if err != nil {
				return 0, err
			}
			left *= right
		} else if p.match('/') {
			right, err := p.parseFactor()
			if err != nil {
				return 0, err
			}
			if right == 0 {
				return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "派生表达式除数不能为 0")
			}
			left /= right
		} else {
			return left, nil
		}
	}
}

func (p *derivedExpressionParser) parseFactor() (float64, error) {
	p.skipSpaces()
	if p.match('(') {
		value, err := p.parseExpression()
		if err != nil {
			return 0, err
		}
		p.skipSpaces()
		if !p.match(')') {
			return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "派生表达式括号不匹配")
		}
		return value, nil
	}
	if p.match('-') {
		value, err := p.parseFactor()
		return -value, err
	}
	if p.pos >= len(p.input) {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "派生表达式格式无效")
	}
	if isIdentStart(p.input[p.pos]) {
		return p.parseVariable()
	}
	return p.parseNumber()
}

func (p *derivedExpressionParser) parseVariable() (float64, error) {
	start := p.pos
	for p.pos < len(p.input) && isIdentPart(p.input[p.pos]) {
		p.pos++
	}
	name := p.input[start:p.pos]
	value, ok := p.values[name]
	if !ok {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "派生表达式引用了不存在的变量")
	}
	return anyToFloat64(value)
}

func (p *derivedExpressionParser) parseNumber() (float64, error) {
	start := p.pos
	for p.pos < len(p.input) && ((p.input[p.pos] >= '0' && p.input[p.pos] <= '9') || p.input[p.pos] == '.') {
		p.pos++
	}
	if start == p.pos {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "派生表达式格式无效")
	}
	value, err := strconv.ParseFloat(p.input[start:p.pos], 64)
	if err != nil {
		return 0, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "派生表达式数字无效", err)
	}
	return value, nil
}

func (p *derivedExpressionParser) skipSpaces() {
	for p.pos < len(p.input) && (p.input[p.pos] == ' ' || p.input[p.pos] == '\t' || p.input[p.pos] == '\n') {
		p.pos++
	}
}

func (p *derivedExpressionParser) match(ch byte) bool {
	if p.pos < len(p.input) && p.input[p.pos] == ch {
		p.pos++
		return true
	}
	return false
}

func isIdentStart(ch byte) bool {
	return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || ch == '_'
}

func isIdentPart(ch byte) bool {
	return isIdentStart(ch) || (ch >= '0' && ch <= '9')
}

func buildAlarmPolicyContract(input normalizedAlarmPolicyInput, policyID string, projectID string, groupEnabled *bool) map[string]any {
	effectiveEnabled := input.IsEnabled
	if groupEnabled != nil {
		effectiveEnabled = effectiveEnabled && *groupEnabled
	}
	return map[string]any{
		"schemaVersion":     "alarm.policy.v1",
		"policyId":          policyID,
		"projectId":         projectID,
		"groupId":           input.GroupID,
		"mode":              input.Mode,
		"targets":           input.Targets,
		"inputs":            input.Inputs,
		"derivedExpression": input.DerivedExpression,
		"conditions":        input.Conditions,
		"suppression":       cloneMap(input.Suppression),
		"messageTemplate":   input.MessageTemplate,
		"enabled":           input.IsEnabled,
		"effectiveEnabled":  effectiveEnabled,
		"updatedAt":         "",
	}
}

func normalizeAlarmPolicyListFilter(filter AlarmPolicyListFilter) (AlarmPolicyListFilter, error) {
	mode, err := validateAlarmPolicyMode(filter.Mode)
	if err != nil {
		return AlarmPolicyListFilter{}, err
	}
	if strings.TrimSpace(filter.Mode) == "" {
		mode = ""
	}
	if strings.TrimSpace(filter.Severity) != "" {
		if _, ok := allowedAlarmSeverities[strings.TrimSpace(filter.Severity)]; !ok {
			return AlarmPolicyListFilter{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "severity 不受支持")
		}
	}
	if strings.TrimSpace(filter.ConditionType) != "" {
		if _, ok := allowedAlarmRuleTypes[strings.TrimSpace(filter.ConditionType)]; !ok {
			return AlarmPolicyListFilter{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "conditionType 不受支持")
		}
	}
	return AlarmPolicyListFilter{
		Search: strings.TrimSpace(filter.Search), GroupID: trimOptionalString(filter.GroupID), Enabled: filter.Enabled,
		Severity: strings.TrimSpace(filter.Severity), ConditionType: strings.TrimSpace(filter.ConditionType),
		TargetPath: strings.TrimSpace(filter.TargetPath), Mode: mode, Page: filter.Page, PageSize: filter.PageSize,
	}, nil
}

func toRepositoryAlarmPolicyFilter(filter AlarmPolicyListFilter) repository.AlarmPolicyListFilter {
	return repository.AlarmPolicyListFilter{
		Search: filter.Search, GroupID: filter.GroupID, Enabled: filter.Enabled, Severity: filter.Severity,
		ConditionType: filter.ConditionType, TargetPath: filter.TargetPath, Mode: filter.Mode, Page: filter.Page, PageSize: filter.PageSize,
	}
}

func toAlarmPolicyGroup(record repository.AlarmPolicyGroupRecord) AlarmPolicyGroup {
	return AlarmPolicyGroup{ID: record.ID, ProjectID: record.ProjectID, Name: record.Name, ParentID: cloneOptionalString(record.ParentID), Description: cloneOptionalString(record.Description), IsEnabled: record.IsEnabled, SortOrder: record.SortOrder, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}

func findAlarmPolicyGroupRecord(records []repository.AlarmPolicyGroupRecord, groupID string) (repository.AlarmPolicyGroupRecord, bool) {
	for _, record := range records {
		if record.ID == groupID {
			return record, true
		}
	}
	return repository.AlarmPolicyGroupRecord{}, false
}

func isDescendantAlarmPolicyGroup(records []repository.AlarmPolicyGroupRecord, groupID, possibleDescendantID string) bool {
	currentID := possibleDescendantID
	visited := map[string]struct{}{}
	for currentID != "" {
		if currentID == groupID {
			return true
		}
		if _, exists := visited[currentID]; exists {
			return false
		}
		visited[currentID] = struct{}{}
		current, ok := findAlarmPolicyGroupRecord(records, currentID)
		if !ok || current.ParentID == nil {
			return false
		}
		currentID = *current.ParentID
	}
	return false
}

func toAlarmPolicy(record repository.AlarmPolicyRecord) AlarmPolicy {
	conditions := alarmConditionsFromMaps(record.Conditions)
	groupEnabled := true
	if record.GroupEnabled != nil {
		groupEnabled = *record.GroupEnabled
	}
	return AlarmPolicy{
		ID: record.ID, ProjectID: record.ProjectID, GroupID: cloneOptionalString(record.GroupID), GroupName: cloneOptionalString(record.GroupName), GroupEnabled: cloneOptionalBool(record.GroupEnabled),
		Name: record.Name, Description: cloneOptionalString(record.Description), Mode: record.Mode, Targets: alarmTargetsFromMaps(record.Targets), Inputs: alarmInputsFromMaps(record.Inputs),
		DerivedExpression: record.DerivedExpression, Conditions: conditions, Suppression: cloneMap(record.Suppression), MessageTemplate: record.MessageTemplate,
		IsEnabled: record.IsEnabled, EffectiveEnabled: record.IsEnabled && groupEnabled, Contract: cloneMap(record.Contract), CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt,
	}
}

func toCreatePolicyParams(projectID, userID string, input normalizedAlarmPolicyInput, contract map[string]any) repository.CreateAlarmPolicyParams {
	return repository.CreateAlarmPolicyParams{
		ProjectID: projectID, UserID: userID, GroupID: input.GroupID, Name: input.Name, Description: input.Description, Mode: input.Mode,
		Targets: alarmTargetsToMaps(input.Targets), Inputs: alarmInputsToMaps(input.Inputs), DerivedExpression: input.DerivedExpression,
		Conditions: alarmConditionsToMaps(input.Conditions), Suppression: input.Suppression, MessageTemplate: input.MessageTemplate,
		IsEnabled: input.IsEnabled, Contract: contract,
	}
}

func toUpdatePolicyParams(policyID, projectID, userID string, input normalizedAlarmPolicyInput, contract map[string]any) repository.UpdateAlarmPolicyParams {
	return repository.UpdateAlarmPolicyParams{
		ID:                policyID,
		ProjectID:         projectID,
		UserID:            userID,
		GroupID:           input.GroupID,
		Name:              input.Name,
		Description:       input.Description,
		Mode:              input.Mode,
		Targets:           alarmTargetsToMaps(input.Targets),
		Inputs:            alarmInputsToMaps(input.Inputs),
		DerivedExpression: input.DerivedExpression,
		Conditions:        alarmConditionsToMaps(input.Conditions),
		Suppression:       input.Suppression,
		MessageTemplate:   input.MessageTemplate,
		IsEnabled:         input.IsEnabled,
		Contract:          contract,
	}
}

func alarmTargetsToMaps(values []AlarmTargetRef) []map[string]any {
	result := make([]map[string]any, 0, len(values))
	for _, value := range values {
		result = append(result, map[string]any{"datapointId": value.DatapointID, "path": value.Path, "name": value.Name, "dataType": value.DataType})
	}
	return result
}

func alarmInputsToMaps(values []AlarmInputRef) []map[string]any {
	result := make([]map[string]any, 0, len(values))
	for _, value := range values {
		result = append(result, map[string]any{"key": value.Key, "datapointId": value.DatapointID, "path": value.Path, "name": value.Name, "dataType": value.DataType})
	}
	return result
}

func alarmConditionsToMaps(values []AlarmCondition) []map[string]any {
	result := make([]map[string]any, 0, len(values))
	for _, value := range values {
		result = append(result, map[string]any{"id": value.ID, "type": value.Type, "name": value.Name, "isEnabled": value.IsEnabled, "severity": value.Severity, "params": cloneMap(value.Params)})
	}
	return result
}

func alarmTargetsFromMaps(values []map[string]any) []AlarmTargetRef {
	result := make([]AlarmTargetRef, 0, len(values))
	for _, value := range values {
		result = append(result, AlarmTargetRef{DatapointID: stringFromAny(value["datapointId"]), Path: stringFromAny(value["path"]), Name: stringFromAny(value["name"]), DataType: stringFromAny(value["dataType"])})
	}
	return result
}

func alarmInputsFromMaps(values []map[string]any) []AlarmInputRef {
	result := make([]AlarmInputRef, 0, len(values))
	for _, value := range values {
		result = append(result, AlarmInputRef{Key: stringFromAny(value["key"]), DatapointID: stringFromAny(value["datapointId"]), Path: stringFromAny(value["path"]), Name: stringFromAny(value["name"]), DataType: stringFromAny(value["dataType"])})
	}
	return result
}

func alarmConditionsFromMaps(values []map[string]any) []AlarmCondition {
	result := make([]AlarmCondition, 0, len(values))
	for _, value := range values {
		params, _ := value["params"].(map[string]any)
		result = append(result, AlarmCondition{ID: stringFromAny(value["id"]), Type: stringFromAny(value["type"]), Name: stringFromAny(value["name"]), IsEnabled: boolFromAny(value["isEnabled"]), Severity: stringFromAny(value["severity"]), Params: cloneMap(params)})
	}
	return result
}

func cloneAlarmConditions(values []AlarmCondition) []AlarmCondition {
	result := make([]AlarmCondition, 0, len(values))
	for _, value := range values {
		value.Params = cloneMap(value.Params)
		result = append(result, value)
	}
	return result
}

func mergeAlarmConditionsByType(current []AlarmCondition, incoming []AlarmCondition) []AlarmCondition {
	byType := map[string]AlarmCondition{}
	order := make([]string, 0, len(current)+len(incoming))
	for _, condition := range current {
		byType[condition.Type] = condition
		order = append(order, condition.Type)
	}
	for _, condition := range incoming {
		if _, exists := byType[condition.Type]; !exists {
			order = append(order, condition.Type)
		}
		byType[condition.Type] = condition
	}
	result := make([]AlarmCondition, 0, len(byType))
	seen := map[string]struct{}{}
	for _, conditionType := range order {
		if _, ok := seen[conditionType]; ok {
			continue
		}
		seen[conditionType] = struct{}{}
		result = append(result, byType[conditionType])
	}
	return result
}

func allTargetsNumeric(targets []AlarmTargetRef) bool {
	return allTargetsKind(targets, isNumericDataType)
}

func allTargetsKind(targets []AlarmTargetRef, match func(string) bool) bool {
	if len(targets) == 0 {
		return true
	}
	for _, target := range targets {
		if !match(target.DataType) {
			return false
		}
	}
	return true
}

func isNumericDataType(dataType string) bool {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "number", "integer", "int", "float", "double", "decimal":
		return true
	default:
		return false
	}
}

func isBooleanDataType(dataType string) bool {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "boolean", "bool":
		return true
	default:
		return false
	}
}

func isStringDataType(dataType string) bool {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "string", "text":
		return true
	default:
		return false
	}
}

func cloneOptionalBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}

func stringFromAny(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return strings.TrimSpace(fmt.Sprintf("%v", value))
}

func boolFromAny(value any) bool {
	if result, ok := value.(bool); ok {
		return result
	}
	return false
}
