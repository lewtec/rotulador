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

func HTTPLogger(handler http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		initialTime := time.Now()
		method := r.Method
		path := r.URL.String()
		wr := NewStatusCodeRecorderResponseWriter(w)
		handler.ServeHTTP(wr, r)
		finalTime := time.Now()
		statusCode := wr.Status
		duration := finalTime.Sub(initialTime)

		logger.Info("http request",
			"method", method,
			"path", path,
			"status", statusCode,
			"duration", duration,
		)
	})
}

type StatusCodeRecorderResponseWriter struct {
	http.ResponseWriter
	Status int
}

func (r *StatusCodeRecorderResponseWriter) WriteHeader(status int) {
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}

func NewStatusCodeRecorderResponseWriter(w http.ResponseWriter) *StatusCodeRecorderResponseWriter {
	return &StatusCodeRecorderResponseWriter{ResponseWriter: w, Status: 200}
}
