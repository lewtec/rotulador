package layout

import "github.com/a-h/templ"

// ShellProps is chrome shared by product pages.
type ShellProps struct {
	Title string
	// Stylesheet is the CSS href (include ?v= hash for cache busting). Empty uses /static/style.css.
	Stylesheet string
}

// StylesheetHref returns the stylesheet link target.
func (p ShellProps) StylesheetHref() string {
	if p.Stylesheet != "" {
		return p.Stylesheet
	}
	return "/static/style.css"
}

// Crumb is one breadcrumb entry.
type Crumb struct {
	Label string
	Href  string // empty = current (non-link)
}

// PageHeaderProps is the standard page title block.
type PageHeaderProps struct {
	Title      string
	Lead       string
	Crumbs     []Crumb
	HasActions bool
	// Compact uses denser type/spacing (annotate and other long-session surfaces).
	Compact bool
	// ActionsAfterTitle puts action children next to the title instead of the right rail.
	ActionsAfterTitle bool
	TitleAfter        templ.Component
}

// Shared daisyUI class strings for PageHeader action children.
const (
	// HeaderBtnPrimary — at most one main CTA.
	HeaderBtnPrimary = "btn btn-sm btn-primary"
	// HeaderBtn — secondary tools (help, related pages).
	HeaderBtn = "btn btn-sm"
)
