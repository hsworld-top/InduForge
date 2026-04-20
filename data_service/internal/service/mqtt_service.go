package service

import (
	"context"
	"net/http"
	"strings"
	"time"

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
	Status    string    `json:"status"`
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

// CreateMqttConnectionInput 表示创建 MQTT 连接的业务输入。
type CreateMqttConnectionInput struct {
	Name             string
	Status           string
	BrokerURL        string
	Protocol         string
	Port             *int
	ClientID         *string
	Username         *string
	Password         *string
	Keepalive        *int
	CleanSession     *bool
	QOS              *int
	ReconnectPeriod  *int
	ConnectTimeoutMS *int
	Will             map[string]any
	SSLConfig        map[string]any
}

// MqttService 承载 MQTT 领域接口的输入校验和响应映射。
// 说明：MQTT 是 Phase 1 的样板协议，配置、短时预览、Tag/Subscription 映射和 artifact
// 都以它作为最完整的中心侧协议基线。
type MqttService struct {
	repository  *repository.MqttRepository
	connections *repository.ConnectionRepository
	datapoints  *repository.DataPointRepository
}

// NewMqttService 创建 MQTT 领域服务。
func NewMqttService(repo *repository.MqttRepository, connectionRepo *repository.ConnectionRepository, datapointRepo *repository.DataPointRepository) *MqttService {
	return &MqttService{
		repository:  repo,
		connections: connectionRepo,
		datapoints:  datapointRepo,
	}
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

	record, err := s.repository.CreateConnection(ctx, repository.CreateMqttConnectionParams{
		ProjectID:        projectID,
		UserID:           userID,
		Name:             name,
		Status:           status,
		BrokerURL:        brokerURL,
		Protocol:         protocol,
		Port:             port,
		ClientID:         normalizeOptionalText(input.ClientID),
		Username:         normalizeOptionalText(input.Username),
		Password:         normalizeOptionalText(input.Password),
		Keepalive:        keepalive,
		CleanSession:     cleanSession,
		QOS:              qos,
		ReconnectPeriod:  reconnectPeriod,
		ConnectTimeoutMS: connectTimeoutMS,
		Will:             cloneMap(input.Will),
		SSLConfig:        cloneMap(input.SSLConfig),
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

	record, err := s.repository.StartConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}

	return &MqttConnectionStatus{Status: record.Status}, nil
}

// GetConnectionStatus 查询指定 MQTT 连接状态。
func (s *MqttService) GetConnectionStatus(ctx context.Context, projectID, connectionID string) (*MqttConnectionStatus, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}

	record, err := s.repository.GetConnectionStatus(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	return &MqttConnectionStatus{Status: record.Status}, nil
}

// ListMessages 查询指定订阅的消息列表。
func (s *MqttService) ListMessages(ctx context.Context, projectID, subscriptionID string, limit int) ([]MqttMessage, error) {
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
	if normalizedLimit > 500 {
		normalizedLimit = 500
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
	return messages, nil
}

func toMqttConnection(record repository.MqttConnectionRecord) MqttConnection {
	return MqttConnection{
		ID:        record.ID,
		ProjectID: record.ProjectID,
		Name:      record.Name,
		Type:      record.Type,
		Status:    record.Status,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}
}

func normalizeMqttInitialStatus(status string) (string, error) {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		return "disconnected", nil
	}
	if _, ok := allowedConnectionStatus[status]; !ok {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "MQTT 连接状态不受支持")
	}
	return status, nil
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
