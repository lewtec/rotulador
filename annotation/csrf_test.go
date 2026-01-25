package annotation

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCSRFMiddleware(t *testing.T) {
	// Setup a simple handler that returns 200 OK
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap it with CSRF middleware
	handler := CSRFMiddleware(okHandler)

	t.Run("GET request sets cookie and passes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "OK", rec.Body.String())

		// Check for cookie
		cookies := rec.Result().Cookies()
		found := false
		for _, c := range cookies {
			if c.Name == "csrf_token" {
				found = true
				assert.NotEmpty(t, c.Value)
				assert.Equal(t, "/", c.Path)
				assert.True(t, c.HttpOnly)
				assert.Equal(t, http.SameSiteStrictMode, c.SameSite)
			}
		}
		assert.True(t, found, "csrf_token cookie should be set")
	})

	t.Run("POST request without token fails", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rec := httptest.NewRecorder()

		// Simulate existing session with cookie
		cookie := &http.Cookie{
			Name:  "csrf_token",
			Value: "valid-token",
		}
		req.AddCookie(cookie)

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("POST request with invalid token fails", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rec := httptest.NewRecorder()

		cookie := &http.Cookie{
			Name:  "csrf_token",
			Value: "valid-token",
		}
		req.AddCookie(cookie)
		req.Header.Set("X-CSRF-Token", "invalid-token")

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("POST request with valid token passes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rec := httptest.NewRecorder()

		cookie := &http.Cookie{
			Name:  "csrf_token",
			Value: "valid-token",
		}
		req.AddCookie(cookie)
		req.Header.Set("X-CSRF-Token", "valid-token")

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Context has token", func(t *testing.T) {
		// Middleware puts token in context.
		// We need to check it from inside the handler.
		checkCtxHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := GetCSRFToken(r.Context())
			assert.True(t, ok)
			assert.NotEmpty(t, token)
			w.WriteHeader(http.StatusOK)
		})

		h := CSRFMiddleware(checkCtxHandler)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
