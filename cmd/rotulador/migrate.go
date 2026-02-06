/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/lewtec/rotulador/annotation"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
)

// migrateCmd represents the migrate-legacy-db command
var migrateCmd = &cobra.Command{
	Use:   "migrate-legacy-db <old-db-path> <new-db-path> <config-file>",
	Short: "Migrate old database schema to new sqlc-based schema",
	Long: `Converts a database using the old dynamic task_* tables to the new unified annotations table.

The old schema used:
- images table with sha256 and filename
- Separate task_<taskid> tables for each annotation phase

The new schema uses:
- images table with id, path, and completion tracking
- Unified annotations table with stage_index

Example: rotulador migrate-legacy-db old.db new.db config.yaml`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger, err := getLogger(cmd)
		if err != nil {
			return err
		}
		oldDBPath := args[0]
		newDBPath := args[1]
		configPath := args[2]

		// Check if old database exists
		if _, err := os.Stat(oldDBPath); os.IsNotExist(err) {
			return fmt.Errorf("old database not found: %s", oldDBPath)
		}

		// Check if config exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return fmt.Errorf("config file not found: %s", configPath)
		}

		// Check if new database already exists
		if _, err := os.Stat(newDBPath); err == nil {
			return fmt.Errorf("new database already exists: %s (delete it first if you want to recreate)", newDBPath)
		}

		logger.Info("Starting database migration...",
			"oldDB", oldDBPath,
			"newDB", newDBPath,
			"config", configPath,
		)

		return migrateLegacyDatabase(cmd.Context(), oldDBPath, newDBPath, configPath, logger)
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}

type LegacyImage struct {
	SHA256   string
	Filename string
}

type LegacyAnnotation struct {
	Image string
	User  string
	Value string
	Sure  int
}

func migrateLegacyDatabase(ctx context.Context, oldDBPath, newDBPath, configPath string, logger *slog.Logger) error {
	// Load config to get task list
	config, err := loadConfigForMigration(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Open old database
	oldDB, err := sql.Open("sqlite3", oldDBPath)
	if err != nil {
		return fmt.Errorf("failed to open old database: %w", err)
	}
	defer func() {
		if err := oldDB.Close(); err != nil {
			annotation.ReportError(ctx, err, "msg", "failed to close old database")
		}
	}()

	// Verify old database has expected schema
	if err := verifyLegacySchema(ctx, oldDB, config.Tasks, logger); err != nil {
		return fmt.Errorf("old database schema validation failed: %w", err)
	}

	// Create new database
	newDB, err := sql.Open("sqlite3", newDBPath)
	if err != nil {
		return fmt.Errorf("failed to create new database: %w", err)
	}
	defer func() {
		if err := newDB.Close(); err != nil {
			annotation.ReportError(ctx, err, "msg", "failed to close new database")
		}
	}()

	// Run migrations on new database
	if err := runMigrations(ctx, newDB); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Start transaction
	tx, err := newDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			annotation.ReportError(ctx, err, "msg", "failed to rollback transaction")
		}
	}()

	// Step 1: Migrate images
	logger.Info("Migrating images...")
	imageMapping, err := migrateImages(ctx, oldDB, tx)
	if err != nil {
		return fmt.Errorf("failed to migrate images: %w", err)
	}
	logger.Info("✓ Migrated images", "count", len(imageMapping))

	// Step 2: Migrate annotations for each task
	for stageIndex, task := range config.Tasks {
		logger.Info("Migrating task", "taskID", task.ID, "stage", stageIndex)
		count, err := migrateTaskAnnotations(ctx, oldDB, tx, task.ID, stageIndex, imageMapping, logger)
		if err != nil {
			return fmt.Errorf("failed to migrate task %s: %w", task.ID, err)
		}
		logger.Info("✓ Migrated annotations", "count", count)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info("✓ Migration completed successfully!")
	logger.Info("You can now use the new database with 'rotulador'", "newDB", newDBPath)
	return nil
}

