package gateway

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

const maxProxyBodyBytes = 2 << 20

type Server struct {
	config     Config
	startedAt  time.Time
	assets     fs.FS
	proxy      *httputil.ReverseProxy
	httpClient *http.Client
}

type envelope struct {
	Code  int    `json:"code"`
	Msg   string `json:"msg"`
	Data  any    `json:"data"`
	ReqID string `json:"reqId"`
}

func New(config Config) (*Server, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	viewerToken, err := loadViewerToken(config.ViewerTokenFile)
	if err != nil {
		return nil, err
	}
	upstream, _ := url.Parse(config.RuntimeAPIURL)
	viewerAuth := "Bearer " + viewerToken
	proxy := httputil.NewSingleHostReverseProxy(upstream)
	defaultDirector := proxy.Director
	proxy.Director = func(request *http.Request) {
		viewerProxy := request.Header.Get("X-InduForge-Viewer-Proxy") == "1"
		forwardedHost := request.Host
		forwardedProto := request.Header.Get("X-Forwarded-Proto")
		if forwardedProto == "" {
			forwardedProto = "http"
			if request.TLS != nil {
				forwardedProto = "https"
			}
		}
		defaultDirector(request)
		request.Host = upstream.Host
		request.Header.Del("Authorization")
		request.Header.Del("X-InduForge-Viewer-Proxy")
		if viewerProxy {
			request.Header.Set("Authorization", viewerAuth)
		}
		request.Header.Set("X-Forwarded-Host", forwardedHost)
		request.Header.Set("X-Forwarded-Proto", forwardedProto)
		request.Header.Set("X-InduForge-Deployment-Id", config.DeploymentID)
		request.Header.Set("X-InduForge-Project-Id", config.ProjectID)
	}
	server := &Server{
		config:     config,
		startedAt:  time.Now().UTC(),
		assets:     osDirFS(config.ClientRoot),
		proxy:      proxy,
		httpClient: &http.Client{Timeout: 2 * time.Second},
	}
	proxy.ErrorHandler = server.proxyError
	return server, nil
}

func loadViewerToken(path string) (string, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取 viewer-token-file: %w", err)
	}
	token := strings.TrimSpace(string(payload))
	if token == "" || len(token) > 4096 || strings.ContainsAny(token, "\r\n") {
		return "", errors.New("viewer-token-file 内容非法")
	}
	return token, nil
}

// osDirFS 单独包一层，便于测试替换与保持 handler 只接触受限 fs.FS。
var osDirFS = func(root string) fs.FS { return osDirFSImpl(root) }

func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(s.serveHTTP)
}

func (s *Server) serveHTTP(writer http.ResponseWriter, request *http.Request) {
	setSecurityHeaders(writer.Header())
	if !sameOriginRequest(request) {
		writeJSON(writer, http.StatusForbidden, envelope{Code: 40301, Msg: "工程入口来源不匹配", Data: nil, ReqID: requestID(request)})
		return
	}
	switch {
	case request.URL.Path == "/health":
		s.handleHealth(writer, request)
	case request.URL.Path == "/api/v1/status":
		s.handleStatus(writer, request)
	case request.URL.Path == "/api/v1/runtime/status":
		request = request.Clone(request.Context())
		request.URL.Path = "/api/v1/status"
		s.serveProxy(writer, request)
	case viewerRuntimeRequest(request):
		s.serveViewerProxy(writer, request)
	case runtimeMutationRequest(request):
		writeJSON(writer, http.StatusForbidden, envelope{Code: 40301, Msg: "发布入口仅允许 viewer 只读运行能力", Data: nil, ReqID: requestID(request)})
	case strings.HasPrefix(request.URL.Path, "/api/v1/runtime/") || strings.HasPrefix(request.URL.Path, "/ws/"):
		http.NotFound(writer, request)
	default:
		s.serveAsset(writer, request)
	}
}

