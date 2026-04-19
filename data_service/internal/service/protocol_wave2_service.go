package service

import (
	"context"
	"net/http"
	"strings"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

var allowedOpcuaSecurityModes = map[string]struct{}{
	"none":           {},
	"sign":           {},
	"signandencrypt": {},
}

var allowedOpcuaAuthTypes = map[string]struct{}{
	"anonymous":         {},
	"username_password": {},
}

var allowedModbusModes = map[string]struct{}{
	"tcp": {},
	"rtu": {},
}

// CreateOpcuaConfigInput 表示创建 OPC UA 配置输入。
type CreateOpcuaConfigInput struct {
	Name           string
	Status         string
	Endpoint       string
	SecurityPolicy string
	SecurityMode   string
	AuthType       string
	Username       *string
	Password       *string
	SamplingMS     *int
	Options        map[string]any
}

// CreateS7ConfigInput 表示创建 S7 配置输入。
type CreateS7ConfigInput struct {
	Name           string
	Status         string
	Host           string
	Port           *int
	Rack           *int
	Slot           *int
	PollIntervalMS *int
	Options        map[string]any
}

// CreateModbusConfigInput 表示创建 Modbus 配置输入。
type CreateModbusConfigInput struct {
	Name           string
	Status         string
	Mode           string
	Host           *string
	Port           *int
	SerialConfig   map[string]any
	SlaveID        *int
	StartAddress   *int
	Quantity       *int
	PollIntervalMS *int
	Options        map[string]any
}

// CreateTdengineConfigInput 表示创建 TDengine 配置输入。
type CreateTdengineConfigInput struct {
	Name     string
	Status   string
	DSN      string
	Database string
	Timezone *string
	Options  map[string]any
}

// OpcdaContractValidateInput 表示 OPC DA 合约校验输入。
type OpcdaContractValidateInput struct {
	ItemPath   string
	SamplingMS int
}

// OpcdaContractValidateResult 表示 OPC DA 合约校验结果。
type OpcdaContractValidateResult struct {
	Valid bool `json:"valid"`
}

// ProtocolWave2Service 负责第二波协议和 OPC DA 合约校验逻辑。
type ProtocolWave2Service struct {
	repository *repository.ProtocolWave2Repository
}

// NewProtocolWave2Service 创建第二波协议服务。
func NewProtocolWave2Service(repo *repository.ProtocolWave2Repository) *ProtocolWave2Service {
	return &ProtocolWave2Service{repository: repo}
}

// CreateOpcuaConfig 创建 OPC UA 配置。
func (s *ProtocolWave2Service) CreateOpcuaConfig(ctx context.Context, projectID, userID string, input CreateOpcuaConfigInput) (*ProtocolConnection, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}

	name, err := normalizeConnectionName(input.Name)
	if err != nil {
		return nil, err
	}
	status, err := normalizeProtocolStatus(input.Status)
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimSpace(input.Endpoint)
	if endpoint == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "endpoint 不能为空")
	}
	securityPolicy := strings.TrimSpace(input.SecurityPolicy)
	if securityPolicy == "" {
		securityPolicy = "None"
	}
	securityMode := strings.TrimSpace(strings.ToLower(input.SecurityMode))
	if securityMode == "" {
		securityMode = "none"
	}
	if _, ok := allowedOpcuaSecurityModes[securityMode]; !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "securityMode 不受支持")
	}
	authType := strings.TrimSpace(strings.ToLower(input.AuthType))
	if authType == "" {
		authType = "anonymous"
	}
	if _, ok := allowedOpcuaAuthTypes[authType]; !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "authType 不受支持")
	}

	samplingMS := 1000
	if input.SamplingMS != nil {
		if *input.SamplingMS <= 0 {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "samplingMs 必须大于 0")
		}
		samplingMS = *input.SamplingMS
	}

	record, err := s.repository.CreateOpcuaConfig(ctx, repository.CreateOpcuaConfigParams{
		ProjectID:      projectID,
		UserID:         userID,
		Name:           name,
		Status:         status,
		Endpoint:       endpoint,
		SecurityPolicy: securityPolicy,
		SecurityMode:   normalizeOpcuaSecurityMode(securityMode),
		AuthType:       authType,
		Username:       normalizeOptionalText(input.Username),
		Password:       normalizeOptionalText(input.Password),
		SamplingMS:     samplingMS,
		Options:        cloneMap(input.Options),
	})
	if err != nil {
		return nil, err
	}

	connection := toProtocolConnection(*record)
	return &connection, nil
}

