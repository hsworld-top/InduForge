package config

import (
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
	t.Setenv("DATA_SERVICE_ADDR", ":19095")
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
