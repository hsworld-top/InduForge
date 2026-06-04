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
	AlarmLow           *float64                               `json:"alarmLow"`
	AlarmHigh          *float64                               `json:"alarmHigh"`
	Tags               []any                                  `json:"tags"`
	RefreshMode        string                                 `json:"refreshMode"`
	RefreshIntervalMS  *int                                   `json:"refreshIntervalMs"`
	Status             string                                 `json:"status"`
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
	repository     *repository.DataPointRepository
	queries        *QueryService
	mqtt           *repository.MqttRepository
	computes       *repository.ComputeRepository
	kafkaWorkbench *repository.KafkaWorkbenchRepository
	httpWorkbench  *repository.HTTPWorkbenchRepository
	websocketWb    *repository.WebSocketWorkbenchRepository
	realtimeStore  *repository.RealtimeStoreRepository
	connections    *repository.ConnectionRepository
	builtinRuntime *BuiltinRuntimeService
	opcuaModeling  *repository.OpcuaModelingRepository
	modbusModeling *repository.ModbusModelingRepository
	s7Modeling     *repository.S7ModelingRepository
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

// SetGeneratedSourceRepositories 注入各协议工作台仓储，用于统一校验自动生成数据点的源对象是否仍然有效。
func (s *DataPointService) SetGeneratedSourceRepositories(
	kafkaWorkbench *repository.KafkaWorkbenchRepository,
	httpWorkbench *repository.HTTPWorkbenchRepository,
	websocketWorkbench *repository.WebSocketWorkbenchRepository,
	realtimeStore *repository.RealtimeStoreRepository,
	connections *repository.ConnectionRepository,
	builtinRuntime *BuiltinRuntimeService,
	opcuaModeling *repository.OpcuaModelingRepository,
	modbusModeling *repository.ModbusModelingRepository,
	s7Modeling *repository.S7ModelingRepository,
) {
	s.kafkaWorkbench = kafkaWorkbench
	s.httpWorkbench = httpWorkbench
	s.websocketWb = websocketWorkbench
	s.realtimeStore = realtimeStore
	s.connections = connections
	s.builtinRuntime = builtinRuntime
	s.opcuaModeling = opcuaModeling
	s.modbusModeling = modbusModeling
	s.s7Modeling = s7Modeling
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
	if err := s.refreshDataPointValidity(ctx, projectID, records); err != nil {
		return nil, err
	}

	items := make([]DataPoint, 0, len(records))
	for _, record := range records {
		item := toDataPoint(record)
		s.enrichDataPointPreview(ctx, projectID, record, &item)
		items = append(items, item)
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
	s.enrichDataPointPreview(ctx, projectID, *record, &dataPoint)
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
	maxPageSize := 200
	if len(normalized.SourceIDs) > 0 {
		maxPageSize = 5000
	}
	page, pageSize := normalizePageAndSize(normalized.Page, normalized.PageSize, 50, maxPageSize)
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
		Quality:            "unknown",
		SourceStatus:       deriveDataPointSourceStatus(record),
		ConsumeMode:        deriveDataPointConsumeMode(record),
		CreatedAt:          record.CreatedAt,
		UpdatedAt:          record.UpdatedAt,
	}
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
	case "calc.output", "db.query", "mqtt.subscription", "mqtt.tag", "kafka.field", "http.request", "websocket.session", "realtime.key", "opcua.node", "modbus.register", "s7.variable":
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
	case "opcua.node":
		return s.isOpcuaNodeDataPointValid(ctx, projectID, record)
	case "modbus.register":
		return s.isModbusRegisterDataPointValid(ctx, projectID, record)
	case "s7.variable":
		return s.isS7VariableDataPointValid(ctx, projectID, record)
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
		outputName = datapointOutputNameFromPath(record.Path)
	}
	for _, output := range extractComputeOutputBindings(*unit) {
		if output.Name == outputName && record.Name == unit.Name && isGeneratedPathMatch(record.Path, output.Path, unit.ID) {
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
	expectedPath := "db." + normalizeDatapointSegment(connection.Name) + "." + normalizeDatapointSegment(query.Name)
	return record.Name == query.Name && isGeneratedPathMatch(record.Path, expectedPath, query.ID), nil
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
	basePath := "realtime." + realtimeDataPointPathSegment(key.KeyPath)
	if provider == "redis" {
		basePath = "redis." + realtimeDataPointPathSegment(key.KeyPath)
	}
	return key.ID == keyID &&
		record.Name == key.KeyPath &&
		isGeneratedPathMatch(record.Path, basePath, key.ID), nil
}

func (s *DataPointService) isOpcuaNodeDataPointValid(ctx context.Context, projectID string, record repository.DataPointRecord) (bool, error) {
	if s.opcuaModeling == nil || s.connections == nil || record.SourceID == nil {
		return true, nil
	}
	node, err := s.opcuaModeling.GetNode(ctx, projectID, strings.TrimSpace(*record.SourceID))
	if isNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	connection, err := s.connections.GetByProjectAndID(ctx, projectID, node.ConnectionID)
	if isNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	expectedPath := "opcua." + normalizeDatapointSegment(connection.Name)
	groups, _ := s.opcuaModeling.ListGroups(ctx, projectID, node.ConnectionID)
	if groupPath := opcuaDataPointGroupPath(groups, node.GroupID); groupPath != "" {
		expectedPath += "." + groupPath
	}
	expectedPath += "." + normalizeDatapointSegment(node.Code)
	return record.Name == node.Name && isGeneratedPathMatch(record.Path, expectedPath, node.ID), nil
}

func (s *DataPointService) isModbusRegisterDataPointValid(ctx context.Context, projectID string, record repository.DataPointRecord) (bool, error) {
	if s.modbusModeling == nil || s.connections == nil || record.SourceID == nil {
		return true, nil
	}
	register, err := s.modbusModeling.GetRegister(ctx, projectID, strings.TrimSpace(*record.SourceID))
	if isNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	connection, err := s.connections.GetByProjectAndID(ctx, projectID, register.ConnectionID)
	if isNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	expectedPath := "modbus." + normalizeDatapointSegment(connection.Name)
	groups, _ := s.modbusModeling.ListGroups(ctx, projectID, register.ConnectionID)
	if groupPath := modbusDataPointGroupPath(groups, register.GroupID); groupPath != "" {
		expectedPath += "." + groupPath
	}
	expectedPath += "." + normalizeDatapointSegment(register.Code)
	return record.Name == register.Name && isGeneratedPathMatch(record.Path, expectedPath, register.ID), nil
}

func (s *DataPointService) isS7VariableDataPointValid(ctx context.Context, projectID string, record repository.DataPointRecord) (bool, error) {
	if s.s7Modeling == nil || s.connections == nil || record.SourceID == nil {
		return true, nil
	}
	connectionID := strings.TrimSpace(firstString(record.SourceConfig, "connectionId"))
	if connectionID == "" {
		return true, nil
	}
	variable, err := s.s7Modeling.GetVariable(ctx, projectID, connectionID, strings.TrimSpace(*record.SourceID))
	if isNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	connection, err := s.connections.GetByProjectAndID(ctx, projectID, variable.ConnectionID)
	if isNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	expectedPath := "s7." + normalizeDatapointSegment(connection.Name)
	groups, _ := s.s7Modeling.ListGroups(ctx, projectID, variable.ConnectionID)
	if groupPath := s7DataPointGroupPath(groups, variable.GroupID); groupPath != "" {
		expectedPath += "." + groupPath
	}
	expectedPath += "." + normalizeDatapointSegment(variable.Code)
	return record.Name == variable.Name && isGeneratedPathMatch(record.Path, expectedPath, variable.ID), nil
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

func realtimeDataPointPathSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unnamed"
	}
	replacer := strings.NewReplacer(":", ".", "/", ".", "\\", ".", " ", "_")
	value = strings.Trim(replacer.Replace(value), ".")
	if value == "" {
		return "unnamed"
	}
	return value
}

