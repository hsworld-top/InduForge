package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/bootstrap"
	"github.com/indu-forge/data_service/internal/cache"
	"github.com/indu-forge/data_service/internal/collectorprotocol"
	"github.com/indu-forge/data_service/internal/config"
	"github.com/indu-forge/data_service/internal/db/postgres"
	enginecompute "github.com/indu-forge/data_service/internal/engine/compute"
	"github.com/indu-forge/data_service/internal/http/handler"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/router"
	previewsocket "github.com/indu-forge/data_service/internal/http/socket"
	"github.com/indu-forge/data_service/internal/repository"
	collectorsecurity "github.com/indu-forge/data_service/internal/security"
	"github.com/indu-forge/data_service/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	defaultReadHeaderTimeout = 5 * time.Second
	defaultReadTimeout       = 10 * time.Second
	// 写超时必须覆盖采集代理最长 25 秒的任务长轮询，并预留响应序列化与网络传输时间。
	defaultWriteTimeout = 30 * time.Second
	defaultIdleTimeout  = 60 * time.Second
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
	if err := validateDataServiceRequiredSchema(context.Background(), pool); err != nil {
		joinCleanup(cleanupFns...)()
		return nil, nil, err
	}

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

	connectionCipher, err := collectorsecurity.NewConnectionSecretCipher(cfg.ConnectionSecretKey, cfg.ConnectionSecretKeyVersion)
	if err != nil {
		joinCleanup(cleanupFns...)()
		return nil, nil, err
	}
	connectionRepository := repository.NewConnectionRepository(pool)
	connectionRepository.SetSecretCipher(connectionCipher)
	connectionSecretRepository := repository.NewConnectionSecretRepository(pool, connectionCipher)
	accessSourceRepository := repository.NewAccessSourceRepository(pool)
	alarmPolicyRepository := repository.NewAlarmPolicyRepository(pool)
	historyStorageRepository := repository.NewHistoryStorageRepository(pool)
	contractCheckRepository := repository.NewContractCheckRepository(pool)
	collectorDevRepository := repository.NewCollectorDevRepository(pool)
	collectorRepository := repository.NewCollectorRepository(pool)
	collectorCatalog, err := collectorprotocol.LoadCatalog(os.DirFS(config.ResolveCollectorProtocolCatalogPath(cfg.CollectorProtocolCatalogPath)))
	if err != nil {
		joinCleanup(cleanupFns...)()
		return nil, nil, err
	}
	queryRepository := repository.NewQueryRepository(pool)
	workbenchGroupRepository := repository.NewWorkbenchGroupRepository(pool)
	dataPointRepository := repository.NewDataPointRepository(pool)
	mqttRepository := repository.NewMqttRepository(pool, connectionCipher)
	projectSnapshotRepository := repository.NewProjectSnapshotRepository(pool)
	protocolWave1Repository := repository.NewProtocolWave1Repository(pool, connectionCipher)
	protocolWave2Repository := repository.NewProtocolWave2Repository(pool, connectionCipher)
	kafkaWorkbenchRepository := repository.NewKafkaWorkbenchRepository(pool)
	httpWorkbenchRepository := repository.NewHTTPWorkbenchRepository(pool, connectionCipher)
	websocketWorkbenchRepository := repository.NewWebSocketWorkbenchRepository(pool, connectionCipher)
	realtimeStoreRepository := repository.NewRealtimeStoreRepository(pool)
	computeRepository := repository.NewComputeRepository(pool)

	accessSourceService := service.NewAccessSourceService(connectionRepository, mqttRepository, accessSourceRepository)
	var alarmCipher *collectorsecurity.AlarmSecretCipher
	if len(cfg.AlarmSecretKey) == 32 {
		alarmCipher, err = collectorsecurity.NewAlarmSecretCipher(cfg.AlarmSecretKey, cfg.AlarmSecretKeyVersion)
		if err != nil {
			joinCleanup(cleanupFns...)()
			return nil, nil, err
		}
	}
	alarmPolicyService := service.NewAlarmPolicyService(alarmPolicyRepository, dataPointRepository, alarmCipher)
	alarmItemService := service.NewAlarmItemService(alarmPolicyRepository, dataPointRepository)
	alarmPolicyService.SetAlarmItemService(alarmItemService)
	historyStorageService := service.NewHistoryStorageService(historyStorageRepository)
	collectorDevService := service.NewCollectorDevService(collectorDevRepository)
	collectorCatalogService := service.NewCollectorCatalogService(collectorCatalog)
	var collectorHandler *handler.CollectorHandler
	var collectorPointHandler *handler.CollectorPointHandler
	var collectorImportHandler *handler.CollectorImportHandler
	if collectorKeyErr := config.ValidateCollectorSecretKey(cfg); collectorKeyErr != nil {
		if strings.EqualFold(strings.TrimSpace(os.Getenv("NODE_ENV")), "production") {
			joinCleanup(cleanupFns...)()
			return nil, nil, collectorKeyErr
		}
		logf("warn: data_service 启动阶段=collector-routes status=disabled reason=%v", collectorKeyErr)
	} else {
		secretCipher, cipherErr := collectorsecurity.NewCollectorSecretCipher(cfg.CollectorSecretKey, cfg.CollectorSecretKeyVersion)
		if cipherErr != nil {
			joinCleanup(cleanupFns...)()
			return nil, nil, cipherErr
		}
		collectorService, serviceErr := service.NewCollectorService(collectorRepository, collectorCatalog, secretCipher)
		if serviceErr != nil {
			joinCleanup(cleanupFns...)()
			return nil, nil, serviceErr
		}
		collectorHandler = handler.NewCollectorHandler(collectorService)
		collectorDevService.ConfigureTaskEnvelope(collectorRepository, secretCipher)
		collectorPointService, pointServiceErr := service.NewCollectorPointService(collectorRepository, collectorCatalog)
		if pointServiceErr != nil {
			joinCleanup(cleanupFns...)()
			return nil, nil, pointServiceErr
		}
		collectorPointHandler = handler.NewCollectorPointHandler(collectorPointService)
		collectorImportService, importServiceErr := service.NewCollectorImportService(collectorRepository, collectorPointService, collectorCatalog)
		if importServiceErr != nil {
			joinCleanup(cleanupFns...)()
			return nil, nil, importServiceErr
		}
		collectorImportHandler = handler.NewCollectorImportHandler(collectorImportService)
	}
	contractCheckService := service.NewContractCheckService(dataPointRepository, computeRepository, alarmPolicyRepository, queryRepository, contractCheckRepository)
	builtinRuntimeService := newBuiltinRuntimeServiceFromConfig(cfg, pool, devPool, &cleanupFns)
	connectionService := service.NewConnectionService(connectionRepository, builtinRuntimeService)
	connectionService.SetSecretRepository(connectionSecretRepository)
	queryService := service.NewQueryService(queryRepository, connectionRepository, dataPointRepository)
	queryService.SetBuiltinRuntime(builtinRuntimeService)
	queryService.SetSecretRepository(connectionSecretRepository)
	workbenchGroupService := service.NewWorkbenchGroupService(workbenchGroupRepository, connectionRepository)
	connectionService.SetWorkbenchGroupService(workbenchGroupService)
	dataPointService := service.NewDataPointService(dataPointRepository, queryService, mqttRepository, computeRepository)
	dataPointService.SetGeneratedSourceRepositories(kafkaWorkbenchRepository, httpWorkbenchRepository, websocketWorkbenchRepository, realtimeStoreRepository, connectionRepository, collectorRepository, builtinRuntimeService)
	mqttService := service.NewMqttService(mqttRepository, connectionRepository, dataPointRepository)
	mqttService.SetSecretRepository(connectionSecretRepository)
	mqttService.ConfigureBuiltinMessageHub(cfg.MessageHubAddr, cfg.MessageHubUsername, cfg.MessageHubPassword)
	projectSnapshotService := service.NewProjectSnapshotService(projectSnapshotRepository)
	protocolWave1Service := service.NewProtocolWave1Service(protocolWave1Repository)
	protocolPreviewService := service.NewProtocolPreviewService(protocolWave1Repository, service.NewDefaultProtocolPreviewAdapters())
	protocolPreviewService.SetSecretRepository(connectionSecretRepository)
	protocolWave2Service := service.NewProtocolWave2Service(protocolWave2Repository)
	kafkaWorkbenchService := service.NewKafkaWorkbenchService(kafkaWorkbenchRepository)
	httpWorkbenchService := service.NewHTTPWorkbenchService(httpWorkbenchRepository, connectionRepository)
	httpWorkbenchService.SetSecretRepository(connectionSecretRepository)
	websocketWorkbenchService := service.NewWebSocketWorkbenchService(websocketWorkbenchRepository, connectionRepository)
	websocketWorkbenchService.SetSecretRepository(connectionSecretRepository)
	realtimeStoreService := service.NewRealtimeStoreService(realtimeStoreRepository, connectionRepository, builtinRuntimeService)
	computeSandboxClient := enginecompute.NewSandboxClient(cfg.ComputeSandboxURL, cfg.ComputeSandboxToken)
	computeService := service.NewComputeService(
		computeRepository,
		computeSandboxClient.Runner("js"),
		computeSandboxClient.Runner("python"),
		dataPointRepository,
		queryService,
	)

	accessSourceHandler := handler.NewAccessSourceHandler(accessSourceService)
	alarmPolicyHandler := handler.NewAlarmPolicyHandler(alarmPolicyService, alarmItemService)
	historyStorageHandler := handler.NewHistoryStorageHandler(historyStorageService)
	contractCheckHandler := handler.NewContractCheckHandler(contractCheckService)
	collectorDevHandler := handler.NewCollectorDevHandler(collectorDevService)
	collectorCatalogHandler := handler.NewCollectorCatalogHandler(collectorCatalogService)
	connectionHandler := handler.NewConnectionHandler(connectionService)
	builtinRuntimeHandler := handler.NewBuiltinRuntimeHandler(builtinRuntimeService, connectionService)
	queryHandler := handler.NewQueryHandler(queryService)
	workbenchGroupHandler := handler.NewWorkbenchGroupHandler(workbenchGroupService)
	dataPointHandler := handler.NewDataPointHandler(dataPointService)
	mqttHandler := handler.NewMqttHandler(mqttService)
	projectSnapshotHandler := handler.NewProjectSnapshotHandler(projectSnapshotService)
	protocolWave1Handler := handler.NewProtocolWave1Handler(protocolWave1Service, protocolPreviewService)
	protocolWave2Handler := handler.NewProtocolWave2Handler(protocolWave2Service)
	kafkaWorkbenchHandler := handler.NewKafkaWorkbenchHandler(kafkaWorkbenchService)
	httpWorkbenchHandler := handler.NewHTTPWorkbenchHandler(httpWorkbenchService)
	websocketWorkbenchHandler := handler.NewWebSocketWorkbenchHandler(websocketWorkbenchService)
	realtimeStoreHandler := handler.NewRealtimeStoreHandler(realtimeStoreService)
	computeHandler := handler.NewComputeHandler(computeService)

	routeOptions := []router.Option{
		router.WithAlarmPolicyRoutes(alarmPolicyHandler, jwtValidator),
		router.WithHistoryStorageRoutes(historyStorageHandler, jwtValidator),
		router.WithBuiltinRuntimeRoutes(builtinRuntimeHandler, jwtValidator),
		router.WithAccessSourceRoutes(accessSourceHandler, jwtValidator),
		router.WithContractCheckRoutes(contractCheckHandler, jwtValidator),
		router.WithCollectorDevRoutes(collectorDevHandler, collectorDevService, jwtValidator),
		router.WithCollectorCatalogRoutes(collectorCatalogHandler, jwtValidator),
		router.WithConnectionRoutes(connectionHandler, jwtValidator),
		router.WithDataRoutes(queryHandler, dataPointHandler, jwtValidator),
		router.WithWorkbenchGroupRoutes(workbenchGroupHandler, jwtValidator),
		router.WithMqttRoutes(mqttHandler, jwtValidator),
		router.WithProjectSnapshotRoutes(projectSnapshotHandler, jwtValidator),
		router.WithProtocolWave1Routes(protocolWave1Handler, jwtValidator),
		router.WithProtocolWave2Routes(protocolWave2Handler, jwtValidator),
		router.WithKafkaWorkbenchRoutes(kafkaWorkbenchHandler, jwtValidator),
		router.WithHTTPWorkbenchRoutes(httpWorkbenchHandler, jwtValidator),
		router.WithWebSocketWorkbenchRoutes(websocketWorkbenchHandler, jwtValidator),
		router.WithRealtimeStoreRoutes(realtimeStoreHandler, jwtValidator),
		router.WithComputeRoutes(computeHandler, jwtValidator),
	}
	if collectorHandler != nil {
		routeOptions = append(routeOptions, router.WithCollectorRoutes(collectorHandler, jwtValidator))
		routeOptions = append(routeOptions, router.WithCollectorPointRoutes(collectorPointHandler, jwtValidator))
		routeOptions = append(routeOptions, router.WithCollectorImportRoutes(collectorImportHandler, jwtValidator))
	}
	routeSummaryParts := []string{
		"connections=enabled",
		"accessSources=enabled",
		"data=enabled",
		"builtinRuntime=enabled",
		"mqtt=enabled",
		"s7Modeling=enabled",
		"projectSnapshot=enabled",
		"protocolWave1=enabled",
		"protocolWave2=enabled",
		"kafkaWorkbench=enabled",
		"httpWorkbench=enabled",
		"websocketWorkbench=enabled",
		"realtimeStore=enabled",
		"compute=enabled",
		"historyStorage=enabled",
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
				previewSocketServer.ConfigureBuiltinMessageHub(cfg.MessageHubAddr, cfg.MessageHubUsername, cfg.MessageHubPassword)
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

func validateDataServiceRequiredSchema(ctx context.Context, pool *pgxpool.Pool) error {
	rows, err := pool.Query(ctx, `
		SELECT k.project_id, k.connection_id, k.provider, k.key_path,
		       k.redis_type, k.value_type, k.default_ttl_seconds, k.description,
		       dp.id, dp.path, k.created_at, k.updated_at
		FROM data_realtime_keys k
		LEFT JOIN data_points dp ON false
		LIMIT 0
	`)
	if err != nil {
		return fmt.Errorf("data_service 数据域库结构未初始化，缺少实时库工作台元数据结构: %w", err)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("data_service 数据域库结构校验失败: %w", err)
	}
	return nil
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

	return service.NewBuiltinRuntimeService(service.BuiltinRuntimeOptions{
		DevPool:           devPool,
		MetaPool:          metaPool,
		RealtimeClient:    realtimeClient,
		RealtimeKeyPrefix: cfg.DevCacheKeyPrefix,
	})
}
