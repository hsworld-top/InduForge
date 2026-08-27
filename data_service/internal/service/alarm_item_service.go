package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/indu-forge/data_service/internal/auth"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

var alarmItemModes = map[string]bool{"point": true, "derived": true}
var alarmPresetSlots = map[string]string{
	"limit": "threshold", "rate": "rate_of_change", "deviation": "deviation",
	"state": "state", "transition": "transition", "text": "text_match",
	"quality": "quality", "stale": "stale", "offline": "offline",
}

type AlarmItemInput struct {
	ID          string `json:"id"`
	DatapointID string `json:"datapointId"`
	Path        string `json:"path"`
	Name        string `json:"name"`
	DataType    string `json:"dataType"`
	InputKey    string `json:"inputKey"`
}

type AlarmItem struct {
	ID                string                    `json:"id"`
	ProjectID         string                    `json:"projectId"`
	DisplayName       string                    `json:"displayName"`
	Mode              string                    `json:"mode"`
	AlarmType         string                    `json:"alarmType"`
	PresetSlot        *string                   `json:"presetSlot"`
	EvaluationMode    string                    `json:"evaluationMode"`
	DerivedExpression string                    `json:"derivedExpression"`
	DatapointID       string                    `json:"datapointId"`
	Path              string                    `json:"path"`
	DatapointName     string                    `json:"datapointName"`
	DataType          string                    `json:"dataType"`
	GroupID           *string                   `json:"groupId"`
	GroupName         *string                   `json:"groupName"`
	Description       *string                   `json:"description"`
	Inputs            []AlarmItemInput          `json:"inputs"`
	Conditions        []AlarmCondition          `json:"conditions"`
	Notification      AlarmNotificationSettings `json:"notification"`
	IsEnabled         bool                      `json:"isEnabled"`
	Revision          int64                     `json:"revision"`
	Contract          map[string]any            `json:"contract"`
	CreatedAt         time.Time                 `json:"createdAt"`
	UpdatedAt         time.Time                 `json:"updatedAt"`
}

type AlarmItemListFilter struct {
	Search, Severity, AlarmType, Mode, DatapointID string
	GroupID                                        *string
	Enabled                                        *bool
	Page, PageSize                                 int
}

type AlarmItemListResult struct {
	List       []AlarmItem `json:"list"`
	Pagination Pagination  `json:"pagination"`
}

type SaveAlarmItemInput struct {
	ItemID                  string                    `json:"itemId,omitempty"`
	DatapointID             string                    `json:"datapointId,omitempty"`
	DisplayName             string                    `json:"displayName"`
	PresetSlot              *string                   `json:"presetSlot,omitempty"`
	GroupID                 *string                   `json:"groupId"`
	Description             *string                   `json:"description"`
	Mode                    string                    `json:"mode"`
	EvaluationMode          string                    `json:"evaluationMode"`
	Inputs                  []AlarmItemInput          `json:"inputs"`
	DerivedExpression       string                    `json:"derivedExpression"`
	Conditions              []AlarmCondition          `json:"conditions"`
	Notification            AlarmNotificationSettings `json:"notification"`
	IsEnabled               *bool                     `json:"isEnabled"`
	Revision                int64                     `json:"revision,omitempty"`
	AcknowledgedWarningKeys []string                  `json:"acknowledgedWarningKeys"`
}

type BatchCreateAlarmItemsInput struct {
	DatapointIDs []string           `json:"datapointIds"`
	Draft        SaveAlarmItemInput `json:"draft"`
}

type SavePresetAlarmConfigurationInput struct {
	DatapointIDs []string             `json:"datapointIds"`
	Drafts       []SaveAlarmItemInput `json:"drafts"`
}

type BatchAlarmItemResult struct {
	AffectedCount int      `json:"affectedCount"`
	ItemIDs       []string `json:"itemIds,omitempty"`
}

type AlarmItemSelection struct {
	IDs    []string             `json:"ids"`
	Filter *AlarmItemListFilter `json:"filter,omitempty"`
}

type BatchAlarmItemPatch struct {
	IsEnabled    *bool                      `json:"isEnabled,omitempty"`
	GroupID      *string                    `json:"groupId"`
	Notification *AlarmNotificationSettings `json:"notification,omitempty"`
	Conditions   []AlarmCondition           `json:"conditions,omitempty"`
}

type BatchUpdateAlarmItemsInput struct {
	Selection               AlarmItemSelection  `json:"selection"`
	Fields                  []string            `json:"fields"`
	Patch                   BatchAlarmItemPatch `json:"patch"`
	AcknowledgedWarningKeys []string            `json:"acknowledgedWarningKeys"`
}

type BatchDeleteAlarmItemsInput struct {
	Selection AlarmItemSelection `json:"selection"`
}

type AlarmDraftIssue struct {
	Type               string `json:"type"`
	Message            string `json:"message"`
	DatapointID        string `json:"datapointId,omitempty"`
	DatapointName      string `json:"datapointName,omitempty"`
	ConflictID         string `json:"conflictAlarmItemId,omitempty"`
	ConflictName       string `json:"conflictAlarmItemName,omitempty"`
	AffectedCount      int    `json:"affectedCount,omitempty"`
	AcknowledgementKey string `json:"ackKey,omitempty"`
}

type AlarmDraftValidation struct {
	Valid    bool              `json:"valid"`
	Errors   []AlarmDraftIssue `json:"errors"`
	Warnings []AlarmDraftIssue `json:"warnings"`
}

type AlarmItemTrialResult struct {
	Triggered         bool             `json:"triggered"`
	State             string           `json:"state"`
	SelectedCondition *AlarmCondition  `json:"selectedCondition,omitempty"`
	Steps             []AlarmTrialStep `json:"steps"`
	Message           string           `json:"message,omitempty"`
}

type AlarmTrialSample struct {
	ObservedAt      time.Time      `json:"observedAt"`
	SourceTimestamp *time.Time     `json:"sourceTimestamp"`
	Value           any            `json:"value"`
	Quality         string         `json:"quality"`
	Offline         bool           `json:"offline"`
	Inputs          map[string]any `json:"inputs,omitempty"`
}

type AlarmTrialStep struct {
	ObservedAt              time.Time       `json:"observedAt"`
	SourceTimestamp         *time.Time      `json:"sourceTimestamp,omitempty"`
	Value                   any             `json:"value"`
	Inputs                  map[string]any  `json:"inputs,omitempty"`
	EvaluationState         string          `json:"evaluationState"`
	Reason                  string          `json:"reason"`
	State                   string          `json:"state"`
	ActiveCondition         *AlarmCondition `json:"activeCondition,omitempty"`
	CandidateCondition      *AlarmCondition `json:"candidateCondition,omitempty"`
	CandidateElapsedMS      int64           `json:"candidateElapsedMs"`
	RemainingTriggerDelayMS int64           `json:"remainingTriggerDelayMs"`
	ClearElapsedMS          int64           `json:"clearElapsedMs"`
	RemainingClearDelayMS   int64           `json:"remainingClearDelayMs"`
	CalculatedRate          *float64        `json:"calculatedRate,omitempty"`
}

