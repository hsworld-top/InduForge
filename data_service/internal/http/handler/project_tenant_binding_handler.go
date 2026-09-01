package handler

import (
	"net/http"

	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// ProjectTenantBindingHandler 承接 control 面写入项目租户归属的内部请求。
type ProjectTenantBindingHandler struct {
	service *service.ProjectTenantBindingService
}

// NewProjectTenantBindingHandler 创建项目租户绑定处理器。
func NewProjectTenantBindingHandler(bindingService *service.ProjectTenantBindingService) *ProjectTenantBindingHandler {
	return &ProjectTenantBindingHandler{service: bindingService}
}

// Put 首次绑定项目租户；重复相同请求可安全重试，改绑被拒绝。
func (h *ProjectTenantBindingHandler) Put(w http.ResponseWriter, r *http.Request) error {
	var input struct {
		TenantID string `json:"tenantId"`
	}
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}

	created, err := h.service.Bind(r.Context(), r.PathValue("projectId"), input.TenantID)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"created": created})
	return nil
}
