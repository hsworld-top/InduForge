package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/indu-forge/data_service/internal/collectorprotocol"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type CollectorPointStore interface {
	GetConnection(context.Context, string, string) (*repository.CollectorConnectionRecord, error)
	ListPointGroups(context.Context, string, string, *string) ([]repository.CollectorPointGroupRecord, error)
	CreatePointGroup(context.Context, repository.CreateCollectorPointGroupParams) (*repository.CollectorPointGroupRecord, error)
	UpdatePointGroup(context.Context, repository.UpdateCollectorPointGroupParams) (*repository.CollectorPointGroupRecord, error)
	DeletePointGroup(context.Context, string, string, string) error
	ListPoints(context.Context, string, string, repository.CollectorPointListFilter) ([]repository.CollectorPointRecord, int, error)
	ListPointCodes(context.Context, string, string) ([]string, error)
	ListExistingPointAddressTexts(context.Context, string, string, []string) ([]string, error)
	GetPointsByIDs(context.Context, string, string, []string) ([]repository.CollectorPointRecord, error)
	CreatePointsBatch(context.Context, []repository.CreateCollectorPointParams) ([]repository.CollectorPointRecord, error)
	UpdatePointsBatch(context.Context, []repository.UpdateCollectorPointParams) ([]repository.CollectorPointRecord, error)
	DeletePointsBatch(context.Context, string, string, []string) error
	MovePointsBatch(context.Context, string, string, *string, []string) error
}

type CreateCollectorPointInput struct {
	GroupID      *string        `json:"groupId"`
	Code         string         `json:"code"`
	Name         string         `json:"name"`
	Description  *string        `json:"description"`
	Address      map[string]any `json:"address"`
	DataType     string         `json:"dataType"`
	ElementCount int            `json:"elementCount"`
	ReadOptions  map[string]any `json:"readOptions"`
	Acquisition  map[string]any `json:"acquisition"`
	Enabled      *bool          `json:"enabled"`
	SortOrder    int            `json:"sortOrder"`
	Metadata     map[string]any `json:"metadata"`
}

type UpdateCollectorPointInput struct {
	ID string `json:"id"`
	CreateCollectorPointInput
}

type CollectorPointGroup struct {
	ID        string         `json:"id"`
	ParentID  *string        `json:"parentId"`
	Name      string         `json:"name"`
	SortOrder int            `json:"sortOrder"`
	Metadata  map[string]any `json:"metadata"`
}
type CollectorPoint struct {
	ID                   string         `json:"id"`
	GroupID              *string        `json:"groupId"`
	Code                 string         `json:"code"`
	Name                 string         `json:"name"`
	Description          *string        `json:"description"`
	Address              map[string]any `json:"address"`
	AddressText          string         `json:"addressText"`
	AddressSchemaVersion int            `json:"addressSchemaVersion"`
	DataType             string         `json:"dataType"`
	ElementCount         int            `json:"elementCount"`
	ReadOptions          map[string]any `json:"readOptions"`
	Acquisition          map[string]any `json:"acquisition"`
	Enabled              bool           `json:"enabled"`
	SortOrder            int            `json:"sortOrder"`
	Metadata             map[string]any `json:"metadata"`
}
type CollectorPointPage struct {
	List       []CollectorPoint          `json:"list"`
	Pagination CollectorDriverPagination `json:"pagination"`
}

type CollectorPointService struct {
	store          CollectorPointStore
	catalog        *collectorprotocol.Catalog
	addressSchemas map[string]*jsonschema.Schema
}

func NewCollectorPointService(store CollectorPointStore, catalog *collectorprotocol.Catalog) (*CollectorPointService, error) {
	if store == nil || catalog == nil {
		return nil, fmt.Errorf("采集点服务依赖不完整")
	}
	result := &CollectorPointService{store: store, catalog: catalog, addressSchemas: map[string]*jsonschema.Schema{}}
	for _, driver := range catalog.Drivers() {
		var document any
		if err := json.Unmarshal(driver.AddressSchema, &document); err != nil {
			return nil, err
		}
		compiler := jsonschema.NewCompiler()
		compiler.DefaultDraft(jsonschema.Draft2020)
		resource := "urn:induforge:collector:" + driver.Manifest.DriverID + ":address"
		if err := compiler.AddResource(resource, document); err != nil {
			return nil, err
		}
		schema, err := compiler.Compile(resource)
		if err != nil {
			return nil, err
		}
		result.addressSchemas[driver.Manifest.DriverID] = schema
	}
	return result, nil
}

