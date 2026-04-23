package middleware

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/indu-forge/data_service/internal/http/response"
)

type optionalResponseWriter struct {
	header          http.Header
	body            bytes.Buffer
	hijacked        bool
	flushed         bool
	pushedTargets   []string
	readFromInvoked bool
}

func (w *optionalResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *optionalResponseWriter) WriteHeader(statusCode int) {}

func (w *optionalResponseWriter) Write(p []byte) (int, error) {
	return w.body.Write(p)
}

func (w *optionalResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.hijacked = true
	return nil, nil, nil
}

func (w *optionalResponseWriter) Flush() {
	w.flushed = true
}

func (w *optionalResponseWriter) Push(target string, opts *http.PushOptions) error {
	w.pushedTargets = append(w.pushedTargets, target)
	return nil
}

func (w *optionalResponseWriter) ReadFrom(r io.Reader) (int64, error) {
	w.readFromInvoked = true
	return io.Copy(&w.body, r)
}

type bareResponseWriter struct {
	header http.Header
	body   bytes.Buffer
}

func (w *bareResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *bareResponseWriter) WriteHeader(statusCode int) {}

func (w *bareResponseWriter) Write(p []byte) (int, error) {
	return w.body.Write(p)
}

func TestCapturingResponseWriter_TunnelsOptionalInterfacesWhenAvailable(t *testing.T) {
	underlying := &optionalResponseWriter{}
	rw := &capturingResponseWriter{ResponseWriter: underlying}

	if _, _, err := rw.Hijack(); err != nil {
		t.Fatalf("hijack failed: %v", err)
	}
	if !underlying.hijacked {
		t.Fatal("expected hijack to reach underlying writer")
	}

	rw.Flush()
	if !underlying.flushed {
		t.Fatal("expected flush to reach underlying writer")
	}

	if err := rw.Push("/assets/app.js", nil); err != nil {
		t.Fatalf("push failed: %v", err)
	}
	if len(underlying.pushedTargets) != 1 || underlying.pushedTargets[0] != "/assets/app.js" {
		t.Fatalf("expected push to reach underlying writer, got %v", underlying.pushedTargets)
	}

	written, err := rw.ReadFrom(strings.NewReader("payload"))
	if err != nil {
		t.Fatalf("readfrom failed: %v", err)
	}
	if written != int64(len("payload")) {
		t.Fatalf("expected %d bytes written, got %d", len("payload"), written)
	}
	if !underlying.readFromInvoked {
		t.Fatal("expected readfrom to reach underlying writer")
	}
	if got := underlying.body.String(); got != "payload" {
		t.Fatalf("expected body %q, got %q", "payload", got)
	}
	if !rw.committed {
		t.Fatal("expected readfrom to mark the response as committed")
	}
}

func TestCapturingResponseWriter_FallsBackWhenOptionalInterfacesAreMissing(t *testing.T) {
	underlying := &bareResponseWriter{}
	rw := &capturingResponseWriter{ResponseWriter: underlying}

	if _, _, err := rw.Hijack(); !errors.Is(err, http.ErrNotSupported) {
		t.Fatalf("expected ErrNotSupported from hijack, got %v", err)
	}

	rw.Flush()

	if err := rw.Push("/assets/app.js", nil); !errors.Is(err, http.ErrNotSupported) {
		t.Fatalf("expected ErrNotSupported from push, got %v", err)
	}

	written, err := rw.ReadFrom(strings.NewReader("payload"))
	if err != nil {
		t.Fatalf("readfrom failed: %v", err)
	}
	if written != int64(len("payload")) {
		t.Fatalf("expected %d bytes written, got %d", len("payload"), written)
	}
	if got := underlying.body.String(); got != "payload" {
		t.Fatalf("expected body %q, got %q", "payload", got)
	}
	if !rw.committed {
		t.Fatal("expected readfrom to mark the response as committed")
	}
}

func TestErrorHandler_DoesNotWriteUnifiedErrorAfterFlush(t *testing.T) {
	handler := RequestIDMiddleware(ErrorHandler(func(w http.ResponseWriter, r *http.Request) error {
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		return errors.New("flush then fail")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(WithRequestID(req.Context(), "req-flush-error"))

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d after flush, got %d", http.StatusOK, rr.Code)
	}
	if rr.Body.Len() != 0 {
		t.Fatalf("expected no unified error body after flush, got %q", rr.Body.String())
	}
	if got := rr.Header().Get("Content-Type"); got != "" {
		t.Fatalf("expected no content type to be written, got %q", got)
	}
	if got := rr.Header().Get("X-Request-ID"); got == "" {
		t.Fatal("expected request id header to be written")
	}
}

func TestCapturingResponseWriter_FlushWithoutFlusherDoesNotCommit(t *testing.T) {
	underlying := &bareResponseWriter{}
	rw := &capturingResponseWriter{ResponseWriter: underlying}

	rw.Flush()

	if rw.committed {
		t.Fatal("expected flush without flusher support to keep committed=false")
	}

	response.WriteError(rw, http.StatusInternalServerError, "req-no-flusher", "INTERNAL_ERROR", "系统内部错误")

	if !rw.committed {
		t.Fatal("expected error write to mark the response as committed")
	}
	if got := underlying.body.String(); got == "" {
		t.Fatal("expected unified error body to be written after flush fallback")
	}
	if got := underlying.body.String(); !strings.Contains(got, "\"code\":30001") {
		t.Fatalf("expected unified error response body, got %q", got)
	}
}
