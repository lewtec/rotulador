package annotation

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
)

const csrfTokenKey contextKey = "csrf_token"
const csrfCookieName = "csrf_token"

// GenerateCSRFToken generates a secure random token.
func GenerateCSRFToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// CSRFMiddleware adds CSRF protection using the Double Submit Cookie pattern.
func CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// retrieve or create token from cookie
		token := ""
		cookie, err := r.Cookie(csrfCookieName)
		if err == nil {
			token = cookie.Value
		}

		// If no token exists, generate one
		if token == "" {
			var err error
			token, err = GenerateCSRFToken()
			if err != nil {
				ReportError(r.Context(), err, "msg", "failed to generate csrf token")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			// Set the cookie
			http.SetCookie(w, &http.Cookie{
				Name:     csrfCookieName,
				Value:    token,
				Path:     "/",
				HttpOnly: true, // JS reads from meta tag, not cookie
				SameSite: http.SameSiteStrictMode,
				Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
			})
		}

		// validate for unsafe methods
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete || r.Method == http.MethodPatch {
			submittedToken := r.Header.Get("X-CSRF-Token")
			if submittedToken == "" {
				submittedToken = r.FormValue("csrf_token")
			}
			// Use constant time compare to prevent timing attacks
			if subtle.ConstantTimeCompare([]byte(submittedToken), []byte(token)) != 1 {
				ReportError(r.Context(), nil, "msg", "csrf token mismatch", "method", r.Method, "path", r.URL.Path)
				http.Error(w, "Forbidden - CSRF token mismatch", http.StatusForbidden)
				return
			}
		}

		// add token to context so templates can render it
		ctx := context.WithValue(r.Context(), csrfTokenKey, token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetCSRFToken retrieves the CSRF token from the context.
func GetCSRFToken(ctx context.Context) string {
	token, ok := ctx.Value(csrfTokenKey).(string)
	if !ok {
		return ""
	}
	return token
}
