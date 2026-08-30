package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/indu-forge/runtime-engine/internal/runtimeengine"
)

var version = "dev"

func main() {
	configPath := flag.String("config", "", "runtime-engine.config.v1 的绝对路径")
	configRoot := flag.String("config-root", "", "可信配置根目录（绝对路径）")
	indexPath := flag.String("index", "", "collector-runtime-index.v1 的绝对路径")
	listen := flag.String("listen", ":8080", "HTTP 监听地址")
	production := flag.Bool("production", true, "生产模式：要求只读安全挂载并禁止 NATS none 凭据")
	flag.Parse()
	host := runtimeengine.New(runtimeengine.Options{ConfigPath: *configPath, ConfigRoot: *configRoot, IndexPath: *indexPath, Production: *production, Version: version})
	server := newServer(*listen, host.State.Handler())
	listener, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Printf("RuntimeEngine HTTP bind failed: %v", err)
		return
	}
	defer listener.Close()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(signals)
	errs := make(chan error, 1)
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errs <- err
		}
	}()
	// HTTP is already listening in STARTING; startup failures remain observable
	// through /health and /api/v1/status instead of terminating the process.
	startup, cancelStartup := context.WithCancel(context.Background())
	defer cancelStartup()
	started := make(chan error, 1)
	go func() { started <- host.Start(startup) }()
	startupFinished := false
	select {
	case err := <-started:
		startupFinished = true
		if err != nil {
			log.Printf("RuntimeEngine preflight failed: %v", err)
		}
		// A failed preflight deliberately leaves HTTP observable until a signal.
		select {
		case err := <-errs:
			log.Print(err)
		case <-host.Fatal():
			log.Print("RuntimeEngine worker failed")
		case <-signals:
		}
	case err := <-errs:
		log.Print(err)
	case <-host.Fatal():
		log.Print("RuntimeEngine worker failed")
	case <-signals:
	}
	// Every outer termination path aborts preflight and shares one total drain
	// budget.  Do not stack a second 20s wait behind a stuck Start mutex.
	cancelStartup()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if !startupFinished {
		select {
		case <-started:
		case <-ctx.Done():
			_ = server.Close()
			return
		}
	}
	if err := host.Shutdown(ctx); err != nil {
		log.Printf("RuntimeEngine shutdown failed: %v", err)
	}
	_ = server.Shutdown(ctx)
}
func newServer(address string, handler http.Handler) *http.Server {
	return &http.Server{Addr: address, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
}
