package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

// publicConnectionTypeCategoryMap 只保留通用连接接口允许新建/改型的类型。
// 说明：协议专用配置不允许通过通用连接接口写入。
var publicConnectionTypeCategoryMap = map[string]string{
	"relational":         "database",
	"mqtt":               "message",
	"websocket":          "protocol",
	"http":               "api",
	"builtin.relation":   "builtin",
	"builtin.timeseries": "builtin",
	"builtin.realtime":   "builtin",
	"builtin.message":    "builtin",
}

// reservedPhase2ConnectionTypes 用于阻止工业协议从通用连接入口写入，避免只生成 metadata 而缺失专用配置表。
var reservedPhase2ConnectionTypes = map[string]string{
	"tdengine": "TDengine",
}

var allowedConnectionStatus = map[string]struct{}{
	"connected":    {},
	"disconnected": {},
	"error":        {},
	"unknown":      {},
}

var allowedKafkaSecurityProtocols = map[string]struct{}{
	"PLAINTEXT":      {},
	"SSL":            {},
	"SASL_PLAINTEXT": {},
	"SASL_SSL":       {},
}

var allowedKafkaSaslMechanisms = map[string]struct{}{
	"PLAIN":         {},
	"SCRAM-SHA-256": {},
	"SCRAM-SHA-512": {},
}