type AlarmItemDatapointSummary struct {
	DatapointID string      `json:"datapointId"`
	Items       []AlarmItem `json:"items"`
	Count       int         `json:"count"`
}

type AlarmItemService struct {
	repository *repository.AlarmRepository
	datapoints *repository.DataPointRepository
}

func NewAlarmItemService(repo *repository.AlarmRepository, datapoints *repository.DataPointRepository) *AlarmItemService {
	return &AlarmItemService{repository: repo, datapoints: datapoints}
}

func (s *AlarmItemService) List(ctx context.Context, claims *auth.Claims, projectID string, filter AlarmItemListFilter) (*AlarmItemListResult, error) {
	if err := s.readAccess(claims, projectID); err != nil {
		return nil, err
	}
	if filter.Mode != "" && !alarmItemModes[filter.Mode] {
		return nil, badAlarm("报警模式筛选无效")
	}
	records, total, err := s.repository.ListAlarmItems(ctx, projectID, repository.AlarmItemListFilter{Search: filter.Search, Severity: filter.Severity, AlarmType: filter.AlarmType, Mode: filter.Mode, DatapointID: filter.DatapointID, GroupID: filter.GroupID, Enabled: filter.Enabled, Page: filter.Page, PageSize: filter.PageSize})
	if err != nil {
		return nil, err
	}
	items := make([]AlarmItem, 0, len(records))
	for _, record := range records {
		items = append(items, toAlarmItem(record))
	}
	return &AlarmItemListResult{List: items, Pagination: Pagination{Page: positive(filter.Page, 1), PageSize: positive(filter.PageSize, 20), Total: total}}, nil
}

