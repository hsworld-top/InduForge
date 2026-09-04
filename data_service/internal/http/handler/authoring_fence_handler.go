package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/repository"
)

type AuthoringFenceHandler struct {
	repository *repository.AuthoringFenceRepository
}

func NewAuthoringFenceHandler(repository *repository.AuthoringFenceRepository) *AuthoringFenceHandler {
	return &AuthoringFenceHandler{repository: repository}
}

type authoringFenceInput struct {
	TenantID               string `json:"tenantId"`
	OwnerID                string `json:"ownerId"`
	FenceToken             string `json:"fenceToken"`
	TTLSeconds             int    `json:"ttlSeconds"`
	ExpectedAuthoringEpoch string `json:"expectedAuthoringEpoch"`
	Mode                   string `json:"mode"`
}

func validateFenceInput(projectID string, input authoringFenceInput, tokenRequired bool) (time.Duration, error) {
	if _, err := uuid.Parse(projectID); err != nil {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "项目 ID 非法")
	}
	if strings.TrimSpace(input.TenantID) == "" {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "租户 ID 不能为空")
	}
	if _, err := uuid.Parse(input.OwnerID); err != nil {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "写保护所有者 ID 非法")
	}
	if tokenRequired && strings.TrimSpace(input.FenceToken) == "" {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "写保护令牌不能为空")
	}
	if input.TTLSeconds != 60 {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "写保护租期必须为 60 秒")
	}
	if input.Mode == "" {
		input.Mode = "capture"
	}
	if input.Mode != "capture" && input.Mode != "restore" {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "写保护模式无效")
	}
	return 60 * time.Second, nil
}

func (h *AuthoringFenceHandler) Acquire(w http.ResponseWriter, r *http.Request) error {
	var input authoringFenceInput
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	if input.Mode == "" {
		input.Mode = "capture"
	}
	ttl, err := validateFenceInput(r.PathValue("projectId"), input, false)
	if err != nil {
		return err
	}
	item, err := h.repository.Acquire(r.Context(), r.PathValue("projectId"), input.TenantID, input.OwnerID, input.Mode, input.ExpectedAuthoringEpoch, ttl)
	if err != nil {
		return err
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), item)
	return nil
}
func (h *AuthoringFenceHandler) Renew(w http.ResponseWriter, r *http.Request) error {
	var input authoringFenceInput
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	ttl, err := validateFenceInput(r.PathValue("projectId"), input, true)
	if err != nil {
		return err
	}
	item, err := h.repository.Renew(r.Context(), r.PathValue("projectId"), input.TenantID, input.OwnerID, input.FenceToken, ttl)
	if err != nil {
		return err
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), item)
	return nil
}
func (h *AuthoringFenceHandler) Release(w http.ResponseWriter, r *http.Request) error {
	var input authoringFenceInput
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	if input.TTLSeconds == 0 {
		input.TTLSeconds = 60
	}
	if _, err := validateFenceInput(r.PathValue("projectId"), input, true); err != nil {
		return err
	}
	if err := h.repository.Release(r.Context(), r.PathValue("projectId"), input.TenantID, input.OwnerID, input.FenceToken); err != nil {
		return err
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"released": true})
	return nil
}
