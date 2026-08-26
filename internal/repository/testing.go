package repository

import (
	"database/sql"
	"testing"

	appdb "github.com/lewtec/rotulador/internal/db"
	_ "modernc.org/sqlite"
)

// SetupTestDB creates an in-memory SQLite database for testing
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", appdb.SQLiteOpenDSN(":memory:"))
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := appdb.RunMigrations(db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return db
}

// CleanupTestDB closes the test database
func CleanupTestDB(t *testing.T, db *sql.DB) {
	t.Helper()
	if err := db.Close(); err != nil {
		t.Errorf("failed to close test database: %v", err)
	}
}

// MustExec executes a SQL statement and fails the test if it errors
func MustExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	_, err := db.ExecContext(t.Context(), query, args...)
	if err != nil {
		t.Fatalf("failed to exec query: %v", err)
	}
}
