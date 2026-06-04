package socket

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	socketio "github.com/ismhdez/socket.io-golang/v4"

	"github.com/indu-forge/data_service/internal/auth"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
	"github.com/indu-forge/data_service/internal/service"
)

const (
	previewSocketMetadataKey      = "previewSessionMetadata"
	previewSocketWriteLockKey     = "previewSocketWriteLock"
	previewSocketRequestEvent     = "response"
	previewSessionSweepInterval   = 15 * time.Second
	previewDatapointPollInterval  = 2 * time.Second
	previewSocketOpTimeout        = 10 * time.Second
	previewSocketSessionCloseWait = 250
	mqttLatestTagSnapshotScan     = 100
)

var previewSocketFallbackWriteMu sync.Mutex

type socketSessionMetadata struct {
	Claims         *auth.Claims
	ProjectID      string
	PreviewSession string
	UserID         string
}

type socketSubscriptionSet struct {
	mqttSubscriptions map[string]struct{}
	mqttTags          map[string]struct{}
	datapoints        map[string]struct{}
}

type mqttRuntimePendingState struct {
	done    chan struct{}
	runtime *mqttPreviewRuntime
	err     error
}

type previewSessionState struct {
	sessionID string
	projectID string
	userID    string

	sockets             map[string]*socketio.Socket
	socketSubscriptions map[string]*socketSubscriptionSet
	tagSubscriptionByID map[string]string
	mqttRuntimes        map[string]*mqttPreviewRuntime
	mqttRuntimePending  map[string]*mqttRuntimePendingState

	datapointFingerprints map[string]string
	datapointPollCancel   context.CancelFunc
}

type mqttPreviewRuntime struct {
	server       *PreviewSocketServer
	sessionID    string
	projectID    string
	connection   repository.MqttConnectionDetailRecord
	subscription repository.MqttSubscriptionRecord

	mu           sync.RWMutex
	client       mqtt.Client
	tags         map[string]repository.MqttTagRecord
	lastMessage  *service.MqttSubscriptionSnapshot
	lastTagValue map[string]service.MqttTagValueSnapshot
}

type datapointValueReader interface {
	GetDataPointValue(ctx context.Context, projectID, path string) (*service.DataPointValue, error)
}

type previewMqttRepository interface {
	GetConnectionDetail(ctx context.Context, projectID, connectionID string) (*repository.MqttConnectionDetailRecord, error)
	GetSubscription(ctx context.Context, projectID, subscriptionID string) (*repository.MqttSubscriptionRecord, error)
	GetTag(ctx context.Context, projectID, tagID string) (*repository.MqttTagRecord, error)
	ListMessages(ctx context.Context, projectID, subscriptionID string, limit int) ([]repository.MqttMessageRecord, error)
	ListMessagesAfter(ctx context.Context, projectID, subscriptionID string, after time.Time, limit int) ([]repository.MqttMessageRecord, error)
	CreateMessage(ctx context.Context, params repository.CreateMqttMessageParams) (*repository.MqttMessageRecord, error)
}

// PreviewSocketServer binds preview sessions to socket.io transport.
type PreviewSocketServer struct {
	io                 *socketio.Io
	jwtValidator       *auth.JWTValidator
	previewService     *service.PreviewSessionService
	dataPoints         datapointValueReader
	mqttRepository     previewMqttRepository
	builtinRuntime     *service.BuiltinRuntimeService
	builtinMessageHub  builtinMessageHubConfig
	mqttRuntimeBuilder func(ctx context.Context, sessionID, projectID, subscriptionID string) (*mqttPreviewRuntime, error)

	mu       sync.Mutex
	sessions map[string]*previewSessionState

	closed    chan struct{}
	closeOnce sync.Once
}

type builtinMessageHubConfig struct {
	Addr     string
	Username string
	Password string
}

// NewPreviewSocketServer creates the in-process preview socket server.
func NewPreviewSocketServer(
	jwtValidator *auth.JWTValidator,
	previewService *service.PreviewSessionService,
	dataPointService *service.DataPointService,
	mqttRepository *repository.MqttRepository,
	builtinRuntime ...*service.BuiltinRuntimeService,
) (*PreviewSocketServer, error) {
	if jwtValidator == nil {
		return nil, fmt.Errorf("preview socket missing JWT validator")
	}
	if previewService == nil {
		return nil, fmt.Errorf("preview socket missing previewSessionService")
	}
	if dataPointService == nil {
		return nil, fmt.Errorf("preview socket missing datapointService")
	}
	if mqttRepository == nil {
		return nil, fmt.Errorf("preview socket missing mqttRepository")
	}

	server := &PreviewSocketServer{
		io:                 socketio.New(),
		jwtValidator:       jwtValidator,
		previewService:     previewService,
		dataPoints:         dataPointService,
		mqttRepository:     mqttRepository,
		builtinRuntime:     firstBuiltinRuntime(builtinRuntime),
		mqttRuntimeBuilder: nil,
		sessions:           make(map[string]*previewSessionState),
		closed:             make(chan struct{}),
	}
	server.mqttRuntimeBuilder = server.buildMqttRuntime
	server.io.OnAuthorization(server.authorizeSocket)
	server.io.OnConnection(server.handleConnection)

	go server.sessionSweeper()
	return server, nil
}

// ConfigureBuiltinMessageHub 配置 IF消息库预览订阅使用的内置 MQTT Broker。
func (s *PreviewSocketServer) ConfigureBuiltinMessageHub(addr, username, password string) {
	if s == nil {
		return
	}
	s.builtinMessageHub = builtinMessageHubConfig{
		Addr:     strings.TrimSpace(addr),
		Username: strings.TrimSpace(username),
		Password: password,
	}
}

// Handler exposes the HTTP handler mounted at /socket.io.
func (s *PreviewSocketServer) Handler() http.Handler {
	if s == nil || s.io == nil {
		return nil
	}
	return s.io.HttpHandler()
}

// CloseSession tears down sockets and runtimes for one preview session.
func (s *PreviewSocketServer) CloseSession(sessionID string) {
	s.closeSessionInternal(strings.TrimSpace(sessionID), "session_closed")
}

// Close releases background resources for the preview socket server.
func (s *PreviewSocketServer) Close() {
	if s == nil {
		return
	}

	s.closeOnce.Do(func() {
		close(s.closed)

		s.mu.Lock()
		sessionIDs := make([]string, 0, len(s.sessions))
		for sessionID := range s.sessions {
			sessionIDs = append(sessionIDs, sessionID)
		}
		s.mu.Unlock()

		for _, sessionID := range sessionIDs {
			s.closeSessionInternal(sessionID, "server_closed")
		}

		if s.io != nil {
			s.io.Close()
		}
	})
}

