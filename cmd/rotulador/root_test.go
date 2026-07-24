package main

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// executeCommand runs a cobra command and captures stdout plus process stderr
// (slog writes to os.Stderr, not the legacy log package).
func executeCommand(args ...string) (string, string, error) {
	var out, errOut bytes.Buffer

	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)
	rootCmd.SetArgs(args)

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		return "", "", err
	}
	os.Stderr = w

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmdErr := rootCmd.ExecuteContext(ctx)

	_ = w.Close()
	os.Stderr = oldStderr
	stderrBytes, _ := io.ReadAll(r)
	_ = r.Close()
	errOut.Write(stderrBytes)

	return out.String(), errOut.String(), cmdErr
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

		if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
			t.Fatal(err)
		}

		_, errOut, err := executeCommand(tempDir)
		if err != nil {
			t.Fatalf("command execution failed: %v, output: %s", err, errOut)
		}

		if !strings.Contains(errOut, "Config file already exists") {
			t.Errorf("expected log output to contain 'Config file already exists', but got: %s", errOut)
		}
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Errorf("expected database file to be created at %s, but it wasn't", dbPath)
		}
		if stat, err := os.Stat(imagesPath); os.IsNotExist(err) || !stat.IsDir() {
			t.Errorf("expected images directory to be created at %s, but it wasn't", imagesPath)
		}
	})

	t.Run("when argument is a file, defaults database and images beside config", func(t *testing.T) {
		// Use an invalid addr so bind fails immediately after path defaults are logged.
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "test-config.yaml")
		dbPath := filepath.Join(tempDir, "annotations.db")
		imagesPath := filepath.Join(tempDir, "images")

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
		if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(imagesPath, 0755); err != nil {
			t.Fatal(err)
		}

		_, errOut, err := executeCommand(configPath, "--addr", "not-a-valid-address")
		if err == nil {
			t.Fatalf("expected bind/address error, got nil; logs: %s", errOut)
		}
		if strings.Contains(errOut, "images flag is required") {
			t.Errorf("should not have prompted for images flag, got: %s", errOut)
		}
		if !strings.Contains(errOut, dbPath) {
			t.Errorf("expected logs to mention default database path %q, got: %s", dbPath, errOut)
		}
		if !strings.Contains(errOut, imagesPath) {
			t.Errorf("expected logs to mention default images path %q, got: %s", imagesPath, errOut)
		}
	})

	t.Run("when argument is an invalid path, returns an error", func(t *testing.T) {
		invalidPath := "/path/to/some/nonexistent/dir"
		_, _, err := executeCommand(invalidPath)

		if err == nil {
			t.Fatal("expected an error for invalid path, but got none")
		}

		if !strings.Contains(err.Error(), "failed to load config") {
			t.Errorf("expected error to be about loading config, but got: %v", err)
		}
	})
}

// TestServeHTTP_StopsOnContextCancel ensures the HTTP server shuts down when the
// command context is cancelled (SIGINT/SIGTERM / test timeout path).
func TestServeHTTP_StopsOnContextCancel(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("close probe listener: %v", err)
	}

	server := &http.Server{
		Addr: addr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		ReadHeaderTimeout: time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- serveHTTP(ctx, server)
	}()

	// Wait until the server accepts connections.
	deadline := time.Now().Add(2 * time.Second)
	for {
		resp, getErr := http.Get("http://" + addr + "/")
		if getErr == nil {
			_ = resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatalf("server never became ready: %v", getErr)
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	select {
	case serveErr := <-errCh:
		if serveErr != nil {
			t.Fatalf("serveHTTP after cancel: %v", serveErr)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("serveHTTP did not return after context cancel")
	}
}

func TestServeHTTP_BindError(t *testing.T) {
	server := &http.Server{
		Addr: "not-a-valid-address",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		}),
		ReadHeaderTimeout: time.Second,
	}
	err := serveHTTP(context.Background(), server)
	if err == nil {
		t.Fatal("expected bind error, got nil")
	}
}
