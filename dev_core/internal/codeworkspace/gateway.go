package codeworkspace

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/indu-forge/dev_core/internal/auth"
	"golang.org/x/net/publicsuffix"
)

const (
	workspaceSessionPath          = "/__if_workspace_session"
	workspaceTicketParameter      = "__if_workspace_ticket"
	workspaceSessionCookie        = "__Host-if_workspace_session"
	workspaceInsecureCookiePrefix = "if_workspace_session_"
	defaultTicketTTL              = 45 * time.Second
	defaultSessionTTL             = 15 * time.Minute
	maxPendingTickets             = 4096
	maxActiveSessions             = 4096
	maxSessionsPerActor           = 64
	cleanupInterval               = 15 * time.Second
)

var (
	ErrGatewayDisabled       = errors.New("代码工作区隔离网关未启用")
	ErrInvalidOriginTemplate = errors.New("工作区公开 Origin 模板无效")
)

var workspaceServicePorts = map[string]int{
	"code": 3000, "ai": 30141, "preview": 5173, "preview-control": 5174, "mcp": 30142,
}

type GatewayConfig struct {
	PublicOriginTemplate string
	CenterPublicOrigin   string
	Namespace            string
	AllowedOrigins       string
	AllowInsecureHTTPDev bool
	TicketTTL            time.Duration
	SessionTTL           time.Duration
	Now                  func() time.Time
	Transport            http.RoundTripper
	ResolveUser          func(context.Context, string) (auth.User, error)
	ResolveMCPToken      func(context.Context, string) (auth.User, string, string, time.Time, bool)
}

type gatewayGrant struct {
	Actor      auth.User
	ProjectID  string
	Epoch      string
	Service    string
	Host       string
	ExpiresAt  time.Time
	LastUsedAt time.Time
	// MCPToken 仅在 MCP 会话转发到工作区时短暂携带，供工作区数据工具回到中心做同一身份校验。
	MCPToken string
}

type grantValidation struct {
	actor     auth.User
	expiresAt time.Time
}

type grantValidationFlight struct {
	done  chan struct{}
	actor auth.User
	err   error
}

// Gateway 是工作区唯一外部入口。公开 Host 只映射到服务端生成的工程 Service，
// 请求中的路径、查询参数和 Header 均不能改变上游工程或端口。
type Gateway struct {
	service              *Service
	template             string
	scheme               string
	allowInsecureHTTPDev bool
	namespace            string
	allowedOrigin        map[string]struct{}
	centerOrigin         string
	centerHost           string
	centerHostname       string
	ticketTTL            time.Duration
	sessionTTL           time.Duration
	now                  func() time.Time
	transport            http.RoundTripper
	resolveUser          func(context.Context, string) (auth.User, error)
	resolveMCPToken      func(context.Context, string) (auth.User, string, string, time.Time, bool)
	mu                   sync.Mutex
	tickets              map[string]gatewayGrant
	ticketByScope        map[string]string
	sessions             map[string]gatewayGrant
	lastCleanup          time.Time
	hostPattern          *regexp.Regexp
	fullHostPattern      *regexp.Regexp
	serviceHostPatterns  map[string]*regexp.Regexp
	validationMu         sync.Mutex
	validations          map[string]grantValidation
	flights              map[string]*grantValidationFlight
}

