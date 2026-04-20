package handler

import (
	"net/http"

	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/repository"
	"github.com/indu-forge/data_service/internal/service"
)

// ProjectSnapshotHandler 负责项目快照读写请求。
type ProjectSnapshotHandler struct {
	service *service.ProjectSnapshotService
}

// NewProjectSnapshotHandler 创建项目快照处理器。
func NewProjectSnapshotHandler(snapshotService *service.ProjectSnapshotService) *ProjectSnapshotHandler {
	return &ProjectSnapshotHandler{service: snapshotService}
}

// Get 返回项目快照。
func (h *ProjectSnapshotHandler) Get(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	result, err := h.service.Get(r.Context(), r.PathValue("projectId"))
	if err != nil {
		return err
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// Replace 用请求快照覆盖项目数据域数据。
func (h *ProjectSnapshotHandler) Replace(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request repository.ProjectSnapshot
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	if err := h.service.Replace(r.Context(), r.PathValue("projectId"), claims.UserID, request); err != nil {
		return err
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"updated": true})
	return nil
}
