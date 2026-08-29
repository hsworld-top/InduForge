package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

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
	GroupID              *string        `json:"groupId"`
	Code                 string         `json:"code"`
	Name                 string         `json:"name"`
	Description          *string        `json:"description"`
	Address              map[string]any `json:"address"`
	DataType             string         `json:"dataType"`
	ElementCount         int            `json:"elementCount"`
	ReadOptions          map[string]any `json:"readOptions"`
	AcquisitionMode      string         `json:"acquisitionMode,omitempty"`
	AcquisitionOverrides map[string]any `json:"acquisitionOverrides,omitempty"`
	Enabled              *bool          `json:"enabled"`
	SortOrder            int            `json:"sortOrder"`
	Metadata             map[string]any `json:"metadata"`
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
	ID                    string                       `json:"id"`
	GroupID               *string                      `json:"groupId"`
	Code                  string                       `json:"code"`
	Name                  string                       `json:"name"`
	Description           *string                      `json:"description"`
	Address               map[string]any               `json:"address"`
	AddressText           string                       `json:"addressText"`
	AddressSchemaVersion  int                          `json:"addressSchemaVersion"`
	DataType              string                       `json:"dataType"`
	DataPointID           string                       `json:"dataPointId"`
	DataPointPath         string                       `json:"dataPointPath"`
	DataPointDefaultValue *string                      `json:"dataPointDefaultValue"`
	ElementCount          int                          `json:"elementCount"`
	ReadOptions           map[string]any               `json:"readOptions"`
	Acquisition           map[string]any               `json:"acquisition"`
	AcquisitionMode       string                       `json:"acquisitionMode"`
	AcquisitionOverrides  map[string]any               `json:"acquisitionOverrides"`
	Enabled               bool                         `json:"enabled"`
	SortOrder             int                          `json:"sortOrder"`
	Metadata              map[string]any               `json:"metadata"`
	LatestDebugSnapshot   *CollectorPointDebugSnapshot `json:"latestDebugSnapshot"`
}
type CollectorPointPage struct {
	List       []CollectorPoint          `json:"list"`
	Pagination CollectorDriverPagination `json:"pagination"`
}

type CollectorAddressNormalizationResult struct {
	Address          map[string]any      `json:"address"`
	AddressText      string              `json:"addressText"`
	AllowedDataTypes []string            `json:"allowedDataTypes"`
	Errors           []map[string]string `json:"errors"`
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

func (s *CollectorPointService) NormalizeAddress(ctx context.Context, projectID, connectionID string, address map[string]any, dataType string, elementCount int) (*CollectorAddressNormalizationResult, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}
	connection, err := s.store.GetConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	driver, ok := s.catalog.Driver(connection.DriverID)
	if !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "采集驱动不存在")
	}
	result := &CollectorAddressNormalizationResult{Address: cloneCollectorMap(address), AllowedDataTypes: append([]string{}, driver.Manifest.DataTypes...), Errors: []map[string]string{}}
	input := CreateCollectorPointInput{Name: "地址预览", Address: cloneCollectorMap(address), DataType: dataType, ElementCount: elementCount}
	params, buildErr := s.buildPointParams(connection, uuid.Nil.String(), uuid.NewString(), "address_preview", input)
	if buildErr != nil {
		result.Errors = append(result.Errors, map[string]string{"field": "address", "message": buildErr.Error()})
		return result, nil
	}
	result.Address = params.Address
	result.AddressText = params.AddressText
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
	candidates := make([]repository.CreateCollectorPointParams, 0, len(inputs))
	for index, input := range inputs {
		pointID := uuid.NewString()
		value, buildErr := s.buildPointParams(connection, userID, pointID, allocateCollectorPointCode(input.Name, usedCodes), input)
		if buildErr != nil {
			result.Failed = append(result.Failed, CollectorPointBatchFailure{Index: index, Name: strings.TrimSpace(input.Name), Code: "VALIDATION_FAILED", Message: buildErr.Error()})
			continue
		}
		candidates = append(candidates, value)
	}
	if len(result.Failed) > 0 {
		sort.Slice(result.Failed, func(i, j int) bool { return result.Failed[i].Index < result.Failed[j].Index })
		return result, nil
	}
	names := make([]string, 0, len(candidates))
	addresses := make([]string, 0, len(candidates))
	for _, item := range candidates {
		names = append(names, strings.ToLower(item.Name))
		addresses = append(addresses, item.AddressText)
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
	params := make([]repository.CreateCollectorPointParams, 0, len(candidates))
	seenNames := make(map[string]struct{}, len(candidates))
	seenAddresses := make(map[string]struct{}, len(candidates))
	for index, item := range candidates {
		nameKey := strings.ToLower(item.Name)
		if _, exists := existingNames[nameKey]; exists {
			result.Failed = append(result.Failed, CollectorPointBatchFailure{Index: index, Name: item.Name, Code: "DUPLICATE_NAME", Message: "变量名称已存在：" + item.Name})
			continue
		}
		if _, exists := existingAddresses[item.AddressText]; exists {
			result.Failed = append(result.Failed, CollectorPointBatchFailure{Index: index, Name: item.Name, Code: "DUPLICATE_ADDRESS", Message: "变量地址已存在：" + item.AddressText})
			continue
		}
		if _, exists := seenNames[nameKey]; exists {
			result.Failed = append(result.Failed, CollectorPointBatchFailure{Index: index, Name: item.Name, Code: "DUPLICATE_NAME", Message: "变量名称已存在：" + item.Name})
			continue
		}
		if _, exists := seenAddresses[item.AddressText]; exists {
			result.Failed = append(result.Failed, CollectorPointBatchFailure{Index: index, Name: item.Name, Code: "DUPLICATE_ADDRESS", Message: "变量地址已存在：" + item.AddressText})
			continue
		}
		seenNames[nameKey] = struct{}{}
		seenAddresses[item.AddressText] = struct{}{}
		params = append(params, item)
	}
	if len(result.Failed) > 0 {
		sort.Slice(result.Failed, func(i, j int) bool { return result.Failed[i].Index < result.Failed[j].Index })
		return result, nil
	}
	records, err := s.store.CreatePointsBatch(ctx, params)
	if err != nil {
		return CollectorPointBatchResult{}, err
	}
	result.List = mapCollectorPoints(records)
	return result, nil
}

