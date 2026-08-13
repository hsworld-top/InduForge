package socketio

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCookiesFromRequest(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/socket.io", nil)
	request.AddCookie(&http.Cookie{Name: "if_access", Value: "access-token"})

	cookies := CookiesFromRequest(request)
	connection := &Conn{cookies: cookies}

	if got := connection.Cookies("if_access"); got != "access-token" {
		t.Fatalf("Cookie 未保留: got %q", got)
	}
	if got := connection.Cookies("missing", "fallback"); got != "fallback" {
		t.Fatalf("默认值错误: got %q", got)
	}
}
