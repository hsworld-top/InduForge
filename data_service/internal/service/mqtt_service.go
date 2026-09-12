package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

var allowedMqttProtocols = map[string]struct{}{
	"mqtt":  {},
	"mqtts": {},
	"ws":    {},
	"wss":   {},
}

// MqttConnection 表示 MQTT 连接响应模型。
type MqttConnection struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"projectId"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// MqttConnectionStatus 表示连接状态响应模型。
type MqttConnectionStatus struct {
	Status string `json:"status"`
}

// MqttMessage 表示 MQTT 消息响应模型。
type MqttMessage struct {
	ID             int64     `json:"id"`
	SubscriptionID string    `json:"subscriptionId"`
	Topic          string    `json:"topic"`
	Payload        string    `json:"payload"`
	QOS            int       `json:"qos"`
	ReceivedAt     time.Time `json:"receivedAt"`
}

// MqttMessageListResult 返回消息列表及后端实际采用的限制数。
type MqttMessageListResult struct {
	Messages []MqttMessage
	Limit    int
}

type MqttPublishInput struct {
	Topic   string
	Payload any
	QOS     *int
}

type MqttPublishResult struct {
	Topic    string   `json:"topic"`
	QOS      int      `json:"qos"`
	Warnings []string `json:"warnings"`
}

// CreateMqttConnectionInput 表示创建 MQTT 连接的业务输入。
type CreateMqttConnectionInput struct {
	Name             string
	Enabled          *bool
	BrokerURL        string
	Protocol         string
	Port             *int
	ClientID         *string
	Username         *string
	Password         *string
	Secrets          map[string]string
	ClearSecretKeys  []string
	Keepalive        *int
	CleanSession     *bool
	QOS              *int
	ReconnectPeriod  *int
	ConnectTimeoutMS *int
	Will             map[string]any
	SSLConfig        map[string]any
}

// MqttService 承载 MQTT 领域接口的输入校验和响应映射。
// 说明：MQTT 是 开发态协议 的样板协议，配置、短时预览、Tag/Subscription 映射和 artifact
// 都以它作为最完整的中心侧协议基线。
type MqttService struct {
	repository  *repository.MqttRepository
	connections *repository.ConnectionRepository
	datapoints  *repository.DataPointRepository
	runtime     *MqttConnectionRuntimeManager
	secrets     *repository.ConnectionSecretRepository

	builtinMessageHubAddr     string
	builtinMessageHubUsername string
	builtinMessageHubPassword string
}

// SetSecretRepository 配置 MQTT 运行时密钥解析器；API 响应始终不返回明文。
func (s *MqttService) SetSecretRepository(secrets *repository.ConnectionSecretRepository) {
	s.secrets = secrets
}

// NewMqttService 创建 MQTT 领域服务。
func NewMqttService(repo *repository.MqttRepository, connectionRepo *repository.ConnectionRepository, datapointRepo *repository.DataPointRepository) *MqttService {
	return &MqttService{
		repository:  repo,
		connections: connectionRepo,
		datapoints:  datapointRepo,
		runtime:     NewMqttConnectionRuntimeManager(),
	}
}

func (s *MqttService) ConfigureBuiltinMessageHub(addr, username, password string) {
	if s == nil {
		return
	}
	s.builtinMessageHubAddr = strings.TrimSpace(addr)
	s.builtinMessageHubUsername = strings.TrimSpace(username)
	s.builtinMessageHubPassword = password
	if s.runtime != nil {
		s.runtime.ConfigureBuiltinMessageHub(addr, username, password)
	}
}

func extractMqttSecrets(input CreateMqttConnectionInput) (map[string]string, map[string]any) {
	secrets := map[string]string{}
	for key, value := range input.Secrets {
		if strings.TrimSpace(key) != "" && value != "" {
			secrets[strings.TrimSpace(key)] = value
		}
	}
	if input.Password != nil && *input.Password != "" {
		secrets["password"] = *input.Password
	}
	sslConfig := cloneMap(input.SSLConfig)
	for _, key := range []string{"ca", "cert", "key"} {
		if value := strings.TrimSpace(toString(sslConfig[key])); value != "" {
			secrets["tls."+key] = value
		}
		delete(sslConfig, key)
	}
	return secrets, sslConfig
}