func formatCollectorBatchFailures(failures []CollectorPointBatchFailure) string {
	parts := make([]string, 0, len(failures))
	for _, failure := range failures {
		parts = append(parts, fmt.Sprintf("第 %d 行 %s", failure.Index+1, failure.Message))
	}
	return "批量创建未写入任何变量：" + strings.Join(parts, "；")
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
		if input.AcquisitionMode == "" && input.AcquisitionOverrides == nil {
			input.AcquisitionMode = current.AcquisitionMode
			input.AcquisitionOverrides = cloneCollectorMap(current.AcquisitionOverrides)
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
	if connection.DriverID == "ge.srtp-tcp" {
		address, err := normalizeGeSrtpAddress(input.Address["address"])
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if strings.HasPrefix(connection.DriverID, "inovance.modbus-") {
		address, err := normalizeInovanceAddress(input.Address["address"])
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if connection.DriverID == "inovance.connected-cip" || connection.DriverID == "inovance.easy-net" || connection.DriverID == "inovance.computer-link" {
		address, err := normalizeInovanceSpecialAddress(input.Address["address"], connection.DriverID == "inovance.connected-cip")
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if connection.DriverID == "fatek.program-tcp" || connection.DriverID == "fatek.program-serial" {
		address, err := normalizeFatekProgramAddress(input.Address["address"], input.DataType)
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if strings.HasPrefix(connection.DriverID, "freedom.") {
		requestHex, err := normalizeFreedomRequestHex(input.Address["requestHex"])
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["requestHex"] = requestHex
	}
	if connection.DriverID == "cimon.hmi-protocol" {
		address, err := normalizeCimonAddress(input.Address["address"])
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if strings.HasPrefix(connection.DriverID, "keyence.") {
		address, err := normalizeKeyenceAddress(input.Address["address"], connection.DriverID)
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if strings.HasPrefix(connection.DriverID, "delta.") {
		address, err := normalizeDeltaAddress(input.Address["address"])
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if strings.HasPrefix(connection.DriverID, "xinje.") {
		address, err := normalizeXinjeAddress(input.Address["address"])
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if strings.HasPrefix(connection.DriverID, "megmeet.") {
		address, err := normalizeMegMeetAddress(input.Address["address"])
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if strings.HasPrefix(connection.DriverID, "fuji.") {
		address, err := normalizeFujiAddress(input.Address["address"], connection.DriverID)
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if strings.HasPrefix(connection.DriverID, "vigor.") {
		address, err := normalizeVigorAddress(input.Address["address"])
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if connection.DriverID == "yokogawa.link-tcp" {
		address, err := normalizeYokogawaAddress(input.Address["address"], input.DataType)
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if strings.HasPrefix(connection.DriverID, "yamatake.") {
		address, err := normalizeYamatakeAddress(input.Address["address"])
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if strings.HasPrefix(connection.DriverID, "yaskawa.") {
		address, err := normalizeYaskawaAddress(input.Address["address"])
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if strings.HasPrefix(connection.DriverID, "cjt188.") {
		address, err := normalizeCjt188Address(input.Address["address"])
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if strings.HasPrefix(connection.DriverID, "rkc.temperature-controller-") {
		address, err := normalizeRkcAddress(input.Address["address"])
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if connection.DriverID == "dam3601.serial" {
		address, err := normalizeDam3601Address(input.Address["address"])
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if connection.DriverID == "yudian.ai-bus" {
		address, err := normalizeYuDianAddress(input.Address["address"])
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if connection.DriverID == "delixi.dtsu6606" {
		address, err := normalizeDtsu6606Address(input.Address["address"])
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if strings.HasPrefix(connection.DriverID, "dlt645.") || strings.HasPrefix(connection.DriverID, "dlt698.") {
		byteCount := 4
		if strings.HasPrefix(connection.DriverID, "dlt645.1997-") {
			byteCount = 2
		}
		address, err := normalizeDltAddress(input.Address["address"], byteCount)
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if connection.DriverID == "dcs.nanjing-auto" {
		address, err := normalizeDcsNanJingAutoAddress(input.Address["address"])
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if strings.HasPrefix(connection.DriverID, "robot.") {
		address, err := normalizeRobotAddress(input.Address["address"])
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if connection.DriverID == "siemens.s7-plus" || connection.DriverID == "mqtt.rpc-device" {
		address, err := normalizeOpaqueCollectorAddress(input.Address["address"], 256, "变量地址无效")
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if connection.DriverID == "oriental-motor.eip" {
		address, err := normalizeOrientalMotorAddress(input.Address["address"])
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if connection.DriverID == "toyo.puc" {
		address, err := normalizeToyoPucAddress(input.Address["address"])
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if connection.DriverID == "turck.reader-tcp" {
		address, err := normalizeTurckAddress(input.Address["address"])
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if connection.DriverID == "omron.cip" || connection.DriverID == "omron.connected-cip" ||
		connection.DriverID == "omron.hostlink" || connection.DriverID == "omron.hostlink-over-tcp" ||
		connection.DriverID == "omron.hostlink-cmode" || connection.DriverID == "omron.hostlink-cmode-over-tcp" {
		cMode := strings.Contains(connection.DriverID, "hostlink")
		address, err := normalizeOmronVariantAddress(input.Address["address"], cMode)
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if isMitsubishiVariantDriver(connection.DriverID) {
		address, err := normalizeMitsubishiNetworkAddress(input.Address["address"], connection.DriverID == "mitsubishi.cip")
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if err := s.addressSchemas[connection.DriverID].Validate(input.Address); err != nil {
		return repository.CreateCollectorPointParams{}, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "采集点地址不符合驱动 Schema", err)
	}
	if strings.HasPrefix(connection.DriverID, "modbus.") {
		if err := validateModbusPointAddress(input.Address, input.DataType, input.ElementCount); err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
	}
	if connection.DriverID == "siemens.s7-tcp" || connection.DriverID == "siemens.ppi" ||
		connection.DriverID == "siemens.ppi-over-tcp" || connection.DriverID == "siemens.mpi" ||
		connection.DriverID == "siemens.fetch-write" {
		if err := validateSiemensS7PointAddress(input.Address, input.DataType, input.ElementCount); err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
	}
	if connection.DriverID == "mitsubishi.mc-3e-tcp" || isMitsubishiNativeDriver(connection.DriverID) {
		if err := validateMitsubishiMcPointAddress(input.Address, input.DataType, input.ElementCount); err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
	}
	if connection.DriverID == "mitsubishi.cip" {
		if err := validateMitsubishiCipPointAddress(input.DataType, input.ElementCount); err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
	}
	if connection.DriverID == "omron.fins-tcp" || connection.DriverID == "omron.fins-udp" {
		if err := validateOmronFinsPointAddress(input.Address, input.DataType, input.ElementCount); err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
	}
	if connection.DriverID == "omron.cip" || connection.DriverID == "omron.connected-cip" ||
		connection.DriverID == "omron.hostlink" || connection.DriverID == "omron.hostlink-over-tcp" ||
		connection.DriverID == "omron.hostlink-cmode" || connection.DriverID == "omron.hostlink-cmode-over-tcp" {
		if err := validateOmronVariantPointAddress(input.DataType, input.ElementCount); err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
	}
	if connection.DriverID == "allen-bradley.ethernet-ip" ||
		connection.DriverID == "allen-bradley.connected-cip" ||
		connection.DriverID == "allen-bradley.micro-cip" {
		if err := validateAllenBradleyPointAddress(input.Address, input.DataType, input.ElementCount); err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
	}
	if connection.DriverID == "allen-bradley.pccc" ||
		connection.DriverID == "allen-bradley.slc" ||
		connection.DriverID == "allen-bradley.df1-serial" {
		if err := validateAllenBradleyLegacyPointAddress(input.Address, input.DataType, input.ElementCount); err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
	}
	if connection.DriverID == "beckhoff.ads-tcp" {
		if err := validateBeckhoffAdsPointAddress(input.Address, input.DataType, input.ElementCount); err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
	}
	if connection.DriverID == "iec.60870-5-104" {
		if err := validateIec104PointAddress(input.Address, input.DataType, input.ElementCount); err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
	}
	if connection.DriverID == "panasonic.mewtocol-tcp" || connection.DriverID == "panasonic.mewtocol-serial" {
		if err := validatePanasonicMewtocolPointAddress(input.Address, input.ElementCount); err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
	}
	if connection.DriverID == "panasonic.mc-binary-tcp" {
		address, err := normalizePanasonicMcAddress(input.Address["address"], input.DataType)
		if err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
		input.Address = cloneCollectorMap(input.Address)
		input.Address["address"] = address
	}
	if strings.HasPrefix(connection.DriverID, "lsis.") {
		if err := validateLsisPointAddress(input.Address, input.ElementCount); err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
	}
	if connection.DriverID == "ge.srtp-tcp" {
		if err := validateGeSrtpPointAddress(input.Address, input.DataType, input.ElementCount); err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
	}
	if strings.HasPrefix(connection.DriverID, "inovance.modbus-") {
		if err := validateInovancePointAddress(input.DataType, input.ElementCount); err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
	}
	if connection.DriverID == "inovance.connected-cip" || connection.DriverID == "inovance.easy-net" || connection.DriverID == "inovance.computer-link" {
		if err := validateInovanceSpecialPointAddress(input.DataType, input.ElementCount, connection.DriverID == "inovance.connected-cip"); err != nil {
			return repository.CreateCollectorPointParams{}, err
		}
	}
	if connection.DriverID == "fatek.program-tcp" && input.ElementCount > 65535 {
		return repository.CreateCollectorPointParams{}, apperrors.NewAppError(
			apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "永宏编程口元素数量不能超过 65535")
	}
	if strings.HasPrefix(connection.DriverID, "keyence.") && input.ElementCount > 65535 {
		return repository.CreateCollectorPointParams{}, apperrors.NewAppError(
			apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "基恩士元素数量不能超过 65535")
	}
	if strings.HasPrefix(connection.DriverID, "delta.") && input.ElementCount > 65535 {
		return repository.CreateCollectorPointParams{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "台达元素数量不能超过 65535")
	}
	if (strings.HasPrefix(connection.DriverID, "freedom.") || connection.DriverID == "cimon.hmi-protocol") && input.ElementCount > 65535 {
		return repository.CreateCollectorPointParams{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "元素数量不能超过 65535")
	}
	if (strings.HasPrefix(connection.DriverID, "yamatake.") || strings.HasPrefix(connection.DriverID, "yaskawa.")) && input.ElementCount > 65535 {
		return repository.CreateCollectorPointParams{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "元素数量不能超过 65535")
	}
	if (connection.DriverID == "oriental-motor.eip" || connection.DriverID == "toyo.puc" || connection.DriverID == "turck.reader-tcp") && input.ElementCount > 65535 {
		return repository.CreateCollectorPointParams{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "元素数量不能超过 65535")
	}
	if strings.HasPrefix(connection.DriverID, "cjt188.") && input.ElementCount > 255 {
		return repository.CreateCollectorPointParams{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "CJT188 元素数量不能超过 255")
	}
	if strings.HasPrefix(connection.DriverID, "rkc.temperature-controller-") && input.ElementCount != 1 {
		return repository.CreateCollectorPointParams{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "RKC 温控器只支持单元素读取")
	}
	if connection.DriverID == "dam3601.serial" && input.ElementCount > 128 {
		return repository.CreateCollectorPointParams{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "DAM3601 元素数量不能超过 128")
	}
	if connection.DriverID == "yudian.ai-bus" && input.ElementCount > 4 {
		return repository.CreateCollectorPointParams{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "宇电 AIBus 元素数量不能超过 4")
	}
	if connection.DriverID == "delixi.dtsu6606" && input.ElementCount > 65535 {
		return repository.CreateCollectorPointParams{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "DTSU6606 元素数量不能超过 65535")
	}
	if (strings.HasPrefix(connection.DriverID, "dlt645.") || strings.HasPrefix(connection.DriverID, "dlt698.")) && input.ElementCount > 65535 {
		return repository.CreateCollectorPointParams{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "DLT 元素数量不能超过 65535")
	}
	if connection.DriverID == "dcs.nanjing-auto" && input.ElementCount > 65535 {
		return repository.CreateCollectorPointParams{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "南京自动化 DCS 元素数量不能超过 65535")
	}
	if strings.HasPrefix(connection.DriverID, "robot.") && input.ElementCount > 65535 {
		return repository.CreateCollectorPointParams{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "机器人变量元素数量不能超过 65535")
	}
	if (connection.DriverID == "siemens.s7-plus" || connection.DriverID == "mqtt.rpc-device") && input.ElementCount > 65535 {
		return repository.CreateCollectorPointParams{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "元素数量不能超过 65535")
	}
	if connection.DriverID == "ec-fan.machine-serial" && input.ElementCount != 1 {
		return repository.CreateCollectorPointParams{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "EC 风机只支持单元素读取")
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
	mode := strings.TrimSpace(input.AcquisitionMode)
	if mode == "" {
		if input.AcquisitionOverrides != nil {
			mode = "override"
		} else {
			mode = "inherit"
		}
	}
	if mode != "inherit" && mode != "override" {
		return repository.CreateCollectorPointParams{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "采集参数模式必须为 inherit 或 override")
	}
	overrides := cloneCollectorMap(input.AcquisitionOverrides)
	if mode == "inherit" {
		overrides = map[string]any{}
	}
	effectiveAcquisition := mergeCollectorPointAcquisition(connection.DefaultAcquisition, mode, overrides)
	if _, err := normalizeCollectorDefaultAcquisition(effectiveAcquisition); err != nil {
		return repository.CreateCollectorPointParams{}, err
	}
	return repository.CreateCollectorPointParams{ID: id, ProjectID: connection.ProjectID, ConnectionID: connection.ID, ConnectionCode: connection.Code, UserID: userID, GroupID: input.GroupID, Code: code, Name: name, Description: input.Description, Address: cloneCollectorMap(input.Address), AddressText: addressText, AddressSchemaVersion: connection.SchemaVersion, DataType: input.DataType, ElementCount: elementCount, ReadOptions: cloneCollectorMap(input.ReadOptions), AcquisitionMode: mode, AcquisitionOverrides: overrides, Acquisition: effectiveAcquisition, Enabled: enabled, SortOrder: input.SortOrder, Metadata: cloneCollectorMap(input.Metadata)}, nil
}

func mergeCollectorPointAcquisition(defaults map[string]any, mode string, overrides map[string]any) map[string]any {
	result := cloneCollectorMap(defaults)
	if mode == "override" {
		for key, value := range overrides {
			result[key] = value
		}
	}
	return result
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

// validateMitsubishiMcPointAddress 与 DevAgent 使用相同的 MC 单次读取限制，避免保存后才发现变量不可读。
func validateMitsubishiMcPointAddress(address map[string]any, dataType string, elementCount int) error {
	value, ok := address["address"].(string)
	value = strings.TrimSpace(value)
	if !ok || value == "" || len([]rune(value)) > 128 || strings.IndexFunc(value, unicode.IsSpace) >= 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Mitsubishi MC 设备地址无效")
	}

	if elementCount < 1 {
		elementCount = 1
	}
	protocolUnits := int64(elementCount)
	maximumUnits := int64(960)
	switch strings.ToLower(dataType) {
	case "bool":
		maximumUnits = 7168
	case "int8", "uint8", "string", "bytes":
		protocolUnits = (int64(elementCount) + 1) / 2
	case "int16", "uint16":
	case "int32", "uint32", "float32":
		protocolUnits = int64(elementCount) * 2
	case "int64", "uint64", "float64":
		protocolUnits = int64(elementCount) * 4
	default:
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Mitsubishi MC 数据类型不受支持")
	}
	if protocolUnits > maximumUnits {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fmt.Sprintf("Mitsubishi MC 单变量读取长度超过协议上限 %d", maximumUnits))
	}
	return nil
}

func isMitsubishiNativeDriver(driverID string) bool {
	switch driverID {
	case "mitsubishi.a1e-ascii-tcp", "mitsubishi.a1e-binary-tcp",
		"mitsubishi.mc-ascii-tcp", "mitsubishi.mc-ascii-udp",
		"mitsubishi.mc-binary-udp", "mitsubishi.mc-r-binary-tcp",
		"mitsubishi.a3c-serial", "mitsubishi.a3c-serial-over-tcp",
		"mitsubishi.fx-links-serial", "mitsubishi.fx-links-over-tcp",
		"mitsubishi.fx-serial", "mitsubishi.fx-serial-over-tcp":
		return true
	default:
		return false
	}
}

func isMitsubishiVariantDriver(driverID string) bool {
	return isMitsubishiNativeDriver(driverID) || driverID == "mitsubishi.cip"
}

// normalizeMitsubishiNetworkAddress 统一保存去除首尾空白后的原生地址；CIP 标签保留用户输入的大小写。
func normalizeMitsubishiNetworkAddress(value any, cip bool) (string, error) {
	address, ok := value.(string)
	address = strings.TrimSpace(address)
	maximumLength := 128
	message := "Mitsubishi MC 设备地址无效"
	if cip {
		maximumLength = 512
		message = "Mitsubishi CIP 标签地址无效"
	}
	if !ok || address == "" || len([]rune(address)) > maximumLength || strings.IndexFunc(address, unicode.IsSpace) >= 0 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, message)
	}
	return address, nil
}

func validateMitsubishiCipPointAddress(dataType string, elementCount int) error {
	if elementCount < 1 || elementCount > 65535 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Mitsubishi CIP 元素数量必须在 1 到 65535 之间")
	}
	if strings.EqualFold(dataType, "datetime") && elementCount != 1 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Mitsubishi CIP datetime 标签只支持单元素读取")
	}
	return nil
}

// validateOmronFinsPointAddress 限制单变量展开后的协议单位，避免调试任务申请异常大的读取缓冲区。
func validateOmronFinsPointAddress(address map[string]any, dataType string, elementCount int) error {
	value, ok := address["address"].(string)
	value = strings.TrimSpace(value)
	if !ok || value == "" || len([]rune(value)) > 128 || strings.IndexFunc(value, unicode.IsSpace) >= 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Omron FINS 设备地址无效")
	}

	if elementCount < 1 {
		elementCount = 1
	}
	protocolUnits := int64(elementCount)
	switch strings.ToLower(dataType) {
	case "bool", "int16", "uint16":
	case "int8", "uint8", "string", "bytes":
		protocolUnits = (int64(elementCount) + 1) / 2
	case "int32", "uint32", "float32":
		protocolUnits = int64(elementCount) * 2
	case "int64", "uint64", "float64":
		protocolUnits = int64(elementCount) * 4
	default:
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Omron FINS 数据类型不受支持")
	}
	if protocolUnits > 65535 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Omron FINS 单变量读取长度超过平台上限 65535")
	}
	return nil
}

var omronHostLinkCModeAddressPattern = regexp.MustCompile(`(?i)^(?:(?:D|DM|C|CIO|LR|H|HR|A|AR|TIM|CNT)\d+(?:\.\d+)?|(?:E|EM)[0-9A-F]+\.\d+(?:\.\d+)?)$`)

// normalizeOmronVariantAddress 将用户输入统一为 DevAgent 可直接读取的标签或 C-Mode 地址。
func normalizeOmronVariantAddress(value any, cMode bool) (string, error) {
	address, ok := value.(string)
	address = strings.TrimSpace(address)
	maximumLength := 512
	invalidMessage := "Omron CIP 标签地址无效"
	if cMode {
		maximumLength = 128
		invalidMessage = "Omron HostLink C-Mode 设备地址无效"
	}
	if !ok || address == "" || len([]rune(address)) > maximumLength || strings.IndexFunc(address, unicode.IsSpace) >= 0 ||
		(cMode && !omronHostLinkCModeAddressPattern.MatchString(address)) {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, invalidMessage)
	}
	return address, nil
}

// validateOmronVariantPointAddress 保持元素数量与底层 ushort 读取长度一致。
func validateOmronVariantPointAddress(dataType string, elementCount int) error {
	if elementCount < 1 {
		elementCount = 1
	}
	if elementCount > 65535 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Omron 元素数量不能超过 65535")
	}
	if strings.EqualFold(dataType, "datetime") && elementCount != 1 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Omron datetime 标签只支持单元素读取")
	}
	return nil
}

// validateAllenBradleyPointAddress 校验 Logix 标签和数组长度，避免无效标签进入调试任务。
func validateAllenBradleyPointAddress(address map[string]any, dataType string, elementCount int) error {
	value, ok := address["address"].(string)
	value = strings.TrimSpace(value)
	if !ok || value == "" || len([]rune(value)) > 512 || strings.IndexFunc(value, unicode.IsSpace) >= 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Allen-Bradley 标签地址无效")
	}
	if elementCount < 1 {
		elementCount = 1
	}
	if elementCount > 65535 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Allen-Bradley 元素数量不能超过 65535")
	}
	if strings.EqualFold(dataType, "datetime") && elementCount != 1 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Allen-Bradley datetime 标签只支持单元素读取")
	}
	return nil
}

// validateAllenBradleyLegacyPointAddress 限制文件协议只使用 Demo 已验证的数据类型。
func validateAllenBradleyLegacyPointAddress(address map[string]any, dataType string, elementCount int) error {
	if err := validateAllenBradleyPointAddress(address, dataType, elementCount); err != nil {
		return err
	}
	supported := map[string]struct{}{
		"bool": {}, "int16": {}, "uint16": {}, "int32": {}, "uint32": {}, "float32": {}, "string": {},
	}
	if _, ok := supported[strings.ToLower(dataType)]; !ok {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "当前 Allen-Bradley 文件协议不支持该数据类型")
	}
	return nil
}

// validateBeckhoffAdsPointAddress 与 DevAgent 保持相同的 ADS 地址和元素数量边界。
func validateBeckhoffAdsPointAddress(address map[string]any, _ string, elementCount int) error {
	value, ok := address["address"].(string)
	value = strings.TrimSpace(value)
	if !ok || value == "" || len([]rune(value)) > 512 || strings.IndexFunc(value, unicode.IsSpace) >= 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "倍福 ADS 地址无效")
	}
	if elementCount < 1 {
		elementCount = 1
	}
	if elementCount > 65535 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "倍福 ADS 元素数量不能超过 65535")
	}
	return nil
}

// validateIec104PointAddress 校验信息类型与平台数据类型的固定映射，保证总召唤结果可无歧义转换。
func validateIec104PointAddress(address map[string]any, dataType string, elementCount int) error {
	informationType, _ := address["informationType"].(string)
	informationObjectAddress, ok := collectorNumericInt(address["informationObjectAddress"])
	if informationObjectAddress < 0 || informationObjectAddress > 0xFFFFFF || !ok {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "IEC 104 信息对象地址必须在 0 到 16777215 之间")
	}
	if elementCount < 1 {
		elementCount = 1
	}
	if elementCount != 1 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "IEC 104 单个信息对象的元素数量必须为 1")
	}

	expectedDataType := map[string]string{
		"singlePoint":        "bool",
		"doublePoint":        "uint8",
		"normalizedMeasured": "int16",
		"scaledMeasured":     "int16",
		"shortFloatMeasured": "float32",
		"bitString32":        "uint32",
		"integratedTotal":    "uint32",
	}[informationType]
	if expectedDataType == "" {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "IEC 104 信息类型不受支持")
	}
	if !strings.EqualFold(dataType, expectedDataType) {
		return apperrors.NewAppError(
			apperrors.ErrorCodeBadRequest,
			http.StatusBadRequest,
			fmt.Sprintf("IEC 104 %s 信息类型必须使用 %s 数据类型", informationType, expectedDataType),
		)
	}
	return nil
}

// validatePanasonicMewtocolPointAddress 与 DevAgent 保持相同的原生地址和元素数量边界。
func validatePanasonicMewtocolPointAddress(address map[string]any, elementCount int) error {
	value, ok := address["address"].(string)
	value = strings.TrimSpace(value)
	if !ok || value == "" || len([]rune(value)) > 128 || strings.IndexFunc(value, unicode.IsSpace) >= 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "松下 Mewtocol 设备地址无效")
	}
	if elementCount < 1 {
		elementCount = 1
	}
	if elementCount > 65535 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "松下 Mewtocol 元素数量不能超过 65535")
	}
	return nil
}

// normalizePanasonicMcAddress 按 Binary Demo 的位区和字区规则规范化地址。
func normalizePanasonicMcAddress(value any, dataType string) (string, error) {
	address, ok := value.(string)
	address = strings.ToUpper(strings.TrimSpace(address))
	if !ok || len(address) < 2 || len([]rune(address)) > 128 || strings.IndexFunc(address, unicode.IsSpace) >= 0 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "松下 MC Binary 设备地址无效")
	}
	area := ""
	for _, candidate := range []string{"TS", "CS", "TN", "CN", "DT", "LD", "SD", "X", "Y", "R", "L", "D"} {
		if strings.HasPrefix(address, candidate) {
			area = candidate
			break
		}
	}
	if area == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "松下 MC Binary 设备地址无效")
	}
	body := address[len(area):]
	parts := strings.Split(body, ".")
	if len(parts) > 2 || parts[0] == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "松下 MC Binary 设备地址无效")
	}
	for _, part := range parts {
		if _, err := strconv.ParseUint(part, 10, 32); err != nil {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "松下 MC Binary 设备地址无效")
		}
	}
	bitArea := area == "X" || area == "Y" || area == "R" || area == "TS" || area == "CS" || area == "L"
	if dataType != "" && strings.EqualFold(dataType, "bool") != bitArea {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "松下 MC Binary 数据类型与地址区域不匹配")
	}
	return address, nil
}

