package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

// AccessSource 表示接入源工作区使用的统一连接视图。
type AccessSource struct {
	ID             string                     `json:"id"`
	ProjectID      string                     `json:"projectId"`
	Name           string                     `json:"name"`
	Type           string                     `json:"type"`
	Category       string                     `json:"category"`
	Status         string                     `json:"status"`
	ConfigSummary  map[string]any             `json:"configSummary"`
	DataPointCount int                        `json:"datapointCount"`
	Capabilities   []string                   `json:"capabilities"`
	RecentRecord   *AccessSourceRecordSummary `json:"recentRecord,omitempty"`
	CreatedAt      time.Time                  `json:"createdAt"`
	UpdatedAt      time.Time                  `json:"updatedAt"`
}

// AccessSourceDetail 表示接入源详情页使用的统一读模型。
type AccessSourceDetail struct {
	AccessSource
	Config  map[string]any                `json:"config"`
	Records []AccessSourceRecordSummary   `json:"records"`
	Tabs    []AccessSourceTabAvailability `json:"tabs"`
}

// AccessSourceRecordSummary 表示接入源最近操作或预览记录。
type AccessSourceRecordSummary struct {
	Type      string         `json:"type"`
	Title     string         `json:"title"`
	Status    string         `json:"status"`
	Detail    map[string]any `json:"detail"`
	CreatedAt time.Time      `json:"createdAt"`
}

// AccessSourceTabAvailability 表示前端可直接启用的详情页标签。
type AccessSourceTabAvailability struct {
	Key       string `json:"key"`
	Enabled   bool   `json:"enabled"`
	Reason    string `json:"reason,omitempty"`
	EntryPath string `json:"entryPath,omitempty"`
}

// AccessSourceService 聚合连接、协议配置与数据点数量，提供开发态接入源读模型。
type AccessSourceService struct {
	connections *repository.ConnectionRepository
	mqtt        *repository.MqttRepository
	repository  *repository.AccessSourceRepository
}

// NewAccessSourceService 创建接入源聚合服务。
func NewAccessSourceService(
	connectionRepo *repository.ConnectionRepository,
	mqttRepo *repository.MqttRepository,
	accessSourceRepo *repository.AccessSourceRepository,
) *AccessSourceService {
	return &AccessSourceService{
		connections: connectionRepo,
		mqtt:        mqttRepo,
		repository:  accessSourceRepo,
	}
}

// ListAccessSources 返回项目下所有接入源摘要。
func (s *AccessSourceService) ListAccessSources(ctx context.Context, projectID string) ([]AccessSource, error) {
	if err := s.validateDependencies(); err != nil {
		return nil, err
	}
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}

	connections, err := s.connections.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	counts, err := s.repository.CountDataPointsByConnections(ctx, projectID)
	if err != nil {
		return nil, err
	}
	recentRecords, err := s.repository.ListRecentRecords(ctx, projectID)
	if err != nil {
		return nil, err
	}

	result := make([]AccessSource, 0, len(connections))
	for _, connection := range connections {
		item := s.buildAccessSource(ctx, connection, counts[connection.ID], recentRecords[connection.ID])
		result = append(result, item)
	}
	return result, nil
}

// GetAccessSource 返回单个接入源详情。
func (s *AccessSourceService) GetAccessSource(ctx context.Context, projectID, sourceID string) (*AccessSourceDetail, error) {
	if err := s.validateDependencies(); err != nil {
		return nil, err
	}
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(sourceID); err != nil {
		return nil, err
	}

	connection, err := s.connections.GetByProjectAndID(ctx, projectID, sourceID)
	if err != nil {
		return nil, err
	}
	counts, err := s.repository.CountDataPointsByConnections(ctx, projectID)
	if err != nil {
		return nil, err
	}
	records, err := s.repository.ListRecordsByConnection(ctx, projectID, sourceID, 20)
	if err != nil {
		return nil, err
	}

	base := s.buildAccessSource(ctx, *connection, counts[connection.ID], repository.AccessSourceRecord{})
	detail := &AccessSourceDetail{
		AccessSource: base,
		Config:       s.connectionConfig(ctx, *connection, false),
		Records:      toAccessSourceRecordSummaries(records),
		Tabs:         accessSourceTabs(*connection),
	}
	if len(detail.Records) > 0 {
		detail.RecentRecord = &detail.Records[0]
	}
	return detail, nil
}

