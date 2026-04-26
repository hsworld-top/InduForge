package handler

import (
	"net/http"

	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// PreviewHandler 负责承接 preview 会话生命周期相关 HTTP 请求。
type PreviewHandler struct {
	service        *service.PreviewSessionService
	runtimeCleaner previewSessionRuntimeCleaner
}

type previewSessionRuntimeCleaner interface {
	CloseSession(sessionID string)
}

// NewPreviewHandler 创建 preview 会话处理器。
func NewPreviewHandler(previewService *service.PreviewSessionService, runtimeCleaner previewSessionRuntimeCleaner) *PreviewHandler {
	return &PreviewHandler{
		service:        previewService,
		runtimeCleaner: runtimeCleaner,
	}
}

// Create 创建项目内预览会话，并返回会话快照。
func (h *PreviewHandler) Create(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Meta map[string]any `json:"meta"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	result, err := h.service.CreateSession(r.Context(), claims, r.PathValue("projectId"), service.CreatePreviewSessionInput{
		Meta: request.Meta,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// Heartbeat 对预览会话执行滑动续期。
func (h *PreviewHandler) Heartbeat(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	result, err := h.service.HeartbeatSession(r.Context(), claims, r.PathValue("sessionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// Diagnose 返回预览链路诊断信息，用于开发态 debug 页面定位握手失败原因。
func (h *PreviewHandler) Diagnose(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	result, err := h.service.DiagnosePreview(r.Context(), claims, r.PathValue("projectId"), r.URL.Query().Get("sessionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// Delete 关闭预览会话并清理 Redis 临时状态。
func (h *PreviewHandler) Delete(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	if err := h.service.CloseSession(r.Context(), claims, r.PathValue("sessionId")); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	if h.runtimeCleaner != nil {
		h.runtimeCleaner.CloseSession(r.PathValue("sessionId"))
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}
