package config

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_UsesDefaultAddr(t *testing.T) {
	t.Setenv("DEV_CORE_URL", "http://center.test")
	t.Setenv("DATA_SERVICE_ADDR", "")
	t.Setenv("DATA_SERVICE_INTERNAL_TOKEN", "test-internal-token")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.Addr != defaultAddr {
		t.Fatalf("expected default addr %q, got %q", defaultAddr, cfg.Addr)
	}
}

func TestLoad_ReadsCollectorSecretKey(t *testing.T) {
	t.Setenv("DEV_CORE_URL", "http://center.test")
	t.Setenv("DATA_SERVICE_INTERNAL_TOKEN", "test-internal-token")
	encoded := base64.StdEncoding.EncodeToString(make([]byte, 32))
	t.Setenv("DATA_SERVICE_COLLECTOR_SECRET_KEY", encoded)
	t.Setenv("DATA_SERVICE_COLLECTOR_SECRET_KEY_VERSION", "v2")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.CollectorSecretKey) != 32 || cfg.CollectorSecretKeyVersion != "v2" {
		t.Fatalf("unexpected collector secret config: length=%d version=%q", len(cfg.CollectorSecretKey), cfg.CollectorSecretKeyVersion)
	}
}

func TestLoad_RejectsInvalidCollectorSecretKey(t *testing.T) {
	t.Setenv("DEV_CORE_URL", "http://center.test")
	t.Setenv("DATA_SERVICE_INTERNAL_TOKEN", "test-internal-token")
	t.Setenv("DATA_SERVICE_COLLECTOR_SECRET_KEY", base64.StdEncoding.EncodeToString([]byte("short")))

	if _, err := Load(); err == nil {
		t.Fatal("expected invalid collector secret key to be rejected")
	}
}

func TestLoad_ReadsIndependentAlarmSecretKey(t *testing.T) {
	t.Setenv("DEV_CORE_URL", "http://center.test")
	t.Setenv("DATA_SERVICE_INTERNAL_TOKEN", "test-internal-token")
	encoded := base64.StdEncoding.EncodeToString(make([]byte, 32))
	t.Setenv("DATA_SERVICE_ALARM_SECRET_KEY", encoded)
	t.Setenv("DATA_SERVICE_ALARM_SECRET_KEY_VERSION", "alarm-v2")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.AlarmSecretKey) != 32 || cfg.AlarmSecretKeyVersion != "alarm-v2" {
		t.Fatalf("unexpected alarm secret config: length=%d version=%q", len(cfg.AlarmSecretKey), cfg.AlarmSecretKeyVersion)
	}
}

func TestLoad_RejectsInvalidAlarmSecretKey(t *testing.T) {
	t.Setenv("DEV_CORE_URL", "http://center.test")
	t.Setenv("DATA_SERVICE_INTERNAL_TOKEN", "test-internal-token")
	t.Setenv("DATA_SERVICE_ALARM_SECRET_KEY", base64.StdEncoding.EncodeToString([]byte("short")))

	if _, err := Load(); err == nil {
		t.Fatal("expected invalid alarm secret key to be rejected")
	}
}

func TestValidateCollectorSecretKeyRejectsMissingKey(t *testing.T) {
	t.Setenv("DEV_CORE_URL", "http://center.test")
	err := ValidateCollectorSecretKey(Config{CollectorSecretKeyVersion: "v1"})
	if err == nil {
		t.Fatal("expected missing collector secret key to be rejected")
	}
}

