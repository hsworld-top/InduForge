package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/indu-forge/data_service/internal/config"
	"github.com/indu-forge/data_service/internal/db/postgres"
	"github.com/indu-forge/data_service/internal/db/schema"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	adminDatabaseName         = "postgres"
	sqlStateDuplicateDatabase = "42P04"
	sqlStateMissingDatabase   = "3D000"
	sqlStateInsufficientPriv  = "42501"
)

var logf = log.Printf

// RuntimeBootstrapper 负责在服务正式装配前完成数据库自举。
// 关键流程固定为：连接目标库 -> 缺库时自动建库 -> 初始化数据库结构。
// 边界上只处理数据库和表结构初始化，不负责创建用户或修复权限。
type RuntimeBootstrapper struct {
	openPool          poolOpener
	openAdminPool     adminPoolOpener
	createInitializer initializerFactory
}

type poolHandle interface {
	Close()
}

type adminExecutor interface {
	Exec(ctx context.Context, sql string) error
	Close()
}

type schemaInitializer interface {
	Ensure(ctx context.Context) error
}

type poolOpener func(ctx context.Context, databaseURL, searchPath string) (poolHandle, error)
type adminPoolOpener func(ctx context.Context, databaseURL string) (adminExecutor, error)
type initializerFactory func(pool poolHandle) (schemaInitializer, error)
type runtimeBootstrapperOption func(*RuntimeBootstrapper)

// NewRuntimeBootstrapper 创建生产环境默认使用的数据库自举器。
func NewRuntimeBootstrapper(options ...runtimeBootstrapperOption) *RuntimeBootstrapper {
	bootstrapper := &RuntimeBootstrapper{
		openPool:          defaultPoolOpener,
		openAdminPool:     defaultAdminPoolOpener,
		createInitializer: defaultInitializerFactory,
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		option(bootstrapper)
	}
	return bootstrapper
}

func withPoolOpener(opener poolOpener) runtimeBootstrapperOption {
	return func(bootstrapper *RuntimeBootstrapper) {
		bootstrapper.openPool = opener
	}
}

func withAdminPoolOpener(opener adminPoolOpener) runtimeBootstrapperOption {
	return func(bootstrapper *RuntimeBootstrapper) {
		bootstrapper.openAdminPool = opener
	}
}

func withInitializerFactory(factory initializerFactory) runtimeBootstrapperOption {
	return func(bootstrapper *RuntimeBootstrapper) {
		bootstrapper.createInitializer = factory
	}
}

// EnsureReady 保证目标数据库已存在且空数据库已完成结构初始化。
// 任一步失败都会直接返回错误，调用方应终止服务启动，避免降级成空路由实例。
func (b *RuntimeBootstrapper) EnsureReady(ctx context.Context, cfg config.Config) error {
	targetPool, err := b.openPool(ctx, strings.TrimSpace(cfg.DatabaseURL), strings.TrimSpace(cfg.DatabaseSearchPath))
	if err == nil {
		logf("info: 目标数据库连接成功，开始执行启动自举")
		return initializeSchema(ctx, targetPool, b.createInitializer)
	}
	if !isMissingDatabaseError(err) {
		return fmt.Errorf("初始化目标数据库连接失败: %w", err)
	}
	logf("warning: 目标数据库不存在，准备自动创建")

	adminDatabaseURL, databaseName, parseErr := deriveAdminDatabaseURL(strings.TrimSpace(cfg.DatabaseURL))
	if parseErr != nil {
		return parseErr
	}

	adminPool, adminErr := b.openAdminPool(ctx, adminDatabaseURL)
	if adminErr != nil {
		return fmt.Errorf("连接 PostgreSQL 维护库失败: %w", adminErr)
	}
	defer adminPool.Close()

	createErr := adminPool.Exec(ctx, buildCreateDatabaseSQL(databaseName))
	if createErr != nil && !isDuplicateDatabaseError(createErr) {
		if isInsufficientPrivilegeError(createErr) {
			return fmt.Errorf("自动创建数据库 %q 失败: 当前账号缺少 CREATE DATABASE 权限: %w", databaseName, createErr)
		}
		return fmt.Errorf("自动创建数据库 %q 失败: %w", databaseName, createErr)
	}
	logf("info: 自动创建数据库成功 database=%s", databaseName)

	targetPool, err = b.openPool(ctx, strings.TrimSpace(cfg.DatabaseURL), strings.TrimSpace(cfg.DatabaseSearchPath))
	if err != nil {
		return fmt.Errorf("创建数据库 %q 后重新连接失败: %w", databaseName, err)
	}

	return initializeSchema(ctx, targetPool, b.createInitializer)
}

