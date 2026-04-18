package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/indu-forge/data_service/internal/http/middleware"
)

func TestRequestIDMiddleware_UsesIncomingHeader(t *testing.T) {
	const requestID = "req-from-header"

	handler := middleware.RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(middleware.RequestID(r.Context())))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", requestID)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get("X-Request-ID"); got != requestID {
		t.Fatalf("expected response header %q, got %q", requestID, got)
	}
	if got := rr.Body.String(); got != requestID {
		t.Fatalf("expected body %q, got %q", requestID, got)
	}
}

func TestRequestIDMiddleware_GeneratesAndEchoesRequestID(t *testing.T) {
	handler := middleware.RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(middleware.RequestID(r.Context())))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	requestID := rr.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Fatal("expected generated request id to be set")
	}
	if got := rr.Body.String(); got != requestID {
		t.Fatalf("expected body to echo request id %q, got %q", requestID, got)
	}
}
