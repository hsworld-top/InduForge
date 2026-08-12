package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestHealthAndEnvelope(t *testing.T) {
	application := New(Options{
		RequestID: func() string { return "test-request" },
	})

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()
	application.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", response.Code, http.StatusOK)
	}

	var payload struct {
		Code  int            `json:"code"`
		Msg   string         `json:"msg"`
		Data  map[string]any `json:"data"`
		ReqID string         `json:"reqId"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != 0 || payload.Msg != "success" || payload.ReqID != "test-request" {
		t.Fatalf("unexpected envelope: %#v", payload)
	}
	if payload.Data["status"] != "ok" {
		t.Fatalf("unexpected health data: %#v", payload.Data)
	}
}

func TestPanicRecoveryUsesUnifiedEnvelope(t *testing.T) {
	application := New(Options{
		RequestID: func() string { return "panic-request" },
		Mount: func(router chi.Router) {
			router.Get("/panic", func(http.ResponseWriter, *http.Request) {
				panic("boom")
			})
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
	response := httptest.NewRecorder()
	application.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status: got %d want %d", response.Code, http.StatusInternalServerError)
	}

	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["code"] != float64(30001) || payload["reqId"] != "panic-request" {
		t.Fatalf("unexpected error envelope: %#v", payload)
	}
}
