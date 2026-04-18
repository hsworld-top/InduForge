package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/indu-forge/data_service/internal/app"
	"github.com/indu-forge/data_service/internal/config"
)

func main() {
	cfg := config.Load()
	server := app.NewServer(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
