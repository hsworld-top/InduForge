package config

import (
	"bufio"
	"encoding/base64"
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
	Addr                         string
	DatabaseURL                  string
	DatabaseSearchPath           string
	JWTSecret                    string
	RedisAddr                    string
	RedisPassword                string
	RedisDB                      int
	DevDatabaseURL               string
	RedisDevDB                   int
	DevCacheKeyPrefix            string
	MessageHubAddr               string
	MessageHubUsername           string
	MessageHubPassword           string
	CollectorSecretKey           []byte
	CollectorSecretKeyVersion    string
	AlarmSecretKey               []byte
	AlarmSecretKeyVersion        string
	ConnectionSecretKey          []byte
	ConnectionSecretKeyVersion   string
	ComputeSandboxURL            string
	ComputeSandboxToken          string
	CollectorProtocolCatalogPath string
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
	redisDevDB, err := parseRedisDB(firstEnv("IF_CACHE_STORE_DEV_RT_DB", "DATA_SERVICE_REDIS_DEV_DB"))
	if err != nil {
		return Config{}, err
	}
	collectorSecretKey, err := parseCollectorSecretKey(os.Getenv("DATA_SERVICE_COLLECTOR_SECRET_KEY"))
	if err != nil {
		return Config{}, err
	}
	alarmSecretKey, err := parseAlarmSecretKey(os.Getenv("DATA_SERVICE_ALARM_SECRET_KEY"))
	if err != nil {
		return Config{}, err
	}
	connectionSecretKey, err := parseConnectionSecretKey(os.Getenv("DATA_SERVICE_CONNECTION_SECRET_KEY"))
	if err != nil {
		return Config{}, err
	}

	return Config{
		Addr:                         addr,
		DatabaseURL:                  buildDatabaseURL(),
		DevDatabaseURL:               buildDevDatabaseURL(),
		DatabaseSearchPath:           strings.TrimSpace(firstEnv("IF_META_STORE_DATA_SCHEMA", "DATA_SERVICE_DATABASE_SCHEMA")),
		JWTSecret:                    strings.TrimSpace(os.Getenv("JWT_ACCESS_SECRET")),
		RedisAddr:                    buildRedisAddr(),
		RedisPassword:                strings.TrimSpace(firstEnv("IF_CACHE_STORE_PASSWORD", "DATA_SERVICE_REDIS_PASSWORD")),
		RedisDB:                      redisDB,
		RedisDevDB:                   redisDevDB,
		DevCacheKeyPrefix:            firstEnvWithDefault("IF_DEV_CACHE_KEY_PREFIX", "ifdev"),
		MessageHubAddr:               buildMessageHubAddr(),
		MessageHubUsername:           strings.TrimSpace(firstEnv("IF_MESSAGE_HUB_USERNAME")),
		MessageHubPassword:           strings.TrimSpace(firstEnv("IF_MESSAGE_HUB_PASSWORD")),
		CollectorSecretKey:           collectorSecretKey,
		CollectorSecretKeyVersion:    firstEnvWithDefault("DATA_SERVICE_COLLECTOR_SECRET_KEY_VERSION", "v1"),
		AlarmSecretKey:               alarmSecretKey,
		AlarmSecretKeyVersion:        firstEnvWithDefault("DATA_SERVICE_ALARM_SECRET_KEY_VERSION", "v1"),
		ConnectionSecretKey:          connectionSecretKey,
		ConnectionSecretKeyVersion:   firstEnvWithDefault("DATA_SERVICE_CONNECTION_SECRET_KEY_VERSION", "v1"),
		ComputeSandboxURL:            strings.TrimSpace(os.Getenv("DATA_SERVICE_COMPUTE_SANDBOX_URL")),
		ComputeSandboxToken:          strings.TrimSpace(os.Getenv("DATA_SERVICE_COMPUTE_SANDBOX_TOKEN")),
		CollectorProtocolCatalogPath: strings.TrimSpace(os.Getenv("DATA_SERVICE_COLLECTOR_PROTOCOL_CATALOG_PATH")),
	}, nil
}

