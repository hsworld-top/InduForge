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
func (h *ProtocolWave2Handler) CreateOpcuaConfig(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Name           string         `json:"name"`
		Status         string         `json:"status"`
		Endpoint       string         `json:"endpoint"`
		SecurityPolicy string         `json:"securityPolicy"`
		SecurityMode   string         `json:"securityMode"`
		AuthType       string         `json:"authType"`
		Username       *string        `json:"username"`
		Password       *string        `json:"password"`
		SamplingMS     *int           `json:"samplingMs"`
		Options        map[string]any `json:"options"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	connection, err := h.service.CreateOpcuaConfig(r.Context(), r.PathValue("projectId"), claims.UserID, service.CreateOpcuaConfigInput{
		Name:           request.Name,
		Status:         request.Status,
		Endpoint:       request.Endpoint,
		SecurityPolicy: request.SecurityPolicy,
		SecurityMode:   request.SecurityMode,
		AuthType:       request.AuthType,
		Username:       request.Username,
		Password:       request.Password,
		SamplingMS:     request.SamplingMS,
		Options:        request.Options,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), connection)
	return nil
}

// CreateS7Config 创建 S7 配置。
func (h *ProtocolWave2Handler) CreateS7Config(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Name           string         `json:"name"`
		Status         string         `json:"status"`
		Host           string         `json:"host"`
		Port           *int           `json:"port"`
		Rack           *int           `json:"rack"`
		Slot           *int           `json:"slot"`
		PollIntervalMS *int           `json:"pollIntervalMs"`
		Options        map[string]any `json:"options"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	connection, err := h.service.CreateS7Config(r.Context(), r.PathValue("projectId"), claims.UserID, service.CreateS7ConfigInput{
		Name:           request.Name,
		Status:         request.Status,
		Host:           request.Host,
		Port:           request.Port,
		Rack:           request.Rack,
		Slot:           request.Slot,
		PollIntervalMS: request.PollIntervalMS,
		Options:        request.Options,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), connection)
	return nil
}

// CreateModbusConfig 创建 Modbus 配置。
func (h *ProtocolWave2Handler) CreateModbusConfig(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Name           string         `json:"name"`
		Status         string         `json:"status"`
		Mode           string         `json:"mode"`
		Host           *string        `json:"host"`
		Port           *int           `json:"port"`
		SerialConfig   map[string]any `json:"serialConfig"`
		SlaveID        *int           `json:"slaveId"`
		StartAddress   *int           `json:"startAddress"`
		Quantity       *int           `json:"quantity"`
		PollIntervalMS *int           `json:"pollIntervalMs"`
		Options        map[string]any `json:"options"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	connection, err := h.service.CreateModbusConfig(r.Context(), r.PathValue("projectId"), claims.UserID, service.CreateModbusConfigInput{
		Name:           request.Name,
		Status:         request.Status,
		Mode:           request.Mode,
		Host:           request.Host,
		Port:           request.Port,
		SerialConfig:   request.SerialConfig,
		SlaveID:        request.SlaveID,
		StartAddress:   request.StartAddress,
		Quantity:       request.Quantity,
		PollIntervalMS: request.PollIntervalMS,
		Options:        request.Options,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), connection)
	return nil
}

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
