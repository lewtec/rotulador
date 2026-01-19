package repository

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// SetupTestDB creates an in-memory SQLite database for testing
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// Create schema
	schema := `
CREATE TABLE images (
  sha256 TEXT PRIMARY KEY,
  filename TEXT NOT NULL,
  ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE annotations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  image_sha256 TEXT NOT NULL,
  username TEXT NOT NULL,
  stage_index INTEGER NOT NULL,
  option_value TEXT NOT NULL,
  annotated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(image_sha256, username, stage_index),
  FOREIGN KEY(image_sha256) REFERENCES images(sha256) ON DELETE CASCADE
);

CREATE INDEX idx_annotations_image_sha256 ON annotations(image_sha256);
CREATE INDEX idx_annotations_username ON annotations(username);
CREATE INDEX idx_annotations_stage ON annotations(stage_index);
`

	_, err = db.Exec(schema)
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
