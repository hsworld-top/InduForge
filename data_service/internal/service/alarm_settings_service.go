package service

import (
	"context"
	"encoding/json"
	"errors"
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

var alarmSeverityKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,29}$`)

var (
	alarmConditionKinds = map[string]bool{"threshold": true, "range": true, "state": true, "transition": true, "text_match": true, "rate_of_change": true, "deviation": true, "offline": true, "quality": true, "stale": true}
	numericTypes        = map[string]bool{"number": true, "integer": true, "float": true, "double": true, "decimal": true, "int": true, "int8": true, "int16": true, "int32": true, "int64": true, "uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true, "float32": true, "float64": true, "numeric": true}
)

type AlarmGroup struct {
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

type AlarmGroupListResult struct {
	List       []AlarmGroup `json:"list"`
	Pagination Pagination   `json:"pagination"`
}

type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
}

type SaveAlarmGroupInput struct {
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

type AlarmSeverityDefinition struct {
	Key         string `json:"key"`
	DisplayName string `json:"displayName"`
	Color       string `json:"color"`
	SortOrder   int    `json:"sortOrder"`
	IsBuiltin   bool   `json:"isBuiltin"`
}

type AlarmEscalationRule struct {
	ID                    string `json:"id"`
	SourceSeverity        string `json:"sourceSeverity"`
	TargetSeverity        string `json:"targetSeverity"`
	UnacknowledgedSeconds int    `json:"unacknowledgedSeconds"`
	IsEnabled             bool   `json:"isEnabled"`
}

type AlarmLevelSettings struct {
	ProjectID           string                    `json:"projectId"`
	SeverityDefinitions []AlarmSeverityDefinition `json:"severityDefinitions"`
	EscalationRules     []AlarmEscalationRule     `json:"escalationRules"`
}

type SaveAlarmLevelSettingsInput struct {
	SeverityDefinitions []AlarmSeverityDefinition `json:"severityDefinitions"`
	EscalationRules     []AlarmEscalationRule     `json:"escalationRules"`
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
	ID                 string         `json:"id"`
	ProjectID          string         `json:"projectId"`
	Name               string         `json:"name"`
	ChannelType        string         `json:"channelType"`
	Config             map[string]any `json:"config"`
	SecretStatus       map[string]any `json:"secretStatus"`
	IsEnabled          bool           `json:"isEnabled"`
	LastTestStatus     string         `json:"lastTestStatus"`
	LastTestedAt       *time.Time     `json:"lastTestedAt,omitempty"`
	LastTestDurationMS *int           `json:"lastTestDurationMs,omitempty"`
	LastTestMessage    *string        `json:"lastTestMessage,omitempty"`
	CreatedAt          time.Time      `json:"createdAt"`
	UpdatedAt          time.Time      `json:"updatedAt"`
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

type AlarmConditionTrialResult struct {
	Triggered           bool             `json:"triggered"`
	State               string           `json:"state"`
	TriggeredConditions []AlarmCondition `json:"triggeredConditions"`
	ConditionResults    []map[string]any `json:"conditionResults"`
	Message             string           `json:"message,omitempty"`
}

type AlarmSettingsService struct {
	repository *repository.AlarmRepository
	datapoints *repository.DataPointRepository
	cipher     *security.AlarmSecretCipher
	alarmItems *AlarmItemService
}

func (s *AlarmSettingsService) SetAlarmItemService(alarmItems *AlarmItemService) {
	s.alarmItems = alarmItems
}

func NewAlarmSettingsService(repo *repository.AlarmRepository, datapoints *repository.DataPointRepository, ciphers ...*security.AlarmSecretCipher) *AlarmSettingsService {
	var cipher *security.AlarmSecretCipher
	if len(ciphers) > 0 {
		cipher = ciphers[0]
	}
	return &AlarmSettingsService{repository: repo, datapoints: datapoints, cipher: cipher}
}

func (s *AlarmSettingsService) ListGroups(ctx context.Context, claims *auth.Claims, projectID, search string, parentID *string, page, pageSize int) (*AlarmGroupListResult, error) {
	if err := s.readAccess(claims, projectID); err != nil {
		return nil, err
	}
	records, total, err := s.repository.ListGroups(ctx, projectID, repository.AlarmGroupListFilter{Search: search, ParentID: parentID, Page: page, PageSize: pageSize})
	if err != nil {
		return nil, err
	}
	items := make([]AlarmGroup, 0, len(records))
	for _, record := range records {
		items = append(items, toAlarmGroup(record))
	}
	return &AlarmGroupListResult{List: items, Pagination: Pagination{Page: positive(page, 1), PageSize: positive(pageSize, 50), Total: total}}, nil
}

func (s *AlarmSettingsService) ListAllGroups(ctx context.Context, claims *auth.Claims, projectID string) ([]AlarmGroup, error) {
	if err := s.readAccess(claims, projectID); err != nil {
		return nil, err
	}
	records, err := s.repository.ListAllGroups(ctx, projectID)
	if err != nil {
		return nil, err
	}
	items := make([]AlarmGroup, 0, len(records))
	for _, record := range records {
		items = append(items, toAlarmGroup(record))
	}
	return items, nil
}

// ListGroupTree 为目录选择器返回完整层级，避免用分页结果拼出不完整的父子树。
func (s *AlarmSettingsService) ListGroupTree(ctx context.Context, claims *auth.Claims, projectID string) (*AlarmGroupListResult, error) {
	items, err := s.ListAllGroups(ctx, claims, projectID)
	if err != nil {
		return nil, err
	}
	return &AlarmGroupListResult{
		List:       items,
		// 空树也必须返回合法分页契约，pageSize=0 会让统一列表 Schema 拒绝响应。
		Pagination: Pagination{Page: 1, PageSize: positive(len(items), 1), Total: len(items)},
	}, nil
}

func (s *AlarmSettingsService) CreateGroup(ctx context.Context, claims *auth.Claims, projectID string, input SaveAlarmGroupInput) (*AlarmGroup, error) {
	return s.saveGroup(ctx, claims, projectID, "", input)
}
func (s *AlarmSettingsService) UpdateGroup(ctx context.Context, claims *auth.Claims, projectID, id string, input SaveAlarmGroupInput) (*AlarmGroup, error) {
	return s.saveGroup(ctx, claims, projectID, id, input)
}
func (s *AlarmSettingsService) saveGroup(ctx context.Context, claims *auth.Claims, projectID, id string, input SaveAlarmGroupInput) (*AlarmGroup, error) {
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
	record, err := s.repository.SaveGroup(ctx, repository.SaveAlarmGroupParams{ID: id, ProjectID: projectID, UserID: claims.UserID, Name: name, ParentID: normalizeID(input.ParentID), Description: normalizeText(input.Description), SortOrder: input.SortOrder})
	if err != nil {
		return nil, err
	}
	result := toAlarmGroup(*record)
	return &result, nil
}
func (s *AlarmSettingsService) DeleteGroup(ctx context.Context, claims *auth.Claims, projectID, id string) error {
	if err := s.writeAccess(claims, projectID); err != nil {
		return err
	}
	return s.repository.DeleteGroup(ctx, projectID, id)
}

func (s *AlarmSettingsService) GetSettings(ctx context.Context, claims *auth.Claims, projectID string) (*AlarmProjectSettings, error) {
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
func (s *AlarmSettingsService) UpdateSettings(ctx context.Context, claims *auth.Claims, projectID string, input SaveAlarmProjectSettingsInput) (*AlarmProjectSettings, error) {
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

func (s *AlarmSettingsService) GetLevelSettings(ctx context.Context, claims *auth.Claims, projectID string) (*AlarmLevelSettings, error) {
	if err := s.readAccess(claims, projectID); err != nil {
		return nil, err
	}
	record, err := s.repository.GetProjectSettings(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if record == nil || len(record.SeverityDefinitions) == 0 {
		return &AlarmLevelSettings{ProjectID: projectID, SeverityDefinitions: defaultAlarmSeverityDefinitions(), EscalationRules: []AlarmEscalationRule{}}, nil
	}
	return alarmLevelSettingsFromRecord(*record), nil
}

func (s *AlarmSettingsService) UpdateLevelSettings(ctx context.Context, claims *auth.Claims, projectID string, input SaveAlarmLevelSettingsInput) (*AlarmLevelSettings, error) {
	if err := s.writeAccess(claims, projectID); err != nil {
		return nil, err
	}
	normalized, err := normalizeAlarmLevelSettings(projectID, input)
	if err != nil {
		return nil, err
	}
	items, err := s.repository.ListAllAlarmItems(ctx, projectID)
	if err != nil {
		return nil, err
	}
	allowed := make(map[string]bool, len(normalized.SeverityDefinitions))
	for _, definition := range normalized.SeverityDefinitions {
		allowed[definition.Key] = true
	}
	for _, item := range items {
		for _, condition := range item.Conditions {
			if !allowed[condition.Severity] {
				return nil, badAlarm("正在使用的报警级别不能删除")
			}
		}
	}
	record, err := s.repository.SaveLevelSettings(ctx, repository.SaveAlarmLevelSettingsParams{
		ProjectID:           projectID,
		UserID:              claims.UserID,
		SeverityDefinitions: toAlarmSeverityDefinitionRecords(normalized.SeverityDefinitions),
		EscalationRules:     toAlarmEscalationRuleRecords(normalized.EscalationRules),
	})
	if err != nil {
		return nil, err
	}
	return alarmLevelSettingsFromRecord(*record), nil
}

func defaultAlarmSeverityDefinitions() []AlarmSeverityDefinition {
	return []AlarmSeverityDefinition{
		{Key: "info", DisplayName: "提示", Color: "#3b82f6", SortOrder: 10, IsBuiltin: true},
		{Key: "warning", DisplayName: "警告", Color: "#f59e0b", SortOrder: 20, IsBuiltin: true},
		{Key: "major", DisplayName: "重要", Color: "#f97316", SortOrder: 30, IsBuiltin: true},
		{Key: "critical", DisplayName: "紧急", Color: "#ef4444", SortOrder: 40, IsBuiltin: true},
	}
}

func normalizeAlarmLevelSettings(projectID string, input SaveAlarmLevelSettingsInput) (*AlarmLevelSettings, error) {
	if len(input.SeverityDefinitions) < 4 || len(input.SeverityDefinitions) > 20 {
		return nil, badAlarm("报警级别数量必须在 4 到 20 之间")
	}
	builtin := map[string]bool{"info": true, "warning": true, "major": true, "critical": true}
	seen, order := map[string]bool{}, map[string]int{}
	definitions := make([]AlarmSeverityDefinition, 0, len(input.SeverityDefinitions))
	for _, source := range input.SeverityDefinitions {
		item := source
		item.Key = strings.ToLower(strings.TrimSpace(item.Key))
		item.DisplayName = strings.TrimSpace(item.DisplayName)
		item.Color = strings.ToLower(strings.TrimSpace(item.Color))
		if !alarmSeverityKeyPattern.MatchString(item.Key) || seen[item.Key] {
			return nil, badAlarm("报警级别标识格式无效或重复")
		}
		if item.DisplayName == "" || len([]rune(item.DisplayName)) > 20 {
			return nil, badAlarm("报警级别名称不能为空且不能超过 20 个字符")
		}
		if !regexp.MustCompile(`^#[0-9a-f]{6}$`).MatchString(item.Color) {
			return nil, badAlarm("报警级别颜色必须使用六位十六进制颜色")
		}
		if builtin[item.Key] != item.IsBuiltin {
			return nil, badAlarm("内置报警级别不能删除或改为自定义级别")
		}
		seen[item.Key], order[item.Key] = true, item.SortOrder
		definitions = append(definitions, item)
	}
	for key := range builtin {
		if !seen[key] {
			return nil, badAlarm("提示、警告、重要和紧急四个内置级别不能删除")
		}
	}
	sort.SliceStable(definitions, func(i, j int) bool { return definitions[i].SortOrder < definitions[j].SortOrder })
	rules := make([]AlarmEscalationRule, 0, len(input.EscalationRules))
	ruleKeys := map[string]bool{}
	for _, source := range input.EscalationRules {
		item := source
		item.SourceSeverity = strings.TrimSpace(item.SourceSeverity)
		item.TargetSeverity = strings.TrimSpace(item.TargetSeverity)
		if !seen[item.SourceSeverity] || !seen[item.TargetSeverity] || item.SourceSeverity == item.TargetSeverity {
			return nil, badAlarm("报警升级规则引用的级别无效")
		}
		if order[item.TargetSeverity] <= order[item.SourceSeverity] {
			return nil, badAlarm("报警只能升级到更高等级")
		}
		if item.UnacknowledgedSeconds < 1 || item.UnacknowledgedSeconds > 604800 {
			return nil, badAlarm("未确认升级时间必须在 1 秒到 7 天之间")
		}
		if item.ID == "" {
			item.ID = uuid.NewString()
		} else if _, err := uuid.Parse(item.ID); err != nil {
			return nil, badAlarm("报警升级规则 ID 格式无效")
		}
		key := item.SourceSeverity + ":" + strconv.Itoa(item.UnacknowledgedSeconds)
		if ruleKeys[key] {
			return nil, badAlarm("同一级别在相同时间只能配置一条升级规则")
		}
		ruleKeys[key] = true
		rules = append(rules, item)
	}
	return &AlarmLevelSettings{ProjectID: projectID, SeverityDefinitions: definitions, EscalationRules: rules}, nil
}

