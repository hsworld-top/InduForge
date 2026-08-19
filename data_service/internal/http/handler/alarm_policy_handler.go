package handler

import (
	"net/http"
	"strconv"
	"strings"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

type AlarmPolicyHandler struct{ service *service.AlarmPolicyService }

func NewAlarmPolicyHandler(alarmService *service.AlarmPolicyService) *AlarmPolicyHandler {
	return &AlarmPolicyHandler{service: alarmService}
}

func (h *AlarmPolicyHandler) ListGroups(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	page, err := alarmQueryInt(r, "page", 1)
	if err != nil {
		return err
	}
	pageSize, err := alarmQueryInt(r, "pageSize", 50)
	if err != nil {
		return err
	}
	var parentID *string
	if value := strings.TrimSpace(r.URL.Query().Get("parentId")); value != "" {
		parentID = &value
	}
	result, err := h.service.ListGroups(r.Context(), claims, r.PathValue("projectId"), r.URL.Query().Get("search"), parentID, page, pageSize)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmPolicyHandler) GroupTree(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.ListAllGroups(r.Context(), claims, r.PathValue("projectId"))
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
	var input service.SaveAlarmPolicyGroupInput
	if err = decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.service.CreateGroup(r.Context(), claims, r.PathValue("projectId"), input)
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
	var input service.SaveAlarmPolicyGroupInput
	if err = decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.service.UpdateGroup(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"), input)
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
	if err = h.service.DeleteGroup(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id")); err != nil {
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
	filter, err := parseAlarmPolicyFilter(r)
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

func (h *AlarmPolicyHandler) Create(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.SaveAlarmPolicyInput
	if err = decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.service.Create(r.Context(), claims, r.PathValue("projectId"), input)
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
	var input service.SaveAlarmPolicyInput
	if err = decodeJSONBody(r, &input); err != nil {
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
	var input struct {
		IsEnabled bool `json:"isEnabled"`
	}
	if err = decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.service.SetEnabled(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"), input.IsEnabled)
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
	if err = h.service.Delete(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id")); err != nil {
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
	var input service.SaveAlarmPolicyInput
	if err = decodeJSONBody(r, &input); err != nil {
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
	var input struct {
		Value   any            `json:"value"`
		Context map[string]any `json:"context"`
	}
	if err = decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.service.Test(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"), input.Value, input.Context)
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

func (h *AlarmPolicyHandler) DatapointSummary(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.DatapointSummary(r.Context(), claims, r.PathValue("projectId"), r.PathValue("datapointId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
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
	var input service.SaveAlarmProjectSettingsInput
	if err = decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.service.UpdateSettings(r.Context(), claims, r.PathValue("projectId"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmPolicyHandler) GetHistorySettings(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.GetHistorySettings(r.Context(), claims, r.PathValue("projectId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmPolicyHandler) UpdateHistorySettings(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.SaveAlarmHistorySettingsInput
	if err = decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.service.UpdateHistorySettings(r.Context(), claims, r.PathValue("projectId"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmPolicyHandler) ListChannels(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.ListChannels(r.Context(), claims, r.PathValue("projectId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmPolicyHandler) CreateChannel(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.SaveAlarmNotificationChannelInput
	if err = decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.service.CreateChannel(r.Context(), claims, r.PathValue("projectId"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmPolicyHandler) UpdateChannel(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.SaveAlarmNotificationChannelInput
	if err = decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.service.UpdateChannel(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmPolicyHandler) DeleteChannel(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	if err = h.service.DeleteChannel(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id")); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}

func (h *AlarmPolicyHandler) SyncConfig(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.AlarmConfigSyncInput
	if err = decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.service.SyncConfig(r.Context(), claims, r.PathValue("projectId"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func parseAlarmPolicyFilter(r *http.Request) (service.AlarmPolicyListFilter, error) {
	page, err := alarmQueryInt(r, "page", 1)
	if err != nil {
		return service.AlarmPolicyListFilter{}, err
	}
	pageSize, err := alarmQueryInt(r, "pageSize", 20)
	if err != nil {
		return service.AlarmPolicyListFilter{}, err
	}
	var groupID *string
	if value := strings.TrimSpace(r.URL.Query().Get("groupId")); value != "" {
		groupID = &value
	}
	var enabled *bool
	if value := strings.TrimSpace(r.URL.Query().Get("enabled")); value != "" {
		parsed, parseErr := strconv.ParseBool(value)
		if parseErr != nil {
			return service.AlarmPolicyListFilter{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "enabled 格式无效")
		}
		enabled = &parsed
	}
	return service.AlarmPolicyListFilter{Search: r.URL.Query().Get("search"), GroupID: groupID, Enabled: enabled, Severity: r.URL.Query().Get("severity"), ConditionKind: r.URL.Query().Get("conditionKind"), Mode: r.URL.Query().Get("mode"), DatapointID: r.URL.Query().Get("datapointId"), Page: page, PageSize: pageSize}, nil
}

func alarmQueryInt(r *http.Request, key string, fallback int) (int, error) {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, key+" 必须为正整数")
	}
	return parsed, nil
}
