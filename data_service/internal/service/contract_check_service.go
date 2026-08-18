package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/indu-forge/data_service/internal/auth"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

type ContractCheckInput struct {
	Scope      string
	ObjectType string
	ObjectID   string
}

type ContractCheckResult struct {
	Status    string                    `json:"status"`
	ProjectID string                    `json:"projectId"`
	Scope     string                    `json:"scope"`
	Summary   map[string]int            `json:"summary"`
	List      []ContractCheckResultItem `json:"list"`
	CheckedAt time.Time                 `json:"checkedAt"`
}

type ContractCheckResultItem struct {
	Module     string `json:"module"`
	ObjectType string `json:"objectType"`
	ObjectID   string `json:"objectId,omitempty"`
	Status     string `json:"status"`
	Title      string `json:"title"`
	Detail     string `json:"detail,omitempty"`
	Action     string `json:"action,omitempty"`
}

type ContractCheckRun struct {
	ID         int64           `json:"id"`
	ProjectID  string          `json:"projectId"`
	Scope      string          `json:"scope"`
	ObjectType *string         `json:"objectType,omitempty"`
	ObjectID   *string         `json:"objectId,omitempty"`
	Status     string          `json:"status"`
	Summary    map[string]int  `json:"summary"`
	Result     json.RawMessage `json:"result"`
	CreatedBy  *string         `json:"createdBy,omitempty"`
	CreatedAt  time.Time       `json:"createdAt"`
}

type ContractCheckRunListResult struct {
	Runs       []ContractCheckRun    `json:"list"`
	Pagination ComputeUnitPagination `json:"pagination"`
}

// ContractCheckService 提供实时 dry-run 检查，不在 data_service 内编排发布流程。
type ContractCheckService struct {
	datapoints *repository.DataPointRepository
	queries    *repository.QueryRepository
	compute    *repository.ComputeRepository
	alarms     *repository.AlarmPolicyRepository
	checks     *repository.ContractCheckRepository
}

func NewContractCheckService(datapoints *repository.DataPointRepository, compute *repository.ComputeRepository, alarms *repository.AlarmPolicyRepository, deps ...any) *ContractCheckService {
	var queries *repository.QueryRepository
	var checks *repository.ContractCheckRepository
	for _, dep := range deps {
		switch typed := dep.(type) {
		case *repository.QueryRepository:
			queries = typed
		case *repository.ContractCheckRepository:
			checks = typed
		}
	}
	return &ContractCheckService{datapoints: datapoints, queries: queries, compute: compute, alarms: alarms, checks: checks}
}

func (s *ContractCheckService) Run(ctx context.Context, claims *auth.Claims, projectID string, input ContractCheckInput) (*ContractCheckResult, error) {
	if err := s.validateAccess(claims, projectID); err != nil {
		return nil, err
	}
	scope := strings.TrimSpace(input.Scope)
	if scope == "" {
		scope = "project"
	}

	items := make([]ContractCheckResultItem, 0)
	datapoints, err := s.loadProjectDatapoints(ctx, projectID)
	if err != nil {
		return nil, err
	}
	datapointByPath := map[string]repository.DataPointRecord{}
	for _, point := range datapoints {
		datapointByPath[point.Path] = point
		if point.Status == "invalid" {
			items = append(items, ContractCheckResultItem{
				Module:     "datapoint",
				ObjectType: "datapoint",
				ObjectID:   point.ID,
				Status:     "failed",
				Title:      "数据点已失效",
				Detail:     point.Path,
				Action:     "恢复数据点来源或移除相关引用",
			})
		}
	}
	if len(datapoints) == 0 {
		items = append(items, ContractCheckResultItem{
			Module:     "datapoint",
			ObjectType: "project",
			Status:     "warning",
			Title:      "项目尚未定义数据点",
			Action:     "先完成接入源、查询或计算输出的数据点同步",
		})
	}

	queryByID, err := s.loadProjectQueryIndex(ctx, projectID)
	if err != nil {
		return nil, err
	}
	computeUnits, err := s.loadProjectComputeUnits(ctx, projectID)
	if err != nil {
		return nil, err
	}
	items = append(items, checkComputeContracts(computeUnits, datapointByPath, queryByID)...)

	alarmPolicies, err := s.loadProjectAlarmPolicies(ctx, projectID)
	if err != nil {
		return nil, err
	}
	items = append(items, checkAlarmContracts(alarmPolicies, datapointByPath)...)

	items = filterContractCheckItems(items, input.ObjectType, input.ObjectID)
	result := buildContractCheckResult(projectID, scope, items)
	if s.checks != nil {
		if _, err := s.checks.SaveRun(ctx, repository.SaveContractCheckRunParams{
			ProjectID:  projectID,
			Scope:      scope,
			ObjectType: optionalTrimmedString(input.ObjectType),
			ObjectID:   optionalTrimmedString(input.ObjectID),
			Status:     result.Status,
			Summary:    result.Summary,
			Result:     result,
			CreatedBy:  optionalTrimmedString(claims.UserID),
		}); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (s *ContractCheckService) Latest(ctx context.Context, claims *auth.Claims, projectID string) (*ContractCheckResult, error) {
	if err := s.validateAccess(claims, projectID); err != nil {
		return nil, err
	}
	if s.checks == nil {
		return s.Run(ctx, claims, projectID, ContractCheckInput{Scope: "project"})
	}
	record, err := s.checks.GetLatestRun(ctx, projectID)
	if err != nil {
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) && appErr.Code == apperrors.ErrorCodeNotFound {
			return s.Run(ctx, claims, projectID, ContractCheckInput{Scope: "project"})
		}
		return nil, err
	}
	var result ContractCheckResult
	if err := json.Unmarshal(record.Result, &result); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析契约检查历史结果失败", err)
	}
	return &result, nil
}