func verifyLegacySchema(ctx context.Context, db *sql.DB, tasks []ConfigTask, logger *slog.Logger) error {
	// Check if images table exists with sha256 column
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='images'").Scan(&count)
	if err != nil || count == 0 {
		return fmt.Errorf("images table not found in old database")
	}

	// Check if sha256 column exists
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('images') WHERE name='sha256'").Scan(&count)
	if err != nil || count == 0 {
		return fmt.Errorf("images table doesn't have sha256 column (not a legacy database?)")
	}

	// Check if task tables exist
	for _, task := range tasks {
		tableName := fmt.Sprintf("task_%s", task.ID)
		err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&count)
		if err != nil || count == 0 {
			logger.Warn("task table not found, skipping", "tableName", tableName)
		}
	}

	return nil
}

func runMigrations(ctx context.Context, db *sql.DB) error {
	// Read migration file
	migrationPath := "db/migrations/20240101000000_initial_schema.sql"
	content, err := os.ReadFile(migrationPath)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Extract the "up" part of the migration
	// Simple approach: everything between "-- migrate:up" and "-- migrate:down"
	migrationSQL := string(content)

	// Execute migration
	_, err = db.ExecContext(ctx, migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	return nil
}

func migrateImages(ctx context.Context, oldDB *sql.DB, newTx *sql.Tx) (map[string]int64, error) {
	// Query all images from old database
	rows, err := oldDB.QueryContext(ctx, "SELECT sha256, filename FROM images")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			annotation.ReportError(ctx, err, "msg", "failed to close rows")
		}
	}()

	// Map from old sha256 to new ID
	imageMapping := make(map[string]int64)

	for rows.Next() {
		var img LegacyImage
		if err := rows.Scan(&img.SHA256, &img.Filename); err != nil {
			return nil, err
		}

		// Insert into new database
		// Use filename as path (keeping same structure)
		result, err := newTx.ExecContext(ctx,
			"INSERT INTO images (path, original_filename) VALUES (?, ?)",
			img.Filename, filepath.Base(img.Filename))
		if err != nil {
			return nil, fmt.Errorf("failed to insert image %s: %w", img.SHA256, err)
		}

		newID, err := result.LastInsertId()
		if err != nil {
			return nil, err
		}

		imageMapping[img.SHA256] = newID
	}

	return imageMapping, rows.Err()
}

func migrateTaskAnnotations(ctx context.Context, oldDB *sql.DB, newTx *sql.Tx, taskID string, stageIndex int, imageMapping map[string]int64, logger *slog.Logger) (int, error) {
	tableName := fmt.Sprintf("task_%s", taskID)

	// Check if table exists
	var count int
	err := oldDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&count)
	if err != nil || count == 0 {
		// Table doesn't exist, skip
		return 0, nil
	}

	// Query all annotations from task table
	query := fmt.Sprintf("SELECT image, user, value FROM %s WHERE value IS NOT NULL", tableName)
	rows, err := oldDB.QueryContext(ctx, query)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			annotation.ReportError(ctx, err, "msg", "failed to close rows")
		}
	}()

	annotationCount := 0
	for rows.Next() {
		var ann LegacyAnnotation
		if err := rows.Scan(&ann.Image, &ann.User, &ann.Value); err != nil {
			return 0, err
		}

		// Get new image ID
		newImageID, ok := imageMapping[ann.Image]
		if !ok {
			logger.Warn("annotation references unknown image, skipping", "image", ann.Image)
			continue
		}

		// Insert annotation (use INSERT OR IGNORE to handle duplicates)
		_, err := newTx.ExecContext(ctx,
			"INSERT OR IGNORE INTO annotations (image_id, username, stage_index, option_value) VALUES (?, ?, ?, ?)",
			newImageID, ann.User, stageIndex, ann.Value)
		if err != nil {
			return 0, fmt.Errorf("failed to insert annotation: %w", err)
		}

		annotationCount++
	}

	return annotationCount, rows.Err()
}

// Minimal config structure for migration (avoiding circular dependency)
type ConfigTask struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

type ConfigMeta struct {
	Description string `yaml:"description"`
}

type Config struct {
	Meta  ConfigMeta   `yaml:"meta"`
	Tasks []ConfigTask `yaml:"tasks"`
}

func loadConfigForMigration(path string) (*Config, error) {
	// Simple YAML parsing for migration purposes
	// We could use the annotation package's LoadConfig but this avoids dependencies
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Use gopkg.in/yaml.v3 for parsing
	var config Config
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}
