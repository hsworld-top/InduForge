package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/app"
	"github.com/indu-forge/dev_core/internal/auditlog"
	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/codeworkspace"
	"github.com/indu-forge/dev_core/internal/config"
	"github.com/indu-forge/dev_core/internal/contextpack"
	"github.com/indu-forge/dev_core/internal/controlplane"
	"github.com/indu-forge/dev_core/internal/deployment"
	"github.com/indu-forge/dev_core/internal/node"
	"github.com/indu-forge/dev_core/internal/objectstore"
	"github.com/indu-forge/dev_core/internal/ops"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
	platformcache "github.com/indu-forge/dev_core/internal/platform/cache"
	platformdb "github.com/indu-forge/dev_core/internal/platform/db"
	"github.com/indu-forge/dev_core/internal/project"
	"github.com/indu-forge/dev_core/internal/realtime"
	"github.com/indu-forge/dev_core/internal/runtimeaccess"
	"github.com/indu-forge/dev_core/internal/sceneasset"
	"github.com/indu-forge/dev_core/internal/tenant"
	"github.com/indu-forge/dev_core/internal/user"
	"github.com/indu-forge/dev_core/internal/worker"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("加载配置失败", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := platformdb.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("连接控制面数据库失败", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	if len(os.Args) > 1 {
		if os.Args[1] != "init" {
			logger.Error("不支持的启动参数", "argument", os.Args[1])
			os.Exit(1)
		}
		if err := initializeDatabase(ctx, pool, cfg, true); err != nil {
			logger.Error("初始化控制面数据库失败", "error", err)
			os.Exit(1)
		}
		logger.Info("控制面数据库初始化完成")
		return
	}
	if err := initializeDatabase(ctx, pool, cfg, cfg.DBAutoSchemaSync); err != nil {
		logger.Error("校验控制面数据库失败", "error", err)
		os.Exit(1)
	}
	cacheStore, err := platformcache.NewRedis(ctx, platformcache.RedisConfig{Address: cfg.CacheAddress, Password: cfg.CachePassword, DB: cfg.CacheDB})
	if err != nil {
		logger.Error("初始化 Redis 缓存失败", "error", err)
		os.Exit(1)
	}
	defer cacheStore.Close()
	tokenManager, err := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	if err != nil {
		logger.Error("初始化认证令牌失败", "error", err)
		os.Exit(1)
	}
	authService := auth.NewService(
		auth.NewPostgreSQLRepository(pool),
		cacheStore,
		tokenManager,
		auth.ServiceConfig{AppName: cfg.AppName},
		cacheStore,
	)
	designObjects, err := objectstore.NewMinIO(ctx, objectstore.Config{Endpoint: cfg.ObjectStoreEndpoint, AccessKey: cfg.ObjectStoreAccessKey, SecretKey: cfg.ObjectStoreSecretKey, Bucket: cfg.ObjectStoreDesignBucket, Region: cfg.ObjectStoreRegion, UseSSL: cfg.ObjectStoreUseSSL})
	if err != nil {
		logger.Error("初始化设计资源对象存储失败", "error", err)
		os.Exit(1)
	}
	authService.SetAssetSigner(designObjects)
	ifpObjects, err := objectstore.NewMinIO(ctx, objectstore.Config{Endpoint: cfg.ObjectStoreEndpoint, AccessKey: cfg.ObjectStoreAccessKey, SecretKey: cfg.ObjectStoreSecretKey, Bucket: cfg.ObjectStoreIFPBucket, Region: cfg.ObjectStoreRegion, UseSSL: cfg.ObjectStoreUseSSL})
	if err != nil {
		logger.Error("初始化发布工件对象存储失败", "error", err)
		os.Exit(1)
	}
	tenantService := tenant.NewService(tenant.NewPostgreSQLRepository(pool), designObjects, tenant.ServiceConfig{DefaultAdminUsername: cfg.DefaultAdminUsername, DefaultAdminPassword: cfg.DefaultAdminPassword})
	controlPlane := controlplane.NewHandler(auth.NewHandler(authService), tenant.NewHandler(tenantService, authService))
	controlPlane.SetUserHandler(user.NewHandler(user.NewService(user.NewPostgreSQLRepository(pool), authService), authService))
	workspace, err := project.NewFileWorkspace(cfg.WorkspaceRoot)
	if err != nil {
		logger.Error("初始化工程工作空间失败", "error", err)
		os.Exit(1)
	}
	projectRepository := project.NewPostgreSQLRepository(pool)
	controlPlane.SetProjectHandler(project.NewHandler(project.NewService(projectRepository, workspace, cfg.DefaultAdminPassword), authService))
	dockerClient, err := codeworkspace.NewDockerClient(cfg.CodeServerDockerHost)
	if err != nil {
		logger.Error("初始化 Docker Engine 客户端失败", "error", err)
		os.Exit(1)
	}
	codeWorkspaceService, err := codeworkspace.NewService(projectRepository, dockerClient, codeworkspace.Config{
		Image: cfg.CodeServerImage, BindHost: cfg.CodeServerBindHost, VolumeName: cfg.CodeWorkspaceVolume,
	})
	if err != nil {
		logger.Error("初始化代码工作区服务失败", "error", err)
		os.Exit(1)
	}
	codeWorkspaceService.SetPresence(cacheStore)
	controlPlane.SetCodeWorkspaceHandler(codeworkspace.NewHandler(codeWorkspaceService, authService))
	controlPlane.SetRuntimeAccessHandler(runtimeaccess.NewHandler(runtimeaccess.NewService(runtimeaccess.NewPostgreSQLRepository(pool)), authService))
	contextPackService := contextpack.NewService(project.NewService(projectRepository, workspace, cfg.DefaultAdminPassword), runtimeaccess.NewService(runtimeaccess.NewPostgreSQLRepository(pool)), workspace, cfg.DataServiceURL)
	contextPackHandler := contextpack.NewHandler(contextPackService)
	controlPlane.SetContextPackHandler(contextPackHandler)
	sceneAssetService := sceneasset.NewService(
		sceneasset.NewPostgreSQLRepository(pool), projectRepository, designObjects, cacheStore,
		sceneasset.NewRegistry(sceneasset.NewHTProvider()),
	)
	sceneAssetHandler := sceneasset.NewHandler(sceneAssetService, cfg.DataServiceURL)
	contextPackService.SetSceneContracts(sceneAssetService)
	realtimeServer := realtime.New(authService, logger)
	defer realtimeServer.Close()
	nodeService := node.NewService(node.NewPostgreSQLRepository(pool), cacheStore)
	nodeService.SetEvents(realtimeServer)
	nodeHandler := node.NewHandler(nodeService, authService)
	opsHandler := ops.NewHandler(ops.NewService(ops.NewPostgreSQLRepository(pool), ops.NewFilePackageStore(cfg.NodePackageDirectory)), authService)
	controlPlane.SetNodeHandler(nodeHandler)
	deploymentService := deployment.NewService(
		deployment.NewPostgreSQLRepository(pool), workspace, ifpObjects,
		deployment.ServiceConfig{ArtifactBucket: cfg.ObjectStoreIFPBucket},
	)
	deploymentService.SetReleaseValidator(sceneAssetService)
	deploymentService.SetEvents(realtimeServer)
	controlPlane.SetDeploymentHandler(deployment.NewHandler(deploymentService, authService))
	auditLogRepository := auditlog.NewPostgreSQLRepository(pool)
	controlPlane.SetAuditLogHandler(auditlog.NewHandler(auditlog.NewService(auditLogRepository), authService))
	application := app.New(app.Options{
		Logger:      logger,
		Middlewares: []func(http.Handler) http.Handler{auth.ResolveUser(authService), auditlog.Middleware(auditLogRepository, logger)},
		Mount: func(router chi.Router) {
			router.Handle("/control-socket.io", realtimeServer.Handler())
			router.Handle("/control-socket.io/*", realtimeServer.Handler())
			nodeHandler.MountAgentRoutes(router)
			opsHandler.MountRoutes(router)
			router.Route("/api/v1", sceneAssetHandler.MountRoutes)
			platformapi.HandlerFromMuxWithBaseURL(controlPlane, router, "/api/v1")
		},
	})
	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           application.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	signalCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go worker.New(worker.NewPostgreSQLRepository(pool), ifpObjects, logger, 10*time.Second).Run(signalCtx)
	go runSceneObjectCleanup(signalCtx, sceneAssetService, logger)

	go func() {
		logger.Info("dev_core 已启动", "addr", cfg.Addr)
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			logger.Error("HTTP 服务异常退出", "error", serveErr)
			stop()
		}
	}()

	<-signalCtx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP 服务关闭失败", "error", err)
	}
}