func initializeSchema(ctx context.Context, pool poolHandle, factory initializerFactory) error {
	defer pool.Close()

	logf("info: 开始检查数据库结构")
	runner, err := factory(pool)
	if err != nil {
		return fmt.Errorf("创建数据库结构初始化器失败: %w", err)
	}
	if err := runner.Ensure(ctx); err != nil {
		return fmt.Errorf("初始化数据库结构失败: %w", err)
	}
	logf("info: 数据库结构检查完成")
	return nil
}

func defaultPoolOpener(ctx context.Context, databaseURL, searchPath string) (poolHandle, error) {
	pool, err := postgres.NewPool(ctx, postgres.PoolConfig{
		DatabaseURL: databaseURL,
		SearchPath:  searchPath,
	})
	if err != nil {
		return nil, err
	}
	return &pgxPoolHandle{pool: pool}, nil
}

func defaultAdminPoolOpener(ctx context.Context, databaseURL string) (adminExecutor, error) {
	pool, err := postgres.NewPoolFromURL(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	return &pgxAdminPool{pool: pool}, nil
}

func defaultInitializerFactory(pool poolHandle) (schemaInitializer, error) {
	pgxPool, ok := unwrapPGXPool(pool)
	if !ok {
		return nil, fmt.Errorf("不支持的 pool 类型 %T", pool)
	}
	return schema.NewInitializer(pgxPool)
}

func deriveAdminDatabaseURL(databaseURL string) (string, string, error) {
	if parsedURL, ok, err := parseDatabaseURL(databaseURL); err != nil {
		return "", "", err
	} else if ok {
		databaseName := strings.TrimPrefix(parsedURL.Path, "/")
		if databaseName == "" {
			return "", "", fmt.Errorf("DATA_SERVICE_DATABASE_URL 缺少数据库名")
		}

		parsedURL.Path = "/" + adminDatabaseName
		parsedURL.RawPath = ""
		return parsedURL.String(), databaseName, nil
	}

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return "", "", fmt.Errorf("解析 DATA_SERVICE_DATABASE_URL 失败: %w", err)
	}

	databaseName := strings.TrimSpace(poolConfig.ConnConfig.Database)
	if databaseName == "" {
		return "", "", fmt.Errorf("DATA_SERVICE_DATABASE_URL 缺少数据库名")
	}

	poolConfig.ConnConfig.Database = adminDatabaseName
	return poolConfig.ConnString(), databaseName, nil
}

func parseDatabaseURL(databaseURL string) (*url.URL, bool, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(databaseURL))
	if err != nil {
		return nil, false, fmt.Errorf("解析 DATA_SERVICE_DATABASE_URL 失败: %w", err)
	}
	if parsedURL == nil || parsedURL.Scheme == "" {
		return nil, false, nil
	}
	return parsedURL, true, nil
}

func buildCreateDatabaseSQL(databaseName string) string {
	return fmt.Sprintf(`CREATE DATABASE "%s"`, strings.ReplaceAll(databaseName, `"`, `""`))
}

func unwrapPGXPool(pool poolHandle) (*pgxpool.Pool, bool) {
	typedPool, ok := pool.(*pgxPoolHandle)
	if !ok || typedPool == nil || typedPool.pool == nil {
		return nil, false
	}
	return typedPool.pool, true
}

func isMissingDatabaseError(err error) bool {
	if hasSQLState(err, sqlStateMissingDatabase) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "database does not exist")
}

func isDuplicateDatabaseError(err error) bool {
	if hasSQLState(err, sqlStateDuplicateDatabase) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "already exists")
}

func isInsufficientPrivilegeError(err error) bool {
	if hasSQLState(err, sqlStateInsufficientPriv) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "permission denied") || strings.Contains(message, "insufficient privilege")
}

func hasSQLState(err error, code string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == code
}

type pgxPoolHandle struct {
	pool *pgxpool.Pool
}

func (p *pgxPoolHandle) Close() {
	if p == nil || p.pool == nil {
		return
	}
	p.pool.Close()
}

type pgxAdminPool struct {
	pool *pgxpool.Pool
}

func (p *pgxAdminPool) Exec(ctx context.Context, sql string) error {
	if p == nil || p.pool == nil {
		return fmt.Errorf("维护库连接池未初始化")
	}
	_, err := p.pool.Exec(ctx, sql)
	return err
}

func (p *pgxAdminPool) Close() {
	if p == nil || p.pool == nil {
		return
	}
	p.pool.Close()
}
