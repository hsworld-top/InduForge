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

// ProtocolConnection 表示 协议连接 协议连接响应结构。
type ProtocolConnection struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"projectId"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CreateKafkaConfigInput 表示创建 Kafka 配置的业务输入。
type CreateKafkaConfigInput struct {
	Name            string
	Enabled         *bool
	Brokers         string
	Topic           string
	ConsumerGroup   string
	StartPosition   string
	Options         map[string]any
	Secrets         map[string]string
	ClearSecretKeys []string
}

// CreateHTTPConfigInput 表示创建 HTTP 配置的业务输入。
type CreateHTTPConfigInput struct {
	Name        string
	Enabled     *bool
	Description string
}

// CreateWebSocketConfigInput 表示创建 WebSocket 配置的业务输入。
type CreateWebSocketConfigInput struct {
	Name        string
	Enabled     *bool
	Description string
}

// CreateRedisConfigInput 表示创建 Redis 配置的业务输入。
type CreateRedisConfigInput struct {
	Name            string
	Enabled         *bool
	Address         string
	DB              *int
	Username        *string
	Password        *string
	KeyPattern      string
	Mode            string
	Options         map[string]any
	Secrets         map[string]string
	ClearSecretKeys []string
}

// ProtocolConnectionService 负责协议连接配置的输入校验和结果映射。
// 说明：`kafka/http/websocket/redis` 属于 开发态协议 正式协议范围，但不是 MQTT 那种完整运行态深度。
// 当前只保证配置层与 artifact 层稳定；其中仅 Kafka 暂时保留 mock preview 作为联调样本。
type ProtocolConnectionService struct {
	repository *repository.ProtocolConnectionRepository
}

// NewProtocolConnectionService 创建协议连接服务。
func NewProtocolConnectionService(repo *repository.ProtocolConnectionRepository) *ProtocolConnectionService {
	return &ProtocolConnectionService{repository: repo}
}

// CreateKafkaConfig 创建 Kafka Source 配置。
func (s *ProtocolConnectionService) CreateKafkaConfig(ctx context.Context, projectID, userID string, input CreateKafkaConfigInput) (*ProtocolConnection, error) {
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
	brokers := strings.TrimSpace(input.Brokers)
	if brokers == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "brokers 不能为空")
	}
	topic := strings.TrimSpace(input.Topic)
	consumerGroup := strings.TrimSpace(input.ConsumerGroup)
	startPosition := strings.TrimSpace(strings.ToLower(input.StartPosition))
	if startPosition == "" {
		startPosition = "latest"
	}
	if _, ok := allowedKafkaStartPositions[startPosition]; !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "startPosition 仅支持 latest/earliest")
	}
	options, secrets := splitKafkaSecrets(input.Options, input.Secrets)
	normalizedConfig, err := normalizeKafkaConnectionConfig(map[string]any{
		"brokers": brokers,
		"options": injectConnectionSecrets(map[string]any{"options": cloneMap(options)}, secrets)["options"],
	})
	if err != nil {
		return nil, err
	}
	options = scrubKafkaSecretOptions(mapFromAny(normalizedConfig["options"]))

	record, err := s.repository.CreateKafkaConfig(ctx, repository.CreateKafkaConfigParams{
		ProjectID:     projectID,
		UserID:        userID,
		Name:          name,
		IsEnabled:     input.Enabled,
		Brokers:       brokers,
		Topic:         topic,
		ConsumerGroup: consumerGroup,
		StartPosition: startPosition,
		Options:       options,
		Secrets:       secrets, ClearSecretKeys: input.ClearSecretKeys,
	})
	if err != nil {
		return nil, err
	}

	connection := toProtocolConnection(*record)
	return &connection, nil
}

// CreateHTTPConfig 创建 HTTP Source 配置。
func (s *ProtocolConnectionService) CreateHTTPConfig(ctx context.Context, projectID, userID string, input CreateHTTPConfigInput) (*ProtocolConnection, error) {
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
	record, err := s.repository.CreateHTTPConfig(ctx, repository.CreateHTTPConfigParams{
		ProjectID:   projectID,
		UserID:      userID,
		Name:        name,
		IsEnabled:   input.Enabled,
		Description: strings.TrimSpace(input.Description),
	})
	if err != nil {
		return nil, err
	}

	connection := toProtocolConnection(*record)
	return &connection, nil
}

