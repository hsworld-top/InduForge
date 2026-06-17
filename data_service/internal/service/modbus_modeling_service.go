package service

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

var modbusCodeSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

// ModbusRegisterGroup 表示前端工作台使用的 Modbus 寄存器组。
type ModbusRegisterGroup struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"projectId"`
	ConnectionID string    `json:"connectionId"`
	ParentID     *string   `json:"parentId"`
	Name         string    `json:"name"`
	Description  *string   `json:"description"`
	SortOrder    int       `json:"sortOrder"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// ModbusSlaveDevice 表示 Modbus 接入源下的真实从站设备。
type ModbusSlaveDevice struct {
	ID                    string    `json:"id"`
	ProjectID             string    `json:"projectId"`
	ConnectionID          string    `json:"connectionId"`
	UnitID                int       `json:"unitId"`
	Name                  string    `json:"name"`
	Description           *string   `json:"description"`
	Enabled               bool      `json:"enabled"`
	DefaultPollIntervalMS int       `json:"defaultPollIntervalMs"`
	DefaultByteOrder      string    `json:"defaultByteOrder"`
	DefaultWordOrder      string    `json:"defaultWordOrder"`
	RequestIntervalMS     *int      `json:"requestIntervalMs"`
	TimeoutMS             *int      `json:"timeoutMs"`
	RetryCount            *int      `json:"retryCount"`
	SortOrder             int       `json:"sortOrder"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
	RegisterCount         int       `json:"registerCount,omitempty"`
}

// ModbusRegister 表示前端工作台使用的 Modbus 变量。
type ModbusRegister struct {
	ID              string     `json:"id"`
	ProjectID       string     `json:"projectId"`
	ConnectionID    string     `json:"connectionId"`
	GroupID         *string    `json:"groupId"`
	Name            string     `json:"name"`
	Code            string     `json:"code"`
	UnitID          int        `json:"unitId"`
	Area            string     `json:"area"`
	Address         int        `json:"address"`
	AddressBase     string     `json:"addressBase"`
	ProtocolAddress int        `json:"protocolAddress"`
	Quantity        int        `json:"quantity"`
	DataType        string     `json:"dataType"`
	ByteOrder       string     `json:"byteOrder"`
	WordOrder       string     `json:"wordOrder"`
	BitIndex        *int       `json:"bitIndex"`
	Scale           float64    `json:"scale"`
	Offset          float64    `json:"offset"`
	Unit            *string    `json:"unit"`
	PollIntervalMS  int        `json:"pollIntervalMs"`
	TimeoutMS       *int       `json:"timeoutMs"`
	RetryCount      *int       `json:"retryCount"`
	AccessLevel     string     `json:"accessLevel"`
	Description     *string    `json:"description"`
	SortOrder       int        `json:"sortOrder"`
	Status          string     `json:"status"`
	DataPointID     *string    `json:"datapointId"`
	DataPointPath   *string    `json:"datapointPath"`
	DataPointStatus *string    `json:"datapointStatus"`
	LastValue       any        `json:"lastValue,omitempty"`
	Quality         string     `json:"quality"`
	LastUpdatedAt   *time.Time `json:"lastUpdatedAt,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

// ModbusReadPlan 表示运行态读取预估中的一次可合并读取。
type ModbusReadPlan struct {
	ID                string   `json:"id"`
	ConnectionID      string   `json:"connectionId"`
	UnitID            int      `json:"unitId"`
	Area              string   `json:"area"`
	StartAddress      int      `json:"startAddress"`
	EndAddress        int      `json:"endAddress"`
	ProtocolStart     int      `json:"protocolStart"`
	Quantity          int      `json:"quantity"`
	PollIntervalMS    int      `json:"pollIntervalMs"`
	RegisterIDs       []string `json:"registerIds"`
	RegisterCount     int      `json:"registerCount"`
	ReadsPerSecond    float64  `json:"readsPerSecond"`
	DisplayRange      string   `json:"displayRange"`
	DisplayArea       string   `json:"displayArea"`
	DisplayCycle      string   `json:"displayCycle"`
	EstimatedPressure string   `json:"estimatedPressure"`
}

// ModbusReadPlanEstimate 表示连接级读取预估。
type ModbusReadPlanEstimate struct {
	RegisterCount  int              `json:"registerCount"`
	UnitCount      int              `json:"unitCount"`
	ReadCount      int              `json:"readCount"`
	ReadsPerSecond float64          `json:"readsPerSecond"`
	Plans          []ModbusReadPlan `json:"plans"`
	Diagnostics    []string         `json:"diagnostics"`
}

// ModbusValidationIssue 表示 Modbus 建模校验问题。
type ModbusValidationIssue struct {
	Severity     string  `json:"severity"`
	Code         string  `json:"code"`
	GroupID      *string `json:"groupId,omitempty"`
	RegisterID   *string `json:"registerId,omitempty"`
	RegisterName string  `json:"registerName,omitempty"`
	Message      string  `json:"message"`
}

// ModbusValidationResult 表示 Modbus 建模校验结果。
type ModbusValidationResult struct {
	Valid  bool                    `json:"valid"`
	Issues []ModbusValidationIssue `json:"issues"`
}

// CreateModbusRegisterGroupInput 描述创建寄存器组输入。
type CreateModbusRegisterGroupInput struct {
	ParentID    *string
	Name        string
	Description *string
	SortOrder   int
}

// UpdateModbusRegisterGroupInput 描述更新寄存器组输入。
type UpdateModbusRegisterGroupInput struct {
	ParentID    *string
	HasParentID bool
	Name        string
	Description *string
	SortOrder   int
}

// CreateModbusSlaveDeviceInput 描述创建从站输入。
type CreateModbusSlaveDeviceInput struct {
	UnitID                *int
	Name                  string
	Description           *string
	Enabled               *bool
	DefaultPollIntervalMS *int
	DefaultByteOrder      string
	DefaultWordOrder      string
	RequestIntervalMS     *int
	TimeoutMS             *int
	RetryCount            *int
	SortOrder             int
}

// UpdateModbusSlaveDeviceInput 描述更新从站输入。
type UpdateModbusSlaveDeviceInput struct {
	UnitID                *int
	Name                  string
	Description           *string
	Enabled               *bool
	DefaultPollIntervalMS *int
	DefaultByteOrder      string
	DefaultWordOrder      string
	RequestIntervalMS     *int
	TimeoutMS             *int
	RetryCount            *int
	SortOrder             int
}

// CreateModbusRegisterInput 描述创建变量输入。
type CreateModbusRegisterInput struct {
	GroupID         *string
	Name            string
	Code            string
	UnitID          *int
	Area            string
	Address         *int
	AddressBase     string
	ProtocolAddress *int
	Quantity        *int
	DataType        string
	ByteOrder       string
	WordOrder       string
	BitIndex        *int
	Scale           *float64
	Offset          *float64
	Unit            *string
	PollIntervalMS  *int
	TimeoutMS       *int
	RetryCount      *int
	AccessLevel     string
	Description     *string
	SortOrder       int
}

// UpdateModbusRegisterInput 描述更新变量输入。
type UpdateModbusRegisterInput struct {
	GroupID         *string
	HasGroupID      bool
	Name            string
	Code            string
	UnitID          *int
	Area            string
	Address         *int
	AddressBase     string
	ProtocolAddress *int
	Quantity        *int
	DataType        string
	ByteOrder       string
	WordOrder       string
	BitIndex        *int
	Scale           *float64
	Offset          *float64
	Unit            *string
	PollIntervalMS  *int
	TimeoutMS       *int
	RetryCount      *int
	AccessLevel     string
	Description     *string
	SortOrder       int
	Status          string
}

// ImportModbusRegisterInput 描述批量导入的单个变量输入。
type ImportModbusRegisterInput struct {
	Name           string   `json:"name"`
	Code           string   `json:"code"`
	UnitID         *int     `json:"unitId"`
	Area           string   `json:"area"`
	Address        int      `json:"address"`
	AddressBase    string   `json:"addressBase"`
	DataType       string   `json:"dataType"`
	ByteOrder      string   `json:"byteOrder"`
	WordOrder      string   `json:"wordOrder"`
	Scale          *float64 `json:"scale"`
	Offset         *float64 `json:"offset"`
	Unit           *string  `json:"unit"`
	PollIntervalMS *int     `json:"pollIntervalMs"`
	AccessLevel    string   `json:"accessLevel"`
	Description    *string  `json:"description"`
}

// ModbusPreviewResult 表示开发态辅助预览结果。
type ModbusPreviewResult struct {
	Values      []ModbusRegister `json:"values"`
	Diagnostics []string         `json:"diagnostics"`
}

// ModbusModelingService 承载 Modbus 寄存器建模业务规则。
type ModbusModelingService struct {
	repository  *repository.ModbusModelingRepository
	connections *repository.ConnectionRepository
	datapoints  *repository.DataPointRepository
}

// NewModbusModelingService 创建 Modbus 建模服务。
func NewModbusModelingService(repo *repository.ModbusModelingRepository, connections *repository.ConnectionRepository, datapoints *repository.DataPointRepository) *ModbusModelingService {
	return &ModbusModelingService{repository: repo, connections: connections, datapoints: datapoints}
}

// ListGroups 返回寄存器组。
func (s *ModbusModelingService) ListGroups(ctx context.Context, projectID, connectionID string) ([]ModbusRegisterGroup, error) {
	if err := s.validateProjectConnection(ctx, projectID, connectionID); err != nil {
		return nil, err
	}
	records, err := s.repository.ListGroups(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	result := make([]ModbusRegisterGroup, 0, len(records))
	for _, record := range records {
		result = append(result, toModbusRegisterGroup(record))
	}
	return result, nil
}

// CreateGroup 创建寄存器组。
func (s *ModbusModelingService) CreateGroup(ctx context.Context, projectID, connectionID, userID string, input CreateModbusRegisterGroupInput) (*ModbusRegisterGroup, error) {
	if err := s.validateProjectConnectionAndUser(ctx, projectID, connectionID, userID); err != nil {
		return nil, err
	}
	name, err := normalizeModbusRequiredText(input.Name, "寄存器组名称不能为空")
	if err != nil {
		return nil, err
	}
	record, err := s.repository.CreateGroup(ctx, repository.CreateModbusRegisterGroupParams{
		ProjectID:    projectID,
		ConnectionID: connectionID,
		ParentID:     normalizeOptionalText(input.ParentID),
		Name:         name,
		Description:  normalizeOptionalText(input.Description),
		SortOrder:    input.SortOrder,
		UserID:       userID,
	})
	if err != nil {
		return nil, err
	}
	result := toModbusRegisterGroup(*record)
	return &result, nil
}

// UpdateGroup 更新寄存器组。
func (s *ModbusModelingService) UpdateGroup(ctx context.Context, projectID, connectionID, groupID, userID string, input UpdateModbusRegisterGroupInput) (*ModbusRegisterGroup, error) {
	if err := s.validateProjectConnectionAndUser(ctx, projectID, connectionID, userID); err != nil {
		return nil, err
	}
	name, err := normalizeModbusRequiredText(input.Name, "寄存器组名称不能为空")
	if err != nil {
		return nil, err
	}
	record, err := s.repository.UpdateGroup(ctx, repository.UpdateModbusRegisterGroupParams{
		ID:           groupID,
		ProjectID:    projectID,
		ConnectionID: connectionID,
		ParentID:     normalizeOptionalText(input.ParentID),
		HasParentID:  input.HasParentID,
		Name:         name,
		Description:  normalizeOptionalText(input.Description),
		SortOrder:    input.SortOrder,
		UserID:       userID,
	})
	if err != nil {
		return nil, err
	}
	result := toModbusRegisterGroup(*record)
	return &result, nil
}

// DeleteGroup 删除寄存器组。
func (s *ModbusModelingService) DeleteGroup(ctx context.Context, projectID, connectionID, groupID, userID string) error {
	if err := s.validateProjectConnectionAndUser(ctx, projectID, connectionID, userID); err != nil {
		return err
	}
	return s.repository.DeleteGroup(ctx, projectID, connectionID, groupID, userID)
}

// ListSlaveDevices 返回 Modbus 从站设备。
func (s *ModbusModelingService) ListSlaveDevices(ctx context.Context, projectID, connectionID string) ([]ModbusSlaveDevice, error) {
	if err := s.validateProjectConnection(ctx, projectID, connectionID); err != nil {
		return nil, err
	}
	records, err := s.repository.ListSlaveDevices(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	registers, err := s.repository.ListRegisters(ctx, projectID, connectionID, nil)
	if err != nil {
		return nil, err
	}
	countByUnit := map[int]int{}
	for _, register := range registers {
		countByUnit[register.UnitID]++
	}
	result := make([]ModbusSlaveDevice, 0, len(records))
	for _, record := range records {
		item := toModbusSlaveDevice(record)
		item.RegisterCount = countByUnit[item.UnitID]
		result = append(result, item)
	}
	return result, nil
}

// CreateSlaveDevice 创建 Modbus 从站设备。
func (s *ModbusModelingService) CreateSlaveDevice(ctx context.Context, projectID, connectionID, userID string, input CreateModbusSlaveDeviceInput) (*ModbusSlaveDevice, error) {
	if err := s.validateProjectConnectionAndUser(ctx, projectID, connectionID, userID); err != nil {
		return nil, err
	}
	params, err := s.normalizeCreateSlaveDeviceInput(projectID, connectionID, userID, input)
	if err != nil {
		return nil, err
	}
	record, err := s.repository.CreateSlaveDevice(ctx, params)
	if err != nil {
		return nil, err
	}
	result := toModbusSlaveDevice(*record)
	return &result, nil
}

// UpdateSlaveDevice 更新 Modbus 从站设备。
func (s *ModbusModelingService) UpdateSlaveDevice(ctx context.Context, projectID, connectionID, slaveID, userID string, input UpdateModbusSlaveDeviceInput) (*ModbusSlaveDevice, error) {
	if err := s.validateProjectConnectionAndUser(ctx, projectID, connectionID, userID); err != nil {
		return nil, err
	}
	params, err := s.normalizeUpdateSlaveDeviceInput(projectID, connectionID, slaveID, userID, input)
	if err != nil {
		return nil, err
	}
	record, err := s.repository.UpdateSlaveDevice(ctx, params)
	if err != nil {
		return nil, err
	}
	result := toModbusSlaveDevice(*record)
	return &result, nil
}

// DeleteSlaveDevice 删除没有变量引用的 Modbus 从站设备。
func (s *ModbusModelingService) DeleteSlaveDevice(ctx context.Context, projectID, connectionID, slaveID, userID string) error {
	if err := s.validateProjectConnectionAndUser(ctx, projectID, connectionID, userID); err != nil {
		return err
	}
	return s.repository.DeleteSlaveDevice(ctx, projectID, connectionID, slaveID)
}

// ListRegisters 返回变量列表。
func (s *ModbusModelingService) ListRegisters(ctx context.Context, projectID, connectionID string, groupID *string) ([]ModbusRegister, error) {
	if err := s.validateProjectConnection(ctx, projectID, connectionID); err != nil {
		return nil, err
	}
	records, err := s.repository.ListRegisters(ctx, projectID, connectionID, normalizeOptionalText(groupID))
	if err != nil {
		return nil, err
	}
	result := make([]ModbusRegister, 0, len(records))
	for _, record := range records {
		result = append(result, toModbusRegister(record))
	}
	return result, nil
}

type ModbusRegisterListResult struct {
	Registers  []ModbusRegister           `json:"list"`
	Pagination ProtocolModelingPagination `json:"pagination"`
}

// ModbusRegisterListFilter 是 Modbus 变量列表的查询条件。
type ModbusRegisterListFilter struct {
	GroupID      *string
	Search       string
	QuickFilter  string
	UnitID       *int
	Area         string
	AddressStart *int
	AddressEnd   *int
	DataType     string
	SlaveEnabled *bool
	SortBy       string
	SortOrder    string
	Page         int
	PageSize     int
}

// ModbusRegisterBulkUpdateInput 描述变量批量更新输入。
type ModbusRegisterBulkUpdateInput struct {
	IDs            []string
	Selection      ModbusRegisterBulkSelection
	GroupID        *string
	HasGroupID     bool
	UnitID         *int
	PollIntervalMS *int
	ByteOrder      string
	WordOrder      string
	Status         string
}

// ModbusRegisterBulkSelection 描述批量操作选择范围；IDs 用于显式勾选，Filter 用于“全部结果”。
type ModbusRegisterBulkSelection struct {
	IDs       []string
	Filter    ModbusRegisterListFilter
	UseFilter bool
}

// ListRegistersPage 返回当前分组下的一页变量，分页条件只影响列表展示，不影响预览和校验等全量流程。
func (s *ModbusModelingService) ListRegistersPage(ctx context.Context, projectID, connectionID string, filter ModbusRegisterListFilter) (*ModbusRegisterListResult, error) {
	if err := s.validateProjectConnection(ctx, projectID, connectionID); err != nil {
		return nil, err
	}
	page, pageSize := normalizePageAndSize(filter.Page, filter.PageSize, 1, 100)
	records, total, err := s.repository.ListRegistersPage(ctx, projectID, connectionID, toRepositoryModbusRegisterFilter(filter), page, pageSize)
	if err != nil {
		return nil, err
	}
	registers := make([]ModbusRegister, 0, len(records))
	for _, record := range records {
		registers = append(registers, toModbusRegister(record))
	}
	return &ModbusRegisterListResult{
		Registers:  registers,
		Pagination: newProtocolModelingPagination(page, pageSize, total),
	}, nil
}

// CreateRegister 创建变量并同步数据点。
func (s *ModbusModelingService) CreateRegister(ctx context.Context, projectID, connectionID, userID string, input CreateModbusRegisterInput) (*ModbusRegister, error) {
	if err := s.validateProjectConnectionAndUser(ctx, projectID, connectionID, userID); err != nil {
		return nil, err
	}
	params, err := s.normalizeCreateRegisterInput(ctx, projectID, connectionID, userID, input)
	if err != nil {
		return nil, err
	}
	record, err := s.repository.CreateRegister(ctx, params)
	if err != nil {
		return nil, err
	}
	if err := s.syncRegisterDatapoint(ctx, *record, userID); err != nil {
		return nil, err
	}
	loaded, err := s.repository.GetRegister(ctx, projectID, record.ID)
	if err != nil {
		return nil, err
	}
	result := toModbusRegister(*loaded)
	return &result, nil
}

// BatchImportRegisters 批量导入变量。
func (s *ModbusModelingService) BatchImportRegisters(ctx context.Context, projectID, connectionID, userID string, groupID *string, registers []ImportModbusRegisterInput) ([]ModbusRegister, error) {
	if len(registers) == 0 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "导入变量不能为空")
	}
	result := make([]ModbusRegister, 0, len(registers))
	for index, item := range registers {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = fmt.Sprintf("%s_%d", normalizeModbusArea(item.Area), item.Address)
		}
		input := CreateModbusRegisterInput{
			GroupID:        groupID,
			Name:           name,
			Code:           item.Code,
			UnitID:         item.UnitID,
			Area:           item.Area,
			Address:        &item.Address,
			AddressBase:    item.AddressBase,
			DataType:       item.DataType,
			ByteOrder:      item.ByteOrder,
			WordOrder:      item.WordOrder,
			Scale:          item.Scale,
			Offset:         item.Offset,
			Unit:           item.Unit,
			PollIntervalMS: item.PollIntervalMS,
			AccessLevel:    item.AccessLevel,
			Description:    item.Description,
			SortOrder:      index,
		}
		created, err := s.CreateRegister(ctx, projectID, connectionID, userID, input)
		if err != nil {
			return nil, err
		}
		result = append(result, *created)
	}
	return result, nil
}

