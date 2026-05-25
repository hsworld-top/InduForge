package service

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/google/uuid"
)

// ProtocolDevConnection 是开发态会话需要的最小接入源投影。
type ProtocolDevConnection struct {
	ID        string
	ProjectID string
	Type      string
	Name      string
	Config    map[string]any
}

// ProtocolDevConnectionReader 隔离会话服务和具体仓储实现，后续可由 connection repository 适配。
type ProtocolDevConnectionReader interface {
	GetProtocolDevConnection(ctx context.Context, projectID, connectionID string) (*ProtocolDevConnection, error)
}

// ProtocolDevOpcuaModelReader 表示 OPC UA 开发态会话需要读取的建模数据接口。
type ProtocolDevOpcuaModelReader interface{}

// ProtocolDevModbusModelReader 表示 Modbus 开发态会话需要读取的建模数据接口。
type ProtocolDevModbusModelReader interface{}

// ProtocolDevSession 表示 OPC UA / Modbus 工作台的一次开发态短时会话。
type ProtocolDevSession struct {
	SessionID   string         `json:"sessionId"`
	ProjectID    string         `json:"projectId"`
	ConnectionID string         `json:"connectionId"`
	Protocol     string         `json:"protocol"`
	Status       string         `json:"status"`
	ConnectedAt  string         `json:"connectedAt"`
	Endpoint     string         `json:"endpoint"`
	Diagnostics  []string       `json:"diagnostics"`
	Config       map[string]any `json:"-"`
	UserID       string         `json:"-"`
	lastUsedAt   time.Time
}

// ProtocolDevSessionService 管理协议工作台的短时开发态会话。
type ProtocolDevSessionService struct {
	connections ProtocolDevConnectionReader
	opcua       ProtocolDevOpcuaModelReader
	modbus      ProtocolDevModbusModelReader
	mu          sync.Mutex
	sessions    map[string]ProtocolDevSession
	now         func() time.Time
}

// NewProtocolDevSessionService 创建协议开发态会话服务。
func NewProtocolDevSessionService(connections ProtocolDevConnectionReader, opcua ProtocolDevOpcuaModelReader, modbus ProtocolDevModbusModelReader) *ProtocolDevSessionService {
	return &ProtocolDevSessionService{
		connections: connections,
		opcua:       opcua,
		modbus:      modbus,
		sessions:    map[string]ProtocolDevSession{},
		now:         time.Now,
	}
}

// CreateSession 创建一个绑定用户、项目和接入源的开发态会话。
func (s *ProtocolDevSessionService) CreateSession(ctx context.Context, projectID, connectionID, userID, protocol string) (*ProtocolDevSession, error) {
	connection, err := s.loadProtocolConnection(ctx, projectID, connectionID, protocol)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	session := ProtocolDevSession{
		SessionID:   uuid.NewString(),
		ProjectID:    projectID,
		ConnectionID: connectionID,
		Protocol:     protocol,
		Status:       "connected",
		ConnectedAt:  now.Format("2006-01-02 15:04:05"),
		Endpoint:     protocolDevEndpoint(protocol, connection.Config),
		Diagnostics:  []string{},
		Config:       connection.Config,
		UserID:       userID,
		lastUsedAt:   now,
	}
	s.mu.Lock()
	s.sessions[session.SessionID] = session
	s.mu.Unlock()
	return &session, nil
}

// CloseSession 释放开发态会话。
func (s *ProtocolDevSessionService) CloseSession(ctx context.Context, projectID, connectionID, sessionID, userID, protocol string) (*ProtocolDevSession, error) {
	session, err := s.requireSession(projectID, connectionID, sessionID, userID, protocol)
	if err != nil {
		return nil, err
	}
	session.Status = "closed"
	s.mu.Lock()
	delete(s.sessions, sessionID)
	s.mu.Unlock()
	return &session, nil
}

func (s *ProtocolDevSessionService) loadProtocolConnection(ctx context.Context, projectID, connectionID, protocol string) (*ProtocolDevConnection, error) {
	if strings.TrimSpace(projectID) == "" || strings.TrimSpace(connectionID) == "" || strings.TrimSpace(protocol) == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "开发态会话参数不完整")
	}
	if s.connections == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态会话连接仓储未初始化")
	}
	connection, err := s.connections.GetProtocolDevConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	if connection == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "接入源不存在")
	}
	if connection.Type != protocol {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "接入源类型与会话协议不匹配")
	}
	return connection, nil
}

func (s *ProtocolDevSessionService) requireSession(projectID, connectionID, sessionID, userID, protocol string) (ProtocolDevSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[sessionID]
	if !ok || session.ProjectID != projectID || session.ConnectionID != connectionID || session.UserID != userID || session.Protocol != protocol {
		return ProtocolDevSession{}, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "开发态会话不存在或已断开")
	}
	session.lastUsedAt = s.now().UTC()
	s.sessions[sessionID] = session
	return session, nil
}

func protocolDevEndpoint(protocol string, config map[string]any) string {
	if protocol == "opcua" {
		return firstProtocolString(config, "endpoint", "url")
	}
	if firstProtocolString(config, "mode") == "rtu" {
		return "RTU"
	}
	host := firstProtocolString(config, "host")
	port := firstProtocolString(config, "port")
	if host == "" {
		return ""
	}
	if port == "" {
		return host
	}
	return host + ":" + port
}

func firstProtocolString(config map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := config[key]; ok {
			text := strings.TrimSpace(toProtocolString(value))
			if text != "" {
				return text
			}
		}
	}
	return ""
}

func toProtocolString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		return strconv.Itoa(int(typed))
	default:
		return ""
	}
}
