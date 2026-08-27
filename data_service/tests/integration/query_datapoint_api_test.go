package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/indu-forge/data_service/internal/app"
	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/config"
	apperrors "github.com/indu-forge/data_service/internal/errors"
)

func TestExecuteQueryAndDataPointValue(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupSchemaInitializer(t, fixture.pool)
	if err := migrator.Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "query-datapoint-secret-01"

	connectionID := insertTestConnection(t, ctx, fixture, projectID, userID)

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
		TenantID:     "tenant-query",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	createdQuery := mustCreateQuery(t, server.URL, token, projectID, map[string]any{
		"name":         "select-one",
		"connectionId": connectionID,
		"queryType":    "sql",
		"config": map[string]any{
			"sql": "SELECT $1::int AS value",
			"parameters": []any{
				map[string]any{
					"name":    "value",
					"default": 1,
				},
			},
		},
	})

	executed := mustExecuteQuery(t, server.URL, token, createdQuery.ID, map[string]any{
		"parameters": map[string]any{},
	})
	if executed.RowCount != 1 {
		t.Fatalf("expected rowCount=1, got %d", executed.RowCount)
	}
	if len(executed.Data) != 1 {
		t.Fatalf("expected 1 result row, got %d", len(executed.Data))
	}

	_, err = fixture.pool.Exec(ctx, `
        INSERT INTO data_points (
            project_id, path, name, source_type, source_id, source_config, data_type, status, created_by
        )
        VALUES ($1, $2, $3, $4, $5, '{}'::jsonb, $6, $7, $8)
    `, projectID, "metrics.query.value", "query-value", "db.query", createdQuery.ID, "object", "active", userID)
	if err != nil {
		t.Fatalf("insert datapoint failed: %v", err)
	}

	value := mustGetDataPointValue(t, server.URL, token, projectID, "metrics.query.value")
	if value.Quality != "good" {
		t.Fatalf("expected quality=good, got %q", value.Quality)
	}
	if value.Status != "active" {
		t.Fatalf("expected status=active, got %q", value.Status)
	}

	dataset, ok := value.Value.(map[string]any)
	if !ok {
		t.Fatalf("expected value to be a dataset object, got %#v", value.Value)
	}
	rows, ok := dataset["rows"].([]any)
	if !ok {
		t.Fatalf("expected dataset rows to be a slice, got %#v", dataset["rows"])
	}
	if len(rows) != 1 {
		t.Fatalf("expected value to contain 1 row, got %d", len(rows))
	}
	fields, ok := dataset["fields"].([]any)
	if !ok || len(fields) != 1 || fields[0] != "value" {
		t.Fatalf("expected dataset fields=[value], got %#v", dataset["fields"])
	}
	if dataset["rowCount"] != float64(1) {
		t.Fatalf("expected dataset rowCount=1, got %#v", dataset)
	}
	for _, omitted := range []string{"total", "truncated"} {
		if _, exists := dataset[omitted]; exists {
			t.Fatalf("dataset must omit %s, got %#v", omitted, dataset)
		}
	}

	rowMap, ok := rows[0].(map[string]any)
	if !ok {
		t.Fatalf("expected row 0 to be an object, got %#v", rows[0])
	}
	if rowMap["value"] != float64(1) {
		t.Fatalf("expected value=1, got %#v", rowMap["value"])
	}

	overriddenValue := mustGetDataPointValueWithParameters(t, server.URL, token, projectID, "metrics.query.value", map[string]any{"value": 7})
	overriddenDataset, ok := overriddenValue.Value.(map[string]any)
	if !ok {
		t.Fatalf("expected overridden value to be a dataset object, got %#v", overriddenValue.Value)
	}
	overriddenRows, ok := overriddenDataset["rows"].([]any)
	if !ok || len(overriddenRows) != 1 {
		t.Fatalf("expected one overridden row, got %#v", overriddenDataset["rows"])
	}
	overriddenRow, ok := overriddenRows[0].(map[string]any)
	if !ok || overriddenRow["value"] != float64(7) {
		t.Fatalf("expected runtime parameter override value=7, got %#v", overriddenRows[0])
	}

	disabledQuery := mustCreateQuery(t, server.URL, token, projectID, map[string]any{
		"name":         "select-disabled",
		"connectionId": connectionID,
		"queryType":    "sql",
		"isEnabled":    false,
		"config": map[string]any{
			"sql": "SELECT 1",
		},
	})
	disabledExecute := doJSONRequestWithStatus(t, http.MethodPost, server.URL+"/api/v1/data/queries/"+disabledQuery.ID+"/execute", token, map[string]any{}, http.StatusOK)
	if disabledExecute.Code != apperrors.PublicCodeBadRequest {
		t.Fatalf("expected code %d for disabled query execute, got %d", apperrors.PublicCodeBadRequest, disabledExecute.Code)
	}

	writeSQLQuery := mustCreateQuery(t, server.URL, token, projectID, map[string]any{
		"name":         "write-sql-query",
		"connectionId": connectionID,
		"queryType":    "sql",
		"config": map[string]any{
			"sql": "UPDATE data_points SET status = 'invalid' WHERE 1 = 0",
		},
	})
	writeSQLExecute := mustExecuteQuery(t, server.URL, token, writeSQLQuery.ID, map[string]any{})
	if writeSQLExecute.RowCount != 0 {
		t.Fatalf("expected zero affected rows for write query, got %d", writeSQLExecute.RowCount)
	}
}

