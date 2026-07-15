package handler

import (
	"net/http"

	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// ProtocolWave2Handler 负责第二波协议和 OPC DA 合约接口。
type ProtocolWave2Handler struct {
	service *service.ProtocolWave2Service
}

// NewProtocolWave2Handler 创建第二波协议处理器。
func NewProtocolWave2Handler(protocolService *service.ProtocolWave2Service) *ProtocolWave2Handler {
	return &ProtocolWave2Handler{service: protocolService}
}

// CreateOpcuaConfig 创建 OPC UA 配置。
// UpdateOpcuaConfig 更新 OPC UA 配置。
// CreateS7Config 创建 S7 配置。
// UpdateS7Config 更新 S7 配置。
// CreateModbusConfig 创建 Modbus 配置。
// UpdateModbusConfig 更新 Modbus 配置。
// CreateTdengineConfig 创建 TDengine 配置。
func (h *ProtocolWave2Handler) CreateTdengineConfig(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Name     string         `json:"name"`
		Status   string         `json:"status"`
		DSN      string         `json:"dsn"`
		Database string         `json:"database"`
		Timezone *string        `json:"timezone"`
		Options  map[string]any `json:"options"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	connection, err := h.service.CreateTdengineConfig(r.Context(), r.PathValue("projectId"), claims.UserID, service.CreateTdengineConfigInput{
		Name:     request.Name,
		Status:   request.Status,
		DSN:      request.DSN,
		Database: request.Database,
		Timezone: request.Timezone,
		Options:  request.Options,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), connection)
	return nil
}

// ValidateOpcdaContract 校验 OPC DA 合约字段。
func (h *ProtocolWave2Handler) ValidateOpcdaContract(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	var request struct {
		ItemPath   string `json:"itemPath"`
		SamplingMS int    `json:"samplingMs"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	result, err := h.service.ValidateOpcdaContract(service.OpcdaContractValidateInput{
		ItemPath:   request.ItemPath,
		SamplingMS: request.SamplingMS,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}
