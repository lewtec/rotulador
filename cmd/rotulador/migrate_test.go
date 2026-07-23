package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestValidateTaskIDForLegacyTable(t *testing.T) {
	valid := []string{"quality", "has_carro", "task1", "A", "a_b_c", "Stage0"}
	for _, id := range valid {
		if err := validateTaskIDForLegacyTable(id); err != nil {
			t.Errorf("validateTaskIDForLegacyTable(%q) = %v, want nil", id, err)
		}
	}

	invalid := []string{
		"",
		"has-carro",
		"task;drop",
		"task quality",
		"task`x`",
		"../etc",
		"task-name",
		"café",
	}
	for _, id := range invalid {
		if err := validateTaskIDForLegacyTable(id); err == nil {
			t.Errorf("validateTaskIDForLegacyTable(%q) = nil, want error", id)
		}
	}
}

func TestMigrateLegacyDatabase_RejectsUnsafeTaskID(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old.db")
	newPath := filepath.Join(dir, "new.db")
	configPath := filepath.Join(dir, "config.yaml")

	// Malicious-looking id that would be interpolated into FROM task_<id>
	if err := os.WriteFile(configPath, []byte(`
meta:
  description: test
tasks:
  - id: "evil;drop"
    name: Evil
`), 0o644); err != nil {
		t.Fatal(err)
	}

	oldDB, err := sql.Open("sqlite", oldPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = oldDB.Exec(`
		CREATE TABLE images (sha256 TEXT PRIMARY KEY, filename TEXT);
		INSERT INTO images (sha256, filename) VALUES ('abc', 'a.png');
	`)
	if err != nil {
		t.Fatal(err)
	}
	if err := oldDB.Close(); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	err = migrateLegacyDatabase(context.Background(), oldPath, newPath, configPath, logger)
	if err == nil {
		t.Fatal("expected error for unsafe task id, got nil")
	}
	if !strings.Contains(err.Error(), "safe SQL identifier") {
		t.Fatalf("error = %v, want mention of safe SQL identifier", err)
	}
	// Validation runs before the new DB is opened, so the target path must stay absent.
	if _, statErr := os.Stat(newPath); !os.IsNotExist(statErr) {
		t.Fatalf("new database should not be created on validation failure, stat err=%v", statErr)
	}
}

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
