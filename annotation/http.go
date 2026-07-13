package annotation

import (
	"log/slog"
	"net/http"
	"time"
)

// i18nMiddleware adds the appropriate localizer to the request context
func i18nMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		localizer := GetLocalizerFromRequest(r)
		ctx := WithLocalizer(r.Context(), localizer)
		handler.ServeHTTP(w, r.WithContext(ctx))
	})
}

// HTTPLogger provides a middleware to log HTTP requests using a structured logger.
type HTTPLogger struct {
	logger *slog.Logger
}

// NewHTTPLogger creates a new HTTPLogger middleware.
func NewHTTPLogger(logger *slog.Logger) *HTTPLogger {
	return &HTTPLogger{logger: logger}
}

// Middleware returns the HTTP handling middleware.
func (l *HTTPLogger) Middleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		initialTime := time.Now()
		method := r.Method
		path := r.URL.String()
		wr := NewStatusCodeRecorderResponseWriter(w)
		handler.ServeHTTP(wr, r)
		finalTime := time.Now()
		statusCode := wr.Status
		duration := finalTime.Sub(initialTime)

		l.logger.Info("http request",
			"method", method,
			"path", path,
			"status", statusCode,
			"duration", duration,
		)
	})
}

// StatusCodeRecorderResponseWriter records the HTTP status code written by a handler.
// It defaults to 200 and captures the implicit 200 from Write when WriteHeader was not called.
type StatusCodeRecorderResponseWriter struct {
	http.ResponseWriter
	Status      int
	wroteHeader bool
}

func (r *StatusCodeRecorderResponseWriter) WriteHeader(status int) {
	if r.wroteHeader {
		return
	}
	r.wroteHeader = true
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *StatusCodeRecorderResponseWriter) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(b)
}

// Unwrap exposes the underlying ResponseWriter for http.ResponseController and friends.
func (r *StatusCodeRecorderResponseWriter) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func NewStatusCodeRecorderResponseWriter(w http.ResponseWriter) *StatusCodeRecorderResponseWriter {
	return &StatusCodeRecorderResponseWriter{ResponseWriter: w, Status: http.StatusOK}
}
