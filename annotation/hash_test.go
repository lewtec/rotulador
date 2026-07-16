package annotation

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.bin")
	content := []byte("rotulador-hash-fixture")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	got, err := HashFile(path)
	if err != nil {
		t.Fatalf("HashFile: %v", err)
	}

	want := fmt.Sprintf("%x", sha256.Sum256(content))
	if got != want {
		t.Fatalf("HashFile digest = %s, want %s", got, want)
	}
}

func TestHashFileMissing(t *testing.T) {
	_, err := HashFile(filepath.Join(t.TempDir(), "missing.bin"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
