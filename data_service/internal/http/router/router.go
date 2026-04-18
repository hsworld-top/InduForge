package router

import (
	"net/http"

	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/http/handler"
	"github.com/indu-forge/data_service/internal/http/middleware"
)

type options struct {
	connectionHandler *handler.ConnectionHandler
	jwtValidator      *auth.JWTValidator
}

// Option 定义路由装配的可选依赖。
type Option func(*options)

// WithConnectionRoutes 注入 connections 领域路由所需依赖。
func WithConnectionRoutes(connectionHandler *handler.ConnectionHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.connectionHandler = connectionHandler
		opts.jwtValidator = jwtValidator
	}
}

// NewRouter 创建 data_service 的基础 HTTP 路由。
func NewRouter(routeOptions ...Option) http.Handler {
	opts := options{}
	for _, option := range routeOptions {
		option(&opts)
	}

	mux := http.NewServeMux()
	mux.Handle("/health", middleware.ErrorHandler(func(w http.ResponseWriter, r *http.Request) error {
		handler.NewHealthHandler().ServeHTTP(w, r)
		return nil
	}))

	mountConnectionRoutes(mux, opts)
	return mux
}

func mountConnectionRoutes(mux *http.ServeMux, opts options) {
	if mux == nil || opts.connectionHandler == nil || opts.jwtValidator == nil {
		return
	}

	mux.Handle(
		"GET /api/v1/data/projects/{projectId}/connections",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:read")(
				middleware.ErrorHandler(opts.connectionHandler.List),
			),
		),
	)
	mux.Handle(
		"POST /api/v1/data/projects/{projectId}/connections",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.connectionHandler.Create),
			),
		),
	)
	mux.Handle(
		"PUT /api/v1/data/projects/{projectId}/connections/{connectionId}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.connectionHandler.Update),
			),
		),
	)
	mux.Handle(
		"DELETE /api/v1/data/projects/{projectId}/connections/{connectionId}",
		middleware.Authenticate(opts.jwtValidator)(
			middleware.RequireCapability("project:write")(
				middleware.ErrorHandler(opts.connectionHandler.Delete),
			),
		),
	)
}