func TestDataPointBatchDelete(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupSchemaInitializer(t, fixture.pool)
	if err := migrator.Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "delete-datapoint-secret-01"

	insertTestConnection(t, ctx, fixture, projectID, userID)

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
		TenantID:     "tenant-delete",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	invalidIDs := insertInvalidDataPoints(t, ctx, fixture, projectID, userID, 2)
	deleteResult := mustDeleteDataPointsBatch(t, server.URL, token, projectID, invalidIDs)
	if deleteResult.DeletedCount != 2 {
		t.Fatalf("expected 2 deleted datapoints, got %d", deleteResult.DeletedCount)
	}

	activeID := insertOneDataPoint(t, ctx, fixture, projectID, userID, "metrics.keep", "active")
	errEnvelope := doJSONRequestWithStatus(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/datapoints/delete-batch", token, map[string]any{
		"ids": []string{activeID},
	}, http.StatusOK)
	if errEnvelope.Code != apperrors.PublicCodeBadRequest {
		t.Fatalf("expected code %d, got %d", apperrors.PublicCodeBadRequest, errEnvelope.Code)
	}
}

func TestQueryAndDataPointCRUD(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupSchemaInitializer(t, fixture.pool)
	if err := migrator.Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "crud-query-secret-01"

	connectionID := insertTestConnection(t, ctx, fixture, projectID, userID)

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
		TenantID:     "tenant-crud",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	initialQueries := mustListQueries(t, server.URL, token, projectID)
	if len(initialQueries.Queries) != 0 {
		t.Fatalf("expected empty query list, got %d", len(initialQueries.Queries))
	}

	// 输出映射在主查询写入后失败时，事务必须回滚查询主记录。
	invalidMappingID := uuid.NewString()
	rollbackEnvelope, rollbackStatus := doJSONRequestAllowStatus(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/queries", token, map[string]any{
		"name": "rollback-query", "connectionId": connectionID, "queryType": "sql",
		"config": map[string]any{"sql": "SELECT 1 AS value"},
		"outputs": []map[string]any{{"id": invalidMappingID, "key": "value", "displayName": "值",
			"selector": map[string]any{"kind": "column", "column": "value"}, "dataType": "int32"}},
	})
	if rollbackStatus != http.StatusOK || rollbackEnvelope.Code == 0 {
		t.Fatalf("expected business error envelope for invalid output mapping, status=%d code=%d", rollbackStatus, rollbackEnvelope.Code)
	}
	var rollbackQueryCount int
	if err := fixture.pool.QueryRow(ctx, `SELECT COUNT(*) FROM data_queries WHERE project_id=$1 AND name='rollback-query'`, projectID).Scan(&rollbackQueryCount); err != nil {
		t.Fatalf("check rolled back query failed: %v", err)
	}
	if rollbackQueryCount != 0 {
		t.Fatalf("output mapping failure must roll back query, count=%d", rollbackQueryCount)
	}

	createdQuery := mustCreateQuery(t, server.URL, token, projectID, map[string]any{
		"name":         "crud-query",
		"connectionId": connectionID,
		"queryType":    "sql",
		"config": map[string]any{
			"sql": "SELECT 1 AS value",
		},
	})
	if createdQuery.ProjectID != projectID {
		t.Fatalf("expected projectId %q, got %q", projectID, createdQuery.ProjectID)
	}
	if len(createdQuery.Outputs) != 0 {
		t.Fatalf("saving a query must not generate datapoints, got %d outputs", len(createdQuery.Outputs))
	}
	var generatedPointCount int
	if err := fixture.pool.QueryRow(ctx, `SELECT COUNT(*) FROM data_points WHERE project_id=$1 AND source_type='db.query' AND source_id=$2`, projectID, createdQuery.ID).Scan(&generatedPointCount); err != nil {
		t.Fatalf("count generated query datapoints failed: %v", err)
	}
	if generatedPointCount != 0 {
		t.Fatalf("saving a query must not create datapoints, got %d", generatedPointCount)
	}

	generatedQuery := mustUpdateQuery(t, server.URL, token, createdQuery.ID, map[string]any{
		"outputs": []map[string]any{{
			"key": "result", "displayName": "查询结果",
			"selector": map[string]any{"kind": "whole"}, "dataType": "object",
		}},
	})
	if len(generatedQuery.Outputs) != 1 || generatedQuery.Outputs[0].DataPointPath == "" {
		t.Fatalf("explicit output generation must return one datapoint, got %#v", generatedQuery.Outputs)
	}
	if generatedQuery.Outputs[0].DataPointPath != "db.test-connection.crud-query" {
		t.Fatalf("whole query dataset must use query path directly, got %q", generatedQuery.Outputs[0].DataPointPath)
	}
	_, err = fixture.pool.Exec(ctx, `
		INSERT INTO data_points(project_id,path,name,source_type,source_id,source_config,data_type,status,created_by)
		VALUES($1,$2,$3,'db.query',$4,'{}'::jsonb,'int32','invalid',$5)
	`, projectID, "db.test-connection.crud-query.value", "旧字段", createdQuery.ID, userID)
	if err != nil {
		t.Fatalf("insert stale generated datapoint failed: %v", err)
	}
	extractedQuery := mustUpdateQuery(t, server.URL, token, createdQuery.ID, map[string]any{
		"outputs": []map[string]any{{
			"id": generatedQuery.Outputs[0].ID, "key": "value", "displayName": "值",
			"selector": map[string]any{"kind": "column", "column": "value"}, "dataType": "int32",
		}},
	})
	if len(extractedQuery.Outputs) != 1 || extractedQuery.Outputs[0].DataPointPath != "db.test-connection.crud-query.value" {
		t.Fatalf("query field must reclaim its exact semantic path, got %#v", extractedQuery.Outputs)
	}

	readOnlyToken := mustSignIntegrationJWT(t, secret, &auth.Claims{
		UserID:       userID,
		TenantID:     "tenant-crud",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read"},
	})
	updateDenied := doJSONRequestWithStatus(t, http.MethodPut, server.URL+"/api/v1/data/queries/"+createdQuery.ID, readOnlyToken, map[string]any{
		"name": "should-fail",
	}, http.StatusForbidden)
	if updateDenied.Code != apperrors.PublicCodePermissionInsufficient {
		t.Fatalf("expected code %d for readonly update, got %d", apperrors.PublicCodePermissionInsufficient, updateDenied.Code)
	}
	deleteDenied := doJSONRequestWithStatus(t, http.MethodDelete, server.URL+"/api/v1/data/queries/"+createdQuery.ID, readOnlyToken, nil, http.StatusForbidden)
	if deleteDenied.Code != apperrors.PublicCodePermissionInsufficient {
		t.Fatalf("expected code %d for readonly delete, got %d", apperrors.PublicCodePermissionInsufficient, deleteDenied.Code)
	}

	listAfterCreate := mustListQueries(t, server.URL, token, projectID)
	if len(listAfterCreate.Queries) != 1 {
		t.Fatalf("expected 1 query after create, got %d", len(listAfterCreate.Queries))
	}
	if listAfterCreate.Pagination.Total != 1 {
		t.Fatalf("expected total=1, got %d", listAfterCreate.Pagination.Total)
	}

	updatedQuery := mustUpdateQuery(t, server.URL, token, createdQuery.ID, map[string]any{
		"name":      "crud-query-updated",
		"timeoutMs": 12345,
	})
	if updatedQuery.Name != "crud-query-updated" {
		t.Fatalf("expected updated query name, got %q", updatedQuery.Name)
	}
	if updatedQuery.TimeoutMS != 12345 {
		t.Fatalf("expected timeoutMs=12345, got %d", updatedQuery.TimeoutMS)
	}
	if len(updatedQuery.Outputs) != 1 || updatedQuery.Outputs[0].DataPointPath != extractedQuery.Outputs[0].DataPointPath {
		t.Fatalf("ordinary query save must preserve generated outputs, got %#v", updatedQuery.Outputs)
	}

	queryExec := mustExecuteQuery(t, server.URL, token, updatedQuery.ID, map[string]any{})
	if queryExec.RowCount != 1 {
		t.Fatalf("expected rowCount=1, got %d", queryExec.RowCount)
	}

	mustDeleteQuery(t, server.URL, token, updatedQuery.ID)

	afterDeleteQueries := mustListQueries(t, server.URL, token, projectID)
	if afterDeleteQueries.Pagination.Total != 0 {
		t.Fatalf("expected 0 queries after delete, got %d", afterDeleteQueries.Pagination.Total)
	}

	dpID := insertOneDataPoint(t, ctx, fixture, projectID, userID, "metrics.crud", "active")

	listAfterInsert := mustListDataPoints(t, server.URL, token, projectID, "search=metrics.crud")
	if len(listAfterInsert.DataPoints) != 1 {
		t.Fatalf("expected 1 datapoint in list, got %d", len(listAfterInsert.DataPoints))
	}

	detail := mustGetDataPoint(t, server.URL, token, projectID, dpID)
	if detail.Path != "metrics.crud" {
		t.Fatalf("expected datapoint path metrics.crud, got %q", detail.Path)
	}

	updatedPoint := mustUpdateDataPoint(t, server.URL, token, projectID, dpID, map[string]any{
		"name":   "metrics.crud.updated",
		"unit":   "kg",
		"status": "active",
	})
	if updatedPoint.Name != "metrics.crud.updated" {
		t.Fatalf("expected updated datapoint name, got %q", updatedPoint.Name)
	}
	if updatedPoint.Unit == nil || *updatedPoint.Unit != "kg" {
		t.Fatalf("expected unit=kg, got %#v", updatedPoint.Unit)
	}
}

