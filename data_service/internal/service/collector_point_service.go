package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
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
	ListExistingPointConflicts(context.Context, string, string, []string, []string) (repository.CollectorPointExistingConflicts, error)
	StreamPointsForExport(context.Context, string, string, repository.CollectorPointExportFilter, func(repository.CollectorPointExportRecord) error) error
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

type CollectorPointBatchFailure struct {
	Index   int    `json:"index"`
	Name    string `json:"name"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type CollectorPointBatchResult struct {
	List   []CollectorPoint             `json:"list"`
	Failed []CollectorPointBatchFailure `json:"failed"`
}

type CollectorPointGroup struct {
	ID        string         `json:"id"`
	ParentID  *string        `json:"parentId"`
	Name      string         `json:"name"`
	SortOrder int            `json:"sortOrder"`
	Metadata  map[string]any `json:"metadata"`
}
type CollectorPointDebugSnapshot struct {
	Value             any     `json:"value"`
	ValueText         *string `json:"valueText"`
	DataType          *string `json:"dataType"`
	Quality           *string `json:"quality"`
	SourceTimestamp   *string `json:"sourceTimestamp"`
	ServerTimestamp   *string `json:"serverTimestamp"`
	ReadAt            *string `json:"readAt"`
	LastAttemptStatus string  `json:"lastAttemptStatus"`
	LastAttemptAt     string  `json:"lastAttemptAt"`
	LastErrorCode     *string `json:"lastErrorCode"`
	LastErrorMessage  *string `json:"lastErrorMessage"`
}

type CollectorPoint struct {
	ID                   string                       `json:"id"`
	GroupID              *string                      `json:"groupId"`
	Code                 string                       `json:"code"`
	Name                 string                       `json:"name"`
	Description          *string                      `json:"description"`
	Address              map[string]any               `json:"address"`
	AddressText          string                       `json:"addressText"`
	AddressSchemaVersion int                          `json:"addressSchemaVersion"`
	DataType             string                       `json:"dataType"`
	ElementCount         int                          `json:"elementCount"`
	ReadOptions          map[string]any               `json:"readOptions"`
	Acquisition          map[string]any               `json:"acquisition"`
	Enabled              bool                         `json:"enabled"`
	SortOrder            int                          `json:"sortOrder"`
	Metadata             map[string]any               `json:"metadata"`
	LatestDebugSnapshot  *CollectorPointDebugSnapshot `json:"latestDebugSnapshot"`
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

func (s *CollectorPointService) CreatePointsBatch(ctx context.Context, projectID, connectionID, userID string, inputs []CreateCollectorPointInput) (CollectorPointBatchResult, error) {
	result := CollectorPointBatchResult{List: []CollectorPoint{}, Failed: []CollectorPointBatchFailure{}}
	if len(inputs) == 0 {
		return result, nil
	}
	connection, err := s.store.GetConnection(ctx, projectID, connectionID)
	if err != nil {
		return CollectorPointBatchResult{}, err
	}
	existingCodes, err := s.store.ListPointCodes(ctx, projectID, connectionID)
	if err != nil {
		return CollectorPointBatchResult{}, err
	}
	usedCodes := make(map[string]struct{}, len(existingCodes)+len(inputs))
	for _, code := range existingCodes {
		usedCodes[code] = struct{}{}
	}
	type candidate struct {
		index int
		point repository.CreateCollectorPointParams
	}
	candidates := make([]candidate, 0, len(inputs))
	for index, input := range inputs {
		pointID := uuid.NewString()
		value, buildErr := s.buildPointParams(connection, userID, pointID, allocateCollectorPointCode(input.Name, usedCodes), input)
		if buildErr != nil {
			result.Failed = append(result.Failed, CollectorPointBatchFailure{Index: index, Name: strings.TrimSpace(input.Name), Code: "VALIDATION_FAILED", Message: buildErr.Error()})
			continue
		}
		candidates = append(candidates, candidate{index: index, point: value})
	}
	if len(candidates) == 0 {
		sort.Slice(result.Failed, func(i, j int) bool { return result.Failed[i].Index < result.Failed[j].Index })
		return result, nil
	}
	names := make([]string, 0, len(candidates))
	addresses := make([]string, 0, len(candidates))
	for _, item := range candidates {
		names = append(names, strings.ToLower(item.point.Name))
		addresses = append(addresses, item.point.AddressText)
	}
	conflicts, err := s.store.ListExistingPointConflicts(ctx, projectID, connectionID, names, addresses)
	if err != nil {
		return CollectorPointBatchResult{}, err
	}
	existingNames := make(map[string]struct{}, len(conflicts.Names))
	for _, name := range conflicts.Names {
		existingNames[strings.ToLower(name)] = struct{}{}
	}
	existingAddresses := make(map[string]struct{}, len(conflicts.AddressTexts))
	for _, address := range conflicts.AddressTexts {
		existingAddresses[address] = struct{}{}
	}
	valid := make([]candidate, 0, len(candidates))
	params := make([]repository.CreateCollectorPointParams, 0, len(candidates))
	seenNames := make(map[string]struct{}, len(candidates))
	seenAddresses := make(map[string]struct{}, len(candidates))
	for _, item := range candidates {
		nameKey := strings.ToLower(item.point.Name)
		if _, exists := existingNames[nameKey]; exists {
			result.Failed = append(result.Failed, CollectorPointBatchFailure{Index: item.index, Name: item.point.Name, Code: "DUPLICATE_NAME", Message: "变量名称已存在：" + item.point.Name})
			continue
		}
		if _, exists := existingAddresses[item.point.AddressText]; exists {
			result.Failed = append(result.Failed, CollectorPointBatchFailure{Index: item.index, Name: item.point.Name, Code: "DUPLICATE_ADDRESS", Message: "变量地址已存在：" + item.point.AddressText})
			continue
		}
		if _, exists := seenNames[nameKey]; exists {
			result.Failed = append(result.Failed, CollectorPointBatchFailure{Index: item.index, Name: item.point.Name, Code: "DUPLICATE_NAME", Message: "变量名称已存在：" + item.point.Name})
			continue
		}
		if _, exists := seenAddresses[item.point.AddressText]; exists {
			result.Failed = append(result.Failed, CollectorPointBatchFailure{Index: item.index, Name: item.point.Name, Code: "DUPLICATE_ADDRESS", Message: "变量地址已存在：" + item.point.AddressText})
			continue
		}
		seenNames[nameKey] = struct{}{}
		seenAddresses[item.point.AddressText] = struct{}{}
		valid = append(valid, item)
		params = append(params, item.point)
	}
	records, err := s.store.CreatePointsBatch(ctx, params)
	if err != nil {
		return CollectorPointBatchResult{}, err
	}
	result.List = mapCollectorPoints(records)
	createdIDs := make(map[string]struct{}, len(records))
	for _, record := range records {
		createdIDs[record.ID] = struct{}{}
	}
	for _, item := range valid {
		if _, created := createdIDs[item.point.ID]; !created {
			result.Failed = append(result.Failed, CollectorPointBatchFailure{Index: item.index, Name: item.point.Name, Code: "CONFLICT", Message: "变量名称或地址已存在：" + item.point.Name})
		}
	}
	sort.Slice(result.Failed, func(i, j int) bool { return result.Failed[i].Index < result.Failed[j].Index })
	return result, nil
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
	if connection.DriverID == "opcua.standard" {
		nodeID, err := normalizeOpcUaNodeID(input.Address["nodeId"])
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["nodeId"] = nodeID
	}
	if err := s.addressSchemas[connection.DriverID].Validate(input.Address); err != nil {
		return repository.CreateCollectorPointParams{}, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "采集点地址不符合驱动 Schema", err)
	}
	if connection.DriverID == "modbus.tcp" || connection.DriverID == "modbus.rtu" {
		if err := validateModbusPointAddress(input.Address, input.DataType, input.ElementCount); err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
	}
	if connection.DriverID == "siemens.s7-tcp" {
		if err := validateSiemensS7PointAddress(input.Address, input.DataType, input.ElementCount); err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
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

// validateModbusPointAddress 校验 Modbus 区域、平台数据类型和寄存器位索引的组合，避免保存驱动无法读取的变量。
func validateModbusPointAddress(address map[string]any, dataType string, elementCount int) error {
	area, _ := address["area"].(string)
	_, hasBitIndex := address["bitIndex"]
	isBoolean := strings.EqualFold(dataType, "bool")

	switch area {
	case "coil", "discreteInput":
		if !isBoolean {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "线圈和离散输入只支持 bool 数据类型")
		}
		if hasBitIndex {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "线圈和离散输入不能配置寄存器位索引")
		}
	case "inputRegister", "holdingRegister":
		if isBoolean && !hasBitIndex {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "寄存器 bool 变量必须配置位索引")
		}
		if !isBoolean && hasBitIndex {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "只有寄存器 bool 变量可以配置位索引")
		}
	}
	if elementCount < 1 {
		elementCount = 1
	}
	protocolUnits := int64(elementCount)
	switch strings.ToLower(dataType) {
	case "bool":
		if hasBitIndex {
			bitIndex, _ := collectorNumericInt(address["bitIndex"])
			protocolUnits = (int64(bitIndex) + int64(elementCount) + 15) / 16
		}
	case "int8", "uint8", "string", "bytes":
		protocolUnits = (int64(elementCount) + 1) / 2
	case "int32", "uint32", "float32":
		protocolUnits = int64(elementCount) * 2
	case "int64", "uint64", "float64":
		protocolUnits = int64(elementCount) * 4
	}
	maximumUnits := int64(125)
	if area == "coil" || area == "discreteInput" {
		maximumUnits = 2000
	}
	if protocolUnits > maximumUnits {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Modbus 单变量读取长度超过协议上限")
	}
	return nil
}

func collectorNumericInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case float64:
		return int(typed), true
	case json.Number:
		parsed, err := strconv.Atoi(typed.String())
		return parsed, err == nil
	default:
		return 0, false
	}
}

// validateSiemensS7PointAddress 校验 S7 区域、数据类型和位偏移组合，规则与 DevAgent 驱动保持一致。
func validateSiemensS7PointAddress(address map[string]any, dataType string, elementCount int) error {
	area, _ := address["area"].(string)
	_, hasBitOffset := address["bitOffset"]
	isBoolean := strings.EqualFold(dataType, "bool")

	if isBoolean {
		if area == "timer" || area == "counter" || !hasBitOffset {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Siemens S7 bool 变量必须配置可位寻址区域和位偏移")
		}
	} else if hasBitOffset {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "只有 Siemens S7 bool 变量可以配置位偏移")
	}

	if area == "timer" || area == "counter" {
		if !strings.EqualFold(dataType, "int16") && !strings.EqualFold(dataType, "uint16") {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "定时器和计数器只支持 int16 或 uint16 数据类型")
		}
	}
	if strings.EqualFold(dataType, "datetime") && elementCount > 1 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Siemens S7 datetime 变量只支持单元素读取")
	}
	return nil
}

// normalizeOpcUaNodeID 在保存阶段校验 OPC UA NodeId，避免无效地址进入变量模型后才在调试读取时失败。
func normalizeOpcUaNodeID(value any) (string, error) {
	nodeID, ok := value.(string)
	nodeID = strings.TrimSpace(nodeID)
	if !ok || nodeID == "" {
		return "", invalidOpcUaNodeID()
	}

	identifier := nodeID
	if strings.HasPrefix(identifier, "ns=") {
		separator := strings.IndexByte(identifier, ';')
		if separator <= len("ns=") {
			return "", invalidOpcUaNodeID()
		}
		if _, err := strconv.ParseUint(identifier[len("ns="):separator], 10, 16); err != nil {
			return "", invalidOpcUaNodeID()
		}
		identifier = identifier[separator+1:]
	}

	if len(identifier) < 3 || identifier[1] != '=' || identifier[2:] == "" {
		return "", invalidOpcUaNodeID()
	}
	valuePart := identifier[2:]
	switch identifier[0] {
	case 'i':
		if _, err := strconv.ParseUint(valuePart, 10, 32); err != nil {
			return "", invalidOpcUaNodeID()
		}
	case 's':
		// 字符串标识允许包含空格和分号，只要求 s= 后存在内容。
	case 'g':
		if _, err := uuid.Parse(valuePart); err != nil {
			return "", invalidOpcUaNodeID()
		}
	case 'b':
		if _, err := base64.StdEncoding.DecodeString(valuePart); err != nil {
			return "", invalidOpcUaNodeID()
		}
	default:
		return "", invalidOpcUaNodeID()
	}
	return nodeID, nil
}

func invalidOpcUaNodeID() error {
	return apperrors.NewAppError(
		apperrors.ErrorCodeBadRequest,
		http.StatusBadRequest,
		"OPC UA NodeId 格式无效，请使用 i=111、ns=2;i=111、s=LastChange 或 ns=2;s=LastChange 等格式",
	)
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
		prefix := map[string]string{
			"input":     "I",
			"output":    "Q",
			"marker":    "M",
			"dataBlock": "DB" + numberText(address["dbNumber"]) + ".",
			"timer":     "T",
			"counter":   "C",
		}[area]
		if prefix == "" {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Siemens S7 地址区域无效")
		}
		result := prefix + offset
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
	point := CollectorPoint{ID: record.ID, GroupID: record.GroupID, Code: record.Code, Name: record.Name, Description: record.Description, Address: record.Address, AddressText: record.AddressText, AddressSchemaVersion: record.AddressSchemaVersion, DataType: record.DataType, ElementCount: record.ElementCount, ReadOptions: record.ReadOptions, Acquisition: record.Acquisition, Enabled: record.Enabled, SortOrder: record.SortOrder, Metadata: record.Metadata}
	if snapshot := record.LatestDebugSnapshot; snapshot != nil {
		point.LatestDebugSnapshot = &CollectorPointDebugSnapshot{
			Value: snapshot.Value, ValueText: snapshot.ValueText, DataType: snapshot.DataType, Quality: snapshot.Quality,
			SourceTimestamp: optionalCollectorTime(snapshot.SourceTimestamp), ServerTimestamp: optionalCollectorTime(snapshot.ServerTimestamp), ReadAt: optionalCollectorTime(snapshot.ReadAt),
			LastAttemptStatus: snapshot.LastAttemptStatus, LastAttemptAt: formatCollectorTime(snapshot.LastAttemptAt),
			LastErrorCode: snapshot.LastErrorCode, LastErrorMessage: snapshot.LastErrorMessage,
		}
	}
	return point
}

func toCollectorPointGroup(record repository.CollectorPointGroupRecord) CollectorPointGroup {
	return CollectorPointGroup{ID: record.ID, ParentID: record.ParentID, Name: record.Name, SortOrder: record.SortOrder, Metadata: record.Metadata}
}
