package middleware_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
)

func TestErrorHandler_WrapsAppErrorInUnifiedResponse(t *testing.T) {
	handler := middleware.ErrorHandler(func(w http.ResponseWriter, r *http.Request) error {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "参数错误")
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(middleware.WithRequestID(req.Context(), "req-app-error"))

	handler.ServeHTTP(rr, req)

	assertApiErrorResponse(t, rr, http.StatusBadRequest, "req-app-error", apperrors.PublicCodeBadRequest, "参数错误")
}

func TestErrorHandler_WrapsNormalErrorInUnifiedResponse(t *testing.T) {
	handler := middleware.ErrorHandler(func(w http.ResponseWriter, r *http.Request) error {
		return errors.New("普通错误")
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(middleware.WithRequestID(req.Context(), "req-normal-error"))

	handler.ServeHTTP(rr, req)

	assertApiErrorResponse(t, rr, http.StatusInternalServerError, "req-normal-error", apperrors.PublicCodeInternal, "系统内部错误")
}

func TestErrorHandler_FallsBackForInvalidStatusCode(t *testing.T) {
	handler := middleware.ErrorHandler(func(w http.ResponseWriter, r *http.Request) error {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, 0, "状态码非法")
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(middleware.WithRequestID(req.Context(), "req-bad-status"))

	handler.ServeHTTP(rr, req)

	assertApiErrorResponse(t, rr, http.StatusInternalServerError, "req-bad-status", apperrors.PublicCodeBadRequest, "状态码非法")
}

func TestErrorHandler_HandlesJoinedAppError(t *testing.T) {
	handler := middleware.ErrorHandler(func(w http.ResponseWriter, r *http.Request) error {
		return errors.Join(errors.New("outer"), apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "资源不存在"))
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(middleware.WithRequestID(req.Context(), "req-joined-error"))

	handler.ServeHTTP(rr, req)

	assertApiErrorResponse(t, rr, http.StatusNotFound, "req-joined-error", apperrors.PublicCodeNotFound, "资源不存在")
}

func TestErrorHandler_DoesNotDoubleWriteAfterCommittedResponse(t *testing.T) {
	handler := middleware.ErrorHandler(func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("already-written"))
		return errors.New("后续错误")
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(middleware.WithRequestID(req.Context(), "req-double-write"))

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if got := rr.Body.String(); got != "already-written" {
		t.Fatalf("expected original body to remain, got %q", got)
	}
	if got := rr.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Fatalf("expected original content type to remain, got %q", got)
	}
}

func TestErrorHandler_RealMuxChainUsesUnifiedResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/app-error", middleware.ErrorHandler(func(http.ResponseWriter, *http.Request) error {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "参数错误")
	}))
	mux.Handle("/normal-error", middleware.ErrorHandler(func(http.ResponseWriter, *http.Request) error {
		return errors.New("普通错误")
	}))

	handler := middleware.RequestIDMiddleware(mux)

	for _, tc := range []struct {
		name        string
		path        string
		wantStatus  int
		wantCode    int
		wantMessage string
	}{
		{name: "app error", path: "/app-error", wantStatus: http.StatusBadRequest, wantCode: apperrors.PublicCodeBadRequest, wantMessage: "参数错误"},
		{name: "normal error", path: "/normal-error", wantStatus: http.StatusInternalServerError, wantCode: apperrors.PublicCodeInternal, wantMessage: "系统内部错误"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)

			handler.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, rr.Code)
			}
			if got := rr.Header().Get("X-Request-ID"); got == "" {
				t.Fatal("expected X-Request-ID header to be set")
			}

			var payload response.ApiResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode response failed: %v", err)
			}

			if payload.Code != tc.wantCode {
				t.Fatalf("expected code %d, got %d", tc.wantCode, payload.Code)
			}
			if payload.Msg != tc.wantMessage {
				t.Fatalf("expected msg %q, got %q", tc.wantMessage, payload.Msg)
			}
			if payload.Data != nil {
				t.Fatalf("expected error data to be nil, got %#v", payload.Data)
			}
			if payload.ReqID == "" {
				t.Fatal("expected reqId to be set")
			}
			if payload.ReqID != rr.Header().Get("X-Request-ID") {
				t.Fatalf("expected response body reqId %q to match header %q", payload.ReqID, rr.Header().Get("X-Request-ID"))
			}
		})
	}
}

func assertApiErrorResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int, expectedRequestID string, expectedCode int, expectedMessage string) {
	t.Helper()

	if rr.Code != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, rr.Code)
	}

	var payload response.ApiResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if payload.Code != expectedCode {
		t.Fatalf("expected code %d, got %d", expectedCode, payload.Code)
	}
	if payload.Msg != expectedMessage {
		t.Fatalf("expected msg %q, got %q", expectedMessage, payload.Msg)
	}
	if payload.Data != nil {
		t.Fatalf("expected error data to be nil, got %#v", payload.Data)
	}
	if payload.ReqID != expectedRequestID {
		t.Fatalf("expected reqId %q, got %q", expectedRequestID, payload.ReqID)
	}
}
