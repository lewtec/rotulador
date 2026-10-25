package main

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lewtec/rotulador/internal/web"
)

func closeAndReport(ctx context.Context, c interface{ Close() error }, msg string) {
	if err := c.Close(); err != nil {
		web.ReportError(ctx, err, "msg", msg)
	}
}

func closeDatabase(ctx context.Context, db interface{ Close() error }) {
	closeAndReport(ctx, db, "failed to close database")
}

func rollbackTx(ctx context.Context, tx interface{ Rollback() error }) {
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		web.ReportError(ctx, err, "msg", "failed to rollback transaction")
	}
}