// Connection 表示面向 HTTP 层返回的连接对象。
type Connection struct {
	ID               string          `json:"id"`
	ProjectID        string          `json:"projectId"`
	TenantID         string          `json:"tenantId"`
	Name             string          `json:"name"`
	Type             string          `json:"type"`
	Status           string          `json:"status"`
	Config           map[string]any  `json:"config"`
	RelationalConfig map[string]any  `json:"relationalConfig,omitempty"`
	MqttConfig       map[string]any  `json:"mqttConfig,omitempty"`
	SecretStatus     map[string]bool `json:"secretStatus"`
	DisplayOrder     int             `json:"displayOrder"`
	VariableCount    int             `json:"variableCount"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
}

// ConnectionTestResult 表示连接测试响应。
type ConnectionTestResult struct {
	Connected bool   `json:"connected"`
	DBType    string `json:"dbType"`
	Type      string `json:"type,omitempty"`
	Detail    string `json:"detail,omitempty"`
	Message   string `json:"message"`
}

// CreateConnectionInput 表示创建连接的业务输入。
type CreateConnectionInput struct {
	Name   string
	Type   string
	Status string
	Config map[string]any
}

// UpdateConnectionInput 表示更新连接的业务输入。
type UpdateConnectionInput struct {
	Name      *string
	Type      *string
	Status    *string
	Config    map[string]any
	HasConfig bool
}

// ConnectionService 承载连接领域的基本业务校验与映射。
type ConnectionService struct {
	repository      *repository.ConnectionRepository
	builtinRuntime  *BuiltinRuntimeService
	workbenchGroups *WorkbenchGroupService
	secrets         *repository.ConnectionSecretRepository
}

// NewConnectionService 创建连接服务。
func NewConnectionService(repo *repository.ConnectionRepository, builtinRuntime ...*BuiltinRuntimeService) *ConnectionService {
	service := &ConnectionService{repository: repo}
	if len(builtinRuntime) > 0 {
		service.builtinRuntime = builtinRuntime[0]
	}
	return service
}

func (s *ConnectionService) SetWorkbenchGroupService(groups *WorkbenchGroupService) {
	if s != nil {
		s.workbenchGroups = groups
	}
}

func (s *ConnectionService) SetSecretRepository(secrets *repository.ConnectionSecretRepository) {
	if s != nil {
		s.secrets = secrets
	}
}

// ListConnections 查询项目下的连接列表。
func (s *ConnectionService) ListConnections(ctx context.Context, projectID, tenantID string) ([]Connection, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}

	records, err := s.repository.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	connections := make([]Connection, 0, len(records))
	statuses, err := s.connectionSecretStatuses(ctx, records)
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		connection := toConnection(record, tenantID)
		connection.SecretStatus = statuses[record.ID]
		connections = append(connections, connection)
	}

	return connections, nil
}

// ListConnectionsPage 查询项目下的分页连接列表。
func (s *ConnectionService) ListConnectionsPage(ctx context.Context, projectID, tenantID string, filter repository.ConnectionListFilter) ([]Connection, int, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, 0, err
	}
	if filter.TypeGroup != "" && filter.TypeGroup != "all" {
		validTypeGroups := map[string]struct{}{"builtin": {}, "database": {}, "stream": {}}
		if _, ok := validTypeGroups[filter.TypeGroup]; !ok {
			return nil, 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "typeGroup 参数格式无效")
		}
	}

	records, total, err := s.repository.ListByProjectPage(ctx, projectID, filter)
	if err != nil {
		return nil, 0, err
	}
	connections := make([]Connection, 0, len(records))
	statuses, err := s.connectionSecretStatuses(ctx, records)
	if err != nil {
		return nil, 0, err
	}
	for _, record := range records {
		connection := toConnection(record, tenantID)
		connection.SecretStatus = statuses[record.ID]
		connections = append(connections, connection)
	}
	return connections, total, nil
}

func (s *ConnectionService) GetConnection(ctx context.Context, projectID, connectionID, tenantID string) (*Connection, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}
	record, err := s.repository.GetByProjectAndID(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	connection := toConnection(*record, tenantID)
	if s.secrets != nil {
		connection.SecretStatus, err = s.secrets.Status(ctx, record.ID)
		if err != nil {
			return nil, err
		}
	}
	return &connection, nil
}

func (s *ConnectionService) connectionSecretStatuses(ctx context.Context, records []repository.ConnectionRecord) (map[string]map[string]bool, error) {
	if s == nil || s.secrets == nil {
		return map[string]map[string]bool{}, nil
	}
	ids := make([]string, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.ID)
	}
	return s.secrets.StatusByConnections(ctx, ids)
}

// UpdateConnectionOrder 持久化接入源卡片展示顺序。
func (s *ConnectionService) UpdateConnectionOrder(ctx context.Context, projectID string, connectionIDs []string) error {
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	if len(connectionIDs) == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接排序不能为空")
	}

	seen := make(map[string]struct{}, len(connectionIDs))
	for _, connectionID := range connectionIDs {
		if err := validateConnectionID(connectionID); err != nil {
			return err
		}
		if _, exists := seen[connectionID]; exists {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接排序包含重复项")
		}
		seen[connectionID] = struct{}{}
	}

	current, err := s.repository.ListByProject(ctx, projectID)
	if err != nil {
		return err
	}
	if len(current) != len(connectionIDs) {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接排序必须包含当前项目全部接入源")
	}
	for _, connection := range current {
		if _, exists := seen[connection.ID]; !exists {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接排序缺少当前项目接入源")
		}
	}

	return s.repository.UpdateDisplayOrder(ctx, projectID, connectionIDs)
}

// CreateConnection 创建项目连接。
func (s *ConnectionService) CreateConnection(ctx context.Context, projectID, tenantID, userID string, input CreateConnectionInput) (*Connection, error) {
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
	connectionType, category, err := normalizeConnectionType(input.Type)
	if err != nil {
		return nil, err
	}
	if isBuiltinStoreType(connectionType) {
		normalized, err := normalizeBuiltinStoreCreateInput(projectID, CreateConnectionInput{
			Name:   input.Name,
			Type:   connectionType,
			Status: input.Status,
			Config: input.Config,
		})
		if err != nil {
			return nil, err
		}
		record, err := s.repository.Create(ctx, repository.CreateConnectionParams{
			ProjectID: projectID,
			UserID:    userID,
			Name:      normalized.Name,
			Type:      normalized.Type,
			Category:  normalized.Category,
			Status:    normalized.Status,
			Config:    normalized.Config,
		})
		if err != nil {
			return nil, err
		}

		connection := toConnection(*record, tenantID)
		return &connection, nil
	}
	status, err := normalizeConnectionStatus(input.Status)
	if err != nil {
		return nil, err
	}
	config, err := normalizeConnectionConfig(input.Config)
	if err != nil {
		return nil, err
	}
	secrets, clearSecretKeys, config := extractGenericConnectionSecrets(config)

	record, err := s.repository.Create(ctx, repository.CreateConnectionParams{
		ProjectID:       projectID,
		UserID:          userID,
		Name:            name,
		Type:            connectionType,
		Category:        category,
		Status:          status,
		Config:          config,
		Secrets:         secrets,
		ClearSecretKeys: clearSecretKeys,
	})
	if err != nil {
		return nil, err
	}

	connection := toConnection(*record, tenantID)
	return &connection, nil
}

// UpdateConnection 更新已有连接。
func (s *ConnectionService) UpdateConnection(ctx context.Context, projectID, connectionID, tenantID, userID string, input UpdateConnectionInput) (*Connection, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	if input.Name == nil && input.Type == nil && input.Status == nil && !input.HasConfig {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "至少需要提供一个待更新字段")
	}

	current, err := s.repository.GetByProjectAndID(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	if isStoredProtocolConnectionType(current.Type) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "该接入源必须使用对应协议的专用更新接口")
	}

	nextName := current.Name
	if input.Name != nil {
		nextName, err = normalizeConnectionName(*input.Name)
		if err != nil {
			return nil, err
		}
	}

	nextType := current.Type
	nextCategory := current.Category
	if nextCategory == "" {
		nextCategory = deriveStoredConnectionCategory(current.Type)
	}
	if input.Type != nil {
		requestedType := strings.TrimSpace(strings.ToLower(*input.Type))
		if isStoredProtocolConnectionType(current.Type) && requestedType == current.Type {
			nextType = current.Type
			nextCategory = deriveStoredConnectionCategory(current.Type)
		} else {
			nextType, nextCategory, err = normalizeConnectionType(*input.Type)
			if err != nil {
				return nil, err
			}
		}
		if isBuiltinStoreType(current.Type) && nextType != current.Type {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "内置运行库类型不允许修改")
		}
		if !isBuiltinStoreType(current.Type) && isBuiltinStoreType(nextType) {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "外部接入源不允许改为内置运行库")
		}
	}

	nextStatus := current.Status
	if input.Status != nil {
		nextStatus, err = normalizeConnectionStatus(*input.Status)
		if err != nil {
			return nil, err
		}
	}

	nextConfig := current.Config
	if input.HasConfig {
		nextConfig, err = normalizeConnectionConfig(input.Config)
		if err != nil {
			return nil, err
		}
	}
	if isBuiltinStoreType(nextType) {
		nextCategory = "builtin"
		nextStatus = "connected"
		if input.HasConfig {
			nextConfig = mergeBuiltinConfigUpdate(current.Config, input.Config)
		}
	}
	secrets, clearSecretKeys, nextConfig := extractGenericConnectionSecrets(nextConfig)

	record, err := s.repository.Update(ctx, repository.UpdateConnectionParams{
		ID:              connectionID,
		ProjectID:       projectID,
		UserID:          userID,
		Name:            nextName,
		Type:            nextType,
		Category:        nextCategory,
		Status:          nextStatus,
		Config:          nextConfig,
		Secrets:         secrets,
		ClearSecretKeys: clearSecretKeys,
	})
	if err != nil {
		return nil, err
	}

	connection := toConnection(*record, tenantID)
	return &connection, nil
}

// DeleteConnection 删除项目连接。
func (s *ConnectionService) DeleteConnection(ctx context.Context, projectID, connectionID string) error {
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return err
	}

	return s.repository.Delete(ctx, projectID, connectionID)
}

// TestConnection 使用临时连接配置测试外部数据源连通性。
func (s *ConnectionService) TestConnection(ctx context.Context, projectID string, input CreateConnectionInput) (*ConnectionTestResult, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}

	connectionType, err := normalizeConnectionTestType(input.Type)
	if err != nil {
		return nil, err
	}
	switch connectionType {
	case "relational":
		return testRelationalConnection(ctx, input.Config)
	case "kafka":
		return testKafkaConnection(ctx, input.Config)
	case "http":
		return testHTTPConnection(ctx, input.Config)
	case "websocket":
		return testWebSocketConnection(ctx, input.Config)
	case "redis":
		return testRedisConnection(ctx, input.Config)
	case "tdengine":
		return testTDengineConnection(ctx, input.Config)
	default:
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "当前接入源类型暂不支持连接测试")
	}
}

func normalizeConnectionTestType(connectionType string) (string, error) {
	connectionType = strings.TrimSpace(strings.ToLower(connectionType))
	switch connectionType {
	case "relational", "kafka", "http", "websocket", "redis", "tdengine":
		return connectionType, nil
	default:
		if displayName, ok := reservedPhase2ConnectionTypes[connectionType]; ok {
			return "", newPhaseBoundaryProtocolError(displayName)
		}
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接类型不受支持")
	}
}

func testTDengineConnection(ctx context.Context, config map[string]any) (*ConnectionTestResult, error) {
	runtime, err := connectTDengineRuntime(ctx, config)
	if err != nil {
		return nil, err
	}
	defer runtime.Close()
	return &ConnectionTestResult{Connected: true, Type: "tdengine", DBType: "tdengine", Message: "TDengine WebSocket 连接成功"}, nil
}

func testRelationalConnection(ctx context.Context, config map[string]any) (*ConnectionTestResult, error) {
	runtime, err := connectRelationalRuntime(ctx, config)
	if err != nil {
		return nil, err
	}
	defer runtime.Close()

	if err := runtime.Ping(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接测试失败", err)
	}

	runtimeConfig, parseErr := parseRelationalRuntimeConfig(config)
	if parseErr != nil {
		return nil, parseErr
	}

	return &ConnectionTestResult{
		Connected: true,
		DBType:    runtimeConfig.DBType,
		Type:      "relational",
		Message:   "数据库连接成功",
	}, nil
}

// testKafkaConnection 使用 Kafka 协议短连接验证 Broker、TLS 与 SASL 基础配置。
// 输入为接入源配置，输出为统一测试结果；不读取 Topic 消息，也不依赖消费组。
// 异常会转成业务失败，便于前端直接展示给配置人员。
func testKafkaConnection(ctx context.Context, config map[string]any) (*ConnectionTestResult, error) {
	startedAt := time.Now()
	normalized, err := normalizeKafkaConnectionConfig(config)
	if err != nil {
		return nil, err
	}
	brokers := splitCSV(strings.TrimSpace(toString(normalized["brokers"])))
	if len(brokers) == 0 || strings.TrimSpace(brokers[0]) == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Kafka 服务器地址不能为空")
	}
	options := mapFromAny(normalized["options"])
	dialer, err := buildKafkaDialer(options)
	if err != nil {
		return nil, err
	}
	requestTimeout := time.Duration(intFromAny(options["requestTimeoutMs"], 5000)) * time.Millisecond
	if requestTimeout <= 0 {
		requestTimeout = 5 * time.Second
	}
	testCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	conn, err := dialer.DialContext(testCtx, "tcp", brokers[0])
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Kafka Broker 连接失败", err)
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions()
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Kafka 元信息读取失败", err)
	}
	topicSet := make(map[string]struct{})
	for _, partition := range partitions {
		topicSet[partition.Topic] = struct{}{}
	}

	return &ConnectionTestResult{
		Connected: true,
		Type:      "kafka",
		Detail:    fmt.Sprintf("Broker 可达，Topic 数 %d，耗时 %dms", len(topicSet), time.Since(startedAt).Milliseconds()),
		Message:   "Kafka 接入源连接成功",
	}, nil
}

// testHTTPConnection 不再执行接入源级连接测试。
// HTTP 没有稳定连接概念，真实 URL、方法、Header 和 Body 均由工作台请求项决定。
func testHTTPConnection(ctx context.Context, config map[string]any) (*ConnectionTestResult, error) {
	return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "HTTP 接入源无需测试连接，请在 HTTP 工作台请求项中发送测试")
}

// testWebSocketConnection 只验证握手链路，不长期读取消息，避免测试连接变成运行态订阅。
// 输入为 WebSocket 配置，输出为握手结果；握手失败时保留下游状态码辅助定位。
func testWebSocketConnection(ctx context.Context, config map[string]any) (*ConnectionTestResult, error) {
	startedAt := time.Now()
	rawURL := strings.TrimSpace(toString(config["url"]))
	if rawURL == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "WebSocket 连接地址不能为空")
	}
	headers := http.Header{}
	for key, value := range mapFromAny(config["headers"]) {
		if strings.TrimSpace(key) == "" {
			continue
		}
		headers.Set(key, toString(value))
	}
	timeout := time.Duration(intFromAny(config["timeoutMs"], 5000)) * time.Millisecond
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	dialer := websocket.Dialer{
		HandshakeTimeout: timeout,
		TLSClientConfig:  &tls.Config{MinVersion: tls.VersionTLS12},
	}
	conn, resp, err := dialer.DialContext(ctx, rawURL, headers)
	if err != nil {
		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
		}
		message := "WebSocket 握手失败"
		if statusCode > 0 {
			message = fmt.Sprintf("%s，状态码 %d", message, statusCode)
		}
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, message, err)
	}
	defer conn.Close()

	return &ConnectionTestResult{
		Connected: true,
		Type:      "websocket",
		Detail:    fmt.Sprintf("握手完成，耗时 %dms", time.Since(startedAt).Milliseconds()),
		Message:   "WebSocket 接入源握手成功",
	}, nil
}

// testRedisConnection 复用短时预览的 Redis client 构造逻辑，执行 PING 验证认证与网络。
// 输入为 Redis 配置，输出为统一连接测试结果；PING 失败按业务失败返回给前端。
func testRedisConnection(ctx context.Context, config map[string]any) (*ConnectionTestResult, error) {
	startedAt := time.Now()
	client, err := newRedisPreviewClient(config, mapFromAny(config["options"]))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Redis 连接测试失败", err)
	}

	return &ConnectionTestResult{
		Connected: true,
		Type:      "redis",
		Detail:    fmt.Sprintf("PING 成功，耗时 %dms", time.Since(startedAt).Milliseconds()),
		Message:   "Redis 接入源连接成功",
	}, nil
}

func connectionTestTimeout(config map[string]any) time.Duration {
	timeout := time.Duration(intFromAny(config["timeoutMs"], 5000)) * time.Millisecond
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return timeout
}

func dialTCP(ctx context.Context, address string, timeout time.Duration) error {
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(dialCtx, "tcp", address)
	if err != nil {
		return err
	}
	return conn.Close()
}

func ensureHostPort(address string, defaultPort int) (string, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "网络地址不能为空")
	}
	if _, _, err := net.SplitHostPort(address); err == nil {
		return address, nil
	}
	if strings.Count(address, ":") > 1 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "网络地址格式无效")
	}
	return net.JoinHostPort(address, fmt.Sprintf("%d", defaultPort)), nil
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func normalizeKafkaConnectionConfig(config map[string]any) (map[string]any, error) {
	normalized := cloneMap(config)
	brokers := strings.TrimSpace(toString(normalized["brokers"]))
	if brokers == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Kafka 服务器地址不能为空")
	}
	options := cloneMap(mapFromAny(normalized["options"]))
	securityProtocol := strings.ToUpper(strings.TrimSpace(toString(options["securityProtocol"])))
	if securityProtocol == "" {
		securityProtocol = "PLAINTEXT"
	}
	if _, ok := allowedKafkaSecurityProtocols[securityProtocol]; !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Kafka securityProtocol 仅支持 PLAINTEXT/SSL/SASL_PLAINTEXT/SASL_SSL")
	}
	options["securityProtocol"] = securityProtocol
	if strings.Contains(securityProtocol, "SASL") {
		mechanism := strings.ToUpper(strings.TrimSpace(toString(options["saslMechanism"])))
		if mechanism == "" {
			mechanism = "PLAIN"
		}
		if _, ok := allowedKafkaSaslMechanisms[mechanism]; !ok {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Kafka saslMechanism 仅支持 PLAIN/SCRAM-SHA-256/SCRAM-SHA-512")
		}
		if strings.TrimSpace(toString(options["username"])) == "" {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Kafka SASL 用户名不能为空")
		}
		if strings.TrimSpace(toString(options["password"])) == "" {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Kafka SASL 密码不能为空")
		}
		options["saslMechanism"] = mechanism
	} else {
		delete(options, "saslMechanism")
		delete(options, "username")
		delete(options, "password")
	}
	normalized["brokers"] = brokers
	normalized["options"] = options
	return normalized, nil
}

func buildKafkaDialer(options map[string]any) (*kafka.Dialer, error) {
	dialTimeout := time.Duration(intFromAny(options["dialTimeoutMs"], 5000)) * time.Millisecond
	if dialTimeout <= 0 {
		dialTimeout = 5 * time.Second
	}
	dialer := &kafka.Dialer{
		Timeout:  dialTimeout,
		ClientID: strings.TrimSpace(toString(options["clientId"])),
	}
	securityProtocol := strings.ToUpper(strings.TrimSpace(toString(options["securityProtocol"])))
	if strings.Contains(securityProtocol, "SSL") {
		tlsConfig, err := kafkaTLSConfig(mapFromAny(options["sslConfig"]))
		if err != nil {
			return nil, err
		}
		dialer.TLS = tlsConfig
	}
	if strings.Contains(securityProtocol, "SASL") {
		mechanism, err := kafkaSaslMechanism(options)
		if err != nil {
			return nil, err
		}
		dialer.SASLMechanism = mechanism
	}
	return dialer, nil
}

func kafkaTLSConfig(sslConfig map[string]any) (*tls.Config, error) {
	config := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if rejectUnauthorized, ok := sslConfig["rejectUnauthorized"].(bool); ok && !rejectUnauthorized {
		// 开发态连接测试允许跳过证书校验，便于调试自签名 Kafka Broker。
		config.InsecureSkipVerify = true //nolint:gosec
	}
	if ca := strings.TrimSpace(toString(sslConfig["ca"])); ca != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(ca)) {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Kafka CA 证书格式无效")
		}
		config.RootCAs = pool
	}
	cert := strings.TrimSpace(toString(sslConfig["cert"]))
	key := strings.TrimSpace(toString(sslConfig["key"]))
	if cert != "" || key != "" {
		if cert == "" || key == "" {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Kafka 客户端证书和私钥需要同时填写")
		}
		certificate, err := tls.X509KeyPair([]byte(cert), []byte(key))
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Kafka 客户端证书或私钥格式无效", err)
		}
		config.Certificates = []tls.Certificate{certificate}
	}
	return config, nil
}

func kafkaSaslMechanism(options map[string]any) (sasl.Mechanism, error) {
	username := strings.TrimSpace(toString(options["username"]))
	password := toString(options["password"])
	switch strings.ToUpper(strings.TrimSpace(toString(options["saslMechanism"]))) {
	case "", "PLAIN":
		return plain.Mechanism{Username: username, Password: password}, nil
	case "SCRAM-SHA-256":
		return scram.Mechanism(scram.SHA256, username, password)
	case "SCRAM-SHA-512":
		return scram.Mechanism(scram.SHA512, username, password)
	default:
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Kafka saslMechanism 仅支持 PLAIN/SCRAM-SHA-256/SCRAM-SHA-512")
	}
}

// UpdateConnectionStatus 更新连接状态。
func (s *ConnectionService) UpdateConnectionStatus(ctx context.Context, projectID, connectionID, status string) (*Connection, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}
	nextStatus, err := normalizeConnectionStatus(status)
	if err != nil {
		return nil, err
	}

	record, err := s.repository.UpdateStatus(ctx, projectID, connectionID, nextStatus)
	if err != nil {
		return nil, err
	}

	current, err := s.repository.GetByProjectAndID(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	current.Status = record.Status
	current.UpdatedAt = record.UpdatedAt
	connection := toConnection(*current, "")
	return &connection, nil
}

// ListTables 返回外部关系库表列表。
func (s *ConnectionService) ListTables(ctx context.Context, projectID, connectionID string) ([]RelationalTable, error) {
	connection, err := s.loadRelationalConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	if connection.Type == "tdengine" {
		return listTDengineTables(ctx, connection.Config)
	}
	if schemaName, ok, err := builtinSQLSchemaFromRecord(connection); ok || err != nil {
		if err != nil {
			return nil, err
		}
		if s.builtinRuntime == nil {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态内置运行库未初始化")
		}
		return s.builtinRuntime.ListTablesInSchema(ctx, schemaName)
	}

	runtime, err := connectRelationalRuntime(ctx, connection.Config)
	if err != nil {
		return nil, err
	}
	defer runtime.Close()

	schema := runtime.SearchPath()
	query, args := relationalListTablesQuery(runtime.DBType(), schema)
	rows, err := runtime.Query(ctx, query, args...)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询表列表失败", err)
	}
	defer rows.Close()

	tables := make([]RelationalTable, 0)
	for rows.Next() {
		table := RelationalTable{}
		if err := rows.Scan(&table.Schema, &table.Name, &table.Type); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取表列表失败", err)
		}
		table.Kind = relationalTableKind(connection.Type, runtime.DBType(), table.Type, table.Name)
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历表列表失败", err)
	}

	return tables, nil
}

// ListTablesPage 在服务端完成对象搜索和分页，供 TDengine 等对象数量可增长的工作台使用。
func (s *ConnectionService) ListTablesPage(ctx context.Context, projectID, connectionID, search string, page, pageSize int) (*RelationalTablePage, error) {
	connection, err := s.loadRelationalConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	if page < 1 || pageSize < 1 || pageSize > 200 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "表对象分页参数无效")
	}
	var tables []RelationalTable
	if connection.Type == "tdengine" {
		tables, err = listTDengineTables(ctx, connection.Config)
	} else {
		tables, err = s.ListTables(ctx, projectID, connectionID)
	}
	if err != nil {
		return nil, err
	}
	keyword := strings.ToLower(strings.TrimSpace(search))
	filtered := make([]RelationalTable, 0, len(tables))
	for _, table := range tables {
		if keyword == "" || strings.Contains(strings.ToLower(table.Name), keyword) {
			filtered = append(filtered, table)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].Kind == filtered[j].Kind {
			return strings.ToLower(filtered[i].Name) < strings.ToLower(filtered[j].Name)
		}
		return filtered[i].Kind < filtered[j].Kind
	})
	total := len(filtered)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	return &RelationalTablePage{
		Items:      filtered[start:end],
		Pagination: RelationalPagination{Page: page, Limit: pageSize, Total: total, TotalPages: totalPages},
	}, nil
}

// CreateTable 按结构化表设计创建物理表。
// 该接口用于低频建模操作：服务端统一做标识符校验和方言生成，调用方不传入 SQL 文本。
func (s *ConnectionService) CreateTable(ctx context.Context, projectID, connectionID string, input CreateRelationalTableInput) (*RelationalTableStructure, error) {
	connection, err := s.loadRelationalConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	connectionType := connection.Type
	if connectionType == "relational" {
		connectionType = strings.TrimSpace(strings.ToLower(toString(connection.Config["dbType"])))
	}
	design, err := normalizeCreateTableInput(connectionType, input)
	if err != nil {
		return nil, err
	}
	if connection.Type == "tdengine" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "TDengine 建表需要启用 TDengine 运行时驱动后才能执行")
	}
	if schemaName, ok, err := builtinSQLSchemaFromRecord(connection); ok || err != nil {
		if err != nil {
			return nil, err
		}
		if s.builtinRuntime == nil {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态内置运行库未初始化")
		}
		if err := s.builtinRuntime.CreateTableInSchema(ctx, schemaName, design); err != nil {
			return nil, err
		}
		return s.builtinRuntime.GetTableStructureInSchema(ctx, schemaName, design.Name)
	}

	runtime, err := connectRelationalRuntime(ctx, connection.Config)
	if err != nil {
		return nil, err
	}
	defer runtime.Close()
	ddl, err := buildCreateTableDDL(runtime.DBType(), runtime.SearchPath(), design)
	if err != nil {
		return nil, err
	}
	if err := runtime.Exec(ctx, ddl); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "创建表失败", err)
	}
	return s.GetTableStructure(ctx, projectID, connectionID, design.Name)
}

func (s *ConnectionService) RenameTable(ctx context.Context, projectID, connectionID, oldName, newName, userID string) error {
	connection, err := s.loadRelationalConnection(ctx, projectID, connectionID)
	if err != nil {
		return err
	}
	oldName, err = normalizeRuntimeIdentifier(oldName, "tableName")
	if err != nil {
		return err
	}
	newName, err = normalizeRuntimeIdentifier(newName, "newTableName")
	if err != nil {
		return err
	}
	if oldName == newName {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "新表名不能与原表名相同")
	}
	if connection.Type == "tdengine" {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "TDengine 表重命名需要启用 TDengine 运行时驱动后才能执行")
	}
	if schemaName, ok, err := builtinSQLSchemaFromRecord(connection); ok || err != nil {
		if err != nil {
			return err
		}
		if s.builtinRuntime == nil {
			return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态内置运行库未初始化")
		}
		if err := s.builtinRuntime.RenameTableInSchema(ctx, schemaName, oldName, newName); err != nil {
			return err
		}
		if s.workbenchGroups != nil {
			return s.workbenchGroups.RenameTableMember(ctx, projectID, connectionID, oldName, newName, userID)
		}
		return nil
	}
	runtime, err := connectRelationalRuntime(ctx, connection.Config)
	if err != nil {
		return err
	}
	defer runtime.Close()
	if err := runtime.Exec(ctx, relationalRenameTableDDL(runtime.DBType(), runtime.SearchPath(), oldName, newName)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "重命名表失败", err)
	}
	if s.workbenchGroups != nil {
		return s.workbenchGroups.RenameTableMember(ctx, projectID, connectionID, oldName, newName, userID)
	}
	return nil
}

func (s *ConnectionService) DeleteTable(ctx context.Context, projectID, connectionID, tableName string) error {
	connection, err := s.loadRelationalConnection(ctx, projectID, connectionID)
	if err != nil {
		return err
	}
	tableName, err = normalizeRuntimeIdentifier(tableName, "tableName")
	if err != nil {
		return err
	}
	if connection.Type == "tdengine" {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "TDengine 删除表需要启用 TDengine 运行时驱动后才能执行")
	}
	if schemaName, ok, err := builtinSQLSchemaFromRecord(connection); ok || err != nil {
		if err != nil {
			return err
		}
		if s.builtinRuntime == nil {
			return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态内置运行库未初始化")
		}
		if err := s.builtinRuntime.DeleteTableInSchema(ctx, schemaName, tableName); err != nil {
			return err
		}
		if s.workbenchGroups != nil {
			return s.workbenchGroups.DeleteTableMember(ctx, projectID, connectionID, tableName)
		}
		return nil
	}
	runtime, err := connectRelationalRuntime(ctx, connection.Config)
	if err != nil {
		return err
	}
	defer runtime.Close()
	if err := runtime.Exec(ctx, relationalDropTableDDL(runtime.DBType(), runtime.SearchPath(), tableName)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "删除表失败", err)
	}
	if s.workbenchGroups != nil {
		return s.workbenchGroups.DeleteTableMember(ctx, projectID, connectionID, tableName)
	}
	return nil
}

// GetTableStructure 返回指定表结构。
func (s *ConnectionService) GetTableStructure(ctx context.Context, projectID, connectionID, tableName string) (*RelationalTableStructure, error) {
	connection, err := s.loadRelationalConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	normalizedTableName, err := normalizeRuntimeIdentifier(tableName, "tableName")
	if err != nil {
		return nil, err
	}
	if connection.Type == "tdengine" {
		return getTDengineTableStructure(ctx, connection.Config, normalizedTableName)
	}
	if schemaName, ok, err := builtinSQLSchemaFromRecord(connection); ok || err != nil {
		if err != nil {
			return nil, err
		}
		if s.builtinRuntime == nil {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态内置运行库未初始化")
		}
		return s.builtinRuntime.GetTableStructureInSchema(ctx, schemaName, normalizedTableName)
	}

	runtime, err := connectRelationalRuntime(ctx, connection.Config)
	if err != nil {
		return nil, err
	}
	defer runtime.Close()

	schema := runtime.SearchPath()

	columns, err := s.listTableColumns(ctx, runtime, schema, normalizedTableName)
	if err != nil {
		return nil, err
	}
	indexes, err := s.listTableIndexes(ctx, runtime, schema, normalizedTableName)
	if err != nil {
		return nil, err
	}
	foreignKeys, err := s.listTableForeignKeys(ctx, runtime, schema, normalizedTableName)
	if err != nil {
		return nil, err
	}

	return &RelationalTableStructure{
		Columns:     columns,
		Indexes:     indexes,
		ForeignKeys: foreignKeys,
	}, nil
}

// UpdateTableStructure 仅允许修改 IF 内置关系/时序库中的用户表结构。
// 输入是目标结构，服务端按当前结构计算低风险 DDL，拒绝字段改名、改类型、主键和自增变更。
func (s *ConnectionService) UpdateTableStructure(ctx context.Context, projectID, connectionID, tableName string, input UpdateRelationalTableInput) (*RelationalTableStructure, error) {
	connection, err := s.loadRelationalConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	if connection.Type != "builtin.relation" && connection.Type != "builtin.timeseries" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "仅内置 IF 关系库和 IF 时序库支持修改表结构")
	}
	normalizedTableName, err := normalizeRuntimeIdentifier(tableName, "tableName")
	if err != nil {
		return nil, err
	}
	design, err := normalizeUpdateTableInput(input)
	if err != nil {
		return nil, err
	}
	schemaName, ok, err := builtinSQLSchemaFromRecord(connection)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "仅内置 IF 运行库支持修改表结构")
	}
	if s.builtinRuntime == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态内置运行库未初始化")
	}
	return s.builtinRuntime.UpdateTableStructureInSchema(ctx, schemaName, normalizedTableName, design)
}

// GetTableData 返回指定表数据预览。
func (s *ConnectionService) GetTableData(ctx context.Context, projectID, connectionID, tableName string, page, limit int) (*RelationalTableData, error) {
	connection, err := s.loadRelationalConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	normalizedTableName, err := normalizeRuntimeIdentifier(tableName, "tableName")
	if err != nil {
		return nil, err
	}

	page, limit = normalizePageAndSize(page, limit, 100, 500)
	if connection.Type == "tdengine" {
		return getTDengineTableData(ctx, connection.Config, normalizedTableName, page, limit)
	}
	if schemaName, ok, err := builtinSQLSchemaFromRecord(connection); ok || err != nil {
		if err != nil {
			return nil, err
		}
		query := fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d", pgx.Identifier{normalizedTableName}.Sanitize(), limit, (page-1)*limit)
		result, err := s.builtinRuntime.ExecuteSQLInSchema(ctx, schemaName, query, nil, limit)
		if err != nil {
			return nil, err
		}
		rows := make([][]any, 0, len(result.Rows))
		for _, row := range result.Rows {
			values := make([]any, 0, len(result.Columns))
			for _, column := range result.Columns {
				values = append(values, row[column])
			}
			rows = append(rows, values)
		}
		return &RelationalTableData{
			Columns: result.Columns,
			Rows:    rows,
			Pagination: RelationalPagination{
				Page:       page,
				Limit:      limit,
				Total:      result.RowCount,
				TotalPages: 1,
			},
		}, nil
	}

	runtime, err := connectRelationalRuntime(ctx, connection.Config)
	if err != nil {
		return nil, err
	}
	defer runtime.Close()

	schema := runtime.SearchPath()
	qualifiedTableName := relationalQualifiedTableName(runtime.DBType(), schema, normalizedTableName)

	var total int
	if err := runtime.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", qualifiedTableName)).Scan(&total); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计表数据失败", err)
	}

	rows, err := runtime.Query(ctx, relationalTableDataQuery(runtime.DBType(), qualifiedTableName, page, limit))
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询表数据失败", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取表字段元信息失败", err)
	}

	resultRows := make([][]any, 0)
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取表数据失败", err)
		}
		row := make([]any, 0, len(values))
		for _, value := range values {
			row = append(row, normalizeQueryValue(value))
		}
		resultRows = append(resultRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历表数据失败", err)
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}

	return &RelationalTableData{
		Columns: columns,
		Rows:    resultRows,
		Pagination: RelationalPagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// ExecuteSQL 执行 SQL 工作台语句。
// 工作台不做表级 DDL/DML 限制，实际权限由数据库账号和内置库 schema 隔离决定。
func (s *ConnectionService) ExecuteSQL(ctx context.Context, projectID, connectionID, sqlText string, parameters []any) (*RelationalQueryResult, error) {
	connection, err := s.loadRelationalConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	sqlText = strings.TrimSpace(sqlText)
	if sqlText == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "SQL 不能为空")
	}
	if err := ensureSQLWorkbenchDatabaseBoundary(sqlText); err != nil {
		return nil, err
	}
	if connection.Type == "tdengine" {
		if err := validateTDengineReadOnlySQL(sqlText); err != nil {
			return nil, err
		}
		runtime, err := connectTDengineRuntime(ctx, connection.Config)
		if err != nil {
			return nil, err
		}
		defer runtime.Close()
		startedAt := time.Now()
		columns, rows, err := runtime.query(ctx, sqlText, parameters...)
		if err != nil {
			return nil, err
		}
		return &RelationalQueryResult{Columns: columns, Rows: rows, RowCount: len(rows), ExecutionTime: time.Since(startedAt).Milliseconds()}, nil
	}
	if schemaName, ok, err := builtinSQLSchemaFromRecord(connection); ok || err != nil {
		if err != nil {
			return nil, err
		}
		if s.builtinRuntime == nil {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态内置运行库未初始化")
		}
		result, err := s.builtinRuntime.ExecuteSQLInSchema(ctx, schemaName, sqlText, parameters, 500)
		if err != nil {
			return nil, err
		}
		return builtinSQLResultToRelational(result), nil
	}

	runtime, err := connectRelationalRuntime(ctx, connection.Config)
	if err != nil {
		return nil, err
	}
	defer runtime.Close()

	startTime := time.Now()
	if isSQLWorkbenchQuery(sqlText) {
		return executeRelationalQuerySQL(ctx, runtime, sqlText, parameters, startTime)
	}
	affected, err := runtime.ExecAffected(ctx, sqlText, parameters...)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "执行 SQL 失败", err)
	}
	return &RelationalQueryResult{
		Columns:       []string{},
		Rows:          [][]any{},
		RowCount:      int(affected),
		ExecutionTime: time.Since(startTime).Milliseconds(),
	}, nil
}

func executeRelationalQuerySQL(ctx context.Context, runtime *relationalRuntime, sqlText string, parameters []any, startTime time.Time) (*RelationalQueryResult, error) {
	rows, err := runtime.Query(ctx, sqlText, parameters...)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "执行 SQL 失败", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 SQL 字段元信息失败", err)
	}

	resultRows := make([][]any, 0)
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 SQL 结果失败", err)
		}
		row := make([]any, 0, len(values))
		for _, value := range values {
			row = append(row, normalizeQueryValue(value))
		}
		resultRows = append(resultRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 SQL 结果失败", err)
	}

	return &RelationalQueryResult{
		Columns:       columns,
		Rows:          resultRows,
		RowCount:      len(resultRows),
		ExecutionTime: time.Since(startTime).Milliseconds(),
	}, nil
}

func isSQLWorkbenchQuery(sqlText string) bool {
	normalized := strings.TrimSpace(strings.ToLower(sqlText))
	return strings.HasPrefix(normalized, "select") || strings.HasPrefix(normalized, "with") || strings.HasPrefix(normalized, "show") || strings.HasPrefix(normalized, "describe") || strings.HasPrefix(normalized, "desc")
}

func ensureSQLWorkbenchDatabaseBoundary(sqlText string) error {
	normalized := strings.ToLower(strings.Join(strings.Fields(sqlText), " "))
	blockedPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\b(create|drop|alter)\s+database\b`),
		regexp.MustCompile(`\b(create|drop|alter)\s+schema\b`),
		regexp.MustCompile(`\buse\s+[a-zA-Z0-9_"'\[\]` + "`" + `.-]+`),
	}
	for _, pattern := range blockedPatterns {
		if pattern.MatchString(normalized) {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "SQL 编辑器不允许执行数据库级创建、删除或切换操作")
		}
	}
	return nil
}

