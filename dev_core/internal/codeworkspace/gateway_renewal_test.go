package codeworkspace

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/indu-forge/dev_core/internal/auth"
)

func renewalRequest(t *testing.T, g *Gateway, actor auth.User, cookie *http.Cookie) (*httptest.ResponseRecorder, string) {
	t.Helper()
	raw, err := g.PublicURL(actor, testProjectID, "epoch-7", "code")
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(raw)
	u.Path = workspaceSessionPath
	req := httptest.NewRequest(http.MethodPost, u.String(), nil)
	req.Header.Set("Origin", "https://center.induforge.test")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	res := httptest.NewRecorder()
	g.ServeHTTP(res, req)
	return res, u.String()
}

func TestGatewaySilentRenewalRetainsSessionAndRequiresFreshTicket(t *testing.T) {
	transport := &recordingTransport{}
	g, _, actor := newGatewayForTest(t, transport)
	now := time.Now()
	g.now = func() time.Time { return now }
	first, _ := renewalRequest(t, g, actor, nil)
	if first.Code != 204 {
		t.Fatal(first.Code, first.Body.String())
	}
	cookie := first.Result().Cookies()[0]
	original := g.sessions[cookie.Value]
	now = now.Add(10 * time.Minute)
	renewed, usedURL := renewalRequest(t, g, actor, cookie)
	if renewed.Code != 204 || renewed.Result().Cookies()[0].Value != cookie.Value {
		t.Fatal("续期必须保留会话ID")
	}
	if !g.sessions[cookie.Value].ExpiresAt.After(original.ExpiresAt) {
		t.Fatal("会话未延长")
	}
	if transport.calls != 0 || renewed.Header().Get("Location") != "" || renewed.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("续期不得代理、重定向或缓存")
	}
	if renewed.Header().Get("Access-Control-Allow-Origin") != "https://center.induforge.test" || renewed.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("续期CORS缺失")
	}
	now = original.ExpiresAt.Add(time.Second)
	if !g.sessionAuthorized(cookie.Value, original) {
		t.Fatal("长连接仍使用旧有效期")
	}
	resource := httptest.NewRequest(http.MethodGet, "https://"+original.Host+"/resource", nil)
	resource.AddCookie(cookie)
	resourceResult := httptest.NewRecorder()
	g.ServeHTTP(resourceResult, resource)
	if resourceResult.Code != http.StatusOK {
		t.Fatal("HTTP请求未使用续期后的会话", resourceResult.Code)
	}
	for _, raw := range []string{usedURL, "https://" + original.Host + workspaceSessionPath} {
		req := httptest.NewRequest(http.MethodPost, raw, nil)
		req.AddCookie(cookie)
		res := httptest.NewRecorder()
		g.ServeHTTP(res, req)
		if res.Code != 401 {
			t.Fatal("旧Cookie不能替代新票据", res.Code)
		}
	}
}

func TestGatewayRenewalDoesNotBorrowDifferentScopeOrExpiredSession(t *testing.T) {
	for _, field := range []string{"actor", "tenant", "project", "epoch", "service", "expired"} {
		t.Run(field, func(t *testing.T) {
			g, _, actor := newGatewayForTest(t, &recordingTransport{})
			first, _ := renewalRequest(t, g, actor, nil)
			cookie := first.Result().Cookies()[0]
			old := g.sessions[cookie.Value]
			switch field {
			case "actor":
				old.Actor.ID = "someone-else"
			case "tenant":
				old.Actor.TenantID = "another-tenant"
			case "project":
				old.ProjectID = "another-project"
			case "epoch":
				old.Epoch = "epoch-6"
			case "service":
				old.Service = "ai"
			case "expired":
				old.ExpiresAt = g.now().Add(-time.Second)
			}
			g.sessions[cookie.Value] = old
			next, _ := renewalRequest(t, g, actor, cookie)
			if next.Code != 204 || next.Result().Cookies()[0].Value == cookie.Value {
				t.Fatal("不同scope或过期会话不得复用")
			}
			if existing, ok := g.sessions[cookie.Value]; ok && !existing.ExpiresAt.Equal(old.ExpiresAt) {
				t.Fatal("不得延长其他scope会话")
			}
		})
	}
}

