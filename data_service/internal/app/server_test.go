package app

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/indu-forge/data_service/internal/config"
	"github.com/indu-forge/data_service/internal/http/response"
)

func TestServer_RealChainInjectsRequestID(t *testing.T) {
	srv := NewServer(config.Config{Addr: ":0"})
	ts := httptest.NewServer(srv.httpServer.Handler)
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

func TestServer_RealChainConvertsErrorsToUnifiedApiResponse(t *testing.T) {
	srv := NewServer(config.Config{Addr: ":0"})
	ts := httptest.NewServer(srv.httpServer.Handler)
	t.Cleanup(ts.Close)

	cases := []struct {
		name        string
		path        string
		wantStatus  int
		wantCode    string
		wantMessage string
	}{
		{
			name:        "app error",
			path:        "/api/v1/_internal/error-handler/app-error",
			wantStatus:  http.StatusBadRequest,
			wantCode:    "BAD_REQUEST",
			wantMessage: "参数错误",
		},
		{
			name:        "normal error",
			path:        "/api/v1/_internal/error-handler/error",
			wantStatus:  http.StatusInternalServerError,
			wantCode:    "INTERNAL_ERROR",
			wantMessage: "系统内部错误",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ts.Client().Get(ts.URL + tc.path)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, resp.StatusCode)
			}
			if got := resp.Header.Get("X-Request-ID"); got == "" {
				t.Fatal("expected X-Request-ID header to be set")
			}

			var payload response.ApiResponse
			if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
				t.Fatalf("decode response failed: %v", err)
			}

			if payload.Success {
				t.Fatal("expected success to be false")
			}
			if payload.ErrorCode != tc.wantCode {
				t.Fatalf("expected errorCode %q, got %q", tc.wantCode, payload.ErrorCode)
			}
			if payload.Message != tc.wantMessage {
				t.Fatalf("expected message %q, got %q", tc.wantMessage, payload.Message)
			}
			if payload.RequestID == "" {
				t.Fatal("expected requestId to be set")
			}
			if payload.RequestID != resp.Header.Get("X-Request-ID") {
				t.Fatalf("expected response body requestId %q to match header %q", payload.RequestID, resp.Header.Get("X-Request-ID"))
			}
		})
	}
}
