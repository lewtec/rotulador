package annotation

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestGetDatabaseAppliesForeignKeys(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := GetDatabase(dbPath)
	if err != nil {
		t.Fatalf("GetDatabase() error = %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}()

	var fk int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatalf("PRAGMA foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Fatalf("foreign_keys = %d, want 1", fk)
	}

	var busy int
	if err := db.QueryRow("PRAGMA busy_timeout").Scan(&busy); err != nil {
		t.Fatalf("PRAGMA busy_timeout: %v", err)
	}
	if busy != 5000 {
		t.Fatalf("busy_timeout = %d, want 5000", busy)
	}
}

func TestGetDatabaseEnforcesForeignKeysOnNewConnections(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "fk.db")

	db, err := GetDatabase(dbPath)
	if err != nil {
		t.Fatalf("GetDatabase() error = %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}()

	// Allow multiple connections from the pool so we exercise DSN PRAGMAs
	// rather than a single sticky connection.
	db.SetMaxOpenConns(4)

	if _, err := db.Exec(`
		CREATE TABLE parent (id INTEGER PRIMARY KEY);
		CREATE TABLE child (
			id INTEGER PRIMARY KEY,
			parent_id INTEGER NOT NULL,
			FOREIGN KEY(parent_id) REFERENCES parent(id)
		);
	`); err != nil {
		t.Fatalf("create tables: %v", err)
	}

	// Each Exec may use a different pooled connection; FK must still be on.
	_, err = db.Exec(`INSERT INTO child (id, parent_id) VALUES (1, 999)`)
	if err == nil {
		t.Fatal("expected foreign key violation, got nil error")
	}
}

func TestSqliteOpenDSNIncludesPragmas(t *testing.T) {
	dsn := sqliteOpenDSN("/tmp/x.db")
	if dsn == "/tmp/x.db" {
		t.Fatal("DSN should include query parameters")
	}
	for _, want := range []string{"_pragma", "foreign_keys", "busy_timeout", "journal_mode"} {
		if !strings.Contains(dsn, want) {
			t.Errorf("DSN %q missing %q", dsn, want)
		}
	}
}
