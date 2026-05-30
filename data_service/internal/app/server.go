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
	"github.com/indu-forge/data_service/internal/bootstrap"
	"github.com/indu-forge/data_service/internal/cache"
	"github.com/indu-forge/data_service/internal/config"
	"github.com/indu-forge/data_service/internal/db/postgres"
	enginecompute "github.com/indu-forge/data_service/internal/engine/compute"
	"github.com/indu-forge/data_service/internal/http/handler"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/router"
	previewsocket "github.com/indu-forge/data_service/internal/http/socket"
	"github.com/indu-forge/data_service/internal/repository"
	"github.com/indu-forge/data_service/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
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
var logf = log.Printf

// NewServer 创建一个带有默认超时配置的 HTTP 服务实例。
func NewServer(cfg config.Config) (*Server, error) {
	routerOptions, cleanup, err := buildRouteDependencies(cfg)
	if err != nil {
		return nil, err
	}

	logf("info: data_service 路由装配完成 addr=%s", cfg.Addr)
	return newServer(cfg, middleware.RequestIDMiddleware(middleware.AccessLogMiddleware(router.NewRouter(routerOptions...))), cleanup), nil
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

	logf("info: data_service 准备启动 HTTP 服务 addr=%s", s.httpServer.Addr)
	errCh := make(chan error, 1)

	go func() {
		errCh <- s.httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		logf("info: data_service 收到退出信号，开始关闭 HTTP 服务")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}
		logf("info: data_service HTTP 服务已关闭")
		return nil
	case err := <-errCh:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			logf("info: data_service HTTP 服务已关闭")
			return nil
		}
		logf("error: data_service HTTP 服务异常退出: %v", err)
		return err
	}
}