func NewGateway(service *Service, config GatewayConfig) (*Gateway, error) {
	if service == nil {
		return nil, fmt.Errorf("代码工作区网关服务不能为空")
	}
	template := strings.TrimSpace(config.PublicOriginTemplate)
	if template == "" {
		return nil, ErrGatewayDisabled
	}
	if !strings.Contains(template, "{projectId}") || !strings.Contains(template, "{service}") {
		return nil, fmt.Errorf("%w: 必须包含 {projectId} 和 {service}", ErrInvalidOriginTemplate)
	}
	probe, err := url.Parse(strings.NewReplacer("{projectId}", "00000000-0000-0000-0000-000000000000", "{service}", "code").Replace(template))
	expectedScheme := "https"
	if config.AllowInsecureHTTPDev {
		expectedScheme = "http"
	}
	if err != nil || probe.Scheme != expectedScheme || probe.Hostname() == "" || probe.User != nil || probe.RawQuery != "" || probe.Fragment != "" || (probe.Path != "" && probe.Path != "/") {
		return nil, fmt.Errorf("%w: 必须为无路径的 %s Origin", ErrInvalidOriginTemplate, strings.ToUpper(expectedScheme))
	}
	center := strings.TrimSpace(config.CenterPublicOrigin)
	centerURL, parseErr := url.Parse(center)
	if center == "" || parseErr != nil || centerURL.Scheme != expectedScheme || centerURL.Host == "" || centerURL.Path != "" || centerURL.User != nil || centerURL.RawQuery != "" || centerURL.Fragment != "" {
		return nil, fmt.Errorf("启用工作区网关时中心公开 Origin 必须为 %s Origin", strings.ToUpper(expectedScheme))
	}
	if strings.EqualFold(centerURL.Host, probe.Host) {
		return nil, fmt.Errorf("%w: 工作区不得与中心同 Origin", ErrInvalidOriginTemplate)
	}
	centerSite, centerSiteErr := publicsuffix.EffectiveTLDPlusOne(centerURL.Hostname())
	workspaceSite, workspaceSiteErr := publicsuffix.EffectiveTLDPlusOne(probe.Hostname())
	if centerSiteErr != nil || workspaceSiteErr != nil || !strings.EqualFold(centerSite, workspaceSite) {
		return nil, fmt.Errorf("%w: 中心与工作区必须同站点且不同 Origin", ErrInvalidOriginTemplate)
	}
	patternURL, _ := url.Parse(strings.NewReplacer("{projectId}", "ifprojectplaceholder", "{service}", "ifserviceplaceholder").Replace(template))
	fullHostExpression := regexp.QuoteMeta(strings.ToLower(patternURL.Host))
	fullHostExpression = strings.Replace(fullHostExpression, "ifprojectplaceholder", `[0-9a-f-]{36}`, 1)
	fullHostExpression = strings.Replace(fullHostExpression, "ifserviceplaceholder", `(?:ai|code|preview|preview-control|mcp)`, 1)
	fullHostPattern, err := regexp.Compile("^" + fullHostExpression + "$")
	if err != nil {
		return nil, fmt.Errorf("%w: Host 模式无法编译", ErrInvalidOriginTemplate)
	}
	hostExpression := regexp.QuoteMeta(strings.ToLower(patternURL.Hostname()))
	hostExpression = strings.Replace(hostExpression, "ifprojectplaceholder", `[0-9a-f-]{36}`, 1)
	hostExpression = strings.Replace(hostExpression, "ifserviceplaceholder", `(?:ai|code|preview|preview-control|mcp)`, 1)
	hostPattern, err := regexp.Compile("^" + hostExpression + "$")
	if err != nil {
		return nil, fmt.Errorf("%w: Hostname 模式无法编译", ErrInvalidOriginTemplate)
	}
	serviceHostPatterns := make(map[string]*regexp.Regexp, len(workspaceServicePorts))
	for serviceName := range workspaceServicePorts {
		serviceURL, _ := url.Parse(strings.NewReplacer("{projectId}", "ifprojectplaceholder", "{service}", serviceName).Replace(template))
		expression := strings.Replace(regexp.QuoteMeta(strings.ToLower(serviceURL.Host)), "ifprojectplaceholder", `[0-9a-f-]{36}`, 1)
		serviceHostPatterns[serviceName], err = regexp.Compile("^" + expression + "$")
		if err != nil {
			return nil, fmt.Errorf("%w: 服务 Host 模式无法编译", ErrInvalidOriginTemplate)
		}
	}
	namespace := strings.TrimSpace(config.Namespace)
	if namespace == "" {
		return nil, fmt.Errorf("工作区网关命名空间不能为空")
	}
	allowed := make(map[string]struct{})
	for _, raw := range strings.Split(config.AllowedOrigins, ",") {
		value := strings.TrimRight(strings.TrimSpace(raw), "/")
		if value == "" {
			continue
		}
		parsed, parseErr := url.Parse(value)
		if parseErr != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.Path != "" {
			return nil, fmt.Errorf("工作区允许 Origin 无效: %s", value)
		}
		allowed[parsed.Scheme+"://"+parsed.Host] = struct{}{}
	}
	allowed[centerURL.Scheme+"://"+centerURL.Host] = struct{}{}
	if config.TicketTTL <= 0 {
		config.TicketTTL = defaultTicketTTL
	}
	if config.TicketTTL < 30*time.Second || config.TicketTTL > 60*time.Second {
		return nil, fmt.Errorf("工作区票据有效期必须为 30 至 60 秒")
	}
	if config.SessionTTL <= 0 {
		config.SessionTTL = defaultSessionTTL
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.Transport == nil {
		config.Transport = http.DefaultTransport
	}
	if config.ResolveUser == nil {
		return nil, fmt.Errorf("工作区网关用户状态解析器不能为空")
	}
	return &Gateway{service: service, template: template, scheme: expectedScheme, allowInsecureHTTPDev: config.AllowInsecureHTTPDev, namespace: namespace, allowedOrigin: allowed, centerOrigin: centerURL.Scheme + "://" + centerURL.Host, centerHost: strings.ToLower(centerURL.Host), centerHostname: strings.ToLower(centerURL.Hostname()), ticketTTL: config.TicketTTL, sessionTTL: config.SessionTTL, now: config.Now, transport: config.Transport, resolveUser: config.ResolveUser, resolveMCPToken: config.ResolveMCPToken, tickets: map[string]gatewayGrant{}, ticketByScope: map[string]string{}, sessions: map[string]gatewayGrant{}, hostPattern: hostPattern, fullHostPattern: fullHostPattern, serviceHostPatterns: serviceHostPatterns, validations: map[string]grantValidation{}, flights: map[string]*grantValidationFlight{}}, nil
}

func (g *Gateway) PublicURL(actor auth.User, projectID, epoch, serviceName string) (string, error) {
	if _, ok := workspaceServicePorts[serviceName]; !ok {
		return "", fmt.Errorf("未知工作区服务: %s", serviceName)
	}
	origin := strings.NewReplacer("{projectId}", strings.ToLower(projectID), "{service}", serviceName).Replace(g.template)
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Hostname() == "" {
		return "", ErrInvalidOriginTemplate
	}
	now := g.now().UTC()
	scope := ticketScope(actor, projectID, epoch, serviceName)
	g.mu.Lock()
	g.cleanupPeriodicLocked(now)
	if existingID, ok := g.ticketByScope[scope]; ok {
		if existing, exists := g.tickets[existingID]; exists && now.Before(existing.ExpiresAt) {
			existing.LastUsedAt = now
			g.tickets[existingID] = existing
			g.mu.Unlock()
			query := parsed.Query()
			query.Set(workspaceTicketParameter, existingID)
			parsed.RawQuery = query.Encode()
			return parsed.String(), nil
		}
		delete(g.ticketByScope, scope)
	}
	g.mu.Unlock()

	ticket, err := secureValue()
	if err != nil {
		return "", err
	}
	grant := gatewayGrant{Actor: actor, ProjectID: projectID, Epoch: epoch, Service: serviceName, Host: strings.ToLower(parsed.Host), ExpiresAt: now.Add(g.ticketTTL), LastUsedAt: now}
	g.mu.Lock()
	// 随机数生成期间可能已有并发请求签发了同 scope 票据，必须再次合并。
	if existingID, ok := g.ticketByScope[scope]; ok {
		if existing, exists := g.tickets[existingID]; exists && now.Before(existing.ExpiresAt) {
			existing.LastUsedAt = now
			g.tickets[existingID] = existing
			g.mu.Unlock()
			query := parsed.Query()
			query.Set(workspaceTicketParameter, existingID)
			parsed.RawQuery = query.Encode()
			return parsed.String(), nil
		}
	}
	if len(g.tickets) >= maxPendingTickets {
		// 容量压力下不等待清理周期，保证过期项始终先于有效 LRU 被移除。
		g.cleanupLocked(now)
	}
	for len(g.tickets) >= maxPendingTickets {
		g.evictOldestTicketLocked()
	}
	g.tickets[ticket] = grant
	g.ticketByScope[scope] = ticket
	g.mu.Unlock()
	query := parsed.Query()
	query.Set(workspaceTicketParameter, ticket)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func (g *Gateway) publicOrigin(projectID, serviceName string) (string, error) {
	if _, ok := workspaceServicePorts[serviceName]; !ok {
		return "", fmt.Errorf("未知工作区服务: %s", serviceName)
	}
	origin := strings.NewReplacer("{projectId}", strings.ToLower(projectID), "{service}", serviceName).Replace(g.template)
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Hostname() == "" {
		return "", ErrInvalidOriginTemplate
	}
	return strings.ToLower(parsed.Host), nil
}

// PublicURLWithoutTicket 返回供无浏览器客户端使用的中心 MCP 地址。
func (g *Gateway) PublicURLWithoutTicket(projectID, serviceName string) (string, error) {
	if serviceName != "mcp" {
		return "", fmt.Errorf("无票据公开地址仅支持 MCP 服务")
	}
	if _, err := uuid.Parse(projectID); err != nil {
		return "", fmt.Errorf("工程 ID 无效: %w", err)
	}
	return g.centerOrigin + "/workspaces/" + url.PathEscape(strings.ToLower(projectID)) + "/mcp", nil
}

// Wrap 根据 Host 分流工作区流量，中心 Host 继续进入原控制面 Handler。
func (g *Gateway) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := workspaceRequestHostname(r.Host)
		if !g.hostPattern.MatchString(host) && !(strings.EqualFold(host, g.centerHostname) && g.isCentralMCPPath(r.URL.Path)) {
			next.ServeHTTP(w, r)
			return
		}
		g.ServeHTTP(w, r)
	})
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := strings.ToLower(r.Host)
	centralMCPProjectID, centralMCP := g.centralMCPProjectID(host, r.URL.Path)
	if (!g.fullHostPattern.MatchString(host) && !centralMCP) || !g.requestSchemeMatches(r) {
		http.Error(w, "工作区访问协议或 Host 无效", http.StatusBadRequest)
		return
	}
	if !g.applyCORS(w, r) {
		return
	}
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if g.isMCPHost(host) && g.serveMCPBearer(w, r, host, "") {
		return
	}
	if centralMCP && g.serveMCPBearer(w, r, host, centralMCPProjectID) {
		return
	}
	if r.URL.Path == workspaceSessionPath {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST, OPTIONS")
			http.Error(w, "工作区会话续期仅支持 POST", http.StatusMethodNotAllowed)
			return
		}
		g.exchangeTicket(w, r, host, r.URL.Query().Get(workspaceTicketParameter))
		return
	}
	if ticket := r.URL.Query().Get(workspaceTicketParameter); ticket != "" {
		g.exchangeTicket(w, r, host, ticket)
		return
	}
	cookieName, ok := g.sessionCookieName(host)
	if !ok {
		http.Error(w, "工作区服务无效", http.StatusBadRequest)
		return
	}
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		http.Error(w, "工作区会话已失效", http.StatusUnauthorized)
		return
	}
	grant, ok := g.session(cookie.Value, host)
	grant, err = g.authorizeGrant(r.Context(), grant)
	if !ok || err != nil {
		g.deleteSession(cookie.Value)
		http.SetCookie(w, g.expiredWorkspaceCookie(cookieName))
		http.Error(w, "工作区会话已失效，请重新打开", http.StatusUnauthorized)
		return
	}
	g.proxy(w, r, grant, cookie.Value)
}