func ResolveCollectorProtocolCatalogPath(configured string) string {
	configured = strings.TrimSpace(configured)
	if configured == "" {
		configured = filepath.Join("contracts", "collector-protocols")
	}
	if filepath.IsAbs(configured) {
		return filepath.Clean(configured)
	}

	// 开发态可能从仓库根目录、data_service 或 Air 的 tmp 目录启动，逐级向上定位仓库公共契约。
	current, err := os.Getwd()
	if err == nil {
		for {
			candidate := filepath.Join(current, configured)
			if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
				return candidate
			}
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
			current = parent
		}
	}
	return filepath.Clean(configured)
}

// ValidateCollectorSecretKey 校验统一采集连接密钥服务是否具备启动条件。
func ValidateCollectorSecretKey(cfg Config) error {
	if len(cfg.CollectorSecretKey) != 32 {
		return fmt.Errorf("缺少有效的 DATA_SERVICE_COLLECTOR_SECRET_KEY，工业采集连接路由不会挂载")
	}
	if strings.TrimSpace(cfg.CollectorSecretKeyVersion) == "" {
		return fmt.Errorf("DATA_SERVICE_COLLECTOR_SECRET_KEY_VERSION 不能为空")
	}
	return nil
}

func parseCollectorSecretKey(value string) ([]byte, error) {
	return parseAES256SecretKey(value, "DATA_SERVICE_COLLECTOR_SECRET_KEY")
}

func parseAlarmSecretKey(value string) ([]byte, error) {
	return parseAES256SecretKey(value, "DATA_SERVICE_ALARM_SECRET_KEY")
}

func parseConnectionSecretKey(value string) ([]byte, error) {
	return parseAES256SecretKey(value, "DATA_SERVICE_CONNECTION_SECRET_KEY")
}

func parseAES256SecretKey(value, envName string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("%s 必须是 Base64 字符串", envName)
	}
	if len(decoded) != 32 {
		return nil, fmt.Errorf("%s 解码后必须为 32 字节", envName)
	}
	return decoded, nil
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
	if len(cfg.ConnectionSecretKey) != 32 || strings.TrimSpace(cfg.ConnectionSecretKeyVersion) == "" {
		return fmt.Errorf("缺少有效的 DATA_SERVICE_CONNECTION_SECRET_KEY 或密钥版本")
	}
	return nil
}

// ValidateJWTSecret 校验 JWT 密钥的最小强度门槛。
func ValidateJWTSecret(secret string) error {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return fmt.Errorf("缺少 JWT_ACCESS_SECRET，connections 路由不会挂载")
	}
	if len(secret) < minJWTSecretLength {
		return fmt.Errorf("JWT_ACCESS_SECRET 长度不能少于 %d 个字符", minJWTSecretLength)
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

func firstEnvWithDefault(name string, defaultValue string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return defaultValue
}

func buildDatabaseURL() string {
	return buildDatabaseURLFor(firstEnv("IF_META_STORE_DATA_DB"), strings.TrimSpace(os.Getenv("DATA_SERVICE_DATABASE_URL")))
}

func buildDevDatabaseURL() string {
	return buildDatabaseURLFor(firstEnv("IF_META_STORE_DEV_DATA_DB"), strings.TrimSpace(os.Getenv("DATA_SERVICE_DEV_DATABASE_URL")))
}

func buildDatabaseURLFor(database string, fallback string) string {
	host := firstEnv("IF_META_STORE_HOST")
	user := firstEnv("IF_META_STORE_USER")
	password := firstEnv("IF_META_STORE_PASSWORD")
	if host == "" || user == "" || database == "" {
		return fallback
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

func buildMessageHubAddr() string {
	host := firstEnv("IF_MESSAGE_HUB_HOST")
	if host == "" {
		return strings.TrimSpace(os.Getenv("DATA_SERVICE_MESSAGE_HUB_ADDR"))
	}

	port := firstEnv("IF_MESSAGE_HUB_MQTT_PORT")
	if port == "" {
		port = "18883"
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
