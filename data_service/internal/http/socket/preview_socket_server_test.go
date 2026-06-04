package socket

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	socketio "github.com/ismhdez/socket.io-golang/v4"

	"github.com/indu-forge/data_service/internal/auth"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
	"github.com/indu-forge/data_service/internal/service"
)

func TestNewPreviewSocketServer_RejectsMissingDependencies(t *testing.T) {
	tests := []struct {
		name          string
		jwtValidator  *auth.JWTValidator
		preview       *service.PreviewSessionService
		datapoints    *service.DataPointService
		mqttRepo      *repository.MqttRepository
		expectedError string
	}{
		{
			name:          "missing jwt validator",
			preview:       &service.PreviewSessionService{},
			datapoints:    &service.DataPointService{},
			mqttRepo:      &repository.MqttRepository{},
			expectedError: "preview socket missing JWT validator",
		},
		{
			name:          "missing preview service",
			jwtValidator:  &auth.JWTValidator{},
			datapoints:    &service.DataPointService{},
			mqttRepo:      &repository.MqttRepository{},
			expectedError: "preview socket missing previewSessionService",
		},
		{
			name:          "missing datapoint service",
			jwtValidator:  &auth.JWTValidator{},
			preview:       &service.PreviewSessionService{},
			mqttRepo:      &repository.MqttRepository{},
			expectedError: "preview socket missing datapointService",
		},
		{
			name:          "missing mqtt repository",
			jwtValidator:  &auth.JWTValidator{},
			preview:       &service.PreviewSessionService{},
			datapoints:    &service.DataPointService{},
			expectedError: "preview socket missing mqttRepository",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server, err := NewPreviewSocketServer(tc.jwtValidator, tc.preview, tc.datapoints, tc.mqttRepo)
			if err == nil {
				t.Fatalf("expected constructor error %q, got nil", tc.expectedError)
			}
			if err.Error() != tc.expectedError {
				t.Fatalf("expected error %q, got %q", tc.expectedError, err.Error())
			}
			if server != nil {
				t.Fatal("expected server to be nil when dependencies are incomplete")
			}
		})
	}
}

func TestResolvePreviewMqttBrokerUsesBuiltinHubForBuiltinMessage(t *testing.T) {
	brokerURL, username, password, err := resolvePreviewMqttBroker(repository.MqttConnectionDetailRecord{
		Type:      "builtin.message",
		BrokerURL: "",
	}, builtinMessageHubConfig{
		Addr:     "127.0.0.1:18883",
		Username: "user",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("resolve builtin broker failed: %v", err)
	}
	if brokerURL != "tcp://127.0.0.1:18883" {
		t.Fatalf("unexpected broker url %q", brokerURL)
	}
	if username == nil || *username != "user" {
		t.Fatalf("unexpected username %#v", username)
	}
	if password == nil || *password != "secret" {
		t.Fatalf("unexpected password %#v", password)
	}
}

func TestPreviewSocketServer_AuthorizeSocketRejectsInvalidHandshake(t *testing.T) {
	server := &PreviewSocketServer{}

	tests := []struct {
		name          string
		socket        *socketio.Socket
		params        map[string]string
		expectedOK    bool
		expectedError string
	}{
		{
			name:          "nil socket",
			socket:        nil,
			params:        map[string]string{},
			expectedOK:    false,
			expectedError: "socket not initialized",
		},
		{
			name:          "missing token",
			socket:        &socketio.Socket{},
			params:        map[string]string{"projectId": "project-1", "previewSessionId": "session-1"},
			expectedOK:    false,
			expectedError: "missing token",
		},
		{
			name:          "missing project id",
			socket:        &socketio.Socket{},
			params:        map[string]string{"token": "jwt-token", "previewSessionId": "session-1"},
			expectedOK:    false,
			expectedError: "missing projectId",
		},
		{
			name:          "missing preview session id",
			socket:        &socketio.Socket{},
			params:        map[string]string{"token": "jwt-token", "projectId": "project-1"},
			expectedOK:    false,
			expectedError: "missing previewSessionId",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ok, errMessage := server.authorizeSocket(tc.socket, tc.params)
			if ok != tc.expectedOK {
				t.Fatalf("expected ok=%v, got %v", tc.expectedOK, ok)
			}
			if errMessage != tc.expectedError {
				t.Fatalf("expected error %q, got %q", tc.expectedError, errMessage)
			}
		})
	}
}

