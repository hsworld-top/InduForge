package config

import (
	"bufio"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const defaultAddr = ":18102"
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
	loadDotEnvFromCurrentTree()

	addr := strings.TrimSpace(os.Getenv("DATA_SERVICE_ADDR"))
	if addr == "" {
		addr = defaultAddr
	}

	if err := validateAddr(addr); err != nil {
		return Config{}, err
	}

	redisDB, err := parseRedisDB(firstEnv("IF_CACHE_STORE_DATA_DB", "DATA_SERVICE_REDIS_DB"))
	if err != nil {
		return Config{}, err
	}

	return Config{
		Addr:               addr,
		DatabaseURL:        buildDatabaseURL(),
		DatabaseSearchPath: strings.TrimSpace(firstEnv("IF_META_STORE_DATA_SCHEMA", "DATA_SERVICE_DATABASE_SCHEMA")),
		JWTSecret:          strings.TrimSpace(os.Getenv("DATA_SERVICE_JWT_SECRET")),
		RedisAddr:          buildRedisAddr(),
		RedisPassword:      strings.TrimSpace(firstEnv("IF_CACHE_STORE_PASSWORD", "DATA_SERVICE_REDIS_PASSWORD")),
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
		return fmt.Errorf("缺少 IF_META_STORE_* 配置，connections 路由不会挂载")
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
		return fmt.Errorf("缺少 IF_CACHE_STORE_* 配置，preview 路由不会挂载")
	}
	return nil
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}

func buildDatabaseURL() string {
	host := firstEnv("IF_META_STORE_HOST")
	user := firstEnv("IF_META_STORE_USER")
	password := firstEnv("IF_META_STORE_PASSWORD")
	database := firstEnv("IF_META_STORE_DATA_DB")
	if host == "" || user == "" || database == "" {
		return strings.TrimSpace(os.Getenv("DATA_SERVICE_DATABASE_URL"))
	}

	port := firstEnv("IF_META_STORE_PORT")
	if port == "" {
		port = "18432"
	}

	sslMode := "disable"
	if strings.EqualFold(firstEnv("IF_META_STORE_SSL"), "true") {
		sslMode = "require"
	}

	dsn := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   net.JoinHostPort(host, port),
		Path:   database,
	}
	query := dsn.Query()
	query.Set("sslmode", sslMode)
	dsn.RawQuery = query.Encode()
	return dsn.String()
}

func buildRedisAddr() string {
	host := firstEnv("IF_CACHE_STORE_HOST")
	if host == "" {
		return strings.TrimSpace(os.Getenv("DATA_SERVICE_REDIS_ADDR"))
	}

	port := firstEnv("IF_CACHE_STORE_PORT")
	if port == "" {
		port = "18379"
	}
	return net.JoinHostPort(host, port)
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

func loadDotEnvFromCurrentTree() {
	workingDir, err := os.Getwd()
	if err != nil {
		return
	}

	for _, candidate := range candidateDotEnvPaths(workingDir) {
		if loadDotEnvFile(candidate) {
			return
		}
	}
}

func candidateDotEnvPaths(startDir string) []string {
	current := filepath.Clean(startDir)
	paths := make([]string, 0, 8)

	for {
		paths = append(paths, filepath.Join(current, ".env"))

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return paths
}

func loadDotEnvFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if existingValue, exists := os.LookupEnv(key); exists && strings.TrimSpace(existingValue) != "" {
			continue
		}

		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		_ = os.Setenv(key, value)
	}

	return true
}
