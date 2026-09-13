package service

import (
	"context"
	"errors"
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
	Tags               []any                                  `json:"tags"`
	AttributeDefaults  map[string]string                      `json:"attributeDefaults"`
	RefreshMode        string                                 `json:"refreshMode"`
	RefreshIntervalMS  *int                                   `json:"refreshIntervalMs"`
	Status             string                                 `json:"status"`
	Capabilities       DataPointCapabilitySummary             `json:"capabilities"`
	LastValue          any                                    `json:"lastValue,omitempty"`
	Quality            string                                 `json:"quality"`
	LastUpdatedAt      *time.Time                             `json:"lastUpdatedAt,omitempty"`
	SourceStatus       string                                 `json:"sourceStatus"`
	SourceError        *string                                `json:"sourceError,omitempty"`
	InvalidReason      *string                                `json:"invalidReason,omitempty"`
	ConsumeMode        string                                 `json:"consumeMode"`
	CreatedAt          time.Time                              `json:"createdAt"`
	UpdatedAt          time.Time                              `json:"updatedAt"`
}

// DataPointListItem 是数据点列表页的轻量投影。
// 列表不返回 lastValue、runtimePermissions、完整 sourceConfig 等大字段，详情接口再按需读取完整数据。
type DataPointListItem struct {
	ID               string                     `json:"id"`
	ProjectID        string                     `json:"projectId"`
	Path             string                     `json:"path"`
	Name             string                     `json:"name"`
	SourceType       string                     `json:"sourceType"`
	SourceID         *string                    `json:"sourceId,omitempty"`
	AccessSourceID   string                     `json:"accessSourceId,omitempty"`
	AccessSourceName string                     `json:"accessSourceName,omitempty"`
	DataType         string                     `json:"dataType"`
	Tags             []any                      `json:"tags"`
	Status           string                     `json:"status"`
	Capabilities     DataPointCapabilitySummary `json:"capabilities"`
	RefCount         int                        `json:"refCount"`
	CreatedAt        time.Time                  `json:"createdAt"`
	UpdatedAt        time.Time                  `json:"updatedAt"`
}

// DataPointCapabilitySummary 描述场景编辑器可以使用的稳定读写能力，不暴露底层数据源细节。
type DataPointCapabilitySummary struct {
	Get DataPointCapability `json:"get"`
	Sub DataPointCapability `json:"sub"`
	Set DataPointCapability `json:"set"`
}

type DataPointCapability struct {
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason,omitempty"`
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
	DataPoints []DataPointListItem `json:"datapoints"`
	Pagination DataPointPagination `json:"pagination"`
}

// DataPointValue 表示数据点值读取结果。
type DataPointValue struct {
	Path            string     `json:"path"`
	Value           any        `json:"value"`
	Quality         string     `json:"quality"`
	Timestamp       time.Time  `json:"timestamp"`
	ObservedAt      *time.Time `json:"observedAt,omitempty"`
	SourceTimestamp *time.Time `json:"sourceTimestamp,omitempty"`
	ValueOrigin     string     `json:"valueOrigin"`
	OriginLabel     string     `json:"originLabel"`
	Status          string     `json:"status"`
}

// CreateDataPointInput 表示 MCP/工程工具创建业务数据点的入参。
// 接入源、查询和计算的存在性由服务端按 sourceType 校验，不能由客户端伪造。
type CreateDataPointInput struct {
	Path              string
	Name              string
	Description       *string
	SourceType        string
	SourceID          *string
	SourceConfig      map[string]any
	DataType          string
	Unit              *string
	PrecisionNum      *int
	DefaultValue      *string
	MinValue          *float64
	MaxValue          *float64
	Tags              []any
	RefreshMode       string
	RefreshIntervalMS *int
	Status            string
}

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
	HasDefaultValue   bool
	MinValue          *float64
	MaxValue          *float64
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
	repository     *repository.DataPointRepository
	queries        *QueryService
	mqtt           *repository.MqttRepository
	computes       *repository.ComputeRepository
	kafkaWorkbench *repository.KafkaWorkbenchRepository
	httpWorkbench  *repository.HTTPWorkbenchRepository
	websocketWb    *repository.WebSocketWorkbenchRepository
	realtimeStore  *repository.RealtimeStoreRepository
	connections    *repository.ConnectionRepository
	collectors     *repository.CollectorRepository
	builtinRuntime *BuiltinRuntimeService
	mqttPublisher  mqttDataPointPublisher
}