func (s *CollectorPointService) ListPointGroups(ctx context.Context, projectID, connectionID string, parentID *string) ([]CollectorPointGroup, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}
	records, err := s.store.ListPointGroups(ctx, projectID, connectionID, parentID)
	if err != nil {
		return nil, err
	}
	result := make([]CollectorPointGroup, 0, len(records))
	for _, record := range records {
		result = append(result, toCollectorPointGroup(record))
	}
	return result, nil
}
func (s *CollectorPointService) CreatePointGroup(ctx context.Context, projectID, connectionID string, parentID *string, name string, sortOrder int, metadata map[string]any) (*CollectorPointGroup, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 100 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "采集点分组名称长度必须为 1 到 100 个字符")
	}
	record, err := s.store.CreatePointGroup(ctx, repository.CreateCollectorPointGroupParams{ID: uuid.NewString(), ProjectID: projectID, ConnectionID: connectionID, ParentID: parentID, Name: name, SortOrder: sortOrder, Metadata: cloneCollectorMap(metadata)})
	if err != nil {
		return nil, err
	}
	return &CollectorPointGroup{ID: record.ID, ParentID: record.ParentID, Name: record.Name, SortOrder: record.SortOrder, Metadata: record.Metadata}, nil
}
func (s *CollectorPointService) UpdatePointGroup(ctx context.Context, projectID, connectionID, groupID, name string) (*CollectorPointGroup, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}
	if _, err := uuid.Parse(groupID); err != nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "采集点分组 ID 无效")
	}
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 100 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "采集点分组名称长度必须为 1 到 100 个字符")
	}
	record, err := s.store.UpdatePointGroup(ctx, repository.UpdateCollectorPointGroupParams{ID: groupID, ProjectID: projectID, ConnectionID: connectionID, Name: name})
	if err != nil {
		return nil, err
	}
	result := toCollectorPointGroup(*record)
	return &result, nil
}

func (s *CollectorPointService) DeletePointGroup(ctx context.Context, projectID, connectionID, groupID string) error {
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return err
	}
	if _, err := uuid.Parse(groupID); err != nil {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "采集点分组 ID 无效")
	}
	return s.store.DeletePointGroup(ctx, projectID, connectionID, groupID)
}

func (s *CollectorPointService) ListPoints(ctx context.Context, projectID, connectionID string, filter repository.CollectorPointListFilter) (CollectorPointPage, error) {
	if filter.Page < 1 || filter.PageSize < 1 || filter.PageSize > 100 {
		return CollectorPointPage{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "分页参数无效")
	}
	records, total, err := s.store.ListPoints(ctx, projectID, connectionID, filter)
	if err != nil {
		return CollectorPointPage{}, err
	}
	list := make([]CollectorPoint, 0, len(records))
	for _, record := range records {
		list = append(list, toCollectorPoint(record))
	}
	pages := 0
	if total > 0 {
		pages = (total + filter.PageSize - 1) / filter.PageSize
	}
	return CollectorPointPage{List: list, Pagination: CollectorDriverPagination{Page: filter.Page, PageSize: filter.PageSize, Total: total, TotalPages: pages}}, nil
}

