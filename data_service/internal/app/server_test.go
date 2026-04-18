package app

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/indu-forge/data_service/internal/config"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/http/router"
)

func TestNewServer_UsesProductionRouter(t *testing.T) {
	restore := replaceRouteDependenciesFactoryForTest(func(config.Config) ([]router.Option, func(), error) {
		return nil, nil, nil
	})
	defer restore()

	srv, err := NewServer(config.Config{Addr: ":0"})
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	resp, err := ts.Client().Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
	if got := resp.Header.Get("X-Request-ID"); got == "" {
		t.Fatal("expected X-Request-ID header to be set")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body failed: %v", err)
	}
	if string(body) != "healthy\n" {
		t.Fatalf("expected healthy body, got %q", string(body))
	}
}

func TestServer_ProductionAssemblyKeepsUnifiedErrorResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/", router.NewRouter())
	mux.Handle("/boom", middleware.ErrorHandler(func(http.ResponseWriter, *http.Request) error {
		return errors.New("forced boom")
	}))

	srv := newServer(config.Config{Addr: ":0"}, middleware.RequestIDMiddleware(mux), nil)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	resp, err := ts.Client().Get(ts.URL + "/boom")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
	}
	requestID := resp.Header.Get("X-Request-ID")
	if requestID == "" {
		t.Fatal("expected X-Request-ID header to be set")
	}

	var payload response.ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	if payload.Success {
		t.Fatal("expected success to be false")
	}
	if payload.ErrorCode != string(apperrors.ErrorCodeInternal) {
		t.Fatalf("expected errorCode %q, got %q", apperrors.ErrorCodeInternal, payload.ErrorCode)
	}
	if payload.Message != "系统内部错误" {
		t.Fatalf("expected message %q, got %q", "系统内部错误", payload.Message)
	}
	if payload.RequestID != requestID {
		t.Fatalf("expected requestId %q to match header %q", payload.RequestID, requestID)
	}
}

func replaceRouteDependenciesFactoryForTest(factory routeDependenciesFactory) func() {
	previous := buildRouteDependencies
	buildRouteDependencies = factory
	return func() {
		buildRouteDependencies = previous
	}
}
