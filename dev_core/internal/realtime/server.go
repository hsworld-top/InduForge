package realtime

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/zishang520/socket.io/v2/socket"
)

type Server struct {
	io          *socket.Server
	handler     http.Handler
	authService *auth.Service
	logger      *slog.Logger
}

func New(authService *auth.Service, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	ioServer := socket.NewServer(nil, nil)
	ioServer.SetPath("/control-socket.io")
	server := &Server{io: ioServer, authService: authService, logger: logger}
	server.handler = server.io.ServeHandler(nil)
	server.io.On("connection", server.onConnection)
	return server
}

func (s *Server) Handler() http.Handler { return s.handler }

func (s *Server) Close() {
	s.io.Close(func(err error) {
		if err != nil {
			s.logger.Warn("关闭 Socket.IO 服务失败", "error", err)
		}
	})
}

func (s *Server) onConnection(arguments ...any) {
	client, ok := arguments[0].(*socket.Socket)
	if !ok {
		return
	}
	token := handshakeToken(client.Handshake())
	actor, err := s.authService.Authenticate(context.Background(), token)
	if err != nil {
		client.Disconnect(true)
		return
	}
	room := socket.Room("ops:tenant:" + actor.TenantID)
	client.On("ops:subscribe", func(...any) { client.Join(room) })
	client.On("ops:unsubscribe", func(...any) { client.Leave(room) })
}

func (s *Server) NodePending(tenantID string, payload map[string]any) {
	s.emit(tenantID, "ops:node:pending", payload)
}

func (s *Server) NodeMetrics(tenantID, nodeID string, metrics map[string]any) {
	s.emit(tenantID, "ops:node:metrics", map[string]any{"nodeId": nodeID, "metrics": metrics})
}

func (s *Server) NodeStatus(tenantID, nodeID, status string) {
	s.emit(tenantID, "ops:node:status", map[string]any{"nodeId": nodeID, "status": status})
}

func (s *Server) ProjectMetrics(tenantID, nodeID, projectID string, metrics map[string]any) {
	s.emit(tenantID, "ops:project:metrics", map[string]any{"nodeId": nodeID, "projectId": projectID, "metrics": metrics})
}

func (s *Server) DeployStatus(tenantID string, payload map[string]any) {
	s.emit(tenantID, "ops:deploy:status", payload)
}

func (s *Server) emit(tenantID, event string, payload map[string]any) {
	if tenantID == "" {
		return
	}
	payload["timestamp"] = time.Now().UnixMilli()
	if err := s.io.To(socket.Room("ops:tenant:"+tenantID)).Emit(event, payload); err != nil {
		s.logger.Warn("发送实时事件失败", "event", event, "tenantId", tenantID, "error", err)
	}
}

func handshakeToken(handshake *socket.Handshake) string {
	if handshake == nil {
		return ""
	}
	for _, rawCookie := range handshake.Headers["Cookie"] {
		for _, item := range strings.Split(rawCookie, ";") {
			pair := strings.SplitN(strings.TrimSpace(item), "=", 2)
			if len(pair) == 2 && pair[0] == "if_access" {
				return strings.TrimSpace(pair[1])
			}
		}
	}
	return ""
}
