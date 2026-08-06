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

// ProgressUI maps PhaseProgress into a list of progress-bar segments.
func ProgressUI(p *PhaseProgress) *components.Progress {
	if p == nil {
		return nil
	}
	return &components.Progress{
		Total: p.Total,
		Segments: []components.ProgressSegment{
			{
				Count:   p.Completed,
				Percent: p.CompletedPercent,
				Label:   "completed",
				Class:   "bg-success text-success-content text-xs font-bold",
			},
			{
				Count:   p.Pending,
				Percent: p.PendingPercent,
				Label:   "pending",
				Class:   "bg-info text-info-content text-xs font-bold",
			},
			{
				Count:   p.NotYetAnnotated,
				Percent: p.NotYetAnnotatedPercent,
				Label:   "not yet annotated in previous phase",
				Class:   "bg-base-300 text-base-content text-xs",
			},
			{
				Count:   p.FilteredWrongClass,
				Percent: p.FilteredPercent,
				Label:   "annotated with wrong class in previous phase",
				Class:   "bg-error text-error-content text-xs",
			},
		},
	}
}
