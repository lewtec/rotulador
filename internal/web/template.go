package web

import (
	"context"
	"embed"
	"html/template"
	"io"
	"net/http"

	"github.com/lewtec/rotulador/internal/i18n"
	"github.com/russross/blackfriday/v2"
)

var (
	//go:embed templates/*
	templateFS embed.FS

	//go:embed assets/css/output.css
	cssContent string

	//go:embed assets/favicon.svg
	faviconContent string

	// Template manager with mold for layout support
	templateManager *TemplateManager = nil

	// TemplateFuncMap contains custom template functions available globally
	TemplateFuncMap = template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"i":   i18n.T, // Internationalization function (uses goroutine-local localizer)
		"markdown": func(text string) template.HTML {
			// Convert markdown to HTML using blackfriday v2
			return template.HTML(blackfriday.Run([]byte(text)))
		},
	}
)

func init() {
	// Initialize template manager with mold
	// Mold will automatically parse all templates from the embed.FS
	var err error
	templateManager, err = NewTemplateManagerWithFuncMap(templateFS, TemplateFuncMap)
	if err != nil {
		panic(err)
	}
}

// RenderPageWithContext renders a page with context-aware i18n
func RenderPageWithContext(ctx context.Context, w io.Writer, pageName string, data map[string]any) error {
	// Inject CSS automatically
	if data == nil {
		data = make(map[string]any)
	}
	data["CSS"] = template.CSS(cssContent)

	// Set goroutine-local localizer for the i18n function in templates
	unbind := i18n.BindLocalizer(i18n.GetLocalizerFromContext(ctx))
	defer unbind()

	return templateManager.Render(w, "pages/"+pageName, data)
}

// RenderPageWithRequest renders a page with request-aware i18n
// ALWAYS use this function for rendering pages to ensure proper i18n support
func RenderPageWithRequest(r *http.Request, w io.Writer, pageName string, data map[string]any) error {
	return RenderPageWithContext(r.Context(), w, pageName, data)
}

// GetFavicon returns the embedded favicon content
func GetFavicon() string {
	return faviconContent
}
