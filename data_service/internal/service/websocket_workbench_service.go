package service

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

const (
	defaultWebSocketTimeoutMS = 5000
	maxWebSocketTimeoutMS     = 30000
	maxWebSocketMessages      = 50
	maxWebSocketMessageBytes  = 2 * 1024 * 1024
)

type WebSocketSessionGroup struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"projectId"`
	ConnectionID string    `json:"connectionId"`
	ParentID     *string   `json:"parentId"`
	Name         string    `json:"name"`
	SortOrder    int       `json:"sortOrder"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type WebSocketSession struct {
	ID             string         `json:"id"`
	ProjectID      string         `json:"projectId"`
	ConnectionID   string         `json:"connectionId"`
	GroupID        *string        `json:"groupId"`
	Name           string         `json:"name"`
	URL            string         `json:"url"`
	Headers        []any          `json:"headers"`
	Auth           map[string]any `json:"auth"`
	Protocols      []any          `json:"protocols"`
	Messages       []any          `json:"messages"`
	Settings       map[string]any `json:"settings"`
	Enabled        bool           `json:"enabled"`
	SortOrder      int            `json:"sortOrder"`
	SourceType     string         `json:"sourceType"`
	DataPointID    string         `json:"dataPointId"`
	DataPointPath  string         `json:"dataPointPath"`
	LastMessage    any            `json:"lastMessage"`
	LastDiagnostic string         `json:"lastDiagnostic"`
	Quality        string         `json:"quality"`
	LastMessageAt  *time.Time     `json:"lastMessageAt"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

type CreateWebSocketSessionGroupInput struct {
	ParentID  *string `json:"parentId"`
	Name      string  `json:"name"`
	SortOrder int     `json:"sortOrder"`
}

type UpdateWebSocketSessionGroupInput struct {
	ParentID    *string `json:"parentId"`
	HasParentID bool    `json:"-"`
	Name        string  `json:"name"`
	SortOrder   int     `json:"sortOrder"`
}

type CreateWebSocketSessionInput struct {
	GroupID   *string        `json:"groupId"`
	Name      string         `json:"name"`
	URL       string         `json:"url"`
	Headers   []any          `json:"headers"`
	Auth      map[string]any `json:"auth"`
	Protocols []any          `json:"protocols"`
	Messages  []any          `json:"messages"`
	Settings  map[string]any `json:"settings"`
	Enabled   bool           `json:"enabled"`
	SortOrder int            `json:"sortOrder"`
}

type UpdateWebSocketSessionInput struct {
	GroupID    *string        `json:"groupId"`
	HasGroupID bool           `json:"-"`
	Name       string         `json:"name"`
	URL        string         `json:"url"`
	Headers    []any          `json:"headers"`
	Auth       map[string]any `json:"auth"`
	Protocols  []any          `json:"protocols"`
	Messages   []any          `json:"messages"`
	Settings   map[string]any `json:"settings"`
	Enabled    bool           `json:"enabled"`
	SortOrder  int            `json:"sortOrder"`
}

type WebSocketSessionListResult struct {
	List       []WebSocketSession         `json:"list"`
	Pagination ProtocolModelingPagination `json:"pagination"`
}

type WebSocketPreviewMessage struct {
	Direction  string    `json:"direction"`
	Type       string    `json:"type"`
	Payload    any       `json:"payload"`
	RawPayload string    `json:"rawPayload"`
	SizeBytes  int       `json:"sizeBytes"`
	Timestamp  time.Time `json:"timestamp"`
}

type WebSocketPreviewResponse struct {
	Status      string                    `json:"status"`
	Messages    []WebSocketPreviewMessage `json:"messages"`
	Diagnostics map[string]any            `json:"diagnostics"`
	DurationMS  int64                     `json:"durationMs"`
	Truncated   bool                      `json:"truncated"`
}

type WebSocketStreamEnvelope struct {
	Type    string                   `json:"type"`
	Status  string                   `json:"status,omitempty"`
	Message string                   `json:"message,omitempty"`
	Data    *WebSocketPreviewMessage `json:"data,omitempty"`
}

type WebSocketWorkbenchService struct {
	repository  *repository.WebSocketWorkbenchRepository
	connections *repository.ConnectionRepository
}

func NewWebSocketWorkbenchService(repo *repository.WebSocketWorkbenchRepository, connections *repository.ConnectionRepository) *WebSocketWorkbenchService {
	return &WebSocketWorkbenchService{repository: repo, connections: connections}
}

func (s *WebSocketWorkbenchService) ListGroups(ctx context.Context, projectID, connectionID string) ([]WebSocketSessionGroup, error) {
	if err := validateProjectAndConnection(projectID, connectionID); err != nil {
		return nil, err
	}
	records, err := s.repository.ListGroups(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	result := make([]WebSocketSessionGroup, 0, len(records))
	for _, record := range records {
		result = append(result, toWebSocketSessionGroup(record))
	}
	return result, nil
}

func (s *WebSocketWorkbenchService) CreateGroup(ctx context.Context, projectID, connectionID, userID string, input CreateWebSocketSessionGroupInput) (*WebSocketSessionGroup, error) {
	if err := validateProjectConnectionAndUser(projectID, connectionID, userID); err != nil {
		return nil, err
	}
	name, err := normalizeWebSocketRequiredText(input.Name, 100, "WebSocket 会话分组名称不能为空")
	if err != nil {
		return nil, err
	}
	parentID, err := s.normalizeGroupParent(ctx, projectID, connectionID, input.ParentID, "")
	if err != nil {
		return nil, err
	}
	record, err := s.repository.CreateGroup(ctx, repository.CreateWebSocketSessionGroupParams{
		ProjectID: projectID, ConnectionID: connectionID, ParentID: parentID, Name: name, SortOrder: input.SortOrder, UserID: userID,
	})
	if err != nil {
		return nil, err
	}
	result := toWebSocketSessionGroup(*record)
	return &result, nil
}

func (s *WebSocketWorkbenchService) UpdateGroup(ctx context.Context, projectID, groupID, userID string, input UpdateWebSocketSessionGroupInput) (*WebSocketSessionGroup, error) {
	if err := validateProjectAndUser(projectID, userID); err != nil {
		return nil, err
	}
	if err := validateUUIDText(groupID, "groupId 格式无效"); err != nil {
		return nil, err
	}
	current, err := s.repository.GetGroup(ctx, projectID, groupID)
	if err != nil {
		return nil, err
	}
	name, err := normalizeWebSocketRequiredText(fallbackTrimmed(input.Name, current.Name), 100, "WebSocket 会话分组名称不能为空")
	if err != nil {
		return nil, err
	}
	parentID := cloneOptionalString(current.ParentID)
	if input.HasParentID {
		parentID, err = s.normalizeGroupParent(ctx, projectID, current.ConnectionID, input.ParentID, current.ID)
		if err != nil {
			return nil, err
		}
	}
	sortOrder := input.SortOrder
	if sortOrder == 0 && current.SortOrder != 0 {
		sortOrder = current.SortOrder
	}
	record, err := s.repository.UpdateGroup(ctx, repository.UpdateWebSocketSessionGroupParams{
		ProjectID: projectID, GroupID: groupID, ParentID: parentID, Name: name, SortOrder: sortOrder, UserID: userID,
	})
	if err != nil {
		return nil, err
	}
	result := toWebSocketSessionGroup(*record)
	return &result, nil
}

func (s *WebSocketWorkbenchService) DeleteGroup(ctx context.Context, projectID, groupID, userID string) error {
	if err := validateProjectAndUser(projectID, userID); err != nil {
		return err
	}
	if err := validateUUIDText(groupID, "groupId 格式无效"); err != nil {
		return err
	}
	return s.repository.DeleteGroup(ctx, projectID, groupID, userID)
}

func (s *WebSocketWorkbenchService) ListSessionsPage(ctx context.Context, projectID, connectionID string, groupID *string, search string, page, pageSize int) (*WebSocketSessionListResult, error) {
	if err := validateProjectAndConnection(projectID, connectionID); err != nil {
		return nil, err
	}
	page, pageSize = normalizePageAndSize(page, pageSize, 20, 100)
	records, total, err := s.repository.ListSessionsPage(ctx, projectID, connectionID, groupID, search, page, pageSize)
	if err != nil {
		return nil, err
	}
	items := make([]WebSocketSession, 0, len(records))
	for _, record := range records {
		items = append(items, toWebSocketSession(record))
	}
	return &WebSocketSessionListResult{List: items, Pagination: newProtocolModelingPagination(page, pageSize, total)}, nil
}

func (s *WebSocketWorkbenchService) GetSession(ctx context.Context, projectID, sessionID string) (*WebSocketSession, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateUUIDText(sessionID, "sessionId 格式无效"); err != nil {
		return nil, err
	}
	record, err := s.repository.GetSession(ctx, projectID, sessionID)
	if err != nil {
		return nil, err
	}
	result := toWebSocketSession(*record)
	return &result, nil
}

func (s *WebSocketWorkbenchService) CreateSession(ctx context.Context, projectID, connectionID, userID string, input CreateWebSocketSessionInput) (*WebSocketSession, error) {
	if err := validateProjectConnectionAndUser(projectID, connectionID, userID); err != nil {
		return nil, err
	}
	connection, err := s.loadWebSocketConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	params, err := s.normalizeCreateSession(ctx, projectID, userID, *connection, input)
	if err != nil {
		return nil, err
	}
	record, err := s.repository.CreateSessionWithDataPoint(ctx, params)
	if err != nil {
		return nil, err
	}
	result := toWebSocketSession(*record)
	return &result, nil
}

func (s *WebSocketWorkbenchService) UpdateSession(ctx context.Context, projectID, sessionID, userID string, input UpdateWebSocketSessionInput) (*WebSocketSession, error) {
	if err := validateProjectAndUser(projectID, userID); err != nil {
		return nil, err
	}
	if err := validateUUIDText(sessionID, "sessionId 格式无效"); err != nil {
		return nil, err
	}
	current, err := s.repository.GetSession(ctx, projectID, sessionID)
	if err != nil {
		return nil, err
	}
	connection, err := s.loadWebSocketConnection(ctx, projectID, current.ConnectionID)
	if err != nil {
		return nil, err
	}
	params, err := s.normalizeUpdateSession(ctx, projectID, userID, *connection, *current, input)
	if err != nil {
		return nil, err
	}
	record, err := s.repository.UpdateSessionWithDataPoint(ctx, params)
	if err != nil {
		return nil, err
	}
	result := toWebSocketSession(*record)
	return &result, nil
}

func (s *WebSocketWorkbenchService) DeleteSession(ctx context.Context, projectID, sessionID string) error {
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	if err := validateUUIDText(sessionID, "sessionId 格式无效"); err != nil {
		return err
	}
	return s.repository.DeleteSessionWithDataPoint(ctx, projectID, sessionID)
}

func (s *WebSocketWorkbenchService) ConnectPreview(ctx context.Context, projectID, sessionID, userID string) (*WebSocketPreviewResponse, error) {
	if err := validateProjectAndUser(projectID, userID); err != nil {
		return nil, err
	}
	if err := validateUUIDText(sessionID, "sessionId 格式无效"); err != nil {
		return nil, err
	}
	record, err := s.repository.GetSession(ctx, projectID, sessionID)
	if err != nil {
		return nil, err
	}
	if !record.Enabled {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "WebSocket 会话已停用")
	}
	connection, err := s.loadWebSocketConnection(ctx, projectID, record.ConnectionID)
	if err != nil {
		return nil, err
	}

	startedAt := time.Now()
	result, previewErr := s.executePreview(ctx, *connection, *record)
	quality := "good"
	diagnostic := ""
	var defaultValue *string
	var lastMessage any
	lastAt := time.Now().UTC()
	if previewErr != nil {
		quality = "bad"
		diagnostic = previewErr.Error()
		result = &WebSocketPreviewResponse{
			Status:      "error",
			Messages:    []WebSocketPreviewMessage{},
			Diagnostics: map[string]any{"error": diagnostic},
			DurationMS:  time.Since(startedAt).Milliseconds(),
		}
		lastMessage = map[string]any{"error": diagnostic}
	} else if len(result.Messages) > 0 {
		last := result.Messages[len(result.Messages)-1]
		lastMessage = last
		lastAt = last.Timestamp
		value := stringifyJSONValue(last)
		defaultValue = &value
	} else {
		lastMessage = map[string]any{"message": "预览未读取到消息"}
	}
	diagPtr := optionalString(diagnostic)
	if err := s.repository.SavePreviewSnapshot(ctx, projectID, repository.WebSocketSessionPreviewSnapshot{
		SessionID:       record.ID,
		LastMessage:     lastMessage,
		LastDiagnostic:  diagPtr,
		Quality:         quality,
		LastMessageAt:   lastAt,
		DefaultValue:    defaultValue,
		DataPointConfig: webSocketSessionSourceConfig(*connection, *record),
		UserID:          userID,
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *WebSocketWorkbenchService) StreamSession(ctx context.Context, w http.ResponseWriter, projectID, sessionID, userID string) error {
	if err := validateProjectAndUser(projectID, userID); err != nil {
		return err
	}
	if err := validateUUIDText(sessionID, "sessionId 格式无效"); err != nil {
		return err
	}
	record, err := s.repository.GetSession(ctx, projectID, sessionID)
	if err != nil {
		return err
	}
	connection, err := s.loadWebSocketConnection(ctx, projectID, record.ConnectionID)
	if err != nil {
		return err
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	clientConn, err := upgrader.Upgrade(w, nil, nil)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "WebSocket 工作台连接升级失败", err)
	}
	defer clientConn.Close()

	targetConn, targetResp, err := dialWebSocketTarget(ctx, *connection, *record)
	if err != nil {
		_ = clientConn.WriteJSON(WebSocketStreamEnvelope{Type: "error", Message: err.Error()})
		return nil
	}
	defer targetConn.Close()
	done := make(chan struct{})
	var closeOnce sync.Once
	var clientWriteMu sync.Mutex
	var targetWriteMu sync.Mutex
	closeDone := func() {
		closeOnce.Do(func() {
			close(done)
			_ = targetConn.Close()
		})
	}
	// 浏览器断开或请求上下文取消时，立即关闭目标连接，避免目标端无消息时读循环长期阻塞。
	go func() {
		defer closeDone()
		for {
			messageType, payload, err := clientConn.ReadMessage()
			if err != nil {
				return
			}
			targetWriteMu.Lock()
			err = targetConn.WriteMessage(messageType, payload)
			targetWriteMu.Unlock()
			if err != nil {
				writeWebSocketStreamEnvelope(clientConn, &clientWriteMu, WebSocketStreamEnvelope{Type: "error", Message: "WebSocket 发送消息失败: " + err.Error()})
				return
			}
			message := buildWebSocketPreviewMessage("out", messageType, payload)
			writeWebSocketStreamEnvelope(clientConn, &clientWriteMu, WebSocketStreamEnvelope{Type: "message", Data: &message})
		}
	}()
	go func() {
		select {
		case <-ctx.Done():
			closeDone()
		case <-done:
		}
	}()

	writeWebSocketStreamEnvelope(clientConn, &clientWriteMu, WebSocketStreamEnvelope{
		Type:    "status",
		Status:  "connected",
		Message: fmt.Sprintf("已连接，状态码 %d", responseStatusCode(targetResp)),
	})

	for {
		select {
		case <-done:
			return nil
		case <-ctx.Done():
			closeDone()
			return nil
		default:
		}
		messageType, payload, readErr := targetConn.ReadMessage()
		if readErr != nil {
			_ = clientConn.WriteJSON(WebSocketStreamEnvelope{Type: "error", Message: "WebSocket 读取消息失败: " + readErr.Error()})
			return nil
		}
		if len(payload) > maxWebSocketMessageBytes {
			payload = payload[:maxWebSocketMessageBytes]
		}
		message := buildWebSocketPreviewMessage("in", messageType, payload)
		if err := writeWebSocketStreamEnvelope(clientConn, &clientWriteMu, WebSocketStreamEnvelope{Type: "message", Data: &message}); err != nil {
			return nil
		}
	}
}

func writeWebSocketStreamEnvelope(conn *websocket.Conn, mu *sync.Mutex, envelope WebSocketStreamEnvelope) error {
	if conn == nil || mu == nil {
		return nil
	}
	mu.Lock()
	defer mu.Unlock()
	return conn.WriteJSON(envelope)
}

func (s *WebSocketWorkbenchService) executePreview(ctx context.Context, connection repository.ConnectionRecord, record repository.WebSocketSessionRecord) (*WebSocketPreviewResponse, error) {
	timeoutMS := webSocketTimeoutMS(record.Settings, connection.Config)
	previewCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMS)*time.Millisecond)
	defer cancel()

	startedAt := time.Now()
	conn, resp, err := dialWebSocketTarget(previewCtx, connection, record)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	messages := make([]WebSocketPreviewMessage, 0)
	for _, row := range enabledMessageRows(record.Messages) {
		if err := conn.WriteMessage(websocket.TextMessage, []byte(row)); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "WebSocket preview 发送订阅消息失败", err)
		}
		messages = append(messages, buildWebSocketPreviewMessage("out", websocket.TextMessage, []byte(row)))
	}

	limit := intFromAny(record.Settings["messageLimit"], 5)
	if limit <= 0 {
		limit = 5
	}
	if limit > maxWebSocketMessages {
		limit = maxWebSocketMessages
	}
	readDeadline := time.Now().Add(time.Duration(timeoutMS) * time.Millisecond)
	_ = conn.SetReadDeadline(readDeadline)
	truncated := false
	for len(receivedMessages(messages)) < limit {
		messageType, payload, readErr := conn.ReadMessage()
		if readErr != nil {
			if len(receivedMessages(messages)) == 0 {
				return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "WebSocket preview 读取消息失败", readErr)
			}
			break
		}
		if len(payload) > maxWebSocketMessageBytes {
			truncated = true
			payload = payload[:maxWebSocketMessageBytes]
		}
		messages = append(messages, buildWebSocketPreviewMessage("in", messageType, payload))
	}

	return &WebSocketPreviewResponse{
		Status:   "ok",
		Messages: messages,
		Diagnostics: map[string]any{
			"url":          conn.RemoteAddr().String(),
			"protocol":     conn.Subprotocol(),
			"statusCode":   responseStatusCode(resp),
			"messageLimit": limit,
		},
		DurationMS: time.Since(startedAt).Milliseconds(),
		Truncated:  truncated,
	}, nil
}

func dialWebSocketTarget(ctx context.Context, connection repository.ConnectionRecord, record repository.WebSocketSessionRecord) (*websocket.Conn, *http.Response, error) {
	timeoutMS := webSocketTimeoutMS(record.Settings, connection.Config)
	targetURL, err := resolveWebSocketURL(connection.Config, record.URL)
	if err != nil {
		return nil, nil, err
	}
	headers := http.Header{}
	for key, value := range enabledKeyValuePairs(record.Headers) {
		headers.Set(key, value)
	}
	if err := applyWebSocketAuth(headers, record.Auth); err != nil {
		return nil, nil, err
	}
	dialer := websocket.Dialer{
		HandshakeTimeout: time.Duration(timeoutMS) * time.Millisecond,
		Subprotocols:     enabledProtocolValues(record.Protocols),
		TLSClientConfig:  webSocketTLSConfig(record.Settings), //nolint:gosec
	}
	conn, resp, err := dialer.DialContext(ctx, targetURL, headers)
	if err != nil {
		return nil, resp, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "WebSocket 连接失败", errWithStatus(err, responseStatusCode(resp)))
	}
	return conn, resp, nil
}

func (s *WebSocketWorkbenchService) normalizeCreateSession(ctx context.Context, projectID, userID string, connection repository.ConnectionRecord, input CreateWebSocketSessionInput) (repository.CreateWebSocketSessionParams, error) {
	name, sessionURL, err := normalizeWebSocketSessionCore(input.Name, input.URL)
	if err != nil {
		return repository.CreateWebSocketSessionParams{}, err
	}
	groupID, err := s.normalizeSessionGroup(ctx, projectID, connection.ID, input.GroupID)
	if err != nil {
		return repository.CreateWebSocketSessionParams{}, err
	}
	record := repository.WebSocketSessionRecord{ProjectID: projectID, ConnectionID: connection.ID, GroupID: groupID, Name: name, URL: sessionURL}
	return repository.CreateWebSocketSessionParams{
		ProjectID: projectID, ConnectionID: connection.ID, GroupID: groupID, Name: name, URL: sessionURL,
		Headers: normalizeKeyValueRows(input.Headers), Auth: normalizeWebSocketAuth(input.Auth),
		Protocols: normalizeProtocolRows(input.Protocols), Messages: normalizeMessageRows(input.Messages),
		Settings: normalizeWebSocketSettings(input.Settings), Enabled: true, SortOrder: input.SortOrder,
		DataPointPath:   buildWebSocketDataPointPath(connection.Name, name),
		DataPointConfig: webSocketSessionSourceConfig(connection, record), UserID: userID,
	}, nil
}

func (s *WebSocketWorkbenchService) normalizeUpdateSession(ctx context.Context, projectID, userID string, connection repository.ConnectionRecord, current repository.WebSocketSessionRecord, input UpdateWebSocketSessionInput) (repository.UpdateWebSocketSessionParams, error) {
	name, sessionURL, err := normalizeWebSocketSessionCore(fallbackTrimmed(input.Name, current.Name), fallbackTrimmed(input.URL, current.URL))
	if err != nil {
		return repository.UpdateWebSocketSessionParams{}, err
	}
	groupID := current.GroupID
	if input.HasGroupID {
		groupID, err = s.normalizeSessionGroup(ctx, projectID, connection.ID, input.GroupID)
		if err != nil {
			return repository.UpdateWebSocketSessionParams{}, err
		}
	}
	record := current
	record.Name = name
	record.URL = sessionURL
	return repository.UpdateWebSocketSessionParams{
		ID: current.ID, ProjectID: projectID, GroupID: groupID, Name: name, URL: sessionURL,
		Headers:   normalizeKeyValueRowsOrDefault(input.Headers, current.Headers),
		Auth:      normalizeMapOrDefault(input.Auth, current.Auth, normalizeWebSocketAuth),
		Protocols: normalizeProtocolRowsOrDefault(input.Protocols, current.Protocols),
		Messages:  normalizeMessageRowsOrDefault(input.Messages, current.Messages),
		Settings:  normalizeMapOrDefault(input.Settings, current.Settings, normalizeWebSocketSettings),
		Enabled:   input.Enabled, SortOrder: input.SortOrder,
		DataPointPath:   buildWebSocketDataPointPath(connection.Name, name),
		DataPointConfig: webSocketSessionSourceConfig(connection, record),
		DefaultValue:    defaultValueFromSnapshot(current.LastMessage),
		UserID:          userID,
	}, nil
}

func (s *WebSocketWorkbenchService) normalizeGroupParent(ctx context.Context, projectID, connectionID string, parentID *string, currentID string) (*string, error) {
	if parentID == nil || strings.TrimSpace(*parentID) == "" {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*parentID)
	if err := validateUUIDText(trimmed, "parentId 格式无效"); err != nil {
		return nil, err
	}
	parent, err := s.repository.GetGroup(ctx, projectID, trimmed)
	if err != nil {
		return nil, err
	}
	if parent.ConnectionID != connectionID {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "上级分组不属于当前 WebSocket 接入源")
	}
	if currentID != "" && trimmed == currentID {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "上级分组不能指向自身")
	}
	return &trimmed, nil
}

func (s *WebSocketWorkbenchService) normalizeSessionGroup(ctx context.Context, projectID, connectionID string, groupID *string) (*string, error) {
	if groupID == nil || strings.TrimSpace(*groupID) == "" {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*groupID)
	if err := validateUUIDText(trimmed, "groupId 格式无效"); err != nil {
		return nil, err
	}
	group, err := s.repository.GetGroup(ctx, projectID, trimmed)
	if err != nil {
		return nil, err
	}
	if group.ConnectionID != connectionID {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "会话分组不属于当前 WebSocket 接入源")
	}
	return &trimmed, nil
}

func (s *WebSocketWorkbenchService) loadWebSocketConnection(ctx context.Context, projectID, connectionID string) (*repository.ConnectionRecord, error) {
	if s.connections == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "WebSocket 工作台连接仓储未初始化")
	}
	connection, err := s.connections.GetByProjectAndID(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(connection.Type) != "websocket" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "接入源类型不是 WebSocket")
	}
	return connection, nil
}

func resolveWebSocketURL(config map[string]any, sessionURL string) (string, error) {
	raw := strings.TrimSpace(sessionURL)
	parsed, err := url.Parse(raw)
	if err == nil && parsed.IsAbs() {
		if parsed.Scheme == "ws" || parsed.Scheme == "wss" {
			return parsed.String(), nil
		}
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "WebSocket URL 必须以 ws:// 或 wss:// 开头")
	}
	baseURL := strings.TrimSpace(toString(config["url"]))
	if baseURL == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "WebSocket 接入源 URL 不能为空")
	}
	base, err := url.Parse(baseURL)
	if err != nil || !base.IsAbs() {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "WebSocket 接入源 URL 格式无效")
	}
	relative, err := url.Parse(raw)
	if err != nil {
		return "", apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "WebSocket 会话 URL 格式无效", err)
	}
	return base.ResolveReference(relative).String(), nil
}

func applyWebSocketAuth(headers http.Header, auth map[string]any) error {
	authType := strings.ToLower(strings.TrimSpace(toString(auth["type"])))
	switch authType {
	case "", "none":
		return nil
	case "bearer":
		token := strings.TrimSpace(toString(auth["token"]))
		if token == "" {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Bearer Token 不能为空")
		}
		headers.Set("Authorization", "Bearer "+token)
	case "basic":
		username := toString(auth["username"])
		password := toString(auth["password"])
		headers.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
	default:
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "WebSocket Auth 类型不受支持")
	}
	return nil
}

func buildWebSocketPreviewMessage(direction string, messageType int, payload []byte) WebSocketPreviewMessage {
	raw := string(payload)
	return WebSocketPreviewMessage{
		Direction:  direction,
		Type:       webSocketMessageTypeName(messageType),
		Payload:    decodeWebSocketPayload(payload),
		RawPayload: raw,
		SizeBytes:  len(payload),
		Timestamp:  time.Now().UTC(),
	}
}

func decodeWebSocketPayload(payload []byte) any {
	trimmed := strings.TrimSpace(string(payload))
	if trimmed == "" {
		return ""
	}
	var decoded any
	if (strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")) && json.Unmarshal(payload, &decoded) == nil {
		return decoded
	}
	return string(payload)
}

func webSocketMessageTypeName(messageType int) string {
	switch messageType {
	case websocket.TextMessage:
		return "text"
	case websocket.BinaryMessage:
		return "binary"
	case websocket.CloseMessage:
		return "close"
	case websocket.PingMessage:
		return "ping"
	case websocket.PongMessage:
		return "pong"
	default:
		return "unknown"
	}
}

func receivedMessages(messages []WebSocketPreviewMessage) []WebSocketPreviewMessage {
	result := make([]WebSocketPreviewMessage, 0, len(messages))
	for _, message := range messages {
		if message.Direction == "in" {
			result = append(result, message)
		}
	}
	return result
}

func enabledMessageRows(rows []any) []string {
	result := make([]string, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		if enabled, exists := item["enabled"]; exists && !boolFromAny(enabled) {
			continue
		}
		payload := strings.TrimSpace(toString(item["payload"]))
		if payload != "" {
			result = append(result, payload)
		}
	}
	return result
}

func enabledProtocolValues(rows []any) []string {
	result := make([]string, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		if enabled, exists := item["enabled"]; exists && !boolFromAny(enabled) {
			continue
		}
		value := strings.TrimSpace(toString(item["value"]))
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func webSocketTLSConfig(settings map[string]any) *tls.Config {
	tlsVerify := true
	if value, ok := settings["tlsVerify"]; ok {
		tlsVerify = boolFromAny(value)
	}
	if tlsVerify {
		return nil
	}
	// 开发态短连接允许临时关闭 TLS 校验，便于调试自签名服务。
	return &tls.Config{InsecureSkipVerify: true} //nolint:gosec
}

func webSocketTimeoutMS(settings, config map[string]any) int {
	timeout := intFromAny(settings["timeoutMs"], 0)
	if timeout <= 0 {
		timeout = intFromAny(config["timeoutMs"], defaultWebSocketTimeoutMS)
	}
	if timeout < 1000 {
		return 1000
	}
	if timeout > maxWebSocketTimeoutMS {
		return maxWebSocketTimeoutMS
	}
	return timeout
}

func normalizeWebSocketSessionCore(name, sessionURL string) (string, string, error) {
	normalizedName, err := normalizeWebSocketRequiredText(name, 100, "WebSocket 会话名称不能为空")
	if err != nil {
		return "", "", err
	}
	normalizedURL, err := normalizeWebSocketRequiredText(sessionURL, 2000, "WebSocket 会话 URL 不能为空")
	if err != nil {
		return "", "", err
	}
	return normalizedName, normalizedURL, nil
}

func normalizeWebSocketRequiredText(value string, maxLen int, message string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, message)
	}
	if len([]rune(trimmed)) > maxLen {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fmt.Sprintf("字段长度不能超过 %d 个字符", maxLen))
	}
	return trimmed, nil
}

func normalizeWebSocketAuth(auth map[string]any) map[string]any {
	if auth == nil {
		return map[string]any{"type": "none"}
	}
	result := cloneMap(auth)
	authType := strings.ToLower(strings.TrimSpace(toString(result["type"])))
	if authType == "" {
		authType = "none"
	}
	result["type"] = authType
	return result
}

func normalizeProtocolRows(rows []any) []any {
	if rows == nil {
		return []any{}
	}
	result := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, map[string]any{
			"enabled":     boolFromAnyWithDefault(item["enabled"], true),
			"value":       strings.TrimSpace(toString(item["value"])),
			"description": toString(item["description"]),
		})
	}
	return result
}

func normalizeProtocolRowsOrDefault(rows, fallback []any) []any {
	if rows == nil {
		return fallback
	}
	return normalizeProtocolRows(rows)
}

func normalizeMessageRows(rows []any) []any {
	if rows == nil {
		return []any{}
	}
	result := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, map[string]any{
			"enabled":     boolFromAnyWithDefault(item["enabled"], true),
			"name":        strings.TrimSpace(toString(item["name"])),
			"payload":     toString(item["payload"]),
			"description": toString(item["description"]),
		})
	}
	return result
}

func normalizeMessageRowsOrDefault(rows, fallback []any) []any {
	if rows == nil {
		return fallback
	}
	return normalizeMessageRows(rows)
}

func normalizeWebSocketSettings(settings map[string]any) map[string]any {
	result := cloneMap(settings)
	if _, ok := result["timeoutMs"]; !ok {
		result["timeoutMs"] = defaultWebSocketTimeoutMS
	}
	if _, ok := result["tlsVerify"]; !ok {
		result["tlsVerify"] = true
	}
	if _, ok := result["messageLimit"]; !ok {
		result["messageLimit"] = 5
	}
	return result
}

func webSocketSessionSourceConfig(connection repository.ConnectionRecord, record repository.WebSocketSessionRecord) map[string]any {
	return map[string]any{
		"connectionId": connection.ID,
		"sessionId":    record.ID,
		"url":          record.URL,
	}
}

func buildWebSocketDataPointPath(connectionName, sessionName string) string {
	return "websocket." + normalizeDatapointSegment(connectionName) + "." + normalizeDatapointSegment(sessionName)
}

func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func responseStatusCode(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}

func toWebSocketSessionGroup(record repository.WebSocketSessionGroupRecord) WebSocketSessionGroup {
	return WebSocketSessionGroup{
		ID: record.ID, ProjectID: record.ProjectID, ConnectionID: record.ConnectionID,
		ParentID: cloneOptionalString(record.ParentID), Name: record.Name, SortOrder: record.SortOrder,
		CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt,
	}
}

func toWebSocketSession(record repository.WebSocketSessionRecord) WebSocketSession {
	return WebSocketSession{
		ID: record.ID, ProjectID: record.ProjectID, ConnectionID: record.ConnectionID,
		GroupID: cloneOptionalString(record.GroupID), Name: record.Name, URL: record.URL,
		Headers: cloneJSONArray(record.Headers), Auth: cloneMap(record.Auth),
		Protocols: cloneJSONArray(record.Protocols), Messages: cloneJSONArray(record.Messages),
		Settings: cloneMap(record.Settings), Enabled: record.Enabled, SortOrder: record.SortOrder,
		SourceType: "websocket.session", DataPointID: httpStringValue(record.DataPointID),
		DataPointPath: httpStringValue(record.DataPointPath), LastMessage: record.LastMessage,
		LastDiagnostic: httpStringValue(record.LastDiagnostic), Quality: fallbackTrimmed(record.Quality, "unknown"),
		LastMessageAt: record.LastMessageAt, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt,
	}
}
