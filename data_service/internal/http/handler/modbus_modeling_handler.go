package handler

import (
	"net/http"

	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// ModbusModelingHandler 暴露 Modbus 寄存器建模工作台接口。
type ModbusModelingHandler struct {
	service *service.ModbusModelingService
}

// NewModbusModelingHandler 创建 Modbus 建模接口处理器。
func NewModbusModelingHandler(service *service.ModbusModelingService) *ModbusModelingHandler {
	return &ModbusModelingHandler{service: service}
}

// ListGroups 返回当前 Modbus 接入源下的寄存器组。
func (h *ModbusModelingHandler) ListGroups(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	result, err := h.service.ListGroups(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"list": result})
	return nil
}

// CreateGroup 创建寄存器组。
func (h *ModbusModelingHandler) CreateGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		ParentID    *string `json:"parentId"`
		Name        string  `json:"name"`
		Description *string `json:"description"`
		SortOrder   int     `json:"sortOrder"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.CreateGroup(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, service.CreateModbusRegisterGroupInput{
		ParentID:    request.ParentID,
		Name:        request.Name,
		Description: request.Description,
		SortOrder:   request.SortOrder,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// UpdateGroup 更新寄存器组。
func (h *ModbusModelingHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		ParentID    *string `json:"parentId"`
		HasParentID bool    `json:"hasParentId"`
		Name        string  `json:"name"`
		Description *string `json:"description"`
		SortOrder   int     `json:"sortOrder"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.UpdateGroup(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("groupId"), claims.UserID, service.UpdateModbusRegisterGroupInput{
		ParentID:    request.ParentID,
		HasParentID: request.HasParentID,
		Name:        request.Name,
		Description: request.Description,
		SortOrder:   request.SortOrder,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// DeleteGroup 删除寄存器组，组内变量保留并移动到未分组。
func (h *ModbusModelingHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	if err := h.service.DeleteGroup(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("groupId"), claims.UserID); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}

// ListSlaveDevices 返回当前 Modbus 接入源下的从站设备。
func (h *ModbusModelingHandler) ListSlaveDevices(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	result, err := h.service.ListSlaveDevices(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"list": result})
	return nil
}

// CreateSlaveDevice 创建 Modbus 从站设备。
func (h *ModbusModelingHandler) CreateSlaveDevice(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request modbusSlaveDeviceRequest
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.CreateSlaveDevice(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, request.toCreateInput())
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// UpdateSlaveDevice 更新 Modbus 从站设备。
func (h *ModbusModelingHandler) UpdateSlaveDevice(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request modbusSlaveDeviceRequest
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.UpdateSlaveDevice(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("slaveId"), claims.UserID, request.toUpdateInput())
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// DeleteSlaveDevice 删除没有变量引用的 Modbus 从站设备。
func (h *ModbusModelingHandler) DeleteSlaveDevice(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	if err := h.service.DeleteSlaveDevice(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("slaveId"), claims.UserID); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}

// ListRegisters 返回变量列表。
func (h *ModbusModelingHandler) ListRegisters(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	query := r.URL.Query()
	var groupID *string
	if value := query.Get("groupId"); value != "" {
		groupID = &value
	}
	page, err := parseOptionalInt(query.Get("page"), 1, "page")
	if err != nil {
		return err
	}
	pageSize, err := parseOptionalInt(firstNonEmpty(query.Get("pageSize"), query.Get("limit")), 20, "pageSize")
	if err != nil {
		return err
	}
	var unitID *int
	if value := query.Get("unitId"); value != "" {
		parsed, parseErr := parseOptionalInt(value, 0, "unitId")
		if parseErr != nil {
			return parseErr
		}
		unitID = &parsed
	}
	var addressStart *int
	if value := query.Get("addressStart"); value != "" {
		parsed, parseErr := parseOptionalInt(value, 0, "addressStart")
		if parseErr != nil {
			return parseErr
		}
		addressStart = &parsed
	}
	var addressEnd *int
	if value := query.Get("addressEnd"); value != "" {
		parsed, parseErr := parseOptionalInt(value, 0, "addressEnd")
		if parseErr != nil {
			return parseErr
		}
		addressEnd = &parsed
	}
	var slaveEnabled *bool
	if value := query.Get("slaveEnabled"); value != "" {
		parsed := value == "true" || value == "1"
		slaveEnabled = &parsed
	}
	result, err := h.service.ListRegistersPage(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), service.ModbusRegisterListFilter{
		GroupID:      groupID,
		Search:       firstNonEmpty(query.Get("q"), query.Get("search")),
		QuickFilter:  query.Get("filter"),
		UnitID:       unitID,
		Area:         query.Get("area"),
		AddressStart: addressStart,
		AddressEnd:   addressEnd,
		DataType:     query.Get("dataType"),
		SlaveEnabled: slaveEnabled,
		SortBy:       firstNonEmpty(query.Get("sortBy"), query.Get("sort")),
		SortOrder:    firstNonEmpty(query.Get("sortOrder"), query.Get("order")),
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// CreateRegister 创建变量并自动同步数据点。
func (h *ModbusModelingHandler) CreateRegister(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request modbusRegisterRequest
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.CreateRegister(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, request.toCreateInput())
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// BatchImportRegisters 批量导入已确认的 Modbus 变量。
func (h *ModbusModelingHandler) BatchImportRegisters(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		GroupID   *string                             `json:"groupId"`
		Registers []service.ImportModbusRegisterInput `json:"registers"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.BatchImportRegisters(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, request.GroupID, request.Registers)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"list": result})
	return nil
}

// UpdateRegister 更新变量并同步数据点。
func (h *ModbusModelingHandler) UpdateRegister(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request modbusRegisterRequest
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.UpdateRegister(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("registerId"), claims.UserID, request.toUpdateInput())
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// DeleteRegister 删除变量并将关联数据点标记失效。
func (h *ModbusModelingHandler) DeleteRegister(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	if err := h.service.DeleteRegister(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("registerId"), claims.UserID); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}

// BatchDeleteRegisters 批量删除变量。
func (h *ModbusModelingHandler) BatchDeleteRegisters(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		IDs []string `json:"ids"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	count, err := h.service.DeleteRegistersBatch(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, service.ModbusRegisterBulkSelection{IDs: request.IDs})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]int{"count": count})
	return nil
}

// DeleteRegistersByFilter 按当前搜索/筛选条件批量删除 Modbus 变量。
func (h *ModbusModelingHandler) DeleteRegistersByFilter(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	selection, err := decodeModbusBulkSelection(r)
	if err != nil {
		return err
	}
	count, err := h.service.DeleteRegistersBatch(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, selection)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]int{"count": count})
	return nil
}

// BatchMoveRegistersGroup 批量移动变量分组。
func (h *ModbusModelingHandler) BatchMoveRegistersGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		IDs     []string `json:"ids"`
		GroupID *string  `json:"groupId"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	count, err := h.service.UpdateRegistersBatch(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, service.ModbusRegisterBulkUpdateInput{
		Selection:  service.ModbusRegisterBulkSelection{IDs: request.IDs},
		GroupID:    request.GroupID,
		HasGroupID: true,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]int{"count": count})
	return nil
}

// MoveRegistersByFilter 按当前搜索/筛选条件批量移动 Modbus 变量分组。
func (h *ModbusModelingHandler) MoveRegistersByFilter(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Filter  modbusRegisterFilterRequest `json:"filter"`
		GroupID *string                     `json:"groupId"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	count, err := h.service.UpdateRegistersBatch(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, service.ModbusRegisterBulkUpdateInput{
		Selection:  service.ModbusRegisterBulkSelection{Filter: request.Filter.toServiceFilter(), UseFilter: true},
		GroupID:    request.GroupID,
		HasGroupID: true,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]int{"count": count})
	return nil
}

// BatchUpdateRegisters 批量更新变量公共配置。
func (h *ModbusModelingHandler) BatchUpdateRegisters(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		IDs            []string `json:"ids"`
		UnitID         *int     `json:"unitId"`
		PollIntervalMS *int     `json:"pollIntervalMs"`
		ByteOrder      string   `json:"byteOrder"`
		WordOrder      string   `json:"wordOrder"`
		Status         string   `json:"status"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	count, err := h.service.UpdateRegistersBatch(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, service.ModbusRegisterBulkUpdateInput{
		Selection:      service.ModbusRegisterBulkSelection{IDs: request.IDs},
		UnitID:         request.UnitID,
		PollIntervalMS: request.PollIntervalMS,
		ByteOrder:      request.ByteOrder,
		WordOrder:      request.WordOrder,
		Status:         request.Status,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]int{"count": count})
	return nil
}

// UpdateRegistersByFilter 按当前搜索/筛选条件批量更新 Modbus 变量公共配置。
func (h *ModbusModelingHandler) UpdateRegistersByFilter(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Filter         modbusRegisterFilterRequest `json:"filter"`
		UnitID         *int                        `json:"unitId"`
		PollIntervalMS *int                        `json:"pollIntervalMs"`
		ByteOrder      string                      `json:"byteOrder"`
		WordOrder      string                      `json:"wordOrder"`
		Status         string                      `json:"status"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	count, err := h.service.UpdateRegistersBatch(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, service.ModbusRegisterBulkUpdateInput{
		Selection:      service.ModbusRegisterBulkSelection{Filter: request.Filter.toServiceFilter(), UseFilter: true},
		UnitID:         request.UnitID,
		PollIntervalMS: request.PollIntervalMS,
		ByteOrder:      request.ByteOrder,
		WordOrder:      request.WordOrder,
		Status:         request.Status,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]int{"count": count})
	return nil
}

// ValidateModel 执行建模校验。
func (h *ModbusModelingHandler) ValidateModel(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	result, err := h.service.ValidateModel(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// PreviewRegisters 返回辅助预览数据。真实采集由运行态执行。
func (h *ModbusModelingHandler) PreviewRegisters(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	var request struct {
		GroupID *string `json:"groupId"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.PreviewRegisters(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), request.GroupID)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// EstimateReadPlans 返回运行态读取预估。
func (h *ModbusModelingHandler) EstimateReadPlans(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	var groupID *string
	if value := r.URL.Query().Get("groupId"); value != "" {
		groupID = &value
	}
	var unitID *int
	if value := r.URL.Query().Get("unitId"); value != "" {
		parsed, parseErr := parseOptionalInt(value, 0, "unitId")
		if parseErr != nil {
			return parseErr
		}
		unitID = &parsed
	}
	result, err := h.service.EstimateReadPlans(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), groupID, unitID)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

type modbusRegisterFilterRequest struct {
	GroupID      *string `json:"groupId"`
	Search       string  `json:"search"`
	QuickFilter  string  `json:"filter"`
	UnitID       *int    `json:"unitId"`
	Area         string  `json:"area"`
	AddressStart *int    `json:"addressStart"`
	AddressEnd   *int    `json:"addressEnd"`
	DataType     string  `json:"dataType"`
	SlaveEnabled *bool   `json:"slaveEnabled"`
	SortBy       string  `json:"sortBy"`
	SortOrder    string  `json:"sortOrder"`
}

func (r modbusRegisterFilterRequest) toServiceFilter() service.ModbusRegisterListFilter {
	return service.ModbusRegisterListFilter{
		GroupID:      r.GroupID,
		Search:       r.Search,
		QuickFilter:  r.QuickFilter,
		UnitID:       r.UnitID,
		Area:         r.Area,
		AddressStart: r.AddressStart,
		AddressEnd:   r.AddressEnd,
		DataType:     r.DataType,
		SlaveEnabled: r.SlaveEnabled,
		SortBy:       r.SortBy,
		SortOrder:    r.SortOrder,
	}
}

func decodeModbusBulkSelection(r *http.Request) (service.ModbusRegisterBulkSelection, error) {
	var request struct {
		Filter modbusRegisterFilterRequest `json:"filter"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return service.ModbusRegisterBulkSelection{}, err
	}
	return service.ModbusRegisterBulkSelection{Filter: request.Filter.toServiceFilter(), UseFilter: true}, nil
}

type modbusRegisterRequest struct {
	GroupID         *string  `json:"groupId"`
	HasGroupID      bool     `json:"hasGroupId"`
	Name            string   `json:"name"`
	Code            string   `json:"code"`
	UnitID          *int     `json:"unitId"`
	Area            string   `json:"area"`
	Address         *int     `json:"address"`
	AddressBase     string   `json:"addressBase"`
	ProtocolAddress *int     `json:"protocolAddress"`
	Quantity        *int     `json:"quantity"`
	DataType        string   `json:"dataType"`
	ByteOrder       string   `json:"byteOrder"`
	WordOrder       string   `json:"wordOrder"`
	BitIndex        *int     `json:"bitIndex"`
	Scale           *float64 `json:"scale"`
	Offset          *float64 `json:"offset"`
	Unit            *string  `json:"unit"`
	PollIntervalMS  *int     `json:"pollIntervalMs"`
	TimeoutMS       *int     `json:"timeoutMs"`
	RetryCount      *int     `json:"retryCount"`
	AccessLevel     string   `json:"accessLevel"`
	Description     *string  `json:"description"`
	SortOrder       int      `json:"sortOrder"`
	Status          string   `json:"status"`
}

type modbusSlaveDeviceRequest struct {
	UnitID                *int    `json:"unitId"`
	Name                  string  `json:"name"`
	Description           *string `json:"description"`
	Enabled               *bool   `json:"enabled"`
	DefaultPollIntervalMS *int    `json:"defaultPollIntervalMs"`
	DefaultByteOrder      string  `json:"defaultByteOrder"`
	DefaultWordOrder      string  `json:"defaultWordOrder"`
	RequestIntervalMS     *int    `json:"requestIntervalMs"`
	TimeoutMS             *int    `json:"timeoutMs"`
	RetryCount            *int    `json:"retryCount"`
	SortOrder             int     `json:"sortOrder"`
}

func (r modbusSlaveDeviceRequest) toCreateInput() service.CreateModbusSlaveDeviceInput {
	return service.CreateModbusSlaveDeviceInput{
		UnitID:                r.UnitID,
		Name:                  r.Name,
		Description:           r.Description,
		Enabled:               r.Enabled,
		DefaultPollIntervalMS: r.DefaultPollIntervalMS,
		DefaultByteOrder:      r.DefaultByteOrder,
		DefaultWordOrder:      r.DefaultWordOrder,
		RequestIntervalMS:     r.RequestIntervalMS,
		TimeoutMS:             r.TimeoutMS,
		RetryCount:            r.RetryCount,
		SortOrder:             r.SortOrder,
	}
}

func (r modbusSlaveDeviceRequest) toUpdateInput() service.UpdateModbusSlaveDeviceInput {
	return service.UpdateModbusSlaveDeviceInput{
		UnitID:                r.UnitID,
		Name:                  r.Name,
		Description:           r.Description,
		Enabled:               r.Enabled,
		DefaultPollIntervalMS: r.DefaultPollIntervalMS,
		DefaultByteOrder:      r.DefaultByteOrder,
		DefaultWordOrder:      r.DefaultWordOrder,
		RequestIntervalMS:     r.RequestIntervalMS,
		TimeoutMS:             r.TimeoutMS,
		RetryCount:            r.RetryCount,
		SortOrder:             r.SortOrder,
	}
}

func (r modbusRegisterRequest) toCreateInput() service.CreateModbusRegisterInput {
	return service.CreateModbusRegisterInput{
		GroupID:         r.GroupID,
		Name:            r.Name,
		Code:            r.Code,
		UnitID:          r.UnitID,
		Area:            r.Area,
		Address:         r.Address,
		AddressBase:     r.AddressBase,
		ProtocolAddress: r.ProtocolAddress,
		Quantity:        r.Quantity,
		DataType:        r.DataType,
		ByteOrder:       r.ByteOrder,
		WordOrder:       r.WordOrder,
		BitIndex:        r.BitIndex,
		Scale:           r.Scale,
		Offset:          r.Offset,
		Unit:            r.Unit,
		PollIntervalMS:  r.PollIntervalMS,
		TimeoutMS:       r.TimeoutMS,
		RetryCount:      r.RetryCount,
		AccessLevel:     r.AccessLevel,
		Description:     r.Description,
		SortOrder:       r.SortOrder,
	}
}

func (r modbusRegisterRequest) toUpdateInput() service.UpdateModbusRegisterInput {
	return service.UpdateModbusRegisterInput{
		GroupID:         r.GroupID,
		HasGroupID:      r.HasGroupID,
		Name:            r.Name,
		Code:            r.Code,
		UnitID:          r.UnitID,
		Area:            r.Area,
		Address:         r.Address,
		AddressBase:     r.AddressBase,
		ProtocolAddress: r.ProtocolAddress,
		Quantity:        r.Quantity,
		DataType:        r.DataType,
		ByteOrder:       r.ByteOrder,
		WordOrder:       r.WordOrder,
		BitIndex:        r.BitIndex,
		Scale:           r.Scale,
		Offset:          r.Offset,
		Unit:            r.Unit,
		PollIntervalMS:  r.PollIntervalMS,
		TimeoutMS:       r.TimeoutMS,
		RetryCount:      r.RetryCount,
		AccessLevel:     r.AccessLevel,
		Description:     r.Description,
		SortOrder:       r.SortOrder,
		Status:          r.Status,
	}
}