// validateLsisPointAddress 与 DevAgent 保持相同的原生地址和元素数量边界。
func validateLsisPointAddress(address map[string]any, elementCount int) error {
	value, ok := address["address"].(string)
	value = strings.TrimSpace(value)
	if !ok || value == "" || len([]rune(value)) > 128 || strings.IndexFunc(value, unicode.IsSpace) >= 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "LSIS 设备地址无效")
	}
	if elementCount < 1 {
		elementCount = 1
	}
	if elementCount > 65535 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "LSIS 元素数量不能超过 65535")
	}
	return nil
}

// normalizeGeSrtpAddress 统一地址区名称和起始编号，避免大小写及前导零形成重复变量。
func normalizeGeSrtpAddress(value any) (string, error) {
	address, ok := value.(string)
	address = strings.ToUpper(strings.TrimSpace(address))
	if !ok || len(address) < 2 || len([]rune(address)) > 128 || strings.IndexFunc(address, unicode.IsSpace) >= 0 {
		return "", invalidGeSrtpAddress()
	}
	area := geSrtpAddressArea(address)
	if area == "" {
		return "", invalidGeSrtpAddress()
	}
	offset, err := strconv.ParseInt(address[len(area):], 10, 32)
	if err != nil || offset < 1 {
		return "", invalidGeSrtpAddress()
	}
	return area + strconv.FormatInt(offset, 10), nil
}

