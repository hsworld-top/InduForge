package runtimeengine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/indu-forge/runtime-engine/internal/httpapi"
)

// These lifecycle tests use only cancellable worker fakes; they never start a
// database, NATS server, or HTTP listener.
func TestShutdownCancelsWorkersThenStops(t *testing.T) {
	h := New(Options{Version: "test"})
	ctx, cancel := context.WithCancel(context.Background())
	h.cancel = cancel
	h.intakeCancel = cancel
	h.State.SetState(httpapi.Running, httpapi.Healthy, "")
	h.wg.Add(1)
	go func() { defer h.wg.Done(); <-ctx.Done() }()
	shutdown, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()
	if err := h.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	h.State.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("stopped health=%d", response.Code)
	}
}

func TestShutdownTimeoutStaysFailed(t *testing.T) {
	h := New(Options{Version: "test"})
	_, cancel := context.WithCancel(context.Background())
	h.cancel = cancel
	h.intakeCancel = cancel
	h.State.SetState(httpapi.Running, httpapi.Healthy, "")
	h.wg.Add(1)
	defer h.wg.Done()
	ctx, stop := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer stop()
	if err := h.Shutdown(ctx); err == nil {
		t.Fatal("drain timeout must fail")
	}
	response := httptest.NewRecorder()
	h.State.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("failed health=%d", response.Code)
	}
}

func TestWorkerFatalCancelsRootAndMarksFailed(t *testing.T) {
	h := New(Options{})
	ctx, cancel := context.WithCancel(context.Background())
	h.cancel = cancel
	h.State.SetState(httpapi.Running, httpapi.Healthy, "")
	h.workerFatal("INGRESS_FATAL")
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("root was not cancelled")
	}
	response := httptest.NewRecorder()
	h.State.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("fatal health=%d", response.Code)
	}
}

func TestConcurrentShutdownSharesDrainResult(t *testing.T) {
	h := New(Options{})
	work, cancel := context.WithCancel(context.Background())
	h.cancel = cancel
	h.intakeCancel = cancel
	h.State.SetState(httpapi.Running, httpapi.Healthy, "")
	h.wg.Add(1)
	go func() { defer h.wg.Done(); <-work.Done() }()
	results := make(chan error, 2)
	for range 2 {
		go func() {
			ctx, stop := context.WithTimeout(context.Background(), time.Second)
			defer stop()
			results <- h.Shutdown(ctx)
		}()
	}
	for range 2 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	if h.store != nil || h.nats != nil {
		t.Fatal("resources must be cleared")
	}
}

func TestShutdownBeforeStartEndsStopped(t *testing.T) {
	h := New(Options{})
	if err := h.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	h.State.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("state=%d", response.Code)
	}
}
