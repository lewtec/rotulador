package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// executeCommand is a helper to run a cobra command and capture its output
func executeCommand(args ...string) (string, string, error) {
	// Redirect log output for capture
	var out, errOut bytes.Buffer
	log.SetOutput(&errOut)
	defer log.SetOutput(os.Stderr) // Restore default logger

	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)
	rootCmd.SetArgs(args)

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := rootCmd.ExecuteContext(ctx) // Use ExecuteContext

	return out.String(), errOut.String(), err
}

func TestRootCmd_SingleArgument(t *testing.T) {
	t.Run("when argument is a directory, creates config, db, and images dir and exits", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")
		dbPath := filepath.Join(tempDir, "annotations.db")
		imagesPath := filepath.Join(tempDir, "images")

		_, errOut, err := executeCommand(tempDir)
		if err != nil {
			t.Fatalf("command execution failed: %v, output: %s", err, errOut)
		}

		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Errorf("expected config file to be created at %s, but it wasn't", configPath)
		}
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Errorf("expected database file to be created at %s, but it wasn't", dbPath)
		}
		if stat, err := os.Stat(imagesPath); os.IsNotExist(err) || !stat.IsDir() {
			t.Errorf("expected images directory to be created at %s, but it wasn't", imagesPath)
		}

		if !strings.Contains(errOut, "Creating default config") {
			t.Errorf("expected log output to contain 'Creating default config', but got: %s", errOut)
		}
		if !strings.Contains(errOut, "Creating empty database") {
			t.Errorf("expected log output to contain 'Creating empty database', but got: %s", errOut)
		}
		if !strings.Contains(errOut, "Creating images directory") {
			t.Errorf("expected log output to contain 'Creating images directory', but got: %s", errOut)
		}
	})

	t.Run("when argument is a directory and config exists, logs message and exits", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")
		dbPath := filepath.Join(tempDir, "annotations.db")
		imagesPath := filepath.Join(tempDir, "images")

		os.WriteFile(configPath, []byte(""), 0644) // Create dummy config

		_, errOut, err := executeCommand(tempDir)
		if err != nil {
			t.Fatalf("command execution failed: %v, output: %s", err, errOut)
		}

		if !strings.Contains(errOut, "Config file already exists") {
			t.Errorf("expected log output to contain 'Config file already exists', but got: %s", errOut)
		}
		// Check that db and images dir are still created if they don't exist
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Errorf("expected database file to be created at %s, but it wasn't", dbPath)
		}
		if stat, err := os.Stat(imagesPath); os.IsNotExist(err) || !stat.IsDir() {
			t.Errorf("expected images directory to be created at %s, but it wasn't", imagesPath)
		}
	})

	t.Run("when argument is a file, assumes it's a config and tries to run", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "test-config.yaml")
		dbPath := filepath.Join(tempDir, "annotations.db") // Expected default path
		imagesPath := filepath.Join(filepath.Dir(configPath), "images") // Expected default path
		
		// Create a valid config file
		validConfig := `
meta:
  description: "Sample annotation project."
auth:
  admin: { password: "changeme" }
  annotator: { password: "changeme" }
tasks:
  - id: quality
    name: "Image Quality Assessment"
    classes:
      good: { name: "Good" }
      bad: { name: "Bad" }
`
		os.WriteFile(configPath, []byte(validConfig), 0644)
		os.Mkdir(imagesPath, 0755)

		// Note: --database and --images flags are omitted to test the new default logic
		_, errOut, err := executeCommand(configPath, "--addr", ":8082")

		// We expect an error because the server will be interrupted or fail to bind in test.
		// The key is that it shouldn't be a "config file not found" or "images flag required" error.
		if err == nil {
			t.Log("command did not return an error, which is unexpected but could be ok if it timed out")
		}

		if strings.Contains(errOut, "images flag is required") {
			t.Errorf("should not have prompted for images flag, got: %s", errOut)
		}

		// The error might be a bind error if another test is running, which is fine.
		// The main thing is to check that it *tried* to start.
		// Note: The log message is "Server is ready and listening on: :8082", not "Starting server on: :8082"
		if !strings.Contains(errOut, "Server is ready and listening on: :8082") && !strings.Contains(errOut, "bind: address already in use") {
			t.Errorf("expected log to show server starting, but it didn't. Got: %s", errOut)
		}

		expectedDbLog := fmt.Sprintf("Database: %s", dbPath)
		if !strings.Contains(errOut, expectedDbLog) {
			t.Errorf("expected log to show default database path '%s', but it didn't. Got: %s", expectedDbLog, errOut)
		}

		expectedImagesLog := fmt.Sprintf("Images: %s", imagesPath)
		if !strings.Contains(errOut, expectedImagesLog) {
			t.Errorf("expected log to show default images path '%s', but it didn't. Got: %s", expectedImagesLog, errOut)
		}
	})

	t.Run("when argument is an invalid path, returns an error", func(t *testing.T) {
		invalidPath := "/path/to/some/nonexistent/dir"
		_, _, err := executeCommand(invalidPath) // No flags needed, it will fail on config load

		if err == nil {
			t.Fatal("expected an error for invalid path, but got none")
		}

		// The error should be about the config file, since it assumes the arg is a config file
		if !strings.Contains(err.Error(), "failed to load config") {
			t.Errorf("expected error to be about loading config, but got: %v", err)
		}
	})
}