// validateGeSrtpPointAddress 校验字寄存器的类型限制及单变量元素数量上限。
func validateGeSrtpPointAddress(address map[string]any, dataType string, elementCount int) error {
	value, _ := address["address"].(string)
	area := geSrtpAddressArea(value)
	if strings.EqualFold(dataType, "bool") && (area == "AI" || area == "AQ" || area == "R") {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "GE SRTP AI、AQ、R 字寄存器不支持 bool 数据类型")
	}
	if elementCount < 1 {
		elementCount = 1
	}
	if elementCount > 65535 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "GE SRTP 元素数量不能超过 65535")
	}
	return nil
}

func geSrtpAddressArea(address string) string {
	for _, area := range []string{"AI", "AQ", "SA", "SB", "SC", "I", "Q", "M", "T", "S", "G", "R"} {
		if strings.HasPrefix(address, area) {
			return area
		}
	}
	return ""
}

func invalidGeSrtpAddress() error {
	return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "GE SRTP 设备地址无效")
}

// normalizeInovanceAddress 统一汇川原生地址大小写，系列相关映射由 Agent 在真实连接上处理。
func normalizeInovanceAddress(value any) (string, error) {
	address, ok := value.(string)
	address = strings.ToUpper(strings.TrimSpace(address))
	if !ok || address == "" || len([]rune(address)) > 128 || strings.IndexFunc(address, unicode.IsSpace) >= 0 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "汇川 PLC 设备地址无效")
	}
	return address, nil
}

// validateInovancePointAddress 按 Modbus 单报文上限校验元素数量，避免运行时才发现请求过长。
func validateInovancePointAddress(dataType string, elementCount int) error {
	if elementCount < 1 {
		elementCount = 1
	}
	protocolUnits := int64(elementCount)
	maximumUnits := int64(125)
	switch strings.ToLower(dataType) {
	case "bool":
		maximumUnits = 2000
	case "int8", "uint8", "string", "bytes":
		protocolUnits = (int64(elementCount) + 1) / 2
	case "int32", "uint32", "float32":
		protocolUnits = int64(elementCount) * 2
	case "int64", "uint64", "float64":
		protocolUnits = int64(elementCount) * 4
	}
	if protocolUnits > maximumUnits {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "汇川 Modbus TCP 单变量读取长度超过协议上限")
	}
	return nil
}

// normalizeInovanceSpecialAddress 保留 CIP 标签大小写，其余专用协议统一使用大写设备地址。
func normalizeInovanceSpecialAddress(value any, preserveCase bool) (string, error) {
	address, ok := value.(string)
	address = strings.TrimSpace(address)
	maximumLength := 128
	if preserveCase {
		maximumLength = 512
	} else {
		address = strings.ToUpper(address)
	}
	if !ok || address == "" || len([]rune(address)) > maximumLength || strings.IndexFunc(address, unicode.IsSpace) >= 0 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "汇川设备地址无效")
	}
	return address, nil
}