// FindExistingPointAddressIndexes 按驱动规则规范化地址后批量判重，避免前端全量读取连接下的采集点。
func (s *CollectorPointService) FindExistingPointAddressIndexes(ctx context.Context, projectID, connectionID string, addresses []map[string]any) ([]int, error) {
	if len(addresses) == 0 {
		return []int{}, nil
	}
	if len(addresses) > 500 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "单次最多检查 500 个采集点地址")
	}
	connection, err := s.store.GetConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	formatted := make([]string, len(addresses))
	unique := make([]string, 0, len(addresses))
	seen := make(map[string]struct{}, len(addresses))
	for index, address := range addresses {
		if err := s.addressSchemas[connection.DriverID].Validate(address); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "采集点地址不符合驱动 Schema", err)
		}
		addressText, err := formatCollectorAddress(connection.DriverID, address)
		if err != nil {
			return nil, err
		}
		formatted[index] = addressText
		if _, exists := seen[addressText]; exists {
			continue
		}
		seen[addressText] = struct{}{}
		unique = append(unique, addressText)
	}
	existing, err := s.store.ListExistingPointAddressTexts(ctx, projectID, connectionID, unique)
	if err != nil {
		return nil, err
	}
	existingSet := make(map[string]struct{}, len(existing))
	for _, addressText := range existing {
		existingSet[addressText] = struct{}{}
	}
	indexes := make([]int, 0, len(existing))
	for index, addressText := range formatted {
		if _, exists := existingSet[addressText]; exists {
			indexes = append(indexes, index)
		}
	}
	return indexes, nil
}

func (s *CollectorPointService) CreatePointsBatch(ctx context.Context, projectID, connectionID, userID string, inputs []CreateCollectorPointInput) ([]CollectorPoint, error) {
	connection, err := s.store.GetConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	existingCodes, err := s.store.ListPointCodes(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	usedCodes := make(map[string]struct{}, len(existingCodes)+len(inputs))
	for _, code := range existingCodes {
		usedCodes[code] = struct{}{}
	}
	params := make([]repository.CreateCollectorPointParams, 0, len(inputs))
	for _, input := range inputs {
		pointID := uuid.NewString()
		value, err := s.buildPointParams(connection, userID, pointID, allocateCollectorPointCode(input.Name, usedCodes), input)
		if err != nil {
			return nil, err
		}
		params = append(params, value)
	}
	records, err := s.store.CreatePointsBatch(ctx, params)
	if err != nil {
		return nil, err
	}
	return mapCollectorPoints(records), nil
}
func (s *CollectorPointService) UpdatePointsBatch(ctx context.Context, projectID, connectionID, userID string, inputs []UpdateCollectorPointInput) ([]CollectorPoint, error) {
	connection, err := s.store.GetConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	pointIDs := make([]string, 0, len(inputs))
	for _, input := range inputs {
		pointIDs = append(pointIDs, input.ID)
	}
	existing, err := s.store.GetPointsByIDs(ctx, projectID, connectionID, pointIDs)
	if err != nil {
		return nil, err
	}
	existingByID := make(map[string]repository.CollectorPointRecord, len(existing))
	for _, point := range existing {
		existingByID[point.ID] = point
	}
	params := make([]repository.UpdateCollectorPointParams, 0, len(inputs))
	for _, input := range inputs {
		if err := validateConnectionID(input.ID); err != nil {
			return nil, err
		}
		current, exists := existingByID[input.ID]
		if !exists {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "采集点不存在")
		}
		value, err := s.buildPointParams(connection, userID, input.ID, current.Code, input.CreateCollectorPointInput)
		if err != nil {
			return nil, err
		}
		params = append(params, value)
	}
	records, err := s.store.UpdatePointsBatch(ctx, params)
	if err != nil {
		return nil, err
	}
	return mapCollectorPoints(records), nil
}
func (s *CollectorPointService) DeletePointsBatch(ctx context.Context, projectID, connectionID string, pointIDs []string) error {
	for _, id := range pointIDs {
		if err := validateConnectionID(id); err != nil {
			return err
		}
	}
	return s.store.DeletePointsBatch(ctx, projectID, connectionID, pointIDs)
}
func (s *CollectorPointService) MovePointsBatch(ctx context.Context, projectID, connectionID string, groupID *string, pointIDs []string) error {
	if groupID != nil {
		if err := validateConnectionID(*groupID); err != nil {
			return err
		}
	}
	for _, id := range pointIDs {
		if err := validateConnectionID(id); err != nil {
			return err
		}
	}
	return s.store.MovePointsBatch(ctx, projectID, connectionID, groupID, pointIDs)
}

