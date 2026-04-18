package router

import (
	"net/http"

	"github.com/indu-forge/data_service/internal/http/handler"
	"github.com/indu-forge/data_service/internal/http/middleware"
)

// NewRouter 创建 data_service 的基础 HTTP 路由。
func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/health", middleware.ErrorHandler(func(w http.ResponseWriter, r *http.Request) error {
		handler.NewHealthHandler().ServeHTTP(w, r)
		return nil
	}))
	return mux
}