func (s *AlarmItemService) Get(ctx context.Context, claims *auth.Claims, projectID, id string) (*AlarmItem, error) {
	if err := s.readAccess(claims, projectID); err != nil {
		return nil, err
	}
	record, err := s.repository.GetAlarmItem(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	item := toAlarmItem(*record)
	return &item, nil
}

func (s *AlarmItemService) Create(ctx context.Context, claims *auth.Claims, projectID string, input SaveAlarmItemInput) (*AlarmItem, error) {
	return s.save(ctx, claims, projectID, "", input)
}
func (s *AlarmItemService) Update(ctx context.Context, claims *auth.Claims, projectID, id string, input SaveAlarmItemInput) (*AlarmItem, error) {
	return s.save(ctx, claims, projectID, id, input)
}
func (s *AlarmItemService) save(ctx context.Context, claims *auth.Claims, projectID, id string, input SaveAlarmItemInput) (*AlarmItem, error) {
	if err := s.writeAccess(claims, projectID); err != nil {
		return nil, err
	}
	normalized, params, validation, err := s.normalize(ctx, projectID, id, input)
	if err != nil {
		return nil, err
	}
	if err = acknowledgeAlarmWarnings(normalized, validation); err != nil {
		return nil, err
	}
	params.UserID = claims.UserID
	record, err := s.repository.SaveAlarmItem(ctx, params)
	if err != nil {
		return nil, err
	}
	item := toAlarmItem(*record)
	return &item, nil
}

func (s *AlarmItemService) BatchCreate(ctx context.Context, claims *auth.Claims, projectID string, input BatchCreateAlarmItemsInput) (*BatchAlarmItemResult, error) {
	if err := s.writeAccess(claims, projectID); err != nil {
		return nil, err
	}
	ids := alarmUniqueStrings(input.DatapointIDs)
	if len(ids) == 0 {
		return nil, badAlarm("至少选择一个数据点")
	}
	paramsList := make([]repository.SaveAlarmItemParams, 0, len(ids))
	for _, datapointID := range ids {
		draft := input.Draft
		draft.Mode = "point"
		draft.DatapointID = datapointID
		normalized, params, validation, err := s.normalize(ctx, projectID, "", draft)
		if err != nil {
			return nil, err
		}
		if err = acknowledgeAlarmWarnings(normalized, validation); err != nil {
			return nil, err
		}
		paramsList = append(paramsList, params)
	}
	created, err := s.repository.BatchCreateAlarmItems(ctx, projectID, claims.UserID, paramsList)
	if err != nil {
		return nil, err
	}
	return &BatchAlarmItemResult{AffectedCount: len(created), ItemIDs: created}, nil
}

// SavePresetConfiguration 将点位式默认配置展开为稳定报警项，并在一个事务内完成新增、更新和移除。
// preset_slot 仅用于默认配置；完整配置创建的高级报警项不设置该字段，因此允许同类型多条。
func (s *AlarmItemService) SavePresetConfiguration(ctx context.Context, claims *auth.Claims, projectID string, input SavePresetAlarmConfigurationInput) (*BatchAlarmItemResult, error) {
	if err := s.writeAccess(claims, projectID); err != nil {
		return nil, err
	}
	ids, paramsList, validation, err := s.preparePresetConfiguration(ctx, projectID, input)
	if err != nil {
		return nil, err
	}
	acknowledged := map[string]bool{}
	for _, draft := range input.Drafts {
		for _, key := range draft.AcknowledgedWarningKeys {
			acknowledged[key] = true
		}
	}
	for _, warning := range validation.Warnings {
		if warning.AcknowledgementKey != "" && !acknowledged[warning.AcknowledgementKey] {
			return nil, badAlarm("报警条件存在重叠，请确认后再保存")
		}
	}
	savedIDs, err := s.repository.ReplacePresetAlarmItems(ctx, projectID, claims.UserID, ids, paramsList)
	if err != nil {
		return nil, err
	}
	return &BatchAlarmItemResult{AffectedCount: len(savedIDs), ItemIDs: savedIDs}, nil
}

func (s *AlarmItemService) ValidatePresetConfiguration(ctx context.Context, claims *auth.Claims, projectID string, input SavePresetAlarmConfigurationInput) (*AlarmDraftValidation, error) {
	if err := s.readAccess(claims, projectID); err != nil {
		return nil, err
	}
	_, _, validation, err := s.preparePresetConfiguration(ctx, projectID, input)
	return validation, err
}

func (s *AlarmItemService) preparePresetConfiguration(ctx context.Context, projectID string, input SavePresetAlarmConfigurationInput) ([]string, []repository.SaveAlarmItemParams, *AlarmDraftValidation, error) {
	ids := alarmUniqueStrings(input.DatapointIDs)
	result := &AlarmDraftValidation{Errors: []AlarmDraftIssue{}, Warnings: []AlarmDraftIssue{}}
	if len(ids) == 0 {
		return nil, nil, result, badAlarm("至少选择一个数据点")
	}
	if len(input.Drafts) == 0 {
		return nil, nil, result, badAlarm("请至少启用一项报警")
	}
	existing, err := s.repository.ListPresetAlarmItemsForDatapoints(ctx, projectID, ids)
	if err != nil {
		return nil, nil, result, err
	}
	byPointSlot := map[string]repository.AlarmItemRecord{}
	for _, item := range existing {
		if item.DatapointID != nil && item.PresetSlot != nil {
			byPointSlot[*item.DatapointID+":"+*item.PresetSlot] = item
		}
	}
	seenSlots := map[string]bool{}
	for _, draft := range input.Drafts {
		slot := strings.TrimSpace(alarmStringPointerValue(draft.PresetSlot))
		if _, ok := alarmPresetSlots[slot]; !ok {
			return nil, nil, result, badAlarm("默认报警槽位不受支持")
		}
		if seenSlots[slot] {
			return nil, nil, result, badAlarm("同一种默认报警配置只能提交一次")
		}
		seenSlots[slot] = true
	}
	paramsList := make([]repository.SaveAlarmItemParams, 0, len(ids)*len(input.Drafts))
	for _, datapointID := range ids {
		for _, source := range input.Drafts {
			draft := source
			draft.Mode, draft.DatapointID = "point", datapointID
			slot := strings.TrimSpace(alarmStringPointerValue(draft.PresetSlot))
			resetAlarmItemChildIDsForCreate(&draft)
			itemID := ""
			if current, ok := byPointSlot[datapointID+":"+slot]; ok {
				itemID, draft.Revision = current.ID, current.Revision
				// 批量编辑时每个点保留自己的名称；单点编辑则使用表单值，清空可恢复自动命名。
				if len(ids) > 1 {
					draft.DisplayName = current.DisplayName
				}
			}
			_, params, validation, normalizeErr := s.normalize(ctx, projectID, itemID, draft)
			if normalizeErr != nil {
				return nil, nil, result, normalizeErr
			}
			if params.AlarmType != alarmPresetSlots[slot] {
				return nil, nil, result, badAlarm("默认报警槽位与条件类型不匹配")
			}
			params.PresetSlot = &slot
			params.Contract["presetSlot"] = slot
			paramsList = append(paramsList, params)
			result.Errors = append(result.Errors, validation.Errors...)
			result.Warnings = append(result.Warnings, validation.Warnings...)
		}
	}
	result.Valid = len(result.Errors) == 0
	if !result.Valid {
		return ids, paramsList, result, badAlarm(result.Errors[0].Message)
	}
	return ids, paramsList, result, nil
}

func alarmStringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// ValidateBatchCreate 对多点创建草稿逐点执行与最终保存完全相同的校验，避免用户在提交阶段才发现后续点位冲突。
func (s *AlarmItemService) ValidateBatchCreate(ctx context.Context, claims *auth.Claims, projectID string, input BatchCreateAlarmItemsInput) (*AlarmDraftValidation, error) {
	if err := s.readAccess(claims, projectID); err != nil {
		return nil, err
	}
	ids := alarmUniqueStrings(input.DatapointIDs)
	if len(ids) == 0 {
		return &AlarmDraftValidation{Valid: false, Errors: []AlarmDraftIssue{{Type: "invalid", Message: "至少选择一个数据点"}}, Warnings: []AlarmDraftIssue{}}, nil
	}
	result := &AlarmDraftValidation{Errors: []AlarmDraftIssue{}, Warnings: []AlarmDraftIssue{}}
	for _, datapointID := range ids {
		draft := input.Draft
		draft.Mode = "point"
		draft.DatapointID = datapointID
		_, _, validation, err := s.normalize(ctx, projectID, "", draft)
		if err != nil {
			result.Errors = append(result.Errors, AlarmDraftIssue{Type: "invalid", Message: err.Error(), DatapointID: datapointID})
			continue
		}
		result.Errors = append(result.Errors, validation.Errors...)
		result.Warnings = append(result.Warnings, validation.Warnings...)
	}
	result.Valid = len(result.Errors) == 0
	return result, nil
}

func (s *AlarmItemService) BatchUpdate(ctx context.Context, claims *auth.Claims, projectID string, input BatchUpdateAlarmItemsInput) (*BatchAlarmItemResult, error) {
	if err := s.writeAccess(claims, projectID); err != nil {
		return nil, err
	}
	fields := map[string]bool{}
	for _, field := range input.Fields {
		field = strings.TrimSpace(field)
		if !map[string]bool{"isEnabled": true, "groupId": true, "notification": true, "conditions": true}[field] {
			return nil, badAlarm("批量修改字段不受支持")
		}
		fields[field] = true
	}
	if len(fields) == 0 {
		return nil, badAlarm("至少选择一个要修改的字段")
	}
	records, err := s.resolveSelection(ctx, projectID, input.Selection)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, badAlarm("当前选择没有可修改的报警项")
	}
	if fields["isEnabled"] && input.Patch.IsEnabled == nil {
		return nil, badAlarm("启停字段不能为空")
	}
	if fields["conditions"] {
		base := records[0]
		for _, record := range records[1:] {
			if record.Mode != base.Mode || record.AlarmType != base.AlarmType || record.EvaluationMode != base.EvaluationMode {
				return nil, badAlarm("只有相同模式和报警类型才能统一替换条件")
			}
		}
	}
	paramsList := make([]repository.SaveAlarmItemParams, 0, len(records))
	for _, record := range records {
		draft := alarmItemRecordToSaveInput(record)
		if fields["isEnabled"] {
			draft.IsEnabled = input.Patch.IsEnabled
		}
		if fields["groupId"] {
			draft.GroupID = normalizeID(input.Patch.GroupID)
		}
		if fields["notification"] {
			if input.Patch.Notification == nil {
				return nil, badAlarm("通知设置不能为空")
			}
			draft.Notification = *input.Patch.Notification
		}
		if fields["conditions"] {
			draft.Conditions = input.Patch.Conditions
		}
		draft.AcknowledgedWarningKeys = input.AcknowledgedWarningKeys
		normalized, params, validation, normalizeErr := s.normalize(ctx, projectID, record.ID, draft)
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		if normalizeErr = acknowledgeAlarmWarnings(normalized, validation); normalizeErr != nil {
			return nil, normalizeErr
		}
		params.UserID = claims.UserID
		paramsList = append(paramsList, params)
	}
	if err = s.repository.BatchSaveAlarmItems(ctx, projectID, claims.UserID, paramsList); err != nil {
		return nil, err
	}
	return &BatchAlarmItemResult{AffectedCount: len(paramsList)}, nil
}

func (s *AlarmItemService) BatchDelete(ctx context.Context, claims *auth.Claims, projectID string, input BatchDeleteAlarmItemsInput) (*BatchAlarmItemResult, error) {
	if err := s.writeAccess(claims, projectID); err != nil {
		return nil, err
	}
	records, err := s.resolveSelection(ctx, projectID, input.Selection)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.ID)
	}
	count, err := s.repository.BatchDeleteAlarmItems(ctx, projectID, ids)
	if err != nil {
		return nil, err
	}
	return &BatchAlarmItemResult{AffectedCount: count}, nil
}

