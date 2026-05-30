package handler

import (
	"encoding/json"
	"net/http"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// HTTPWorkbenchHandler 暴露 HTTP 工作台集合树、请求配置与发送接口。
type HTTPWorkbenchHandler struct {
	service *service.HTTPWorkbenchService
}

// NewHTTPWorkbenchHandler 创建 HTTP 工作台处理器。
func NewHTTPWorkbenchHandler(httpService *service.HTTPWorkbenchService) *HTTPWorkbenchHandler {
	return &HTTPWorkbenchHandler{service: httpService}
}

// ListGroups 查询 HTTP 请求分组。
func (h *HTTPWorkbenchHandler) ListGroups(w http.ResponseWriter, r *http.Request) error {
	groups, err := h.service.ListGroups(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"list": groups})
	return nil
}

// CreateGroup 创建 HTTP 请求分组。
func (h *HTTPWorkbenchHandler) CreateGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.CreateHTTPRequestGroupInput
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

// UpdateGroup 更新 HTTP 请求分组。
func (h *HTTPWorkbenchHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.UpdateHTTPRequestGroupInput
	raw, err := decodeHTTPWorkbenchBody(r, &input)
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

// DeleteGroup 删除 HTTP 请求分组。
func (h *HTTPWorkbenchHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) error {
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

// ListRequests 查询 HTTP 请求分页。
func (h *HTTPWorkbenchHandler) ListRequests(w http.ResponseWriter, r *http.Request) error {
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
	result, err := h.service.ListRequestsPage(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), groupID, query.Get("q"), page, pageSize)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// GetRequest 读取单个 HTTP 请求。
func (h *HTTPWorkbenchHandler) GetRequest(w http.ResponseWriter, r *http.Request) error {
	request, err := h.service.GetRequest(r.Context(), r.PathValue("projectId"), r.PathValue("requestId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), request)
	return nil
}

// CreateRequest 创建 HTTP 请求。
func (h *HTTPWorkbenchHandler) CreateRequest(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.CreateHTTPRequestInput
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	request, err := h.service.CreateRequest(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), request)
	return nil
}

// UpdateRequest 更新 HTTP 请求。
func (h *HTTPWorkbenchHandler) UpdateRequest(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.UpdateHTTPRequestInput
	raw, err := decodeHTTPWorkbenchBody(r, &input)
	if err != nil {
		return err
	}
	_, input.HasGroupID = raw["groupId"]
	request, err := h.service.UpdateRequest(r.Context(), r.PathValue("projectId"), r.PathValue("requestId"), claims.UserID, input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), request)
	return nil
}

// DeleteRequest 删除 HTTP 请求。
func (h *HTTPWorkbenchHandler) DeleteRequest(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	if err := h.service.DeleteRequest(r.Context(), r.PathValue("projectId"), r.PathValue("requestId")); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"deleted": true})
	return nil
}

// SendRequest 执行一次开发态 HTTP 请求。
func (h *HTTPWorkbenchHandler) SendRequest(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.SendRequest(r.Context(), r.PathValue("projectId"), r.PathValue("requestId"), claims.UserID)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func decodeHTTPWorkbenchBody(r *http.Request, target any) (map[string]json.RawMessage, error) {
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
