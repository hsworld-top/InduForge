package service

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

var allowedDataPointStatuses = map[string]struct{}{
	"active":   {},
	"inactive": {},
	"invalid":  {},
}

var allowedRefreshModes = map[string]struct{}{
	"auto":         {},
	"manual":       {},
	"subscription": {},
}

// DataPoint 表示面向 HTTP 层返回的数据点对象。
type DataPoint struct {
	ID                 string                                 `json:"id"`
	ProjectID          string                                 `json:"projectId"`
	Path               string                                 `json:"path"`
	Name               string                                 `json:"name"`
	Description        *string                                `json:"description"`
	SourceType         string                                 `json:"sourceType"`
	SourceID           *string                                `json:"sourceId"`
	SourceConfig       map[string]any                         `json:"sourceConfig"`
	DataType           string                                 `json:"dataType"`
	RuntimePermissions repository.DataPointRuntimePermissions `json:"runtimePermissions"`
	Unit               *string                                `json:"unit"`
	PrecisionNum       *int                                   `json:"precisionNum"`
	DefaultValue       *string                                `json:"defaultValue"`
	MinValue           *float64                               `json:"minValue"`
	MaxValue           *float64                               `json:"maxValue"`
	AlarmLow           *float64                               `json:"alarmLow"`
	AlarmHigh          *float64                               `json:"alarmHigh"`
	Tags               []any                                  `json:"tags"`
	RefreshMode        string                                 `json:"refreshMode"`
	RefreshIntervalMS  *int                                   `json:"refreshIntervalMs"`
	Status             string                                 `json:"status"`
	CreatedAt          time.Time                              `json:"createdAt"`
	UpdatedAt          time.Time                              `json:"updatedAt"`
}

// DataPointPagination 表示数据点列表分页信息。
type DataPointPagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

// DataPointListResult 表示数据点列表与分页结果。
type DataPointListResult struct {
	DataPoints []DataPoint         `json:"datapoints"`
	Pagination DataPointPagination `json:"pagination"`
}

// DataPointValue 表示数据点值读取结果。
type DataPointValue struct {
	Path      string    `json:"path"`
	Value     any       `json:"value"`
	Quality   string    `json:"quality"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"`
}

// CreateDataPointInput 预留给后续扩展的创建入参。
type CreateDataPointInput struct{}

// UpdateDataPointInput 表示更新数据点时的业务输入。
type UpdateDataPointInput struct {
	Name              *string
	Description       *string
	SourceType        *string
	SourceID          *string
	SourceConfig      map[string]any
	HasSourceConfig   bool
	DataType          *string
	Unit              *string
	PrecisionNum      *int
	DefaultValue      *string
	MinValue          *float64
	MaxValue          *float64
	AlarmLow          *float64
	AlarmHigh         *float64
	Tags              []any
	HasTags           bool
	RefreshMode       *string
	RefreshIntervalMS *int
	Status            *string
}

// UpdateDataPointRuntimePermissionsInput 表示运行态权限更新入参。
type UpdateDataPointRuntimePermissionsInput struct {
	Write repository.RuntimePermissionGrant
}

// DataPointService 承载数据点领域的校验、映射与值读取逻辑。
type DataPointService struct {
	repository *repository.DataPointRepository
	queries    *QueryService
	mqtt       *repository.MqttRepository
}

// NewDataPointService ????????
// MQTT ?????????????????????? MQTT ???????
func NewDataPointService(repo *repository.DataPointRepository, queryService *QueryService, mqttRepository *repository.MqttRepository) *DataPointService {
	return &DataPointService{
		repository: repo,
		queries:    queryService,
		mqtt:       mqttRepository,
	}
}