func opcuaDataPointGroupPath(groups []repository.OpcuaNodeGroupRecord, groupID *string) string {
	if groupID == nil || strings.TrimSpace(*groupID) == "" {
		return ""
	}
	byID := make(map[string]repository.OpcuaNodeGroupRecord, len(groups))
	for _, group := range groups {
		byID[group.ID] = group
	}
	return buildDataPointGroupPath(strings.TrimSpace(*groupID), func(id string) (string, *string, bool) {
		group, ok := byID[id]
		return group.Name, group.ParentID, ok
	})
}

func modbusDataPointGroupPath(groups []repository.ModbusRegisterGroupRecord, groupID *string) string {
	if groupID == nil || strings.TrimSpace(*groupID) == "" {
		return ""
	}
	byID := make(map[string]repository.ModbusRegisterGroupRecord, len(groups))
	for _, group := range groups {
		byID[group.ID] = group
	}
	return buildDataPointGroupPath(strings.TrimSpace(*groupID), func(id string) (string, *string, bool) {
		group, ok := byID[id]
		return group.Name, group.ParentID, ok
	})
}

func s7DataPointGroupPath(groups []repository.S7VariableGroupRecord, groupID *string) string {
	if groupID == nil || strings.TrimSpace(*groupID) == "" {
		return ""
	}
	byID := make(map[string]repository.S7VariableGroupRecord, len(groups))
	for _, group := range groups {
		byID[group.ID] = group
	}
	return buildDataPointGroupPath(strings.TrimSpace(*groupID), func(id string) (string, *string, bool) {
		group, ok := byID[id]
		return group.Code, group.ParentID, ok
	})
}

func buildDataPointGroupPath(groupID string, resolve func(string) (string, *string, bool)) string {
	segments := make([]string, 0)
	visited := map[string]struct{}{}
	for currentID := groupID; currentID != ""; {
		if _, ok := visited[currentID]; ok {
			break
		}
		visited[currentID] = struct{}{}
		name, parentID, ok := resolve(currentID)
		if !ok {
			break
		}
		segments = append([]string{normalizeDatapointSegment(name)}, segments...)
		if parentID == nil {
			break
		}
		currentID = strings.TrimSpace(*parentID)
	}
	return strings.Join(segments, ".")
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
	case "opcua.node":
		return "OPC UA 变量数据点已失效：变量、接入源不存在，或路径已变更"
	case "modbus.register":
		return "Modbus 变量数据点已失效：变量、接入源不存在，或路径已变更"
	case "s7.variable":
		return "S7 变量数据点已失效：变量、接入源不存在，或路径已变更"
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
