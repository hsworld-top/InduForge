package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/indu-forge/data_service/internal/auth"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
	security "github.com/indu-forge/data_service/internal/security"
)

const defaultAlarmMessageTemplate = "{{pointName}} 当前值 {{value}}，触发 {{conditionLabel}}"
const defaultAlarmHistoryRetentionDays = 30

var (
	alarmModes      = map[string]bool{"per_target": true, "derived": true}
	alarmSeverities = map[string]bool{"info": true, "warning": true, "major": true, "critical": true}
	alarmKinds      = map[string]bool{"threshold": true, "range": true, "state": true, "transition": true, "text_match": true, "rate_of_change": true, "deviation": true, "offline": true, "expression": true}
	numericTypes    = map[string]bool{"number": true, "integer": true, "float": true, "double": true, "decimal": true, "int": true, "int8": true, "int16": true, "int32": true, "int64": true, "uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true, "float32": true, "float64": true, "numeric": true}
)

type AlarmPolicyGroup struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	Name        string    `json:"name"`
	ParentID    *string   `json:"parentId"`
	Description *string   `json:"description"`
	SortOrder   int       `json:"sortOrder"`
	FullPath    string    `json:"fullPath"`
	HasChildren bool      `json:"hasChildren"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type AlarmBinding struct {
	DatapointID string  `json:"datapointId"`
	Path        string  `json:"path"`
	Name        string  `json:"name"`
	DataType    string  `json:"dataType"`
	Role        string  `json:"role"`
	InputKey    *string `json:"inputKey,omitempty"`
}

type AlarmCondition struct {
	ID             string         `json:"id"`
	Kind           string         `json:"kind"`
	Operator       string         `json:"operator"`
	Label          string         `json:"label"`
	Severity       string         `json:"severity"`
	Params         map[string]any `json:"params"`
	TriggerDelayMS int64          `json:"triggerDelayMs"`
	ClearDelayMS   int64          `json:"clearDelayMs"`
	Deadband       float64        `json:"deadband"`
}

type AlarmNotificationSettings struct {
	Mode                  string   `json:"mode"`
	NotifyOnRaise         *bool    `json:"notifyOnRaise,omitempty"`
	NotifyOnClear         *bool    `json:"notifyOnClear,omitempty"`
	RepeatIntervalSeconds *int     `json:"repeatIntervalSeconds,omitempty"`
	ChannelIDs            []string `json:"channelIds"`
	MessageTemplate       string   `json:"messageTemplate"`
}

type AlarmPolicy struct {
	ID                string                    `json:"id"`
	ProjectID         string                    `json:"projectId"`
	Name              string                    `json:"name"`
	Mode              string                    `json:"mode"`
	DerivedExpression string                    `json:"derivedExpression"`
	GroupID           *string                   `json:"groupId"`
	GroupName         *string                   `json:"groupName"`
	Description       *string                   `json:"description"`
	Bindings          []AlarmBinding            `json:"bindings"`
	Conditions        []AlarmCondition          `json:"conditions"`
	Notification      AlarmNotificationSettings `json:"notification"`
	IsEnabled         bool                      `json:"isEnabled"`
	Revision          int64                     `json:"revision"`
	Contract          map[string]any            `json:"contract"`
	CreatedAt         time.Time                 `json:"createdAt"`
	UpdatedAt         time.Time                 `json:"updatedAt"`
}

type AlarmPolicyListFilter struct {
	Search, Severity, ConditionKind, Mode string
	GroupID                               *string
	Enabled                               *bool
	DatapointID                           string
	Page, PageSize                        int
}

type AlarmPolicyListResult struct {
	List       []AlarmPolicy `json:"list"`
	Pagination Pagination    `json:"pagination"`
}

type AlarmPolicyGroupListResult struct {
	List       []AlarmPolicyGroup `json:"list"`
	Pagination Pagination         `json:"pagination"`
}

type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
}

type SaveAlarmPolicyInput struct {
	GroupID           *string                   `json:"groupId"`
	Name              string                    `json:"name"`
	Description       *string                   `json:"description"`
	Mode              string                    `json:"mode"`
	Bindings          []AlarmBinding            `json:"bindings"`
	DerivedExpression string                    `json:"derivedExpression"`
	Conditions        []AlarmCondition          `json:"conditions"`
	Notification      AlarmNotificationSettings `json:"notification"`
	IsEnabled         *bool                     `json:"isEnabled"`
}

type SaveAlarmPolicyGroupInput struct {
	Name        string  `json:"name"`
	ParentID    *string `json:"parentId"`
	Description *string `json:"description"`
	SortOrder   int     `json:"sortOrder"`
}

type AlarmProjectSettings struct {
	ProjectID              string    `json:"projectId"`
	NotifyOnRaise          bool      `json:"notifyOnRaise"`
	NotifyOnClear          bool      `json:"notifyOnClear"`
	RepeatIntervalSeconds  *int      `json:"repeatIntervalSeconds"`
	DefaultMessageTemplate string    `json:"defaultMessageTemplate"`
	DefaultChannelIDs      []string  `json:"defaultChannelIds"`
	CreatedAt              time.Time `json:"createdAt"`
	UpdatedAt              time.Time `json:"updatedAt"`
}

type SaveAlarmProjectSettingsInput struct {
	NotifyOnRaise          bool     `json:"notifyOnRaise"`
	NotifyOnClear          bool     `json:"notifyOnClear"`
	RepeatIntervalSeconds  *int     `json:"repeatIntervalSeconds"`
	DefaultMessageTemplate string   `json:"defaultMessageTemplate"`
	DefaultChannelIDs      []string `json:"defaultChannelIds"`
}

type AlarmHistorySettings struct {
	ProjectID                   string    `json:"projectId"`
	IsEnabled                   bool      `json:"isEnabled"`
	RetentionDays               *int      `json:"retentionDays"`
	StoreNotificationDeliveries bool      `json:"storeNotificationDeliveries"`
	CreatedAt                   time.Time `json:"createdAt"`
	UpdatedAt                   time.Time `json:"updatedAt"`
}

type SaveAlarmHistorySettingsInput struct {
	IsEnabled                   bool `json:"isEnabled"`
	RetentionDays               *int `json:"retentionDays"`
	StoreNotificationDeliveries bool `json:"storeNotificationDeliveries"`
}

type AlarmNotificationChannel struct {
	ID           string         `json:"id"`
	ProjectID    string         `json:"projectId"`
	Name         string         `json:"name"`
	ChannelType  string         `json:"channelType"`
	Config       map[string]any `json:"config"`
	SecretStatus map[string]any `json:"secretStatus"`
	IsEnabled    bool           `json:"isEnabled"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
}

type SaveAlarmNotificationChannelInput struct {
	Name             string            `json:"name"`
	ChannelType      string            `json:"channelType"`
	Config           map[string]any    `json:"config"`
	Secrets          map[string]string `json:"secrets"`
	DeleteSecretKeys []string          `json:"deleteSecretKeys"`
	IsEnabled        *bool             `json:"isEnabled"`
}

type AlarmConfigSyncOperation struct {
	Resource string          `json:"resource"`
	Action   string          `json:"action"`
	ID       string          `json:"id"`
	Data     json.RawMessage `json:"data"`
}

