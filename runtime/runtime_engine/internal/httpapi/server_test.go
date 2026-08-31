package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/indu-forge/runtime-engine/internal/loader"
	"github.com/indu-forge/runtime-engine/internal/model"
)

func TestStatusUsesStrictSuccessEnvelope(t *testing.T) {
	state := NewEngineState("test", nil, nil)
	response := httptest.NewRecorder()
	state.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/status", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d", response.Code)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 4 || body["code"] == nil || body["msg"] == nil || body["data"] == nil || body["reqId"] == nil {
		t.Fatalf("not a strict envelope: %s", response.Body.String())
	}
	var code int
	if err := json.Unmarshal(body["code"], &code); err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
}

func TestHTTPPathsAndLifecycleHealth(t *testing.T) {
	state := NewEngineState("test", nil, nil)
	for _, path := range []string{"/health", "/unknown"} {
		response := httptest.NewRecorder()
		state.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusServiceUnavailable && response.Code != http.StatusNotFound {
			t.Fatalf("%s=%d", path, response.Code)
		}
	}
	response := httptest.NewRecorder()
	state.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/health", nil))
	if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != "GET" {
		t.Fatalf("405=%d allow=%q", response.Code, response.Header().Get("Allow"))
	}
	state.SetState(Running, Healthy, "")
	response = httptest.NewRecorder()
	state.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("running=%d", response.Code)
	}
	state.SetState(Stopping, Unavailable, "")
	state.ReportDegraded("late")
	response = httptest.NewRecorder()
	state.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("draining must stay down=%d", response.Code)
	}
}
func TestFreshnessDoesNotChangeHealth(t *testing.T) {
	state := NewEngineState("test", nil, nil)
	state.SetState(Running, Healthy, "")
	state.RecordBusinessSuccess(time.Now().UTC())
	if got := state.snapshot(); got.BusinessFreshness.State != "FRESH" || got.HealthState != "HEALTHY" {
		t.Fatalf("fresh=%+v", got)
	}
	state.RecordBusinessSuccess(time.Now().UTC().Add(-3 * time.Minute))
	if got := state.snapshot(); got.BusinessFreshness.State != "STALE" || got.HealthState != "HEALTHY" {
		t.Fatalf("stale=%+v", got)
	}
}

func TestStatusReportsValidationFailure(t *testing.T) {
	state := NewEngineState("test", nil, errors.New("/trusted/mount artifact sha256:secret"))
	response := httptest.NewRecorder()
	state.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", response.Code)
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code == 0 || body.Data.Status != "DOWN" {
		t.Fatalf("failure state=%+v", body)
	}
}

func TestNativeEngineStatusReportsNodeAndProcess(t *testing.T) {
	state := NewEngineState("test", &loader.Loaded{Config: model.EngineConfig{
		SiteID: "site-a", NodeID: "node-line1-01", ProjectID: "11111111-1111-4111-8111-111111111111",
		DeploymentID: "deployment-line1-prod", AccountID: "account-line1", ExecutionForm: "native-linux",
	}}, nil)
	value := state.snapshot()
	if value.NodeID != "node-line1-01" || value.ProcessID < 1 {
		t.Fatalf("native status missing process identity: %+v", value)
	}
}

func TestFailedStateCannotRegressToRunning(t *testing.T) {
	state := NewEngineState("test", nil, nil)
	state.SetState(Failed, Unavailable, "INGRESS_FATAL")
	state.SetState(Running, Healthy, "")
	response := httptest.NewRecorder()
	state.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("failed state regressed: %d", response.Code)
	}
}
