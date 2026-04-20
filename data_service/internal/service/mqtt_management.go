package service

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

var datapointSegmentSanitizer = regexp.MustCompile(`[./\\\s]+|[^0-9A-Za-z_-]+`)

// MqttRuntimeStatus 表示兼容旧前端的 MQTT 运行状态。
type MqttRuntimeStatus struct {
	Connected  bool   `json:"connected"`
	Connecting bool   `json:"connecting"`
	Message    string `json:"message"`
}

// MqttConnectionDetail 表示 MQTT 连接详情响应。
type MqttConnectionDetail struct {
	ID                  string            `json:"id"`
	ProjectID           string            `json:"projectId"`
	Name                string            `json:"name"`
	Type                string            `json:"type"`
	Status              string            `json:"status"`
	IsEnabled           bool              `json:"isEnabled"`
	RetryCount          int               `json:"retryCount"`
	RetryInterval       int               `json:"retryInterval"`
	HealthCheckInterval int               `json:"healthCheckInterval"`
	BrokerURL           string            `json:"brokerUrl"`
	Protocol            string            `json:"protocol"`
	Port                int               `json:"port"`
	ClientID            *string           `json:"clientId"`
	Username            *string           `json:"username"`
	Password            *string           `json:"password,omitempty"`
	Keepalive           int               `json:"keepalive"`
	CleanSession        bool              `json:"cleanSession"`
	QOS                 int               `json:"qos"`
	ReconnectPeriod     int               `json:"reconnectPeriod"`
	ConnectTimeout      int               `json:"connectTimeout"`
	Will                map[string]any    `json:"will"`
	SSLConfig           map[string]any    `json:"sslConfig"`
	RuntimeStatus       MqttRuntimeStatus `json:"runtimeStatus"`
	CreatedAt           time.Time         `json:"createdAt"`
	UpdatedAt           time.Time         `json:"updatedAt"`
}

