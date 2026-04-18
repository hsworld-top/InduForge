package router

import (
	"net/http"

	"github.com/indu-forge/data_service/internal/http/handler"
)

// NewRouter 创建 data_service 的基础 HTTP 路由。
func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/health", handler.NewHealthHandler())
	return mux
}