func (g *Gateway) isMCPHost(host string) bool {
	pattern, ok := g.serviceHostPatterns["mcp"]
	return ok && pattern.MatchString(host)
}

func (g *Gateway) isCentralMCPPath(path string) bool {
	_, ok := parseCentralMCPPath(path)
	return ok
}

func (g *Gateway) centralMCPProjectID(host, path string) (string, bool) {
	if !strings.EqualFold(host, g.centerHost) {
		return "", false
	}
	return parseCentralMCPPath(path)
}

func parseCentralMCPPath(path string) (string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 3 || parts[0] != "workspaces" || parts[2] != "mcp" {
		return "", false
	}
	projectID, err := uuid.Parse(parts[1])
	if err != nil {
		return "", false
	}
	return strings.ToLower(projectID.String()), true
}

// serveMCPBearer authenticates external MCP clients without browser cookies.
func (g *Gateway) serveMCPBearer(w http.ResponseWriter, r *http.Request, host, pathProjectID string) bool {
	if g.resolveMCPToken == nil {
		return false
	}
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return false
	}
	actor, projectID, epoch, expiresAt, ok := g.resolveMCPToken(r.Context(), parts[1])
	if !ok || !g.now().Before(expiresAt) {
		http.Error(w, "MCP 连接令牌已失效", http.StatusUnauthorized)
		return true
	}
	if pathProjectID != "" && !strings.EqualFold(pathProjectID, projectID) {
		http.Error(w, "MCP 连接令牌与工程不匹配", http.StatusForbidden)
		return true
	}
	if pathProjectID == "" {
		expected, err := g.publicOrigin(projectID, "mcp")
		if err != nil || expected != host {
			http.Error(w, "MCP 连接令牌与工程不匹配", http.StatusForbidden)
			return true
		}
	}
	if pathProjectID != "" {
		r = r.Clone(r.Context())
		r.URL.Path = "/mcp"
		r.URL.RawPath = ""
	}
	sessionID, err := secureValue()
	if err != nil {
		http.Error(w, "建立 MCP 会话失败", http.StatusInternalServerError)
		return true
	}
	grant := gatewayGrant{Actor: actor, ProjectID: projectID, Epoch: epoch, Service: "mcp", Host: host, ExpiresAt: expiresAt, LastUsedAt: g.now(), MCPToken: parts[1]}
	g.mu.Lock()
	g.sessions[sessionID] = grant
	g.mu.Unlock()
	defer func() {
		g.mu.Lock()
		delete(g.sessions, sessionID)
		g.mu.Unlock()
	}()
	g.proxy(w, r, grant, sessionID)
	return true
}