// CreateS7Config 创建 S7 配置。
func (s *ProtocolWave2Service) CreateS7Config(ctx context.Context, projectID, userID string, input CreateS7ConfigInput) (*ProtocolConnection, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}

	name, err := normalizeConnectionName(input.Name)
	if err != nil {
		return nil, err
	}
	status, err := normalizeProtocolStatus(input.Status)
	if err != nil {
		return nil, err
	}
	host := strings.TrimSpace(input.Host)
	if host == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "host 不能为空")
	}
	port := 102
	if input.Port != nil {
		if *input.Port <= 0 || *input.Port > 65535 {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "port 范围必须在 1 到 65535 之间")
		}
		port = *input.Port
	}
	rack := 0
	if input.Rack != nil {
		if *input.Rack < 0 {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "rack 不能小于 0")
		}
		rack = *input.Rack
	}
	slot := 1
	if input.Slot != nil {
		if *input.Slot < 0 {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "slot 不能小于 0")
		}
		slot = *input.Slot
	}
	pollInterval := 1000
	if input.PollIntervalMS != nil {
		if *input.PollIntervalMS <= 0 {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "pollIntervalMs 必须大于 0")
		}
		pollInterval = *input.PollIntervalMS
	}

	record, err := s.repository.CreateS7Config(ctx, repository.CreateS7ConfigParams{
		ProjectID:      projectID,
		UserID:         userID,
		Name:           name,
		Status:         status,
		Host:           host,
		Port:           port,
		Rack:           rack,
		Slot:           slot,
		PollIntervalMS: pollInterval,
		Options:        cloneMap(input.Options),
	})
	if err != nil {
		return nil, err
	}

	connection := toProtocolConnection(*record)
	return &connection, nil
}

// CreateModbusConfig 创建 Modbus 配置。
func (s *ProtocolWave2Service) CreateModbusConfig(ctx context.Context, projectID, userID string, input CreateModbusConfigInput) (*ProtocolConnection, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}

	name, err := normalizeConnectionName(input.Name)
	if err != nil {
		return nil, err
	}
	status, err := normalizeProtocolStatus(input.Status)
	if err != nil {
		return nil, err
	}
	mode := strings.TrimSpace(strings.ToLower(input.Mode))
	if mode == "" {
		mode = "tcp"
	}
	if _, ok := allowedModbusModes[mode]; !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "mode 仅支持 tcp/rtu")
	}

	host := normalizeOptionalText(input.Host)
	var port *int
	if input.Port != nil {
		if *input.Port <= 0 || *input.Port > 65535 {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "port 范围必须在 1 到 65535 之间")
		}
		value := *input.Port
		port = &value
	}
	if mode == "tcp" {
		if host == nil {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "tcp 模式要求提供 host")
		}
		if port == nil {
			defaultPort := 502
			port = &defaultPort
		}
	}

	slaveID := 1
	if input.SlaveID != nil {
		if *input.SlaveID < 0 {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "slaveId 不能小于 0")
		}
		slaveID = *input.SlaveID
	}
	startAddress := 0
	if input.StartAddress != nil {
		if *input.StartAddress < 0 {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "startAddress 不能小于 0")
		}
		startAddress = *input.StartAddress
	}
	quantity := 1
	if input.Quantity != nil {
		if *input.Quantity <= 0 {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "quantity 必须大于 0")
		}
		quantity = *input.Quantity
	}
	pollInterval := 1000
	if input.PollIntervalMS != nil {
		if *input.PollIntervalMS <= 0 {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "pollIntervalMs 必须大于 0")
		}
		pollInterval = *input.PollIntervalMS
	}

	record, err := s.repository.CreateModbusConfig(ctx, repository.CreateModbusConfigParams{
		ProjectID:      projectID,
		UserID:         userID,
		Name:           name,
		Status:         status,
		Mode:           mode,
		Host:           host,
		Port:           port,
		SerialConfig:   cloneMap(input.SerialConfig),
		HasSerial:      input.SerialConfig != nil,
		SlaveID:        slaveID,
		StartAddress:   startAddress,
		Quantity:       quantity,
		PollIntervalMS: pollInterval,
		Options:        cloneMap(input.Options),
	})
	if err != nil {
		return nil, err
	}

	connection := toProtocolConnection(*record)
	return &connection, nil
}

// CreateTdengineConfig 创建 TDengine 配置。
func (s *ProtocolWave2Service) CreateTdengineConfig(ctx context.Context, projectID, userID string, input CreateTdengineConfigInput) (*ProtocolConnection, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}

	name, err := normalizeConnectionName(input.Name)
	if err != nil {
		return nil, err
	}
	status, err := normalizeProtocolStatus(input.Status)
	if err != nil {
		return nil, err
	}
	dsn := strings.TrimSpace(input.DSN)
	if dsn == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "dsn 不能为空")
	}
	database := strings.TrimSpace(input.Database)
	if database == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "database 不能为空")
	}

	record, err := s.repository.CreateTdengineConfig(ctx, repository.CreateTdengineConfigParams{
		ProjectID:    projectID,
		UserID:       userID,
		Name:         name,
		Status:       status,
		DSN:          dsn,
		DatabaseName: database,
		Timezone:     normalizeOptionalText(input.Timezone),
		Options:      cloneMap(input.Options),
	})
	if err != nil {
		return nil, err
	}

	connection := toProtocolConnection(*record)
	return &connection, nil
}

// ValidateOpcdaContract 校验 OPC DA 发布契约字段。
func (s *ProtocolWave2Service) ValidateOpcdaContract(input OpcdaContractValidateInput) (*OpcdaContractValidateResult, error) {
	itemPath := strings.TrimSpace(input.ItemPath)
	if itemPath == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "itemPath 不能为空")
	}
	if input.SamplingMS <= 0 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "samplingMs 必须大于 0")
	}
	if input.SamplingMS > 600000 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "samplingMs 不能超过 600000")
	}

	return &OpcdaContractValidateResult{
		Valid: true,
	}, nil
}

func normalizeOpcuaSecurityMode(mode string) string {
	switch mode {
	case "sign":
		return "Sign"
	case "signandencrypt":
		return "SignAndEncrypt"
	default:
		return "None"
	}
}