// CreateDataPoint 创建手工或基于已有来源的数据点。
func (s *DataPointService) CreateDataPoint(ctx context.Context, projectID, userID string, input CreateDataPointInput) (*DataPoint, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	pathValue, err := normalizeDataPointPath(input.Path)
	if err != nil {
		return nil, err
	}
	name, err := normalizeDataPointName(input.Name)
	if err != nil {
		return nil, err
	}
	sourceType, err := normalizeDataPointSourceType(input.SourceType)
	if err != nil {
		return nil, err
	}
	dataType, err := normalizeDataPointDataType(input.DataType)
	if err != nil {
		return nil, err
	}
	refreshMode := strings.TrimSpace(input.RefreshMode)
	if refreshMode == "" {
		refreshMode = "manual"
	}
	refreshMode, err = normalizeRefreshMode(refreshMode)
	if err != nil {
		return nil, err
	}
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = "active"
	}
	status, err = normalizeDataPointStatus(status)
	if err != nil {
		return nil, err
	}
	var sourceID *string
	if input.SourceID != nil && strings.TrimSpace(*input.SourceID) != "" {
		sourceID, err = normalizeOptionalUUIDPtr(*input.SourceID)
		if err != nil {
			return nil, err
		}
	}
	if sourceType != "manual.input" && sourceID == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "非手工数据点必须引用已有 sourceId")
	}
	if sourceType == "db.query" && s.queries != nil && sourceID != nil {
		if _, err = s.queries.repository.GetByProjectAndID(ctx, projectID, *sourceID); err != nil {
			return nil, err
		}
	}
	if sourceType == "calc.output" && s.computes != nil && sourceID != nil {
		if _, err = s.computes.GetUnitByProjectAndID(ctx, projectID, *sourceID); err != nil {
			return nil, err
		}
	}
	record, err := s.repository.Create(ctx, repository.CreateDataPointParams{ProjectID: projectID, UserID: &userID, Path: pathValue, Name: name, Description: cloneOptionalString(input.Description), SourceType: sourceType, SourceID: sourceID, SourceConfig: cloneMap(input.SourceConfig), DataType: dataType, Unit: cloneOptionalString(input.Unit), PrecisionNum: cloneOptionalInt(input.PrecisionNum), DefaultValue: cloneOptionalString(input.DefaultValue), MinValue: cloneOptionalFloat64(input.MinValue), MaxValue: cloneOptionalFloat64(input.MaxValue), Tags: cloneJSONArray(input.Tags), RefreshMode: refreshMode, RefreshIntervalMS: cloneOptionalInt(input.RefreshIntervalMS), Status: status})
	if err != nil {
		return nil, err
	}
	result := toDataPoint(*record)
	return &result, nil
}

type mqttDataPointPublisher interface {
	PublishSubscriptionMessage(ctx context.Context, projectID, subscriptionID string, payload any) (*MqttPublishResult, error)
}

// NewDataPointService 创建数据点服务。
// 工作台类源对象仓储通过 SetGeneratedSourceRepositories 可选注入，避免构造函数继续膨胀。
func NewDataPointService(repo *repository.DataPointRepository, queryService *QueryService, mqttRepository *repository.MqttRepository, computeRepository *repository.ComputeRepository) *DataPointService {
	return &DataPointService{
		repository: repo,
		queries:    queryService,
		mqtt:       mqttRepository,
		computes:   computeRepository,
	}
}

// SetMqttPublisher 注入 MQTT 发布能力，使订阅型数据点的 publish/set 走真实消息通道。
func (s *DataPointService) SetMqttPublisher(publisher mqttDataPointPublisher) {
	s.mqttPublisher = publisher
}

// SetGeneratedSourceRepositories 注入各协议工作台仓储，用于统一校验自动生成数据点的源对象是否仍然有效。
func (s *DataPointService) SetGeneratedSourceRepositories(
	kafkaWorkbench *repository.KafkaWorkbenchRepository,
	httpWorkbench *repository.HTTPWorkbenchRepository,
	websocketWorkbench *repository.WebSocketWorkbenchRepository,
	realtimeStore *repository.RealtimeStoreRepository,
	connections *repository.ConnectionRepository,
	collectors *repository.CollectorRepository,
	builtinRuntime *BuiltinRuntimeService,
) {
	s.kafkaWorkbench = kafkaWorkbench
	s.httpWorkbench = httpWorkbench
	s.websocketWb = websocketWorkbench
	s.realtimeStore = realtimeStore
	s.connections = connections
	s.collectors = collectors
	s.builtinRuntime = builtinRuntime
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
		Type:           normalizedFilter.Type,
		DataType:       normalizedFilter.DataType,
		Status:         normalizedFilter.Status,
		Search:         normalizedFilter.Search,
		AccessSourceID: normalizedFilter.AccessSourceID,
		SourceID:       normalizedFilter.SourceID,
		SourceIDs:      normalizedFilter.SourceIDs,
		Tags:           normalizedFilter.Tags,
		SortField:      normalizedFilter.SortField,
		SortOrder:      normalizedFilter.SortOrder,
		Page:           normalizedFilter.Page,
		PageSize:       normalizedFilter.PageSize,
	})
	if err != nil {
		return nil, err
	}
	recordIDs := make([]string, 0, len(records))
	for _, record := range records {
		recordIDs = append(recordIDs, record.ID)
	}
	usageCounts, err := s.repository.ListUsageCounts(ctx, projectID, recordIDs)
	if err != nil {
		return nil, err
	}

	items := make([]DataPointListItem, 0, len(records))
	for _, record := range records {
		item := s.toDataPointListItem(ctx, projectID, record)
		item.RefCount = usageCounts[record.ID]
		items = append(items, item)
	}

	page, pageSize := normalizePageAndSize(normalizedFilter.Page, normalizedFilter.PageSize, 50, 500)
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
	s.enrichDataPointAccessSource(ctx, projectID, *record, &dataPoint)
	s.enrichDataPointPreview(ctx, projectID, *record, &dataPoint)
	return &dataPoint, nil
}

