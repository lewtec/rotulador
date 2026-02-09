package annotation

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCSRFTokenInjection(t *testing.T) {
	// Initialize context with a CSRF token
	token := "test-csrf-token-123"
	ctx := context.WithValue(context.Background(), csrfTokenKey, token)

	// Capture the output
	var buf bytes.Buffer

	// Render the home page (which uses layout.html)
	// Note: this relies on init() in template.go having run, which loads templates from embed.FS
	err := RenderPageWithContext(ctx, &buf, "home.html", map[string]any{
		"Title": "Test Home",
		"Description": "Test Description",
	})

	if err != nil {
		t.Fatalf("Failed to render page: %v", err)
	}

	html := buf.String()

	// Check for the meta tag
	expectedMeta := `<meta name="csrf-token" content="test-csrf-token-123">`
	assert.Contains(t, html, expectedMeta, "CSRF meta tag not found in rendered HTML")

	// Check for the script
	expectedScript := `evt.detail.headers['X-CSRF-Token'] = csrfToken;`
	assert.Contains(t, html, expectedScript, "CSRF HTMX script not found in rendered HTML")
}
