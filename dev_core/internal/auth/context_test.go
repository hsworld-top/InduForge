package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestForwardAuthorizationPrefersBearerAndFallsBackToAccessCookie(t *testing.T) {
	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set("Authorization", "Bearer header-token")
	request.AddCookie(&http.Cookie{Name: accessCookieName, Value: "cookie-token"})
	request.AddCookie(&http.Cookie{Name: refreshCookieName, Value: "refresh-token"})
	if got := ForwardAuthorization(request); got != "Bearer header-token" {
		t.Fatalf("显式 Bearer 必须优先: %q", got)
	}
	request.Header.Del("Authorization")
	if got := ForwardAuthorization(request); got != "Bearer cookie-token" {
		t.Fatalf("必须回退到访问 Cookie，且不得使用刷新 Cookie: %q", got)
	}
}

func TestForwardAuthorizationRejectsMalformedOrMissingCredentials(t *testing.T) {
	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set("Authorization", "Basic no")
	request.AddCookie(&http.Cookie{Name: refreshCookieName, Value: "refresh-token"})
	if got := ForwardAuthorization(request); got != "" {
		t.Fatalf("畸形 header 和刷新 Cookie 不得转发: %q", got)
	}
}
