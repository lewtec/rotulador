package annotation

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestGetDatabaseEnablesForeignKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fk.db")
	db, err := GetDatabase(path)
	if err != nil {
		t.Fatalf("GetDatabase: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close: %v", err)
		}
	})

	var fk int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatalf("PRAGMA foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Fatalf("foreign_keys = %d, want 1 (enforced on connection)", fk)
	}

	// Schema FK: annotations.image_sha256 → images.sha256
	if _, err := db.Exec(`
		CREATE TABLE images (
			sha256 TEXT PRIMARY KEY,
			filename TEXT NOT NULL
		);
		CREATE TABLE annotations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			image_sha256 TEXT NOT NULL,
			username TEXT NOT NULL,
			stage_index INTEGER NOT NULL,
			option_value TEXT NOT NULL,
			FOREIGN KEY(image_sha256) REFERENCES images(sha256) ON DELETE CASCADE
		);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	_, err = db.Exec(
		`INSERT INTO annotations (image_sha256, username, stage_index, option_value)
		 VALUES ('missing-hash', 'user', 0, 'good')`,
	)
	if err == nil {
		t.Fatal("expected FK violation inserting annotation for missing image")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "foreign key") &&
		!strings.Contains(strings.ToLower(err.Error()), "constraint") {
		t.Fatalf("unexpected error (want foreign key / constraint): %v", err)
	}
}

func TestGetDatabaseBusyTimeoutAndWAL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pragmas.db")
	db, err := GetDatabase(path)
	if err != nil {
		t.Fatalf("GetDatabase: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close: %v", err)
		}
	})

	var timeout int
	if err := db.QueryRow("PRAGMA busy_timeout").Scan(&timeout); err != nil {
		t.Fatalf("PRAGMA busy_timeout: %v", err)
	}
	if timeout != 5000 {
		t.Fatalf("busy_timeout = %d, want 5000", timeout)
	}

	var mode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if !strings.EqualFold(mode, "wal") {
		t.Fatalf("journal_mode = %q, want wal", mode)
	}
}

func TestSqliteDSNMemory(t *testing.T) {
	dsn := sqliteDSN(":memory:")
	if !strings.Contains(dsn, "foreign_keys(1)") {
		t.Fatalf("memory DSN missing foreign_keys: %s", dsn)
	}
	if strings.Contains(dsn, "journal_mode") {
		t.Fatalf("memory DSN should not request WAL: %s", dsn)
	}
}