// validateInovanceSpecialPointAddress 与 Agent 的 ushort 读取长度及 CIP 日期读取限制保持一致。
func validateInovanceSpecialPointAddress(dataType string, elementCount int, allowDateTime bool) error {
	if elementCount < 1 {
		elementCount = 1
	}
	if elementCount > 65535 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "汇川元素数量不能超过 65535")
	}
	if strings.EqualFold(dataType, "datetime") && (!allowDateTime || elementCount != 1) {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "当前汇川协议不支持该 datetime 配置")
	}
	return nil
}

// normalizeFatekProgramAddress 统一站号覆盖参数与设备区编号，并提前阻止位区和字区类型混用。
func normalizeFatekProgramAddress(value any, dataType string) (string, error) {
	address, ok := value.(string)
	address = strings.TrimSpace(address)
	if !ok || len(address) < 2 || len([]rune(address)) > 128 || strings.IndexFunc(address, unicode.IsSpace) >= 0 {
		return "", invalidFatekProgramAddress("永宏 PLC 设备地址无效")
	}
	body := address
	stationPrefix := ""
	if separator := strings.IndexByte(address, ';'); separator >= 0 {
		stationText := address[:separator]
		if !strings.HasPrefix(strings.ToLower(stationText), "s=") {
			return "", invalidFatekProgramAddress("永宏 PLC 设备地址无效")
		}
		station, err := strconv.ParseUint(stationText[2:], 10, 8)
		if err != nil {
			return "", invalidFatekProgramAddress("永宏 PLC 设备地址无效")
		}
		stationPrefix = fmt.Sprintf("s=%d;", station)
		body = address[separator+1:]
	}
	body = strings.ToUpper(body)
	areas := []string{"RT", "RC", "D", "R"}
	message := "永宏数值变量只支持 RT、RC、D、R 地址"
	if strings.EqualFold(dataType, "bool") {
		areas = []string{"M", "X", "Y", "S", "T", "C"}
		message = "永宏 bool 变量只支持 M、X、Y、S、T、C 地址"
	}
	area := ""
	for _, candidate := range areas {
		if strings.HasPrefix(body, candidate) {
			area = candidate
			break
		}
	}
	if area == "" {
		return "", invalidFatekProgramAddress(message)
	}
	offset, err := strconv.ParseInt(body[len(area):], 10, 32)
	if err != nil || offset < 0 {
		return "", invalidFatekProgramAddress(message)
	}
	return stationPrefix + area + strconv.FormatInt(offset, 10), nil
}

func invalidFatekProgramAddress(message string) error {
	return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, message)
}

// normalizeFreedomRequestHex 将用户输入统一成按字节分隔的大写十六进制，保证重复地址判断和代理读取使用同一表示。
func normalizeFreedomRequestHex(value any) (string, error) {
	requestHex, ok := value.(string)
	if !ok {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "自由协议请求报文必须是十六进制字节")
	}
	var compact strings.Builder
	for _, character := range strings.TrimSpace(requestHex) {
		if unicode.IsSpace(character) || character == '-' {
			continue
		}
		if !strings.ContainsRune("0123456789abcdefABCDEF", character) {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "自由协议请求报文必须是十六进制字节")
		}
		compact.WriteRune(unicode.ToUpper(character))
	}
	valueText := compact.String()
	if len(valueText) < 2 || len(valueText)%2 != 0 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "自由协议请求报文必须是偶数长度的十六进制字节")
	}
	bytes := make([]string, 0, len(valueText)/2)
	for index := 0; index < len(valueText); index += 2 {
		bytes = append(bytes, valueText[index:index+2])
	}
	return strings.Join(bytes, " "), nil
}

// normalizeCimonAddress 统一设备地址大小写，避免同一地址因录入形式不同形成重复变量。
func normalizeCimonAddress(value any) (string, error) {
	address, ok := value.(string)
	address = strings.ToUpper(strings.TrimSpace(address))
	if !ok || len(address) < 2 || len([]rune(address)) > 128 || strings.IndexFunc(address, unicode.IsSpace) >= 0 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Cimon HMI 设备地址无效")
	}
	return address, nil
}

var keyenceMcAddressPattern = regexp.MustCompile(`^(?:R|MR|LR|CR|CM|DM|EM|FM|ZF|W|TN|TS|CN|CS)[0-9A-F]+$`)
var keyenceKvOldAddressPattern = regexp.MustCompile(`^[A-Z]{1,4}[0-9]+(?:\.[0-9]+)?$`)
var keyenceNanoAddressPattern = regexp.MustCompile(`^(?:(?:R|B|MR|LR|CR|VB|DM|EM|FM|ZF|W|TM|Z|AT|CM|VM|T|C|TC|CC|TS|CS)[0-9]+|UNIT=[0-9]+;[0-9]+)$`)

// normalizeKeyenceAddress 统一软元件地址大小写，并使用与节点驱动一致的协议地址语法。
func normalizeKeyenceAddress(value any, driverID string) (string, error) {
	address, ok := value.(string)
	address = strings.ToUpper(strings.TrimSpace(address))
	pattern := keyenceNanoAddressPattern
	if driverID == "keyence.mc-3e-tcp" || driverID == "keyence.mc-ascii-tcp" {
		pattern = keyenceMcAddressPattern
	} else if driverID == "keyence.kv-old-tcp" {
		pattern = keyenceKvOldAddressPattern
	}
	if !ok || len(address) < 2 || len([]rune(address)) > 128 || strings.IndexFunc(address, unicode.IsSpace) >= 0 || !pattern.MatchString(address) {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "基恩士设备地址无效")
	}
	return address, nil
}

var deltaAddressPattern = regexp.MustCompile(`^(?:X|Y|M|SM|S|SR|D|T|C|HC|E)[0-9]+(?:\.[0-9]+)?$`)

// normalizeDeltaAddress 规范站号覆盖和软元件地址，避免大小写差异形成重复变量。
func normalizeDeltaAddress(value any) (string, error) {
	address, ok := value.(string)
	address = strings.TrimSpace(address)
	if !ok || len(address) < 2 || len([]rune(address)) > 128 || strings.IndexFunc(address, unicode.IsSpace) >= 0 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "台达 PLC 设备地址无效")
	}
	prefix := ""
	if separator := strings.IndexByte(address, ';'); separator >= 0 {
		stationText := address[:separator]
		if !strings.HasPrefix(strings.ToLower(stationText), "s=") {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "台达地址站号覆盖格式无效")
		}
		station, err := strconv.ParseUint(stationText[2:], 10, 8)
		if err != nil {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "台达地址站号覆盖格式无效")
		}
		prefix = fmt.Sprintf("s=%d;", station)
		address = address[separator+1:]
	}
	address = strings.ToUpper(address)
	if len(address) < 2 || !deltaAddressPattern.MatchString(address) {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "台达 PLC 设备地址无效")
	}
	return prefix + address, nil
}

var xinjeAddressPattern = regexp.MustCompile(`^[A-Z]{1,4}[0-9]+$`)

// normalizeXinjeAddress 统一站号覆盖及软元件地址大小写。
func normalizeXinjeAddress(value any) (string, error) {
	address, ok := value.(string)
	address = strings.TrimSpace(address)
	if !ok || len(address) < 2 || len([]rune(address)) > 128 || strings.IndexFunc(address, unicode.IsSpace) >= 0 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "信捷 PLC 设备地址无效")
	}
	prefix := ""
	if separator := strings.IndexByte(address, ';'); separator >= 0 {
		stationText := address[:separator]
		if !strings.HasPrefix(strings.ToLower(stationText), "s=") {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "信捷地址站号覆盖格式无效")
		}
		station, err := strconv.ParseUint(stationText[2:], 10, 8)
		if err != nil {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "信捷地址站号覆盖格式无效")
		}
		prefix = fmt.Sprintf("s=%d;", station)
		address = address[separator+1:]
	}
	address = strings.ToUpper(address)
	if len(address) < 2 || !xinjeAddressPattern.MatchString(address) {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "信捷 PLC 设备地址无效")
	}
	return prefix + address, nil
}

var megMeetAddressPattern = regexp.MustCompile(`^(?:X|Y|M|SM|S|D|SD|Z|R|T|C)[0-9]+(?:\.[0-9]+)?$`)

// normalizeMegMeetAddress 统一站号覆盖及软元件地址大小写，避免同一地址因格式差异重复保存。
func normalizeMegMeetAddress(value any) (string, error) {
	address, ok := value.(string)
	address = strings.TrimSpace(address)
	if !ok || len(address) < 2 || len([]rune(address)) > 128 || strings.IndexFunc(address, unicode.IsSpace) >= 0 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "麦格米特 PLC 设备地址无效")
	}
	prefix := ""
	if separator := strings.IndexByte(address, ';'); separator >= 0 {
		stationText := address[:separator]
		if !strings.HasPrefix(strings.ToLower(stationText), "s=") {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "麦格米特地址站号覆盖格式无效")
		}
		station, err := strconv.ParseUint(stationText[2:], 10, 8)
		if err != nil {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "麦格米特地址站号覆盖格式无效")
		}
		prefix = fmt.Sprintf("s=%d;", station)
		address = address[separator+1:]
	}
	address = strings.ToUpper(address)
	if len(address) < 2 || !megMeetAddressPattern.MatchString(address) {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "麦格米特 PLC 设备地址无效")
	}
	return prefix + address, nil
}

