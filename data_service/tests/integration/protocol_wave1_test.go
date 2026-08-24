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
	migrator := setupSchemaInitializer(t, fixture.pool)
	if err := migrator.Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "protocol-wave1-secret-01"

	srv, err := app.NewServer(config.Config{
		Addr:                       ":0",
		DatabaseURL:                fixture.databaseURL,
		DatabaseSearchPath:         fixture.schemaName,
		JWTSecret:                  secret,
		ConnectionSecretKey:        []byte("0123456789abcdef0123456789abcdef"),
		ConnectionSecretKeyVersion: "v1",
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
		"secrets":       map[string]string{"option.password": "kafka-first-secret"},
	})
	if kafka.Type != "kafka" {
		t.Fatalf("expected kafka type, got %q", kafka.Type)
	}
	assertUpdatedProtocolConnection(t, doJSONRequest(t, http.MethodPut, server.URL+"/api/v1/data/projects/"+projectID+"/kafka/configs/"+kafka.ID, token, map[string]any{
		"name": "kafka-source-updated", "status": "disconnected", "brokers": "127.0.0.1:9093", "topic": "factory.events", "consumerGroup": "dc-wave1", "startPosition": "latest",
		"secrets": map[string]string{"option.password": "kafka-second-secret"},
	}), "kafka-source-updated")

	httpConn := mustCreateHTTPConfig(t, server.URL, token, projectID, map[string]any{
		"name":        "http-source-main",
		"description": "请求由开发态 HTTP 工作台维护",
	})
	if httpConn.Type != "http" {
		t.Fatalf("expected http type, got %q", httpConn.Type)
	}
	assertUpdatedProtocolConnection(t, doJSONRequest(t, http.MethodPut, server.URL+"/api/v1/data/projects/"+projectID+"/http/configs/"+httpConn.ID, token, map[string]any{
		"name": "http-source-updated", "status": "disconnected", "description": "updated",
	}), "http-source-updated")

	wsConn := mustCreateWebSocketConfig(t, server.URL, token, projectID, map[string]any{
		"name":        "ws-source-main",
		"description": "WebSocket 会话在工作台维护",
	})
	if wsConn.Type != "websocket" {
		t.Fatalf("expected websocket type, got %q", wsConn.Type)
	}
	assertUpdatedProtocolConnection(t, doJSONRequest(t, http.MethodPut, server.URL+"/api/v1/data/projects/"+projectID+"/websocket/configs/"+wsConn.ID, token, map[string]any{
		"name": "ws-source-updated", "status": "disconnected", "description": "updated",
	}), "ws-source-updated")

	redisConn := mustCreateRedisConfig(t, server.URL, token, projectID, map[string]any{
		"name":       "redis-source-main",
		"address":    "127.0.0.1:6379",
		"keyPattern": "factory:*",
		"secrets":    map[string]string{"password": "redis-first-secret"},
	})
	if redisConn.Type != "redis" {
		t.Fatalf("expected redis type, got %q", redisConn.Type)
	}
	assertUpdatedProtocolConnection(t, doJSONRequest(t, http.MethodPut, server.URL+"/api/v1/data/projects/"+projectID+"/redis/configs/"+redisConn.ID, token, map[string]any{
		"name": "redis-source-updated", "status": "disconnected", "address": "127.0.0.1:6380", "keyPattern": "factory:*", "mode": "standalone",
		"secrets": map[string]string{"password": "redis-second-secret"},
	}), "redis-source-updated")

	var plaintextMatches int
	if err := fixture.pool.QueryRow(ctx, `SELECT COUNT(*) FROM data_connection_secrets WHERE connection_id IN ($1, $2) AND encrypted_value IN (convert_to('kafka-first-secret', 'UTF8'), convert_to('kafka-second-secret', 'UTF8'), convert_to('redis-first-secret', 'UTF8'), convert_to('redis-second-secret', 'UTF8'))`, kafka.ID, redisConn.ID).Scan(&plaintextMatches); err != nil {
		t.Fatalf("check protocol secret ciphertext failed: %v", err)
	}
	if plaintextMatches != 0 {
		t.Fatal("protocol secrets must not be stored as plaintext")
	}
}

func assertUpdatedProtocolConnection(t *testing.T, envelope apiEnvelope, expectedName string) {
	t.Helper()
	updated := decodeProtocolWave1Connection(t, envelope.Data)
	if updated.Name != expectedName {
		t.Fatalf("expected updated connection name %q, got %q", expectedName, updated.Name)
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