// UpdateRegisterLastValue 保存开发态读取后的最近值、质量和时间戳。
func (s *ModbusModelingService) UpdateRegisterLastValue(ctx context.Context, projectID, connectionID, registerID, userID string, value any, quality string) error {
	return s.repository.UpdateRegisterLastValue(ctx, repository.UpdateModbusRegisterLastValueParams{
		ProjectID:    projectID,
		ConnectionID: connectionID,
		RegisterID:   registerID,
		LastValue:    value,
		Quality:      firstNonEmpty(quality, "Good"),
		UserID:       userID,
	})
}

// DeleteRegistersBatch 批量删除变量并标记数据点失效。
func (s *ModbusModelingService) DeleteRegistersBatch(ctx context.Context, projectID, connectionID, userID string, selection ModbusRegisterBulkSelection) (int, error) {
	if err := s.validateProjectConnectionAndUser(ctx, projectID, connectionID, userID); err != nil {
		return 0, err
	}
	ids, err := s.resolveBulkRegisterIDs(ctx, projectID, connectionID, selection)
	if err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	deleted, err := s.repository.DeleteRegistersBatch(ctx, projectID, connectionID, ids, userID)
	if err != nil {
		return 0, err
	}
	return len(deleted), nil
}

// UpdateRegistersBatch 批量更新变量公共字段并同步数据点。
func (s *ModbusModelingService) UpdateRegistersBatch(ctx context.Context, projectID, connectionID, userID string, input ModbusRegisterBulkUpdateInput) (int, error) {
	if err := s.validateProjectConnectionAndUser(ctx, projectID, connectionID, userID); err != nil {
		return 0, err
	}
	selection := input.Selection
	if len(selection.IDs) == 0 && !selection.UseFilter {
		selection.IDs = input.IDs
	}
	if input.UnitID != nil {
		if err := s.ensureSlaveDevice(ctx, projectID, connectionID, userID, *input.UnitID); err != nil {
			return 0, err
		}
	}
	if input.PollIntervalMS != nil && *input.PollIntervalMS <= 0 {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "轮询周期必须大于 0")
	}
	status := ""
	if strings.TrimSpace(input.Status) != "" {
		status = strings.TrimSpace(input.Status)
		if _, ok := allowedDataPointStatuses[status]; !ok {
			return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "变量状态不合法")
		}
	}
	records, err := s.repository.UpdateRegistersBatch(ctx, repository.BatchUpdateModbusRegistersParams{
		ProjectID:      projectID,
		ConnectionID:   connectionID,
		IDs:            selection.IDs,
		Filter:         toRepositoryModbusRegisterFilter(selection.Filter),
		UseFilter:      selection.UseFilter,
		GroupID:        normalizeOptionalText(input.GroupID),
		HasGroupID:     input.HasGroupID,
		UnitID:         input.UnitID,
		PollIntervalMS: input.PollIntervalMS,
		ByteOrder:      normalizeOptionalValue(input.ByteOrder, normalizeModbusByteOrder),
		WordOrder:      normalizeOptionalValue(input.WordOrder, normalizeModbusWordOrder),
		Status:         status,
		UserID:         userID,
	})
	if err != nil {
		return 0, err
	}
	for _, record := range records {
		if err := s.syncRegisterDatapoint(ctx, record, userID); err != nil {
			return 0, err
		}
	}
	return len(records), nil
}

