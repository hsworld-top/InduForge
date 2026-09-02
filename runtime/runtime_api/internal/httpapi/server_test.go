package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/indu-forge/runtime-api/internal/artifact"
	runtimeauth "github.com/indu-forge/runtime-api/internal/auth"
	"github.com/indu-forge/runtime-api/internal/realtime"
	"github.com/indu-forge/runtime-api/internal/runtimeview"
)

type fakeStore struct {
	pingErr error
	current runtimeview.PointCurrent
	history []runtimeview.PointSample
	alarms  []runtimeview.AlarmState
	ack     runtimeview.AlarmAcknowledgement
	ackErr  error
	ackCall struct {
		deploymentID, itemID, subjectID, comment string
		expectedVersion                          int64
	}
}

func (f *fakeStore) ReserveManualSequence(context.Context, string, int64) (int64, error) {
	return 1, nil
}

type fakePublisher struct{}

func (fakePublisher) PublishRaw(context.Context, string, []byte) error { return nil }

func (f *fakeStore) Ping(context.Context) error { return f.pingErr }

func (f *fakeStore) Current(context.Context, string, string) (runtimeview.PointCurrent, error) {
	if len(f.current.Value) == 0 {
		return runtimeview.PointCurrent{}, runtimeview.ErrNotFound
	}
	return f.current, nil
}

func (f *fakeStore) History(context.Context, string, string, runtimeview.HistoryQuery) ([]runtimeview.PointSample, error) {
	return f.history, nil
}

func (f *fakeStore) AlarmStates(context.Context, string, int) ([]runtimeview.AlarmState, error) {
	return f.alarms, nil
}

func (f *fakeStore) AcknowledgeAlarm(_ context.Context, deploymentID, itemID string, expectedVersion int64, subjectID, comment string, _ time.Time) (runtimeview.AlarmAcknowledgement, error) {
	f.ackCall.deploymentID, f.ackCall.itemID, f.ackCall.expectedVersion = deploymentID, itemID, expectedVersion
	f.ackCall.subjectID, f.ackCall.comment = subjectID, comment
	return f.ack, f.ackErr
}

func TestRuntimeAPISessionAndCurrentPointUseGatewayIdentity(t *testing.T) {
	server, _, token := newTestRuntimeAPI(t, &fakeStore{current: testCurrent()})
	login := authenticatedRequest(http.MethodPost, "/api/v1/runtime/session", token, nil)
	loginResponse := httptest.NewRecorder()
	server.Handler().ServeHTTP(loginResponse, login)
	if loginResponse.Code != http.StatusOK || len(loginResponse.Result().Cookies()) != 1 {
		t.Fatalf("login status=%d body=%s", loginResponse.Code, loginResponse.Body.String())
	}

	request := authenticatedRequest(http.MethodGet, "/api/v1/runtime/points/line.temperature", "", loginResponse.Result().Cookies()[0])
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"value":42.5`) || !strings.Contains(response.Body.String(), `"path":"line.temperature"`) {
		t.Fatalf("current status=%d body=%s", response.Code, response.Body.String())
	}

	request = authenticatedRequest(http.MethodGet, "/api/v1/runtime/points/line.temperature", "", loginResponse.Result().Cookies()[0])
	request.Header.Set("X-InduForge-Deployment-Id", "other")
	response = httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("cross deployment request must fail: %d", response.Code)
	}
}

func TestRuntimeAPIAcknowledgesActiveAlarmForOperator(t *testing.T) {
	store := &fakeStore{ack: runtimeview.AlarmAcknowledgement{Version: 8, AcknowledgedAt: time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)}}
	server, token := newTestRuntimeAPIWithRoles(t, store, []string{"operator"})
	request := authenticatedRequest(http.MethodPost, "/api/v1/runtime/alarms/33333333-3333-4333-8333-333333333333/acknowledge", token, nil)
	request.Body = io.NopCloser(strings.NewReader(`{"expectedVersion":7,"comment":"现场已确认"}`))
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"version":8`) {
		t.Fatalf("ack status=%d body=%s", response.Code, response.Body.String())
	}
	if store.ackCall.deploymentID != "deployment-1" || store.ackCall.itemID != "33333333-3333-4333-8333-333333333333" || store.ackCall.expectedVersion != 7 || store.ackCall.subjectID != "user-1" || store.ackCall.comment != "现场已确认" {
		t.Fatalf("unexpected acknowledgement: %#v", store.ackCall)
	}
}

