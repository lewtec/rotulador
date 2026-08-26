package main

import (
	"context"

	"github.com/lewtec/rotulador/internal/web"
)

func closeDatabase(ctx context.Context, db interface{ Close() error }) {
	if err := db.Close(); err != nil {
		web.ReportError(ctx, err, "msg", "failed to close database")
	}
}