// GetDataPointValue 按路径读取数据点值，若来源为 db.query 则回放查询执行结果。
func (s *DataPointService) GetDataPointValue(ctx context.Context, projectID, path string) (*DataPointValue, error) {
	return s.GetDataPointValueWithParameters(ctx, projectID, path, nil)
}

// GetDataPointValueWithParameters 读取数据点并用本次请求参数覆盖查询参数默认值。
// 写入型 SQL 数据点由调用方显式 GET 触发一次执行，参数不会持久化到查询定义。
func (s *DataPointService) GetDataPointValueWithParameters(ctx context.Context, projectID, path string, parameters map[string]any) (*DataPointValue, error) {
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
	return s.buildValueFromRecordWithParameters(ctx, projectID, *record, parameters)
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
	if input.SourceType != nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "数据点来源类型只能在所属工作台中修改")
	}
	if isGeneratedDataPoint(current.SourceType) && (input.DataType != nil || input.SourceID != nil || input.HasSourceConfig) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "生成数据点的来源身份和数据类型只能在所属工作台中修改")
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
	if input.HasDefaultValue {
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

	deleted, err := s.repository.DeleteInvalidBatch(ctx, projectID, []string{id})
	if err != nil {
		return err
	}
	if deleted != 1 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "数据点不存在")
	}
	return nil
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

// DeleteInvalidDataPointsByFilter 按当前列表筛选条件批量清理无效数据点。
// 说明：用于跨页“选择全部结果”场景，service 固定 status=invalid，避免误删有效数据点。
func (s *DataPointService) DeleteInvalidDataPointsByFilter(ctx context.Context, projectID string, filter DataPointListFilter) (int, error) {
	if err := validateProjectID(projectID); err != nil {
		return 0, err
	}
	normalizedFilter, err := normalizeDataPointListFilter(filter)
	if err != nil {
		return 0, err
	}
	normalizedFilter.Status = "invalid"

	deleted, err := s.repository.DeleteInvalidByFilter(ctx, projectID, repository.DataPointListFilter{
		Type:           normalizedFilter.Type,
		DataType:       normalizedFilter.DataType,
		Status:         normalizedFilter.Status,
		Search:         normalizedFilter.Search,
		AccessSourceID: normalizedFilter.AccessSourceID,
		SourceID:       normalizedFilter.SourceID,
		SourceIDs:      normalizedFilter.SourceIDs,
		Tags:           normalizedFilter.Tags,
	})
	if err != nil {
		return 0, err
	}
	return deleted, nil
}

// AppendDataPointTagsByFilter 按当前列表筛选条件批量追加标签。
// 说明：标签只做追加与去重，不覆盖原有标签；筛选条件由后端执行，避免前端跨页拉取大批量数据。
func (s *DataPointService) AppendDataPointTagsByFilter(ctx context.Context, projectID, userID string, filter DataPointListFilter, tags []string) (int, error) {
	if err := validateProjectID(projectID); err != nil {
		return 0, err
	}
	if err := validateUserID(userID); err != nil {
		return 0, err
	}
	normalizedFilter, err := normalizeDataPointListFilter(filter)
	if err != nil {
		return 0, err
	}
	normalizedTags := uniqueStrings(tags)
	if len(normalizedTags) == 0 {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "标签不能为空")
	}

	updated, err := s.repository.AppendTagsByFilter(ctx, projectID, repository.DataPointListFilter{
		Type:           normalizedFilter.Type,
		DataType:       normalizedFilter.DataType,
		Status:         normalizedFilter.Status,
		Search:         normalizedFilter.Search,
		AccessSourceID: normalizedFilter.AccessSourceID,
		SourceID:       normalizedFilter.SourceID,
		SourceIDs:      normalizedFilter.SourceIDs,
		Tags:           normalizedFilter.Tags,
	}, normalizedTags, userID)
	if err != nil {
		return 0, err
	}
	return updated, nil
}

