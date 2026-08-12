package auditlog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/indu-forge/dev_core/internal/auth"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
)

const responseCaptureLimit = 4096

type Writer interface {
	Create(context.Context, Log) error
}

// Middleware 只旁路观察响应，不读取请求体，避免密码、Token、节点密钥和文件内容进入审计日志。
func Middleware(writer Writer, logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, ok := auth.UserFromContext(r.Context())
			if !ok || !strings.HasPrefix(r.URL.Path, "/api/v1/") {
				next.ServeHTTP(w, r)
				return
			}

			startedAt := time.Now()
			observer := &responseObserver{ResponseWriter: w}
			defer func() {
				if recovered := recover(); recovered != nil {
					observer.statusCode = http.StatusInternalServerError
					recordRequest(r, actor, observer, startedAt, writer, logger)
					panic(recovered)
				}
				recordRequest(r, actor, observer, startedAt, writer, logger)
			}()
			next.ServeHTTP(observer, r)
		})
	}
}

type responseObserver struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (w *responseObserver) WriteHeader(statusCode int) {
	if w.statusCode == 0 {
		w.statusCode = statusCode
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseObserver) Write(payload []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	remaining := responseCaptureLimit - w.body.Len()
	if remaining > 0 {
		_, _ = w.body.Write(payload[:min(len(payload), remaining)])
	}
	return w.ResponseWriter.Write(payload)
}

func (w *responseObserver) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func recordRequest(r *http.Request, actor auth.User, response *responseObserver, startedAt time.Time, writer Writer, logger *slog.Logger) {
	statusCode := response.statusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	businessCode := responseBusinessCode(response)
	result := "success"
	level := "INFO"
	if statusCode >= http.StatusBadRequest || businessCode != 0 {
		result = "failure"
		level = "WARN"
	}
	resource := strings.TrimPrefix(r.URL.Path, "/api/v1/")
	if separator := strings.IndexByte(resource, '/'); separator >= 0 {
		resource = resource[:separator]
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 2*time.Second)
	defer cancel()
	err := writer.Create(ctx, Log{
		TenantID: actor.TenantID, UserID: actor.ID, Username: actor.Username, FullName: actor.FullName,
		Level: level, Action: actionForMethod(r.Method), Resource: resource,
		Message: fmt.Sprintf("%s %s", r.Method, r.URL.Path), RequestID: platformapi.RequestIDFromContext(r.Context()),
		Method: r.Method, Path: r.URL.Path, Result: result, IP: requestIP(r), UserAgent: r.UserAgent(),
		Metadata:  map[string]any{"statusCode": statusCode, "businessCode": businessCode, "durationMs": time.Since(startedAt).Milliseconds()},
		CreatedAt: time.Now(),
	})
	if err != nil {
		logger.ErrorContext(r.Context(), "写入审计日志失败", "requestId", platformapi.RequestIDFromContext(r.Context()), "error", err)
	}
}

func responseBusinessCode(response *responseObserver) int {
	if !strings.Contains(response.Header().Get("Content-Type"), "application/json") {
		return 0
	}
	var payload struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(response.body.Bytes(), &payload); err != nil {
		return 0
	}
	return payload.Code
}

func actionForMethod(method string) string {
	switch method {
	case http.MethodGet:
		return "read"
	case http.MethodPost:
		return "create"
	case http.MethodPut, http.MethodPatch:
		return "update"
	case http.MethodDelete:
		return "delete"
	default:
		return strings.ToLower(method)
	}
}

func requestIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
		return forwarded
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
