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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		require.NoError(t, err, "command execution failed")

		assert.FileExists(t, configPath, "expected config file to be created")
		assert.FileExists(t, dbPath, "expected database file to be created")
		assert.DirExists(t, imagesPath, "expected images directory to be created")

		assert.Contains(t, errOut, "Creating default config", "expected log output to contain 'Creating default config'")
		assert.Contains(t, errOut, "Creating empty database", "expected log output to contain 'Creating empty database'")
		assert.Contains(t, errOut, "Creating images directory", "expected log output to contain 'Creating images directory'")
	})

	t.Run("when argument is a directory and config exists, logs message and exits", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")
		dbPath := filepath.Join(tempDir, "annotations.db")
		imagesPath := filepath.Join(tempDir, "images")

		err := os.WriteFile(configPath, []byte(""), 0644) // Create dummy config
		require.NoError(t, err)

		_, errOut, err := executeCommand(tempDir)
		require.NoError(t, err, "command execution failed")

		assert.Contains(t, errOut, "Config file already exists", "expected log output to contain 'Config file already exists'")

		// Check that db and images dir are still created if they don't exist
		assert.FileExists(t, dbPath, "expected database file to be created")
		assert.DirExists(t, imagesPath, "expected images directory to be created")
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
		err := os.WriteFile(configPath, []byte(validConfig), 0644)
		require.NoError(t, err)
		err = os.Mkdir(imagesPath, 0755)
		require.NoError(t, err)

		// Note: --database and --images flags are omitted to test the new default logic
		_, errOut, err := executeCommand(configPath, "--addr", ":8082")

		// We expect an error because the server will be interrupted or fail to bind in test.
		// The key is that it shouldn't be a "config file not found" or "images flag required" error.
		if err == nil {
			t.Log("command did not return an error, which is unexpected but could be ok if it timed out")
		}

		assert.NotContains(t, errOut, "images flag is required", "should not have prompted for images flag")

		// The error might be a bind error if another test is running, which is fine.
		// The main thing is to check that it *tried* to start.
		isStarting := strings.Contains(errOut, "Server is ready and listening on: :8082")
		isBindError := strings.Contains(errOut, "bind: address already in use")
		assert.True(t, isStarting || isBindError, "expected log to show server starting or bind error, but got: %s", errOut)

		expectedDbLog := fmt.Sprintf("Database: %s", dbPath)
		assert.Contains(t, errOut, expectedDbLog, "expected log to show default database path")

		expectedImagesLog := fmt.Sprintf("Images: %s", imagesPath)
		assert.Contains(t, errOut, expectedImagesLog, "expected log to show default images path")
	})

	t.Run("when argument is an invalid path, returns an error", func(t *testing.T) {
		invalidPath := "/path/to/some/nonexistent/dir"
		_, _, err := executeCommand(invalidPath) // No flags needed, it will fail on config load

		require.Error(t, err, "expected an error for invalid path")

		// The error should be about the config file, since it assumes the arg is a config file
		assert.Contains(t, err.Error(), "failed to load config", "expected error to be about loading config")
	})
}