// normalizeDataPointListFilter 统一处理列表过滤参数。
func normalizeDataPointListFilter(filter DataPointListFilter) (DataPointListFilter, error) {
	normalized := DataPointListFilter{
		Type:           strings.TrimSpace(filter.Type),
		DataType:       strings.TrimSpace(filter.DataType),
		Status:         strings.TrimSpace(filter.Status),
		Search:         strings.TrimSpace(filter.Search),
		AccessSourceID: strings.TrimSpace(filter.AccessSourceID),
		SourceID:       strings.TrimSpace(filter.SourceID),
		SourceIDs:      uniqueStrings(filter.SourceIDs),
		Tags:           uniqueStrings(filter.Tags),
		SortField:      strings.TrimSpace(filter.SortField),
		SortOrder:      strings.TrimSpace(filter.SortOrder),
		Page:           filter.Page,
		PageSize:       filter.PageSize,
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
	if normalized.AccessSourceID != "" {
		if _, err := normalizeOptionalUUIDPtr(normalized.AccessSourceID); err != nil {
			return DataPointListFilter{}, err
		}
	}
	for _, id := range normalized.SourceIDs {
		if _, err := normalizeOptionalUUIDPtr(id); err != nil {
			return DataPointListFilter{}, err
		}
	}
	normalized.SortField = normalizeDataPointSortField(normalized.SortField)
	normalized.SortOrder = normalizeDataPointSortOrder(normalized.SortOrder)
	maxPageSize := 500
	if len(normalized.SourceIDs) > 0 {
		maxPageSize = 5000
	}
	page, pageSize := normalizePageAndSize(normalized.Page, normalized.PageSize, 50, maxPageSize)
	normalized.Page = page
	normalized.PageSize = pageSize
	return normalized, nil
}

func normalizeDataPointSortField(value string) string {
	switch strings.TrimSpace(value) {
	case "name":
		return "name"
	case "path":
		return "path"
	default:
		return "createdAt"
	}
}

func normalizeDataPointSortOrder(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "asc") {
		return "asc"
	}
	return "desc"
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
		Tags:               cloneJSONArray(record.Tags),
		AttributeDefaults:  cloneStringMap(record.AttributeDefaults),
		RefreshMode:        record.RefreshMode,
		RefreshIntervalMS:  cloneOptionalInt(record.RefreshIntervalMS),
		Status:             record.Status,
		Capabilities:       dataPointCapabilities(record),
		Quality:            "unknown",
		SourceStatus:       deriveDataPointSourceStatus(record),
		ConsumeMode:        deriveDataPointConsumeMode(record),
		CreatedAt:          record.CreatedAt,
		UpdatedAt:          record.UpdatedAt,
	}
}

func (s *DataPointService) toDataPointListItem(ctx context.Context, projectID string, record repository.DataPointRecord) DataPointListItem {
	connectionID, connectionName := s.resolveDataPointAccessSource(ctx, projectID, record)
	return DataPointListItem{
		ID:               record.ID,
		ProjectID:        record.ProjectID,
		Path:             record.Path,
		Name:             record.Name,
		SourceType:       record.SourceType,
		SourceID:         cloneOptionalString(record.SourceID),
		AccessSourceID:   connectionID,
		AccessSourceName: connectionName,
		DataType:         record.DataType,
		Tags:             cloneJSONArray(record.Tags),
		Status:           record.Status,
		Capabilities:     dataPointCapabilities(record),
		CreatedAt:        record.CreatedAt,
		UpdatedAt:        record.UpdatedAt,
	}
}

func dataPointCapabilities(record repository.DataPointRecord) DataPointCapabilitySummary {
	if record.Status != "active" {
		reason := "数据点未启用或来源已失效"
		return DataPointCapabilitySummary{Get: DataPointCapability{Reason: reason}, Sub: DataPointCapability{Reason: reason}, Set: DataPointCapability{Reason: reason}}
	}
	result := DataPointCapabilitySummary{Get: DataPointCapability{Enabled: true}}
	switch record.SourceType {
	case "mqtt.tag", "mqtt.subscription", "kafka.field", "websocket.session":
		result.Sub = DataPointCapability{Enabled: true}
	default:
		result.Sub = DataPointCapability{Reason: "当前来源不提供订阅能力"}
	}
	switch record.SourceType {
	case "manual", "default":
		result.Set = DataPointCapability{Enabled: true}
	default:
		result.Set = DataPointCapability{Reason: "生成点和只读采集点不能在数据中心直接写入"}
	}
	return result
}

