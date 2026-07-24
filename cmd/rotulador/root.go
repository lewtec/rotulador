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

	"github.com/lewtec/rotulador/annotation"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "rotulador [folder|config.yaml]",
	Short: "Quickly make image annotations",
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

				if _, err := os.Stat(configFile); os.IsNotExist(err) {
					logger.Info("Creating default config", "configFile", configFile)
					if err := createSampleConfig(configFile, arg); err != nil {
						return fmt.Errorf("failed to create config: %w", err)
					}
					logger.Info("✓ Config file created.")
				} else {
					logger.Info("✓ Config file already exists.", "configFile", configFile)
				}

				// Create empty database file
				if _, err := os.Stat(databaseFile); os.IsNotExist(err) {
					logger.Info("Creating empty database", "databaseFile", databaseFile)
					file, err := os.Create(databaseFile)
					if err != nil {
						return fmt.Errorf("failed to create database file: %w", err)
					}
					if err := file.Close(); err != nil {
						annotation.ReportError(cmd.Context(), err, "msg", "failed to close database file", "path", databaseFile)
					}
					logger.Info("✓ Database file created.")
				} else {
					logger.Info("✓ Database file already exists.", "databaseFile", databaseFile)
				}

				// Create images directory
				if _, err := os.Stat(imagesDir); os.IsNotExist(err) {
					logger.Info("Creating images directory", "imagesDir", imagesDir)
					if err := os.MkdirAll(imagesDir, 0755); err != nil {
						return fmt.Errorf("failed to create images directory: %w", err)
					}
					logger.Info("✓ Images directory created.")
				} else {
					logger.Info("✓ Images directory already exists.", "imagesDir", imagesDir)
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
			c, _ := cmd.Flags().GetString("config")
			if c == "" {
				return fmt.Errorf("config file must be provided via argument or --config flag")
			}
			configFile = c
		}

		// 3. Determine databaseFile
		databaseFile, _ := cmd.Flags().GetString("database")
		if databaseFile == "" {
			databaseFile = filepath.Join(filepath.Dir(configFile), "annotations.db")
		}

		// 4. Determine imagesDir
		imagesDir, _ := cmd.Flags().GetString("images")
		if imagesDir == "" {
			imagesDir = filepath.Join(filepath.Dir(configFile), "images")
		}

		// 5. Server startup logic
		logger.Info("Initializing project...")

		config, err := annotation.LoadConfig(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		db, err := annotation.GetDatabase(databaseFile)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer func() {
			if err := db.Close(); err != nil {
				annotation.ReportError(cmd.Context(), err, "msg", "failed to close database")
			}
		}()

		app := &annotation.AnnotatorApp{
			ImagesDir: imagesDir,
			Database:  db,
			Config:    config,
			Logger:    logger,
		}

		// Run database migrations synchronously before starting the server
		if err := app.PrepareDatabaseMigrations(cmd.Context()); err != nil {
			return fmt.Errorf("failed to prepare database: %w", err)
		}

		addr, _ := cmd.Flags().GetString("addr")

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
				annotation.ReportError(cmd.Context(), err, "msg", "background image ingestion failed")
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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			annotation.ReportError(context.Background(), err, "msg", "HTTP server shutdown failed")
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
		return nil, fmt.Errorf("failed to read 'json' flag: %w", err)
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
