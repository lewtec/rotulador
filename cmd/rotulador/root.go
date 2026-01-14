/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
		// 1. Handle directory argument and exit
		if len(args) == 1 {
			arg := args[0]
			if stat, err := os.Stat(arg); err == nil && stat.IsDir() {
				// It's a folder. Check for config, create if needed, then exit.
				log.Printf("Detected folder argument: %s", arg)
				configFile := filepath.Join(arg, "config.yaml")
				databaseFile := filepath.Join(arg, "annotations.db")
				imagesDir := filepath.Join(arg, "images")

				if _, err := os.Stat(configFile); os.IsNotExist(err) {
					log.Printf("Creating default config: %s", configFile)
					if err := createSampleConfig(configFile, arg); err != nil {
						return fmt.Errorf("failed to create config: %w", err)
					}
					log.Printf("✓ Config file created.")
				} else {
					log.Printf("✓ Config file already exists: %s.", configFile)
				}

				// Create empty database file
				if _, err := os.Stat(databaseFile); os.IsNotExist(err) {
					log.Printf("Creating empty database: %s", databaseFile)
					file, err := os.Create(databaseFile)
					if err != nil {
						return fmt.Errorf("failed to create database file: %w", err)
					}
					file.Close()
					log.Printf("✓ Database file created.")
				} else {
					log.Printf("✓ Database file already exists: %s.", databaseFile)
				}

				// Create images directory
				if _, err := os.Stat(imagesDir); os.IsNotExist(err) {
					log.Printf("Creating images directory: %s", imagesDir)
					if err := os.MkdirAll(imagesDir, 0755); err != nil {
						return fmt.Errorf("failed to create images directory: %w", err)
					}
					log.Printf("✓ Images directory created.")
				} else {
					log.Printf("✓ Images directory already exists: %s.", imagesDir)
				}

				log.Printf("You can now run 'rotulador %s' to start the server.", arg)
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
		log.Printf("Initializing project...")

		config, err := annotation.LoadConfig(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		db, err := annotation.GetDatabase(databaseFile)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		app := &annotation.AnnotatorApp{
			ImagesDir: imagesDir,
			Database:  db,
			Config:    config,
		}

		// Run database migrations synchronously before starting the server
		if err := app.PrepareDatabaseMigrations(cmd.Context()); err != nil {
			return fmt.Errorf("failed to prepare database: %w", err)
		}

		addr, _ := cmd.Flags().GetString("addr")

		log.Printf("Configuration: %s", configFile)
		log.Printf("Database: %s", databaseFile)
		log.Printf("Images: %s", imagesDir)
		log.Printf("Tasks configured: %d", len(config.Tasks))
		for _, task := range config.Tasks {
			log.Printf("  - %s: %s", task.ID, task.Name)
		}

		// Start image ingestion in background (non-blocking)
		go func() {
			if err := app.IngestImages(context.Background()); err != nil {
				log.Printf("Error during background image ingestion: %v", err)
			}
		}()

		log.Printf("Server is ready and listening on: %s", addr)
		log.Printf("Images are being loaded in the background...")

		return http.ListenAndServe(addr, app.GetHTTPHandler())
	},
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		log.Fatalf("Error executing command: %v", err)
		os.Exit(1)
	}
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(hashPasswordCmd)

	// Optional flags (only used when not providing a folder argument)
	rootCmd.Flags().StringP("config", "c", "", "Config file for the annotation")
	rootCmd.Flags().StringP("database", "d", "", "Database file path (defaults to annotations.db in config file's directory)")
	rootCmd.Flags().StringP("images", "i", "", "Images directory path (defaults to 'images' in config file's directory)")
	rootCmd.Flags().StringP("addr", "a", ":8080", "Address to bind the webserver")
}