func (s *ConnectionService) loadRelationalConnection(ctx context.Context, projectID, connectionID string) (*repository.ConnectionRecord, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}

	connection, err := s.repository.GetByProjectAndID(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	if connection.Type != "relational" {
		if connection.Type != "builtin.relation" && connection.Type != "builtin.timeseries" && connection.Type != "tdengine" {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "当前仅支持关系型连接")
		}
	}
	if s.secrets != nil && (connection.Type == "relational" || connection.Type == "tdengine") {
		secrets, err := s.secrets.ResolveAll(ctx, connection.ID)
		if err != nil {
			return nil, err
		}
		connection.Config = injectConnectionSecrets(connection.Config, secrets)
	}
	return connection, nil
}

func injectConnectionSecrets(config map[string]any, secrets map[string]string) map[string]any {
	result := cloneMap(config)
	for key, value := range secrets {
		switch {
		case key == "password":
			result["password"] = value
		case strings.HasPrefix(key, "option."):
			options := mapFromAny(result["options"])
			options[strings.TrimPrefix(key, "option.")] = value
			result["options"] = options
		case strings.HasPrefix(key, "tls."):
			sslConfig := mapFromAny(result["sslConfig"])
			sslConfig[strings.TrimPrefix(key, "tls.")] = value
			result["sslConfig"] = sslConfig
		case strings.HasPrefix(key, "header."):
			headers := mapFromAny(result["headers"])
			headers[strings.TrimPrefix(key, "header.")] = value
			result["headers"] = headers
		}
	}
	return result
}

