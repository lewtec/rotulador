/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"io/fs"

	"github.com/lewtec/rotulador/internal/web"
	"github.com/spf13/cobra"
)

// cliError is a stable CLI-level sentinel. Prefer these (or fmt.Errorf %w
// wrapping them) over bare fmt.Errorf so callers can errors.Is.
type cliError string

func (e cliError) Error() string { return string(e) }

// CLI error table for root command validation (go/fmt-errorf-missing-wrap,
// go/error-prefer-err-tables).
const (
	errConfigRequired cliError = "config file must be provided via argument or --config flag"
)

// version is set at link time via -ldflags (GoReleaser / mise release).
// Local builds default to "dev".
var version = "dev"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "rotulador [folder|config.yaml]",
	Short:   "Quickly make image annotations",
	Version: version,
	Long: strings.TrimSpace(`
With a set of trivial choices scale the classification of a set of images to many people to build datasets to train classifiers.
    `),
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger, err := getLogger(cmd)
		if err != nil {
			return err
		}
		// 1. Handle directory argument and exit
		if len(args) == 1 {
			arg := args[0]
			if stat, err := os.Stat(arg); err == nil && stat.IsDir() {
				// It's a folder. Check for config, create if needed, then exit.
				logger.Info("Detected folder argument", "arg", arg)
				configFile := filepath.Join(arg, "config.yaml")
				databaseFile := filepath.Join(arg, "annotations.db")
				imagesDir := filepath.Join(arg, "images")

				// Create only on ErrNotExist. Other Stat failures (e.g. permission
				// denied) must not be treated as "already exists".
				if err := ensureAbsent(logger, absentPath{
					path:      configFile,
					key:       "configFile",
					statErr:   "stat config file",
					createErr: "create config",
					creating:  "Creating default config",
					created:   "✓ Config file created.",
					exists:    "✓ Config file already exists.",
					create:    func() error { return createSampleConfig(configFile, arg) },
				}); err != nil {
					return err
				}
				if err := ensureAbsent(logger, absentPath{
					path:      databaseFile,
					key:       "databaseFile",
					statErr:   "stat database file",
					createErr: "create database file",
					creating:  "Creating empty database",
					created:   "✓ Database file created.",
					exists:    "✓ Database file already exists.",
					create: func() error {
						file, err := os.Create(databaseFile)
						if err != nil {
							return err
						}
						if err := file.Close(); err != nil {
							web.ReportError(cmd.Context(), err, "msg", "failed to close database file", "path", databaseFile)
						}
						return nil
					},
				}); err != nil {
					return err
				}
				if err := ensureAbsent(logger, absentPath{
					path:      imagesDir,
					key:       "imagesDir",
					statErr:   "stat images directory",
					createErr: "create images directory",
					creating:  "Creating images directory",
					created:   "✓ Images directory created.",
					exists:    "✓ Images directory already exists.",
					create:    func() error { return os.MkdirAll(imagesDir, 0755) },
				}); err != nil {
					return err
				}

				logger.Info("You can now run 'rotulador' to start the server.", "arg", arg)
				return nil // Always exit after handling a directory argument
			}
		}

		// 2. Determine configFile
		var configFile string
		if len(args) == 1 {
			// This runs only if the arg was not a directory.
			configFile = args[0]
		} else {
			c, err := cmd.Flags().GetString("config")
			if err != nil {
				return fmt.Errorf("read config flag: %w", err)
			}
			if c == "" {
				return errConfigRequired
			}
			configFile = c
		}

		// 3. Determine databaseFile
		databaseFile, err := cmd.Flags().GetString("database")
		if err != nil {
			return fmt.Errorf("read database flag: %w", err)
		}
		if databaseFile == "" {
			databaseFile = filepath.Join(filepath.Dir(configFile), "annotations.db")
		}

		// 4. Determine imagesDir
		imagesDir, err := cmd.Flags().GetString("images")
		if err != nil {
			return fmt.Errorf("read images flag: %w", err)
		}
		if imagesDir == "" {
			imagesDir = filepath.Join(filepath.Dir(configFile), "images")
		}

		// 5. Server startup logic
		logger.Info("Initializing project...")

		config, err := web.LoadConfig(configFile)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		db, err := web.GetDatabase(databaseFile)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer closeDatabase(cmd.Context(), db)

		app := &web.AnnotatorApp{
			ImagesDir: imagesDir,
			Database:  db,
			Config:    config,
			Logger:    logger,
		}

		// Run database migrations synchronously before starting the server
		if err := app.PrepareDatabaseMigrations(cmd.Context()); err != nil {
			return fmt.Errorf("prepare database: %w", err)
		}

		addr, err := cmd.Flags().GetString("addr")
		if err != nil {
			return fmt.Errorf("read addr flag: %w", err)
		}

		logger.Info("Configuration",
			"configFile", configFile,
			"databaseFile", databaseFile,
			"imagesDir", imagesDir,
		)
		logger.Info("Tasks configured", "count", len(config.Tasks))
		for _, task := range config.Tasks {
			logger.Info("  -", "id", task.ID, "name", task.Name)
		}

		// Start image ingestion in background (non-blocking)
		go func() {
			if err := app.IngestImages(cmd.Context()); err != nil {
				// Shutdown cancels the command context; treat that as a normal stop.
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					logger.Info("background image ingestion stopped", "err", err)
					return
				}
				web.ReportError(cmd.Context(), err, "msg", "background image ingestion failed")
			}
		}()

		logger.Info("Server is ready and listening", "addr", addr)
		logger.Info("Images are being loaded in the background...")

		// Bound request lifetimes to mitigate slowloris / stuck clients.
		server := &http.Server{
			Addr:              addr,
			Handler:           app.GetHTTPHandler(),
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      60 * time.Second,
			IdleTimeout:       120 * time.Second,
			BaseContext: func(_ net.Listener) context.Context {
				return cmd.Context()
			},
		}
		// Honor SIGINT/SIGTERM (and test timeouts) via command context instead of
		// blocking forever in ListenAndServe.
		return serveHTTP(cmd.Context(), server)
	},
}

