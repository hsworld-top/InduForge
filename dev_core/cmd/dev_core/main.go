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
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/app"
	"github.com/indu-forge/dev_core/internal/auditlog"
	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/authoringsnapshot"
	"github.com/indu-forge/dev_core/internal/codeworkspace"
	"github.com/indu-forge/dev_core/internal/config"
	"github.com/indu-forge/dev_core/internal/contextpack"
	"github.com/indu-forge/dev_core/internal/controlplane"
	"github.com/indu-forge/dev_core/internal/dataservice"
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

// newCodeWorkspaceEngine 在 K3s 控制面中使用 Kubernetes API 创建工作区 Pod。
// 非 Kubernetes 环境保持原 Docker 路径，避免影响 Mac 本地开发。
func newCodeWorkspaceEngine(cfg config.Config) (codeworkspace.Engine, error) {
	engine := strings.ToLower(strings.TrimSpace(cfg.CodeWorkspaceEngine))
	if engine == "" && strings.TrimSpace(os.Getenv("KUBERNETES_SERVICE_HOST")) != "" {
		engine = "kubernetes"
	}
	switch engine {
	case "", "docker":
		return codeworkspace.NewDockerClient(cfg.CodeServerDockerHost)
	case "kubernetes", "k3s":
		return codeworkspace.NewKubernetesEngine(codeworkspace.KubernetesConfig{Namespace: cfg.CodeWorkspaceNamespace, WorkspaceRoot: cfg.CodeWorkspaceHostPath})
	default:
		return nil, fmt.Errorf("不支持的代码工作区引擎: %s", engine)
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("加载配置失败", "error", err)
		os.Exit(1)
	}
	dataServiceClient, err := dataservice.NewInternalClient(cfg.DataServiceURL, cfg.DataServiceInternalToken)
	if err != nil {
		logger.Error("初始化数据服务内部客户端失败", "error", err)
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
		if err := initializeDatabase(ctx, pool, cfg, true, dataServiceClient); err != nil {
			logger.Error("初始化控制面数据库失败", "error", err)
			os.Exit(1)
		}
		logger.Info("控制面数据库初始化完成")
		return
	}
	if err := initializeDatabase(ctx, pool, cfg, cfg.DBAutoSchemaSync, dataServiceClient); err != nil {
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
	authRepository := auth.NewPostgreSQLRepository(pool)
	authService := auth.NewService(
		authRepository,
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
	projectService := project.NewService(projectRepository, workspace, cfg.DefaultAdminPassword)
	projectService.SetTenantBindingEnsurer(dataServiceClient)
	controlPlane.SetProjectHandler(project.NewHandler(projectService, authService))
	workspaceEngine, err := newCodeWorkspaceEngine(cfg)
	if err != nil {
		logger.Error("初始化代码工作区引擎失败", "error", err)
		os.Exit(1)
	}
	codeWorkspaceService, err := codeworkspace.NewService(projectRepository, workspaceEngine, codeworkspace.Config{
		Image: cfg.CodeServerImage, BindHost: cfg.CodeServerBindHost, AllowedOrigins: cfg.CodeWorkspaceAllowedOrigins, VolumeName: cfg.CodeWorkspaceVolume,
		DefaultTemplateProjectID: platformdb.BuiltinDemoProjectID, DefaultTemplateID: "vite-vue-js",
	})
	if err != nil {
		logger.Error("初始化代码工作区服务失败", "error", err)
		os.Exit(1)
	}
	codeWorkspaceService.SetPresence(cacheStore)
	codeWorkspaceService.SetAuthoringEpochGuard(projectRepository)
	codeWorkspaceHandler := codeworkspace.NewHandler(codeWorkspaceService, authService)
	var codeWorkspaceGateway *codeworkspace.Gateway
	if cfg.WorkspacePublicOriginTemplate != "" {
		codeWorkspaceGateway, err = codeworkspace.NewGateway(codeWorkspaceService, codeworkspace.GatewayConfig{
			PublicOriginTemplate: cfg.WorkspacePublicOriginTemplate,
			CenterPublicOrigin:   cfg.CenterPublicOrigin,
			Namespace:            cfg.CodeWorkspaceNamespace,
			AllowedOrigins:       cfg.CodeWorkspaceAllowedOrigins,
			ResolveUser:          authService.GetActiveUser,
		})
		if err != nil {
			logger.Error("初始化代码工作区隔离网关失败", "error", err)
			os.Exit(1)
		}
		codeWorkspaceHandler.SetGateway(codeWorkspaceGateway)
	}
	controlPlane.SetCodeWorkspaceHandler(codeWorkspaceHandler)
	controlPlane.SetRuntimeAccessHandler(runtimeaccess.NewHandler(runtimeaccess.NewService(runtimeaccess.NewPostgreSQLRepository(pool)), authService))
	contextPackService := contextpack.NewService(project.NewService(projectRepository, workspace, cfg.DefaultAdminPassword), runtimeaccess.NewService(runtimeaccess.NewPostgreSQLRepository(pool)), workspace, cfg.DataServiceURL)
	contextPackHandler := contextpack.NewHandler(contextPackService)
	controlPlane.SetContextPackHandler(contextPackHandler)
	sceneAssetService := sceneasset.NewService(
		sceneasset.NewPostgreSQLRepository(pool), projectRepository, designObjects, cacheStore,
		sceneasset.NewRegistry(sceneasset.NewHTProvider()),
	)
	sceneAssetService.SetAuthoringEpochGuard(projectRepository)
	sceneAssetHandler := sceneasset.NewHandler(sceneAssetService, cfg.DataServiceURL)
	contextPackService.SetSceneContracts(sceneAssetService)
	realtimeServer := realtime.New(authService, logger)
	defer realtimeServer.Close()
	nodeService := node.NewService(node.NewPostgreSQLRepository(pool), cacheStore)
	nodeService.SetEvents(realtimeServer)
	nodeHandler := node.NewHandler(nodeService, authService)
	opsRepository := ops.NewPostgreSQLRepository(pool)
	opsRepository.SetEvents(realtimeServer)
	opsRepository.SetK3sAPIPort(cfg.OpsK3sAPIPort)
	if cfg.OpsCenterNodeID != "" {
		if err := opsRepository.EnsureRuntimeCluster(ctx, cfg.OpsCenterNodeID); err != nil {
			logger.Error("初始化中心运行集群失败", "nodeId", cfg.OpsCenterNodeID, "error", err)
			os.Exit(1)
		}
	}
	opsService := ops.NewService(opsRepository, ops.NewFilePackageStore(cfg.NodePackageDirectory), ifpObjects)
	opsService.SetClusterTokenKey([]byte(cfg.JWTSecret))
	opsHandler := ops.NewHandler(opsService, authService)
	opsHandler.SetEvents(realtimeServer)
	controlPlane.SetNodeHandler(nodeHandler)
	deploymentRepository := deployment.NewPostgreSQLRepository(pool)
	deploymentService := deployment.NewService(
		deploymentRepository, workspace, ifpObjects,
		deployment.ServiceConfig{ArtifactBucket: cfg.ObjectStoreIFPBucket, MinNodeAgentVersion: cfg.MinNodeAgentVersion, MinRuntimeVersion: cfg.MinRuntimeVersion},
	)
	if err := configureReleasePublishing(deploymentService, deploymentRepository, workspace, sceneAssetService, codeWorkspaceService, ifpObjects, cfg, dataServiceClient); err != nil {
		logger.Error("初始化正式 Release 发布失败", "error", err)
		os.Exit(1)
	}
	var restoreExecutor *deployment.RestoreExecutor
	if cfg.ReleaseBuilderEnabled && codeWorkspaceGateway != nil {
		keys, keyErr := authoringsnapshot.NewKeyring(cfg.AuthoringSnapshotCurrentKeyID, cfg.AuthoringSnapshotKeyring)
		if keyErr != nil {
			logger.Error("初始化开发态恢复密钥环失败", "error", keyErr)
			os.Exit(1)
		}
		runner, runnerErr := deployment.NewDockerFrontendBuildRunner(deployment.DockerFrontendBuildRunnerConfig{DockerHost: cfg.CodeServerDockerHost, Image: cfg.ReleaseBuilderImage, WorkspaceVolume: cfg.CodeWorkspaceVolume, WorkspaceRoot: cfg.WorkspaceRoot, BootstrapProjectID: platformdb.BuiltinDemoProjectID, BootstrapTemplateID: "vite-vue-js"})
		if runnerErr != nil {
			logger.Error("初始化开发态恢复构建输入失败", "error", runnerErr)
			os.Exit(1)
		}
		source, sourceErr := deployment.NewProjectReleaseSourceBuilder(deployment.ProjectReleaseSourceBuilderConfig{DataServiceURL: cfg.DataServiceURL, BuilderID: cfg.ReleaseBuilderID, TenantBindingEnsurer: dataServiceClient}, runner)
		if sourceErr != nil {
			logger.Error("初始化开发态恢复快照读取器失败", "error", sourceErr)
			os.Exit(1)
		}
		authoring := deployment.NewAuthoringReleaseBuilder(workspace, sceneAssetService, dataServiceClient, ifpObjects, keys, source)
		restoreExecutor = deployment.NewRestoreExecutor(deploymentRepository, authoring, workspace, sceneAssetService, dataServiceClient, deployment.NewAuthoringCaptureCoordinator(deploymentRepository, dataServiceClient, codeWorkspaceService), authService, logger)
		restoreExecutor.SetChangePublisher(realtimeServer)
		deploymentService.SetRestoreScheduler(restoreExecutor)
	}
	deploymentService.SetReleaseValidator(sceneAssetService)
	deploymentService.SetEvents(realtimeServer)
	opsService.SetDevelopmentArtifactBuilder(func(buildCtx context.Context, actor auth.User, projectID, authorization string) (ops.DevelopmentArtifact, error) {
		artifact, err := deploymentService.BuildDevelopmentArtifact(buildCtx, actor, projectID, authorization)
		if err != nil {
			return ops.DevelopmentArtifact{}, err
		}
		return ops.DevelopmentArtifact{ReleaseID: artifact.ReleaseID, Version: artifact.Version, Bucket: artifact.Bucket, ArtifactKey: artifact.ArtifactKey, ArtifactHash: artifact.ArtifactHash, ArtifactSize: artifact.ArtifactSize, Manifest: append([]byte(nil), artifact.Manifest...), ManifestHash: artifact.ManifestHash, ChecksumsHash: artifact.ChecksumsHash, SigningKeyID: artifact.SigningKeyID}, nil
	})
	opsService.SetDevelopmentRequirementsBuilder(func(buildCtx context.Context, actor auth.User, projectID, authorization string) ([]string, error) {
		return deploymentService.DevelopmentEngineRequirements(buildCtx, actor, projectID, authorization)
	})
	controlPlane.SetDeploymentHandler(deployment.NewHandler(deploymentService, authService))
	auditLogRepository := auditlog.NewPostgreSQLRepository(pool)
	controlPlane.SetAuditLogHandler(auditlog.NewHandler(auditlog.NewService(auditLogRepository), authService))
	application := app.New(app.Options{
		Logger:      logger,
		Middlewares: []func(http.Handler) http.Handler{project.ResolveAuthoringEpoch, auth.ResolveUser(authService), auditlog.Middleware(auditLogRepository, logger)},
		Mount: func(router chi.Router) {
			router.Handle("/control-socket.io", realtimeServer.Handler())
			router.Handle("/control-socket.io/*", realtimeServer.Handler())
			nodeHandler.MountAgentRoutes(router)
			opsHandler.MountRoutes(router)
			router.Route("/api/v1", sceneAssetHandler.MountRoutes)
			platformapi.HandlerFromMuxWithBaseURL(controlPlane, router, "/api/v1")
		},
	})
	var serverHandler http.Handler = application.Handler()
	if codeWorkspaceGateway != nil {
		serverHandler = codeWorkspaceGateway.Wrap(serverHandler)
	}
	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           serverHandler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	signalCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if restoreExecutor != nil {
		restoreExecutor.Start(signalCtx)
	}
	go worker.New(worker.NewPostgreSQLRepository(pool), ifpObjects, logger, 10*time.Second).Run(signalCtx)
	go runSceneObjectCleanup(signalCtx, sceneAssetService, logger)
	go runOpsLivenessReconciler(signalCtx, opsRepository, logger)
	if reconciler, reconcileErr := ops.NewInClusterProjectReconciler(); reconcileErr != nil {
		logger.Info("项目 K3s 调和器未启用", "reason", reconcileErr)
	} else {
		reconciler.SetHostNodeAddressLoader(opsRepository)
		opsService.SetFoundationNodePreflight(reconciler)
		reconciler.SetRuntimeContextLoader(opsRepository)
		reconciler.SetDeploymentSecretManager(ops.NewDeploymentSecretManager(reconciler))
		reconciler.SetCollectorBindingBundleClient(dataServiceClient)
		go runProjectWorkloadReconciler(signalCtx, opsRepository, reconciler, logger)
	}

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

// configureReleasePublishing 默认不注入构建器，使未启用环境的 Publish 保持失败关闭。
func configureReleasePublishing(service *deployment.Service, repository *deployment.PostgreSQLRepository, workspace *project.FileWorkspace, scenes *sceneasset.Service, codeWorkspaces *codeworkspace.Service, objects *objectstore.MinIO, cfg config.Config, dataServiceClient *dataservice.InternalClient) error {
	if !cfg.ReleaseBuilderEnabled {
		return nil
	}
	key, err := deployment.LoadEd25519SigningKey(cfg.ReleaseSigningKeyFile, cfg.ReleaseSigningKeyID)
	if err != nil {
		return err
	}
	runner, err := deployment.NewDockerFrontendBuildRunner(deployment.DockerFrontendBuildRunnerConfig{DockerHost: cfg.CodeServerDockerHost, Image: cfg.ReleaseBuilderImage, WorkspaceVolume: cfg.CodeWorkspaceVolume, WorkspaceRoot: cfg.WorkspaceRoot, BootstrapProjectID: platformdb.BuiltinDemoProjectID, BootstrapTemplateID: "vite-vue-js"})
	if err != nil {
		return err
	}
	builder, err := deployment.NewProjectReleaseSourceBuilder(deployment.ProjectReleaseSourceBuilderConfig{DataServiceURL: cfg.DataServiceURL, BuilderID: cfg.ReleaseBuilderID, TenantBindingEnsurer: dataServiceClient}, runner)
	if err != nil {
		return err
	}
	service.SetReleaseSourceBuilder(builder)
	keys, err := authoringsnapshot.NewKeyring(cfg.AuthoringSnapshotCurrentKeyID, cfg.AuthoringSnapshotKeyring)
	if err != nil {
		return err
	}
	service.SetFormalAuthoringBuilder(deployment.NewAuthoringReleaseBuilder(workspace, scenes, dataServiceClient, objects, keys, builder))
	service.SetAuthoringCaptureCoordinator(deployment.NewAuthoringCaptureCoordinator(repository, dataServiceClient, codeWorkspaces))
	service.SetSigningConfig(key)
	return nil
}

func runOpsLivenessReconciler(ctx context.Context, repository *ops.PostgreSQLRepository, logger *slog.Logger) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := repository.ReconcileNodeLiveness(ctx); err != nil {
				logger.Warn("更新运维节点在线状态失败", "error", err)
			}
			if err := repository.ReconcileFoundationFreshness(ctx); err != nil {
				logger.Warn("更新基础服务观测时效失败", "error", err)
			}
		}
	}
}

