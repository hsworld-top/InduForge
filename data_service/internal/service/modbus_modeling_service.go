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
			AccessLevel:    "Read",
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
	issues := make([]ModbusValidationIssue, 0)
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
		if _, ok := allowedModbusAreas[register.Area]; !ok {
			issues = append(issues, modbusRegisterIssue("error", "area.invalid", registerID, register.Name, "寄存器区域不合法"))
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
func (s *ModbusModelingService) EstimateReadPlans(ctx context.Context, projectID, connectionID string, groupID *string) (*ModbusReadPlanEstimate, error) {
	registers, err := s.ListRegisters(ctx, projectID, connectionID, groupID)
	if err != nil {
		return nil, err
	}
	estimate := BuildModbusReadPlanEstimate(registers)
	return &estimate, nil
}

func (s *ModbusModelingService) normalizeCreateRegisterInput(ctx context.Context, projectID, connectionID, userID string, input CreateModbusRegisterInput) (repository.CreateModbusRegisterParams, error) {
	defaultUnitID := s.defaultUnitID(ctx, projectID, connectionID)
	unitID := intOrDefault(input.UnitID, defaultUnitID)
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
		RefreshMode:       "polling",
		RefreshIntervalMS: &refreshInterval,
		Status:            "active",
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
	"coil":           {},
	"discrete_input": {},
	"input_register": {},
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
			return 0, 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "用户地址与寄存器区域不匹配")
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
		return "Coil / Discrete Input 默认只支持 bool"
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
		return "Coil"
	case "discrete_input":
		return "Discrete Input"
	case "input_register":
		return "Input Register"
	default:
		return "Holding Register"
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
		Quality:         "unknown",
		CreatedAt:       record.CreatedAt,
		UpdatedAt:       record.UpdatedAt,
	}
}