// ListAccessSourceRecords 返回单个接入源的最近开发态记录。
func (s *AccessSourceService) ListAccessSourceRecords(ctx context.Context, projectID, sourceID string, limit int) ([]AccessSourceRecordSummary, error) {
	if err := s.validateDependencies(); err != nil {
		return nil, err
	}
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(sourceID); err != nil {
		return nil, err
	}
	if _, err := s.connections.GetByProjectAndID(ctx, projectID, sourceID); err != nil {
		return nil, err
	}

	records, err := s.repository.ListRecordsByConnection(ctx, projectID, sourceID, limit)
	if err != nil {
		return nil, err
	}
	return toAccessSourceRecordSummaries(records), nil
}

func (s *AccessSourceService) validateDependencies() error {
	if s == nil || s.connections == nil || s.repository == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "接入源服务依赖未初始化")
	}
	return nil
}

func (s *AccessSourceService) buildAccessSource(ctx context.Context, connection repository.ConnectionRecord, dataPointCount int, recent repository.AccessSourceRecord) AccessSource {
	item := AccessSource{
		ID:             connection.ID,
		ProjectID:      connection.ProjectID,
		Name:           connection.Name,
		Type:           connection.Type,
		Category:       connection.Category,
		Status:         connection.Status,
		ConfigSummary:  s.connectionConfig(ctx, connection, true),
		DataPointCount: dataPointCount,
		Capabilities:   accessSourceCapabilities(connection.Type),
		CreatedAt:      connection.CreatedAt,
		UpdatedAt:      connection.UpdatedAt,
	}
	if strings.TrimSpace(item.Category) == "" {
		item.Category = deriveStoredConnectionCategory(item.Type)
	}
	if recent.ConnectionID != "" {
		summary := toAccessSourceRecordSummary(recent)
		item.RecentRecord = &summary
	}
	return item
}

func (s *AccessSourceService) connectionConfig(ctx context.Context, connection repository.ConnectionRecord, summaryOnly bool) map[string]any {
	config := cloneMap(connection.Config)
	if connection.Type == "mqtt" && s.mqtt != nil {
		if detail, err := s.mqtt.GetConnectionDetail(ctx, connection.ProjectID, connection.ID); err == nil {
			config = map[string]any{
				"brokerUrl":         detail.BrokerURL,
				"protocol":          detail.Protocol,
				"port":              detail.Port,
				"clientId":          optionalStringValue(detail.ClientID),
				"username":          optionalStringValue(detail.Username),
				"password":          optionalStringValue(detail.Password),
				"keepalive":         detail.Keepalive,
				"cleanSession":      detail.CleanSession,
				"qos":               detail.QOS,
				"reconnectPeriodMs": detail.ReconnectPeriodMS,
				"connectTimeoutMs":  detail.ConnectTimeoutMS,
				"sslConfig":         cloneMap(detail.SSLConfig),
				"will":              cloneMap(detail.Will),
			}
		}
	}

	sanitized := sanitizeConfigMap(config)
	if !summaryOnly {
		return sanitized
	}
	return compactConfigSummary(connection.Type, sanitized)
}

