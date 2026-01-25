package annotation

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
)

type csrfTokenKey struct{}

// CSRFMiddleware implements the Double Submit Cookie pattern for CSRF protection.
// It generates a token and sets it as a cookie if missing.
// It verifies the token in the X-CSRF-Token header against the cookie for state-changing requests.
func CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Get or generate token
		token := ""
		cookie, err := r.Cookie("csrf_token")
		if err == nil {
			token = cookie.Value
		}

		if token == "" {
			// Generate new token
			randomBytes := make([]byte, 32)
			_, err := rand.Read(randomBytes)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			token = base64.StdEncoding.EncodeToString(randomBytes)

			// Set cookie
			http.SetCookie(w, &http.Cookie{
				Name:     "csrf_token",
				Value:    token,
				Path:     "/",
				HttpOnly: true,                   // Prevent JS access to cookie (we inject token via template)
				SameSite: http.SameSiteStrictMode, // Strict CSRF protection
			})
		}

		// 2. Add token to context so templates can access it
		ctx := context.WithValue(r.Context(), csrfTokenKey{}, token)
		r = r.WithContext(ctx)

		// 3. Verify on state-changing methods
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete || r.Method == http.MethodPatch {
			headerToken := r.Header.Get("X-CSRF-Token")
			if headerToken == "" {
				http.Error(w, "Forbidden - CSRF token missing", http.StatusForbidden)
				return
			}

			if subtle.ConstantTimeCompare([]byte(token), []byte(headerToken)) != 1 {
				http.Error(w, "Forbidden - CSRF token invalid", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// GetCSRFToken retrieves the CSRF token from the context.
func GetCSRFToken(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(csrfTokenKey{}).(string)
	return token, ok
}