func (s *DataPointService) refreshDataPointValidity(ctx context.Context, projectID string, records []repository.DataPointRecord) error {
	if s == nil || s.repository == nil {
		return nil
	}
	for index := range records {
		record := &records[index]
		if !isGeneratedDataPoint(record.SourceType) || record.SourceID == nil || strings.TrimSpace(*record.SourceID) == "" {
			continue
		}
		valid, err := s.isDataPointSourceValid(ctx, projectID, *record)
		if err != nil {
			return err
		}
		if valid {
			if record.Status == "invalid" {
				if err := s.repository.MarkActiveByID(ctx, projectID, record.ID, record.UpdatedBy); err != nil {
					return err
				}
				record.Status = "active"
				record.UpdatedAt = time.Now()
			}
			continue
		}
		if record.Status == "invalid" {
			continue
		}
		if err := s.repository.MarkInvalidByID(ctx, projectID, record.ID, record.UpdatedBy); err != nil {
			return err
		}
		record.Status = "invalid"
		record.UpdatedAt = time.Now()
	}
	return nil
}

func isGeneratedDataPoint(sourceType string) bool {
	switch strings.TrimSpace(sourceType) {
	case "calc.output", "db.query", "mqtt.subscription", "mqtt.tag", "kafka.field", "http.request", "websocket.session", "realtime.key", "collector.point":
		return true
	default:
		return false
	}
}

func (s *DataPointService) isDataPointSourceValid(ctx context.Context, projectID string, record repository.DataPointRecord) (bool, error) {
	switch record.SourceType {
	case "calc.output":
		return s.isComputeOutputDataPointValid(ctx, projectID, record)
	case "db.query":
		return s.isQueryDataPointValid(ctx, projectID, record)
	case "mqtt.subscription":
		return s.isMqttSubscriptionDataPointValid(ctx, projectID, record)
	case "mqtt.tag":
		return s.isMqttTagDataPointValid(ctx, projectID, record)
	case "kafka.field":
		return s.isKafkaFieldDataPointValid(ctx, projectID, record)
	case "http.request":
		return s.isHTTPRequestDataPointValid(ctx, projectID, record)
	case "websocket.session":
		return s.isWebSocketSessionDataPointValid(ctx, projectID, record)
	case "realtime.key":
		return s.isRealtimeKeyDataPointValid(ctx, projectID, record)
	default:
		return true, nil
	}
}

func (s *DataPointService) isComputeOutputDataPointValid(ctx context.Context, projectID string, record repository.DataPointRecord) (bool, error) {
	if s.computes == nil || record.SourceID == nil {
		return true, nil
	}
	unit, err := s.computes.GetUnitByProjectAndID(ctx, projectID, *record.SourceID)
	if isNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	outputName := strings.TrimSpace(firstString(record.SourceConfig, "outputName"))
	if outputName == "" {
		outputName = strings.TrimSpace(firstString(record.SourceConfig, "outputKey"))
	}
	if outputName == "" {
		outputName = datapointOutputNameFromPath(record.Path)
	}
	for _, output := range unit.Outputs {
		if output.OutputKey == outputName && record.Name == output.Name && isGeneratedPathMatch(record.Path, output.Path, unit.ID) {
			return true, nil
		}
	}
	return false, nil
}

func (s *DataPointService) isQueryDataPointValid(ctx context.Context, projectID string, record repository.DataPointRecord) (bool, error) {
	if s.queries == nil || s.queries.repository == nil || s.queries.connections == nil || record.SourceID == nil {
		return true, nil
	}
	query, err := s.queries.repository.GetByProjectAndID(ctx, projectID, *record.SourceID)
	if isNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if query.QueryType != "sql" {
		return false, nil
	}
	connection, err := s.queries.connections.GetByProjectAndID(ctx, projectID, query.ConnectionID)
	if isNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if connection.Type != "relational" && connection.Type != "builtin.relation" && connection.Type != "builtin.timeseries" {
		return false, nil
	}
	return queryOutputMappingMatchesDataPoint(*query, record), nil
}

func queryOutputMappingMatchesDataPoint(query repository.QueryRecord, record repository.DataPointRecord) bool {
	for _, output := range query.Outputs {
		// 查询数据点的名称和路径由输出映射决定：字段提取使用显示名称和后缀，
		// 完整结果则直接占用查询路径。不能再用 query.Name 判断，否则所有字段输出都会被误标失效。
		if output.DataPointID == record.ID && output.DataPointPath == record.Path && output.DisplayName == record.Name {
			return true
		}
	}
	return false
}