func TestPreviewSocketServer_RegisterAndUnregisterDatapointSubscription(t *testing.T) {
	cancelCount := 0
	server := &PreviewSocketServer{
		sessions: map[string]*previewSessionState{
			"session-1": {
				socketSubscriptions: map[string]*socketSubscriptionSet{
					"socket-1": newSocketSubscriptionSet(),
				},
				datapointFingerprints: map[string]string{
					"metrics.temperature": "fingerprint-1",
				},
				datapointPollCancel: func() {
					cancelCount++
				},
			},
		},
	}

	server.registerDatapointSubscription("session-1", "socket-1", " metrics.temperature ")

	session := server.sessions["session-1"]
	if session == nil {
		t.Fatal("expected session to exist after registration")
	}
	if _, ok := session.socketSubscriptions["socket-1"].datapoints["metrics.temperature"]; !ok {
		t.Fatal("expected datapoint subscription to be registered")
	}

	server.unregisterDatapointSubscription("session-1", "socket-1", "metrics.temperature")

	if _, ok := session.socketSubscriptions["socket-1"].datapoints["metrics.temperature"]; ok {
		t.Fatal("expected datapoint subscription to be removed")
	}
	if _, ok := session.datapointFingerprints["metrics.temperature"]; ok {
		t.Fatal("expected datapoint fingerprint to be cleared once no socket is interested")
	}
	if cancelCount != 1 {
		t.Fatalf("expected datapoint poller to stop once, got %d", cancelCount)
	}
	if session.datapointPollCancel != nil {
		t.Fatal("expected datapoint poll cancel function to be cleared after unsubscribe")
	}
}

func TestPreviewSocketServer_CloseSessionRemovesSessionAndCancelsPoller(t *testing.T) {
	cancelCount := 0
	server := &PreviewSocketServer{
		sessions: map[string]*previewSessionState{
			"session-1": {
				socketSubscriptions:   map[string]*socketSubscriptionSet{},
				tagSubscriptionByID:   map[string]string{},
				mqttRuntimes:          map[string]*mqttPreviewRuntime{},
				mqttRuntimePending:    map[string]*mqttRuntimePendingState{},
				datapointFingerprints: map[string]string{},
				datapointPollCancel: func() {
					cancelCount++
				},
			},
		},
	}

	server.CloseSession("session-1")

	if _, ok := server.sessions["session-1"]; ok {
		t.Fatal("expected session to be removed after CloseSession")
	}
	if cancelCount != 1 {
		t.Fatalf("expected datapoint poll cancel to run once, got %d", cancelCount)
	}
}

func TestPreviewSocketServer_SweepSessionsClosesInvalidSessions(t *testing.T) {
	cancelCount := 0
	server := &PreviewSocketServer{
		previewService: &service.PreviewSessionService{},
		sessions: map[string]*previewSessionState{
			"session-1": {
				socketSubscriptions:   map[string]*socketSubscriptionSet{},
				tagSubscriptionByID:   map[string]string{},
				mqttRuntimes:          map[string]*mqttPreviewRuntime{},
				mqttRuntimePending:    map[string]*mqttRuntimePendingState{},
				datapointFingerprints: map[string]string{},
				datapointPollCancel: func() {
					cancelCount++
				},
			},
		},
	}

	server.sweepSessions()

	if _, ok := server.sessions["session-1"]; ok {
		t.Fatal("expected invalid preview session to be swept")
	}
	if cancelCount != 1 {
		t.Fatalf("expected session cleanup to stop datapoint poller once, got %d", cancelCount)
	}
}

func TestPayloadString(t *testing.T) {
	payload := map[string]any{
		"text":     "  hello  ",
		"stringer": testStringer("  world  "),
		"number":   42,
	}

	if got := payloadString(payload, "text"); got != "hello" {
		t.Fatalf("expected trimmed string, got %q", got)
	}
	if got := payloadString(payload, "stringer"); got != "world" {
		t.Fatalf("expected stringer value, got %q", got)
	}
	if got := payloadString(payload, "number"); got != "42" {
		t.Fatalf("expected number fallback, got %q", got)
	}
	if got := payloadString(payload, "missing"); got != "" {
		t.Fatalf("expected missing key to return empty string, got %q", got)
	}
}

func TestPayloadStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		payload  map[string]any
		expected []string
	}{
		{
			name:     "string slice is copied",
			payload:  map[string]any{"paths": []string{"a", "b"}},
			expected: []string{"a", "b"},
		},
		{
			name:     "any slice is trimmed",
			payload:  map[string]any{"paths": []any{" a ", "", 9}},
			expected: []string{"a", "9"},
		},
		{
			name:     "single scalar becomes single item",
			payload:  map[string]any{"paths": " only-one "},
			expected: []string{"only-one"},
		},
		{
			name:     "missing key returns nil",
			payload:  map[string]any{},
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := payloadStringSlice(tc.payload, "paths")
			if !reflect.DeepEqual(got, tc.expected) {
				t.Fatalf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}

func TestSocketErrorMessage(t *testing.T) {
	appErr := apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "preview session closed")
	if got := socketErrorMessage(appErr); got != "preview session closed" {
		t.Fatalf("expected app error message, got %q", got)
	}

	if got := socketErrorMessage(errors.New("raw message")); got != "raw message" {
		t.Fatalf("expected raw error message, got %q", got)
	}

	if got := socketErrorMessage(blankError{}); got != "request failed" {
		t.Fatalf("expected fallback error message, got %q", got)
	}
}

func TestNormalizeSocketToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{name: "empty token", token: "   ", expected: ""},
		{name: "raw token", token: "jwt-token", expected: "jwt-token"},
		{name: "bearer token", token: "Bearer jwt-token", expected: "jwt-token"},
		{name: "mixed case bearer token", token: "bearer jwt-token", expected: "jwt-token"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeSocketToken(tc.token); got != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestBuildDatapointPayload(t *testing.T) {
	timestamp := time.Date(2026, 4, 20, 10, 11, 12, 0, time.FixedZone("CST", 8*3600))
	payload := buildDatapointPayload(service.DataPointValue{
		Path:      "metrics.temperature",
		Value:     26.5,
		Quality:   "good",
		Timestamp: timestamp,
		Status:    "active",
	})

	if payload["path"] != "metrics.temperature" {
		t.Fatalf("expected datapoint path, got %#v", payload["path"])
	}
	if payload["value"] != 26.5 {
		t.Fatalf("expected datapoint value, got %#v", payload["value"])
	}
	gotTimestamp, ok := payload["timestamp"].(time.Time)
	if !ok {
		t.Fatalf("expected timestamp to be time.Time, got %T", payload["timestamp"])
	}
	if gotTimestamp.Location() != time.UTC {
		t.Fatalf("expected UTC timestamp, got %v", gotTimestamp.Location())
	}
}

func TestBuildMqttBrokerURL(t *testing.T) {
	t.Run("rejects empty broker url", func(t *testing.T) {
		_, err := buildMqttBrokerURL(repository.MqttConnectionDetailRecord{})
		if err == nil {
			t.Fatal("expected empty brokerUrl to be rejected")
		}
	})

	t.Run("keeps explicit scheme and appends port when missing", func(t *testing.T) {
		got, err := buildMqttBrokerURL(repository.MqttConnectionDetailRecord{
			BrokerURL: "mqtts://broker.example.com",
			Port:      8883,
		})
		if err != nil {
			t.Fatalf("build broker url failed: %v", err)
		}
		if got != "mqtts://broker.example.com:8883" {
			t.Fatalf("expected mqtts url with port, got %q", got)
		}
	})

	t.Run("builds scheme from protocol when raw host is provided", func(t *testing.T) {
		got, err := buildMqttBrokerURL(repository.MqttConnectionDetailRecord{
			BrokerURL: "broker.example.com",
			Protocol:  "mqtts",
			Port:      1884,
		})
		if err != nil {
			t.Fatalf("build broker url failed: %v", err)
		}
		if got != "ssl://broker.example.com:1884" {
			t.Fatalf("expected ssl url from raw host, got %q", got)
		}
	})
}

func TestBuildPreviewMqttClientID(t *testing.T) {
	configured := "forge"
	got := buildPreviewMqttClientID("session-123456789", "subscription-987654321", &configured)
	if got != "forge-preview-session--subscrip" {
		t.Fatalf("unexpected preview mqtt client id: %q", got)
	}

	got = buildPreviewMqttClientID("session-1", "subscription-1", nil)
	if got != "preview-preview-session--subscrip" && got != "preview-preview-session-1-subscrip" {
		t.Fatalf("unexpected default preview mqtt client id: %q", got)
	}
}

func TestUniqueStrings(t *testing.T) {
	got := uniqueStrings([]string{" a ", "", "b", "a", " b ", "c"})
	expected := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected %v, got %v", expected, got)
	}

	if got := uniqueStrings(nil); got != nil {
		t.Fatalf("expected nil input to return nil, got %v", got)
	}
}

func TestPreviewSocketServer_HandleDatapointBatchSubscribeTracksSuccessfulPaths(t *testing.T) {
	reader := &fakeDataPointValueReader{
		values: map[string]*service.DataPointValue{
			"metrics.temperature": {
				Path:      "metrics.temperature",
				Value:     26.5,
				Quality:   "good",
				Timestamp: time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC),
				Status:    "active",
			},
			"metrics.pressure": {
				Path:      "metrics.pressure",
				Value:     101.2,
				Quality:   "good",
				Timestamp: time.Date(2026, 4, 20, 12, 0, 1, 0, time.UTC),
				Status:    "active",
			},
		},
	}
	server := newPreviewSocketServerForTests()
	server.dataPoints = reader
	server.sessions["session-1"].datapointPollCancel = func() {}

	socket := newTestSocket("socket-1", "project-1", "session-1")
	event := &socketio.EventPayload{
		Data: []interface{}{
			map[string]any{
				"requestId": "req-1",
				"paths":     []any{" metrics.temperature ", "metrics.pressure", "metrics.temperature"},
			},
		},
	}

	server.handleDatapointBatchSubscribe(socket, event)

	subscriptions := server.sessions["session-1"].socketSubscriptions["socket-1"].datapoints
	if _, ok := subscriptions["metrics.temperature"]; !ok {
		t.Fatal("expected metrics.temperature to be tracked after batch subscribe")
	}
	if _, ok := subscriptions["metrics.pressure"]; !ok {
		t.Fatal("expected metrics.pressure to be tracked after batch subscribe")
	}
	if len(reader.calls) != 2 {
		t.Fatalf("expected datapoint reader to receive 2 unique paths, got %d", len(reader.calls))
	}
	if _, ok := server.sessions["session-1"].datapointFingerprints["metrics.temperature"]; !ok {
		t.Fatal("expected successful datapoint to update fingerprint cache")
	}
	if _, ok := server.sessions["session-1"].datapointFingerprints["metrics.pressure"]; !ok {
		t.Fatal("expected successful datapoint to update fingerprint cache")
	}
}

