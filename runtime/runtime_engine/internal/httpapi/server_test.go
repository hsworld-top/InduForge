package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatusUsesStrictSuccessEnvelope(t *testing.T) {
	state := NewEngineState("test", nil, nil)
	response := httptest.NewRecorder()
	state.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/runtime/status", nil))
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
			LifecycleState string  `json:"lifecycleState"`
			HealthState    string  `json:"healthState"`
			LastError      *string `json:"lastError"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != 1 || body.Data.LifecycleState != "FAILED" || body.Data.HealthState != "UNAVAILABLE" || body.Data.LastError == nil || *body.Data.LastError != "运行配置或项目产物校验未通过" {
		t.Fatalf("failure state=%+v", body)
	}
}