// CreateWebSocketConfig 创建 WebSocket Source 配置。
func (s *ProtocolConnectionService) CreateWebSocketConfig(ctx context.Context, projectID, userID string, input CreateWebSocketConfigInput) (*ProtocolConnection, error) {
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
	record, err := s.repository.CreateWebSocketConfig(ctx, repository.CreateWebSocketConfigParams{
		ProjectID:   projectID,
		UserID:      userID,
		Name:        name,
		IsEnabled:   input.Enabled,
		Description: strings.TrimSpace(input.Description),
	})
	if err != nil {
		return nil, err
	}

	connection := toProtocolConnection(*record)
	return &connection, nil
}

// CreateRedisConfig 创建 Redis Source 配置。
func (s *ProtocolConnectionService) CreateRedisConfig(ctx context.Context, projectID, userID string, input CreateRedisConfigInput) (*ProtocolConnection, error) {
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
	if err := validateConnectionOptionsContainNoSecrets(input.Options); err != nil {
		return nil, err
	}

	record, err := s.repository.CreateRedisConfig(ctx, repository.CreateRedisConfigParams{
		ProjectID:  projectID,
		UserID:     userID,
		Name:       name,
		IsEnabled:  input.Enabled,
		Address:    address,
		DB:         db,
		Username:   normalizeOptionalText(input.Username),
		KeyPattern: keyPattern,
		Mode:       mode,
		Options:    cloneMap(input.Options),
		Secrets:    mergePasswordSecret(input.Secrets, input.Password), ClearSecretKeys: input.ClearSecretKeys,
	})
	if err != nil {
		return nil, err
	}

	connection := toProtocolConnection(*record)
	return &connection, nil
}

func (s *ProtocolConnectionService) UpdateKafkaConfig(ctx context.Context, projectID, connectionID, userID string, input CreateKafkaConfigInput) (*ProtocolConnection, error) {
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}
	// 复用创建校验但改为调用更新仓储，确保创建和编辑语义一致。
	name, err := normalizeConnectionName(input.Name)
	if err != nil {
		return nil, err
	}
	brokers := strings.TrimSpace(input.Brokers)
	if brokers == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "brokers 不能为空")
	}
	start := strings.ToLower(strings.TrimSpace(input.StartPosition))
	if start == "" {
		start = "latest"
	}
	if _, ok := allowedKafkaStartPositions[start]; !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "startPosition 仅支持 latest/earliest")
	}
	options, secrets := splitKafkaSecrets(input.Options, input.Secrets)
	normalized, err := normalizeKafkaConnectionConfig(map[string]any{"brokers": brokers, "options": injectConnectionSecrets(map[string]any{"options": options}, secrets)["options"]})
	if err != nil {
		return nil, err
	}
	options = scrubKafkaSecretOptions(mapFromAny(normalized["options"]))
	record, err := s.repository.UpdateKafkaConfig(ctx, connectionID, repository.CreateKafkaConfigParams{ProjectID: projectID, UserID: userID, Name: name, IsEnabled: input.Enabled, Brokers: brokers, Topic: strings.TrimSpace(input.Topic), ConsumerGroup: strings.TrimSpace(input.ConsumerGroup), StartPosition: start, Options: options, Secrets: secrets, ClearSecretKeys: input.ClearSecretKeys})
	if err != nil {
		return nil, err
	}
	result := toProtocolConnection(*record)
	return &result, nil
}

