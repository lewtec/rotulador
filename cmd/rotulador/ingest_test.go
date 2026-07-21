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

// TestIngestCmd_WalkErrorDoesNotHang ensures a WalkDir failure returns an
// error instead of deadlocking on worker Wait with an unclosed channel.
func TestIngestCmd_WalkErrorDoesNotHang(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in")
	out := filepath.Join(dir, "out")
	if err := os.MkdirAll(in, 0o755); err != nil {
		t.Fatal(err)
	}

	// Unreadable subdirectory: WalkDir fails when it tries to enter it.
	locked := filepath.Join(in, "locked")
	if err := os.Mkdir(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(locked, 0o755)
	})

	prevJobs := jobs
	t.Cleanup(func() { jobs = prevJobs })

	rootCmd.SetArgs([]string{"ingest", "--jobs", "1", in, out})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := rootCmd.ExecuteContext(ctx)
	if err == nil {
		// Root or elevated environments may still read mode 000 dirs; skip.
		if _, statErr := os.ReadDir(locked); statErr == nil {
			t.Skip("environment can read mode 000 directories; cannot force WalkDir error")
		}
		t.Fatal("expected walk error, got nil")
	}
	if ctx.Err() != nil {
		t.Fatalf("ingest hung until context deadline: %v (walk err: %v)", ctx.Err(), err)
	}
	if !strings.Contains(err.Error(), "walking input directory") {
		t.Fatalf("error = %v, want walking input directory", err)
	}
}