func TestPreviewSocketServer_HandleDatapointBatchSubscribeKeepsSuccessfulFingerprintsOnPartialFailure(t *testing.T) {
	reader := &fakeDataPointValueReader{
		values: map[string]*service.DataPointValue{
			"metrics.temperature": {
				Path:      "metrics.temperature",
				Value:     26.5,
				Quality:   "good",
				Timestamp: time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC),
				Status:    "active",
			},
		},
		errs: map[string]error{
			"metrics.pressure": errors.New("load failed"),
		},
	}
	server := newPreviewSocketServerForTests()
	server.dataPoints = reader
	server.sessions["session-1"].datapointPollCancel = func() {}

	socket := newTestSocket("socket-1", "project-1", "session-1")
	event := &socketio.EventPayload{
		Data: []interface{}{
			map[string]any{
				"requestId": "req-2",
				"paths":     []string{"metrics.temperature", "metrics.pressure"},
			},
		},
	}

	server.handleDatapointBatchSubscribe(socket, event)

	subscriptions := server.sessions["session-1"].socketSubscriptions["socket-1"].datapoints
	if _, ok := subscriptions["metrics.temperature"]; !ok {
		t.Fatal("expected successful datapoint to remain subscribed")
	}
	if _, ok := subscriptions["metrics.pressure"]; !ok {
		t.Fatal("expected failed datapoint to stay registered for later poll retries")
	}
	if _, ok := server.sessions["session-1"].datapointFingerprints["metrics.temperature"]; !ok {
		t.Fatal("expected successful datapoint to update fingerprint cache")
	}
	if _, ok := server.sessions["session-1"].datapointFingerprints["metrics.pressure"]; ok {
		t.Fatal("expected failed datapoint not to write fingerprint cache")
	}
}