func TestRuntimeAPIRejectsAlarmAcknowledgementWithoutRoleOrFreshVersion(t *testing.T) {
	server, _, token := newTestRuntimeAPI(t, &fakeStore{})
	request := authenticatedRequest(http.MethodPost, "/api/v1/runtime/alarms/33333333-3333-4333-8333-333333333333/acknowledge", token, nil)
	request.Body = io.NopCloser(strings.NewReader(`{"expectedVersion":1,"comment":""}`))
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), `"code":40301`) {
		t.Fatalf("viewer acknowledgement=%d %s", response.Code, response.Body.String())
	}

	server, token = newTestRuntimeAPIWithRoles(t, &fakeStore{ackErr: runtimeview.ErrAlarmVersionConflict}, []string{"admin"})
	request = authenticatedRequest(http.MethodPost, "/api/v1/runtime/alarms/33333333-3333-4333-8333-333333333333/acknowledge", token, nil)
	request.Body = io.NopCloser(strings.NewReader(`{"expectedVersion":1,"comment":""}`))
	response = httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "报警状态已变更") {
		t.Fatalf("stale acknowledgement=%d %s", response.Code, response.Body.String())
	}
}

func TestRuntimeAPIHealthAndStatusReportRealDependencyState(t *testing.T) {
	server, _, _ := newTestRuntimeAPI(t, &fakeStore{pingErr: errors.New("postgres down")})
	health := httptest.NewRecorder()
	server.Handler().ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/health", nil))
	var healthPayload struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Status     string `json:"status"`
			ReasonCode string `json:"reasonCode"`
		} `json:"data"`
	}
	if err := json.Unmarshal(health.Body.Bytes(), &healthPayload); err != nil {
		t.Fatal(err)
	}
	if health.Code != http.StatusServiceUnavailable || healthPayload.Code != 50031 || healthPayload.Msg != "运行依赖不可用：PostgreSQL" || healthPayload.Data.Status != "DOWN" || healthPayload.Data.ReasonCode != "postgres-unavailable" {
		t.Fatalf("health=%d %s", health.Code, health.Body.String())
	}
	healthy, _, _ := newTestRuntimeAPI(t, &fakeStore{})
	health = httptest.NewRecorder()
	healthy.Handler().ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/health", nil))
	if err := json.Unmarshal(health.Body.Bytes(), &healthPayload); err != nil {
		t.Fatal(err)
	}
	if health.Code != http.StatusOK || healthPayload.Code != 0 || healthPayload.Msg != "ok" || healthPayload.Data.Status != "UP" {
		t.Fatalf("healthy health=%d %s", health.Code, health.Body.String())
	}
	status := httptest.NewRecorder()
	server.Handler().ServeHTTP(status, httptest.NewRequest(http.MethodGet, "/api/v1/status", nil))
	if status.Code != http.StatusOK || !strings.Contains(status.Body.String(), `"healthState":"UNAVAILABLE"`) || !strings.Contains(status.Body.String(), `"componentRole":"runtime-api"`) {
		t.Fatalf("status=%d %s", status.Code, status.Body.String())
	}
	var payload struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(status.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Data["schemaVersion"] != "runtime-health-status.v1" || payload.Data["accountId"] != "account-1" || payload.Data["processId"].(float64) < 1 {
		t.Fatalf("status identity does not satisfy runtime-health-status.v1: %#v", payload.Data)
	}
	freshness, ok := payload.Data["businessFreshness"].(map[string]any)
	if !ok || freshness["state"] != "UNKNOWN" || freshness["evaluatedAt"] == nil {
		t.Fatalf("invalid business freshness: %#v", payload.Data["businessFreshness"])
	}
	if _, exists := payload.Data["upstreamStatus"]; exists {
		t.Fatal("runtime-health-status.v1 must not contain upstreamStatus")
	}
}