func (s *ContractCheckService) ListRuns(ctx context.Context, claims *auth.Claims, projectID string, page, pageSize int) (*ContractCheckRunListResult, error) {
	if err := s.validateAccess(claims, projectID); err != nil {
		return nil, err
	}
	if s.checks == nil {
		return &ContractCheckRunListResult{
			Runs:       []ContractCheckRun{},
			Pagination: ComputeUnitPagination{Page: 1, PageSize: 20},
		}, nil
	}
	records, total, err := s.checks.ListRuns(ctx, projectID, page, pageSize)
	if err != nil {
		return nil, err
	}
	runs := make([]ContractCheckRun, 0, len(records))
	for _, record := range records {
		run, convertErr := toContractCheckRun(record)
		if convertErr != nil {
			return nil, convertErr
		}
		runs = append(runs, run)
	}
	page, pageSize = normalizePageAndSize(page, pageSize, 20, 100)
	return &ContractCheckRunListResult{
		Runs: runs,
		Pagination: ComputeUnitPagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages(total, pageSize),
		},
	}, nil
}

func (s *ContractCheckService) validateAccess(claims *auth.Claims, projectID string) error {
	if s == nil || s.datapoints == nil || s.compute == nil || s.alarms == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "contract check 服务未初始化")
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

func (s *ContractCheckService) loadProjectDatapoints(ctx context.Context, projectID string) ([]repository.DataPointRecord, error) {
	records, _, err := s.datapoints.ListByProject(ctx, projectID, repository.DataPointListFilter{Page: 1, PageSize: 1000})
	return records, err
}

func (s *ContractCheckService) loadProjectQueryIndex(ctx context.Context, projectID string) (map[string]repository.QueryRecord, error) {
	result := map[string]repository.QueryRecord{}
	if s.queries == nil {
		return result, nil
	}
	records, _, err := s.queries.ListByProject(ctx, projectID, repository.QueryListFilter{Page: 1, PageSize: 1000})
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		result[record.ID] = record
	}
	return result, nil
}

func (s *ContractCheckService) loadProjectComputeUnits(ctx context.Context, projectID string) ([]repository.ComputeUnitRecord, error) {
	records, _, err := s.compute.ListUnits(ctx, projectID, repository.ComputeUnitListFilter{Page: 1, PageSize: 1000})
	return records, err
}

func (s *ContractCheckService) loadProjectAlarmPolicies(ctx context.Context, projectID string) ([]repository.AlarmPolicyRecord, error) {
	return s.alarms.ListAllPolicies(ctx, projectID)
}

func checkComputeContracts(units []repository.ComputeUnitRecord, datapoints map[string]repository.DataPointRecord, queries map[string]repository.QueryRecord) []ContractCheckResultItem {
	items := make([]ContractCheckResultItem, 0)
	for _, unit := range units {
		if !unit.IsEnabled {
			items = append(items, ContractCheckResultItem{
				Module:     "compute",
				ObjectType: "computeUnit",
				ObjectID:   unit.ID,
				Status:     "warning",
				Title:      "计算单元已禁用",
				Detail:     unit.Name,
			})
		}
		for _, path := range collectBindingPaths(unit.InputBindings) {
			if _, ok := datapoints[path]; !ok {
				items = append(items, ContractCheckResultItem{
					Module:     "compute",
					ObjectType: "computeInput",
					ObjectID:   unit.ID,
					Status:     "failed",
					Title:      "计算输入数据点不存在",
					Detail:     path,
					Action:     "修正 inputBindings 或补齐数据点",
				})
			}
		}
		for key, binding := range extractComputeSQLBindings(unit.InputBindings) {
			query, ok := queries[binding.QueryID]
			if !ok {
				items = append(items, ContractCheckResultItem{
					Module:     "compute",
					ObjectType: "computeQuery",
					ObjectID:   unit.ID,
					Status:     "failed",
					Title:      "计算 SQL 绑定查询不存在",
					Detail:     key + ":" + binding.QueryID,
					Action:     "修正 inputBindings.queries 或补齐查询定义",
				})
				continue
			}
			if !query.IsEnabled {
				items = append(items, ContractCheckResultItem{
					Module:     "compute",
					ObjectType: "computeQuery",
					ObjectID:   unit.ID,
					Status:     "warning",
					Title:      "计算 SQL 绑定查询已禁用",
					Detail:     key + ":" + query.Name,
				})
			}
		}
		for _, binding := range extractComputeOutputBindings(unit) {
			if point, ok := datapoints[binding.Path]; !ok || point.Status != "active" {
				items = append(items, ContractCheckResultItem{
					Module:     "compute",
					ObjectType: "computeOutput",
					ObjectID:   unit.ID,
					Status:     "warning",
					Title:      "计算输出数据点未处于 active 状态",
					Detail:     binding.Path,
					Action:     "保存计算单元以重新同步 calc.output 数据点",
				})
			}
		}
	}
	return items
}

