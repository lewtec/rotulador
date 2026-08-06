// Package tools holds module-level go:generate directives.
// Working directory for these is the module root (this directory).
package tools

//go:generate npm install
//go:generate npm run build:css
//go:generate go tool sqlc generate
// goi18n extract/merge is driven separately via mise until templ lands.
