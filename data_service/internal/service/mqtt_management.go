package service

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

var datapointSegmentSanitizer = regexp.MustCompile(`[./\\\s]+|[^0-9A-Za-z_-]+`)

const (
	mqttSubscriptionUsageRawDatapoint   = "raw_datapoint"
	mqttSubscriptionUsageSingleVariable = "single_variable"
	mqttSubscriptionUsageBatchVariable  = "batch_variable"
	defaultMqttMessageRetention         = 5000
)

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
	ID               string         `json:"id"`
	ProjectID        string         `json:"projectId"`
	ConnectionID     string         `json:"connectionId"`
	GroupID          *string        `json:"groupId,omitempty"`
	Name             string         `json:"name"`
	Topic            string         `json:"topic"`
	QOS              int            `json:"qos"`
	UsageMode        string         `json:"usageMode"`
	Description      *string        `json:"description"`
	MessageRetention int            `json:"messageRetention"`
	DefaultBatchRule map[string]any `json:"defaultBatchParseRule"`
	Order            int            `json:"order"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

// MqttSubscriptionGroup 表示 MQTT 订阅树分组响应。
type MqttSubscriptionGroup struct {
	ID           string                   `json:"id"`
	ProjectID    string                   `json:"projectId"`
	ConnectionID string                   `json:"connectionId"`
	Name         string                   `json:"name"`
	ParentID     *string                  `json:"parentId,omitempty"`
	Children     []*MqttSubscriptionGroup `json:"children,omitempty"`
	CreatedAt    time.Time                `json:"createdAt"`
	UpdatedAt    time.Time                `json:"updatedAt"`
}

// MqttTag 表示 MQTT 变量响应。
type MqttTag struct {
	ID             string         `json:"id"`
	ProjectID      string         `json:"projectId"`
	SubscriptionID string         `json:"subscriptionId"`
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
	Order          int            `json:"order"`
	CurrentValue   any            `json:"currentValue"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

// CreateMqttSubscriptionGroupInput 描述创建订阅分组的输入。
type CreateMqttSubscriptionGroupInput struct {
	Name     string
	ParentID *string
}

// UpdateMqttSubscriptionGroupInput 描述更新订阅分组的输入。
type UpdateMqttSubscriptionGroupInput struct {
	Name        *string
	ParentID    *string
	HasParentID bool
}

// CreateMqttSubscriptionInput 描述创建 MQTT 订阅的输入。
type CreateMqttSubscriptionInput struct {
	ConnectionID     string
	GroupID          *string
	Name             string
	Topic            string
	QOS              int
	UsageMode        string
	Description      *string
	MessageRetention int
	Order            int
}

// UpdateMqttSubscriptionInput 描述更新 MQTT 订阅的输入。
type UpdateMqttSubscriptionInput struct {
	GroupID          *string
	HasGroupID       bool
	Name             string
	Topic            string
	QOS              int
	UsageMode        string
	Description      *string
	MessageRetention int
	Order            int
}

