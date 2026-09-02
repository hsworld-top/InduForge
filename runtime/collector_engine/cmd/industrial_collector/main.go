package main

import (
	"context"
	"flag"
	"github.com/indu-forge/collector-engine/internal/driver"
	"github.com/indu-forge/collector-engine/internal/engine"
	"github.com/indu-forge/collector-engine/internal/health"
	"github.com/indu-forge/collector-engine/internal/loader"
	"github.com/indu-forge/collector-engine/internal/publisher"
	"github.com/indu-forge/collector-engine/internal/resolver"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	artifact := flag.String("artifact", "", "artifact 绝对路径")
	binding := flag.String("binding", "", "binding 绝对路径")
	index := flag.String("index", "", "resolver index 绝对路径")
	listen := flag.String("listen", "127.0.0.1:18082", "健康检查地址")
	flag.Parse()
	loaded, err := loader.Load(*artifact, *binding, *index)
	if err != nil {
		// 仅记录配置阶段，不记录 Secret、连接串或原始 binding。
		log.Printf("collector 配置无效 (%s)", loaderErrorClass(err))
		os.Exit(1)
	}
	r, err := resolver.New(loaded)
	if err != nil {
		log.Print("collector resolver 无效")
		os.Exit(1)
	}
	natsConfig, err := r.ResolveNATS(context.Background(), loaded.Binding.NATS.ServerResourceRef, loaded.Binding.NATS.CredentialSecretRef, loaded.Binding.AccountID)
	if err != nil {
		log.Print("collector NATS credential 不可用")
		os.Exit(1)
	}
	p, err := publisher.Connect(natsConfig.URL, natsConfig.Options)
	if err != nil {
		log.Print("collector NATS 未连接")
		os.Exit(1)
	}
	defer p.Close()
	h := &health.State{}
	collector, err := engine.New(loaded, driver.NewRegistry(driver.NewModbusTCP(r), driver.NewOPCUA(r)), p, h)
	if err != nil {
		log.Print("collector 初始化失败")
		os.Exit(1)
	}
	server := &http.Server{Addr: *listen, Handler: h.Handler(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Print("health server stopped")
		}
	}()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := collector.Run(ctx); err != nil {
		log.Print("collector 未就绪")
	}
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	_ = server.Shutdown(shutdownCtx)
}

func loaderErrorClass(err error) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "artifact 契约"):
		return "artifact-schema"
	case strings.Contains(message, "binding 契约"):
		return "binding-schema"
	case strings.Contains(message, "resolver index"):
		return "resolver-index"
	case strings.Contains(message, "artifact identity"):
		return "artifact-identity"
	case strings.Contains(message, "WAL"):
		return "wal-capacity"
	case strings.Contains(message, "secretRef"):
		return "connection-secret-reference"
	case strings.Contains(message, "driver"):
		return "driver-allowlist"
	case strings.Contains(message, "connection"):
		return "connection-reference"
	case strings.Contains(message, "NATS"):
		return "nats-reference"
	case strings.Contains(message, "mapping"):
		return "mapping-reference"
	default:
		return "json-or-cross-field"
	}
}
