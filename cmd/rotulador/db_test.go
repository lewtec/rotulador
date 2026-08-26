package main

import (
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
