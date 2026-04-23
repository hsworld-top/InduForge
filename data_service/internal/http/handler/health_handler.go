package handler

import (
	"net/http"

	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
)

// NewHealthHandler 返回健康检查处理器。
func NewHealthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]string{
			"status": "healthy",
		})
	})
}
