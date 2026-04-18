package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/indu-forge/data_service/internal/http/middleware"
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

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

func TestRequestIDMiddleware_GeneratesForInvalidHeader(t *testing.T) {
	const invalidRequestID = "bad id with spaces and punctuation!"

	handler := middleware.RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(middleware.RequestID(r.Context())))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", invalidRequestID)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	requestID := rr.Header().Get("X-Request-ID")
	if requestID == invalidRequestID {
		t.Fatal("expected invalid request id to be replaced")
	}
	if requestID == "" {
		t.Fatal("expected generated request id to be set")
	}
	if !requestIDPattern.MatchString(requestID) {
		t.Fatalf("expected generated request id to match whitelist, got %q", requestID)
	}
	if got := strings.TrimSpace(rr.Body.String()); got != requestID {
		t.Fatalf("expected body to echo generated request id %q, got %q", requestID, got)
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
	if !requestIDPattern.MatchString(requestID) {
		t.Fatalf("expected generated request id to match whitelist, got %q", requestID)
	}
	if got := rr.Body.String(); got != requestID {
		t.Fatalf("expected body to echo request id %q, got %q", requestID, got)
	}
}