type AlarmConfigSyncInput struct {
	SyncEpoch      string                     `json:"syncEpoch"`
	Sequence       int64                      `json:"sequence"`
	IdempotencyKey string                     `json:"idempotencyKey"`
	Operations     []AlarmConfigSyncOperation `json:"operations"`
}

type AlarmConfigSyncResult struct {
	ConfigRevision   int64 `json:"configRevision"`
	AcceptedSequence int64 `json:"acceptedSequence"`
	Idempotent       bool  `json:"idempotent"`
}

type AlarmDatapointSummary struct {
	DatapointID string        `json:"datapointId"`
	Policies    []AlarmPolicy `json:"policies"`
	Count       int           `json:"count"`
}

type AlarmPolicyTrialResult struct {
	Triggered           bool             `json:"triggered"`
	State               string           `json:"state"`
	TriggeredConditions []AlarmCondition `json:"triggeredConditions"`
	ConditionResults    []map[string]any `json:"conditionResults"`
	Message             string           `json:"message,omitempty"`
}

type AlarmPolicyService struct {
	repository *repository.AlarmPolicyRepository
	datapoints *repository.DataPointRepository
	cipher     *security.AlarmSecretCipher
}

func NewAlarmPolicyService(repo *repository.AlarmPolicyRepository, datapoints *repository.DataPointRepository, ciphers ...*security.AlarmSecretCipher) *AlarmPolicyService {
	var cipher *security.AlarmSecretCipher
	if len(ciphers) > 0 {
		cipher = ciphers[0]
	}
	return &AlarmPolicyService{repository: repo, datapoints: datapoints, cipher: cipher}
}

func (s *AlarmPolicyService) ListGroups(ctx context.Context, claims *auth.Claims, projectID, search string, parentID *string, page, pageSize int) (*AlarmPolicyGroupListResult, error) {
	if err := s.readAccess(claims, projectID); err != nil {
		return nil, err
	}
	records, total, err := s.repository.ListGroups(ctx, projectID, repository.AlarmPolicyGroupListFilter{Search: search, ParentID: parentID, Page: page, PageSize: pageSize})
	if err != nil {
		return nil, err
	}
	items := make([]AlarmPolicyGroup, 0, len(records))
	for _, record := range records {
		items = append(items, toAlarmGroup(record))
	}
	return &AlarmPolicyGroupListResult{List: items, Pagination: Pagination{Page: positive(page, 1), PageSize: positive(pageSize, 50), Total: total}}, nil
}

func (s *AlarmPolicyService) ListAllGroups(ctx context.Context, claims *auth.Claims, projectID string) ([]AlarmPolicyGroup, error) {
	if err := s.readAccess(claims, projectID); err != nil {
		return nil, err
	}
	records, err := s.repository.ListAllGroups(ctx, projectID)
	if err != nil {
		return nil, err
	}
	items := make([]AlarmPolicyGroup, 0, len(records))
	for _, record := range records {
		items = append(items, toAlarmGroup(record))
	}
	return items, nil
}

func (s *AlarmPolicyService) CreateGroup(ctx context.Context, claims *auth.Claims, projectID string, input SaveAlarmPolicyGroupInput) (*AlarmPolicyGroup, error) {
	return s.saveGroup(ctx, claims, projectID, "", input)
}
func (s *AlarmPolicyService) UpdateGroup(ctx context.Context, claims *auth.Claims, projectID, id string, input SaveAlarmPolicyGroupInput) (*AlarmPolicyGroup, error) {
	return s.saveGroup(ctx, claims, projectID, id, input)
}
func (s *AlarmPolicyService) saveGroup(ctx context.Context, claims *auth.Claims, projectID, id string, input SaveAlarmPolicyGroupInput) (*AlarmPolicyGroup, error) {
	if err := s.writeAccess(claims, projectID); err != nil {
		return nil, err
	}
	name, err := requiredName(input.Name, "目录名称")
	if err != nil {
		return nil, err
	}
	if input.ParentID != nil {
		if *input.ParentID == id && id != "" {
			return nil, badAlarm("目录不能移动到自身")
		}
		if _, err = s.repository.GetGroup(ctx, projectID, *input.ParentID); err != nil {
			return nil, err
		}
		if id != "" {
			groups, _ := s.repository.ListAllGroups(ctx, projectID)
			if groupIsDescendant(groups, id, *input.ParentID) {
				return nil, badAlarm("目录不能移动到自己的子目录")
			}
		}
	}
	record, err := s.repository.SaveGroup(ctx, repository.SaveAlarmPolicyGroupParams{ID: id, ProjectID: projectID, UserID: claims.UserID, Name: name, ParentID: normalizeID(input.ParentID), Description: normalizeText(input.Description), SortOrder: input.SortOrder})
	if err != nil {
		return nil, err
	}
	result := toAlarmGroup(*record)
	return &result, nil
}
func (s *AlarmPolicyService) DeleteGroup(ctx context.Context, claims *auth.Claims, projectID, id string) error {
	if err := s.writeAccess(claims, projectID); err != nil {
		return err
	}
	return s.repository.DeleteGroup(ctx, projectID, id)
}