func (g *Gateway) exchangeTicket(w http.ResponseWriter, r *http.Request, host, ticket string) {
	now := g.now().UTC()
	g.mu.Lock()
	grant, ok := g.tickets[ticket]
	g.deleteTicketLocked(ticket, grant) // 无论 Host 是否匹配，票据只允许尝试一次。
	g.mu.Unlock()
	grant, err := g.authorizeGrantUncached(r.Context(), grant)
	if !ok || grant.Host != host || !now.Before(grant.ExpiresAt) || err != nil {
		if r.URL.Path == workspaceSessionPath {
			http.Error(w, "工作区票据无效或已过期", http.StatusUnauthorized)
			return
		}
		// iframe 刷新会重新访问浏览器保存的原始 ticket URL。票据仍然不可
		// 重放；只有同一 Host 已有的有效 HttpOnly 会话可跳过交换并清理 URL。
		cookieName, cookieNameOK := g.sessionCookieName(host)
		cookie, cookieErr := r.Cookie(cookieName)
		existing, sessionOK := gatewayGrant{}, false
		if cookieNameOK && cookieErr == nil {
			existing, sessionOK = g.session(cookie.Value, host)
			existing, cookieErr = g.authorizeGrant(r.Context(), existing)
		}
		if !sessionOK || cookieErr != nil || existing.Host != host {
			http.Error(w, "工作区票据无效或已过期", http.StatusUnauthorized)
			return
		}
		g.redirectWithoutTicket(w, r)
		return
	}
	sessionID, err := secureValue()
	if err != nil {
		http.Error(w, "建立工作区会话失败", http.StatusInternalServerError)
		return
	}
	// 新票据验证成功后才允许续期；同身份、工程、代次和服务复用会话 ID，
	// 长连接据此读取最新有效期。不同 scope 或已过期 Cookie 一律建立新会话。
	cookieName, cookieNameOK := g.sessionCookieName(host)
	if !cookieNameOK {
		http.Error(w, "工作区服务无效", http.StatusBadRequest)
		return
	}
	cookie, cookieErr := r.Cookie(cookieName)
	now = g.now().UTC()
	grant.ExpiresAt = now.Add(g.sessionTTL)
	grant.LastUsedAt = now
	g.mu.Lock()
	g.cleanupPeriodicLocked(now)
	if cookieErr == nil {
		if existing, exists := g.sessions[cookie.Value]; exists && existing.Host == host && now.Before(existing.ExpiresAt) && ticketScope(existing.Actor, existing.ProjectID, existing.Epoch, existing.Service) == ticketScope(grant.Actor, grant.ProjectID, grant.Epoch, grant.Service) {
			sessionID = cookie.Value
		}
	}
	_, renewing := g.sessions[sessionID]
	if !renewing && (len(g.sessions) >= maxActiveSessions || g.actorSessionCountLocked(grant.Actor) >= maxSessionsPerActor) {
		g.cleanupLocked(now)
	}
	for !renewing && g.actorSessionCountLocked(grant.Actor) >= maxSessionsPerActor {
		g.evictOldestSessionLocked(func(item gatewayGrant) bool { return sameActor(item.Actor, grant.Actor) })
	}
	for !renewing && len(g.sessions) >= maxActiveSessions {
		g.evictOldestSessionLocked(nil)
	}
	g.sessions[sessionID] = grant
	g.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: sessionID, Path: "/", HttpOnly: true, Secure: !g.allowInsecureHTTPDev, SameSite: http.SameSiteLaxMode, MaxAge: int(g.sessionTTL.Seconds())})
	if r.URL.Path == workspaceSessionPath {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	g.redirectWithoutTicket(w, r)
}

