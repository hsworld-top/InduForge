package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/indu-forge/data_service/internal/config"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/http/router"
)

func TestNewServer_UsesProductionRouter(t *testing.T) {
	reset := stubBuildRouteDependencies(func(config.Config) ([]router.Option, func(), error) {
		return nil, nil, nil
	})
	t.Cleanup(reset)

	srv, err := NewServer(config.Config{Addr: ":0"})
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}
	t.Cleanup(srv.Close)

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
	t.Cleanup(srv.Close)
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

func TestNewServer_FailsWhenRouteDependenciesInitializationFails(t *testing.T) {
	expectedErr := errors.New("bootstrap failed")
	reset := stubBuildRouteDependencies(func(config.Config) ([]router.Option, func(), error) {
		return nil, nil, expectedErr
	})
	t.Cleanup(reset)

	srv, err := NewServer(config.Config{Addr: ":0"})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
	if srv != nil {
		t.Fatal("expected server to be nil when dependencies initialization fails")
	}
}

func TestNewServer_LogsInitializationSummary(t *testing.T) {
	var buffer bytes.Buffer
	resetLog := stubAppLogf(func(format string, args ...any) {
		_, _ = buffer.WriteString(strings.TrimSpace(formatLogf(format, args...)) + "\n")
	})
	t.Cleanup(resetLog)

	resetRoutes := stubBuildRouteDependencies(func(config.Config) ([]router.Option, func(), error) {
		logf("info: data_service 路由摘要 routeSummary=connections=enabled,data=enabled,mqtt=enabled,preview=disabled")
		return nil, nil, nil
	})
	t.Cleanup(resetRoutes)

	srv, err := NewServer(config.Config{Addr: ":19602"})
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}
	t.Cleanup(srv.Close)

	logOutput := buffer.String()
	if !strings.Contains(logOutput, "data_service 路由装配完成") {
		t.Fatalf("expected initialization log, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "addr=:19602") {
		t.Fatalf("expected addr in initialization log, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "routeSummary=") {
		t.Fatalf("expected route summary in initialization log, got %q", logOutput)
	}
}

func TestServer_Run_LogsLifecycle(t *testing.T) {
	var buffer bytes.Buffer
	resetLog := stubAppLogf(func(format string, args ...any) {
		_, _ = buffer.WriteString(strings.TrimSpace(formatLogf(format, args...)) + "\n")
	})
	t.Cleanup(resetLog)

	srv := newServer(config.Config{Addr: "127.0.0.1:0"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), nil)
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)

	go func() {
		errCh <- srv.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	if err := <-errCh; err != nil {
		t.Fatalf("run failed: %v", err)
	}

	logOutput := buffer.String()
	if !strings.Contains(logOutput, "data_service 准备启动 HTTP 服务") {
		t.Fatalf("expected startup log, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "data_service 收到退出信号") {
		t.Fatalf("expected shutdown log, got %q", logOutput)
	}
}

func stubBuildRouteDependencies(factory routeDependenciesFactory) func() {
	previous := buildRouteDependencies
	buildRouteDependencies = factory
	return func() {
		buildRouteDependencies = previous
	}
}

func stubAppLogf(logger func(string, ...any)) func() {
	previous := logf
	logf = logger
	return func() {
		logf = previous
	}
}

func formatLogf(format string, args ...any) string {
	return strings.TrimSpace(fmt.Sprintf(format, args...))
}
