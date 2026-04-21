package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAccessLogMiddleware_LogsRequestSummary(t *testing.T) {
	messages := make([]string, 0, 1)
	reset := stubAccessLogf(func(format string, args ...any) {
		messages = append(messages, fmt.Sprintf(format, args...))
	})
	t.Cleanup(reset)

	handler := RequestIDMiddleware(AccessLogMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	})))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/data/projects/demo/connections?debug=1", nil)
	req.Header.Set("X-Request-ID", "req-123")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if len(messages) != 1 {
		t.Fatalf("expected exactly one log message, got %d", len(messages))
	}
	logLine := messages[0]
	for _, expected := range []string{
		"requestId=req-123",
		"method=POST",
		"uri=/api/v1/data/projects/demo/connections?debug=1",
		"status=201",
		"bytes=2",
	} {
		if !strings.Contains(logLine, expected) {
			t.Fatalf("expected log line %q to contain %q", logLine, expected)
		}
	}
}

func stubAccessLogf(logger func(string, ...any)) func() {
	previous := accessLogf
	accessLogf = logger
	return func() {
		accessLogf = previous
	}
}