func (s *DataPointService) isMqttSubscriptionDataPointValid(ctx context.Context, projectID string, record repository.DataPointRecord) (bool, error) {
	if s.mqtt == nil || record.SourceID == nil {
		return true, nil
	}
	subscription, err := s.mqtt.GetSubscription(ctx, projectID, *record.SourceID)
	if isNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	connection, err := s.mqtt.GetConnectionSummary(ctx, projectID, subscription.ConnectionID)
	if isNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	expectedPath := "mqtt." + normalizeDatapointSegment(connection.Name) + "." + mqttSubscriptionPathSegment(*subscription)
	return record.Name == subscription.Name && isGeneratedPathMatch(record.Path, expectedPath, subscription.ID), nil
}

func (s *DataPointService) isMqttTagDataPointValid(ctx context.Context, projectID string, record repository.DataPointRecord) (bool, error) {
	if s.mqtt == nil || record.SourceID == nil {
		return true, nil
	}
	tag, err := s.mqtt.GetTag(ctx, projectID, *record.SourceID)
	if isNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	subscription, err := s.mqtt.GetSubscription(ctx, projectID, tag.SubscriptionID)
	if isNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	connection, err := s.mqtt.GetConnectionSummary(ctx, projectID, subscription.ConnectionID)
	if isNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	expectedPath := "mqtt." + normalizeDatapointSegment(connection.Name) + "." + mqttSubscriptionPathSegment(*subscription) + "." + normalizeDatapointSegment(tag.Name)
	return record.Name == tag.Name && isGeneratedPathMatch(record.Path, expectedPath, tag.ID), nil
}

func (s *DataPointService) isKafkaFieldDataPointValid(ctx context.Context, projectID string, record repository.DataPointRecord) (bool, error) {
	fieldID := strings.TrimSpace(firstString(record.SourceConfig, "fieldId"))
	if s.kafkaWorkbench == nil || fieldID == "" {
		return true, nil
	}
	field, err := s.kafkaWorkbench.GetField(ctx, projectID, fieldID)
	if isNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	mapping, err := s.kafkaWorkbench.GetTopicMapping(ctx, projectID, field.TopicMappingID)
	if isNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	expectedPath := buildKafkaDataPointPath(mapping.Name, field.Name)
	return record.Name == field.Name &&
		record.SourceID != nil && strings.TrimSpace(*record.SourceID) == field.ConnectionID &&
		isGeneratedPathMatch(record.Path, expectedPath, field.ID), nil
}

func (s *DataPointService) isHTTPRequestDataPointValid(ctx context.Context, projectID string, record repository.DataPointRecord) (bool, error) {
	requestID := strings.TrimSpace(firstString(record.SourceConfig, "requestId"))
	if s.httpWorkbench == nil || s.connections == nil || requestID == "" {
		return true, nil
	}
	request, err := s.httpWorkbench.GetRequest(ctx, projectID, requestID)
	if isNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	connection, err := s.connections.GetByProjectAndID(ctx, projectID, request.ConnectionID)
	if isNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	expectedPath := buildHTTPDataPointPath(connection.Name, request.Name)
	return record.Name == request.Name &&
		record.SourceID != nil && strings.TrimSpace(*record.SourceID) == request.ConnectionID &&
		isGeneratedPathMatch(record.Path, expectedPath, request.ID), nil
}

func (s *DataPointService) isWebSocketSessionDataPointValid(ctx context.Context, projectID string, record repository.DataPointRecord) (bool, error) {
	sessionID := strings.TrimSpace(firstString(record.SourceConfig, "sessionId"))
	if s.websocketWb == nil || s.connections == nil || sessionID == "" {
		return true, nil
	}
	session, err := s.websocketWb.GetSession(ctx, projectID, sessionID)
	if isNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	connection, err := s.connections.GetByProjectAndID(ctx, projectID, session.ConnectionID)
	if isNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	expectedPath := buildWebSocketDataPointPath(connection.Name, session.Name)
	return record.Name == session.Name &&
		record.SourceID != nil && strings.TrimSpace(*record.SourceID) == session.ConnectionID &&
		isGeneratedPathMatch(record.Path, expectedPath, session.ID), nil
}

