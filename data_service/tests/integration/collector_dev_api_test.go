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

func TestCollectorDevVerticalLoop(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	fixture := setupTestDatabase(t, ctx)
	if err := setupSchemaInitializer(t, fixture.pool).Ensure(ctx); err != nil {
		t.Fatalf("初始化数据库结构失败: %v", err)
	}
	projectID, userID, tenantID := uuid.NewString(), uuid.NewString(), "tenant-collector"
	secret := "collector-dev-secret-123"
	srv, err := app.NewServer(config.Config{DataServiceInternalToken: "integration-test-internal-token", Addr: ":0", DatabaseURL: fixture.databaseURL, DatabaseSearchPath: fixture.schemaName, JWTSecret: secret, CollectorSecretKey: []byte("0123456789abcdef0123456789abcdef"), CollectorSecretKeyVersion: "v1", ConnectionSecretKey: []byte("0123456789abcdef0123456789abcdef"), ConnectionSecretKeyVersion: "v1", CollectorProtocolCatalogPath: "../../../contracts/collector-protocols"})
	if err != nil {
		t.Fatalf("创建服务失败: %v", err)
	}
	t.Cleanup(srv.Close)
	server := httptest.NewServer(srv.Handler())
	t.Cleanup(server.Close)
	adminToken := mustSignIntegrationJWT(t, secret, &auth.Claims{UserID: userID, TenantID: tenantID, Role: "SYSTEM_ADMIN", ProjectIDs: []string{projectID}, Capabilities: []string{"project:read", "project:write"}})

	registration := doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/collector-dev/registration-codes", adminToken, map[string]any{})
	var registrationPayload struct {
		Code string `json:"code"`
	}
	mustDecodeCollectorData(t, registration.Data, &registrationPayload)
	registered := doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/collector-dev/agent/register", "", map[string]any{
		"registrationCode": registrationPayload.Code,
		"machineId":        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"name":             "integration-agent",
		"os":               "windows",
		"arch":             "x64",
		"version":          "0.1.0",
		"capabilities":     []map[string]any{{"driverId": "opcua.standard", "driverVersion": "1.0.0", "schemaVersions": []int{1}, "operations": []string{"connection.test", "device.browse", "point.read"}}},
	})
	var agent struct {
		AgentID    string `json:"agentId"`
		AgentToken string `json:"agentToken"`
	}
	mustDecodeCollectorData(t, registered.Data, &agent)
	if agent.AgentID == "" || agent.AgentToken == "" {
		t.Fatal("期望注册返回 agentId 和 agentToken")
	}
	capabilities := []map[string]any{{"driverId": "opcua.standard", "driverVersion": "1.0.0", "schemaVersions": []int{1}, "operations": []string{"connection.test", "device.browse", "point.read"}}}
	doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/collector-dev/agent/heartbeat", agent.AgentToken, map[string]any{"capabilities": capabilities})
	agentsResponse := doJSONRequest(t, http.MethodGet, server.URL+"/api/v1/data/collector-dev/agents?page=1&pageSize=10", adminToken, nil)
	var agentsPage struct {
		List []struct {
			IPAddress string `json:"ipAddress"`
			Status    string `json:"status"`
		} `json:"list"`
		Pagination struct {
			Page       int `json:"page"`
			PageSize   int `json:"pageSize"`
			Total      int `json:"total"`
			TotalPages int `json:"totalPages"`
		} `json:"pagination"`
	}
	mustDecodeCollectorData(t, agentsResponse.Data, &agentsPage)
	if len(agentsPage.List) != 1 || agentsPage.List[0].IPAddress != "127.0.0.1" || agentsPage.List[0].Status != "online" {
		t.Fatalf("期望代理在线且记录 IP 127.0.0.1，实际 %#v", agentsPage.List)
	}
	if agentsPage.Pagination.Page != 1 || agentsPage.Pagination.PageSize != 10 || agentsPage.Pagination.Total != 1 || agentsPage.Pagination.TotalPages != 1 {
		t.Fatalf("采集调试代理分页信息不符合预期: %#v", agentsPage.Pagination)
	}
	doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/collector-dev/agent/disconnect", agent.AgentToken, map[string]any{})
	disconnectedResponse := doJSONRequest(t, http.MethodGet, server.URL+"/api/v1/data/collector-dev/agents?page=1&pageSize=10", adminToken, nil)
	mustDecodeCollectorData(t, disconnectedResponse.Data, &agentsPage)
	if agentsPage.List[0].Status != "offline" {
		t.Fatalf("主动断开后期望 offline，实际 %q", agentsPage.List[0].Status)
	}
	doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/collector-dev/agent/heartbeat", agent.AgentToken, map[string]any{"capabilities": capabilities})
	connectionResponse := doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/collector/connections", adminToken, map[string]any{
		"name": "integration opcua", "driverId": "opcua.standard",
		"config":  map[string]any{"host": "127.0.0.1", "port": 18540, "endpointPath": "/induforge/sim", "securityMode": "None", "securityPolicy": "None", "authenticationType": "anonymous"},
		"secrets": map[string]string{}, "metadata": map[string]any{},
	})
	var connection struct {
		ID string `json:"id"`
	}
	mustDecodeCollectorData(t, connectionResponse.Data, &connection)
	created := doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/collector-dev/tasks", adminToken, map[string]any{
		"agentId":        agent.AgentID,
		"connectionId":   connection.ID,
		"operation":      "connection.test",
		"input":          map[string]any{},
		"timeoutSeconds": 30,
	})
	var task struct {
		TaskID string `json:"taskId"`
		Status string `json:"status"`
	}
	mustDecodeCollectorData(t, created.Data, &task)
	if task.Status != "queued" {
		t.Fatalf("期望 queued，实际 %q", task.Status)
	}
	claimed := doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/collector-dev/agent/tasks/claim", agent.AgentToken, map[string]any{})
	mustDecodeCollectorData(t, claimed.Data, &task)
	if task.Status != "running" {
		t.Fatalf("期望 running，实际 %q", task.Status)
	}
	doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/collector-dev/agent/tasks/"+task.TaskID+"/complete", agent.AgentToken, map[string]any{"status": "succeeded", "result": map[string]any{"connected": true}, "error": nil})
	queried := doJSONRequest(t, http.MethodGet, server.URL+"/api/v1/data/projects/"+projectID+"/collector-dev/tasks/"+task.TaskID, adminToken, nil)
	mustDecodeCollectorData(t, queried.Data, &task)
	if task.Status != "succeeded" {
		t.Fatalf("期望 succeeded，实际 %q", task.Status)
	}
	pointResponse := doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/collector/connections/"+connection.ID+"/points/batch", adminToken, map[string]any{
		"points": []map[string]any{{
			"name": "Temperature", "address": map[string]any{"nodeId": "ns=2;s=Temperature"},
			"dataType": "float32", "elementCount": 1, "readOptions": map[string]any{},
			"acquisitionMode": "inherit", "acquisitionOverrides": map[string]any{},
			"enabled": true, "sortOrder": 0, "metadata": map[string]any{},
		}},
	})
	var pointBatch struct {
		List []struct {
			ID string `json:"id"`
		} `json:"list"`
	}
	mustDecodeCollectorData(t, pointResponse.Data, &pointBatch)
	if len(pointBatch.List) != 1 {
		t.Fatalf("期望创建 1 个变量，实际 %#v", pointBatch.List)
	}
	pointID := pointBatch.List[0].ID
	workspaceSessionID := uuid.NewString()

	readTaskResponse := doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/collector-dev/tasks", adminToken, map[string]any{
		"agentId": agent.AgentID, "connectionId": connection.ID, "operation": "point.read",
		"input": map[string]any{"workspaceSessionId": workspaceSessionID, "pointIds": []string{pointID}}, "timeoutSeconds": 30,
	})
	mustDecodeCollectorData(t, readTaskResponse.Data, &task)
	claimed = doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/collector-dev/agent/tasks/claim", agent.AgentToken, map[string]any{})
	mustDecodeCollectorData(t, claimed.Data, &task)
	doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/collector-dev/agent/tasks/"+task.TaskID+"/complete", agent.AgentToken, map[string]any{
		"status": "succeeded",
		"result": map[string]any{"values": []map[string]any{{
			"pointId": pointID, "succeeded": true, "value": 12.5, "dataType": "float64", "quality": "Good",
			"sourceTimestamp": "2026-07-20T01:02:03Z", "serverTimestamp": "2026-07-20T01:02:04Z",
			"errorCode": nil, "errorMessage": nil,
		}}, "diagnostics": []any{}},
		"error": nil,
	})

	type pointSnapshotPage struct {
		List []struct {
			ID                  string `json:"id"`
			LatestDebugSnapshot *struct {
				Value             float64 `json:"value"`
				ValueText         string  `json:"valueText"`
				Quality           string  `json:"quality"`
				LastAttemptStatus string  `json:"lastAttemptStatus"`
				LastErrorMessage  *string `json:"lastErrorMessage"`
			} `json:"latestDebugSnapshot"`
		} `json:"list"`
	}
	loadSnapshotPage := func() pointSnapshotPage {
		response := doJSONRequest(t, http.MethodGet, server.URL+"/api/v1/data/projects/"+projectID+"/collector/connections/"+connection.ID+"/points?page=1&pageSize=50", adminToken, nil)
		var page pointSnapshotPage
		mustDecodeCollectorData(t, response.Data, &page)
		return page
	}
	page := loadSnapshotPage()
	if len(page.List) != 1 || page.List[0].LatestDebugSnapshot == nil || page.List[0].LatestDebugSnapshot.ValueText != "12.5" || page.List[0].LatestDebugSnapshot.Quality != "Good" {
		t.Fatalf("变量最近调试快照不符合预期: %#v", page.List)
	}

	failedTaskResponse := doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/collector-dev/tasks", adminToken, map[string]any{
		"agentId": agent.AgentID, "connectionId": connection.ID, "operation": "point.read",
		"input": map[string]any{"workspaceSessionId": workspaceSessionID, "pointIds": []string{pointID}}, "timeoutSeconds": 30,
	})
	mustDecodeCollectorData(t, failedTaskResponse.Data, &task)
	claimed = doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/collector-dev/agent/tasks/claim", agent.AgentToken, map[string]any{})
	mustDecodeCollectorData(t, claimed.Data, &task)
	doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/collector-dev/agent/tasks/"+task.TaskID+"/complete", agent.AgentToken, map[string]any{
		"status": "failed", "result": nil,
		"error": map[string]any{"code": "COLLECTOR_SESSION_LOST", "message": "连接已断开", "retryable": true},
	})
	page = loadSnapshotPage()
	snapshot := page.List[0].LatestDebugSnapshot
	if snapshot == nil || snapshot.Value != 12.5 || snapshot.LastAttemptStatus != "failed" || snapshot.LastErrorMessage == nil || *snapshot.LastErrorMessage != "连接已断开" {
		t.Fatalf("读取失败后应保留最近成功值: %#v", snapshot)
	}

	doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/collector-dev/agent/revoke", agent.AgentToken, map[string]any{})
	doJSONRequestWithStatus(t, http.MethodPost, server.URL+"/api/v1/data/collector-dev/agent/heartbeat", agent.AgentToken, map[string]any{"capabilities": capabilities}, http.StatusUnauthorized)
	invalidResponse := doJSONRequest(t, http.MethodGet, server.URL+"/api/v1/data/collector-dev/agents?page=1&pageSize=10", adminToken, nil)
	mustDecodeCollectorData(t, invalidResponse.Data, &agentsPage)
	if agentsPage.List[0].Status != "invalid" {
		t.Fatalf("撤销后期望 invalid，实际 %q", agentsPage.List[0].Status)
	}

	secondCodeResponse := doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/collector-dev/registration-codes", adminToken, map[string]any{})
	mustDecodeCollectorData(t, secondCodeResponse.Data, &registrationPayload)
	secondRegistration := doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/collector-dev/agent/register", "", map[string]any{
		"registrationCode": registrationPayload.Code,
		"machineId":        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"name":             "integration-agent-reconnected",
		"os":               "windows",
		"arch":             "x64",
		"version":          "0.2.0",
		"capabilities":     capabilities,
	})
	var reactivated struct {
		AgentID    string `json:"agentId"`
		AgentToken string `json:"agentToken"`
	}
	mustDecodeCollectorData(t, secondRegistration.Data, &reactivated)
	if reactivated.AgentID != agent.AgentID {
		t.Fatalf("同一机器重新注册应复用节点 ID，原 %q，新 %q", agent.AgentID, reactivated.AgentID)
	}
	if reactivated.AgentToken == agent.AgentToken {
		t.Fatal("重新注册必须轮换 Agent Token")
	}
	doJSONRequest(t, http.MethodDelete, server.URL+"/api/v1/data/collector-dev/agents/"+reactivated.AgentID, adminToken, nil)
	doJSONRequestWithStatus(t, http.MethodPost, server.URL+"/api/v1/data/collector-dev/agent/heartbeat", reactivated.AgentToken, map[string]any{"capabilities": capabilities}, http.StatusUnauthorized)
}

func mustDecodeCollectorData(t *testing.T, raw json.RawMessage, target any) {
	t.Helper()
	if err := json.Unmarshal(raw, target); err != nil {
		t.Fatalf("解析采集调试响应失败: %v", err)
	}
}