func (s *AlarmPolicyService) List(ctx context.Context, claims *auth.Claims, projectID string, f AlarmPolicyListFilter) (*AlarmPolicyListResult, error) {
	if err := s.readAccess(claims, projectID); err != nil {
		return nil, err
	}
	if err := validateAlarmFilters(f); err != nil {
		return nil, err
	}
	records, total, err := s.repository.ListPolicies(ctx, projectID, repository.AlarmPolicyListFilter{Search: f.Search, GroupID: f.GroupID, Enabled: f.Enabled, Severity: f.Severity, ConditionKind: f.ConditionKind, Mode: f.Mode, DatapointID: f.DatapointID, Page: f.Page, PageSize: f.PageSize})
	if err != nil {
		return nil, err
	}
	items := make([]AlarmPolicy, 0, len(records))
	for _, record := range records {
		items = append(items, toAlarmPolicy(record))
	}
	return &AlarmPolicyListResult{List: items, Pagination: Pagination{Page: positive(f.Page, 1), PageSize: positive(f.PageSize, 20), Total: total}}, nil
}
func (s *AlarmPolicyService) Get(ctx context.Context, claims *auth.Claims, projectID, id string) (*AlarmPolicy, error) {
	if err := s.readAccess(claims, projectID); err != nil {
		return nil, err
	}
	record, err := s.repository.GetPolicy(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	item := toAlarmPolicy(*record)
	return &item, nil
}
func (s *AlarmPolicyService) Create(ctx context.Context, claims *auth.Claims, projectID string, input SaveAlarmPolicyInput) (*AlarmPolicy, error) {
	return s.savePolicy(ctx, claims, projectID, "", input)
}
func (s *AlarmPolicyService) Update(ctx context.Context, claims *auth.Claims, projectID, id string, input SaveAlarmPolicyInput) (*AlarmPolicy, error) {
	return s.savePolicy(ctx, claims, projectID, id, input)
}

func (s *AlarmPolicyService) savePolicy(ctx context.Context, claims *auth.Claims, projectID, id string, input SaveAlarmPolicyInput) (*AlarmPolicy, error) {
	if err := s.writeAccess(claims, projectID); err != nil {
		return nil, err
	}
	normalized, err := s.normalizePolicy(ctx, projectID, input)
	if err != nil {
		return nil, err
	}
	// 创建时先确定策略 ID，确保持久化契约与策略主记录使用同一标识。
	isCreate := id == ""
	if isCreate {
		id = uuid.NewString()
	}
	contract := buildAlarmContract(projectID, id, normalized)
	params := toSaveAlarmPolicyParams(projectID, id, claims.UserID, normalized, contract)
	params.IsCreate = isCreate
	record, err := s.repository.SavePolicy(ctx, params)
	if err != nil {
		return nil, err
	}
	item := toAlarmPolicy(*record)
	return &item, nil
}

func (s *AlarmPolicyService) SetEnabled(ctx context.Context, claims *auth.Claims, projectID, id string, enabled bool) (*AlarmPolicy, error) {
	if err := s.writeAccess(claims, projectID); err != nil {
		return nil, err
	}
	if enabled {
		record, err := s.repository.GetPolicy(ctx, projectID, id)
		if err != nil {
			return nil, err
		}
		if err = validatePolicyReady(*record); err != nil {
			return nil, err
		}
	}
	record, err := s.repository.SetPolicyEnabled(ctx, projectID, id, claims.UserID, enabled)
	if err != nil {
		return nil, err
	}
	item := toAlarmPolicy(*record)
	return &item, nil
}
func (s *AlarmPolicyService) Delete(ctx context.Context, claims *auth.Claims, projectID, id string) error {
	if err := s.writeAccess(claims, projectID); err != nil {
		return err
	}
	return s.repository.DeletePolicy(ctx, projectID, id)
}

func (s *AlarmPolicyService) ValidateDraft(ctx context.Context, claims *auth.Claims, projectID string, input SaveAlarmPolicyInput) (map[string]any, error) {
	if err := s.readAccess(claims, projectID); err != nil {
		return nil, err
	}
	_, err := s.normalizePolicy(ctx, projectID, input)
	if err != nil {
		return map[string]any{"valid": false, "errors": []string{err.Error()}}, nil
	}
	return map[string]any{"valid": true, "errors": []string{}}, nil
}

func (s *AlarmPolicyService) DatapointSummary(ctx context.Context, claims *auth.Claims, projectID, datapointID string) (*AlarmDatapointSummary, error) {
	if err := s.readAccess(claims, projectID); err != nil {
		return nil, err
	}
	if _, err := s.datapoints.GetByProjectAndID(ctx, projectID, datapointID); err != nil {
		return nil, err
	}
	records, err := s.repository.ListPoliciesByDatapoint(ctx, projectID, datapointID)
	if err != nil {
		return nil, err
	}
	items := make([]AlarmPolicy, 0, len(records))
	for _, record := range records {
		items = append(items, toAlarmPolicy(record))
	}
	return &AlarmDatapointSummary{DatapointID: datapointID, Policies: items, Count: len(items)}, nil
}

func (s *AlarmPolicyService) GetSettings(ctx context.Context, claims *auth.Claims, projectID string) (*AlarmProjectSettings, error) {
	if err := s.readAccess(claims, projectID); err != nil {
		return nil, err
	}
	record, err := s.repository.GetProjectSettings(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return &AlarmProjectSettings{ProjectID: projectID, NotifyOnRaise: true, NotifyOnClear: true, DefaultMessageTemplate: defaultAlarmMessageTemplate, DefaultChannelIDs: []string{"runtime_inapp"}}, nil
	}
	item := toAlarmSettings(*record)
	return &item, nil
}
func (s *AlarmPolicyService) UpdateSettings(ctx context.Context, claims *auth.Claims, projectID string, input SaveAlarmProjectSettingsInput) (*AlarmProjectSettings, error) {
	if err := s.writeAccess(claims, projectID); err != nil {
		return nil, err
	}
	if input.RepeatIntervalSeconds != nil && *input.RepeatIntervalSeconds <= 0 {
		return nil, badAlarm("重复提醒间隔必须为正整数")
	}
	if err := s.validateChannelIDs(ctx, projectID, input.DefaultChannelIDs); err != nil {
		return nil, err
	}
	template := strings.TrimSpace(input.DefaultMessageTemplate)
	if template == "" {
		template = defaultAlarmMessageTemplate
	}
	record, err := s.repository.SaveProjectSettings(ctx, repository.SaveAlarmProjectSettingsParams{ProjectID: projectID, UserID: claims.UserID, NotifyOnRaise: input.NotifyOnRaise, NotifyOnClear: input.NotifyOnClear, RepeatIntervalSeconds: input.RepeatIntervalSeconds, DefaultMessageTemplate: template, DefaultChannelIDs: alarmUniqueStrings(input.DefaultChannelIDs)})
	if err != nil {
		return nil, err
	}
	item := toAlarmSettings(*record)
	return &item, nil
}

func (s *AlarmPolicyService) GetHistorySettings(ctx context.Context, claims *auth.Claims, projectID string) (*AlarmHistorySettings, error) {
	if err := s.readAccess(claims, projectID); err != nil {
		return nil, err
	}
	record, err := s.repository.GetHistorySettings(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		item := defaultAlarmHistorySettings(projectID)
		return &item, nil
	}
	item := toAlarmHistorySettings(*record)
	return &item, nil
}

func (s *AlarmPolicyService) UpdateHistorySettings(ctx context.Context, claims *auth.Claims, projectID string, input SaveAlarmHistorySettingsInput) (*AlarmHistorySettings, error) {
	if err := s.writeAccess(claims, projectID); err != nil {
		return nil, err
	}
	if err := validateAlarmHistorySettings(input); err != nil {
		return nil, err
	}
	record, err := s.repository.SaveHistorySettings(ctx, repository.SaveAlarmHistorySettingsParams{
		ProjectID:                   projectID,
		UserID:                      claims.UserID,
		IsEnabled:                   input.IsEnabled,
		RetentionDays:               input.RetentionDays,
		StoreNotificationDeliveries: input.StoreNotificationDeliveries,
	})
	if err != nil {
		return nil, err
	}
	item := toAlarmHistorySettings(*record)
	return &item, nil
}

func (s *AlarmPolicyService) ListChannels(ctx context.Context, claims *auth.Claims, projectID string) ([]AlarmNotificationChannel, error) {
	if err := s.readAccess(claims, projectID); err != nil {
		return nil, err
	}
	records, err := s.repository.ListChannels(ctx, projectID)
	if err != nil {
		return nil, err
	}
	items := []AlarmNotificationChannel{{ID: "runtime_inapp", ProjectID: projectID, Name: "运行端站内通知", ChannelType: "runtime_inapp", Config: map[string]any{"recipientScope": "project_runtime_users"}, SecretStatus: map[string]any{}, IsEnabled: true}}
	for _, record := range records {
		items = append(items, toAlarmChannel(record))
	}
	return items, nil
}
func (s *AlarmPolicyService) CreateChannel(ctx context.Context, claims *auth.Claims, projectID string, input SaveAlarmNotificationChannelInput) (*AlarmNotificationChannel, error) {
	return s.saveChannel(ctx, claims, projectID, "", input)
}
func (s *AlarmPolicyService) UpdateChannel(ctx context.Context, claims *auth.Claims, projectID, id string, input SaveAlarmNotificationChannelInput) (*AlarmNotificationChannel, error) {
	if id == "runtime_inapp" {
		return nil, badAlarm("内置站内通知不能修改")
	}
	return s.saveChannel(ctx, claims, projectID, id, input)
}
func (s *AlarmPolicyService) saveChannel(ctx context.Context, claims *auth.Claims, projectID, id string, input SaveAlarmNotificationChannelInput) (*AlarmNotificationChannel, error) {
	if err := s.writeAccess(claims, projectID); err != nil {
		return nil, err
	}
	name, err := requiredName(input.Name, "渠道名称")
	if err != nil {
		return nil, err
	}
	channelType := strings.TrimSpace(input.ChannelType)
	if err = validateChannelConfig(channelType, input.Config); err != nil {
		return nil, err
	}
	enabled := true
	if input.IsEnabled != nil {
		enabled = *input.IsEnabled
	}
	status := map[string]any{}
	encrypted := make([]repository.EncryptedAlarmChannelSecret, 0, len(input.Secrets))
	if len(input.Secrets) > 0 && s.cipher == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "报警渠道密钥服务未初始化")
	}
	for key, value := range input.Secrets {
		key = strings.TrimSpace(key)
		if key == "" || strings.TrimSpace(value) == "" {
			continue
		}
		payload, cipherErr := s.cipher.Encrypt([]byte(value))
		if cipherErr != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "加密报警渠道密钥失败", cipherErr)
		}
		encrypted = append(encrypted, repository.EncryptedAlarmChannelSecret{Key: key, Value: payload, KeyVersion: s.cipher.KeyVersion()})
		status[key] = true
	}
	if id != "" {
		current, currentErr := s.repository.GetChannel(ctx, projectID, id)
		if currentErr != nil {
			return nil, currentErr
		}
		for key, value := range current.SecretStatus {
			status[key] = value
		}
		for _, key := range input.DeleteSecretKeys {
			delete(status, key)
		}
		for key := range input.Secrets {
			if strings.TrimSpace(input.Secrets[key]) != "" {
				status[key] = true
			}
		}
	}
	record, err := s.repository.SaveChannel(ctx, repository.SaveAlarmNotificationChannelParams{ID: id, ProjectID: projectID, UserID: claims.UserID, Name: name, ChannelType: channelType, Config: alarmCloneMap(input.Config), SecretStatus: status, Secrets: encrypted, DeleteSecretKeys: alarmUniqueStrings(input.DeleteSecretKeys), IsEnabled: enabled})
	if err != nil {
		return nil, err
	}
	item := toAlarmChannel(*record)
	return &item, nil
}
func (s *AlarmPolicyService) DeleteChannel(ctx context.Context, claims *auth.Claims, projectID, id string) error {
	if err := s.writeAccess(claims, projectID); err != nil {
		return err
	}
	if id == "runtime_inapp" {
		return badAlarm("内置站内通知不能删除")
	}
	return s.repository.DeleteChannel(ctx, projectID, id)
}