// extractGenericConnectionSecrets 将关系库等通用连接表单中的敏感字段移出 metadata。
// 空字符串表示用户明确清除；未出现字段则保持已有密钥不变。
func extractGenericConnectionSecrets(config map[string]any) (map[string]string, []string, map[string]any) {
	result := cloneMap(config)
	secrets := map[string]string{}
	clearKeys := make([]string, 0)
	if value, exists := result["password"]; exists {
		password := toString(value)
		if password == "" {
			clearKeys = append(clearKeys, "password")
		} else {
			secrets["password"] = password
		}
		delete(result, "password")
	}
	sslConfig := mapFromAny(result["sslConfig"])
	for _, key := range []string{"ca", "cert", "key"} {
		value, exists := sslConfig[key]
		if !exists {
			continue
		}
		secretKey := "tls." + key
		textValue := toString(value)
		if textValue == "" {
			clearKeys = append(clearKeys, secretKey)
		} else {
			secrets[secretKey] = textValue
		}
		delete(sslConfig, key)
	}
	if len(sslConfig) > 0 {
		result["sslConfig"] = sslConfig
	} else {
		delete(result, "sslConfig")
	}
	return secrets, clearKeys, result
}

func builtinSQLSchemaFromRecord(connection *repository.ConnectionRecord) (string, bool, error) {
	if connection == nil {
		return "", false, nil
	}
	if connection.Type != "builtin.relation" && connection.Type != "builtin.timeseries" {
		return "", false, nil
	}
	schemaName := builtinConfigString(connection.Config, "devSchema", "")
	if schemaName == "" {
		runtimeKey := builtinConfigString(connection.Config, "runtimeKey", "")
		if runtimeKey == "" {
			return "", true, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "内置运行库缺少系统标识")
		}
		schemaName = deriveBuiltinConnectionSchema(connection.ProjectID, runtimeKey)
	}
	return schemaName, true, nil
}

