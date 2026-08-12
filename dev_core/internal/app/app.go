package app

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
)

type Options struct {
	Logger      *slog.Logger
	RequestID   func() string
	Middlewares []func(http.Handler) http.Handler
	Mount       func(chi.Router)
}

type App struct {
	handler http.Handler
}

func New(options Options) *App {
	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}
	requestID := options.RequestID
	if requestID == nil {
		requestID = newRequestID
	}

	router := chi.NewRouter()
	router.Use(platformapi.RequestID(requestID))
	router.Use(platformapi.Recovery(logger))
	for _, middleware := range options.Middlewares {
		router.Use(middleware)
	}
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		platformapi.WriteSuccess(w, r, map[string]string{"status": "ok"})
	})
	if options.Mount != nil {
		options.Mount(router)
	}

	return &App{handler: router}
}

func (a *App) Handler() http.Handler {
	return a.handler
}

func newRequestID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "request-id-unavailable"
	}
	return hex.EncodeToString(buffer)
}