// UpdateRegister 更新变量并同步数据点。
func (s *ModbusModelingService) UpdateRegister(ctx context.Context, projectID, connectionID, registerID, userID string, input UpdateModbusRegisterInput) (*ModbusRegister, error) {
	if err := s.validateProjectConnectionAndUser(ctx, projectID, connectionID, userID); err != nil {
		return nil, err
	}
	current, err := s.repository.GetRegister(ctx, projectID, registerID)
	if err != nil {
		return nil, err
	}
	if current.ConnectionID != connectionID {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "Modbus 变量不存在")
	}
	params, err := s.normalizeUpdateRegisterInput(ctx, projectID, userID, *current, input)
	if err != nil {
		return nil, err
	}
	record, err := s.repository.UpdateRegister(ctx, params)
	if err != nil {
		return nil, err
	}
	if err := s.syncRegisterDatapoint(ctx, *record, userID); err != nil {
		return nil, err
	}
	loaded, err := s.repository.GetRegister(ctx, projectID, record.ID)
	if err != nil {
		return nil, err
	}
	result := toModbusRegister(*loaded)
	return &result, nil
}

// DeleteRegister 删除变量并将数据点标记失效。
func (s *ModbusModelingService) DeleteRegister(ctx context.Context, projectID, connectionID, registerID, userID string) error {
	if err := s.validateProjectConnectionAndUser(ctx, projectID, connectionID, userID); err != nil {
		return err
	}
	current, err := s.repository.GetRegister(ctx, projectID, registerID)
	if err != nil {
		return err
	}
	if current.ConnectionID != connectionID {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "Modbus 变量不存在")
	}
	if err := s.repository.DeleteRegister(ctx, projectID, connectionID, registerID); err != nil {
		return err
	}
	_, _ = s.datapoints.MarkInvalidBySource(ctx, projectID, "modbus.register", registerID, stringPtr(userID))
	return nil
}