func (g *Gateway) redirectWithoutTicket(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	target := *r.URL
	query := target.Query()
	query.Del(workspaceTicketParameter)
	target.RawQuery = query.Encode()
	http.Redirect(w, r, target.RequestURI(), http.StatusSeeOther)
}

func (g *Gateway) proxy(w http.ResponseWriter, r *http.Request, grant gatewayGrant, sessionID string) {
	done := g.service.beginActivity(grant.ProjectID)
	defer done()
	port := workspaceServicePorts[grant.Service]
	upstream := &url.URL{Scheme: "http", Host: containerName(grant.ProjectID) + "." + g.namespace + ".svc.cluster.local:" + fmt.Sprint(port)}
	proxy := httputil.NewSingleHostReverseProxy(upstream)
	proxy.Transport = g.transport
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// 上游连接固定为集群 Service，但应用层 Host 必须保持已由网关 Host 正则、
		// 一次票据和会话校验绑定的公开 Origin，供 Pi Web/Vite 精确允许主机校验。
		req.Host = grant.Host
		stripWorkspaceCredentials(req.Header)
		// 身份来自已验证的网关会话，覆盖浏览器同名头，供工作区执行用户隔离。
		req.Header.Set("X-InduForge-User-Id", grant.Actor.ID)
		req.Header.Set("X-InduForge-Tenant-Id", grant.Actor.TenantID)
		req.Header.Set("X-InduForge-Project-Id", grant.ProjectID)
		req.Header.Set("X-InduForge-Role", grant.Actor.Role)
		if grant.Service == "mcp" && grant.MCPToken != "" {
			req.Header.Set("X-InduForge-MCP-Token", grant.MCPToken)
		}
	}
	proxy.ModifyResponse = func(response *http.Response) error {
		response.Header.Del("Set-Cookie")
		if location := response.Header.Get("Location"); location != "" {
			parsed, err := url.Parse(location)
			if err != nil || parsed.IsAbs() || strings.HasPrefix(location, "//") {
				response.Header.Del("Location")
			}
		}
		return nil
	}
	proxy.ErrorHandler = func(writer http.ResponseWriter, _ *http.Request, _ error) {
		http.Error(writer, "代码工作区暂不可用", http.StatusBadGateway)
	}
	// ReverseProxy 会处理 Upgrade。记录被 Hijack 的连接并周期复核 epoch，恢复
	// bump epoch 后至多一秒关闭旧 WebSocket，不依赖客户端再次发 HTTP。
	check := func() bool { return g.sessionAuthorized(sessionID, grant) }
	// SSE 不会 Hijack，取消请求上下文使上游流也随失权关闭。
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !check() {
					cancel()
					return
				}
			}
		}
	}()
	r = r.WithContext(ctx)
	guarded := &epochResponseWriter{ResponseWriter: w, check: check}
	proxy.ServeHTTP(guarded, r)
}

