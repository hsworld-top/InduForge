package handler

import (
	"net/http"
	"time"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// AlarmRuleHandler 承接报警规则开发态 HTTP 请求。
type AlarmRuleHandler struct {
	service *service.AlarmRuleService
}

func NewAlarmRuleHandler(alarmRuleService *service.AlarmRuleService) *AlarmRuleHandler {
	return &AlarmRuleHandler{service: alarmRuleService}
}

func (h *AlarmRuleHandler) List(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	filter, err := parseAlarmRuleListFilter(r)
	if err != nil {
		return err
	}
	result, err := h.service.List(r.Context(), claims, r.PathValue("projectId"), filter)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmRuleHandler) Create(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Name            string         `json:"name"`
		Description     *string        `json:"description"`
		TargetPath      string         `json:"targetPath"`
		RuleType        string         `json:"ruleType"`
		Condition       map[string]any `json:"condition"`
		Severity        string         `json:"severity"`
		Hysteresis      *float64       `json:"hysteresis"`
		SampleWindowMS  *int           `json:"sampleWindowMs"`
		Suppression     map[string]any `json:"suppression"`
		MessageTemplate string         `json:"messageTemplate"`
		Contract        map[string]any `json:"contract"`
		IsEnabled       *bool          `json:"isEnabled"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.Create(r.Context(), claims, r.PathValue("projectId"), service.CreateAlarmRuleInput{
		Name:            request.Name,
		Description:     request.Description,
		TargetPath:      request.TargetPath,
		RuleType:        request.RuleType,
		Condition:       request.Condition,
		Severity:        request.Severity,
		Hysteresis:      request.Hysteresis,
		SampleWindowMS:  request.SampleWindowMS,
		Suppression:     request.Suppression,
		MessageTemplate: request.MessageTemplate,
		Contract:        request.Contract,
		IsEnabled:       request.IsEnabled,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmRuleHandler) Get(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.Get(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmRuleHandler) Update(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	input, err := decodeUpdateAlarmRuleInput(r)
	if err != nil {
		return err
	}
	result, err := h.service.Update(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmRuleHandler) Delete(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	if err := h.service.Delete(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id")); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}

func (h *AlarmRuleHandler) ToggleEnabled(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		IsEnabled bool `json:"isEnabled"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.ToggleEnabled(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"), request.IsEnabled)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmRuleHandler) ValidateDraft(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Name            string         `json:"name"`
		Description     *string        `json:"description"`
		TargetPath      string         `json:"targetPath"`
		RuleType        string         `json:"ruleType"`
		Condition       map[string]any `json:"condition"`
		Severity        string         `json:"severity"`
		Hysteresis      *float64       `json:"hysteresis"`
		SampleWindowMS  *int           `json:"sampleWindowMs"`
		Suppression     map[string]any `json:"suppression"`
		MessageTemplate string         `json:"messageTemplate"`
		Contract        map[string]any `json:"contract"`
		IsEnabled       *bool          `json:"isEnabled"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.ValidateDraft(r.Context(), claims, r.PathValue("projectId"), service.CreateAlarmRuleInput{
		Name:            request.Name,
		Description:     request.Description,
		TargetPath:      request.TargetPath,
		RuleType:        request.RuleType,
		Condition:       request.Condition,
		Severity:        request.Severity,
		Hysteresis:      request.Hysteresis,
		SampleWindowMS:  request.SampleWindowMS,
		Suppression:     request.Suppression,
		MessageTemplate: request.MessageTemplate,
		Contract:        request.Contract,
		IsEnabled:       request.IsEnabled,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmRuleHandler) ValidateTarget(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.ValidateTarget(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmRuleHandler) Test(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Value     any            `json:"value"`
		Timestamp *string        `json:"timestamp"`
		Context   map[string]any `json:"context"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	var timestamp *time.Time
	if request.Timestamp != nil && *request.Timestamp != "" {
		parsed, err := time.Parse(time.RFC3339, *request.Timestamp)
		if err != nil {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "timestamp 格式无效")
		}
		timestamp = &parsed
	}
	result, err := h.service.Test(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"), service.AlarmRuleTestInput{
		Value:     request.Value,
		Timestamp: timestamp,
		Context:   request.Context,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmRuleHandler) Contract(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.Contract(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func parseAlarmRuleListFilter(r *http.Request) (service.AlarmRuleListFilter, error) {
	query := r.URL.Query()
	page, err := parseOptionalInt(query.Get("page"), 1, "page")
	if err != nil {
		return service.AlarmRuleListFilter{}, err
	}
	pageSize, err := parseOptionalInt(firstNonEmpty(query.Get("pageSize"), query.Get("limit")), 20, "pageSize")
	if err != nil {
		return service.AlarmRuleListFilter{}, err
	}
	enabled, err := parseOptionalBool(query.Get("enabled"))
	if err != nil {
		return service.AlarmRuleListFilter{}, err
	}
	return service.AlarmRuleListFilter{
		TargetPath: query.Get("targetPath"),
		RuleType:   query.Get("ruleType"),
		Severity:   query.Get("severity"),
		Enabled:    enabled,
		Search:     query.Get("search"),
		Page:       page,
		PageSize:   pageSize,
	}, nil
}

func decodeUpdateAlarmRuleInput(r *http.Request) (service.UpdateAlarmRuleInput, error) {
	var raw map[string]any
	if err := decodeJSONBody(r, &raw); err != nil {
		return service.UpdateAlarmRuleInput{}, err
	}
	var input service.UpdateAlarmRuleInput
	if value, ok, err := alarmStringField(raw, "name"); err != nil {
		return service.UpdateAlarmRuleInput{}, err
	} else if ok {
		input.Name = value
	}
	if value, ok, err := alarmStringField(raw, "description"); err != nil {
		return service.UpdateAlarmRuleInput{}, err
	} else if ok {
		input.Description = value
	}
	if value, ok, err := alarmStringField(raw, "targetPath"); err != nil {
		return service.UpdateAlarmRuleInput{}, err
	} else if ok {
		input.TargetPath = value
	}
	if value, ok, err := alarmStringField(raw, "ruleType"); err != nil {
		return service.UpdateAlarmRuleInput{}, err
	} else if ok {
		input.RuleType = value
	}
	if value, ok, err := alarmStringField(raw, "severity"); err != nil {
		return service.UpdateAlarmRuleInput{}, err
	} else if ok {
		input.Severity = value
	}
	if value, ok, err := alarmObjectField(raw, "condition"); err != nil {
		return service.UpdateAlarmRuleInput{}, err
	} else if ok {
		input.Condition = value
		input.HasCondition = true
	}
	if value, ok, err := alarmObjectField(raw, "suppression"); err != nil {
		return service.UpdateAlarmRuleInput{}, err
	} else if ok {
		input.Suppression = value
		input.HasSuppression = true
	}
	if value, ok, err := alarmStringField(raw, "messageTemplate"); err != nil {
		return service.UpdateAlarmRuleInput{}, err
	} else if ok {
		input.MessageTemplate = value
	}
	if value, ok, err := alarmObjectField(raw, "contract"); err != nil {
		return service.UpdateAlarmRuleInput{}, err
	} else if ok {
		input.Contract = value
		input.HasContract = true
	}
	if value, ok, clear, err := alarmFloatField(raw, "hysteresis"); err != nil {
		return service.UpdateAlarmRuleInput{}, err
	} else if ok {
		input.Hysteresis = value
		input.ClearHysteresis = clear
	}
	if value, ok, clear, err := alarmIntField(raw, "sampleWindowMs"); err != nil {
		return service.UpdateAlarmRuleInput{}, err
	} else if ok {
		input.SampleWindowMS = value
		input.ClearSampleWindowMS = clear
	}
	if value, ok, err := alarmBoolField(raw, "isEnabled"); err != nil {
		return service.UpdateAlarmRuleInput{}, err
	} else if ok {
		input.IsEnabled = value
	}
	return input, nil
}

func alarmStringField(raw map[string]any, field string) (*string, bool, error) {
	value, exists := raw[field]
	if !exists {
		return nil, false, nil
	}
	if value == nil {
		empty := ""
		return &empty, true, nil
	}
	text, ok := value.(string)
	if !ok {
		return nil, false, invalidAlarmField(field)
	}
	return &text, true, nil
}

func alarmObjectField(raw map[string]any, field string) (map[string]any, bool, error) {
	value, exists := raw[field]
	if !exists {
		return nil, false, nil
	}
	if value == nil {
		return map[string]any{}, true, nil
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, false, invalidAlarmField(field)
	}
	return object, true, nil
}

func alarmFloatField(raw map[string]any, field string) (*float64, bool, bool, error) {
	value, exists := raw[field]
	if !exists {
		return nil, false, false, nil
	}
	if value == nil {
		return nil, true, true, nil
	}
	number, ok := value.(float64)
	if !ok {
		return nil, false, false, invalidAlarmField(field)
	}
	return &number, true, false, nil
}

func alarmIntField(raw map[string]any, field string) (*int, bool, bool, error) {
	value, exists := raw[field]
	if !exists {
		return nil, false, false, nil
	}
	if value == nil {
		return nil, true, true, nil
	}
	number, ok := value.(float64)
	if !ok || number != float64(int(number)) {
		return nil, false, false, invalidAlarmField(field)
	}
	result := int(number)
	return &result, true, false, nil
}

func alarmBoolField(raw map[string]any, field string) (*bool, bool, error) {
	value, exists := raw[field]
	if !exists {
		return nil, false, nil
	}
	boolean, ok := value.(bool)
	if !ok {
		return nil, false, invalidAlarmField(field)
	}
	return &boolean, true, nil
}

func invalidAlarmField(field string) error {
	return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, field+" 字段格式无效")
}