func TestPreviewSocketServer_HandleMqttSubscribeDoesNotPolluteSessionStateOnRuntimeFailure(t *testing.T) {
	repo := &fakePreviewMqttRepository{
		getSubscriptionErr: errors.New("subscription load failed"),
	}
	server := newPreviewSocketServerForTests()
	server.mqttRepository = repo

	socket := newTestSocket("socket-1", "project-1", "session-1")
	event := &socketio.EventPayload{
		Data: []interface{}{
			map[string]any{
				"requestId":      "req-3",
				"subscriptionId": "sub-1",
			},
		},
	}

	server.handleMqttSubscribe(socket, event)

	session := server.sessions["session-1"]
	if len(session.socketSubscriptions["socket-1"].mqttSubscriptions) != 0 {
		t.Fatal("expected failed MQTT subscribe not to register socket subscription state")
	}
	if len(session.mqttRuntimePending) != 0 {
		t.Fatal("expected failed MQTT runtime build to clear pending state")
	}
	if len(session.mqttRuntimes) != 0 {
		t.Fatal("expected failed MQTT runtime build not to retain runtime state")
	}
	if repo.getSubscriptionCalls != 1 {
		t.Fatalf("expected MQTT repository GetSubscription to be called once, got %d", repo.getSubscriptionCalls)
	}
}

func TestPreviewSocketServer_HandleDatapointUnsubscribeClearsState(t *testing.T) {
	cancelCount := 0
	server := newPreviewSocketServerForTests()
	session := server.sessions["session-1"]
	session.socketSubscriptions["socket-1"].datapoints["metrics.temperature"] = struct{}{}
	session.datapointFingerprints["metrics.temperature"] = "fingerprint-1"
	session.datapointPollCancel = func() {
		cancelCount++
	}

	socket := newTestSocket("socket-1", "project-1", "session-1")
	event := &socketio.EventPayload{
		Data: []interface{}{
			map[string]any{
				"requestId": "req-4",
				"path":      "metrics.temperature",
			},
		},
	}

	server.handleDatapointUnsubscribe(socket, event)

	if _, ok := session.socketSubscriptions["socket-1"].datapoints["metrics.temperature"]; ok {
		t.Fatal("expected datapoint subscription to be removed")
	}
	if _, ok := session.datapointFingerprints["metrics.temperature"]; ok {
		t.Fatal("expected datapoint fingerprint to be cleared after unsubscribe")
	}
	if cancelCount != 1 {
		t.Fatalf("expected datapoint poller to stop once, got %d", cancelCount)
	}
	if session.datapointPollCancel != nil {
		t.Fatal("expected datapoint poller cancel function to be cleared")
	}
}

func TestPreviewSocketServer_HandleMqttSubscribeRegistersRuntimeOnSuccess(t *testing.T) {
	builderCalls := 0
	server := newPreviewSocketServerForTests()
	server.mqttRepository = &fakePreviewMqttRepository{}
	runtime := &mqttPreviewRuntime{
		server:       server,
		sessionID:    "session-1",
		projectID:    "project-1",
		connection:   repository.MqttConnectionDetailRecord{ID: "conn-1"},
		subscription: repository.MqttSubscriptionRecord{ID: "sub-1", ConnectionID: "conn-1"},
		tags:         map[string]repository.MqttTagRecord{},
		lastTagValue: map[string]service.MqttTagValueSnapshot{},
	}
	server.mqttRuntimeBuilder = func(ctx context.Context, sessionID, projectID, subscriptionID string) (*mqttPreviewRuntime, error) {
		builderCalls++
		if sessionID != "session-1" || projectID != "project-1" || subscriptionID != "sub-1" {
			t.Fatalf("unexpected runtime builder input: %s %s %s", sessionID, projectID, subscriptionID)
		}
		return runtime, nil
	}

	socket := newTestSocket("socket-1", "project-1", "session-1")
	event := &socketio.EventPayload{
		Data: []interface{}{
			map[string]any{
				"requestId":      "req-5",
				"subscriptionId": "sub-1",
			},
		},
	}

	server.handleMqttSubscribe(socket, event)

	session := server.sessions["session-1"]
	if _, ok := session.socketSubscriptions["socket-1"].mqttSubscriptions["sub-1"]; !ok {
		t.Fatal("expected MQTT subscription interest to be registered")
	}
	if session.mqttRuntimes["sub-1"] != runtime {
		t.Fatal("expected MQTT runtime to be cached on successful subscribe")
	}
	if builderCalls != 1 {
		t.Fatalf("expected runtime builder to be called once, got %d", builderCalls)
	}
}

