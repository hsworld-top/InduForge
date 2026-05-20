package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/indu-forge/data_service/internal/app"
	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/config"
	apperrors "github.com/indu-forge/data_service/internal/errors"
)

func TestComputeRunTimeout(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skipf("node binary is not available: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupMigrator(t, fixture.pool)
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("migrate up failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "compute-timeout-secret-01"

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
		TenantID:     "tenant-compute",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	unit := mustCreateComputeUnit(t, server.URL, token, projectID, map[string]any{
		"name":       "timeout-js-unit",
		"language":   "js",
		"scriptCode": "while(true){}",
		"timeoutMs":  100,
	})
	if unit.ID == "" {
		t.Fatal("expected compute unit id")
	}

	timeoutEnvelope := doJSONRequestWithStatus(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/compute-units/"+unit.ID+"/run", token, map[string]any{
		"input": map[string]any{},
	}, http.StatusBadRequest)
	if timeoutEnvelope.Code != apperrors.PublicCodeBadRequest {
		t.Fatalf("expected code %d for timeout, got %d", apperrors.PublicCodeBadRequest, timeoutEnvelope.Code)
	}
	if !strings.Contains(timeoutEnvelope.Msg, "超时") {
		t.Fatalf("expected timeout message, got %q", timeoutEnvelope.Msg)
	}
}

func TestComputeRunJSPython(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skipf("node binary is not available: %v", err)
	}
	if _, err := findPythonBinaryForTest(); err != nil {
		t.Skipf("python binary is not available: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupMigrator(t, fixture.pool)
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("migrate up failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "compute-jspy-secret-01"

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
		TenantID:     "tenant-compute",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	jsUnit := mustCreateComputeUnit(t, server.URL, token, projectID, map[string]any{
		"name":       "sum-js-unit",
		"language":   "js",
		"scriptCode": "return (argv[0] || 0) + (argv[1] || 0);",
		"timeoutMs":  3000,
	})
	jsRun := mustRunComputeUnit(t, server.URL, token, projectID, jsUnit.ID, map[string]any{
		"argv": []any{1, 2},
	})
	if jsRun.Status != "success" {
		t.Fatalf("expected js run status success, got %q", jsRun.Status)
	}
	if jsRun.Output != float64(3) {
		t.Fatalf("expected js output 3, got %#v", jsRun.Output)
	}

	pythonUnit := mustCreateComputeUnit(t, server.URL, token, projectID, map[string]any{
		"name":       "sum-python-unit",
		"language":   "python",
		"scriptCode": "def main(argv, dp, ctx):\n    return (argv[0] or 0) + (argv[1] or 0)",
		"timeoutMs":  3000,
	})
	pythonRun := mustRunComputeUnit(t, server.URL, token, projectID, pythonUnit.ID, map[string]any{
		"argv": []any{4, 5},
	})
	if pythonRun.Status != "success" {
		t.Fatalf("expected python run status success, got %q", pythonRun.Status)
	}
	if pythonRun.Output != float64(9) {
		t.Fatalf("expected python output 9, got %#v", pythonRun.Output)
	}
}

func TestComputeOutputDataPointGeneratedOnSave(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupMigrator(t, fixture.pool)
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("migrate up failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "compute-output-datapoint-secret-01"

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
		TenantID:     "tenant-compute-output",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	unit := mustCreateComputeUnit(t, server.URL, token, projectID, map[string]any{
		"name":       "output-default-unit",
		"language":   "js",
		"scriptCode": "return argv[0] ?? null;",
		"timeoutMs":  3000,
	})

	createdOutputs := mustListDataPoints(t, server.URL, token, projectID, "type=calc.output&search=calc.output-default-unit.result")
	if len(createdOutputs.DataPoints) != 1 {
		t.Fatalf("expected created compute output datapoint, got %d", len(createdOutputs.DataPoints))
	}
	if createdOutputs.DataPoints[0].Path != "calc.output-default-unit.result" {
		t.Fatalf("expected output datapoint path calc.output-default-unit.result, got %q", createdOutputs.DataPoints[0].Path)
	}
	if createdOutputs.DataPoints[0].SourceID == nil || *createdOutputs.DataPoints[0].SourceID != unit.ID {
		t.Fatalf("expected output datapoint source id %q, got %#v", unit.ID, createdOutputs.DataPoints[0].SourceID)
	}

	if _, err := fixture.pool.Exec(ctx, `
		UPDATE data_compute_units
		SET output_bindings = '{}'::jsonb
		WHERE project_id = $1 AND id = $2
	`, projectID, unit.ID); err != nil {
		t.Fatalf("reset compute output bindings failed: %v", err)
	}
	if _, err := fixture.pool.Exec(ctx, `
		DELETE FROM data_points
		WHERE project_id = $1 AND source_type = 'calc.output' AND source_id = $2
	`, projectID, unit.ID); err != nil {
		t.Fatalf("delete generated output datapoint failed: %v", err)
	}

	mustUpdateComputeUnit(t, server.URL, token, projectID, unit.ID, map[string]any{
		"name": "output-default-unit",
	})

	savedOutputs := mustListDataPoints(t, server.URL, token, projectID, "type=calc.output&search=calc.output-default-unit.result")
	if len(savedOutputs.DataPoints) != 1 {
		t.Fatalf("expected saved compute output datapoint, got %d", len(savedOutputs.DataPoints))
	}
	if savedOutputs.DataPoints[0].Status != "active" {
		t.Fatalf("expected saved output datapoint active, got %q", savedOutputs.DataPoints[0].Status)
	}
}

type computeUnitPayload struct {
	ID         string `json:"id"`
	ProjectID  string `json:"projectId"`
	Name       string `json:"name"`
	Language   string `json:"language"`
	ScriptCode string `json:"scriptCode"`
	TimeoutMS  int    `json:"timeoutMs"`
}

type computeRunPayload struct {
	Status     string `json:"status"`
	DurationMS int    `json:"durationMs"`
	Output     any    `json:"output"`
}

func mustCreateComputeUnit(t *testing.T, baseURL, token, projectID string, payload map[string]any) computeUnitPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodPost, baseURL+"/api/v1/data/projects/"+projectID+"/compute-units", token, payload)
	var result computeUnitPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode compute create response failed: %v", err)
	}
	return result
}

func mustUpdateComputeUnit(t *testing.T, baseURL, token, projectID, unitID string, payload map[string]any) computeUnitPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodPut, baseURL+"/api/v1/data/projects/"+projectID+"/compute-units/"+unitID, token, payload)
	var result computeUnitPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode compute update response failed: %v", err)
	}
	return result
}