func runProjectWorkloadReconciler(ctx context.Context, repository *ops.PostgreSQLRepository, reconciler *ops.KubernetesProjectReconciler, logger *slog.Logger) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := repository.ReconcilePendingProjectWorkloads(ctx, reconciler); err != nil {
				logger.Warn("调和项目 K3s 工作负载失败", "error", err)
			}
			if _, err := repository.ReconcileStoppedProjectDeployments(ctx, reconciler); err != nil {
				logger.Warn("停止项目 K3s 工作负载失败", "error", err)
			}
		}
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
func initializeDatabase(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, allowCreate bool, dataServiceClient *dataservice.InternalClient) error {
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
	if err := syncBuiltinDemoProjectTenantBinding(ctx, pool, dataServiceClient); err != nil {
		return fmt.Errorf("同步内置教程工程项目租户绑定失败，可稍后重试: %w", err)
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

// builtinProjectTenantBindingEnsurer 将控制面已提交的工程归属同步至数据域。
// Bootstrap 可能在工程恢复、默认租户配置调整后再次执行，因此必须以控制库的
// tenant_id 与 authoring_epoch 为事实源，不能重新使用启动配置推导历史工程归属。
type builtinProjectTenantBindingEnsurer interface {
	EnsureProjectTenantBindingAtEpoch(context.Context, string, string, string) error
}

func syncBuiltinDemoProjectTenantBinding(ctx context.Context, pool *pgxpool.Pool, ensurer builtinProjectTenantBindingEnsurer) error {
	if ensurer == nil {
		return fmt.Errorf("数据服务项目租户绑定客户端未配置")
	}
	var tenantID string
	var authoringEpoch int64
	if err := pool.QueryRow(ctx, `SELECT tenant_id,authoring_epoch FROM projects WHERE id=$1 AND status<>'deleted'`, platformdb.BuiltinDemoProjectID).Scan(&tenantID, &authoringEpoch); err != nil {
		return fmt.Errorf("读取内置教程工程项目归属失败: %w", err)
	}
	return ensureBuiltinProjectTenantBinding(ctx, ensurer, tenantID, authoringEpoch)
}

func ensureBuiltinProjectTenantBinding(ctx context.Context, ensurer builtinProjectTenantBindingEnsurer, tenantID string, authoringEpoch int64) error {
	if authoringEpoch < 1 {
		return fmt.Errorf("内置教程工程开发态代次无效")
	}
	if err := ensurer.EnsureProjectTenantBindingAtEpoch(ctx, platformdb.BuiltinDemoProjectID, tenantID, project.FormatAuthoringEpoch(authoringEpoch)); err != nil {
		return err
	}
	return nil
}
