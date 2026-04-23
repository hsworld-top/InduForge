package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/handler"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/http/router"
)

func TestHealthHandler_ReturnsHealthy(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	handler.NewHealthHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var payload response.ApiResponse
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode health response failed: %v", err)
	}
	if payload.Code != apperrors.SuccessCode {
		t.Fatalf("expected code %d, got %d", apperrors.SuccessCode, payload.Code)
	}
	var data map[string]string
	raw, err := json.Marshal(payload.Data)
	if err != nil {
		t.Fatalf("marshal health data failed: %v", err)
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("decode health data failed: %v", err)
	}
	if data["status"] != "healthy" {
		t.Fatalf("expected status=healthy, got %q", data["status"])
	}
}

func TestRouter_HealthRouteMounted(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	router.NewRouter().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var payload response.ApiResponse
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode health response failed: %v", err)
	}
	if payload.Code != apperrors.SuccessCode {
		t.Fatalf("expected code %d, got %d", apperrors.SuccessCode, payload.Code)
	}
	var data map[string]string
	raw, err := json.Marshal(payload.Data)
	if err != nil {
		t.Fatalf("marshal health data failed: %v", err)
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("decode health data failed: %v", err)
	}
	if data["status"] != "healthy" {
		t.Fatalf("expected status=healthy, got %q", data["status"])
	}
}