var fujiCommandAddressPattern = regexp.MustCompile(`^(?:B|M|K|F|A|D|S|W|TS|TR|CS|CR|BD|WL)[0-9]+(?:\.[0-9]+)?$`)
var fujiSphAddressPattern = regexp.MustCompile(`^(?:M(?:1|3|10)\.[0-9]+(?:\.[0-9]+)?|[IQ][0-9]+(?:\.[0-9]+)?)$`)
var fujiSpbAddressPattern = regexp.MustCompile(`^(?:X|Y|L|M|D|TN|CN|TC|CC|R|W)[0-9]+(?:\.[0-9]+)?$`)

// normalizeFujiAddress 统一富士软元件地址大小写，SPB 协议额外支持站号覆盖。
func normalizeFujiAddress(value any, driverID string) (string, error) {
	address, ok := value.(string)
	address = strings.TrimSpace(address)
	if !ok || len(address) < 2 || len([]rune(address)) > 128 || strings.IndexFunc(address, unicode.IsSpace) >= 0 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "富士 PLC 设备地址无效")
	}
	prefix := ""
	if separator := strings.IndexByte(address, ';'); separator >= 0 {
		if driverID != "fuji.spb" && driverID != "fuji.spb-over-tcp" {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "当前富士协议不支持地址内覆盖站号")
		}
		stationText := address[:separator]
		if !strings.HasPrefix(strings.ToLower(stationText), "s=") {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "富士地址站号覆盖格式无效")
		}
		station, err := strconv.ParseUint(stationText[2:], 10, 8)
		if err != nil {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "富士地址站号覆盖格式无效")
		}
		prefix = fmt.Sprintf("s=%d;", station)
		address = address[separator+1:]
	}
	address = strings.ToUpper(address)
	pattern := fujiCommandAddressPattern
	if driverID == "fuji.sph-tcp" {
		pattern = fujiSphAddressPattern
	} else if driverID == "fuji.spb" || driverID == "fuji.spb-over-tcp" {
		pattern = fujiSpbAddressPattern
	}
	if len(address) < 2 || !pattern.MatchString(address) {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "富士 PLC 设备地址无效")
	}
	return prefix + address, nil
}

var vigorAddressPattern = regexp.MustCompile(`^(?:X|Y|M|SM|S|TS|TC|CS|CC|D|SD|R|T|C)[0-9]+$`)

// normalizeVigorAddress 统一站号覆盖和 VS 系列软元件地址大小写。
func normalizeVigorAddress(value any) (string, error) {
	address, ok := value.(string)
	address = strings.TrimSpace(address)
	if !ok || len(address) < 2 || len([]rune(address)) > 128 || strings.IndexFunc(address, unicode.IsSpace) >= 0 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "丰炜 PLC 设备地址无效")
	}
	prefix := ""
	if separator := strings.IndexByte(address, ';'); separator >= 0 {
		stationText := address[:separator]
		if !strings.HasPrefix(strings.ToLower(stationText), "s=") {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "丰炜地址站号覆盖格式无效")
		}
		station, err := strconv.ParseUint(stationText[2:], 10, 8)
		if err != nil {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "丰炜地址站号覆盖格式无效")
		}
		prefix = fmt.Sprintf("s=%d;", station)
		address = address[separator+1:]
	}
	address = strings.ToUpper(address)
	if len(address) < 2 || !vigorAddressPattern.MatchString(address) {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "丰炜 PLC 设备地址无效")
	}
	return prefix + address, nil
}

// normalizeYokogawaAddress 统一 CPU 覆盖、软元件和特殊模块地址，并在保存时校验位区与数据类型。
func normalizeYokogawaAddress(value any, dataType string) (string, error) {
	address, ok := value.(string)
	address = strings.TrimSpace(address)
	if !ok || len(address) < 2 || len([]rune(address)) > 128 || strings.IndexFunc(address, unicode.IsSpace) >= 0 {
		return "", invalidYokogawaAddress("横河 PLC 设备地址无效")
	}

	if strings.HasPrefix(strings.ToLower(address), "special:") {
		normalized, err := normalizeYokogawaSpecialAddress(address[len("Special:"):])
		if err != nil {
			return "", err
		}
		if strings.EqualFold(dataType, "bool") {
			return "", invalidYokogawaAddress("横河特殊模块地址不支持 bool 类型")
		}
		return "Special:" + normalized, nil
	}

	prefix := ""
	if separator := strings.IndexByte(address, ';'); separator >= 0 {
		cpuText := address[:separator]
		if !strings.HasPrefix(strings.ToLower(cpuText), "cpu=") {
			return "", invalidYokogawaAddress("横河地址 CPU 覆盖格式无效")
		}
		cpu, err := strconv.ParseUint(cpuText[len("cpu="):], 10, 8)
		if err != nil {
			return "", invalidYokogawaAddress("横河地址 CPU 覆盖格式无效")
		}
		prefix = fmt.Sprintf("cpu=%d;", cpu)
		address = address[separator+1:]
	}

	area, offset, ok := splitYokogawaDeviceAddress(address)
	if !ok {
		return "", invalidYokogawaAddress("横河 PLC 设备地址无效")
	}
	bitArea := area == "X" || area == "Y" || area == "I" || area == "E" || area == "M" || area == "T" || area == "C" || area == "L"
	if dataType != "" && strings.EqualFold(dataType, "bool") != bitArea {
		if bitArea {
			return "", invalidYokogawaAddress("横河继电器地址只支持 bool 类型")
		}
		return "", invalidYokogawaAddress("横河 bool 变量必须使用继电器地址")
	}
	return prefix + area + offset, nil
}

func normalizeYokogawaSpecialAddress(value string) (string, error) {
	parts := strings.Split(value, ";")
	if len(parts) != 3 && len(parts) != 4 {
		return "", invalidYokogawaAddress("横河特殊模块地址无效")
	}
	index := 0
	prefix := ""
	if len(parts) == 4 {
		cpu, ok := parseYokogawaNamedByte(parts[index], "cpu")
		if !ok {
			return "", invalidYokogawaAddress("横河特殊模块 CPU 编号无效")
		}
		prefix = fmt.Sprintf("cpu=%d;", cpu)
		index++
	}
	unit, ok := parseYokogawaNamedByte(parts[index], "unit")
	if !ok {
		return "", invalidYokogawaAddress("横河特殊模块单元编号无效")
	}
	slot, ok := parseYokogawaNamedByte(parts[index+1], "slot")
	if !ok {
		return "", invalidYokogawaAddress("横河特殊模块槽位编号无效")
	}
	offset, err := strconv.ParseUint(parts[index+2], 10, 32)
	if err != nil {
		return "", invalidYokogawaAddress("横河特殊模块偏移地址无效")
	}
	return fmt.Sprintf("%sunit=%d;slot=%d;%d", prefix, unit, slot, offset), nil
}

func parseYokogawaNamedByte(value string, name string) (uint64, bool) {
	key, raw, found := strings.Cut(value, "=")
	if !found || !strings.EqualFold(key, name) {
		return 0, false
	}
	parsed, err := strconv.ParseUint(raw, 10, 8)
	return parsed, err == nil
}

func splitYokogawaDeviceAddress(value string) (string, string, bool) {
	upper := strings.ToUpper(value)
	for _, area := range []string{"TN", "CN", "X", "Y", "I", "E", "M", "T", "C", "L", "D", "B", "F", "R", "V", "Z", "W"} {
		if !strings.HasPrefix(upper, area) {
			continue
		}
		offset := upper[len(area):]
		if offset == "" {
			return "", "", false
		}
		if _, err := strconv.ParseUint(offset, 10, 32); err != nil {
			return "", "", false
		}
		return area, offset, true
	}
	return "", "", false
}

func invalidYokogawaAddress(message string) error {
	return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, message)
}

var yamatakeAddressPattern = regexp.MustCompile(`(?i)^(?:s=([0-9]{1,3});)?([0-9]+)$`)

// normalizeYamatakeAddress 统一站号覆盖与数字地址，避免前导零和大小写差异形成重复变量。
func normalizeYamatakeAddress(value any) (string, error) {
	address, ok := value.(string)
	address = strings.TrimSpace(address)
	if !ok {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "山武 Digitron 地址无效")
	}
	match := yamatakeAddressPattern.FindStringSubmatch(address)
	if len(match) == 0 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "山武 Digitron 地址必须是数字，可选使用 s=<站号>; 前缀")
	}
	prefix := ""
	if match[1] != "" {
		station, err := strconv.ParseUint(match[1], 10, 8)
		if err != nil {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "山武 Digitron 站号必须在 0 到 255 之间")
		}
		prefix = fmt.Sprintf("s=%d;", station)
	}
	register, err := strconv.ParseUint(match[2], 10, 63)
	if err != nil {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "山武 Digitron 数字地址无效")
	}
	return prefix + strconv.FormatUint(register, 10), nil
}

// normalizeYaskawaAddress 保留 Memobus 扩展参数语义，仅清理首尾空白并阻止协议不接受的内部空格。
func normalizeYaskawaAddress(value any) (string, error) {
	address, ok := value.(string)
	address = strings.TrimSpace(address)
	if !ok || address == "" || len([]rune(address)) > 128 || strings.IndexFunc(address, unicode.IsSpace) >= 0 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "安川 Memobus 地址无效")
	}
	return address, nil
}