func (s *MqttService) hydrateMqttSecrets(ctx context.Context, connection *repository.MqttConnectionDetailRecord) error {
	if s.secrets == nil || connection == nil || connection.Type != "mqtt" {
		return nil
	}
	values, err := s.secrets.ResolveAll(ctx, connection.ID)
	if err != nil {
		return err
	}
	if password, ok := values["password"]; ok {
		connection.Password = &password
	}
	if connection.SSLConfig == nil {
		connection.SSLConfig = map[string]any{}
	}
	for _, key := range []string{"ca", "cert", "key"} {
		if value, ok := values["tls."+key]; ok {
			connection.SSLConfig[key] = value
		}
	}
	return nil
}

// CreateConnection 创建 MQTT 连接。
func (s *MqttService) CreateConnection(ctx context.Context, projectID, userID string, input CreateMqttConnectionInput) (*MqttConnection, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}

	name, err := normalizeConnectionName(input.Name)
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
	keepalive, err := normalizeNonNegativeInt(input.Keepalive, 60, "keepalive 不能小于 0")
	if err != nil {
		return nil, err
	}
	qos, err := normalizeMqttQOS(input.QOS)
	if err != nil {
		return nil, err
	}
	reconnectPeriod, err := normalizeNonNegativeInt(input.ReconnectPeriod, 5000, "reconnectPeriod 不能小于 0")
	if err != nil {
		return nil, err
	}
	connectTimeoutMS, err := normalizeNonNegativeInt(input.ConnectTimeoutMS, 30000, "connectTimeout 不能小于 0")
	if err != nil {
		return nil, err
	}

	cleanSession := true
	if input.CleanSession != nil {
		cleanSession = *input.CleanSession
	}

	secrets, sslConfig := extractMqttSecrets(input)
	record, err := s.repository.CreateConnection(ctx, repository.CreateMqttConnectionParams{
		ProjectID:        projectID,
		UserID:           userID,
		Name:             name,
		IsEnabled:        input.Enabled,
		BrokerURL:        brokerURL,
		Protocol:         protocol,
		Port:             port,
		ClientID:         normalizeOptionalText(input.ClientID),
		Username:         normalizeOptionalText(input.Username),
		Secrets:          secrets,
		ClearSecretKeys:  input.ClearSecretKeys,
		Keepalive:        keepalive,
		CleanSession:     cleanSession,
		QOS:              qos,
		ReconnectPeriod:  reconnectPeriod,
		ConnectTimeoutMS: connectTimeoutMS,
		Will:             cloneMap(input.Will),
		SSLConfig:        sslConfig,
	})
	if err != nil {
		return nil, err
	}

	connection := toMqttConnection(*record)
	return &connection, nil
}

// StartConnection 启动指定 MQTT 连接。
func (s *MqttService) StartConnection(ctx context.Context, projectID, connectionID string) (*MqttConnectionStatus, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}

	summary, err := s.repository.GetConnectionSummary(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	if s.runtime != nil {
		if summary.Type == "builtin.message" {
			if err := s.runtime.ConnectBuiltin(ctx, *summary); err != nil {
				return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接 消息库失败", err)
			}
		} else {
			connection, err := s.repository.GetConnectionDetail(ctx, projectID, connectionID)
			if err != nil {
				return nil, err
			}
			if err := s.hydrateMqttSecrets(ctx, connection); err != nil {
				return nil, err
			}
			if err := s.runtime.ConnectExternal(ctx, *connection); err != nil {
				return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接 MQTT Broker 失败", err)
			}
		}
	}

	record, err := s.repository.StartConnection(ctx, projectID, connectionID)
	if err != nil {
		if s.runtime != nil {
			s.runtime.Disconnect(projectID, connectionID)
		}
		return nil, err
	}

	return &MqttConnectionStatus{Status: record.Status}, nil
}