// UpdateMqttSubscriptionDefaultBatchRuleInput 描述保存订阅默认批量解析规则的输入。
type UpdateMqttSubscriptionDefaultBatchRuleInput struct {
	Rule map[string]any
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

// GetConnectionSummary 读取 MQTT 连接主表摘要。
func (s *MqttService) GetConnectionSummary(ctx context.Context, projectID, connectionID string) (*repository.MqttConnectionSummaryRecord, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}
	return s.repository.GetConnectionSummary(ctx, projectID, connectionID)
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

// TestConnectionConfig 使用临时 MQTT 客户端验证 Broker 连接。
// 输入来自前端未保存表单，不落库；输出仅表示本次短连接是否成功。
func (s *MqttService) TestConnectionConfig(ctx context.Context, input CreateMqttConnectionInput) error {
	if _, err := normalizeConnectionName(input.Name); err != nil && strings.TrimSpace(input.Name) != "" {
		return err
	}
	brokerURL, err := normalizeMqttBrokerURL(input.BrokerURL)
	if err != nil {
		return err
	}
	protocol, err := normalizeMqttProtocol(input.Protocol)
	if err != nil {
		return err
	}
	port, err := normalizeMqttPort(input.Port)
	if err != nil {
		return err
	}
	if _, err := normalizeMqttQOS(input.QOS); err != nil {
		return err
	}
	keepalive, err := normalizeNonNegativeInt(input.Keepalive, 60, "keepalive 不能小于 0")
	if err != nil {
		return err
	}
	connectTimeoutMS, err := normalizeNonNegativeInt(input.ConnectTimeoutMS, 5000, "connectTimeout 不能小于 0")
	if err != nil {
		return err
	}
	if connectTimeoutMS <= 0 {
		connectTimeoutMS = 5000
	}

	options := mqtt.NewClientOptions()
	options.AddBroker(buildMqttTestBrokerAddress(protocol, brokerURL, port))
	options.SetClientID(buildMqttTestClientID(input.ClientID))
	options.SetCleanSession(input.CleanSession == nil || *input.CleanSession)
	options.SetConnectTimeout(time.Duration(connectTimeoutMS) * time.Millisecond)
	if keepalive > 0 {
		options.SetKeepAlive(time.Duration(keepalive) * time.Second)
	}
	if username := normalizeOptionalText(input.Username); username != nil {
		options.SetUsername(*username)
	}
	if password := normalizeOptionalText(input.Password); password != nil {
		options.SetPassword(*password)
	}

	client := mqtt.NewClient(options)
	token := client.Connect()
	if !waitMqttToken(ctx, token, time.Duration(connectTimeoutMS)*time.Millisecond) {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "MQTT 连接超时")
	}
	if err := token.Error(); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "MQTT 连接失败", err)
	}
	client.Disconnect(250)
	return nil
}

func buildMqttTestBrokerAddress(protocol, brokerURL string, port int) string {
	broker := strings.TrimSpace(brokerURL)
	if strings.Contains(broker, "://") {
		return broker
	}
	scheme := strings.TrimSpace(strings.ToLower(protocol))
	if scheme == "" {
		scheme = "mqtt"
	}
	if scheme == "mqtt" {
		scheme = "tcp"
	} else if scheme == "mqtts" {
		scheme = "ssl"
	}
	if port > 0 {
		return fmt.Sprintf("%s://%s:%s", scheme, broker, strconv.Itoa(port))
	}
	return fmt.Sprintf("%s://%s", scheme, broker)
}

func buildMqttTestClientID(configured *string) string {
	if configured != nil && strings.TrimSpace(*configured) != "" {
		return strings.TrimSpace(*configured)
	}
	return fmt.Sprintf("indu-forge-test-%d", time.Now().UnixNano())
}

func waitMqttToken(ctx context.Context, token mqtt.Token, timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		token.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return false
	case <-time.After(timeout):
		return false
	case <-done:
		return true
	}
}

// StopConnection 停止指定 MQTT 连接。
func (s *MqttService) StopConnection(ctx context.Context, projectID, connectionID string) (*MqttConnectionStatus, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}

	if s.runtime != nil {
		s.runtime.Disconnect(projectID, connectionID)
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
func (s *MqttService) CreateSubscription(ctx context.Context, projectID, userID string, input CreateMqttSubscriptionInput) (*MqttSubscription, error) {
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
	if !isMqttManagedMessageConnectionType(connection.Type) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接不是 MQTT 消息工作台类型")
	}

	params := repository.CreateMqttSubscriptionParams{
		ProjectID:        projectID,
		ConnectionID:     input.ConnectionID,
		GroupID:          normalizeOptionalMqttID(input.GroupID),
		UserID:           userID,
		Name:             strings.TrimSpace(input.Name),
		Topic:            strings.TrimSpace(input.Topic),
		QOS:              input.QOS,
		UsageMode:        normalizeMqttSubscriptionUsageMode(input.UsageMode),
		Description:      trimOptionalString(input.Description),
		MessageRetention: input.MessageRetention,
		Order:            input.Order,
	}
	if params.Name == "" || params.Topic == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "订阅名称和主题不能为空")
	}
	if params.MessageRetention <= 0 {
		params.MessageRetention = defaultMqttMessageRetention
	}
	if err := s.ensureSubscriptionGroupInConnection(ctx, projectID, input.ConnectionID, params.GroupID); err != nil {
		return nil, err
	}

	record, err := s.repository.CreateSubscription(ctx, params)
	if err != nil {
		return nil, err
	}
	if isMqttRawDatapointSubscription(*record) {
		if err := s.syncSubscriptionDatapoint(ctx, *record, userID); err != nil {
			return nil, err
		}
	}

	result := toMqttSubscription(*record)
	return &result, nil
}

