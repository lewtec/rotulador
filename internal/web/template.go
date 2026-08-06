package web

import (
	"crypto/sha256"
	"encoding/hex"
	_ "embed"
	"fmt"

	"github.com/lewtec/rotulador/internal/ui/layout"
)

//go:embed assets/css/output.css
var cssContent string

//go:embed assets/favicon.svg
var faviconContent string

// cssETag is a short content hash for cache-busting the stylesheet URL.
var cssETag = hashCSS(cssContent)

func hashCSS(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8])
}

// CSS returns the embedded stylesheet.
func CSS() string {
	return cssContent
}

// StylesheetHref is the cache-busted public URL for the embedded CSS.
// The query param changes whenever output.css content changes so long-lived
// Cache-Control does not leave clients on a stale layout.
func StylesheetHref() string {
	return fmt.Sprintf("/static/style.css?v=%s", cssETag)
}

// PageShell builds layout.ShellProps with title and the cache-busted stylesheet.
func PageShell(title string) layout.ShellProps {
	return layout.ShellProps{Title: title, Stylesheet: StylesheetHref()}
}

// GetFavicon returns the embedded favicon content.
func GetFavicon() string {
	return faviconContent
}