func (s *PreviewSocketServer) authorizeSocket(socket *socketio.Socket, params map[string]string) (bool, string) {
	if socket == nil {
		return false, "socket not initialized"
	}

	token := normalizeSocketToken(params["token"])
	if token == "" {
		return false, "missing token"
	}

	projectID := strings.TrimSpace(params["projectId"])
	if projectID == "" {
		return false, "missing projectId"
	}

	previewSessionID := strings.TrimSpace(params["previewSessionId"])
	if previewSessionID == "" {
		return false, "missing previewSessionId"
	}

	claims, err := s.jwtValidator.Validate(token)
	if err != nil {
		return false, socketErrorMessage(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), previewSocketOpTimeout)
	defer cancel()

	session, err := s.previewService.AuthorizeSession(ctx, claims, projectID, previewSessionID)
	if err != nil {
		return false, socketErrorMessage(err)
	}

	socket.Metadata(previewSocketMetadataKey, &socketSessionMetadata{
		Claims:         claims,
		ProjectID:      projectID,
		PreviewSession: session.ID,
		UserID:         session.UserID,
	})
	return true, ""
}

func (s *PreviewSocketServer) handleConnection(socket *socketio.Socket) {
	metadata := socketMetadataFromSocket(socket)
	if metadata == nil {
		_ = disconnectSocket(socket)
		return
	}

	socket.Metadata(previewSocketWriteLockKey, &sync.Mutex{})

	s.mu.Lock()
	session := s.sessions[metadata.PreviewSession]
	if session == nil {
		session = &previewSessionState{
			sessionID:             metadata.PreviewSession,
			projectID:             metadata.ProjectID,
			userID:                metadata.UserID,
			sockets:               make(map[string]*socketio.Socket),
			socketSubscriptions:   make(map[string]*socketSubscriptionSet),
			tagSubscriptionByID:   make(map[string]string),
			mqttRuntimes:          make(map[string]*mqttPreviewRuntime),
			mqttRuntimePending:    make(map[string]*mqttRuntimePendingState),
			datapointFingerprints: make(map[string]string),
		}
		s.sessions[metadata.PreviewSession] = session
	}
	session.sockets[socket.Id] = socket
	if session.socketSubscriptions[socket.Id] == nil {
		session.socketSubscriptions[socket.Id] = newSocketSubscriptionSet()
	}
	s.mu.Unlock()

	socket.On("disconnect", func(_ *socketio.EventPayload) {
		s.handleSocketDisconnect(socket)
	})
	socket.On("mqtt:subscribe", func(event *socketio.EventPayload) {
		s.handleMqttSubscribe(socket, event)
	})
	socket.On("mqtt:unsubscribe", func(event *socketio.EventPayload) {
		s.handleMqttUnsubscribe(socket, event)
	})
	socket.On("mqtt:tag:subscribe", func(event *socketio.EventPayload) {
		s.handleMqttTagSubscribe(socket, event)
	})
	socket.On("mqtt:tag:unsubscribe", func(event *socketio.EventPayload) {
		s.handleMqttTagUnsubscribe(socket, event)
	})
	socket.On("datapoint:subscribe", func(event *socketio.EventPayload) {
		s.handleDatapointSubscribe(socket, event)
	})
	socket.On("datapoint:unsubscribe", func(event *socketio.EventPayload) {
		s.handleDatapointUnsubscribe(socket, event)
	})
	socket.On("datapoint:subscribe:batch", func(event *socketio.EventPayload) {
		s.handleDatapointBatchSubscribe(socket, event)
	})
}

func (s *PreviewSocketServer) handleSocketDisconnect(socket *socketio.Socket) {
	metadata := socketMetadataFromSocket(socket)
	if metadata == nil {
		return
	}

	var (
		runtimesToClose []*mqttPreviewRuntime
		pollCancel      context.CancelFunc
	)

	s.mu.Lock()
	session := s.sessions[metadata.PreviewSession]
	if session == nil {
		s.mu.Unlock()
		return
	}

	delete(session.sockets, socket.Id)
	delete(session.socketSubscriptions, socket.Id)

	for path := range session.datapointFingerprints {
		if !session.hasDatapointInterestLocked(path) {
			delete(session.datapointFingerprints, path)
		}
	}
	for tagID := range session.tagSubscriptionByID {
		if !session.hasTagInterestLocked(tagID) {
			delete(session.tagSubscriptionByID, tagID)
		}
	}
	if !session.hasAnyDatapointInterestLocked() && session.datapointPollCancel != nil {
		pollCancel = session.datapointPollCancel
		session.datapointPollCancel = nil
	}

	for subscriptionID, runtime := range session.mqttRuntimes {
		if !session.hasMqttInterestLocked(subscriptionID) {
			delete(session.mqttRuntimes, subscriptionID)
			runtimesToClose = append(runtimesToClose, runtime)
		}
	}

	if len(session.sockets) == 0 {
		for subscriptionID, runtime := range session.mqttRuntimes {
			delete(session.mqttRuntimes, subscriptionID)
			runtimesToClose = append(runtimesToClose, runtime)
		}
		if session.datapointPollCancel != nil {
			pollCancel = session.datapointPollCancel
			session.datapointPollCancel = nil
		}
		delete(s.sessions, metadata.PreviewSession)
	}
	s.mu.Unlock()

	if pollCancel != nil {
		pollCancel()
	}
	for _, runtime := range runtimesToClose {
		runtime.close()
	}
}

