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

var allowedHTTPMethods = map[string]struct{}{
	"GET":    {},
	"POST":   {},
	"PUT":    {},
	"PATCH":  {},
	"DELETE": {},
}

var allowedKafkaStartPositions = map[string]struct{}{
	"latest":   {},
	"earliest": {},
}

var allowedRedisModes = map[string]struct{}{
	"standalone": {},
	"sentinel":   {},
	"cluster":    {},
}

// ProtocolConnection 表示 wave1 协议连接响应结构。
type ProtocolConnection struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"projectId"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// KafkaPreview 表示 Kafka 预览响应结构。
type KafkaPreview struct {
	Topic   string `json:"topic"`
	Payload string `json:"payload"`
}

// CreateKafkaConfigInput 表示创建 Kafka 配置的业务输入。
type CreateKafkaConfigInput struct {
	Name          string
	Status        string
	Brokers       string
	Topic         string
	ConsumerGroup string
	StartPosition string
	Options       map[string]any
}

// CreateHTTPConfigInput 表示创建 HTTP 配置的业务输入。
type CreateHTTPConfigInput struct {
	Name         string
	Status       string
	BaseURL      string
	Method       string
	Headers      map[string]any
	TimeoutMS    *int
	BodyTemplate map[string]any
}

// CreateWebSocketConfigInput 表示创建 WebSocket 配置的业务输入。
type CreateWebSocketConfigInput struct {
	Name                string
	Status              string
	URL                 string
	Topic               *string
	Headers             map[string]any
	HeartbeatIntervalMS *int
}

// CreateRedisConfigInput 表示创建 Redis 配置的业务输入。
type CreateRedisConfigInput struct {
	Name       string
	Status     string
	Address    string
	DB         *int
	Username   *string
	Password   *string
	KeyPattern string
	Mode       string
	Options    map[string]any
}

// ProtocolWave1Service 负责第一波协议配置的输入校验和结果映射。
type ProtocolWave1Service struct {
	repository *repository.ProtocolWave1Repository
}

// NewProtocolWave1Service 创建第一波协议服务。
func NewProtocolWave1Service(repo *repository.ProtocolWave1Repository) *ProtocolWave1Service {
	return &ProtocolWave1Service{repository: repo}
}

// CreateKafkaConfig 创建 Kafka Source 配置。
func (s *ProtocolWave1Service) CreateKafkaConfig(ctx context.Context, projectID, userID string, input CreateKafkaConfigInput) (*ProtocolConnection, error) {
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
	status, err := normalizeProtocolStatus(input.Status)
	if err != nil {
		return nil, err
	}
	brokers := strings.TrimSpace(input.Brokers)
	if brokers == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "brokers 不能为空")
	}
	topic := strings.TrimSpace(input.Topic)
	if topic == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "topic 不能为空")
	}
	consumerGroup := strings.TrimSpace(input.ConsumerGroup)
	if consumerGroup == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "consumerGroup 不能为空")
	}
	startPosition := strings.TrimSpace(strings.ToLower(input.StartPosition))
	if startPosition == "" {
		startPosition = "latest"
	}
	if _, ok := allowedKafkaStartPositions[startPosition]; !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "startPosition 仅支持 latest/earliest")
	}

	record, err := s.repository.CreateKafkaConfig(ctx, repository.CreateKafkaConfigParams{
		ProjectID:     projectID,
		UserID:        userID,
		Name:          name,
		Status:        status,
		Brokers:       brokers,
		Topic:         topic,
		ConsumerGroup: consumerGroup,
		StartPosition: startPosition,
		Options:       cloneMap(input.Options),
	})
	if err != nil {
		return nil, err
	}

	connection := toProtocolConnection(*record)
	return &connection, nil
}

// PreviewKafkaTopic 返回 Kafka 预览结果。
func (s *ProtocolWave1Service) PreviewKafkaTopic(ctx context.Context, projectID, connectionID string) ([]KafkaPreview, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}

	records, err := s.repository.PreviewKafkaTopic(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}

	result := make([]KafkaPreview, 0, len(records))
	for _, record := range records {
		result = append(result, KafkaPreview{
			Topic:   record.Topic,
			Payload: record.Payload,
		})
	}
	return result, nil
}