func builtinSQLResultToRelational(result *BuiltinSQLExecuteResult) *RelationalQueryResult {
	if result == nil {
		return &RelationalQueryResult{Columns: []string{}, Rows: [][]any{}}
	}
	rows := make([][]any, 0, len(result.Rows))
	for _, row := range result.Rows {
		values := make([]any, 0, len(result.Columns))
		for _, column := range result.Columns {
			values = append(values, row[column])
		}
		rows = append(rows, values)
	}
	return &RelationalQueryResult{
		Columns:       append([]string{}, result.Columns...),
		Rows:          rows,
		RowCount:      result.RowCount,
		ExecutionTime: result.ExecutionTime,
	}
}

func normalizeRuntimeIdentifier(value, fieldName string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fieldName+" 不能为空")
	}
	if strings.ContainsAny(value, "\"'; \t\r\n") {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fieldName+" 包含非法字符")
	}
	return value, nil
}

func (s *ConnectionService) listTableColumns(ctx context.Context, runtime *relationalRuntime, schema, tableName string) ([]RelationalTableColumn, error) {
	query, args := relationalTableColumnsQuery(runtime.DBType(), schema, tableName)
	rows, err := runtime.Query(ctx, query, args...)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询表字段失败", err)
	}
	defer rows.Close()

	columns := make([]RelationalTableColumn, 0)
	for rows.Next() {
		column := RelationalTableColumn{}
		var (
			maxLength sql.NullInt64
			precision sql.NullInt64
			scale     sql.NullInt64
			nullable  bool
			comment   sql.NullString
			defValue  sql.NullString
			isPrimary bool
			isUnique  bool
			autoIncr  bool
		)
		if err := rows.Scan(
			&column.Name,
			&column.Type,
			&maxLength,
			&precision,
			&scale,
			&nullable,
			&defValue,
			&comment,
			&isPrimary,
			&isUnique,
			&autoIncr,
		); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取表字段失败", err)
		}
		column.MaxLength = nullInt64ToInt(maxLength)
		column.NumericPrecision = nullInt64ToInt(precision)
		column.NumericScale = nullInt64ToInt(scale)
		column.Nullable = nullable
		column.DefaultValue = nullStringPointer(defValue)
		column.Comment = nullStringPointer(comment)
		column.IsPrimary = isPrimary
		column.IsUnique = isUnique
		column.AutoIncrement = autoIncr
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历表字段失败", err)
	}

	return columns, nil
}

