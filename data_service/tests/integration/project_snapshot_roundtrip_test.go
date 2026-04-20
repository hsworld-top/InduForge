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
	"github.com/indu-forge/data_service/internal/repository"
)

func TestProjectSnapshotGetAndReplaceRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupMigrator(t, fixture.pool)
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("migrate up failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "snapshot-roundtrip-secret-01"

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
		TenantID:     "tenant-snapshot",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	relational := mustCreateConnection(t, server.URL, token, projectID, map[string]any{
		"name":   "pg-main",
		"type":   "relational",
		"status": "connected",
		"config": map[string]any{
			"dbType":   "postgresql",
			"host":     "127.0.0.1",
			"port":     5432,
			"database": "factory",
			"username": "postgres",
			"password": "postgres",
		},
	})
	query := mustCreateQuery(t, server.URL, token, projectID, map[string]any{
		"name":         "query-main",
		"connectionId": relational.ID,
		"queryType":    "sql",
		"config": map[string]any{
			"sql": "SELECT 1 AS value",
		},
	})
	insertOneDataPoint(t, ctx, fixture, projectID, userID, "metrics.main.temp", "active")

	mqttConnection := mustCreateMqttConnection(t, server.URL, token, projectID, map[string]any{
		"name":      "mqtt-main",
		"brokerUrl": "tcp://localhost:1883",
		"protocol":  "mqtt",
		"port":      1883,
	})
	subscriptionID := insertTestMqttSubscription(t, ctx, fixture, projectID, mqttConnection.ID, userID)
	groupID := insertTestMqttTagGroup(t, ctx, fixture, projectID, subscriptionID, userID)
	tagID := insertTestMqttTag(t, ctx, fixture, projectID, subscriptionID, groupID, userID)

	currentSnapshot := mustGetProjectSnapshot(t, server.URL, token, projectID)
	if len(currentSnapshot.Connections) != 2 {
		t.Fatalf("expected 2 connections in snapshot, got %d", len(currentSnapshot.Connections))
	}
	if len(currentSnapshot.RelationalConfigs) != 1 {
		t.Fatalf("expected 1 relational config in snapshot, got %d", len(currentSnapshot.RelationalConfigs))
	}
	if currentSnapshot.RelationalConfigs[0].ConnectionID != relational.ID {
		t.Fatalf("expected relational config to point to %q, got %q", relational.ID, currentSnapshot.RelationalConfigs[0].ConnectionID)
	}
	if len(currentSnapshot.Queries) != 1 || currentSnapshot.Queries[0].ID != query.ID {
		t.Fatalf("expected snapshot queries include %q", query.ID)
	}
	if len(currentSnapshot.DataPoints) != 1 || currentSnapshot.DataPoints[0].Path != "metrics.main.temp" {
		t.Fatal("expected snapshot datapoints include metrics.main.temp")
	}
	if len(currentSnapshot.MqttConfigs) != 1 || currentSnapshot.MqttConfigs[0].ConnectionID != mqttConnection.ID {
		t.Fatalf("expected mqtt config to point to %q", mqttConnection.ID)
	}
	if len(currentSnapshot.MqttSubscriptions) != 1 || currentSnapshot.MqttSubscriptions[0].ID != subscriptionID {
		t.Fatalf("expected mqtt subscription %q in snapshot", subscriptionID)
	}
	if len(currentSnapshot.MqttTagGroups) != 1 || currentSnapshot.MqttTagGroups[0].ID != groupID {
		t.Fatalf("expected mqtt tag group %q in snapshot", groupID)
	}
	if len(currentSnapshot.MqttTags) != 1 || currentSnapshot.MqttTags[0].ID != tagID {
		t.Fatalf("expected mqtt tag %q in snapshot", tagID)
	}

	replacementRelationalID := uuid.NewString()
	replacementMqttID := uuid.NewString()
	replacementQueryID := uuid.NewString()
	replacementSubscriptionID := uuid.NewString()
	replacementGroupID := uuid.NewString()
	replacementTagID := uuid.NewString()
	replacementSourceID := replacementQueryID
	refreshInterval := 5000

	replacement := repository.ProjectSnapshot{
		Connections: []repository.ConnectionRecord{
			{
				ID:        replacementRelationalID,
				ProjectID: projectID,
				Name:      "pg-replaced",
				Type:      "relational",
				Status:    "connected",
				Config: map[string]any{
					"dbType":   "postgresql",
					"host":     "10.0.0.9",
					"port":     5432,
					"database": "factory_v2",
					"username": "svc_user",
					"password": "svc_pass",
				},
			},
			{
				ID:        replacementMqttID,
				ProjectID: projectID,
				Name:      "mqtt-replaced",
				Type:      "mqtt",
				Status:    "connected",
				Config: map[string]any{
					"brokerUrl": "tcp://127.0.0.1:1884",
				},
			},
		},
		Queries: []repository.QueryRecord{
			{
				ID:              replacementQueryID,
				ProjectID:       projectID,
				ConnectionID:    replacementRelationalID,
				Name:            "query-replaced",
				QueryType:       "sql",
				Config:          map[string]any{"sql": "SELECT 2 AS value"},
				IsEnabled:       true,
				TimeoutMS:       15000,
				CacheEnabled:    true,
				CacheTtlSeconds: 30,
			},
		},
		MqttConfigs: []repository.SnapshotMqttConfigRecord{
			{
				ConnectionID:      replacementMqttID,
				BrokerURL:         "tcp://127.0.0.1:1884",
				Protocol:          "mqtt",
				Port:              1884,
				Keepalive:         60,
				CleanSession:      true,
				QOS:               1,
				ReconnectPeriodMS: 2000,
				ConnectTimeoutMS:  30000,
				Will:              map[string]any{},
				SSLConfig:         map[string]any{},
			},
		},
		MqttSubscriptions: []repository.SnapshotMqttSubscriptionRecord{
			{
				ID:               replacementSubscriptionID,
				ProjectID:        projectID,
				ConnectionID:     replacementMqttID,
				Name:             "sub-replaced",
				Topic:            "factory/line1/temp",
				QOS:              1,
				IsEnabled:        true,
				MessageRetention: 50,
			},
		},
		MqttTagGroups: []repository.SnapshotMqttTagGroupRecord{
			{
				ID:             replacementGroupID,
				ProjectID:      projectID,
				SubscriptionID: replacementSubscriptionID,
				Name:           "group-replaced",
				Code:           "group_replaced",
				Order:          1,
			},
		},
		MqttTags: []repository.SnapshotMqttTagRecord{
			{
				ID:             replacementTagID,
				ProjectID:      projectID,
				SubscriptionID: replacementSubscriptionID,
				GroupID:        &replacementGroupID,
				Name:           "tag-replaced",
				Code:           "tag_replaced",
				DataType:       "number",
				ParseType:      "jsonpath",
				ParseRule:      "$.value",
				Validation:     map[string]any{},
				IsEnabled:      true,
				Order:          1,
			},
		},
		DataPoints: []repository.DataPointRecord{
			{
				ID:                uuid.NewString(),
				ProjectID:         projectID,
				Path:              "metrics.replaced.temp",
				Name:              "metrics.replaced.temp",
				SourceType:        "query",
				SourceID:          &replacementSourceID,
				SourceConfig:      map[string]any{"column": "value"},
				DataType:          "number",
				Tags:              []any{"temperature"},
				RefreshMode:       "interval",
				RefreshIntervalMS: &refreshInterval,
				Status:            "active",
			},
		},
	}

	doJSONRequest(t, http.MethodPut, server.URL+"/api/v1/data/projects/"+projectID+"/snapshot", token, replacement)

	replacedSnapshot := mustGetProjectSnapshot(t, server.URL, token, projectID)
	if len(replacedSnapshot.Connections) != 2 {
		t.Fatalf("expected 2 connections after replace, got %d", len(replacedSnapshot.Connections))
	}
	if !snapshotHasConnection(replacedSnapshot.Connections, replacementRelationalID, "pg-replaced") {
		t.Fatalf("expected replaced relational connection %q in snapshot", replacementRelationalID)
	}
	if !snapshotHasConnection(replacedSnapshot.Connections, replacementMqttID, "mqtt-replaced") {
		t.Fatalf("expected replaced mqtt connection %q in snapshot", replacementMqttID)
	}
	if snapshotHasConnection(replacedSnapshot.Connections, relational.ID, relational.Name) {
		t.Fatalf("did not expect old connection %q after replace", relational.ID)
	}
	if len(replacedSnapshot.Queries) != 1 || replacedSnapshot.Queries[0].ID != replacementQueryID {
		t.Fatalf("expected replaced query %q in snapshot", replacementQueryID)
	}
	if len(replacedSnapshot.MqttConfigs) != 1 || replacedSnapshot.MqttConfigs[0].ConnectionID != replacementMqttID {
		t.Fatalf("expected replaced mqtt config to point to %q", replacementMqttID)
	}
	if len(replacedSnapshot.MqttSubscriptions) != 1 || replacedSnapshot.MqttSubscriptions[0].ID != replacementSubscriptionID {
		t.Fatalf("expected replaced mqtt subscription %q in snapshot", replacementSubscriptionID)
	}
	if len(replacedSnapshot.MqttTagGroups) != 1 || replacedSnapshot.MqttTagGroups[0].ID != replacementGroupID {
		t.Fatalf("expected replaced mqtt tag group %q in snapshot", replacementGroupID)
	}
	if len(replacedSnapshot.MqttTags) != 1 || replacedSnapshot.MqttTags[0].ID != replacementTagID {
		t.Fatalf("expected replaced mqtt tag %q in snapshot", replacementTagID)
	}
	if len(replacedSnapshot.DataPoints) != 1 || replacedSnapshot.DataPoints[0].Path != "metrics.replaced.temp" {
		t.Fatal("expected replaced datapoint metrics.replaced.temp in snapshot")
	}
}

