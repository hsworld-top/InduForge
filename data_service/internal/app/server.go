package app

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/indu-forge/data_service/internal/config"
	"github.com/indu-forge/data_service/internal/http/router"
)

// Server 封装 data_service 的 HTTP 服务生命周期。
type Server struct {
	httpServer *http.Server
}

// NewServer 创建一个带有默认路由的 HTTP 服务实例。
func NewServer(cfg config.Config) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:              cfg.Addr,
			Handler:           router.NewRouter(),
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

// Run 启动服务并在收到取消信号后优雅退出。
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		errCh <- s.httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	case err := <-errCh:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