var orientalMotorAddressPattern = regexp.MustCompile(`(?i)^(input|output)\[([0-9]+)\]$`)
var toyoPucAddressPattern = regexp.MustCompile(`(?i)^(?:prg=([0-9]{1,3});)?(EK|EV|ET|EC|EL|EX|EY|EM|ES|EN|GX|GY|GM|EB|K|V|T|C|L|X|Y|M|S|N|R|D|B|H|U)([0-9A-F]+)$`)
var cjt188AddressPattern = regexp.MustCompile(`(?i)^(?:s=([0-9A]{1,14});)?([0-9A-F]{2})[- ]?([0-9A-F]{2})$`)
var rkcAddressPattern = regexp.MustCompile(`(?i)^(?:s=([0-9]{1,2});)?([A-Z][A-Z0-9])$`)
var dam3601AddressPattern = regexp.MustCompile(`(?i)^(?:s=([0-9]{1,3});)?temperature\[([0-9]{1,3})\]$`)
var yuDianAddressPattern = regexp.MustCompile(`(?i)^(?:s=([0-9]{1,3});)?([0-9]{1,3})$`)
var dltAddressPattern = regexp.MustCompile(`(?i)^(?:s=([0-9A]{1,12});)?((?:[0-9A-F]{2}-){1,3}[0-9A-F]{2})$`)

// normalizeCjt188Address 统一表地址和两字节数据标识，避免分隔符、大小写造成重复变量。
func normalizeCjt188Address(value any) (string, error) {
	address, ok := value.(string)
	if !ok {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "CJT188 数据标识无效")
	}
	match := cjt188AddressPattern.FindStringSubmatch(strings.TrimSpace(address))
	if len(match) == 0 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "CJT188 数据标识应为 90-1F，可选使用 s=<表地址>; 前缀")
	}
	prefix := ""
	if match[1] != "" {
		prefix = "s=" + strings.Repeat("0", 14-len(match[1])) + strings.ToUpper(match[1]) + ";"
	}
	return prefix + strings.ToUpper(match[2]) + "-" + strings.ToUpper(match[3]), nil
}

// normalizeRkcAddress 将站号和参数代码转换为 HSL 接受的标准格式。
func normalizeRkcAddress(value any) (string, error) {
	address, ok := value.(string)
	if !ok {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "RKC 温控器地址无效")
	}
	match := rkcAddressPattern.FindStringSubmatch(strings.TrimSpace(address))
	if len(match) == 0 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "RKC 地址应为 M1、AA、ER 等两位代码，可选使用 s=<站号>; 前缀")
	}
	prefix := ""
	if match[1] != "" {
		station, err := strconv.ParseUint(match[1], 10, 8)
		if err != nil || station > 99 {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "RKC 温控器站号必须在 0 到 99 之间")
		}
		prefix = fmt.Sprintf("s=%d;", station)
	}
	return prefix + strings.ToUpper(match[2]), nil
}

func normalizeDam3601Address(value any) (string, error) {
	address, ok := value.(string)
	if !ok {
		return "", invalidInstrumentAddress("DAM3601 地址无效")
	}
	match := dam3601AddressPattern.FindStringSubmatch(strings.TrimSpace(address))
	if len(match) == 0 {
		return "", invalidInstrumentAddress("DAM3601 地址应为 temperature[0] 到 temperature[127]")
	}
	channel, err := strconv.ParseUint(match[2], 10, 8)
	if err != nil || channel > 127 {
		return "", invalidInstrumentAddress("DAM3601 温度通道必须在 0 到 127 之间")
	}
	prefix := ""
	if match[1] != "" {
		station, err := strconv.ParseUint(match[1], 10, 8)
		if err != nil {
			return "", invalidInstrumentAddress("DAM3601 站号必须在 0 到 255 之间")
		}
		prefix = fmt.Sprintf("s=%d;", station)
	}
	return fmt.Sprintf("%stemperature[%d]", prefix, channel), nil
}

func normalizeYuDianAddress(value any) (string, error) {
	address, ok := value.(string)
	if !ok {
		return "", invalidInstrumentAddress("宇电 AIBus 地址无效")
	}
	match := yuDianAddressPattern.FindStringSubmatch(strings.TrimSpace(address))
	if len(match) == 0 {
		return "", invalidInstrumentAddress("宇电 AIBus 地址必须是 0 到 255 的参数号")
	}
	parameter, err := strconv.ParseUint(match[2], 10, 8)
	if err != nil {
		return "", invalidInstrumentAddress("宇电 AIBus 参数号必须在 0 到 255 之间")
	}
	prefix := ""
	if match[1] != "" {
		station, err := strconv.ParseUint(match[1], 10, 8)
		if err != nil {
			return "", invalidInstrumentAddress("宇电 AIBus 站号必须在 0 到 255 之间")
		}
		prefix = fmt.Sprintf("s=%d;", station)
	}
	return prefix + strconv.FormatUint(parameter, 10), nil
}

func normalizeDtsu6606Address(value any) (string, error) {
	address, ok := value.(string)
	address = strings.TrimSpace(address)
	if !ok || address == "" || len([]rune(address)) > 128 || strings.IndexFunc(address, unicode.IsSpace) >= 0 {
		return "", invalidInstrumentAddress("DTSU6606 Modbus 地址无效")
	}
	return address, nil
}

// normalizeDltAddress 统一 DLT 表地址和数据标识大小写，保证重复地址判断与代理实际读取地址一致。
func normalizeDltAddress(value any, byteCount int) (string, error) {
	address, ok := value.(string)
	if !ok {
		return "", invalidInstrumentAddress("DLT 数据标识无效")
	}
	match := dltAddressPattern.FindStringSubmatch(strings.TrimSpace(address))
	if len(match) != 3 || len(strings.Split(match[2], "-")) != byteCount {
		return "", invalidInstrumentAddress("DLT645-2007/698 地址应为 00-00-00-00，DLT645-1997 地址应为 B6-11")
	}
	prefix := ""
	if match[1] != "" {
		prefix = "s=" + strings.Repeat("0", 12-len(match[1])) + strings.ToUpper(match[1]) + ";"
	}
	return prefix + strings.ToUpper(match[2]), nil
}

func normalizeDcsNanJingAutoAddress(value any) (string, error) {
	address, ok := value.(string)
	address = strings.TrimSpace(address)
	if !ok || address == "" || len([]rune(address)) > 128 || strings.IndexFunc(address, unicode.IsSpace) >= 0 {
		return "", invalidInstrumentAddress("南京自动化 DCS 地址无效")
	}
	return address, nil
}

func normalizeRobotAddress(value any) (string, error) {
	address, ok := value.(string)
	address = strings.TrimSpace(address)
	if !ok || address == "" || len([]rune(address)) > 128 || strings.IndexFunc(address, unicode.IsSpace) >= 0 {
		return "", invalidInstrumentAddress("机器人变量地址无效")
	}
	return address, nil
}

func normalizeOpaqueCollectorAddress(value any, maxLength int, message string) (string, error) {
	address, ok := value.(string)
	address = strings.TrimSpace(address)
	if !ok || address == "" || len([]rune(address)) > maxLength || strings.IndexFunc(address, unicode.IsSpace) >= 0 {
		return "", invalidInstrumentAddress(message)
	}
	return address, nil
}

func invalidInstrumentAddress(message string) error {
	return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, message)
}

// normalizeOrientalMotorAddress 统一 I/O 方向和偏移写法，保证相同内存位置只产生一个地址文本。
func normalizeOrientalMotorAddress(value any) (string, error) {
	address, ok := value.(string)
	if !ok {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "东方马达地址无效")
	}
	match := orientalMotorAddressPattern.FindStringSubmatch(strings.TrimSpace(address))
	if len(match) == 0 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "东方马达地址必须使用 input[n] 或 output[n]")
	}
	offset, err := strconv.ParseUint(match[2], 10, 63)
	if err != nil {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "东方马达地址偏移无效")
	}
	return fmt.Sprintf("%s[%d]", strings.ToLower(match[1]), offset), nil
}

// normalizeToyoPucAddress 统一程序号、软元件区和十六进制偏移，避免大小写与前导零导致重复变量。
func normalizeToyoPucAddress(value any) (string, error) {
	address, ok := value.(string)
	if !ok {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "东洋 PUC 地址无效")
	}
	match := toyoPucAddressPattern.FindStringSubmatch(strings.TrimSpace(address))
	if len(match) == 0 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "东洋 PUC 软元件地址无效")
	}
	prefix := ""
	if match[1] != "" {
		program, err := strconv.ParseUint(match[1], 10, 8)
		if err != nil {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "东洋 PUC 程序号必须在 0 到 255 之间")
		}
		prefix = fmt.Sprintf("prg=%d;", program)
	}
	offset, err := strconv.ParseUint(match[3], 16, 64)
	if err != nil {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "东洋 PUC 十六进制偏移无效")
	}
	return fmt.Sprintf("%s%s%X", prefix, strings.ToUpper(match[2]), offset), nil
}