// 每次长连接复核读取当前会话，禁止使用连接建立时捕获的旧有效期。
func (g *Gateway) sessionAuthorized(id string, original gatewayGrant) bool {
	current, ok := g.session(id, original.Host)
	if !ok || ticketScope(current.Actor, current.ProjectID, current.Epoch, current.Service) != ticketScope(original.Actor, original.ProjectID, original.Epoch, original.Service) {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := g.authorizeGrant(ctx, current)
	return err == nil
}

func (g *Gateway) applyCORS(w http.ResponseWriter, r *http.Request) bool {
	origin := strings.TrimRight(strings.TrimSpace(r.Header.Get("Origin")), "/")
	if origin == "" {
		return true
	}
	selfOrigin := g.scheme + "://" + strings.ToLower(r.Host)
	_, explicitlyAllowed := g.allowedOrigin[origin]
	if !explicitlyAllowed && !strings.EqualFold(origin, selfOrigin) {
		http.Error(w, "不允许的工作区来源", http.StatusForbidden)
		return false
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Vary", "Origin")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-InduForge-Authoring-Epoch")
		w.Header().Set("Access-Control-Max-Age", "300")
	}
	return true
}

func (g *Gateway) session(id, host string) (gatewayGrant, bool) {
	now := g.now().UTC()
	g.mu.Lock()
	defer g.mu.Unlock()
	grant, ok := g.sessions[id]
	if !ok || grant.Host != host || !now.Before(grant.ExpiresAt) {
		delete(g.sessions, id)
		return gatewayGrant{}, false
	}
	grant.LastUsedAt = now
	g.sessions[id] = grant
	g.cleanupPeriodicLocked(now)
	return grant, true
}