func normalizeMqttSubscriptionUsageMode(value string) string {
	switch strings.TrimSpace(value) {
	case mqttSubscriptionUsageRawDatapoint:
		return mqttSubscriptionUsageRawDatapoint
	case mqttSubscriptionUsageBatchVariable:
		return mqttSubscriptionUsageBatchVariable
	default:
		return mqttSubscriptionUsageSingleVariable
	}
}

func isMqttRawDatapointSubscription(record repository.MqttSubscriptionRecord) bool {
	return normalizeMqttSubscriptionUsageMode(record.UsageMode) == mqttSubscriptionUsageRawDatapoint
}

func normalizeMqttBatchParseRule(rule map[string]any) map[string]any {
	normalized := map[string]any{
		"arrayPath":   normalizeRulePath(rule["arrayPath"], "$"),
		"namePath":    normalizeRulePath(rule["namePath"], "N"),
		"valuePath":   normalizeRulePath(rule["valuePath"], "V"),
		"qualityPath": normalizeRulePath(rule["qualityPath"], "Q"),
		"timePath":    normalizeRulePath(rule["timePath"], "T"),
	}
	if matchName := strings.TrimSpace(fmt.Sprint(rule["matchName"])); matchName != "" && matchName != "<nil>" {
		normalized["matchName"] = matchName
	}
	return normalized
}

func normalizeRulePath(value any, fallback string) string {
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "" || text == "<nil>" {
		return fallback
	}
	return text
}

func (s *MqttService) syncRawSubscriptionDatapointIfNeeded(ctx context.Context, record repository.MqttSubscriptionRecord, userID string) error {
	if !isMqttRawDatapointSubscription(record) {
		return nil
	}
	if err := s.syncSubscriptionDatapoint(ctx, record, userID); err != nil {
		return err
	}
	return nil
}

// UpdateSubscription 更新订阅并同步数据点。
func (s *MqttService) UpdateSubscription(ctx context.Context, projectID, subscriptionID, userID string, input UpdateMqttSubscriptionInput) (*MqttSubscription, error) {
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
		ProjectID:      projectID,
		SubscriptionID: subscriptionID,
		GroupID:        normalizeOptionalMqttID(input.GroupID),
		HasGroupID:     input.HasGroupID,
		UserID:         userID,
		Name:           fallbackTrimmed(input.Name, current.Name),
		Topic:          fallbackTrimmed(input.Topic, current.Topic),
		QOS:            input.QOS,
		// 订阅使用方式决定数据点生成和默认打开行为，创建后不允许通过编辑订阅修改。
		UsageMode:        current.UsageMode,
		Description:      trimOptionalString(input.Description),
		MessageRetention: input.MessageRetention,
		Order:            input.Order,
	}
	if params.QOS < 0 || params.QOS > 2 {
		params.QOS = current.QOS
	}
	if params.MessageRetention <= 0 {
		params.MessageRetention = current.MessageRetention
	}
	if params.Order == 0 && current.Order != 0 {
		params.Order = current.Order
	}
	if !input.HasGroupID {
		params.GroupID = cloneOptionalString(current.GroupID)
		params.HasGroupID = true
	}
	if err := s.ensureSubscriptionGroupInConnection(ctx, projectID, current.ConnectionID, params.GroupID); err != nil {
		return nil, err
	}

	record, err := s.repository.UpdateSubscription(ctx, params)
	if err != nil {
		return nil, err
	}
	if err := s.syncRawSubscriptionDatapointIfNeeded(ctx, *record, userID); err != nil {
		return nil, err
	}

	result := toMqttSubscription(*record)
	return &result, nil
}