func (s *AlarmItemService) resolveSelection(ctx context.Context, projectID string, selection AlarmItemSelection) ([]repository.AlarmItemRecord, error) {
	ids := alarmUniqueStrings(selection.IDs)
	if len(ids) > 0 && selection.Filter != nil {
		return nil, badAlarm("批量范围只能选择报警项 ID 或当前筛选结果")
	}
	if len(ids) == 0 && selection.Filter == nil {
		return nil, badAlarm("批量范围不能为空")
	}
	var filter *repository.AlarmItemListFilter
	if selection.Filter != nil {
		filter = &repository.AlarmItemListFilter{Search: selection.Filter.Search, Severity: selection.Filter.Severity, AlarmType: selection.Filter.AlarmType, Mode: selection.Filter.Mode, DatapointID: selection.Filter.DatapointID, GroupID: selection.Filter.GroupID, Enabled: selection.Filter.Enabled}
	}
	records, err := s.repository.ListAlarmItemsForBatch(ctx, projectID, ids, filter)
	if err != nil {
		return nil, err
	}
	if len(ids) > 0 && len(records) != len(ids) {
		return nil, badAlarm("部分报警项不存在或不属于当前工程")
	}
	return records, nil
}

func alarmItemRecordToSaveInput(record repository.AlarmItemRecord) SaveAlarmItemInput {
	conditions := make([]AlarmCondition, 0, len(record.Conditions))
	for _, condition := range record.Conditions {
		conditions = append(conditions, AlarmCondition{ID: condition.ID, Kind: condition.Kind, Operator: condition.Operator, Label: condition.Label, Severity: condition.Severity, Params: alarmCloneMap(condition.Params), TriggerDelayMS: condition.TriggerDelayMS, ClearDelayMS: condition.ClearDelayMS, Deadband: condition.Deadband})
	}
	inputs := make([]AlarmItemInput, 0, len(record.Inputs))
	for _, input := range record.Inputs {
		inputs = append(inputs, AlarmItemInput{ID: input.ID, DatapointID: input.DatapointID, Path: input.Path, Name: input.Name, DataType: input.DataType, InputKey: input.InputKey})
	}
	datapointID := ""
	if record.DatapointID != nil {
		datapointID = *record.DatapointID
	}
	return SaveAlarmItemInput{ItemID: record.ID, DatapointID: datapointID, DisplayName: record.DisplayName, GroupID: record.GroupID, Description: record.Description, Mode: record.Mode, EvaluationMode: record.EvaluationMode, Inputs: inputs, DerivedExpression: record.DerivedExpression, Conditions: conditions, Notification: AlarmNotificationSettings{Mode: record.NotificationMode, NotifyOnRaise: record.NotifyOnRaise, NotifyOnClear: record.NotifyOnClear, RepeatIntervalSeconds: record.RepeatIntervalSeconds, ChannelIDs: record.NotificationChannelIDs, MessageTemplate: record.MessageTemplate}, IsEnabled: &record.IsEnabled, Revision: record.Revision}
}

func acknowledgeAlarmWarnings(input SaveAlarmItemInput, validation *AlarmDraftValidation) error {
	if len(validation.Errors) > 0 {
		return alarmDraftConflict(validation.Errors[0].Message)
	}
	ack := map[string]bool{}
	for _, key := range input.AcknowledgedWarningKeys {
		ack[key] = true
	}
	for _, warning := range validation.Warnings {
		if !ack[warning.AcknowledgementKey] {
			return alarmDraftConflict("报警条件与已有报警重叠，请确认后再保存")
		}
	}
	return nil
}

func (s *AlarmItemService) ValidateDraft(ctx context.Context, claims *auth.Claims, projectID string, input SaveAlarmItemInput) (*AlarmDraftValidation, error) {
	if err := s.readAccess(claims, projectID); err != nil {
		return nil, err
	}
	_, _, validation, err := s.normalize(ctx, projectID, strings.TrimSpace(input.ItemID), input)
	if err != nil {
		return &AlarmDraftValidation{Valid: false, Errors: []AlarmDraftIssue{{Type: "invalid", Message: err.Error()}}, Warnings: []AlarmDraftIssue{}}, nil
	}
	validation.Valid = len(validation.Errors) == 0
	return validation, nil
}
func (s *AlarmItemService) SetEnabled(ctx context.Context, claims *auth.Claims, projectID, id string, enabled bool) (*AlarmItem, error) {
	if err := s.writeAccess(claims, projectID); err != nil {
		return nil, err
	}
	record, err := s.repository.SetAlarmItemEnabled(ctx, projectID, id, claims.UserID, enabled)
	if err != nil {
		return nil, err
	}
	item := toAlarmItem(*record)
	return &item, nil
}
func (s *AlarmItemService) Delete(ctx context.Context, claims *auth.Claims, projectID, id string) error {
	if err := s.writeAccess(claims, projectID); err != nil {
		return err
	}
	return s.repository.DeleteAlarmItem(ctx, projectID, id)
}
func (s *AlarmItemService) Contract(ctx context.Context, claims *auth.Claims, projectID, id string) (map[string]any, error) {
	item, err := s.Get(ctx, claims, projectID, id)
	if err != nil {
		return nil, err
	}
	return item.Contract, nil
}
func (s *AlarmItemService) DatapointSummary(ctx context.Context, claims *auth.Claims, projectID, datapointID string) (*AlarmItemDatapointSummary, error) {
	if err := s.readAccess(claims, projectID); err != nil {
		return nil, err
	}
	if _, err := s.datapoints.GetByProjectAndID(ctx, projectID, datapointID); err != nil {
		return nil, err
	}
	records, err := s.repository.ListAlarmItemsByDatapoint(ctx, projectID, datapointID)
	if err != nil {
		return nil, err
	}
	items := make([]AlarmItem, 0, len(records))
	for _, record := range records {
		items = append(items, toAlarmItem(record))
	}
	return &AlarmItemDatapointSummary{DatapointID: datapointID, Items: items, Count: len(items)}, nil
}

func (s *AlarmItemService) TestDraft(ctx context.Context, claims *auth.Claims, projectID string, input SaveAlarmItemInput, samples []AlarmTrialSample) (*AlarmItemTrialResult, error) {
	if err := s.readAccess(claims, projectID); err != nil {
		return nil, err
	}
	normalized, _, validation, err := s.normalize(ctx, projectID, strings.TrimSpace(input.ItemID), input)
	if err != nil {
		return nil, err
	}
	if len(validation.Errors) > 0 {
		return nil, badAlarm(validation.Errors[0].Message)
	}
	if len(samples) == 0 {
		return nil, badAlarm("报警试算至少需要一个带时间戳的样本")
	}
	return runAlarmTrial(normalized, samples)
}