func (s *DataPointService) isRealtimeKeyDataPointValid(ctx context.Context, projectID string, record repository.DataPointRecord) (bool, error) {
	keyID := strings.TrimSpace(firstString(record.SourceConfig, "keyId"))
	if s.realtimeStore == nil || s.connections == nil || keyID == "" || record.SourceID == nil {
		return true, nil
	}
	connectionID := strings.TrimSpace(*record.SourceID)
	connection, err := s.connections.GetByProjectAndID(ctx, projectID, connectionID)
	if isNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	provider := "builtin"
	if connection.Type == "redis" {
		provider = "redis"
	}
	if connection.Type != "redis" && connection.Type != "builtin.realtime" {
		return false, nil
	}
	keyPath := strings.TrimSpace(firstString(record.SourceConfig, "key"))
	if keyPath == "" {
		keyPath = record.Name
	}
	key, err := s.realtimeStore.GetByKey(ctx, projectID, connectionID, provider, keyPath)
	if isNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	for _, output := range key.Outputs {
		if output.DataPointID != record.ID {
			continue
		}
		return key.ID == keyID && output.DataPointPath == record.Path, nil
	}
	return false, nil
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	var appErr *apperrors.AppError
	return errors.As(err, &appErr) && appErr.Code == apperrors.ErrorCodeNotFound
}

func isGeneratedPathMatch(actualPath, basePath, sourceID string) bool {
	actualPath = strings.TrimSpace(actualPath)
	basePath = strings.TrimSpace(basePath)
	if actualPath == "" || basePath == "" {
		return false
	}
	if actualPath == basePath {
		return true
	}
	if len(sourceID) >= 8 && actualPath == basePath+"_"+sourceID[:8] {
		return true
	}
	return strings.HasPrefix(actualPath, basePath+"_")
}

func (s *DataPointService) enrichDataPointPreview(ctx context.Context, projectID string, record repository.DataPointRecord, target *DataPoint) {
	if target == nil {
		return
	}
	if record.Status == "invalid" {
		target.Quality = "bad"
		target.SourceStatus = "invalid"
		reason := deriveDataPointInvalidReason(record)
		target.SourceError = &reason
		target.InvalidReason = &reason
		return
	}

	value, err := s.buildValueFromRecord(ctx, projectID, record)
	if err != nil {
		target.Quality = "bad"
		target.SourceStatus = "error"
		message := err.Error()
		target.SourceError = &message
		return
	}
	target.LastValue = value.Value
	target.Quality = value.Quality
	target.LastUpdatedAt = &value.Timestamp
	if value.Quality == "good" {
		target.SourceStatus = "ready"
	}
}

// enrichDataPointAccessSource 在返回数据点时补充接入源 ID。
// source_id 对不同数据点含义不同：有的是查询/变量 ID，有的才是连接 ID；前端跳工作台统一读取 sourceConfig.connectionId。
// 解析失败时静默保留原配置，避免列表接口因为单个来源缺失而不可用。
func (s *DataPointService) enrichDataPointAccessSource(ctx context.Context, projectID string, record repository.DataPointRecord, target *DataPoint) {
	if target == nil {
		return
	}
	connectionID, connectionName := s.resolveDataPointAccessSource(ctx, projectID, record)
	if connectionID == "" {
		return
	}
	if target.SourceConfig == nil {
		target.SourceConfig = map[string]any{}
	}
	target.SourceConfig["connectionId"] = connectionID
	if connectionName != "" {
		target.SourceConfig["accessSourceName"] = connectionName
	}
}

func (s *DataPointService) resolveDataPointAccessSource(ctx context.Context, projectID string, record repository.DataPointRecord) (string, string) {
	if connectionID := strings.TrimSpace(firstString(record.SourceConfig, "connectionId")); connectionID != "" {
		return s.resolveConnectionName(ctx, projectID, connectionID)
	}
	if connectionID := strings.TrimSpace(firstString(record.SourceConfig, "sourceConnectionId")); connectionID != "" {
		return s.resolveConnectionName(ctx, projectID, connectionID)
	}

	sourceID := ""
	if record.SourceID != nil {
		sourceID = strings.TrimSpace(*record.SourceID)
	}
	switch strings.TrimSpace(record.SourceType) {
	case "collector.point":
		if s.collectors == nil || sourceID == "" {
			return "", ""
		}
		point, err := s.collectors.GetPointByID(ctx, projectID, sourceID)
		if err != nil || point == nil {
			return "", ""
		}
		connection, err := s.collectors.GetConnection(ctx, projectID, point.ConnectionID)
		if err != nil || connection == nil {
			return point.ConnectionID, ""
		}
		return point.ConnectionID, connection.Name
	case "db.query":
		if s.queries == nil || s.queries.repository == nil || sourceID == "" {
			return "", ""
		}
		query, err := s.queries.repository.GetByProjectAndID(ctx, projectID, sourceID)
		if err != nil || query == nil {
			return "", ""
		}
		return s.resolveConnectionName(ctx, projectID, query.ConnectionID)
	case "mqtt.subscription":
		if s.mqtt == nil || sourceID == "" {
			return "", ""
		}
		subscription, err := s.mqtt.GetSubscription(ctx, projectID, sourceID)
		if err != nil || subscription == nil {
			return "", ""
		}
		return s.resolveConnectionName(ctx, projectID, subscription.ConnectionID)
	case "mqtt.tag":
		if s.mqtt == nil || sourceID == "" {
			return "", ""
		}
		tag, err := s.mqtt.GetTag(ctx, projectID, sourceID)
		if err != nil || tag == nil {
			return "", ""
		}
		subscription, err := s.mqtt.GetSubscription(ctx, projectID, tag.SubscriptionID)
		if err != nil || subscription == nil {
			return "", ""
		}
		return s.resolveConnectionName(ctx, projectID, subscription.ConnectionID)
	case "calc.output", "alarm.state":
		return "", ""
	default:
		if sourceID != "" && sourceTypeUsesSourceIDAsConnection(record.SourceType) {
			return s.resolveConnectionName(ctx, projectID, sourceID)
		}
		return "", ""
	}
}

