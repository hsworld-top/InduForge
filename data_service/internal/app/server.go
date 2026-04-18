package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/config"
	"github.com/indu-forge/data_service/internal/db/postgres"
	"github.com/indu-forge/data_service/internal/http/handler"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/router"
	"github.com/indu-forge/data_service/internal/repository"
	"github.com/indu-forge/data_service/internal/service"
)

const (
	defaultReadHeaderTimeout = 5 * time.Second
	defaultReadTimeout       = 10 * time.Second
	defaultWriteTimeout      = 10 * time.Second
	defaultIdleTimeout       = 60 * time.Second
)

// Server 封装 data_service 的 HTTP 服务生命周期。
type Server struct {
	httpServer  *http.Server
	cleanup     func()
	cleanupOnce sync.Once
}

type routeDependenciesFactory func(config.Config) ([]router.Option, func(), error)

var buildRouteDependencies routeDependenciesFactory = defaultRouteDependenciesFactory

// NewServer 创建一个带有默认超时配置的 HTTP 服务实例，并默认装配 connections 路由依赖。
func NewServer(cfg config.Config) (*Server, error) {
	routerOptions, cleanup, err := buildRouteDependencies(cfg)
	if err != nil {
		return nil, err
	}

	return newServer(cfg, middleware.RequestIDMiddleware(router.NewRouter(routerOptions...)), cleanup), nil
}

// newServer 允许测试复用生产级 HTTP Server 装配逻辑。
func newServer(cfg config.Config, handler http.Handler, cleanup func()) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:              cfg.Addr,
			Handler:           handler,
			ReadHeaderTimeout: defaultReadHeaderTimeout,
			ReadTimeout:       defaultReadTimeout,
			WriteTimeout:      defaultWriteTimeout,
			IdleTimeout:       defaultIdleTimeout,
		},
		cleanup: cleanup,
	}
}

// Handler 返回当前服务使用的 HTTP 处理器，便于测试复用默认装配路径。
func (s *Server) Handler() http.Handler {
	if s == nil || s.httpServer == nil {
		return nil
	}
	return s.httpServer.Handler
}

// Close 释放由 Server 持有的外部依赖资源。
func (s *Server) Close() {
	if s == nil {
		return
	}
	s.cleanupOnce.Do(func() {
		if s.cleanup != nil {
			s.cleanup()
		}
	})
}

// Run 启动服务并在收到取消信号后优雅退出。
func (s *Server) Run(ctx context.Context) error {
	defer s.Close()

	errCh := make(chan error, 1)

	go func() {
		errCh <- s.httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	case err := <-errCh:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func defaultRouteDependenciesFactory(cfg config.Config) ([]router.Option, func(), error) {
	databaseURL := strings.TrimSpace(cfg.DatabaseURL)
	if databaseURL == "" {
		return nil, nil, fmt.Errorf("DATA_SERVICE_DATABASE_URL 未配置")
	}

	jwtValidator, err := auth.NewJWTValidator(cfg.JWTSecret)
	if err != nil {
		return nil, nil, err
	}

	pool, err := postgres.NewPool(context.Background(), postgres.PoolConfig{
		DatabaseURL: databaseURL,
		SearchPath:  strings.TrimSpace(cfg.DatabaseSearchPath),
	})
	if err != nil {
		return nil, nil, err
	}

	connectionRepository := repository.NewConnectionRepository(pool)
	connectionService := service.NewConnectionService(connectionRepository)
	connectionHandler := handler.NewConnectionHandler(connectionService)

	return []router.Option{
		router.WithConnectionRoutes(connectionHandler, jwtValidator),
	}, pool.Close, nil
}