// UpdateSubscriptionDefaultBatchRule 保存订阅默认批量解析规则。
func (s *MqttService) UpdateSubscriptionDefaultBatchRule(ctx context.Context, projectID, subscriptionID, userID string, input UpdateMqttSubscriptionDefaultBatchRuleInput) (*MqttSubscription, error) {
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
	if normalizeMqttSubscriptionUsageMode(current.UsageMode) != mqttSubscriptionUsageBatchVariable {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "只有批量变量订阅支持默认批量解析规则")
	}

	rule := normalizeMqttBatchParseRule(input.Rule)
	record, err := s.repository.UpdateSubscriptionDefaultBatchRule(ctx, repository.UpdateMqttSubscriptionDefaultBatchRuleParams{
		ProjectID:      projectID,
		SubscriptionID: subscriptionID,
		UserID:         userID,
		Rule:           rule,
	})
	if err != nil {
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

// ListSubscriptionGroups 查询连接下的订阅分组树。
func (s *MqttService) ListSubscriptionGroups(ctx context.Context, projectID, connectionID string) ([]*MqttSubscriptionGroup, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}
	if _, err := s.repository.GetConnectionSummary(ctx, projectID, connectionID); err != nil {
		return nil, err
	}
	records, err := s.repository.ListSubscriptionGroups(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	return toMqttSubscriptionGroupTree(records), nil
}

// CreateSubscriptionGroup 创建订阅分组。
func (s *MqttService) CreateSubscriptionGroup(ctx context.Context, projectID, connectionID, userID string, input CreateMqttSubscriptionGroupInput) (*MqttSubscriptionGroup, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	name, err := normalizeMqttSubscriptionGroupName(input.Name)
	if err != nil {
		return nil, err
	}
	parentID := normalizeOptionalMqttID(input.ParentID)
	if err := s.ensureSubscriptionGroupInConnection(ctx, projectID, connectionID, parentID); err != nil {
		return nil, err
	}
	record, err := s.repository.CreateSubscriptionGroup(ctx, repository.CreateMqttSubscriptionGroupParams{
		ProjectID:    projectID,
		ConnectionID: connectionID,
		UserID:       userID,
		Name:         name,
		ParentID:     parentID,
	})
	if err != nil {
		return nil, err
	}
	group := toMqttSubscriptionGroup(*record)
	return &group, nil
}

// UpdateSubscriptionGroup 更新订阅分组名称或上级分组。
func (s *MqttService) UpdateSubscriptionGroup(ctx context.Context, projectID, groupID, userID string, input UpdateMqttSubscriptionGroupInput) (*MqttSubscriptionGroup, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	if _, err := uuid.Parse(strings.TrimSpace(groupID)); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "groupId 格式无效", err)
	}
	currentRecord, err := s.repository.GetSubscriptionGroup(ctx, projectID, groupID)
	if err != nil {
		return nil, err
	}
	records, err := s.repository.ListSubscriptionGroups(ctx, projectID, currentRecord.ConnectionID)
	if err != nil {
		return nil, err
	}
	current, ok := findMqttSubscriptionGroupRecord(records, groupID)
	if !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "MQTT 订阅分组不存在")
	}
	if input.Name == nil && !input.HasParentID {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "至少需要提供一个待更新字段")
	}
	name := current.Name
	if input.Name != nil {
		name, err = normalizeMqttSubscriptionGroupName(*input.Name)
		if err != nil {
			return nil, err
		}
	}
	parentID := cloneOptionalString(current.ParentID)
	if input.HasParentID {
		parentID = normalizeOptionalMqttID(input.ParentID)
		if parentID != nil {
			if *parentID == groupID {
				return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "上级分组不能指向自身")
			}
			parent, ok := findMqttSubscriptionGroupRecord(records, *parentID)
			if !ok || parent.ConnectionID != current.ConnectionID {
				return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "上级订阅分组不存在")
			}
			if isDescendantMqttSubscriptionGroup(records, groupID, *parentID) {
				return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "上级分组不能指向下级分组")
			}
		}
	}
	record, err := s.repository.UpdateSubscriptionGroup(ctx, repository.UpdateMqttSubscriptionGroupParams{
		ProjectID: projectID,
		GroupID:   groupID,
		UserID:    userID,
		Name:      name,
		ParentID:  parentID,
	})
	if err != nil {
		return nil, err
	}
	group := toMqttSubscriptionGroup(*record)
	return &group, nil
}

