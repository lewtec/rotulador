package annotation

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/lewtec/rotulador/db/migrations"
	"github.com/lewtec/rotulador/internal/repository"
)

func (a *AnnotatorApp) init() {
	if a.ImagesDir[len(a.ImagesDir)-1] == '/' {
		a.ImagesDir = a.ImagesDir[:len(a.ImagesDir)-1]
	}
	if a.OffsetAdvance == 0 {
		a.OffsetAdvance = 10
	}
	// Initialize repositories
	a.imageRepo = repository.NewImageRepository(a.Database)
	a.annotationRepo = repository.NewAnnotationRepository(a.Database)
}

// PrepareDatabase runs both database migrations and image ingestion synchronously.
// For better startup performance, consider using PrepareDatabaseMigrations() synchronously
// and IngestImages() asynchronously instead.
func (a *AnnotatorApp) PrepareDatabase(ctx context.Context) error {
	if err := a.PrepareDatabaseMigrations(ctx); err != nil {
		return err
	}
	if err := a.IngestImages(ctx); err != nil {
		return err
	}
	a.Logger.Info("PrepareDatabase: success! Database is ready")
	return nil
}

// PrepareDatabaseMigrations runs database schema migrations.
// This must be called synchronously before starting the HTTP server.
func (a *AnnotatorApp) PrepareDatabaseMigrations(ctx context.Context) error {
	a.init()
	db, err := sqlite.WithInstance(a.Database, &sqlite.Config{})
	if err != nil {
		return err
	}
	migrationsFS, err := iofs.New(migrations.Migrations, ".")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", migrationsFS, "sqlite", db)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	a.Logger.Info("PrepareDatabaseMigrations: migrations completed successfully")
	return nil
}

// IngestImages scans the images directory and loads all images into the database.
// This can be called asynchronously after the HTTP server starts.
func (a *AnnotatorApp) IngestImages(ctx context.Context) error {
	a.Logger.Info("IngestImages: starting image ingestion from directory", "dir", a.ImagesDir)

	err := filepath.WalkDir(a.ImagesDir, func(fullPath string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if fullPath == a.ImagesDir {
			return nil
		}
		if info.IsDir() {
			return fmt.Errorf("while checking if item '%s' is a file: datasets must be organized in a flat folder structure (hint: use the 'ingest' subcommand)", fullPath)
		}

		a.Logger.Debug("IngestImages: processing image", "path", fullPath)

		// Verify it's an image
		_, err = DecodeImage(fullPath)
		if err != nil {
			return fmt.Errorf("while checking if item '%s' is an image: %w", fullPath, err)
		}

		// Hash the file to get SHA256
		fileHash, err := HashFile(fullPath)
		if err != nil {
			return fmt.Errorf("while hashing image '%s': %w", fullPath, err)
		}

		// Use repository to create image (with upsert behavior via ON CONFLICT)
		_, err = a.imageRepo.Create(ctx, fileHash, info.Name())
		if err != nil {
			// Ignore duplicate errors (hash already exists)
			if !strings.Contains(err.Error(), "UNIQUE constraint") {
				return fmt.Errorf("while inserting image '%s': %w", fullPath, err)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("while ingesting images: %w", err)
	}

	a.Logger.Info("IngestImages: completed successfully!")
	return nil
}