func (s *PreviewSocketServer) handleMqttSubscribe(socket *socketio.Socket, event *socketio.EventPayload) {
	payload := firstPayloadMap(event)
	requestID := payloadString(payload, "requestId")
	subscriptionID := payloadString(payload, "subscriptionId")
	if subscriptionID == "" {
		respondSocketRequest(socket, requestID, nil, fmt.Errorf("subscriptionId is required"))
		return
	}

	metadata := socketMetadataFromSocket(socket)
	if metadata == nil {
		respondSocketRequest(socket, requestID, nil, errors.New("socket metadata missing"))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), previewSocketOpTimeout)
	defer cancel()

	runtime, err := s.ensureMqttRuntime(ctx, metadata.PreviewSession, subscriptionID)
	if err != nil {
		respondSocketRequest(socket, requestID, nil, err)
		s.emitSessionEvent(metadata.PreviewSession, "mqtt:subscription:status", map[string]any{
			"subscriptionId": subscriptionID,
			"status":         "error",
			"error":          socketErrorMessage(err),
		})
		return
	}

	s.mu.Lock()
	if session := s.sessions[metadata.PreviewSession]; session != nil {
		subscriptions := session.socketSubscriptions[socket.Id]
		if subscriptions == nil {
			subscriptions = newSocketSubscriptionSet()
			session.socketSubscriptions[socket.Id] = subscriptions
		}
		subscriptions.mqttSubscriptions[subscriptionID] = struct{}{}
	}
	s.mu.Unlock()

	if snapshot := s.loadLatestSubscriptionSnapshot(ctx, metadata.ProjectID, runtime); snapshot != nil {
		_ = emitSocket(socket, "mqtt:message", snapshot)
	}
	s.emitSessionEvent(metadata.PreviewSession, "mqtt:subscription:status", map[string]any{
		"subscriptionId": subscriptionID,
		"status":         "subscribed",
	})
	respondSocketRequest(socket, requestID, map[string]any{"subscriptionId": subscriptionID}, nil)
}

func (s *PreviewSocketServer) handleMqttUnsubscribe(socket *socketio.Socket, event *socketio.EventPayload) {
	payload := firstPayloadMap(event)
	requestID := payloadString(payload, "requestId")
	subscriptionID := payloadString(payload, "subscriptionId")
	if subscriptionID == "" {
		respondSocketRequest(socket, requestID, nil, fmt.Errorf("subscriptionId is required"))
		return
	}

	metadata := socketMetadataFromSocket(socket)
	if metadata == nil {
		respondSocketRequest(socket, requestID, nil, errors.New("socket metadata missing"))
		return
	}

	var runtimesToClose []*mqttPreviewRuntime

	s.mu.Lock()
	session := s.sessions[metadata.PreviewSession]
	if session != nil {
		if subscriptions := session.socketSubscriptions[socket.Id]; subscriptions != nil {
			delete(subscriptions.mqttSubscriptions, subscriptionID)
		}
		if runtime := session.mqttRuntimes[subscriptionID]; runtime != nil && !session.hasMqttInterestLocked(subscriptionID) {
			delete(session.mqttRuntimes, subscriptionID)
			runtimesToClose = append(runtimesToClose, runtime)
		}
	}
	s.mu.Unlock()

	for _, runtime := range runtimesToClose {
		runtime.close()
	}

	s.emitSessionEvent(metadata.PreviewSession, "mqtt:subscription:status", map[string]any{
		"subscriptionId": subscriptionID,
		"status":         "unsubscribed",
	})
	respondSocketRequest(socket, requestID, map[string]any{"subscriptionId": subscriptionID}, nil)
}

func (s *PreviewSocketServer) handleMqttTagSubscribe(socket *socketio.Socket, event *socketio.EventPayload) {
	payload := firstPayloadMap(event)
	requestID := payloadString(payload, "requestId")
	tagID := payloadString(payload, "tagId")
	if tagID == "" {
		respondSocketRequest(socket, requestID, nil, fmt.Errorf("tagId is required"))
		return
	}

	metadata := socketMetadataFromSocket(socket)
	if metadata == nil {
		respondSocketRequest(socket, requestID, nil, errors.New("socket metadata missing"))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), previewSocketOpTimeout)
	defer cancel()

	tag, err := s.mqttRepository.GetTag(ctx, metadata.ProjectID, tagID)
	if err != nil {
		respondSocketRequest(socket, requestID, nil, err)
		return
	}

	runtime, err := s.ensureMqttRuntime(ctx, metadata.PreviewSession, tag.SubscriptionID)
	if err != nil {
		respondSocketRequest(socket, requestID, nil, err)
		return
	}
	runtime.addTag(*tag)

	s.mu.Lock()
	if session := s.sessions[metadata.PreviewSession]; session != nil {
		session.tagSubscriptionByID[tag.ID] = tag.SubscriptionID
		subscriptions := session.socketSubscriptions[socket.Id]
		if subscriptions == nil {
			subscriptions = newSocketSubscriptionSet()
			session.socketSubscriptions[socket.Id] = subscriptions
		}
		subscriptions.mqttTags[tag.ID] = struct{}{}
	}
	s.mu.Unlock()

	if snapshot := s.loadLatestTagSnapshot(ctx, metadata.ProjectID, runtime, *tag); snapshot != nil {
		_ = emitSocket(socket, "mqtt:tag:value", snapshot)
	}
	respondSocketRequest(socket, requestID, map[string]any{"tagId": tag.ID}, nil)
}

func (s *PreviewSocketServer) handleMqttTagUnsubscribe(socket *socketio.Socket, event *socketio.EventPayload) {
	payload := firstPayloadMap(event)
	requestID := payloadString(payload, "requestId")
	tagID := payloadString(payload, "tagId")
	if tagID == "" {
		respondSocketRequest(socket, requestID, nil, fmt.Errorf("tagId is required"))
		return
	}

	metadata := socketMetadataFromSocket(socket)
	if metadata == nil {
		respondSocketRequest(socket, requestID, nil, errors.New("socket metadata missing"))
		return
	}

	var runtimesToClose []*mqttPreviewRuntime

	s.mu.Lock()
	session := s.sessions[metadata.PreviewSession]
	if session != nil {
		if subscriptions := session.socketSubscriptions[socket.Id]; subscriptions != nil {
			delete(subscriptions.mqttTags, tagID)
		}
		subscriptionID := session.tagSubscriptionByID[tagID]
		if !session.hasTagInterestLocked(tagID) {
			delete(session.tagSubscriptionByID, tagID)
			if subscriptionID != "" {
				if runtime := session.mqttRuntimes[subscriptionID]; runtime != nil {
					runtime.removeTag(tagID)
				}
			}
		}
		if subscriptionID != "" {
			if runtime := session.mqttRuntimes[subscriptionID]; runtime != nil && !session.hasMqttInterestLocked(subscriptionID) {
				delete(session.mqttRuntimes, subscriptionID)
				runtimesToClose = append(runtimesToClose, runtime)
			}
		}
	}
	s.mu.Unlock()

	for _, runtime := range runtimesToClose {
		runtime.close()
	}
	respondSocketRequest(socket, requestID, map[string]any{"tagId": tagID}, nil)
}

