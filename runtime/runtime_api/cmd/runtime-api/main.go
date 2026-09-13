package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/indu-forge/runtime-api/internal/artifact"
	runtimeauth "github.com/indu-forge/runtime-api/internal/auth"
	"github.com/indu-forge/runtime-api/internal/httpapi"
	"github.com/indu-forge/runtime-api/internal/postgres"
	"github.com/indu-forge/runtime-api/internal/realtime"
	"github.com/indu-forge/runtime-api/internal/securefile"
)

type options struct {
	listen, artifactPath, postgresSecret, tokenSecret, natsURL, natsCredentials, assetRoot  string
	deploymentID, projectID, accountID, siteID, nodeID, version, executionForm, manualOwner string
	manualEpoch                                                                             int64
	secureCookies                                                                           bool
}

type postgresSecret struct {
	SchemaVersion string `json:"schemaVersion"`
	DSN           string `json:"dsn"`
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		if err := runHealthcheckCommand(os.Args[2:]); err != nil {
			fatal("runtime-api 健康检查失败", err)
		}
		return
	}
	var input options
	flag.StringVar(&input.listen, "listen", "127.0.0.1:17801", "Runtime API 本机回环监听地址")
	flag.StringVar(&input.artifactPath, "artifact", "", "runtime-project-artifact.v1 文件")
	flag.StringVar(&input.postgresSecret, "postgres-secret", "", "postgres-dsn.v1 secret 文件")
	flag.StringVar(&input.tokenSecret, "token-secret", "", "runtime-api-tokens.v1 secret 文件")
	flag.StringVar(&input.natsURL, "nats-url", "", "当前 deployment NATS Account URL")
	flag.StringVar(&input.natsCredentials, "nats-credentials", "", "NATS User Credentials 文件")
	flag.StringVar(&input.assetRoot, "asset-root", "", "运行态对象库资源目录（可选，目录内需有 manifest.json）")
	flag.StringVar(&input.deploymentID, "deployment-id", "", "工程部署 ID")
	flag.StringVar(&input.projectID, "project-id", "", "工程 ID")
	flag.StringVar(&input.accountID, "account-id", "", "工程 NATS Account ID")
	flag.StringVar(&input.siteID, "site-id", "", "站点 ID")
	flag.StringVar(&input.nodeID, "node-id", "", "物理节点 ID")
	flag.StringVar(&input.version, "version", "", "Release 版本")
	flag.StringVar(&input.executionForm, "execution-form", nativeExecutionForm(), "native-linux 或 native-windows")
	flag.StringVar(&input.manualOwner, "manual-owner", "", "部署 binding 下发的 manual producer owner")
	flag.Int64Var(&input.manualEpoch, "manual-epoch", 0, "部署 binding 下发的 manual producer epoch")
	flag.BoolVar(&input.secureCookies, "secure-cookies", true, "仅通过 HTTPS 发送运行会话 Cookie")
	flag.Parse()
	if strings.TrimSpace(input.natsURL) == "" {
		input.natsURL = strings.TrimSpace(os.Getenv("IF_RUNTIME_NATS_URL"))
	}

	if err := validateOptions(input); err != nil {
		slog.Error("runtime-api 配置无效", "error", err)
		os.Exit(2)
	}
	catalog, err := artifact.Load(input.artifactPath, input.projectID)
	if err != nil {
		fatal("runtime-api Artifact 无效", err)
	}
	authorizer, err := runtimeauth.Load(input.tokenSecret)
	if err != nil {
		fatal("runtime-api token secret 无效", err)
	}
	secret, err := loadPostgresSecret(input.postgresSecret)
	if err != nil {
		fatal("runtime-api PostgreSQL secret 无效", err)
	}
	startup, cancelStartup := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelStartup()
	store, err := postgres.Open(startup, secret.DSN)
	if err != nil {
		fatal("runtime-api PostgreSQL 不可用", err)
	}
	defer store.Close()
	hub := realtime.NewHub(input.deploymentID, input.accountID, catalog)
	natsToken, err := loadNATSToken(input.natsCredentials)
	if err != nil {
		fatal("runtime-api NATS 凭据无效", err)
	}
	natsSubscriber, err := realtime.ConnectNATS(startup, realtime.NATSOptions{
		URL: input.natsURL, Token: natsToken, Name: "runtime-api-" + input.deploymentID,
	}, hub)
	if err != nil {
		fatal("runtime-api NATS 不可用", err)
	}
	defer natsSubscriber.Close()
	var assetStore httpapi.AssetStore
	if strings.TrimSpace(input.assetRoot) != "" {
		assetStore, err = httpapi.NewFileAssetStore(input.assetRoot)
		if err != nil {
			fatal("运行态对象库索引无效", err)
		}
	}
	app, err := httpapi.New(httpapi.Config{
		DeploymentID: input.deploymentID, ProjectID: input.projectID, AccountID: input.accountID,
		SiteID: input.siteID, NodeID: input.nodeID, Version: input.version, ExecutionForm: input.executionForm,
		SecureCookies: input.secureCookies, Catalog: catalog, Store: store, Authorizer: authorizer, Realtime: hub, ManualEpoch: input.manualEpoch, ManualWriter: store, Publisher: natsSubscriber, CommandStore: store, CommandPublisher: natsSubscriber, Assets: assetStore,
	})
	if err != nil {
		fatal("runtime-api 初始化失败", err)
	}
	server := &http.Server{
		Addr: input.listen, Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
	}
	stop, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	go func() {
		<-stop.Done()
		ctx, done := context.WithTimeout(context.Background(), 20*time.Second)
		defer done()
		if shutdownErr := server.Shutdown(ctx); shutdownErr != nil {
			slog.Error("runtime-api 排空失败", "error", shutdownErr)
		}
	}()
	slog.Info("runtime-api 已启动", "listen", input.listen, "deploymentId", input.deploymentID)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fatal("runtime-api 退出", err)
	}
}

