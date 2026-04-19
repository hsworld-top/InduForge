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

func TestProtocolWave2(t *testing.T) {
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

	opcuaConn := mustCreateOpcuaConfig(t, server.URL, token, projectID, map[string]any{
		"name":     "opcua-main",
		"endpoint": "opc.tcp://127.0.0.1:4840",
	})
	if opcuaConn.Type != "opcua" {
		t.Fatalf("expected opcua type, got %q", opcuaConn.Type)
	}

	s7Conn := mustCreateS7Config(t, server.URL, token, projectID, map[string]any{
		"name": "s7-main",
		"host": "192.168.0.10",
		"rack": 0,
		"slot": 1,
	})
	if s7Conn.Type != "s7" {
		t.Fatalf("expected s7 type, got %q", s7Conn.Type)
	}

	modbusConn := mustCreateModbusConfig(t, server.URL, token, projectID, map[string]any{
		"name": "modbus-main",
		"mode": "tcp",
		"host": "192.168.0.20",
		"port": 502,
	})
	if modbusConn.Type != "modbus" {
		t.Fatalf("expected modbus type, got %q", modbusConn.Type)
	}

	tdengineConn := mustCreateTdengineConfig(t, server.URL, token, projectID, map[string]any{
		"name":     "td-main",
		"dsn":      "taos://root:taosdata@127.0.0.1:6030",
		"database": "factory",
	})
	if tdengineConn.Type != "tdengine" {
		t.Fatalf("expected tdengine type, got %q", tdengineConn.Type)
	}

	invalidContract := doJSONRequestWithStatus(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/opcda/contracts/validate", token, map[string]any{
		"itemPath":   "",
		"samplingMs": 1000,
	}, http.StatusBadRequest)
	if invalidContract.ErrorCode != "BAD_REQUEST" {
		t.Fatalf("expected BAD_REQUEST for invalid opcda contract, got %q", invalidContract.ErrorCode)
	}

	validContract := doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/opcda/contracts/validate", token, map[string]any{
		"itemPath":   "Channel1.Device1.TagA",
		"samplingMs": 1000,
	})
	var payload struct {
		Valid bool `json:"valid"`
	}
	if err := json.Unmarshal(validContract.Data, &payload); err != nil {
		t.Fatalf("decode opcda validation response failed: %v", err)
	}
	if !payload.Valid {
		t.Fatal("expected opcda contract to be valid")
	}
}

func mustCreateOpcuaConfig(t *testing.T, baseURL, token, projectID string, payload map[string]any) protocolWave1ConnectionPayload {
	t.Helper()
	responseEnvelope := doJSONRequest(t, http.MethodPost, baseURL+"/api/v1/data/projects/"+projectID+"/opcua/configs", token, payload)
	return decodeProtocolWave1Connection(t, responseEnvelope.Data)
}

func mustCreateS7Config(t *testing.T, baseURL, token, projectID string, payload map[string]any) protocolWave1ConnectionPayload {
	t.Helper()
	responseEnvelope := doJSONRequest(t, http.MethodPost, baseURL+"/api/v1/data/projects/"+projectID+"/s7/configs", token, payload)
	return decodeProtocolWave1Connection(t, responseEnvelope.Data)
}

func mustCreateModbusConfig(t *testing.T, baseURL, token, projectID string, payload map[string]any) protocolWave1ConnectionPayload {
	t.Helper()
	responseEnvelope := doJSONRequest(t, http.MethodPost, baseURL+"/api/v1/data/projects/"+projectID+"/modbus/configs", token, payload)
	return decodeProtocolWave1Connection(t, responseEnvelope.Data)
}

func mustCreateTdengineConfig(t *testing.T, baseURL, token, projectID string, payload map[string]any) protocolWave1ConnectionPayload {
	t.Helper()
	responseEnvelope := doJSONRequest(t, http.MethodPost, baseURL+"/api/v1/data/projects/"+projectID+"/tdengine/configs", token, payload)
	return decodeProtocolWave1Connection(t, responseEnvelope.Data)
}