func (s *PreviewSocketServer) handleDatapointSubscribe(socket *socketio.Socket, event *socketio.EventPayload) {
	payload := firstPayloadMap(event)
	requestID := payloadString(payload, "requestId")
	path := payloadString(payload, "path")
	if path == "" {
		respondSocketRequest(socket, requestID, nil, fmt.Errorf("path is required"))
		return
	}

	metadata := socketMetadataFromSocket(socket)
	if metadata == nil {
		respondSocketRequest(socket, requestID, nil, errors.New("socket metadata missing"))
		return
	}

	s.registerDatapointSubscription(metadata.PreviewSession, socket.Id, path)
	if err := s.ensureDatapointPoller(metadata.PreviewSession); err != nil {
		respondSocketRequest(socket, requestID, nil, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), previewSocketOpTimeout)
	defer cancel()
	value, err := s.dataPoints.GetDataPointValue(ctx, metadata.ProjectID, path)
	if err != nil {
		_ = emitSocket(socket, "datapoint:status", map[string]any{
			"path":   path,
			"status": "error",
			"error":  socketErrorMessage(err),
		})
		respondSocketRequest(socket, requestID, nil, err)
		return
	}

	payloadValue := buildDatapointPayload(*value)
	s.updateDatapointFingerprint(metadata.PreviewSession, path, payloadValue)
	_ = emitSocket(socket, "datapoint:value", payloadValue)
	_ = emitSocket(socket, "datapoint:status", map[string]any{
		"path":   path,
		"status": "subscribed",
	})
	respondSocketRequest(socket, requestID, map[string]any{"path": path}, nil)
}

func (s *PreviewSocketServer) handleDatapointUnsubscribe(socket *socketio.Socket, event *socketio.EventPayload) {
	payload := firstPayloadMap(event)
	requestID := payloadString(payload, "requestId")
	path := payloadString(payload, "path")
	if path == "" {
		respondSocketRequest(socket, requestID, nil, fmt.Errorf("path is required"))
		return
	}

	metadata := socketMetadataFromSocket(socket)
	if metadata == nil {
		respondSocketRequest(socket, requestID, nil, errors.New("socket metadata missing"))
		return
	}

	s.unregisterDatapointSubscription(metadata.PreviewSession, socket.Id, path)
	respondSocketRequest(socket, requestID, map[string]any{"path": path}, nil)
}

func (s *PreviewSocketServer) handleDatapointBatchSubscribe(socket *socketio.Socket, event *socketio.EventPayload) {
	payload := firstPayloadMap(event)
	requestID := payloadString(payload, "requestId")
	paths := uniqueStrings(payloadStringSlice(payload, "paths"))
	if len(paths) == 0 {
		respondSocketRequest(socket, requestID, nil, fmt.Errorf("paths is required"))
		return
	}

	metadata := socketMetadataFromSocket(socket)
	if metadata == nil {
		respondSocketRequest(socket, requestID, nil, errors.New("socket metadata missing"))
		return
	}

	for _, path := range paths {
		s.registerDatapointSubscription(metadata.PreviewSession, socket.Id, path)
	}
	if err := s.ensureDatapointPoller(metadata.PreviewSession); err != nil {
		respondSocketRequest(socket, requestID, nil, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), previewSocketOpTimeout)
	defer cancel()

	values := make([]map[string]any, 0, len(paths))
	for _, path := range paths {
		value, err := s.dataPoints.GetDataPointValue(ctx, metadata.ProjectID, path)
		if err != nil {
			_ = emitSocket(socket, "datapoint:status", map[string]any{
				"path":   path,
				"status": "error",
				"error":  socketErrorMessage(err),
			})
			continue
		}
		payloadValue := buildDatapointPayload(*value)
		s.updateDatapointFingerprint(metadata.PreviewSession, path, payloadValue)
		values = append(values, payloadValue)
	}

	if len(values) > 0 {
		_ = emitSocket(socket, "datapoint:values", values)
	}
	respondSocketRequest(socket, requestID, map[string]any{"paths": paths}, nil)
}

func (s *PreviewSocketServer) registerDatapointSubscription(sessionID, socketID, path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session := s.sessions[sessionID]
	if session == nil {
		return
	}
	subscriptions := session.socketSubscriptions[socketID]
	if subscriptions == nil {
		subscriptions = newSocketSubscriptionSet()
		session.socketSubscriptions[socketID] = subscriptions
	}
	subscriptions.datapoints[path] = struct{}{}
}

func (s *PreviewSocketServer) unregisterDatapointSubscription(sessionID, socketID, path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}

	var pollCancel context.CancelFunc

	s.mu.Lock()
	session := s.sessions[sessionID]
	if session != nil {
		if subscriptions := session.socketSubscriptions[socketID]; subscriptions != nil {
			delete(subscriptions.datapoints, path)
		}
		if !session.hasDatapointInterestLocked(path) {
			delete(session.datapointFingerprints, path)
		}
		if !session.hasAnyDatapointInterestLocked() && session.datapointPollCancel != nil {
			pollCancel = session.datapointPollCancel
			session.datapointPollCancel = nil
		}
	}
	s.mu.Unlock()

	if pollCancel != nil {
		pollCancel()
	}
}

func (s *PreviewSocketServer) ensureDatapointPoller(sessionID string) error {
	s.mu.Lock()
	session := s.sessions[sessionID]
	if session == nil {
		s.mu.Unlock()
		return fmt.Errorf("preview session not found")
	}
	if session.datapointPollCancel != nil {
		s.mu.Unlock()
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	session.datapointPollCancel = cancel
	projectID := session.projectID
	s.mu.Unlock()

	go s.runDatapointPoller(ctx, sessionID, projectID)
	return nil
}

func (s *PreviewSocketServer) runDatapointPoller(ctx context.Context, sessionID, projectID string) {
	ticker := time.NewTicker(previewDatapointPollInterval)
	defer ticker.Stop()

	s.pollDatapoints(sessionID, projectID)

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.closed:
			return
		case <-ticker.C:
			s.pollDatapoints(sessionID, projectID)
		}
	}
}

func (s *PreviewSocketServer) pollDatapoints(sessionID, projectID string) {
	paths := s.snapshotDatapointPaths(sessionID)
	if len(paths) == 0 {
		return
	}

	for _, path := range paths {
		ctx, cancel := context.WithTimeout(context.Background(), previewSocketOpTimeout)
		value, err := s.dataPoints.GetDataPointValue(ctx, projectID, path)
		cancel()
		if err != nil {
			s.emitDatapointStatus(sessionID, path, "error", socketErrorMessage(err))
			continue
		}

		payload := buildDatapointPayload(*value)
		if !s.shouldEmitDatapointValue(sessionID, path, payload) {
			continue
		}
		s.emitDatapointValue(sessionID, path, payload)
	}
}

