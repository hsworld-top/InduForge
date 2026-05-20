package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/indu-forge/data_service/internal/app"
	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/config"
	apperrors "github.com/indu-forge/data_service/internal/errors"
)

func TestAlarmRuleAPIFinalContract(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupMigrator(t, fixture.pool)
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("migrate up failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "alarm-rule-secret-01"
	server, token := startAlarmRuleIntegrationServer(t, fixture.databaseURL, fixture.schemaName, secret, userID, projectID)

	datapointID := uuid.NewString()
	insertActiveAlarmDatapoint(t, ctx, fixture.pool, projectID, datapointID, "metrics.temperature", "number", userID)

	createEnvelope := doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/alarm-rules", token, map[string]any{
		"name":            "温度高报",
		"targetPath":      "metrics.temperature",
		"ruleType":        "H",
		"condition":       map[string]any{"limit": 80, "hysteresis": 2, "durationMs": 30000},
		"severity":        "major",
		"suppression":     map[string]any{"enabled": false},
		"messageTemplate": "{{targetPath}} {{value}}",
	})
	var rule alarmRulePayload
	decodeAlarmRuleData(t, createEnvelope.Data, &rule)
	if rule.RuleType != "H" || rule.Severity != "major" || rule.TargetDataType != "number" {
		t.Fatalf("unexpected created rule: %#v", rule)
	}
	if rule.Suppression["enabled"] != false || rule.MessageTemplate != "{{targetPath}} {{value}}" {
		t.Fatalf("unexpected persisted extension fields: %#v", rule)
	}

	listEnvelope := doJSONRequest(t, http.MethodGet, server.URL+"/api/v1/data/projects/"+projectID+"/alarm-rules?search=温度&page=1&pageSize=10", token, nil)
	var list alarmRuleListPayload
	if err := json.Unmarshal(listEnvelope.Data, &list); err != nil {
		t.Fatalf("decode list response failed: %v", err)
	}
	if len(list.List) != 1 || list.Pagination.Total != 1 {
		t.Fatalf("unexpected list response: %#v", list)
	}

	detailEnvelope := doJSONRequest(t, http.MethodGet, server.URL+"/api/v1/data/projects/"+projectID+"/alarm-rules/"+rule.ID, token, nil)
	var detail alarmRulePayload
	decodeAlarmRuleData(t, detailEnvelope.Data, &detail)
	if detail.ID != rule.ID || detail.TargetDatapointID != datapointID {
		t.Fatalf("unexpected detail response: %#v", detail)
	}

	updateEnvelope := doJSONRequest(t, http.MethodPut, server.URL+"/api/v1/data/projects/"+projectID+"/alarm-rules/"+rule.ID, token, map[string]any{
		"name":            "温度高高报",
		"targetPath":      "metrics.temperature",
		"ruleType":        "HH",
		"condition":       map[string]any{"limit": 90},
		"severity":        "critical",
		"suppression":     map[string]any{"enabled": true, "durationMs": 60000},
		"messageTemplate": "{{targetPath}} critical {{value}}",
	})
	var updated alarmRulePayload
	decodeAlarmRuleData(t, updateEnvelope.Data, &updated)
	if updated.RuleType != "HH" || updated.Severity != "critical" || updated.Suppression["enabled"] != true {
		t.Fatalf("unexpected updated rule: %#v", updated)
	}

	toggleEnvelope := doJSONRequest(t, http.MethodPatch, server.URL+"/api/v1/data/projects/"+projectID+"/alarm-rules/"+rule.ID+"/enabled", token, map[string]any{
		"isEnabled": false,
	})
	var toggled alarmRulePayload
	decodeAlarmRuleData(t, toggleEnvelope.Data, &toggled)
	if toggled.IsEnabled {
		t.Fatalf("expected disabled rule: %#v", toggled)
	}

	validateEnvelope := doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/alarm-rules/validate-draft", token, map[string]any{
		"name":       "草稿检查",
		"targetPath": "metrics.temperature",
		"ruleType":   "H",
		"condition":  map[string]any{"limit": 80},
		"severity":   "major",
	})
	var draft alarmRuleDraftValidationPayload
	if err := json.Unmarshal(validateEnvelope.Data, &draft); err != nil {
		t.Fatalf("decode validate-draft response failed: %v", err)
	}
	if !draft.Valid || draft.Contract["schemaVersion"] != "alarm.rule.v1" {
		t.Fatalf("unexpected validate-draft response: %#v", draft)
	}

	testEnvelope := doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/alarm-rules/"+rule.ID+"/test", token, map[string]any{
		"value": 92,
	})
	var trial alarmRuleTrialPayload
	if err := json.Unmarshal(testEnvelope.Data, &trial); err != nil {
		t.Fatalf("decode trial response failed: %v", err)
	}
	if trial.State != "triggered" || !trial.Triggered {
		t.Fatalf("trial = %#v, want triggered", trial)
	}

	contractEnvelope := doJSONRequest(t, http.MethodGet, server.URL+"/api/v1/data/projects/"+projectID+"/alarm-rules/"+rule.ID+"/contract", token, nil)
	var contract map[string]any
	if err := json.Unmarshal(contractEnvelope.Data, &contract); err != nil {
		t.Fatalf("decode contract response failed: %v", err)
	}
	if contract["schemaVersion"] != "alarm.rule.v1" || contract["enabled"] != false {
		t.Fatalf("unexpected contract response: %#v", contract)
	}

	deleteEnvelope := doJSONRequest(t, http.MethodDelete, server.URL+"/api/v1/data/projects/"+projectID+"/alarm-rules/"+rule.ID, token, nil)
	var deleted struct {
		Deleted bool `json:"deleted"`
	}
	if err := json.Unmarshal(deleteEnvelope.Data, &deleted); err != nil {
		t.Fatalf("decode delete response failed: %v", err)
	}
	if !deleted.Deleted {
		t.Fatalf("unexpected delete response: %#v", deleted)
	}
}

