package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/indu-forge/data_service/internal/app"
	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/config"
)

func TestProtocolWave2PhaseBoundary(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupMigrator(t, fixture.pool)
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("migrate up failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "protocol-wave2-secret-01"

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
		TenantID:     "tenant-wave2",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	assertPhaseBoundaryError(t, doJSONRequestWithStatus(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/opcua/configs", token, map[string]any{
		"name":     "opcua-main",
		"endpoint": "opc.tcp://127.0.0.1:4840",
	}, http.StatusBadRequest), "OPC UA")

	assertPhaseBoundaryError(t, doJSONRequestWithStatus(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/s7/configs", token, map[string]any{
		"name": "s7-main",
		"host": "192.168.0.10",
		"rack": 0,
		"slot": 1,
	}, http.StatusBadRequest), "S7")

	assertPhaseBoundaryError(t, doJSONRequestWithStatus(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/modbus/configs", token, map[string]any{
		"name": "modbus-main",
		"mode": "tcp",
		"host": "192.168.0.20",
		"port": 502,
	}, http.StatusBadRequest), "Modbus")

	assertPhaseBoundaryError(t, doJSONRequestWithStatus(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/tdengine/configs", token, map[string]any{
		"name":     "td-main",
		"dsn":      "taos://root:taosdata@127.0.0.1:6030",
		"database": "factory",
	}, http.StatusBadRequest), "TDengine")

	assertPhaseBoundaryError(t, doJSONRequestWithStatus(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/opcda/contracts/validate", token, map[string]any{
		"itemPath":   "Channel1.Device1.TagA",
		"samplingMs": 1000,
	}, http.StatusBadRequest), "OPC DA")
}

func assertPhaseBoundaryError(t *testing.T, envelope apiEnvelope, protocolName string) {
	t.Helper()

	if envelope.Success {
		t.Fatalf("expected %s endpoint to be blocked by phase boundary", protocolName)
	}
	if envelope.ErrorCode != "BAD_REQUEST" {
		t.Fatalf("expected BAD_REQUEST for %s phase boundary, got %q", protocolName, envelope.ErrorCode)
	}
	if !strings.Contains(envelope.Message, "Phase 1 正式范围") {
		t.Fatalf("expected %s phase boundary message, got %q", protocolName, envelope.Message)
	}
}