func TestGatewayRenewalRejectsRevocationAndEpochChange(t *testing.T) {
	for _, revokeUser := range []bool{true, false} {
		g, guard, actor := newGatewayForTest(t, &recordingTransport{})
		first, _ := renewalRequest(t, g, actor, nil)
		cookie := first.Result().Cookies()[0]
		original := g.sessions[cookie.Value]
		if revokeUser {
			g.resolveUser = func(context.Context, string) (auth.User, error) { return auth.User{}, errors.New("revoked") }
		} else {
			guard.set("epoch-8")
		}
		res, _ := renewalRequest(t, g, actor, cookie)
		if res.Code != 401 {
			t.Fatal("失权续期应拒绝", res.Code)
		}
		if !g.sessions[cookie.Value].ExpiresAt.Equal(original.ExpiresAt) {
			t.Fatal("失权不得延长会话")
		}
		if g.sessionAuthorized(cookie.Value, original) {
			t.Fatal("失权长连接应拒绝")
		}
	}
}

func TestGatewayWebSocketSurvivesRenewalThenClosesAtNewExpiry(t *testing.T) {
	g, _, actor := newGatewayForTest(t, &recordingTransport{})
	var clock atomic.Int64
	clock.Store(time.Now().UnixNano())
	g.now = func() time.Time { return time.Unix(0, clock.Load()) }
	first, _ := renewalRequest(t, g, actor, nil)
	cookie := first.Result().Cookies()[0]
	original := g.sessions[cookie.Value]
	recorder := newHijackRecorder()
	defer recorder.client.Close()
	writer := &epochResponseWriter{ResponseWriter: recorder, check: func() bool { return g.sessionAuthorized(cookie.Value, original) }}
	conn, _, err := writer.Hijack()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	clock.Store(original.ExpiresAt.Add(-time.Minute).UnixNano())
	next, _ := renewalRequest(t, g, actor, cookie)
	if next.Code != 204 {
		t.Fatal(next.Code)
	}
	clock.Store(original.ExpiresAt.Add(time.Second).UnixNano())
	_ = recorder.client.SetReadDeadline(time.Now().Add(1200 * time.Millisecond))
	_, err = recorder.client.Read(make([]byte, 1))
	if ne, ok := err.(interface{ Timeout() bool }); !ok || !ne.Timeout() {
		t.Fatal("旧有效期不应关闭已续期连接", err)
	}
	clock.Store(g.now().Add(defaultSessionTTL).UnixNano())
	_ = recorder.client.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, err = recorder.client.Read(make([]byte, 1))
	if err == nil {
		t.Fatal("新有效期到期应关闭连接")
	}
	if ne, ok := err.(interface{ Timeout() bool }); ok && ne.Timeout() {
		t.Fatal("连接未及时关闭")
	}
}

// 本地模拟长响应，不启动服务；请求上下文取消必须结束 SSE 的上游读取。
type renewalStreamTransport struct{ started chan struct{} }

func (s renewalStreamTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	close(s.started)
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: renewalStreamBody{ctx: r.Context()}, Request: r}, nil
}

type renewalStreamBody struct{ ctx context.Context }

func (b renewalStreamBody) Read([]byte) (int, error) { <-b.ctx.Done(); return 0, io.EOF }
func (b renewalStreamBody) Close() error             { return nil }

func TestGatewaySSEUsesRenewedExpiry(t *testing.T) {
	transport := renewalStreamTransport{started: make(chan struct{})}
	g, _, actor := newGatewayForTest(t, transport)
	var clock atomic.Int64
	clock.Store(time.Now().UnixNano())
	g.now = func() time.Time { return time.Unix(0, clock.Load()) }
	first, _ := renewalRequest(t, g, actor, nil)
	cookie := first.Result().Cookies()[0]
	original := g.sessions[cookie.Value]
	req := httptest.NewRequest(http.MethodGet, "https://"+original.Host+"/events", nil)
	req.AddCookie(cookie)
	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()
	req = req.WithContext(ctx)
	done := make(chan struct{})
	go func() { defer close(done); g.ServeHTTP(httptest.NewRecorder(), req) }()
	<-transport.started
	clock.Store(original.ExpiresAt.Add(-time.Minute).UnixNano())
	renewed, _ := renewalRequest(t, g, actor, cookie)
	if renewed.Code != 204 {
		t.Fatal(renewed.Code)
	}
	clock.Store(original.ExpiresAt.Add(time.Second).UnixNano())
	select {
	case <-done:
		t.Fatal("SSE按旧有效期关闭")
	case <-time.After(1200 * time.Millisecond):
	}
	clock.Store(g.now().Add(defaultSessionTTL).UnixNano())
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SSE新会话过期后未关闭")
	}
}