type queryExecutePayload struct {
	Data          []map[string]any `json:"data"`
	ExecutionTime int64            `json:"executionTime"`
	RowCount      int              `json:"rowCount"`
}

type queryListPayload struct {
	Queries    []queryPayload    `json:"queries"`
	Pagination paginationPayload `json:"pagination"`
}

type queryPayload struct {
	ID              string               `json:"id"`
	ProjectID       string               `json:"projectId"`
	ConnectionID    string               `json:"connectionId"`
	Name            string               `json:"name"`
	QueryType       string               `json:"queryType"`
	Config          map[string]any       `json:"config"`
	TimeoutMS       int                  `json:"timeoutMs"`
	CacheTtlSeconds int                  `json:"cacheTtlSeconds"`
	Outputs         []queryOutputPayload `json:"outputs"`
}

type queryOutputPayload struct {
	ID            string `json:"id"`
	DataPointPath string `json:"datapointPath"`
}

type paginationPayload struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

type dataPointPayload struct {
	ID                 string                   `json:"id"`
	ProjectID          string                   `json:"projectId"`
	Path               string                   `json:"path"`
	Name               string                   `json:"name"`
	SourceType         string                   `json:"sourceType"`
	SourceID           *string                  `json:"sourceId"`
	Unit               *string                  `json:"unit"`
	Status             string                   `json:"status"`
	InvalidReason      *string                  `json:"invalidReason"`
	RuntimePermissions dataPointPermissionGroup `json:"runtimePermissions"`
}

