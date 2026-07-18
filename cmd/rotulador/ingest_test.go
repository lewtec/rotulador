package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIngestCmd_RejectsZeroJobs(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in")
	out := filepath.Join(dir, "out")
	if err := os.MkdirAll(in, 0o755); err != nil {
		t.Fatal(err)
	}
	// Non-image file is fine: validation must fail before walking.
	if err := os.WriteFile(filepath.Join(in, "note.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Isolate package-level cobra state used by other tests in this package.
	prevJobs := jobs
	t.Cleanup(func() { jobs = prevJobs })

	rootCmd.SetArgs([]string{"ingest", "--jobs", "0", in, out})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := rootCmd.ExecuteContext(ctx)
	if err == nil {
		t.Fatal("expected error for --jobs 0, got nil")
	}
	if !strings.Contains(err.Error(), "--jobs must be at least 1") {
		t.Fatalf("error = %v, want --jobs must be at least 1", err)
	}
}