// ValidateModel 校验当前连接下的 Modbus 建模结果。
func (s *ModbusModelingService) ValidateModel(ctx context.Context, projectID, connectionID string) (*ModbusValidationResult, error) {
	registers, err := s.ListRegisters(ctx, projectID, connectionID, nil)
	if err != nil {
		return nil, err
	}
	slaves, err := s.ListSlaveDevices(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	issues := make([]ModbusValidationIssue, 0)
	slaveByUnit := make(map[int]ModbusSlaveDevice, len(slaves))
	for _, slave := range slaves {
		slaveByUnit[slave.UnitID] = slave
		if slave.UnitID < 0 || slave.UnitID > 247 {
			issues = append(issues, ModbusValidationIssue{
				Severity: "error",
				Code:     "slave.unit.invalid",
				Message:  fmt.Sprintf("从站「%s」地址必须在 0-247 范围内", slave.Name),
			})
		}
	}
	codeSeen := map[string]string{}
	type rangeItem struct {
		register ModbusRegister
		start    int
		end      int
	}
	ranges := map[string][]rangeItem{}
	for _, register := range registers {
		registerID := register.ID
		if strings.TrimSpace(register.Name) == "" {
			issues = append(issues, modbusRegisterIssue("error", "name.empty", registerID, register.Name, "变量名不能为空"))
		}
		if strings.TrimSpace(register.Code) == "" {
			issues = append(issues, modbusRegisterIssue("error", "code.empty", registerID, register.Name, "变量标识符不能为空"))
		}
		if previous, ok := codeSeen[register.Code]; ok && previous != register.ID {
			issues = append(issues, modbusRegisterIssue("error", "code.duplicate", registerID, register.Name, "变量标识符重复"))
		}
		codeSeen[register.Code] = register.ID
		if register.UnitID < 0 || register.UnitID > 247 {
			issues = append(issues, modbusRegisterIssue("error", "unit.invalid", registerID, register.Name, "从站地址必须在 0-247 范围内"))
		}
		slave, slaveExists := slaveByUnit[register.UnitID]
		if !slaveExists {
			issues = append(issues, modbusRegisterIssue("error", "slave.missing", registerID, register.Name, "变量引用的从站不存在"))
		} else if !slave.Enabled {
			issues = append(issues, modbusRegisterIssue("warning", "slave.disabled", registerID, register.Name, fmt.Sprintf("所属从站「%s」已停用，运行态不会采集该变量", slave.Name)))
		}
		if _, ok := allowedModbusAreas[register.Area]; !ok {
			issues = append(issues, modbusRegisterIssue("error", "area.invalid", registerID, register.Name, "Modbus 数据区不合法"))
		}
		if register.ProtocolAddress < 0 {
			issues = append(issues, modbusRegisterIssue("error", "address.invalid", registerID, register.Name, "协议地址不能小于 0"))
		}
		if register.Quantity <= 0 {
			issues = append(issues, modbusRegisterIssue("error", "quantity.invalid", registerID, register.Name, "寄存器数量必须大于 0"))
		}
		if register.PollIntervalMS <= 0 {
			issues = append(issues, modbusRegisterIssue("error", "poll.invalid", registerID, register.Name, "轮询周期必须大于 0"))
		}
		if issue := validateModbusDataType(register); issue != "" {
			issues = append(issues, modbusRegisterIssue("error", "dataType.invalid", registerID, register.Name, issue))
		}
		if (register.Area == "discrete_input" || register.Area == "input_register") && register.AccessLevel != "Read" {
			issues = append(issues, modbusRegisterIssue("error", "access.invalid", registerID, register.Name, "输入区域不允许写入"))
		}
		if register.DataPointPath == nil || strings.TrimSpace(*register.DataPointPath) == "" {
			issues = append(issues, modbusRegisterIssue("error", "datapoint.missing", registerID, register.Name, "数据点未生成"))
		}
		if register.DataPointStatus != nil && *register.DataPointStatus == "invalid" {
			issues = append(issues, modbusRegisterIssue("warning", "datapoint.invalid", registerID, register.Name, "数据点已失效"))
		}
		key := fmt.Sprintf("%d|%s", register.UnitID, register.Area)
		ranges[key] = append(ranges[key], rangeItem{register: register, start: register.ProtocolAddress, end: register.ProtocolAddress + register.Quantity - 1})
	}
	for _, items := range ranges {
		sort.Slice(items, func(i, j int) bool {
			return items[i].start < items[j].start
		})
		for index := 1; index < len(items); index++ {
			if items[index].start <= items[index-1].end {
				issues = append(issues, modbusRegisterIssue("warning", "address.overlap", items[index].register.ID, items[index].register.Name, "地址范围与另一个变量重叠"))
			}
		}
	}
	return &ModbusValidationResult{Valid: len(issues) == 0, Issues: issues}, nil
}

// PreviewRegisters 返回开发态辅助预览占位结果。真实采集由运行态执行。
func (s *ModbusModelingService) PreviewRegisters(ctx context.Context, projectID, connectionID string, groupID *string) (*ModbusPreviewResult, error) {
	registers, err := s.ListRegisters(ctx, projectID, connectionID, groupID)
	if err != nil {
		return nil, err
	}
	estimate := BuildModbusReadPlanEstimate(registers)
	return &ModbusPreviewResult{
		Values: registers,
		Diagnostics: []string{
			fmt.Sprintf("Modbus 真实采集由运行态执行；当前预览返回 %d 个已建模变量用于核对地址、类型和数据点。", len(registers)),
			fmt.Sprintf("当前范围预计合并为 %d 次读取，预计 %.2f reads/s。", estimate.ReadCount, estimate.ReadsPerSecond),
		},
	}, nil
}

// EstimateReadPlans 返回运行态读取预估。
func (s *ModbusModelingService) EstimateReadPlans(ctx context.Context, projectID, connectionID string, groupID *string, unitID *int) (*ModbusReadPlanEstimate, error) {
	registers, err := s.ListRegisters(ctx, projectID, connectionID, groupID)
	if err != nil {
		return nil, err
	}
	if unitID != nil {
		filtered := make([]ModbusRegister, 0, len(registers))
		for _, register := range registers {
			if register.UnitID == *unitID {
				filtered = append(filtered, register)
			}
		}
		registers = filtered
	}
	slaves, err := s.ListSlaveDevices(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	enabledByUnit := make(map[int]bool, len(slaves))
	for _, slave := range slaves {
		enabledByUnit[slave.UnitID] = slave.Enabled
	}
	filtered := make([]ModbusRegister, 0, len(registers))
	skipped := 0
	for _, register := range registers {
		if enabled, ok := enabledByUnit[register.UnitID]; ok && !enabled {
			skipped++
			continue
		}
		filtered = append(filtered, register)
	}
	estimate := BuildModbusReadPlanEstimate(filtered)
	if skipped > 0 {
		estimate.Diagnostics = append(estimate.Diagnostics, fmt.Sprintf("已跳过 %d 个禁用从站下的变量。", skipped))
	}
	return &estimate, nil
}

func (s *ModbusModelingService) normalizeCreateSlaveDeviceInput(projectID, connectionID, userID string, input CreateModbusSlaveDeviceInput) (repository.CreateModbusSlaveDeviceParams, error) {
	unitID := intOrDefault(input.UnitID, 1)
	if unitID < 0 || unitID > 247 {
		return repository.CreateModbusSlaveDeviceParams{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "从站地址必须在 0-247 范围内")
	}
	name, err := normalizeModbusRequiredText(firstNonEmpty(input.Name, fmt.Sprintf("从站 %d", unitID)), "从站名称不能为空")
	if err != nil {
		return repository.CreateModbusSlaveDeviceParams{}, err
	}
	poll := intOrDefault(input.DefaultPollIntervalMS, 1000)
	if poll <= 0 {
		return repository.CreateModbusSlaveDeviceParams{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "默认轮询周期必须大于 0")
	}
	return repository.CreateModbusSlaveDeviceParams{
		ProjectID:             projectID,
		ConnectionID:          connectionID,
		UnitID:                unitID,
		Name:                  name,
		Description:           normalizeOptionalText(input.Description),
		Enabled:               boolOrDefault(input.Enabled, true),
		DefaultPollIntervalMS: poll,
		DefaultByteOrder:      normalizeModbusByteOrder(input.DefaultByteOrder),
		DefaultWordOrder:      normalizeModbusWordOrder(input.DefaultWordOrder),
		RequestIntervalMS:     input.RequestIntervalMS,
		TimeoutMS:             input.TimeoutMS,
		RetryCount:            input.RetryCount,
		SortOrder:             input.SortOrder,
		UserID:                userID,
	}, nil
}

func (s *ModbusModelingService) normalizeUpdateSlaveDeviceInput(projectID, connectionID, slaveID, userID string, input UpdateModbusSlaveDeviceInput) (repository.UpdateModbusSlaveDeviceParams, error) {
	params, err := s.normalizeCreateSlaveDeviceInput(projectID, connectionID, userID, CreateModbusSlaveDeviceInput{
		UnitID:                input.UnitID,
		Name:                  input.Name,
		Description:           input.Description,
		Enabled:               input.Enabled,
		DefaultPollIntervalMS: input.DefaultPollIntervalMS,
		DefaultByteOrder:      input.DefaultByteOrder,
		DefaultWordOrder:      input.DefaultWordOrder,
		RequestIntervalMS:     input.RequestIntervalMS,
		TimeoutMS:             input.TimeoutMS,
		RetryCount:            input.RetryCount,
		SortOrder:             input.SortOrder,
	})
	if err != nil {
		return repository.UpdateModbusSlaveDeviceParams{}, err
	}
	return repository.UpdateModbusSlaveDeviceParams{
		ID:                    slaveID,
		ProjectID:             params.ProjectID,
		ConnectionID:          params.ConnectionID,
		UnitID:                params.UnitID,
		Name:                  params.Name,
		Description:           params.Description,
		Enabled:               params.Enabled,
		DefaultPollIntervalMS: params.DefaultPollIntervalMS,
		DefaultByteOrder:      params.DefaultByteOrder,
		DefaultWordOrder:      params.DefaultWordOrder,
		RequestIntervalMS:     params.RequestIntervalMS,
		TimeoutMS:             params.TimeoutMS,
		RetryCount:            params.RetryCount,
		SortOrder:             params.SortOrder,
		UserID:                userID,
	}, nil
}

func (s *ModbusModelingService) normalizeCreateRegisterInput(ctx context.Context, projectID, connectionID, userID string, input CreateModbusRegisterInput) (repository.CreateModbusRegisterParams, error) {
	defaultUnitID := s.defaultUnitID(ctx, projectID, connectionID)
	unitID := intOrDefault(input.UnitID, defaultUnitID)
	if err := s.ensureSlaveDevice(ctx, projectID, connectionID, userID, unitID); err != nil {
		return repository.CreateModbusRegisterParams{}, err
	}
	area := normalizeModbusArea(input.Area)
	addressBase := normalizeModbusAddressBase(input.AddressBase)
	address, protocolAddress, err := normalizeModbusAddresses(area, addressBase, input.Address, input.ProtocolAddress)
	if err != nil {
		return repository.CreateModbusRegisterParams{}, err
	}
	dataType, err := normalizeModbusRequiredText(input.DataType, "数据类型不能为空")
	if err != nil {
		return repository.CreateModbusRegisterParams{}, err
	}
	quantity := intOrDefault(input.Quantity, modbusQuantityForDataType(dataType))
	name, err := normalizeModbusRequiredText(input.Name, "变量名不能为空")
	if err != nil {
		return repository.CreateModbusRegisterParams{}, err
	}
	return repository.CreateModbusRegisterParams{
		ProjectID:       projectID,
		ConnectionID:    connectionID,
		GroupID:         normalizeOptionalText(input.GroupID),
		Name:            name,
		Code:            normalizeModbusRegisterCode(input.Code, name, area, address),
		UnitID:          unitID,
		Area:            area,
		Address:         address,
		AddressBase:     addressBase,
		ProtocolAddress: protocolAddress,
		Quantity:        quantity,
		DataType:        strings.TrimSpace(dataType),
		ByteOrder:       normalizeModbusByteOrder(input.ByteOrder),
		WordOrder:       normalizeModbusWordOrder(input.WordOrder),
		BitIndex:        input.BitIndex,
		Scale:           floatOrDefault(input.Scale, 1),
		Offset:          floatOrDefault(input.Offset, 0),
		Unit:            normalizeOptionalText(input.Unit),
		PollIntervalMS:  intOrDefault(input.PollIntervalMS, 1000),
		TimeoutMS:       input.TimeoutMS,
		RetryCount:      input.RetryCount,
		AccessLevel:     normalizeModbusAccessLevel(input.AccessLevel),
		Description:     normalizeOptionalText(input.Description),
		SortOrder:       input.SortOrder,
		UserID:          userID,
	}, nil
}

func (s *ModbusModelingService) normalizeUpdateRegisterInput(ctx context.Context, projectID, userID string, current repository.ModbusRegisterRecord, input UpdateModbusRegisterInput) (repository.UpdateModbusRegisterParams, error) {
	unitID := current.UnitID
	if input.UnitID != nil {
		unitID = *input.UnitID
	}
	if err := s.ensureSlaveDevice(ctx, projectID, current.ConnectionID, userID, unitID); err != nil {
		return repository.UpdateModbusRegisterParams{}, err
	}
	area := firstNonEmpty(input.Area, current.Area)
	addressBase := firstNonEmpty(input.AddressBase, current.AddressBase)
	address := input.Address
	if address == nil {
		address = &current.Address
	}
	protocolAddress := input.ProtocolAddress
	if protocolAddress == nil {
		protocolAddress = &current.ProtocolAddress
	}
	normalizedArea := normalizeModbusArea(area)
	normalizedBase := normalizeModbusAddressBase(addressBase)
	userAddress, normalizedProtocolAddress, err := normalizeModbusAddresses(normalizedArea, normalizedBase, address, protocolAddress)
	if err != nil {
		return repository.UpdateModbusRegisterParams{}, err
	}
	dataType := firstNonEmpty(input.DataType, current.DataType)
	quantity := current.Quantity
	if input.Quantity != nil {
		quantity = *input.Quantity
	}
	if quantity <= 0 {
		quantity = modbusQuantityForDataType(dataType)
	}
	status := firstNonEmpty(input.Status, current.Status)
	if _, ok := allowedDataPointStatuses[status]; !ok {
		return repository.UpdateModbusRegisterParams{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "变量状态不合法")
	}
	name := firstNonEmpty(input.Name, current.Name)
	return repository.UpdateModbusRegisterParams{
		ID:              current.ID,
		ProjectID:       projectID,
		ConnectionID:    current.ConnectionID,
		GroupID:         normalizeOptionalText(input.GroupID),
		HasGroupID:      input.HasGroupID,
		Name:            strings.TrimSpace(name),
		Code:            normalizeModbusRegisterCode(firstNonEmpty(input.Code, current.Code), name, normalizedArea, userAddress),
		UnitID:          unitID,
		Area:            normalizedArea,
		Address:         userAddress,
		AddressBase:     normalizedBase,
		ProtocolAddress: normalizedProtocolAddress,
		Quantity:        quantity,
		DataType:        strings.TrimSpace(dataType),
		ByteOrder:       normalizeModbusByteOrder(firstNonEmpty(input.ByteOrder, current.ByteOrder)),
		WordOrder:       normalizeModbusWordOrder(firstNonEmpty(input.WordOrder, current.WordOrder)),
		BitIndex:        input.BitIndex,
		Scale:           floatOrDefault(input.Scale, current.Scale),
		Offset:          floatOrDefault(input.Offset, current.Offset),
		Unit:            optionalTextOrCurrent(input.Unit, current.Unit),
		PollIntervalMS:  intOrDefault(input.PollIntervalMS, current.PollIntervalMS),
		TimeoutMS:       optionalIntOrCurrent(input.TimeoutMS, current.TimeoutMS),
		RetryCount:      optionalIntOrCurrent(input.RetryCount, current.RetryCount),
		AccessLevel:     normalizeModbusAccessLevel(firstNonEmpty(input.AccessLevel, current.AccessLevel)),
		Description:     optionalTextOrCurrent(input.Description, current.Description),
		SortOrder:       input.SortOrder,
		Status:          status,
		UserID:          userID,
	}, nil
}

func (s *ModbusModelingService) syncRegisterDatapoint(ctx context.Context, register repository.ModbusRegisterRecord, userID string) error {
	connection, err := s.connections.GetByProjectAndID(ctx, register.ProjectID, register.ConnectionID)
	if err != nil {
		return err
	}
	basePath := "modbus." + normalizeDatapointSegment(connection.Name)
	groups, _ := s.repository.ListGroups(ctx, register.ProjectID, register.ConnectionID)
	groupPath := s.groupPathSegment(groups, register.GroupID)
	if groupPath != "" {
		basePath += "." + groupPath
	}
	basePath += "." + normalizeDatapointSegment(register.Code)
	path := s.allocateDataPointPath(ctx, register.ProjectID, basePath, register.ID, "modbus.register")
	sourceID := register.ID
	refreshInterval := register.PollIntervalMS
	_, err = s.datapoints.UpsertBySource(ctx, repository.CreateDataPointParams{
		ProjectID:   register.ProjectID,
		UserID:      stringPtr(userID),
		Path:        path,
		Name:        register.Name,
		Description: cloneOptionalString(register.Description),
		SourceType:  "modbus.register",
		SourceID:    &sourceID,
		SourceConfig: map[string]any{
			"connectionId":     register.ConnectionID,
			"unitId":           register.UnitID,
			"area":             register.Area,
			"address":          register.Address,
			"addressBase":      register.AddressBase,
			"protocolAddress":  register.ProtocolAddress,
			"quantity":         register.Quantity,
			"dataType":         register.DataType,
			"byteOrder":        register.ByteOrder,
			"wordOrder":        register.WordOrder,
			"bitIndex":         register.BitIndex,
			"scale":            register.Scale,
			"offset":           register.Offset,
			"pollIntervalMs":   register.PollIntervalMS,
			"timeoutMs":        register.TimeoutMS,
			"retryCount":       register.RetryCount,
			"accessLevel":      register.AccessLevel,
			"runtimeOwnerHint": "readPlan",
		},
		DataType:          normalizeModbusDataPointType(register.DataType),
		Unit:              cloneOptionalString(register.Unit),
		Tags:              []any{},
		RefreshMode:       "auto",
		RefreshIntervalMS: &refreshInterval,
		Status:            "active",
		DisplayOrder:      &register.SortOrder,
	})
	return err
}

func (s *ModbusModelingService) allocateDataPointPath(ctx context.Context, projectID, basePath, sourceID, sourceType string) string {
	candidate := strings.TrimSpace(basePath)
	if candidate == "" {
		candidate = "modbus.unnamed"
	}
	for index := 2; index < 100; index++ {
		record, err := s.datapoints.GetByProjectAndPath(ctx, projectID, candidate)
		if err != nil || record == nil {
			return candidate
		}
		if record.SourceType == sourceType && record.SourceID != nil && *record.SourceID == sourceID {
			return candidate
		}
		candidate = fmt.Sprintf("%s_%d", basePath, index)
	}
	return basePath + "_" + sourceID[:8]
}

func (s *ModbusModelingService) groupPathSegment(groups []repository.ModbusRegisterGroupRecord, groupID *string) string {
	if groupID == nil || *groupID == "" {
		return ""
	}
	byID := make(map[string]repository.ModbusRegisterGroupRecord, len(groups))
	for _, group := range groups {
		byID[group.ID] = group
	}
	segments := make([]string, 0)
	currentID := *groupID
	visited := map[string]struct{}{}
	for currentID != "" {
		if _, ok := visited[currentID]; ok {
			break
		}
		visited[currentID] = struct{}{}
		group, ok := byID[currentID]
		if !ok {
			break
		}
		segments = append([]string{normalizeDatapointSegment(group.Name)}, segments...)
		if group.ParentID == nil {
			break
		}
		currentID = *group.ParentID
	}
	return strings.Join(segments, ".")
}

func (s *ModbusModelingService) validateProjectConnection(ctx context.Context, projectID, connectionID string) error {
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	connection, err := s.connections.GetByProjectAndID(ctx, projectID, connectionID)
	if err != nil {
		return err
	}
	if connection.Type != "modbus" {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "接入源不是 Modbus 类型")
	}
	return nil
}

func (s *ModbusModelingService) validateProjectConnectionAndUser(ctx context.Context, projectID, connectionID, userID string) error {
	if err := validateUserID(userID); err != nil {
		return err
	}
	return s.validateProjectConnection(ctx, projectID, connectionID)
}

func (s *ModbusModelingService) defaultUnitID(ctx context.Context, projectID, connectionID string) int {
	connection, err := s.connections.GetByProjectAndID(ctx, projectID, connectionID)
	if err != nil {
		return 1
	}
	if parsed := parseModbusConfigInt(connection.Config["slaveId"]); parsed >= 0 {
		return parsed
	}
	return 1
}

func (s *ModbusModelingService) ensureSlaveDevice(ctx context.Context, projectID, connectionID, userID string, unitID int) error {
	if unitID < 0 || unitID > 247 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "从站地址必须在 0-247 范围内")
	}
	existing, err := s.repository.GetSlaveDeviceByUnitID(ctx, projectID, connectionID, unitID)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}
	_, err = s.repository.CreateSlaveDevice(ctx, repository.CreateModbusSlaveDeviceParams{
		ProjectID:             projectID,
		ConnectionID:          connectionID,
		UnitID:                unitID,
		Name:                  fmt.Sprintf("从站 %d", unitID),
		Enabled:               true,
		DefaultPollIntervalMS: 1000,
		DefaultByteOrder:      "ABCD",
		DefaultWordOrder:      "high_first",
		SortOrder:             unitID,
		UserID:                userID,
	})
	return err
}