func TestProjectSnapshotReplaceRejectsIncompleteConnections(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupMigrator(t, fixture.pool)
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("migrate up failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "snapshot-roundtrip-secret-02"

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
		TenantID:     "tenant-snapshot",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	original := mustCreateConnection(t, server.URL, token, projectID, map[string]any{
		"name":   "pg-stable",
		"type":   "relational",
		"status": "connected",
		"config": map[string]any{
			"dbType": "postgresql",
			"host":   "127.0.0.1",
			"port":   5432,
		},
	})

	invalidEnvelope := doJSONRequestWithStatus(t, http.MethodPut, server.URL+"/api/v1/data/projects/"+projectID+"/snapshot", token, repository.ProjectSnapshot{
		Connections: []repository.ConnectionRecord{
			{
				ID:        uuid.NewString(),
				ProjectID: projectID,
				Type:      "relational",
				Status:    "connected",
				Config:    map[string]any{},
			},
		},
	}, http.StatusBadRequest)
	if invalidEnvelope.Success {
		t.Fatal("expected replace snapshot response to fail for incomplete connection")
	}

	currentSnapshot := mustGetProjectSnapshot(t, server.URL, token, projectID)
	if len(currentSnapshot.Connections) != 1 {
		t.Fatalf("expected original snapshot to keep 1 connection, got %d", len(currentSnapshot.Connections))
	}
	if currentSnapshot.Connections[0].ID != original.ID {
		t.Fatalf("expected original connection %q to remain after rejected replace, got %q", original.ID, currentSnapshot.Connections[0].ID)
	}
}

func TestProjectSnapshotReplaceRejectsPhase2ReservedConnections(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupMigrator(t, fixture.pool)
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("migrate up failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "snapshot-roundtrip-secret-03"

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
		TenantID:     "tenant-snapshot",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	original := mustCreateConnection(t, server.URL, token, projectID, map[string]any{
		"name":   "pg-stable",
		"type":   "relational",
		"status": "connected",
		"config": map[string]any{
			"dbType": "postgresql",
			"host":   "127.0.0.1",
			"port":   5432,
		},
	})

	assertPhaseBoundaryError(t, doJSONRequestWithStatus(t, http.MethodPut, server.URL+"/api/v1/data/projects/"+projectID+"/snapshot", token, repository.ProjectSnapshot{
		Connections: []repository.ConnectionRecord{
			{
				ID:        uuid.NewString(),
				ProjectID: projectID,
				Name:      "opcua-rejected",
				Type:      "opcua",
				Status:    "connected",
				Config: map[string]any{
					"endpoint": "opc.tcp://127.0.0.1:4840",
				},
			},
		},
	}, http.StatusBadRequest), "OPC UA")

	currentSnapshot := mustGetProjectSnapshot(t, server.URL, token, projectID)
	if len(currentSnapshot.Connections) != 1 {
		t.Fatalf("expected original snapshot to keep 1 connection, got %d", len(currentSnapshot.Connections))
	}
	if currentSnapshot.Connections[0].ID != original.ID {
		t.Fatalf("expected original connection %q to remain after rejected replace, got %q", original.ID, currentSnapshot.Connections[0].ID)
	}
}

func mustGetProjectSnapshot(t *testing.T, baseURL, token, projectID string) repository.ProjectSnapshot {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodGet, baseURL+"/api/v1/data/projects/"+projectID+"/snapshot", token, nil)

	var snapshot repository.ProjectSnapshot
	if err := json.Unmarshal(responseEnvelope.Data, &snapshot); err != nil {
		t.Fatalf("decode snapshot response failed: %v", err)
	}
	return snapshot
}

func snapshotHasConnection(connections []repository.ConnectionRecord, id, name string) bool {
	for _, connection := range connections {
		if connection.ID == id && connection.Name == name {
			return true
		}
	}
	return false
}