func alarmLevelSettingsFromRecord(record repository.AlarmProjectSettingsRecord) *AlarmLevelSettings {
	definitions := make([]AlarmSeverityDefinition, 0, len(record.SeverityDefinitions))
	for _, item := range record.SeverityDefinitions {
		definitions = append(definitions, AlarmSeverityDefinition{Key: item.Key, DisplayName: item.DisplayName, Color: item.Color, SortOrder: item.SortOrder, IsBuiltin: item.IsBuiltin})
	}
	rules := make([]AlarmEscalationRule, 0, len(record.EscalationRules))
	for _, item := range record.EscalationRules {
		rules = append(rules, AlarmEscalationRule{ID: item.ID, SourceSeverity: item.SourceSeverity, TargetSeverity: item.TargetSeverity, UnacknowledgedSeconds: item.UnacknowledgedSeconds, IsEnabled: item.IsEnabled})
	}
	return &AlarmLevelSettings{ProjectID: record.ProjectID, SeverityDefinitions: definitions, EscalationRules: rules}
}

func toAlarmSeverityDefinitionRecords(items []AlarmSeverityDefinition) []repository.AlarmSeverityDefinitionRecord {
	result := make([]repository.AlarmSeverityDefinitionRecord, 0, len(items))
	for _, item := range items {
		result = append(result, repository.AlarmSeverityDefinitionRecord{Key: item.Key, DisplayName: item.DisplayName, Color: item.Color, SortOrder: item.SortOrder, IsBuiltin: item.IsBuiltin})
	}
	return result
}

