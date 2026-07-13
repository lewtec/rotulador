package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/rotulador/annotation"
	"github.com/lewtec/rotulador/db/migrations"
	"github.com/golang-migrate/migrate/v4"
	migrateSqlite "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite"
)

func setupQueryTestDB(t *testing.T) (dbPath string, cleanup func()) {
	t.Helper()
	dir := t.TempDir()
	dbPath = filepath.Join(dir, "annotations.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	driver, err := migrateSqlite.WithInstance(db, &migrateSqlite.Config{})
	if err != nil {
		t.Fatalf("driver: %v", err)
	}
	src, err := iofs.New(migrations.Migrations, ".")
	if err != nil {
		t.Fatalf("iofs: %v", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "sqlite", driver)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("up: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO images (sha256, filename) VALUES
			('abc123', 'photo.jpg'),
			('def456', 'other.png');
		INSERT INTO annotations (image_sha256, username, stage_index, option_value) VALUES
			('abc123', 'admin', 0, 'landscape'),
			('def456', 'admin', 0, 'portrait'),
			('abc123', 'admin', 1, 'good');
	`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close seed db: %v", err)
	}

	return dbPath, func() {}
}

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	err = fn()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String(), err
}

func TestQueryAgainstCurrentSchema(t *testing.T) {
	dbPath, _ := setupQueryTestDB(t)
	ctx := context.Background()

	db, err := annotation.GetDatabase(dbPath)
	if err != nil {
		t.Fatalf("GetDatabase: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("close db: %v", err)
		}
	}()

	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadUncommitted})
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			t.Errorf("rollback: %v", err)
		}
	}()

	t.Run("list stages", func(t *testing.T) {
		out, err := captureStdout(t, func() error {
			return PrintQuery(ctx, tx, "SELECT DISTINCT stage_index FROM annotations ORDER BY stage_index")
		})
		if err != nil {
			t.Fatalf("query: %v", err)
		}
		if !strings.Contains(out, "0") || !strings.Contains(out, "1") {
			t.Fatalf("expected stage indexes, got %q", out)
		}
	})

	t.Run("list option values", func(t *testing.T) {
		out, err := captureStdout(t, func() error {
			return PrintQuery(ctx, tx, "SELECT DISTINCT option_value FROM annotations WHERE stage_index = ?", 0)
		})
		if err != nil {
			t.Fatalf("query: %v", err)
		}
		if !strings.Contains(out, "landscape") || !strings.Contains(out, "portrait") {
			t.Fatalf("expected option values, got %q", out)
		}
	})

	t.Run("join images by sha256 returns filename", func(t *testing.T) {
		out, err := captureStdout(t, func() error {
			return PrintQuery(ctx, tx,
				`SELECT images.filename FROM annotations
				 JOIN images ON annotations.image_sha256 = images.sha256
				 WHERE annotations.stage_index = ? AND annotations.option_value = ?
				 ORDER BY images.filename`,
				0, "landscape")
		})
		if err != nil {
			t.Fatalf("query (would fail on legacy image_id/path columns): %v", err)
		}
		if strings.TrimSpace(out) != "photo.jpg" {
			t.Fatalf("expected photo.jpg, got %q", out)
		}
	})

	t.Run("filter by filename or sha256", func(t *testing.T) {
		out, err := captureStdout(t, func() error {
			return PrintQuery(ctx, tx,
				`SELECT images.sha256 FROM annotations
				 JOIN images ON annotations.image_sha256 = images.sha256
				 WHERE annotations.stage_index = ? AND annotations.option_value = ?
				 AND (images.sha256 = ? OR images.filename = ?)
				 ORDER BY images.filename`,
				0, "landscape", "photo.jpg", "photo.jpg")
		})
		if err != nil {
			t.Fatalf("query: %v", err)
		}
		if strings.TrimSpace(out) != "abc123" {
			t.Fatalf("expected abc123, got %q", out)
		}
	})
}
