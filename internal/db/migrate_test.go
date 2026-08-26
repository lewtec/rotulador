package db

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestRunMigrationsAppliesSchemaAndIsIdempotent(t *testing.T) {
	conn, err := sql.Open("sqlite", SQLiteOpenDSN(":memory:"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Errorf("close: %v", err)
		}
	})

	if err := RunMigrations(conn); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	var n int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='images'`).Scan(&n); err != nil {
		t.Fatalf("images table lookup: %v", err)
	}
	if n != 1 {
		t.Fatalf("images table count = %d, want 1", n)
	}

	if err := RunMigrations(conn); err != nil {
		t.Fatalf("second RunMigrations: %v", err)
	}
}