// ListDataPoints 查询项目下的数据点列表。
func (s *DataPointService) ListDataPoints(ctx context.Context, projectID string, filter DataPointListFilter) (*DataPointListResult, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}

	normalizedFilter, err := normalizeDataPointListFilter(filter)
	if err != nil {
		return nil, err
	}

	records, total, err := s.repository.ListByProject(ctx, projectID, repository.DataPointListFilter{
		Type:      normalizedFilter.Type,
		Status:    normalizedFilter.Status,
		Search:    normalizedFilter.Search,
		SourceID:  normalizedFilter.SourceID,
		SourceIDs: normalizedFilter.SourceIDs,
		Page:      normalizedFilter.Page,
		PageSize:  normalizedFilter.PageSize,
	})
	if err != nil {
		return nil, err
	}

	items := make([]DataPoint, 0, len(records))
	for _, record := range records {
		items = append(items, toDataPoint(record))
	}

	page, pageSize := normalizePageAndSize(normalizedFilter.Page, normalizedFilter.PageSize, 50, 200)
	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}

	return &DataPointListResult{
		DataPoints: items,
		Pagination: DataPointPagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// GetDataPoint 获取单个数据点详情。
func (s *DataPointService) GetDataPoint(ctx context.Context, projectID, id string) (*DataPoint, error) {
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

	dataPoint := toDataPoint(*record)
	return &dataPoint, nil
}

// GetDataPointValue 按路径读取数据点值，若来源为 db.query 则回放查询执行结果。
func (s *DataPointService) GetDataPointValue(ctx context.Context, projectID, path string) (*DataPointValue, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}

	normalizedPath, err := normalizeDataPointPath(path)
	if err != nil {
		return nil, err
	}

	record, err := s.repository.GetByProjectAndPath(ctx, projectID, normalizedPath)
	if err != nil {
		return nil, err
	}
	return s.buildValueFromRecord(ctx, projectID, *record)
}

