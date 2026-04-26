package handler

import (
	"net/http"

	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// AccessSourceHandler 负责接入源工作区的统一读模型接口。
type AccessSourceHandler struct {
	service *service.AccessSourceService
}

// NewAccessSourceHandler 创建接入源处理器。
func NewAccessSourceHandler(accessSourceService *service.AccessSourceService) *AccessSourceHandler {
	return &AccessSourceHandler{service: accessSourceService}
}

// List 返回项目下的统一接入源列表。
func (h *AccessSourceHandler) List(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	result, err := h.service.ListAccessSources(r.Context(), r.PathValue("projectId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"list": result})
	return nil
}

// Get 返回单个接入源详情。
func (h *AccessSourceHandler) Get(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	result, err := h.service.GetAccessSource(r.Context(), r.PathValue("projectId"), r.PathValue("sourceId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// Records 返回单个接入源的最近开发态记录。
func (h *AccessSourceHandler) Records(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	limit, err := parseOptionalInt(r.URL.Query().Get("limit"), 20, "limit")
	if err != nil {
		return err
	}
	result, err := h.service.ListAccessSourceRecords(r.Context(), r.PathValue("projectId"), r.PathValue("sourceId"), limit)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"list": result})
	return nil
}
