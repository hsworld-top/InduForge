package handler

import (
	"net/http"

	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// ComputeHandler 负责承接 compute 领域 HTTP 请求。
type ComputeHandler struct {
	service *service.ComputeService
}

// NewComputeHandler 创建 compute 处理器。
func NewComputeHandler(computeService *service.ComputeService) *ComputeHandler {
	return &ComputeHandler{service: computeService}
}

// Create 创建计算单元定义。
func (h *ComputeHandler) Create(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Name          string         `json:"name"`
		Language      string         `json:"language"`
		ScriptCode    string         `json:"scriptCode"`
		TriggerType   string         `json:"triggerType"`
		TriggerConfig map[string]any `json:"triggerConfig"`
		InputBindings map[string]any `json:"inputBindings"`
		OutputBinding map[string]any `json:"outputBindings"`
		TimeoutMS     *int           `json:"timeoutMs"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	result, err := h.service.CreateComputeUnit(r.Context(), claims, r.PathValue("projectId"), service.CreateComputeUnitInput{
		Name:          request.Name,
		Language:      request.Language,
		ScriptCode:    request.ScriptCode,
		TriggerType:   request.TriggerType,
		TriggerConfig: request.TriggerConfig,
		InputBindings: request.InputBindings,
		OutputBinding: request.OutputBinding,
		TimeoutMS:     request.TimeoutMS,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// Run 执行一次计算单元（正式运行模式）。
func (h *ComputeHandler) Run(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Input map[string]any `json:"input"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	result, err := h.service.RunComputeUnit(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"), service.RunComputeUnitInput{
		Input: request.Input,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// Debug 执行一次计算单元（调试模式）。
func (h *ComputeHandler) Debug(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Input map[string]any `json:"input"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	result, err := h.service.DebugComputeUnit(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"), service.RunComputeUnitInput{
		Input: request.Input,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}
