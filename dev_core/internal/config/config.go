package config

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const defaultAddr = ":18101"

var strictSemVerPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`)

type Config struct {
	Addr                          string
	DatabaseURL                   string
	DBAutoSchemaSync              bool
	JWTSecret                     string
	JWTIssuer                     string
	JWTAudience                   string
	AccessTokenTTL                time.Duration
	RefreshTokenTTL               time.Duration
	AppName                       string
	ObjectStoreEndpoint           string
	ObjectStoreAccessKey          string
	ObjectStoreSecretKey          string
	ObjectStoreDesignBucket       string
	ObjectStoreIFPBucket          string
	ObjectStoreRegion             string
	ObjectStoreUseSSL             bool
	DefaultTenantID               string
	DefaultTenantCode             string
	SuperAdminUserID              string
	SuperAdminUsername            string
	SuperAdminPassword            string
	WorkspaceRoot                 string
	CodeWorkspaceVolume           string
	CodeServerDockerHost          string
	CodeServerImage               string
	CodeServerBindHost            string
	CodeWorkspaceAllowedOrigins   string
	CodeWorkspaceEngine           string
	CodeWorkspaceNamespace        string
	CodeWorkspaceHostPath         string
	WorkspacePublicOriginTemplate string
	CenterPublicOrigin            string
	WorkspaceAllowInsecureHTTPDev bool
	DataServiceURL                string
	DataServiceInternalToken      string
	CacheAddress                  string
	CachePassword                 string
	CacheDB                       int
	NodeConnectionURL             string
	NodePackageDirectory          string
	OpsCenterNodeName             string
	OpsCenterNodeIP               string
	OpsK3sAPIPort                 int
	OpsK3sToken                   string
	CenterReleaseRoot             string
	CenterReleaseHostRoot         string
	ReleaseBuilderEnabled         bool
	ReleaseBuilderImage           string
	ReleaseBuilderID              string
	ReleaseSigningKeyFile         string
	ReleaseSigningKeyID           string
	AuthoringSnapshotCurrentKeyID string
	AuthoringSnapshotKeyring      map[string]string
	MinNodeAgentVersion           string
	MinRuntimeVersion             string
}

func Load() (Config, error) {
	loadDotEnvFromCurrentTree()

	addr := firstEnvWithDefault("DEV_CORE_ADDR", defaultAddr)
	if _, err := net.ResolveTCPAddr("tcp", addr); err != nil {
		return Config{}, fmt.Errorf("DEV_CORE_ADDR 无效: %w", err)
	}
	accessTTL, err := parseDuration(firstEnvWithDefault("JWT_ACCESS_EXPIRES_IN", "2h"))
	if err != nil {
		return Config{}, fmt.Errorf("JWT_ACCESS_EXPIRES_IN 无效: %w", err)
	}
	refreshTTL, err := parseDuration(firstEnvWithDefault("JWT_REFRESH_EXPIRES_IN", "7d"))
	if err != nil {
		return Config{}, fmt.Errorf("JWT_REFRESH_EXPIRES_IN 无效: %w", err)
	}
	jwtSecret := firstEnv("JWT_ACCESS_SECRET", "JWT_SECRET")
	if len(jwtSecret) < 16 {
		return Config{}, fmt.Errorf("JWT_ACCESS_SECRET 或 JWT_SECRET 至少需要 16 个字符")
	}
	databaseURL := buildDatabaseURL()
	if databaseURL == "" {
		return Config{}, fmt.Errorf("缺少 IF_META_STORE_* 或 DEV_CORE_DATABASE_URL 配置")
	}
	dataServiceInternalToken := strings.TrimSpace(firstEnv("DATA_SERVICE_INTERNAL_TOKEN"))
	if dataServiceInternalToken == "" {
		return Config{}, fmt.Errorf("DATA_SERVICE_INTERNAL_TOKEN 不能为空")
	}
	autoSchema, err := strconv.ParseBool(firstEnvWithDefault("DB_AUTO_SCHEMA_SYNC", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("DB_AUTO_SCHEMA_SYNC 无效: %w", err)
	}
	cacheDB, err := strconv.Atoi(firstEnvWithDefault("IF_CACHE_STORE_CORE_DB", "0"))
	if err != nil || cacheDB < 0 {
		return Config{}, fmt.Errorf("IF_CACHE_STORE_CORE_DB 无效")
	}
	k3sAPIPort, err := strconv.Atoi(firstEnvWithDefault("IF_OPS_K3S_API_PORT", "6443"))
	if err != nil || k3sAPIPort < 1 || k3sAPIPort > 65535 {
		return Config{}, fmt.Errorf("IF_OPS_K3S_API_PORT 无效")
	}
	releaseEnabled, err := strconv.ParseBool(firstEnvWithDefault("RELEASE_BUILDER_ENABLED", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("RELEASE_BUILDER_ENABLED 无效: %w", err)
	}
	releaseImage, releaseBuilderID := strings.TrimSpace(firstEnv("RELEASE_BUILDER_IMAGE")), strings.TrimSpace(firstEnv("RELEASE_BUILDER_ID"))
	releaseKeyFile, releaseKeyID := strings.TrimSpace(firstEnv("RELEASE_SIGNING_KEY_FILE")), strings.TrimSpace(firstEnv("RELEASE_SIGNING_KEY_ID"))
	authoringKeyID := strings.TrimSpace(firstEnv("AUTHORING_SNAPSHOT_CURRENT_KEY_ID"))
	authoringKeyring, err := parseAuthoringKeyring(firstEnv("AUTHORING_SNAPSHOT_KEYRING_JSON"))
	if err != nil {
		return Config{}, err
	}
	minAgent, minRuntime := strings.TrimSpace(firstEnv("RELEASE_MIN_NODE_AGENT_VERSION")), strings.TrimSpace(firstEnv("RELEASE_MIN_RUNTIME_VERSION"))
	if releaseEnabled && (releaseImage == "" || releaseBuilderID == "" || releaseKeyFile == "" || releaseKeyID == "" || !strictSemVer(minAgent) || !strictSemVer(minRuntime) || authoringKeyID == "" || authoringKeyring[authoringKeyID] == "") {
		return Config{}, fmt.Errorf("启用正式 Release 构建时必须配置镜像、构建器、签名密钥及严格最小版本")
	}

	workspaceRoot := firstEnvWithDefault("CODE_WORKSPACE_ROOT", filepath.Join(".data", "workspaces"))
	workspaceEngine := strings.ToLower(strings.TrimSpace(firstEnv("CODE_WORKSPACE_ENGINE")))
	workspaceOriginTemplate := strings.TrimSpace(firstEnv("WORKSPACE_PUBLIC_ORIGIN_TEMPLATE"))
	centerPublicOrigin := strings.TrimRight(strings.TrimSpace(firstEnv("CENTER_PUBLIC_ORIGIN")), "/")
	workspaceAllowInsecureHTTPDev, err := strconv.ParseBool(firstEnvWithDefault("WORKSPACE_ALLOW_INSECURE_HTTP_DEV", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("WORKSPACE_ALLOW_INSECURE_HTTP_DEV 无效: %w", err)
	}
	if workspaceEngine == "kubernetes" && (workspaceOriginTemplate == "" || centerPublicOrigin == "") {
		return Config{}, fmt.Errorf("Kubernetes 代码工作区必须配置 WORKSPACE_PUBLIC_ORIGIN_TEMPLATE 和 CENTER_PUBLIC_ORIGIN")
	}
	centerReleaseRoot := filepath.Clean(firstEnvWithDefault("IF_CENTER_RELEASE_ROOT", "/var/lib/induforge/center-releases"))
	centerReleaseHostRoot := filepath.Clean(firstEnvWithDefault("IF_CENTER_RELEASE_HOST_ROOT", centerReleaseRoot))
	if !filepath.IsAbs(centerReleaseRoot) || !filepath.IsAbs(centerReleaseHostRoot) {
		return Config{}, fmt.Errorf("中心 Release 缓存路径必须是绝对路径")
	}
	return Config{
		Addr: addr, DatabaseURL: databaseURL, DBAutoSchemaSync: autoSchema,
		JWTSecret: jwtSecret, JWTIssuer: firstEnvWithDefault("JWT_ISSUER", "induforge"),
		JWTAudience:    firstEnvWithDefault("JWT_AUDIENCE", "induforge-api"),
		AccessTokenTTL: accessTTL, RefreshTokenTTL: refreshTTL,
		AppName:                       firstEnvWithDefault("DEFAULT_TENANT_NAME", "InduFrame"),
		ObjectStoreEndpoint:           net.JoinHostPort(firstEnvWithDefault("IF_OBJECT_STORE_ENDPOINT", "127.0.0.1"), firstEnvWithDefault("IF_OBJECT_STORE_PORT", "18500")),
		ObjectStoreAccessKey:          firstEnv("IF_OBJECT_STORE_ACCESS_KEY"),
		ObjectStoreSecretKey:          firstEnv("IF_OBJECT_STORE_SECRET_KEY"),
		ObjectStoreDesignBucket:       firstEnvWithDefault("IF_OBJECT_STORE_BUCKET_DESIGN", "design-assets"),
		ObjectStoreIFPBucket:          firstEnvWithDefault("IF_OBJECT_STORE_BUCKET_IFP", "ifp-artifacts"),
		ObjectStoreRegion:             firstEnvWithDefault("IF_OBJECT_STORE_REGION", "us-east-1"),
		ObjectStoreUseSSL:             strings.EqualFold(firstEnv("IF_OBJECT_STORE_USE_SSL"), "true"),
		DefaultTenantID:               firstEnvWithDefault("DEFAULT_TENANT_ID", "550e8400-e29b-41d4-a716-446655440000"),
		DefaultTenantCode:             firstEnvWithDefault("DEFAULT_TENANT_CODE", "default"),
		SuperAdminUserID:              firstEnvWithDefault("SUPER_ADMIN_USER_ID", "550e8400-e29b-41d4-a716-446655440001"),
		SuperAdminUsername:            firstEnvWithDefault("SUPER_ADMIN_USERNAME", "superadmin"),
		SuperAdminPassword:            firstEnv("SUPER_ADMIN_PASSWORD"),
		WorkspaceRoot:                 workspaceRoot,
		CodeWorkspaceVolume:           firstEnvWithDefault("CODE_WORKSPACE_VOLUME", "induforge-control-workspaces"),
		CodeServerDockerHost:          firstEnvWithDefault("CODE_SERVER_DOCKER_HOST", "unix:///var/run/docker.sock"),
		CodeServerImage:               firstEnvWithDefault("CODE_SERVER_IMAGE", "induforge/designer-code-server:workspace-templates-source"),
		CodeServerBindHost:            firstEnvWithDefault("CODE_SERVER_BIND_HOST", "127.0.0.1"),
		CodeWorkspaceAllowedOrigins:   firstEnvWithDefault("CODE_WORKSPACE_ALLOWED_ORIGINS", "http://127.0.0.1:18601,http://localhost:18601,http://127.0.0.1:18603,http://localhost:18603"),
		CodeWorkspaceEngine:           workspaceEngine,
		CodeWorkspaceNamespace:        firstEnvWithDefault("POD_NAMESPACE", "induforge-system"),
		CodeWorkspaceHostPath:         firstEnvWithDefault("CODE_WORKSPACE_HOST_PATH", workspaceRoot),
		WorkspacePublicOriginTemplate: workspaceOriginTemplate,
		CenterPublicOrigin:            centerPublicOrigin,
		WorkspaceAllowInsecureHTTPDev: workspaceAllowInsecureHTTPDev,
		DataServiceURL:                strings.TrimRight(firstEnv("DATA_SERVICE_URL"), "/"),
		DataServiceInternalToken:      dataServiceInternalToken,
		CacheAddress:                  net.JoinHostPort(firstEnvWithDefault("IF_CACHE_STORE_HOST", "127.0.0.1"), firstEnvWithDefault("IF_CACHE_STORE_PORT", "18379")),
		CachePassword:                 firstEnv("IF_CACHE_STORE_PASSWORD"),
		CacheDB:                       cacheDB,
		NodeConnectionURL:             strings.TrimSpace(os.Getenv("NODE_CONNECTION_URL")),
		NodePackageDirectory:          firstEnvWithDefault("NODE_PACKAGE_DIRECTORY", defaultNodePackageDirectory()),
		OpsCenterNodeName:             firstEnvWithDefault("IF_OPS_CENTER_NODE_NAME", "induframe-center"),
		OpsCenterNodeIP:               strings.TrimSpace(firstEnv("IF_OPS_CENTER_NODE_IP")),
		OpsK3sAPIPort:                 k3sAPIPort,
		OpsK3sToken:                   strings.TrimSpace(firstEnv("IF_OPS_K3S_TOKEN")),
		CenterReleaseRoot:             centerReleaseRoot,
		CenterReleaseHostRoot:         centerReleaseHostRoot,
		ReleaseBuilderEnabled:         releaseEnabled, ReleaseBuilderImage: releaseImage, ReleaseBuilderID: releaseBuilderID,
		ReleaseSigningKeyFile: releaseKeyFile, ReleaseSigningKeyID: releaseKeyID, MinNodeAgentVersion: minAgent, MinRuntimeVersion: minRuntime,
		AuthoringSnapshotCurrentKeyID: authoringKeyID, AuthoringSnapshotKeyring: authoringKeyring,
	}, nil
}

func parseAuthoringKeyring(raw string) (map[string]string, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]string{}, nil
	}
	var values map[string]string
	if json.Unmarshal([]byte(raw), &values) != nil || values == nil {
		return nil, fmt.Errorf("AUTHORING_SNAPSHOT_KEYRING_JSON 无效")
	}
	for id, encoded := range values {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
		if strings.TrimSpace(id) == "" || err != nil || len(decoded) != 32 {
			return nil, fmt.Errorf("AUTHORING_SNAPSHOT_KEYRING_JSON 包含无效密钥")
		}
	}
	return values, nil
}

func strictSemVer(value string) bool {
	if !strictSemVerPattern.MatchString(value) {
		return false
	}
	core := strings.SplitN(value, "+", 2)[0]
	separator := strings.IndexByte(core, '-')
	if separator < 0 {
		return true
	}
	for _, item := range strings.Split(core[separator+1:], ".") {
		numeric := item != ""
		for _, char := range item {
			if char < '0' || char > '9' {
				numeric = false
				break
			}
		}
		if numeric && len(item) > 1 && item[0] == '0' {
			return false
		}
	}
	return true
}

// 开发命令会在 dev_core 目录启动进程，安装包仍统一存放在仓库根 .data 下。
func defaultNodePackageDirectory() string {
	workingDir, err := os.Getwd()
	if err != nil {
		return filepath.Join(".data", "node-packages")
	}
	for current := filepath.Clean(workingDir); ; current = filepath.Dir(current) {
		if info, statErr := os.Stat(filepath.Join(current, ".env")); statErr == nil && !info.IsDir() {
			return filepath.Join(current, ".data", "node-packages")
		}
		parent := filepath.Dir(current)
		if parent == current {
			return filepath.Join(workingDir, ".data", "node-packages")
		}
	}
}

func buildDatabaseURL() string {
	host := firstEnv("IF_META_STORE_HOST")
	user := firstEnv("IF_META_STORE_USER")
	database := firstEnv("IF_META_STORE_CORE_DB")
	if host == "" || user == "" || database == "" {
		return firstEnv("DEV_CORE_DATABASE_URL")
	}
	port := firstEnvWithDefault("IF_META_STORE_PORT", "18432")
	sslMode := "disable"
	if strings.EqualFold(firstEnv("IF_META_STORE_SSL"), "true") {
		sslMode = "require"
	}
	dsn := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, firstEnv("IF_META_STORE_PASSWORD")),
		Host:   net.JoinHostPort(host, port),
		Path:   database,
	}
	query := dsn.Query()
	query.Set("sslmode", sslMode)
	dsn.RawQuery = query.Encode()
	return dsn.String()
}

func parseDuration(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasSuffix(raw, "d") {
		days, err := strconv.ParseInt(strings.TrimSuffix(raw, "d"), 10, 64)
		if err != nil || days <= 0 {
			return 0, fmt.Errorf("天数格式错误")
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(raw)
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}

func firstEnvWithDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

// 从当前目录向上寻找根 .env，保持 Windows 本地启动与其他 Go 模块一致。
func loadDotEnvFromCurrentTree() {
	workingDir, err := os.Getwd()
	if err != nil {
		return
	}
	for current := filepath.Clean(workingDir); ; current = filepath.Dir(current) {
		if loadDotEnvFile(filepath.Join(current, ".env")) {
			return
		}
		parent := filepath.Dir(current)
		if parent == current {
			return
		}
	}
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
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(strings.TrimPrefix(key, "export "))
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key != "" {
			if _, exists := os.LookupEnv(key); !exists {
				_ = os.Setenv(key, value)
			}
		}
	}
	return true
}