// DeleteSubscriptionGroup 删除分组，订阅会自动回到根节点。
func (s *MqttService) DeleteSubscriptionGroup(ctx context.Context, projectID, groupID string) error {
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	if _, err := uuid.Parse(strings.TrimSpace(groupID)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "groupId 格式无效", err)
	}
	return s.repository.DeleteSubscriptionGroup(ctx, projectID, groupID)
}

// ListTagsBySubscription 查询订阅下变量。
func (s *MqttService) ListTagsBySubscription(ctx context.Context, projectID, subscriptionID string, search string, page, pageSize int, sortBy, sortOrder string) ([]MqttTag, int, error) {
	records, total, err := s.repository.ListTagsBySubscription(ctx, projectID, subscriptionID, search, page, pageSize, sortBy, sortOrder)
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
func (s *MqttService) ListTagsByProject(ctx context.Context, projectID string, page, pageSize int, subscriptionID string) ([]MqttTag, int, error) {
	records, total, err := s.repository.ListTagsByProject(ctx, projectID, page, pageSize, subscriptionID)
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
	params := normalizeMqttTagCreateParams(projectID, subscriptionID, userID, input)
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

func normalizeMqttTagCreateParams(projectID, subscriptionID, userID string, input repository.CreateMqttTagParams) repository.CreateMqttTagParams {
	return repository.CreateMqttTagParams{
		ProjectID:      projectID,
		SubscriptionID: subscriptionID,
		UserID:         userID,
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
		Order:          input.Order,
	}
}

// CreateTagsBatch 批量创建变量。
func (s *MqttService) CreateTagsBatch(ctx context.Context, projectID, subscriptionID, userID string, inputs []repository.CreateMqttTagParams) ([]MqttTag, error) {
	subscription, err := s.repository.GetSubscription(ctx, projectID, subscriptionID)
	if err != nil {
		return nil, err
	}
	connection, err := s.connections.GetByProjectAndID(ctx, projectID, subscription.ConnectionID)
	if err != nil {
		return nil, err
	}
	datapointPathPrefix := "mqtt." + normalizeDatapointSegment(connection.Name) + "." + mqttSubscriptionPathSegment(*subscription) + "."

	batchParams := make([]repository.BatchMqttTagDataPointParams, 0, len(inputs))
	for index, input := range inputs {
		if input.Order == 0 {
			input.Order = index
		}
		params := normalizeMqttTagCreateParams(projectID, subscriptionID, userID, input)
		if params.Name == "" || params.Code == "" || params.ParseRule == "" {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "变量名称、标识和解析规则不能为空")
		}
		// 批量创建使用事务内集合 SQL；路径不再逐条追加后缀，冲突由数据库唯一约束统一拦截。
		batchParams = append(batchParams, repository.BatchMqttTagDataPointParams{
			Tag:        params,
			DataPath:   datapointPathPrefix + normalizeDatapointSegment(params.Name),
			DataName:   params.Name,
			SourceType: "mqtt.tag",
			SourceConfig: map[string]any{
				"connectionId":   subscription.ConnectionID,
				"subscriptionId": subscriptionID,
			},
			RefreshMode: "subscription",
			Status:      "active",
		})
	}

	records, err := s.repository.CreateTagsBatchWithDataPoints(ctx, batchParams)
	if err != nil {
		return nil, err
	}
	result := make([]MqttTag, 0, len(records))
	for _, record := range records {
		result = append(result, toMqttTag(record))
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

// DeleteTagsBatch 批量删除变量，并沿用单变量删除的数据点失效处理。
func (s *MqttService) DeleteTagsBatch(ctx context.Context, projectID string, tagIDs []string, userID string) (int, error) {
	uniqueTagIDs := uniqueStrings(tagIDs)
	if len(uniqueTagIDs) == 0 {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "tagIds 不能为空")
	}

	return s.repository.DeleteTagsBatchWithDataPoints(ctx, projectID, uniqueTagIDs, userID)
}

// DeleteTagsBySubscriptionFilter 删除当前订阅、当前搜索条件命中的全部变量。
func (s *MqttService) DeleteTagsBySubscriptionFilter(ctx context.Context, projectID, subscriptionID, search, userID string) (int, error) {
	if err := validateProjectID(projectID); err != nil {
		return 0, err
	}
	if _, err := uuid.Parse(strings.TrimSpace(subscriptionID)); err != nil {
		return 0, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "subscriptionId 格式无效", err)
	}
	if err := validateUserID(userID); err != nil {
		return 0, err
	}
	if _, err := s.repository.GetSubscription(ctx, projectID, subscriptionID); err != nil {
		return 0, err
	}

	tagIDs, err := s.repository.ListTagIDsBySubscriptionFilter(ctx, projectID, subscriptionID, search)
	if err != nil {
		return 0, err
	}
	if len(tagIDs) == 0 {
		return 0, nil
	}
	return s.repository.DeleteTagsBatchWithDataPoints(ctx, projectID, tagIDs, userID)
}

// UpdateTagsOrder 批量更新变量顺序。
func (s *MqttService) UpdateTagsOrder(ctx context.Context, projectID string, tagIDs []string, userID string) error {
	return s.repository.UpdateTagsOrder(ctx, projectID, tagIDs, userID)
}

// GetTagValue 返回单个变量最近值。
// 优先从订阅最近消息解析变量值；没有消息时使用 defaultValue 降级，保证前端下次打开仍有可解释的快照。
func (s *MqttService) GetTagValue(ctx context.Context, projectID, tagID string) (map[string]any, error) {
	record, err := s.repository.GetTag(ctx, projectID, tagID)
	if err != nil {
		return nil, err
	}
	snapshot, err := s.resolveLatestTagValue(ctx, projectID, *record)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"tagId":          snapshot.TagID,
		"subscriptionId": snapshot.SubscriptionID,
		"topic":          snapshot.Topic,
		"payload":        snapshot.Payload,
		"value":          snapshot.Value,
		"parsedValue":    snapshot.ParsedValue,
		"quality":        snapshot.Quality,
		"qualityCode":    snapshot.QualityCode,
		"timestamp":      snapshot.Timestamp,
		"receivedAt":     snapshot.ReceivedAt,
		"error":          snapshot.Error,
	}, nil
}

