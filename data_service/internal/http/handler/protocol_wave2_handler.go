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

	var request opcuaConfigRequest
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
		Redundancy:     request.Redundancy,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), connection)
	return nil
}

// UpdateOpcuaConfig 更新 OPC UA 配置。
func (h *ProtocolWave2Handler) UpdateOpcuaConfig(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request opcuaConfigRequest
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	connection, err := h.service.UpdateOpcuaConfig(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, service.UpdateOpcuaConfigInput{
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
		Redundancy:     request.Redundancy,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), connection)
	return nil
}

type opcuaConfigRequest struct {
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
	Redundancy     map[string]any `json:"redundancy"`
}

// CreateS7Config 创建 S7 配置。
func (h *ProtocolWave2Handler) CreateS7Config(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request s7ConfigRequest
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	connection, err := h.service.CreateS7Config(r.Context(), r.PathValue("projectId"), claims.UserID, request.toInput())
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), connection)
	return nil
}

// UpdateS7Config 更新 S7 配置。
func (h *ProtocolWave2Handler) UpdateS7Config(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request s7ConfigRequest
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	connection, err := h.service.UpdateS7Config(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, request.toInput())
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), connection)
	return nil
}

type s7ConfigRequest struct {
	Name              string         `json:"name"`
	Status            string         `json:"status"`
	Host              string         `json:"host"`
	Port              *int           `json:"port"`
	Rack              *int           `json:"rack"`
	Slot              *int           `json:"slot"`
	PollIntervalMS    *int           `json:"pollIntervalMs"`
	PlcFamily         string         `json:"plcFamily"`
	CommunicationMode string         `json:"communicationMode"`
	Options           map[string]any `json:"options"`
	Redundancy        map[string]any `json:"redundancy"`
}

func (r s7ConfigRequest) toInput() service.CreateS7ConfigInput {
	return service.CreateS7ConfigInput{
		Name:              r.Name,
		Status:            r.Status,
		Host:              r.Host,
		Port:              r.Port,
		Rack:              r.Rack,
		Slot:              r.Slot,
		PollIntervalMS:    r.PollIntervalMS,
		PlcFamily:         r.PlcFamily,
		CommunicationMode: r.CommunicationMode,
		Options:           r.Options,
		Redundancy:        r.Redundancy,
	}
}

// CreateModbusConfig 创建 Modbus 配置。
func (h *ProtocolWave2Handler) CreateModbusConfig(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request modbusConfigRequest
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	connection, err := h.service.CreateModbusConfig(r.Context(), r.PathValue("projectId"), claims.UserID, request.toInput())
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), connection)
	return nil
}

// UpdateModbusConfig 更新 Modbus 配置。
func (h *ProtocolWave2Handler) UpdateModbusConfig(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request modbusConfigRequest
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	connection, err := h.service.UpdateModbusConfig(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, request.toInput())
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), connection)
	return nil
}

type modbusConfigRequest struct {
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
	Redundancy     map[string]any `json:"redundancy"`
}

func (r modbusConfigRequest) toInput() service.CreateModbusConfigInput {
	return service.CreateModbusConfigInput{
		Name:           r.Name,
		Status:         r.Status,
		Mode:           r.Mode,
		Host:           r.Host,
		Port:           r.Port,
		SerialConfig:   r.SerialConfig,
		SlaveID:        r.SlaveID,
		StartAddress:   r.StartAddress,
		Quantity:       r.Quantity,
		PollIntervalMS: r.PollIntervalMS,
		Options:        r.Options,
		Redundancy:     r.Redundancy,
	}
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
