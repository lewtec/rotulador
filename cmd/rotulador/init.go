package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lewtec/rotulador/annotation"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new annotation project",
	Long: `Initialize a new annotation project by creating:
- A sample configuration file (config.yaml)
- An empty SQLite database (annotations.db)
- Optionally, scan an images directory

Example:
  rotulador init --images-dir ./images
  rotulador init --images-dir ./images --config custom-config.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		imagesDir, _ := cmd.Flags().GetString("images-dir")
		configFile, _ := cmd.Flags().GetString("config")
		databaseFile, _ := cmd.Flags().GetString("database")

		// Create sample config if it doesn't exist
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			fmt.Printf("Creating sample configuration file: %s\n", configFile)
			if err := createSampleConfig(configFile, imagesDir); err != nil {
				return fmt.Errorf("failed to create config file: %w", err)
			}
			fmt.Printf("✓ Configuration file created successfully\n\n")
		} else {
			fmt.Printf("Configuration file already exists: %s\n", configFile)
		}

		// Load config
		config, err := annotation.LoadConfig(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Create database
		fmt.Printf("Creating database: %s\n", databaseFile)
		db, err := annotation.GetDatabase(databaseFile)
		if err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}
		defer func() {
			if err := db.Close(); err != nil {
				fmt.Printf("error closing database: %v\n", err)
			}
		}()

		// Initialize database if images directory is provided
		if imagesDir != "" {
			absPath, err := filepath.Abs(imagesDir)
			if err != nil {
				return fmt.Errorf("failed to resolve images path: %w", err)
			}

			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				return fmt.Errorf("images directory does not exist: %s", absPath)
			}

			fmt.Printf("Scanning images directory: %s\n", absPath)
			app := &annotation.AnnotatorApp{
				ImagesDir: absPath,
				Database:  db,
				Config:    config,
			}

			if err := app.PrepareDatabase(cmd.Context()); err != nil {
				return fmt.Errorf("failed to prepare database: %w", err)
			}
			fmt.Printf("✓ Database initialized with images from %s\n\n", absPath)
		} else {
			fmt.Printf("✓ Empty database created\n")
			fmt.Printf("  Run 'rotulador init --images-dir <path>' to populate with images\n\n")
		}

		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("✓ Initialization complete!")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Review and customize your config file:", configFile)
		fmt.Println("  2. Start the annotation server:")
		fmt.Printf("     rotulador annotator -c %s -d %s -i %s\n", configFile, databaseFile, imagesDir)
		fmt.Println("\nThen open http://localhost:8080 in your browser")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringP("images-dir", "i", "", "Directory containing images to annotate")
	initCmd.Flags().StringP("config", "c", "config.yaml", "Configuration file to create")
	initCmd.Flags().StringP("database", "d", "annotations.db", "Database file to create")
}

func createSampleConfig(filename, imagesDir string) error {
	sampleConfig := `# rotulador configuration file
# This file defines your annotation project

meta:
  description: |
    Sample annotation project.
    Edit this description to explain what you're annotating.

# Authentication - users who can access the annotation tool
auth:
  admin:
    password: "changeme"
  annotator:
    password: "changeme"

# Tasks - define the annotation workflow
tasks:
  # Example 1: Simple classification task
  - id: quality
    name: "Image Quality Assessment"
    short_name: "Quality"
    classes:
      good:
        name: "Good Quality"
        description: "Image is clear and well-focused"
      bad:
        name: "Poor Quality"
        description: "Image is blurry, dark, or has issues"
      unclear:
        name: "Unclear"
        description: "Cannot determine quality"

  # Example 2: Boolean task (built-in type)
  - id: contains_person
    name: "Does the image contain a person?"
    short_name: "Person"
    type: boolean  # Automatically creates Yes/No classes

  # Example 3: Conditional task (only shown if previous task matches)
  - id: person_age
    name: "Estimate person's age group"
    short_name: "Age"
    if:
      contains_person: "true"  # Only show if previous task answered "Yes"
    classes:
      child:
        name: "Child (0-12)"
      teen:
        name: "Teenager (13-19)"
      adult:
        name: "Adult (20-64)"
      senior:
        name: "Senior (65+)"

# Internationalization (optional)
# i18n:
#   - name: "Welcome"
#     value: "Bem-vindo"
#   - name: "Help"
#     value: "Ajuda"
`

	return os.WriteFile(filename, []byte(sampleConfig), 0644)
}