func (s *PreviewSocketServer) snapshotDatapointPaths(sessionID string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := s.sessions[sessionID]
	if session == nil {
		return nil
	}

	paths := make([]string, 0)
	seen := make(map[string]struct{})
	for _, subscriptions := range session.socketSubscriptions {
		for path := range subscriptions.datapoints {
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			paths = append(paths, path)
		}
	}
	return paths
}

func (s *PreviewSocketServer) shouldEmitDatapointValue(sessionID, path string, payload map[string]any) bool {
	fingerprint := fmt.Sprintf("%v|%v|%v|%v", payload["value"], payload["quality"], payload["timestamp"], payload["status"])

	s.mu.Lock()
	defer s.mu.Unlock()

	session := s.sessions[sessionID]
	if session == nil {
		return false
	}
	if session.datapointFingerprints[path] == fingerprint {
		return false
	}
	session.datapointFingerprints[path] = fingerprint
	return true
}

func (s *PreviewSocketServer) updateDatapointFingerprint(sessionID, path string, payload map[string]any) {
	fingerprint := fmt.Sprintf("%v|%v|%v|%v", payload["value"], payload["quality"], payload["timestamp"], payload["status"])

	s.mu.Lock()
	defer s.mu.Unlock()
	if session := s.sessions[sessionID]; session != nil {
		session.datapointFingerprints[path] = fingerprint
	}
}

func (s *PreviewSocketServer) emitDatapointValue(sessionID, path string, payload map[string]any) {
	recipients := s.datapointRecipients(sessionID, path)
	for _, socket := range recipients {
		_ = emitSocket(socket, "datapoint:value", payload)
	}
}

func (s *PreviewSocketServer) emitDatapointStatus(sessionID, path, status, errorMessage string) {
	recipients := s.datapointRecipients(sessionID, path)
	payload := map[string]any{
		"path":   path,
		"status": status,
	}
	if errorMessage != "" {
		payload["error"] = errorMessage
	}
	for _, socket := range recipients {
		_ = emitSocket(socket, "datapoint:status", payload)
	}
}

func (s *PreviewSocketServer) datapointRecipients(sessionID, path string) []*socketio.Socket {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := s.sessions[sessionID]
	if session == nil {
		return nil
	}

	recipients := make([]*socketio.Socket, 0)
	for socketID, subscriptions := range session.socketSubscriptions {
		if _, ok := subscriptions.datapoints[path]; !ok {
			continue
		}
		if socket := session.sockets[socketID]; socket != nil {
			recipients = append(recipients, socket)
		}
	}
	return recipients
}

func (s *PreviewSocketServer) ensureMqttRuntime(ctx context.Context, sessionID, subscriptionID string) (*mqttPreviewRuntime, error) {
	for {
		s.mu.Lock()
		session := s.sessions[sessionID]
		if session == nil {
			s.mu.Unlock()
			return nil, fmt.Errorf("preview session not found")
		}

		if runtime := session.mqttRuntimes[subscriptionID]; runtime != nil {
			s.mu.Unlock()
			return runtime, nil
		}

		if pending := session.mqttRuntimePending[subscriptionID]; pending != nil {
			s.mu.Unlock()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-pending.done:
				if pending.err != nil {
					return nil, pending.err
				}
				if pending.runtime == nil {
					return nil, fmt.Errorf("MQTT runtime build completed without result")
				}
				return pending.runtime, nil
			}
		}

		pending := &mqttRuntimePendingState{done: make(chan struct{})}
		session.mqttRuntimePending[subscriptionID] = pending
		projectID := session.projectID
		s.mu.Unlock()

		builder := s.mqttRuntimeBuilder
		if builder == nil {
			builder = s.buildMqttRuntime
		}
		runtime, err := builder(ctx, sessionID, projectID, subscriptionID)

		s.mu.Lock()
		session = s.sessions[sessionID]
		if session != nil {
			if err == nil && runtime != nil {
				session.mqttRuntimes[subscriptionID] = runtime
			}
			pending.runtime = runtime
			pending.err = err
			close(pending.done)
			delete(session.mqttRuntimePending, subscriptionID)
		}
		s.mu.Unlock()

		if session == nil {
			if runtime != nil {
				runtime.close()
			}
			return nil, fmt.Errorf("preview session closed")
		}
		if err != nil {
			return nil, err
		}
		return runtime, nil
	}
}

func (s *PreviewSocketServer) buildMqttRuntime(ctx context.Context, sessionID, projectID, subscriptionID string) (*mqttPreviewRuntime, error) {
	subscription, err := s.mqttRepository.GetSubscription(ctx, projectID, subscriptionID)
	if err != nil {
		return nil, err
	}

	connection, err := s.mqttRepository.GetConnectionDetail(ctx, projectID, subscription.ConnectionID)
	if err != nil {
		return nil, err
	}

	runtime := &mqttPreviewRuntime{
		server:       s,
		sessionID:    sessionID,
		projectID:    projectID,
		connection:   *connection,
		subscription: *subscription,
		tags:         make(map[string]repository.MqttTagRecord),
		lastTagValue: make(map[string]service.MqttTagValueSnapshot),
	}

	if err := runtime.start(); err != nil {
		return nil, err
	}
	return runtime, nil
}

