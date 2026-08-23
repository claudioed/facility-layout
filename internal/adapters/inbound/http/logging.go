package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// RequestLogger returns chi middleware that logs one structured line per
// request: method, path, status, duration, response size, and the chi
// request ID (so a request can be traced across the router's own
// middleware.Recoverer output and the application logs). Responses with a
// 5xx status are logged at Error; everything else at Info.
//
// The line is logged with the request's context, so when the tracing
// middleware upstream has opened a span the record also carries its
// trace_id/span_id — that is what joins this log line to its trace.
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()

			next.ServeHTTP(ww, r)

			duration := time.Since(start)
			attrs := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"duration_ms", duration.Milliseconds(),
				"bytes", ww.BytesWritten(),
				"request_id", middleware.GetReqID(r.Context()),
			}

			if ww.Status() >= http.StatusInternalServerError {
				logger.ErrorContext(r.Context(), "http request", attrs...)
				return
			}
			logger.InfoContext(r.Context(), "http request", attrs...)
		})
	}
}
