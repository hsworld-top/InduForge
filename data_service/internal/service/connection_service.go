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

var connectionTypeCategoryMap = map[string]string{
	"relational": "database",
	"mqtt":       "message",
	"websocket":  "protocol",
	"opcua":      "protocol",
	"modbus":     "protocol",
	"http":       "api",
	"s7":         "protocol",
}

var allowedConnectionStatus = map[string]struct{}{
	"connected":    {},
	"disconnected": {},
	"error":        {},
	"unknown":      {},
}

// Connection 表示面向 HTTP 层返回的连接对象。
type Connection struct {
	ID        string         `json:"id"`
	ProjectID string         `json:"projectId"`
	TenantID  string         `json:"tenantId"`
	Name      string         `json:"name"`
	Type      string         `json:"type"`
	Status    string         `json:"status"`
	Config    map[string]any `json:"config"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
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
	repository *repository.ConnectionRepository
}

// NewConnectionService 创建连接服务。
func NewConnectionService(repo *repository.ConnectionRepository) *ConnectionService {
	return &ConnectionService{repository: repo}
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
	for _, record := range records {
		connections = append(connections, toConnection(record, tenantID))
	}

	return connections, nil
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
	status, err := normalizeConnectionStatus(input.Status)
	if err != nil {
		return nil, err
	}
	config, err := normalizeConnectionConfig(input.Config)
	if err != nil {
		return nil, err
	}

	record, err := s.repository.Create(ctx, repository.CreateConnectionParams{
		ProjectID: projectID,
		UserID:    userID,
		Name:      name,
		Type:      connectionType,
		Category:  category,
		Status:    status,
		Config:    config,
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

	nextName := current.Name
	if input.Name != nil {
		nextName, err = normalizeConnectionName(*input.Name)
		if err != nil {
			return nil, err
		}
	}

	nextType := current.Type
	nextCategory := connectionTypeCategoryMap[current.Type]
	if input.Type != nil {
		nextType, nextCategory, err = normalizeConnectionType(*input.Type)
		if err != nil {
			return nil, err
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

	record, err := s.repository.Update(ctx, repository.UpdateConnectionParams{
		ID:        connectionID,
		ProjectID: projectID,
		UserID:    userID,
		Name:      nextName,
		Type:      nextType,
		Category:  nextCategory,
		Status:    nextStatus,
		Config:    nextConfig,
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

func toConnection(record repository.ConnectionRecord, tenantID string) Connection {
	return Connection{
		ID:        record.ID,
		ProjectID: record.ProjectID,
		TenantID:  tenantID,
		Name:      record.Name,
		Type:      record.Type,
		Status:    record.Status,
		Config:    cloneMap(record.Config),
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
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
	category, ok := connectionTypeCategoryMap[connectionType]
	if !ok {
		return "", "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接类型不受支持")
	}
	return connectionType, category, nil
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