// UpdateDataPoint 更新单个数据点。
func (s *DataPointService) UpdateDataPoint(ctx context.Context, projectID, id, userID string, input UpdateDataPointInput) (*DataPoint, error) {
	var err error

	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateDataPointID(id); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	if !hasDataPointUpdateChanges(input) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "至少需要提供一个待更新字段")
	}

	current, err := s.repository.GetByProjectAndID(ctx, projectID, id)
	if err != nil {
		return nil, err
	}

	nextName := current.Name
	if input.Name != nil {
		nextName, err = normalizeDataPointName(*input.Name)
		if err != nil {
			return nil, err
		}
	}

	nextDescription := cloneOptionalString(current.Description)
	if input.Description != nil {
		nextDescription = cloneOptionalString(input.Description)
	}

	nextSourceType := current.SourceType
	if input.SourceType != nil {
		nextSourceType, err = normalizeDataPointSourceType(*input.SourceType)
		if err != nil {
			return nil, err
		}
	}

	nextSourceID := cloneOptionalString(current.SourceID)
	if input.SourceID != nil {
		nextSourceID, err = normalizeOptionalUUIDPtr(*input.SourceID)
		if err != nil {
			return nil, err
		}
	}

	nextSourceConfig := cloneMap(current.SourceConfig)
	if input.HasSourceConfig {
		nextSourceConfig, err = normalizeQueryConfig(input.SourceConfig)
		if err != nil {
			return nil, err
		}
	}

	nextDataType := current.DataType
	if input.DataType != nil {
		nextDataType, err = normalizeDataPointDataType(*input.DataType)
		if err != nil {
			return nil, err
		}
	}

	nextUnit := cloneOptionalString(current.Unit)
	if input.Unit != nil {
		nextUnit = cloneOptionalString(input.Unit)
	}

	nextPrecisionNum := current.PrecisionNum
	if input.PrecisionNum != nil {
		nextPrecisionNum = cloneOptionalInt(input.PrecisionNum)
	}

	nextDefaultValue := cloneOptionalString(current.DefaultValue)
	if input.DefaultValue != nil {
		nextDefaultValue = cloneOptionalString(input.DefaultValue)
	}

	nextMinValue := cloneOptionalFloat64(current.MinValue)
	if input.MinValue != nil {
		nextMinValue = cloneOptionalFloat64(input.MinValue)
	}

	nextMaxValue := cloneOptionalFloat64(current.MaxValue)
	if input.MaxValue != nil {
		nextMaxValue = cloneOptionalFloat64(input.MaxValue)
	}

	nextAlarmLow := cloneOptionalFloat64(current.AlarmLow)
	if input.AlarmLow != nil {
		nextAlarmLow = cloneOptionalFloat64(input.AlarmLow)
	}

	nextAlarmHigh := cloneOptionalFloat64(current.AlarmHigh)
	if input.AlarmHigh != nil {
		nextAlarmHigh = cloneOptionalFloat64(input.AlarmHigh)
	}

	nextTags := cloneJSONArray(current.Tags)
	if input.HasTags {
		nextTags = cloneJSONArray(input.Tags)
	}

	nextRefreshMode := current.RefreshMode
	if input.RefreshMode != nil {
		nextRefreshMode, err = normalizeRefreshMode(*input.RefreshMode)
		if err != nil {
			return nil, err
		}
	}

	nextRefreshIntervalMS := current.RefreshIntervalMS
	if input.RefreshIntervalMS != nil {
		nextRefreshIntervalMS = cloneOptionalInt(input.RefreshIntervalMS)
	}

	nextStatus := current.Status
	if input.Status != nil {
		nextStatus, err = normalizeDataPointStatus(*input.Status)
		if err != nil {
			return nil, err
		}
	}

	updated, err := s.repository.Update(ctx, repository.UpdateDataPointParams{
		ID:                current.ID,
		ProjectID:         current.ProjectID,
		UserID:            userID,
		Name:              nextName,
		Description:       nextDescription,
		SourceType:        nextSourceType,
		SourceID:          nextSourceID,
		SourceConfig:      nextSourceConfig,
		DataType:          nextDataType,
		Unit:              nextUnit,
		PrecisionNum:      nextPrecisionNum,
		DefaultValue:      nextDefaultValue,
		MinValue:          nextMinValue,
		MaxValue:          nextMaxValue,
		AlarmLow:          nextAlarmLow,
		AlarmHigh:         nextAlarmHigh,
		Tags:              nextTags,
		RefreshMode:       nextRefreshMode,
		RefreshIntervalMS: nextRefreshIntervalMS,
		Status:            nextStatus,
	})
	if err != nil {
		return nil, err
	}

	dataPoint := toDataPoint(*updated)
	return &dataPoint, nil
}

// UpdateDataPointRuntimePermissions 更新数据点运行态写权限。
// 说明：当前只开放 write 节点，service 负责项目/用户边界校验，仓储层只做参数化持久化。
func (s *DataPointService) UpdateDataPointRuntimePermissions(ctx context.Context, projectID, id, userID string, input UpdateDataPointRuntimePermissionsInput) (*DataPoint, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateDataPointID(id); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}

	updated, err := s.repository.UpdateRuntimePermissions(ctx, repository.UpdateDataPointRuntimePermissionsParams{
		ID:        id,
		ProjectID: projectID,
		UserID:    userID,
		RuntimePermissions: repository.DataPointRuntimePermissions{
			Write: input.Write,
		},
	})
	if err != nil {
		return nil, err
	}

	dataPoint := toDataPoint(*updated)
	return &dataPoint, nil
}

// DeleteDataPoint 删除单个数据点，但仅允许删除 invalid 状态的数据点。
func (s *DataPointService) DeleteDataPoint(ctx context.Context, projectID, id string) error {
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	if err := validateDataPointID(id); err != nil {
		return err
	}

	record, err := s.repository.GetByProjectAndID(ctx, projectID, id)
	if err != nil {
		return err
	}
	if record.Status != "invalid" {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "仅允许删除无效数据点")
	}

	return s.repository.Delete(ctx, projectID, id)
}

