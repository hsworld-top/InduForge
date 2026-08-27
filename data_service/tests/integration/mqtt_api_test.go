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
	migrator := setupSchemaInitializer(t, fixture.pool)
	if err := migrator.Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "mqtt-lifecycle-secret-01"

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
		"secrets":   map[string]string{"password": "mqtt-first-secret"},
	})
	updated := doJSONRequest(t, http.MethodPut, server.URL+"/api/v1/data/projects/"+projectID+"/mqtt/connections/"+connection.ID, token, map[string]any{
		"name": "mqtt-main-updated", "brokerUrl": "tcp://localhost:1883", "protocol": "mqtt", "port": 1883, "qos": 1,
		"secrets": map[string]string{"password": "mqtt-second-secret"},
	})
	var updatedConnection mqttConnectionPayload
	if err := json.Unmarshal(updated.Data, &updatedConnection); err != nil || updatedConnection.Name != "mqtt-main-updated" {
		t.Fatalf("expected dedicated MQTT update, got %#v, err=%v", updatedConnection, err)
	}
	var encrypted []byte
	var keyVersion string
	if err := fixture.pool.QueryRow(ctx, `SELECT encrypted_value, encryption_key_version FROM data_connection_secrets WHERE connection_id = $1 AND secret_key = 'password'`, connection.ID).Scan(&encrypted, &keyVersion); err != nil {
		t.Fatalf("read encrypted MQTT secret failed: %v", err)
	}
	if string(encrypted) == "mqtt-first-secret" || string(encrypted) == "mqtt-second-secret" || keyVersion != "v1" {
		t.Fatalf("MQTT secret was not encrypted/versioned: value=%q version=%q", string(encrypted), keyVersion)
	}

	subscriptionID := insertTestMqttSubscription(t, ctx, fixture, projectID, connection.ID, userID)

	startFailure, _ := doJSONRequestAllowStatus(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/mqtt/connections/"+connection.ID+"/start", token, nil)
	if startFailure.Code == 0 {
		t.Fatal("未启动 MQTT Broker 时不得伪造连接成功")
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
	migrator := setupSchemaInitializer(t, fixture.pool)
	if err := migrator.Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "mqtt-chinese-subscription-secret-01"

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
		"usageMode":        "raw_datapoint",
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
	if point.Path != "mqtt.aaaa.撒大苏打" {
		t.Fatalf("expected datapoint path mqtt.aaaa.撒大苏打, got %q", point.Path)
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
	// 列表读取不应修复持久化状态；来源生命周期由保存/删除事务显式维护。
	if refreshed.DataPoints[0].Status != "invalid" {
		t.Fatalf("expected persisted invalid status after read, got %q", refreshed.DataPoints[0].Status)
	}
}

func TestMqttSubscriptionDataPointValidWithoutMqttConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupSchemaInitializer(t, fixture.pool)
	if err := migrator.Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "mqtt-subscription-no-config-secret-01"

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
		TenantID:     "tenant-mqtt-no-config",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	connectionID := uuid.NewString()
	if _, err := fixture.pool.Exec(ctx, `
		INSERT INTO data_connections (id, project_id, name, type, category, is_enabled, metadata, created_by, updated_by)
		VALUES ($1, $2, 'aaaa', 'mqtt', 'protocol', true, '{}'::jsonb, $3, $3)
	`, connectionID, projectID, userID); err != nil {
		t.Fatalf("insert mqtt connection without config failed: %v", err)
	}

	subscription := mustCreateMqttSubscription(t, server.URL, token, projectID, connectionID, map[string]any{
		"name":             "仅订阅测试",
		"topic":            "only/subscription",
		"qos":              0,
		"usageMode":        "raw_datapoint",
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

func TestMqttTagsListSupportsPaginationAndSearch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupSchemaInitializer(t, fixture.pool)
	if err := migrator.Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "mqtt-tag-pagination-secret-01"

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
		TenantID:     "tenant-mqtt-tag-pagination",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	connection := mustCreateMqttConnection(t, server.URL, token, projectID, map[string]any{
		"name":      "mqtt-tags",
		"brokerUrl": "tcp://localhost:1883",
		"protocol":  "mqtt",
		"port":      1883,
		"qos":       1,
	})
	subscriptionID := insertTestMqttSubscription(t, ctx, fixture, projectID, connection.ID, userID)
	insertTestMqttTagWithName(t, ctx, fixture, projectID, subscriptionID, userID, "tag-a", "tag_a", 1)
	insertTestMqttTagWithName(t, ctx, fixture, projectID, subscriptionID, userID, "tag-b", "tag_b", 2)
	insertTestMqttTagWithName(t, ctx, fixture, projectID, subscriptionID, userID, "tag-root", "tag_root", 3)

	paged := mustListMqttTags(t, server.URL, token, projectID, subscriptionID, "page=1&pageSize=2")
	if paged.Pagination.Total != 3 || len(paged.List) != 2 {
		t.Fatalf("expected paged mqtt tags, got %#v", paged)
	}
	filtered := mustListMqttTags(t, server.URL, token, projectID, subscriptionID, "search=tag_b&page=1&pageSize=20")
	if filtered.Pagination.Total != 1 || len(filtered.List) != 1 || filtered.List[0].Code != "tag_b" {
		t.Fatalf("expected searched mqtt tag, got %#v", filtered)
	}
}

func TestMqttTagCreateRollsBackWhenGeneratedPointConflicts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	if err := setupSchemaInitializer(t, fixture.pool).Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}
	projectID, userID := uuid.NewString(), uuid.NewString()
	secret := "mqtt-tag-transaction-secret-01"
	srv, err := app.NewServer(config.Config{
		Addr: ":0", DatabaseURL: fixture.databaseURL, DatabaseSearchPath: fixture.schemaName, JWTSecret: secret,
		ConnectionSecretKey: []byte("0123456789abcdef0123456789abcdef"), ConnectionSecretKeyVersion: "v1",
	})
	if err != nil {
		t.Fatalf("create server failed: %v", err)
	}
	t.Cleanup(srv.Close)
	server := httptest.NewServer(srv.Handler())
	t.Cleanup(server.Close)
	token := mustSignIntegrationJWT(t, secret, &auth.Claims{UserID: userID, TenantID: "tenant-mqtt-tx", ProjectIDs: []string{projectID}, Capabilities: []string{"project:read", "project:write"}})
	connection := mustCreateMqttConnection(t, server.URL, token, projectID, map[string]any{
		"name": "mqtt-tags", "brokerUrl": "tcp://localhost:1883", "protocol": "mqtt", "port": 1883,
	})
	subscriptionID := insertTestMqttSubscription(t, ctx, fixture, projectID, connection.ID, userID)
	if _, err := fixture.pool.Exec(ctx, `
		INSERT INTO data_points(project_id,path,name,source_type,source_config,data_type,tags,attribute_defaults,
		 refresh_mode,status,runtime_permissions,created_by,updated_by)
		VALUES($1,'mqtt.mqtt-tags.sub-temp.temperature','occupied','manual','{}'::jsonb,'float64','[]'::jsonb,
		 '{}'::jsonb,'manual','active','{"write":{"inherit":true,"denyRoles":[],"allowRoles":[]}}'::jsonb,$2,$2)
	`, projectID, userID); err != nil {
		t.Fatalf("insert conflicting datapoint failed: %v", err)
	}

	response, status := doJSONRequestAllowStatus(t, http.MethodPost,
		server.URL+"/api/v1/data/projects/"+projectID+"/mqtt/subscriptions/"+subscriptionID+"/tags", token,
		map[string]any{"name": "temperature", "code": "temperature", "dataType": "float64", "parseType": "jsonpath", "parseRule": "$.value"})
	if response.Code == 0 {
		t.Fatalf("generated point conflict must reject MQTT tag create: status=%d response=%#v", status, response)
	}
	var count int
	if err := fixture.pool.QueryRow(ctx, `SELECT COUNT(*) FROM data_mqtt_tags WHERE project_id=$1 AND code='temperature'`, projectID).Scan(&count); err != nil {
		t.Fatalf("count MQTT tag failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("MQTT tag main record must roll back with datapoint failure, count=%d", count)
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

type mqttTagPayload struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

type mqttListResponse[T any] struct {
	List       []T                `json:"list"`
	Pagination mqttPaginationData `json:"pagination"`
}

func mustListMqttTags(t *testing.T, baseURL, token, projectID, subscriptionID, rawQuery string) mqttListResponse[mqttTagPayload] {
	t.Helper()

	endpoint := baseURL + "/api/v1/data/projects/" + projectID + "/mqtt/subscriptions/" + subscriptionID + "/tags"
	if rawQuery != "" {
		endpoint += "?" + rawQuery
	}
	responseEnvelope := doJSONRequest(t, http.MethodGet, endpoint, token, nil)
	var result mqttListResponse[mqttTagPayload]
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode mqtt tags response failed: %v", err)
	}
	return result
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

func insertTestMqttTagWithName(t *testing.T, ctx context.Context, fixture *testDatabase, projectID, subscriptionID string, userID, name, code string, order int) string {
	t.Helper()

	tagID := uuid.NewString()
	_, err := fixture.pool.Exec(ctx, `
		INSERT INTO data_mqtt_tags (
			id,
			project_id,
			subscription_id,
			name,
			code,
			data_type,
			parse_type,
			parse_rule,
			validation,
			display_order,
			created_by,
			updated_by
		)
		VALUES ($1, $2, $3, $4, $5, 'float64', 'jsonpath', '$.value', '{}'::jsonb, $6, $7, $7)
	`, tagID, projectID, subscriptionID, name, code, order, userID)
	if err != nil {
		t.Fatalf("insert mqtt tag %s failed: %v", code, err)
	}
	return tagID
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
