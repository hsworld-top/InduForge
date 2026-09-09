package codeworkspace

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/project"
)

type gatewayEpochGuard struct {
	mu      sync.Mutex
	current string
}

func (g *gatewayEpochGuard) RequireAuthoringEpoch(_ context.Context, _, _, epoch string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if epoch != g.current {
		return errors.New("authoring epoch conflict")
	}
	return nil
}

func (g *gatewayEpochGuard) set(value string) { g.mu.Lock(); g.current = value; g.mu.Unlock() }

type recordingTransport struct {
	request *http.Request
	calls   int
}

func (t *recordingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	t.calls++
	t.request = request.Clone(request.Context())
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header: http.Header{
			"Set-Cookie": []string{"upstream=must-not-escape"},
			"Location":   []string{"https://untrusted.example/escape"},
		},
		Body:    io.NopCloser(strings.NewReader("workspace")),
		Request: request,
	}, nil
}

func newGatewayForTest(t *testing.T, transport http.RoundTripper) (*Gateway, *gatewayEpochGuard, auth.User) {
	return newGatewayForTestMode(t, transport, "https://{service}-{projectId}.workspace.induforge.test:18443", "https://center.induforge.test", false)
}

func newGatewayForTestMode(t *testing.T, transport http.RoundTripper, template, center string, allowInsecureHTTPDev bool) (*Gateway, *gatewayEpochGuard, auth.User) {
	t.Helper()
	item := project.Project{ID: testProjectID, TenantID: "tenant", CreatedBy: "owner", Visibility: "private", AuthoringEpoch: 7, WorkspacePath: t.TempDir()}
	service, err := NewService(fakeProjects{item: item}, &fakeEngine{}, Config{Image: "workspace:test", VolumeName: "workspaces", WorkspacePublicOriginTemplate: template, AllowInsecureHTTPDev: allowInsecureHTTPDev})
	if err != nil {
		t.Fatal(err)
	}
	guard := &gatewayEpochGuard{current: "epoch-7"}
	service.SetAuthoringEpochGuard(guard)
	gateway, err := NewGateway(service, GatewayConfig{
		PublicOriginTemplate: template,
		CenterPublicOrigin:   center,
		Namespace:            "induforge-system",
		AllowedOrigins:       "https://center.induforge.test",
		AllowInsecureHTTPDev: allowInsecureHTTPDev,
		Transport:            transport,
		ResolveUser: func(_ context.Context, _ string) (auth.User, error) {
			return auth.User{ID: "owner", TenantID: "tenant", Username: "developer", Role: "PROJECT_ADMIN", Status: "active", TenantStatus: "active"}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return gateway, guard, auth.User{ID: "owner", TenantID: "tenant", Username: "developer", Role: "PROJECT_ADMIN"}
}

func TestGatewayInsecureHTTPDevelopmentMode(t *testing.T) {
	transport := &recordingTransport{}
	gateway, _, actor := newGatewayForTestMode(t, transport, "http://{service}-{projectId}.workspace.induforge.test:18080", "http://center.induforge.test:18080", true)
	publicURL, err := gateway.PublicURL(actor, testProjectID, "epoch-7", "code")
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ := url.Parse(publicURL)
	if parsed.Scheme != "http" || parsed.Host != "code-"+testProjectID+".workspace.induforge.test:18080" {
		t.Fatalf("HTTP 开发 URL 错误: %s", publicURL)
	}

	wrongScheme := httptest.NewRequest(http.MethodGet, publicURL, nil)
	wrongScheme.Host = parsed.Host
	wrongScheme.Header.Set("X-Forwarded-Proto", "https")
	wrongScheme.Header.Set("Origin", "http://center.induforge.test:18080")
	wrongResult := httptest.NewRecorder()
	gateway.ServeHTTP(wrongResult, wrongScheme)
	if wrongResult.Code != http.StatusBadRequest {
		t.Fatalf("HTTP 模式接受了 HTTPS 协议头: %d", wrongResult.Code)
	}
	maliciousOrigin := httptest.NewRequest(http.MethodOptions, publicURL, nil)
	maliciousOrigin.Host = parsed.Host
	maliciousOrigin.Header.Set("X-Forwarded-Proto", "http")
	maliciousOrigin.Header.Set("Origin", "http://attacker.example")
	maliciousOriginResult := httptest.NewRecorder()
	gateway.ServeHTTP(maliciousOriginResult, maliciousOrigin)
	if maliciousOriginResult.Code != http.StatusForbidden {
		t.Fatalf("HTTP 开发模式接受了恶意显式 Origin: %d", maliciousOriginResult.Code)
	}
	// 协议错配必须在票据读取前返回，随后正确协议仍可完成交换。
	exchange := httptest.NewRequest(http.MethodGet, publicURL, nil)
	exchange.Host = parsed.Host
	exchange.Header.Set("X-Forwarded-Proto", "http")
	goodResult := httptest.NewRecorder()
	gateway.ServeHTTP(goodResult, exchange)
	if goodResult.Code != http.StatusSeeOther {
		t.Fatalf("协议错配错误消费了票据: %d", goodResult.Code)
	}
	cookies := goodResult.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != workspaceInsecureCookiePrefix+"code" || cookies[0].Secure || !cookies[0].HttpOnly || cookies[0].Domain != "" {
		t.Fatalf("HTTP 开发 Cookie 属性错误: %#v", cookies)
	}

	proxyRequest := httptest.NewRequest(http.MethodGet, "http://"+parsed.Host+"/", nil)
	proxyRequest.Host = parsed.Host
	proxyRequest.Header.Set("X-Forwarded-Proto", "http")
	proxyRequest.AddCookie(cookies[0])
	proxyResult := httptest.NewRecorder()
	gateway.ServeHTTP(proxyResult, proxyRequest)
	if proxyResult.Code != http.StatusOK || transport.calls != 1 {
		t.Fatalf("HTTP 开发会话未代理: code=%d calls=%d", proxyResult.Code, transport.calls)
	}
}

func TestGatewayProductionRejectsHTTPWithoutConsumingTicket(t *testing.T) {
	gateway, _, actor := newGatewayForTest(t, &recordingTransport{})
	publicURL, err := gateway.PublicURL(actor, testProjectID, "epoch-7", "code")
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ := url.Parse(publicURL)
	request := httptest.NewRequest(http.MethodGet, "http://"+parsed.Host+parsed.RequestURI(), nil)
	request.Host = parsed.Host
	request.Header.Set("X-Forwarded-Proto", "http")
	result := httptest.NewRecorder()
	gateway.ServeHTTP(result, request)
	if result.Code != http.StatusBadRequest {
		t.Fatalf("生产模式接受 HTTP: %d", result.Code)
	}

	request = httptest.NewRequest(http.MethodGet, publicURL, nil)
	request.Host = parsed.Host
	result = httptest.NewRecorder()
	gateway.ServeHTTP(result, request)
	if result.Code != http.StatusSeeOther {
		t.Fatalf("协议错配消耗了票据: %d", result.Code)
	}
}

func TestGatewayExchangesOneTimeTicketAndStripsCredentials(t *testing.T) {
	transport := &recordingTransport{}
	gateway, guard, actor := newGatewayForTest(t, transport)
	publicURL, err := gateway.PublicURL(actor, testProjectID, "epoch-7", "code")
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ := url.Parse(publicURL)
	ticket := parsed.Query().Get(workspaceTicketParameter)
	if ticket == "" || parsed.Host != "code-"+testProjectID+".workspace.induforge.test:18443" {
		t.Fatalf("公开工作区 URL 不符合独立 Origin 契约: %s", publicURL)
	}

	exchange := httptest.NewRequest(http.MethodGet, publicURL, nil)
	exchange.Host = parsed.Host
	exchangeResult := httptest.NewRecorder()
	gateway.ServeHTTP(exchangeResult, exchange)
	if exchangeResult.Code != http.StatusSeeOther || strings.Contains(exchangeResult.Header().Get("Location"), workspaceTicketParameter) {
		t.Fatalf("票据交换必须以 303 移除查询参数: code=%d location=%s", exchangeResult.Code, exchangeResult.Header().Get("Location"))
	}
	cookies := exchangeResult.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != workspaceSessionCookie || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].Domain != "" {
		t.Fatalf("工作区会话 Cookie 属性不安全: %#v", cookies)
	}

	replay := httptest.NewRequest(http.MethodGet, publicURL, nil)
	replay.Host = parsed.Host
	replayResult := httptest.NewRecorder()
	gateway.ServeHTTP(replayResult, replay)
	if replayResult.Code != http.StatusUnauthorized {
		t.Fatalf("一次性票据被重复使用: %d", replayResult.Code)
	}

	request := httptest.NewRequest(http.MethodGet, "https://"+parsed.Host+"/folder/file?x=1", nil)
	request.Host = parsed.Host
	request.AddCookie(cookies[0])
	request.Header.Set("Authorization", "Bearer must-not-reach-workspace")
	request.Header.Set("X-Forwarded-Host", "attacker.example")
	request.Header.Set("X-InduForge-Internal-Token", "secret")
	request.Header.Set("X-InduForge-User-Id", "spoofed-user")
	request.Header.Set("X-InduForge-Tenant-Id", "spoofed-tenant")
	result := httptest.NewRecorder()
	gateway.ServeHTTP(result, request)
	if result.Code != http.StatusOK || result.Body.String() != "workspace" || transport.calls != 1 {
		t.Fatalf("工作区代理结果错误: code=%d body=%q calls=%d", result.Code, result.Body.String(), transport.calls)
	}
	if transport.request.URL.Host != "induforge-code-"+testProjectID+".induforge-system.svc.cluster.local:3000" || transport.request.URL.Path != "/folder/file" {
		t.Fatalf("工作区上游不是固定白名单目标: %s", transport.request.URL)
	}
	if transport.request.Host != parsed.Host {
		t.Fatalf("工作区上游请求必须保留已授权的公开 Host: got=%s want=%s", transport.request.Host, parsed.Host)
	}
	for _, header := range []string{"Authorization", "Cookie", "X-Forwarded-Host", "X-InduForge-Internal-Token"} {
		if transport.request.Header.Get(header) != "" {
			t.Fatalf("敏感请求头泄露到工作区: %s", header)
		}
	}
	if transport.request.Header.Get("X-InduForge-User-Id") != actor.ID || transport.request.Header.Get("X-InduForge-Tenant-Id") != actor.TenantID || transport.request.Header.Get("X-InduForge-Project-Id") != testProjectID || transport.request.Header.Get("X-InduForge-Role") != actor.Role {
		t.Fatal("上游用户上下文必须由网关会话覆盖，不得信任浏览器请求头")
	}
	if result.Header().Get("Set-Cookie") != "" || result.Header().Get("Location") != "" {
		t.Fatalf("上游凭据或绝对重定向泄露: %#v", result.Header())
	}

	guard.set("epoch-8")
	time.Sleep(1100 * time.Millisecond)
	stale := httptest.NewRequest(http.MethodGet, "https://"+parsed.Host+"/", nil)
	stale.Host = parsed.Host
	stale.AddCookie(cookies[0])
	staleResult := httptest.NewRecorder()
	gateway.ServeHTTP(staleResult, stale)
	if staleResult.Code != http.StatusUnauthorized || transport.calls != 1 {
		t.Fatalf("旧 epoch 会话仍可访问工作区: code=%d calls=%d", staleResult.Code, transport.calls)
	}
}

func TestGatewayRejectsUntrustedCORSOrigin(t *testing.T) {
	gateway, _, actor := newGatewayForTest(t, &recordingTransport{})
	publicURL, _ := gateway.PublicURL(actor, testProjectID, "epoch-7", "preview-control")
	parsed, _ := url.Parse(publicURL)
	request := httptest.NewRequest(http.MethodOptions, publicURL, nil)
	request.Host = parsed.Host
	request.Header.Set("Origin", "https://attacker.example")
	result := httptest.NewRecorder()
	gateway.ServeHTTP(result, request)
	if result.Code != http.StatusForbidden || result.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("不可信 Origin 未被拒绝: code=%d headers=%#v", result.Code, result.Header())
	}
}

func TestGatewayReusesPendingTicketUnderConcurrentStatusPolling(t *testing.T) {
	gateway, _, actor := newGatewayForTest(t, &recordingTransport{})
	const callers = 200
	urls := make(chan string, callers)
	errs := make(chan error, callers)
	var group sync.WaitGroup
	for range callers {
		group.Add(1)
		go func() {
			defer group.Done()
			value, err := gateway.PublicURL(actor, testProjectID, "epoch-7", "code")
			urls <- value
			errs <- err
		}()
	}
	group.Wait()
	close(urls)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var expected string
	for value := range urls {
		if expected == "" {
			expected = value
		}
		if value != expected {
			t.Fatalf("同 scope 并发轮询生成了不同票据: first=%s current=%s", expected, value)
		}
	}
	gateway.mu.Lock()
	if len(gateway.tickets) != 1 || len(gateway.ticketByScope) != 1 {
		t.Fatalf("并发轮询制造了多余票据: tickets=%d scopes=%d", len(gateway.tickets), len(gateway.ticketByScope))
	}
	gateway.mu.Unlock()

	parsed, _ := url.Parse(expected)
	exchange := httptest.NewRequest(http.MethodGet, expected, nil)
	exchange.Host = parsed.Host
	result := httptest.NewRecorder()
	gateway.ServeHTTP(result, exchange)
	if result.Code != http.StatusSeeOther {
		t.Fatalf("复用票据无法正常消费: %d", result.Code)
	}
	fresh, err := gateway.PublicURL(actor, testProjectID, "epoch-7", "code")
	if err != nil || fresh == expected {
		t.Fatalf("已消费票据未被替换: fresh=%s err=%v", fresh, err)
	}
}

func TestGatewayTicketEvictionUsesExpiredThenLRU(t *testing.T) {
	gateway, _, actor := newGatewayForTest(t, &recordingTransport{})
	now := time.Now().UTC()
	gateway.mu.Lock()
	for index := 0; index < maxPendingTickets; index++ {
		id := fmt.Sprintf("ticket-%04d", index)
		grant := gatewayGrant{Actor: actor, ProjectID: fmt.Sprintf("project-%04d", index), Epoch: "epoch-7", Service: "code", ExpiresAt: now.Add(time.Minute), LastUsedAt: now.Add(time.Duration(index) * time.Millisecond)}
		gateway.tickets[id] = grant
		gateway.ticketByScope[ticketScope(actor, grant.ProjectID, grant.Epoch, grant.Service)] = id
	}
	// 周期清理先移除过期项，不应挤掉仍有效的最近票据。
	expired := gateway.tickets["ticket-2048"]
	expired.ExpiresAt = now.Add(-time.Second)
	gateway.tickets["ticket-2048"] = expired
	gateway.lastCleanup = time.Time{}
	gateway.mu.Unlock()

	if _, err := gateway.PublicURL(actor, "new-project", "epoch-7", "preview"); err != nil {
		t.Fatal(err)
	}
	gateway.mu.Lock()
	defer gateway.mu.Unlock()
	if _, ok := gateway.tickets["ticket-2048"]; ok {
		t.Fatal("过期票据未被优先清理")
	}
	if _, ok := gateway.tickets["ticket-0000"]; !ok {
		t.Fatal("存在过期票据时错误淘汰了有效 LRU 票据")
	}
	if len(gateway.tickets) != maxPendingTickets {
		t.Fatalf("票据上限错误: %d", len(gateway.tickets))
	}
}

func TestGatewayTicketCapacityEvictsLeastRecentlyUsed(t *testing.T) {
	gateway, _, actor := newGatewayForTest(t, &recordingTransport{})
	now := time.Now().UTC()
	gateway.mu.Lock()
	for index := 0; index < maxPendingTickets; index++ {
		id := fmt.Sprintf("ticket-%04d", index)
		grant := gatewayGrant{Actor: actor, ProjectID: fmt.Sprintf("project-%04d", index), Epoch: "epoch-7", Service: "code", ExpiresAt: now.Add(time.Minute), LastUsedAt: now.Add(time.Duration(index) * time.Millisecond)}
		gateway.tickets[id] = grant
		gateway.ticketByScope[ticketScope(actor, grant.ProjectID, grant.Epoch, grant.Service)] = id
	}
	gateway.lastCleanup = now
	gateway.mu.Unlock()

	if _, err := gateway.PublicURL(actor, "new-project", "epoch-7", "preview"); err != nil {
		t.Fatal(err)
	}
	gateway.mu.Lock()
	defer gateway.mu.Unlock()
	if _, ok := gateway.tickets["ticket-0000"]; ok {
		t.Fatal("容量淘汰未选择最久未使用票据")
	}
	if _, ok := gateway.tickets["ticket-0001"]; !ok {
		t.Fatal("容量淘汰删除了非 LRU 票据")
	}
}

func TestGatewaySessionCleanupAndPerActorLimitUseLRU(t *testing.T) {
	gateway, _, actor := newGatewayForTest(t, &recordingTransport{})
	now := time.Now().UTC()
	gateway.mu.Lock()
	gateway.sessions["expired"] = gatewayGrant{Actor: actor, ExpiresAt: now.Add(-time.Second), LastUsedAt: now.Add(-time.Minute)}
	for index := 0; index < maxSessionsPerActor; index++ {
		gateway.sessions[fmt.Sprintf("session-%02d", index)] = gatewayGrant{Actor: actor, ExpiresAt: now.Add(time.Minute), LastUsedAt: now.Add(time.Duration(index) * time.Millisecond)}
	}
	gateway.cleanupPeriodicLocked(now)
	if _, ok := gateway.sessions["expired"]; ok {
		gateway.mu.Unlock()
		t.Fatal("周期清理未移除过期会话")
	}
	if gateway.actorSessionCountLocked(actor) != maxSessionsPerActor {
		gateway.mu.Unlock()
		t.Fatal("测试会话数量错误")
	}
	gateway.evictOldestSessionLocked(func(item gatewayGrant) bool { return sameActor(item.Actor, actor) })
	if _, ok := gateway.sessions["session-00"]; ok {
		gateway.mu.Unlock()
		t.Fatal("单用户会话限流未按 LRU 淘汰")
	}
	gateway.mu.Unlock()
}

func TestGatewayWrapNeverFallsThroughWorkspaceShapedHost(t *testing.T) {
	gateway, _, _ := newGatewayForTest(t, &recordingTransport{})
	called := false
	handler := gateway.Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	request := httptest.NewRequest(http.MethodGet, "https://code-"+testProjectID+".workspace.induforge.test:18443/api/v1/auth/me", nil)
	request.Host = "code-" + testProjectID + ".workspace.induforge.test:18443"
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, request)
	if called || result.Code != http.StatusUnauthorized {
		t.Fatalf("工作区 Host 意外回退到中心 Handler: called=%v code=%d", called, result.Code)
	}
}