// SyncConfig 接收后续节点同步代理上报的开发态配置增量，不记录节点来源。
func (s *AlarmPolicyService) SyncConfig(ctx context.Context, claims *auth.Claims, projectID string, input AlarmConfigSyncInput) (*AlarmConfigSyncResult, error) {
	if err := s.writeAccess(claims, projectID); err != nil {
		return nil, err
	}
	input.SyncEpoch = strings.TrimSpace(input.SyncEpoch)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.SyncEpoch == "" || len(input.SyncEpoch) > 100 {
		return nil, badAlarm("syncEpoch 不能为空且不能超过 100 个字符")
	}
	if input.IdempotencyKey == "" || len(input.IdempotencyKey) > 200 {
		return nil, badAlarm("idempotencyKey 不能为空且不能超过 200 个字符")
	}
	if input.Sequence <= 0 {
		return nil, badAlarm("sequence 必须为正整数")
	}
	if len(input.Operations) == 0 {
		return nil, badAlarm("同步操作不能为空")
	}
	operations := make([]repository.AlarmConfigSyncOperationParams, 0, len(input.Operations))
	for _, operation := range input.Operations {
		normalized, err := s.normalizeSyncOperation(ctx, projectID, claims.UserID, operation)
		if err != nil {
			return nil, err
		}
		operations = append(operations, normalized)
	}
	record, err := s.repository.ApplyConfigSync(ctx, projectID, claims.UserID, input.SyncEpoch, input.Sequence, input.IdempotencyKey, operations)
	if err != nil {
		return nil, err
	}
	return &AlarmConfigSyncResult{ConfigRevision: record.ConfigRevision, AcceptedSequence: record.AcceptedSequence, Idempotent: record.Idempotent}, nil
}

func (s *AlarmPolicyService) normalizeSyncOperation(ctx context.Context, projectID, actorID string, operation AlarmConfigSyncOperation) (repository.AlarmConfigSyncOperationParams, error) {
	resource := strings.TrimSpace(operation.Resource)
	action := strings.TrimSpace(operation.Action)
	if action != "upsert" && action != "delete" {
		return repository.AlarmConfigSyncOperationParams{}, badAlarm("同步操作 action 仅支持 upsert 或 delete")
	}
	id := strings.TrimSpace(operation.ID)
	if resource != "settings" && resource != "history_settings" {
		if _, err := uuid.Parse(id); err != nil {
			return repository.AlarmConfigSyncOperationParams{}, badAlarm("同步操作 id 格式无效")
		}
	}
	result := repository.AlarmConfigSyncOperationParams{Resource: resource, Action: action, ID: id}
	if action == "delete" {
		if resource != "group" && resource != "policy" && resource != "settings" && resource != "history_settings" && resource != "channel" {
			return result, badAlarm("同步资源类型不受支持")
		}
		if resource == "history_settings" && id != "history" {
			return result, badAlarm("报警历史同步资源 id 必须为 history")
		}
		return result, nil
	}
	switch resource {
	case "group":
		var data SaveAlarmPolicyGroupInput
		if err := decodeAlarmSyncData(operation.Data, &data); err != nil {
			return result, err
		}
		name, err := requiredName(data.Name, "目录名称")
		if err != nil {
			return result, err
		}
		data.ParentID = normalizeID(data.ParentID)
		if data.ParentID != nil && *data.ParentID == id {
			return result, badAlarm("目录不能移动到自身")
		}
		result.Group = &repository.SaveAlarmPolicyGroupParams{ID: id, ProjectID: projectID, UserID: actorID, Name: name, ParentID: data.ParentID, Description: normalizeText(data.Description), SortOrder: data.SortOrder}
	case "policy":
		var data SaveAlarmPolicyInput
		if err := decodeAlarmSyncData(operation.Data, &data); err != nil {
			return result, err
		}
		normalized, err := s.normalizePolicyForSync(ctx, projectID, data)
		if err != nil {
			return result, err
		}
		contract := buildAlarmContract(projectID, id, normalized)
		params := toSaveAlarmPolicyParams(projectID, id, actorID, normalized, contract)
		result.Policy = &params
	case "settings":
		var data SaveAlarmProjectSettingsInput
		if err := decodeAlarmSyncData(operation.Data, &data); err != nil {
			return result, err
		}
		if data.RepeatIntervalSeconds != nil && *data.RepeatIntervalSeconds <= 0 {
			return result, badAlarm("重复提醒间隔必须为正整数")
		}
		template := strings.TrimSpace(data.DefaultMessageTemplate)
		if template == "" {
			template = defaultAlarmMessageTemplate
		}
		result.Settings = &repository.SaveAlarmProjectSettingsParams{ProjectID: projectID, UserID: actorID, NotifyOnRaise: data.NotifyOnRaise, NotifyOnClear: data.NotifyOnClear, RepeatIntervalSeconds: data.RepeatIntervalSeconds, DefaultMessageTemplate: template, DefaultChannelIDs: alarmUniqueStrings(data.DefaultChannelIDs)}
	case "history_settings":
		if id != "history" {
			return result, badAlarm("报警历史同步资源 id 必须为 history")
		}
		var data SaveAlarmHistorySettingsInput
		if err := decodeAlarmSyncData(operation.Data, &data); err != nil {
			return result, err
		}
		if err := validateAlarmHistorySettings(data); err != nil {
			return result, err
		}
		result.HistorySettings = &repository.SaveAlarmHistorySettingsParams{ProjectID: projectID, UserID: actorID, IsEnabled: data.IsEnabled, RetentionDays: data.RetentionDays, StoreNotificationDeliveries: data.StoreNotificationDeliveries}
	case "channel":
		var data struct {
			SaveAlarmNotificationChannelInput
			SecretStatus map[string]any `json:"secretStatus"`
		}
		if err := decodeAlarmSyncData(operation.Data, &data); err != nil {
			return result, err
		}
		params, err := s.normalizeSyncChannel(projectID, actorID, id, data.SaveAlarmNotificationChannelInput, data.SecretStatus)
		if err != nil {
			return result, err
		}
		result.Channel = &params
	default:
		return result, badAlarm("同步资源类型不受支持")
	}
	return result, nil
}

