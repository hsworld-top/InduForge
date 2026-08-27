package service

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/indu-forge/data_service/internal/auth"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

const (
	defaultHistoryStoragePageSize     = 20
	defaultHistoryStorageMaxSilenceMS = int64(60 * 60 * 1000)
)

var historyStorageWriteModes = map[string]struct{}{
	"every_sample": {}, "interval_latest": {}, "on_change": {}, "periodic_snapshot": {},
}

type HistoryStorageService struct {
	repository *repository.HistoryStorageRepository
}

type HistoryStorageScopeRef struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	Name       string `json:"name,omitempty"`
	SourceType string `json:"sourceType,omitempty"`
}

type HistoryStorageTarget struct {
	ConnectionID   string `json:"connectionId"`
	ConnectionName string `json:"connectionName"`
	ConnectionType string `json:"connectionType"`
	LastTestStatus string `json:"lastTestStatus"`
	IsPrimary      bool   `json:"isPrimary"`
	SortOrder      int    `json:"sortOrder"`
	RetentionDays  *int64 `json:"retentionDays"`
}

type HistoryStorageConfiguration struct {
	WriteMode       string                 `json:"writeMode"`
	IntervalMS      *int64                 `json:"intervalMs"`
	Deadband        *float64               `json:"deadband"`
	MaxSilenceMS    *int64                 `json:"maxSilenceMs"`
	OfflineBehavior string                 `json:"offlineBehavior"`
	Targets         []HistoryStorageTarget `json:"targets"`
}

type HistoryStorageSourceItem struct {
	Scope              HistoryStorageScopeRef `json:"scope"`
	DatapointCount     int                    `json:"datapointCount"`
	PointOverrideCount int                    `json:"pointOverrideCount"`
	HistoryState       string                 `json:"historyState"`
	WriteMode          *string                `json:"writeMode"`
	TargetCount        int                    `json:"targetCount"`
	PrimaryTargetName  *string                `json:"primaryTargetName"`
	PrimaryTargetType  *string                `json:"primaryTargetType"`
	RetentionDays      *int64                 `json:"retentionDays"`
}

type HistoryStorageSourceListFilter struct {
	Search       string
	ScopeType    string
	HistoryState string
	Page         int
	PageSize     int
}

type HistoryStorageSourceListResult struct {
	List       []HistoryStorageSourceItem `json:"list"`
	Pagination HistoryStoragePagination   `json:"pagination"`
}

type HistoryStoragePagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

type HistoryStorageSourceDetail struct {
	Scope              HistoryStorageScopeRef       `json:"scope"`
	Behavior           string                       `json:"behavior"`
	Configuration      *HistoryStorageConfiguration `json:"configuration"`
	PointOverrideCount int                          `json:"pointOverrideCount"`
}

type HistoryStorageDatapointDetail struct {
	DatapointID      string                       `json:"datapointId"`
	DatapointName    string                       `json:"datapointName"`
	DatapointPath    string                       `json:"datapointPath"`
	DataType         string                       `json:"dataType"`
	Behavior         string                       `json:"behavior"`
	EffectiveEnabled bool                         `json:"effectiveEnabled"`
	Source           *HistoryStorageScopeRef      `json:"source"`
	Configuration    *HistoryStorageConfiguration `json:"configuration"`
}

