package router

import (
	"errors"
	"net/http"

	apperrors "github.com/indu-forge/data_service/internal/errors"
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
	mux.Handle("/api/v1/_internal/error-handler/app-error", middleware.ErrorHandler(func(http.ResponseWriter, *http.Request) error {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "参数错误")
	}))
	mux.Handle("/api/v1/_internal/error-handler/error", middleware.ErrorHandler(func(http.ResponseWriter, *http.Request) error {
		return errors.New("普通错误")
	}))
	return mux
}