// GetTagValues 返回多个变量当前值。
func (s *MqttService) GetTagValues(ctx context.Context, projectID string, tagIDs []string) ([]map[string]any, error) {
	return s.GetTagValuesWithOptions(ctx, projectID, tagIDs, false)
}

// GetTagValuesWithOptions 返回多个变量当前值；compact 模式只返回列表/监控所需字段，避免批量场景传输原始 payload。
func (s *MqttService) GetTagValuesWithOptions(ctx context.Context, projectID string, tagIDs []string, compact bool) ([]map[string]any, error) {
	result := make([]map[string]any, 0, len(tagIDs))
	for _, tagID := range uniqueStrings(tagIDs) {
		value, err := s.GetTagValue(ctx, projectID, tagID)
		if err != nil {
			return nil, err
		}
		if compact {
			value = compactMqttTagValue(value)
		}
		result = append(result, value)
	}
	return result, nil
}

func compactMqttTagValue(value map[string]any) map[string]any {
	return map[string]any{
		"tagId":          value["tagId"],
		"subscriptionId": value["subscriptionId"],
		"value":          value["value"],
		"parsedValue":    value["parsedValue"],
		"quality":        value["quality"],
		"qualityCode":    value["qualityCode"],
		"timestamp":      value["timestamp"],
		"receivedAt":     value["receivedAt"],
		"error":          value["error"],
	}
}

