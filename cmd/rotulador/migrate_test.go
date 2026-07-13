package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestMigrateLegacyDatabase_CurrentSchema(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old.db")
	newPath := filepath.Join(dir, "new.db")
	configPath := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(configPath, []byte(`
meta:
  description: test
tasks:
  - id: quality
    name: Quality
`), 0o644); err != nil {
		t.Fatal(err)
	}

	oldDB, err := sql.Open("sqlite", oldPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = oldDB.Exec(`
		CREATE TABLE images (sha256 TEXT PRIMARY KEY, filename TEXT);
		CREATE TABLE task_quality (image TEXT, user TEXT, value TEXT);
		INSERT INTO images (sha256, filename) VALUES ('abc', 'a.png');
		INSERT INTO task_quality (image, user, value) VALUES ('abc', 'admin', 'good');
	`)
	if err != nil {
		t.Fatal(err)
	}
	if err := oldDB.Close(); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	if err := migrateLegacyDatabase(context.Background(), oldPath, newPath, configPath, logger); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	newDB, err := sql.Open("sqlite", newPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := newDB.Close(); err != nil {
			t.Errorf("close new db: %v", err)
		}
	}()

	var filename string
	if err := newDB.QueryRow(`SELECT filename FROM images WHERE sha256 = ?`, "abc").Scan(&filename); err != nil {
		t.Fatalf("image row: %v", err)
	}
	if filename != "a.png" {
		t.Fatalf("filename = %q", filename)
	}

	var value string
	if err := newDB.QueryRow(
		`SELECT option_value FROM annotations WHERE image_sha256 = ? AND username = ? AND stage_index = ?`,
		"abc", "admin", 0,
	).Scan(&value); err != nil {
		t.Fatalf("annotation row: %v", err)
	}
	if value != "good" {
		t.Fatalf("value = %q", value)
	}
}
