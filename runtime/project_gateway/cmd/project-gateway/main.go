package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/indu-forge/project-gateway/internal/gateway"
)

func main() {
	var config gateway.Config
	flag.StringVar(&config.Listen, "listen", "127.0.0.1:17800", "工程入口监听地址")
	flag.StringVar(&config.ClientRoot, "client-root", "", "当前 Release 的 client 目录")
	flag.StringVar(&config.RuntimeAPIURL, "runtime-api", "http://127.0.0.1:17801", "本机 Runtime API URL")
	flag.StringVar(&config.DeploymentID, "deployment-id", "", "工程部署 ID")
	flag.StringVar(&config.AccountID, "account-id", "", "工程 NATS Account ID")
	flag.StringVar(&config.ProjectID, "project-id", "", "工程 ID")
	flag.StringVar(&config.SiteID, "site-id", "", "运行站点 ID")
	flag.StringVar(&config.NodeID, "node-id", "", "物理节点 ID")
	flag.StringVar(&config.Version, "version", "", "Release 版本")
	flag.StringVar(&config.ExecutionForm, "execution-form", "", "native-linux 或 native-windows")
	flag.Parse()

	app, err := gateway.New(config)
	if err != nil {
		slog.Error("project-gateway 配置无效", "error", err)
		os.Exit(2)
	}
	server := &http.Server{
		Addr:              config.Listen,
		Handler:           app.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	stop, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	go func() {
		<-stop.Done()
		ctx, done := context.WithTimeout(context.Background(), 20*time.Second)
		defer done()
		if shutdownErr := server.Shutdown(ctx); shutdownErr != nil {
			slog.Error("project-gateway 排空失败", "error", shutdownErr)
		}
	}()
	slog.Info("project-gateway 已启动", "listen", config.Listen, "deploymentId", config.DeploymentID)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("project-gateway 退出", "error", err)
		os.Exit(1)
	}
}
