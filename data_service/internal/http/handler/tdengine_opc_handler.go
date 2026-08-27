package handler

import (
	"net/http"

	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// TDengineOPCHandler 负责 TDengine 配置和 OPC DA 合约接口。
type TDengineOPCHandler struct {
	service *service.TDengineOPCService
}

// NewTDengineOPCHandler 创建TDengine 与 OPC DA处理器。
func NewTDengineOPCHandler(protocolService *service.TDengineOPCService) *TDengineOPCHandler {
	return &TDengineOPCHandler{service: protocolService}
}

// CreateTdengineConfig 创建 TDengine 配置。
func (h *TDengineOPCHandler) CreateTdengineConfig(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Name            string            `json:"name"`
		Enabled         *bool             `json:"enabled"`
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
		Enabled:  request.Enabled,
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

func (h *TDengineOPCHandler) UpdateTdengineConfig(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Name            string            `json:"name"`
		Enabled         *bool             `json:"enabled"`
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
	connection, err := h.service.UpdateTdengineConfig(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, service.CreateTdengineConfigInput{Name: request.Name, Enabled: request.Enabled, Protocol: request.Protocol, Host: request.Host, Port: request.Port, Username: request.Username, Database: request.Database, Timezone: request.Timezone, TLSSkipVerify: request.TLSSkipVerify, Options: request.Options, Secrets: request.Secrets, ClearSecretKeys: request.ClearSecretKeys})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), connection)
	return nil
}

// ValidateOpcdaContract 校验 OPC DA 合约字段。
func (h *TDengineOPCHandler) ValidateOpcdaContract(w http.ResponseWriter, r *http.Request) error {
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