func (s *ProtocolConnectionService) UpdateHTTPConfig(ctx context.Context, projectID, connectionID, userID string, input CreateHTTPConfigInput) (*ProtocolConnection, error) {
	return s.updateSimple(ctx, projectID, connectionID, userID, "http", input.Name, input.Enabled, input.Description)
}
func (s *ProtocolConnectionService) UpdateWebSocketConfig(ctx context.Context, projectID, connectionID, userID string, input CreateWebSocketConfigInput) (*ProtocolConnection, error) {
	return s.updateSimple(ctx, projectID, connectionID, userID, "websocket", input.Name, input.Enabled, input.Description)
}
func (s *ProtocolConnectionService) updateSimple(ctx context.Context, projectID, connectionID, userID, protocolType, rawName string, isEnabled *bool, description string) (*ProtocolConnection, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	name, err := normalizeConnectionName(rawName)
	if err != nil {
		return nil, err
	}
	record, err := s.repository.UpdateSimpleConfig(ctx, connectionID, protocolType, projectID, userID, name, isEnabled, map[string]any{"mode": "workbench", "description": strings.TrimSpace(description)})
	if err != nil {
		return nil, err
	}
	result := toProtocolConnection(*record)
	return &result, nil
}

func (s *ProtocolConnectionService) UpdateRedisConfig(ctx context.Context, projectID, connectionID, userID string, input CreateRedisConfigInput) (*ProtocolConnection, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	name, err := normalizeConnectionName(input.Name)
	if err != nil {
		return nil, err
	}
	address := strings.TrimSpace(input.Address)
	if address == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "address 不能为空")
	}
	db := 0
	if input.DB != nil {
		db = *input.DB
	}
	if db < 0 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "db 不能小于 0")
	}
	keyPattern := strings.TrimSpace(input.KeyPattern)
	if keyPattern == "" {
		keyPattern = "*"
	}
	mode := strings.ToLower(strings.TrimSpace(input.Mode))
	if mode == "" {
		mode = "standalone"
	}
	if _, ok := allowedRedisModes[mode]; !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "redis mode 不受支持")
	}
	if err := validateConnectionOptionsContainNoSecrets(input.Options); err != nil {
		return nil, err
	}
	record, err := s.repository.UpdateRedisConfig(ctx, connectionID, repository.CreateRedisConfigParams{ProjectID: projectID, UserID: userID, Name: name, IsEnabled: input.Enabled, Address: address, DB: db, Username: normalizeOptionalText(input.Username), KeyPattern: keyPattern, Mode: mode, Options: cloneMap(input.Options), Secrets: mergePasswordSecret(input.Secrets, input.Password), ClearSecretKeys: input.ClearSecretKeys})
	if err != nil {
		return nil, err
	}
	result := toProtocolConnection(*record)
	return &result, nil
}

func mergePasswordSecret(input map[string]string, password *string) map[string]string {
	result := normalizeSecretInput(input)
	if password != nil && *password != "" {
		result["password"] = *password
	}
	return result
}
func splitKafkaSecrets(options map[string]any, input map[string]string) (map[string]any, map[string]string) {
	result := cloneMap(options)
	secrets := normalizeSecretInput(input)
	if value := toString(result["password"]); value != "" {
		secrets["option.password"] = value
	}
	delete(result, "password")
	ssl := mapFromAny(result["sslConfig"])
	for _, key := range []string{"ca", "cert", "key"} {
		if value := toString(ssl[key]); value != "" {
			secrets["tls."+key] = value
		}
		delete(ssl, key)
	}
	if len(ssl) > 0 {
		result["sslConfig"] = ssl
	} else {
		delete(result, "sslConfig")
	}
	return result, secrets
}
func scrubKafkaSecretOptions(options map[string]any) map[string]any {
	result := cloneMap(options)
	delete(result, "password")
	ssl := mapFromAny(result["sslConfig"])
	delete(ssl, "ca")
	delete(ssl, "cert")
	delete(ssl, "key")
	if len(ssl) > 0 {
		result["sslConfig"] = ssl
	} else {
		delete(result, "sslConfig")
	}
	return result
}

func toProtocolConnection(record repository.ProtocolConnectionRecord) ProtocolConnection {
	return ProtocolConnection{
		ID:        record.ID,
		ProjectID: record.ProjectID,
		Name:      record.Name,
		Type:      record.Type,
		Enabled:   record.IsEnabled,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}
}

func validateProtocolConnectionID(connectionID string) error {
	if _, err := uuid.Parse(strings.TrimSpace(connectionID)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "connectionId 格式无效", err)
	}
	return nil
}
