package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"

	"github.com/indu-forge/data_service/internal/app"
	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/config"
)

func TestPreviewSessionSlidingTTL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupSchemaInitializer(t, fixture.pool)
	if err := migrator.Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}

	redisServer, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis failed: %v", err)
	}
	t.Cleanup(redisServer.Close)

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "preview-session-secret-01"

	srv, err := app.NewServer(config.Config{DataServiceInternalToken: "integration-test-internal-token",
		Addr:                       ":0",
		DatabaseURL:                fixture.databaseURL,
		DatabaseSearchPath:         fixture.schemaName,
		JWTSecret:                  secret,
		ConnectionSecretKey:        []byte("0123456789abcdef0123456789abcdef"),
		ConnectionSecretKeyVersion: "v1",
		RedisAddr:                  redisServer.Addr(),
		RedisDB:                    0,
	})
	if err != nil {
		t.Fatalf("create server failed: %v", err)
	}
	t.Cleanup(srv.Close)

	server := httptest.NewServer(srv.Handler())
	t.Cleanup(server.Close)

	token := mustSignIntegrationJWT(t, secret, &auth.Claims{
		UserID:       userID,
		TenantID:     "tenant-preview",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	session := mustCreatePreviewSession(t, server.URL, token, projectID, map[string]any{
		"meta": map[string]any{
			"scene": "integration-test",
		},
	})
	if session.Status != "active" {
		t.Fatalf("expected status active, got %q", session.Status)
	}

	mustHeartbeatPreviewSession(t, server.URL, token, session.ID)

	ttl := redisServer.TTL(previewSessionRedisKey(session.ID))
	if ttl <= 0 {
		t.Fatalf("expected ttl > 0, got %s", ttl)
	}

	mustDeletePreviewSession(t, server.URL, token, session.ID)

	if redisServer.Exists(previewSessionRedisKey(session.ID)) {
		t.Fatalf("expected redis key %s to be deleted", previewSessionRedisKey(session.ID))
	}
}

type previewSessionPayload struct {
	ID           string         `json:"id"`
	ProjectID    string         `json:"projectId"`
	UserID       string         `json:"userId"`
	Status       string         `json:"status"`
	StartedAt    time.Time      `json:"startedAt"`
	LastActiveAt time.Time      `json:"lastActiveAt"`
	ExpiredAt    time.Time      `json:"expiredAt"`
	Meta         map[string]any `json:"meta"`
}

func mustCreatePreviewSession(t *testing.T, baseURL, token, projectID string, payload map[string]any) previewSessionPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodPost, baseURL+"/api/v1/data/projects/"+projectID+"/preview/sessions", token, payload)
	var result previewSessionPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode preview session create response failed: %v", err)
	}
	if result.ID == "" {
		t.Fatal("expected preview session id in response")
	}
	return result
}

func mustHeartbeatPreviewSession(t *testing.T, baseURL, token, sessionID string) previewSessionPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodPost, baseURL+"/api/v1/data/preview/sessions/"+sessionID+"/heartbeat", token, map[string]any{})
	var result previewSessionPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode preview heartbeat response failed: %v", err)
	}
	return result
}

func mustDeletePreviewSession(t *testing.T, baseURL, token, sessionID string) {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodDelete, baseURL+"/api/v1/data/preview/sessions/"+sessionID, token, nil)
	var deleted struct {
		Deleted bool `json:"deleted"`
	}
	if err := json.Unmarshal(responseEnvelope.Data, &deleted); err != nil {
		t.Fatalf("decode preview delete response failed: %v", err)
	}
	if !deleted.Deleted {
		t.Fatal("expected deleted=true")
	}
	_ = responseEnvelope
}

func previewSessionRedisKey(sessionID string) string {
	return "preview:session:" + sessionID
}
