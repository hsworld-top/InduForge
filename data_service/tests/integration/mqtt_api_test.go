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

type mqttMessagePayload struct {
	ID             int64     `json:"id"`
	SubscriptionID string    `json:"subscriptionId"`
	Topic          string    `json:"topic"`
	Payload        string    `json:"payload"`
	QOS            int       `json:"qos"`
	ReceivedAt     time.Time `json:"receivedAt"`
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
	var result []mqttMessagePayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode mqtt messages response failed: %v", err)
	}
	return result
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
			is_enabled,
			message_retention,
			created_by,
			updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
	`, subscriptionID, projectID, connectionID, "sub-temp", "factory/line1/temp", 1, true, 100, userID)
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
