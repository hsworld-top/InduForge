package handler

import (
	"encoding/json"
	"net/http"

	"github.com/indu-forge/data_service/internal/auth"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// ConnectionHandler 负责承接 connections 领域的 HTTP 请求。
type ConnectionHandler struct {
	service *service.ConnectionService
}

// NewConnectionHandler 创建连接处理器。
func NewConnectionHandler(connectionService *service.ConnectionService) *ConnectionHandler {
	return &ConnectionHandler{service: connectionService}
}

// List 返回项目连接列表。
func (h *ConnectionHandler) List(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	connections, err := h.service.ListConnections(r.Context(), r.PathValue("projectId"), claims.TenantID)
	if err != nil {
		return err
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), connections)
	return nil
}

// Create 创建项目连接。
func (h *ConnectionHandler) Create(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Name   string         `json:"name"`
		Type   string         `json:"type"`
		Status string         `json:"status"`
		Config map[string]any `json:"config"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	connection, err := h.service.CreateConnection(r.Context(), r.PathValue("projectId"), claims.TenantID, claims.UserID, service.CreateConnectionInput{
		Name:   request.Name,
		Type:   request.Type,
		Status: request.Status,
		Config: request.Config,
	})
	if err != nil {
		return err
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), connection)
	return nil
}

// Update 更新项目连接。
func (h *ConnectionHandler) Update(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var raw map[string]json.RawMessage
	if err := decodeJSONBody(r, &raw); err != nil {
		return err
	}
	if err := validateUpdatePayloadKeys(raw); err != nil {
		return err
	}

	input := service.UpdateConnectionInput{}
	if value, ok := raw["name"]; ok {
		var name string
		if err := json.Unmarshal(value, &name); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "name 字段格式无效", err)
		}
		input.Name = &name
	}
	if value, ok := raw["type"]; ok {
		var connectionType string
		if err := json.Unmarshal(value, &connectionType); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "type 字段格式无效", err)
		}
		input.Type = &connectionType
	}
	if value, ok := raw["status"]; ok {
		var status string
		if err := json.Unmarshal(value, &status); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "status 字段格式无效", err)
		}
		input.Status = &status
	}
	if value, ok := raw["config"]; ok {
		var config map[string]any
		if err := json.Unmarshal(value, &config); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "config 字段格式无效", err)
		}
		input.Config = config
		input.HasConfig = true
	}

	connection, err := h.service.UpdateConnection(
		r.Context(),
		r.PathValue("projectId"),
		r.PathValue("connectionId"),
		claims.TenantID,
		claims.UserID,
		input,
	)
	if err != nil {
		return err
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), connection)
	return nil
}

// Delete 删除项目连接。
func (h *ConnectionHandler) Delete(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	if err := h.service.DeleteConnection(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId")); err != nil {
		return err
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}

func requireClaims(r *http.Request) (*auth.Claims, error) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeAuthTokenRequired, http.StatusUnauthorized, "请先完成认证")
	}
	return claims, nil
}

func decodeJSONBody(r *http.Request, target any) error {
	if r.Body == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "请求体不能为空")
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "请求体 JSON 格式无效", err)
	}
	if decoder.More() {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "请求体只能包含一个 JSON 对象")
	}
	return nil
}

func validateUpdatePayloadKeys(raw map[string]json.RawMessage) error {
	allowedFields := map[string]struct{}{
		"name":   {},
		"type":   {},
		"status": {},
		"config": {},
	}

	for field := range raw {
		if _, ok := allowedFields[field]; ok {
			continue
		}
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "更新请求包含不支持的字段: "+field)
	}

	return nil
}