func TestGatewayRejectsSameOriginOrPathTemplate(t *testing.T) {
	service, _ := NewService(fakeProjects{}, &fakeEngine{}, Config{Image: "workspace:test", VolumeName: "workspaces"})
	for _, template := range []string{
		"https://center.test/{service}/{projectId}",
		"http://{service}-{projectId}.workspace.test",
		"https://workspace.test",
		"https://{service}-{projectId}.different.test",
	} {
		if _, err := NewGateway(service, GatewayConfig{PublicOriginTemplate: template, CenterPublicOrigin: "https://center.induforge.test", Namespace: "default", ResolveUser: func(context.Context, string) (auth.User, error) { return auth.User{}, nil }}); err == nil {
			t.Fatalf("不安全模板被接受: %s", template)
		}
	}
}

func TestGatewayInsecureDevelopmentRejectsHTTPSMixing(t *testing.T) {
	service, err := NewService(fakeProjects{}, &fakeEngine{}, Config{Image: "workspace:test", VolumeName: "workspaces"})
	if err != nil {
		t.Fatal(err)
	}
	resolver := func(context.Context, string) (auth.User, error) { return auth.User{}, nil }
	for _, config := range []GatewayConfig{
		{PublicOriginTemplate: "https://{service}-{projectId}.workspace.induforge.test", CenterPublicOrigin: "https://center.induforge.test", Namespace: "default", AllowInsecureHTTPDev: true, ResolveUser: resolver},
		{PublicOriginTemplate: "http://{service}-{projectId}.workspace.induforge.test", CenterPublicOrigin: "https://center.induforge.test", Namespace: "default", AllowInsecureHTTPDev: true, ResolveUser: resolver},
	} {
		if _, err := NewGateway(service, config); err == nil {
			t.Fatalf("HTTP 开发模式接受了 HTTPS 混配: %#v", config)
		}
	}
}

