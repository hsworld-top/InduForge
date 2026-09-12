package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	// 精简离线镜像也能解析 IANA 时区，不依赖宿主机或容器的 zoneinfo。
	_ "time/tzdata"

	"github.com/indu-forge/data_service/internal/app"
	"github.com/indu-forge/data_service/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	server, err := app.NewServer(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer server.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
