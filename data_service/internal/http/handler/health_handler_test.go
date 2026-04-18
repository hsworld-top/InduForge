package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/indu-forge/data_service/internal/http/handler"
	"github.com/indu-forge/data_service/internal/http/router"
)

func TestHealthHandler_ReturnsHealthy(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	handler.NewHealthHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Body.String() != "healthy\n" {
		t.Fatalf("expected healthy body, got %q", rr.Body.String())
	}
}

func TestRouter_HealthRouteMounted(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	router.NewRouter().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Body.String() != "healthy\n" {
		t.Fatalf("expected healthy body, got %q", rr.Body.String())
	}
}