func TestRuntimeAPIRejectsInvalidHistoryBoundsAndCollectorWrites(t *testing.T) {
	server, _, token := newTestRuntimeAPI(t, &fakeStore{})
	request := authenticatedRequest(http.MethodGet, "/api/v1/runtime/points/line.temperature/history?from=2026-08-31T11:00:00Z&to=2026-08-31T10:00:00Z", token, nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid range status=%d body=%s", response.Code, response.Body.String())
	}
	request = authenticatedRequest(http.MethodPost, "/api/v1/runtime/points/line.temperature/write", token, nil)
	response = httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), "无权") {
		t.Fatalf("collector write status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestRuntimeAPIAlarmSocketRequiresGatewaySessionAndFiltersAlarmIdentity(t *testing.T) {
	server, hub, token := newTestRuntimeAPI(t, &fakeStore{})
	ws := httptest.NewServer(server.Handler())
	defer ws.Close()
	url := "ws" + strings.TrimPrefix(ws.URL, "http") + "/ws/v1/alarms"
	connection, _, err := websocket.DefaultDialer.Dial(url, http.Header{"X-InduForge-Deployment-Id": []string{"deployment-1"}, "X-InduForge-Project-Id": []string{testProjectID}, "Authorization": []string{"Bearer " + token}})
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if err := connection.WriteJSON(map[string]any{"action": "subscribe"}); err != nil {
		t.Fatal(err)
	}
	var subscribed map[string]any
	if err := connection.ReadJSON(&subscribed); err != nil || subscribed["type"] != "subscribed" {
		t.Fatalf("subscribe=%#v err=%v", subscribed, err)
	}
	payload := `{"schemaVersion":"alarm.event.v1","subject":"alarm.event","eventId":"alarm-event-1","deploymentId":"deployment-1","accountId":"account-1","kind":"alarm-transition","operation":"RAISE","alarmId":"temperature-high","alarmItemId":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","pointIds":["22222222-2222-4222-8222-222222222222"],"transition":{"alarmState":"OPEN"}}`
	if err := hub.AcceptAlarm("alarm.event", []byte(payload)); err != nil {
		t.Fatal(err)
	}
	_ = connection.SetReadDeadline(time.Now().Add(time.Second))
	var event map[string]any
	if err := connection.ReadJSON(&event); err != nil || event["type"] != "alarm" {
		t.Fatalf("event=%#v err=%v", event, err)
	}
	if err := hub.AcceptAlarm("alarm.event", []byte(strings.Replace(payload, `"deployment-1"`, `"other"`, 1))); err == nil {
		t.Fatal("cross-deployment event must fail")
	}
}

func TestRuntimeAPIWebSocketStreamsValidatedPointEvents(t *testing.T) {
	server, hub, token := newTestRuntimeAPI(t, &fakeStore{})
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	loginRequest, _ := http.NewRequest(http.MethodPost, httpServer.URL+"/api/v1/runtime/session", nil)
	setIdentity(loginRequest)
	loginRequest.Header.Set("Authorization", "Bearer "+token)
	loginResponse, err := http.DefaultClient.Do(loginRequest)
	if err != nil {
		t.Fatal(err)
	}
	loginResponse.Body.Close()
	if len(loginResponse.Cookies()) != 1 {
		t.Fatal("expected runtime session cookie")
	}

	header := http.Header{}
	header.Set("X-InduForge-Deployment-Id", "deployment-1")
	header.Set("X-InduForge-Project-Id", testProjectID)
	header.Set("Cookie", loginResponse.Cookies()[0].String())
	header.Set("Origin", httpServer.URL)
	connection, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(httpServer.URL, "http")+"/ws/v1/points", header)
	if err != nil {
		if response != nil {
			t.Fatalf("websocket status=%d err=%v", response.StatusCode, err)
		}
		t.Fatal(err)
	}
	defer connection.Close()
	if err := connection.WriteJSON(map[string]any{"action": "subscribe", "paths": []string{"line.temperature"}}); err != nil {
		t.Fatal(err)
	}
	var ack map[string]any
	if err := connection.ReadJSON(&ack); err != nil || ack["type"] != "subscribed" {
		t.Fatalf("subscribe ack=%#v err=%v", ack, err)
	}
	payload := `{"schemaVersion":"data.raw.v1","deploymentId":"deployment-1","accountId":"account-1","pointId":"22222222-2222-4222-8222-222222222222","eventId":"event-1","value":42.5,"quality":"good","sourceTimestamp":"2026-08-31T10:00:00Z","serverTimestamp":"2026-08-31T10:00:01Z","sequence":1}`
	if err := hub.Accept("data.raw.22222222-2222-4222-8222-222222222222", []byte(payload)); err != nil {
		t.Fatal(err)
	}
	_ = connection.SetReadDeadline(time.Now().Add(time.Second))
	var event map[string]any
	if err := connection.ReadJSON(&event); err != nil || event["type"] != "point" {
		t.Fatalf("point event=%#v err=%v", event, err)
	}
}

