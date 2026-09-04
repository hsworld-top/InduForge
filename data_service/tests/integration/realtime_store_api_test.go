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
	redisServer := miniredis.RunT(t)
	srv, err := app.NewServer(config.Config{DataServiceInternalToken: "integration-test-internal-token",
		Addr:                       ":0",
		DatabaseURL:                fixture.databaseURL,
		DatabaseSearchPath:         fixture.schemaName,
		JWTSecret:                  secret,
		ConnectionSecretKey:        []byte("0123456789abcdef0123456789abcdef"),
		ConnectionSecretKeyVersion: "v1",
		RedisAddr:                  redisServer.Addr(),
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

	builtinConnection := mustCreateConnection(t, server.URL, token, projectID, map[string]any{
		"name":   "IF实时库",
		"type":   "builtin.realtime",
		"config": map[string]any{},
	})
	redisConnection := mustCreateRedisConfig(t, server.URL, token, projectID, map[string]any{
		"name":       "Redis",
		"address":    redisServer.Addr(),
		"mode":       "standalone",
		"keyPattern": "codex:review:*",
	})
	for _, connectionID := range []string{builtinConnection.ID, redisConnection.ID} {
		responseEnvelope := doJSONRequest(
			t,
			http.MethodGet,
			server.URL+"/api/v1/data/projects/"+projectID+"/realtime-stores/"+connectionID+"/keys",
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

	keyEndpoint := server.URL + "/api/v1/data/projects/" + projectID + "/realtime-stores/" + redisConnection.ID + "/key"
	savedEnvelope := doJSONRequest(t, http.MethodPut, keyEndpoint, token, map[string]any{
		"key": "aa:bb:cc", "type": "string", "ttlSeconds": 0, "valueType": "json", "value": `{}`,
	})
	var saved struct {
		DataPointPath string `json:"dataPointPath"`
		Outputs       []struct {
			DisplayName string `json:"displayName"`
			DataType    string `json:"dataType"`
		} `json:"outputs"`
	}
	if err := json.Unmarshal(savedEnvelope.Data, &saved); err != nil {
		t.Fatalf("解析 Redis Key 保存结果失败: %v", err)
	}
	if saved.DataPointPath != "redis.Redis.aa.bb.cc" {
		t.Fatalf("Redis 保存后应自动生成数据点，路径错误: %q", saved.DataPointPath)
	}
	if len(saved.Outputs) != 1 || saved.Outputs[0].DisplayName != "cc" || saved.Outputs[0].DataType != "object" {
		t.Fatalf("Redis 自动数据点名称或类型错误: %#v", saved.Outputs)
	}
	responseEnvelope := doJSONRequest(
		t, http.MethodGet,
		server.URL+"/api/v1/data/projects/"+projectID+"/realtime-stores/"+redisConnection.ID+"/keys",
		token, nil,
	)
	var listed struct {
		List []struct {
			Key string `json:"key"`
		} `json:"list"`
	}
	if err := json.Unmarshal(responseEnvelope.Data, &listed); err != nil {
		t.Fatalf("解析 Redis Key 列表失败: %v", err)
	}
	found := false
	for _, item := range listed.List {
		if item.Key == "aa:bb:cc" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("工作台保存的 Key 不应被连接扫描规则隐藏")
	}

	builtinKeyEndpoint := server.URL + "/api/v1/data/projects/" + projectID + "/realtime-stores/" + builtinConnection.ID + "/key"
	builtinSavedEnvelope := doJSONRequest(t, http.MethodPut, builtinKeyEndpoint, token, map[string]any{
		"key": "device:line1:online", "type": "string", "ttlSeconds": 0, "valueType": "json", "value": `true`,
	})
	var builtinSaved struct {
		DataPointPath string `json:"dataPointPath"`
		Outputs       []struct {
			DisplayName string `json:"displayName"`
			DataType    string `json:"dataType"`
		} `json:"outputs"`
	}
	if err := json.Unmarshal(builtinSavedEnvelope.Data, &builtinSaved); err != nil {
		t.Fatalf("解析内置实时库 Key 保存结果失败: %v", err)
	}
	if builtinSaved.DataPointPath != "realtime.IF实时库.device.line1.online" {
		t.Fatalf("内置实时库保存后应自动生成数据点，路径错误: %q", builtinSaved.DataPointPath)
	}
	if len(builtinSaved.Outputs) != 1 || builtinSaved.Outputs[0].DisplayName != "online" || builtinSaved.Outputs[0].DataType != "bool" {
		t.Fatalf("内置实时库自动数据点名称或类型错误: %#v", builtinSaved.Outputs)
	}
}

func TestRealtimeStoreDataPointRecreateDoesNotReuseDeletedMapping(t *testing.T) {
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
	redisServer := miniredis.RunT(t)
	srv, err := app.NewServer(config.Config{DataServiceInternalToken: "integration-test-internal-token",
		Addr:                       ":0",
		DatabaseURL:                fixture.databaseURL,
		DatabaseSearchPath:         fixture.schemaName,
		JWTSecret:                  secret,
		ConnectionSecretKey:        []byte("0123456789abcdef0123456789abcdef"),
		ConnectionSecretKeyVersion: "v1",
		RedisAddr:                  redisServer.Addr(),
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
	first := realtimeDataPointFromSavedKey(t, doJSONRequest(t, http.MethodPut, keyEndpoint, token, savePayload).Data)
	if first.Path != "realtime.IF实时库.test11" {
		t.Fatalf("expected first path realtime.IF实时库.test11, got %q", first.Path)
	}

	doJSONRequest(t, http.MethodDelete, keyEndpoint+"?key=test11", token, nil)
	second := realtimeDataPointFromSavedKey(t, doJSONRequest(t, http.MethodPut, keyEndpoint, token, savePayload).Data)
	if second.Path == first.Path {
		t.Fatalf("删除输出映射后重新建点不应复用已失效映射路径 %q", first.Path)
	}
	if second.ID == first.ID {
		t.Fatalf("删除输出映射后重新建点不应复用已失效数据点 %q", first.ID)
	}
}

func realtimeDataPointFromSavedKey(t *testing.T, raw json.RawMessage) dataPointPayload {
	t.Helper()
	var saved struct {
		DataPointID   string `json:"dataPointId"`
		DataPointPath string `json:"dataPointPath"`
	}
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatalf("解析实时库 Key 自动数据点失败: %v", err)
	}
	if saved.DataPointID == "" || saved.DataPointPath == "" {
		t.Fatalf("保存 Key 后应返回自动生成的数据点: %#v", saved)
	}
	return dataPointPayload{ID: saved.DataPointID, Path: saved.DataPointPath}
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
	var created struct {
		DataPointID   string `json:"dataPointId"`
		DataPointPath string `json:"dataPointPath"`
	}
	if err := json.Unmarshal(responseEnvelope.Data, &created); err != nil {
		t.Fatalf("解析实时库数据点失败: %v", err)
	}
	return dataPointPayload{ID: created.DataPointID, Path: created.DataPointPath}
}