func (s *AlarmPolicyService) normalizePolicyForSync(ctx context.Context, projectID string, input SaveAlarmPolicyInput) (SaveAlarmPolicyInput, error) {
	name, err := requiredName(input.Name, "策略名称")
	if err != nil {
		return input, err
	}
	input.Name = name
	input.Mode = strings.TrimSpace(input.Mode)
	if input.Mode == "" {
		input.Mode = "per_target"
	}
	if !alarmModes[input.Mode] {
		return input, badAlarm("报警类型不受支持")
	}
	input.GroupID = normalizeID(input.GroupID)
	enabled := valueOr(input.IsEnabled, input.Mode == "per_target")
	input.IsEnabled = &enabled
	bindings, category, err := s.normalizeBindings(ctx, projectID, input.Mode, input.Bindings)
	if err != nil {
		return input, err
	}
	input.Bindings = bindings
	input.DerivedExpression = strings.TrimSpace(input.DerivedExpression)
	if input.Mode == "derived" && input.DerivedExpression == "" {
		return input, badAlarm("组合报警必须填写计算表达式")
	}
	if input.Mode == "per_target" {
		input.DerivedExpression = ""
	}
	input.Conditions, err = normalizeConditions(input.Conditions, category, input.Mode)
	if err != nil {
		return input, err
	}
	input.Notification, err = normalizeNotificationShape(input.Notification)
	return input, err
}

func (s *AlarmPolicyService) normalizeSyncChannel(projectID, actorID, id string, input SaveAlarmNotificationChannelInput, secretStatus map[string]any) (repository.SaveAlarmNotificationChannelParams, error) {
	name, err := requiredName(input.Name, "渠道名称")
	if err != nil {
		return repository.SaveAlarmNotificationChannelParams{}, err
	}
	channelType := strings.TrimSpace(input.ChannelType)
	if err = validateChannelConfig(channelType, input.Config); err != nil {
		return repository.SaveAlarmNotificationChannelParams{}, err
	}
	enabled := valueOr(input.IsEnabled, true)
	status := alarmCloneMap(secretStatus)
	encrypted := make([]repository.EncryptedAlarmChannelSecret, 0, len(input.Secrets))
	if len(input.Secrets) > 0 && s.cipher == nil {
		return repository.SaveAlarmNotificationChannelParams{}, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "报警渠道密钥服务未初始化")
	}
	for key, value := range input.Secrets {
		key = strings.TrimSpace(key)
		if key == "" || strings.TrimSpace(value) == "" {
			continue
		}
		payload, cipherErr := s.cipher.Encrypt([]byte(value))
		if cipherErr != nil {
			return repository.SaveAlarmNotificationChannelParams{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "加密同步报警渠道密钥失败", cipherErr)
		}
		encrypted = append(encrypted, repository.EncryptedAlarmChannelSecret{Key: key, Value: payload, KeyVersion: s.cipher.KeyVersion()})
		status[key] = true
	}
	for _, key := range input.DeleteSecretKeys {
		delete(status, key)
	}
	return repository.SaveAlarmNotificationChannelParams{ID: id, ProjectID: projectID, UserID: actorID, Name: name, ChannelType: channelType, Config: alarmCloneMap(input.Config), SecretStatus: status, Secrets: encrypted, DeleteSecretKeys: alarmUniqueStrings(input.DeleteSecretKeys), IsEnabled: enabled}, nil
}

func decodeAlarmSyncData(raw json.RawMessage, target any) error {
	if len(raw) == 0 || string(raw) == "null" {
		return badAlarm("同步 upsert 操作必须提供 data")
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "同步操作 data 格式无效", err)
	}
	return nil
}

func (s *AlarmPolicyService) Contract(ctx context.Context, claims *auth.Claims, projectID, id string) (map[string]any, error) {
	policy, err := s.Get(ctx, claims, projectID, id)
	if err != nil {
		return nil, err
	}
	return alarmCloneMap(policy.Contract), nil
}
func (s *AlarmPolicyService) Test(ctx context.Context, claims *auth.Claims, projectID, id string, value any, context map[string]any) (*AlarmPolicyTrialResult, error) {
	policy, err := s.Get(ctx, claims, projectID, id)
	if err != nil {
		return nil, err
	}
	return evaluateAlarmConditions(policy.Conditions, value, context), nil
}

func (s *AlarmPolicyService) normalizePolicy(ctx context.Context, projectID string, input SaveAlarmPolicyInput) (SaveAlarmPolicyInput, error) {
	name, err := requiredName(input.Name, "策略名称")
	if err != nil {
		return input, err
	}
	input.Name = name
	input.Mode = strings.TrimSpace(input.Mode)
	if input.Mode == "" {
		input.Mode = "per_target"
	}
	if !alarmModes[input.Mode] {
		return input, badAlarm("报警类型不受支持")
	}
	if input.GroupID != nil {
		input.GroupID = normalizeID(input.GroupID)
		if input.GroupID != nil {
			if _, err = s.repository.GetGroup(ctx, projectID, *input.GroupID); err != nil {
				return input, err
			}
		}
	}
	enabled := input.Mode == "per_target"
	if input.IsEnabled != nil {
		enabled = *input.IsEnabled
	}
	input.IsEnabled = &enabled
	bindings, category, err := s.normalizeBindings(ctx, projectID, input.Mode, input.Bindings)
	if err != nil {
		return input, err
	}
	input.Bindings = bindings
	input.DerivedExpression = strings.TrimSpace(input.DerivedExpression)
	if input.Mode == "derived" && input.DerivedExpression == "" {
		return input, badAlarm("组合报警必须填写计算表达式")
	}
	if input.Mode == "per_target" {
		input.DerivedExpression = ""
	}
	conditions, err := normalizeConditions(input.Conditions, category, input.Mode)
	if err != nil {
		return input, err
	}
	input.Conditions = conditions
	notification, err := s.normalizeNotification(ctx, projectID, input.Notification)
	if err != nil {
		return input, err
	}
	input.Notification = notification
	return input, nil
}

