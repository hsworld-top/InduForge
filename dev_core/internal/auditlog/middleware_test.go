package auditlog_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/indu-forge/dev_core/internal/auditlog"
	"github.com/indu-forge/dev_core/internal/auth"
)

func TestAuditMiddlewareDoesNotReadRequestBody(t *testing.T) {
	body := &countingBody{Reader: strings.NewReader(`{"password":"secret"}`)}
	writer := &captureWriter{}
	handler := auditlog.Middleware(writer, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"msg":"success"}`))
	}))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/projects", nil)
	request.Body = body
	request = request.WithContext(auth.WithUser(request.Context(), auth.User{ID: "user-id", TenantID: "tenant-id", Username: "tester"}))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if body.reads != 0 {
		t.Fatalf("审计中间件不应读取请求体: reads=%d", body.reads)
	}
	if len(writer.items) != 1 || writer.items[0].Result != "success" || writer.items[0].Metadata["businessCode"] != 0 {
		t.Fatalf("审计记录错误: %+v", writer.items)
	}
}

type countingBody struct {
	io.Reader
	reads int
}

func (b *countingBody) Read(payload []byte) (int, error) {
	b.reads++
	return b.Reader.Read(payload)
}

func (b *countingBody) Close() error { return nil }

type captureWriter struct{ items []auditlog.Log }

func (w *captureWriter) Create(_ context.Context, item auditlog.Log) error {
	w.items = append(w.items, item)
	return nil
}
