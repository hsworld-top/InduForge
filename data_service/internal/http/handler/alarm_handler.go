package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

const maxAlarmImportFileSize = 20 << 20

type AlarmHandler struct {
	service     *service.AlarmSettingsService
	itemService *service.AlarmItemService
}

func NewAlarmHandler(alarmService *service.AlarmSettingsService, itemServices ...*service.AlarmItemService) *AlarmHandler {
	var itemService *service.AlarmItemService
	if len(itemServices) > 0 {
		itemService = itemServices[0]
	}
	return &AlarmHandler{service: alarmService, itemService: itemService}
}

func (h *AlarmHandler) ListGroups(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	if r.URL.Query().Get("view") == "tree" {
		result, treeErr := h.service.ListGroupTree(r.Context(), claims, r.PathValue("projectId"))
		if treeErr != nil {
			return normalizeRepresentativeHandlerError(treeErr)
		}
		response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
		return nil
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

func (h *AlarmHandler) CreateGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.SaveAlarmGroupInput
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

func (h *AlarmHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.SaveAlarmGroupInput
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

func (h *AlarmHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) error {
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

func (h *AlarmHandler) List(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	filter, err := parseAlarmItemFilter(r)
	if err != nil {
		return err
	}
	result, err := h.itemService.List(r.Context(), claims, r.PathValue("projectId"), filter)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmHandler) Get(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.itemService.Get(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmHandler) Create(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.SaveAlarmItemInput
	if err = decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.itemService.Create(r.Context(), claims, r.PathValue("projectId"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmHandler) Update(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.SaveAlarmItemInput
	if err = decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.itemService.Update(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmHandler) ToggleEnabled(w http.ResponseWriter, r *http.Request) error {
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
	result, err := h.itemService.SetEnabled(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"), input.IsEnabled)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmHandler) Delete(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	if err = h.itemService.Delete(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id")); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}

func (h *AlarmHandler) ValidateDraft(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.SaveAlarmItemInput
	if err = decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.itemService.ValidateDraft(r.Context(), claims, r.PathValue("projectId"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmHandler) TestDraft(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input struct {
		Draft   service.SaveAlarmItemInput `json:"draft"`
		Samples []service.AlarmTrialSample `json:"samples"`
	}
	if err = decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.itemService.TestDraft(r.Context(), claims, r.PathValue("projectId"), input.Draft, input.Samples)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmHandler) Contract(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.itemService.Contract(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmHandler) DatapointSummary(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.itemService.DatapointSummary(r.Context(), claims, r.PathValue("projectId"), r.PathValue("datapointId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmHandler) GetSettings(w http.ResponseWriter, r *http.Request) error {
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

func (h *AlarmHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) error {
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

func (h *AlarmHandler) GetLevelSettings(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.GetLevelSettings(r.Context(), claims, r.PathValue("projectId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmHandler) UpdateLevelSettings(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.SaveAlarmLevelSettingsInput
	if err = decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.service.UpdateLevelSettings(r.Context(), claims, r.PathValue("projectId"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmHandler) GetHistorySettings(w http.ResponseWriter, r *http.Request) error {
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

func (h *AlarmHandler) UpdateHistorySettings(w http.ResponseWriter, r *http.Request) error {
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

func (h *AlarmHandler) ListChannels(w http.ResponseWriter, r *http.Request) error {
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

func (h *AlarmHandler) CreateChannel(w http.ResponseWriter, r *http.Request) error {
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

func (h *AlarmHandler) UpdateChannel(w http.ResponseWriter, r *http.Request) error {
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

func (h *AlarmHandler) DeleteChannel(w http.ResponseWriter, r *http.Request) error {
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

func (h *AlarmHandler) TestChannel(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.TestChannel(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmHandler) SyncConfig(w http.ResponseWriter, r *http.Request) error {
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

func (h *AlarmHandler) BatchCreate(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.BatchCreateAlarmItemsInput
	if err = decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.itemService.BatchCreate(r.Context(), claims, r.PathValue("projectId"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmHandler) ValidateBatchCreate(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.BatchCreateAlarmItemsInput
	if err = decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.itemService.ValidateBatchCreate(r.Context(), claims, r.PathValue("projectId"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmHandler) SavePresetConfiguration(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.SavePresetAlarmConfigurationInput
	if err = decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.itemService.SavePresetConfiguration(r.Context(), claims, r.PathValue("projectId"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmHandler) ValidatePresetConfiguration(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.SavePresetAlarmConfigurationInput
	if err = decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.itemService.ValidatePresetConfiguration(r.Context(), claims, r.PathValue("projectId"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmHandler) BatchUpdate(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.BatchUpdateAlarmItemsInput
	if err = decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.itemService.BatchUpdate(r.Context(), claims, r.PathValue("projectId"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmHandler) BatchDelete(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.BatchDeleteAlarmItemsInput
	if err = decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.itemService.BatchDelete(r.Context(), claims, r.PathValue("projectId"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmHandler) Export(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input struct {
		Selection service.AlarmItemSelection `json:"selection"`
	}
	if err = decodeJSONBody(r, &input); err != nil {
		return err
	}
	workbook, err := h.itemService.ExportWorkbook(r.Context(), claims, r.PathValue("projectId"), input.Selection)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	writeAlarmWorkbook(w, workbook)
	return nil
}

func (h *AlarmHandler) ImportTemplate(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	workbook, err := h.itemService.BuildImportTemplate(r.Context(), claims, r.PathValue("projectId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	writeAlarmWorkbook(w, workbook)
	return nil
}

func (h *AlarmHandler) ImportPreview(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	fileName, content, err := readAlarmImportFile(w, r)
	if err != nil {
		return err
	}
	result, err := h.itemService.PreviewImport(r.Context(), claims, r.PathValue("projectId"), fileName, content)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *AlarmHandler) ImportErrorWorkbook(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	fileName, content, err := readAlarmImportFile(w, r)
	if err != nil {
		return err
	}
	workbook, err := h.itemService.BuildImportErrorWorkbook(r.Context(), claims, r.PathValue("projectId"), fileName, content)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	writeAlarmWorkbook(w, workbook)
	return nil
}

func (h *AlarmHandler) ImportApply(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	fileName, content, err := readAlarmImportFile(w, r)
	if err != nil {
		return err
	}
	warningKeys := []string{}
	if raw := strings.TrimSpace(r.FormValue("acknowledgedWarningKeys")); raw != "" {
		if err = json.Unmarshal([]byte(raw), &warningKeys); err != nil {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "已确认警告键格式无效")
		}
	}
	result, err := h.itemService.ApplyImport(r.Context(), claims, r.PathValue("projectId"), fileName, content, r.FormValue("digest"), warningKeys)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func readAlarmImportFile(w http.ResponseWriter, r *http.Request) (string, []byte, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAlarmImportFileSize)
	if err := r.ParseMultipartForm(maxAlarmImportFileSize); err != nil {
		return "", nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "导入文件超过 20MB 或表单格式无效", err)
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return "", nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "缺少导入文件", err)
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		return "", nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "读取导入文件失败", err)
	}
	return header.Filename, content, nil
}

func writeAlarmWorkbook(w http.ResponseWriter, workbook service.AlarmItemWorkbook) {
	w.Header().Set("Content-Type", workbook.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename*=UTF-8''%s`, url.PathEscape(workbook.FileName)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(workbook.Content)
}

func parseAlarmItemFilter(r *http.Request) (service.AlarmItemListFilter, error) {
	page, err := alarmQueryInt(r, "page", 1)
	if err != nil {
		return service.AlarmItemListFilter{}, err
	}
	pageSize, err := alarmQueryInt(r, "pageSize", 20)
	if err != nil {
		return service.AlarmItemListFilter{}, err
	}
	var groupID *string
	if value := strings.TrimSpace(r.URL.Query().Get("groupId")); value != "" {
		groupID = &value
	}
	var enabled *bool
	if value := strings.TrimSpace(r.URL.Query().Get("enabled")); value != "" {
		parsed, parseErr := strconv.ParseBool(value)
		if parseErr != nil {
			return service.AlarmItemListFilter{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "enabled 格式无效")
		}
		enabled = &parsed
	}
	return service.AlarmItemListFilter{Search: r.URL.Query().Get("search"), GroupID: groupID, Enabled: enabled, Severity: r.URL.Query().Get("severity"), AlarmType: r.URL.Query().Get("alarmType"), Mode: r.URL.Query().Get("mode"), DatapointID: r.URL.Query().Get("datapointId"), Page: page, PageSize: pageSize}, nil
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