// BuildModbusReadPlanEstimate 按运行态合并规则生成读取预估。
func BuildModbusReadPlanEstimate(registers []ModbusRegister) ModbusReadPlanEstimate {
	active := make([]ModbusRegister, 0, len(registers))
	unitSet := map[int]struct{}{}
	for _, register := range registers {
		if register.Status != "active" {
			continue
		}
		active = append(active, register)
		unitSet[register.UnitID] = struct{}{}
	}
	grouped := map[string][]ModbusRegister{}
	for _, register := range active {
		key := fmt.Sprintf("%d|%s|%d", register.UnitID, register.Area, register.PollIntervalMS)
		grouped[key] = append(grouped[key], register)
	}
	plans := make([]ModbusReadPlan, 0)
	for _, items := range grouped {
		sort.Slice(items, func(i, j int) bool {
			return items[i].ProtocolAddress < items[j].ProtocolAddress
		})
		maxQuantity := modbusMaxReadQuantity(items[0].Area)
		var current *ModbusReadPlan
		for _, item := range items {
			start := item.ProtocolAddress
			end := item.ProtocolAddress + item.Quantity - 1
			// 运行态可合并连续或小间隔地址，但必须遵守协议单次读取上限。
			if current == nil || start > current.ProtocolStart+maxQuantity-1 || start > current.EndAddress+2 {
				plan := newModbusReadPlan(item, start, end)
				current = &plan
				plans = append(plans, plan)
				continue
			}
			if end > current.EndAddress {
				current.EndAddress = end
				current.Quantity = current.EndAddress - current.ProtocolStart + 1
				current.DisplayRange = formatModbusReadRange(current.Area, current.StartAddress, protocolToUserAddress(current.Area, current.EndAddress))
			}
			current.RegisterIDs = append(current.RegisterIDs, item.ID)
			current.RegisterCount++
			plans[len(plans)-1] = *current
		}
	}
	sort.Slice(plans, func(i, j int) bool {
		if plans[i].UnitID != plans[j].UnitID {
			return plans[i].UnitID < plans[j].UnitID
		}
		if plans[i].Area != plans[j].Area {
			return plans[i].Area < plans[j].Area
		}
		return plans[i].ProtocolStart < plans[j].ProtocolStart
	})
	readsPerSecond := 0.0
	for index := range plans {
		plans[index].ID = fmt.Sprintf("plan-%03d", index+1)
		plans[index].ReadsPerSecond = 1000 / float64(plans[index].PollIntervalMS)
		plans[index].DisplayCycle = formatModbusCycle(plans[index].PollIntervalMS)
		plans[index].EstimatedPressure = estimateModbusPressure(plans[index].ReadsPerSecond)
		readsPerSecond += plans[index].ReadsPerSecond
	}
	diagnostics := []string{fmt.Sprintf("你建了 %d 个变量，运行态预计合并为 %d 次读取。", len(active), len(plans))}
	if readsPerSecond > 50 {
		diagnostics = append(diagnostics, "预计 reads/s 较高，建议调大轮询周期或整理地址连续性。")
	}
	return ModbusReadPlanEstimate{
		RegisterCount:  len(active),
		UnitCount:      len(unitSet),
		ReadCount:      len(plans),
		ReadsPerSecond: readsPerSecond,
		Plans:          plans,
		Diagnostics:    diagnostics,
	}
}