func TestLoad_RejectsInvalidAddr(t *testing.T) {
	t.Setenv("DEV_CORE_URL", "http://center.test")
	t.Setenv("DATA_SERVICE_INTERNAL_TOKEN", "test-internal-token")
	t.Setenv("DATA_SERVICE_ADDR", "invalid-addr")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoad_ReadsOptionalDependencyConfig(t *testing.T) {
	t.Setenv("DEV_CORE_URL", "http://center.test")
	t.Setenv("DATA_SERVICE_INTERNAL_TOKEN", "test-internal-token")
	t.Setenv("DATA_SERVICE_ADDR", ":18102")
	t.Setenv("DATA_SERVICE_DATABASE_URL", "")
	t.Setenv("DATA_SERVICE_DATABASE_SCHEMA", "")
	t.Setenv("DATA_SERVICE_REDIS_ADDR", "")
	t.Setenv("DATA_SERVICE_REDIS_PASSWORD", "")
	t.Setenv("DATA_SERVICE_REDIS_DB", "")
	t.Setenv("IF_META_STORE_HOST", "meta-store")
	t.Setenv("IF_META_STORE_PORT", "18432")
	t.Setenv("IF_META_STORE_USER", "induforge")
	t.Setenv("IF_META_STORE_PASSWORD", "secret")
	t.Setenv("IF_META_STORE_DATA_DB", "if_data")
	t.Setenv("IF_META_STORE_SSL", "false")
	t.Setenv("IF_META_STORE_DATA_SCHEMA", "tenant_a")
	t.Setenv("JWT_ACCESS_SECRET", "shared-access-secret")
	t.Setenv("IF_CACHE_STORE_HOST", "cache-store")
	t.Setenv("IF_CACHE_STORE_PORT", "18379")
	t.Setenv("IF_CACHE_STORE_PASSWORD", "cache-pass")
	t.Setenv("IF_CACHE_STORE_DATA_DB", "2")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.DatabaseURL != "postgres://induforge:secret@meta-store:18432/if_data?sslmode=disable" {
		t.Fatalf("expected database url to be loaded, got %q", cfg.DatabaseURL)
	}
	if cfg.DatabaseSearchPath != "tenant_a" {
		t.Fatalf("expected search path to be loaded, got %q", cfg.DatabaseSearchPath)
	}
	if cfg.JWTSecret != "shared-access-secret" {
		t.Fatalf("expected jwt secret to be loaded, got %q", cfg.JWTSecret)
	}
	if cfg.RedisAddr != "cache-store:18379" {
		t.Fatalf("expected cache addr to be loaded, got %q", cfg.RedisAddr)
	}
	if cfg.RedisPassword != "cache-pass" {
		t.Fatalf("expected cache password to be loaded, got %q", cfg.RedisPassword)
	}
	if cfg.RedisDB != 2 {
		t.Fatalf("expected redis db to be loaded as 2, got %d", cfg.RedisDB)
	}

}

func TestLoad_RejectsInvalidRedisDB(t *testing.T) {
	t.Setenv("DEV_CORE_URL", "http://center.test")
	t.Setenv("DATA_SERVICE_INTERNAL_TOKEN", "test-internal-token")
	t.Setenv("IF_CACHE_STORE_DATA_DB", "bad")

	_, err := Load()
	if err == nil {
		t.Fatal("expected invalid redis db to be rejected")
	}
}

func TestLoad_ReadsDataServiceConfigFromParentDotEnv(t *testing.T) {
	t.Setenv("DEV_CORE_URL", "http://center.test")
	rootDir := t.TempDir()
	childDir := filepath.Join(rootDir, "data_service")
	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatalf("create child dir failed: %v", err)
	}

	dotenvPath := filepath.Join(rootDir, ".env")
	dotenvContent := []byte("DATA_SERVICE_ADDR=:18102\n" +
		"DATA_SERVICE_INTERNAL_TOKEN=dotenv-internal-token\n" +
		"IF_META_STORE_HOST=dotenv-meta\n" +
		"IF_META_STORE_PORT=18432\n" +
		"IF_META_STORE_USER=dotenv-user\n" +
		"IF_META_STORE_PASSWORD=dotenv-pass\n" +
		"IF_META_STORE_DATA_DB=if_data\n" +
		"IF_META_STORE_DATA_SCHEMA=dotenv_schema\n" +
		"JWT_ACCESS_SECRET=dotenv-secret-1234\n" +
		"IF_CACHE_STORE_HOST=dotenv-cache\n" +
		"IF_CACHE_STORE_PORT=18379\n" +
		"IF_CACHE_STORE_PASSWORD=dotenv-cache-pass\n" +
		"IF_CACHE_STORE_DATA_DB=3\n")
	if err := os.WriteFile(dotenvPath, dotenvContent, 0o644); err != nil {
		t.Fatalf("write dotenv failed: %v", err)
	}

	for _, key := range []string{
		"DATA_SERVICE_ADDR",
		"DATA_SERVICE_INTERNAL_TOKEN",
		"DATA_SERVICE_DATABASE_URL",
		"DATA_SERVICE_DATABASE_SCHEMA",
		"DATA_SERVICE_REDIS_ADDR",
		"DATA_SERVICE_REDIS_PASSWORD",
		"DATA_SERVICE_REDIS_DB",
		"IF_META_STORE_HOST",
		"IF_META_STORE_PORT",
		"IF_META_STORE_USER",
		"IF_META_STORE_PASSWORD",
		"IF_META_STORE_DATA_DB",
		"IF_META_STORE_DATA_SCHEMA",
		"JWT_ACCESS_SECRET",
		"IF_CACHE_STORE_HOST",
		"IF_CACHE_STORE_PORT",
		"IF_CACHE_STORE_PASSWORD",
		"IF_CACHE_STORE_DATA_DB",
	} {
		t.Setenv(key, "")
	}

	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(childDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(currentDir)
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.Addr != ":18102" {
		t.Fatalf("expected addr from dotenv, got %q", cfg.Addr)
	}
	if cfg.DatabaseURL != "postgres://dotenv-user:dotenv-pass@dotenv-meta:18432/if_data?sslmode=disable" {
		t.Fatalf("expected database url from dotenv, got %q", cfg.DatabaseURL)
	}
	if cfg.DatabaseSearchPath != "dotenv_schema" {
		t.Fatalf("expected schema from dotenv, got %q", cfg.DatabaseSearchPath)
	}
	if cfg.JWTSecret != "dotenv-secret-1234" {
		t.Fatalf("expected jwt secret from dotenv, got %q", cfg.JWTSecret)
	}
	if cfg.RedisAddr != "dotenv-cache:18379" {
		t.Fatalf("expected cache addr from dotenv, got %q", cfg.RedisAddr)
	}
	if cfg.RedisPassword != "dotenv-cache-pass" {
		t.Fatalf("expected cache password from dotenv, got %q", cfg.RedisPassword)
	}
	if cfg.RedisDB != 3 {
		t.Fatalf("expected redis db from dotenv, got %d", cfg.RedisDB)
	}
}

func TestLoad_RejectsMissingInternalToken(t *testing.T) {
	t.Setenv("DEV_CORE_URL", "http://center.test")
	t.Setenv("DATA_SERVICE_INTERNAL_TOKEN", "")
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(currentDir) })

	if _, err := Load(); err == nil {
		t.Fatal("expected missing internal token to be rejected")
	}
}