func (r *mqttPreviewRuntime) start() error {
	brokerURL, username, password, err := resolvePreviewMqttBroker(r.connection, r.server.builtinMessageHub)
	if err != nil {
		return err
	}

	options := mqtt.NewClientOptions()
	options.AddBroker(brokerURL)
	options.SetClientID(buildPreviewMqttClientID(r.sessionID, r.subscription.ID, r.connection.ClientID))
	options.SetOrderMatters(false)
	options.SetAutoReconnect(true)
	options.SetConnectRetry(true)
	options.SetCleanSession(resolveMqttCleanSession(r.connection))
	options.SetKeepAlive(resolveMqttKeepalive(r.connection))
	options.SetConnectRetryInterval(resolveMqttReconnectInterval(r.connection))
	options.SetConnectTimeout(resolveMqttConnectTimeout(r.connection))
	if username != nil {
		options.SetUsername(strings.TrimSpace(*username))
	}
	if password != nil {
		options.SetPassword(strings.TrimSpace(*password))
	}
	options.OnConnectionLost = func(_ mqtt.Client, lostErr error) {
		r.server.emitSessionEvent(r.sessionID, "mqtt:connection:status", map[string]any{
			"connectionId":   r.connection.ID,
			"subscriptionId": r.subscription.ID,
			"status":         "error",
			"error":          socketErrorMessage(lostErr),
		})
	}
	options.OnConnect = func(client mqtt.Client) {
		r.server.emitSessionEvent(r.sessionID, "mqtt:connection:status", map[string]any{
			"connectionId":   r.connection.ID,
			"subscriptionId": r.subscription.ID,
			"status":         "connected",
		})

		token := client.Subscribe(r.subscription.Topic, byte(r.subscription.QOS), func(_ mqtt.Client, message mqtt.Message) {
			go r.handleMessage(message)
		})
		if !token.WaitTimeout(previewSocketOpTimeout) {
			r.server.emitSessionEvent(r.sessionID, "mqtt:subscription:status", map[string]any{
				"subscriptionId": r.subscription.ID,
				"status":         "error",
				"error":          "MQTT subscribe timeout",
			})
			return
		}
		if err := token.Error(); err != nil {
			r.server.emitSessionEvent(r.sessionID, "mqtt:subscription:status", map[string]any{
				"subscriptionId": r.subscription.ID,
				"status":         "error",
				"error":          socketErrorMessage(err),
			})
			return
		}

		r.server.emitSessionEvent(r.sessionID, "mqtt:subscription:status", map[string]any{
			"subscriptionId": r.subscription.ID,
			"status":         "subscribed",
		})
	}

	client := mqtt.NewClient(options)
	token := client.Connect()
	if !token.WaitTimeout(previewSocketOpTimeout) {
		return fmt.Errorf("MQTT connect timeout")
	}
	if err := token.Error(); err != nil {
		return err
	}

	r.mu.Lock()
	r.client = client
	r.mu.Unlock()
	return nil
}

func (r *mqttPreviewRuntime) close() {
	r.mu.Lock()
	client := r.client
	r.client = nil
	r.mu.Unlock()

	if client != nil && client.IsConnected() {
		client.Disconnect(previewSocketSessionCloseWait)
	}

	r.server.emitSessionEvent(r.sessionID, "mqtt:connection:status", map[string]any{
		"connectionId":   r.connection.ID,
		"subscriptionId": r.subscription.ID,
		"status":         "disconnected",
	})
}

func (r *mqttPreviewRuntime) addTag(tag repository.MqttTagRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tags[tag.ID] = tag
}

func (r *mqttPreviewRuntime) removeTag(tagID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tags, tagID)
	delete(r.lastTagValue, tagID)
}

func (r *mqttPreviewRuntime) latestMessageSnapshot() *service.MqttSubscriptionSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.lastMessage == nil {
		return nil
	}
	snapshot := *r.lastMessage
	return &snapshot
}

func (r *mqttPreviewRuntime) latestTagSnapshot(tagID string) *service.MqttTagValueSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	snapshot, ok := r.lastTagValue[tagID]
	if !ok {
		return nil
	}
	next := snapshot
	return &next
}

func (r *mqttPreviewRuntime) handleMessage(message mqtt.Message) {
	receivedAt := time.Now().UTC()
	record, persistErr := r.persistMessage(repository.CreateMqttMessageParams{
		ProjectID:      r.projectID,
		ConnectionID:   r.connection.ID,
		SubscriptionID: r.subscription.ID,
		Topic:          message.Topic(),
		Payload:        string(message.Payload()),
		QOS:            int(message.Qos()),
		ReceivedAt:     receivedAt,
		Metadata: map[string]any{
			"source":    "preview_socket",
			"sessionId": r.sessionID,
		},
		RetentionLimit: r.subscription.MessageRetention,
	})
	if persistErr != nil {
		log.Printf("warning: preview MQTT message persist failed: %v", persistErr)
		record = &repository.MqttMessageRecord{
			SubscriptionID: r.subscription.ID,
			Topic:          message.Topic(),
			Payload:        string(message.Payload()),
			QOS:            int(message.Qos()),
			ReceivedAt:     receivedAt,
		}
	}

	subscriptionSnapshot := service.BuildMqttSubscriptionSnapshot(*record)

	r.mu.Lock()
	r.lastMessage = &subscriptionSnapshot
	tags := make([]repository.MqttTagRecord, 0, len(r.tags))
	for _, tag := range r.tags {
		tags = append(tags, tag)
	}
	tagSnapshots := make([]service.MqttTagValueSnapshot, 0, len(tags))
	for _, tag := range tags {
		snapshot, ok := service.BuildMqttTagSnapshotUpdateFromMessage(tag, *record)
		if !ok {
			continue
		}
		r.lastTagValue[tag.ID] = snapshot
		tagSnapshots = append(tagSnapshots, snapshot)
	}
	r.mu.Unlock()

	r.server.emitMqttMessage(r.sessionID, subscriptionSnapshot)
	for _, snapshot := range tagSnapshots {
		r.server.emitMqttTagValue(r.sessionID, snapshot)
	}
}

func (r *mqttPreviewRuntime) persistMessage(params repository.CreateMqttMessageParams) (*repository.MqttMessageRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), previewSocketOpTimeout)
	defer cancel()
	return r.server.mqttRepository.CreateMessage(ctx, params)
}

func (s *PreviewSocketServer) loadLatestSubscriptionSnapshot(ctx context.Context, projectID string, runtime *mqttPreviewRuntime) *service.MqttSubscriptionSnapshot {
	if runtime != nil {
		if snapshot := runtime.latestMessageSnapshot(); snapshot != nil {
			return snapshot
		}
	}

	messages, err := s.mqttRepository.ListMessages(ctx, projectID, runtime.subscription.ID, 1)
	if err != nil || len(messages) == 0 {
		return nil
	}
	snapshot := service.BuildMqttSubscriptionSnapshot(messages[0])
	return &snapshot
}

func (s *PreviewSocketServer) loadLatestTagSnapshot(ctx context.Context, projectID string, runtime *mqttPreviewRuntime, tag repository.MqttTagRecord) *service.MqttTagValueSnapshot {
	if runtime != nil {
		if snapshot := runtime.latestTagSnapshot(tag.ID); snapshot != nil {
			return snapshot
		}
	}

	messages, err := s.mqttRepository.ListMessagesAfter(ctx, projectID, tag.SubscriptionID, tag.CreatedAt, mqttLatestTagSnapshotScan)
	if err != nil {
		return nil
	}
	if len(messages) == 0 {
		return nil
	}
	for _, message := range messages {
		snapshot, ok := service.BuildMqttTagSnapshotUpdateFromMessage(tag, message)
		if ok {
			return &snapshot
		}
	}
	return nil
}