func newModbusReadPlan(item ModbusRegister, start int, end int) ModbusReadPlan {
	return ModbusReadPlan{
		ConnectionID:   item.ConnectionID,
		UnitID:         item.UnitID,
		Area:           item.Area,
		StartAddress:   item.Address,
		EndAddress:     protocolToUserAddress(item.Area, end),
		ProtocolStart:  start,
		Quantity:       end - start + 1,
		PollIntervalMS: item.PollIntervalMS,
		RegisterIDs:    []string{item.ID},
		RegisterCount:  1,
		DisplayRange:   formatModbusReadRange(item.Area, item.Address, protocolToUserAddress(item.Area, end)),
		DisplayArea:    formatModbusArea(item.Area),
	}
}

func normalizeModbusRequiredText(value string, message string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, message)
	}
	return trimmed, nil
}

var allowedModbusAreas = map[string]struct{}{
	"coil":             {},
	"discrete_input":   {},
	"input_register":   {},
	"holding_register": {},
}

func normalizeModbusArea(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "coil", "coils", "0x", "00001":
		return "coil"
	case "discrete_input", "discrete input", "input", "1x", "10001":
		return "discrete_input"
	case "input_register", "input register", "3x", "30001":
		return "input_register"
	default:
		return "holding_register"
	}
}

