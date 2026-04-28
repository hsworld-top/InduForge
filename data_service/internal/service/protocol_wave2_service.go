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

const phase2ProtocolBoundaryMessage = "已进入工业协议配置阶段，请使用对应的工业协议专用配置接口"

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

// ProtocolWave2Service 负责第二波工业协议配置和 OPC DA 合约校验逻辑。
// 当前阶段只交付平台侧配置与 artifact 契约，不启动节点侧长期采集，也不在 data_service 内常驻协议会话。
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
	securityMode, err := normalizeOpcuaSecurityMode(input.SecurityMode)
	if err != nil {
		return nil, err
	}
	authType, err := normalizeOpcuaAuthType(input.AuthType)
	if err != nil {
		return nil, err
	}
	username := normalizeOptionalText(input.Username)
	password := normalizeOptionalText(input.Password)
	if authType == "username_password" && username == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "OPC UA username_password 认证需要 username")
	}
	samplingMS, err := normalizePositiveInt(input.SamplingMS, 1000, "samplingMs 必须大于 0")
	if err != nil {
		return nil, err
	}

	record, err := s.repository.CreateOpcuaConfig(ctx, repository.CreateOpcuaConfigParams{
		ProjectID:      projectID,
		UserID:         userID,
		Name:           name,
		Status:         status,
		Endpoint:       endpoint,
		SecurityPolicy: securityPolicy,
		SecurityMode:   opcuaSecurityModeForStorage(securityMode),
		AuthType:       authType,
		Username:       username,
		Password:       password,
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
	port, err := normalizePort(input.Port, 102, "S7 port 范围必须在 1 到 65535 之间")
	if err != nil {
		return nil, err
	}
	rack, err := normalizeNonNegativeInt(input.Rack, 0, "rack 不能小于 0")
	if err != nil {
		return nil, err
	}
	slot, err := normalizeNonNegativeInt(input.Slot, 1, "slot 不能小于 0")
	if err != nil {
		return nil, err
	}
	pollIntervalMS, err := normalizePositiveInt(input.PollIntervalMS, 1000, "pollIntervalMs 必须大于 0")
	if err != nil {
		return nil, err
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
		PollIntervalMS: pollIntervalMS,
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
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "modbus mode 仅支持 tcp/rtu")
	}
	host := normalizeOptionalText(input.Host)
	defaultModbusPort := 0
	if mode == "tcp" {
		defaultModbusPort = 502
	}
	port, err := normalizeOptionalPort(input.Port, defaultModbusPort, "Modbus port 范围必须在 1 到 65535 之间")
	if err != nil {
		return nil, err
	}
	if mode == "tcp" {
		if host == nil {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Modbus TCP 需要 host")
		}
		if port == nil {
			defaultPort := 502
			port = &defaultPort
		}
	}
	hasSerial := input.SerialConfig != nil
	if mode == "rtu" && !hasSerial {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Modbus RTU 需要 serialConfig")
	}
	slaveID, err := normalizeNonNegativeInt(input.SlaveID, 1, "slaveId 不能小于 0")
	if err != nil {
		return nil, err
	}
	startAddress, err := normalizeNonNegativeInt(input.StartAddress, 0, "startAddress 不能小于 0")
	if err != nil {
		return nil, err
	}
	quantity, err := normalizePositiveInt(input.Quantity, 1, "quantity 必须大于 0")
	if err != nil {
		return nil, err
	}
	pollIntervalMS, err := normalizePositiveInt(input.PollIntervalMS, 1000, "pollIntervalMs 必须大于 0")
	if err != nil {
		return nil, err
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
		HasSerial:      hasSerial,
		SlaveID:        slaveID,
		StartAddress:   startAddress,
		Quantity:       quantity,
		PollIntervalMS: pollIntervalMS,
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
	return &OpcdaContractValidateResult{Valid: true}, nil
}

func normalizeOpcuaSecurityMode(value string) (string, error) {
	normalized := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "_", "")
	if normalized == "" || normalized == "none" {
		return "none", nil
	}
	if _, ok := allowedOpcuaSecurityModes[normalized]; !ok {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "securityMode 仅支持 none/sign/signandencrypt")
	}
	return normalized, nil
}

func opcuaSecurityModeForStorage(value string) string {
	switch value {
	case "sign":
		return "Sign"
	case "signandencrypt":
		return "SignAndEncrypt"
	default:
		return "None"
	}
}

func normalizeOpcuaAuthType(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		normalized = "anonymous"
	}
	if _, ok := allowedOpcuaAuthTypes[normalized]; !ok {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "authType 仅支持 anonymous/username_password")
	}
	return normalized, nil
}

func normalizePort(value *int, defaultValue int, errorMessage string) (int, error) {
	if value == nil {
		return defaultValue, nil
	}
	if *value <= 0 || *value > 65535 {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, errorMessage)
	}
	return *value, nil
}

func normalizeOptionalPort(value *int, defaultValue int, errorMessage string) (*int, error) {
	if value == nil {
		if defaultValue <= 0 {
			return nil, nil
		}
		return &defaultValue, nil
	}
	if *value <= 0 || *value > 65535 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, errorMessage)
	}
	return value, nil
}

func normalizePositiveInt(value *int, defaultValue int, errorMessage string) (int, error) {
	if value == nil {
		return defaultValue, nil
	}
	if *value <= 0 {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, errorMessage)
	}
	return *value, nil
}

func newPhaseBoundaryProtocolError(protocolName string) error {
	return apperrors.NewAppError(
		apperrors.ErrorCodeBadRequest,
		http.StatusBadRequest,
		protocolName+" "+phase2ProtocolBoundaryMessage,
	)
}
