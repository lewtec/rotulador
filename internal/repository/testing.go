package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/lewtec/rotulador/db/migrations"
	_ "modernc.org/sqlite"
)

// SetupTestDB creates an in-memory SQLite database for testing
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// Create schema from migration
	schemaBytes, err := migrations.Migrations.ReadFile("20240101000000_initial_schema.up.sql")
	if err != nil {
		t.Fatalf("failed to read migration file: %v", err)
	}

	_, err = db.Exec(string(schemaBytes))
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
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
func MustExec(t *testing.T, db *sql.DB, query string, args ...interface{}) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), query, args...)
	if err != nil {
		t.Fatalf("failed to exec query: %v", err)
	}
}