func (s *AlarmPolicyService) normalizeBindings(ctx context.Context, projectID, mode string, bindings []AlarmBinding) ([]AlarmBinding, string, error) {
	if len(bindings) == 0 {
		return nil, "", badAlarm("请至少选择一个数据点")
	}
	ids := make([]string, 0, len(bindings))
	seen := map[string]bool{}
	for _, binding := range bindings {
		id := strings.TrimSpace(binding.DatapointID)
		if id == "" || seen[id] {
			if seen[id] {
				continue
			}
			return nil, "", badAlarm("数据点 ID 不能为空")
		}
		seen[id] = true
		ids = append(ids, id)
	}
	points, err := s.datapoints.GetByProjectAndIDs(ctx, projectID, ids)
	if err != nil {
		return nil, "", err
	}
	if len(points) != len(ids) {
		return nil, "", badAlarm("部分数据点不存在或不属于当前工程")
	}
	byID := map[string]repository.DataPointRecord{}
	for _, point := range points {
		byID[point.ID] = point
	}
	result := make([]AlarmBinding, 0, len(ids))
	category := ""
	keys := map[string]bool{}
	for _, source := range bindings {
		point, ok := byID[source.DatapointID]
		if !ok || !seen[source.DatapointID] {
			continue
		}
		seen[source.DatapointID] = false
		currentCategory := alarmDataCategory(point.DataType)
		if category == "" {
			category = currentCategory
		} else if mode == "per_target" && category != currentCategory {
			return nil, "", badAlarm("批量报警只能选择兼容的数据类型")
		}
		role := "target"
		var inputKey *string
		if mode == "derived" {
			role = "input"
			key := strings.TrimSpace(valueOrEmpty(source.InputKey))
			if key == "" || !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(key) {
				return nil, "", badAlarm("组合报警输入别名只能使用字母、数字和下划线")
			}
			if keys[key] {
				return nil, "", badAlarm("组合报警输入别名不能重复")
			}
			keys[key] = true
			inputKey = &key
		}
		result = append(result, AlarmBinding{DatapointID: point.ID, Path: point.Path, Name: point.Name, DataType: point.DataType, Role: role, InputKey: inputKey})
	}
	return result, category, nil
}

