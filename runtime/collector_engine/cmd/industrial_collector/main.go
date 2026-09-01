package main

import (
	"context"
	"flag"
	"github.com/indu-forge/collector-engine/internal/driver"
	"github.com/indu-forge/collector-engine/internal/engine"
	"github.com/indu-forge/collector-engine/internal/health"
	"github.com/indu-forge/collector-engine/internal/loader"
	"github.com/indu-forge/collector-engine/internal/publisher"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	artifact := flag.String("artifact", "", "artifact 绝对路径")
	binding := flag.String("binding", "", "binding 绝对路径")
	index := flag.String("index", "", "resolver index 绝对路径")
	natsURL := flag.String("nats-url", "", "由安全 resolver 注入的 NATS URL")
	listen := flag.String("listen", "127.0.0.1:18082", "健康检查地址")
	flag.Parse()
	loaded, err := loader.Load(*artifact, *binding, *index)
	if err != nil {
		log.Print("collector 配置无效")
		os.Exit(1)
	}
	p, err := publisher.Connect(*natsURL, nil)
	if err != nil {
		log.Print("collector NATS 未连接")
		os.Exit(1)
	}
	defer p.Close()
	h := &health.State{}
	collector, err := engine.New(loaded, driver.NewRegistry(), p, h)
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