func TestPreviewSocketServer_HandleMqttTagSubscribeRegistersTagInterest(t *testing.T) {
	builderCalls := 0
	repo := &fakePreviewMqttRepository{
		tag: &repository.MqttTagRecord{
			ID:             "tag-1",
			ProjectID:      "project-1",
			SubscriptionID: "sub-1",
			ParseType:      "fixed",
			UpdatedAt:      time.Date(2026, 4, 20, 13, 0, 0, 0, time.UTC),
		},
	}
	server := newPreviewSocketServerForTests()
	server.mqttRepository = repo
	runtime := &mqttPreviewRuntime{
		server:       server,
		sessionID:    "session-1",
		projectID:    "project-1",
		connection:   repository.MqttConnectionDetailRecord{ID: "conn-1"},
		subscription: repository.MqttSubscriptionRecord{ID: "sub-1", ConnectionID: "conn-1"},
		tags:         map[string]repository.MqttTagRecord{},
		lastTagValue: map[string]service.MqttTagValueSnapshot{},
	}
	server.mqttRuntimeBuilder = func(context.Context, string, string, string) (*mqttPreviewRuntime, error) {
		builderCalls++
		return runtime, nil
	}

	socket := newTestSocket("socket-1", "project-1", "session-1")
	event := &socketio.EventPayload{
		Data: []interface{}{
			map[string]any{
				"requestId": "req-6",
				"tagId":     "tag-1",
			},
		},
	}

	server.handleMqttTagSubscribe(socket, event)

	session := server.sessions["session-1"]
	if _, ok := session.socketSubscriptions["socket-1"].mqttTags["tag-1"]; !ok {
		t.Fatal("expected MQTT tag interest to be registered")
	}
	if got := session.tagSubscriptionByID["tag-1"]; got != "sub-1" {
		t.Fatalf("expected tag subscription mapping to point at sub-1, got %q", got)
	}
	if _, ok := runtime.tags["tag-1"]; !ok {
		t.Fatal("expected runtime to track subscribed tag")
	}
	if builderCalls != 1 {
		t.Fatalf("expected runtime builder to be called once, got %d", builderCalls)
	}
	if repo.getTagCalls != 1 {
		t.Fatalf("expected repository GetTag to be called once, got %d", repo.getTagCalls)
	}
}

func TestPreviewSocketServer_HandleMqttTagUnsubscribeClearsTagMappingAndRuntime(t *testing.T) {
	server := newPreviewSocketServerForTests()
	session := server.sessions["session-1"]
	runtime := &mqttPreviewRuntime{
		server:       server,
		sessionID:    "session-1",
		projectID:    "project-1",
		connection:   repository.MqttConnectionDetailRecord{ID: "conn-1"},
		subscription: repository.MqttSubscriptionRecord{ID: "sub-1", ConnectionID: "conn-1"},
		tags: map[string]repository.MqttTagRecord{
			"tag-1": {ID: "tag-1", SubscriptionID: "sub-1"},
		},
		lastTagValue: map[string]service.MqttTagValueSnapshot{},
	}
	session.socketSubscriptions["socket-1"].mqttTags["tag-1"] = struct{}{}
	session.tagSubscriptionByID["tag-1"] = "sub-1"
	session.mqttRuntimes["sub-1"] = runtime

	socket := newTestSocket("socket-1", "project-1", "session-1")
	event := &socketio.EventPayload{
		Data: []interface{}{
			map[string]any{
				"requestId": "req-7",
				"tagId":     "tag-1",
			},
		},
	}

	server.handleMqttTagUnsubscribe(socket, event)

	if _, ok := session.socketSubscriptions["socket-1"].mqttTags["tag-1"]; ok {
		t.Fatal("expected MQTT tag subscription to be removed")
	}
	if _, ok := session.tagSubscriptionByID["tag-1"]; ok {
		t.Fatal("expected stale tag-to-subscription mapping to be cleared")
	}
	if _, ok := session.mqttRuntimes["sub-1"]; ok {
		t.Fatal("expected MQTT runtime to stop when no interest remains")
	}
}

