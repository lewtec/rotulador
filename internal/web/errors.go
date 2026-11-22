package web

import (
	"context"
	"io"
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

// reportClose closes c and reports a close failure. The close error is not returned.
// args are slog key/value pairs, same as ReportError, including "msg".
func reportClose(ctx context.Context, c io.Closer, args ...any) {
	if err := c.Close(); err != nil {
		ReportError(ctx, err, args...)
	}
}
