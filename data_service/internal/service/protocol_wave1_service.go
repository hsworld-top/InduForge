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
	Name            string
	Status          string
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
	Status      string
	Description string
}

// CreateWebSocketConfigInput 表示创建 WebSocket 配置的业务输入。
type CreateWebSocketConfigInput struct {
	Name        string
	Status      string
	Description string
}

// CreateRedisConfigInput 表示创建 Redis 配置的业务输入。
type CreateRedisConfigInput struct {
	Name            string
	Status          string
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

// ProtocolWave1Service 负责第一波协议配置的输入校验和结果映射。
// 说明：`kafka/http/websocket/redis` 属于 Phase 1 正式协议范围，但不是 MQTT 那种完整运行态深度。
// 当前只保证配置层与 artifact 层稳定；其中仅 Kafka 暂时保留 mock preview 作为联调样本。
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
		Status:        status,
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

	record, err := s.repository.CreateHTTPConfig(ctx, repository.CreateHTTPConfigParams{
		ProjectID:   projectID,
		UserID:      userID,
		Name:        name,
		Status:      status,
		Description: strings.TrimSpace(input.Description),
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
	record, err := s.repository.CreateWebSocketConfig(ctx, repository.CreateWebSocketConfigParams{
		ProjectID:   projectID,
		UserID:      userID,
		Name:        name,
		Status:      status,
		Description: strings.TrimSpace(input.Description),
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
	if err := validateConnectionOptionsContainNoSecrets(input.Options); err != nil {
		return nil, err
	}

	record, err := s.repository.CreateRedisConfig(ctx, repository.CreateRedisConfigParams{
		ProjectID:  projectID,
		UserID:     userID,
		Name:       name,
		Status:     status,
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

func (s *ProtocolWave1Service) UpdateKafkaConfig(ctx context.Context, projectID, connectionID, userID string, input CreateKafkaConfigInput) (*ProtocolConnection, error) {
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}
	// 复用创建校验但改为调用更新仓储，确保创建和编辑语义一致。
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
	record, err := s.repository.UpdateKafkaConfig(ctx, connectionID, repository.CreateKafkaConfigParams{ProjectID: projectID, UserID: userID, Name: name, Status: status, Brokers: brokers, Topic: strings.TrimSpace(input.Topic), ConsumerGroup: strings.TrimSpace(input.ConsumerGroup), StartPosition: start, Options: options, Secrets: secrets, ClearSecretKeys: input.ClearSecretKeys})
	if err != nil {
		return nil, err
	}
	result := toProtocolConnection(*record)
	return &result, nil
}

func (s *ProtocolWave1Service) UpdateHTTPConfig(ctx context.Context, projectID, connectionID, userID string, input CreateHTTPConfigInput) (*ProtocolConnection, error) {
	return s.updateSimple(ctx, projectID, connectionID, userID, "http", input.Name, input.Status, input.Description)
}
func (s *ProtocolWave1Service) UpdateWebSocketConfig(ctx context.Context, projectID, connectionID, userID string, input CreateWebSocketConfigInput) (*ProtocolConnection, error) {
	return s.updateSimple(ctx, projectID, connectionID, userID, "websocket", input.Name, input.Status, input.Description)
}
func (s *ProtocolWave1Service) updateSimple(ctx context.Context, projectID, connectionID, userID, protocolType, rawName, rawStatus, description string) (*ProtocolConnection, error) {
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
	status, err := normalizeProtocolStatus(rawStatus)
	if err != nil {
		return nil, err
	}
	record, err := s.repository.UpdateSimpleConfig(ctx, connectionID, protocolType, projectID, userID, name, status, map[string]any{"mode": "workbench", "description": strings.TrimSpace(description)})
	if err != nil {
		return nil, err
	}
	result := toProtocolConnection(*record)
	return &result, nil
}

func (s *ProtocolWave1Service) UpdateRedisConfig(ctx context.Context, projectID, connectionID, userID string, input CreateRedisConfigInput) (*ProtocolConnection, error) {
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
	record, err := s.repository.UpdateRedisConfig(ctx, connectionID, repository.CreateRedisConfigParams{ProjectID: projectID, UserID: userID, Name: name, Status: status, Address: address, DB: db, Username: normalizeOptionalText(input.Username), KeyPattern: keyPattern, Mode: mode, Options: cloneMap(input.Options), Secrets: mergePasswordSecret(input.Secrets, input.Password), ClearSecretKeys: input.ClearSecretKeys})
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