// normalizeTurckAddress 将数字偏移转换为无前导零的标准文本。
func normalizeTurckAddress(value any) (string, error) {
	address, ok := value.(string)
	if !ok {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Turck Reader 地址无效")
	}
	offset, err := strconv.ParseUint(strings.TrimSpace(address), 10, 64)
	if err != nil {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Turck Reader 地址必须是非负数字偏移")
	}
	return strconv.FormatUint(offset, 10), nil
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
	case "modbus.tcp", "modbus.rtu", "modbus.rtu-over-tcp", "modbus.ascii", "modbus.ascii-over-tcp", "modbus.udp":
		station := numberText(address["station"])
		area, _ := address["area"].(string)
		offset := numberText(address["address"])
		result := station + ":" + area + ":" + offset
		if bit, ok := address["bitIndex"]; ok {
			result += "." + numberText(bit)
		}
		return result, nil
	case "siemens.s7-tcp", "siemens.ppi", "siemens.ppi-over-tcp", "siemens.mpi", "siemens.fetch-write":
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
	case "mitsubishi.mc-3e-tcp", "mitsubishi.a1e-ascii-tcp", "mitsubishi.a1e-binary-tcp",
		"mitsubishi.mc-ascii-tcp", "mitsubishi.mc-ascii-udp", "mitsubishi.mc-binary-udp",
		"mitsubishi.mc-r-binary-tcp", "mitsubishi.cip", "mitsubishi.a3c-serial",
		"mitsubishi.a3c-serial-over-tcp", "mitsubishi.fx-links-serial", "mitsubishi.fx-links-over-tcp",
		"mitsubishi.fx-serial", "mitsubishi.fx-serial-over-tcp":
		value, _ := address["address"].(string)
		value = strings.TrimSpace(value)
		if value == "" {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Mitsubishi MC 设备地址无效")
		}
		return value, nil
	case "omron.fins-tcp", "omron.fins-udp":
		value, _ := address["address"].(string)
		value = strings.TrimSpace(value)
		if value == "" {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Omron FINS 设备地址无效")
		}
		return value, nil
	case "omron.cip", "omron.connected-cip", "omron.hostlink", "omron.hostlink-over-tcp",
		"omron.hostlink-cmode", "omron.hostlink-cmode-over-tcp":
		value, _ := address["address"].(string)
		value = strings.TrimSpace(value)
		if value == "" {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Omron 设备地址无效")
		}
		return value, nil
	case "allen-bradley.ethernet-ip", "allen-bradley.connected-cip", "allen-bradley.micro-cip",
		"allen-bradley.pccc", "allen-bradley.slc", "allen-bradley.df1-serial":
		value, _ := address["address"].(string)
		value = strings.TrimSpace(value)
		if value == "" {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Allen-Bradley 标签地址无效")
		}
		return value, nil
	case "beckhoff.ads-tcp":
		value, _ := address["address"].(string)
		value = strings.TrimSpace(value)
		if value == "" {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "倍福 ADS 地址无效")
		}
		return value, nil
	case "iec.60870-5-104":
		informationType, _ := address["informationType"].(string)
		informationObjectAddress, ok := collectorNumericInt(address["informationObjectAddress"])
		if informationType == "" || !ok {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "IEC 104 地址无效")
		}
		return fmt.Sprintf("%s:%d", informationType, informationObjectAddress), nil
	case "panasonic.mewtocol-tcp", "panasonic.mewtocol-serial":
		value, _ := address["address"].(string)
		value = strings.TrimSpace(value)
		if value == "" {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "松下 Mewtocol 设备地址无效")
		}
		return value, nil
	case "panasonic.mc-binary-tcp":
		return normalizePanasonicMcAddress(address["address"], "")
	case "lsis.fast-enet", "lsis.cnet", "lsis.cnet-over-tcp", "lsis.cpu-serial":
		value, _ := address["address"].(string)
		value = strings.TrimSpace(value)
		if value == "" {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "LSIS 设备地址无效")
		}
		return value, nil
	case "ge.srtp-tcp":
		return normalizeGeSrtpAddress(address["address"])
	case "inovance.modbus-tcp", "inovance.modbus-serial", "inovance.modbus-rtu-over-tcp":
		return normalizeInovanceAddress(address["address"])
	case "inovance.connected-cip", "inovance.easy-net", "inovance.computer-link":
		return normalizeInovanceSpecialAddress(address["address"], driverID == "inovance.connected-cip")
	case "fatek.program-tcp", "fatek.program-serial":
		value, _ := address["address"].(string)
		value = strings.TrimSpace(value)
		if value == "" {
			return "", invalidFatekProgramAddress("永宏 PLC 设备地址无效")
		}
		return value, nil
	case "freedom.tcp", "freedom.udp", "freedom.serial":
		requestHex, err := normalizeFreedomRequestHex(address["requestHex"])
		if err != nil {
			return "", err
		}
		responseOffset, ok := collectorNumericInt(address["responseOffset"])
		if ok && responseOffset > 0 {
			return fmt.Sprintf("offset=%d;%s", responseOffset, requestHex), nil
		}
		return requestHex, nil
	case "cimon.hmi-protocol":
		return normalizeCimonAddress(address["address"])
	case "cjt188.serial", "cjt188.tcp":
		return normalizeCjt188Address(address["address"])
	case "rkc.temperature-controller-serial", "rkc.temperature-controller-tcp":
		return normalizeRkcAddress(address["address"])
	case "dam3601.serial":
		return normalizeDam3601Address(address["address"])
	case "yudian.ai-bus":
		return normalizeYuDianAddress(address["address"])
	case "delixi.dtsu6606":
		return normalizeDtsu6606Address(address["address"])
	case "dlt645.2007-serial", "dlt645.2007-over-tcp", "dlt698.serial", "dlt698.over-tcp", "dlt698.tcp-net":
		return normalizeDltAddress(address["address"], 4)
	case "dlt645.1997-serial", "dlt645.1997-over-tcp":
		return normalizeDltAddress(address["address"], 2)
	case "dcs.nanjing-auto":
		return normalizeDcsNanJingAutoAddress(address["address"])
	case "robot.estun-tcp", "robot.fanuc-interface":
		return normalizeRobotAddress(address["address"])
	case "siemens.s7-plus", "mqtt.rpc-device":
		return normalizeOpaqueCollectorAddress(address["address"], 256, "变量地址无效")
	case "ec-fan.machine-serial":
		return normalizeOpaqueCollectorAddress(address["address"], 32, "EC 风机数据项无效")
	case "siemens.web-api":
		value, _ := address["tag"].(string)
		value = strings.TrimSpace(value)
		if value == "" {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Siemens Web API 标签无效")
		}
		return value, nil
	case "keyence.mc-3e-tcp", "keyence.mc-ascii-tcp", "keyence.kv-old-tcp", "keyence.nano-tcp",
		"keyence.nano-serial", "keyence.nano-serial-over-tcp":
		return normalizeKeyenceAddress(address["address"], driverID)
	case "delta.tcp", "delta.rtu-over-tcp", "delta.ascii-over-tcp", "delta.rtu", "delta.ascii":
		return normalizeDeltaAddress(address["address"])
	case "xinje.tcp", "xinje.rtu-over-tcp", "xinje.rtu", "xinje.internal-tcp":
		return normalizeXinjeAddress(address["address"])
	case "megmeet.tcp", "megmeet.rtu-over-tcp", "megmeet.rtu":
		return normalizeMegMeetAddress(address["address"])
	case "fuji.command-setting-tcp", "fuji.sph-tcp":
		return normalizeFujiAddress(address["address"], driverID)
	case "fuji.spb-over-tcp", "fuji.spb":
		return normalizeFujiAddress(address["address"], driverID)
	case "vigor.serial-over-tcp", "vigor.serial":
		return normalizeVigorAddress(address["address"])
	case "yokogawa.link-tcp":
		return normalizeYokogawaAddress(address["address"], "")
	case "yamatake.digitron-serial", "yamatake.digitron-tcp":
		return normalizeYamatakeAddress(address["address"])
	case "yaskawa.memobus-tcp", "yaskawa.memobus-udp":
		return normalizeYaskawaAddress(address["address"])
	case "oriental-motor.eip":
		return normalizeOrientalMotorAddress(address["address"])
	case "toyo.puc":
		return normalizeToyoPucAddress(address["address"])
	case "turck.reader-tcp":
		return normalizeTurckAddress(address["address"])
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
	point := CollectorPoint{ID: record.ID, GroupID: record.GroupID, Code: record.Code, Name: record.Name, Description: record.Description, Address: record.Address, AddressText: record.AddressText, AddressSchemaVersion: record.AddressSchemaVersion, DataType: record.DataType, DataPointID: record.DataPointID, DataPointPath: record.DataPointPath, DataPointDefaultValue: cloneOptionalString(record.DataPointDefaultValue), ElementCount: record.ElementCount, ReadOptions: record.ReadOptions, Acquisition: record.Acquisition, AcquisitionMode: record.AcquisitionMode, AcquisitionOverrides: record.AcquisitionOverrides, Enabled: record.Enabled, SortOrder: record.SortOrder, Metadata: record.Metadata}
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