// DeleteDataPointsBatch 批量删除 invalid 状态的数据点。
func (s *DataPointService) DeleteDataPointsBatch(ctx context.Context, projectID string, ids []string) (int, error) {
	if err := validateProjectID(projectID); err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "删除列表不能为空")
	}

	uniqueIDs := uniqueStrings(ids)
	if len(uniqueIDs) == 0 {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "删除列表不能为空")
	}
	for _, id := range uniqueIDs {
		if err := validateDataPointID(id); err != nil {
			return 0, err
		}
	}

	records, err := s.repository.GetByProjectAndIDs(ctx, projectID, uniqueIDs)
	if err != nil {
		return 0, err
	}
	if len(records) != len(uniqueIDs) {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "部分数据点不存在")
	}
	for _, record := range records {
		if record.Status != "invalid" {
			return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "仅允许删除无效数据点")
		}
	}

	deleted, err := s.repository.DeleteInvalidBatch(ctx, projectID, uniqueIDs)
	if err != nil {
		return 0, err
	}
	return deleted, nil
}

// normalizeDataPointListFilter 统一处理列表过滤参数。
func normalizeDataPointListFilter(filter DataPointListFilter) (DataPointListFilter, error) {
	normalized := DataPointListFilter{
		Type:      strings.TrimSpace(filter.Type),
		Status:    strings.TrimSpace(filter.Status),
		Search:    strings.TrimSpace(filter.Search),
		SourceID:  strings.TrimSpace(filter.SourceID),
		SourceIDs: uniqueStrings(filter.SourceIDs),
		Page:      filter.Page,
		PageSize:  filter.PageSize,
	}

	if normalized.Type != "" {
		if _, err := normalizeDataPointSourceType(normalized.Type); err != nil {
			return DataPointListFilter{}, err
		}
	}
	if normalized.Status != "" {
		if _, err := normalizeDataPointStatus(normalized.Status); err != nil {
			return DataPointListFilter{}, err
		}
	}
	if normalized.SourceID != "" {
		if _, err := normalizeOptionalUUIDPtr(normalized.SourceID); err != nil {
			return DataPointListFilter{}, err
		}
	}
	for _, id := range normalized.SourceIDs {
		if _, err := normalizeOptionalUUIDPtr(id); err != nil {
			return DataPointListFilter{}, err
		}
	}
	page, pageSize := normalizePageAndSize(normalized.Page, normalized.PageSize, 50, 200)
	normalized.Page = page
	normalized.PageSize = pageSize
	return normalized, nil
}

// toDataPoint 将仓储记录映射为 HTTP 返回对象。
func toDataPoint(record repository.DataPointRecord) DataPoint {
	return DataPoint{
		ID:                 record.ID,
		ProjectID:          record.ProjectID,
		Path:               record.Path,
		Name:               record.Name,
		Description:        cloneOptionalString(record.Description),
		SourceType:         record.SourceType,
		SourceID:           cloneOptionalString(record.SourceID),
		SourceConfig:       cloneMap(record.SourceConfig),
		DataType:           record.DataType,
		RuntimePermissions: record.RuntimePermissions,
		Unit:               cloneOptionalString(record.Unit),
		PrecisionNum:       cloneOptionalInt(record.PrecisionNum),
		DefaultValue:       cloneOptionalString(record.DefaultValue),
		MinValue:           cloneOptionalFloat64(record.MinValue),
		MaxValue:           cloneOptionalFloat64(record.MaxValue),
		AlarmLow:           cloneOptionalFloat64(record.AlarmLow),
		AlarmHigh:          cloneOptionalFloat64(record.AlarmHigh),
		Tags:               cloneJSONArray(record.Tags),
		RefreshMode:        record.RefreshMode,
		RefreshIntervalMS:  cloneOptionalInt(record.RefreshIntervalMS),
		Status:             record.Status,
		CreatedAt:          record.CreatedAt,
		UpdatedAt:          record.UpdatedAt,
	}
}

func normalizeDataPointPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "path 不能为空")
	}
	return path, nil
}

func normalizeDataPointName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "数据点名称不能为空")
	}
	if len([]rune(name)) > 100 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "数据点名称长度不能超过 100 个字符")
	}
	return name, nil
}

func normalizeDataPointSourceType(sourceType string) (string, error) {
	sourceType = strings.TrimSpace(strings.ToLower(sourceType))
	if sourceType == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "sourceType 不能为空")
	}
	if len([]rune(sourceType)) > 50 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "sourceType 长度不能超过 50 个字符")
	}
	return sourceType, nil
}

func normalizeDataPointDataType(dataType string) (string, error) {
	dataType = strings.TrimSpace(dataType)
	if dataType == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "dataType 不能为空")
	}
	if len([]rune(dataType)) > 20 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "dataType 长度不能超过 20 个字符")
	}
	return dataType, nil
}

func normalizeRefreshMode(refreshMode string) (string, error) {
	refreshMode = strings.TrimSpace(strings.ToLower(refreshMode))
	if refreshMode == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "refreshMode 不能为空")
	}
	if _, ok := allowedRefreshModes[refreshMode]; !ok {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "refreshMode 不受支持")
	}
	return refreshMode, nil
}

func normalizeDataPointStatus(status string) (string, error) {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "status 不能为空")
	}
	if _, ok := allowedDataPointStatuses[status]; !ok {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "数据点状态不受支持")
	}
	return status, nil
}

func dataPointQueryParameters(sourceConfig map[string]any) (map[string]any, error) {
	if sourceConfig == nil {
		return map[string]any{}, nil
	}

	rawParameters, ok := sourceConfig["parameters"]
	if !ok || rawParameters == nil {
		return map[string]any{}, nil
	}

	parameters, ok := rawParameters.(map[string]any)
	if !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "数据点 source_config.parameters 格式无效")
	}

	return cloneMap(parameters), nil
}

func hasDataPointUpdateChanges(input UpdateDataPointInput) bool {
	return input.Name != nil ||
		input.Description != nil ||
		input.SourceType != nil ||
		input.SourceID != nil ||
		input.HasSourceConfig ||
		input.DataType != nil ||
		input.Unit != nil ||
		input.PrecisionNum != nil ||
		input.DefaultValue != nil ||
		input.MinValue != nil ||
		input.MaxValue != nil ||
		input.AlarmLow != nil ||
		input.AlarmHigh != nil ||
		input.HasTags ||
		input.RefreshMode != nil ||
		input.RefreshIntervalMS != nil ||
		input.Status != nil
}

func validateDataPointID(id string) error {
	if _, err := uuid.Parse(strings.TrimSpace(id)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "dataPointId 格式无效", err)
	}
	return nil
}

func normalizeOptionalUUIDPtr(value string) (*string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "UUID 不能为空")
	}
	if _, err := uuid.Parse(value); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "UUID 格式无效", err)
	}
	return &value, nil
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
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
	return result
}

func cloneJSONArray(values []any) []any {
	if values == nil {
		return []any{}
	}
	result := make([]any, len(values))
	copy(result, values)
	return result
}

func cloneOptionalInt(value *int) *int {
	if value == nil {
		return nil
	}
	next := *value
	return &next
}

func cloneOptionalFloat64(value *float64) *float64 {
	if value == nil {
		return nil
	}
	next := *value
	return &next
}

// DataPointListFilter 表示 service 层对数据点列表的入参封装，供 HTTP 层与仓储层之间转换。
type DataPointListFilter struct {
	Type      string
	Status    string
	Search    string
	SourceID  string
	SourceIDs []string
	Page      int
	PageSize  int
}
