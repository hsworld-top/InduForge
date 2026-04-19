package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

const defaultAddr = ":9095"
const minJWTSecretLength = 16

// Config 定义 data_service 的基础运行配置。
type Config struct {
	Addr               string
	DatabaseURL        string
	DatabaseSearchPath string
	JWTSecret          string
	RedisAddr          string
	RedisPassword      string
	RedisDB            int
}

// Load 从环境变量读取服务配置，并在缺省时使用内置默认值。
func Load() (Config, error) {
	addr := strings.TrimSpace(os.Getenv("DATA_SERVICE_ADDR"))
	if addr == "" {
		addr = defaultAddr
	}

	if err := validateAddr(addr); err != nil {
		return Config{}, err
	}

	redisDB, err := parseRedisDB(os.Getenv("DATA_SERVICE_REDIS_DB"))
	if err != nil {
		return Config{}, err
	}

	return Config{
		Addr:               addr,
		DatabaseURL:        strings.TrimSpace(os.Getenv("DATA_SERVICE_DATABASE_URL")),
		DatabaseSearchPath: strings.TrimSpace(os.Getenv("DATA_SERVICE_DATABASE_SCHEMA")),
		JWTSecret:          strings.TrimSpace(os.Getenv("DATA_SERVICE_JWT_SECRET")),
		RedisAddr:          strings.TrimSpace(os.Getenv("DATA_SERVICE_REDIS_ADDR")),
		RedisPassword:      strings.TrimSpace(os.Getenv("DATA_SERVICE_REDIS_PASSWORD")),
		RedisDB:            redisDB,
	}, nil
}

// validateAddr 校验监听地址是否为合法的 TCP 地址。
func validateAddr(addr string) error {
	if _, err := net.ResolveTCPAddr("tcp", addr); err != nil {
		return fmt.Errorf("无效的 DATA_SERVICE_ADDR %q: %w", addr, err)
	}
	return nil
}

// ValidateConnectionsDependencies 校验 connections 路由可选依赖。
func ValidateConnectionsDependencies(cfg Config) error {
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		return fmt.Errorf("缺少 DATA_SERVICE_DATABASE_URL，connections 路由不会挂载")
	}
	if err := ValidateJWTSecret(cfg.JWTSecret); err != nil {
		return err
	}
	return nil
}

// ValidateJWTSecret 校验 JWT 密钥的最小强度门槛。
func ValidateJWTSecret(secret string) error {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return fmt.Errorf("缺少 DATA_SERVICE_JWT_SECRET，connections 路由不会挂载")
	}
	if len(secret) < minJWTSecretLength {
		return fmt.Errorf("DATA_SERVICE_JWT_SECRET 长度不能少于 %d 个字符", minJWTSecretLength)
	}
	return nil
}

// ValidatePreviewDependencies 校验 preview 会话路由是否具备运行依赖。
func ValidatePreviewDependencies(cfg Config) error {
	if strings.TrimSpace(cfg.RedisAddr) == "" {
		return fmt.Errorf("缺少 DATA_SERVICE_REDIS_ADDR，preview 路由不会挂载")
	}
	return nil
}

func parseRedisDB(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("无效的 DATA_SERVICE_REDIS_DB %q: %w", raw, err)
	}
	if value < 0 {
		return 0, fmt.Errorf("DATA_SERVICE_REDIS_DB 不能小于 0")
	}
	return value, nil
}
