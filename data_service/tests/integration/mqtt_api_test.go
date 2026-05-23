package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/indu-forge/data_service/internal/app"
	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/config"
)

func TestMqttConnectionLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupMigrator(t, fixture.pool)
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("migrate up failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "mqtt-lifecycle-secret-01"

	srv, err := app.NewServer(config.Config{
		Addr:               ":0",
		DatabaseURL:        fixture.databaseURL,
		DatabaseSearchPath: fixture.schemaName,
		JWTSecret:          secret,
	})
	if err != nil {
		t.Fatalf("create server failed: %v", err)
	}
	t.Cleanup(srv.Close)

	server := httptest.NewServer(srv.Handler())
	t.Cleanup(server.Close)

	token := mustSignIntegrationJWT(t, secret, &auth.Claims{
		UserID:       userID,
		TenantID:     "tenant-mqtt",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	connection := mustCreateMqttConnection(t, server.URL, token, projectID, map[string]any{
		"name":      "mqtt-main",
		"brokerUrl": "tcp://localhost:1883",
		"protocol":  "mqtt",
		"port":      1883,
		"qos":       1,
	})

	subscriptionID := insertTestMqttSubscription(t, ctx, fixture, projectID, connection.ID, userID)

	startResult := mustStartMqttConnection(t, server.URL, token, projectID, connection.ID)
	if startResult.Status != "connected" {
		t.Fatalf("expected start status connected, got %q", startResult.Status)
	}

	statusResult := mustGetMqttConnectionStatus(t, server.URL, token, projectID, connection.ID)
	if statusResult.Status != "connected" {
		t.Fatalf("expected status endpoint return connected, got %q", statusResult.Status)
	}

	insertTestMqttMessage(t, ctx, fixture, projectID, connection.ID, subscriptionID, "factory/line1/temp", `{"value": 88.5}`, 1)

	messages := mustListMqttMessages(t, server.URL, token, projectID, subscriptionID, 10)
	if len(messages) != 1 {
		t.Fatalf("expected 1 mqtt message, got %d", len(messages))
	}
	if messages[0].Topic != "factory/line1/temp" {
		t.Fatalf("expected topic factory/line1/temp, got %q", messages[0].Topic)
	}
	if messages[0].Payload != `{"value": 88.5}` {
		t.Fatalf("expected payload {\"value\": 88.5}, got %q", messages[0].Payload)
	}
}

func TestMqttSubscriptionChineseNameDataPointStaysActive(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupMigrator(t, fixture.pool)
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("migrate up failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "mqtt-chinese-subscription-secret-01"

	srv, err := app.NewServer(config.Config{
		Addr:               ":0",
		DatabaseURL:        fixture.databaseURL,
		DatabaseSearchPath: fixture.schemaName,
		JWTSecret:          secret,
	})
	if err != nil {
		t.Fatalf("create server failed: %v", err)
	}
	t.Cleanup(srv.Close)

	server := httptest.NewServer(srv.Handler())
	t.Cleanup(server.Close)

	token := mustSignIntegrationJWT(t, secret, &auth.Claims{
		UserID:       userID,
		TenantID:     "tenant-mqtt-chinese",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	connection := mustCreateMqttConnection(t, server.URL, token, projectID, map[string]any{
		"name":      "aaaa",
		"brokerUrl": "tcp://localhost:1883",
		"protocol":  "mqtt",
		"port":      1883,
		"qos":       1,
	})
	subscription := mustCreateMqttSubscription(t, server.URL, token, projectID, connection.ID, map[string]any{
		"name":             "撒大苏打",
		"topic":            "aaaa",
		"qos":              0,
		"messageRetention": 100,
	})

	datapoints := mustListDataPoints(t, server.URL, token, projectID, "type=mqtt.subscription&search=撒大苏打")
	if len(datapoints.DataPoints) != 1 {
		t.Fatalf("expected mqtt subscription datapoint, got %d", len(datapoints.DataPoints))
	}
	point := datapoints.DataPoints[0]
	if point.Name != "撒大苏打" {
		t.Fatalf("expected datapoint name 撒大苏打, got %q", point.Name)
	}
	if point.Path != "mqtt.aaaa.aaaa" {
		t.Fatalf("expected datapoint path mqtt.aaaa.aaaa, got %q", point.Path)
	}
	if point.Status != "active" {
		t.Fatalf("expected datapoint active, got %q", point.Status)
	}
	if point.SourceID == nil || *point.SourceID != subscription.ID {
		t.Fatalf("expected source id %q, got %#v", subscription.ID, point.SourceID)
	}

	if _, err := fixture.pool.Exec(ctx, `
		UPDATE data_points
		SET status = 'invalid',
		    updated_at = now()
		WHERE project_id = $1 AND id = $2
	`, projectID, point.ID); err != nil {
		t.Fatalf("mark subscription datapoint invalid failed: %v", err)
	}
	refreshed := mustListDataPoints(t, server.URL, token, projectID, "type=mqtt.subscription&search=撒大苏打")
	if len(refreshed.DataPoints) != 1 {
		t.Fatalf("expected refreshed mqtt subscription datapoint, got %d", len(refreshed.DataPoints))
	}
	if refreshed.DataPoints[0].Status != "active" {
		t.Fatalf("expected refreshed datapoint active, got %q", refreshed.DataPoints[0].Status)
	}
}

func TestMqttSubscriptionDataPointValidWithoutMqttConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupMigrator(t, fixture.pool)
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("migrate up failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "mqtt-subscription-no-config-secret-01"

	srv, err := app.NewServer(config.Config{
		Addr:               ":0",
		DatabaseURL:        fixture.databaseURL,
		DatabaseSearchPath: fixture.schemaName,
		JWTSecret:          secret,
	})
	if err != nil {
		t.Fatalf("create server failed: %v", err)
	}
	t.Cleanup(srv.Close)

	server := httptest.NewServer(srv.Handler())
	t.Cleanup(server.Close)

	token := mustSignIntegrationJWT(t, secret, &auth.Claims{
		UserID:       userID,
		TenantID:     "tenant-mqtt-no-config",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	connectionID := uuid.NewString()
	if _, err := fixture.pool.Exec(ctx, `
		INSERT INTO data_connections (id, project_id, name, type, category, status, metadata, created_by, updated_by)
		VALUES ($1, $2, 'aaaa', 'mqtt', 'protocol', 'unknown', '{}'::jsonb, $3, $3)
	`, connectionID, projectID, userID); err != nil {
		t.Fatalf("insert mqtt connection without config failed: %v", err)
	}

	subscription := mustCreateMqttSubscription(t, server.URL, token, projectID, connectionID, map[string]any{
		"name":             "仅订阅测试",
		"topic":            "only/subscription",
		"qos":              0,
		"messageRetention": 100,
	})
	datapoints := mustListDataPoints(t, server.URL, token, projectID, "type=mqtt.subscription&search=仅订阅测试")
	if len(datapoints.DataPoints) != 1 {
		t.Fatalf("expected subscription datapoint, got %d", len(datapoints.DataPoints))
	}
	if datapoints.DataPoints[0].SourceID == nil || *datapoints.DataPoints[0].SourceID != subscription.ID {
		t.Fatalf("expected source id %q, got %#v", subscription.ID, datapoints.DataPoints[0].SourceID)
	}
	if datapoints.DataPoints[0].Status != "active" {
		t.Fatalf("expected subscription datapoint active, got %q", datapoints.DataPoints[0].Status)
	}
}

type mqttConnectionPayload struct {
	ID        string `json:"id"`
	ProjectID string `json:"projectId"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Status    string `json:"status"`
}

type mqttConnectionStatusPayload struct {
	Status string `json:"status"`
}

type mqttSubscriptionPayload struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type mqttMessagePayload struct {
	ID             int64     `json:"id"`
	SubscriptionID string    `json:"subscriptionId"`
	Topic          string    `json:"topic"`
	Payload        string    `json:"payload"`
	QOS            int       `json:"qos"`
	ReceivedAt     time.Time `json:"receivedAt"`
}

type mqttListResponse[T any] struct {
	List       []T                `json:"list"`
	Pagination mqttPaginationData `json:"pagination"`
}

type mqttPaginationData struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

func mustCreateMqttConnection(t *testing.T, baseURL, token, projectID string, payload map[string]any) mqttConnectionPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodPost, baseURL+"/api/v1/data/projects/"+projectID+"/mqtt/connections", token, payload)
	var result mqttConnectionPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode mqtt create connection response failed: %v", err)
	}
	if result.ID == "" {
		t.Fatal("expected mqtt connection id in response")
	}
	return result
}

func mustCreateMqttSubscription(t *testing.T, baseURL, token, projectID, connectionID string, payload map[string]any) mqttSubscriptionPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodPost, baseURL+"/api/v1/data/projects/"+projectID+"/mqtt/connections/"+connectionID+"/subscriptions", token, payload)
	var result mqttSubscriptionPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode mqtt create subscription response failed: %v", err)
	}
	if result.ID == "" {
		t.Fatal("expected mqtt subscription id in response")
	}
	return result
}

func mustStartMqttConnection(t *testing.T, baseURL, token, projectID, connectionID string) mqttConnectionStatusPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodPost, baseURL+"/api/v1/data/projects/"+projectID+"/mqtt/connections/"+connectionID+"/start", token, map[string]any{})
	var result mqttConnectionStatusPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode mqtt start response failed: %v", err)
	}
	return result
}

func mustGetMqttConnectionStatus(t *testing.T, baseURL, token, projectID, connectionID string) mqttConnectionStatusPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodGet, baseURL+"/api/v1/data/projects/"+projectID+"/mqtt/connections/"+connectionID+"/status", token, nil)
	var result mqttConnectionStatusPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode mqtt status response failed: %v", err)
	}
	return result
}

func mustListMqttMessages(t *testing.T, baseURL, token, projectID, subscriptionID string, limit int) []mqttMessagePayload {
	t.Helper()

	url := baseURL + "/api/v1/data/projects/" + projectID + "/mqtt/subscriptions/" + subscriptionID + "/messages?limit=" + strconv.Itoa(limit)

	responseEnvelope := doJSONRequest(t, http.MethodGet, url, token, nil)
	var result mqttListResponse[mqttMessagePayload]
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode mqtt messages response failed: %v", err)
	}
	if result.Pagination.Page != 1 {
		t.Fatalf("expected message page 1, got %d", result.Pagination.Page)
	}
	if result.Pagination.PageSize != limit {
		t.Fatalf("expected message pageSize %d, got %d", limit, result.Pagination.PageSize)
	}
	return result.List
}

func insertTestMqttSubscription(t *testing.T, ctx context.Context, fixture *testDatabase, projectID, connectionID, userID string) string {
	t.Helper()

	subscriptionID := uuid.NewString()
	_, err := fixture.pool.Exec(ctx, `
		INSERT INTO data_mqtt_subscriptions (
			id,
			project_id,
			connection_id,
			name,
			topic,
			qos,
			message_retention,
			created_by,
			updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
	`, subscriptionID, projectID, connectionID, "sub-temp", "factory/line1/temp", 1, 100, userID)
	if err != nil {
		t.Fatalf("insert mqtt subscription failed: %v", err)
	}

	return subscriptionID
}

func insertTestMqttMessage(t *testing.T, ctx context.Context, fixture *testDatabase, projectID, connectionID, subscriptionID, topic, payload string, qos int) {
	t.Helper()

	_, err := fixture.pool.Exec(ctx, `
		INSERT INTO data_mqtt_messages (
			project_id,
			connection_id,
			subscription_id,
			topic,
			payload,
			qos
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, projectID, connectionID, subscriptionID, topic, payload, qos)
	if err != nil {
		t.Fatalf("insert mqtt message failed: %v", err)
	}
}
