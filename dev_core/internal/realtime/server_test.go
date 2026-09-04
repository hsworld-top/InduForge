package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/indu-forge/dev_core/internal/auth"
)

func protocolServer(t *testing.T, validate sessionValidator) (*Server, *httptest.Server) {
	t.Helper()
	s := newServer(validate, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(func() { s.Close(); ts.Close() })
	return s, ts
}

func protocolActor(context.Context, string) (auth.User, time.Time, error) {
	return auth.User{ID: "user", TenantID: "tenant", Role: "OPS_ADMIN"}, time.Now().Add(time.Hour), nil
}

func dialEngine(t *testing.T, ts *httptest.Server) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(ts.URL, "http")+"/control-socket.io/?EIO=4&transport=websocket", http.Header{"Cookie": []string{"if_access=test-session"}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if raw := readFrame(t, conn); !strings.HasPrefix(raw, "0{") {
		t.Fatalf("missing EngineIO open: %s", raw)
	}
	return conn
}

func readFrame(t *testing.T, c *websocket.Conn) string {
	t.Helper()
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, raw, err := c.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func writeFrame(t *testing.T, c *websocket.Conn, data string) {
	t.Helper()
	if err := c.WriteMessage(websocket.TextMessage, []byte(data)); err != nil {
		t.Fatal(err)
	}
}

func TestSocketProtocolAuthenticatesBeforeConnectAndFirstWatch(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	s, ts := protocolServer(t, func(ctx context.Context, token string) (auth.User, time.Time, error) {
		if token != "test-session" {
			return auth.User{}, time.Time{}, errors.New("bad cookie")
		}
		close(entered)
		select {
		case <-release:
			return protocolActor(ctx, token)
		case <-ctx.Done():
			return auth.User{}, time.Time{}, ctx.Err()
		}
	})
	c := dialEngine(t, ts)
	writeFrame(t, c, "40")
	<-entered
	type frame struct {
		data string
		err  error
	}
	frames := make(chan frame, 1)
	go func() { _, b, err := c.ReadMessage(); frames <- frame{string(b), err} }()
	select {
	case got := <-frames:
		t.Fatalf("namespace CONNECT escaped slow auth: %+v", got)
	case <-time.After(30 * time.Millisecond):
	}
	close(release)
	select {
	case got := <-frames:
		if got.err != nil || !strings.HasPrefix(got.data, "40{") {
			t.Fatalf("namespace CONNECT=%+v", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("connect timed out")
	}
	// 模拟官方客户端connect回调中立即watch，不能依赖额外延时等待服务端安装监听器。
	writeFrame(t, c, `42["ops:watch",{"subscriptionId":"first","topics":["nodes","deployments"]}]`)
	raw := readFrame(t, c)
	if !strings.HasPrefix(raw, `42["ops:ready",`) {
		t.Fatalf("first watch did not get ready: %s", raw)
	}
	s.PublishChange("tenant", []string{"deployments"}, []string{"deployment"}, true)
	if raw = readFrame(t, c); !strings.HasPrefix(raw, `42["ops:change",`) {
		t.Fatalf("missing change after ready: %s", raw)
	}
}

func TestSocketProtocolSlowAuthDisconnectDoesNotLeakHubClient(t *testing.T) {
	entered, canceled := make(chan struct{}), make(chan struct{})
	s, ts := protocolServer(t, func(ctx context.Context, _ string) (auth.User, time.Time, error) {
		close(entered)
		<-ctx.Done()
		close(canceled)
		return auth.User{}, time.Time{}, ctx.Err()
	})
	c := dialEngine(t, ts)
	writeFrame(t, c, "40")
	<-entered
	_ = c.Close()
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("transport close did not cancel authentication")
	}
	s.ops.mu.Lock()
	count := len(s.ops.clients)
	s.ops.mu.Unlock()
	if count != 0 {
		t.Fatalf("closed auth retained %d hub clients", count)
	}
}

func TestSocketProtocolRejectsUnauthorizedBeforeNamespaceConnect(t *testing.T) {
	s, ts := protocolServer(t, func(context.Context, string) (auth.User, time.Time, error) {
		return auth.User{}, time.Time{}, errors.New("secret internal reason")
	})
	c := dialEngine(t, ts)
	writeFrame(t, c, "40")
	raw := readFrame(t, c)
	if !strings.HasPrefix(raw, `44{"`) || !strings.Contains(raw, "unauthorized") || !strings.Contains(raw, opsAuthForbidden) || strings.Contains(raw, "secret") {
		t.Fatalf("bad auth response: %s", raw)
	}
	s.ops.mu.Lock()
	count := len(s.ops.clients)
	s.ops.mu.Unlock()
	if count != 0 {
		t.Fatal("unauthorized client admitted")
	}
}

func TestSocketProtocolExpiredHandshakeHasRefreshableCode(t *testing.T) {
	_, ts := protocolServer(t, func(context.Context, string) (auth.User, time.Time, error) {
		return auth.User{}, time.Time{}, auth.ErrAccessTokenExpired
	})
	c := dialEngine(t, ts)
	writeFrame(t, c, "40")
	if response := readFrame(t, c); !strings.HasPrefix(response, `44{`) || !strings.Contains(response, opsAuthExpired) {
		t.Fatalf("missing expiry code: %s", response)
	}
}

func TestSocketProtocolAuthReasonPrecedesForcedDisconnect(t *testing.T) {
	for _, code := range []string{opsAuthExpired, opsAuthForbidden} {
		t.Run(code, func(t *testing.T) {
			s, ts := protocolServer(t, protocolActor)
			c := dialEngine(t, ts)
			writeFrame(t, c, "40")
			if response := readFrame(t, c); !strings.HasPrefix(response, "40{") {
				t.Fatalf("connect=%s", response)
			}
			writeFrame(t, c, `42["ops:watch",{"subscriptionId":"auth","topics":["deployments"]}]`)
			if response := readFrame(t, c); !strings.HasPrefix(response, `42["ops:ready",`) {
				t.Fatalf("ready=%s", response)
			}
			s.ops.mu.Lock()
			for _, client := range s.ops.clients {
				if code == opsAuthExpired {
					client.expires = time.Now().Add(-time.Second)
				} else {
					s.ops.checks[client.token] = time.Now().Add(-61 * time.Second)
				}
			}
			s.ops.mu.Unlock()
			s.ops.maintain()
			s.PublishChange("tenant", []string{"deployments"}, nil, true)
			if response := readFrame(t, c); !strings.HasPrefix(response, `42["ops:auth",`) || !strings.Contains(response, code) {
				t.Fatalf("auth reason missing before disconnect: %s", response)
			}
			_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
			_, raw, err := c.ReadMessage()
			if err == nil && string(raw) != "41" && string(raw) != "1" {
				t.Fatalf("business packet followed auth close: %s", raw)
			}
			waitUntil(t, func() bool { s.ops.mu.Lock(); defer s.ops.mu.Unlock(); return len(s.ops.clients) == 0 })
		})
	}
}

func TestSocketProtocolAuthenticationConcurrencyIsBoundedAndCancelable(t *testing.T) {
	var calls atomic.Int32
	s, ts := protocolServer(t, func(ctx context.Context, _ string) (auth.User, time.Time, error) {
		calls.Add(1)
		<-ctx.Done()
		return auth.User{}, time.Time{}, ctx.Err()
	})
	// 用更小的同型信号量验证饱和，生产固定64个槽位，避免测试制造无关连接负载。
	s.authSlots = make(chan struct{}, 2)
	first, second := dialEngine(t, ts), dialEngine(t, ts)
	writeFrame(t, first, "40")
	writeFrame(t, second, "40")
	waitUntil(t, func() bool { return calls.Load() == 2 })
	third := dialEngine(t, ts)
	writeFrame(t, third, "40")
	if response := readFrame(t, third); !strings.HasPrefix(response, `44{`) || !strings.Contains(response, "busy") {
		t.Fatalf("authentication overload was not rejected: %s", response)
	}
	if calls.Load() != 2 {
		t.Fatal("saturated admission called validator")
	}
	_ = first.Close()
	waitUntil(t, func() bool { return len(s.authSlots) == 1 })
	fourth := dialEngine(t, ts)
	writeFrame(t, fourth, "40")
	waitUntil(t, func() bool { return calls.Load() == 3 })
}

func pollingRequest(t *testing.T, method, url, body string) string {
	t.Helper()
	request, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "text/plain;charset=UTF-8")
	request.AddCookie(&http.Cookie{Name: "if_access", Value: "test-session"})
	response, err := (&http.Client{Timeout: 3 * time.Second}).Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("polling HTTP %d: %s", response.StatusCode, raw)
	}
	return string(raw)
}

func TestSocketProtocolPollingBackpressureRemainsInBoundedHubQueue(t *testing.T) {
	s, ts := protocolServer(t, protocolActor)
	endpoint := ts.URL + "/control-socket.io/?EIO=4&transport=polling"
	open := pollingRequest(t, http.MethodGet, endpoint, "")
	var handshake struct {
		SID string `json:"sid"`
	}
	if err := json.Unmarshal([]byte(open[1:]), &handshake); err != nil {
		t.Fatal(err)
	}
	endpoint += "&sid=" + handshake.SID
	pollingRequest(t, http.MethodPost, endpoint, "40")
	if packet := pollingRequest(t, http.MethodGet, endpoint, ""); !strings.HasPrefix(packet, "40{") {
		t.Fatalf("namespace connect=%s", packet)
	}
	pollingRequest(t, http.MethodPost, endpoint, `42["ops:watch",{"subscriptionId":"slow","topics":["deployments"]}]`)
	waitUntil(t, func() bool {
		s.ops.mu.Lock()
		defer s.ops.mu.Unlock()
		for _, c := range s.ops.clients {
			return len(c.outbox) > 0 && !c.writable()
		}
		return false
	})
	for i := 0; i < 1000; i++ {
		s.PublishChange("tenant", []string{"deployments"}, nil, true)
	}
	waitUntil(t, func() bool {
		s.ops.mu.Lock()
		defer s.ops.mu.Unlock()
		for _, c := range s.ops.clients {
			return len(c.outbox) == 2
		}
		return false
	})
	// polling没有悬挂GET时transport不可写，ready/change应留在hub而非Emit到无界engine队列。
	s.ops.mu.Lock()
	for _, c := range s.ops.clients {
		if c.writable() || len(c.outbox) > 2 {
			t.Error("transport backpressure not respected")
		}
		c.blockedAt = time.Now().Add(-6 * time.Second)
	}
	s.ops.mu.Unlock()
	s.ops.signal()
	waitUntil(t, func() bool { s.ops.mu.Lock(); defer s.ops.mu.Unlock(); return len(s.ops.clients) == 0 })
}
