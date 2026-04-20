package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/indu-forge/data_service/internal/app"
	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/config"
)

func TestProtocolWave1(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupMigrator(t, fixture.pool)
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("migrate up failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "protocol-wave1-secret-01"

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
		TenantID:     "tenant-wave1",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	kafka := mustCreateKafkaConfig(t, server.URL, token, projectID, map[string]any{
		"name":          "kafka-source-main",
		"brokers":       "127.0.0.1:9092",
		"topic":         "factory.events",
		"consumerGroup": "dc-wave1",
	})
	if kafka.Type != "kafka" {
		t.Fatalf("expected kafka type, got %q", kafka.Type)
	}

	previewRows := mustPreviewKafkaTopic(t, server.URL, token, projectID, kafka.ID)
	if len(previewRows) == 0 {
		t.Fatal("expected kafka preview rows")
	}
	if previewRows[0].Payload != `{"source":"kafka-preview-mock","status":"ok"}` {
		t.Fatalf("expected kafka preview to stay on mock sample in phase1, got %q", previewRows[0].Payload)
	}

	httpConn := mustCreateHTTPConfig(t, server.URL, token, projectID, map[string]any{
		"name":    "http-source-main",
		"baseUrl": "https://example.com/data",
		"method":  "GET",
	})
	if httpConn.Type != "http" {
		t.Fatalf("expected http type, got %q", httpConn.Type)
	}

	wsConn := mustCreateWebSocketConfig(t, server.URL, token, projectID, map[string]any{
		"name":  "ws-source-main",
		"url":   "ws://localhost:8080/ws",
		"topic": "factory/ws/events",
	})
	if wsConn.Type != "websocket" {
		t.Fatalf("expected websocket type, got %q", wsConn.Type)
	}

	redisConn := mustCreateRedisConfig(t, server.URL, token, projectID, map[string]any{
		"name":       "redis-source-main",
		"address":    "127.0.0.1:6379",
		"keyPattern": "factory:*",
	})
	if redisConn.Type != "redis" {
		t.Fatalf("expected redis type, got %q", redisConn.Type)
	}
}

type protocolWave1ConnectionPayload struct {
	ID        string `json:"id"`
	ProjectID string `json:"projectId"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Status    string `json:"status"`
}

type kafkaPreviewPayload struct {
	Topic   string `json:"topic"`
	Payload string `json:"payload"`
}

func mustCreateKafkaConfig(t *testing.T, baseURL, token, projectID string, payload map[string]any) protocolWave1ConnectionPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodPost, baseURL+"/api/v1/data/projects/"+projectID+"/kafka/configs", token, payload)
	return decodeProtocolWave1Connection(t, responseEnvelope.Data)
}

func mustPreviewKafkaTopic(t *testing.T, baseURL, token, projectID, connectionID string) []kafkaPreviewPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodGet, baseURL+"/api/v1/data/projects/"+projectID+"/kafka/configs/"+connectionID+"/preview", token, nil)
	var result []kafkaPreviewPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode kafka preview response failed: %v", err)
	}
	return result
}

func mustCreateHTTPConfig(t *testing.T, baseURL, token, projectID string, payload map[string]any) protocolWave1ConnectionPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodPost, baseURL+"/api/v1/data/projects/"+projectID+"/http/configs", token, payload)
	return decodeProtocolWave1Connection(t, responseEnvelope.Data)
}

func mustCreateWebSocketConfig(t *testing.T, baseURL, token, projectID string, payload map[string]any) protocolWave1ConnectionPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodPost, baseURL+"/api/v1/data/projects/"+projectID+"/websocket/configs", token, payload)
	return decodeProtocolWave1Connection(t, responseEnvelope.Data)
}

func mustCreateRedisConfig(t *testing.T, baseURL, token, projectID string, payload map[string]any) protocolWave1ConnectionPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodPost, baseURL+"/api/v1/data/projects/"+projectID+"/redis/configs", token, payload)
	return decodeProtocolWave1Connection(t, responseEnvelope.Data)
}

func decodeProtocolWave1Connection(t *testing.T, raw json.RawMessage) protocolWave1ConnectionPayload {
	t.Helper()

	var payload protocolWave1ConnectionPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode protocol wave1 connection response failed: %v", err)
	}
	if payload.ID == "" {
		t.Fatal("expected protocol connection id in response")
	}
	return payload
}