func normalizeModbusAddressBase(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "zero_based", "0", "0-based":
		return "zero_based"
	case "one_based", "1", "1-based":
		return "one_based"
	default:
		return "modicon"
	}
}

func normalizeModbusAddresses(area, addressBase string, address *int, protocolAddress *int) (int, int, error) {
	if address == nil && protocolAddress == nil {
		return 0, 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "地址不能为空")
	}
	if protocolAddress != nil && *protocolAddress < 0 {
		return 0, 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "协议地址不能小于 0")
	}
	if address != nil && *address < 0 {
		return 0, 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "用户地址不能小于 0")
	}
	if protocolAddress != nil && address == nil {
		return protocolToUserAddress(area, *protocolAddress), *protocolAddress, nil
	}
	userAddress := *address
	switch addressBase {
	case "zero_based":
		return userAddress, userAddress, nil
	case "one_based":
		return userAddress, userAddress - 1, nil
	default:
		base := modbusAreaAddressPrefix(area)
		protocol := userAddress - base - 1
		if protocol < 0 {
			return 0, 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "用户地址与 Modbus 数据区不匹配")
		}
		return userAddress, protocol, nil
	}
}

func modbusAreaAddressPrefix(area string) int {
	switch area {
	case "coil":
		return 0
	case "discrete_input":
		return 10000
	case "input_register":
		return 30000
	default:
		return 40000
	}
}

func protocolToUserAddress(area string, protocolAddress int) int {
	return modbusAreaAddressPrefix(area) + protocolAddress + 1
}

func modbusQuantityForDataType(dataType string) int {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "float64", "double", "int64", "uint64":
		return 4
	case "float32", "float", "int32", "uint32":
		return 2
	default:
		return 1
	}
}

func normalizeModbusByteOrder(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "BADC", "CDAB", "DCBA":
		return strings.ToUpper(strings.TrimSpace(value))
	default:
		return "ABCD"
	}
}

