package gateway

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
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
	upstream, _ := url.Parse(config.RuntimeAPIURL)
	proxy := httputil.NewSingleHostReverseProxy(upstream)
	defaultDirector := proxy.Director
	proxy.Director = func(request *http.Request) {
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

// osDirFS 单独包一层，便于测试替换与保持 handler 只接触受限 fs.FS。
var osDirFS = func(root string) fs.FS { return osDirFSImpl(root) }

func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(s.serveHTTP)
}

func (s *Server) serveHTTP(writer http.ResponseWriter, request *http.Request) {
	setSecurityHeaders(writer.Header())
	switch {
	case request.URL.Path == "/health":
		s.handleHealth(writer, request)
	case request.URL.Path == "/api/v1/status":
		s.handleStatus(writer, request)
	case request.URL.Path == "/api/v1/runtime/status":
		request = request.Clone(request.Context())
		request.URL.Path = "/api/v1/status"
		s.serveProxy(writer, request)
	case strings.HasPrefix(request.URL.Path, "/api/v1/") || request.URL.Path == "/api/v1":
		s.serveProxy(writer, request)
	case request.URL.Path == "/ws" || strings.HasPrefix(request.URL.Path, "/ws/"):
		s.serveProxy(writer, request)
	default:
		s.serveAsset(writer, request)
	}
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