func (s *DataPointService) resolveConnectionName(ctx context.Context, projectID, connectionID string) (string, string) {
	connectionID = strings.TrimSpace(connectionID)
	if connectionID == "" {
		return "", ""
	}
	if s.connections == nil {
		return connectionID, ""
	}
	connection, err := s.connections.GetByProjectAndID(ctx, projectID, connectionID)
	if err != nil || connection == nil {
		return connectionID, ""
	}
	return connection.ID, connection.Name
}

func sourceTypeUsesSourceIDAsConnection(sourceType string) bool {
	switch strings.TrimSpace(sourceType) {
	case "http.request", "websocket.session", "realtime.key", "kafka.field", "kafka.raw":
		return true
	default:
		return false
	}
}

func deriveDataPointInvalidReason(record repository.DataPointRecord) string {
	switch strings.TrimSpace(record.SourceType) {
	case "calc.output":
		return "计算单元输出已失效：计算单元不存在，或输出名称、路径已变更"
	case "db.query":
		return "查询数据点已失效：查询不存在、查询类型已变更，或连接、路径已变更"
	case "mqtt.subscription":
		return "MQTT 订阅数据点已失效：订阅不存在，或订阅名称、路径已变更"
	case "mqtt.tag":
		return "MQTT 变量数据点已失效：变量、订阅、分组或连接不存在，或路径已变更"
	case "http.request":
		return "HTTP 请求数据点已失效：接口请求或接入源不存在，或路径已变更"
	case "websocket.session":
		return "WebSocket 会话数据点已失效：会话或接入源不存在，或路径已变更"
	case "realtime.key":
		return "实时库 key 数据点已失效：key 元数据、接入源不存在，或路径已变更"
	case "kafka.field":
		return "Kafka 变量数据点已失效：变量、Topic 映射不存在，或路径已变更"
	default:
		return "数据点已失效"
	}
}

func deriveDataPointSourceStatus(record repository.DataPointRecord) string {
	switch record.Status {
	case "active":
		return "ready"
	case "inactive":
		return "inactive"
	case "invalid":
		return "invalid"
	default:
		return "unknown"
	}
}

func deriveDataPointConsumeMode(record repository.DataPointRecord) string {
	switch record.RefreshMode {
	case "subscription":
		return "subscribe"
	case "manual":
		return "query"
	default:
		switch record.SourceType {
		case "mqtt.tag", "mqtt.subscription":
			return "subscribe"
		default:
			return "query"
		}
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
	normalized, ok := canonicalDataPointType(dataType)
	if strings.TrimSpace(dataType) == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "dataType 不能为空")
	}
	if !ok {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "dataType 不属于规范数据点类型")
	}
	return normalized, nil
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

	if parameters, ok := rawParameters.(map[string]any); ok {
		return cloneMap(parameters), nil
	}

	// 查询输出生成点保存的是参数定义数组；GET 回放时只使用其中明确配置的默认值。
	definitions, ok := rawParameters.([]any)
	if !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "数据点 source_config.parameters 格式无效")
	}
	parameters := make(map[string]any, len(definitions))
	for _, rawDefinition := range definitions {
		definition, definitionOK := rawDefinition.(map[string]any)
		if !definitionOK {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "数据点 source_config.parameters 参数定义无效")
		}
		name := strings.TrimSpace(firstString(definition, "name"))
		if name == "" {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "数据点 source_config.parameters 参数名称无效")
		}
		if defaultValue, exists := definition["default"]; exists {
			parameters[name] = defaultValue
		}
	}
	return parameters, nil
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
		input.HasDefaultValue ||
		input.MinValue != nil ||
		input.MaxValue != nil ||
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
	Type           string
	DataType       string
	Status         string
	Search         string
	AccessSourceID string
	SourceID       string
	SourceIDs      []string
	Tags           []string
	SortField      string
	SortOrder      string
	Page           int
	PageSize       int
}
