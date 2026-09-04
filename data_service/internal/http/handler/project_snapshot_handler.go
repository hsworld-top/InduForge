package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/repository"
	"github.com/indu-forge/data_service/internal/service"
)

const internalProjectSnapshotMaxBytes int64 = 512 << 20

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

func validateInternalSnapshotIdentity(projectID, tenantID, actorID string, actorRequired bool) error {
	if _, err := uuid.Parse(projectID); err != nil {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "项目 ID 非法")
	}
	if _, err := uuid.Parse(tenantID); err != nil {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "租户 ID 非法")
	}
	if actorRequired {
		if _, err := uuid.Parse(actorID); err != nil {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "操作人 ID 非法")
		}
	}
	return nil
}

// GetInternal 为控制面返回完整项目数据域快照。
func (h *ProjectSnapshotHandler) GetInternal(w http.ResponseWriter, r *http.Request) error {
	projectID, tenantID := r.PathValue("projectId"), r.URL.Query().Get("tenantId")
	if err := validateInternalSnapshotIdentity(projectID, tenantID, "", false); err != nil {
		return err
	}
	result, err := h.service.Get(r.Context(), projectID, tenantID)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// PutInternal 从控制面恢复完整项目数据域快照。
func (h *ProjectSnapshotHandler) PutInternal(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, internalProjectSnapshotMaxBytes)
	var input struct {
		TenantID               string                     `json:"tenantId"`
		ActorID                string                     `json:"actorId"`
		OwnerID                string                     `json:"ownerId"`
		FenceToken             string                     `json:"fenceToken"`
		ExpectedAuthoringEpoch string                     `json:"expectedAuthoringEpoch"`
		TargetAuthoringEpoch   string                     `json:"targetAuthoringEpoch"`
		Direction              string                     `json:"direction"`
		Snapshot               repository.ProjectSnapshot `json:"snapshot"`
	}
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	projectID := r.PathValue("projectId")
	if err := validateInternalSnapshotIdentity(projectID, input.TenantID, input.ActorID, true); err != nil {
		return err
	}
	if _, err := uuid.Parse(input.OwnerID); err != nil {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "写保护所有者 ID 非法")
	}
	if err := h.service.ReplaceInternal(r.Context(), projectID, input.TenantID, input.ActorID, input.OwnerID, input.FenceToken, input.ExpectedAuthoringEpoch, input.TargetAuthoringEpoch, input.Direction, input.Snapshot); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"updated": true, "authoringEpoch": input.TargetAuthoringEpoch})
	return nil
}

// BuildArtifactInternal 从请求快照纯构建运行制品，不把密文或请求体写入日志。
func (h *ProjectSnapshotHandler) BuildArtifactInternal(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, internalProjectSnapshotMaxBytes)
	var input struct {
		TenantID string                     `json:"tenantId"`
		Snapshot repository.ProjectSnapshot `json:"snapshot"`
	}
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	projectID := r.PathValue("projectId")
	if err := validateInternalSnapshotIdentity(projectID, input.TenantID, "", false); err != nil {
		return err
	}
	result, err := h.service.BuildArtifactFromSnapshot(projectID, input.TenantID, input.Snapshot)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// BuildArtifactsInternal 纯函数式生成 runtime 与可选 collector 制品。
func (h *ProjectSnapshotHandler) BuildArtifactsInternal(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, internalProjectSnapshotMaxBytes)
	var input struct {
		TenantID   string                            `json:"tenantId"`
		CapturedAt time.Time                         `json:"capturedAt"`
		Snapshot   repository.ProjectSnapshot        `json:"snapshot"`
		Collector  *service.CollectorArtifactRequest `json:"collector"`
	}
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	projectID := r.PathValue("projectId")
	if err := validateInternalSnapshotIdentity(projectID, input.TenantID, "", false); err != nil {
		return err
	}
	result, err := h.service.BuildArtifactsFromSnapshot(projectID, input.TenantID, input.CapturedAt, input.Snapshot, input.Collector)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}
