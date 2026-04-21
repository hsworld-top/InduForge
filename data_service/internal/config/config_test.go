package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_UsesDefaultAddr(t *testing.T) {
	t.Setenv("DATA_SERVICE_ADDR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.Addr != defaultAddr {
		t.Fatalf("expected default addr %q, got %q", defaultAddr, cfg.Addr)
	}
}

func TestLoad_RejectsInvalidAddr(t *testing.T) {
	t.Setenv("DATA_SERVICE_ADDR", "invalid-addr")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoad_ReadsOptionalDependencyConfig(t *testing.T) {
	t.Setenv("DATA_SERVICE_ADDR", ":19602")
	t.Setenv("DATA_SERVICE_DATABASE_URL", "postgres://demo")
	t.Setenv("DATA_SERVICE_DATABASE_SCHEMA", "tenant_a")
	t.Setenv("DATA_SERVICE_JWT_SECRET", "secret-123")
	t.Setenv("DATA_SERVICE_REDIS_ADDR", "127.0.0.1:6379")
	t.Setenv("DATA_SERVICE_REDIS_PASSWORD", "redis-pass")
	t.Setenv("DATA_SERVICE_REDIS_DB", "2")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.DatabaseURL != "postgres://demo" {
		t.Fatalf("expected database url to be loaded, got %q", cfg.DatabaseURL)
	}
	if cfg.DatabaseSearchPath != "tenant_a" {
		t.Fatalf("expected search path to be loaded, got %q", cfg.DatabaseSearchPath)
	}
	if cfg.JWTSecret != "secret-123" {
		t.Fatalf("expected jwt secret to be loaded, got %q", cfg.JWTSecret)
	}
	if cfg.RedisAddr != "127.0.0.1:6379" {
		t.Fatalf("expected redis addr to be loaded, got %q", cfg.RedisAddr)
	}
	if cfg.RedisPassword != "redis-pass" {
		t.Fatalf("expected redis password to be loaded, got %q", cfg.RedisPassword)
	}
	if cfg.RedisDB != 2 {
		t.Fatalf("expected redis db to be loaded as 2, got %d", cfg.RedisDB)
	}
}

func TestLoad_RejectsInvalidRedisDB(t *testing.T) {
	t.Setenv("DATA_SERVICE_REDIS_DB", "bad")

	_, err := Load()
	if err == nil {
		t.Fatal("expected invalid redis db to be rejected")
	}
}

func TestLoad_ReadsDataServiceConfigFromParentDotEnv(t *testing.T) {
	rootDir := t.TempDir()
	childDir := filepath.Join(rootDir, "data_service")
	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatalf("create child dir failed: %v", err)
	}

	dotenvPath := filepath.Join(rootDir, ".env")
	dotenvContent := []byte("DATA_SERVICE_ADDR=:29602\n" +
		"DATA_SERVICE_DATABASE_URL=postgres://dotenv-demo\n" +
		"DATA_SERVICE_DATABASE_SCHEMA=dotenv_schema\n" +
		"DATA_SERVICE_JWT_SECRET=dotenv-secret-1234\n" +
		"DATA_SERVICE_REDIS_ADDR=127.0.0.1:6380\n" +
		"DATA_SERVICE_REDIS_PASSWORD=dotenv-redis-pass\n" +
		"DATA_SERVICE_REDIS_DB=3\n")
	if err := os.WriteFile(dotenvPath, dotenvContent, 0o644); err != nil {
		t.Fatalf("write dotenv failed: %v", err)
	}

	for _, key := range []string{
		"DATA_SERVICE_ADDR",
		"DATA_SERVICE_DATABASE_URL",
		"DATA_SERVICE_DATABASE_SCHEMA",
		"DATA_SERVICE_JWT_SECRET",
		"DATA_SERVICE_REDIS_ADDR",
		"DATA_SERVICE_REDIS_PASSWORD",
		"DATA_SERVICE_REDIS_DB",
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
	if cfg.Addr != ":29602" {
		t.Fatalf("expected addr from dotenv, got %q", cfg.Addr)
	}
	if cfg.DatabaseURL != "postgres://dotenv-demo" {
		t.Fatalf("expected database url from dotenv, got %q", cfg.DatabaseURL)
	}
	if cfg.DatabaseSearchPath != "dotenv_schema" {
		t.Fatalf("expected schema from dotenv, got %q", cfg.DatabaseSearchPath)
	}
	if cfg.JWTSecret != "dotenv-secret-1234" {
		t.Fatalf("expected jwt secret from dotenv, got %q", cfg.JWTSecret)
	}
	if cfg.RedisAddr != "127.0.0.1:6380" {
		t.Fatalf("expected redis addr from dotenv, got %q", cfg.RedisAddr)
	}
	if cfg.RedisPassword != "dotenv-redis-pass" {
		t.Fatalf("expected redis password from dotenv, got %q", cfg.RedisPassword)
	}
	if cfg.RedisDB != 3 {
		t.Fatalf("expected redis db from dotenv, got %d", cfg.RedisDB)
	}
}

func TestValidateJWTSecret_RejectsWeakSecret(t *testing.T) {
	err := ValidateJWTSecret("short-secret")
	if err == nil {
		t.Fatal("expected weak secret to be rejected")
	}
	if got := err.Error(); got != "DATA_SERVICE_JWT_SECRET 长度不能少于 16 个字符" {
		t.Fatalf("unexpected error message: %q", got)
	}
}

func TestValidateConnectionsDependencies_RejectsMissingDatabaseURL(t *testing.T) {
	err := ValidateConnectionsDependencies(Config{JWTSecret: "1234567890abcdef"})
	if err == nil {
		t.Fatal("expected missing database url to be rejected")
	}
	if got := err.Error(); got != "缺少 DATA_SERVICE_DATABASE_URL，connections 路由不会挂载" {
		t.Fatalf("unexpected error message: %q", got)
	}
}

func TestValidatePreviewDependencies_RejectsMissingRedisAddr(t *testing.T) {
	err := ValidatePreviewDependencies(Config{})
	if err == nil {
		t.Fatal("expected missing redis addr to be rejected")
	}
	if got := err.Error(); got != "缺少 DATA_SERVICE_REDIS_ADDR，preview 路由不会挂载" {
		t.Fatalf("unexpected error message: %q", got)
	}
}