// serveHTTP runs server until it fails to bind/serve or ctx is cancelled.
// On cancel it shuts the server down gracefully and returns nil so the CLI
// exits cleanly (signal stop is not an error).
func serveHTTP(ctx context.Context, server *http.Server) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		// Parent ctx is already cancelled; WithoutCancel keeps values (e.g. logger)
		// while WithTimeout still bounds Shutdown independently of the signal.
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			web.ReportError(shutdownCtx, err, "msg", "HTTP server shutdown failed")
			// Drain ListenAndServe so we do not leak the goroutine on return.
			<-errCh
			return fmt.Errorf("http server shutdown: %w", err)
		}
		err := <-errCh
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

// absentPath is one project path created when missing.
// statErr and createErr are the prefixes wrapped around Stat and create errors.
type absentPath struct {
	path      string
	key       string
	statErr   string
	createErr string
	creating  string
	created   string
	exists    string
	create    func() error
}

func ensureAbsent(logger *slog.Logger, p absentPath) error {
	if _, err := os.Stat(p.path); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("%s: %w", p.statErr, err)
		}
		logger.Info(p.creating, p.key, p.path)
		if err := p.create(); err != nil {
			return fmt.Errorf("%s: %w", p.createErr, err)
		}
		logger.Info(p.created)
		return nil
	}
	logger.Info(p.exists, p.key, p.path)
	return nil
}

func main() {
	var logger *slog.Logger
	// Pre-initialize logger before cobra parsing
	if len(os.Args) > 1 {
		for _, arg := range os.Args[1:] {
			if arg == "--json" {
				logger = slog.New(slog.NewJSONHandler(os.Stderr, nil))
				break
			}
		}
	}
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	ctx := context.WithValue(context.Background(), loggerKey, logger)
	// Cancel the command context on SIGINT/SIGTERM so the HTTP server can shut down.
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		logger.Error("Error executing command", "err", err)
		os.Exit(1)
	}
}

type contextKey string

const loggerKey contextKey = "logger"

func getLogger(cmd *cobra.Command) (*slog.Logger, error) {
	// 1. Get from context (highest priority)
	if logger, ok := cmd.Context().Value(loggerKey).(*slog.Logger); ok {
		return logger, nil
	}

	// 2. Get from --json flag
	useJSON, err := cmd.Flags().GetBool("json")
	if err != nil {
		return nil, fmt.Errorf("read 'json' flag: %w", err)
	}
	if useJSON {
		return slog.New(slog.NewJSONHandler(os.Stderr, nil)), nil
	}

	// 3. Default to text handler
	return slog.New(slog.NewTextHandler(os.Stderr, nil)), nil
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(hashPasswordCmd)

	// Optional flags (only used when not providing a folder argument)
	rootCmd.Flags().StringP("config", "c", "", "Config file for the annotation")
	rootCmd.Flags().StringP("database", "d", "", "Database file path (defaults to annotations.db in config file's directory)")
	rootCmd.Flags().StringP("images", "i", "", "Images directory path (defaults to 'images' in config file's directory)")
	rootCmd.Flags().StringP("addr", "a", ":8080", "Address to bind the webserver")
	rootCmd.PersistentFlags().Bool("json", false, "Enable JSON logging")
}
