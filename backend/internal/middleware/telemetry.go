package middleware

import (
	"net/http"
	"time"

	"github.com/shahariaazam/smart-insights/internal/telemetry"
)

func TelemetryMiddleware(tel *telemetry.Telemetry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startTime := time.Now()

			// Create a custom response writer to capture the status code
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Start a new span
			ctx, span := tel.StartSpan(r.Context(), "http_request")
			defer span.End()

			// Pass the context with the span to the next handler
			next.ServeHTTP(rw, r.WithContext(ctx))

			// Record metrics after the request is handled
			duration := time.Since(startTime)
			tel.RecordRequest(r.Method, r.URL.Path, rw.statusCode, duration)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