// ListMessages 查询指定订阅的消息列表。
func (s *MqttService) ListMessages(ctx context.Context, projectID, subscriptionID string, limit int) (*MqttMessageListResult, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if _, err := uuid.Parse(strings.TrimSpace(subscriptionID)); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "subscriptionId 格式无效", err)
	}

	normalizedLimit := limit
	if normalizedLimit <= 0 {
		normalizedLimit = 100
	}
	if normalizedLimit > 5000 {
		normalizedLimit = 5000
	}

	records, err := s.repository.ListMessages(ctx, projectID, subscriptionID, normalizedLimit)
	if err != nil {
		return nil, err
	}

	messages := make([]MqttMessage, 0, len(records))
	for _, record := range records {
		messages = append(messages, MqttMessage{
			ID:             record.ID,
			SubscriptionID: record.SubscriptionID,
			Topic:          record.Topic,
			Payload:        record.Payload,
			QOS:            record.QOS,
			ReceivedAt:     record.ReceivedAt,
		})
	}
	return &MqttMessageListResult{Messages: messages, Limit: normalizedLimit}, nil
}

// ClearMessages 清空指定订阅的工作台消息缓存。
func (s *MqttService) ClearMessages(ctx context.Context, projectID, subscriptionID string) error {
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	if _, err := uuid.Parse(strings.TrimSpace(subscriptionID)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "subscriptionId 格式无效", err)
	}
	return s.repository.ClearMessages(ctx, projectID, subscriptionID)
}

// PublishMessage 使用已保存的 MQTT 连接做一次发布测试。
func (s *MqttService) PublishMessage(ctx context.Context, projectID, connectionID string, input MqttPublishInput) (*MqttPublishResult, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}
	topic := strings.Trim(strings.TrimSpace(input.Topic), "/")
	if topic == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "topic 不能为空")
	}
	qos, err := normalizeMqttQOS(input.QOS)
	if err != nil {
		return nil, err
	}

	connection, err := s.repository.GetPublishConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	if input.QOS == nil {
		qos = connection.QOS
	}
	if connection.Type == "builtin.message" {
		if err := s.applyBuiltinMessagePublishConnection(connection); err != nil {
			return nil, err
		}
	} else if connection.Type == "mqtt" && s.secrets != nil {
		password, ok, err := s.secrets.Resolve(ctx, connection.ID, "password")
		if err != nil {
			return nil, err
		}
		if ok {
			connection.Password = &password
		}
	}

	payload, err := json.Marshal(input.Payload)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "MQTT payload 必须可序列化为 JSON", err)
	}

	if err := publishMQTTWorkbenchMessage(ctx, connection, topic, payload, qos); err != nil {
		return nil, err
	}

	result := &MqttPublishResult{Topic: topic, QOS: qos, Warnings: []string{}}
	if _, recordErr := s.repository.CreatePublishedMessage(ctx, repository.CreateMqttMessageParams{
		ProjectID:      projectID,
		ConnectionID:   connectionID,
		Topic:          topic,
		Payload:        string(payload),
		QOS:            qos,
		ReceivedAt:     time.Now().UTC(),
		Metadata:       map[string]any{"source": "workbench-publish"},
		RetentionLimit: 100,
	}); recordErr != nil {
		result.Warnings = append(result.Warnings, "消息已发送，但保存开发态预览记录失败")
	}

	return result, nil
}

// PublishSubscriptionMessage 按订阅既有连接、主题和 QOS 发布数据点消息。
func (s *MqttService) PublishSubscriptionMessage(ctx context.Context, projectID, subscriptionID string, payload any) (*MqttPublishResult, error) {
	subscription, err := s.GetSubscription(ctx, projectID, subscriptionID)
	if err != nil {
		return nil, err
	}
	qos := subscription.QOS
	return s.PublishMessage(ctx, projectID, subscription.ConnectionID, MqttPublishInput{
		Topic:   subscription.Topic,
		Payload: payload,
		QOS:     &qos,
	})
}

func (s *MqttService) applyBuiltinMessagePublishConnection(connection *repository.MqttPublishConnectionRecord) error {
	if connection == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "消息库连接不存在")
	}
	addr := strings.TrimSpace(s.builtinMessageHubAddr)
	if addr == "" {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "消息库服务地址未配置")
	}
	connection.BrokerURL = addr
	connection.Protocol = "mqtt"
	connection.Port = 0
	connection.CleanSession = true
	if s.builtinMessageHubUsername != "" {
		username := s.builtinMessageHubUsername
		password := s.builtinMessageHubPassword
		connection.Username = &username
		connection.Password = &password
	} else {
		connection.Username = nil
		connection.Password = nil
	}
	return nil
}

