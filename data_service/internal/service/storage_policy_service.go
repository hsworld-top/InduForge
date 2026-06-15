package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/indu-forge/data_service/internal/auth"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

const (
	defaultStoragePolicyPageSize = 20
	defaultStorageSampleInterval = 1000
	defaultStorageChangeRate     = 0.3
	defaultStorageBadQualityRate = 0.05
)

var storageWriteModes = map[string]struct{}{
	"every_sample":      {},
	"on_change":         {},
	"periodic_snapshot": {},
}

var storageBindingModes = map[string]struct{}{
	"static":  {},
	"dynamic": {},
}

var storageTargetTableModes = map[string]struct{}{
	"auto_create":      {},
	"existing_mapping": {},
}

var storagePolicyStatuses = map[string]struct{}{
	"enabled":  {},
	"disabled": {},
	"error":    {},
}

var storageQualities = map[string]struct{}{
	"Good":      {},
	"Uncertain": {},
	"Bad":       {},
}

// StoragePolicy 描述统一数据点历史归档策略。
type StoragePolicy struct {
	ID                 string                    `json:"id"`
	ProjectID          string                    `json:"projectId"`
	Name               string                    `json:"name"`
	Description        *string                   `json:"description,omitempty"`
	Target             StorageTarget             `json:"target"`
	TargetCapability   string                    `json:"targetCapability"`
	BindingMode        string                    `json:"bindingMode"`
	BindingFilter      StorageDataPointFilter    `json:"bindingFilter"`
	WriteMode          string                    `json:"writeMode"`
	MinIntervalMS      *int                      `json:"minIntervalMs,omitempty"`
	Deadband           *float64                  `json:"deadband,omitempty"`
	SnapshotIntervalMS *int                      `json:"snapshotIntervalMs,omitempty"`
	IncludeQualities   []string                  `json:"includeQualities"`
	RetentionDays      int                       `json:"retentionDays"`
	TargetTableMode    string                    `json:"targetTableMode"`
	TargetTableConfig  map[string]any            `json:"targetTableConfig"`
	Status             string                    `json:"status"`
	Diagnostics        []StoragePolicyDiagnostic `json:"diagnostics"`
	BindingCount       int                       `json:"bindingCount"`
	Estimate           StoragePolicyEstimate     `json:"estimate"`
	CreatedAt          time.Time                 `json:"createdAt"`
	UpdatedAt          time.Time                 `json:"updatedAt"`
}

type StoragePolicyDetail struct {
	StoragePolicy
	Bindings []StoragePolicyBinding `json:"bindings"`
}

