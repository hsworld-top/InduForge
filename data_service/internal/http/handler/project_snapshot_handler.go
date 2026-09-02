package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/repository"
	"github.com/indu-forge/data_service/internal/service"
)

// ProjectSnapshotHandler 负责项目快照读写请求。
type ProjectSnapshotHandler struct {
	service *service.ProjectSnapshotService
}

// BuildCollectorArtifact 仅把调用方 snapshot 当作一致性证明，实际内容由服务端权威快照生成。
func (h *ProjectSnapshotHandler) BuildCollectorArtifact(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input struct {
		TenantID       string          `json:"tenantId"`
		ProjectID      string          `json:"projectId"`
		ReleaseID      string          `json:"releaseId"`
		Revision       int64           `json:"revision"`
		SourceSnapshot json.RawMessage `json:"sourceSnapshot"`
	}
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	projectID := r.PathValue("projectId")
	if input.ProjectID != projectID || input.TenantID != claims.TenantID || input.Revision < 1 || len(input.SourceSnapshot) == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "采集工件请求身份不一致")
	}
	if _, err := uuid.Parse(projectID); err != nil {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "项目 ID 非法")
	}
	if _, err := uuid.Parse(input.ReleaseID); err != nil {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Release ID 非法")
	}
	result, err := h.service.BuildCollectorArtifact(r.Context(), projectID, claims.TenantID, "collector-"+input.ReleaseID, input.Revision, input.SourceSnapshot)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// NewProjectSnapshotHandler 创建项目快照处理器。
func NewProjectSnapshotHandler(snapshotService *service.ProjectSnapshotService) *ProjectSnapshotHandler {
	return &ProjectSnapshotHandler{service: snapshotService}
}

// Get 返回项目快照。
func (h *ProjectSnapshotHandler) Get(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	result, err := h.service.Get(r.Context(), r.PathValue("projectId"), claims.TenantID)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// GetArtifact 返回项目级 artifact v1 产物。
func (h *ProjectSnapshotHandler) GetArtifact(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	result, err := h.service.GetArtifact(r.Context(), r.PathValue("projectId"), claims.TenantID)
	if err != nil {
		// 响应对外保持统一内部错误，日志保留受控构建失败原因供运维排障。
		log.Printf("runtime artifact build failed projectId=%s: %v", r.PathValue("projectId"), err)
		return normalizeRepresentativeHandlerError(err)
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
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"updated": true})
	return nil
}
