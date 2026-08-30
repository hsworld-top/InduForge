package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/indu-forge/runtime-engine/internal/httpapi"
	"github.com/indu-forge/runtime-engine/internal/loader"
)

var version = "dev"

func main() {
	configPath := flag.String("config", "", "runtime-engine.config.v1 的绝对路径")
	configRoot := flag.String("config-root", "", "可信配置根目录（绝对路径）")
	listenAddress := flag.String("listen", ":8080", "HTTP 监听地址")
	flag.Parse()

	requireReadOnlyMount := true
	loaded, err := loader.Load(loader.Options{ConfigPath: *configPath, ConfigRoot: *configRoot, RequireReadOnlyMount: &requireReadOnlyMount})
	if err != nil {
		log.Printf("RuntimeEngine 未就绪: %v", err)
	}
	state := httpapi.NewEngineState(version, loaded, err)
	server := newServer(*listenAddress, state.Handler())
	if serveErr := serveUntilSignal(server); serveErr != nil {
		log.Fatal(serveErr)
	}
}

func newServer(address string, handler http.Handler) *http.Server {
	return &http.Server{Addr: address, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
}

func serveUntilSignal(server *http.Server) error {
	errorsChannel := make(chan error, 1)
	go func() { errorsChannel <- server.ListenAndServe() }()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(signals)
	select {
	case err := <-errorsChannel:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-signals:
		context, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(context)
	}
}