func checkAlarmContracts(rules []repository.AlarmPolicyRecord, datapoints map[string]repository.DataPointRecord) []ContractCheckResultItem {
	items := make([]ContractCheckResultItem, 0)
	for _, rule := range rules {
		for _, binding := range rule.Bindings {
			point, ok := datapoints[binding.Path]
			if !ok || point.Status != "active" {
				items = append(items, ContractCheckResultItem{
					Module: "alarm", ObjectType: "alarmPolicy", ObjectID: rule.ID, Status: "failed",
					Title: "报警数据点不可用", Detail: binding.Path, Action: "恢复数据点或解除报警绑定",
				})
			}
		}
		if rule.Mode == "derived" && strings.TrimSpace(rule.DerivedExpression) == "" {
			items = append(items, ContractCheckResultItem{Module: "alarm", ObjectType: "alarmPolicy", ObjectID: rule.ID, Status: "failed", Title: "组合报警表达式为空", Detail: rule.Name})
		}
		if len(rule.Conditions) == 0 {
			items = append(items, ContractCheckResultItem{Module: "alarm", ObjectType: "alarmPolicy", ObjectID: rule.ID, Status: "failed", Title: "报警策略没有条件", Detail: rule.Name})
		}
		if !rule.IsEnabled {
			items = append(items, ContractCheckResultItem{
				Module:     "alarm",
				ObjectType: "alarmPolicy",
				ObjectID:   rule.ID,
				Status:     "warning",
				Title:      "报警策略已停用",
				Detail:     rule.Name,
			})
		}
	}
	return items
}

func collectBindingPaths(input map[string]any) []string {
	paths := make([]string, 0)
	if variables, ok := input["datapointVariables"].([]any); ok {
		for _, item := range variables {
			if mapped, ok := item.(map[string]any); ok {
				if rawPath, ok := mapped["path"].(string); ok && strings.TrimSpace(rawPath) != "" {
					paths = append(paths, strings.TrimSpace(rawPath))
				}
			}
		}
	}
	return uniqueContractCheckStrings(paths)
}

func uniqueContractCheckStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func buildContractCheckResult(projectID, scope string, items []ContractCheckResultItem) *ContractCheckResult {
	summary := map[string]int{"passed": 0, "warning": 0, "pending": 0, "failed": 0}
	status := "passed"
	for _, item := range items {
		summary[item.Status]++
		if item.Status == "failed" {
			status = "failed"
		} else if item.Status == "warning" && status != "failed" {
			status = "warning"
		} else if item.Status == "pending" && status == "passed" {
			status = "pending"
		}
	}
	if len(items) == 0 {
		summary["passed"] = 1
	}
	return &ContractCheckResult{
		Status:    status,
		ProjectID: projectID,
		Scope:     scope,
		Summary:   summary,
		List:      items,
		CheckedAt: time.Now().UTC(),
	}
}

func filterContractCheckItems(items []ContractCheckResultItem, objectType, objectID string) []ContractCheckResultItem {
	objectType = strings.TrimSpace(objectType)
	objectID = strings.TrimSpace(objectID)
	if objectType == "" && objectID == "" {
		return items
	}
	filtered := make([]ContractCheckResultItem, 0, len(items))
	for _, item := range items {
		if objectType != "" && item.ObjectType != objectType {
			continue
		}
		if objectID != "" && item.ObjectID != objectID {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func toContractCheckRun(record repository.ContractCheckRunRecord) (ContractCheckRun, error) {
	summary := map[string]int{}
	if len(record.Summary) > 0 {
		if err := json.Unmarshal(record.Summary, &summary); err != nil {
			return ContractCheckRun{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析契约检查摘要失败", err)
		}
	}
	return ContractCheckRun{
		ID:         record.ID,
		ProjectID:  record.ProjectID,
		Scope:      record.Scope,
		ObjectType: cloneOptionalString(record.ObjectType),
		ObjectID:   cloneOptionalString(record.ObjectID),
		Status:     record.Status,
		Summary:    summary,
		Result:     append(json.RawMessage(nil), record.Result...),
		CreatedBy:  cloneOptionalString(record.CreatedBy),
		CreatedAt:  record.CreatedAt,
	}, nil
}

func optionalTrimmedString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
