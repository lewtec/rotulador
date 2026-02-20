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

func (a *AnnotatorApp) authenticationMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			var item *ConfigAuth
			item, ok = a.Config.Authentication[username]
			if ok {
				// SECURITY: Use bcrypt to compare the provided password with the stored hash.
				if CheckPasswordHash(password, item.Password) {
					a.Logger.Info("auth for user: success", "username", username)
					handler.ServeHTTP(w, r)
					return
				}
				a.Logger.Warn("auth for user: bad password", "username", username)
			} else {
				a.Logger.Warn("auth for user: no such user", "username", username)
			}
		} else {
			a.Logger.Warn("auth: no credentials provided")
		}
		a.Logger.Warn("auth: not ok")
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}