// MqttSubscription 表示 MQTT 订阅响应。
type MqttSubscription struct {
	ID               string    `json:"id"`
	ProjectID        string    `json:"projectId"`
	ConnectionID     string    `json:"connectionId"`
	Name             string    `json:"name"`
	Topic            string    `json:"topic"`
	QOS              int       `json:"qos"`
	Description      *string   `json:"description"`
	IsEnabled        bool      `json:"isEnabled"`
	MessageRetention int       `json:"messageRetention"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// MqttTagGroup 表示 MQTT 变量组响应。
type MqttTagGroup struct {
	ID             string    `json:"id"`
	ProjectID      string    `json:"projectId"`
	SubscriptionID string    `json:"subscriptionId"`
	Name           string    `json:"name"`
	Code           string    `json:"code"`
	Description    *string   `json:"description"`
	Color          *string   `json:"color"`
	Icon           *string   `json:"icon"`
	Order          int       `json:"order"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// MqttTag 表示 MQTT 变量响应。
type MqttTag struct {
	ID             string         `json:"id"`
	ProjectID      string         `json:"projectId"`
	SubscriptionID string         `json:"subscriptionId"`
	GroupID        *string        `json:"groupId"`
	Name           string         `json:"name"`
	Code           string         `json:"code"`
	Description    *string        `json:"description"`
	DataType       string         `json:"dataType"`
	ParseType      string         `json:"parseType"`
	ParseRule      string         `json:"parseRule"`
	DefaultValue   *string        `json:"defaultValue"`
	Unit           *string        `json:"unit"`
	Transform      *string        `json:"transform"`
	Validation     map[string]any `json:"validation"`
	IsEnabled      bool           `json:"isEnabled"`
	Order          int            `json:"order"`
	CurrentValue   any            `json:"currentValue"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

// ListMqttConnections 返回项目下 MQTT 连接列表。
func (s *MqttService) ListMqttConnections(ctx context.Context, projectID string, page, pageSize int) ([]MqttConnectionDetail, int, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, 0, err
	}

	records, total, err := s.repository.ListConnectionDetails(ctx, projectID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	result := make([]MqttConnectionDetail, 0, len(records))
	for _, record := range records {
		result = append(result, toMqttConnectionDetail(record))
	}
	return result, total, nil
}

// GetMqttConnection 返回单个 MQTT 连接。
func (s *MqttService) GetMqttConnection(ctx context.Context, projectID, connectionID string) (*MqttConnectionDetail, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}

	record, err := s.repository.GetConnectionDetail(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	result := toMqttConnectionDetail(*record)
	return &result, nil
}

// UpdateMqttConnection 更新 MQTT 连接配置。
func (s *MqttService) UpdateMqttConnection(ctx context.Context, projectID, connectionID, userID string, input CreateMqttConnectionInput) (*MqttConnectionDetail, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}

	current, err := s.repository.GetConnectionDetail(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}

	name, err := normalizeConnectionName(input.Name)
	if err != nil {
		return nil, err
	}
	status, err := normalizeMqttInitialStatus(input.Status)
	if err != nil {
		return nil, err
	}
	brokerURL, err := normalizeMqttBrokerURL(input.BrokerURL)
	if err != nil {
		return nil, err
	}
	protocol, err := normalizeMqttProtocol(input.Protocol)
	if err != nil {
		return nil, err
	}
	port, err := normalizeMqttPort(input.Port)
	if err != nil {
		return nil, err
	}
	keepalive, err := normalizeNonNegativeInt(input.Keepalive, current.Keepalive, "keepalive 不能小于 0")
	if err != nil {
		return nil, err
	}
	qos, err := normalizeMqttQOS(input.QOS)
	if err != nil {
		return nil, err
	}
	reconnectPeriod, err := normalizeNonNegativeInt(input.ReconnectPeriod, current.ReconnectPeriodMS, "reconnectPeriod 不能小于 0")
	if err != nil {
		return nil, err
	}
	connectTimeoutMS, err := normalizeNonNegativeInt(input.ConnectTimeoutMS, current.ConnectTimeoutMS, "connectTimeout 不能小于 0")
	if err != nil {
		return nil, err
	}

	cleanSession := current.CleanSession
	if input.CleanSession != nil {
		cleanSession = *input.CleanSession
	}

	record, err := s.repository.UpdateConnectionDetail(ctx, repository.UpdateMqttConnectionParams{
		ProjectID:             projectID,
		ConnectionID:          connectionID,
		UserID:                userID,
		Name:                  name,
		Status:                status,
		IsEnabled:             current.IsEnabled,
		RetryCount:            current.RetryCount,
		RetryIntervalMS:       current.RetryIntervalMS,
		HealthCheckIntervalMS: current.HealthCheckIntervalMS,
		BrokerURL:             brokerURL,
		Protocol:              protocol,
		Port:                  port,
		ClientID:              normalizeOptionalText(input.ClientID),
		Username:              normalizeOptionalText(input.Username),
		Password:              normalizeOptionalText(input.Password),
		Keepalive:             keepalive,
		CleanSession:          cleanSession,
		QOS:                   qos,
		ReconnectPeriodMS:     reconnectPeriod,
		ConnectTimeoutMS:      connectTimeoutMS,
		Will:                  cloneMap(input.Will),
		SSLConfig:             cloneMap(input.SSLConfig),
	})
	if err != nil {
		return nil, err
	}

	result := toMqttConnectionDetail(*record)
	return &result, nil
}

// DeleteMqttConnection 删除 MQTT 连接。
func (s *MqttService) DeleteMqttConnection(ctx context.Context, projectID, connectionID string) error {
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return err
	}
	return s.connections.Delete(ctx, projectID, connectionID)
}

// TestConnectionConfig 承接 MQTT 连接测试，当前以配置校验为主。
func (s *MqttService) TestConnectionConfig(_ context.Context, input CreateMqttConnectionInput) error {
	if _, err := normalizeConnectionName(input.Name); err != nil && strings.TrimSpace(input.Name) != "" {
		return err
	}
	if _, err := normalizeMqttBrokerURL(input.BrokerURL); err != nil {
		return err
	}
	if _, err := normalizeMqttProtocol(input.Protocol); err != nil {
		return err
	}
	if _, err := normalizeMqttPort(input.Port); err != nil {
		return err
	}
	if _, err := normalizeMqttQOS(input.QOS); err != nil {
		return err
	}
	return nil
}

// StopConnection 停止指定 MQTT 连接。
func (s *MqttService) StopConnection(ctx context.Context, projectID, connectionID string) (*MqttConnectionStatus, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}

	record, err := s.repository.StopConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	return &MqttConnectionStatus{Status: record.Status}, nil
}

// ListSubscriptions 查询项目下 MQTT 订阅。
func (s *MqttService) ListSubscriptions(ctx context.Context, projectID, connectionID string, page, pageSize int) ([]MqttSubscription, int, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, 0, err
	}
	if strings.TrimSpace(connectionID) != "" {
		if err := validateConnectionID(connectionID); err != nil {
			return nil, 0, err
		}
	}

	records, total, err := s.repository.ListSubscriptions(ctx, projectID, connectionID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	result := make([]MqttSubscription, 0, len(records))
	for _, record := range records {
		result = append(result, toMqttSubscription(record))
	}
	return result, total, nil
}

// GetSubscription 读取单个订阅。
func (s *MqttService) GetSubscription(ctx context.Context, projectID, subscriptionID string) (*MqttSubscription, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if _, err := uuid.Parse(strings.TrimSpace(subscriptionID)); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "subscriptionId 格式无效", err)
	}
	record, err := s.repository.GetSubscription(ctx, projectID, subscriptionID)
	if err != nil {
		return nil, err
	}
	result := toMqttSubscription(*record)
	return &result, nil
}

// CreateSubscription 创建订阅并同步数据点。
func (s *MqttService) CreateSubscription(ctx context.Context, projectID, userID string, input repository.CreateMqttSubscriptionParams) (*MqttSubscription, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(input.ConnectionID); err != nil {
		return nil, err
	}

	connection, err := s.connections.GetByProjectAndID(ctx, projectID, input.ConnectionID)
	if err != nil {
		return nil, err
	}
	if connection.Type != "mqtt" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接不是 MQTT 类型")
	}

	params := repository.CreateMqttSubscriptionParams{
		ProjectID:        projectID,
		ConnectionID:     input.ConnectionID,
		UserID:           userID,
		Name:             strings.TrimSpace(input.Name),
		Topic:            strings.TrimSpace(input.Topic),
		QOS:              input.QOS,
		Description:      trimOptionalString(input.Description),
		IsEnabled:        input.IsEnabled,
		MessageRetention: input.MessageRetention,
	}
	if params.Name == "" || params.Topic == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "订阅名称和主题不能为空")
	}
	if params.MessageRetention <= 0 {
		params.MessageRetention = 100
	}

	record, err := s.repository.CreateSubscription(ctx, params)
	if err != nil {
		return nil, err
	}
	if err := s.syncSubscriptionDatapoint(ctx, *record, userID); err != nil {
		return nil, err
	}

	result := toMqttSubscription(*record)
	return &result, nil
}

// UpdateSubscription 更新订阅并同步数据点。
func (s *MqttService) UpdateSubscription(ctx context.Context, projectID, subscriptionID, userID string, input repository.UpdateMqttSubscriptionParams) (*MqttSubscription, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	if _, err := uuid.Parse(strings.TrimSpace(subscriptionID)); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "subscriptionId 格式无效", err)
	}

	current, err := s.repository.GetSubscription(ctx, projectID, subscriptionID)
	if err != nil {
		return nil, err
	}

	params := repository.UpdateMqttSubscriptionParams{
		ProjectID:        projectID,
		SubscriptionID:   subscriptionID,
		UserID:           userID,
		Name:             fallbackTrimmed(input.Name, current.Name),
		Topic:            fallbackTrimmed(input.Topic, current.Topic),
		QOS:              input.QOS,
		Description:      trimOptionalString(input.Description),
		IsEnabled:        input.IsEnabled,
		MessageRetention: input.MessageRetention,
	}
	if params.QOS < 0 || params.QOS > 2 {
		params.QOS = current.QOS
	}
	if params.MessageRetention <= 0 {
		params.MessageRetention = current.MessageRetention
	}

	record, err := s.repository.UpdateSubscription(ctx, params)
	if err != nil {
		return nil, err
	}
	if err := s.syncSubscriptionDatapoint(ctx, *record, userID); err != nil {
		return nil, err
	}

	result := toMqttSubscription(*record)
	return &result, nil
}

// DeleteSubscription 删除订阅并标记其数据点失效。
func (s *MqttService) DeleteSubscription(ctx context.Context, projectID, subscriptionID, userID string) error {
	current, err := s.repository.GetSubscription(ctx, projectID, subscriptionID)
	if err != nil {
		return err
	}
	if err := s.repository.DeleteSubscription(ctx, projectID, subscriptionID); err != nil {
		return err
	}
	_, _ = s.datapoints.MarkInvalidBySource(ctx, projectID, "mqtt.subscription", subscriptionID, stringPtr(userID))
	_ = current
	return nil
}

// ToggleSubscription 切换订阅启用状态。
func (s *MqttService) ToggleSubscription(ctx context.Context, projectID, subscriptionID, userID string) (*MqttSubscription, error) {
	current, err := s.repository.GetSubscription(ctx, projectID, subscriptionID)
	if err != nil {
		return nil, err
	}
	record, err := s.repository.UpdateSubscriptionEnabled(ctx, projectID, subscriptionID, userID, !current.IsEnabled)
	if err != nil {
		return nil, err
	}
	result := toMqttSubscription(*record)
	return &result, nil
}

// ListTagGroups 查询变量组。
func (s *MqttService) ListTagGroups(ctx context.Context, projectID, subscriptionID string) ([]MqttTagGroup, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if _, err := uuid.Parse(strings.TrimSpace(subscriptionID)); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "subscriptionId 格式无效", err)
	}
	records, err := s.repository.ListTagGroups(ctx, projectID, subscriptionID)
	if err != nil {
		return nil, err
	}
	result := make([]MqttTagGroup, 0, len(records))
	for _, record := range records {
		result = append(result, toMqttTagGroup(record))
	}
	return result, nil
}

// GetTagGroup 读取单个变量组。
func (s *MqttService) GetTagGroup(ctx context.Context, projectID, groupID string) (*MqttTagGroup, error) {
	record, err := s.repository.GetTagGroup(ctx, projectID, groupID)
	if err != nil {
		return nil, err
	}
	result := toMqttTagGroup(*record)
	return &result, nil
}

// CreateTagGroup 创建变量组。
func (s *MqttService) CreateTagGroup(ctx context.Context, projectID, subscriptionID, userID string, input repository.CreateMqttTagGroupParams) (*MqttTagGroup, error) {
	params := repository.CreateMqttTagGroupParams{
		ProjectID:      projectID,
		SubscriptionID: subscriptionID,
		UserID:         userID,
		Name:           strings.TrimSpace(input.Name),
		Code:           strings.TrimSpace(input.Code),
		Description:    trimOptionalString(input.Description),
		Color:          trimOptionalString(input.Color),
		Icon:           trimOptionalString(input.Icon),
		Order:          input.Order,
	}
	if params.Name == "" || params.Code == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "变量组名称和标识不能为空")
	}
	record, err := s.repository.CreateTagGroup(ctx, params)
	if err != nil {
		return nil, err
	}
	result := toMqttTagGroup(*record)
	return &result, nil
}

// UpdateTagGroup 更新变量组。
func (s *MqttService) UpdateTagGroup(ctx context.Context, projectID, groupID, userID string, input repository.UpdateMqttTagGroupParams) (*MqttTagGroup, error) {
	current, err := s.repository.GetTagGroup(ctx, projectID, groupID)
	if err != nil {
		return nil, err
	}

	params := repository.UpdateMqttTagGroupParams{
		ProjectID:   projectID,
		GroupID:     groupID,
		UserID:      userID,
		Name:        fallbackTrimmed(input.Name, current.Name),
		Code:        fallbackTrimmed(input.Code, current.Code),
		Description: coalesceOptionalString(trimOptionalString(input.Description), current.Description),
		Color:       coalesceOptionalString(trimOptionalString(input.Color), current.Color),
		Icon:        coalesceOptionalString(trimOptionalString(input.Icon), current.Icon),
		Order:       input.Order,
	}
	if params.Order == 0 && current.Order != 0 {
		params.Order = current.Order
	}

	record, err := s.repository.UpdateTagGroup(ctx, params)
	if err != nil {
		return nil, err
	}
	result := toMqttTagGroup(*record)
	return &result, nil
}

// DeleteTagGroup 删除变量组并同步组内变量数据点路径。
func (s *MqttService) DeleteTagGroup(ctx context.Context, projectID, groupID string) error {
	group, err := s.repository.GetTagGroup(ctx, projectID, groupID)
	if err != nil {
		return err
	}

	tags, _, err := s.repository.ListTagsBySubscription(ctx, projectID, group.SubscriptionID, 1, 500, nil)
	if err != nil {
		return err
	}

	if err := s.repository.DeleteTagGroup(ctx, projectID, groupID); err != nil {
		return err
	}

	for _, tag := range tags {
		if tag.GroupID == nil || *tag.GroupID != groupID {
			continue
		}
		tag.GroupID = nil
		if err := s.syncTagDatapoint(ctx, tag, ""); err != nil {
			return err
		}
	}
	return nil
}

// UpdateTagGroupsOrder 批量更新变量组顺序。
func (s *MqttService) UpdateTagGroupsOrder(ctx context.Context, projectID, userID string, groups []MqttTagGroup) error {
	records := make([]repository.MqttTagGroupRecord, 0, len(groups))
	for _, group := range groups {
		records = append(records, repository.MqttTagGroupRecord{
			ID:    group.ID,
			Order: group.Order,
		})
	}
	return s.repository.UpdateTagGroupsOrder(ctx, projectID, records, userID)
}

// ListTagsBySubscription 查询订阅下变量。
func (s *MqttService) ListTagsBySubscription(ctx context.Context, projectID, subscriptionID string, page, pageSize int, isEnabled *bool) ([]MqttTag, int, error) {
	records, total, err := s.repository.ListTagsBySubscription(ctx, projectID, subscriptionID, page, pageSize, isEnabled)
	if err != nil {
		return nil, 0, err
	}
	result := make([]MqttTag, 0, len(records))
	for _, record := range records {
		result = append(result, toMqttTag(record))
	}
	return result, total, nil
}

// ListTagsByProject 查询项目下变量。
func (s *MqttService) ListTagsByProject(ctx context.Context, projectID string, page, pageSize int, subscriptionID string, isEnabled *bool) ([]MqttTag, int, error) {
	records, total, err := s.repository.ListTagsByProject(ctx, projectID, page, pageSize, subscriptionID, isEnabled)
	if err != nil {
		return nil, 0, err
	}
	result := make([]MqttTag, 0, len(records))
	for _, record := range records {
		result = append(result, toMqttTag(record))
	}
	return result, total, nil
}

// GetTag 读取单个变量。
func (s *MqttService) GetTag(ctx context.Context, projectID, tagID string) (*MqttTag, error) {
	record, err := s.repository.GetTag(ctx, projectID, tagID)
	if err != nil {
		return nil, err
	}
	result := toMqttTag(*record)
	return &result, nil
}

// CreateTag 创建变量并同步数据点。
func (s *MqttService) CreateTag(ctx context.Context, projectID, subscriptionID, userID string, input repository.CreateMqttTagParams) (*MqttTag, error) {
	params := repository.CreateMqttTagParams{
		ProjectID:      projectID,
		SubscriptionID: subscriptionID,
		UserID:         userID,
		GroupID:        input.GroupID,
		Name:           strings.TrimSpace(input.Name),
		Code:           strings.TrimSpace(input.Code),
		Description:    trimOptionalString(input.Description),
		DataType:       defaultString(strings.TrimSpace(input.DataType), "string"),
		ParseType:      defaultString(strings.TrimSpace(input.ParseType), "jsonpath"),
		ParseRule:      strings.TrimSpace(input.ParseRule),
		DefaultValue:   trimOptionalString(input.DefaultValue),
		Unit:           trimOptionalString(input.Unit),
		Transform:      trimOptionalString(input.Transform),
		Validation:     cloneMap(input.Validation),
		IsEnabled:      input.IsEnabled,
		Order:          input.Order,
	}
	if params.Name == "" || params.Code == "" || params.ParseRule == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "变量名称、标识和解析规则不能为空")
	}

	record, err := s.repository.CreateTag(ctx, params)
	if err != nil {
		return nil, err
	}
	if err := s.syncTagDatapoint(ctx, *record, userID); err != nil {
		return nil, err
	}
	result := toMqttTag(*record)
	return &result, nil
}

// CreateTagsBatch 批量创建变量。
func (s *MqttService) CreateTagsBatch(ctx context.Context, projectID, subscriptionID, userID string, inputs []repository.CreateMqttTagParams) ([]MqttTag, error) {
	result := make([]MqttTag, 0, len(inputs))
	for index, input := range inputs {
		if input.Order == 0 {
			input.Order = index
		}
		tag, err := s.CreateTag(ctx, projectID, subscriptionID, userID, input)
		if err != nil {
			return nil, err
		}
		result = append(result, *tag)
	}
	return result, nil
}

// UpdateTag 更新变量并同步数据点。
func (s *MqttService) UpdateTag(ctx context.Context, projectID, tagID, userID string, input repository.UpdateMqttTagParams) (*MqttTag, error) {
	current, err := s.repository.GetTag(ctx, projectID, tagID)
	if err != nil {
		return nil, err
	}

	params := repository.UpdateMqttTagParams{
		ProjectID:    projectID,
		TagID:        tagID,
		UserID:       userID,
		GroupID:      coalesceOptionalString(input.GroupID, current.GroupID),
		Name:         fallbackTrimmed(input.Name, current.Name),
		Code:         fallbackTrimmed(input.Code, current.Code),
		Description:  coalesceOptionalString(trimOptionalString(input.Description), current.Description),
		DataType:     defaultString(strings.TrimSpace(input.DataType), current.DataType),
		ParseType:    defaultString(strings.TrimSpace(input.ParseType), current.ParseType),
		ParseRule:    defaultString(strings.TrimSpace(input.ParseRule), current.ParseRule),
		DefaultValue: coalesceOptionalString(trimOptionalString(input.DefaultValue), current.DefaultValue),
		Unit:         coalesceOptionalString(trimOptionalString(input.Unit), current.Unit),
		Transform:    coalesceOptionalString(trimOptionalString(input.Transform), current.Transform),
		Validation:   cloneMap(input.Validation),
		IsEnabled:    input.IsEnabled,
		Order:        input.Order,
	}
	if len(params.Validation) == 0 {
		params.Validation = cloneMap(current.Validation)
	}
	if params.Order == 0 && current.Order != 0 {
		params.Order = current.Order
	}

	record, err := s.repository.UpdateTag(ctx, params)
	if err != nil {
		return nil, err
	}
	if err := s.syncTagDatapoint(ctx, *record, userID); err != nil {
		return nil, err
	}
	result := toMqttTag(*record)
	return &result, nil
}

// DeleteTag 删除变量并标记数据点失效。
func (s *MqttService) DeleteTag(ctx context.Context, projectID, tagID, userID string) error {
	if _, err := s.repository.GetTag(ctx, projectID, tagID); err != nil {
		return err
	}
	if err := s.repository.DeleteTag(ctx, projectID, tagID); err != nil {
		return err
	}
	_, _ = s.datapoints.MarkInvalidBySource(ctx, projectID, "mqtt.tag", tagID, stringPtr(userID))
	return nil
}

// ToggleTag 切换变量启用状态。
func (s *MqttService) ToggleTag(ctx context.Context, projectID, tagID, userID string) (*MqttTag, error) {
	current, err := s.repository.GetTag(ctx, projectID, tagID)
	if err != nil {
		return nil, err
	}
	record, err := s.repository.UpdateTagEnabled(ctx, projectID, tagID, userID, !current.IsEnabled)
	if err != nil {
		return nil, err
	}
	result := toMqttTag(*record)
	return &result, nil
}

// UpdateTagsOrder 批量更新变量顺序。
func (s *MqttService) UpdateTagsOrder(ctx context.Context, projectID string, tagIDs []string, userID string) error {
	return s.repository.UpdateTagsOrder(ctx, projectID, tagIDs, userID)
}

// GetTagValue 返回单个变量当前值，当前降级为 defaultValue。
func (s *MqttService) GetTagValue(ctx context.Context, projectID, tagID string) (map[string]any, error) {
	record, err := s.repository.GetTag(ctx, projectID, tagID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"tagId":       record.ID,
		"parsedValue": defaultValueOrNil(record.DefaultValue),
		"quality":     "unknown",
		"timestamp":   record.UpdatedAt,
	}, nil
}

// GetTagValues 返回多个变量当前值。
func (s *MqttService) GetTagValues(ctx context.Context, projectID string, tagIDs []string) ([]map[string]any, error) {
	result := make([]map[string]any, 0, len(tagIDs))
	for _, tagID := range uniqueStrings(tagIDs) {
		value, err := s.GetTagValue(ctx, projectID, tagID)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (s *MqttService) syncSubscriptionDatapoint(ctx context.Context, subscription repository.MqttSubscriptionRecord, userID string) error {
	connection, err := s.connections.GetByProjectAndID(ctx, subscription.ProjectID, subscription.ConnectionID)
	if err != nil {
		return err
	}
	basePath := "mqtt." + normalizeDatapointSegment(connection.Name) + "." + normalizeDatapointSegment(subscription.Name)
	path := s.allocateDataPointPath(ctx, subscription.ProjectID, basePath, subscription.ID, "mqtt.subscription")

	return s.upsertMqttDataPoint(ctx, repository.CreateDataPointParams{
		ProjectID:    subscription.ProjectID,
		UserID:       stringPtr(userID),
		Path:         path,
		Name:         subscription.Name,
		SourceType:   "mqtt.subscription",
		SourceID:     &subscription.ID,
		SourceConfig: map[string]any{"mode": "subscription"},
		DataType:     "object",
		Tags:         []any{},
		RefreshMode:  "subscription",
		Status:       "active",
	})
}

func (s *MqttService) syncTagDatapoint(ctx context.Context, tag repository.MqttTagRecord, userID string) error {
	subscription, err := s.repository.GetSubscription(ctx, tag.ProjectID, tag.SubscriptionID)
	if err != nil {
		return err
	}
	connection, err := s.connections.GetByProjectAndID(ctx, tag.ProjectID, subscription.ConnectionID)
	if err != nil {
		return err
	}

	groupSegment := "默认分组"
	if tag.GroupID != nil && strings.TrimSpace(*tag.GroupID) != "" {
		group, err := s.repository.GetTagGroup(ctx, tag.ProjectID, *tag.GroupID)
		if err == nil {
			groupSegment = group.Name
		}
	}

	basePath := "mqtt." + normalizeDatapointSegment(connection.Name) + "." + normalizeDatapointSegment(groupSegment) + "." + normalizeDatapointSegment(tag.Name)
	path := s.allocateDataPointPath(ctx, tag.ProjectID, basePath, tag.ID, "mqtt.tag")

	return s.upsertMqttDataPoint(ctx, repository.CreateDataPointParams{
		ProjectID:    tag.ProjectID,
		UserID:       stringPtr(userID),
		Path:         path,
		Name:         tag.Name,
		SourceType:   "mqtt.tag",
		SourceID:     &tag.ID,
		SourceConfig: map[string]any{},
		DataType:     tag.DataType,
		Unit:         cloneOptionalString(tag.Unit),
		DefaultValue: cloneOptionalString(tag.DefaultValue),
		Tags:         []any{},
		RefreshMode:  "subscription",
		Status:       "active",
	})
}

func (s *MqttService) upsertMqttDataPoint(ctx context.Context, input repository.CreateDataPointParams) error {
	existing, err := s.datapoints.GetByProjectAndSource(ctx, input.ProjectID, input.SourceType, *input.SourceID)
	if err == nil && existing != nil {
		_, updateErr := s.datapoints.Update(ctx, repository.UpdateDataPointParams{
			ID:                existing.ID,
			ProjectID:         existing.ProjectID,
			UserID:            valueOrDefault(input.UserID, existing.UpdatedBy),
			Name:              input.Name,
			Description:       cloneOptionalString(input.Description),
			SourceType:        input.SourceType,
			SourceID:          cloneOptionalString(input.SourceID),
			SourceConfig:      cloneMap(input.SourceConfig),
			DataType:          input.DataType,
			Unit:              cloneOptionalString(input.Unit),
			PrecisionNum:      cloneOptionalInt(input.PrecisionNum),
			DefaultValue:      cloneOptionalString(input.DefaultValue),
			MinValue:          cloneOptionalFloat64(input.MinValue),
			MaxValue:          cloneOptionalFloat64(input.MaxValue),
			AlarmLow:          cloneOptionalFloat64(input.AlarmLow),
			AlarmHigh:         cloneOptionalFloat64(input.AlarmHigh),
			Tags:              cloneJSONArray(input.Tags),
			RefreshMode:       input.RefreshMode,
			RefreshIntervalMS: cloneOptionalInt(input.RefreshIntervalMS),
			Status:            input.Status,
		})
		return updateErr
	}

	_, createErr := s.datapoints.Create(ctx, input)
	return createErr
}

func (s *MqttService) allocateDataPointPath(ctx context.Context, projectID, basePath, sourceID, sourceType string) string {
	basePath = strings.TrimSpace(basePath)
	if basePath == "" {
		basePath = "mqtt.unnamed"
	}
	candidate := basePath
	for index := 2; index < 100; index++ {
		record, err := s.datapoints.GetByProjectAndPath(ctx, projectID, candidate)
		if err != nil || record == nil {
			return candidate
		}
		if record.SourceType == sourceType && record.SourceID != nil && *record.SourceID == sourceID {
			return candidate
		}
		candidate = fmt.Sprintf("%s_%d", basePath, index)
	}
	return basePath + "_" + sourceID[:8]
}

func normalizeDatapointSegment(segment string) string {
	segment = strings.TrimSpace(segment)
	if segment == "" {
		return "unnamed"
	}
	normalized := datapointSegmentSanitizer.ReplaceAllString(segment, "_")
	normalized = strings.Trim(normalized, "_")
	if normalized == "" {
		return "unnamed"
	}
	return normalized
}

func toMqttConnectionDetail(record repository.MqttConnectionDetailRecord) MqttConnectionDetail {
	return MqttConnectionDetail{
		ID:                  record.ID,
		ProjectID:           record.ProjectID,
		Name:                record.Name,
		Type:                record.Type,
		Status:              record.Status,
		IsEnabled:           record.IsEnabled,
		RetryCount:          record.RetryCount,
		RetryInterval:       record.RetryIntervalMS,
		HealthCheckInterval: record.HealthCheckIntervalMS,
		BrokerURL:           record.BrokerURL,
		Protocol:            record.Protocol,
		Port:                record.Port,
		ClientID:            cloneOptionalString(record.ClientID),
		Username:            cloneOptionalString(record.Username),
		Keepalive:           record.Keepalive,
		CleanSession:        record.CleanSession,
		QOS:                 record.QOS,
		ReconnectPeriod:     record.ReconnectPeriodMS,
		ConnectTimeout:      record.ConnectTimeoutMS,
		Will:                cloneMap(record.Will),
		SSLConfig:           cloneMap(record.SSLConfig),
		RuntimeStatus: MqttRuntimeStatus{
			Connected:  record.Status == "connected",
			Connecting: false,
			Message:    record.Status,
		},
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}
}

func toMqttSubscription(record repository.MqttSubscriptionRecord) MqttSubscription {
	return MqttSubscription{
		ID:               record.ID,
		ProjectID:        record.ProjectID,
		ConnectionID:     record.ConnectionID,
		Name:             record.Name,
		Topic:            record.Topic,
		QOS:              record.QOS,
		Description:      cloneOptionalString(record.Description),
		IsEnabled:        record.IsEnabled,
		MessageRetention: record.MessageRetention,
		CreatedAt:        record.CreatedAt,
		UpdatedAt:        record.UpdatedAt,
	}
}

func toMqttTagGroup(record repository.MqttTagGroupRecord) MqttTagGroup {
	return MqttTagGroup{
		ID:             record.ID,
		ProjectID:      record.ProjectID,
		SubscriptionID: record.SubscriptionID,
		Name:           record.Name,
		Code:           record.Code,
		Description:    cloneOptionalString(record.Description),
		Color:          cloneOptionalString(record.Color),
		Icon:           cloneOptionalString(record.Icon),
		Order:          record.Order,
		CreatedAt:      record.CreatedAt,
		UpdatedAt:      record.UpdatedAt,
	}
}

func toMqttTag(record repository.MqttTagRecord) MqttTag {
	return MqttTag{
		ID:             record.ID,
		ProjectID:      record.ProjectID,
		SubscriptionID: record.SubscriptionID,
		GroupID:        cloneOptionalString(record.GroupID),
		Name:           record.Name,
		Code:           record.Code,
		Description:    cloneOptionalString(record.Description),
		DataType:       record.DataType,
		ParseType:      record.ParseType,
		ParseRule:      record.ParseRule,
		DefaultValue:   cloneOptionalString(record.DefaultValue),
		Unit:           cloneOptionalString(record.Unit),
		Transform:      cloneOptionalString(record.Transform),
		Validation:     cloneMap(record.Validation),
		IsEnabled:      record.IsEnabled,
		Order:          record.Order,
		CurrentValue:   defaultValueOrNil(record.DefaultValue),
		CreatedAt:      record.CreatedAt,
		UpdatedAt:      record.UpdatedAt,
	}
}

func trimOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func fallbackTrimmed(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func stringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	next := strings.TrimSpace(value)
	return &next
}

func coalesceOptionalString(value, fallback *string) *string {
	if value != nil {
		return value
	}
	return cloneOptionalString(fallback)
}

func valueOrDefault(primary *string, fallback *string) string {
	if primary != nil && strings.TrimSpace(*primary) != "" {
		return strings.TrimSpace(*primary)
	}
	if fallback != nil && strings.TrimSpace(*fallback) != "" {
		return strings.TrimSpace(*fallback)
	}
	return ""
}
