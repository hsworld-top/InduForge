package middleware

import (
	"bufio"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

var accessLogf = log.Printf

// AccessLogMiddleware 记录每个 HTTP 请求的最小访问摘要。
// 这里刻意只输出方法、URI、状态码、耗时、响应字节数和 requestId，
// 既方便排障，也避免把敏感业务载荷直接打到日志里。
func AccessLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		loggedWriter := &accessLoggingResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(loggedWriter, r)

		accessLogf(
			"info: access requestId=%s method=%s uri=%s status=%d duration=%s bytes=%d",
			RequestID(r.Context()),
			r.Method,
			r.URL.RequestURI(),
			loggedWriter.statusCode,
			time.Since(startedAt).Round(time.Millisecond),
			loggedWriter.bytesWritten,
		)
	})
}

type accessLoggingResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (w *accessLoggingResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *accessLoggingResponseWriter) Write(payload []byte) (int, error) {
	w.ensureImplicitStatusCode()

	written, err := w.ResponseWriter.Write(payload)
	w.bytesWritten += written
	return written, err
}

func (w *accessLoggingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return hijacker.Hijack()
}

func (w *accessLoggingResponseWriter) Flush() {
	w.ensureImplicitStatusCode()

	flusher, ok := w.ResponseWriter.(http.Flusher)
	if !ok {
		return
	}
	flusher.Flush()
}

func (w *accessLoggingResponseWriter) Push(target string, opts *http.PushOptions) error {
	pusher, ok := w.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, opts)
}

func (w *accessLoggingResponseWriter) ReadFrom(reader io.Reader) (int64, error) {
	w.ensureImplicitStatusCode()

	if readerFrom, ok := w.ResponseWriter.(io.ReaderFrom); ok {
		written, err := readerFrom.ReadFrom(reader)
		w.bytesWritten += int(written)
		return written, err
	}

	written, err := io.Copy(w.ResponseWriter, reader)
	w.bytesWritten += int(written)
	return written, err
}

func (w *accessLoggingResponseWriter) ensureImplicitStatusCode() {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
}