func TestGatewaySharesValidationWithoutMixingServiceScope(t *testing.T) {
	gateway, _, actor := newGatewayForTest(t, &recordingTransport{})
	calls := 0
	gateway.resolveUser = func(context.Context, string) (auth.User, error) {
		calls++
		return actor, nil
	}
	expires := time.Now().Add(time.Minute)
	code, err := gateway.authorizeGrant(context.Background(), gatewayGrant{Actor: actor, ProjectID: testProjectID, Epoch: "epoch-7", Service: "code", Host: "code.example", ExpiresAt: expires})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := gateway.authorizeGrant(context.Background(), gatewayGrant{Actor: actor, ProjectID: testProjectID, Epoch: "epoch-7", Service: "preview", Host: "preview.example", ExpiresAt: expires})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || code.Service != "code" || preview.Service != "preview" || preview.Host != "preview.example" {
		t.Fatalf("共享校验缓存污染了服务 scope: calls=%d code=%#v preview=%#v", calls, code, preview)
	}
}

type hijackRecorder struct{ client, server net.Conn }

func newHijackRecorder() *hijackRecorder {
	client, server := net.Pipe()
	return &hijackRecorder{client: client, server: server}
}

func (*hijackRecorder) Header() http.Header       { return make(http.Header) }
func (*hijackRecorder) Write([]byte) (int, error) { return 0, nil }
func (*hijackRecorder) WriteHeader(int)           {}
func (w *hijackRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.server, bufio.NewReadWriter(bufio.NewReader(w.server), bufio.NewWriter(w.server)), nil
}