type StoragePolicyBinding struct {
	ID            string    `json:"id"`
	DatapointID   string    `json:"datapointId"`
	DatapointPath string    `json:"datapointPath"`
	DatapointName string    `json:"datapointName"`
	DataType      string    `json:"dataType"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
}

type StoragePolicyDiagnostic struct {
	Severity string `json:"severity"`
	Type     string `json:"type"`
	Message  string `json:"message"`
	Suggest  string `json:"suggest,omitempty"`
}

type StorageTarget struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Category     string    `json:"category"`
	Status       string    `json:"status"`
	Capabilities []string  `json:"capabilities"`
	CreatedAt    time.Time `json:"createdAt,omitempty"`
	UpdatedAt    time.Time `json:"updatedAt,omitempty"`
}

type StoragePolicyEstimate struct {
	MatchedDataPointCount int     `json:"matchedDataPointCount"`
	EventsPerSecond       float64 `json:"eventsPerSecond"`
	RowsPerDay            int64   `json:"rowsPerDay"`
	RowsByRetention       int64   `json:"rowsByRetention"`
	Assumption            string  `json:"assumption"`
}

type StoragePolicyListFilter struct {
	Search             string
	Status             string
	TargetConnectionID string
	WriteMode          string
	BindingMode        string
	Page               int
	PageSize           int
}

type StoragePolicyPagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

type StoragePolicyListResult struct {
	Policies   []StoragePolicy         `json:"list"`
	Pagination StoragePolicyPagination `json:"pagination"`
	Summary    StoragePolicySummary    `json:"summary"`
}

type StoragePolicySummary struct {
	EnabledCount          int     `json:"enabledCount"`
	ErrorCount            int     `json:"errorCount"`
	TotalBindingCount     int     `json:"totalBindingCount"`
	EstimatedRowsPerDay   int64   `json:"estimatedRowsPerDay"`
	EstimatedEventsSecond float64 `json:"estimatedEventsSecond"`
}

type StorageTargetsResult struct {
	Targets []StorageTarget `json:"targets"`
}

type StorageDataPointFilter struct {
	Type           string   `json:"type,omitempty"`
	Status         string   `json:"status,omitempty"`
	Search         string   `json:"search,omitempty"`
	AccessSourceID string   `json:"accessSourceId,omitempty"`
	SourceID       string   `json:"sourceId,omitempty"`
	SourceIDs      []string `json:"sourceIds,omitempty"`
	Tags           []string `json:"tags,omitempty"`
}

type CreateStoragePolicyInput struct {
	Name               string
	Description        *string
	TargetConnectionID string
	TargetCapability   string
	BindingMode        string
	BindingFilter      StorageDataPointFilter
	DatapointIDs       []string
	WriteMode          string
	MinIntervalMS      *int
	Deadband           *float64
	SnapshotIntervalMS *int
	IncludeQualities   []string
	RetentionDays      int
	TargetTableMode    string
	TargetTableConfig  map[string]any
	Status             *string
}

type UpdateStoragePolicyInput struct {
	CreateStoragePolicyInput
}

type StoragePolicyCoverageResult struct {
	DatapointID        string `json:"datapointId,omitempty"`
	DatapointPath      string `json:"datapointPath,omitempty"`
	MatchedPolicyCount int    `json:"matchedPolicyCount"`
}

type normalizedStoragePolicyInput struct {
	Name               string
	Description        *string
	TargetConnectionID string
	TargetCapability   string
	BindingMode        string
	BindingFilter      StorageDataPointFilter
	DatapointIDs       []string
	WriteMode          string
	MinIntervalMS      *int
	Deadband           *float64
	SnapshotIntervalMS *int
	IncludeQualities   []string
	RetentionDays      int
	TargetTableMode    string
	TargetTableConfig  map[string]any
	Status             string
	Diagnostics        []StoragePolicyDiagnostic
	Estimate           StoragePolicyEstimate
}

type StoragePolicyService struct {
	repository *repository.StoragePolicyRepository
	datapoints *repository.DataPointRepository
}

func NewStoragePolicyService(repo *repository.StoragePolicyRepository, datapoints *repository.DataPointRepository) *StoragePolicyService {
	return &StoragePolicyService{repository: repo, datapoints: datapoints}
}

func (s *StoragePolicyService) List(ctx context.Context, claims *auth.Claims, projectID string, filter StoragePolicyListFilter) (*StoragePolicyListResult, error) {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return nil, err
	}
	normalized, err := normalizeStoragePolicyListFilter(filter)
	if err != nil {
		return nil, err
	}
	records, total, err := s.repository.ListPolicies(ctx, projectID, repository.StoragePolicyListFilter{
		Search: normalized.Search, Status: normalized.Status, TargetConnectionID: normalized.TargetConnectionID,
		WriteMode: normalized.WriteMode, BindingMode: normalized.BindingMode, Page: normalized.Page, PageSize: normalized.PageSize,
	})
	if err != nil {
		return nil, err
	}
	policies := make([]StoragePolicy, 0, len(records))
	summary := StoragePolicySummary{}
	for _, record := range records {
		policy := toStoragePolicy(record)
		policy.Estimate, _ = s.estimateRecord(ctx, projectID, record)
		policies = append(policies, policy)
		if policy.Status == "enabled" {
			summary.EnabledCount++
		}
		if policy.Status == "error" {
			summary.ErrorCount++
		}
		summary.TotalBindingCount += policy.BindingCount
		summary.EstimatedRowsPerDay += policy.Estimate.RowsPerDay
		summary.EstimatedEventsSecond += policy.Estimate.EventsPerSecond
	}
	page, pageSize := normalizePageAndSize(filter.Page, filter.PageSize, defaultStoragePolicyPageSize, 100)
	return &StoragePolicyListResult{
		Policies:   policies,
		Pagination: StoragePolicyPagination{Page: page, PageSize: pageSize, Total: total, TotalPages: totalPages(total, pageSize)},
		Summary:    summary,
	}, nil
}

func (s *StoragePolicyService) Get(ctx context.Context, claims *auth.Claims, projectID, policyID string) (*StoragePolicyDetail, error) {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return nil, err
	}
	if err := validateStoragePolicyID(policyID); err != nil {
		return nil, err
	}
	record, err := s.repository.GetPolicyByProjectAndID(ctx, projectID, policyID)
	if err != nil {
		return nil, err
	}
	bindings, err := s.repository.ListBindings(ctx, projectID, policyID)
	if err != nil {
		return nil, err
	}
	policy := toStoragePolicy(*record)
	policy.Estimate, _ = s.estimateRecord(ctx, projectID, *record)
	return &StoragePolicyDetail{StoragePolicy: policy, Bindings: toStoragePolicyBindings(bindings)}, nil
}

func (s *StoragePolicyService) Create(ctx context.Context, claims *auth.Claims, projectID string, input CreateStoragePolicyInput) (*StoragePolicyDetail, error) {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return nil, err
	}
	normalized, err := s.normalizeInput(ctx, projectID, input)
	if err != nil {
		return nil, err
	}
	record, err := s.repository.CreatePolicy(ctx, toCreateStoragePolicyParams(projectID, claims.UserID, normalized))
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, claims, projectID, record.ID)
}

func (s *StoragePolicyService) Update(ctx context.Context, claims *auth.Claims, projectID, policyID string, input UpdateStoragePolicyInput) (*StoragePolicyDetail, error) {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return nil, err
	}
	if err := validateStoragePolicyID(policyID); err != nil {
		return nil, err
	}
	if _, err := s.repository.GetPolicyByProjectAndID(ctx, projectID, policyID); err != nil {
		return nil, err
	}
	normalized, err := s.normalizeInput(ctx, projectID, input.CreateStoragePolicyInput)
	if err != nil {
		return nil, err
	}
	record, err := s.repository.UpdatePolicy(ctx, toUpdateStoragePolicyParams(projectID, policyID, claims.UserID, normalized))
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, claims, projectID, record.ID)
}

func (s *StoragePolicyService) Delete(ctx context.Context, claims *auth.Claims, projectID, policyID string) error {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return err
	}
	if err := validateStoragePolicyID(policyID); err != nil {
		return err
	}
	return s.repository.DeletePolicy(ctx, projectID, policyID)
}

func (s *StoragePolicyService) Targets(ctx context.Context, claims *auth.Claims, projectID string) (*StorageTargetsResult, error) {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return nil, err
	}
	records, err := s.repository.ListTargets(ctx, projectID)
	if err != nil {
		return nil, err
	}
	targets := make([]StorageTarget, 0, len(records))
	for _, record := range records {
		targets = append(targets, toStorageTarget(record))
	}
	return &StorageTargetsResult{Targets: targets}, nil
}

func (s *StoragePolicyService) Estimate(ctx context.Context, claims *auth.Claims, projectID string, input CreateStoragePolicyInput) (*StoragePolicyEstimate, error) {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return nil, err
	}
	normalized, err := s.normalizeInput(ctx, projectID, input)
	if err != nil {
		return nil, err
	}
	return &normalized.Estimate, nil
}

func (s *StoragePolicyService) Coverage(ctx context.Context, claims *auth.Claims, projectID, datapointID, datapointPath string) (*StoragePolicyCoverageResult, error) {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return nil, err
	}
	datapointID = strings.TrimSpace(datapointID)
	datapointPath = strings.TrimSpace(datapointPath)
	if datapointID == "" && datapointPath == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "datapointId 或 path 至少需要一个")
	}
	if datapointID != "" {
		if err := validateStoragePolicyID(datapointID); err != nil {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "datapointId 格式无效")
		}
	}
	count, err := s.repository.CountPoliciesForDataPoint(ctx, projectID, datapointID, datapointPath)
	if err != nil {
		return nil, err
	}
	return &StoragePolicyCoverageResult{DatapointID: datapointID, DatapointPath: datapointPath, MatchedPolicyCount: count}, nil
}

func (s *StoragePolicyService) validateReadAccess(claims *auth.Claims, projectID string) error {
	if s == nil || s.repository == nil || s.datapoints == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "storage policy 服务未初始化")
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

func (s *StoragePolicyService) validateWriteAccess(claims *auth.Claims, projectID string) error {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return err
	}
	return validateUserID(claims.UserID)
}

func (s *StoragePolicyService) normalizeInput(ctx context.Context, projectID string, input CreateStoragePolicyInput) (normalizedStoragePolicyInput, error) {
	name, err := normalizeNamedField(input.Name, "name", 100)
	if err != nil {
		return normalizedStoragePolicyInput{}, err
	}
	targetConnectionID := strings.TrimSpace(input.TargetConnectionID)
	if err := validateStoragePolicyID(targetConnectionID); err != nil {
		return normalizedStoragePolicyInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "targetConnectionId 格式无效")
	}
	target, err := s.repository.GetTarget(ctx, projectID, targetConnectionID)
	if err != nil {
		return normalizedStoragePolicyInput{}, err
	}
	targetCapability := strings.TrimSpace(input.TargetCapability)
	if targetCapability == "" {
		targetCapability = firstStorageCapability(target.Capabilities)
	}
	if !stringInSlice(targetCapability, target.Capabilities) {
		return normalizedStoragePolicyInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "目标接入源不具备所选历史写入能力")
	}
	bindingMode, err := normalizeStorageEnum(input.BindingMode, "bindingMode", storageBindingModes)
	if err != nil {
		return normalizedStoragePolicyInput{}, err
	}
	bindingFilter := normalizeStorageDataPointFilter(input.BindingFilter)
	datapointIDs, err := normalizeStorageDatapointIDs(input.DatapointIDs)
	if err != nil {
		return normalizedStoragePolicyInput{}, err
	}
	if bindingMode == "static" {
		if len(datapointIDs) == 0 {
			return normalizedStoragePolicyInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "静态绑定至少需要选择一个数据点")
		}
		if _, err := s.datapoints.GetByProjectAndIDs(ctx, projectID, datapointIDs); err != nil {
			return normalizedStoragePolicyInput{}, err
		}
	}
	writeMode, err := normalizeStorageEnum(input.WriteMode, "writeMode", storageWriteModes)
	if err != nil {
		return normalizedStoragePolicyInput{}, err
	}
	targetTableMode := strings.TrimSpace(input.TargetTableMode)
	if targetTableMode == "" {
		targetTableMode = "auto_create"
	}
	if _, ok := storageTargetTableModes[targetTableMode]; !ok {
		return normalizedStoragePolicyInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "targetTableMode 不受支持")
	}
	status := "enabled"
	if input.Status != nil {
		status, err = normalizeStorageEnum(*input.Status, "status", storagePolicyStatuses)
		if err != nil {
			return normalizedStoragePolicyInput{}, err
		}
	}
	includeQualities, err := normalizeStorageQualities(input.IncludeQualities)
	if err != nil {
		return normalizedStoragePolicyInput{}, err
	}
	retentionDays := input.RetentionDays
	if retentionDays <= 0 {
		return normalizedStoragePolicyInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "retentionDays 必须大于 0")
	}
	if err := validateStorageWriteModeFields(writeMode, input); err != nil {
		return normalizedStoragePolicyInput{}, err
	}
	estimate, err := s.estimateInput(ctx, projectID, bindingMode, bindingFilter, datapointIDs, writeMode, input.SnapshotIntervalMS, retentionDays, includeQualities)
	if err != nil {
		return normalizedStoragePolicyInput{}, err
	}
	diagnostics := buildStoragePolicyDiagnostics(*target, bindingMode, estimate, input.TargetTableMode, status)
	return normalizedStoragePolicyInput{
		Name: name, Description: trimOptionalString(input.Description), TargetConnectionID: targetConnectionID,
		TargetCapability: targetCapability, BindingMode: bindingMode, BindingFilter: bindingFilter, DatapointIDs: datapointIDs,
		WriteMode: writeMode, MinIntervalMS: input.MinIntervalMS, Deadband: input.Deadband, SnapshotIntervalMS: input.SnapshotIntervalMS,
		IncludeQualities: includeQualities, RetentionDays: retentionDays, TargetTableMode: targetTableMode,
		TargetTableConfig: cloneMap(input.TargetTableConfig), Status: status, Diagnostics: diagnostics, Estimate: estimate,
	}, nil
}

func (s *StoragePolicyService) estimateRecord(ctx context.Context, projectID string, record repository.StoragePolicyRecord) (StoragePolicyEstimate, error) {
	filter := storageFilterFromMap(record.BindingFilter)
	qualities := stringsFromAnySlice(record.IncludeQualities)
	datapointIDs := []string(nil)
	if record.BindingMode == "static" && record.BindingCount > 0 {
		// 列表查询只返回静态绑定数量；这里构造等长切片用于复用统一估算公式。
		datapointIDs = make([]string, record.BindingCount)
	}
	return s.estimateInput(ctx, projectID, record.BindingMode, filter, datapointIDs, record.WriteMode, record.SnapshotIntervalMS, record.RetentionDays, qualities)
}

func (s *StoragePolicyService) estimateInput(ctx context.Context, projectID, bindingMode string, filter StorageDataPointFilter, datapointIDs []string, writeMode string, snapshotIntervalMS *int, retentionDays int, includeQualities []string) (StoragePolicyEstimate, error) {
	matched := len(datapointIDs)
	if bindingMode == "dynamic" {
		count, err := s.repository.CountDataPointsByFilter(ctx, projectID, toRepositoryStorageDataPointFilter(filter))
		if err != nil {
			return StoragePolicyEstimate{}, err
		}
		matched = count
	}
	eventsPerSecond := float64(matched) * 1000 / defaultStorageSampleInterval
	assumption := "按每个数据点 1s 采样估算"
	switch writeMode {
	case "on_change":
		eventsPerSecond = eventsPerSecond * defaultStorageChangeRate
		assumption = "on_change 按 30% 保守变化率估算"
	case "periodic_snapshot":
		interval := defaultStorageSampleInterval
		if snapshotIntervalMS != nil && *snapshotIntervalMS > 0 {
			interval = *snapshotIntervalMS
		}
		eventsPerSecond = float64(matched) * 1000 / float64(interval)
		assumption = fmt.Sprintf("periodic_snapshot 按 %dms 快照周期估算", interval)
	}
	if !stringInSlice("Bad", includeQualities) {
		eventsPerSecond = eventsPerSecond * (1 - defaultStorageBadQualityRate)
		assumption += "，Bad 质量默认不写历史值"
	}
	rowsPerDay := int64(eventsPerSecond * 86400)
	return StoragePolicyEstimate{
		MatchedDataPointCount: matched,
		EventsPerSecond:       eventsPerSecond,
		RowsPerDay:            rowsPerDay,
		RowsByRetention:       rowsPerDay * int64(retentionDays),
		Assumption:            assumption,
	}, nil
}

func normalizeStoragePolicyListFilter(filter StoragePolicyListFilter) (StoragePolicyListFilter, error) {
	status := strings.TrimSpace(filter.Status)
	if status != "" {
		if _, ok := storagePolicyStatuses[status]; !ok {
			return StoragePolicyListFilter{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "status 不受支持")
		}
	}
	writeMode := strings.TrimSpace(filter.WriteMode)
	if writeMode != "" {
		if _, ok := storageWriteModes[writeMode]; !ok {
			return StoragePolicyListFilter{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "writeMode 不受支持")
		}
	}
	bindingMode := strings.TrimSpace(filter.BindingMode)
	if bindingMode != "" {
		if _, ok := storageBindingModes[bindingMode]; !ok {
			return StoragePolicyListFilter{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "bindingMode 不受支持")
		}
	}
	return StoragePolicyListFilter{
		Search: strings.TrimSpace(filter.Search), Status: status, TargetConnectionID: strings.TrimSpace(filter.TargetConnectionID),
		WriteMode: writeMode, BindingMode: bindingMode, Page: filter.Page, PageSize: filter.PageSize,
	}, nil
}

func normalizeStorageEnum(value, field string, allowed map[string]struct{}) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, field+" 不能为空")
	}
	if _, ok := allowed[value]; !ok {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, field+" 不受支持")
	}
	return value, nil
}

func normalizeStorageDataPointFilter(filter StorageDataPointFilter) StorageDataPointFilter {
	return StorageDataPointFilter{
		Type: strings.TrimSpace(filter.Type), Status: strings.TrimSpace(filter.Status), Search: strings.TrimSpace(filter.Search),
		AccessSourceID: strings.TrimSpace(filter.AccessSourceID), SourceID: strings.TrimSpace(filter.SourceID),
		SourceIDs: uniqueTrimmedStrings(filter.SourceIDs), Tags: uniqueTrimmedStrings(filter.Tags),
	}
}

func normalizeStorageDatapointIDs(values []string) ([]string, error) {
	result := uniqueTrimmedStrings(values)
	for _, value := range result {
		if _, err := uuid.Parse(value); err != nil {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "datapointIds 包含无效 ID")
		}
	}
	return result, nil
}

func normalizeStorageQualities(values []string) ([]string, error) {
	if len(values) == 0 {
		return []string{"Good", "Uncertain"}, nil
	}
	result := uniqueTrimmedStrings(values)
	for _, value := range result {
		if _, ok := storageQualities[value]; !ok {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "includeQualities 包含不支持的质量")
		}
	}
	if len(result) == 0 {
		return []string{"Good", "Uncertain"}, nil
	}
	return result, nil
}

func validateStorageWriteModeFields(writeMode string, input CreateStoragePolicyInput) error {
	if input.MinIntervalMS != nil && *input.MinIntervalMS < 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "minIntervalMs 不能小于 0")
	}
	if input.Deadband != nil && *input.Deadband < 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "deadband 不能小于 0")
	}
	if writeMode == "periodic_snapshot" {
		if input.SnapshotIntervalMS == nil || *input.SnapshotIntervalMS <= 0 {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "periodic_snapshot 需要 snapshotIntervalMs")
		}
	}
	return nil
}

func buildStoragePolicyDiagnostics(target repository.StorageTargetRecord, bindingMode string, estimate StoragePolicyEstimate, targetTableMode string, status string) []StoragePolicyDiagnostic {
	result := make([]StoragePolicyDiagnostic, 0)
	if target.Status != "connected" {
		result = append(result, StoragePolicyDiagnostic{Severity: "warning", Type: "storage", Message: "存储目标当前不是 connected 状态", Suggest: "发布前执行目标连接校验或试写"})
	}
	if estimate.MatchedDataPointCount == 0 {
		result = append(result, StoragePolicyDiagnostic{Severity: "warning", Type: "storage", Message: "当前策略未命中任何数据点", Suggest: "检查静态绑定或动态筛选条件"})
	}
	if estimate.RowsPerDay > 10_000_000 {
		result = append(result, StoragePolicyDiagnostic{Severity: "warning", Type: "performance", Message: "预计日写入量较高", Suggest: "调整写入模式、周期或拆分目标库"})
	}
	if targetTableMode == "existing_mapping" {
		result = append(result, StoragePolicyDiagnostic{Severity: "info", Type: "storage", Message: "使用已有表映射", Suggest: "发布前验证字段映射和写入权限"})
	}
	if bindingMode == "dynamic" && status == "enabled" {
		result = append(result, StoragePolicyDiagnostic{Severity: "info", Type: "storage", Message: "动态规则会自动纳入后续满足条件的数据点"})
	}
	return result
}

func toStoragePolicy(record repository.StoragePolicyRecord) StoragePolicy {
	qualities := stringsFromAnySlice(record.IncludeQualities)
	diagnostics := diagnosticsFromAnySlice(record.Diagnostics)
	return StoragePolicy{
		ID: record.ID, ProjectID: record.ProjectID, Name: record.Name, Description: cloneOptionalString(record.Description),
		Target:           StorageTarget{ID: record.TargetConnectionID, Name: record.TargetName, Type: record.TargetType, Status: record.TargetStatus, Capabilities: capabilitiesForStorageTargetType(record.TargetType)},
		TargetCapability: record.TargetCapability, BindingMode: record.BindingMode, BindingFilter: storageFilterFromMap(record.BindingFilter),
		WriteMode: record.WriteMode, MinIntervalMS: cloneOptionalInt(record.MinIntervalMS), Deadband: cloneOptionalFloat64(record.Deadband),
		SnapshotIntervalMS: cloneOptionalInt(record.SnapshotIntervalMS), IncludeQualities: qualities, RetentionDays: record.RetentionDays,
		TargetTableMode: record.TargetTableMode, TargetTableConfig: cloneMap(record.TargetTableConfig), Status: record.Status,
		Diagnostics: diagnostics, BindingCount: record.BindingCount, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt,
	}
}

func toStorageTarget(record repository.StorageTargetRecord) StorageTarget {
	return StorageTarget{ID: record.ID, Name: record.Name, Type: record.Type, Category: record.Category, Status: record.Status, Capabilities: append([]string{}, record.Capabilities...), CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}

func toStoragePolicyBindings(records []repository.StoragePolicyBindingRecord) []StoragePolicyBinding {
	result := make([]StoragePolicyBinding, 0, len(records))
	for _, record := range records {
		result = append(result, StoragePolicyBinding{
			ID: record.ID, DatapointID: record.DatapointID, DatapointPath: record.DatapointPath,
			DatapointName: record.DatapointName, DataType: record.DataType, Status: record.Status, CreatedAt: record.CreatedAt,
		})
	}
	return result
}

func toCreateStoragePolicyParams(projectID, userID string, input normalizedStoragePolicyInput) repository.CreateStoragePolicyParams {
	return repository.CreateStoragePolicyParams{
		ProjectID: projectID, UserID: userID, Name: input.Name, Description: input.Description, TargetConnectionID: input.TargetConnectionID,
		TargetCapability: input.TargetCapability, BindingMode: input.BindingMode, BindingFilter: storageFilterToMap(input.BindingFilter),
		WriteMode: input.WriteMode, MinIntervalMS: input.MinIntervalMS, Deadband: input.Deadband, SnapshotIntervalMS: input.SnapshotIntervalMS,
		IncludeQualities: stringsToAnySlice(input.IncludeQualities), RetentionDays: input.RetentionDays, TargetTableMode: input.TargetTableMode,
		TargetTableConfig: input.TargetTableConfig, Status: input.Status, Diagnostics: diagnosticsToAnySlice(input.Diagnostics), DatapointIDs: input.DatapointIDs,
	}
}

func toUpdateStoragePolicyParams(projectID, policyID, userID string, input normalizedStoragePolicyInput) repository.UpdateStoragePolicyParams {
	return repository.UpdateStoragePolicyParams{
		ID: policyID, ProjectID: projectID, UserID: userID, Name: input.Name, Description: input.Description, TargetConnectionID: input.TargetConnectionID,
		TargetCapability: input.TargetCapability, BindingMode: input.BindingMode, BindingFilter: storageFilterToMap(input.BindingFilter),
		WriteMode: input.WriteMode, MinIntervalMS: input.MinIntervalMS, Deadband: input.Deadband, SnapshotIntervalMS: input.SnapshotIntervalMS,
		IncludeQualities: stringsToAnySlice(input.IncludeQualities), RetentionDays: input.RetentionDays, TargetTableMode: input.TargetTableMode,
		TargetTableConfig: input.TargetTableConfig, Status: input.Status, Diagnostics: diagnosticsToAnySlice(input.Diagnostics), DatapointIDs: input.DatapointIDs,
	}
}

func toRepositoryStorageDataPointFilter(filter StorageDataPointFilter) repository.StoragePolicyDataPointFilter {
	return repository.StoragePolicyDataPointFilter{
		Type: filter.Type, Status: filter.Status, Search: filter.Search, AccessSourceID: filter.AccessSourceID,
		SourceID: filter.SourceID, SourceIDs: filter.SourceIDs, Tags: filter.Tags,
	}
}

func storageFilterToMap(filter StorageDataPointFilter) map[string]any {
	result := map[string]any{}
	if filter.Type != "" {
		result["type"] = filter.Type
	}
	if filter.Status != "" {
		result["status"] = filter.Status
	}
	if filter.Search != "" {
		result["search"] = filter.Search
	}
	if filter.AccessSourceID != "" {
		result["accessSourceId"] = filter.AccessSourceID
	}
	if filter.SourceID != "" {
		result["sourceId"] = filter.SourceID
	}
	if len(filter.SourceIDs) > 0 {
		result["sourceIds"] = filter.SourceIDs
	}
	if len(filter.Tags) > 0 {
		result["tags"] = filter.Tags
	}
	return result
}

func storageFilterFromMap(value map[string]any) StorageDataPointFilter {
	return normalizeStorageDataPointFilter(StorageDataPointFilter{
		Type: stringFromAny(value["type"]), Status: stringFromAny(value["status"]), Search: stringFromAny(value["search"]),
		AccessSourceID: stringFromAny(value["accessSourceId"]), SourceID: stringFromAny(value["sourceId"]),
		SourceIDs: stringsFromAny(value["sourceIds"]), Tags: stringsFromAny(value["tags"]),
	})
}

func diagnosticsToAnySlice(values []StoragePolicyDiagnostic) []any {
	result := make([]any, 0, len(values))
	for _, value := range values {
		result = append(result, map[string]any{"severity": value.Severity, "type": value.Type, "message": value.Message, "suggest": value.Suggest})
	}
	return result
}

func diagnosticsFromAnySlice(values []any) []StoragePolicyDiagnostic {
	result := make([]StoragePolicyDiagnostic, 0, len(values))
	for _, value := range values {
		if item, ok := value.(map[string]any); ok {
			result = append(result, StoragePolicyDiagnostic{Severity: stringFromAny(item["severity"]), Type: stringFromAny(item["type"]), Message: stringFromAny(item["message"]), Suggest: stringFromAny(item["suggest"])})
		}
	}
	return result
}

func stringsFromAnySlice(values []any) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		text := strings.TrimSpace(fmt.Sprintf("%v", value))
		if text != "" {
			result = append(result, text)
		}
	}
	return result
}

func stringsFromAny(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		return stringsFromAnySlice(typed)
	default:
		return []string{}
	}
}

func capabilitiesForStorageTargetType(targetType string) []string {
	switch strings.TrimSpace(targetType) {
	case "builtin.timeseries", "tdengine":
		return []string{"timeseriesAppend"}
	case "relational", "builtin.relation":
		return []string{"relationalAppend"}
	default:
		return []string{}
	}
}

func firstStorageCapability(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func stringInSlice(value string, values []string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func uniqueTrimmedStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
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
	return result
}

func validateStoragePolicyID(id string) error {
	if _, err := uuid.Parse(strings.TrimSpace(id)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "id 格式无效", err)
	}
	return nil
}