func defaultRouteDependenciesFactory(cfg config.Config) ([]router.Option, func(), error) {
	if err := config.ValidateConnectionsDependencies(cfg); err != nil {
		return nil, nil, err
	}
	logf("info: data_service 启动阶段=connections-deps status=ready")
	if err := bootstrap.NewRuntimeBootstrapper().EnsureReady(context.Background(), cfg); err != nil {
		return nil, nil, err
	}
	logf("info: data_service 启动阶段=database-bootstrap status=ready")

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

	var devPool *pgxpool.Pool
	if strings.TrimSpace(cfg.DevDatabaseURL) != "" {
		devPool, err = postgres.NewPool(context.Background(), postgres.PoolConfig{
			DatabaseURL: strings.TrimSpace(cfg.DevDatabaseURL),
			SearchPath:  "",
		})
		if err != nil {
			return nil, nil, err
		}
		cleanupFns = append(cleanupFns, devPool.Close)
	}

	connectionRepository := repository.NewConnectionRepository(pool)
	accessSourceRepository := repository.NewAccessSourceRepository(pool)
	alarmRuleRepository := repository.NewAlarmRuleRepository(pool)
	alarmPolicyRepository := repository.NewAlarmPolicyRepository(pool)
	contractCheckRepository := repository.NewContractCheckRepository(pool)
	queryRepository := repository.NewQueryRepository(pool)
	workbenchGroupRepository := repository.NewWorkbenchGroupRepository(pool)
	dataPointRepository := repository.NewDataPointRepository(pool)
	modbusModelingRepository := repository.NewModbusModelingRepository(pool)
	mqttRepository := repository.NewMqttRepository(pool)
	opcuaModelingRepository := repository.NewOpcuaModelingRepository(pool)
	s7ModelingRepository := repository.NewS7ModelingRepository(pool)
	projectSnapshotRepository := repository.NewProjectSnapshotRepository(pool)
	protocolWave1Repository := repository.NewProtocolWave1Repository(pool)
	protocolWave2Repository := repository.NewProtocolWave2Repository(pool)
	kafkaWorkbenchRepository := repository.NewKafkaWorkbenchRepository(pool)
	httpWorkbenchRepository := repository.NewHTTPWorkbenchRepository(pool)
	websocketWorkbenchRepository := repository.NewWebSocketWorkbenchRepository(pool)
	computeRepository := repository.NewComputeRepository(pool)

	accessSourceService := service.NewAccessSourceService(connectionRepository, mqttRepository, accessSourceRepository)
	alarmRuleService := service.NewAlarmRuleService(alarmRuleRepository, dataPointRepository)
	alarmPolicyService := service.NewAlarmPolicyService(alarmPolicyRepository, dataPointRepository)
	contractCheckService := service.NewContractCheckService(dataPointRepository, computeRepository, alarmRuleRepository, queryRepository, contractCheckRepository)
	builtinRuntimeService := newBuiltinRuntimeServiceFromConfig(cfg, pool, devPool, &cleanupFns)
	connectionService := service.NewConnectionService(connectionRepository, builtinRuntimeService)
	queryService := service.NewQueryService(queryRepository, connectionRepository, dataPointRepository)
	queryService.SetBuiltinRuntime(builtinRuntimeService)
	workbenchGroupService := service.NewWorkbenchGroupService(workbenchGroupRepository, connectionRepository)
	connectionService.SetWorkbenchGroupService(workbenchGroupService)
	dataPointService := service.NewDataPointService(dataPointRepository, queryService, mqttRepository, computeRepository)
	modbusModelingService := service.NewModbusModelingService(modbusModelingRepository, connectionRepository, dataPointRepository)
	mqttService := service.NewMqttService(mqttRepository, connectionRepository, dataPointRepository)
	opcuaModelingService := service.NewOpcuaModelingService(opcuaModelingRepository, connectionRepository, dataPointRepository)
	s7ModelingService := service.NewS7ModelingService(s7ModelingRepository, connectionRepository, dataPointRepository)
	protocolDevSessionService := service.NewProtocolDevSessionService(
		service.NewProtocolDevConnectionRepositoryAdapter(connectionRepository),
		service.NewProtocolDevOpcuaModelingAdapter(opcuaModelingService),
		service.NewProtocolDevModbusModelingAdapter(modbusModelingService),
		s7ModelingService,
	)
	projectSnapshotService := service.NewProjectSnapshotService(projectSnapshotRepository)
	protocolWave1Service := service.NewProtocolWave1Service(protocolWave1Repository)
	protocolPreviewService := service.NewProtocolPreviewService(protocolWave1Repository, service.NewDefaultProtocolPreviewAdapters())
	protocolWave2Service := service.NewProtocolWave2Service(protocolWave2Repository)
	kafkaWorkbenchService := service.NewKafkaWorkbenchService(kafkaWorkbenchRepository)
	httpWorkbenchService := service.NewHTTPWorkbenchService(httpWorkbenchRepository, connectionRepository)
	websocketWorkbenchService := service.NewWebSocketWorkbenchService(websocketWorkbenchRepository, connectionRepository)
	computeService := service.NewComputeService(
		computeRepository,
		enginecompute.NewNodeRunner("", ""),
		enginecompute.NewPythonRunner("", ""),
		enginecompute.NewScheduler(),
		dataPointRepository,
		queryService,
		mqttRepository,
	)

	accessSourceHandler := handler.NewAccessSourceHandler(accessSourceService)
	alarmRuleHandler := handler.NewAlarmRuleHandler(alarmRuleService)
	alarmPolicyHandler := handler.NewAlarmPolicyHandler(alarmPolicyService)
	contractCheckHandler := handler.NewContractCheckHandler(contractCheckService)
	connectionHandler := handler.NewConnectionHandler(connectionService)
	builtinRuntimeHandler := handler.NewBuiltinRuntimeHandler(builtinRuntimeService, connectionService)
	queryHandler := handler.NewQueryHandler(queryService)
	workbenchGroupHandler := handler.NewWorkbenchGroupHandler(workbenchGroupService)
	dataPointHandler := handler.NewDataPointHandler(dataPointService)
	modbusModelingHandler := handler.NewModbusModelingHandler(modbusModelingService)
	mqttHandler := handler.NewMqttHandler(mqttService)
	opcuaModelingHandler := handler.NewOpcuaModelingHandler(opcuaModelingService)
	s7ModelingHandler := handler.NewS7ModelingHandler(s7ModelingService)
	protocolDevSessionHandler := handler.NewProtocolDevSessionHandler(protocolDevSessionService)
	projectSnapshotHandler := handler.NewProjectSnapshotHandler(projectSnapshotService)
	protocolWave1Handler := handler.NewProtocolWave1Handler(protocolWave1Service, protocolPreviewService)
	protocolWave2Handler := handler.NewProtocolWave2Handler(protocolWave2Service)
	kafkaWorkbenchHandler := handler.NewKafkaWorkbenchHandler(kafkaWorkbenchService)
	httpWorkbenchHandler := handler.NewHTTPWorkbenchHandler(httpWorkbenchService)
	websocketWorkbenchHandler := handler.NewWebSocketWorkbenchHandler(websocketWorkbenchService)
	computeHandler := handler.NewComputeHandler(computeService)

	routeOptions := []router.Option{
		router.WithAlarmRuleRoutes(alarmRuleHandler, jwtValidator),
		router.WithAlarmPolicyRoutes(alarmPolicyHandler, jwtValidator),
		router.WithBuiltinRuntimeRoutes(builtinRuntimeHandler, jwtValidator),
		router.WithAccessSourceRoutes(accessSourceHandler, jwtValidator),
		router.WithContractCheckRoutes(contractCheckHandler, jwtValidator),
		router.WithConnectionRoutes(connectionHandler, jwtValidator),
		router.WithDataRoutes(queryHandler, dataPointHandler, jwtValidator),
		router.WithWorkbenchGroupRoutes(workbenchGroupHandler, jwtValidator),
		router.WithMqttRoutes(mqttHandler, jwtValidator),
		router.WithModbusModelingRoutes(modbusModelingHandler, jwtValidator),
		router.WithOpcuaModelingRoutes(opcuaModelingHandler, jwtValidator),
		router.WithS7ModelingRoutes(s7ModelingHandler, jwtValidator),
		router.WithProtocolDevSessionRoutes(protocolDevSessionHandler, jwtValidator),
		router.WithProjectSnapshotRoutes(projectSnapshotHandler, jwtValidator),
		router.WithProtocolWave1Routes(protocolWave1Handler, jwtValidator),
		router.WithProtocolWave2Routes(protocolWave2Handler, jwtValidator),
		router.WithKafkaWorkbenchRoutes(kafkaWorkbenchHandler, jwtValidator),
		router.WithHTTPWorkbenchRoutes(httpWorkbenchHandler, jwtValidator),
		router.WithWebSocketWorkbenchRoutes(websocketWorkbenchHandler, jwtValidator),
		router.WithComputeRoutes(computeHandler, jwtValidator),
	}
	routeSummaryParts := []string{
		"connections=enabled",
		"accessSources=enabled",
		"data=enabled",
		"builtinRuntime=enabled",
		"mqtt=enabled",
		"modbusModeling=enabled",
		"opcuaModeling=enabled",
		"s7Modeling=enabled",
		"protocolDevSession=enabled",
		"projectSnapshot=enabled",
		"protocolWave1=enabled",
		"protocolWave2=enabled",
		"kafkaWorkbench=enabled",
		"httpWorkbench=enabled",
		"websocketWorkbench=enabled",
		"compute=enabled",
		"preview=disabled",
	}

	if err := config.ValidatePreviewDependencies(cfg); err != nil {
		logf("warning: data_service preview 路由未启用 reason=%v", err)
	} else {
		redisClient, redisErr := cache.NewRedisClient(context.Background(), cache.RedisConfig{
			Addr:     strings.TrimSpace(cfg.RedisAddr),
			Password: strings.TrimSpace(cfg.RedisPassword),
			DB:       cfg.RedisDB,
		})
		if redisErr != nil {
			logf("warning: data_service preview 路由未启用 reason=%v", redisErr)
		} else {
			cleanupFns = append(cleanupFns, func() {
				_ = redisClient.Close()
			})

			previewRepository := repository.NewPreviewSessionRepository(pool)
			previewService := service.NewPreviewSessionService(previewRepository, redisClient)
			previewSocketServer, previewSocketErr := previewsocket.NewPreviewSocketServer(jwtValidator, previewService, dataPointService, mqttRepository, builtinRuntimeService)
			if previewSocketErr != nil {
				logf("warning: data_service preview socket 未启用 reason=%v", previewSocketErr)
			}
			if previewSocketServer != nil {
				cleanupFns = append(cleanupFns, previewSocketServer.Close)
				routeOptions = append(routeOptions, router.WithPreviewSocketHandler(previewSocketServer.Handler()))
				routeSummaryParts[len(routeSummaryParts)-1] = "preview=http+socket"
			}

			previewHandler := handler.NewPreviewHandler(previewService, previewSocketServer)
			routeOptions = append(routeOptions, router.WithPreviewRoutes(previewHandler, jwtValidator))
			if previewSocketServer == nil {
				routeSummaryParts[len(routeSummaryParts)-1] = "preview=http-only"
			}
		}
	}

	logf("info: data_service 路由摘要 routeSummary=%s", strings.Join(routeSummaryParts, ","))

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

func newBuiltinRuntimeServiceFromConfig(cfg config.Config, metaPool *pgxpool.Pool, devPool *pgxpool.Pool, cleanupFns *[]func()) *service.BuiltinRuntimeService {
	var realtimeClient redis.UniversalClient
	if strings.TrimSpace(cfg.RedisAddr) != "" {
		client := redis.NewClient(&redis.Options{
			Addr:     strings.TrimSpace(cfg.RedisAddr),
			Password: strings.TrimSpace(cfg.RedisPassword),
			DB:       cfg.RedisDevDB,
		})
		if err := client.Ping(context.Background()).Err(); err != nil {
			logf("warning: IF实时库开发态客户端未启用 reason=%v", err)
			_ = client.Close()
		} else {
			realtimeClient = client
			*cleanupFns = append(*cleanupFns, func() {
				_ = client.Close()
			})
		}
	}

	var messagePublisher service.BuiltinMessagePublisher
	if strings.TrimSpace(cfg.MessageHubAddr) != "" {
		publisher, err := service.NewPahoBuiltinMessagePublisher(cfg.MessageHubAddr, cfg.MessageHubUsername, cfg.MessageHubPassword)
		if err != nil {
			logf("warning: IF消息库开发态发布器未启用 reason=%v", err)
		} else {
			messagePublisher = publisher
			*cleanupFns = append(*cleanupFns, publisher.Close)
		}
	}

	return service.NewBuiltinRuntimeService(service.BuiltinRuntimeOptions{
		DevPool:            devPool,
		MetaPool:           metaPool,
		RealtimeClient:     realtimeClient,
		RealtimeKeyPrefix:  cfg.DevCacheKeyPrefix,
		MessagePublisher:   messagePublisher,
		MessageTopicPrefix: cfg.MessageTopicPrefix,
	})
}