func runHealthcheckCommand(args []string) error {
	flags := flag.NewFlagSet("healthcheck", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	rawURL := flags.String("url", "", "仅允许 Runtime API 回环 /health URL")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return errors.New("healthcheck 参数无效")
	}
	return checkLoopbackHealth(*rawURL, &http.Client{Timeout: 2 * time.Second})
}

func checkLoopbackHealth(rawURL string, client *http.Client) error {
	endpoint, err := url.Parse(rawURL)
	if err != nil || endpoint.Scheme != "http" || endpoint.User != nil || endpoint.Path != "/health" || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return errors.New("healthcheck URL 必须是 HTTP 回环 /health")
	}
	ip := net.ParseIP(endpoint.Hostname())
	if ip == nil || !ip.IsLoopback() || endpoint.Port() == "" {
		return errors.New("healthcheck URL 必须包含回环 IP 和端口")
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("请求 health endpoint: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("health endpoint 返回 HTTP %d", response.StatusCode)
	}
	return nil
}

func validateOptions(input options) error {
	host, _, err := net.SplitHostPort(input.listen)
	if err != nil {
		return fmt.Errorf("listen 格式无效: %w", err)
	}
	ip := net.ParseIP(host)
	if !strings.EqualFold(host, "localhost") && (ip == nil || !ip.IsLoopback()) {
		return errors.New("runtime-api 必须只监听本机回环地址，由 project-gateway 对外提供入口")
	}
	values := []string{input.artifactPath, input.postgresSecret, input.tokenSecret, input.natsURL, input.natsCredentials, input.deploymentID, input.projectID, input.accountID, input.siteID, input.nodeID, input.version, input.manualOwner}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return errors.New("artifact、secret、NATS 与部署身份参数均不能为空")
		}
	}
	if input.manualOwner != "runtime-api" || input.manualEpoch < 1 {
		return errors.New("manual producer fence 参数无效")
	}
	if input.executionForm != "native-linux" && input.executionForm != "native-windows" && input.executionForm != "k3s-workload" {
		return errors.New("execution-form 不受支持")
	}
	return nil
}

func loadNATSToken(path string) (string, error) {
	payload, err := securefile.ReadSecret(path, 1<<20)
	if err != nil {
		return "", err
	}
	var credential struct {
		SchemaVersion string `json:"schemaVersion"`
		AuthType      string `json:"authType"`
		Token         string `json:"token"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&credential); err != nil {
		return "", err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return "", errors.New("NATS 凭据包含尾随内容")
	}
	if credential.SchemaVersion != "nats-credential.v1" || credential.AuthType != "token" || strings.TrimSpace(credential.Token) == "" {
		return "", errors.New("NATS 凭据格式无效")
	}
	return credential.Token, nil
}

func loadPostgresSecret(path string) (postgresSecret, error) {
	payload, err := securefile.ReadSecret(path, 1<<20)
	if err != nil {
		return postgresSecret{}, err
	}
	var secret postgresSecret
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&secret); err != nil {
		return postgresSecret{}, err
	}
	if secret.SchemaVersion != "postgres-dsn.v1" || strings.TrimSpace(secret.DSN) == "" {
		return postgresSecret{}, errors.New("PostgreSQL secret schemaVersion 或 dsn 无效")
	}
	return secret, nil
}

func nativeExecutionForm() string {
	if runtime.GOOS == "windows" {
		return "native-windows"
	}
	return "native-linux"
}

func fatal(message string, err error) {
	slog.Error(message, "error", err)
	os.Exit(1)
}