func (s *PreviewSocketServer) emitMqttMessage(sessionID string, snapshot service.MqttSubscriptionSnapshot) {
	recipients := s.mqttSubscriptionRecipients(sessionID, snapshot.SubscriptionID)
	for _, socket := range recipients {
		_ = emitSocket(socket, "mqtt:message", snapshot)
	}
}

func (s *PreviewSocketServer) emitMqttTagValue(sessionID string, snapshot service.MqttTagValueSnapshot) {
	recipients := s.mqttTagRecipients(sessionID, snapshot.TagID)
	for _, socket := range recipients {
		_ = emitSocket(socket, "mqtt:tag:value", snapshot)
	}
}

func (s *PreviewSocketServer) emitSessionEvent(sessionID, event string, payload any) {
	recipients := s.sessionRecipients(sessionID)
	for _, socket := range recipients {
		_ = emitSocket(socket, event, payload)
	}
}

func (s *PreviewSocketServer) sessionRecipients(sessionID string) []*socketio.Socket {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := s.sessions[sessionID]
	if session == nil {
		return nil
	}

	recipients := make([]*socketio.Socket, 0, len(session.sockets))
	for _, socket := range session.sockets {
		recipients = append(recipients, socket)
	}
	return recipients
}

func (s *PreviewSocketServer) mqttSubscriptionRecipients(sessionID, subscriptionID string) []*socketio.Socket {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := s.sessions[sessionID]
	if session == nil {
		return nil
	}

	recipients := make([]*socketio.Socket, 0)
	for socketID, subscriptions := range session.socketSubscriptions {
		if _, ok := subscriptions.mqttSubscriptions[subscriptionID]; !ok {
			continue
		}
		if socket := session.sockets[socketID]; socket != nil {
			recipients = append(recipients, socket)
		}
	}
	return recipients
}

func (s *PreviewSocketServer) mqttTagRecipients(sessionID, tagID string) []*socketio.Socket {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := s.sessions[sessionID]
	if session == nil {
		return nil
	}

	recipients := make([]*socketio.Socket, 0)
	for socketID, subscriptions := range session.socketSubscriptions {
		if _, ok := subscriptions.mqttTags[tagID]; !ok {
			continue
		}
		if socket := session.sockets[socketID]; socket != nil {
			recipients = append(recipients, socket)
		}
	}
	return recipients
}

func (s *PreviewSocketServer) closeSessionInternal(sessionID, reason string) {
	if sessionID == "" {
		return
	}

	var (
		sockets       []*socketio.Socket
		runtimes      []*mqttPreviewRuntime
		datapointStop context.CancelFunc
	)

	s.mu.Lock()
	session := s.sessions[sessionID]
	if session == nil {
		s.mu.Unlock()
		return
	}
	delete(s.sessions, sessionID)

	for _, socket := range session.sockets {
		sockets = append(sockets, socket)
	}
	for _, runtime := range session.mqttRuntimes {
		runtimes = append(runtimes, runtime)
	}
	datapointStop = session.datapointPollCancel
	session.datapointPollCancel = nil
	s.mu.Unlock()

	if datapointStop != nil {
		datapointStop()
	}
	for _, runtime := range runtimes {
		runtime.close()
	}
	for _, socket := range sockets {
		if reason != "" {
			_ = emitSocket(socket, "preview:session:closed", map[string]any{
				"sessionId": sessionID,
				"reason":    reason,
			})
		}
		_ = disconnectSocket(socket)
	}
}

func (s *PreviewSocketServer) sessionSweeper() {
	ticker := time.NewTicker(previewSessionSweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.closed:
			return
		case <-ticker.C:
			s.sweepSessions()
		}
	}
}

func (s *PreviewSocketServer) sweepSessions() {
	s.mu.Lock()
	sessionIDs := make([]string, 0, len(s.sessions))
	for sessionID := range s.sessions {
		sessionIDs = append(sessionIDs, sessionID)
	}
	s.mu.Unlock()

	for _, sessionID := range sessionIDs {
		ctx, cancel := context.WithTimeout(context.Background(), previewSocketOpTimeout)
		session, err := s.previewService.GetSession(ctx, sessionID)
		cancel()
		if err != nil {
			s.closeSessionInternal(sessionID, "session_invalid")
			continue
		}
		if session.Status != "active" || (!session.ExpiredAt.IsZero() && time.Now().UTC().After(session.ExpiredAt.UTC())) {
			s.closeSessionInternal(sessionID, "session_expired")
		}
	}
}

func (s *PreviewSocketServer) updateDatapointPathsAfterRuntimeChange(session *previewSessionState) {
	for path := range session.datapointFingerprints {
		if !session.hasDatapointInterestLocked(path) {
			delete(session.datapointFingerprints, path)
		}
	}
}

func (session *previewSessionState) hasMqttInterestLocked(subscriptionID string) bool {
	for _, subscriptions := range session.socketSubscriptions {
		if _, ok := subscriptions.mqttSubscriptions[subscriptionID]; ok {
			return true
		}
		for tagID := range subscriptions.mqttTags {
			if session.tagSubscriptionByID[tagID] == subscriptionID {
				return true
			}
		}
	}
	return false
}

func (session *previewSessionState) hasTagInterestLocked(tagID string) bool {
	for _, subscriptions := range session.socketSubscriptions {
		if _, ok := subscriptions.mqttTags[tagID]; ok {
			return true
		}
	}
	return false
}

func (session *previewSessionState) hasDatapointInterestLocked(path string) bool {
	for _, subscriptions := range session.socketSubscriptions {
		if _, ok := subscriptions.datapoints[path]; ok {
			return true
		}
	}
	return false
}

func (session *previewSessionState) hasAnyDatapointInterestLocked() bool {
	for _, subscriptions := range session.socketSubscriptions {
		if len(subscriptions.datapoints) > 0 {
			return true
		}
	}
	return false
}

func newSocketSubscriptionSet() *socketSubscriptionSet {
	return &socketSubscriptionSet{
		mqttSubscriptions: make(map[string]struct{}),
		mqttTags:          make(map[string]struct{}),
		datapoints:        make(map[string]struct{}),
	}
}

func firstBuiltinRuntime(items []*service.BuiltinRuntimeService) *service.BuiltinRuntimeService {
	if len(items) == 0 {
		return nil
	}
	return items[0]
}

func socketMetadataFromSocket(socket *socketio.Socket) *socketSessionMetadata {
	if socket == nil {
		return nil
	}
	metadata, _ := socket.Metadata(previewSocketMetadataKey).(*socketSessionMetadata)
	return metadata
}