const testProjectID = "11111111-1111-4111-8111-111111111111"

func newTestRuntimeAPI(t *testing.T, store *fakeStore) (*Server, *realtime.Hub, string) {
	server, hub, token := newTestRuntimeAPIWithRolesAndHub(t, store, []string{"viewer"})
	return server, hub, token
}

func newTestRuntimeAPIWithRoles(t *testing.T, store *fakeStore, roles []string) (*Server, string) {
	server, _, token := newTestRuntimeAPIWithRolesAndHub(t, store, roles)
	return server, token
}

func newTestRuntimeAPIWithRolesAndHub(t *testing.T, store *fakeStore, roles []string) (*Server, *realtime.Hub, string) {
	t.Helper()
	catalogPath := filepath.Join(t.TempDir(), "artifact.json")
	catalogPayload := `{"schemaVersion":"runtime-project-artifact.v1","projectArtifactVersion":"1.0","projectId":"11111111-1111-4111-8111-111111111111","dataPoints":[{"id":"22222222-2222-4222-8222-222222222222","path":"line.temperature","name":"温度","dataType":"float64","sourceType":"collector.point","sourceId":"33333333-3333-4333-8333-333333333333","runtimePermissions":{"write":{"allowRoles":[],"denyRoles":[],"inherit":true}},"refreshMode":"subscription","status":"active","unit":"C","precisionNum":1,"defaultValue":null,"tags":[],"attributeDefaults":{}}],"computeUnits":[],"alarmItems":[]}`
	if err := os.WriteFile(catalogPath, []byte(catalogPayload), 0o600); err != nil {
		t.Fatal(err)
	}
	catalog, err := artifact.Load(catalogPath, testProjectID)
	if err != nil {
		t.Fatal(err)
	}
	token := "test-runtime-token"
	authPath := filepath.Join(t.TempDir(), "tokens.json")
	rolesPayload, err := json.Marshal(roles)
	if err != nil {
		t.Fatal(err)
	}
	authPayload := fmt.Sprintf(`{"schemaVersion":"runtime-api-tokens.v1","tokens":[{"tokenSha256":"%s","subjectId":"user-1","roles":%s}]}`, runtimeauth.DigestToken(token), rolesPayload)
	if err := os.WriteFile(authPath, []byte(authPayload), 0o600); err != nil {
		t.Fatal(err)
	}
	authorizer, err := runtimeauth.Load(authPath)
	if err != nil {
		t.Fatal(err)
	}
	hub := realtime.NewHub("deployment-1", "account-1", catalog)
	server, err := New(Config{
		DeploymentID: "deployment-1", ProjectID: testProjectID, AccountID: "account-1",
		SiteID: "site-1", NodeID: "node-1", Version: "release-1", ExecutionForm: "native-linux",
		Catalog: catalog, Store: store, Authorizer: authorizer, Realtime: hub, ManualEpoch: 1, ManualWriter: store, Publisher: fakePublisher{},
		Now: func() time.Time { return time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	return server, hub, token
}

func authenticatedRequest(method, target, token string, cookie *http.Cookie) *http.Request {
	request := httptest.NewRequest(method, target, nil)
	setIdentity(request)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	if cookie != nil {
		request.AddCookie(cookie)
	}
	return request
}

func setIdentity(request *http.Request) {
	request.Header.Set("X-InduForge-Deployment-Id", "deployment-1")
	request.Header.Set("X-InduForge-Project-Id", testProjectID)
}

func testCurrent() runtimeview.PointCurrent {
	timestamp := time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)
	return runtimeview.PointCurrent{
		PointID: "22222222-2222-4222-8222-222222222222", Value: json.RawMessage(`42.5`), Quality: "good",
		SourceTimestamp: timestamp, ServerTimestamp: timestamp.Add(time.Second), Sequence: 1, EventID: "event-1", UpdatedAt: timestamp,
	}
}
