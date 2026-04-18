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

	assertApiErrorResponse(t, rr, http.StatusBadRequest, "req-app-error", "BAD_REQUEST", "参数错误")
}

func TestErrorHandler_WrapsNormalErrorInUnifiedResponse(t *testing.T) {
	handler := middleware.ErrorHandler(func(w http.ResponseWriter, r *http.Request) error {
		return errors.New("普通错误")
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(middleware.WithRequestID(req.Context(), "req-normal-error"))

	handler.ServeHTTP(rr, req)

	assertApiErrorResponse(t, rr, http.StatusInternalServerError, "req-normal-error", "INTERNAL_ERROR", "系统内部错误")
}

func TestErrorHandler_FallsBackForInvalidStatusCode(t *testing.T) {
	handler := middleware.ErrorHandler(func(w http.ResponseWriter, r *http.Request) error {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, 0, "状态码非法")
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(middleware.WithRequestID(req.Context(), "req-bad-status"))

	handler.ServeHTTP(rr, req)

	assertApiErrorResponse(t, rr, http.StatusInternalServerError, "req-bad-status", "BAD_REQUEST", "状态码非法")
}

func TestErrorHandler_HandlesJoinedAppError(t *testing.T) {
	handler := middleware.ErrorHandler(func(w http.ResponseWriter, r *http.Request) error {
		return errors.Join(errors.New("outer"), apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "资源不存在"))
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(middleware.WithRequestID(req.Context(), "req-joined-error"))

	handler.ServeHTTP(rr, req)

	assertApiErrorResponse(t, rr, http.StatusNotFound, "req-joined-error", "NOT_FOUND", "资源不存在")
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

func assertApiErrorResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int, expectedRequestID, expectedErrorCode, expectedMessage string) {
	t.Helper()

	if rr.Code != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, rr.Code)
	}

	var payload response.ApiResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if payload.Success {
		t.Fatal("expected success to be false")
	}
	if payload.ErrorCode != expectedErrorCode {
		t.Fatalf("expected errorCode %q, got %q", expectedErrorCode, payload.ErrorCode)
	}
	if payload.Message != expectedMessage {
		t.Fatalf("expected message %q, got %q", expectedMessage, payload.Message)
	}
	if payload.RequestID != expectedRequestID {
		t.Fatalf("expected requestId %q, got %q", expectedRequestID, payload.RequestID)
	}
}