func toAlarmEscalationRuleRecords(items []AlarmEscalationRule) []repository.AlarmEscalationRuleRecord {
	result := make([]repository.AlarmEscalationRuleRecord, 0, len(items))
	for _, item := range items {
		result = append(result, repository.AlarmEscalationRuleRecord{ID: item.ID, SourceSeverity: item.SourceSeverity, TargetSeverity: item.TargetSeverity, UnacknowledgedSeconds: item.UnacknowledgedSeconds, IsEnabled: item.IsEnabled})
	}
	return result
}

func (s *AlarmSettingsService) GetHistorySettings(ctx context.Context, claims *auth.Claims, projectID string) (*AlarmHistorySettings, error) {
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

func (s *AlarmSettingsService) UpdateHistorySettings(ctx context.Context, claims *auth.Claims, projectID string, input SaveAlarmHistorySettingsInput) (*AlarmHistorySettings, error) {
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

func (s *AlarmSettingsService) ListChannels(ctx context.Context, claims *auth.Claims, projectID string) ([]AlarmNotificationChannel, error) {
	if err := s.readAccess(claims, projectID); err != nil {
		return nil, err
	}
	records, err := s.repository.ListChannels(ctx, projectID)
	if err != nil {
		return nil, err
	}
	items := []AlarmNotificationChannel{defaultAlarmNotificationChannel(projectID)}
	for _, record := range records {
		items = append(items, toAlarmChannel(record))
	}
	return items, nil
}

func defaultAlarmNotificationChannel(projectID string) AlarmNotificationChannel {
	return AlarmNotificationChannel{ID: "runtime_inapp", ProjectID: projectID, Name: "运行端站内通知", ChannelType: "runtime_inapp", Config: map[string]any{"recipientScope": "project_runtime_users"}, SecretStatus: map[string]any{}, IsEnabled: true, LastTestStatus: "not_tested"}
}
func (s *AlarmSettingsService) CreateChannel(ctx context.Context, claims *auth.Claims, projectID string, input SaveAlarmNotificationChannelInput) (*AlarmNotificationChannel, error) {
	return s.saveChannel(ctx, claims, projectID, "", input)
}
func (s *AlarmSettingsService) UpdateChannel(ctx context.Context, claims *auth.Claims, projectID, id string, input SaveAlarmNotificationChannelInput) (*AlarmNotificationChannel, error) {
	if id == "runtime_inapp" {
		return nil, badAlarm("内置站内通知不能修改")
	}
	return s.saveChannel(ctx, claims, projectID, id, input)
}
func (s *AlarmSettingsService) saveChannel(ctx context.Context, claims *auth.Claims, projectID, id string, input SaveAlarmNotificationChannelInput) (*AlarmNotificationChannel, error) {
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
func (s *AlarmSettingsService) DeleteChannel(ctx context.Context, claims *auth.Claims, projectID, id string) error {
	if err := s.writeAccess(claims, projectID); err != nil {
		return err
	}
	if id == "runtime_inapp" {
		return badAlarm("内置站内通知不能删除")
	}
	return s.repository.DeleteChannel(ctx, projectID, id)
}

func (s *AlarmSettingsService) TestChannel(ctx context.Context, claims *auth.Claims, projectID, id string) (*AlarmNotificationChannel, error) {
	if err := s.writeAccess(claims, projectID); err != nil {
		return nil, err
	}
	if id == "runtime_inapp" {
		return nil, badAlarm("内置站内通知不需要测试")
	}
	record, err := s.repository.GetChannel(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	startedAt := time.Now()
	testErr := sendAlarmChannelTest(ctx, *record)
	duration := int(time.Since(startedAt).Milliseconds())
	status, message := "succeeded", "测试消息已送达目标服务器"
	if testErr != nil {
		status, message = "failed", sanitizeAlarmTestMessage(testErr.Error())
	}
	updated, saveErr := s.repository.SaveChannelTestResult(ctx, projectID, id, status, duration, message)
	if saveErr != nil {
		return nil, saveErr
	}
	if testErr != nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, message)
	}
	item := toAlarmChannel(*updated)
	return &item, nil
}

// SyncConfig 接收后续节点同步代理上报的开发态配置增量，不记录节点来源。
func (s *AlarmSettingsService) SyncConfig(ctx context.Context, claims *auth.Claims, projectID string, input AlarmConfigSyncInput) (*AlarmConfigSyncResult, error) {
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

func (s *AlarmSettingsService) normalizeSyncOperation(ctx context.Context, projectID, actorID string, operation AlarmConfigSyncOperation) (repository.AlarmConfigSyncOperationParams, error) {
	resource := strings.TrimSpace(operation.Resource)
	action := strings.TrimSpace(operation.Action)
	if action != "upsert" && action != "delete" {
		return repository.AlarmConfigSyncOperationParams{}, badAlarm("同步操作 action 仅支持 upsert 或 delete")
	}
	id := strings.TrimSpace(operation.ID)
	if resource != "project_settings" && resource != "history_settings" {
		if _, err := uuid.Parse(id); err != nil {
			return repository.AlarmConfigSyncOperationParams{}, badAlarm("同步操作 id 格式无效")
		}
	}
	result := repository.AlarmConfigSyncOperationParams{Resource: resource, Action: action, ID: id}
	if action == "delete" {
		if resource != "group" && resource != "alarm_item" && resource != "project_settings" && resource != "history_settings" && resource != "channel" {
			return result, badAlarm("同步资源类型不受支持")
		}
		if resource == "history_settings" && id != "history" {
			return result, badAlarm("报警历史同步资源 id 必须为 history")
		}
		return result, nil
	}
	switch resource {
	case "group":
		var data SaveAlarmGroupInput
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
		result.Group = &repository.SaveAlarmGroupParams{ID: id, ProjectID: projectID, UserID: actorID, Name: name, ParentID: data.ParentID, Description: normalizeText(data.Description), SortOrder: data.SortOrder}
	case "alarm_item":
		if s.alarmItems == nil {
			return result, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "报警项同步服务未初始化")
		}
		var data SaveAlarmItemInput
		if err := decodeAlarmSyncData(operation.Data, &data); err != nil {
			return result, err
		}
		_, currentErr := s.repository.GetAlarmItem(ctx, projectID, id)
		isCreate := false
		if currentErr != nil {
			var appErr *apperrors.AppError
			if !errors.As(currentErr, &appErr) || appErr.Code != apperrors.ErrorCodeNotFound {
				return result, currentErr
			}
			isCreate = true
		}
		normalizeItemID := id
		if isCreate {
			normalizeItemID = ""
		}
		normalized, params, validation, err := s.alarmItems.normalize(ctx, projectID, normalizeItemID, data)
		if err != nil {
			return result, err
		}
		if err = acknowledgeAlarmWarnings(normalized, validation); err != nil {
			return result, err
		}
		params.UserID = actorID
		params.ID = id
		params.IsCreate = isCreate
		params.Contract = buildAlarmItemContract(projectID, id, normalized, params, params.TriggerFingerprint)
		result.AlarmItem = &params
	case "project_settings":
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

func (s *AlarmSettingsService) normalizeSyncChannel(projectID, actorID, id string, input SaveAlarmNotificationChannelInput, secretStatus map[string]any) (repository.SaveAlarmNotificationChannelParams, error) {
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

func (s *AlarmSettingsService) validateChannelIDs(ctx context.Context, projectID string, ids []string) error {
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
		if c.Operator == "from_to" {
			from, fromOK := params["from"]
			to, toOK := params["to"]
			if !fromOK || !toOK || fmt.Sprint(from) == fmt.Sprint(to) {
				return badAlarm("指定状态变化必须填写不同的起始值和目标值")
			}
		}
		if c.TriggerDelayMS != 0 {
			return badAlarm("状态变化报警不支持触发延时")
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
		if c.Operator != "gt" {
			return badAlarm("变化率报警只支持超过设定速率")
		}
		limit, lok := number("limit")
		window, wok := number("windowMs")
		direction, _ := params["direction"].(string)
		if !lok || limit <= 0 || !wok || window <= 0 || (direction != "rise" && direction != "fall" && direction != "absolute") {
			return badAlarm("变化率必须选择方向，并填写正数阈值和统计窗口")
		}
	case "deviation":
		if _, ok := number("baseline"); !ok {
			return badAlarm("偏差条件必须填写基准值")
		}
		limit, ok := number("limit")
		if !ok || limit < 0 {
			return badAlarm("偏差值不能为负数")
		}
	case "quality":
		if c.Operator != "in" {
			return badAlarm("质量报警操作符不受支持")
		}
		qualities := alarmStringList(params["qualities"])
		if len(qualities) == 0 {
			return badAlarm("质量报警至少选择一种非良好质量")
		}
		seen := map[string]bool{}
		for _, quality := range qualities {
			if quality != "bad" && quality != "unknown" {
				return badAlarm("质量报警只支持 bad 或 unknown")
			}
			seen[quality] = true
		}
		if len(seen) != len(qualities) {
			return badAlarm("质量报警不能重复选择质量")
		}
	case "stale":
		if c.Operator != "age_gte" {
			return badAlarm("数据陈旧报警操作符不受支持")
		}
		maxAge, ok := number("maxAgeMs")
		if !ok || maxAge <= 0 {
			return badAlarm("数据陈旧时间必须为正数")
		}
	}
	return nil
}

func alarmStringList(value any) []string {
	result := []string{}
	switch raw := value.(type) {
	case []string:
		for _, item := range raw {
			result = append(result, strings.TrimSpace(item))
		}
	case []any:
		for _, item := range raw {
			result = append(result, strings.TrimSpace(fmt.Sprint(item)))
		}
	}
	return result
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

func evaluateAlarmConditions(conditions []AlarmCondition, value any, context map[string]any) *AlarmConditionTrialResult {
	result := &AlarmConditionTrialResult{State: "not_triggered", TriggeredConditions: []AlarmCondition{}, ConditionResults: []map[string]any{}}
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
	active, _ := context["alarmActive"].(bool)
	switch c.Kind {
	case "threshold":
		v, ok := anyFloat(value)
		threshold, tok := anyFloat(c.Params["threshold"])
		if !ok || !tok {
			return false, "invalid_numeric_value"
		}
		switch c.Operator {
		case "gt":
			if active {
				return v > threshold-c.Deadband, "evaluated"
			}
			return v > threshold, "evaluated"
		case "gte":
			if active {
				return v >= threshold-c.Deadband, "evaluated"
			}
			return v >= threshold, "evaluated"
		case "lt":
			if active {
				return v < threshold+c.Deadband, "evaluated"
			}
			return v < threshold, "evaluated"
		case "lte":
			if active {
				return v <= threshold+c.Deadband, "evaluated"
			}
			return v <= threshold, "evaluated"
		}
	case "range":
		v, ok := anyFloat(value)
		lower, lok := anyFloat(c.Params["lower"])
		upper, uok := anyFloat(c.Params["upper"])
		if !ok || !lok || !uok {
			return false, "invalid_numeric_value"
		}
		if active {
			if c.Operator == "outside" {
				return v < lower+c.Deadband || v > upper-c.Deadband, "evaluated"
			}
			return v >= lower-c.Deadband && v <= upper+c.Deadband, "evaluated"
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
		if _, exists := context["previousValue"]; !exists {
			return false, "insufficient_previous_value"
		}
		switch c.Operator {
		case "changed":
			return fmt.Sprint(previous) != fmt.Sprint(value), "evaluated"
		case "rising":
			return fmt.Sprint(previous) == "false" && fmt.Sprint(value) == "true", "evaluated"
		case "falling":
			return fmt.Sprint(previous) == "true" && fmt.Sprint(value) == "false", "evaluated"
		case "from_to":
			return fmt.Sprint(previous) == fmt.Sprint(c.Params["from"]) && fmt.Sprint(value) == fmt.Sprint(c.Params["to"]), "evaluated"
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
		if active {
			return abs(v-baseline) > max(limit-c.Deadband, 0), "evaluated"
		}
		return abs(v-baseline) > limit, "evaluated"
	case "rate_of_change":
		previous, exists := context["previousValue"]
		if !exists {
			return false, "insufficient_previous_value"
		}
		current, currentOK := anyFloat(value)
		before, beforeOK := anyFloat(previous)
		limit, limitOK := anyFloat(c.Params["limit"])
		window, windowOK := anyFloat(c.Params["windowMs"])
		if !currentOK || !beforeOK || !limitOK || !windowOK || window <= 0 {
			return false, "invalid_numeric_value"
		}
		if sampleInterval, ok := anyFloat(context["sampleIntervalMs"]); ok && sampleInterval > 0 {
			window = sampleInterval
		}
		delta := current - before
		direction, _ := c.Params["direction"].(string)
		rate := abs(delta) / (window / 1000)
		if direction == "rise" {
			rate = delta / (window / 1000)
		}
		if direction == "fall" {
			rate = -delta / (window / 1000)
		}
		threshold := limit
		if active {
			threshold = max(limit-c.Deadband, 0)
		}
		return rate > threshold, "evaluated"
	}
	return false, "unsupported_trial"
}

func toAlarmGroup(record repository.AlarmGroupRecord) AlarmGroup {
	return AlarmGroup{ID: record.ID, ProjectID: record.ProjectID, Name: record.Name, ParentID: record.ParentID, Description: record.Description, SortOrder: record.SortOrder, FullPath: record.FullPath, HasChildren: record.HasChildren, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
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
	return AlarmNotificationChannel{ID: record.ID, ProjectID: record.ProjectID, Name: record.Name, ChannelType: record.ChannelType, Config: alarmCloneMap(record.Config), SecretStatus: alarmCloneMap(record.SecretStatus), IsEnabled: record.IsEnabled,
		LastTestStatus: record.LastTestStatus, LastTestedAt: record.LastTestedAt, LastTestDurationMS: record.LastTestDurationMS, LastTestMessage: record.LastTestMessage,
		CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}

func (s *AlarmSettingsService) readAccess(claims *auth.Claims, projectID string) error {
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
func (s *AlarmSettingsService) writeAccess(claims *auth.Claims, projectID string) error {
	if err := s.readAccess(claims, projectID); err != nil {
		return err
	}
	return validateUserID(claims.UserID)
}
func conditionAllowedForCategory(kind, category string) bool {
	if kind == "offline" || kind == "quality" || kind == "stale" {
		return true
	}
	switch category {
	case "number":
		return kind == "threshold" || kind == "range" || kind == "rate_of_change" || kind == "deviation"
	case "boolean":
		return kind == "state" || kind == "transition"
	case "text":
		return kind == "state" || kind == "transition" || kind == "text_match"
	default:
		return false
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
	if value == "object" || value == "array" || value == "json" || value == "bytes" || value == "byte[]" || value == "datetime" || value == "date" || value == "timestamp" {
		return "structured"
	}
	return "text"
}
func defaultConditionLabel(c AlarmCondition) string {
	labels := map[string]string{"threshold": "阈值报警", "range": "区间报警", "state": "状态报警", "transition": "状态变化报警", "text_match": "文本报警", "rate_of_change": "变化率报警", "deviation": "偏差报警", "offline": "数据点离线", "quality": "数据质量异常", "stale": "数据长时间未更新"}
	return labels[c.Kind]
}
func groupIsDescendant(groups []repository.AlarmGroupRecord, parentID, candidateID string) bool {
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
