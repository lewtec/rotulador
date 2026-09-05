// Package tools holds module-level go:generate directives.
// Working directory for these is the module root (this directory).
package tools

//go:generate go tool tailwind -i internal/ui/input.css -o internal/web/assets/css/output.css
//go:generate go tool sqlc generate
//go:generate go tool templ generate
//go:generate go tool goi18n extract -format json -outdir .
//go:generate go tool goi18n merge -format json -outdir internal/i18n/locales active.en.json internal/i18n/locales/en.json internal/i18n/locales/pt-BR.json
//go:generate rm -f active.en.json