func TestMqttPreviewRuntimeHandleMessageSkipsBatchTagWithoutCurrentPayloadValue(t *testing.T) {
	server := newPreviewSocketServerForTests()
	server.mqttRepository = &fakePreviewMqttRepository{}
	runtime := &mqttPreviewRuntime{
		server:       server,
		sessionID:    "session-1",
		projectID:    "project-1",
		connection:   repository.MqttConnectionDetailRecord{ID: "conn-1"},
		subscription: repository.MqttSubscriptionRecord{ID: "sub-1", ConnectionID: "conn-1"},
		tags: map[string]repository.MqttTagRecord{
			"tag-982-id": {
				ID:             "tag-982-id",
				SubscriptionID: "sub-1",
				DataType:       "number",
				ParseType:      "batch_jsonpath",
				ParseRule:      `{"arrayPath":"$","namePath":"N","matchName":"tag982","valuePath":"V","qualityPath":"Q"}`,
			},
		},
		lastTagValue: map[string]service.MqttTagValueSnapshot{},
	}

	runtime.handleMessage(fakeMqttMessage{
		topic:   "device/demo",
		payload: []byte(`[{"N":"tag6","V":51,"Q":192},{"N":"tag7","V":154587.47,"Q":192}]`),
	})

	if _, ok := runtime.lastTagValue["tag-982-id"]; ok {
		t.Fatal("expected batch tag without matching payload value to keep previous value unchanged")
	}
}

func TestPreviewSocketServerLoadLatestTagSnapshotSkipsBatchHistoryWithoutCurrentTag(t *testing.T) {
	server := newPreviewSocketServerForTests()
	server.mqttRepository = &fakePreviewMqttRepository{
		messages: []repository.MqttMessageRecord{
			{
				SubscriptionID: "sub-1",
				Topic:          "device/demo",
				Payload:        `[{"N":"tag2","V":93,"Q":192}]`,
				ReceivedAt:     time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC),
			},
		},
	}
	tag := repository.MqttTagRecord{
		ID:             "tag-982-id",
		SubscriptionID: "sub-1",
		DataType:       "number",
		ParseType:      "batch_jsonpath",
		ParseRule:      `{"arrayPath":"$","namePath":"N","matchName":"tag982","valuePath":"V","qualityPath":"Q"}`,
		CreatedAt:      time.Date(2026, 6, 3, 9, 0, 0, 0, time.UTC),
	}

	snapshot := server.loadLatestTagSnapshot(context.Background(), "project-1", nil, tag)
	if snapshot != nil {
		t.Fatalf("expected missing batch tag history to be skipped, got %#v", snapshot)
	}
}

func TestPreviewSocketServerLoadLatestTagSnapshotSkipsEmptyHistory(t *testing.T) {
	server := newPreviewSocketServerForTests()
	server.mqttRepository = &fakePreviewMqttRepository{}
	tag := repository.MqttTagRecord{
		ID:             "tag-1",
		SubscriptionID: "sub-1",
		DataType:       "number",
		ParseType:      "batch_jsonpath",
		ParseRule:      `{"arrayPath":"$","namePath":"N","matchName":"tag1","valuePath":"V","qualityPath":"Q"}`,
		CreatedAt:      time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC),
	}

	snapshot := server.loadLatestTagSnapshot(context.Background(), "project-1", nil, tag)
	if snapshot != nil {
		t.Fatalf("expected empty tag history to skip initial socket value, got %#v", snapshot)
	}
}

type testStringer string

func (s testStringer) String() string {
	return string(s)
}

type blankError struct{}

func (blankError) Error() string {
	return "   "
}

func newPreviewSocketServerForTests() *PreviewSocketServer {
	return &PreviewSocketServer{
		sessions: map[string]*previewSessionState{
			"session-1": {
				sessionID:             "session-1",
				projectID:             "project-1",
				userID:                "user-1",
				sockets:               map[string]*socketio.Socket{},
				socketSubscriptions:   map[string]*socketSubscriptionSet{"socket-1": newSocketSubscriptionSet()},
				tagSubscriptionByID:   map[string]string{},
				mqttRuntimes:          map[string]*mqttPreviewRuntime{},
				mqttRuntimePending:    map[string]*mqttRuntimePendingState{},
				datapointFingerprints: map[string]string{},
			},
		},
		closed: make(chan struct{}),
	}
}

func newTestSocket(socketID, projectID, sessionID string) *socketio.Socket {
	socket := &socketio.Socket{Id: socketID}
	socket.Metadata(previewSocketMetadataKey, &socketSessionMetadata{
		ProjectID:      projectID,
		PreviewSession: sessionID,
		UserID:         "user-1",
	})
	return socket
}

