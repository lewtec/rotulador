package main

import (
	"database/sql"
	"errors"
	"log/slog"
	"testing"
)

var errCloseBoom = errors.New("boom")

type stubCloser struct {
	err    error
	closed int
}

func (s *stubCloser) Close() error {
	s.closed++
	return s.err
}

type stubTx struct {
	err       error
	rollbacks int
}

func (s *stubTx) Rollback() error {
	s.rollbacks++
	return s.err
}

func TestRollbackTx(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		tx := &stubTx{}
		rollbackTx(t.Context(), tx)
		if tx.rollbacks != 1 {
			t.Fatalf("Rollback() calls = %d, want 1", tx.rollbacks)
		}
	})

	t.Run("ignores already committed", func(t *testing.T) {
		t.Parallel()
		tx := &stubTx{err: sql.ErrTxDone}
		rollbackTx(t.Context(), tx)
		if tx.rollbacks != 1 {
			t.Fatalf("Rollback() calls = %d, want 1", tx.rollbacks)
		}
	})

	t.Run("reports rollback error", func(t *testing.T) {
		t.Parallel()
		prev := slog.Default()
		slog.SetDefault(slog.New(slog.DiscardHandler))
		t.Cleanup(func() { slog.SetDefault(prev) })

		tx := &stubTx{err: errCloseBoom}
		rollbackTx(t.Context(), tx)
		if tx.rollbacks != 1 {
			t.Fatalf("Rollback() calls = %d, want 1", tx.rollbacks)
		}
	})
}

func TestCloseDatabase(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		db := &stubCloser{}
		closeDatabase(t.Context(), db)
		if db.closed != 1 {
			t.Fatalf("Close() calls = %d, want 1", db.closed)
		}
	})

	t.Run("reports close error", func(t *testing.T) {
		t.Parallel()
		// ReportError writes through the default slog logger; keep the
		// test from leaking "boom" onto the suite's stderr.
		prev := slog.Default()
		slog.SetDefault(slog.New(slog.DiscardHandler))
		t.Cleanup(func() { slog.SetDefault(prev) })

		db := &stubCloser{err: errCloseBoom}
		closeDatabase(t.Context(), db)
		if db.closed != 1 {
			t.Fatalf("Close() calls = %d, want 1", db.closed)
		}
	})
}