type dataPointListPayload struct {
	DataPoints []dataPointPayload `json:"datapoints"`
	Pagination paginationPayload  `json:"pagination"`
}

type dataPointPermissionGroup struct {
	Write dataPointRuntimeGrant `json:"write"`
}

type dataPointRuntimeGrant struct {
	AllowRoles []string `json:"allowRoles"`
	DenyRoles  []string `json:"denyRoles"`
	Inherit    bool     `json:"inherit"`
}

type dataPointValuePayload struct {
	Path    string `json:"path"`
	Value   any    `json:"value"`
	Quality string `json:"quality"`
	Status  string `json:"status"`
}

type deleteBatchPayload struct {
	DeletedCount int `json:"deletedCount"`
}

func mustCreateQuery(t *testing.T, baseURL, token, projectID string, payload map[string]any) queryPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodPost, baseURL+"/api/v1/data/projects/"+projectID+"/queries", token, payload)
	var result queryPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode query create response failed: %v", err)
	}
	if result.ID == "" {
		t.Fatal("expected query id in response")
	}
	return result
}

func mustListQueries(t *testing.T, baseURL, token, projectID string) queryListPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodGet, baseURL+"/api/v1/data/projects/"+projectID+"/queries", token, nil)
	var result queryListPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode query list response failed: %v", err)
	}
	return result
}

