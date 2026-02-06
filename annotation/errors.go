package annotation

import (
	"context"
	"log/slog"
	"runtime/debug"
)

// ReportError centralizes error reporting.
// It logs the error using slog and can be extended to send to Sentry.
func ReportError(ctx context.Context, err error, args ...any) {
	if err == nil {
		return
	}

	// Add stack trace to the arguments
	allArgs := append([]any{"stack", string(debug.Stack())}, args...)

	// Log using slog
	slog.ErrorContext(ctx, err.Error(), allArgs...)
}