func (g *Gateway) deleteSession(id string) { g.mu.Lock(); delete(g.sessions, id); g.mu.Unlock() }

func (g *Gateway) authorizeGrant(ctx context.Context, grant gatewayGrant) (gatewayGrant, error) {
	key := grant.Actor.TenantID + "\x00" + grant.Actor.ID + "\x00" + grant.ProjectID + "\x00" + grant.Epoch
	now := g.now().UTC()
	if !now.Before(grant.ExpiresAt) {
		return gatewayGrant{}, auth.ErrPermissionDenied
	}
	g.validationMu.Lock()
	if cached, ok := g.validations[key]; ok && now.Before(cached.expiresAt) {
		g.validationMu.Unlock()
		grant.Actor = cached.actor
		return grant, nil
	}
	if flight, ok := g.flights[key]; ok {
		g.validationMu.Unlock()
		select {
		case <-ctx.Done():
			return gatewayGrant{}, ctx.Err()
		case <-flight.done:
			grant.Actor = flight.actor
			return grant, flight.err
		}
	}
	flight := &grantValidationFlight{done: make(chan struct{})}
	g.flights[key] = flight
	g.validationMu.Unlock()

	validated, err := g.authorizeGrantUncached(ctx, grant)
	g.validationMu.Lock()
	flight.actor, flight.err = validated.Actor, err
	if err == nil {
		// 一秒共享缓存同时约束 HTTP 静态资源和所有 WS 连接。恢复期间工作区
		// Pod 先被冻结，epoch bump 后旧权限的最坏容忍窗口不超过一秒。
		g.validations[key] = grantValidation{actor: validated.Actor, expiresAt: g.now().UTC().Add(time.Second)}
	}
	delete(g.flights, key)
	close(flight.done)
	g.validationMu.Unlock()
	return validated, err
}

func (g *Gateway) authorizeGrantUncached(ctx context.Context, grant gatewayGrant) (gatewayGrant, error) {
	actor, err := g.resolveUser(ctx, grant.Actor.ID)
	if err != nil || actor.TenantID != grant.Actor.TenantID {
		return gatewayGrant{}, auth.ErrPermissionDenied
	}
	grant.Actor = actor
	if err := g.service.RequireProxyEpoch(ctx, actor, grant.ProjectID, grant.Epoch); err != nil {
		return gatewayGrant{}, err
	}
	return grant, nil
}

func (g *Gateway) cleanupLocked(now time.Time) {
	for id, item := range g.tickets {
		if !now.Before(item.ExpiresAt) {
			g.deleteTicketLocked(id, item)
		}
	}
	for id, item := range g.sessions {
		if !now.Before(item.ExpiresAt) {
			delete(g.sessions, id)
		}
	}
}

func (g *Gateway) cleanupPeriodicLocked(now time.Time) {
	if g.lastCleanup.IsZero() || now.Sub(g.lastCleanup) >= cleanupInterval {
		g.cleanupLocked(now)
		g.lastCleanup = now
	}
}

func ticketScope(actor auth.User, projectID, epoch, service string) string {
	return actor.TenantID + "\x00" + actor.ID + "\x00" + projectID + "\x00" + epoch + "\x00" + service
}