// CreateHTTPConfig 创建 HTTP Source 配置。
func (s *ProtocolWave1Service) CreateHTTPConfig(ctx context.Context, projectID, userID string, input CreateHTTPConfigInput) (*ProtocolConnection, error) {
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
	status, err := normalizeProtocolStatus(input.Status)
	if err != nil {
		return nil, err
	}
	baseURL := strings.TrimSpace(input.BaseURL)
	if baseURL == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "baseUrl 不能为空")
	}
	method := strings.ToUpper(strings.TrimSpace(input.Method))
	if method == "" {
		method = "GET"
	}
	if _, ok := allowedHTTPMethods[method]; !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "HTTP method 不受支持")
	}

	timeoutMS := 30000
	if input.TimeoutMS != nil {
		if *input.TimeoutMS < 0 {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "timeoutMs 不能小于 0")
		}
		timeoutMS = *input.TimeoutMS
	}

	record, err := s.repository.CreateHTTPConfig(ctx, repository.CreateHTTPConfigParams{
		ProjectID:     projectID,
		UserID:        userID,
		Name:          name,
		Status:        status,
		BaseURL:       baseURL,
		Method:        method,
		Headers:       cloneMap(input.Headers),
		TimeoutMS:     timeoutMS,
		BodyTemplate:  cloneMap(input.BodyTemplate),
		HasBodyObject: input.BodyTemplate != nil,
	})
	if err != nil {
		return nil, err
	}

	connection := toProtocolConnection(*record)
	return &connection, nil
}

// CreateWebSocketConfig 创建 WebSocket Source 配置。
func (s *ProtocolWave1Service) CreateWebSocketConfig(ctx context.Context, projectID, userID string, input CreateWebSocketConfigInput) (*ProtocolConnection, error) {
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
	status, err := normalizeProtocolStatus(input.Status)
	if err != nil {
		return nil, err
	}
	url := strings.TrimSpace(input.URL)
	if url == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "url 不能为空")
	}
	heartbeatIntervalMS := 30000
	if input.HeartbeatIntervalMS != nil {
		if *input.HeartbeatIntervalMS < 0 {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "heartbeatIntervalMs 不能小于 0")
		}
		heartbeatIntervalMS = *input.HeartbeatIntervalMS
	}
	var topic *string
	if input.Topic != nil {
		value := strings.TrimSpace(*input.Topic)
		if value != "" {
			topic = &value
		}
	}

	record, err := s.repository.CreateWebSocketConfig(ctx, repository.CreateWebSocketConfigParams{
		ProjectID:           projectID,
		UserID:              userID,
		Name:                name,
		Status:              status,
		URL:                 url,
		Topic:               topic,
		Headers:             cloneMap(input.Headers),
		HeartbeatIntervalMS: heartbeatIntervalMS,
	})
	if err != nil {
		return nil, err
	}

	connection := toProtocolConnection(*record)
	return &connection, nil
}

// CreateRedisConfig 创建 Redis Source 配置。
func (s *ProtocolWave1Service) CreateRedisConfig(ctx context.Context, projectID, userID string, input CreateRedisConfigInput) (*ProtocolConnection, error) {
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
	status, err := normalizeProtocolStatus(input.Status)
	if err != nil {
		return nil, err
	}
	address := strings.TrimSpace(input.Address)
	if address == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "address 不能为空")
	}
	db := 0
	if input.DB != nil {
		if *input.DB < 0 {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "db 不能小于 0")
		}
		db = *input.DB
	}
	keyPattern := strings.TrimSpace(input.KeyPattern)
	if keyPattern == "" {
		keyPattern = "*"
	}
	mode := strings.TrimSpace(strings.ToLower(input.Mode))
	if mode == "" {
		mode = "standalone"
	}
	if _, ok := allowedRedisModes[mode]; !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "redis mode 不受支持")
	}

	record, err := s.repository.CreateRedisConfig(ctx, repository.CreateRedisConfigParams{
		ProjectID:  projectID,
		UserID:     userID,
		Name:       name,
		Status:     status,
		Address:    address,
		DB:         db,
		Username:   normalizeOptionalText(input.Username),
		Password:   normalizeOptionalText(input.Password),
		KeyPattern: keyPattern,
		Mode:       mode,
		Options:    cloneMap(input.Options),
	})
	if err != nil {
		return nil, err
	}

	connection := toProtocolConnection(*record)
	return &connection, nil
}

func toProtocolConnection(record repository.ProtocolConnectionRecord) ProtocolConnection {
	return ProtocolConnection{
		ID:        record.ID,
		ProjectID: record.ProjectID,
		Name:      record.Name,
		Type:      record.Type,
		Status:    record.Status,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}
}

func normalizeProtocolStatus(status string) (string, error) {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		return "disconnected", nil
	}
	if _, ok := allowedConnectionStatus[status]; !ok {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接状态不受支持")
	}
	return status, nil
}

func validateProtocolConnectionID(connectionID string) error {
	if _, err := uuid.Parse(strings.TrimSpace(connectionID)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "connectionId 格式无效", err)
	}
	return nil
}