func (s *AlarmItemService) normalize(ctx context.Context, projectID, id string, input SaveAlarmItemInput) (SaveAlarmItemInput, repository.SaveAlarmItemParams, *AlarmDraftValidation, error) {
	validation := &AlarmDraftValidation{Errors: []AlarmDraftIssue{}, Warnings: []AlarmDraftIssue{}}
	input.Mode = strings.TrimSpace(input.Mode)
	if !alarmItemModes[input.Mode] {
		return input, repository.SaveAlarmItemParams{}, validation, badAlarm("报警模式不受支持")
	}
	if input.EvaluationMode == "" {
		input.EvaluationMode = "single"
	}
	if input.EvaluationMode != "single" && input.EvaluationMode != "highest_matching" {
		return input, repository.SaveAlarmItemParams{}, validation, badAlarm("报警计算方式不受支持")
	}
	var err error
	input.Notification, err = s.normalizeNotification(ctx, projectID, input.Notification)
	if err != nil {
		return input, repository.SaveAlarmItemParams{}, validation, err
	}
	create := id == ""
	if create {
		id = uuid.NewString()
		resetAlarmItemChildIDsForCreate(&input)
	}
	params := repository.SaveAlarmItemParams{ID: id, ProjectID: projectID, IsCreate: create, Mode: input.Mode, PresetSlot: normalizeID(input.PresetSlot), EvaluationMode: input.EvaluationMode, DerivedExpression: strings.TrimSpace(input.DerivedExpression), GroupID: normalizeID(input.GroupID), Description: normalizeText(input.Description), NotificationMode: input.Notification.Mode, NotifyOnRaise: input.Notification.NotifyOnRaise, NotifyOnClear: input.Notification.NotifyOnClear, RepeatIntervalSeconds: input.Notification.RepeatIntervalSeconds, NotificationChannelIDs: input.Notification.ChannelIDs, MessageTemplate: input.Notification.MessageTemplate, IsEnabled: valueOr(input.IsEnabled, input.Mode == "point"), Revision: input.Revision}
	if input.Mode == "derived" && params.PresetSlot != nil {
		return input, params, validation, badAlarm("组合报警不能使用默认报警槽位")
	}
	if input.GroupID != nil {
		if _, groupErr := s.repository.GetGroup(ctx, projectID, *input.GroupID); groupErr != nil {
			return input, params, validation, groupErr
		}
	}
	category := ""
	datapointName := ""
	if input.Mode == "point" {
		if strings.TrimSpace(input.DatapointID) == "" {
			return input, params, validation, badAlarm("普通报警必须选择一个数据点")
		}
		point, pointErr := s.datapoints.GetByProjectAndID(ctx, projectID, input.DatapointID)
		if pointErr != nil {
			return input, params, validation, pointErr
		}
		if point.Status == "invalid" && params.IsEnabled {
			return input, params, validation, badAlarm("失效数据点不能创建或启用报警")
		}
		params.DatapointID = &point.ID
		datapointName = strings.TrimSpace(point.Name)
		category = alarmDataCategory(point.DataType)
		input.DerivedExpression = ""
		params.DerivedExpression = ""
		if input.EvaluationMode == "highest_matching" && category != "number" {
			return input, params, validation, badAlarm("只有数值数据点支持分级越限报警")
		}
		if input.EvaluationMode == "single" && len(input.Conditions) != 1 {
			return input, params, validation, badAlarm("普通报警必须且只能配置一个条件")
		}
		if input.EvaluationMode == "highest_matching" && len(input.Conditions) < 1 {
			return input, params, validation, badAlarm("越限报警至少需要一个等级")
		}
	} else {
		if strings.TrimSpace(input.DatapointID) != "" {
			return input, params, validation, badAlarm("组合报警不能设置主体数据点")
		}
		if input.EvaluationMode != "single" {
			return input, params, validation, badAlarm("组合报警只支持单条件")
		}
		if len(input.Inputs) < 2 {
			return input, params, validation, badAlarm("组合报警至少需要两个输入点")
		}
		if params.DerivedExpression == "" {
			return input, params, validation, badAlarm("组合报警表达式不能为空")
		}
		if len(input.Conditions) != 1 {
			return input, params, validation, badAlarm("组合报警必须且只能配置一个结果条件")
		}
		if err = normalizeAlarmItemInputs(ctx, s, projectID, &input, &params); err != nil {
			return input, params, validation, err
		}
		aliases := make(map[string]bool, len(input.Inputs))
		for _, entry := range input.Inputs {
			aliases[entry.InputKey] = true
		}
		if err = validateDerivedAlarmExpression(params.DerivedExpression, aliases); err != nil {
			return input, params, validation, err
		}
	}
	allowedSeverities, severityOrder, err := s.alarmSeverityPolicy(ctx, projectID)
	if err != nil {
		return input, params, validation, err
	}
	input.Conditions, err = normalizeConfigurationConditionsWithPolicy(input.Conditions, category, input.EvaluationMode, input.Mode == "derived", allowedSeverities, severityOrder)
	if err != nil {
		return input, params, validation, err
	}
	if input.Mode == "derived" {
		if err = validateDerivedAlarmExpressionTypes(params.DerivedExpression, input.Inputs, input.Conditions[0]); err != nil {
			return input, params, validation, err
		}
	}
	params.AlarmType = alarmItemType(input)
	displayName := strings.TrimSpace(input.DisplayName)
	if input.Mode == "derived" && displayName == "" {
		return input, params, validation, badAlarm("组合报警名称不能为空")
	}
	if displayName == "" {
		baseName := defaultAlarmItemName(params.AlarmType)
		if datapointName != "" {
			baseName = datapointName + "_" + baseName
		}
		if create {
			displayName, err = s.repository.NextAlarmItemDisplayName(ctx, projectID, *params.DatapointID, baseName)
			if err != nil {
				return input, params, validation, err
			}
		} else {
			displayName = baseName
		}
	}
	if len([]rune(displayName)) > 100 {
		return input, params, validation, badAlarm("报警显示名称不能超过 100 个字符")
	}
	input.DisplayName = displayName
	params.DisplayName = displayName
	params.NameKey = normalizeAlarmName(displayName)
	params.Conditions = toAlarmItemConditionRecords(input.Conditions)
	fingerprint, err := alarmTriggerFingerprint(input)
	if err != nil {
		return input, params, validation, err
	}
	params.TriggerFingerprint = fingerprint
	params.Contract = buildAlarmItemContract(projectID, id, input, params, fingerprint)
	conflicts, err := s.repository.FindAlarmItemConflicts(ctx, projectID, id, params.DatapointID, params.NameKey, fingerprint, input.Mode == "derived")
	if err != nil {
		return input, params, validation, err
	}
	for _, conflict := range conflicts {
		issue := AlarmDraftIssue{Type: conflict.Kind, DatapointID: conflict.DatapointID, DatapointName: conflict.DatapointName, ConflictID: conflict.AlarmItemID, ConflictName: conflict.AlarmItemName}
		if conflict.Kind == "name" {
			issue.Message = "数据点内已存在同名报警"
		} else {
			issue.Message = "数据点内已存在完全相同的报警"
		}
		validation.Errors = append(validation.Errors, issue)
	}
	if input.Mode == "point" && len(validation.Errors) == 0 && params.DatapointID != nil {
		candidates, candidateErr := s.repository.FindAlarmItemOverlapCandidates(ctx, projectID, id, *params.DatapointID, fingerprint)
		if candidateErr != nil {
			return input, params, validation, candidateErr
		}
		for _, candidate := range candidates {
			if !alarmConditionsOverlap(input.Conditions, candidate.Conditions) {
				continue
			}
			ackKey := alarmOverlapAcknowledgementKey(*params.DatapointID, fingerprint, candidate.AlarmItemID)
			validation.Warnings = append(validation.Warnings, AlarmDraftIssue{Type: "overlap", Message: "报警触发区域与已有报警重叠", DatapointID: candidate.DatapointID, DatapointName: candidate.DatapointName, ConflictID: candidate.AlarmItemID, ConflictName: candidate.AlarmItemName, AffectedCount: 1, AcknowledgementKey: ackKey})
		}
	}
	validation.Valid = len(validation.Errors) == 0
	return input, params, validation, nil
}

