/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	migrateSqlite "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/lewtec/rotulador/annotation"
	"github.com/lewtec/rotulador/db/migrations"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
)

// migrateCmd represents the migrate-legacy-db command
var migrateCmd = &cobra.Command{
	Use:   "migrate-legacy-db <old-db-path> <new-db-path> <config-file>",
	Short: "Migrate old database schema to the current sqlc-based schema",
	Long: `Converts a database using the old dynamic task_* tables to the current unified annotations table.

The old schema used:
- images table with sha256 and filename
- Separate task_<taskid> tables for each annotation phase

The current schema uses:
- images table keyed by sha256 with filename
- Unified annotations table with image_sha256 and stage_index

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

		if _, err := os.Stat(oldDBPath); os.IsNotExist(err) {
			return fmt.Errorf("old database not found: %s", oldDBPath)
		}
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return fmt.Errorf("config file not found: %s", configPath)
		}
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
}

func migrateLegacyDatabase(ctx context.Context, oldDBPath, newDBPath, configPath string, logger *slog.Logger) error {
	config, err := loadConfigForMigration(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	oldDB, err := sql.Open("sqlite", oldDBPath)
	if err != nil {
		return fmt.Errorf("failed to open old database: %w", err)
	}
	defer func() {
		if err := oldDB.Close(); err != nil {
			annotation.ReportError(ctx, err, "msg", "failed to close old database")
		}
	}()

	if err := verifyLegacySchema(ctx, oldDB, config.Tasks, logger); err != nil {
		return fmt.Errorf("old database schema validation failed: %w", err)
	}

	newDB, err := annotation.GetDatabase(newDBPath)
	if err != nil {
		return fmt.Errorf("failed to create new database: %w", err)
	}
	defer func() {
		if err := newDB.Close(); err != nil {
			annotation.ReportError(ctx, err, "msg", "failed to close new database")
		}
	}()

	if err := runMigrations(newDB); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	tx, err := newDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			annotation.ReportError(ctx, err, "msg", "failed to rollback transaction")
		}
	}()

	logger.Info("Migrating images...")
	knownImages, err := migrateImages(ctx, oldDB, tx)
	if err != nil {
		return fmt.Errorf("failed to migrate images: %w", err)
	}
	logger.Info("Migrated images", "count", len(knownImages))

	for stageIndex, task := range config.Tasks {
		logger.Info("Migrating task", "taskID", task.ID, "stage", stageIndex)
		count, err := migrateTaskAnnotations(ctx, oldDB, tx, task.ID, stageIndex, knownImages, logger)
		if err != nil {
			return fmt.Errorf("failed to migrate task %s: %w", task.ID, err)
		}
		logger.Info("Migrated annotations", "count", count)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info("Migration completed successfully!", "newDB", newDBPath)
	return nil
}

func verifyLegacySchema(ctx context.Context, db *sql.DB, tasks []ConfigTask, logger *slog.Logger) error {
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='images'").Scan(&count)
	if err != nil || count == 0 {
		return fmt.Errorf("images table not found in old database")
	}

	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('images') WHERE name='sha256'").Scan(&count)
	if err != nil || count == 0 {
		return fmt.Errorf("images table doesn't have sha256 column (not a legacy database?)")
	}

	for _, task := range tasks {
		tableName := fmt.Sprintf("task_%s", task.ID)
		err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&count)
		if err != nil || count == 0 {
			logger.Warn("task table not found, skipping", "tableName", tableName)
		}
	}

	return nil
}

func runMigrations(db *sql.DB) error {
	driver, err := migrateSqlite.WithInstance(db, &migrateSqlite.Config{})
	if err != nil {
		return err
	}
	src, err := iofs.New(migrations.Migrations, ".")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", src, "sqlite", driver)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// migrateImages copies images into the current schema and returns the set of known sha256 keys.
func migrateImages(ctx context.Context, oldDB *sql.DB, newTx *sql.Tx) (map[string]struct{}, error) {
	rows, err := oldDB.QueryContext(ctx, "SELECT sha256, filename FROM images")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			annotation.ReportError(ctx, err, "msg", "failed to close rows")
		}
	}()

	known := make(map[string]struct{})
	for rows.Next() {
		var img LegacyImage
		if err := rows.Scan(&img.SHA256, &img.Filename); err != nil {
			return nil, err
		}
		_, err := newTx.ExecContext(ctx,
			"INSERT INTO images (sha256, filename) VALUES (?, ?) ON CONFLICT(sha256) DO UPDATE SET filename = excluded.filename",
			img.SHA256, img.Filename)
		if err != nil {
			return nil, fmt.Errorf("failed to insert image %s: %w", img.SHA256, err)
		}
		known[img.SHA256] = struct{}{}
	}
	return known, rows.Err()
}

func migrateTaskAnnotations(ctx context.Context, oldDB *sql.DB, newTx *sql.Tx, taskID string, stageIndex int, knownImages map[string]struct{}, logger *slog.Logger) (int, error) {
	tableName := fmt.Sprintf("task_%s", taskID)

	var count int
	err := oldDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&count)
	if err != nil || count == 0 {
		return 0, nil
	}

	// #nosec G201 -- tableName is derived from config task IDs validated earlier
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
		if _, ok := knownImages[ann.Image]; !ok {
			logger.Warn("annotation references unknown image, skipping", "image", ann.Image)
			continue
		}
		_, err := newTx.ExecContext(ctx,
			`INSERT INTO annotations (image_sha256, username, stage_index, option_value)
			 VALUES (?, ?, ?, ?)
			 ON CONFLICT(image_sha256, username, stage_index)
			 DO UPDATE SET option_value = excluded.option_value`,
			ann.Image, ann.User, stageIndex, ann.Value)
		if err != nil {
			return 0, fmt.Errorf("failed to insert annotation: %w", err)
		}
		annotationCount++
	}
	return annotationCount, rows.Err()
}

// Minimal config structure for migration
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
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config Config
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return &config, nil
}