func mustExecuteQuery(t *testing.T, baseURL, token, queryID string, payload map[string]any) queryExecutePayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodPost, baseURL+"/api/v1/data/queries/"+queryID+"/execute", token, payload)
	var result queryExecutePayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode query execute response failed: %v", err)
	}
	return result
}

func mustUpdateQuery(t *testing.T, baseURL, token, queryID string, payload map[string]any) queryPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodPut, baseURL+"/api/v1/data/queries/"+queryID, token, payload)
	var result queryPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode query update response failed: %v", err)
	}
	return result
}

func mustDeleteQuery(t *testing.T, baseURL, token, queryID string) {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodDelete, baseURL+"/api/v1/data/queries/"+queryID, token, nil)
	var deleted struct {
		Deleted bool `json:"deleted"`
	}
	if err := json.Unmarshal(responseEnvelope.Data, &deleted); err != nil {
		t.Fatalf("decode query delete response failed: %v", err)
	}
	if !deleted.Deleted {
		t.Fatal("expected deleted=true")
	}
}

func mustListDataPoints(t *testing.T, baseURL, token, projectID, query string) dataPointListPayload {
	t.Helper()

	url := baseURL + "/api/v1/data/projects/" + projectID + "/datapoints"
	if query != "" {
		url += "?" + query
	}
	responseEnvelope := doJSONRequest(t, http.MethodGet, url, token, nil)
	var result dataPointListPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode datapoint list response failed: %v", err)
	}
	return result
}

func mustGetDataPoint(t *testing.T, baseURL, token, projectID, id string) dataPointPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodGet, baseURL+"/api/v1/data/projects/"+projectID+"/datapoints/"+id, token, nil)
	var result dataPointPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode datapoint detail response failed: %v", err)
	}
	return result
}

func mustGetDataPointValue(t *testing.T, baseURL, token, projectID, path string) dataPointValuePayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodGet, baseURL+"/api/v1/data/projects/"+projectID+"/datapoints/value?path="+url.QueryEscape(path), token, nil)
	var result dataPointValuePayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode datapoint value response failed: %v", err)
	}
	return result
}

func mustGetDataPointValueWithParameters(t *testing.T, baseURL, token, projectID, path string, parameters map[string]any) dataPointValuePayload {
	t.Helper()

	parameterJSON, err := json.Marshal(parameters)
	if err != nil {
		t.Fatalf("encode datapoint runtime parameters failed: %v", err)
	}
	endpoint := baseURL + "/api/v1/data/projects/" + projectID + "/datapoints/value?path=" + url.QueryEscape(path) + "&parameters=" + url.QueryEscape(string(parameterJSON))
	responseEnvelope := doJSONRequest(t, http.MethodGet, endpoint, token, nil)
	var result dataPointValuePayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode parametrized datapoint value response failed: %v", err)
	}
	return result
}

func mustUpdateDataPoint(t *testing.T, baseURL, token, projectID, id string, payload map[string]any) dataPointPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodPut, baseURL+"/api/v1/data/projects/"+projectID+"/datapoints/"+id, token, payload)
	var result dataPointPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode datapoint update response failed: %v", err)
	}
	return result
}

func mustDeleteDataPointsBatch(t *testing.T, baseURL, token, projectID string, ids []string) deleteBatchPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodPost, baseURL+"/api/v1/data/projects/"+projectID+"/datapoints/delete-batch", token, map[string]any{
		"ids": ids,
	})
	var result deleteBatchPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode batch delete response failed: %v", err)
	}
	return result
}

func mustUpdateDataPointRuntimePermissions(t *testing.T, baseURL, token, projectID, id string, payload map[string]any) dataPointPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodPut, baseURL+"/api/v1/data/projects/"+projectID+"/datapoints/"+id+"/runtime-permissions", token, payload)
	var result dataPointPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode datapoint runtime permissions response failed: %v", err)
	}
	return result
}

func assertRuntimeGrant(t *testing.T, grant dataPointRuntimeGrant, allowRoles, denyRoles []string, inherit bool) {
	t.Helper()

	if grant.Inherit != inherit {
		t.Fatalf("expected inherit=%v, got %v", inherit, grant.Inherit)
	}
	if fmt.Sprintf("%v", grant.AllowRoles) != fmt.Sprintf("%v", allowRoles) {
		t.Fatalf("expected allowRoles=%v, got %v", allowRoles, grant.AllowRoles)
	}
	if fmt.Sprintf("%v", grant.DenyRoles) != fmt.Sprintf("%v", denyRoles) {
		t.Fatalf("expected denyRoles=%v, got %v", denyRoles, grant.DenyRoles)
	}
}

