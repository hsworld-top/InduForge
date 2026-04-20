package service

import (
	"context"
	"net/http"

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

const phase2ProtocolBoundaryMessage = "未纳入 data_service Phase 1 正式范围，当前仅保留 Phase 2 规划占位"

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
// 说明：`opcua/s7/modbus/tdengine/opcda` 属于 Phase 2 预留范围，Phase 1 只保留接口边界，
// 不再继续把“可创建配置对象”误导成“正式可交付协议能力”。
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
	return nil, newPhaseBoundaryProtocolError("OPC UA")
}

// CreateS7Config 创建 S7 配置。
func (s *ProtocolWave2Service) CreateS7Config(ctx context.Context, projectID, userID string, input CreateS7ConfigInput) (*ProtocolConnection, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	return nil, newPhaseBoundaryProtocolError("S7")
}

// CreateModbusConfig 创建 Modbus 配置。
func (s *ProtocolWave2Service) CreateModbusConfig(ctx context.Context, projectID, userID string, input CreateModbusConfigInput) (*ProtocolConnection, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	return nil, newPhaseBoundaryProtocolError("Modbus")
}

// CreateTdengineConfig 创建 TDengine 配置。
func (s *ProtocolWave2Service) CreateTdengineConfig(ctx context.Context, projectID, userID string, input CreateTdengineConfigInput) (*ProtocolConnection, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	return nil, newPhaseBoundaryProtocolError("TDengine")
}

// ValidateOpcdaContract 校验 OPC DA 发布契约字段。
func (s *ProtocolWave2Service) ValidateOpcdaContract(input OpcdaContractValidateInput) (*OpcdaContractValidateResult, error) {
	return nil, newPhaseBoundaryProtocolError("OPC DA")
}

func newPhaseBoundaryProtocolError(protocolName string) error {
	return apperrors.NewAppError(
		apperrors.ErrorCodeBadRequest,
		http.StatusBadRequest,
		protocolName+" "+phase2ProtocolBoundaryMessage,
	)
}