func runSceneObjectCleanup(ctx context.Context, service *sceneasset.Service, logger *slog.Logger) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := service.CleanupOrphans(ctx, time.Hour); err != nil {
				logger.Warn("清理场景孤立对象失败", "error", err)
			}
		}
	}
}

// initializeDatabase 只允许空库创建最终基线；已有业务表时仅校验，不执行迁移或修补。
func initializeDatabase(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, allowCreate bool) error {
	if err := platformdb.EnsureSchema(ctx, pool, allowCreate); err != nil {
		return err
	}
	superAdminPasswordHash, err := auth.HashPassword(cfg.SuperAdminPassword)
	if err != nil {
		return err
	}
	defaultAdminPasswordHash, err := auth.HashPassword(cfg.DefaultAdminPassword)
	if err != nil {
		return err
	}
	workspaceRoot, err := filepath.Abs(cfg.WorkspaceRoot)
	if err != nil {
		return fmt.Errorf("解析工程工作区根目录失败: %w", err)
	}
	demoWorkspacePath := filepath.Join(workspaceRoot, platformdb.BuiltinDemoProjectID, "workspace")
	if err := platformdb.EnsureInitialData(ctx, pool, platformdb.SeedConfig{
		TenantID: cfg.DefaultTenantID, TenantName: cfg.AppName, TenantCode: cfg.DefaultTenantCode,
		SuperAdminUserID: cfg.SuperAdminUserID, SuperAdminUsername: cfg.SuperAdminUsername, SuperAdminPasswordHash: superAdminPasswordHash,
		DefaultAdminUsername: cfg.DefaultAdminUsername, DefaultAdminPasswordHash: defaultAdminPasswordHash,
		DemoWorkspacePath: demoWorkspacePath,
	}); err != nil {
		return err
	}
	if err := platformdb.EnsureBuiltinDemoProject(ctx, pool, platformdb.SeedConfig{
		TenantID: cfg.DefaultTenantID, DefaultAdminUsername: cfg.DefaultAdminUsername,
		DefaultAdminPasswordHash: defaultAdminPasswordHash, DemoWorkspacePath: demoWorkspacePath,
	}); err != nil {
		return err
	}
	var demoProjectActive bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM projects WHERE id=$1 AND status='active')`, platformdb.BuiltinDemoProjectID).Scan(&demoProjectActive); err != nil {
		return fmt.Errorf("检查内置教程工程失败: %w", err)
	}
	if demoProjectActive {
		demoWorkspace, err := project.NewFileWorkspace(workspaceRoot)
		if err != nil {
			return fmt.Errorf("准备内置教程工程工作区失败: %w", err)
		}
		initializedPath, err := demoWorkspace.Initialize(platformdb.BuiltinDemoProjectID)
		if err != nil {
			return fmt.Errorf("初始化内置教程工程工作区失败: %w", err)
		}
		if filepath.Clean(initializedPath) != filepath.Clean(demoWorkspacePath) {
			return fmt.Errorf("内置教程工程工作区路径不一致")
		}
	}
	return nil
}