func toMqttConnection(record repository.MqttConnectionRecord) MqttConnection {
	return MqttConnection{
		ID:        record.ID,
		ProjectID: record.ProjectID,
		Name:      record.Name,
		Type:      record.Type,
		Enabled:   record.IsEnabled,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}
}

func publishMQTTWorkbenchMessage(ctx context.Context, connection *repository.MqttPublishConnectionRecord, topic string, payload []byte, qos int) error {
	if connection == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "MQTT 连接不存在")
	}
	timeout := time.Duration(connection.ConnectTimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	options := mqtt.NewClientOptions()
	options.AddBroker(repository.BuildMqttBrokerAddress(connection.Protocol, connection.BrokerURL, connection.Port))
	options.SetClientID(buildMqttPublishClientID(connection))
	options.SetCleanSession(connection.CleanSession)
	options.SetConnectTimeout(timeout)
	if connection.Keepalive > 0 {
		options.SetKeepAlive(time.Duration(connection.Keepalive) * time.Second)
	}
	if connection.Username != nil && strings.TrimSpace(*connection.Username) != "" {
		options.SetUsername(strings.TrimSpace(*connection.Username))
	}
	if connection.Password != nil && strings.TrimSpace(*connection.Password) != "" {
		options.SetPassword(strings.TrimSpace(*connection.Password))
	}

	client := mqtt.NewClient(options)
	token := client.Connect()
	if !waitMqttToken(ctx, token, timeout) {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "MQTT 连接超时")
	}
	if err := token.Error(); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "MQTT 连接失败", err)
	}
	defer client.Disconnect(250)

	token = client.Publish(topic, byte(qos), false, payload)
	if !waitMqttToken(ctx, token, timeout) {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "MQTT 发布超时")
	}
	if err := token.Error(); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "MQTT 发布失败", err)
	}
	return nil
}

func buildMqttPublishClientID(connection *repository.MqttPublishConnectionRecord) string {
	if connection != nil && connection.ClientID != nil && strings.TrimSpace(*connection.ClientID) != "" {
		return strings.TrimSpace(*connection.ClientID) + "-publish"
	}
	if connection != nil && strings.TrimSpace(connection.ID) != "" {
		return "workbench-publish-" + connection.ID
	}
	return fmt.Sprintf("workbench-publish-%d", time.Now().UnixNano())
}

func normalizeMqttBrokerURL(brokerURL string) (string, error) {
	brokerURL = strings.TrimSpace(brokerURL)
	if brokerURL == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "brokerUrl 不能为空")
	}
	if len([]rune(brokerURL)) > 500 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "brokerUrl 长度不能超过 500 个字符")
	}
	return brokerURL, nil
}

func normalizeMqttProtocol(protocol string) (string, error) {
	protocol = strings.TrimSpace(strings.ToLower(protocol))
	if protocol == "" {
		return "mqtt", nil
	}
	if _, ok := allowedMqttProtocols[protocol]; !ok {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "MQTT protocol 不受支持")
	}
	return protocol, nil
}

func normalizeMqttPort(port *int) (int, error) {
	if port == nil {
		return 1883, nil
	}
	if *port <= 0 || *port > 65535 {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "MQTT port 范围必须在 1 到 65535 之间")
	}
	return *port, nil
}

func normalizeMqttQOS(qos *int) (int, error) {
	if qos == nil {
		return 0, nil
	}
	if *qos < 0 || *qos > 2 {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "MQTT qos 仅支持 0/1/2")
	}
	return *qos, nil
}

func normalizeNonNegativeInt(value *int, defaultValue int, errorMessage string) (int, error) {
	if value == nil {
		return defaultValue, nil
	}
	if *value < 0 {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, errorMessage)
	}
	return *value, nil
}

func normalizeOptionalText(value *string) *string {
	if value == nil {
		return nil
	}
	text := strings.TrimSpace(*value)
	if text == "" {
		return nil
	}
	return &text
}
