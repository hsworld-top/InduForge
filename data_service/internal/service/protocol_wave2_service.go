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
// UpdateOpcuaConfigInput 表示更新 OPC UA 配置输入。

// CreateS7ConfigInput 表示创建 S7 配置输入。
// UpdateS7ConfigInput 表示更新 S7 配置输入。

// CreateModbusConfigInput 表示创建 Modbus 配置输入。
// UpdateModbusConfigInput 表示更新 Modbus 配置输入。

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
// UpdateOpcuaConfig 更新 OPC UA 配置。
// CreateS7Config 创建 S7 配置。
// UpdateS7Config 更新 S7 配置。
// CreateModbusConfig 创建 Modbus 配置。
// UpdateModbusConfig 更新 Modbus 配置。
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

func wave2TextFromMap(input map[string]any, key string, fallback string) string {
	if value, ok := input[key]; ok {
		if text := strings.TrimSpace(toWave2String(value)); text != "" {
			return text
		}
	}
	return fallback
}

func wave2OptionalStringFromMap(input map[string]any, key string) *string {
	text := wave2TextFromMap(input, key, "")
	if text == "" {
		return nil
	}
	return &text
}

func wave2PositiveIntFromMap(input map[string]any, key string, fallback int) int {
	value := wave2IntFromAny(input[key], fallback)
	if value <= 0 {
		return fallback
	}
	return value
}

func wave2NonNegativeIntFromMap(input map[string]any, key string, fallback int) int {
	value := wave2IntFromAny(input[key], fallback)
	if value < 0 {
		return fallback
	}
	return value
}

func wave2OptionalPositiveIntFromMap(input map[string]any, key string) *int {
	value := wave2IntFromAny(input[key], 0)
	if value <= 0 {
		return nil
	}
	return &value
}

func wave2BoolFromMap(input map[string]any, key string, fallback bool) bool {
	value, ok := input[key]
	if !ok {
		return fallback
	}
	typed, ok := value.(bool)
	if !ok {
		return fallback
	}
	return typed
}

func wave2AnySliceFromMap(input map[string]any, key string, fallback []any) []any {
	raw, ok := input[key]
	if !ok {
		return fallback
	}
	values, ok := raw.([]any)
	if !ok || len(values) == 0 {
		return fallback
	}
	result := make([]any, 0, len(values))
	for _, value := range values {
		if text := strings.TrimSpace(toWave2String(value)); text != "" {
			result = append(result, text)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}

func wave2IntFromAny(value any, fallback int) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case float32:
		return int(typed)
	default:
		return fallback
	}
}

func toWave2String(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
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