func TestEpochResponseWriterClosesHijackedConnectionWhenEpochChanges(t *testing.T) {
	recorder := newHijackRecorder()
	defer recorder.client.Close()
	var valid atomic.Bool
	valid.Store(true)
	writer := &epochResponseWriter{ResponseWriter: recorder, check: func() bool { return valid.Load() }}
	if _, _, err := writer.Hijack(); err != nil {
		t.Fatal(err)
	}
	valid.Store(false)
	_ = recorder.client.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, err := recorder.client.Read(make([]byte, 1)); err == nil {
		t.Fatal("epoch 变化后旧 WebSocket 连接未关闭")
	}
}

func TestEpochResponseWriterClosesHijackedConnectionWhenSessionExpires(t *testing.T) {
	gateway, _, actor := newGatewayForTest(t, &recordingTransport{})
	var unix atomic.Int64
	unix.Store(time.Now().UnixNano())
	gateway.now = func() time.Time { return time.Unix(0, unix.Load()) }
	grant := gatewayGrant{Actor: actor, ProjectID: testProjectID, Epoch: "epoch-7", Service: "code", Host: "code.example", ExpiresAt: gateway.now().Add(500 * time.Millisecond)}
	recorder := newHijackRecorder()
	defer recorder.client.Close()
	writer := &epochResponseWriter{ResponseWriter: recorder, check: func() bool {
		_, err := gateway.authorizeGrant(context.Background(), grant)
		return err == nil
	}}
	if _, _, err := writer.Hijack(); err != nil {
		t.Fatal(err)
	}
	unix.Store(grant.ExpiresAt.Add(time.Second).UnixNano())
	_ = recorder.client.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, err := recorder.client.Read(make([]byte, 1)); err == nil {
		t.Fatal("短会话到期后旧 WebSocket 连接未关闭")
	}
}