// 客户端草稿 ID 只用于界面行键。批量创建会复用同一草稿，持久化子记录必须逐项生成独立 ID。
func resetAlarmItemChildIDsForCreate(input *SaveAlarmItemInput) {
	for index := range input.Inputs {
		input.Inputs[index].ID = ""
	}
	for index := range input.Conditions {
		input.Conditions[index].ID = ""
	}
}

type alarmNumericInterval struct{ min, max float64 }

func alarmConditionsOverlap(current []AlarmCondition, existing []repository.AlarmItemConditionRecord) bool {
	currentIntervals, currentNumeric := alarmConditionIntervals(current)
	existingConditions := make([]AlarmCondition, 0, len(existing))
	for _, condition := range existing {
		existingConditions = append(existingConditions, AlarmCondition{Kind: condition.Kind, Operator: condition.Operator, Params: condition.Params})
	}
	existingIntervals, existingNumeric := alarmConditionIntervals(existingConditions)
	if currentNumeric && existingNumeric {
		for _, left := range currentIntervals {
			for _, right := range existingIntervals {
				if left.min < right.max && right.min < left.max {
					return true
				}
			}
		}
		return false
	}
	if len(current) != 1 || len(existing) != 1 || current[0].Kind != existing[0].Kind {
		return false
	}
	left := current[0]
	right := AlarmCondition{Kind: existing[0].Kind, Operator: existing[0].Operator, Params: existing[0].Params}
	switch left.Kind {
	case "state":
		leftExpected := fmt.Sprint(left.Params["expected"])
		rightExpected := fmt.Sprint(right.Params["expected"])
		if left.Operator == "eq" && right.Operator == "eq" {
			return leftExpected == rightExpected
		}
		if left.Operator == "eq" && right.Operator == "ne" {
			return leftExpected != rightExpected
		}
		if left.Operator == "ne" && right.Operator == "eq" {
			return leftExpected != rightExpected
		}
		return true
	case "quality":
		rightValues := map[string]bool{}
		for _, value := range alarmStringList(right.Params["qualities"]) {
			rightValues[value] = true
		}
		for _, value := range alarmStringList(left.Params["qualities"]) {
			if rightValues[value] {
				return true
			}
		}
		return false
	case "stale":
		return true
	case "text_match":
		leftValue := fmt.Sprint(left.Params["expected"])
		rightValue := fmt.Sprint(right.Params["expected"])
		if left.Operator == "eq" && right.Operator == "eq" {
			return leftValue == rightValue
		}
		if left.Operator == "eq" && right.Operator == "contains" {
			return strings.Contains(leftValue, rightValue)
		}
		if left.Operator == "contains" && right.Operator == "eq" {
			return strings.Contains(rightValue, leftValue)
		}
		return left.Operator == right.Operator && leftValue == rightValue
	default:
		return false
	}
}

func alarmConditionIntervals(conditions []AlarmCondition) ([]alarmNumericInterval, bool) {
	const infinity = 1.7976931348623157e+308
	result := []alarmNumericInterval{}
	for _, condition := range conditions {
		switch condition.Kind {
		case "threshold":
			threshold, ok := anyFloat(condition.Params["threshold"])
			if !ok {
				return nil, false
			}
			switch condition.Operator {
			case "gt", "gte":
				result = append(result, alarmNumericInterval{threshold, infinity})
			case "lt", "lte":
				result = append(result, alarmNumericInterval{-infinity, threshold})
			default:
				return nil, false
			}
		case "range":
			lower, lowOK := anyFloat(condition.Params["lower"])
			upper, highOK := anyFloat(condition.Params["upper"])
			if !lowOK || !highOK {
				return nil, false
			}
			if condition.Operator == "between" {
				result = append(result, alarmNumericInterval{lower, upper})
			} else if condition.Operator == "outside" {
				result = append(result, alarmNumericInterval{-infinity, lower}, alarmNumericInterval{upper, infinity})
			} else {
				return nil, false
			}
		default:
			return nil, false
		}
	}
	return result, len(result) > 0
}

func alarmOverlapAcknowledgementKey(datapointID, fingerprint, conflictID string) string {
	parts := []string{datapointID, fingerprint, conflictID}
	sort.Strings(parts)
	sum := sha256.Sum256([]byte(strings.Join(parts, ":")))
	return "overlap:" + hex.EncodeToString(sum[:])
}

func normalizeAlarmItemInputs(ctx context.Context, s *AlarmItemService, projectID string, input *SaveAlarmItemInput, params *repository.SaveAlarmItemParams) error {
	ids := []string{}
	keys := map[string]bool{}
	for _, item := range input.Inputs {
		item.InputKey = strings.TrimSpace(item.InputKey)
		if item.InputKey == "" || keys[item.InputKey] || !alarmInputKeyPattern.MatchString(item.InputKey) {
			return badAlarm("组合报警输入别名必须以字母或下划线开头，且只能包含字母、数字和下划线")
		}
		keys[item.InputKey] = true
		ids = append(ids, item.DatapointID)
	}
	unique := alarmUniqueStrings(ids)
	if len(unique) != len(ids) {
		return badAlarm("组合报警不能重复选择数据点")
	}
	points, err := s.datapoints.GetByProjectAndIDs(ctx, projectID, unique)
	if err != nil {
		return err
	}
	byID := map[string]repository.DataPointRecord{}
	for _, point := range points {
		byID[point.ID] = point
	}
	for index, item := range input.Inputs {
		point, ok := byID[item.DatapointID]
		if !ok {
			return badAlarm("组合报警输入点不存在或不属于当前工程")
		}
		if point.Status == "invalid" && params.IsEnabled {
			return badAlarm("失效数据点不能作为组合报警输入")
		}
		if strings.TrimSpace(item.ID) == "" {
			item.ID = uuid.NewString()
		}
		item.Path = point.Path
		item.Name = point.Name
		item.DataType = point.DataType
		input.Inputs[index] = item
		params.Inputs = append(params.Inputs, repository.AlarmItemInputRecord{ID: item.ID, DatapointID: item.DatapointID, Path: item.Path, Name: item.Name, DataType: item.DataType, InputKey: item.InputKey, SortOrder: index})
	}
	return nil
}