func (g *Gateway) deleteTicketLocked(id string, grant gatewayGrant) {
	delete(g.tickets, id)
	scope := ticketScope(grant.Actor, grant.ProjectID, grant.Epoch, grant.Service)
	if g.ticketByScope[scope] == id {
		delete(g.ticketByScope, scope)
	}
}

func (g *Gateway) evictOldestTicketLocked() {
	var oldestID string
	var oldest gatewayGrant
	for id, item := range g.tickets {
		if oldestID == "" || item.LastUsedAt.Before(oldest.LastUsedAt) {
			oldestID, oldest = id, item
		}
	}
	if oldestID != "" {
		g.deleteTicketLocked(oldestID, oldest)
	}
}

func sameActor(left, right auth.User) bool {
	return left.TenantID == right.TenantID && left.ID == right.ID
}

func (g *Gateway) actorSessionCountLocked(actor auth.User) int {
	count := 0
	for _, item := range g.sessions {
		if sameActor(item.Actor, actor) {
			count++
		}
	}
	return count
}

func (g *Gateway) evictOldestSessionLocked(match func(gatewayGrant) bool) {
	var oldestID string
	var oldest gatewayGrant
	for id, item := range g.sessions {
		if match != nil && !match(item) {
			continue
		}
		if oldestID == "" || item.LastUsedAt.Before(oldest.LastUsedAt) {
			oldestID, oldest = id, item
		}
	}
	if oldestID != "" {
		delete(g.sessions, oldestID)
	}
}

func stripWorkspaceCredentials(header http.Header) {
	for name := range header {
		lower := strings.ToLower(name)
		if lower == "authorization" || lower == "cookie" || lower == "proxy-authorization" || strings.HasPrefix(lower, "x-forwarded-") || strings.HasPrefix(lower, "x-induforge-internal-") || lower == "x-induforge-user-id" || lower == "x-induforge-tenant-id" || lower == "x-induforge-project-id" || lower == "x-induforge-role" {
			header.Del(name)
		}
	}
}

func (g *Gateway) expiredWorkspaceCookie(name string) *http.Cookie {
	return &http.Cookie{Name: name, Value: "", Path: "/", HttpOnly: true, Secure: !g.allowInsecureHTTPDev, SameSite: http.SameSiteLaxMode, MaxAge: -1}
}

func workspaceRequestHostname(host string) string {
	parsed, err := url.Parse("//" + host)
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Hostname())
}

func (g *Gateway) sessionCookieName(host string) (string, bool) {
	if !g.allowInsecureHTTPDev {
		return workspaceSessionCookie, true
	}
	for serviceName, pattern := range g.serviceHostPatterns {
		if pattern.MatchString(host) {
			return workspaceInsecureCookiePrefix + serviceName, true
		}
	}
	return "", false
}

// requestSchemeMatches 只接受 TLS 或受控反向代理明确标注的协议，避免 HTTP 请求借
// X-Forwarded-Proto 混入生产工作区。边缘代理固定覆盖该 Header，应用不猜测原始协议。
func (g *Gateway) requestSchemeMatches(r *http.Request) bool {
	forwarded := strings.TrimSpace(strings.ToLower(r.Header.Get("X-Forwarded-Proto")))
	if forwarded != "" && forwarded != "http" && forwarded != "https" {
		return false
	}
	if r.TLS != nil {
		return g.scheme == "https" && (forwarded == "" || forwarded == "https")
	}
	return forwarded == g.scheme
}

func secureValue() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

type epochResponseWriter struct {
	http.ResponseWriter
	check func() bool
}

func (w *epochResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("WebSocket Hijack 不受支持")
	}
	conn, buffer, err := hijacker.Hijack()
	if err == nil {
		guardedConn := &epochGuardedConn{Conn: conn, done: make(chan struct{})}
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-guardedConn.done:
					return
				case <-ticker.C:
					if !w.check() {
						_ = guardedConn.Close()
						return
					}
				}
			}
		}()
		return guardedConn, buffer, nil
	}
	return conn, buffer, err
}

func (w *epochResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

type epochGuardedConn struct {
	net.Conn
	done chan struct{}
	once sync.Once
}

func (c *epochGuardedConn) Close() error {
	c.once.Do(func() { close(c.done) })
	return c.Conn.Close()
}