func normalizeModbusWordOrder(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low_first", "low", "little":
		return "low_first"
	default:
		return "high_first"
	}
}

func normalizeModbusAccessLevel(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "write":
		return "Write"
	case "readwrite", "read_write":
		return "ReadWrite"
	default:
		return "Read"
	}
}

func normalizeModbusRegisterCode(code, name, area string, address int) string {
	source := firstNonEmpty(code, name, fmt.Sprintf("%s_%d", area, address))
	source = strings.TrimSpace(strings.ToLower(source))
	normalized := modbusCodeSanitizer.ReplaceAllString(source, "_")
	normalized = strings.Trim(normalized, "_")
	if normalized == "" {
		return "register"
	}
	return normalized
}

func normalizeModbusDataPointType(dataType string) string {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "bool", "boolean":
		return "boolean"
	case "json", "object":
		return "object"
	case "string":
		return "string"
	default:
		return "number"
	}
}

func validateModbusDataType(register ModbusRegister) string {
	dataType := strings.ToLower(strings.TrimSpace(register.DataType))
	if (register.Area == "coil" || register.Area == "discrete_input") && dataType != "bool" && dataType != "boolean" {
		return "开关量数据区默认只支持 bool"
	}
	required := modbusQuantityForDataType(dataType)
	if register.Quantity < required {
		return fmt.Sprintf("%s 至少需要 %d 个寄存器", register.DataType, required)
	}
	return ""
}

func modbusMaxReadQuantity(area string) int {
	if area == "coil" || area == "discrete_input" {
		return 2000
	}
	return 125
}

func formatModbusReadRange(area string, start int, end int) string {
	if start == end {
		return fmt.Sprintf("%d", start)
	}
	return fmt.Sprintf("%d-%d", start, end)
}

func formatModbusArea(area string) string {
	switch area {
	case "coil":
		return "输出开关(0x)"
	case "discrete_input":
		return "输入开关(1x)"
	case "input_register":
		return "只读寄存器(3x)"
	default:
		return "读写寄存器(4x)"
	}
}

func formatModbusCycle(ms int) string {
	if ms%1000 == 0 {
		return fmt.Sprintf("%ds", ms/1000)
	}
	return fmt.Sprintf("%dms", ms)
}

func estimateModbusPressure(readsPerSecond float64) string {
	switch {
	case readsPerSecond >= 20:
		return "high"
	case readsPerSecond >= 5:
		return "medium"
	default:
		return "low"
	}
}

func intOrDefault(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func floatOrDefault(value *float64, fallback float64) float64 {
	if value == nil {
		return fallback
	}
	return *value
}

func optionalIntOrCurrent(next *int, current *int) *int {
	if next == nil {
		if current == nil {
			return nil
		}
		cloned := *current
		return &cloned
	}
	cloned := *next
	return &cloned
}

func boolOrDefault(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func normalizeOptionalValue(value string, normalize func(string) string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return normalize(value)
}

func parseModbusConfigInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return -1
	}
}

func modbusRegisterIssue(severity, code, registerID, registerName, message string) ModbusValidationIssue {
	return ModbusValidationIssue{Severity: severity, Code: code, RegisterID: &registerID, RegisterName: registerName, Message: message}
}

func (s *ModbusModelingService) resolveBulkRegisterIDs(ctx context.Context, projectID, connectionID string, selection ModbusRegisterBulkSelection) ([]string, error) {
	if selection.UseFilter {
		return s.repository.ListRegisterIDsByFilter(ctx, projectID, connectionID, toRepositoryModbusRegisterFilter(selection.Filter))
	}
	return uniqueStrings(selection.IDs), nil
}

func toRepositoryModbusRegisterFilter(filter ModbusRegisterListFilter) repository.ModbusRegisterListFilter {
	return repository.ModbusRegisterListFilter{
		GroupID:      normalizeOptionalText(filter.GroupID),
		Search:       strings.TrimSpace(filter.Search),
		QuickFilter:  strings.TrimSpace(filter.QuickFilter),
		UnitID:       filter.UnitID,
		Area:         strings.TrimSpace(filter.Area),
		AddressStart: filter.AddressStart,
		AddressEnd:   filter.AddressEnd,
		DataType:     strings.TrimSpace(filter.DataType),
		SlaveEnabled: filter.SlaveEnabled,
		SortBy:       strings.TrimSpace(filter.SortBy),
		SortOrder:    strings.TrimSpace(filter.SortOrder),
	}
}

func toModbusRegisterGroup(record repository.ModbusRegisterGroupRecord) ModbusRegisterGroup {
	return ModbusRegisterGroup{
		ID:           record.ID,
		ProjectID:    record.ProjectID,
		ConnectionID: record.ConnectionID,
		ParentID:     cloneOptionalString(record.ParentID),
		Name:         record.Name,
		Description:  cloneOptionalString(record.Description),
		SortOrder:    record.SortOrder,
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	}
}

func toModbusSlaveDevice(record repository.ModbusSlaveDeviceRecord) ModbusSlaveDevice {
	return ModbusSlaveDevice{
		ID:                    record.ID,
		ProjectID:             record.ProjectID,
		ConnectionID:          record.ConnectionID,
		UnitID:                record.UnitID,
		Name:                  record.Name,
		Description:           cloneOptionalString(record.Description),
		Enabled:               record.Enabled,
		DefaultPollIntervalMS: record.DefaultPollIntervalMS,
		DefaultByteOrder:      record.DefaultByteOrder,
		DefaultWordOrder:      record.DefaultWordOrder,
		RequestIntervalMS:     cloneOptionalInt(record.RequestIntervalMS),
		TimeoutMS:             cloneOptionalInt(record.TimeoutMS),
		RetryCount:            cloneOptionalInt(record.RetryCount),
		SortOrder:             record.SortOrder,
		CreatedAt:             record.CreatedAt,
		UpdatedAt:             record.UpdatedAt,
	}
}

func toModbusRegister(record repository.ModbusRegisterRecord) ModbusRegister {
	return ModbusRegister{
		ID:              record.ID,
		ProjectID:       record.ProjectID,
		ConnectionID:    record.ConnectionID,
		GroupID:         cloneOptionalString(record.GroupID),
		Name:            record.Name,
		Code:            record.Code,
		UnitID:          record.UnitID,
		Area:            record.Area,
		Address:         record.Address,
		AddressBase:     record.AddressBase,
		ProtocolAddress: record.ProtocolAddress,
		Quantity:        record.Quantity,
		DataType:        record.DataType,
		ByteOrder:       record.ByteOrder,
		WordOrder:       record.WordOrder,
		BitIndex:        cloneOptionalInt(record.BitIndex),
		Scale:           record.Scale,
		Offset:          record.Offset,
		Unit:            cloneOptionalString(record.Unit),
		PollIntervalMS:  record.PollIntervalMS,
		TimeoutMS:       cloneOptionalInt(record.TimeoutMS),
		RetryCount:      cloneOptionalInt(record.RetryCount),
		AccessLevel:     record.AccessLevel,
		Description:     cloneOptionalString(record.Description),
		SortOrder:       record.SortOrder,
		Status:          record.Status,
		DataPointID:     cloneOptionalString(record.DataPointID),
		DataPointPath:   cloneOptionalString(record.DataPointPath),
		DataPointStatus: cloneOptionalString(record.DataPointStatus),
		LastValue:       bytesToAny(record.LastValue),
		Quality:         firstNonEmpty(record.Quality, "unknown"),
		LastUpdatedAt:   cloneOptionalTime(record.LastUpdatedAt),
		CreatedAt:       record.CreatedAt,
		UpdatedAt:       record.UpdatedAt,
	}
}
