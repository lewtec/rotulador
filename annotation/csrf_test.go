package annotation

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCSRFMiddleware(t *testing.T) {
	// Create a dummy handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with CSRF middleware
	csrfHandler := CSRFMiddleware(handler)

	t.Run("GET request sets cookie and succeeds", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		csrfHandler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		// Check for cookie
		cookies := rec.Result().Cookies()
		require.NotEmpty(t, cookies)
		found := false
		for _, c := range cookies {
			if c.Name == "csrf_token" {
				found = true
				assert.NotEmpty(t, c.Value)
				assert.True(t, c.HttpOnly)
				assert.Equal(t, http.SameSiteStrictMode, c.SameSite)
			}
		}
		assert.True(t, found, "CSRF cookie not found")
	})

	t.Run("POST request without token fails", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rec := httptest.NewRecorder()

		// We need to set the cookie first, otherwise middleware will generate one and fail comparison with empty header
		// Wait, if no cookie is sent in request, middleware generates a new one.
		// Then it compares header with that new token.
		// Header is empty. Token is not empty. Mismatch.

		csrfHandler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("POST request with matching token succeeds", func(t *testing.T) {
		// 1. Get a token first by making a GET request
		reqGet := httptest.NewRequest(http.MethodGet, "/", nil)
		recGet := httptest.NewRecorder()
		csrfHandler.ServeHTTP(recGet, reqGet)

		cookies := recGet.Result().Cookies()
		var csrfCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "csrf_token" {
				csrfCookie = c
				break
			}
		}
		require.NotNil(t, csrfCookie)
		token := csrfCookie.Value

		// 2. Make POST request with token in header and cookie
		reqPost := httptest.NewRequest(http.MethodPost, "/", nil)
		reqPost.AddCookie(csrfCookie)
		reqPost.Header.Set("X-CSRF-Token", token)

		recPost := httptest.NewRecorder()
		csrfHandler.ServeHTTP(recPost, reqPost)

		assert.Equal(t, http.StatusOK, recPost.Code)
	})

	t.Run("POST request with mismatch token fails", func(t *testing.T) {
		// 1. Get a token first
		reqGet := httptest.NewRequest(http.MethodGet, "/", nil)
		recGet := httptest.NewRecorder()
		csrfHandler.ServeHTTP(recGet, reqGet)

		cookies := recGet.Result().Cookies()
		var csrfCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "csrf_token" {
				csrfCookie = c
				break
			}
		}
		require.NotNil(t, csrfCookie)

		// 2. Make POST request with WRONG token in header
		reqPost := httptest.NewRequest(http.MethodPost, "/", nil)
		reqPost.AddCookie(csrfCookie)
		reqPost.Header.Set("X-CSRF-Token", "wrong-token")

		recPost := httptest.NewRecorder()
		csrfHandler.ServeHTTP(recPost, reqPost)

		assert.Equal(t, http.StatusForbidden, recPost.Code)
	})
}