// sameOriginRequest 只接受当前工程入口自身的浏览器来源；客户端不能借 Gateway
// 覆盖部署身份或把同一 viewer token 转发给其他站点。
func sameOriginRequest(request *http.Request) bool {
	if strings.TrimSpace(request.Host) == "" {
		return false
	}
	origin := strings.TrimSpace(request.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != "" && strings.EqualFold(parsed.Host, request.Host)
}

func viewerRuntimeRequest(request *http.Request) bool {
	if request.Method != http.MethodGet {
		return false
	}
	path := request.URL.Path
	if path == "/ws/v1/points" || path == "/ws/v1/alarms" {
		return true
	}
	if path == "/api/v1/runtime/catalog" || path == "/api/v1/runtime/alarms/current" || path == "/api/v1/runtime/computes" {
		return true
	}
	if !strings.HasPrefix(path, "/api/v1/runtime/points/") {
		return false
	}
	pointPath := strings.TrimPrefix(path, "/api/v1/runtime/points/")
	return pointPath != "" && (!strings.Contains(pointPath, "/") || (strings.Count(pointPath, "/") == 1 && strings.HasSuffix(pointPath, "/history")))
}

func runtimeMutationRequest(request *http.Request) bool {
	return strings.HasPrefix(request.URL.Path, "/api/v1/runtime/") || strings.HasPrefix(request.URL.Path, "/ws/")
}

func (s *Server) serveViewerProxy(writer http.ResponseWriter, request *http.Request) {
	request = request.Clone(request.Context())
	request.Header.Del("Authorization")
	request.Header.Set("X-InduForge-Viewer-Proxy", "1")
	s.serveProxy(writer, request)
}

func (s *Server) serveProxy(writer http.ResponseWriter, request *http.Request) {
	if request.Body != nil {
		request.Body = http.MaxBytesReader(writer, request.Body, maxProxyBodyBytes)
	}
	s.proxy.ServeHTTP(writer, request)
}

func (s *Server) serveAsset(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		writer.Header().Set("Allow", "GET, HEAD")
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if strings.ContainsRune(request.URL.Path, '\x00') || strings.Contains(request.URL.Path, "\\") || hasParentSegment(request.URL.Path) {
		http.NotFound(writer, request)
		return
	}
	clean := strings.TrimPrefix(path.Clean("/"+request.URL.Path), "/")
	if clean == "." || clean == "" {
		clean = "index.html"
	}
	if !fs.ValidPath(clean) || hasHiddenSegment(clean) {
		http.NotFound(writer, request)
		return
	}
	if info, err := fs.Stat(s.assets, clean); err != nil || info.IsDir() {
		// 只有无扩展名的前端路由才回退到 SPA 入口；缺失的 JS/CSS/图片保持 404，
		// 避免浏览器把 index.html 当成静态资源并掩盖 Release 不完整。
		if path.Ext(clean) != "" {
			http.NotFound(writer, request)
			return
		}
		clean = "index.html"
	}
	content, err := fs.ReadFile(s.assets, clean)
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	if contentType := mime.TypeByExtension(path.Ext(clean)); contentType != "" {
		writer.Header().Set("Content-Type", contentType)
	}
	if clean == "index.html" {
		writer.Header().Set("Cache-Control", "no-cache")
	} else {
		writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	writer.Header().Set("Content-Length", strconvItoa(len(content)))
	if request.Method == http.MethodHead {
		writer.WriteHeader(http.StatusOK)
		return
	}
	_, _ = writer.Write(content)
}

func hasParentSegment(value string) bool {
	for _, segment := range strings.Split(value, "/") {
		if segment == ".." {
			return true
		}
	}
	return false
}

func (s *Server) handleHealth(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", "GET")
		writeJSON(writer, http.StatusMethodNotAllowed, envelope{Code: 40005, Msg: "method not allowed", Data: nil, ReqID: requestID(request)})
		return
	}
	upstream := s.runtimeAPIAvailable(request)
	status := http.StatusOK
	code, message := 0, "ok"
	if !upstream {
		// readinessProbe 使用此端点；静态入口已加载但 Runtime API 未连接时不能
		// 接收流量，避免把失败的动态请求伪装为可用工程。
		status, code, message = http.StatusServiceUnavailable, 50031, "runtime-api unavailable"
	}
	writeJSON(writer, status, envelope{Code: code, Msg: message, Data: map[string]any{
		"status":         "UP",
		"upstreamStatus": map[bool]string{true: "CONNECTED", false: "DISCONNECTED"}[upstream],
		"observedAt":     time.Now().UTC().Format(time.RFC3339Nano),
	}, ReqID: requestID(request)})
}

func (s *Server) handleStatus(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", "GET")
		writeJSON(writer, http.StatusMethodNotAllowed, envelope{Code: 40005, Msg: "method not allowed", Data: nil, ReqID: requestID(request)})
		return
	}
	now := time.Now().UTC()
	upstream := s.runtimeAPIAvailable(request)
	health := "HEALTHY"
	reason := any(nil)
	if !upstream {
		health = "DEGRADED"
		reason = "runtime-api-unavailable"
	}
	writeJSON(writer, http.StatusOK, envelope{Code: 0, Msg: "ok", Data: map[string]any{
		"schemaVersion":  "runtime-health-status.v1",
		"lifecycleState": "RUNNING",
		"healthState":    health,
		"componentRole":  "project-gateway",
		"deploymentId":   s.config.DeploymentID,
		"accountId":      s.config.AccountID,
		"projectId":      s.config.ProjectID,
		"version":        s.config.Version,
		"executionForm":  s.config.ExecutionForm,
		"siteId":         s.config.SiteID,
		"nodeId":         s.config.NodeID,
		"processId":      os.Getpid(),
		"startedAt":      s.startedAt.Format(time.RFC3339Nano),
		"uptimeSeconds":  int64(now.Sub(s.startedAt).Seconds()),
		"businessFreshness": map[string]any{
			"state":               "UNKNOWN",
			"lastBusinessEventAt": nil,
			"evaluatedAt":         now.Format(time.RFC3339Nano),
		},
		"reasonCode": reason,
		"lastError":  reason,
		"observedAt": now.Format(time.RFC3339Nano),
	}, ReqID: requestID(request)})
}

func (s *Server) runtimeAPIAvailable(request *http.Request) bool {
	probe, err := http.NewRequestWithContext(request.Context(), http.MethodGet, s.config.RuntimeAPIURL+"/health", nil)
	if err != nil {
		return false
	}
	response, err := s.httpClient.Do(probe)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	return response.StatusCode >= 200 && response.StatusCode < 300
}

func (s *Server) proxyError(writer http.ResponseWriter, request *http.Request, _ error) {
	writeJSON(writer, http.StatusBadGateway, envelope{Code: 50031, Msg: "runtime-api unavailable", Data: nil, ReqID: requestID(request)})
}

func setSecurityHeaders(header http.Header) {
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("X-Frame-Options", "SAMEORIGIN")
	header.Set("Referrer-Policy", "same-origin")
	header.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
}

func hasHiddenSegment(name string) bool {
	for _, segment := range strings.Split(name, "/") {
		if strings.HasPrefix(segment, ".") {
			return true
		}
	}
	return false
}

func writeJSON(writer http.ResponseWriter, status int, value envelope) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func requestID(request *http.Request) string {
	if existing := strings.TrimSpace(request.Header.Get("X-Request-Id")); existing != "" && len(existing) <= 128 {
		return existing
	}
	var value [12]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "req_unavailable"
	}
	return "req_" + hex.EncodeToString(value[:])
}