func (s *CollectorPointService) buildPointParams(connection *repository.CollectorConnectionRecord, userID, id, code string, input CreateCollectorPointInput) (repository.CreateCollectorPointParams, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" || len([]rune(name)) > 200 {
		return repository.CreateCollectorPointParams{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "采集点名称长度必须为 1 到 200 个字符")
	}
	driver, _ := s.catalog.Driver(connection.DriverID)
	if !containsFold(driver.Manifest.DataTypes, input.DataType) {
		return repository.CreateCollectorPointParams{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "驱动不支持该平台数据类型")
	}
	if err := s.addressSchemas[connection.DriverID].Validate(input.Address); err != nil {
		return repository.CreateCollectorPointParams{}, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "采集点地址不符合驱动 Schema", err)
	}
	addressText, err := formatCollectorAddress(connection.DriverID, input.Address)
	if err != nil {
		return repository.CreateCollectorPointParams{}, err
	}
	elementCount := input.ElementCount
	if elementCount < 1 {
		elementCount = 1
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	acquisition := cloneCollectorMap(input.Acquisition)
	if len(acquisition) == 0 {
		acquisition = map[string]any{"mode": "polling", "intervalMs": 1000, "timeoutMs": 3000, "deadband": nil, "changeOnly": false, "priority": "normal"}
	}
	return repository.CreateCollectorPointParams{ID: id, ProjectID: connection.ProjectID, ConnectionID: connection.ID, ConnectionCode: connection.Code, UserID: userID, GroupID: input.GroupID, Code: code, Name: name, Description: input.Description, Address: cloneCollectorMap(input.Address), AddressText: addressText, AddressSchemaVersion: connection.SchemaVersion, DataType: input.DataType, ElementCount: elementCount, ReadOptions: cloneCollectorMap(input.ReadOptions), Acquisition: acquisition, Enabled: enabled, SortOrder: input.SortOrder, Metadata: cloneCollectorMap(input.Metadata)}, nil
}

func formatCollectorAddress(driverID string, address map[string]any) (string, error) {
	switch driverID {
	case "opcua.standard":
		value, _ := address["nodeId"].(string)
		return value, nil
	case "modbus.tcp", "modbus.rtu":
		station := numberText(address["station"])
		area, _ := address["area"].(string)
		offset := numberText(address["address"])
		result := station + ":" + area + ":" + offset
		if bit, ok := address["bitIndex"]; ok {
			result += "." + numberText(bit)
		}
		return result, nil
	case "siemens.s7-tcp":
		area, _ := address["area"].(string)
		offset := numberText(address["byteOffset"])
		if area == "dataBlock" {
			area = "DB" + numberText(address["dbNumber"])
		}
		result := area + ":" + offset
		if bit, ok := address["bitOffset"]; ok {
			result += "." + numberText(bit)
		}
		return result, nil
	default:
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "驱动缺少地址格式化器")
	}
}
func numberText(value any) string {
	switch typed := value.(type) {
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case int:
		return strconv.Itoa(typed)
	case json.Number:
		return typed.String()
	default:
		return fmt.Sprintf("%v", typed)
	}
}
func mapCollectorPoints(records []repository.CollectorPointRecord) []CollectorPoint {
	result := make([]CollectorPoint, 0, len(records))
	for _, record := range records {
		result = append(result, toCollectorPoint(record))
	}
	return result
}
func toCollectorPoint(record repository.CollectorPointRecord) CollectorPoint {
	return CollectorPoint{ID: record.ID, GroupID: record.GroupID, Code: record.Code, Name: record.Name, Description: record.Description, Address: record.Address, AddressText: record.AddressText, AddressSchemaVersion: record.AddressSchemaVersion, DataType: record.DataType, ElementCount: record.ElementCount, ReadOptions: record.ReadOptions, Acquisition: record.Acquisition, Enabled: record.Enabled, SortOrder: record.SortOrder, Metadata: record.Metadata}
}

func toCollectorPointGroup(record repository.CollectorPointGroupRecord) CollectorPointGroup {
	return CollectorPointGroup{ID: record.ID, ParentID: record.ParentID, Name: record.Name, SortOrder: record.SortOrder, Metadata: record.Metadata}
}
