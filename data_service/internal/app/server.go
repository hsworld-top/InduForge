package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/cache"
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

// NewServer 创建一个带有默认超时配置的 HTTP 服务实例。
func NewServer(cfg config.Config) (*Server, error) {
	routerOptions, cleanup, err := buildRouteDependencies(cfg)
	if err != nil {
		log.Printf("warning: optional routes disabled: %v", err)
		routerOptions = nil
		cleanup = nil
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

// Handler 返回当前服务使用的 HTTP 处理器。
func (s *Server) Handler() http.Handler {
	if s == nil || s.httpServer == nil {
		return nil
	}
	return s.httpServer.Handler
}

// Close 释放 Server 持有的外部资源。
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
	if err := config.ValidateConnectionsDependencies(cfg); err != nil {
		return nil, nil, err
	}

	jwtValidator, err := auth.NewJWTValidator(cfg.JWTSecret)
	if err != nil {
		return nil, nil, err
	}

	pool, err := postgres.NewPool(context.Background(), postgres.PoolConfig{
		DatabaseURL: strings.TrimSpace(cfg.DatabaseURL),
		SearchPath:  strings.TrimSpace(cfg.DatabaseSearchPath),
	})
	if err != nil {
		return nil, nil, err
	}
	cleanupFns := []func(){pool.Close}

	connectionRepository := repository.NewConnectionRepository(pool)
	queryRepository := repository.NewQueryRepository(pool)
	dataPointRepository := repository.NewDataPointRepository(pool)
	mqttRepository := repository.NewMqttRepository(pool)
	protocolWave1Repository := repository.NewProtocolWave1Repository(pool)
	protocolWave2Repository := repository.NewProtocolWave2Repository(pool)

	connectionService := service.NewConnectionService(connectionRepository)
	queryService := service.NewQueryService(queryRepository, connectionRepository, pool)
	dataPointService := service.NewDataPointService(dataPointRepository, queryService)
	mqttService := service.NewMqttService(mqttRepository)
	protocolWave1Service := service.NewProtocolWave1Service(protocolWave1Repository)
	protocolWave2Service := service.NewProtocolWave2Service(protocolWave2Repository)

	connectionHandler := handler.NewConnectionHandler(connectionService)
	queryHandler := handler.NewQueryHandler(queryService)
	dataPointHandler := handler.NewDataPointHandler(dataPointService)
	mqttHandler := handler.NewMqttHandler(mqttService)
	protocolWave1Handler := handler.NewProtocolWave1Handler(protocolWave1Service)
	protocolWave2Handler := handler.NewProtocolWave2Handler(protocolWave2Service)

	routeOptions := []router.Option{
		router.WithConnectionRoutes(connectionHandler, jwtValidator),
		router.WithDataRoutes(queryHandler, dataPointHandler, jwtValidator),
		router.WithMqttRoutes(mqttHandler, jwtValidator),
		router.WithProtocolWave1Routes(protocolWave1Handler, jwtValidator),
		router.WithProtocolWave2Routes(protocolWave2Handler, jwtValidator),
	}

	if err := config.ValidatePreviewDependencies(cfg); err != nil {
		log.Printf("warning: preview routes disabled: %v", err)
	} else {
		redisClient, redisErr := cache.NewRedisClient(context.Background(), cache.RedisConfig{
			Addr:     strings.TrimSpace(cfg.RedisAddr),
			Password: strings.TrimSpace(cfg.RedisPassword),
			DB:       cfg.RedisDB,
		})
		if redisErr != nil {
			log.Printf("warning: preview routes disabled: %v", redisErr)
		} else {
			cleanupFns = append(cleanupFns, func() {
				_ = redisClient.Close()
			})

			previewRepository := repository.NewPreviewSessionRepository(pool)
			previewService := service.NewPreviewSessionService(previewRepository, redisClient)
			previewHandler := handler.NewPreviewHandler(previewService)
			routeOptions = append(routeOptions, router.WithPreviewRoutes(previewHandler, jwtValidator))
		}
	}

	return routeOptions, joinCleanup(cleanupFns...), nil
}

func joinCleanup(cleanups ...func()) func() {
	return func() {
		for index := len(cleanups) - 1; index >= 0; index-- {
			if cleanups[index] == nil {
				continue
			}
			cleanups[index]()
		}
	}
}