type HistoryStorageTargetOption struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Type           string    `json:"type"`
	LastTestStatus string    `json:"lastTestStatus"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type HistoryStorageTargetsResult struct {
	Targets []HistoryStorageTargetOption `json:"targets"`
}

type HistoryStorageTargetInput struct {
	ConnectionID  string `json:"connectionId"`
	IsPrimary     bool   `json:"isPrimary"`
	SortOrder     int    `json:"sortOrder"`
	RetentionDays *int64 `json:"retentionDays"`
}

type HistoryStorageConfigurationInput struct {
	WriteMode       string                      `json:"writeMode"`
	IntervalMS      *int64                      `json:"intervalMs"`
	Deadband        *float64                    `json:"deadband"`
	MaxSilenceMS    *int64                      `json:"maxSilenceMs"`
	OfflineBehavior string                      `json:"offlineBehavior"`
	Targets         []HistoryStorageTargetInput `json:"targets"`
}

type SaveHistoryStorageInput struct {
	Behavior      string
	Configuration *HistoryStorageConfigurationInput
}

type HistoryStorageBulkSelection struct {
	Mode                string
	DatapointIDs        []string
	ExcludeDatapointIDs []string
	Filters             DataPointListFilter
}

type BatchSaveHistoryStorageInput struct {
	Selection     HistoryStorageBulkSelection
	Behavior      string
	Configuration *HistoryStorageConfigurationInput
}

type HistoryStorageBatchResult struct {
	MatchedCount   int `json:"matchedCount"`
	ChangedCount   int `json:"changedCount"`
	UnchangedCount int `json:"unchangedCount"`
}

func NewHistoryStorageService(repo *repository.HistoryStorageRepository) *HistoryStorageService {
	return &HistoryStorageService{repository: repo}
}

func (s *HistoryStorageService) ListSources(ctx context.Context, claims *auth.Claims, projectID string, filter HistoryStorageSourceListFilter) (*HistoryStorageSourceListResult, error) {
	if err := s.validateAccess(claims, projectID, false); err != nil {
		return nil, err
	}
	filter.Search = strings.TrimSpace(filter.Search)
	filter.ScopeType = strings.TrimSpace(filter.ScopeType)
	filter.HistoryState = strings.TrimSpace(filter.HistoryState)
	if filter.ScopeType != "" && filter.ScopeType != "access_source" && filter.ScopeType != "collector_connection" && filter.ScopeType != "compute_unit" {
		return nil, badHistoryStorageRequest("scopeType 不受支持")
	}
	if filter.HistoryState != "" && filter.HistoryState != "enabled" && filter.HistoryState != "disabled" {
		return nil, badHistoryStorageRequest("historyState 不受支持")
	}
	page, pageSize := normalizePageAndSize(filter.Page, filter.PageSize, defaultHistoryStoragePageSize, 100)
	records, total, err := s.repository.ListSources(ctx, projectID, repository.HistoryStorageSourceListFilter{
		Search: filter.Search, ScopeType: filter.ScopeType, HistoryState: filter.HistoryState, Page: page, PageSize: pageSize,
	})
	if err != nil {
		return nil, err
	}
	items := make([]HistoryStorageSourceItem, 0, len(records))
	for _, record := range records {
		state := "disabled"
		if record.IsEnabled {
			state = "enabled"
		}
		items = append(items, HistoryStorageSourceItem{
			Scope:          HistoryStorageScopeRef{Type: record.ScopeType, ID: record.ScopeID, Name: record.Name, SourceType: record.SourceType},
			DatapointCount: record.DatapointCount, PointOverrideCount: record.PointOverrideCount, HistoryState: state,
			WriteMode: record.WriteMode, TargetCount: record.TargetCount, PrimaryTargetName: record.PrimaryTargetName,
			PrimaryTargetType: record.PrimaryTargetType, RetentionDays: record.RetentionDays,
		})
	}
	return &HistoryStorageSourceListResult{List: items, Pagination: HistoryStoragePagination{Page: page, PageSize: pageSize, Total: total, TotalPages: totalPages(total, pageSize)}}, nil
}

func (s *HistoryStorageService) GetSource(ctx context.Context, claims *auth.Claims, projectID string, scope HistoryStorageScopeRef) (*HistoryStorageSourceDetail, error) {
	if err := s.validateAccess(claims, projectID, false); err != nil {
		return nil, err
	}
	repoScope, err := normalizeHistoryStorageScope(scope.Type, scope.ID, false)
	if err != nil {
		return nil, err
	}
	exists, err := s.repository.ScopeExists(ctx, projectID, repoScope)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, historyStorageNotFound("历史存储来源不存在")
	}
	config, err := s.repository.GetConfigByScope(ctx, projectID, repoScope)
	if err != nil {
		return nil, err
	}
	count, err := s.repository.CountPointOverridesByScope(ctx, projectID, repoScope)
	if err != nil {
		return nil, err
	}
	detail := &HistoryStorageSourceDetail{Scope: scope, Behavior: "off", PointOverrideCount: count}
	if config != nil {
		detail.Configuration = toHistoryStorageConfiguration(config)
		if config.IsEnabled {
			detail.Behavior = "custom"
		}
	}
	return detail, nil
}

func (s *HistoryStorageService) SaveSource(ctx context.Context, claims *auth.Claims, projectID string, scope HistoryStorageScopeRef, input SaveHistoryStorageInput) (*HistoryStorageSourceDetail, error) {
	if err := s.validateAccess(claims, projectID, true); err != nil {
		return nil, err
	}
	repoScope, err := normalizeHistoryStorageScope(scope.Type, scope.ID, false)
	if err != nil {
		return nil, err
	}
	exists, err := s.repository.ScopeExists(ctx, projectID, repoScope)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, historyStorageNotFound("历史存储来源不存在")
	}
	params, err := s.normalizeSaveInput(ctx, projectID, claims.UserID, repoScope, input, false)
	if err != nil {
		return nil, err
	}
	if _, err := s.repository.SaveConfig(ctx, params); err != nil {
		return nil, err
	}
	return s.GetSource(ctx, claims, projectID, scope)
}

func (s *HistoryStorageService) ListTargets(ctx context.Context, claims *auth.Claims, projectID string) (*HistoryStorageTargetsResult, error) {
	if err := s.validateAccess(claims, projectID, false); err != nil {
		return nil, err
	}
	records, err := s.repository.ListTargetOptions(ctx, projectID)
	if err != nil {
		return nil, err
	}
	result := make([]HistoryStorageTargetOption, 0, len(records))
	for _, record := range records {
		result = append(result, HistoryStorageTargetOption{ID: record.ID, Name: record.Name, Type: record.Type, LastTestStatus: record.LastTestStatus, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt})
	}
	return &HistoryStorageTargetsResult{Targets: result}, nil
}

func (s *HistoryStorageService) GetDatapoint(ctx context.Context, claims *auth.Claims, projectID, datapointID string) (*HistoryStorageDatapointDetail, error) {
	if err := s.validateAccess(claims, projectID, false); err != nil {
		return nil, err
	}
	if _, err := uuid.Parse(strings.TrimSpace(datapointID)); err != nil {
		return nil, badHistoryStorageRequest("datapointId 格式无效")
	}
	origin, err := s.repository.GetDataPointOrigin(ctx, projectID, datapointID)
	if err != nil {
		return nil, err
	}
	result := &HistoryStorageDatapointDetail{DatapointID: origin.DatapointID, DatapointName: origin.DatapointName, DatapointPath: origin.DatapointPath, DataType: origin.DataType, Behavior: "inherit"}
	if origin.ScopeID != "" {
		result.Source = &HistoryStorageScopeRef{Type: origin.ScopeType, ID: origin.ScopeID, Name: origin.ScopeName, SourceType: origin.ScopeSourceType}
	}
	pointConfig, err := s.repository.GetConfigByScope(ctx, projectID, repository.HistoryStorageScope{Type: "datapoint", ID: datapointID})
	if err != nil {
		return nil, err
	}
	if pointConfig != nil {
		result.Configuration = toHistoryStorageConfiguration(pointConfig)
		if pointConfig.IsEnabled {
			result.Behavior = "custom"
			result.EffectiveEnabled = true
		} else {
			result.Behavior = "off"
		}
		return result, nil
	}
	if origin.ScopeID == "" {
		return result, nil
	}
	sourceConfig, err := s.repository.GetConfigByScope(ctx, projectID, repository.HistoryStorageScope{Type: origin.ScopeType, ID: origin.ScopeID})
	if err != nil {
		return nil, err
	}
	if sourceConfig != nil && sourceConfig.IsEnabled {
		result.EffectiveEnabled = true
		result.Configuration = toHistoryStorageConfiguration(sourceConfig)
	}
	return result, nil
}

func (s *HistoryStorageService) SaveDatapoint(ctx context.Context, claims *auth.Claims, projectID, datapointID string, input SaveHistoryStorageInput) (*HistoryStorageDatapointDetail, error) {
	if err := s.validateAccess(claims, projectID, true); err != nil {
		return nil, err
	}
	scope, err := normalizeHistoryStorageScope("datapoint", datapointID, true)
	if err != nil {
		return nil, err
	}
	origin, err := s.repository.GetDataPointOrigin(ctx, projectID, datapointID)
	if err != nil {
		return nil, err
	}
	behavior := strings.TrimSpace(input.Behavior)
	if origin.Status == "invalid" && behavior == "custom" {
		return nil, badHistoryStorageRequest("失效数据点不能启用历史存储")
	}
	if behavior == "inherit" {
		if err := s.repository.DeleteConfigByScope(ctx, projectID, scope); err != nil {
			return nil, err
		}
		return s.GetDatapoint(ctx, claims, projectID, datapointID)
	}
	params, err := s.normalizeSaveInput(ctx, projectID, claims.UserID, scope, input, true)
	if err != nil {
		return nil, err
	}
	if _, err := s.repository.SaveConfig(ctx, params); err != nil {
		return nil, err
	}
	return s.GetDatapoint(ctx, claims, projectID, datapointID)
}

func (s *HistoryStorageService) BatchConfigure(ctx context.Context, claims *auth.Claims, projectID string, input BatchSaveHistoryStorageInput) (*HistoryStorageBatchResult, error) {
	if err := s.validateAccess(claims, projectID, true); err != nil {
		return nil, err
	}
	behavior := strings.TrimSpace(input.Behavior)
	if behavior != "inherit" && behavior != "off" && behavior != "custom" {
		return nil, badHistoryStorageRequest("behavior 不受支持")
	}
	selection := input.Selection
	var ids []string
	var filter *repository.DataPointListFilter
	switch strings.TrimSpace(selection.Mode) {
	case "ids":
		ids = uniqueHistoryStorageStrings(selection.DatapointIDs)
		if len(ids) == 0 {
			return nil, badHistoryStorageRequest("datapointIds 不能为空")
		}
		for _, id := range ids {
			if _, err := uuid.Parse(id); err != nil {
				return nil, badHistoryStorageRequest("datapointIds 包含无效 ID")
			}
		}
	case "filtered":
		normalized := repository.DataPointListFilter{Type: selection.Filters.Type, Status: selection.Filters.Status, Search: selection.Filters.Search, AccessSourceID: selection.Filters.AccessSourceID, SourceID: selection.Filters.SourceID, SourceIDs: selection.Filters.SourceIDs, Tags: selection.Filters.Tags}
		filter = &normalized
	default:
		return nil, badHistoryStorageRequest("selection.mode 不受支持")
	}
	resolved, err := s.repository.ResolveDataPointIDs(ctx, projectID, ids, filter)
	if err != nil {
		return nil, err
	}
	excluded := make(map[string]struct{}, len(selection.ExcludeDatapointIDs))
	for _, id := range uniqueHistoryStorageStrings(selection.ExcludeDatapointIDs) {
		excluded[id] = struct{}{}
	}
	ids = ids[:0]
	for _, id := range resolved {
		if _, skip := excluded[id]; !skip {
			ids = append(ids, id)
		}
	}
	var normalized *repository.SaveHistoryStorageConfigParams
	if behavior == "custom" {
		params, err := s.normalizeSaveInput(ctx, projectID, claims.UserID, repository.HistoryStorageScope{Type: "datapoint", ID: uuid.NewString()}, SaveHistoryStorageInput{Behavior: behavior, Configuration: input.Configuration}, true)
		if err != nil {
			return nil, err
		}
		normalized = &params
	}
	updated, err := s.repository.SaveDataPointConfigsBatch(ctx, projectID, claims.UserID, ids, behavior, normalized)
	if err != nil {
		return nil, err
	}
	unchanged := len(ids) - updated
	if unchanged < 0 {
		unchanged = 0
	}
	return &HistoryStorageBatchResult{MatchedCount: len(ids), ChangedCount: updated, UnchangedCount: unchanged}, nil
}

func (s *HistoryStorageService) normalizeSaveInput(ctx context.Context, projectID, userID string, scope repository.HistoryStorageScope, input SaveHistoryStorageInput, allowInherit bool) (repository.SaveHistoryStorageConfigParams, error) {
	behavior := strings.TrimSpace(input.Behavior)
	if behavior == "inherit" && !allowInherit {
		return repository.SaveHistoryStorageConfigParams{}, badHistoryStorageRequest("来源配置不支持 inherit")
	}
	if behavior != "off" && behavior != "custom" {
		return repository.SaveHistoryStorageConfigParams{}, badHistoryStorageRequest("behavior 不受支持")
	}
	params := repository.SaveHistoryStorageConfigParams{ProjectID: projectID, UserID: userID, Scope: scope, IsEnabled: false, WriteMode: "on_change", Deadband: float64Ptr(0), MaxSilenceMS: int64Ptr(defaultHistoryStorageMaxSilenceMS), OfflineBehavior: "store_stale"}
	if behavior == "off" {
		current, err := s.repository.GetConfigByScope(ctx, projectID, scope)
		if err != nil {
			return params, err
		}
		if current != nil {
			params.WriteMode = current.WriteMode
			params.IntervalMS = current.IntervalMS
			params.Deadband = current.Deadband
			params.MaxSilenceMS = current.MaxSilenceMS
			params.OfflineBehavior = current.OfflineBehavior
		}
		return params, nil
	}
	if input.Configuration == nil {
		return params, badHistoryStorageRequest("configuration 不能为空")
	}
	config := input.Configuration
	mode, offline, err := validateHistoryStorageConfigurationFields(config)
	if err != nil {
		return params, err
	}
	targets, err := s.normalizeTargets(ctx, projectID, config.Targets)
	if err != nil {
		return params, err
	}
	params.IsEnabled = true
	params.WriteMode = mode
	params.IntervalMS = config.IntervalMS
	params.Deadband = config.Deadband
	params.MaxSilenceMS = config.MaxSilenceMS
	params.OfflineBehavior = offline
	params.Targets = targets
	params.ReplaceTargets = true
	return params, nil
}

func validateHistoryStorageConfigurationFields(config *HistoryStorageConfigurationInput) (string, string, error) {
	mode := strings.TrimSpace(config.WriteMode)
	if _, ok := historyStorageWriteModes[mode]; !ok {
		return "", "", badHistoryStorageRequest("writeMode 不受支持")
	}
	offline := strings.TrimSpace(config.OfflineBehavior)
	if offline == "" {
		offline = "store_stale"
	}
	if offline != "store_stale" && offline != "skip" {
		return "", "", badHistoryStorageRequest("offlineBehavior 不受支持")
	}
	if config.Deadband != nil && *config.Deadband < 0 {
		return "", "", badHistoryStorageRequest("deadband 不能小于 0")
	}
	if config.MaxSilenceMS != nil && *config.MaxSilenceMS <= 0 {
		return "", "", badHistoryStorageRequest("maxSilenceMs 必须大于 0")
	}
	if (mode == "interval_latest" || mode == "periodic_snapshot") && (config.IntervalMS == nil || *config.IntervalMS <= 0) {
		return "", "", badHistoryStorageRequest("当前保存方式需要大于 0 的 intervalMs")
	}
	if mode != "interval_latest" && mode != "periodic_snapshot" && config.IntervalMS != nil {
		return "", "", badHistoryStorageRequest("当前保存方式不接受 intervalMs")
	}
	if mode != "on_change" && (config.Deadband != nil || config.MaxSilenceMS != nil) {
		return "", "", badHistoryStorageRequest("只有变化时保存支持 deadband 和 maxSilenceMs")
	}
	if mode != "periodic_snapshot" && config.OfflineBehavior != "" && config.OfflineBehavior != "store_stale" {
		return "", "", badHistoryStorageRequest("只有按周期保存支持修改 offlineBehavior")
	}
	return mode, offline, nil
}

func (s *HistoryStorageService) normalizeTargets(ctx context.Context, projectID string, values []HistoryStorageTargetInput) ([]repository.SaveHistoryStorageTargetParams, error) {
	ids, result, err := normalizeHistoryStorageTargetInputs(values)
	if err != nil {
		return nil, err
	}
	records, err := s.repository.GetTargetOptionsByIDs(ctx, projectID, ids)
	if err != nil {
		return nil, err
	}
	if len(records) != len(ids) {
		return nil, badHistoryStorageRequest("存储目标不存在、跨工程或类型不受支持")
	}
	return result, nil
}

func normalizeHistoryStorageTargetInputs(values []HistoryStorageTargetInput) ([]string, []repository.SaveHistoryStorageTargetParams, error) {
	if len(values) == 0 {
		return nil, nil, badHistoryStorageRequest("开启历史存储至少需要一个目标")
	}
	ids := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	primaryCount := 0
	result := make([]repository.SaveHistoryStorageTargetParams, 0, len(values))
	for index, value := range values {
		id := strings.TrimSpace(value.ConnectionID)
		if _, err := uuid.Parse(id); err != nil {
			return nil, nil, badHistoryStorageRequest("connectionId 格式无效")
		}
		if _, ok := seen[id]; ok {
			return nil, nil, badHistoryStorageRequest("同一历史配置不能重复选择存储目标")
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
		if value.IsPrimary {
			primaryCount++
		}
		if value.RetentionDays != nil && *value.RetentionDays <= 0 {
			return nil, nil, badHistoryStorageRequest("retentionDays 必须为正整数或 null")
		}
		result = append(result, repository.SaveHistoryStorageTargetParams{ConnectionID: id, IsPrimary: value.IsPrimary, SortOrder: index, RetentionDays: value.RetentionDays})
	}
	if primaryCount != 1 {
		return nil, nil, badHistoryStorageRequest("历史存储必须且只能有一个主目标")
	}
	return ids, result, nil
}

func (s *HistoryStorageService) validateAccess(claims *auth.Claims, projectID string, write bool) error {
	if s == nil || s.repository == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "历史存储服务未初始化")
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
	if write {
		return validateUserID(claims.UserID)
	}
	return nil
}

func normalizeHistoryStorageScope(scopeType, scopeID string, allowDatapoint bool) (repository.HistoryStorageScope, error) {
	scopeType = strings.TrimSpace(scopeType)
	scopeID = strings.TrimSpace(scopeID)
	if scopeType != "access_source" && scopeType != "collector_connection" && scopeType != "compute_unit" && !(allowDatapoint && scopeType == "datapoint") {
		return repository.HistoryStorageScope{}, badHistoryStorageRequest("scopeType 不受支持")
	}
	if _, err := uuid.Parse(scopeID); err != nil {
		return repository.HistoryStorageScope{}, badHistoryStorageRequest("scopeId 格式无效")
	}
	return repository.HistoryStorageScope{Type: scopeType, ID: scopeID}, nil
}

func toHistoryStorageConfiguration(config *repository.HistoryStorageConfigRecord) *HistoryStorageConfiguration {
	if config == nil {
		return nil
	}
	targets := make([]HistoryStorageTarget, 0, len(config.Targets))
	for _, target := range config.Targets {
		targets = append(targets, HistoryStorageTarget{ConnectionID: target.ConnectionID, ConnectionName: target.ConnectionName, ConnectionType: target.ConnectionType, LastTestStatus: target.LastTestStatus, IsPrimary: target.IsPrimary, SortOrder: target.SortOrder, RetentionDays: target.RetentionDays})
	}
	return &HistoryStorageConfiguration{WriteMode: config.WriteMode, IntervalMS: config.IntervalMS, Deadband: config.Deadband, MaxSilenceMS: config.MaxSilenceMS, OfflineBehavior: config.OfflineBehavior, Targets: targets}
}

func badHistoryStorageRequest(message string) error {
	return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, message)
}
func historyStorageNotFound(message string) error {
	return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, message)
}
func int64Ptr(value int64) *int64       { return &value }
func float64Ptr(value float64) *float64 { return &value }

func uniqueHistoryStorageStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