func firstPayloadMap(event *socketio.EventPayload) map[string]any {
	if event == nil || len(event.Data) == 0 || event.Data[0] == nil {
		return map[string]any{}
	}
	payload, ok := event.Data[0].(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return payload
}

func payloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func payloadStringSlice(payload map[string]any, key string) []string {
	if payload == nil {
		return nil
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(fmt.Sprintf("%v", item))
			if text != "" {
				result = append(result, text)
			}
		}
		return result
	default:
		text := strings.TrimSpace(fmt.Sprintf("%v", typed))
		if text == "" {
			return nil
		}
		return []string{text}
	}
}

func respondSocketRequest(socket *socketio.Socket, requestID string, result any, err error) {
	if socket == nil || strings.TrimSpace(requestID) == "" {
		return
	}

	payload := map[string]any{
		"requestId": requestID,
		"success":   err == nil,
	}
	if err != nil {
		payload["error"] = socketErrorMessage(err)
	} else if result != nil {
		payload["result"] = result
	}
	_ = emitSocket(socket, previewSocketRequestEvent, payload)
}

func emitSocket(socket *socketio.Socket, event string, payload any) error {
	if socket == nil {
		return nil
	}

	writeMu, _ := socket.Metadata(previewSocketWriteLockKey).(*sync.Mutex)
	if writeMu == nil {
		// gorilla/websocket 不允许同一连接并发写；异常路径或测试桩没有初始化写锁时，
		// 使用包级兜底锁保证错误响应、关闭通知这类写入也不会触发 panic。
		previewSocketFallbackWriteMu.Lock()
		defer previewSocketFallbackWriteMu.Unlock()
		return socket.Emit(event, payload)
	}

	writeMu.Lock()
	defer writeMu.Unlock()
	return socket.Emit(event, payload)
}

func disconnectSocket(socket *socketio.Socket) error {
	if socket == nil {
		return nil
	}

	writeMu, _ := socket.Metadata(previewSocketWriteLockKey).(*sync.Mutex)
	if writeMu == nil {
		previewSocketFallbackWriteMu.Lock()
		defer previewSocketFallbackWriteMu.Unlock()
		return socket.Disconnect()
	}

	writeMu.Lock()
	defer writeMu.Unlock()
	return socket.Disconnect()
}

func socketErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	var appErr *apperrors.AppError
	if errors.As(err, &appErr) && strings.TrimSpace(appErr.Message) != "" {
		return appErr.Message
	}
	if strings.TrimSpace(err.Error()) == "" {
		return "request failed"
	}
	return err.Error()
}

func normalizeSocketToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}

	if parts := strings.Fields(token); len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}
	return token
}

func buildDatapointPayload(value service.DataPointValue) map[string]any {
	return map[string]any{
		"path":      value.Path,
		"value":     value.Value,
		"quality":   value.Quality,
		"timestamp": value.Timestamp.UTC(),
		"status":    value.Status,
	}
}

func buildMqttBrokerURL(connection repository.MqttConnectionDetailRecord) (string, error) {
	raw := strings.TrimSpace(connection.BrokerURL)
	if raw == "" {
		return "", fmt.Errorf("brokerUrl is required")
	}

	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		if err != nil {
			return "", err
		}
		if parsed.Port() == "" && connection.Port > 0 {
			host := parsed.Hostname()
			if host != "" {
				parsed.Host = net.JoinHostPort(host, strconv.Itoa(connection.Port))
			}
		}
		return parsed.String(), nil
	}

	scheme := "tcp"
	switch strings.ToLower(strings.TrimSpace(connection.Protocol)) {
	case "mqtt":
		scheme = "tcp"
	case "mqtts":
		scheme = "ssl"
	case "ws":
		scheme = "ws"
	case "wss":
		scheme = "wss"
	}

	parsed, err := url.Parse(fmt.Sprintf("%s://%s", scheme, raw))
	if err != nil {
		return "", err
	}
	if parsed.Port() == "" && connection.Port > 0 {
		host := parsed.Hostname()
		if host == "" {
			host = parsed.Host
		}
		parsed.Host = net.JoinHostPort(host, strconv.Itoa(connection.Port))
	}
	return parsed.String(), nil
}

func buildBuiltinMessageBrokerURL(addr string) (string, error) {
	raw := strings.TrimSpace(addr)
	if raw == "" {
		return "", fmt.Errorf("IF消息库 message-hub 地址为空")
	}
	if strings.Contains(raw, "://") {
		return raw, nil
	}
	return "tcp://" + raw, nil
}

func resolvePreviewMqttBroker(connection repository.MqttConnectionDetailRecord, builtin builtinMessageHubConfig) (string, *string, *string, error) {
	if connection.Type == "builtin.message" {
		brokerURL, err := buildBuiltinMessageBrokerURL(builtin.Addr)
		if err != nil {
			return "", nil, nil, err
		}
		return brokerURL, optionalString(builtin.Username), optionalString(builtin.Password), nil
	}
	brokerURL, err := buildMqttBrokerURL(connection)
	if err != nil {
		return "", nil, nil, err
	}
	return brokerURL, connection.Username, connection.Password, nil
}

func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func resolveMqttCleanSession(connection repository.MqttConnectionDetailRecord) bool {
	if connection.Type == "builtin.message" {
		return true
	}
	return connection.CleanSession
}

func resolveMqttKeepalive(connection repository.MqttConnectionDetailRecord) time.Duration {
	if connection.Keepalive <= 0 {
		return 30 * time.Second
	}
	return time.Duration(connection.Keepalive) * time.Second
}

func resolveMqttReconnectInterval(connection repository.MqttConnectionDetailRecord) time.Duration {
	if connection.ReconnectPeriodMS <= 0 {
		return time.Second
	}
	return time.Duration(connection.ReconnectPeriodMS) * time.Millisecond
}

func resolveMqttConnectTimeout(connection repository.MqttConnectionDetailRecord) time.Duration {
	if connection.ConnectTimeoutMS <= 0 {
		return 5 * time.Second
	}
	return time.Duration(connection.ConnectTimeoutMS) * time.Millisecond
}

func buildPreviewMqttClientID(sessionID, subscriptionID string, configured *string) string {
	base := "preview"
	if configured != nil && strings.TrimSpace(*configured) != "" {
		base = strings.TrimSpace(*configured)
	}

	sessionPart := sessionID
	if len(sessionPart) > 8 {
		sessionPart = sessionPart[:8]
	}
	subscriptionPart := subscriptionID
	if len(subscriptionPart) > 8 {
		subscriptionPart = subscriptionPart[:8]
	}
	return fmt.Sprintf("%s-preview-%s-%s", base, sessionPart, subscriptionPart)
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}
