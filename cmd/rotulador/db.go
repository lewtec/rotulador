package main

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lewtec/rotulador/internal/web"
)

func closeDatabase(ctx context.Context, db interface{ Close() error }) {
	if err := db.Close(); err != nil {
		web.ReportError(ctx, err, "msg", "failed to close database")
	}
}

func rollbackTx(ctx context.Context, tx interface{ Rollback() error }) {
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		web.ReportError(ctx, err, "msg", "failed to rollback transaction")
	}
}