func alarmItemType(input SaveAlarmItemInput) string {
	if input.Mode == "derived" {
		return "derived"
	}
	if input.EvaluationMode == "highest_matching" {
		return "threshold"
	}
	if len(input.Conditions) > 0 {
		return input.Conditions[0].Kind
	}
	return "threshold"
}
func defaultAlarmItemName(kind string) string {
	return map[string]string{"threshold": "越限", "range": "区间", "state": "状态", "transition": "状态变化", "text_match": "文本", "rate_of_change": "变化率", "deviation": "偏差", "offline": "离线", "quality": "质量", "stale": "数据陈旧", "derived": "组合报警"}[kind]
}

func normalizeConfigurationConditions(conditions []AlarmCondition, category, evaluationMode string, derived bool) ([]AlarmCondition, error) {
	allowed, order := defaultAlarmSeverityPolicy()
	return normalizeConfigurationConditionsWithPolicy(conditions, category, evaluationMode, derived, allowed, order)
}

func normalizeConfigurationConditionsWithPolicy(conditions []AlarmCondition, category, evaluationMode string, derived bool, allowedSeverities map[string]bool, severityOrder map[string]int) ([]AlarmCondition, error) {
	result := make([]AlarmCondition, 0, len(conditions))
	ids := map[string]bool{}
	for _, condition := range conditions {
		if condition.Params == nil {
			condition.Params = map[string]any{}
		}
		delete(condition.Params, "priority")
		if !alarmConditionKinds[condition.Kind] || !allowedSeverities[condition.Severity] {
			return nil, badAlarm("报警条件类型或等级不受支持")
		}
		if condition.TriggerDelayMS < 0 || condition.ClearDelayMS < 0 || condition.Deadband < 0 {
			return nil, badAlarm("报警延时和死区不能为负数")
		}
		if !derived && !conditionAllowedForCategory(condition.Kind, category) {
			return nil, badAlarm("报警条件与数据点类型不兼容")
		}
		if derived && (condition.Kind == "offline" || condition.Kind == "quality" || condition.Kind == "stale") {
			return nil, badAlarm("组合报警的结果条件不支持离线、质量、陈旧或表达式类型")
		}
		if condition.Deadband > 0 && !alarmConditionSupportsDeadband(condition.Kind) {
			return nil, badAlarm("当前报警条件不支持死区")
		}
		if err := validateConditionParams(condition); err != nil {
			return nil, err
		}
		// 组合表达式的 state 结果固定为布尔语义；文本结果请使用 text_match，避免把任意文本误当状态。
		if derived && condition.Kind == "state" {
			if _, ok := condition.Params["expected"].(bool); !ok {
				return nil, badAlarm("组合报警的状态结果只能选择 true 或 false")
			}
		}
		if condition.ID == "" {
			condition.ID = uuid.NewString()
		}
		if ids[condition.ID] {
			return nil, badAlarm("报警条件 ID 不能重复")
		}
		ids[condition.ID] = true
		condition.Label = strings.TrimSpace(condition.Label)
		if condition.Label == "" {
			condition.Label = defaultConditionLabel(condition)
		}
		condition.Params = alarmCloneMap(condition.Params)
		result = append(result, condition)
	}
	if evaluationMode == "highest_matching" {
		if err := validateHighestMatchingConditions(result, severityOrder); err != nil {
			return nil, err
		}
		sort.SliceStable(result, func(i, j int) bool {
			di, vi := thresholdDirectionAndValue(result[i])
			dj, vj := thresholdDirectionAndValue(result[j])
			if di != dj {
				return di == "high"
			}
			if di == "high" {
				return vi < vj
			}
			return vi > vj
		})
	}
	return result, nil
}

func defaultAlarmSeverityPolicy() (map[string]bool, map[string]int) {
	allowed, order := map[string]bool{}, map[string]int{}
	for _, definition := range defaultAlarmSeverityDefinitions() {
		allowed[definition.Key] = true
		order[definition.Key] = definition.SortOrder
	}
	return allowed, order
}

func (s *AlarmItemService) alarmSeverityPolicy(ctx context.Context, projectID string) (map[string]bool, map[string]int, error) {
	record, err := s.repository.GetProjectSettings(ctx, projectID)
	if err != nil {
		return nil, nil, err
	}
	if record == nil || len(record.SeverityDefinitions) == 0 {
		allowed, order := defaultAlarmSeverityPolicy()
		return allowed, order, nil
	}
	allowed, order := map[string]bool{}, map[string]int{}
	for _, definition := range record.SeverityDefinitions {
		allowed[definition.Key] = true
		order[definition.Key] = definition.SortOrder
	}
	return allowed, order, nil
}

