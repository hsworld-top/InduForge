package handler

import (
	"net/http"

	apperrors "github.com/indu-forge/data_service/internal/errors"
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

// List 返回项目内计算单元列表。
func (h *ComputeHandler) List(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	filter, err := parseComputeUnitListFilter(r)
	if err != nil {
		return err
	}

	result, err := h.service.ListComputeUnits(r.Context(), claims, r.PathValue("projectId"), filter)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// Get 返回单个计算单元详情。
func (h *ComputeHandler) Get(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	result, err := h.service.GetComputeUnit(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// Create 创建计算单元定义。
func (h *ComputeHandler) Create(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Name          string         `json:"name"`
		Description   *string        `json:"description"`
		Language      string         `json:"language"`
		Lang          string         `json:"lang"`
		ScriptCode    string         `json:"scriptCode"`
		Code          string         `json:"code"`
		FolderID      *string        `json:"folderId"`
		TriggerType   string         `json:"triggerType"`
		TriggerConfig map[string]any `json:"triggerConfig"`
		InputBindings map[string]any `json:"inputBindings"`
		OutputBinding map[string]any `json:"outputBindings"`
		Dependencies  []any          `json:"dependencies"`
		TimeoutMS     *int           `json:"timeoutMs"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	result, err := h.service.CreateComputeUnit(r.Context(), claims, r.PathValue("projectId"), service.CreateComputeUnitInput{
		Name:          request.Name,
		Description:   request.Description,
		FolderID:      request.FolderID,
		Language:      firstNonEmpty(request.Language, request.Lang),
		ScriptCode:    firstNonEmpty(request.ScriptCode, request.Code),
		TriggerType:   request.TriggerType,
		TriggerConfig: request.TriggerConfig,
		InputBindings: request.InputBindings,
		OutputBinding: request.OutputBinding,
		Dependencies:  request.Dependencies,
		TimeoutMS:     request.TimeoutMS,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// Dependencies 返回计算单元可用依赖清单。
func (h *ComputeHandler) Dependencies(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	result, err := h.service.ListComputeDependencies(r.Context(), claims, r.PathValue("projectId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// ListFolders 返回项目内计算单元文件夹树。
func (h *ComputeHandler) ListFolders(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	result, err := h.service.ListComputeFolders(r.Context(), claims, r.PathValue("projectId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// CreateFolder 创建计算单元文件夹。
func (h *ComputeHandler) CreateFolder(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Name     string  `json:"name"`
		ParentID *string `json:"parentId"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	result, err := h.service.CreateComputeFolder(r.Context(), claims, r.PathValue("projectId"), service.CreateComputeFolderInput{
		Name:     request.Name,
		ParentID: request.ParentID,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// UpdateFolder 更新计算单元文件夹。
func (h *ComputeHandler) UpdateFolder(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	input, err := decodeUpdateComputeFolderInput(r)
	if err != nil {
		return err
	}

	result, err := h.service.UpdateComputeFolder(r.Context(), claims, r.PathValue("projectId"), r.PathValue("folderId"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// DeleteFolder 删除计算单元文件夹。
func (h *ComputeHandler) DeleteFolder(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	if err := h.service.DeleteComputeFolder(r.Context(), claims, r.PathValue("projectId"), r.PathValue("folderId")); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}

// Update 更新计算单元定义，并同步输出数据点。
func (h *ComputeHandler) Update(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	input, err := decodeUpdateComputeUnitInput(r)
	if err != nil {
		return err
	}

	result, err := h.service.UpdateComputeUnit(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// Delete 删除计算单元，并让关联 calc.output 数据点失效。
func (h *ComputeHandler) Delete(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	if err := h.service.DeleteComputeUnit(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id")); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}

// ToggleEnabled 切换计算单元启用状态。
func (h *ComputeHandler) ToggleEnabled(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Enabled bool `json:"enabled"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	result, err := h.service.ToggleComputeUnit(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"), request.Enabled)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// Runs 返回计算单元运行记录。
func (h *ComputeHandler) Runs(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	filter, err := parseComputeRunListFilter(r)
	if err != nil {
		return err
	}
	filter.UnitID = r.PathValue("id")

	result, err := h.service.ListComputeRuns(r.Context(), claims, r.PathValue("projectId"), filter)
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
		Input  map[string]any `json:"input"`
		DryRun bool           `json:"dryRun"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	result, err := h.service.DebugComputeUnit(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"), service.RunComputeUnitInput{
		Input:  request.Input,
		DryRun: request.DryRun,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func parseComputeUnitListFilter(r *http.Request) (service.ComputeUnitListFilter, error) {
	query := r.URL.Query()
	page, err := parseOptionalInt(query.Get("page"), 1, "page")
	if err != nil {
		return service.ComputeUnitListFilter{}, err
	}
	pageSize, err := parseOptionalInt(firstNonEmpty(query.Get("pageSize"), query.Get("limit")), 20, "pageSize")
	if err != nil {
		return service.ComputeUnitListFilter{}, err
	}
	enabled, err := parseOptionalBool(query.Get("enabled"))
	if err != nil {
		return service.ComputeUnitListFilter{}, err
	}
	return service.ComputeUnitListFilter{
		Language: query.Get("language"),
		Enabled:  enabled,
		Search:   query.Get("search"),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func parseComputeRunListFilter(r *http.Request) (service.ComputeRunListFilter, error) {
	query := r.URL.Query()
	page, err := parseOptionalInt(query.Get("page"), 1, "page")
	if err != nil {
		return service.ComputeRunListFilter{}, err
	}
	pageSize, err := parseOptionalInt(firstNonEmpty(query.Get("pageSize"), query.Get("limit")), 20, "pageSize")
	if err != nil {
		return service.ComputeRunListFilter{}, err
	}
	return service.ComputeRunListFilter{
		Status:      query.Get("status"),
		TriggerMode: query.Get("triggerMode"),
		Page:        page,
		PageSize:    pageSize,
	}, nil
}

func decodeUpdateComputeUnitInput(r *http.Request) (service.UpdateComputeUnitInput, error) {
	var raw map[string]any
	if err := decodeJSONBody(r, &raw); err != nil {
		return service.UpdateComputeUnitInput{}, err
	}

	var input service.UpdateComputeUnitInput
	if value, ok, err := optionalStringField(raw, "name"); err != nil {
		return service.UpdateComputeUnitInput{}, err
	} else if ok {
		input.Name = value
	}
	if value, ok, err := optionalNullableStringField(raw, "description"); err != nil {
		return service.UpdateComputeUnitInput{}, err
	} else if ok {
		input.Description = value
		input.HasDescription = true
	}
	if value, ok, err := optionalStringField(raw, "language"); err != nil {
		return service.UpdateComputeUnitInput{}, err
	} else if ok {
		input.Language = value
	}
	if input.Language == nil {
		if value, ok, err := optionalStringField(raw, "lang"); err != nil {
			return service.UpdateComputeUnitInput{}, err
		} else if ok {
			input.Language = value
		}
	}
	if value, ok, err := optionalStringField(raw, "scriptCode"); err != nil {
		return service.UpdateComputeUnitInput{}, err
	} else if ok {
		input.ScriptCode = value
	}
	if input.ScriptCode == nil {
		if value, ok, err := optionalStringField(raw, "code"); err != nil {
			return service.UpdateComputeUnitInput{}, err
		} else if ok {
			input.ScriptCode = value
		}
	}
	if value, ok, err := optionalNullableStringField(raw, "folderId"); err != nil {
		return service.UpdateComputeUnitInput{}, err
	} else if ok {
		input.FolderID = value
		input.HasFolderID = true
	}
	if value, ok, err := optionalStringField(raw, "triggerType"); err != nil {
		return service.UpdateComputeUnitInput{}, err
	} else if ok {
		input.TriggerType = value
	}
	if value, ok, err := optionalObjectField(raw, "triggerConfig"); err != nil {
		return service.UpdateComputeUnitInput{}, err
	} else if ok {
		input.TriggerConfig = value
		input.HasTriggerConfig = true
	}
	if value, ok, err := optionalObjectField(raw, "inputBindings"); err != nil {
		return service.UpdateComputeUnitInput{}, err
	} else if ok {
		input.InputBindings = value
		input.HasInputBindings = true
	}
	if value, ok, err := optionalObjectField(raw, "outputBindings"); err != nil {
		return service.UpdateComputeUnitInput{}, err
	} else if ok {
		input.OutputBinding = value
		input.HasOutputBinding = true
	}
	if value, ok, err := optionalArrayField(raw, "dependencies"); err != nil {
		return service.UpdateComputeUnitInput{}, err
	} else if ok {
		input.Dependencies = value
		input.HasDependencies = true
	}
	if value, ok, err := optionalIntField(raw, "timeoutMs"); err != nil {
		return service.UpdateComputeUnitInput{}, err
	} else if ok {
		input.TimeoutMS = value
	}
	if value, ok, err := optionalBoolField(raw, "isEnabled"); err != nil {
		return service.UpdateComputeUnitInput{}, err
	} else if ok {
		input.IsEnabled = value
	}
	return input, nil
}

func decodeUpdateComputeFolderInput(r *http.Request) (service.UpdateComputeFolderInput, error) {
	var raw map[string]any
	if err := decodeJSONBody(r, &raw); err != nil {
		return service.UpdateComputeFolderInput{}, err
	}

	var input service.UpdateComputeFolderInput
	if value, ok, err := optionalStringField(raw, "name"); err != nil {
		return service.UpdateComputeFolderInput{}, err
	} else if ok {
		input.Name = value
	}
	if value, ok, err := optionalNullableStringField(raw, "parentId"); err != nil {
		return service.UpdateComputeFolderInput{}, err
	} else if ok {
		input.ParentID = value
		input.HasParentID = true
	}
	return input, nil
}

func optionalStringField(raw map[string]any, field string) (*string, bool, error) {
	value, exists := raw[field]
	if !exists {
		return nil, false, nil
	}
	text, ok := value.(string)
	if !ok {
		return nil, false, invalidComputeField(field)
	}
	return &text, true, nil
}

func optionalNullableStringField(raw map[string]any, field string) (*string, bool, error) {
	value, exists := raw[field]
	if !exists {
		return nil, false, nil
	}
	if value == nil {
		return nil, true, nil
	}
	text, ok := value.(string)
	if !ok {
		return nil, false, invalidComputeField(field)
	}
	return &text, true, nil
}

func optionalObjectField(raw map[string]any, field string) (map[string]any, bool, error) {
	value, exists := raw[field]
	if !exists {
		return nil, false, nil
	}
	if value == nil {
		return map[string]any{}, true, nil
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, false, invalidComputeField(field)
	}
	return object, true, nil
}

func optionalArrayField(raw map[string]any, field string) ([]any, bool, error) {
	value, exists := raw[field]
	if !exists {
		return nil, false, nil
	}
	if value == nil {
		return []any{}, true, nil
	}
	list, ok := value.([]any)
	if !ok {
		return nil, false, invalidComputeField(field)
	}
	return list, true, nil
}

func optionalIntField(raw map[string]any, field string) (*int, bool, error) {
	value, exists := raw[field]
	if !exists {
		return nil, false, nil
	}
	number, ok := value.(float64)
	if !ok || number != float64(int(number)) {
		return nil, false, invalidComputeField(field)
	}
	result := int(number)
	return &result, true, nil
}

func optionalBoolField(raw map[string]any, field string) (*bool, bool, error) {
	value, exists := raw[field]
	if !exists {
		return nil, false, nil
	}
	boolean, ok := value.(bool)
	if !ok {
		return nil, false, invalidComputeField(field)
	}
	return &boolean, true, nil
}

func invalidComputeField(field string) error {
	return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, field+" 字段格式无效")
}