func normalizeConditions(conditions []AlarmCondition, category, mode string) ([]AlarmCondition, error) {
	if len(conditions) == 0 {
		return nil, badAlarm("请至少添加一个报警条件")
	}
	result := make([]AlarmCondition, 0, len(conditions))
	ids := map[string]bool{}
	for _, condition := range conditions {
		condition.Kind = strings.TrimSpace(condition.Kind)
		condition.Operator = strings.TrimSpace(condition.Operator)
		condition.Severity = strings.TrimSpace(condition.Severity)
		if !alarmKinds[condition.Kind] {
			return nil, badAlarm("报警条件类型不受支持")
		}
		if !alarmSeverities[condition.Severity] {
			return nil, badAlarm("报警等级不受支持")
		}
		if condition.TriggerDelayMS < 0 || condition.ClearDelayMS < 0 {
			return nil, badAlarm("触发和清除延时不能为负数")
		}
		if condition.Deadband < 0 {
			return nil, badAlarm("恢复死区不能为负数")
		}
		if mode == "per_target" && !conditionAllowedForCategory(condition.Kind, category) {
			return nil, badAlarm("报警条件与数据点类型不兼容")
		}
		if mode == "derived" && condition.Kind == "offline" {
			return nil, badAlarm("组合报警不支持离线条件")
		}
		if err := validateConditionParams(condition); err != nil {
			return nil, err
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
	return result, nil
}

func (s *AlarmPolicyService) normalizeNotification(ctx context.Context, projectID string, input AlarmNotificationSettings) (AlarmNotificationSettings, error) {
	normalized, err := normalizeNotificationShape(input)
	if err != nil {
		return normalized, err
	}
	if normalized.Mode == "custom" {
		if err = s.validateChannelIDs(ctx, projectID, normalized.ChannelIDs); err != nil {
			return normalized, err
		}
	}
	return normalized, nil
}

func normalizeNotificationShape(input AlarmNotificationSettings) (AlarmNotificationSettings, error) {
	input.Mode = strings.TrimSpace(input.Mode)
	if input.Mode == "" {
		input.Mode = "inherit"
	}
	if input.Mode != "inherit" && input.Mode != "off" && input.Mode != "custom" {
		return input, badAlarm("通知模式不受支持")
	}
	if input.Mode != "custom" {
		return AlarmNotificationSettings{Mode: input.Mode, ChannelIDs: []string{}}, nil
	}
	if input.NotifyOnRaise == nil || input.NotifyOnClear == nil {
		return input, badAlarm("单独通知设置必须明确触发和恢复通知")
	}
	if input.RepeatIntervalSeconds != nil && *input.RepeatIntervalSeconds <= 0 {
		return input, badAlarm("重复提醒间隔必须为正整数")
	}
	if len(input.ChannelIDs) == 0 {
		return input, badAlarm("单独通知设置必须选择通知渠道")
	}
	input.ChannelIDs = alarmUniqueStrings(input.ChannelIDs)
	input.MessageTemplate = strings.TrimSpace(input.MessageTemplate)
	return input, nil
}

func (s *AlarmPolicyService) validateChannelIDs(ctx context.Context, projectID string, ids []string) error {
	channels, err := s.repository.ListChannels(ctx, projectID)
	if err != nil {
		return err
	}
	allowed := map[string]bool{"runtime_inapp": true}
	for _, channel := range channels {
		if channel.IsEnabled {
			allowed[channel.ID] = true
		}
	}
	for _, id := range alarmUniqueStrings(ids) {
		if !allowed[id] {
			return badAlarm("通知渠道不存在、已停用或不属于当前工程")
		}
	}
	return nil
}

func validateConditionParams(c AlarmCondition) error {
	params := c.Params
	if params == nil {
		params = map[string]any{}
	}
	number := func(key string) (float64, bool) {
		value, ok := params[key]
		if !ok {
			return 0, false
		}
		return anyFloat(value)
	}
	switch c.Kind {
	case "threshold":
		if c.Operator != "gt" && c.Operator != "gte" && c.Operator != "lt" && c.Operator != "lte" {
			return badAlarm("阈值条件操作符不受支持")
		}
		if _, ok := number("threshold"); !ok {
			return badAlarm("阈值条件必须填写阈值")
		}
	case "range":
		if c.Operator != "between" && c.Operator != "outside" {
			return badAlarm("区间条件操作符不受支持")
		}
		lower, lok := number("lower")
		upper, uok := number("upper")
		if !lok || !uok || lower >= upper {
			return badAlarm("区间下限必须小于上限")
		}
	case "state":
		if c.Operator != "eq" && c.Operator != "ne" {
			return badAlarm("状态条件操作符不受支持")
		}
		if _, ok := params["expected"]; !ok {
			return badAlarm("状态条件必须填写期望值")
		}
	case "transition":
		if c.Operator != "changed" && c.Operator != "rising" && c.Operator != "falling" && c.Operator != "from_to" {
			return badAlarm("变化条件操作符不受支持")
		}
	case "text_match":
		if c.Operator != "eq" && c.Operator != "ne" && c.Operator != "contains" && c.Operator != "regex" {
			return badAlarm("文本条件操作符不受支持")
		}
		expected, ok := params["expected"].(string)
		if !ok || strings.TrimSpace(expected) == "" {
			return badAlarm("文本条件必须填写匹配内容")
		}
		if c.Operator == "regex" {
			if _, err := regexp.Compile(expected); err != nil {
				return badAlarm("正则表达式无效")
			}
		}
	case "rate_of_change":
		limit, lok := number("limit")
		window, wok := number("windowMs")
		if !lok || limit < 0 || !wok || window <= 0 {
			return badAlarm("变化率阈值不能为负且统计窗口必须为正数")
		}
	case "deviation":
		if _, ok := number("baseline"); !ok {
			return badAlarm("偏差条件必须填写基准值")
		}
		limit, ok := number("limit")
		if !ok || limit < 0 {
			return badAlarm("偏差值不能为负数")
		}
	case "expression":
		expression, ok := params["expression"].(string)
		if !ok || strings.TrimSpace(expression) == "" {
			return badAlarm("表达式条件不能为空")
		}
	}
	return nil
}

func validateChannelConfig(channelType string, config map[string]any) error {
	if channelType != "webhook" && channelType != "dingtalk" && channelType != "wecom" {
		return badAlarm("通知渠道类型不受支持")
	}
	raw, _ := config["webhookUrl"].(string)
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return badAlarm("通知渠道 Webhook 地址无效")
	}
	return nil
}

func evaluateAlarmConditions(conditions []AlarmCondition, value any, context map[string]any) *AlarmPolicyTrialResult {
	result := &AlarmPolicyTrialResult{State: "not_triggered", TriggeredConditions: []AlarmCondition{}, ConditionResults: []map[string]any{}}
	quality, _ := context["quality"].(string)
	offline, _ := context["offline"].(bool)
	for _, condition := range conditions {
		triggered, reason := evaluateAlarmCondition(condition, value, quality, offline, context)
		result.ConditionResults = append(result.ConditionResults, map[string]any{"conditionId": condition.ID, "triggered": triggered, "reason": reason})
		if triggered {
			result.Triggered = true
			result.TriggeredConditions = append(result.TriggeredConditions, condition)
		}
	}
	if result.Triggered {
		result.State = "triggered"
	} else if len(conditions) == 0 {
		result.State = "insufficient_input"
	}
	return result
}
func evaluateAlarmCondition(c AlarmCondition, value any, quality string, offline bool, context map[string]any) (bool, string) {
	if c.Kind == "offline" {
		return offline, "offline"
	}
	if offline || (quality != "" && quality != "good") {
		return false, "quality_paused"
	}
	switch c.Kind {
	case "threshold":
		v, ok := anyFloat(value)
		threshold, tok := anyFloat(c.Params["threshold"])
		if !ok || !tok {
			return false, "invalid_numeric_value"
		}
		switch c.Operator {
		case "gt":
			return v > threshold, "evaluated"
		case "gte":
			return v >= threshold, "evaluated"
		case "lt":
			return v < threshold, "evaluated"
		case "lte":
			return v <= threshold, "evaluated"
		}
	case "range":
		v, ok := anyFloat(value)
		lower, lok := anyFloat(c.Params["lower"])
		upper, uok := anyFloat(c.Params["upper"])
		if !ok || !lok || !uok {
			return false, "invalid_numeric_value"
		}
		inside := v >= lower && v <= upper
		return map[string]bool{"between": inside, "outside": !inside}[c.Operator], "evaluated"
	case "state":
		expected := c.Params["expected"]
		equal := fmt.Sprint(value) == fmt.Sprint(expected)
		if c.Operator == "ne" {
			equal = !equal
		}
		return equal, "evaluated"
	case "transition":
		previous := context["previousValue"]
		switch c.Operator {
		case "changed":
			return fmt.Sprint(previous) != fmt.Sprint(value), "evaluated"
		case "rising":
			return fmt.Sprint(previous) == "false" && fmt.Sprint(value) == "true", "evaluated"
		case "falling":
			return fmt.Sprint(previous) == "true" && fmt.Sprint(value) == "false", "evaluated"
		}
	case "text_match":
		actual := fmt.Sprint(value)
		expected := fmt.Sprint(c.Params["expected"])
		switch c.Operator {
		case "eq":
			return actual == expected, "evaluated"
		case "ne":
			return actual != expected, "evaluated"
		case "contains":
			return strings.Contains(actual, expected), "evaluated"
		case "regex":
			matched, _ := regexp.MatchString(expected, actual)
			return matched, "evaluated"
		}
	case "deviation":
		v, ok := anyFloat(value)
		baseline, bok := anyFloat(c.Params["baseline"])
		limit, lok := anyFloat(c.Params["limit"])
		if !ok || !bok || !lok {
			return false, "invalid_numeric_value"
		}
		return abs(v-baseline) > limit, "evaluated"
	}
	return false, "unsupported_trial"
}

func buildAlarmContract(projectID, policyID string, input SaveAlarmPolicyInput) map[string]any {
	return map[string]any{"schemaVersion": "alarm.policy.v1", "projectId": projectID, "policyId": policyID, "mode": input.Mode, "bindings": input.Bindings, "derivedExpression": input.DerivedExpression, "conditions": input.Conditions, "notification": input.Notification, "isEnabled": valueOr(input.IsEnabled, input.Mode == "per_target"), "qualityBehavior": "pause_invalid", "lifecycle": map[string]any{"autoClear": true, "acknowledgement": "audit_only"}}
}

func toSaveAlarmPolicyParams(projectID, id, userID string, input SaveAlarmPolicyInput, contract map[string]any) repository.SaveAlarmPolicyParams {
	bindings := make([]repository.AlarmPolicyBindingRecord, 0, len(input.Bindings))
	for _, binding := range input.Bindings {
		bindings = append(bindings, repository.AlarmPolicyBindingRecord{DatapointID: binding.DatapointID, Role: binding.Role, InputKey: binding.InputKey})
	}
	conditions := make([]repository.AlarmPolicyConditionRecord, 0, len(input.Conditions))
	for _, condition := range input.Conditions {
		conditions = append(conditions, repository.AlarmPolicyConditionRecord{ID: condition.ID, Kind: condition.Kind, Operator: condition.Operator, Label: condition.Label, Severity: condition.Severity, Params: condition.Params, TriggerDelayMS: condition.TriggerDelayMS, ClearDelayMS: condition.ClearDelayMS, Deadband: condition.Deadband})
	}
	return repository.SaveAlarmPolicyParams{ID: id, ProjectID: projectID, UserID: userID, GroupID: input.GroupID, Name: input.Name, Description: normalizeText(input.Description), Mode: input.Mode, DerivedExpression: input.DerivedExpression, NotificationMode: input.Notification.Mode, NotifyOnRaise: input.Notification.NotifyOnRaise, NotifyOnClear: input.Notification.NotifyOnClear, RepeatIntervalSeconds: input.Notification.RepeatIntervalSeconds, NotificationChannelIDs: input.Notification.ChannelIDs, MessageTemplate: input.Notification.MessageTemplate, IsEnabled: valueOr(input.IsEnabled, input.Mode == "per_target"), Contract: contract, Bindings: bindings, Conditions: conditions}
}

func toAlarmPolicy(record repository.AlarmPolicyRecord) AlarmPolicy {
	bindings := make([]AlarmBinding, 0, len(record.Bindings))
	for _, binding := range record.Bindings {
		bindings = append(bindings, AlarmBinding{DatapointID: binding.DatapointID, Path: binding.Path, Name: binding.Name, DataType: binding.DataType, Role: binding.Role, InputKey: binding.InputKey})
	}
	conditions := make([]AlarmCondition, 0, len(record.Conditions))
	for _, condition := range record.Conditions {
		conditions = append(conditions, AlarmCondition{ID: condition.ID, Kind: condition.Kind, Operator: condition.Operator, Label: condition.Label, Severity: condition.Severity, Params: alarmCloneMap(condition.Params), TriggerDelayMS: condition.TriggerDelayMS, ClearDelayMS: condition.ClearDelayMS, Deadband: condition.Deadband})
	}
	return AlarmPolicy{ID: record.ID, ProjectID: record.ProjectID, GroupID: record.GroupID, GroupName: record.GroupName, Name: record.Name, Description: record.Description, Mode: record.Mode, Bindings: bindings, DerivedExpression: record.DerivedExpression, Conditions: conditions, Notification: AlarmNotificationSettings{Mode: record.NotificationMode, NotifyOnRaise: record.NotifyOnRaise, NotifyOnClear: record.NotifyOnClear, RepeatIntervalSeconds: record.RepeatIntervalSeconds, ChannelIDs: record.NotificationChannelIDs, MessageTemplate: record.MessageTemplate}, IsEnabled: record.IsEnabled, Revision: record.Revision, Contract: alarmCloneMap(record.Contract), CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}
func toAlarmGroup(record repository.AlarmPolicyGroupRecord) AlarmPolicyGroup {
	return AlarmPolicyGroup{ID: record.ID, ProjectID: record.ProjectID, Name: record.Name, ParentID: record.ParentID, Description: record.Description, SortOrder: record.SortOrder, FullPath: record.FullPath, HasChildren: record.HasChildren, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}
func toAlarmSettings(record repository.AlarmProjectSettingsRecord) AlarmProjectSettings {
	return AlarmProjectSettings{ProjectID: record.ProjectID, NotifyOnRaise: record.NotifyOnRaise, NotifyOnClear: record.NotifyOnClear, RepeatIntervalSeconds: record.RepeatIntervalSeconds, DefaultMessageTemplate: record.DefaultMessageTemplate, DefaultChannelIDs: record.DefaultChannelIDs, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}
func toAlarmHistorySettings(record repository.AlarmHistorySettingsRecord) AlarmHistorySettings {
	return AlarmHistorySettings{ProjectID: record.ProjectID, IsEnabled: record.IsEnabled, RetentionDays: record.RetentionDays, StoreNotificationDeliveries: record.StoreNotificationDeliveries, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}

func validateAlarmHistorySettings(input SaveAlarmHistorySettingsInput) error {
	if input.RetentionDays != nil && *input.RetentionDays <= 0 {
		return badAlarm("报警历史保留天数必须为正整数或永久")
	}
	return nil
}
func defaultAlarmHistorySettings(projectID string) AlarmHistorySettings {
	retentionDays := defaultAlarmHistoryRetentionDays
	return AlarmHistorySettings{ProjectID: projectID, IsEnabled: true, RetentionDays: &retentionDays, StoreNotificationDeliveries: true}
}
func toAlarmChannel(record repository.AlarmNotificationChannelRecord) AlarmNotificationChannel {
	return AlarmNotificationChannel{ID: record.ID, ProjectID: record.ProjectID, Name: record.Name, ChannelType: record.ChannelType, Config: alarmCloneMap(record.Config), SecretStatus: alarmCloneMap(record.SecretStatus), IsEnabled: record.IsEnabled, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}

func (s *AlarmPolicyService) readAccess(claims *auth.Claims, projectID string) error {
	if s == nil || s.repository == nil || s.datapoints == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "报警服务未初始化")
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
func (s *AlarmPolicyService) writeAccess(claims *auth.Claims, projectID string) error {
	if err := s.readAccess(claims, projectID); err != nil {
		return err
	}
	return validateUserID(claims.UserID)
}
func validateAlarmFilters(f AlarmPolicyListFilter) error {
	if f.Mode != "" && !alarmModes[f.Mode] {
		return badAlarm("报警类型筛选不受支持")
	}
	if f.Severity != "" && !alarmSeverities[f.Severity] {
		return badAlarm("报警等级筛选不受支持")
	}
	if f.ConditionKind != "" && !alarmKinds[f.ConditionKind] {
		return badAlarm("报警条件筛选不受支持")
	}
	return nil
}
func validatePolicyReady(record repository.AlarmPolicyRecord) error {
	if len(record.Bindings) == 0 || len(record.Conditions) == 0 {
		return badAlarm("报警策略尚未配置完整")
	}
	if record.Mode == "derived" && strings.TrimSpace(record.DerivedExpression) == "" {
		return badAlarm("组合报警尚未配置表达式")
	}
	return nil
}
func conditionAllowedForCategory(kind, category string) bool {
	if kind == "offline" {
		return true
	}
	switch category {
	case "number":
		return kind == "threshold" || kind == "range" || kind == "rate_of_change" || kind == "deviation"
	case "boolean":
		return kind == "state" || kind == "transition"
	default:
		return kind == "state" || kind == "text_match"
	}
}
func alarmDataCategory(dataType string) string {
	value := strings.ToLower(strings.TrimSpace(dataType))
	if numericTypes[value] {
		return "number"
	}
	if value == "bool" || value == "boolean" {
		return "boolean"
	}
	return "text"
}
func defaultConditionLabel(c AlarmCondition) string {
	labels := map[string]string{"threshold": "阈值报警", "range": "区间报警", "state": "状态报警", "transition": "状态变化报警", "text_match": "文本报警", "rate_of_change": "变化率报警", "deviation": "偏差报警", "offline": "数据点离线", "expression": "表达式报警"}
	return labels[c.Kind]
}
func groupIsDescendant(groups []repository.AlarmPolicyGroupRecord, parentID, candidateID string) bool {
	parents := map[string]*string{}
	for _, group := range groups {
		parents[group.ID] = group.ParentID
	}
	for current := candidateID; current != ""; {
		if current == parentID {
			return true
		}
		parent := parents[current]
		if parent == nil {
			return false
		}
		current = *parent
	}
	return false
}
func requiredName(value, label string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", badAlarm(label + "不能为空")
	}
	if len([]rune(value)) > 100 {
		return "", badAlarm(label + "不能超过 100 个字符")
	}
	return value, nil
}
func normalizeID(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
func normalizeText(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
func alarmUniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}
func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
func valueOr(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}
func positive(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}
func anyFloat(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case jsonNumber:
		parsed, err := strconv.ParseFloat(string(typed), 64)
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	}
	return 0, false
}

type jsonNumber string

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
func alarmCloneMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

// boolFromAny 是同包协议工作台共用的宽松布尔读取函数。
func boolFromAny(value any) bool {
	result, ok := value.(bool)
	return ok && result
}
func badAlarm(message string) error {
	return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, message)
}