type fakeDataPointValueReader struct {
	values map[string]*service.DataPointValue
	errs   map[string]error
	calls  []string
}

func (f *fakeDataPointValueReader) GetDataPointValue(_ context.Context, _ string, path string) (*service.DataPointValue, error) {
	f.calls = append(f.calls, path)
	if err := f.errs[path]; err != nil {
		return nil, err
	}
	if value := f.values[path]; value != nil {
		copyValue := *value
		return &copyValue, nil
	}
	return nil, errors.New("datapoint not found")
}

type fakePreviewMqttRepository struct {
	getSubscriptionErr   error
	getConnectionErr     error
	getTagErr            error
	listMessagesErr      error
	createMessageErr     error
	subscription         *repository.MqttSubscriptionRecord
	connection           *repository.MqttConnectionDetailRecord
	tag                  *repository.MqttTagRecord
	messages             []repository.MqttMessageRecord
	getSubscriptionCalls int
	getConnectionCalls   int
	getTagCalls          int
	listMessagesCalls    int
	createMessageCalls   int
}

func (f *fakePreviewMqttRepository) GetConnectionDetail(_ context.Context, _ string, _ string) (*repository.MqttConnectionDetailRecord, error) {
	f.getConnectionCalls++
	if f.getConnectionErr != nil {
		return nil, f.getConnectionErr
	}
	if f.connection != nil {
		record := *f.connection
		return &record, nil
	}
	return &repository.MqttConnectionDetailRecord{}, nil
}

func (f *fakePreviewMqttRepository) GetSubscription(_ context.Context, _ string, _ string) (*repository.MqttSubscriptionRecord, error) {
	f.getSubscriptionCalls++
	if f.getSubscriptionErr != nil {
		return nil, f.getSubscriptionErr
	}
	if f.subscription != nil {
		record := *f.subscription
		return &record, nil
	}
	return &repository.MqttSubscriptionRecord{}, nil
}

func (f *fakePreviewMqttRepository) GetTag(_ context.Context, _ string, _ string) (*repository.MqttTagRecord, error) {
	f.getTagCalls++
	if f.getTagErr != nil {
		return nil, f.getTagErr
	}
	if f.tag != nil {
		record := *f.tag
		return &record, nil
	}
	return &repository.MqttTagRecord{}, nil
}

func (f *fakePreviewMqttRepository) ListMessages(_ context.Context, _ string, _ string, _ int) ([]repository.MqttMessageRecord, error) {
	f.listMessagesCalls++
	if f.listMessagesErr != nil {
		return nil, f.listMessagesErr
	}
	if len(f.messages) == 0 {
		return nil, nil
	}
	return append([]repository.MqttMessageRecord(nil), f.messages...), nil
}

func (f *fakePreviewMqttRepository) ListMessagesAfter(ctx context.Context, projectID, subscriptionID string, _ time.Time, limit int) ([]repository.MqttMessageRecord, error) {
	return f.ListMessages(ctx, projectID, subscriptionID, limit)
}

func (f *fakePreviewMqttRepository) CreateMessage(_ context.Context, params repository.CreateMqttMessageParams) (*repository.MqttMessageRecord, error) {
	f.createMessageCalls++
	if f.createMessageErr != nil {
		return nil, f.createMessageErr
	}
	return &repository.MqttMessageRecord{
		SubscriptionID: params.SubscriptionID,
		Topic:          params.Topic,
		Payload:        params.Payload,
		QOS:            params.QOS,
		ReceivedAt:     params.ReceivedAt,
	}, nil
}

type fakeMqttMessage struct {
	topic     string
	payload   []byte
	qos       byte
	retained  bool
	duplicate bool
	messageID uint16
}

func (m fakeMqttMessage) Duplicate() bool {
	return m.duplicate
}

func (m fakeMqttMessage) Qos() byte {
	return m.qos
}

func (m fakeMqttMessage) Retained() bool {
	return m.retained
}

func (m fakeMqttMessage) Topic() string {
	return m.topic
}

func (m fakeMqttMessage) MessageID() uint16 {
	return m.messageID
}

func (m fakeMqttMessage) Payload() []byte {
	return m.payload
}

func (m fakeMqttMessage) Ack() {}