type alarmRulePayload struct {
	ID                string         `json:"id"`
	ProjectID         string         `json:"projectId"`
	Name              string         `json:"name"`
	TargetDatapointID string         `json:"targetDatapointId"`
	TargetPath        string         `json:"targetPath"`
	TargetDataType    string         `json:"targetDataType"`
	RuleType          string         `json:"ruleType"`
	Condition         map[string]any `json:"condition"`
	Severity          string         `json:"severity"`
	IsEnabled         bool           `json:"isEnabled"`
	Suppression       map[string]any `json:"suppression"`
	MessageTemplate   string         `json:"messageTemplate"`
	Contract          map[string]any `json:"contract"`
}

type alarmRuleListPayload struct {
	List       []alarmRulePayload `json:"list"`
	Pagination struct {
		Page  int `json:"page"`
		Total int `json:"total"`
	} `json:"pagination"`
}

type alarmRuleTrialPayload struct {
	Triggered bool   `json:"triggered"`
	State     string `json:"state"`
}

type alarmRuleDraftValidationPayload struct {
	Valid    bool           `json:"valid"`
	Contract map[string]any `json:"contract"`
}

func startAlarmRuleIntegrationServer(t *testing.T, databaseURL, schemaName, secret, userID, projectID string) (*httptest.Server, string) {
	t.Helper()

	srv, err := app.NewServer(config.Config{
		Addr:               ":0",
		DatabaseURL:        databaseURL,
		DatabaseSearchPath: schemaName,
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
		TenantID:     "tenant-alarm-rule",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})
	return server, token
}

func insertActiveAlarmDatapoint(t *testing.T, ctx context.Context, pool *pgxpool.Pool, projectID, id, path, dataType, userID string) {
	t.Helper()

	if _, err := pool.Exec(ctx, `
        INSERT INTO data_points (
            id, project_id, path, name, source_type, source_config, data_type, status, created_by
        )
        VALUES ($1, $2, $3, $4, 'manual', '{}'::jsonb, $5, 'active', $6)
    `, id, projectID, path, path, dataType, userID); err != nil {
		t.Fatalf("insert alarm datapoint failed: %v", err)
	}
}

func decodeAlarmRuleData(t *testing.T, raw json.RawMessage, rule *alarmRulePayload) {
	t.Helper()

	if err := json.Unmarshal(raw, rule); err != nil {
		t.Fatalf("decode alarm rule response failed: %v", err)
	}
	if rule.ID == "" {
		t.Fatalf("expected alarm rule id, got %#v", rule)
	}
	if rule.Contract == nil {
		t.Fatalf("expected alarm rule contract, got %#v", rule)
	}
	if rule.Contract["schemaVersion"] != "alarm.rule.v1" {
		t.Fatalf("expected alarm contract schemaVersion, got %#v", rule.Contract)
	}
}

func TestAlarmRuleAPIDraftValidationError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupMigrator(t, fixture.pool)
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("migrate up failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "alarm-rule-invalid-secret-01"
	server, token := startAlarmRuleIntegrationServer(t, fixture.databaseURL, fixture.schemaName, secret, userID, projectID)

	insertActiveAlarmDatapoint(t, ctx, fixture.pool, projectID, uuid.NewString(), "metrics.switch", "string", userID)

	envelope := doJSONRequestWithStatus(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/alarm-rules/validate-draft", token, map[string]any{
		"name":       "非法草稿",
		"targetPath": "metrics.switch",
		"ruleType":   "H",
		"condition":  map[string]any{"limit": 1},
		"severity":   "major",
	}, http.StatusBadRequest)
	if envelope.Code != apperrors.PublicCodeBadRequest {
		t.Fatalf("expected bad request code, got %d msg=%q", envelope.Code, envelope.Msg)
	}
}
