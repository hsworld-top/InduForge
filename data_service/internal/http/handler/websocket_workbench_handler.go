package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/indu-forge/data_service/internal/auth"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// WebSocketWorkbenchHandler 暴露 WebSocket 工作台集合树、会话配置与短连接预览接口。
type WebSocketWorkbenchHandler struct {
	service *service.WebSocketWorkbenchService
}

func NewWebSocketWorkbenchHandler(websocketService *service.WebSocketWorkbenchService) *WebSocketWorkbenchHandler {
	return &WebSocketWorkbenchHandler{service: websocketService}
}

func (h *WebSocketWorkbenchHandler) ListGroups(w http.ResponseWriter, r *http.Request) error {
	groups, err := h.service.ListGroups(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"list": groups})
	return nil
}

func (h *WebSocketWorkbenchHandler) CreateGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.CreateWebSocketSessionGroupInput
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	group, err := h.service.CreateGroup(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), group)
	return nil
}

func (h *WebSocketWorkbenchHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.UpdateWebSocketSessionGroupInput
	raw, err := decodeWebSocketWorkbenchBody(r, &input)
	if err != nil {
		return err
	}
	_, input.HasParentID = raw["parentId"]
	group, err := h.service.UpdateGroup(r.Context(), r.PathValue("projectId"), r.PathValue("groupId"), claims.UserID, input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), group)
	return nil
}

func (h *WebSocketWorkbenchHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	if err := h.service.DeleteGroup(r.Context(), r.PathValue("projectId"), r.PathValue("groupId"), claims.UserID); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"deleted": true})
	return nil
}

func (h *WebSocketWorkbenchHandler) ListSessions(w http.ResponseWriter, r *http.Request) error {
	query := r.URL.Query()
	var groupID *string
	if value := query.Get("groupId"); value != "" {
		groupID = &value
	}
	page, err := parseOptionalInt(query.Get("page"), 1, "page")
	if err != nil {
		return err
	}
	pageSize, err := parseOptionalInt(firstNonEmpty(query.Get("pageSize"), query.Get("limit")), 20, "pageSize")
	if err != nil {
		return err
	}
	result, err := h.service.ListSessionsPage(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), groupID, query.Get("q"), page, pageSize)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *WebSocketWorkbenchHandler) GetSession(w http.ResponseWriter, r *http.Request) error {
	session, err := h.service.GetSession(r.Context(), r.PathValue("projectId"), r.PathValue("sessionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), session)
	return nil
}

func (h *WebSocketWorkbenchHandler) CreateSession(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.CreateWebSocketSessionInput
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	session, err := h.service.CreateSession(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), session)
	return nil
}

func (h *WebSocketWorkbenchHandler) UpdateSession(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.UpdateWebSocketSessionInput
	raw, err := decodeWebSocketWorkbenchBody(r, &input)
	if err != nil {
		return err
	}
	_, input.HasGroupID = raw["groupId"]
	session, err := h.service.UpdateSession(r.Context(), r.PathValue("projectId"), r.PathValue("sessionId"), claims.UserID, input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), session)
	return nil
}

func (h *WebSocketWorkbenchHandler) DeleteSession(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	if err := h.service.DeleteSession(r.Context(), r.PathValue("projectId"), r.PathValue("sessionId")); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"deleted": true})
	return nil
}

func (h *WebSocketWorkbenchHandler) ConnectPreview(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.ConnectPreview(r.Context(), r.PathValue("projectId"), r.PathValue("sessionId"), claims.UserID)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *WebSocketWorkbenchHandler) StreamSession(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	if err := h.service.StreamSession(r.Context(), w, r.PathValue("projectId"), r.PathValue("sessionId"), claims.UserID); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	return nil
}

func (h *WebSocketWorkbenchHandler) StreamSessionWithQueryToken(validator *auth.JWTValidator) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if validator == nil {
			response.WriteAppError(w, http.StatusInternalServerError, middleware.RequestID(r.Context()), apperrors.ErrorCodeAuthSecretRequired, "JWT 校验器未初始化")
			return
		}
		token := ""
		if cookie, err := r.Cookie("if_access"); err == nil {
			token = normalizeWorkbenchStreamToken(cookie.Value)
		}
		if token == "" {
			token = normalizeWorkbenchStreamToken(r.Header.Get("Authorization"))
		}
		if token == "" {
			response.WriteAppError(w, http.StatusUnauthorized, middleware.RequestID(r.Context()), apperrors.ErrorCodeAuthTokenRequired, "请提供 Bearer JWT")
			return
		}
		claims, err := validator.Validate(token)
		if err != nil {
			response.WriteAppError(w, http.StatusUnauthorized, middleware.RequestID(r.Context()), apperrors.ErrorCodeAuthTokenInvalid, "JWT 校验失败")
			return
		}
		if !claims.HasCapability("project:read") {
			response.WriteAppError(w, http.StatusForbidden, middleware.RequestID(r.Context()), apperrors.ErrorCodePermissionInsufficient, "缺少 project:read 权限")
			return
		}
		if err := h.service.StreamSession(auth.WithClaims(r.Context(), claims), w, r.PathValue("projectId"), r.PathValue("sessionId"), claims.UserID); err != nil {
			writeWorkbenchStreamError(w, r, normalizeRepresentativeHandlerError(err))
		}
	})
}

func writeWorkbenchStreamError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) && appErr != nil {
		response.WriteAppError(w, appErr.StatusCode, middleware.RequestID(r.Context()), appErr.Code, appErr.Message)
		return
	}
	response.WriteAppError(w, http.StatusInternalServerError, middleware.RequestID(r.Context()), apperrors.ErrorCodeInternal, "WebSocket 工作台连接失败")
}

func normalizeWorkbenchStreamToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	if parts := strings.Fields(token); len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}
	return token
}

func decodeWebSocketWorkbenchBody(r *http.Request, target any) (map[string]json.RawMessage, error) {
	if r.Body == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "请求体不能为空")
	}
	var raw map[string]json.RawMessage
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "请求体 JSON 格式无效", err)
	}
	payload, err := json.Marshal(raw)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "请求体 JSON 格式无效", err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "请求体 JSON 格式无效", err)
	}
	return raw, nil
}
