package handler

import (
	"net/http"

	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// ProtocolDevSessionHandler 暴露 OPC UA / Modbus 开发态短时会话接口。
type ProtocolDevSessionHandler struct {
	service *service.ProtocolDevSessionService
}

// NewProtocolDevSessionHandler 创建协议开发态会话接口处理器。
func NewProtocolDevSessionHandler(service *service.ProtocolDevSessionService) *ProtocolDevSessionHandler {
	return &ProtocolDevSessionHandler{service: service}
}

// CreateOpcua 创建 OPC UA 开发态会话。
func (h *ProtocolDevSessionHandler) CreateOpcua(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.CreateSession(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, "opcua")
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// CloseOpcua 断开 OPC UA 开发态会话。
func (h *ProtocolDevSessionHandler) CloseOpcua(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.CloseSession(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("sessionId"), claims.UserID, "opcua")
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// BrowseOpcua 返回 OPC UA 开发态浏览树。
func (h *ProtocolDevSessionHandler) BrowseOpcua(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.BrowseOpcua(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("sessionId"), claims.UserID)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// ReadOpcua 读取 OPC UA 开发态变量当前值。
func (h *ProtocolDevSessionHandler) ReadOpcua(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request protocolDevReadRequest
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.ReadOpcua(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("sessionId"), claims.UserID, request.IDs, request.GroupID)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// SubscribeOpcua 返回 OPC UA 短时订阅快照。
func (h *ProtocolDevSessionHandler) SubscribeOpcua(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request protocolDevReadRequest
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.SubscribeOpcua(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("sessionId"), claims.UserID, request.IDs, request.GroupID)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// StopSubscribeOpcua 结束 OPC UA 短时订阅。
func (h *ProtocolDevSessionHandler) StopSubscribeOpcua(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.StopSubscribeOpcua(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("sessionId"), claims.UserID)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// CreateModbus 创建 Modbus 开发态会话。
func (h *ProtocolDevSessionHandler) CreateModbus(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.CreateSession(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, "modbus")
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// CloseModbus 断开 Modbus 开发态会话。
func (h *ProtocolDevSessionHandler) CloseModbus(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.CloseSession(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("sessionId"), claims.UserID, "modbus")
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// ReadModbus 读取 Modbus 开发态寄存器当前值。
func (h *ProtocolDevSessionHandler) ReadModbus(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request protocolDevReadRequest
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.ReadModbus(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("sessionId"), claims.UserID, request.IDs, request.GroupID)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// PollModbus 返回 Modbus 开发态短时轮询快照。
func (h *ProtocolDevSessionHandler) PollModbus(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request protocolDevReadRequest
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.PollModbus(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("sessionId"), claims.UserID, request.GroupID)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// StopPollModbus 结束 Modbus 短时轮询。
func (h *ProtocolDevSessionHandler) StopPollModbus(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.StopPollModbus(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("sessionId"), claims.UserID)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

type protocolDevReadRequest struct {
	IDs     []string `json:"ids"`
	GroupID *string  `json:"groupId"`
}