func (s *ConnectionService) listTableIndexes(ctx context.Context, runtime *relationalRuntime, schema, tableName string) ([]RelationalIndex, error) {
	if runtime.DBType() == "postgresql" {
		return s.listPostgresTableIndexes(ctx, runtime, schema, tableName)
	}

	query, args := relationalTableIndexesQuery(runtime.DBType(), schema, tableName)
	rows, err := runtime.Query(ctx, query, args...)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询表索引失败", err)
	}
	defer rows.Close()

	indexMap := make(map[string]*RelationalIndex)
	for rows.Next() {
		var name, indexType, method, columnName string
		if err := rows.Scan(&name, &indexType, &method, &columnName); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取表索引失败", err)
		}

		current, ok := indexMap[name]
		if !ok {
			current = &RelationalIndex{
				Name:    name,
				Type:    indexType,
				Method:  strings.ToLower(method),
				Columns: []string{},
			}
			indexMap[name] = current
		}
		if strings.TrimSpace(columnName) != "" {
			current.Columns = append(current.Columns, columnName)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历表索引失败", err)
	}

	indexes := make([]RelationalIndex, 0, len(indexMap))
	for _, index := range indexMap {
		indexes = append(indexes, *index)
	}
	return indexes, nil
}

func (s *ConnectionService) listTableForeignKeys(ctx context.Context, runtime *relationalRuntime, schema, tableName string) ([]RelationalForeignKey, error) {
	query, args := relationalTableForeignKeysQuery(runtime.DBType(), schema, tableName)
	rows, err := runtime.Query(ctx, query, args...)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询表外键失败", err)
	}
	defer rows.Close()

	foreignKeys := make([]RelationalForeignKey, 0)
	for rows.Next() {
		foreignKey := RelationalForeignKey{}
		if err := rows.Scan(
			&foreignKey.Name,
			&foreignKey.ColumnName,
			&foreignKey.ReferencedTable,
			&foreignKey.ReferencedColumn,
			&foreignKey.UpdateRule,
			&foreignKey.DeleteRule,
		); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取表外键失败", err)
		}
		foreignKeys = append(foreignKeys, foreignKey)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历表外键失败", err)
	}

	return foreignKeys, nil
}

func nullInt64ToInt(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	result := int(value.Int64)
	return &result
}

func nullStringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	text := value.String
	return &text
}

func toConnection(record repository.ConnectionRecord, tenantID string) Connection {
	relationalConfig := map[string]any(nil)
	mqttConfig := map[string]any(nil)
	switch record.Type {
	case "relational":
		relationalConfig = cloneMap(record.Config)
	case "mqtt":
		mqttConfig = cloneMap(record.Config)
	}
	return Connection{
		ID:               record.ID,
		ProjectID:        record.ProjectID,
		TenantID:         tenantID,
		Name:             record.Name,
		Type:             record.Type,
		Status:           record.Status,
		Config:           cloneMap(record.Config),
		RelationalConfig: relationalConfig,
		MqttConfig:       mqttConfig,
		DisplayOrder:     record.DisplayOrder,
		VariableCount:    record.VariableCount,
		CreatedAt:        record.CreatedAt,
		UpdatedAt:        record.UpdatedAt,
	}
}

func validateProjectID(projectID string) error {
	if _, err := uuid.Parse(strings.TrimSpace(projectID)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "projectId 格式无效", err)
	}
	return nil
}

func validateConnectionID(connectionID string) error {
	if _, err := uuid.Parse(strings.TrimSpace(connectionID)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "connectionId 格式无效", err)
	}
	return nil
}

func validateUserID(userID string) error {
	if _, err := uuid.Parse(strings.TrimSpace(userID)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "用户标识格式无效", err)
	}
	return nil
}

func normalizeConnectionName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接名称不能为空")
	}
	if len([]rune(name)) > 100 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接名称长度不能超过 100 个字符")
	}
	return name, nil
}

func normalizeConnectionType(connectionType string) (string, string, error) {
	connectionType = strings.TrimSpace(strings.ToLower(connectionType))
	if displayName, ok := reservedPhase2ConnectionTypes[connectionType]; ok {
		return "", "", newPhaseBoundaryProtocolError(displayName)
	}
	category, ok := publicConnectionTypeCategoryMap[connectionType]
	if !ok {
		return "", "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接类型不受支持")
	}
	return connectionType, category, nil
}

