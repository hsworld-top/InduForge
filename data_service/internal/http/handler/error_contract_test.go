package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

func TestNormalizeRepresentativeHandlerError_ConvertsBusinessBadRequestTo200(t *testing.T) {
	normalized := normalizeRepresentativeHandlerError(
		apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "查询已被禁用"),
	)

	appErr := mustExtractAppError(t, normalized)
	if appErr.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, appErr.StatusCode)
	}
}

func TestNormalizeRepresentativeHandlerError_ConvertsNotFoundTo200(t *testing.T) {
	normalized := normalizeRepresentativeHandlerError(
		apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "资源不存在"),
	)

	appErr := mustExtractAppError(t, normalized)
	if appErr.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, appErr.StatusCode)
	}
}

func TestNormalizeRepresentativeHandlerError_ConvertsPermissionTo200(t *testing.T) {
	normalized := normalizeRepresentativeHandlerError(
		apperrors.NewAppError(apperrors.ErrorCodePermissionProjectMismatch, http.StatusForbidden, "项目范围不足"),
	)

	appErr := mustExtractAppError(t, normalized)
	if appErr.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, appErr.StatusCode)
	}
}

func TestNormalizeRepresentativeHandlerError_KeepsWrappedBadRequestStatus(t *testing.T) {
	normalized := normalizeRepresentativeHandlerError(
		apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "数据库执行异常", errors.New("dial tcp timeout")),
	)

	appErr := mustExtractAppError(t, normalized)
	if appErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, appErr.StatusCode)
	}
}

func TestNormalizeRepresentativeHandlerError_KeepsAuthErrorStatus(t *testing.T) {
	normalized := normalizeRepresentativeHandlerError(
		apperrors.NewAppError(apperrors.ErrorCodeAuthTokenInvalid, http.StatusUnauthorized, "JWT 校验失败"),
	)

	appErr := mustExtractAppError(t, normalized)
	if appErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, appErr.StatusCode)
	}
}

func TestNormalizeRepresentativeHandlerError_PreservesNonAppError(t *testing.T) {
	original := errors.New("plain error")
	normalized := normalizeRepresentativeHandlerError(original)
	if !errors.Is(normalized, original) {
		t.Fatal("expected non-app error to pass through unchanged")
	}
}

func TestDecodeJSONBody_KeepsTechnicalBadRequestStatus(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{\"name\":"))
	var payload map[string]any

	err := decodeJSONBody(req, &payload)
	appErr := mustExtractAppError(t, err)
	if appErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, appErr.StatusCode)
	}
	if appErr.Code != apperrors.ErrorCodeBadRequest {
		t.Fatalf("expected code %s, got %s", apperrors.ErrorCodeBadRequest, appErr.Code)
	}
}

func TestRequireClaims_MissingClaimsKeepsUnauthorizedStatus(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	_, err := requireClaims(req)
	appErr := mustExtractAppError(t, err)
	if appErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, appErr.StatusCode)
	}
	if appErr.Code != apperrors.ErrorCodeAuthTokenRequired {
		t.Fatalf("expected code %s, got %s", apperrors.ErrorCodeAuthTokenRequired, appErr.Code)
	}
}

func mustExtractAppError(t *testing.T, err error) *apperrors.AppError {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr == nil {
		t.Fatalf("expected AppError, got %T", err)
	}
	return appErr
}
