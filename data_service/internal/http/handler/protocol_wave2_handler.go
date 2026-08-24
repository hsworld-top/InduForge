package handler

import (
	"net/http"

	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// ProtocolWave2Handler 负责 TDengine 配置和 OPC DA 合约接口。
type ProtocolWave2Handler struct {
	service *service.ProtocolWave2Service
}

// NewProtocolWave2Handler 创建第二波协议处理器。
func NewProtocolWave2Handler(protocolService *service.ProtocolWave2Service) *ProtocolWave2Handler {
	return &ProtocolWave2Handler{service: protocolService}
}

// CreateTdengineConfig 创建 TDengine 配置。
func (h *ProtocolWave2Handler) CreateTdengineConfig(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Name            string            `json:"name"`
		Status          string            `json:"status"`
		Protocol        string            `json:"protocol"`
		Host            string            `json:"host"`
		Port            *int              `json:"port"`
		Username        string            `json:"username"`
		Database        string            `json:"databaseName"`
		Timezone        *string           `json:"timezone"`
		TLSSkipVerify   bool              `json:"tlsSkipVerify"`
		Options         map[string]any    `json:"options"`
		Secrets         map[string]string `json:"secrets"`
		ClearSecretKeys []string          `json:"clearSecretKeys"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	connection, err := h.service.CreateTdengineConfig(r.Context(), r.PathValue("projectId"), claims.UserID, service.CreateTdengineConfigInput{
		Name:     request.Name,
		Status:   request.Status,
		Protocol: request.Protocol, Host: request.Host, Port: request.Port, Username: request.Username,
		Database:      request.Database,
		Timezone:      request.Timezone,
		TLSSkipVerify: request.TLSSkipVerify,
		Options:       request.Options,
		Secrets:       request.Secrets, ClearSecretKeys: request.ClearSecretKeys,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), connection)
	return nil
}

func (h *ProtocolWave2Handler) UpdateTdengineConfig(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Name            string            `json:"name"`
		Status          string            `json:"status"`
		Protocol        string            `json:"protocol"`
		Host            string            `json:"host"`
		Port            *int              `json:"port"`
		Username        string            `json:"username"`
		Database        string            `json:"databaseName"`
		Timezone        *string           `json:"timezone"`
		TLSSkipVerify   bool              `json:"tlsSkipVerify"`
		Options         map[string]any    `json:"options"`
		Secrets         map[string]string `json:"secrets"`
		ClearSecretKeys []string          `json:"clearSecretKeys"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	connection, err := h.service.UpdateTdengineConfig(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, service.CreateTdengineConfigInput{Name: request.Name, Status: request.Status, Protocol: request.Protocol, Host: request.Host, Port: request.Port, Username: request.Username, Database: request.Database, Timezone: request.Timezone, TLSSkipVerify: request.TLSSkipVerify, Options: request.Options, Secrets: request.Secrets, ClearSecretKeys: request.ClearSecretKeys})
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