func mustRunComputeUnit(t *testing.T, baseURL, token, projectID, unitID string, input map[string]any) computeRunPayload {
	t.Helper()

	responseEnvelope, statusCode := doJSONRequestAllowStatus(t, http.MethodPost, baseURL+"/api/v1/data/projects/"+projectID+"/compute-units/"+unitID+"/run", token, map[string]any{
		"input": input,
	})
	if statusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d, code=%d msg=%q", statusCode, responseEnvelope.Code, responseEnvelope.Msg)
	}
	if responseEnvelope.Code != apperrors.SuccessCode {
		t.Fatalf("expected code=%d, got %d", apperrors.SuccessCode, responseEnvelope.Code)
	}
	var result computeRunPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode compute run response failed: %v", err)
	}
	return result
}

func findPythonBinaryForTest() (string, error) {
	if path, err := exec.LookPath("python"); err == nil {
		if exec.Command(path, "--version").Run() == nil {
			return path, nil
		}
	}
	if path, err := exec.LookPath("python3"); err == nil {
		if exec.Command(path, "--version").Run() == nil {
			return path, nil
		}
	}
	return "", exec.ErrNotFound
}

func doJSONRequestAllowStatus(t *testing.T, method, url, token string, payload any) (apiEnvelope, int) {
	t.Helper()

	var bodyBytes []byte
	if payload != nil {
		var err error
		bodyBytes, err = json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal request payload failed: %v", err)
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "rid-compute-it")

	resp, err := integrationHTTPClient.Do(req)
	if err != nil {
		t.Fatalf("execute request failed: %v", err)
	}
	defer resp.Body.Close()

	var envelope apiEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	return envelope, resp.StatusCode
}
