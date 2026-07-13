package integration_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/indu-forge/data_service/internal/app"
	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/config"
	apperrors "github.com/indu-forge/data_service/internal/errors"
)

func TestConnectionsCRUD(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupMigrator(t, fixture.pool)
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("执行迁移失败: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	tenantID := "tenant-alpha"
	secret := "connections-secret"

	srv, err := app.NewServer(config.Config{
		Addr:               ":0",
		DatabaseURL:        fixture.databaseURL,
		DatabaseSearchPath: fixture.schemaName,
		JWTSecret:          secret,
	})
	if err != nil {
		t.Fatalf("创建默认服务失败: %v", err)
	}
	t.Cleanup(srv.Close)

	server := httptest.NewServer(srv.Handler())
	t.Cleanup(server.Close)

	token := mustSignIntegrationJWT(t, secret, &auth.Claims{
		UserID:       userID,
		TenantID:     tenantID,
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	initialList := mustListConnections(t, server.URL, token, projectID)
	if len(initialList) != 0 {
		t.Fatalf("期望初始列表为空，实际为 %d", len(initialList))
	}

	created := mustCreateConnection(t, server.URL, token, projectID, map[string]any{
		"name":   "pg-main",
		"type":   "relational",
		"status": "connected",
		"config": map[string]any{
			"host":     "localhost",
			"port":     5432,
			"database": "factory",
		},
	})
	if created.ProjectID != projectID {
		t.Fatalf("期望 projectId 为 %q，实际为 %q", projectID, created.ProjectID)
	}
	if created.TenantID != tenantID {
		t.Fatalf("期望 tenantId 为 %q，实际为 %q", tenantID, created.TenantID)
	}
	if created.Type != "relational" {
		t.Fatalf("期望 type 为 relational，实际为 %q", created.Type)
	}

	listAfterCreate := mustListConnections(t, server.URL, token, projectID)
	if len(listAfterCreate) != 1 {
		t.Fatalf("期望创建后列表长度为 1，实际为 %d", len(listAfterCreate))
	}
	if listAfterCreate[0].ID != created.ID {
		t.Fatalf("期望列表首项 ID 为 %q，实际为 %q", created.ID, listAfterCreate[0].ID)
	}

	updated := mustUpdateConnection(t, server.URL, token, projectID, created.ID, map[string]any{
		"name":   "pg-main-2",
		"status": "disconnected",
		"config": map[string]any{
			"host":     "127.0.0.1",
			"port":     5432,
			"database": "factory_v2",
		},
	})
	if updated.Name != "pg-main-2" {
		t.Fatalf("期望更新后的名称为 pg-main-2，实际为 %q", updated.Name)
	}
	if updated.Status != "disconnected" {
		t.Fatalf("期望更新后的状态为 disconnected，实际为 %q", updated.Status)
	}
	if updated.Config["database"] != "factory_v2" {
		t.Fatalf("期望更新后的 config.database 为 factory_v2，实际为 %#v", updated.Config["database"])
	}

	mustDeleteConnection(t, server.URL, token, projectID, created.ID)

	finalList := mustListConnections(t, server.URL, token, projectID)
	if len(finalList) != 0 {
		t.Fatalf("期望删除后列表为空，实际为 %d", len(finalList))
	}
}

func TestConnectionsRejectPhase2ReservedTypes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupMigrator(t, fixture.pool)
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("执行迁移失败: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "connections-phase2-secret"

	srv, err := app.NewServer(config.Config{
		Addr:               ":0",
		DatabaseURL:        fixture.databaseURL,
		DatabaseSearchPath: fixture.schemaName,
		JWTSecret:          secret,
	})
	if err != nil {
		t.Fatalf("创建默认服务失败: %v", err)
	}
	t.Cleanup(srv.Close)

	server := httptest.NewServer(srv.Handler())
	t.Cleanup(server.Close)

	token := mustSignIntegrationJWT(t, secret, &auth.Claims{
		UserID:       userID,
		TenantID:     "tenant-phase2",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	assertPhaseBoundaryError(t, doJSONRequestWithStatus(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/connections", token, map[string]any{
		"name":   "opcua-legacy",
		"type":   "opcua",
		"status": "connected",
		"config": map[string]any{
			"endpoint": "opc.tcp://127.0.0.1:4840",
		},
	}, http.StatusOK), "OPC UA")

	currentList := mustListConnections(t, server.URL, token, projectID)
	if len(currentList) != 0 {
		t.Fatalf("expected no connections after rejected phase boundary request, got %d", len(currentList))
	}
}

type connectionPayload struct {
	ID        string         `json:"id"`
	ProjectID string         `json:"projectId"`
	TenantID  string         `json:"tenantId"`
	Name      string         `json:"name"`
	Type      string         `json:"type"`
	Status    string         `json:"status"`
	Config    map[string]any `json:"config"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

type apiEnvelope struct {
	Code  int             `json:"code"`
	Msg   string          `json:"msg"`
	ReqID string          `json:"reqId"`
	Data  json.RawMessage `json:"data"`
}

var integrationHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
}

func mustListConnections(t *testing.T, baseURL, token, projectID string) []connectionPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodGet, baseURL+"/api/v1/data/projects/"+projectID+"/connections", token, nil)
	var connections []connectionPayload
	if err := json.Unmarshal(responseEnvelope.Data, &connections); err != nil {
		t.Fatalf("解析连接列表失败: %v", err)
	}
	return connections
}

func mustCreateConnection(t *testing.T, baseURL, token, projectID string, payload map[string]any) connectionPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodPost, baseURL+"/api/v1/data/projects/"+projectID+"/connections", token, payload)
	return decodeConnectionPayload(t, responseEnvelope.Data)
}

func mustUpdateConnection(t *testing.T, baseURL, token, projectID, connectionID string, payload map[string]any) connectionPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodPut, baseURL+"/api/v1/data/projects/"+projectID+"/connections/"+connectionID, token, payload)
	return decodeConnectionPayload(t, responseEnvelope.Data)
}

func mustDeleteConnection(t *testing.T, baseURL, token, projectID, connectionID string) {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodDelete, baseURL+"/api/v1/data/projects/"+projectID+"/connections/"+connectionID, token, nil)
	var deleted struct {
		Deleted bool `json:"deleted"`
	}
	if err := json.Unmarshal(responseEnvelope.Data, &deleted); err != nil {
		t.Fatalf("解析删除响应失败: %v", err)
	}
	if !deleted.Deleted {
		t.Fatal("期望删除响应标记 deleted=true")
	}
}

func decodeConnectionPayload(t *testing.T, raw json.RawMessage) connectionPayload {
	t.Helper()

	var payload connectionPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("解析连接响应失败: %v", err)
	}
	if payload.ID == "" {
		t.Fatal("期望连接响应包含 id")
	}
	return payload
}

func doJSONRequest(t *testing.T, method, url, token string, payload any) apiEnvelope {
	t.Helper()

	var bodyBytes []byte
	if payload != nil {
		var err error
		bodyBytes, err = json.Marshal(payload)
		if err != nil {
			t.Fatalf("序列化请求体失败: %v", err)
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("创建请求失败: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "rid-connections-it")

	resp, err := integrationHTTPClient.Do(req)
	if err != nil {
		t.Fatalf("执行请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("期望状态码为 200，实际为 %d，响应体=%s", resp.StatusCode, string(body))
	}

	var envelope apiEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if envelope.Code != apperrors.SuccessCode {
		t.Fatalf("期望 code=%d，实际 code=%d msg=%q", apperrors.SuccessCode, envelope.Code, envelope.Msg)
	}
	if envelope.ReqID == "" {
		t.Fatal("期望响应包含 reqId")
	}

	return envelope
}

func mustSignIntegrationJWT(t *testing.T, secret string, claims *auth.Claims) string {
	t.Helper()

	headerJSON := []byte(`{"alg":"HS256","typ":"JWT"}`)
	now := time.Now().UTC()
	payload := map[string]any{
		"userId":       claims.UserID,
		"tenantId":     claims.TenantID,
		"role":         claims.Role,
		"projectIds":   claims.ProjectIDs,
		"capabilities": claims.Capabilities,
		"exp":          now.Add(time.Hour).Unix(),
		"nbf":          now.Add(-time.Minute).Unix(),
		"iat":          now.Add(-time.Minute).Unix(),
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("序列化 JWT payload 失败: %v", err)
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := encodedHeader + "." + encodedPayload

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signingInput))
	signature := mac.Sum(nil)

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
}
