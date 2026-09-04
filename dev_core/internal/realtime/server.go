package realtime

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/zishang520/socket.io/v2/socket"
)

type Server struct {
	io        *socket.Server
	handler   http.Handler
	logger    *slog.Logger
	ops       *opsHub
	authSlots chan struct{}
}

func New(authService *auth.Service, logger *slog.Logger) *Server {
	return newServer(authService.AuthenticateSession, logger)
}

func newServer(validate sessionValidator, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	ioServer := socket.NewServer(nil, nil)
	ioServer.SetPath("/control-socket.io")
	server := &Server{io: ioServer, logger: logger, authSlots: make(chan struct{}, 64)}
	server.ops = newOpsHub(validate)
	server.handler = server.io.ServeHandler(nil)
	// namespace CONNECT 回复必须晚于鉴权及监听器安装，否则客户端首个watch可能丢失。
	server.io.Use(server.authenticateConnection)
	return server
}

func (s *Server) Handler() http.Handler { return s.handler }

func (s *Server) Close() {
	s.ops.close()
	s.io.Close(func(err error) {
		if err != nil {
			s.logger.Warn("关闭 Socket.IO 服务失败", "error", err)
		}
	})
}

func (s *Server) authenticateConnection(client *socket.Socket, next func(*socket.ExtendedError)) {
	// 第三方库在EngineIO读循环同步调用middleware。鉴权必须异步且有并发上限，
	// 否则慢数据库会阻塞close/ping处理，连接断开也不能取消鉴权。
	select {
	case s.authSlots <- struct{}{}:
		go func() {
			defer func() { <-s.authSlots }()
			s.admitConnection(client, next)
		}()
	default:
		next(socket.NewExtendedError("realtime authentication busy", nil))
	}
}

func (s *Server) admitConnection(client *socket.Socket, next func(*socket.ExtendedError)) {
	id := string(client.Id())
	token := handshakeToken(client.Handshake())
	ctx, cancel := context.WithTimeout(s.ops.ctx, 3*time.Second)
	defer cancel()
	// Engine连接可在慢鉴权期间关闭，此时namespace尚未connected，不会发disconnect。
	// 同一把锁覆盖关闭标记和hub准入，保证关闭与add交错也不留下幽灵连接。
	var admission sync.Mutex
	closed := false
	cleanup := func(...any) {
		admission.Lock()
		closed = true
		admission.Unlock()
		cancel()
		s.ops.remove(id)
	}
	client.Conn().Once("close", cleanup)
	client.On("disconnect", cleanup)
	if client.Conn().ReadyState() != "open" {
		next(socket.NewExtendedError("connection closed", nil))
		return
	}
	actor, expires, err := s.ops.validate(ctx, token)
	if err != nil || ctx.Err() != nil {
		code := opsAuthForbidden
		if errors.Is(err, auth.ErrAccessTokenExpired) {
			code = opsAuthExpired
		}
		next(socket.NewExtendedError("unauthorized", map[string]any{"code": code}))
		return
	}
	writable := func() bool {
		return client.Connected() && client.Conn().ReadyState() == "open" && client.Conn().Transport().Writable()
	}
	admission.Lock()
	if closed || client.Conn().ReadyState() != "open" {
		admission.Unlock()
		next(socket.NewExtendedError("connection closed", nil))
		return
	}
	added := s.ops.add(id, token, actor, expires, func(event string, payload any) error {
		// Emit本身不返回底层队列错误，必须在进入EngineIO前检查真实transport背压。
		if !writable() {
			return fmt.Errorf("实时连接发送背压")
		}
		return client.Emit(event, payload)
	}, writable, func() { client.Disconnect(true) })
	admission.Unlock()
	if !added {
		next(socket.NewExtendedError("realtime capacity exceeded", nil))
		return
	}
	client.On("ops:watch", func(values ...any) {
		if len(values) > 0 {
			s.ops.watch(id, values[0], false)
		}
	})
	client.On("ops:unwatch", func(values ...any) {
		if len(values) > 0 {
			s.ops.watch(id, values[0], true)
		}
	})
	room := socket.Room("ops:tenant:" + actor.TenantID)
	client.On("ops:subscribe", func(...any) {
		if auth.RequireCapability(actor, auth.CapabilityNodeRead) == nil {
			client.Join(room)
		}
	})
	client.On("ops:unsubscribe", func(...any) { client.Leave(room) })
	next(nil)
}

// PublishChange 只发布状态已提交后的失效通知；HTTP 快照始终是权威数据源。
func (s *Server) PublishChange(tenantID string, topics, entityIDs []string, terminal bool) {
	s.ops.publish(tenantID, topics, entityIDs, terminal)
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
