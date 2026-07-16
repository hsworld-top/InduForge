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

func TestRealtimeStoreListAfterConnectionCreate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupSchemaInitializer(t, fixture.pool)
	if err := migrator.Ensure(ctx); err != nil {
		t.Fatalf("初始化数据库结构失败: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "realtime-store-secret"
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
		TenantID:     "tenant-realtime",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	for _, input := range []map[string]any{
		{
			"name":   "IF实时库",
			"type":   "builtin.realtime",
			"config": map[string]any{},
		},
		{
			"name":   "Redis",
			"type":   "redis",
			"status": "connected",
			"config": map[string]any{
				"address": "127.0.0.1:6379",
			},
		},
	} {
		created := mustCreateConnection(t, server.URL, token, projectID, input)
		responseEnvelope := doJSONRequest(
			t,
			http.MethodGet,
			server.URL+"/api/v1/data/projects/"+projectID+"/realtime-stores/"+created.ID+"/keys",
			token,
			nil,
		)
		var payload struct {
			List   []any `json:"list"`
			Groups []any `json:"groups"`
		}
		if err := json.Unmarshal(responseEnvelope.Data, &payload); err != nil {
			t.Fatalf("解析实时库 Key 列表失败: %v", err)
		}
		if payload.List == nil || payload.Groups == nil {
			t.Fatalf("实时库 Key 列表响应结构不完整: %#v", payload)
		}
	}
}

func TestRealtimeStoreDataPointRevivesInvalidPath(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupSchemaInitializer(t, fixture.pool)
	if err := migrator.Ensure(ctx); err != nil {
		t.Fatalf("初始化数据库结构失败: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "realtime-store-revive-secret"
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
		TenantID:     "tenant-realtime-revive",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	connection := mustCreateConnection(t, server.URL, token, projectID, map[string]any{
		"name":   "IF实时库",
		"type":   "builtin.realtime",
		"config": map[string]any{},
	})
	keyEndpoint := server.URL + "/api/v1/data/projects/" + projectID + "/realtime-stores/" + connection.ID + "/key"
	savePayload := map[string]any{
		"key":        "test11",
		"type":       "string",
		"ttlSeconds": -1,
		"valueType":  "json",
		"value":      "{}",
	}
	doJSONRequest(t, http.MethodPut, keyEndpoint, token, savePayload)
	first := mustCreateRealtimeKeyDataPoint(t, server.URL, token, projectID, connection.ID, "test11")
	if first.Path != "realtime.test11" {
		t.Fatalf("expected first path realtime.test11, got %q", first.Path)
	}

	doJSONRequest(t, http.MethodDelete, keyEndpoint+"?key=test11", token, nil)
	doJSONRequest(t, http.MethodPut, keyEndpoint, token, savePayload)
	second := mustCreateRealtimeKeyDataPoint(t, server.URL, token, projectID, connection.ID, "test11")
	if second.Path != first.Path {
		t.Fatalf("expected revived path %q, got %q", first.Path, second.Path)
	}
	if second.ID != first.ID {
		t.Fatalf("expected invalid datapoint to be revived, got new id %q want %q", second.ID, first.ID)
	}
}

func mustCreateRealtimeKeyDataPoint(t *testing.T, baseURL, token, projectID, connectionID, key string) dataPointPayload {
	t.Helper()
	responseEnvelope := doJSONRequest(
		t,
		http.MethodPost,
		baseURL+"/api/v1/data/projects/"+projectID+"/realtime-stores/"+connectionID+"/key/datapoint?key="+key,
		token,
		map[string]any{"dataType": "object"},
	)
	var result dataPointPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("解析实时库数据点失败: %v", err)
	}
	return result
}
