package annotation

import (
	"database/sql"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestPathTraversalVulnerability(t *testing.T) {
	// 1. Setup temporary directory structure
	tmpDir := t.TempDir()
	imagesDir := filepath.Join(tmpDir, "images")
	if err := os.Mkdir(imagesDir, 0755); err != nil {
		t.Fatalf("failed to create images dir: %v", err)
	}

	// Create a secret file outside images dir
	secretFile := filepath.Join(tmpDir, "secret.txt")
	secretContent := "This is a secret!"
	if err := os.WriteFile(secretFile, []byte(secretContent), 0644); err != nil {
		t.Fatalf("failed to create secret file: %v", err)
	}

	// 2. Setup in-memory database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// Create schema manually
	_, err = db.Exec(`
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
	`)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	// 3. Insert malicious record
	// Pointing to ../secret.txt
	maliciousFilename := "../secret.txt"
	maliciousHash := "fakehash"
	_, err = db.Exec("INSERT INTO images (sha256, filename) VALUES (?, ?)", maliciousHash, maliciousFilename)
	if err != nil {
		t.Fatalf("failed to insert malicious record: %v", err)
	}

	// 4. Initialize App with Auth
	password := "password"
	hash, _ := HashPassword(password)
	authConfig := map[string]*ConfigAuth{
		"admin": {Password: hash},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	app := &AnnotatorApp{
		ImagesDir: imagesDir,
		Database:  db,
		Config:    &Config{Tasks: []*ConfigTask{}, Authentication: authConfig},
		Logger:    logger,
	}

	// 5. Test the handler
	req := httptest.NewRequest("GET", "/asset/"+maliciousHash, nil)
	req.SetBasicAuth("admin", password)
	w := httptest.NewRecorder()

	handler := app.GetHTTPHandler()
	handler.ServeHTTP(w, req)

	resp := w.Result()

	// 6. Assertions
	if resp.StatusCode == http.StatusOK {
		body := w.Body.String()
		if body == secretContent {
			t.Fatalf("VULNERABILITY DETECTED: Managed to read secret file content: %s", body)
		}
	}
}