func optionalStringValue(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func sanitizeConfigMap(input map[string]any) map[string]any {
	result := make(map[string]any, len(input))
	for key, value := range input {
		if isSensitiveConfigKey(key) {
			if value == nil || strings.TrimSpace(toString(value)) == "" {
				result[key] = nil
			} else {
				result[key] = "******"
			}
			continue
		}
		if nested, ok := value.(map[string]any); ok {
			result[key] = sanitizeConfigMap(nested)
			continue
		}
		result[key] = value
	}
	return result
}

func isSensitiveConfigKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	if normalized == "" {
		return false
	}
	sensitiveParts := []string{"password", "passwd", "secret", "token", "credential", "privatekey", "private_key", "certificate"}
	for _, part := range sensitiveParts {
		if strings.Contains(normalized, part) {
			return true
		}
	}
	return false
}

func compactConfigSummary(connectionType string, config map[string]any) map[string]any {
	switch strings.TrimSpace(strings.ToLower(connectionType)) {
	case "relational":
		return pickConfigKeys(config, "dbType", "host", "port", "database", "schema", "username")
	case "mqtt":
		return pickConfigKeys(config, "brokerUrl", "protocol", "port", "clientId", "username")
	case "kafka":
		return pickConfigKeys(config, "brokers", "topic")
	case "http":
		return pickConfigKeys(config, "baseUrl", "method")
	case "websocket":
		return pickConfigKeys(config, "url", "topic")
	case "redis":
		return pickConfigKeys(config, "address", "db", "username", "keyPattern", "mode")
	default:
		return cloneMap(config)
	}
}

func pickConfigKeys(config map[string]any, keys ...string) map[string]any {
	result := make(map[string]any)
	for _, key := range keys {
		if value, ok := config[key]; ok {
			result[key] = value
		}
	}
	return result
}

func accessSourceCapabilities(connectionType string) []string {
	switch strings.TrimSpace(strings.ToLower(connectionType)) {
	case "relational":
		return []string{"tables", "structure", "tableData", "executeSql", "datapoints"}
	case "mqtt":
		return []string{"subscriptions", "messages", "tags", "preview", "datapoints"}
	case "kafka":
		return []string{"config", "mockPreview", "artifact"}
	case "http", "websocket", "redis":
		return []string{"config", "artifact"}
	default:
		return []string{"config"}
	}
}

func accessSourceTabs(connection repository.ConnectionRecord) []AccessSourceTabAvailability {
	tabs := []AccessSourceTabAvailability{
		{Key: "overview", Enabled: true},
		{Key: "config", Enabled: true},
		{Key: "datapoints", Enabled: true},
		{Key: "records", Enabled: true},
	}
	switch connection.Type {
	case "relational":
		tabs = append(tabs,
			AccessSourceTabAvailability{Key: "mapping", Enabled: true, EntryPath: "queries"},
			AccessSourceTabAvailability{Key: "preview", Enabled: true, EntryPath: "tables"},
		)
	case "mqtt":
		tabs = append(tabs,
			AccessSourceTabAvailability{Key: "mapping", Enabled: true, EntryPath: "mqtt/tags"},
			AccessSourceTabAvailability{Key: "preview", Enabled: true, EntryPath: "mqtt/messages"},
		)
	default:
		tabs = append(tabs,
			AccessSourceTabAvailability{Key: "mapping", Enabled: false, Reason: "该协议当前仅提供配置与 artifact 输出"},
			AccessSourceTabAvailability{Key: "preview", Enabled: false, Reason: "该协议当前未提供独立 preview 能力"},
		)
	}
	return tabs
}

func toAccessSourceRecordSummaries(records []repository.AccessSourceRecord) []AccessSourceRecordSummary {
	result := make([]AccessSourceRecordSummary, 0, len(records))
	for _, record := range records {
		result = append(result, toAccessSourceRecordSummary(record))
	}
	return result
}

func toAccessSourceRecordSummary(record repository.AccessSourceRecord) AccessSourceRecordSummary {
	return AccessSourceRecordSummary{
		Type:      record.RecordType,
		Title:     record.Title,
		Status:    record.Status,
		Detail:    cloneMap(record.Detail),
		CreatedAt: record.CreatedAt,
	}
}

func toString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}
