package errors

import "testing"

func TestErrorCode_PublicCode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		code     ErrorCode
		expected int
	}{
		{name: "unknown", code: ErrorCodeUnknown, expected: PublicCodeUnknown},
		{name: "internal", code: ErrorCodeInternal, expected: PublicCodeInternal},
		{name: "bad request", code: ErrorCodeBadRequest, expected: PublicCodeBadRequest},
		{name: "not found", code: ErrorCodeNotFound, expected: PublicCodeNotFound},
		{name: "token required", code: ErrorCodeAuthTokenRequired, expected: PublicCodeAuthTokenRequired},
		{name: "token invalid", code: ErrorCodeAuthTokenInvalid, expected: PublicCodeAuthTokenInvalid},
		{name: "secret required", code: ErrorCodeAuthSecretRequired, expected: PublicCodeAuthSecretRequired},
		{name: "permission insufficient", code: ErrorCodePermissionInsufficient, expected: PublicCodePermissionInsufficient},
		{name: "project mismatch", code: ErrorCodePermissionProjectMismatch, expected: PublicCodePermissionProjectMismatch},
		{name: "unmapped code fallback", code: ErrorCode("SOME_NEW_CODE"), expected: PublicCodeUnknown},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.code.PublicCode(); got != tc.expected {
				t.Fatalf("expected public code %d, got %d", tc.expected, got)
			}
		})
	}
}

func TestResolvePublicCode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		rawCode  string
		expected int
	}{
		{name: "success keyword", rawCode: "SUCCESS", expected: SuccessCode},
		{name: "success legacy", rawCode: "00000", expected: SuccessCode},
		{name: "legacy bad request", rawCode: "BAD_REQUEST", expected: PublicCodeBadRequest},
		{name: "legacy auth invalid", rawCode: "AUTH_TOKEN_INVALID", expected: PublicCodeAuthTokenInvalid},
		{name: "abc style A", rawCode: "A0002", expected: 10002},
		{name: "abc style B", rawCode: "B1001", expected: 21001},
		{name: "abc style C", rawCode: "C0001", expected: 30001},
		{name: "numeric passthrough", rawCode: "21001", expected: 21001},
		{name: "trim and upper", rawCode: "  bad_request  ", expected: PublicCodeBadRequest},
		{name: "unknown fallback", rawCode: "NOT_A_REAL_CODE", expected: PublicCodeUnknown},
		{name: "empty fallback", rawCode: "   ", expected: PublicCodeUnknown},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := ResolvePublicCode(tc.rawCode); got != tc.expected {
				t.Fatalf("rawCode=%q expected %d, got %d", tc.rawCode, tc.expected, got)
			}
		})
	}
}
