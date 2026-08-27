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
	apperrors "github.com/indu-forge/data_service/internal/errors"
)

func TestTDengineAndOPCDevelopmentBoundary(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupSchemaInitializer(t, fixture.pool)
	if err := migrator.Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "tdengine-opc-secret-01"

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
		TenantID:     "tenant-tdengine-opc",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	tdengine := doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/tdengine/configs", token, map[string]any{
		"name": "td-main", "protocol": "ws", "host": "127.0.0.1", "port": 6041,
		"username": "root", "databaseName": "factory", "timezone": "Asia/Shanghai",
		"secrets": map[string]string{"password": "taosdata"},
	})
	var created protocolConnectionConnectionPayload
	if err := json.Unmarshal(tdengine.Data, &created); err != nil || created.ID == "" || created.Type != "tdengine" {
		t.Fatalf("expected dedicated TDengine connection, got %#v, err=%v", created, err)
	}
	updatedTDengine := doJSONRequest(t, http.MethodPut, server.URL+"/api/v1/data/projects/"+projectID+"/tdengine/configs/"+created.ID, token, map[string]any{
		"name": "td-updated", "enabled": false, "protocol": "wss", "host": "td.example.local", "port": 6041,
		"username": "root", "databaseName": "factory", "timezone": "Asia/Shanghai", "tlsSkipVerify": true,
		"secrets": map[string]string{"password": "updated-taos-secret"},
	})
	var updated protocolConnectionConnectionPayload
	if err := json.Unmarshal(updatedTDengine.Data, &updated); err != nil || updated.Name != "td-updated" {
		t.Fatalf("expected dedicated TDengine update, got %#v, err=%v", updated, err)
	}
	var plaintextMatches int
	if err := fixture.pool.QueryRow(ctx, `SELECT COUNT(*) FROM data_connection_secrets WHERE connection_id = $1 AND encrypted_value = convert_to('updated-taos-secret', 'UTF8')`, created.ID).Scan(&plaintextMatches); err != nil || plaintextMatches != 0 {
		t.Fatalf("TDengine password must be encrypted, matches=%d err=%v", plaintextMatches, err)
	}

	validOpcda := doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/opcda/contracts/validate", token, map[string]any{
		"itemPath":   "Channel1.Device1.TagA",
		"samplingMs": 1000,
	})
	if validOpcda.Code != apperrors.SuccessCode {
		t.Fatalf("expected OPC DA contract validation success, got %#v", validOpcda)
	}
}