func deriveStoredConnectionCategory(connectionType string) string {
	switch strings.TrimSpace(strings.ToLower(connectionType)) {
	case "relational":
		return "database"
	case "mqtt":
		return "message"
	case "builtin.relation", "builtin.timeseries", "builtin.realtime", "builtin.message":
		return "builtin"
	case "kafka", "http", "websocket", "redis", "tdengine":
		return "protocol"
	default:
		return ""
	}
}

func isStoredProtocolConnectionType(connectionType string) bool {
	switch strings.TrimSpace(strings.ToLower(connectionType)) {
	case "kafka", "http", "websocket", "redis":
		return true
	default:
		return false
	}
}

func normalizeConnectionStatus(status string) (string, error) {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		return "unknown", nil
	}
	if _, ok := allowedConnectionStatus[status]; !ok {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接状态不受支持")
	}
	return status, nil
}

func normalizeConnectionConfig(config map[string]any) (map[string]any, error) {
	if config == nil {
		return map[string]any{}, nil
	}
	return cloneMap(config), nil
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}

	result := make(map[string]any, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func (s *ConnectionService) listPostgresTableIndexes(ctx context.Context, runtime *relationalRuntime, schema, tableName string) ([]RelationalIndex, error) {
	rows, err := runtime.Query(ctx, `
        SELECT indexname, indexdef
        FROM pg_indexes
        WHERE schemaname = $1
          AND tablename = $2
        ORDER BY indexname
    `, schema, tableName)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询表索引失败", err)
	}
	defer rows.Close()

	indexes := make([]RelationalIndex, 0)
	for rows.Next() {
		var (
			name     string
			indexDef string
		)
		if err := rows.Scan(&name, &indexDef); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取表索引失败", err)
		}

		indexType := "INDEX"
		if strings.Contains(indexDef, "PRIMARY KEY") {
			indexType = "PRIMARY"
		} else if strings.Contains(indexDef, "UNIQUE INDEX") {
			indexType = "UNIQUE"
		}

		method := "btree"
		if methodMatch := regexp.MustCompile(`USING ([a-zA-Z0-9_]+)`).FindStringSubmatch(indexDef); len(methodMatch) == 2 {
			method = strings.ToLower(methodMatch[1])
		}

		columnPart := regexp.MustCompile(`\((.*)\)`).FindStringSubmatch(indexDef)
		columns := []string{}
		if len(columnPart) == 2 {
			for _, item := range strings.Split(columnPart[1], ",") {
				columnName := strings.Trim(strings.TrimSpace(item), "\"")
				if columnName != "" {
					columns = append(columns, columnName)
				}
			}
		}

		indexes = append(indexes, RelationalIndex{
			Name:    name,
			Type:    indexType,
			Method:  method,
			Columns: columns,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历表索引失败", err)
	}

	return indexes, nil
}

func relationalListTablesQuery(dbType, schema string) (string, []any) {
	switch dbType {
	case "mysql", "sqlserver":
		return `
        SELECT TABLE_SCHEMA, TABLE_NAME, TABLE_TYPE
        FROM information_schema.tables
        WHERE TABLE_SCHEMA = ?
          AND TABLE_TYPE IN ('BASE TABLE', 'VIEW')
        ORDER BY TABLE_NAME
    `, []any{schema}
	default:
		return `
        SELECT table_schema, table_name, table_type
        FROM information_schema.tables
        WHERE table_schema = $1
          AND table_type IN ('BASE TABLE', 'VIEW')
        ORDER BY table_name
    `, []any{schema}
	}
}

func relationalQualifiedTableName(dbType, schema, tableName string) string {
	switch dbType {
	case "mysql":
		return fmt.Sprintf("`%s`.`%s`", escapeMySQLIdentifier(schema), escapeMySQLIdentifier(tableName))
	case "sqlserver":
		return fmt.Sprintf("[%s].[%s]", escapeSQLServerIdentifier(schema), escapeSQLServerIdentifier(tableName))
	default:
		return pgx.Identifier{schema, tableName}.Sanitize()
	}
}

func relationalRenameTableDDL(dbType, schema, oldName, newName string) string {
	switch dbType {
	case "mysql":
		return fmt.Sprintf(
			"RENAME TABLE `%s`.`%s` TO `%s`.`%s`",
			escapeMySQLIdentifier(schema),
			escapeMySQLIdentifier(oldName),
			escapeMySQLIdentifier(schema),
			escapeMySQLIdentifier(newName),
		)
	case "sqlserver":
		return fmt.Sprintf(
			"EXEC sp_rename N'%s.%s', N'%s'",
			escapeSQLServerLiteral(schema),
			escapeSQLServerLiteral(oldName),
			escapeSQLServerLiteral(newName),
		)
	default:
		return fmt.Sprintf(
			"ALTER TABLE %s RENAME TO %s",
			pgx.Identifier{schema, oldName}.Sanitize(),
			pgx.Identifier{newName}.Sanitize(),
		)
	}
}

func relationalDropTableDDL(dbType, schema, tableName string) string {
	return fmt.Sprintf("DROP TABLE %s", relationalQualifiedTableName(dbType, schema, tableName))
}

func relationalTableDataQuery(dbType, qualifiedTableName string, page, limit int) string {
	offset := (page - 1) * limit
	switch dbType {
	case "sqlserver":
		return fmt.Sprintf(
			"SELECT * FROM %s ORDER BY (SELECT NULL) OFFSET %d ROWS FETCH NEXT %d ROWS ONLY",
			qualifiedTableName,
			offset,
			limit,
		)
	default:
		return fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d", qualifiedTableName, limit, offset)
	}
}

func relationalTableColumnsQuery(dbType, schema, tableName string) (string, []any) {
	switch dbType {
	case "mysql":
		return `
        SELECT
            COLUMN_NAME,
            COLUMN_TYPE,
            CHARACTER_MAXIMUM_LENGTH,
            NUMERIC_PRECISION,
            NUMERIC_SCALE,
            CASE WHEN IS_NULLABLE = 'YES' THEN TRUE ELSE FALSE END AS nullable,
            COLUMN_DEFAULT,
            COLUMN_COMMENT,
            CASE WHEN COLUMN_KEY = 'PRI' THEN TRUE ELSE FALSE END AS is_primary,
            CASE WHEN COLUMN_KEY = 'UNI' THEN TRUE ELSE FALSE END AS is_unique,
            CASE WHEN EXTRA LIKE '%auto_increment%' THEN TRUE ELSE FALSE END AS auto_increment
        FROM information_schema.columns
        WHERE TABLE_SCHEMA = ?
          AND TABLE_NAME = ?
        ORDER BY ORDINAL_POSITION
    `, []any{schema, tableName}
	case "sqlserver":
		return `
        SELECT
            c.COLUMN_NAME,
            c.DATA_TYPE,
            c.CHARACTER_MAXIMUM_LENGTH,
            c.NUMERIC_PRECISION,
            c.NUMERIC_SCALE,
            CASE WHEN c.IS_NULLABLE = 'YES' THEN CAST(1 AS bit) ELSE CAST(0 AS bit) END AS nullable,
            CAST(c.COLUMN_DEFAULT AS nvarchar(4000)) AS column_default,
            CAST(ep.value AS nvarchar(4000)) AS column_comment,
            CASE WHEN pk.COLUMN_NAME IS NOT NULL THEN CAST(1 AS bit) ELSE CAST(0 AS bit) END AS is_primary,
            CASE WHEN uq.COLUMN_NAME IS NOT NULL THEN CAST(1 AS bit) ELSE CAST(0 AS bit) END AS is_unique,
            CAST(COLUMNPROPERTY(OBJECT_ID(QUOTENAME(c.TABLE_SCHEMA) + '.' + QUOTENAME(c.TABLE_NAME)), c.COLUMN_NAME, 'IsIdentity') AS bit) AS auto_increment
        FROM INFORMATION_SCHEMA.COLUMNS c
        LEFT JOIN (
            SELECT ku.TABLE_SCHEMA, ku.TABLE_NAME, ku.COLUMN_NAME
            FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
            JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE ku
              ON tc.CONSTRAINT_NAME = ku.CONSTRAINT_NAME
             AND tc.TABLE_SCHEMA = ku.TABLE_SCHEMA
             AND tc.TABLE_NAME = ku.TABLE_NAME
            WHERE tc.CONSTRAINT_TYPE = 'PRIMARY KEY'
        ) pk
          ON pk.TABLE_SCHEMA = c.TABLE_SCHEMA
         AND pk.TABLE_NAME = c.TABLE_NAME
         AND pk.COLUMN_NAME = c.COLUMN_NAME
        LEFT JOIN (
            SELECT ku.TABLE_SCHEMA, ku.TABLE_NAME, ku.COLUMN_NAME
            FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
            JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE ku
              ON tc.CONSTRAINT_NAME = ku.CONSTRAINT_NAME
             AND tc.TABLE_SCHEMA = ku.TABLE_SCHEMA
             AND tc.TABLE_NAME = ku.TABLE_NAME
            WHERE tc.CONSTRAINT_TYPE = 'UNIQUE'
        ) uq
          ON uq.TABLE_SCHEMA = c.TABLE_SCHEMA
         AND uq.TABLE_NAME = c.TABLE_NAME
         AND uq.COLUMN_NAME = c.COLUMN_NAME
        LEFT JOIN sys.extended_properties ep
          ON ep.major_id = OBJECT_ID(QUOTENAME(c.TABLE_SCHEMA) + '.' + QUOTENAME(c.TABLE_NAME))
         AND ep.minor_id = COLUMNPROPERTY(OBJECT_ID(QUOTENAME(c.TABLE_SCHEMA) + '.' + QUOTENAME(c.TABLE_NAME)), c.COLUMN_NAME, 'ColumnId')
         AND ep.name = 'MS_Description'
        WHERE c.TABLE_SCHEMA = @schema
          AND c.TABLE_NAME = @table
        ORDER BY c.ORDINAL_POSITION
    `, []any{schema, tableName}
	default:
		return `
        SELECT
            c.column_name,
            c.data_type,
            c.character_maximum_length,
            c.numeric_precision,
            c.numeric_scale,
            c.is_nullable = 'YES' AS nullable,
            c.column_default,
            pgd.description,
            EXISTS (
                SELECT 1
                FROM information_schema.table_constraints tc
                JOIN information_schema.key_column_usage kcu
                  ON tc.constraint_name = kcu.constraint_name
                 AND tc.table_schema = kcu.table_schema
                 AND tc.table_name = kcu.table_name
                WHERE tc.table_schema = c.table_schema
                  AND tc.table_name = c.table_name
                  AND tc.constraint_type = 'PRIMARY KEY'
                  AND kcu.column_name = c.column_name
            ) AS is_primary,
            EXISTS (
                SELECT 1
                FROM information_schema.table_constraints tc
                JOIN information_schema.key_column_usage kcu
                  ON tc.constraint_name = kcu.constraint_name
                 AND tc.table_schema = kcu.table_schema
                 AND tc.table_name = kcu.table_name
                WHERE tc.table_schema = c.table_schema
                  AND tc.table_name = c.table_name
                  AND tc.constraint_type = 'UNIQUE'
                  AND kcu.column_name = c.column_name
            ) AS is_unique,
            POSITION('nextval(' IN COALESCE(c.column_default, '')) > 0 AS auto_increment
        FROM information_schema.columns c
        LEFT JOIN pg_catalog.pg_statio_all_tables st
          ON st.relname = c.table_name
        LEFT JOIN pg_catalog.pg_description pgd
          ON pgd.objoid = st.relid
         AND pgd.objsubid = c.ordinal_position
        WHERE c.table_schema = $1
          AND c.table_name = $2
        ORDER BY c.ordinal_position
    `, []any{schema, tableName}
	}
}

func relationalTableIndexesQuery(dbType, schema, tableName string) (string, []any) {
	switch dbType {
	case "mysql":
		return `
        SELECT
            INDEX_NAME,
            CASE
                WHEN INDEX_NAME = 'PRIMARY' THEN 'PRIMARY'
                WHEN NON_UNIQUE = 0 THEN 'UNIQUE'
                ELSE 'INDEX'
            END AS index_type,
            INDEX_TYPE,
            COLUMN_NAME
        FROM information_schema.statistics
        WHERE TABLE_SCHEMA = ?
          AND TABLE_NAME = ?
        ORDER BY INDEX_NAME, SEQ_IN_INDEX
    `, []any{schema, tableName}
	case "sqlserver":
		return `
        SELECT
            i.name,
            CASE
                WHEN i.is_primary_key = 1 THEN 'PRIMARY'
                WHEN i.is_unique = 1 THEN 'UNIQUE'
                ELSE 'INDEX'
            END AS index_type,
            i.type_desc,
            c.name
        FROM sys.indexes i
        JOIN sys.index_columns ic
          ON i.object_id = ic.object_id
         AND i.index_id = ic.index_id
        JOIN sys.columns c
          ON ic.object_id = c.object_id
         AND ic.column_id = c.column_id
        JOIN sys.objects o
          ON i.object_id = o.object_id
        JOIN sys.schemas s
          ON o.schema_id = s.schema_id
        WHERE s.name = @schema
          AND o.name = @table
          AND i.name IS NOT NULL
        ORDER BY i.name, ic.key_ordinal
    `, []any{schema, tableName}
	default:
		return "", nil
	}
}

func relationalTableForeignKeysQuery(dbType, schema, tableName string) (string, []any) {
	switch dbType {
	case "mysql":
		return `
        SELECT
            kcu.CONSTRAINT_NAME,
            kcu.COLUMN_NAME,
            kcu.REFERENCED_TABLE_NAME,
            kcu.REFERENCED_COLUMN_NAME,
            rc.UPDATE_RULE,
            rc.DELETE_RULE
        FROM information_schema.KEY_COLUMN_USAGE kcu
        JOIN information_schema.REFERENTIAL_CONSTRAINTS rc
          ON rc.CONSTRAINT_SCHEMA = kcu.CONSTRAINT_SCHEMA
         AND rc.CONSTRAINT_NAME = kcu.CONSTRAINT_NAME
        WHERE kcu.TABLE_SCHEMA = ?
          AND kcu.TABLE_NAME = ?
          AND kcu.REFERENCED_TABLE_NAME IS NOT NULL
        ORDER BY kcu.CONSTRAINT_NAME, kcu.ORDINAL_POSITION
    `, []any{schema, tableName}
	case "sqlserver":
		return `
        SELECT
            fk.name,
            pc.name AS column_name,
            ro.name AS referenced_table,
            rc.name AS referenced_column,
            fk.update_referential_action_desc,
            fk.delete_referential_action_desc
        FROM sys.foreign_keys fk
        JOIN sys.foreign_key_columns fkc
          ON fk.object_id = fkc.constraint_object_id
        JOIN sys.objects po
          ON fk.parent_object_id = po.object_id
        JOIN sys.schemas ps
          ON po.schema_id = ps.schema_id
        JOIN sys.columns pc
          ON fkc.parent_object_id = pc.object_id
         AND fkc.parent_column_id = pc.column_id
        JOIN sys.objects ro
          ON fk.referenced_object_id = ro.object_id
        JOIN sys.columns rc
          ON fkc.referenced_object_id = rc.object_id
         AND fkc.referenced_column_id = rc.column_id
        WHERE ps.name = @schema
          AND po.name = @table
        ORDER BY fk.name, fkc.constraint_column_id
    `, []any{schema, tableName}
	default:
		return `
        SELECT
            tc.constraint_name,
            kcu.column_name,
            ccu.table_name AS referenced_table,
            ccu.column_name AS referenced_column,
            rc.update_rule,
            rc.delete_rule
        FROM information_schema.table_constraints tc
        JOIN information_schema.key_column_usage kcu
          ON tc.constraint_name = kcu.constraint_name
         AND tc.table_schema = kcu.table_schema
        JOIN information_schema.constraint_column_usage ccu
          ON ccu.constraint_name = tc.constraint_name
         AND ccu.table_schema = tc.table_schema
        JOIN information_schema.referential_constraints rc
          ON rc.constraint_name = tc.constraint_name
         AND rc.constraint_schema = tc.table_schema
        WHERE tc.table_schema = $1
          AND tc.table_name = $2
          AND tc.constraint_type = 'FOREIGN KEY'
        ORDER BY tc.constraint_name, kcu.ordinal_position
    `, []any{schema, tableName}
	}
}

func escapeMySQLIdentifier(value string) string {
	return strings.ReplaceAll(value, "`", "``")
}

func escapeSQLServerIdentifier(value string) string {
	return strings.ReplaceAll(value, "]", "]]")
}

func escapeSQLServerLiteral(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}