func TestValidateInternalTokenRejectsMissingToken(t *testing.T) {
	t.Setenv("DEV_CORE_URL", "http://center.test")
	if err := ValidateInternalToken(Config{}); err == nil {
		t.Fatal("expected missing internal token to be rejected")
	}
}

func TestResolveCollectorProtocolCatalogPathFindsPublicContractFromChildDirectory(t *testing.T) {
	t.Setenv("DEV_CORE_URL", "http://center.test")
	rootDir := t.TempDir()
	catalogDir := filepath.Join(rootDir, "contracts", "collector-protocols")
	childDir := filepath.Join(rootDir, "data_service", "tmp")
	if err := os.MkdirAll(catalogDir, 0o755); err != nil {
		t.Fatalf("create catalog dir failed: %v", err)
	}
	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatalf("create child dir failed: %v", err)
	}

	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(childDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(currentDir)
	})

	resolved := ResolveCollectorProtocolCatalogPath(filepath.Join("contracts", "collector-protocols"))
	expected, err := filepath.EvalSymlinks(catalogDir)
	if err != nil {
		t.Fatalf("resolve expected catalog path failed: %v", err)
	}
	if resolved != expected {
		t.Fatalf("expected catalog path %q, got %q", catalogDir, resolved)
	}
}

func TestResolveCollectorProtocolCatalogPathUsesPublicContractByDefault(t *testing.T) {
	t.Setenv("DEV_CORE_URL", "http://center.test")
	rootDir := t.TempDir()
	catalogDir := filepath.Join(rootDir, "contracts", "collector-protocols")
	childDir := filepath.Join(rootDir, "data_service", "tmp")
	if err := os.MkdirAll(catalogDir, 0o755); err != nil {
		t.Fatalf("create catalog dir failed: %v", err)
	}
	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatalf("create child dir failed: %v", err)
	}

	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(childDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(currentDir)
	})

	expected, err := filepath.EvalSymlinks(catalogDir)
	if err != nil {
		t.Fatalf("resolve expected catalog path failed: %v", err)
	}
	if resolved := ResolveCollectorProtocolCatalogPath(""); resolved != expected {
		t.Fatalf("expected default catalog path %q, got %q", catalogDir, resolved)
	}
}

func TestResolveCollectorProtocolCatalogPathKeepsAbsolutePath(t *testing.T) {
	t.Setenv("DEV_CORE_URL", "http://center.test")
	configured := filepath.Join(t.TempDir(), "collector-protocols")
	if resolved := ResolveCollectorProtocolCatalogPath(configured); resolved != configured {
		t.Fatalf("expected absolute path %q, got %q", configured, resolved)
	}
}

func TestValidateJWTSecret_RejectsWeakSecret(t *testing.T) {
	t.Setenv("DEV_CORE_URL", "http://center.test")
	err := ValidateJWTSecret("short-secret")
	if err == nil {
		t.Fatal("expected weak secret to be rejected")
	}
	if got := err.Error(); got != "JWT_ACCESS_SECRET 长度不能少于 16 个字符" {
		t.Fatalf("unexpected error message: %q", got)
	}
}

func TestValidateConnectionsDependencies_RejectsMissingDatabaseURL(t *testing.T) {
	t.Setenv("DEV_CORE_URL", "http://center.test")
	err := ValidateConnectionsDependencies(Config{JWTSecret: "1234567890abcdef"})
	if err == nil {
		t.Fatal("expected missing database url to be rejected")
	}
	if got := err.Error(); got != "缺少 IF_META_STORE_* 配置，connections 路由不会挂载" {
		t.Fatalf("unexpected error message: %q", got)
	}
}

func TestValidatePreviewDependencies_RejectsMissingRedisAddr(t *testing.T) {
	t.Setenv("DEV_CORE_URL", "http://center.test")
	err := ValidatePreviewDependencies(Config{})
	if err == nil {
		t.Fatal("expected missing redis addr to be rejected")
	}
	if got := err.Error(); got != "缺少 IF_CACHE_STORE_* 配置，preview 路由不会挂载" {
		t.Fatalf("unexpected error message: %q", got)
	}
}