func (s *MqttService) resolveLatestTagValue(ctx context.Context, projectID string, tag repository.MqttTagRecord) (MqttTagValueSnapshot, error) {
	subscription, err := s.repository.GetSubscription(ctx, projectID, tag.SubscriptionID)
	if err != nil {
		return MqttTagValueSnapshot{}, err
	}

	messages, err := s.repository.ListMessagesAfter(ctx, projectID, subscription.ID, tag.CreatedAt, 100)
	if err != nil {
		return MqttTagValueSnapshot{}, err
	}
	if len(messages) == 0 {
		return BuildFallbackMqttTagSnapshot(tag, tag.UpdatedAt, ""), nil
	}

	for _, message := range messages {
		if snapshot, ok := BuildMqttTagSnapshotUpdateFromMessage(tag, message); ok {
			return snapshot, nil
		}
	}

	return BuildFallbackMqttTagSnapshot(tag, tag.UpdatedAt, ""), nil
}

func (s *MqttService) syncSubscriptionDatapoint(ctx context.Context, subscription repository.MqttSubscriptionRecord, userID string) error {
	connection, err := s.connections.GetByProjectAndID(ctx, subscription.ProjectID, subscription.ConnectionID)
	if err != nil {
		return err
	}
	basePath := "mqtt." + normalizeDatapointSegment(connection.Name) + "." + mqttSubscriptionPathSegment(subscription)
	path := s.allocateDataPointPath(ctx, subscription.ProjectID, basePath, subscription.ID, "mqtt.subscription")

	return s.upsertMqttDataPoint(ctx, repository.CreateDataPointParams{
		ProjectID:    subscription.ProjectID,
		UserID:       stringPtr(userID),
		Path:         path,
		Name:         subscription.Name,
		SourceType:   "mqtt.subscription",
		SourceID:     &subscription.ID,
		SourceConfig: map[string]any{"mode": "subscription", "connectionId": subscription.ConnectionID},
		DataType:     "object",
		Tags:         []any{},
		RefreshMode:  "subscription",
		Status:       "active",
		DisplayOrder: &subscription.Order,
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

	prefix := "mqtt." + normalizeDatapointSegment(connection.Name) + "." + mqttSubscriptionPathSegment(*subscription) + "."
	return s.syncTagDatapointWithPrefix(ctx, tag, userID, prefix)
}

func (s *MqttService) syncTagDatapointWithPrefix(ctx context.Context, tag repository.MqttTagRecord, userID string, prefix string) error {
	subscription, err := s.repository.GetSubscription(ctx, tag.ProjectID, tag.SubscriptionID)
	if err != nil {
		return err
	}
	basePath := prefix + normalizeDatapointSegment(tag.Name)
	path := s.allocateDataPointPath(ctx, tag.ProjectID, basePath, tag.ID, "mqtt.tag")

	return s.upsertMqttDataPoint(ctx, repository.CreateDataPointParams{
		ProjectID:  tag.ProjectID,
		UserID:     stringPtr(userID),
		Path:       path,
		Name:       tag.Name,
		SourceType: "mqtt.tag",
		SourceID:   &tag.ID,
		SourceConfig: map[string]any{
			"connectionId":   subscription.ConnectionID,
			"subscriptionId": tag.SubscriptionID,
		},
		DataType:     tag.DataType,
		Unit:         cloneOptionalString(tag.Unit),
		DefaultValue: cloneOptionalString(tag.DefaultValue),
		Tags:         []any{},
		RefreshMode:  "subscription",
		Status:       "active",
		DisplayOrder: &tag.Order,
	})
}

func (s *MqttService) upsertMqttDataPoint(ctx context.Context, input repository.CreateDataPointParams) error {
	existing, err := s.datapoints.GetByProjectAndSource(ctx, input.ProjectID, input.SourceType, *input.SourceID)
	if err == nil && existing != nil {
		input.UserID = stringPtr(valueOrDefault(input.UserID, existing.UpdatedBy))
		_, updateErr := s.datapoints.UpdateGeneratedOutput(ctx, existing.ID, input)
		return updateErr
	}

	if existingByPath, pathErr := s.datapoints.GetByProjectAndPath(ctx, input.ProjectID, input.Path); pathErr == nil && existingByPath != nil && existingByPath.Status == "invalid" && existingByPath.SourceType == input.SourceType {
		input.UserID = stringPtr(valueOrDefault(input.UserID, existingByPath.UpdatedBy))
		_, updateErr := s.datapoints.UpdateGeneratedOutput(ctx, existingByPath.ID, input)
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
		if record.Status == "invalid" && record.SourceType == sourceType {
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

func mqttSubscriptionPathSegment(subscription repository.MqttSubscriptionRecord) string {
	segment := normalizeDatapointSegment(subscription.Name)
	if segment != "unnamed" {
		return segment
	}
	return normalizeDatapointSegment(subscription.Topic)
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
		GroupID:          cloneOptionalString(record.GroupID),
		Name:             record.Name,
		Topic:            record.Topic,
		QOS:              record.QOS,
		UsageMode:        normalizeMqttSubscriptionUsageMode(record.UsageMode),
		Description:      cloneOptionalString(record.Description),
		MessageRetention: record.MessageRetention,
		DefaultBatchRule: cloneMap(record.DefaultBatchRule),
		Order:            record.Order,
		CreatedAt:        record.CreatedAt,
		UpdatedAt:        record.UpdatedAt,
	}
}

func toMqttSubscriptionGroup(record repository.MqttSubscriptionGroupRecord) MqttSubscriptionGroup {
	return MqttSubscriptionGroup{
		ID:           record.ID,
		ProjectID:    record.ProjectID,
		ConnectionID: record.ConnectionID,
		Name:         record.Name,
		ParentID:     cloneOptionalString(record.ParentID),
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	}
}

func toMqttSubscriptionGroupTree(records []repository.MqttSubscriptionGroupRecord) []*MqttSubscriptionGroup {
	nodes := make(map[string]*MqttSubscriptionGroup, len(records))
	result := make([]*MqttSubscriptionGroup, 0)
	for _, record := range records {
		group := toMqttSubscriptionGroup(record)
		nodes[group.ID] = &group
	}
	for _, record := range records {
		node := nodes[record.ID]
		if node == nil {
			continue
		}
		if record.ParentID != nil {
			if parent := nodes[*record.ParentID]; parent != nil {
				parent.Children = append(parent.Children, node)
				continue
			}
		}
		result = append(result, node)
	}
	return result
}

func toMqttTag(record repository.MqttTagRecord) MqttTag {
	return MqttTag{
		ID:             record.ID,
		ProjectID:      record.ProjectID,
		SubscriptionID: record.SubscriptionID,
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

func normalizeOptionalMqttID(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizeMqttSubscriptionGroupName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "分组名称不能为空")
	}
	if len([]rune(name)) > 100 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "分组名称不能超过 100 个字符")
	}
	return name, nil
}

func (s *MqttService) ensureSubscriptionGroupInConnection(ctx context.Context, projectID, connectionID string, groupID *string) error {
	if groupID == nil {
		return nil
	}
	if _, err := uuid.Parse(strings.TrimSpace(*groupID)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "groupId 格式无效", err)
	}
	group, err := s.repository.GetSubscriptionGroup(ctx, projectID, *groupID)
	if err != nil {
		return err
	}
	if group.ConnectionID != connectionID {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "订阅分组不属于当前连接")
	}
	return nil
}

func isMqttManagedMessageConnectionType(connectionType string) bool {
	switch strings.TrimSpace(strings.ToLower(connectionType)) {
	case "mqtt", "builtin.message":
		return true
	default:
		return false
	}
}

func findMqttSubscriptionGroupRecord(records []repository.MqttSubscriptionGroupRecord, groupID string) (repository.MqttSubscriptionGroupRecord, bool) {
	for _, record := range records {
		if record.ID == groupID {
			return record, true
		}
	}
	return repository.MqttSubscriptionGroupRecord{}, false
}

func isDescendantMqttSubscriptionGroup(records []repository.MqttSubscriptionGroupRecord, groupID, possibleDescendantID string) bool {
	currentID := possibleDescendantID
	visited := map[string]struct{}{}
	for currentID != "" {
		if currentID == groupID {
			return true
		}
		if _, exists := visited[currentID]; exists {
			return false
		}
		visited[currentID] = struct{}{}
		current, ok := findMqttSubscriptionGroupRecord(records, currentID)
		if !ok || current.ParentID == nil {
			return false
		}
		currentID = *current.ParentID
	}
	return false
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