func insertTestConnection(t *testing.T, ctx context.Context, fixture *testDatabase, projectID, userID string) string {
	t.Helper()

	metadataBytes, err := json.Marshal(map[string]any{
		"databaseUrl": fixture.databaseURL,
		"searchPath":  fixture.schemaName,
	})
	if err != nil {
		t.Fatalf("marshal connection metadata failed: %v", err)
	}

	connectionID := uuid.NewString()
	_, err = fixture.pool.Exec(ctx, `
		INSERT INTO data_connections (id, project_id, name, type, category, is_enabled, metadata, created_by)
		VALUES ($1, $2, $3, $4, 'database', true, $5::jsonb, $6)
	`, connectionID, projectID, "test-connection", "relational", string(metadataBytes), userID)
	if err != nil {
		t.Fatalf("insert connection failed: %v", err)
	}
	return connectionID
}

func insertOneDataPoint(t *testing.T, ctx context.Context, fixture *testDatabase, projectID, userID, path, status string) string {
	t.Helper()

	id := uuid.NewString()
	_, err := fixture.pool.Exec(ctx, `
        INSERT INTO data_points (
            id, project_id, path, name, source_type, source_config, data_type, status, created_by
        )
        VALUES ($1, $2, $3, $4, $5, '{}'::jsonb, $6, $7, $8)
    `, id, projectID, path, path, "manual", "string", status, userID)
	if err != nil {
		t.Fatalf("insert datapoint failed: %v", err)
	}
	return id
}

func insertInvalidDataPoints(t *testing.T, ctx context.Context, fixture *testDatabase, projectID, userID string, count int) []string {
	t.Helper()

	ids := make([]string, 0, count)
	for i := 0; i < count; i++ {
		ids = append(ids, insertOneDataPoint(t, ctx, fixture, projectID, userID, fmt.Sprintf("metrics.invalid.%d", i), "invalid"))
	}
	return ids
}

func TestDataPointRuntimePermissionsListAndSave(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupSchemaInitializer(t, fixture.pool)
	if err := migrator.Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "runtime-permission-secret-01"

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
		TenantID:     "tenant-runtime-permission",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	dpID := insertOneDataPoint(t, ctx, fixture, projectID, userID, "metrics.runtime.permission", "active")

	initialList := mustListDataPoints(t, server.URL, token, projectID, "search=metrics.runtime.permission")
	if len(initialList.DataPoints) != 1 {
		t.Fatalf("expected 1 datapoint in list, got %d", len(initialList.DataPoints))
	}
	initialDetail := mustGetDataPoint(t, server.URL, token, projectID, dpID)
	assertRuntimeGrant(t, initialDetail.RuntimePermissions.Write, []string{}, []string{}, true)

	updated := mustUpdateDataPointRuntimePermissions(t, server.URL, token, projectID, dpID, map[string]any{
		"write": map[string]any{
			"allowRoles": []string{"operator", "maintainer"},
			"denyRoles":  []string{"guest"},
			"inherit":    false,
		},
	})
	assertRuntimeGrant(t, updated.RuntimePermissions.Write, []string{"operator", "maintainer"}, []string{"guest"}, false)

	detail := mustGetDataPoint(t, server.URL, token, projectID, dpID)
	assertRuntimeGrant(t, detail.RuntimePermissions.Write, []string{"operator", "maintainer"}, []string{"guest"}, false)

	refreshedList := mustListDataPoints(t, server.URL, token, projectID, "search=metrics.runtime.permission")
	if len(refreshedList.DataPoints) != 1 {
		t.Fatalf("expected datapoint to remain visible after permission update, got %d", len(refreshedList.DataPoints))
	}
}

func doJSONRequestWithStatus(t *testing.T, method, url, token string, payload any, statusCode int) apiEnvelope {
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
	req.Header.Set("X-Request-ID", "rid-query-datapoint-it")

	resp, err := integrationHTTPClient.Do(req)
	if err != nil {
		t.Fatalf("execute request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != statusCode {
		t.Fatalf("expected status %d, got %d", statusCode, resp.StatusCode)
	}

	var envelope apiEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	return envelope
}
