package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/indu-forge/data_service/internal/auth"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// AlarmPolicyHandler 承接报警策略开发态 HTTP 请求。
type AlarmPolicyHandler struct {
	service *service.AlarmPolicyService
}

func NewAlarmPolicyHandler(alarmPolicyService *service.AlarmPolicyService) *AlarmPolicyHandler {
	return &AlarmPolicyHandler{service: alarmPolicyService}
}

func (h *AlarmPolicyHandler) GetSettings(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.GetSettings(r.Context(), claims, r.PathValue("projectId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmPolicyHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		EscalationIntervalSeconds         int `json:"escalationIntervalSeconds"`
		RepeatNotificationIntervalSeconds int `json:"repeatNotificationIntervalSeconds"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.UpdateSettings(r.Context(), claims, r.PathValue("projectId"), service.UpdateAlarmProjectSettingsInput{
		EscalationIntervalSeconds: request.EscalationIntervalSeconds, RepeatNotificationIntervalSeconds: request.RepeatNotificationIntervalSeconds,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmPolicyHandler) ListGroups(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.ListGroups(r.Context(), claims, r.PathValue("projectId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmPolicyHandler) CreateGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Name        string  `json:"name"`
		ParentID    *string `json:"parentId"`
		Description *string `json:"description"`
		IsEnabled   *bool   `json:"isEnabled"`
		SortOrder   int     `json:"sortOrder"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.CreateGroup(r.Context(), claims, r.PathValue("projectId"), service.CreateAlarmPolicyGroupInput{
		Name: request.Name, ParentID: request.ParentID, Description: request.Description, IsEnabled: request.IsEnabled, SortOrder: request.SortOrder,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmPolicyHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSONBody(r, &raw); err != nil {
		return err
	}
	input := service.UpdateAlarmPolicyGroupInput{}
	if rawValue, ok := raw["parentId"]; ok {
		input.HasParentID = true
		if string(rawValue) != "null" {
			var value string
			if err := json.Unmarshal(rawValue, &value); err != nil {
				return invalidAlarmField("parentId")
			}
			input.ParentID = &value
		}
	}
	if err := decodePolicyRaw(raw, "name", &input.Name); err != nil {
		return err
	}
	if err := decodePolicyRaw(raw, "description", &input.Description); err != nil {
		return err
	}
	if rawValue, ok := raw["isEnabled"]; ok {
		var value bool
		if err := json.Unmarshal(rawValue, &value); err != nil {
			return invalidAlarmField("isEnabled")
		}
		input.IsEnabled = &value
	}
	if rawValue, ok := raw["sortOrder"]; ok {
		var value int
		if err := json.Unmarshal(rawValue, &value); err != nil {
			return invalidAlarmField("sortOrder")
		}
		input.SortOrder = &value
	}
	result, err := h.service.UpdateGroup(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"), service.UpdateAlarmPolicyGroupInput{
		Name: input.Name, ParentID: input.ParentID, HasParentID: input.HasParentID, Description: input.Description, IsEnabled: input.IsEnabled, SortOrder: input.SortOrder,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmPolicyHandler) ToggleGroupEnabled(w http.ResponseWriter, r *http.Request) error {
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
	result, err := h.service.ToggleGroupEnabled(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"), request.IsEnabled)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmPolicyHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	if err := h.service.DeleteGroup(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id")); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}

func (h *AlarmPolicyHandler) List(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	filter, err := parseAlarmPolicyListFilter(r)
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

func (h *AlarmPolicyHandler) Tree(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	filter, err := parseAlarmPolicyListFilter(r)
	if err != nil {
		return err
	}
	result, err := h.service.Tree(r.Context(), claims, r.PathValue("projectId"), filter)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmPolicyHandler) Coverage(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	query := r.URL.Query()
	result, err := h.service.Coverage(
		r.Context(),
		claims,
		r.PathValue("projectId"),
		query.Get("datapointId"),
		query.Get("path"),
		query.Get("excludePolicyId"),
	)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmPolicyHandler) Create(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	input, err := decodeCreateAlarmPolicyInput(r)
	if err != nil {
		return err
	}
	result, err := h.service.Create(r.Context(), claims, r.PathValue("projectId"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmPolicyHandler) Get(w http.ResponseWriter, r *http.Request) error {
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

func (h *AlarmPolicyHandler) Update(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	input, err := decodeUpdateAlarmPolicyInput(r)
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

func (h *AlarmPolicyHandler) ToggleEnabled(w http.ResponseWriter, r *http.Request) error {
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

func (h *AlarmPolicyHandler) Delete(w http.ResponseWriter, r *http.Request) error {
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

func (h *AlarmPolicyHandler) ValidateDraft(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	input, err := decodeCreateAlarmPolicyInput(r)
	if err != nil {
		return err
	}
	result, err := h.service.ValidateDraft(r.Context(), claims, r.PathValue("projectId"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmPolicyHandler) Test(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Value     any            `json:"value"`
		Values    map[string]any `json:"values"`
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
	result, err := h.service.Test(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"), service.AlarmPolicyTestInput{
		Value: request.Value, Values: request.Values, Timestamp: timestamp, Context: request.Context,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmPolicyHandler) Contract(w http.ResponseWriter, r *http.Request) error {
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

func (h *AlarmPolicyHandler) BatchEnable(w http.ResponseWriter, r *http.Request) error {
	return h.batchSelection(w, r, func(selection service.AlarmBulkSelection, claims *auth.Claims, projectID string) error {
		return h.service.BatchEnable(r.Context(), claims, projectID, selection)
	})
}

func (h *AlarmPolicyHandler) BatchDisable(w http.ResponseWriter, r *http.Request) error {
	return h.batchSelection(w, r, func(selection service.AlarmBulkSelection, claims *auth.Claims, projectID string) error {
		return h.service.BatchDisable(r.Context(), claims, projectID, selection)
	})
}

func (h *AlarmPolicyHandler) BatchMove(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Selection selectionRequest `json:"selection"`
		PolicyIDs []string         `json:"policyIds"`
		GroupID   *string          `json:"groupId"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	selection := request.Selection.toService()
	if selection.Mode == "" && len(request.PolicyIDs) > 0 {
		selection = service.AlarmBulkSelection{Mode: "ids", PolicyIDs: request.PolicyIDs}
	}
	if err := h.service.BatchMove(r.Context(), claims, r.PathValue("projectId"), selection, request.GroupID); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"updated": true})
	return nil
}

func (h *AlarmPolicyHandler) BatchApplyConditions(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Selection  selectionRequest         `json:"selection"`
		PolicyIDs  []string                 `json:"policyIds"`
		Conditions []service.AlarmCondition `json:"conditions"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	selection := request.Selection.toService()
	if selection.Mode == "" && len(request.PolicyIDs) > 0 {
		selection = service.AlarmBulkSelection{Mode: "ids", PolicyIDs: request.PolicyIDs}
	}
	if err := h.service.BatchApplyConditions(r.Context(), claims, r.PathValue("projectId"), selection, request.Conditions); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"updated": true})
	return nil
}

func (h *AlarmPolicyHandler) batchSelection(w http.ResponseWriter, r *http.Request, action func(service.AlarmBulkSelection, *auth.Claims, string) error) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Selection selectionRequest `json:"selection"`
		PolicyIDs []string         `json:"policyIds"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	selection := request.Selection.toService()
	if selection.Mode == "" && len(request.PolicyIDs) > 0 {
		selection = service.AlarmBulkSelection{Mode: "ids", PolicyIDs: request.PolicyIDs}
	}
	if err := action(selection, claims, r.PathValue("projectId")); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"updated": true})
	return nil
}

type selectionRequest struct {
	Mode             string                `json:"mode"`
	PolicyIDs        []string              `json:"policyIds"`
	Filters          alarmPolicyFilterJSON `json:"filters"`
	ExcludePolicyIDs []string              `json:"excludePolicyIds"`
}

type alarmPolicyFilterJSON struct {
	Search        string  `json:"search"`
	GroupID       *string `json:"groupId"`
	Enabled       *bool   `json:"enabled"`
	Severity      string  `json:"severity"`
	ConditionType string  `json:"conditionType"`
	TargetPath    string  `json:"targetPath"`
	Mode          string  `json:"mode"`
}

func (r selectionRequest) toService() service.AlarmBulkSelection {
	return service.AlarmBulkSelection{
		Mode:      r.Mode,
		PolicyIDs: r.PolicyIDs,
		Filters: service.AlarmPolicyListFilter{
			Search: r.Filters.Search, GroupID: r.Filters.GroupID, Enabled: r.Filters.Enabled, Severity: r.Filters.Severity,
			ConditionType: r.Filters.ConditionType, TargetPath: r.Filters.TargetPath, Mode: r.Filters.Mode,
		},
		ExcludePolicyIDs: r.ExcludePolicyIDs,
	}
}

func parseAlarmPolicyListFilter(r *http.Request) (service.AlarmPolicyListFilter, error) {
	query := r.URL.Query()
	page, err := parseOptionalInt(query.Get("page"), 1, "page")
	if err != nil {
		return service.AlarmPolicyListFilter{}, err
	}
	pageSize, err := parseOptionalInt(firstNonEmpty(query.Get("pageSize"), query.Get("limit")), 20, "pageSize")
	if err != nil {
		return service.AlarmPolicyListFilter{}, err
	}
	enabled, err := parseOptionalBool(query.Get("enabled"))
	if err != nil {
		return service.AlarmPolicyListFilter{}, err
	}
	return service.AlarmPolicyListFilter{
		Search: query.Get("search"), GroupID: nullablePolicyQueryString(query.Get("groupId")), Enabled: enabled,
		Severity: query.Get("severity"), ConditionType: query.Get("conditionType"), TargetPath: query.Get("targetPath"),
		Mode: query.Get("mode"), Page: page, PageSize: pageSize,
	}, nil
}

func decodeCreateAlarmPolicyInput(r *http.Request) (service.CreateAlarmPolicyInput, error) {
	var request struct {
		GroupID           *string                  `json:"groupId"`
		Name              string                   `json:"name"`
		Description       *string                  `json:"description"`
		Mode              string                   `json:"mode"`
		Targets           []service.AlarmTargetRef `json:"targets"`
		Inputs            []service.AlarmInputRef  `json:"inputs"`
		DerivedExpression string                   `json:"derivedExpression"`
		Conditions        []service.AlarmCondition `json:"conditions"`
		Suppression       map[string]any           `json:"suppression"`
		MessageTemplate   string                   `json:"messageTemplate"`
		IsEnabled         *bool                    `json:"isEnabled"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return service.CreateAlarmPolicyInput{}, err
	}
	return service.CreateAlarmPolicyInput{
		GroupID: request.GroupID, Name: request.Name, Description: request.Description, Mode: request.Mode,
		Targets: request.Targets, Inputs: request.Inputs, DerivedExpression: request.DerivedExpression,
		Conditions: request.Conditions, Suppression: request.Suppression, MessageTemplate: request.MessageTemplate,
		IsEnabled: request.IsEnabled,
	}, nil
}

func decodeUpdateAlarmPolicyInput(r *http.Request) (service.UpdateAlarmPolicyInput, error) {
	var raw map[string]json.RawMessage
	if err := decodeJSONBody(r, &raw); err != nil {
		return service.UpdateAlarmPolicyInput{}, err
	}
	var input service.UpdateAlarmPolicyInput
	if rawValue, ok := raw["groupId"]; ok {
		input.HasGroupID = true
		if string(rawValue) != "null" {
			var value string
			if err := json.Unmarshal(rawValue, &value); err != nil {
				return service.UpdateAlarmPolicyInput{}, invalidAlarmField("groupId")
			}
			input.GroupID = &value
		}
	}
	if err := decodePolicyRaw(raw, "name", &input.Name); err != nil {
		return service.UpdateAlarmPolicyInput{}, err
	}
	if err := decodePolicyRaw(raw, "description", &input.Description); err != nil {
		return service.UpdateAlarmPolicyInput{}, err
	}
	if err := decodePolicyRaw(raw, "mode", &input.Mode); err != nil {
		return service.UpdateAlarmPolicyInput{}, err
	}
	if rawValue, ok := raw["targets"]; ok {
		input.HasTargets = true
		if err := json.Unmarshal(rawValue, &input.Targets); err != nil {
			return service.UpdateAlarmPolicyInput{}, invalidAlarmField("targets")
		}
	}
	if rawValue, ok := raw["inputs"]; ok {
		input.HasInputs = true
		if err := json.Unmarshal(rawValue, &input.Inputs); err != nil {
			return service.UpdateAlarmPolicyInput{}, invalidAlarmField("inputs")
		}
	}
	if err := decodePolicyRaw(raw, "derivedExpression", &input.DerivedExpression); err != nil {
		return service.UpdateAlarmPolicyInput{}, err
	}
	if rawValue, ok := raw["conditions"]; ok {
		input.HasConditions = true
		if err := json.Unmarshal(rawValue, &input.Conditions); err != nil {
			return service.UpdateAlarmPolicyInput{}, invalidAlarmField("conditions")
		}
	}
	if rawValue, ok := raw["suppression"]; ok {
		input.HasSuppression = true
		if err := json.Unmarshal(rawValue, &input.Suppression); err != nil {
			return service.UpdateAlarmPolicyInput{}, invalidAlarmField("suppression")
		}
	}
	if err := decodePolicyRaw(raw, "messageTemplate", &input.MessageTemplate); err != nil {
		return service.UpdateAlarmPolicyInput{}, err
	}
	if rawValue, ok := raw["isEnabled"]; ok {
		var value bool
		if err := json.Unmarshal(rawValue, &value); err != nil {
			return service.UpdateAlarmPolicyInput{}, invalidAlarmField("isEnabled")
		}
		input.IsEnabled = &value
	}
	return input, nil
}

func decodePolicyRaw(raw map[string]json.RawMessage, field string, target **string) error {
	rawValue, ok := raw[field]
	if !ok {
		return nil
	}
	if string(rawValue) == "null" {
		empty := ""
		*target = &empty
		return nil
	}
	var value string
	if err := json.Unmarshal(rawValue, &value); err != nil {
		return invalidAlarmField(field)
	}
	*target = &value
	return nil
}

func nullablePolicyQueryString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
