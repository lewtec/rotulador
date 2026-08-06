package web

import (
	_ "embed"
)

//go:embed assets/css/output.css
var cssContent string

//go:embed assets/favicon.svg
var faviconContent string

// CSS returns the embedded stylesheet bytes.
func CSS() string {
	return cssContent
}

// GetFavicon returns the embedded favicon content.
func GetFavicon() string {
	return faviconContent
}
