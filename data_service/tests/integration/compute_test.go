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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupSchemaInitializer(t, fixture.pool)
	if err := migrator.Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "compute-timeout-secret-01"
	sandbox := newComputeSandboxStub(t)

	srv, err := app.NewServer(config.Config{
		Addr:                       ":0",
		DatabaseURL:                fixture.databaseURL,
		DatabaseSearchPath:         fixture.schemaName,
		JWTSecret:                  secret,
		ConnectionSecretKey:        []byte("0123456789abcdef0123456789abcdef"),
		ConnectionSecretKeyVersion: "v1",
		ComputeSandboxURL:          sandbox.URL,
		ComputeSandboxToken:        "integration-sandbox-token",
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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupSchemaInitializer(t, fixture.pool)
	if err := migrator.Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "compute-jspy-secret-01"
	sandbox := newComputeSandboxStub(t)

	srv, err := app.NewServer(config.Config{
		Addr:                       ":0",
		DatabaseURL:                fixture.databaseURL,
		DatabaseSearchPath:         fixture.schemaName,
		JWTSecret:                  secret,
		ConnectionSecretKey:        []byte("0123456789abcdef0123456789abcdef"),
		ConnectionSecretKeyVersion: "v1",
		ComputeSandboxURL:          sandbox.URL,
		ComputeSandboxToken:        "integration-sandbox-token",
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
		"outputs": []map[string]any{{
			"key": "result", "name": "结果", "dataType": "float64", "nullPolicy": "error",
		}},
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
		"outputs": []map[string]any{{
			"key": "result", "name": "结果", "dataType": "float64", "nullPolicy": "error",
		}},
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

func TestComputeSchedulePreviewReturnsNormalizedRunsAndFieldErrors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	fixture := setupTestDatabase(t, ctx)
	if err := setupSchemaInitializer(t, fixture.pool).Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}
	projectID, userID, secret := uuid.NewString(), uuid.NewString(), "compute-schedule-preview-secret"
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
	token := mustSignIntegrationJWT(t, secret, &auth.Claims{UserID: userID, TenantID: "tenant-compute", ProjectIDs: []string{projectID}, Capabilities: []string{"project:read", "project:write"}})

	valid := doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/compute-units/schedule-preview", token, map[string]any{
		"triggerType":   "schedule",
		"triggerConfig": map[string]any{"kind": "weekly", "weekdays": []int{5, 1, 5}, "time": "08:30:00", "timezone": "Asia/Shanghai"},
	})
	var preview struct {
		TriggerConfig map[string]any `json:"triggerConfig"`
		Summary       string         `json:"summary"`
		NextRuns      []time.Time    `json:"nextRuns"`
		Errors        []struct {
			Field string `json:"field"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(valid.Data, &preview); err != nil {
		t.Fatal(err)
	}
	if len(preview.Errors) != 0 || len(preview.NextRuns) != 5 || preview.Summary == "" {
		t.Fatalf("expected five normalized weekly runs, got %#v", preview)
	}
	weekdays, ok := preview.TriggerConfig["weekdays"].([]any)
	if !ok || len(weekdays) != 2 || weekdays[0] != float64(1) || weekdays[1] != float64(5) {
		t.Fatalf("expected sorted unique weekdays, got %#v", preview.TriggerConfig["weekdays"])
	}

	invalid := doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/compute-units/schedule-preview", token, map[string]any{
		"triggerType":   "schedule",
		"triggerConfig": map[string]any{"kind": "daily", "time": "08:30", "timezone": "Asia/Shanghai"},
	})
	if err := json.Unmarshal(invalid.Data, &preview); err != nil {
		t.Fatal(err)
	}
	if len(preview.Errors) != 1 || preview.Errors[0].Field != "triggerConfig.time" {
		t.Fatalf("expected field-level time error, got %#v", preview.Errors)
	}
}

// newComputeSandboxStub 只验证 data_service 与独立沙箱的 HTTP 边界；脚本隔离和真实语言执行由 compute_sandbox 测试负责。
func newComputeSandboxStub(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer integration-sandbox-token" {
			http.Error(writer, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/v1/execute":
			var payload struct {
				Language string `json:"language"`
				Script   string `json:"script"`
			}
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				http.Error(writer, `{"error":"invalid request"}`, http.StatusBadRequest)
				return
			}
			if strings.Contains(payload.Script, "while(true)") {
				writer.WriteHeader(http.StatusRequestTimeout)
				_, _ = writer.Write([]byte(`{"error":"execution timeout"}`))
				return
			}
			output := 3
			if payload.Language == "python" {
				output = 9
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"output": output, "sideEffects": []any{}, "durationMs": 1})
		case "/v1/capabilities":
			_ = json.NewEncoder(writer).Encode(map[string]any{"available": true, "languages": []any{}, "sdk": []any{}, "dependencies": []any{}, "triggers": []any{}})
		default:
			http.NotFound(writer, request)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestComputeOutputDataPointGeneratedOnSave(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupSchemaInitializer(t, fixture.pool)
	if err := migrator.Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "compute-output-datapoint-secret-01"

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
	if createdOutputs.DataPoints[0].Name != "result" {
		t.Fatalf("expected output datapoint name result, got %q", createdOutputs.DataPoints[0].Name)
	}
	if createdOutputs.DataPoints[0].SourceID == nil || *createdOutputs.DataPoints[0].SourceID != unit.ID {
		t.Fatalf("expected output datapoint source id %q, got %#v", unit.ID, createdOutputs.DataPoints[0].SourceID)
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
	if savedOutputs.DataPoints[0].Name != "result" {
		t.Fatalf("expected saved output datapoint name result, got %q", savedOutputs.DataPoints[0].Name)
	}
}

func TestComputeUnitRenameMoveUpdatesOutputDataPoint(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupSchemaInitializer(t, fixture.pool)
	if err := migrator.Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "compute-rename-move-secret-01"

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
		TenantID:     "tenant-compute-rename-move",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	folder := mustCreateComputeFolder(t, server.URL, token, projectID, map[string]any{
		"name": "目标分组",
	})
	childFolder := mustCreateComputeFolder(t, server.URL, token, projectID, map[string]any{
		"name": "子分组", "parentId": folder.ID,
	})
	rootPageEnvelope := doJSONRequest(t, http.MethodGet, server.URL+"/api/v1/data/projects/"+projectID+"/compute-units/folders?page=1&pageSize=1", token, nil)
	var rootPage struct {
		List []struct {
			ID          string `json:"id"`
			HasChildren bool   `json:"hasChildren"`
		} `json:"list"`
		Pagination paginationPayload `json:"pagination"`
	}
	if err := json.Unmarshal(rootPageEnvelope.Data, &rootPage); err != nil {
		t.Fatalf("decode compute folder page failed: %v", err)
	}
	if rootPage.Pagination.Total != 1 || len(rootPage.List) != 1 || rootPage.List[0].ID != folder.ID || !rootPage.List[0].HasChildren {
		t.Fatalf("expected paged root folder with lazy child marker, got %#v", rootPage)
	}
	childPageEnvelope := doJSONRequest(t, http.MethodGet, server.URL+"/api/v1/data/projects/"+projectID+"/compute-units/folders?parentId="+folder.ID, token, nil)
	var childPage struct {
		List []computeFolderPayload `json:"list"`
	}
	if err := json.Unmarshal(childPageEnvelope.Data, &childPage); err != nil {
		t.Fatalf("decode compute child folder page failed: %v", err)
	}
	if len(childPage.List) != 1 || childPage.List[0].ID != childFolder.ID {
		t.Fatalf("expected direct child folder page, got %#v", childPage.List)
	}
	unit := mustCreateComputeUnit(t, server.URL, token, projectID, map[string]any{
		"name":       "old-unit",
		"language":   "js",
		"scriptCode": "return argv[0] ?? null;",
		"timeoutMs":  3000,
	})

	initialOutputs := mustListDataPoints(t, server.URL, token, projectID, "type=calc.output&search=calc.old-unit.result")
	if len(initialOutputs.DataPoints) != 1 {
		t.Fatalf("expected initial output datapoint, got %d", len(initialOutputs.DataPoints))
	}
	initialID := initialOutputs.DataPoints[0].ID

	updated := mustUpdateComputeUnit(t, server.URL, token, projectID, unit.ID, map[string]any{
		"name":     "new-unit",
		"folderId": folder.ID,
	})
	if updated.Name != "new-unit" {
		t.Fatalf("expected compute unit renamed, got %q", updated.Name)
	}
	if updated.FolderID == nil || *updated.FolderID != folder.ID {
		t.Fatalf("expected compute unit moved to folder %q, got %#v", folder.ID, updated.FolderID)
	}

	// 计算输出路径是公开稳定身份；仅修改单元名称或目录不得隐式改写输出路径。
	newOutputs := mustListDataPoints(t, server.URL, token, projectID, "type=calc.output&search=calc.old-unit.result")
	if len(newOutputs.DataPoints) != 1 {
		t.Fatalf("expected renamed output datapoint, got %d", len(newOutputs.DataPoints))
	}
	if newOutputs.DataPoints[0].ID != initialID {
		t.Fatalf("expected output datapoint id unchanged, got %q want %q", newOutputs.DataPoints[0].ID, initialID)
	}
	if newOutputs.DataPoints[0].Name != "result" {
		t.Fatalf("expected output datapoint name result, got %q", newOutputs.DataPoints[0].Name)
	}
	if newOutputs.DataPoints[0].Status != "active" {
		t.Fatalf("expected output datapoint active, got %q", newOutputs.DataPoints[0].Status)
	}

	renamedPathOutputs := mustListDataPoints(t, server.URL, token, projectID, "type=calc.output&search=calc.new-unit.result")
	if len(renamedPathOutputs.DataPoints) != 0 {
		t.Fatalf("expected unit rename not to create a second output datapoint, got %d", len(renamedPathOutputs.DataPoints))
	}
}

func TestDataPointListDoesNotMutateValidityAfterRead(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupSchemaInitializer(t, fixture.pool)
	if err := migrator.Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "datapoint-validity-secret-01"

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
		TenantID:     "tenant-datapoint-validity",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	unit := mustCreateComputeUnit(t, server.URL, token, projectID, map[string]any{
		"name":       "valid-source-unit",
		"language":   "js",
		"scriptCode": "return argv[0] ?? null;",
		"timeoutMs":  3000,
	})

	outputs := mustListDataPoints(t, server.URL, token, projectID, "type=calc.output&search=calc.valid-source-unit.result")
	if len(outputs.DataPoints) != 1 {
		t.Fatalf("expected generated output datapoint, got %d", len(outputs.DataPoints))
	}
	point := outputs.DataPoints[0]
	if _, err := fixture.pool.Exec(ctx, `
		UPDATE data_points
		SET path = $3,
		    name = $4,
		    updated_at = now()
		WHERE project_id = $1 AND id = $2
	`, projectID, point.ID, "calc.wrong-source-unit.result", "wrong-source-unit"); err != nil {
		t.Fatalf("corrupt generated datapoint failed: %v", err)
	}

	refreshed := mustListDataPoints(t, server.URL, token, projectID, "type=calc.output&search=calc.wrong-source-unit.result")
	if len(refreshed.DataPoints) != 1 {
		t.Fatalf("expected mismatched output datapoint visible, got %d", len(refreshed.DataPoints))
	}
	if refreshed.DataPoints[0].Status != "active" {
		t.Fatalf("list must not mutate persisted validity, got %q", refreshed.DataPoints[0].Status)
	}
	if refreshed.DataPoints[0].SourceID == nil || *refreshed.DataPoints[0].SourceID != unit.ID {
		t.Fatalf("expected source id %q unchanged, got %#v", unit.ID, refreshed.DataPoints[0].SourceID)
	}
	if refreshed.DataPoints[0].InvalidReason != nil {
		t.Fatalf("list must not synthesize invalid reason after pagination, got %#v", refreshed.DataPoints[0].InvalidReason)
	}
	detail := mustGetDataPoint(t, server.URL, token, projectID, refreshed.DataPoints[0].ID)
	if detail.Status != "active" || detail.InvalidReason != nil {
		t.Fatalf("detail must reflect persisted state without read-side writes, got status=%q reason=%#v", detail.Status, detail.InvalidReason)
	}
}

type computeUnitPayload struct {
	ID         string  `json:"id"`
	ProjectID  string  `json:"projectId"`
	Name       string  `json:"name"`
	Language   string  `json:"language"`
	ScriptCode string  `json:"scriptCode"`
	TimeoutMS  int     `json:"timeoutMs"`
	FolderID   *string `json:"folderId"`
}

type computeFolderPayload struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	ParentID *string `json:"parentId"`
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

func mustCreateComputeFolder(t *testing.T, baseURL, token, projectID string, payload map[string]any) computeFolderPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodPost, baseURL+"/api/v1/data/projects/"+projectID+"/compute-units/folders", token, payload)
	var result computeFolderPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode compute folder create response failed: %v", err)
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
