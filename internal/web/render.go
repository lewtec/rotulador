package web

import (
	"context"
	"net/http"

	"github.com/a-h/templ"
	"github.com/lewtec/rotulador/internal/ui/components"
)

// Render writes a templ component as text/html.
func Render(ctx context.Context, w http.ResponseWriter, c templ.Component) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return c.Render(ctx, w)
}

// ProgressUI maps PhaseProgress into the UI progress bar model.
func ProgressUI(p *PhaseProgress) *components.Progress {
	if p == nil {
		return nil
	}
	return &components.Progress{
		Completed:              p.Completed,
		Pending:                p.Pending,
		FilteredWrongClass:     p.FilteredWrongClass,
		NotYetAnnotated:        p.NotYetAnnotated,
		Total:                  p.Total,
		CompletedPercent:       p.CompletedPercent,
		PendingPercent:         p.PendingPercent,
		FilteredPercent:        p.FilteredPercent,
		NotYetAnnotatedPercent: p.NotYetAnnotatedPercent,
	}
}