func alarmConditionSupportsDeadband(kind string) bool {
	return kind == "threshold" || kind == "range" || kind == "rate_of_change" || kind == "deviation"
}
func validateHighestMatchingConditions(conditions []AlarmCondition, severityOrder map[string]int) error {
	highs, lows := []AlarmCondition{}, []AlarmCondition{}
	seen := map[string]bool{}
	for _, condition := range conditions {
		if condition.Kind != "threshold" {
			return badAlarm("分级越限报警只支持阈值等级")
		}
		direction, value := thresholdDirectionAndValue(condition)
		if direction == "" {
			return badAlarm("越限等级操作符只支持高限或低限")
		}
		key := strconv.FormatFloat(value, 'g', -1, 64)
		if seen[key] {
			return badAlarm("越限等级阈值不能重复")
		}
		seen[key] = true
		if direction == "high" {
			highs = append(highs, condition)
		} else {
			lows = append(lows, condition)
		}
	}
	sort.Slice(highs, func(i, j int) bool {
		_, a := thresholdDirectionAndValue(highs[i])
		_, b := thresholdDirectionAndValue(highs[j])
		return a < b
	})
	sort.Slice(lows, func(i, j int) bool {
		_, a := thresholdDirectionAndValue(lows[i])
		_, b := thresholdDirectionAndValue(lows[j])
		return a > b
	})
	for _, group := range [][]AlarmCondition{highs, lows} {
		for i := 1; i < len(group); i++ {
			if severityOrder[group[i].Severity] < severityOrder[group[i-1].Severity] {
				return badAlarm("向外越限时报警等级不能降低")
			}
		}
	}
	if len(highs) > 0 && len(lows) > 0 {
		_, high := thresholdDirectionAndValue(highs[0])
		_, low := thresholdDirectionAndValue(lows[0])
		if low+lows[0].Deadband >= high-highs[0].Deadband {
			return badAlarm("高低限之间必须保留有效正常区间")
		}
	}
	return nil
}
func thresholdDirectionAndValue(condition AlarmCondition) (string, float64) {
	value, _ := anyFloat(condition.Params["threshold"])
	switch condition.Operator {
	case "gt", "gte":
		return "high", value
	case "lt", "lte":
		return "low", value
	}
	return "", value
}
func alarmTriggerFingerprint(input SaveAlarmItemInput) (string, error) {
	type fingerprintCondition struct {
		Kind, Operator               string
		Params                       map[string]any
		TriggerDelayMS, ClearDelayMS int64
		Deadband                     float64
	}
	conditions := []fingerprintCondition{}
	for _, condition := range input.Conditions {
		params := make(map[string]any, len(condition.Params))
		for key, value := range condition.Params {
			if key != "priority" && key != "configurationMode" {
				params[key] = value
			}
		}
		conditions = append(conditions, fingerprintCondition{condition.Kind, condition.Operator, params, condition.TriggerDelayMS, condition.ClearDelayMS, condition.Deadband})
	}
	payload := map[string]any{"mode": input.Mode, "evaluationMode": input.EvaluationMode, "conditions": conditions}
	if input.Mode == "derived" {
		inputs := []string{}
		for _, entry := range input.Inputs {
			inputs = append(inputs, entry.DatapointID+":"+entry.InputKey)
		}
		sort.Strings(inputs)
		payload["expression"] = strings.TrimSpace(input.DerivedExpression)
		payload["inputs"] = inputs
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", badAlarm("生成报警触发指纹失败")
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}
func normalizeAlarmName(value string) string { return strings.ToLower(strings.TrimSpace(value)) }
func toAlarmItemConditionRecords(items []AlarmCondition) []repository.AlarmItemConditionRecord {
	result := make([]repository.AlarmItemConditionRecord, 0, len(items))
	for index, item := range items {
		result = append(result, repository.AlarmItemConditionRecord{ID: item.ID, Kind: item.Kind, Operator: item.Operator, Label: item.Label, Severity: item.Severity, Params: item.Params, TriggerDelayMS: item.TriggerDelayMS, ClearDelayMS: item.ClearDelayMS, Deadband: item.Deadband, SortOrder: index})
	}
	return result
}
func buildAlarmItemContract(projectID, id string, input SaveAlarmItemInput, params repository.SaveAlarmItemParams, fingerprint string) map[string]any {
	return map[string]any{"schemaVersion": "alarm.item.v1", "projectId": projectID, "alarmItemId": id, "datapointId": params.DatapointID, "displayName": params.DisplayName, "mode": input.Mode, "alarmType": params.AlarmType, "evaluationMode": input.EvaluationMode, "inputs": params.Inputs, "derivedExpression": input.DerivedExpression, "conditions": input.Conditions, "notification": input.Notification, "isEnabled": params.IsEnabled, "triggerFingerprint": fingerprint}
}
func toAlarmItem(record repository.AlarmItemRecord) AlarmItem {
	inputs := make([]AlarmItemInput, 0, len(record.Inputs))
	for _, item := range record.Inputs {
		inputs = append(inputs, AlarmItemInput{ID: item.ID, DatapointID: item.DatapointID, Path: item.Path, Name: item.Name, DataType: item.DataType, InputKey: item.InputKey})
	}
	conditions := make([]AlarmCondition, 0, len(record.Conditions))
	for _, item := range record.Conditions {
		conditions = append(conditions, AlarmCondition{ID: item.ID, Kind: item.Kind, Operator: item.Operator, Label: item.Label, Severity: item.Severity, Params: item.Params, TriggerDelayMS: item.TriggerDelayMS, ClearDelayMS: item.ClearDelayMS, Deadband: item.Deadband})
	}
	value := func(pointer *string) string {
		if pointer == nil {
			return ""
		}
		return *pointer
	}
	return AlarmItem{ID: record.ID, ProjectID: record.ProjectID, DisplayName: record.DisplayName, Mode: record.Mode, AlarmType: record.AlarmType, PresetSlot: record.PresetSlot, EvaluationMode: record.EvaluationMode, DerivedExpression: record.DerivedExpression, DatapointID: value(record.DatapointID), Path: value(record.Path), DatapointName: value(record.DatapointName), DataType: value(record.DataType), GroupID: record.GroupID, GroupName: record.GroupName, Description: record.Description, Inputs: inputs, Conditions: conditions, Notification: AlarmNotificationSettings{Mode: record.NotificationMode, NotifyOnRaise: record.NotifyOnRaise, NotifyOnClear: record.NotifyOnClear, RepeatIntervalSeconds: record.RepeatIntervalSeconds, ChannelIDs: record.NotificationChannelIDs, MessageTemplate: record.MessageTemplate}, IsEnabled: record.IsEnabled, Revision: record.Revision, Contract: record.Contract, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}
func selectAlarmCondition(evaluationMode string, conditions []AlarmCondition, value any, contextValues map[string]any) *AlarmCondition {
	if evaluationMode != "highest_matching" {
		result := evaluateAlarmConditions(conditions, value, contextValues)
		if len(result.TriggeredConditions) > 0 {
			return &result.TriggeredConditions[0]
		}
		return nil
	}
	var selected *AlarmCondition
	for index := range conditions {
		condition := &conditions[index]
		matched, _ := evaluateAlarmCondition(*condition, value, contextString(contextValues, "quality"), contextBool(contextValues, "offline"), contextValues)
		if !matched {
			continue
		}
		direction, threshold := thresholdDirectionAndValue(*condition)
		if selected == nil {
			selected = condition
			continue
		}
		selectedDirection, selectedThreshold := thresholdDirectionAndValue(*selected)
		if direction == "high" && (selectedDirection != "high" || threshold > selectedThreshold) {
			selected = condition
		}
		if direction == "low" && (selectedDirection != "low" || threshold < selectedThreshold) {
			selected = condition
		}
	}
	return selected
}

func contextString(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return value
}
func contextBool(values map[string]any, key string) bool {
	value, _ := values[key].(bool)
	return value
}
func (s *AlarmItemService) normalizeNotification(ctx context.Context, projectID string, input AlarmNotificationSettings) (AlarmNotificationSettings, error) {
	normalized, err := normalizeNotificationShape(input)
	if err != nil {
		return normalized, err
	}
	if normalized.Mode == "custom" {
		channels, channelErr := s.repository.ListChannels(ctx, projectID)
		if channelErr != nil {
			return normalized, channelErr
		}
		allowed := map[string]bool{"runtime_inapp": true}
		for _, channel := range channels {
			if channel.IsEnabled {
				allowed[channel.ID] = true
			}
		}
		for _, id := range normalized.ChannelIDs {
			if !allowed[id] {
				return normalized, badAlarm("通知渠道不存在、已停用或不属于当前工程")
			}
		}
	}
	return normalized, nil
}
func (s *AlarmItemService) readAccess(claims *auth.Claims, projectID string) error {
	if s == nil || s.repository == nil || s.datapoints == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "报警项服务未初始化")
	}
	if claims == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeAuthTokenRequired, http.StatusUnauthorized, "请先完成认证")
	}
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	if !claims.HasProjectAccess(projectID) {
		return apperrors.NewAppError(apperrors.ErrorCodePermissionProjectMismatch, http.StatusForbidden, "项目范围不足")
	}
	return nil
}
func (s *AlarmItemService) writeAccess(claims *auth.Claims, projectID string) error {
	if err := s.readAccess(claims, projectID); err != nil {
		return err
	}
	return validateUserID(claims.UserID)
}
func alarmDraftConflict(message string) error {
	return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, message)
}
func _alarmItemDebug(value any) string { return fmt.Sprint(value) }
